package history

import (
	"errors"
	"fmt"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

const (
	RetentionDuration = 14 * 24 * time.Hour
	RawRangeLimit     = 6 * time.Hour
	MaxChartPoints    = 600
)

var (
	ErrRangeOutsideRetention = errors.New("history range is outside retained history")
	ErrRangeInFuture         = errors.New("history range ends in the future")
)

var presetDurations = map[domain.RangePreset]time.Duration{
	domain.RangeLastMinute:   time.Minute,
	domain.RangeLast5Minutes: 5 * time.Minute,
	domain.RangeLast15Min:    15 * time.Minute,
	domain.RangeLast30Min:    30 * time.Minute,
	domain.RangeLastHour:     time.Hour,
	domain.RangeLast6Hours:   6 * time.Hour,
	domain.RangeLast24Hours:  24 * time.Hour,
	domain.RangeLast7Days:    7 * 24 * time.Hour,
	domain.RangeLast14Days:   RetentionDuration,
}

// ResolveRange converts a validated selection into a UTC half-open interval.
// Presets end at the next minute boundary so each named minute corresponds to
// exactly one possible collection position.
func ResolveRange(selection domain.RangeSelection, now time.Time) (domain.ResolvedRange, error) {
	if err := selection.Validate(); err != nil {
		return domain.ResolvedRange{}, err
	}
	if now.IsZero() {
		return domain.ResolvedRange{}, errors.New("current time is required")
	}
	now = now.UTC()

	if selection.Preset == domain.RangeCustom {
		start := selection.Start.UTC()
		end := selection.End.UTC()
		if end.After(now) {
			return domain.ResolvedRange{}, ErrRangeInFuture
		}
		if start.Before(now.Add(-RetentionDuration)) {
			return domain.ResolvedRange{}, ErrRangeOutsideRetention
		}
		resolved := domain.ResolvedRange{Preset: selection.Preset, Start: start, End: end}
		if err := validateResolvedRange(resolved); err != nil {
			return domain.ResolvedRange{}, err
		}
		return resolved, nil
	}

	duration, ok := presetDurations[selection.Preset]
	if !ok {
		return domain.ResolvedRange{}, fmt.Errorf("unsupported history range preset %q", selection.Preset)
	}
	end := now.Truncate(time.Minute).Add(time.Minute)
	resolved := domain.ResolvedRange{Preset: selection.Preset, Start: end.Add(-duration), End: end}
	if err := validateResolvedRange(resolved); err != nil {
		return domain.ResolvedRange{}, err
	}
	return resolved, nil
}

func validateResolvedRange(resolved domain.ResolvedRange) error {
	if err := resolved.Validate(); err != nil {
		return err
	}
	if resolved.End.Sub(resolved.Start) > RetentionDuration {
		return ErrRangeOutsideRetention
	}
	return nil
}

func bucketDuration(resolved domain.ResolvedRange) time.Duration {
	duration := resolved.End.Sub(resolved.Start)
	if duration <= RawRangeLimit {
		return time.Minute
	}
	minutes := int64((duration + time.Minute - 1) / time.Minute)
	bucketMinutes := (minutes + MaxChartPoints - 1) / MaxChartPoints
	if bucketMinutes < 1 {
		bucketMinutes = 1
	}
	return time.Duration(bucketMinutes) * time.Minute
}
