package contracts

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

var ErrInvalidCPUHistoryRange = errors.New("invalid CPU history range")

type CPUMetric string

const (
	CPUMetricOverall CPUMetric = "overall"
	CPUMetricCore    CPUMetric = "core"
	CPUMetricLoad1   CPUMetric = "load_1"
	CPUMetricLoad5   CPUMetric = "load_5"
	CPUMetricLoad15  CPUMetric = "load_15"
)

func (metric CPUMetric) Valid() bool {
	switch metric {
	case CPUMetricOverall, CPUMetricCore, CPUMetricLoad1, CPUMetricLoad5, CPUMetricLoad15:
		return true
	default:
		return false
	}
}

func (metric CPUMetric) Unit() domain.Unit {
	if metric == CPUMetricOverall || metric == CPUMetricCore {
		return domain.UnitPercent
	}
	return domain.UnitLoad
}

type CPUHistoryRequest struct {
	Metric    CPUMetric
	CoreIndex *int
	Range     domain.RangeSelection
}

func (request CPUHistoryRequest) Validate() error {
	if !request.Metric.Valid() {
		return fmt.Errorf("invalid CPU metric %q", request.Metric)
	}
	if request.Metric == CPUMetricCore {
		if request.CoreIndex == nil || *request.CoreIndex < 0 {
			return fmt.Errorf("core metric requires a non-negative core index")
		}
	} else if request.CoreIndex != nil {
		return fmt.Errorf("core index is valid only for the core metric")
	}
	return request.Range.Validate()
}

type CPUHistoryPointState string

const (
	CPUHistoryObserved    CPUHistoryPointState = "observed"
	CPUHistoryUnavailable CPUHistoryPointState = "unavailable"
	CPUHistoryGap         CPUHistoryPointState = "gap"
)

type CPUHistoryPoint struct {
	Start            time.Time            `json:"start"`
	End              time.Time            `json:"end"`
	State            CPUHistoryPointState `json:"state"`
	ObservedSamples  int                  `json:"observedSamples"`
	AvailableSamples int                  `json:"availableSamples"`
	Minimum          *float64             `json:"minimum"`
	Average          *float64             `json:"average"`
	Maximum          *float64             `json:"maximum"`
}

type CPUHistorySeries struct {
	Resource              domain.ResourceRef   `json:"resource"`
	Metric                CPUMetric            `json:"metric"`
	CoreIndex             *int                 `json:"coreIndex"`
	Unit                  domain.Unit          `json:"unit"`
	Range                 domain.ResolvedRange `json:"range"`
	BucketDurationSeconds int64                `json:"bucketDurationSeconds"`
	Points                []CPUHistoryPoint    `json:"points"`
}

func (series CPUHistorySeries) Validate() error {
	if err := validateResourceKind(series.Resource, domain.ResourceCPU); err != nil {
		return err
	}
	request := CPUHistoryRequest{
		Metric: series.Metric, CoreIndex: series.CoreIndex,
		Range: domain.RangeSelection{Preset: series.Range.Preset},
	}
	if series.Range.Preset == domain.RangeCustom {
		request.Range.Start = &series.Range.Start
		request.Range.End = &series.Range.End
	}
	if err := request.Validate(); err != nil {
		return err
	}
	if err := series.Range.Validate(); err != nil {
		return fmt.Errorf("range: %w", err)
	}
	if series.Unit != series.Metric.Unit() {
		return fmt.Errorf("unit does not match CPU metric")
	}
	if series.BucketDurationSeconds < 60 {
		return fmt.Errorf("bucket duration must be at least 60 seconds")
	}
	if series.Points == nil {
		return fmt.Errorf("points must be an array")
	}
	if len(series.Points) > 600 {
		return fmt.Errorf("CPU history cannot exceed 600 points")
	}
	for index, point := range series.Points {
		if err := point.validate(series); err != nil {
			return fmt.Errorf("points[%d]: %w", index, err)
		}
	}
	return nil
}

func (point CPUHistoryPoint) validate(series CPUHistorySeries) error {
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
	case CPUHistoryObserved:
		if point.ObservedSamples == 0 || point.AvailableSamples == 0 || !valuesPresent {
			return fmt.Errorf("observed bucket requires samples and minimum/average/maximum")
		}
	case CPUHistoryUnavailable:
		if point.ObservedSamples == 0 || point.AvailableSamples != 0 || point.Minimum != nil || point.Average != nil || point.Maximum != nil {
			return fmt.Errorf("unavailable bucket requires observations without values")
		}
	case CPUHistoryGap:
		if point.ObservedSamples != 0 || point.AvailableSamples != 0 || point.Minimum != nil || point.Average != nil || point.Maximum != nil {
			return fmt.Errorf("gap bucket cannot contain evidence")
		}
	default:
		return fmt.Errorf("invalid CPU history point state %q", point.State)
	}
	if valuesPresent {
		if err := validateSummaryValues(*point.Minimum, *point.Average, *point.Maximum, series.Unit); err != nil {
			return err
		}
	}
	return nil
}

func validateSummaryValues(minimum, average, maximum float64, unit domain.Unit) error {
	for _, value := range []float64{minimum, average, maximum} {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			return fmt.Errorf("summary values must be finite and non-negative")
		}
		if unit == domain.UnitPercent && value > 100 {
			return fmt.Errorf("percentage summary cannot exceed 100")
		}
	}
	if minimum > average || average > maximum {
		return fmt.Errorf("summary values must satisfy minimum <= average <= maximum")
	}
	return nil
}
