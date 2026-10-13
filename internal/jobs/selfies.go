package jobs

import (
	"context"
	"time"

	"go.uber.org/zap"

	"pc/internal/db"
	"pc/internal/storage"
)

const selfieBatch = 200

// CleanupSelfies deletes selfie files of attempts older than cutoff and clears their image column.
// The attempt rows themselves are kept. It returns how many files were removed.
func CleanupSelfies(ctx context.Context, q *db.Queries, store storage.Storage, cutoff time.Time) (int, error) {
	total := 0
	for {
		rows, err := q.ListExpiredSelfies(ctx, db.ListExpiredSelfiesParams{AttemptedAt: tstz(cutoff), Limit: selfieBatch})
		if err != nil {
			return total, err
		}
		if len(rows) == 0 {
			return total, nil
		}
		for _, r := range rows {
			// Delete the file first: if clearing the column then fails, the next run retries safely
			// because Delete treats a missing file as success.
			if err := store.Delete(ctx, r.Image); err != nil {
				return total, err
			}
			if err := q.ClearAttemptImage(ctx, r.ID); err != nil {
				return total, err
			}
			total++
		}
	}
}

// RunSelfieCleanup runs CleanupSelfies once at start and then every interval until ctx is done.
func RunSelfieCleanup(ctx context.Context, q *db.Queries, store storage.Storage, log *zap.Logger, retentionDays int, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		cutoff := time.Now().AddDate(0, 0, -retentionDays)
		n, err := CleanupSelfies(ctx, q, store, cutoff)
		switch {
		case err != nil:
			log.Error("selfie cleanup failed", zap.Error(err), zap.Int("removed", n))
		default:
			log.Info("selfie cleanup done", zap.Int("removed", n), zap.Int("retention_days", retentionDays))
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
