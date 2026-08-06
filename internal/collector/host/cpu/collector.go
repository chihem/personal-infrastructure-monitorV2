package cpu

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/collector"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

const (
	errorCollectionFailed = "cpu_collection_failed"
	errorBaselinePending  = "cpu_baseline_pending"
	errorCounterInvalid   = "cpu_counter_invalid"
	errorLoadUnavailable  = "cpu_load_unavailable"
	errorPartial          = "cpu_partial"
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
	previous  []Counter
	latest    *Snapshot
}

func New() *Collector {
	value, err := NewWithOptions(Options{Source: gopsutilSource{}, Now: time.Now})
	if err != nil {
		panic(err)
	}
	return value
}

func NewWithOptions(options Options) (*Collector, error) {
	if options.Source == nil {
		options.Source = gopsutilSource{}
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
	counters, counterErr := value.source.Counters(ctx)
	loadAverage, loadErr := value.source.LoadAverage(ctx)
	if counterErr != nil {
		return collector.Failure(errorCollectionFailed)
	}
	counters, counterErr = normalizeCounters(counters)
	if counterErr != nil {
		return collector.Failure(errorCollectionFailed)
	}
	if loadErr == nil {
		loadErr = loadAverage.validate()
	}

	value.mu.Lock()
	defer value.mu.Unlock()
	snapshot, calculationErr := buildSnapshot(value.previous, counters, loadAverage, loadErr, value.now().UTC())
	value.previous = cloneCounters(counters)
	if err := snapshot.Validate(); err != nil {
		return collector.Failure(errorCollectionFailed)
	}
	value.latest = snapshotPointer(snapshot.Clone())

	switch {
	case calculationErr == nil && loadErr == nil:
		return collector.Success(snapshot)
	case errors.Is(calculationErr, errBaselineRequired) && loadErr == nil:
		return collector.Partial(snapshot, errorBaselinePending)
	case calculationErr == nil && loadErr != nil:
		return collector.Partial(snapshot, errorLoadUnavailable)
	case loadErr == nil:
		return collector.Partial(snapshot, errorCounterInvalid)
	default:
		return collector.Partial(snapshot, errorPartial)
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

var errBaselineRequired = errors.New("CPU usage baseline is required")

func buildSnapshot(
	previous []Counter,
	current []Counter,
	loadAverage LoadAverage,
	loadErr error,
	observedAt time.Time,
) (Snapshot, error) {
	previousByIndex := make(map[int]Counter, len(previous))
	for _, counter := range previous {
		previousByIndex[counter.LogicalIndex] = counter
	}

	reasonNotCollected := domain.ReasonNotCollected
	reasonCollectorError := domain.ReasonCollectorError
	snapshot := Snapshot{
		ObservedAt: observedAt, LogicalCPUCount: len(current),
		Overall: unavailable(domain.UnitPercent, reasonNotCollected),
		Cores:   make([]CoreSnapshot, 0, len(current)),
	}
	var calculationErr error
	coreCalculationInvalid := false
	topologyStable := len(previous) == len(current) && len(previous) > 0
	for _, counter := range current {
		metric := unavailable(domain.UnitPercent, reasonNotCollected)
		prior, present := previousByIndex[counter.LogicalIndex]
		if !present {
			topologyStable = false
		} else if usage, err := calculateUsage(prior, counter); err == nil {
			metric = available(usage, domain.UnitPercent)
		} else {
			metric = unavailable(domain.UnitPercent, reasonCollectorError)
			coreCalculationInvalid = true
			calculationErr = errors.Join(calculationErr, err)
		}
		snapshot.Cores = append(snapshot.Cores, CoreSnapshot{LogicalIndex: counter.LogicalIndex, Usage: metric})
	}
	if len(previous) == 0 {
		calculationErr = errBaselineRequired
	} else if coreCalculationInvalid {
		snapshot.Overall = unavailable(domain.UnitPercent, reasonCollectorError)
	} else if topologyStable {
		usage, err := calculateUsage(aggregateCounters(previous), aggregateCounters(current))
		if err == nil {
			snapshot.Overall = available(usage, domain.UnitPercent)
		} else {
			snapshot.Overall = unavailable(domain.UnitPercent, reasonCollectorError)
			calculationErr = errors.Join(calculationErr, err)
		}
	} else {
		calculationErr = errors.Join(calculationErr, errBaselineRequired)
	}

	if loadErr == nil {
		snapshot.Load = LoadSnapshot{
			OneMinute: available(loadAverage.One, domain.UnitLoad), FiveMinutes: available(loadAverage.Five, domain.UnitLoad),
			FifteenMinutes: available(loadAverage.Fifteen, domain.UnitLoad),
		}
	} else {
		snapshot.Load = LoadSnapshot{
			OneMinute: unavailable(domain.UnitLoad, reasonCollectorError), FiveMinutes: unavailable(domain.UnitLoad, reasonCollectorError),
			FifteenMinutes: unavailable(domain.UnitLoad, reasonCollectorError),
		}
	}
	return snapshot, calculationErr
}

func cloneCounters(counters []Counter) []Counter {
	return append([]Counter(nil), counters...)
}

func snapshotPointer(snapshot Snapshot) *Snapshot {
	return &snapshot
}

var _ interface {
	Collect(context.Context) collector.Result
} = (*Collector)(nil)
