package contracts

import (
	"testing"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

func TestCPUHistorySeriesValidatesObservedUnavailableAndGapBuckets(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	minimum, average, maximum := 10.0, 20.0, 30.0
	series := CPUHistorySeries{
		Resource: domain.ResourceRef{Kind: domain.ResourceCPU, ID: "host-cpu", DisplayName: "CPU"},
		Metric:   CPUMetricOverall, Unit: domain.UnitPercent,
		Range:                 domain.ResolvedRange{Preset: domain.RangeLast5Minutes, Start: start, End: start.Add(3 * time.Minute)},
		BucketDurationSeconds: 60,
		Points: []CPUHistoryPoint{
			{Start: start, End: start.Add(time.Minute), State: CPUHistoryObserved, ObservedSamples: 1, AvailableSamples: 1, Minimum: &minimum, Average: &average, Maximum: &maximum},
			{Start: start.Add(time.Minute), End: start.Add(2 * time.Minute), State: CPUHistoryUnavailable, ObservedSamples: 1},
			{Start: start.Add(2 * time.Minute), End: start.Add(3 * time.Minute), State: CPUHistoryGap},
		},
	}
	if err := series.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	series.Points[2].Average = &average
	if err := series.Validate(); err == nil {
		t.Fatal("gap containing a fabricated average was accepted")
	}
}

func TestCPUHistoryRequestRequiresCoreOnlyForCoreMetric(t *testing.T) {
	t.Parallel()
	core := 2
	request := CPUHistoryRequest{Metric: CPUMetricCore, CoreIndex: &core, Range: domain.RangeSelection{Preset: domain.RangeLastHour}}
	if err := request.Validate(); err != nil {
		t.Fatalf("valid core request error = %v", err)
	}
	request.CoreIndex = nil
	if err := request.Validate(); err == nil {
		t.Fatal("core request without an index was accepted")
	}
	request.Metric = CPUMetricOverall
	request.CoreIndex = &core
	if err := request.Validate(); err == nil {
		t.Fatal("overall request with a core index was accepted")
	}
}
