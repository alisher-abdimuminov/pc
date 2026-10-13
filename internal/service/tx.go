package service

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"pc/internal/db"
)

// InTx runs fn in a transaction; fn receives the tx (for audit.Write) and tx-bound queries.
func InTx(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx, q *db.Queries) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if err := fn(tx, db.New(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
