package contracts

import (
	"errors"
	"fmt"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

var ErrInvalidMemoryHistoryRange = errors.New("invalid memory history range")

type MemoryMetric string

const (
	MemoryMetricTotal            MemoryMetric = "total"
	MemoryMetricUsed             MemoryMetric = "used"
	MemoryMetricAvailable        MemoryMetric = "available"
	MemoryMetricFree             MemoryMetric = "free"
	MemoryMetricCached           MemoryMetric = "cached"
	MemoryMetricBuffered         MemoryMetric = "buffered"
	MemoryMetricUsage            MemoryMetric = "usage"
	MemoryMetricSwapTotal        MemoryMetric = "swap_total"
	MemoryMetricSwapUsed         MemoryMetric = "swap_used"
	MemoryMetricPSISomeAverage10 MemoryMetric = "pressure_some_avg10"
	MemoryMetricPSIFullAverage10 MemoryMetric = "pressure_full_avg10"
	MemoryMetricPSISomeTotal     MemoryMetric = "pressure_some_total"
	MemoryMetricPSIFullTotal     MemoryMetric = "pressure_full_total"
)

func (metric MemoryMetric) Valid() bool {
	switch metric {
	case MemoryMetricTotal, MemoryMetricUsed, MemoryMetricAvailable, MemoryMetricFree,
		MemoryMetricCached, MemoryMetricBuffered, MemoryMetricUsage,
		MemoryMetricSwapTotal, MemoryMetricSwapUsed,
		MemoryMetricPSISomeAverage10, MemoryMetricPSIFullAverage10,
		MemoryMetricPSISomeTotal, MemoryMetricPSIFullTotal:
		return true
	default:
		return false
	}
}

func (metric MemoryMetric) Unit() domain.Unit {
	switch metric {
	case MemoryMetricUsage, MemoryMetricPSISomeAverage10, MemoryMetricPSIFullAverage10:
		return domain.UnitPercent
	case MemoryMetricPSISomeTotal, MemoryMetricPSIFullTotal:
		return domain.UnitMicroseconds
	default:
		return domain.UnitBytes
	}
}

type MemoryHistoryRequest struct {
	Metric MemoryMetric
	Range  domain.RangeSelection
}

func (request MemoryHistoryRequest) Validate() error {
	if !request.Metric.Valid() {
		return fmt.Errorf("invalid memory metric %q", request.Metric)
	}
	return request.Range.Validate()
}

type MemoryHistoryPointState string

const (
	MemoryHistoryObserved    MemoryHistoryPointState = "observed"
	MemoryHistoryUnavailable MemoryHistoryPointState = "unavailable"
	MemoryHistoryGap         MemoryHistoryPointState = "gap"
)

type MemoryHistoryPoint struct {
	Start            time.Time               `json:"start"`
	End              time.Time               `json:"end"`
	State            MemoryHistoryPointState `json:"state"`
	ObservedSamples  int                     `json:"observedSamples"`
	AvailableSamples int                     `json:"availableSamples"`
	Minimum          *float64                `json:"minimum"`
	Average          *float64                `json:"average"`
	Maximum          *float64                `json:"maximum"`
}

type MemoryHistorySeries struct {
	Resource              domain.ResourceRef   `json:"resource"`
	Metric                MemoryMetric         `json:"metric"`
	Unit                  domain.Unit          `json:"unit"`
	Range                 domain.ResolvedRange `json:"range"`
	BucketDurationSeconds int64                `json:"bucketDurationSeconds"`
	Points                []MemoryHistoryPoint `json:"points"`
}

func (series MemoryHistorySeries) Validate() error {
	if err := validateResourceKind(series.Resource, domain.ResourceMemory); err != nil {
		return err
	}
	selection := domain.RangeSelection{Preset: series.Range.Preset}
	if series.Range.Preset == domain.RangeCustom {
		selection.Start = &series.Range.Start
		selection.End = &series.Range.End
	}
	if err := (MemoryHistoryRequest{Metric: series.Metric, Range: selection}).Validate(); err != nil {
		return err
	}
	if err := series.Range.Validate(); err != nil {
		return fmt.Errorf("range: %w", err)
	}
	if series.Unit != series.Metric.Unit() {
		return fmt.Errorf("unit does not match memory metric")
	}
	if series.BucketDurationSeconds < 60 {
		return fmt.Errorf("bucket duration must be at least 60 seconds")
	}
	if series.Points == nil {
		return fmt.Errorf("points must be an array")
	}
	if len(series.Points) > 600 {
		return fmt.Errorf("memory history cannot exceed 600 points")
	}
	for index, point := range series.Points {
		if err := point.validate(series); err != nil {
			return fmt.Errorf("points[%d]: %w", index, err)
		}
	}
	return nil
}

func (point MemoryHistoryPoint) validate(series MemoryHistorySeries) error {
	if err := domain.ValidateUTC(point.Start); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	if err := domain.ValidateUTC(point.End); err != nil {
		return fmt.Errorf("end: %w", err)
	}
	if !point.End.After(point.Start) || point.Start.Before(series.Range.Start) || point.End.After(series.Range.End) {
		return fmt.Errorf("bucket must be a positive interval inside the resolved range")
	}
	if point.ObservedSamples < 0 || point.AvailableSamples < 0 || point.AvailableSamples > point.ObservedSamples {
		return fmt.Errorf("sample counts are inconsistent")
	}
	valuesPresent := point.Minimum != nil && point.Average != nil && point.Maximum != nil
	switch point.State {
	case MemoryHistoryObserved:
		if point.ObservedSamples == 0 || point.AvailableSamples == 0 || !valuesPresent {
			return fmt.Errorf("observed bucket requires samples and summaries")
		}
	case MemoryHistoryUnavailable:
		if point.ObservedSamples == 0 || point.AvailableSamples != 0 || point.Minimum != nil || point.Average != nil || point.Maximum != nil {
			return fmt.Errorf("unavailable bucket requires observations without values")
		}
	case MemoryHistoryGap:
		if point.ObservedSamples != 0 || point.AvailableSamples != 0 || point.Minimum != nil || point.Average != nil || point.Maximum != nil {
			return fmt.Errorf("gap bucket cannot contain evidence")
		}
	default:
		return fmt.Errorf("invalid memory history point state %q", point.State)
	}
	if valuesPresent {
		return validateSummaryValues(*point.Minimum, *point.Average, *point.Maximum, series.Unit)
	}
	return nil
}
