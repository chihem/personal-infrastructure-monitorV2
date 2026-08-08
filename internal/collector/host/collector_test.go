package host

import (
	"context"
	"testing"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/collector"
	hostcpu "github.com/chihem/personal-infrastructure-monitorV2/internal/collector/host/cpu"
	hostmemory "github.com/chihem/personal-infrastructure-monitorV2/internal/collector/host/memory"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

func TestCompositeCollectorKeepsSuccessfulSubsystemEvidence(t *testing.T) {
	t.Parallel()
	cpuSnapshot := validCPUSnapshot()
	memorySnapshot := validMemorySnapshot()
	tests := []struct {
		name       string
		cpu        collector.Result
		memory     collector.Result
		wantStatus domain.CollectionStatus
		wantCPU    bool
		wantMemory bool
	}{
		{"all available", collector.Success(cpuSnapshot), collector.Success(memorySnapshot), domain.CollectionSucceeded, true, true},
		{"pressure partial", collector.Success(cpuSnapshot), collector.Partial(memorySnapshot, "memory_pressure_unavailable"), domain.CollectionPartial, true, true},
		{"CPU failed", collector.Failure("cpu_collection_failed"), collector.Success(memorySnapshot), domain.CollectionPartial, false, true},
		{"all failed", collector.Failure("cpu_collection_failed"), collector.Failure("memory_collection_failed"), domain.CollectionFailed, false, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, err := NewCollector(providerResult{test.cpu}, providerResult{test.memory})
			if err != nil {
				t.Fatalf("NewCollector() error = %v", err)
			}
			result := value.Collect(context.Background())
			if result.Status != test.wantStatus {
				t.Fatalf("status = %q, want %q", result.Status, test.wantStatus)
			}
			if result.Status == domain.CollectionFailed {
				if result.Snapshot != nil {
					t.Fatal("failed host collection contained a snapshot")
				}
				return
			}
			snapshot := result.Snapshot.(Snapshot)
			if (snapshot.CPU != nil) != test.wantCPU || (snapshot.Memory != nil) != test.wantMemory {
				t.Fatalf("snapshot = %+v", snapshot)
			}
		})
	}
}

type providerResult struct{ result collector.Result }

func (provider providerResult) Collect(context.Context) collector.Result { return provider.result }

func validCPUSnapshot() hostcpu.Snapshot {
	at := testObservedAt()
	percent, load := 10.0, 0.1
	return hostcpu.Snapshot{
		ObservedAt: at, Overall: availableFloatMetric(percent, domain.UnitPercent),
		Cores: []hostcpu.CoreSnapshot{{LogicalIndex: 0, Usage: availableFloatMetric(percent, domain.UnitPercent)}},
		Load: hostcpu.LoadSnapshot{
			OneMinute: availableFloatMetric(load, domain.UnitLoad), FiveMinutes: availableFloatMetric(load, domain.UnitLoad),
			FifteenMinutes: availableFloatMetric(load, domain.UnitLoad),
		},
		LogicalCPUCount: 1,
	}
}

func validMemorySnapshot() hostmemory.Snapshot {
	value := int64(1024)
	usage, pressure := 50.0, 0.0
	byteMetric := availableIntegerMetric(value, domain.UnitBytes)
	pressureWindow := hostmemory.PressureWindow{
		Average10Seconds:  availableFloatMetric(pressure, domain.UnitPercent),
		Average60Seconds:  availableFloatMetric(pressure, domain.UnitPercent),
		Average300Seconds: availableFloatMetric(pressure, domain.UnitPercent),
		Total:             availableIntegerMetric(0, domain.UnitMicroseconds),
	}
	notConfigured := domain.ReasonNotConfigured
	swapConfigured := false
	return hostmemory.Snapshot{
		ObservedAt: testObservedAt(), Total: byteMetric, Used: byteMetric, Available: byteMetric,
		Free: byteMetric, Cached: byteMetric, Buffered: byteMetric, Usage: availableFloatMetric(usage, domain.UnitPercent),
		Swap: hostmemory.SwapSnapshot{
			Configured: &swapConfigured,
			Total:      unavailableIntegerMetric(domain.UnitBytes, notConfigured), Used: unavailableIntegerMetric(domain.UnitBytes, notConfigured),
			Free: unavailableIntegerMetric(domain.UnitBytes, notConfigured),
		},
		Pressure: hostmemory.PressureSnapshot{
			Availability: domain.AvailabilityAvailable, Some: pressureWindow, Full: pressureWindow,
		},
	}
}

func availableFloatMetric(value float64, unit domain.Unit) domain.Metric[float64] {
	return domain.Metric[float64]{Availability: domain.AvailabilityAvailable, Value: &value, Unit: unit}
}

func availableIntegerMetric(value int64, unit domain.Unit) domain.Metric[int64] {
	return domain.Metric[int64]{Availability: domain.AvailabilityAvailable, Value: &value, Unit: unit}
}

func unavailableIntegerMetric(unit domain.Unit, reason domain.UnavailabilityReason) domain.Metric[int64] {
	return domain.Metric[int64]{Availability: domain.AvailabilityUnavailable, Unit: unit, ReasonCode: &reason}
}

func testObservedAt() time.Time { return time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC) }
