package history

import (
	"errors"
	"testing"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

func TestResolveRangeProducesApprovedPresetWindows(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 5, 12, 34, 45, 0, time.UTC)
	tests := []struct {
		preset   domain.RangePreset
		duration time.Duration
	}{
		{domain.RangeLastMinute, time.Minute},
		{domain.RangeLast5Minutes, 5 * time.Minute},
		{domain.RangeLast15Min, 15 * time.Minute},
		{domain.RangeLast30Min, 30 * time.Minute},
		{domain.RangeLastHour, time.Hour},
		{domain.RangeLast6Hours, 6 * time.Hour},
		{domain.RangeLast24Hours, 24 * time.Hour},
		{domain.RangeLast7Days, 7 * 24 * time.Hour},
		{domain.RangeLast14Days, RetentionDuration},
	}
	for _, test := range tests {
		t.Run(string(test.preset), func(t *testing.T) {
			resolved, err := ResolveRange(domain.RangeSelection{Preset: test.preset}, now)
			if err != nil {
				t.Fatalf("ResolveRange() error = %v", err)
			}
			if resolved.End != time.Date(2026, 8, 5, 12, 35, 0, 0, time.UTC) || resolved.End.Sub(resolved.Start) != test.duration {
				t.Fatalf("resolved range = %+v", resolved)
			}
		})
	}
}

func TestResolveRangeRejectsCustomWindowOutsideRetentionOrInFuture(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		start time.Time
		end   time.Time
		want  error
	}{
		{name: "before retention", start: now.Add(-RetentionDuration - time.Second), end: now.Add(-time.Hour), want: ErrRangeOutsideRetention},
		{name: "future", start: now.Add(-time.Hour), end: now.Add(time.Second), want: ErrRangeInFuture},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ResolveRange(domain.RangeSelection{Preset: domain.RangeCustom, Start: &test.start, End: &test.end}, now)
			if !errors.Is(err, test.want) {
				t.Fatalf("ResolveRange() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestResolveRangeRejectsNonUTCCustomBoundaries(t *testing.T) {
	t.Parallel()
	zone := time.FixedZone("not-utc", int(time.Hour/time.Second))
	start := time.Date(2026, 8, 5, 10, 0, 0, 0, zone)
	end := start.Add(time.Hour)
	if _, err := ResolveRange(
		domain.RangeSelection{Preset: domain.RangeCustom, Start: &start, End: &end},
		time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC),
	); err == nil {
		t.Fatal("non-UTC custom range was accepted")
	}
}

func TestBucketDurationBoundsLongRanges(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	resolved := domain.ResolvedRange{Preset: domain.RangeLast14Days, Start: start, End: start.Add(RetentionDuration)}
	duration := bucketDuration(resolved)
	if duration != 34*time.Minute {
		t.Fatalf("bucket duration = %s, want 34m", duration)
	}
	points := (resolved.End.Sub(resolved.Start) + duration - 1) / duration
	if points > MaxChartPoints {
		t.Fatalf("point count = %d, maximum = %d", points, MaxChartPoints)
	}
}
