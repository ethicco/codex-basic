// Package user provides PostgreSQL-backed account and refresh-session storage.
package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"uptime-backend/internal/monitor"
)

var (
	ErrEmailTaken     = errors.New("email is already registered")
	ErrSessionInvalid = errors.New("refresh session is invalid")
	ErrProfileExists  = errors.New("profile already exists")
	ErrProfileMissing = errors.New("profile does not exist")
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

type Session struct {
	ID             string
	UserID         string
	RefreshJTIHash []byte
	ExpiresAt      time.Time
	CreatedAt      time.Time
}

type Profile struct {
	UserID     string
	Name       string
	AvatarFile string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Repository interface {
	CreateUserWithSession(ctx context.Context, account User, session Session) error
	FindByEmail(ctx context.Context, email string) (User, bool, error)
	CreateSession(ctx context.Context, session Session) error
	RotateSession(ctx context.Context, session Session, previousHash []byte) error
	RevokeSession(ctx context.Context, sessionID, userID string, refreshJTIHash []byte) error
	CreateProfile(ctx context.Context, profile Profile) error
	FindProfileByUserID(ctx context.Context, userID string) (Profile, bool, error)
	UpdateProfileName(ctx context.Context, userID, name string) (Profile, error)
	UpdateProfileAvatar(ctx context.Context, userID, avatarFile string) (Profile, error)
}

type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) CreateUserWithSession(ctx context.Context, account User, session Session) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin account creation: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `INSERT INTO users (id, email, password_hash, created_at) VALUES ($1, $2, $3, $4)`, account.ID, normalizeEmail(account.Email), account.PasswordHash, account.CreatedAt); err != nil {
		if isUniqueViolation(err) {
			return ErrEmailTaken
		}
		return fmt.Errorf("insert user: %w", err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO refresh_sessions (id, user_id, refresh_jti_hash, expires_at, created_at) VALUES ($1, $2, $3, $4, $5)`, session.ID, session.UserID, session.RefreshJTIHash, session.ExpiresAt, session.CreatedAt); err != nil {
		return fmt.Errorf("insert refresh session: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit account creation: %w", err)
	}
	return nil
}

func (s *Store) FindByEmail(ctx context.Context, email string) (User, bool, error) {
	var account User
	err := s.pool.QueryRow(ctx, `SELECT id, email, password_hash, created_at FROM users WHERE email=$1`, normalizeEmail(email)).Scan(&account.ID, &account.Email, &account.PasswordHash, &account.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, false, nil
	}
	if err != nil {
		return User{}, false, fmt.Errorf("find user: %w", err)
	}
	return account, true, nil
}

func (s *Store) CreateSession(ctx context.Context, session Session) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO refresh_sessions (id, user_id, refresh_jti_hash, expires_at, created_at) VALUES ($1, $2, $3, $4, $5)`, session.ID, session.UserID, session.RefreshJTIHash, session.ExpiresAt, session.CreatedAt)
	if err != nil {
		return fmt.Errorf("create refresh session: %w", err)
	}
	return nil
}

// RotateSession replaces a current refresh identifier. A mismatch is treated as
// a refresh-token replay and revokes that session before returning ErrSessionInvalid.
func (s *Store) RotateSession(ctx context.Context, session Session, previousHash []byte) error {
	command, err := s.pool.Exec(ctx, `UPDATE refresh_sessions SET refresh_jti_hash=$1, expires_at=$2 WHERE id=$3 AND user_id=$4 AND refresh_jti_hash=$5 AND revoked_at IS NULL AND expires_at > now()`, session.RefreshJTIHash, session.ExpiresAt, session.ID, session.UserID, previousHash)
	if err != nil {
		return fmt.Errorf("rotate refresh session: %w", err)
	}
	if command.RowsAffected() == 1 {
		return nil
	}
	if _, err := s.pool.Exec(ctx, `UPDATE refresh_sessions SET revoked_at=now() WHERE id=$1 AND user_id=$2 AND revoked_at IS NULL`, session.ID, session.UserID); err != nil {
		return fmt.Errorf("revoke replayed refresh session: %w", err)
	}
	return ErrSessionInvalid
}

func (s *Store) RevokeSession(ctx context.Context, sessionID, userID string, refreshJTIHash []byte) error {
	command, err := s.pool.Exec(ctx, `UPDATE refresh_sessions SET revoked_at=now() WHERE id=$1 AND user_id=$2 AND refresh_jti_hash=$3 AND revoked_at IS NULL AND expires_at > now()`, sessionID, userID, refreshJTIHash)
	if err != nil {
		return fmt.Errorf("revoke refresh session: %w", err)
	}
	if command.RowsAffected() != 1 {
		return ErrSessionInvalid
	}
	return nil
}

