package app

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/observability"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/storage/history"
)

func TestHistoryRetentionRunsImmediatelyAndStopsWithContext(t *testing.T) {
	t.Parallel()
	database, err := history.Open(context.Background(), filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatalf("open history: %v", err)
	}
	defer database.Close()
	repository, err := history.New(database)
	if err != nil {
		t.Fatalf("new history repository: %v", err)
	}

	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	oldRun := appCollectionRun(now.Add(-history.RetentionDuration-time.Minute), domain.CollectionSucceeded)
	if _, err := repository.RecordCollectionRun(context.Background(), oldRun); err != nil {
		t.Fatalf("record expired collection run: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		runHistoryRetention(ctx, repository, func() time.Time { return now }, observability.Discard())
	}()

	deadline := time.Now().Add(2 * time.Second)
	for {
		series, queryErr := repository.QueryMetricSeries(context.Background(), history.MetricQuery{
			Metric: history.MetricOverallCPUPercent,
			Range: domain.ResolvedRange{
				Preset: domain.RangeCustom,
				Start:  oldRun.StartedAt,
				End:    oldRun.StartedAt.Add(time.Minute),
			},
		})
		if queryErr != nil {
			cancel()
			t.Fatalf("query retained history: %v", queryErr)
		}
		if len(series.Points) == 1 && series.Points[0].State == history.MetricPointGap {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatalf("expired collection run was not removed: %+v", series.Points)
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("retention worker did not stop after cancellation")
	}
}
