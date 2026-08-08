package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/api/contracts"
	hostmemory "github.com/chihem/personal-infrastructure-monitorV2/internal/collector/host/memory"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/storage/history"
)

func TestMemoryDataSourceReturnsUnavailableFreshPartialAndStaleStates(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	collectorValue, err := hostmemory.NewWithOptions(hostmemory.Options{
		Source: &appMemorySourceStub{pressureErr: os.ErrNotExist}, Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("new memory collector: %v", err)
	}
	source := memoryDataSource{collector: collectorValue, now: func() time.Time { return now }, staleAfter: 2 * time.Minute}
	before := source.CurrentMemory()
	if before.Freshness.State != domain.FreshnessUnavailable || before.Total.Value != nil {
		t.Fatalf("before collection = %+v", before)
	}

	collectorValue.Collect(context.Background())
	fresh := source.CurrentMemory()
	if fresh.Freshness.State != domain.FreshnessFresh || fresh.Total.Value == nil ||
		fresh.Pressure.Some.Average10Seconds.Availability != domain.AvailabilityUnavailable {
		t.Fatalf("fresh partial snapshot = %+v", fresh)
	}
	now = now.Add(2 * time.Minute)
	if stale := source.CurrentMemory(); stale.Freshness.State != domain.FreshnessStale {
		t.Fatalf("stale snapshot = %+v", stale)
	}
}

func TestMemoryDataSourceMapsHistoryAndRejectsExpiredCustomRange(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 6, 12, 0, 30, 0, time.UTC)
	database, err := history.Open(context.Background(), filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatalf("open history: %v", err)
	}
	defer database.Close()
	repository, err := history.New(database)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	observedAt := now.Truncate(time.Minute).Add(-time.Minute)
	usage := 42.0
	psiReason := domain.ReasonNotSupported
	_, err = repository.RecordCollection(context.Background(), history.CollectionRecord{
		Run: appCollectionRun(observedAt, domain.CollectionSucceeded),
		Host: &history.HostSampleRecord{
			ObservedAt: observedAt, Availability: domain.AvailabilityAvailable, MemoryUsagePercent: &usage,
			PSIAvailability: domain.AvailabilityUnavailable, PSIUnavailableReason: &psiReason,
		},
	})
	if err != nil {
		t.Fatalf("record history: %v", err)
	}
	collectorValue, _ := hostmemory.NewWithOptions(hostmemory.Options{Source: &appMemorySourceStub{}, Now: time.Now})
	source := memoryDataSource{collector: collectorValue, history: repository, now: func() time.Time { return now }, staleAfter: 2 * time.Minute}
	series, err := source.MemoryHistory(context.Background(), contracts.MemoryHistoryRequest{
		Metric: contracts.MemoryMetricUsage, Range: domain.RangeSelection{Preset: domain.RangeLast5Minutes},
	})
	if err != nil || len(series.Points) != 5 {
		t.Fatalf("history series = %+v, error = %v", series, err)
	}
	observed := 0
	for _, point := range series.Points {
		if point.State == contracts.MemoryHistoryObserved {
			observed++
		}
	}
	if observed != 1 {
		t.Fatalf("observed buckets = %d", observed)
	}

	start := now.Add(-history.RetentionDuration - time.Minute)
	end := start.Add(time.Minute)
	_, err = source.MemoryHistory(context.Background(), contracts.MemoryHistoryRequest{
		Metric: contracts.MemoryMetricUsage,
		Range:  domain.RangeSelection{Preset: domain.RangeCustom, Start: &start, End: &end},
	})
	if !errors.Is(err, contracts.ErrInvalidMemoryHistoryRange) {
		t.Fatalf("expired range error = %v", err)
	}
}

type appMemorySourceStub struct{ pressureErr error }

func (*appMemorySourceStub) MemInfo(context.Context) ([]byte, error) {
	return []byte("MemTotal: 11264 kB\nMemFree: 2048 kB\nMemAvailable: 8192 kB\nBuffers: 256 kB\nCached: 5120 kB\nSwapTotal: 0 kB\nSwapFree: 0 kB\n"), nil
}

func (source *appMemorySourceStub) Pressure(context.Context) ([]byte, error) {
	if source.pressureErr != nil {
		return nil, source.pressureErr
	}
	return []byte("some avg10=0.2 avg60=0.1 avg300=0.0 total=10\nfull avg10=0 avg60=0 avg300=0 total=0\n"), nil
}
