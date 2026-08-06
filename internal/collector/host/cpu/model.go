package cpu

import (
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

type CoreSnapshot struct {
	LogicalIndex int
	Usage        domain.Metric[float64]
}

type LoadSnapshot struct {
	OneMinute      domain.Metric[float64]
	FiveMinutes    domain.Metric[float64]
	FifteenMinutes domain.Metric[float64]
}

type Snapshot struct {
	ObservedAt      time.Time
	Overall         domain.Metric[float64]
	Cores           []CoreSnapshot
	Load            LoadSnapshot
	LogicalCPUCount int
}

func (snapshot Snapshot) Validate() error {
	if err := domain.ValidateUTC(snapshot.ObservedAt); err != nil {
		return fmt.Errorf("observed time: %w", err)
	}
	if err := validatePercentMetric(snapshot.Overall); err != nil {
		return fmt.Errorf("overall usage: %w", err)
	}
	if snapshot.Cores == nil {
		return fmt.Errorf("CPU cores must be an array")
	}
	if snapshot.LogicalCPUCount != len(snapshot.Cores) {
		return fmt.Errorf("logical CPU count must match reported cores")
	}
	seen := make(map[int]struct{}, len(snapshot.Cores))
	previousIndex := -1
	for index, core := range snapshot.Cores {
		if core.LogicalIndex < 0 {
			return fmt.Errorf("core %d has a negative logical index", index)
		}
		if _, duplicate := seen[core.LogicalIndex]; duplicate {
			return fmt.Errorf("duplicate logical CPU index %d", core.LogicalIndex)
		}
		if core.LogicalIndex <= previousIndex {
			return fmt.Errorf("logical CPU indexes must be strictly ordered")
		}
		seen[core.LogicalIndex] = struct{}{}
		previousIndex = core.LogicalIndex
		if err := validatePercentMetric(core.Usage); err != nil {
			return fmt.Errorf("core %d usage: %w", core.LogicalIndex, err)
		}
	}
	for name, metric := range map[string]domain.Metric[float64]{
		"one-minute": snapshot.Load.OneMinute, "five-minute": snapshot.Load.FiveMinutes,
		"fifteen-minute": snapshot.Load.FifteenMinutes,
	} {
		if err := validateLoadMetric(metric); err != nil {
			return fmt.Errorf("%s load: %w", name, err)
		}
	}
	return nil
}

func (snapshot Snapshot) Clone() Snapshot {
	clone := snapshot
	clone.Overall = cloneCPUFloatMetric(snapshot.Overall)
	clone.Load.OneMinute = cloneCPUFloatMetric(snapshot.Load.OneMinute)
	clone.Load.FiveMinutes = cloneCPUFloatMetric(snapshot.Load.FiveMinutes)
	clone.Load.FifteenMinutes = cloneCPUFloatMetric(snapshot.Load.FifteenMinutes)
	clone.Cores = slices.Clone(snapshot.Cores)
	for index := range clone.Cores {
		clone.Cores[index].Usage = cloneCPUFloatMetric(clone.Cores[index].Usage)
	}
	return clone
}

func cloneCPUFloatMetric(metric domain.Metric[float64]) domain.Metric[float64] {
	clone := metric
	if metric.Value != nil {
		value := *metric.Value
		clone.Value = &value
	}
	if metric.ReasonCode != nil {
		reason := *metric.ReasonCode
		clone.ReasonCode = &reason
	}
	return clone
}

func validatePercentMetric(metric domain.Metric[float64]) error {
	if err := metric.Validate(); err != nil {
		return err
	}
	if metric.Unit != domain.UnitPercent {
		return fmt.Errorf("unit must be percent")
	}
	if metric.Value != nil && (math.IsNaN(*metric.Value) || math.IsInf(*metric.Value, 0) || *metric.Value < 0 || *metric.Value > 100) {
		return fmt.Errorf("value must be finite and between 0 and 100")
	}
	return nil
}

func validateLoadMetric(metric domain.Metric[float64]) error {
	if err := metric.Validate(); err != nil {
		return err
	}
	if metric.Unit != domain.UnitLoad {
		return fmt.Errorf("unit must be load")
	}
	if metric.Value != nil && (math.IsNaN(*metric.Value) || math.IsInf(*metric.Value, 0) || *metric.Value < 0) {
		return fmt.Errorf("value must be finite and non-negative")
	}
	return nil
}

func available(value float64, unit domain.Unit) domain.Metric[float64] {
	return domain.Metric[float64]{Availability: domain.AvailabilityAvailable, Value: &value, Unit: unit}
}

func unavailable(unit domain.Unit, reason domain.UnavailabilityReason) domain.Metric[float64] {
	return domain.Metric[float64]{Availability: domain.AvailabilityUnavailable, Unit: unit, ReasonCode: &reason}
}
