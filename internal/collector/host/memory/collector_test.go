package memory

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

func TestCollectorReadsDistinctMemoryNoSwapAndPressureFields(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	collectorValue := newTestCollector(t, &sourceStub{
		memInfo: readTestFixture(t, "meminfo"), pressure: readTestFixture(t, "pressure_memory"),
	}, at)

	result := collectorValue.Collect(context.Background())
	if result.Status != domain.CollectionSucceeded {
		t.Fatalf("status = %q, error = %q", result.Status, result.ErrorCode)
	}
	snapshot := result.Snapshot.(Snapshot)
	assertIntegerMetric(t, snapshot.Total, 11*1024*1024)
	assertIntegerMetric(t, snapshot.Available, 8*1024*1024)
	assertIntegerMetric(t, snapshot.Used, 3*1024*1024)
	assertIntegerMetric(t, snapshot.Free, 2*1024*1024)
	assertIntegerMetric(t, snapshot.Cached, 5*1024*1024)
	assertIntegerMetric(t, snapshot.Buffered, 256*1024)
	if snapshot.Usage.Value == nil || *snapshot.Usage.Value < 27.27 || *snapshot.Usage.Value > 27.28 {
		t.Fatalf("usage = %+v", snapshot.Usage)
	}
	if snapshot.Swap.Configured == nil || *snapshot.Swap.Configured || snapshot.Swap.Total.Availability != domain.AvailabilityUnavailable ||
		snapshot.Swap.Total.ReasonCode == nil || *snapshot.Swap.Total.ReasonCode != domain.ReasonNotConfigured {
		t.Fatalf("no-swap state = %+v", snapshot.Swap)
	}
	if snapshot.Pressure.Availability != domain.AvailabilityAvailable ||
		snapshot.Pressure.Some.Average10Seconds.Value == nil || *snapshot.Pressure.Some.Average10Seconds.Value != 0.2 ||
		snapshot.Pressure.Full.Total.Value == nil || *snapshot.Pressure.Full.Total.Value != 9 {
		t.Fatalf("pressure = %+v", snapshot.Pressure)
	}
	if latest, ok := collectorValue.Latest(); !ok || latest.ObservedAt != at {
		t.Fatalf("latest = %+v, %v", latest, ok)
	}
}

func TestCollectorCalculatesConfiguredSwap(t *testing.T) {
	t.Parallel()
	collectorValue := newTestCollector(t, &sourceStub{memInfo: []byte(configuredSwapMemInfo), pressure: []byte(validPressure)}, testTime())
	result := collectorValue.Collect(context.Background())
	if result.Status != domain.CollectionSucceeded {
		t.Fatalf("status = %q, error = %q", result.Status, result.ErrorCode)
	}
	swap := result.Snapshot.(Snapshot).Swap
	if swap.Configured == nil || !*swap.Configured {
		t.Fatal("configured swap was reported as disabled")
	}
	assertIntegerMetric(t, swap.Total, 2*1024*1024)
	assertIntegerMetric(t, swap.Used, 1536*1024)
	assertIntegerMetric(t, swap.Free, 512*1024)
}

func TestCollectorKeepsRAMWhenPressureIsUnsupported(t *testing.T) {
	t.Parallel()
	collectorValue := newTestCollector(t, &sourceStub{
		memInfo: []byte(validMemInfo), pressureErr: os.ErrNotExist,
	}, testTime())
	result := collectorValue.Collect(context.Background())
	if result.Status != domain.CollectionPartial || result.ErrorCode != errorPressureUnavailable {
		t.Fatalf("result = %+v", result)
	}
	snapshot := result.Snapshot.(Snapshot)
	if snapshot.Total.Availability != domain.AvailabilityAvailable ||
		snapshot.Pressure.Availability != domain.AvailabilityUnavailable ||
		snapshot.Pressure.ReasonCode == nil || *snapshot.Pressure.ReasonCode != domain.ReasonNotSupported {
		t.Fatalf("partial snapshot = %+v", snapshot)
	}
}

