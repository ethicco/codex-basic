package database

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMigrateCreatesAuthTables(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	defer pool.Close()
	if err := Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	for _, table := range []string{"users", "refresh_sessions"} {
		var exists bool
		if err := pool.QueryRow(context.Background(), `SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name=$1)`, table).Scan(&exists); err != nil || !exists {
			t.Fatalf("table %s exists = %t, err = %v", table, exists, err)
		}
	}
}
