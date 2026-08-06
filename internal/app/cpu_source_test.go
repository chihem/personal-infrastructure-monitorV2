package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/api/contracts"
	hostcpu "github.com/chihem/personal-infrastructure-monitorV2/internal/collector/host/cpu"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/storage/history"
)

func TestCPUDataSourceReturnsUnavailableBeforeCollectionAndStaleAfterBoundary(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	collectorValue, err := hostcpu.NewWithOptions(hostcpu.Options{Source: &appCPUSourceStub{}, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("new CPU collector: %v", err)
	}
	source := cpuDataSource{collector: collectorValue, now: func() time.Time { return now }, staleAfter: 2 * time.Minute}
	before := source.CurrentCPU()
	if before.Freshness.State != domain.FreshnessUnavailable || before.Cores == nil {
		t.Fatalf("before collection = %+v", before)
	}

	collectorValue.Collect(context.Background())
	fresh := source.CurrentCPU()
	if fresh.Freshness.State != domain.FreshnessFresh || fresh.LogicalCPU != 1 {
		t.Fatalf("fresh snapshot = %+v", fresh)
	}
	now = now.Add(2 * time.Minute)
	stale := source.CurrentCPU()
	if stale.Freshness.State != domain.FreshnessStale {
		t.Fatalf("stale snapshot = %+v", stale)
	}
}

func TestCPUDataSourceClassifiesOutOfRetentionRange(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	start := now.Add(-history.RetentionDuration - time.Minute)
	end := now.Add(-history.RetentionDuration)
	collectorValue, _ := hostcpu.NewWithOptions(hostcpu.Options{Source: &appCPUSourceStub{}, Now: time.Now})
	source := cpuDataSource{collector: collectorValue, now: func() time.Time { return now }, staleAfter: 2 * time.Minute}
	_, err := source.CPUHistory(context.Background(), contracts.CPUHistoryRequest{
		Metric: contracts.CPUMetricOverall,
		Range:  domain.RangeSelection{Preset: domain.RangeCustom, Start: &start, End: &end},
	})
	if !errors.Is(err, contracts.ErrInvalidCPUHistoryRange) {
		t.Fatalf("CPUHistory() error = %v", err)
	}
}

func TestCPUDataSourceMapsBoundedHistory(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 5, 12, 0, 30, 0, time.UTC)
	database, err := history.Open(context.Background(), filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatalf("open history: %v", err)
	}
	defer database.Close()
	repository, err := history.New(database)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	observedAt := now.UTC().Truncate(time.Minute).Add(-time.Minute)
	value := 42.0
	psiReason := domain.ReasonNotCollected
	_, err = repository.RecordCollection(context.Background(), history.CollectionRecord{
		Run: appCollectionRun(observedAt, domain.CollectionSucceeded),
		Host: &history.HostSampleRecord{
			ObservedAt: observedAt, Availability: domain.AvailabilityAvailable, OverallCPUPercent: &value,
			PSIAvailability: domain.AvailabilityUnavailable, PSIUnavailableReason: &psiReason,
		},
	})
	if err != nil {
		t.Fatalf("record history: %v", err)
	}
	collectorValue, _ := hostcpu.NewWithOptions(hostcpu.Options{Source: &appCPUSourceStub{}, Now: time.Now})
	source := cpuDataSource{collector: collectorValue, history: repository, now: func() time.Time { return now }, staleAfter: 2 * time.Minute}
	series, err := source.CPUHistory(context.Background(), contracts.CPUHistoryRequest{
		Metric: contracts.CPUMetricOverall, Range: domain.RangeSelection{Preset: domain.RangeLast5Minutes},
	})
	if err != nil {
		t.Fatalf("CPUHistory() error = %v", err)
	}
	if len(series.Points) != 5 || series.BucketDurationSeconds != 60 {
		t.Fatalf("series shape = %+v", series)
	}
	observed := 0
	for _, point := range series.Points {
		if point.State == contracts.CPUHistoryObserved {
			observed++
		}
	}
	if observed != 1 {
		t.Fatalf("observed buckets = %d, series = %+v", observed, series)
	}
}

type appCPUSourceStub struct{}

func (*appCPUSourceStub) Counters(context.Context) ([]hostcpu.Counter, error) {
	return []hostcpu.Counter{{LogicalIndex: 0, User: 10, Idle: 90}}, nil
}

func (*appCPUSourceStub) LoadAverage(context.Context) (hostcpu.LoadAverage, error) {
	return hostcpu.LoadAverage{One: 0.1, Five: 0.2, Fifteen: 0.3}, nil
}
