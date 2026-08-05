package domain

import (
	"testing"
	"time"
)

func TestMetricAvailabilityInvariant(t *testing.T) {
	value := 42.0
	reason := ReasonNotCollected

	tests := []struct {
		name    string
		metric  Metric[float64]
		wantErr bool
	}{
		{
			name: "available value",
			metric: Metric[float64]{
				Availability: AvailabilityAvailable,
				Value:        &value,
				Unit:         UnitPercent,
			},
		},
		{
			name: "unavailable reason",
			metric: Metric[float64]{
				Availability: AvailabilityUnavailable,
				Unit:         UnitPercent,
				ReasonCode:   &reason,
			},
		},
		{
			name: "available without value",
			metric: Metric[float64]{
				Availability: AvailabilityAvailable,
				Unit:         UnitPercent,
			},
			wantErr: true,
		},
		{
			name: "unavailable with fake zero",
			metric: Metric[float64]{
				Availability: AvailabilityUnavailable,
				Value:        new(float64),
				Unit:         UnitPercent,
				ReasonCode:   &reason,
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.metric.Validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestFreshnessStatesRemainDistinct(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		freshness Freshness
		wantErr   bool
	}{
		{
			name: "fresh",
			freshness: Freshness{
				State: FreshnessFresh, ObservedAt: &now, LastSuccessfulAt: &now,
			},
		},
		{
			name: "stale retains last success",
			freshness: Freshness{
				State: FreshnessStale, ObservedAt: &now, LastSuccessfulAt: &now,
			},
		},
		{
			name:      "never available",
			freshness: Freshness{State: FreshnessUnavailable},
		},
		{
			name:      "stale without evidence",
			freshness: Freshness{State: FreshnessStale},
			wantErr:   true,
		},
		{
			name: "unavailable with fabricated timestamp",
			freshness: Freshness{
				State: FreshnessUnavailable, ObservedAt: &now,
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.freshness.Validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestRangeSelection(t *testing.T) {
	start := time.Date(2026, 8, 4, 11, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	if err := (RangeSelection{Preset: RangeLastHour}).Validate(); err != nil {
		t.Fatalf("preset range rejected: %v", err)
	}
	if err := (RangeSelection{Preset: RangeCustom, Start: &start, End: &end}).Validate(); err != nil {
		t.Fatalf("custom range rejected: %v", err)
	}
	if err := (RangeSelection{Preset: RangeCustom, Start: &end, End: &start}).Validate(); err == nil {
		t.Fatal("backwards custom range accepted")
	}
	if err := (RangeSelection{Preset: RangeLastHour, Start: &start, End: &end}).Validate(); err == nil {
		t.Fatal("preset range accepted custom boundaries")
	}
}
