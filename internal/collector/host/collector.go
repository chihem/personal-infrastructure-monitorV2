package host

import (
	"context"
	"fmt"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/collector"
	hostcpu "github.com/chihem/personal-infrastructure-monitorV2/internal/collector/host/cpu"
	hostmemory "github.com/chihem/personal-infrastructure-monitorV2/internal/collector/host/memory"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

const (
	errorHostCollectionFailed = "host_collection_failed"
	errorHostPartial          = "host_collection_partial"
)

type Snapshot struct {
	CPU    *hostcpu.Snapshot
	Memory *hostmemory.Snapshot
}

func (snapshot Snapshot) Validate() error {
	if snapshot.CPU == nil && snapshot.Memory == nil {
		return fmt.Errorf("host snapshot requires CPU or memory evidence")
	}
	if snapshot.CPU != nil {
		if err := snapshot.CPU.Validate(); err != nil {
			return fmt.Errorf("CPU: %w", err)
		}
	}
	if snapshot.Memory != nil {
		if err := snapshot.Memory.Validate(); err != nil {
			return fmt.Errorf("memory: %w", err)
		}
	}
	return nil
}

func (snapshot Snapshot) Clone() Snapshot {
	var clone Snapshot
	if snapshot.CPU != nil {
		value := snapshot.CPU.Clone()
		clone.CPU = &value
	}
	if snapshot.Memory != nil {
		value := snapshot.Memory.Clone()
		clone.Memory = &value
	}
	return clone
}

type Collector struct {
	cpu    Provider
	memory Provider
}

func NewCollector(cpu Provider, memory Provider) (*Collector, error) {
	if cpu == nil || memory == nil {
		return nil, fmt.Errorf("CPU and memory providers are required")
	}
	return &Collector{cpu: cpu, memory: memory}, nil
}

func (value *Collector) Collect(ctx context.Context) collector.Result {
	cpuResult := value.cpu.Collect(ctx)
	memoryResult := value.memory.Collect(ctx)
	if err := cpuResult.Validate(); err != nil {
		cpuResult = collector.Failure(errorHostCollectionFailed)
	}
	if err := memoryResult.Validate(); err != nil {
		memoryResult = collector.Failure(errorHostCollectionFailed)
	}

	snapshot := Snapshot{}
	if cpuResult.Status != domain.CollectionFailed {
		cpuSnapshot, ok := cpuResult.Snapshot.(hostcpu.Snapshot)
		if !ok || cpuSnapshot.Validate() != nil {
			cpuResult = collector.Failure(errorHostCollectionFailed)
		} else {
			snapshot.CPU = &cpuSnapshot
		}
	}
	if memoryResult.Status != domain.CollectionFailed {
		memorySnapshot, ok := memoryResult.Snapshot.(hostmemory.Snapshot)
		if !ok || memorySnapshot.Validate() != nil {
			memoryResult = collector.Failure(errorHostCollectionFailed)
		} else {
			snapshot.Memory = &memorySnapshot
		}
	}

	if cpuResult.Status == domain.CollectionFailed && memoryResult.Status == domain.CollectionFailed {
		return collector.Failure(errorHostCollectionFailed)
	}
	if err := snapshot.Validate(); err != nil {
		return collector.Failure(errorHostCollectionFailed)
	}
	if cpuResult.Status == domain.CollectionSucceeded && memoryResult.Status == domain.CollectionSucceeded {
		return collector.Success(snapshot)
	}
	return collector.Partial(snapshot, errorHostPartial)
}

var _ Provider = (*Collector)(nil)
