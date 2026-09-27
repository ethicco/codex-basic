package user

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"uptime-backend/internal/database"
)

func TestStorePersistsUsersAndRefreshSessionLifecycle(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	defer pool.Close()
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	stamp := time.Now().UTC().Format("20060102150405.000000000")
	account := User{ID: "integration-user-" + stamp, Email: "Person-" + stamp + "@example.com", PasswordHash: "hash", CreatedAt: time.Now().UTC()}
	session := Session{ID: "integration-session-" + stamp, UserID: account.ID, RefreshJTIHash: []byte("first-token-hash"), ExpiresAt: time.Now().UTC().Add(time.Hour), CreatedAt: time.Now().UTC()}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM refresh_sessions WHERE user_id=$1`, account.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, account.ID)
	}()

	store := NewStore(pool)
	if err := store.CreateUserWithSession(ctx, account, session); err != nil {
		t.Fatalf("CreateUserWithSession() error = %v", err)
	}
	found, ok, err := store.FindByEmail(ctx, account.Email)
	if err != nil || !ok || found.ID != account.ID || found.Email != "person-"+stamp+"@example.com" {
		t.Fatalf("FindByEmail() = %#v, %t, %v", found, ok, err)
	}
	if err := store.CreateUserWithSession(ctx, account, session); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("duplicate CreateUserWithSession() error = %v", err)
	}
	rotated := session
	rotated.RefreshJTIHash = []byte("second-token-hash")
	if err := store.RotateSession(ctx, rotated, session.RefreshJTIHash); err != nil {
		t.Fatalf("RotateSession() error = %v", err)
	}
	if err := store.RotateSession(ctx, rotated, session.RefreshJTIHash); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("replayed RotateSession() error = %v", err)
	}
	if err := store.RevokeSession(ctx, session.ID, account.ID, rotated.RefreshJTIHash); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("RevokeSession() after replay error = %v", err)
	}
}
