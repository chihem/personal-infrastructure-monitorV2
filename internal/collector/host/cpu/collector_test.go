package cpu

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

func TestCollectorKeepsFirstUsageUnavailableThenCalculatesDeltas(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	source := &sourceStub{
		counterResults: [][]Counter{
			{counter(0, 10, 90), counter(1, 20, 80)},
			{counter(0, 20, 100), counter(1, 30, 90)},
		},
		loadResults: []LoadAverage{{One: 0.5, Five: 0.4, Fifteen: 0.3}, {One: 0.6, Five: 0.5, Fifteen: 0.4}},
	}
	collectorValue := newTestCollector(t, source, func() time.Time { return now })

	first := collectorValue.Collect(context.Background())
	firstSnapshot := requireSnapshot(t, first.Snapshot)
	if first.Status != domain.CollectionPartial || first.ErrorCode != errorBaselinePending {
		t.Fatalf("first result = %+v", first)
	}
	assertUnavailable(t, firstSnapshot.Overall, domain.ReasonNotCollected)
	for _, core := range firstSnapshot.Cores {
		assertUnavailable(t, core.Usage, domain.ReasonNotCollected)
	}
	if firstSnapshot.Load.OneMinute.Value == nil || *firstSnapshot.Load.OneMinute.Value != 0.5 {
		t.Fatalf("first load = %+v", firstSnapshot.Load)
	}

	now = now.Add(time.Minute)
	second := collectorValue.Collect(context.Background())
	secondSnapshot := requireSnapshot(t, second.Snapshot)
	if second.Status != domain.CollectionSucceeded || second.ErrorCode != "" {
		t.Fatalf("second result = %+v", second)
	}
	assertAvailableNear(t, secondSnapshot.Overall, 50)
	assertAvailableNear(t, secondSnapshot.Cores[0].Usage, 50)
	assertAvailableNear(t, secondSnapshot.Cores[1].Usage, 50)
	latest, ok := collectorValue.Latest()
	if !ok || !latest.ObservedAt.Equal(now) {
		t.Fatalf("Latest() = %+v, available = %v", latest, ok)
	}
}

func TestCollectorHandlesTopologyChangesWithoutInventingOverallUsage(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	source := &sourceStub{
		counterResults: [][]Counter{
			{counter(0, 10, 90)},
			{counter(0, 20, 100), counter(3, 1, 9)},
			{counter(0, 30, 110), counter(3, 6, 14)},
		},
		loadResults: []LoadAverage{{}, {}, {}},
	}
	collectorValue := newTestCollector(t, source, func() time.Time { return now })
	collectorValue.Collect(context.Background())
	now = now.Add(time.Minute)
	changed := requireSnapshot(t, collectorValue.Collect(context.Background()).Snapshot)
	if changed.LogicalCPUCount != 2 || changed.Cores[0].LogicalIndex != 0 || changed.Cores[1].LogicalIndex != 3 {
		t.Fatalf("changed topology = %+v", changed.Cores)
	}
	assertAvailableNear(t, changed.Cores[0].Usage, 50)
	assertUnavailable(t, changed.Cores[1].Usage, domain.ReasonNotCollected)
	assertUnavailable(t, changed.Overall, domain.ReasonNotCollected)

	now = now.Add(time.Minute)
	stable := requireSnapshot(t, collectorValue.Collect(context.Background()).Snapshot)
	assertAvailableNear(t, stable.Overall, 50)
	assertAvailableNear(t, stable.Cores[1].Usage, 50)
}

func TestCollectorRejectsResetAndZeroProgressCounters(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		next Counter
	}{
		{name: "reset", next: counter(0, 9, 91)},
		{name: "no progress", next: counter(0, 10, 90)},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := &sourceStub{
				counterResults: [][]Counter{{counter(0, 10, 90)}, {test.next}},
				loadResults:    []LoadAverage{{}, {}},
			}
			collectorValue := newTestCollector(t, source, time.Now)
			collectorValue.Collect(context.Background())
			result := collectorValue.Collect(context.Background())
			snapshot := requireSnapshot(t, result.Snapshot)
			if result.Status != domain.CollectionPartial || result.ErrorCode != errorCounterInvalid {
				t.Fatalf("result = %+v", result)
			}
			assertUnavailable(t, snapshot.Overall, domain.ReasonCollectorError)
			assertUnavailable(t, snapshot.Cores[0].Usage, domain.ReasonCollectorError)
		})
	}
}

func TestCollectorDoesNotMaskOneCoreResetInOverallUsage(t *testing.T) {
	t.Parallel()
	source := &sourceStub{
		counterResults: [][]Counter{
			{counter(0, 100, 100), counter(1, 10, 90)},
			{counter(0, 10, 110), counter(1, 110, 190)},
		},
		loadResults: []LoadAverage{{}, {}},
	}
	collectorValue := newTestCollector(t, source, time.Now)
	collectorValue.Collect(context.Background())

	result := collectorValue.Collect(context.Background())
	snapshot := requireSnapshot(t, result.Snapshot)
	if result.Status != domain.CollectionPartial || result.ErrorCode != errorCounterInvalid {
		t.Fatalf("result = %+v", result)
	}
	assertUnavailable(t, snapshot.Overall, domain.ReasonCollectorError)
	assertUnavailable(t, snapshot.Cores[0].Usage, domain.ReasonCollectorError)
	assertAvailableNear(t, snapshot.Cores[1].Usage, 50)
}

