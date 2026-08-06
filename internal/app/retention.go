package app

import (
	"context"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/observability"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/storage/history"
)

func runHistoryRetention(
	ctx context.Context,
	repository *history.Repository,
	now func() time.Time,
	logger *observability.Logger,
) {
	cleanup := func() {
		result, err := repository.CleanupExpiredBatches(
			ctx,
			now().UTC(),
			history.DefaultRetentionBatchSize,
			history.DefaultRetentionMaxBatches,
		)
		if err != nil {
			if ctx.Err() == nil {
				_ = logger.Warning(observability.Event{Component: "history", Code: "history.retention.failed"})
			}
			return
		}
		if result.More {
			_ = logger.Warning(observability.Event{Component: "history", Code: "history.retention.backlog"})
			return
		}
		_ = logger.Info(observability.Event{Component: "history", Code: "history.retention.completed"})
	}

	cleanup()
	ticker := time.NewTicker(history.RetentionRunInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanup()
		}
	}
}
