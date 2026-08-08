package app

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/collector/host"
	hostcpu "github.com/chihem/personal-infrastructure-monitorV2/internal/collector/host/cpu"
	hostmemory "github.com/chihem/personal-infrastructure-monitorV2/internal/collector/host/memory"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/scheduler"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/storage/history"
)

func TestCollectionRecorderStoresAvailableAndUnavailableCPUEvidence(t *testing.T) {
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

	at := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	overall, load, core := 55.0, 0.75, 60.0
	notCollected := domain.ReasonNotCollected
	snapshot := hostcpu.Snapshot{
		ObservedAt: at,
		Overall:    testAvailableMetric(overall, domain.UnitPercent),
		Cores: []hostcpu.CoreSnapshot{
			{LogicalIndex: 0, Usage: testAvailableMetric(core, domain.UnitPercent)},
			{LogicalIndex: 4, Usage: testUnavailableMetric(domain.UnitPercent, notCollected)},
		},
		Load: hostcpu.LoadSnapshot{
			OneMinute:      testAvailableMetric(load, domain.UnitLoad),
			FiveMinutes:    testAvailableMetric(load, domain.UnitLoad),
			FifteenMinutes: testAvailableMetric(load, domain.UnitLoad),
		},
		LogicalCPUCount: 2,
	}
	memoryUsage := 27.3
	memoryBytes := int64(11 << 30)
	psiReason := domain.ReasonNotSupported
	swapConfigured := false
	pressureUnavailable := hostmemory.PressureWindow{
		Average10Seconds:  testUnavailableMetric(domain.UnitPercent, domain.ReasonNotSupported),
		Average60Seconds:  testUnavailableMetric(domain.UnitPercent, domain.ReasonNotSupported),
		Average300Seconds: testUnavailableMetric(domain.UnitPercent, domain.ReasonNotSupported),
		Total:             testUnavailableIntegerMetric(domain.UnitMicroseconds, domain.ReasonNotSupported),
	}
	memorySnapshot := hostmemory.Snapshot{
		ObservedAt: at, Total: testAvailableIntegerMetric(memoryBytes, domain.UnitBytes),
		Used: testAvailableIntegerMetric(memoryBytes, domain.UnitBytes), Available: testAvailableIntegerMetric(memoryBytes, domain.UnitBytes),
		Free: testAvailableIntegerMetric(memoryBytes, domain.UnitBytes), Cached: testAvailableIntegerMetric(memoryBytes, domain.UnitBytes),
		Buffered: testAvailableIntegerMetric(memoryBytes, domain.UnitBytes), Usage: testAvailableMetric(memoryUsage, domain.UnitPercent),
		Swap: hostmemory.SwapSnapshot{
			Configured: &swapConfigured,
			Total:      testUnavailableIntegerMetric(domain.UnitBytes, domain.ReasonNotConfigured),
			Used:       testUnavailableIntegerMetric(domain.UnitBytes, domain.ReasonNotConfigured),
			Free:       testUnavailableIntegerMetric(domain.UnitBytes, domain.ReasonNotConfigured),
		},
		Pressure: hostmemory.PressureSnapshot{
			Availability: domain.AvailabilityUnavailable, ReasonCode: &psiReason,
			Some: pressureUnavailable, Full: pressureUnavailable,
		},
	}
	completed := scheduler.CompletedRun{
		Record: appCollectionRun(at, domain.CollectionPartial), HostSnapshot: host.Snapshot{CPU: &snapshot, Memory: &memorySnapshot},
	}
	completed.Record.DockerResult.Status = domain.CollectionFailed
	completed.Record.DockerResult.ErrorCode = "docker_not_implemented"

	if err := (collectionRecorder{history: repository}).RecordCollection(context.Background(), completed); err != nil {
		t.Fatalf("RecordCollection() error = %v", err)
	}
	series, err := repository.QueryMetricSeries(context.Background(), history.MetricQuery{
		Metric: history.MetricCPUCoreUsagePercent, ResourceID: "4",
		Range: domain.ResolvedRange{Preset: domain.RangeLastMinute, Start: at, End: at.Add(time.Minute)},
	})
	if err != nil {
		t.Fatalf("QueryMetricSeries() error = %v", err)
	}
	if len(series.Points) != 1 || series.Points[0].State != history.MetricPointUnavailable {
		t.Fatalf("unavailable core history = %+v", series.Points)
	}
	memorySeries, err := repository.QueryMetricSeries(context.Background(), history.MetricQuery{
		Metric: history.MetricMemoryUsagePercent,
		Range:  domain.ResolvedRange{Preset: domain.RangeLastMinute, Start: at, End: at.Add(time.Minute)},
	})
	if err != nil || len(memorySeries.Points) != 1 || memorySeries.Points[0].Average == nil || *memorySeries.Points[0].Average != memoryUsage {
		t.Fatalf("memory history = %+v, error = %v", memorySeries.Points, err)
	}
}

func appCollectionRun(at time.Time, status domain.CollectionStatus) domain.CollectionRun {
	finished := at.Add(time.Second)
	return domain.CollectionRun{
		StartedAt: at, FinishedAt: finished, Trigger: domain.CollectionTriggerScheduled, Status: status,
		HostResult: domain.CollectionOutcome{
			Subsystem: domain.CollectionSubsystemHost, Status: domain.CollectionSucceeded,
			StartedAt: at, FinishedAt: finished,
		},
		DockerResult: domain.CollectionOutcome{
			Subsystem: domain.CollectionSubsystemDocker, Status: domain.CollectionSucceeded,
			StartedAt: at, FinishedAt: finished,
		},
	}
}

func testAvailableMetric(value float64, unit domain.Unit) domain.Metric[float64] {
	return domain.Metric[float64]{Availability: domain.AvailabilityAvailable, Value: &value, Unit: unit}
}

func testUnavailableMetric(unit domain.Unit, reason domain.UnavailabilityReason) domain.Metric[float64] {
	return domain.Metric[float64]{Availability: domain.AvailabilityUnavailable, Unit: unit, ReasonCode: &reason}
}

func testAvailableIntegerMetric(value int64, unit domain.Unit) domain.Metric[int64] {
	return domain.Metric[int64]{Availability: domain.AvailabilityAvailable, Value: &value, Unit: unit}
}

func testUnavailableIntegerMetric(unit domain.Unit, reason domain.UnavailabilityReason) domain.Metric[int64] {
	return domain.Metric[int64]{Availability: domain.AvailabilityUnavailable, Unit: unit, ReasonCode: &reason}
}
