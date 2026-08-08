package memory

import (
	"fmt"
	"math"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

type SwapSnapshot struct {
	Configured *bool
	Total      domain.Metric[int64]
	Used       domain.Metric[int64]
	Free       domain.Metric[int64]
}

type PressureWindow struct {
	Average10Seconds  domain.Metric[float64]
	Average60Seconds  domain.Metric[float64]
	Average300Seconds domain.Metric[float64]
	Total             domain.Metric[int64]
}

type PressureSnapshot struct {
	Availability domain.Availability
	ReasonCode   *domain.UnavailabilityReason
	Some         PressureWindow
	Full         PressureWindow
}

type Snapshot struct {
	ObservedAt time.Time
	Total      domain.Metric[int64]
	Used       domain.Metric[int64]
	Available  domain.Metric[int64]
	Free       domain.Metric[int64]
	Cached     domain.Metric[int64]
	Buffered   domain.Metric[int64]
	Usage      domain.Metric[float64]
	Swap       SwapSnapshot
	Pressure   PressureSnapshot
}

func (snapshot Snapshot) Validate() error {
	if err := domain.ValidateUTC(snapshot.ObservedAt); err != nil {
		return fmt.Errorf("observed time: %w", err)
	}
	for name, metric := range map[string]domain.Metric[int64]{
		"total": snapshot.Total, "used": snapshot.Used, "available": snapshot.Available,
		"free": snapshot.Free, "cached": snapshot.Cached, "buffered": snapshot.Buffered,
		"swap.total": snapshot.Swap.Total, "swap.used": snapshot.Swap.Used, "swap.free": snapshot.Swap.Free,
	} {
		if err := validateIntegerMetric(metric, domain.UnitBytes); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	if err := validateFloatMetric(snapshot.Usage, domain.UnitPercent, true); err != nil {
		return fmt.Errorf("usage: %w", err)
	}
	if !snapshot.Pressure.Availability.Valid() {
		return fmt.Errorf("pressure availability is invalid")
	}
	if snapshot.Pressure.Availability == domain.AvailabilityAvailable && snapshot.Pressure.ReasonCode != nil {
		return fmt.Errorf("available pressure cannot contain a reason")
	}
	if snapshot.Pressure.Availability == domain.AvailabilityUnavailable &&
		(snapshot.Pressure.ReasonCode == nil || !snapshot.Pressure.ReasonCode.Valid()) {
		return fmt.Errorf("unavailable pressure requires a valid reason")
	}
	for name, window := range map[string]PressureWindow{"some": snapshot.Pressure.Some, "full": snapshot.Pressure.Full} {
		for field, metric := range map[string]domain.Metric[float64]{
			"average10Seconds":  window.Average10Seconds,
			"average60Seconds":  window.Average60Seconds,
			"average300Seconds": window.Average300Seconds,
		} {
			if err := validateFloatMetric(metric, domain.UnitPercent, true); err != nil {
				return fmt.Errorf("pressure.%s.%s: %w", name, field, err)
			}
			if metric.Availability != snapshot.Pressure.Availability {
				return fmt.Errorf("pressure.%s.%s availability does not match pressure state", name, field)
			}
		}
		if err := validateIntegerMetric(window.Total, domain.UnitMicroseconds); err != nil {
			return fmt.Errorf("pressure.%s.total: %w", name, err)
		}
		if window.Total.Availability != snapshot.Pressure.Availability {
			return fmt.Errorf("pressure.%s.total availability does not match pressure state", name)
		}
	}
	return nil
}

func (snapshot Snapshot) Clone() Snapshot {
	clone := snapshot
	clone.Total = cloneIntegerMetric(snapshot.Total)
	clone.Used = cloneIntegerMetric(snapshot.Used)
	clone.Available = cloneIntegerMetric(snapshot.Available)
	clone.Free = cloneIntegerMetric(snapshot.Free)
	clone.Cached = cloneIntegerMetric(snapshot.Cached)
	clone.Buffered = cloneIntegerMetric(snapshot.Buffered)
	clone.Usage = cloneFloatMetric(snapshot.Usage)
	clone.Swap.Total = cloneIntegerMetric(snapshot.Swap.Total)
	clone.Swap.Used = cloneIntegerMetric(snapshot.Swap.Used)
	clone.Swap.Free = cloneIntegerMetric(snapshot.Swap.Free)
	if snapshot.Swap.Configured != nil {
		configured := *snapshot.Swap.Configured
		clone.Swap.Configured = &configured
	}
	clone.Pressure.ReasonCode = cloneReason(snapshot.Pressure.ReasonCode)
	clone.Pressure.Some = clonePressureWindow(snapshot.Pressure.Some)
	clone.Pressure.Full = clonePressureWindow(snapshot.Pressure.Full)
	return clone
}

func validateIntegerMetric(metric domain.Metric[int64], unit domain.Unit) error {
	if err := metric.Validate(); err != nil {
		return err
	}
	if metric.Unit != unit {
		return fmt.Errorf("unit must be %s", unit)
	}
	if metric.Value != nil && *metric.Value < 0 {
		return fmt.Errorf("value must be non-negative")
	}
	return nil
}

func validateFloatMetric(metric domain.Metric[float64], unit domain.Unit, boundedPercent bool) error {
	if err := metric.Validate(); err != nil {
		return err
	}
	if metric.Unit != unit {
		return fmt.Errorf("unit must be %s", unit)
	}
	if metric.Value != nil && (math.IsNaN(*metric.Value) || math.IsInf(*metric.Value, 0) || *metric.Value < 0) {
		return fmt.Errorf("value must be finite and non-negative")
	}
	if boundedPercent && metric.Value != nil && *metric.Value > 100 {
		return fmt.Errorf("percentage cannot exceed 100")
	}
	return nil
}

func availableInteger(value int64, unit domain.Unit) domain.Metric[int64] {
	return domain.Metric[int64]{Availability: domain.AvailabilityAvailable, Value: &value, Unit: unit}
}

func unavailableInteger(unit domain.Unit, reason domain.UnavailabilityReason) domain.Metric[int64] {
	return domain.Metric[int64]{Availability: domain.AvailabilityUnavailable, Unit: unit, ReasonCode: &reason}
}

func availableFloat(value float64, unit domain.Unit) domain.Metric[float64] {
	return domain.Metric[float64]{Availability: domain.AvailabilityAvailable, Value: &value, Unit: unit}
}

func unavailableFloat(unit domain.Unit, reason domain.UnavailabilityReason) domain.Metric[float64] {
	return domain.Metric[float64]{Availability: domain.AvailabilityUnavailable, Unit: unit, ReasonCode: &reason}
}

func cloneIntegerMetric(metric domain.Metric[int64]) domain.Metric[int64] {
	clone := metric
	if metric.Value != nil {
		value := *metric.Value
		clone.Value = &value
	}
	clone.ReasonCode = cloneReason(metric.ReasonCode)
	return clone
}

func cloneFloatMetric(metric domain.Metric[float64]) domain.Metric[float64] {
	clone := metric
	if metric.Value != nil {
		value := *metric.Value
		clone.Value = &value
	}
	clone.ReasonCode = cloneReason(metric.ReasonCode)
	return clone
}

func cloneReason(reason *domain.UnavailabilityReason) *domain.UnavailabilityReason {
	if reason == nil {
		return nil
	}
	value := *reason
	return &value
}

func clonePressureWindow(window PressureWindow) PressureWindow {
	return PressureWindow{
		Average10Seconds:  cloneFloatMetric(window.Average10Seconds),
		Average60Seconds:  cloneFloatMetric(window.Average60Seconds),
		Average300Seconds: cloneFloatMetric(window.Average300Seconds),
		Total:             cloneIntegerMetric(window.Total),
	}
}