func TestCollectorPreservesPartialMemInfoWithoutInventingDerivedValues(t *testing.T) {
	t.Parallel()
	partial := "MemTotal: 11264 kB\nMemFree: 2048 kB\nBuffers: 256 kB\nCached: broken kB\nSwapTotal: 0 kB\nSwapFree: 0 kB\n"
	collectorValue := newTestCollector(t, &sourceStub{memInfo: []byte(partial), pressure: []byte(validPressure)}, testTime())
	result := collectorValue.Collect(context.Background())
	if result.Status != domain.CollectionPartial || result.ErrorCode != errorMemoryPartial {
		t.Fatalf("result = %+v", result)
	}
	snapshot := result.Snapshot.(Snapshot)
	if snapshot.Total.Value == nil || snapshot.Available.Value != nil || snapshot.Cached.Value != nil ||
		snapshot.Used.Value != nil || snapshot.Usage.Value != nil {
		t.Fatalf("partial memory values were fabricated or discarded: %+v", snapshot)
	}
}

func TestCollectorRejectsUnavailableMemInfoAndMalformedPressure(t *testing.T) {
	t.Parallel()
	readFailure := newTestCollector(t, &sourceStub{memInfoErr: errors.New("unreadable")}, testTime())
	result := readFailure.Collect(context.Background())
	if result.Status != domain.CollectionFailed || result.Snapshot != nil {
		t.Fatalf("meminfo failure = %+v", result)
	}

	badPressure := newTestCollector(t, &sourceStub{
		memInfo: []byte(validMemInfo), pressure: []byte("some avg10=invalid avg60=0 avg300=0 total=0\n"),
	}, testTime())
	result = badPressure.Collect(context.Background())
	if result.Status != domain.CollectionPartial {
		t.Fatalf("malformed pressure result = %+v", result)
	}
	snapshot := result.Snapshot.(Snapshot)
	if snapshot.Pressure.ReasonCode == nil || *snapshot.Pressure.ReasonCode != domain.ReasonCollectorError {
		t.Fatalf("malformed pressure reason = %+v", snapshot.Pressure)
	}
}

func TestMemInfoParserRejectsDuplicateOverflowAndInvalidUnits(t *testing.T) {
	t.Parallel()
	input := validMemInfo + "MemTotal: 1 kB\nMemFree: 1 MB\nCached: 999999999999999999 kB\n"
	values, err := parseMemInfo([]byte(input))
	if err == nil {
		t.Fatal("invalid meminfo was accepted")
	}
	if _, ok := values["MemTotal"]; ok {
		t.Fatal("duplicated MemTotal remained available")
	}
	if _, ok := values["MemFree"]; ok {
		t.Fatal("invalid-unit MemFree remained available")
	}
	if _, ok := values["Cached"]; ok {
		t.Fatal("overflowing Cached remained available")
	}
}

type sourceStub struct {
	memInfo     []byte
	memInfoErr  error
	pressure    []byte
	pressureErr error
}

func (source *sourceStub) MemInfo(context.Context) ([]byte, error) {
	return source.memInfo, source.memInfoErr
}

func (source *sourceStub) Pressure(context.Context) ([]byte, error) {
	return source.pressure, source.pressureErr
}

func newTestCollector(t *testing.T, source Source, at time.Time) *Collector {
	t.Helper()
	value, err := NewWithOptions(Options{Source: source, Now: func() time.Time { return at }})
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	return value
}

func testTime() time.Time { return time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC) }

func assertIntegerMetric(t *testing.T, metric domain.Metric[int64], want int64) {
	t.Helper()
	if metric.Value == nil || *metric.Value != want {
		t.Fatalf("metric = %+v, want %d", metric, want)
	}
}

func readTestFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "proc", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

const validMemInfo = `MemTotal:       11264 kB
MemFree:         2048 kB
MemAvailable:    8192 kB
Buffers:          256 kB
Cached:          5120 kB
SwapTotal:          0 kB
SwapFree:           0 kB
`

const validPressure = `some avg10=0.20 avg60=0.10 avg300=0.05 total=1200
full avg10=0.01 avg60=0.00 avg300=0.00 total=9
`

const configuredSwapMemInfo = `MemTotal:       11264 kB
MemFree:         2048 kB
MemAvailable:    8192 kB
Buffers:          256 kB
Cached:          5120 kB
SwapTotal:       2048 kB
SwapFree:         512 kB
`
