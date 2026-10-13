package jobs

import (
	"context"
	"time"

	"go.uber.org/zap"

	"pc/internal/db"
)

// RunSessionCleanup deletes expired sessions every interval until ctx is done.
func RunSessionCleanup(ctx context.Context, q *db.Queries, log *zap.Logger, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		n, err := q.DeleteExpiredSessions(ctx)
		if err != nil {
			log.Error("session cleanup failed", zap.Error(err))
		} else if n > 0 {
			log.Info("expired sessions removed", zap.Int64("count", n))
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
