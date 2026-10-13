package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"pc/migrations"
)

// Run executes a goose command ("up", "down", "status") against the embedded migrations.
func Run(ctx context.Context, databaseURL, command string) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	var runErr error
	switch command {
	case "up":
		runErr = goose.UpContext(ctx, db, ".")
	case "down":
		runErr = goose.DownContext(ctx, db, ".")
	case "status":
		runErr = goose.StatusContext(ctx, db, ".")
	default:
		return fmt.Errorf("unknown migrate command %q (use up|down|status)", command)
	}
	// Phase 0 ships no migrations yet; that is not a failure.
	if errors.Is(runErr, goose.ErrNoMigrationFiles) {
		return nil
	}
	return runErr
}
