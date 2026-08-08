package contracts

import (
	"testing"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

func TestMemoryHistoryContractPreservesObservedUnavailableAndGapStates(t *testing.T) {
	t.Parallel()
	end := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	start := end.Add(-5 * time.Minute)
	value := 50.0
	series := MemoryHistorySeries{
		Resource: domain.ResourceRef{Kind: domain.ResourceMemory, ID: "host-memory", DisplayName: "Memory"},
		Metric:   MemoryMetricUsage, Unit: domain.UnitPercent,
		Range:                 domain.ResolvedRange{Preset: domain.RangeLast5Minutes, Start: start, End: end},
		BucketDurationSeconds: 60,
		Points: []MemoryHistoryPoint{
			{Start: start, End: start.Add(time.Minute), State: MemoryHistoryObserved, ObservedSamples: 1, AvailableSamples: 1, Minimum: &value, Average: &value, Maximum: &value},
			{Start: start.Add(time.Minute), End: start.Add(2 * time.Minute), State: MemoryHistoryUnavailable, ObservedSamples: 1},
			{Start: start.Add(2 * time.Minute), End: start.Add(3 * time.Minute), State: MemoryHistoryGap},
		},
	}
	if err := series.Validate(); err != nil {
		t.Fatalf("valid series rejected: %v", err)
	}
	series.Points[2].Average = &value
	if err := series.Validate(); err == nil {
		t.Fatal("gap with a fabricated average was accepted")
	}
}

func TestMemoryMetricUnitsAndRequestValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		metric MemoryMetric
		unit   domain.Unit
	}{
		{MemoryMetricTotal, domain.UnitBytes},
		{MemoryMetricUsage, domain.UnitPercent},
		{MemoryMetricPSISomeAverage10, domain.UnitPercent},
		{MemoryMetricPSIFullTotal, domain.UnitMicroseconds},
	}
	for _, test := range tests {
		request := MemoryHistoryRequest{Metric: test.metric, Range: domain.RangeSelection{Preset: domain.RangeLastHour}}
		if err := request.Validate(); err != nil || test.metric.Unit() != test.unit {
			t.Errorf("metric %q validation = %v, unit = %q", test.metric, err, test.metric.Unit())
		}
	}
	invalid := MemoryHistoryRequest{Metric: MemoryMetric("host_samples; DROP TABLE host_samples"), Range: domain.RangeSelection{Preset: domain.RangeLastHour}}
	if err := invalid.Validate(); err == nil {
		t.Fatal("unallowlisted memory metric was accepted")
	}
}
