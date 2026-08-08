package memory

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/collector"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

const (
	errorCollectionFailed    = "memory_collection_failed"
	errorMemoryPartial       = "memory_partial"
	errorPressureUnavailable = "memory_pressure_unavailable"
)

type Options struct {
	Source Source
	Now    func() time.Time
}

type Collector struct {
	source Source
	now    func() time.Time

	collectMu sync.Mutex
	mu        sync.RWMutex
	latest    *Snapshot
}

func New() *Collector {
	value, err := NewWithOptions(Options{})
	if err != nil {
		panic(err)
	}
	return value
}

func NewWithOptions(options Options) (*Collector, error) {
	if options.Source == nil {
		options.Source = procSource{}
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &Collector{source: options.Source, now: options.Now}, nil
}

func (value *Collector) Collect(ctx context.Context) collector.Result {
	value.collectMu.Lock()
	defer value.collectMu.Unlock()
	if err := ctx.Err(); err != nil {
		return collector.Failure(errorCollectionFailed)
	}
	memInfoData, err := value.source.MemInfo(ctx)
	if err != nil {
		return collector.Failure(errorCollectionFailed)
	}
	memInfo, memInfoErr := parseMemInfo(memInfoData)
	snapshot := buildMemorySnapshot(memInfo, value.now().UTC())
	if err := ctx.Err(); err != nil {
		return collector.Failure(errorCollectionFailed)
	}

	pressureData, pressureReadErr := value.source.Pressure(ctx)
	var pressureErr error
	if pressureReadErr != nil {
		pressureErr = pressureReadErr
		snapshot.Pressure = unavailablePressure(pressureReason(pressureReadErr))
	} else if pressure, parseErr := parsePressure(pressureData); parseErr != nil {
		pressureErr = parseErr
		snapshot.Pressure = unavailablePressure(domain.ReasonCollectorError)
	} else {
		snapshot.Pressure = availablePressure(pressure)
	}
	if err := snapshot.Validate(); err != nil {
		return collector.Failure(errorCollectionFailed)
	}
	value.mu.Lock()
	value.latest = snapshotPointer(snapshot.Clone())
	value.mu.Unlock()

	switch {
	case memInfoErr != nil:
		return collector.Partial(snapshot, errorMemoryPartial)
	case pressureErr != nil:
		return collector.Partial(snapshot, errorPressureUnavailable)
	default:
		return collector.Success(snapshot)
	}
}

func (value *Collector) Latest() (Snapshot, bool) {
	if value == nil {
		return Snapshot{}, false
	}
	value.mu.RLock()
	defer value.mu.RUnlock()
	if value.latest == nil {
		return Snapshot{}, false
	}
	return value.latest.Clone(), true
}

func buildMemorySnapshot(values map[string]int64, observedAt time.Time) Snapshot {
	reason := domain.ReasonCollectorError
	snapshot := Snapshot{
		ObservedAt: observedAt,
		Total:      integerFrom(values, "MemTotal", reason), Available: integerFrom(values, "MemAvailable", reason),
		Free: integerFrom(values, "MemFree", reason), Cached: integerFrom(values, "Cached", reason),
		Buffered: integerFrom(values, "Buffers", reason),
		Used:     unavailableInteger(domain.UnitBytes, reason), Usage: unavailableFloat(domain.UnitPercent, reason),
		Pressure: unavailablePressure(domain.ReasonNotCollected),
	}
	if total, totalOK := values["MemTotal"]; totalOK {
		if available, availableOK := values["MemAvailable"]; availableOK && total > 0 && available <= total {
			used := total - available
			snapshot.Used = availableInteger(used, domain.UnitBytes)
			snapshot.Usage = availableFloat(float64(used)/float64(total)*100, domain.UnitPercent)
		}
	}

	total, totalOK := values["SwapTotal"]
	free, freeOK := values["SwapFree"]
	switch {
	case totalOK && total == 0:
		configured := false
		snapshot.Swap = unavailableSwap(&configured, domain.ReasonNotConfigured)
	case totalOK && freeOK && total > 0 && free <= total:
		configured := true
		snapshot.Swap = SwapSnapshot{
			Configured: &configured, Total: availableInteger(total, domain.UnitBytes),
			Used: availableInteger(total-free, domain.UnitBytes), Free: availableInteger(free, domain.UnitBytes),
		}
	case totalOK && total > 0:
		configured := true
		snapshot.Swap = unavailableSwap(&configured, reason)
	default:
		snapshot.Swap = unavailableSwap(nil, reason)
	}
	return snapshot
}

func integerFrom(values map[string]int64, name string, reason domain.UnavailabilityReason) domain.Metric[int64] {
	if value, ok := values[name]; ok {
		return availableInteger(value, domain.UnitBytes)
	}
	return unavailableInteger(domain.UnitBytes, reason)
}

func unavailableSwap(configured *bool, reason domain.UnavailabilityReason) SwapSnapshot {
	return SwapSnapshot{
		Configured: configured, Total: unavailableInteger(domain.UnitBytes, reason),
		Used: unavailableInteger(domain.UnitBytes, reason), Free: unavailableInteger(domain.UnitBytes, reason),
	}
}

func availablePressure(value pressureValues) PressureSnapshot {
	return PressureSnapshot{
		Availability: domain.AvailabilityAvailable,
		Some:         pressureWindow(value.Some), Full: pressureWindow(value.Full),
	}
}

func pressureWindow(value pressureWindowValues) PressureWindow {
	return PressureWindow{
		Average10Seconds:  availableFloat(value.Average10, domain.UnitPercent),
		Average60Seconds:  availableFloat(value.Average60, domain.UnitPercent),
		Average300Seconds: availableFloat(value.Average300, domain.UnitPercent),
		Total:             availableInteger(value.Total, domain.UnitMicroseconds),
	}
}

func unavailablePressure(reason domain.UnavailabilityReason) PressureSnapshot {
	window := PressureWindow{
		Average10Seconds:  unavailableFloat(domain.UnitPercent, reason),
		Average60Seconds:  unavailableFloat(domain.UnitPercent, reason),
		Average300Seconds: unavailableFloat(domain.UnitPercent, reason),
		Total:             unavailableInteger(domain.UnitMicroseconds, reason),
	}
	return PressureSnapshot{Availability: domain.AvailabilityUnavailable, ReasonCode: &reason, Some: window, Full: clonePressureWindow(window)}
}

func pressureReason(err error) domain.UnavailabilityReason {
	switch {
	case errors.Is(err, os.ErrNotExist):
		return domain.ReasonNotSupported
	case errors.Is(err, os.ErrPermission):
		return domain.ReasonPermission
	default:
		return domain.ReasonCollectorError
	}
}

func snapshotPointer(snapshot Snapshot) *Snapshot { return &snapshot }

var _ interface {
	Collect(context.Context) collector.Result
} = (*Collector)(nil)