func (s *Store) CreateProfile(ctx context.Context, profile Profile) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO profiles (user_id, name, created_at, updated_at) VALUES ($1, $2, $3, $4)`, profile.UserID, profile.Name, profile.CreatedAt, profile.UpdatedAt)
	if isUniqueViolation(err) {
		return ErrProfileExists
	}
	if err != nil {
		return fmt.Errorf("create profile: %w", err)
	}
	return nil
}

func (s *Store) FindProfileByUserID(ctx context.Context, userID string) (Profile, bool, error) {
	var profile Profile
	err := s.pool.QueryRow(ctx, `SELECT user_id, name, avatar_file, created_at, updated_at FROM profiles WHERE user_id=$1`, userID).Scan(&profile.UserID, &profile.Name, &profile.AvatarFile, &profile.CreatedAt, &profile.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, false, nil
	}
	if err != nil {
		return Profile{}, false, fmt.Errorf("find profile: %w", err)
	}
	return profile, true, nil
}

func (s *Store) UpdateProfileName(ctx context.Context, userID, name string) (Profile, error) {
	var profile Profile
	err := s.pool.QueryRow(ctx, `UPDATE profiles SET name=$1, updated_at=now() WHERE user_id=$2 RETURNING user_id, name, avatar_file, created_at, updated_at`, name, userID).Scan(&profile.UserID, &profile.Name, &profile.AvatarFile, &profile.CreatedAt, &profile.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrProfileMissing
	}
	if err != nil {
		return Profile{}, fmt.Errorf("update profile: %w", err)
	}
	return profile, nil
}

func (s *Store) UpdateProfileAvatar(ctx context.Context, userID, avatarFile string) (Profile, error) {
	var profile Profile
	err := s.pool.QueryRow(ctx, `UPDATE profiles SET avatar_file=$1, updated_at=now() WHERE user_id=$2 RETURNING user_id, name, avatar_file, created_at, updated_at`, avatarFile, userID).Scan(&profile.UserID, &profile.Name, &profile.AvatarFile, &profile.CreatedAt, &profile.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrProfileMissing
	}
	if err != nil {
		return Profile{}, fmt.Errorf("update profile avatar: %w", err)
	}
	return profile, nil
}

func (s *Store) CreateMonitor(ctx context.Context, entry monitor.Monitor) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin monitor creation: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT id FROM users WHERE id=$1 FOR UPDATE`, entry.UserID); err != nil {
		return fmt.Errorf("lock monitor owner: %w", err)
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM monitors WHERE user_id=$1`, entry.UserID).Scan(&count); err != nil {
		return fmt.Errorf("count monitors: %w", err)
	}
	if count >= monitor.MaxPerUser {
		return monitor.ErrLimitReached
	}
	if _, err := tx.Exec(ctx, `INSERT INTO monitors (id, user_id, target_url, interval_seconds, created_at) VALUES ($1, $2, $3, $4, $5)`, entry.ID, entry.UserID, entry.TargetURL, entry.IntervalSeconds, entry.CreatedAt); err != nil {
		return fmt.Errorf("create monitor: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit monitor creation: %w", err)
	}
	return nil
}

func (s *Store) ListMonitorsByUserID(ctx context.Context, userID string, limit int, cursor *monitor.Cursor) ([]monitor.Monitor, error) {
	if limit < 1 || limit > monitor.MaxPageSize+1 {
		limit = monitor.DefaultPageSize
	}
	query := `SELECT id, user_id, target_url, interval_seconds, created_at FROM monitors WHERE user_id=$1`
	args := []any{userID}
	if cursor != nil {
		query += ` AND (created_at, id) < ($2, $3)`
		args = append(args, cursor.CreatedAt, cursor.ID)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT $` + fmt.Sprint(len(args)+1)
	args = append(args, limit)
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list monitors: %w", err)
	}
	defer rows.Close()
	monitors := []monitor.Monitor{}
	for rows.Next() {
		var entry monitor.Monitor
		if err := rows.Scan(&entry.ID, &entry.UserID, &entry.TargetURL, &entry.IntervalSeconds, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan monitor: %w", err)
		}
		monitors = append(monitors, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate monitors: %w", err)
	}
	return monitors, nil
}

func normalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
