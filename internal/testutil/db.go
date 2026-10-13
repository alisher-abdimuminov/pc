// Package testutil provides the shared test database helpers.
package testutil

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"pc/internal/migrate"
)

var migrateOnce sync.Once

// Pool returns a pool on the test database with all tables emptied.
// It skips when DATABASE_URL is unset and refuses to touch a non-test database.
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(cfg.ConnConfig.Database, "_test") {
		t.Fatalf("refusing to run DB tests against %q (name must end with _test)", cfg.ConnConfig.Database)
	}
	var migErr error
	migrateOnce.Do(func() { migErr = migrate.Run(ctx, url, "up") })
	if migErr != nil {
		t.Fatalf("migrate: %v", migErr)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, `TRUNCATE audit_logs, submissions, assignments, attendance_attempts,
		attendance_steps, attendances, schedules, locations, sessions, groups, users, faculties CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return pool
}
