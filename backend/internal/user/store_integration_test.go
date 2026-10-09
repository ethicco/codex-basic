package user

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"uptime-backend/internal/database"
	"uptime-backend/internal/monitor"
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
	first := monitor.Monitor{ID: "monitor-1-" + stamp, UserID: account.ID, TargetURL: "https://example.com", IntervalSeconds: 60, CreatedAt: time.Now().UTC()}
	second := monitor.Monitor{ID: "monitor-2-" + stamp, UserID: account.ID, TargetURL: "https://example.org", IntervalSeconds: 5, CreatedAt: first.CreatedAt.Add(time.Second)}
	if err := store.CreateMonitor(ctx, first); err != nil {
		t.Fatalf("CreateMonitor(first) error = %v", err)
	}
	if err := store.CreateMonitor(ctx, second); err != nil {
		t.Fatalf("CreateMonitor(second) error = %v", err)
	}
	monitors, err := store.ListMonitorsByUserID(ctx, account.ID, 1, nil)
	if err != nil || len(monitors) != 1 || monitors[0].ID != second.ID {
		t.Fatalf("ListMonitorsByUserID() = %#v, %v", monitors, err)
	}
	page, err := store.ListMonitorsByUserID(ctx, account.ID, 1, &monitor.Cursor{CreatedAt: monitors[0].CreatedAt, ID: monitors[0].ID})
	if err != nil || len(page) != 1 || page[0].ID != first.ID {
		t.Fatalf("ListMonitorsByUserID(cursor) = %#v, %v", page, err)
	}
	updated, err := store.UpdateMonitor(ctx, monitor.Monitor{
		ID:              first.ID,
		UserID:          account.ID,
		TargetURL:       "https://example.net/health",
		IntervalSeconds: 3600,
	})
	if err != nil || updated.TargetURL != "https://example.net/health" || updated.IntervalSeconds != 3600 {
		t.Fatalf("UpdateMonitor() = %#v, %v", updated, err)
	}
	if err := store.DeleteMonitor(ctx, account.ID, second.ID); err != nil {
		t.Fatalf("DeleteMonitor() error = %v", err)
	}
	if err := store.DeleteMonitor(ctx, account.ID, second.ID); !errors.Is(err, monitor.ErrNotFound) {
		t.Fatalf("repeated DeleteMonitor() error = %v", err)
	}
}