func TestCollectorPreservesCPUWhenLoadIsUnavailable(t *testing.T) {
	t.Parallel()
	source := &sourceStub{
		counterResults: [][]Counter{{counter(0, 10, 90)}, {counter(0, 20, 100)}},
		loadResults:    []LoadAverage{{}, {}},
		loadErrors:     []error{nil, errors.New("synthetic load failure")},
	}
	collectorValue := newTestCollector(t, source, time.Now)
	collectorValue.Collect(context.Background())
	result := collectorValue.Collect(context.Background())
	snapshot := requireSnapshot(t, result.Snapshot)
	if result.Status != domain.CollectionPartial || result.ErrorCode != errorLoadUnavailable {
		t.Fatalf("result = %+v", result)
	}
	assertAvailableNear(t, snapshot.Overall, 50)
	assertUnavailable(t, snapshot.Load.OneMinute, domain.ReasonCollectorError)
}

func TestCollectorFailureRetainsLastSnapshot(t *testing.T) {
	t.Parallel()
	source := &sourceStub{
		counterResults: [][]Counter{{counter(0, 10, 90)}, nil},
		counterErrors:  []error{nil, errors.New("synthetic counter failure")},
		loadResults:    []LoadAverage{{}, {}},
	}
	collectorValue := newTestCollector(t, source, time.Now)
	collectorValue.Collect(context.Background())
	before, _ := collectorValue.Latest()
	result := collectorValue.Collect(context.Background())
	after, ok := collectorValue.Latest()
	if result.Status != domain.CollectionFailed || result.Snapshot != nil || !ok || !after.ObservedAt.Equal(before.ObservedAt) {
		t.Fatalf("failure result = %+v, latest = %+v", result, after)
	}
}

func TestCollectorRejectsDuplicateCPUIndexesAndInvalidLoad(t *testing.T) {
	t.Parallel()
	duplicateSource := &sourceStub{
		counterResults: [][]Counter{{counter(0, 10, 90), counter(0, 20, 80)}},
		loadResults:    []LoadAverage{{}},
	}
	duplicateCollector := newTestCollector(t, duplicateSource, time.Now)
	if result := duplicateCollector.Collect(context.Background()); result.Status != domain.CollectionFailed {
		t.Fatalf("duplicate result = %+v", result)
	}

	invalidLoadSource := &sourceStub{
		counterResults: [][]Counter{{counter(0, 10, 90)}},
		loadResults:    []LoadAverage{{One: math.NaN()}},
	}
	invalidLoadCollector := newTestCollector(t, invalidLoadSource, time.Now)
	result := invalidLoadCollector.Collect(context.Background())
	snapshot := requireSnapshot(t, result.Snapshot)
	if result.Status != domain.CollectionPartial || result.ErrorCode != errorPartial {
		t.Fatalf("invalid load result = %+v", result)
	}
	assertUnavailable(t, snapshot.Load.OneMinute, domain.ReasonCollectorError)
}

func TestCalculateUsageKnownFixture(t *testing.T) {
	t.Parallel()
	previous := Counter{User: 10, Nice: 5, System: 5, Idle: 70, IOWait: 10}
	current := Counter{User: 20, Nice: 5, System: 10, Idle: 80, IOWait: 15}
	usage, err := calculateUsage(previous, current)
	if err != nil {
		t.Fatalf("calculateUsage() error = %v", err)
	}
	// Delta total is 30 and idle+iowait delta is 15.
	if math.Abs(usage-50) > 0.000001 {
		t.Fatalf("usage = %f, want 50", usage)
	}
}

func counter(index int, busy, idle float64) Counter {
	return Counter{LogicalIndex: index, User: busy, Idle: idle}
}

type sourceStub struct {
	counterResults [][]Counter
	counterErrors  []error
	loadResults    []LoadAverage
	loadErrors     []error
	counterCall    int
	loadCall       int
}

func (source *sourceStub) Counters(context.Context) ([]Counter, error) {
	index := source.counterCall
	source.counterCall++
	var err error
	if index < len(source.counterErrors) {
		err = source.counterErrors[index]
	}
	return cloneCounters(source.counterResults[index]), err
}

func (source *sourceStub) LoadAverage(context.Context) (LoadAverage, error) {
	index := source.loadCall
	source.loadCall++
	var err error
	if index < len(source.loadErrors) {
		err = source.loadErrors[index]
	}
	return source.loadResults[index], err
}

func newTestCollector(t *testing.T, source Source, now func() time.Time) *Collector {
	t.Helper()
	collectorValue, err := NewWithOptions(Options{Source: source, Now: now})
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	return collectorValue
}

func requireSnapshot(t *testing.T, value any) Snapshot {
	t.Helper()
	snapshot, ok := value.(Snapshot)
	if !ok {
		t.Fatalf("snapshot type = %T", value)
	}
	if err := snapshot.Validate(); err != nil {
		t.Fatalf("snapshot validation error = %v", err)
	}
	return snapshot
}

func assertAvailableNear(t *testing.T, metric domain.Metric[float64], want float64) {
	t.Helper()
	if metric.Availability != domain.AvailabilityAvailable || metric.Value == nil || math.Abs(*metric.Value-want) > 0.000001 {
		t.Fatalf("metric = %+v, want available %f", metric, want)
	}
}

func assertUnavailable(t *testing.T, metric domain.Metric[float64], want domain.UnavailabilityReason) {
	t.Helper()
	if metric.Availability != domain.AvailabilityUnavailable || metric.Value != nil || metric.ReasonCode == nil || *metric.ReasonCode != want {
		t.Fatalf("metric = %+v, want unavailable %q", metric, want)
	}
}
