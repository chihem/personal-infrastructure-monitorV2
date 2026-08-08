package app

import (
	"context"
	"fmt"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/api/contracts"
	hostmemory "github.com/chihem/personal-infrastructure-monitorV2/internal/collector/host/memory"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/storage/history"
)

type memoryDataSource struct {
	collector  *hostmemory.Collector
	history    *history.Repository
	now        func() time.Time
	staleAfter time.Duration
}

func (source memoryDataSource) CurrentMemory() contracts.MemorySnapshot {
	snapshot, ok := source.collector.Latest()
	if !ok {
		return unavailableMemorySnapshot()
	}
	observedAt := snapshot.ObservedAt
	freshness := domain.Freshness{
		State: domain.FreshnessFresh, ObservedAt: &observedAt, LastSuccessfulAt: &observedAt,
	}
	if !source.now().UTC().Before(observedAt.Add(source.staleAfter)) {
		freshness.State = domain.FreshnessStale
	}
	return contracts.MemorySnapshot{
		Resource: memoryResource(), Freshness: freshness,
		Total: cloneIntegerMetric(snapshot.Total), Used: cloneIntegerMetric(snapshot.Used),
		Available: cloneIntegerMetric(snapshot.Available), Free: cloneIntegerMetric(snapshot.Free),
		Cached: cloneIntegerMetric(snapshot.Cached), Buffered: cloneIntegerMetric(snapshot.Buffered),
		Usage: cloneMetric(snapshot.Usage),
		Swap: contracts.SwapSnapshot{
			Configured: cloneBool(snapshot.Swap.Configured), Total: cloneIntegerMetric(snapshot.Swap.Total),
			Used: cloneIntegerMetric(snapshot.Swap.Used), Free: cloneIntegerMetric(snapshot.Swap.Free),
		},
		Pressure: contracts.MemoryPressure{
			Some: pressureWindowContract(snapshot.Pressure.Some),
			Full: pressureWindowContract(snapshot.Pressure.Full),
		},
	}
}

func (source memoryDataSource) MemoryHistory(ctx context.Context, request contracts.MemoryHistoryRequest) (contracts.MemoryHistorySeries, error) {
	if err := request.Validate(); err != nil {
		return contracts.MemoryHistorySeries{}, err
	}
	resolved, err := history.ResolveRange(request.Range, source.now().UTC())
	if err != nil {
		return contracts.MemoryHistorySeries{}, fmt.Errorf("%w: %v", contracts.ErrInvalidMemoryHistoryRange, err)
	}
	series, err := source.history.QueryMetricSeries(ctx, history.MetricQuery{
		Metric: memoryHistoryMetric(request.Metric), Range: resolved,
	})
	if err != nil {
		return contracts.MemoryHistorySeries{}, err
	}
	points := make([]contracts.MemoryHistoryPoint, 0, len(series.Points))
	for _, point := range series.Points {
		points = append(points, contracts.MemoryHistoryPoint{
			Start: point.Start, End: point.End, State: memoryHistoryPointState(point.State),
			ObservedSamples: point.ObservedSamples, AvailableSamples: point.AvailableSamples,
			Minimum: cloneFloat(point.Minimum), Average: cloneFloat(point.Average), Maximum: cloneFloat(point.Maximum),
		})
	}
	response := contracts.MemoryHistorySeries{
		Resource: memoryResource(), Metric: request.Metric, Unit: series.Unit, Range: series.Range,
		BucketDurationSeconds: int64(series.BucketDuration / time.Second), Points: points,
	}
	if err := response.Validate(); err != nil {
		return contracts.MemoryHistorySeries{}, fmt.Errorf("build memory history response: %w", err)
	}
	return response, nil
}

func unavailableMemorySnapshot() contracts.MemorySnapshot {
	notCollected := domain.ReasonNotCollected
	byteMetric := unavailableIntegerMetric(domain.UnitBytes, notCollected)
	pressure := contracts.PressureWindow{
		Average10Seconds:  unavailableMetric(domain.UnitPercent, notCollected),
		Average60Seconds:  unavailableMetric(domain.UnitPercent, notCollected),
		Average300Seconds: unavailableMetric(domain.UnitPercent, notCollected),
		Total:             unavailableIntegerMetric(domain.UnitMicroseconds, notCollected),
	}
	return contracts.MemorySnapshot{
		Resource: memoryResource(), Freshness: domain.Freshness{State: domain.FreshnessUnavailable},
		Total: cloneIntegerMetric(byteMetric), Used: cloneIntegerMetric(byteMetric),
		Available: cloneIntegerMetric(byteMetric), Free: cloneIntegerMetric(byteMetric),
		Cached: cloneIntegerMetric(byteMetric), Buffered: cloneIntegerMetric(byteMetric),
		Usage: unavailableMetric(domain.UnitPercent, notCollected),
		Swap: contracts.SwapSnapshot{
			Total: cloneIntegerMetric(byteMetric), Used: cloneIntegerMetric(byteMetric), Free: cloneIntegerMetric(byteMetric),
		},
		Pressure: contracts.MemoryPressure{Some: pressure, Full: clonePressureContract(pressure)},
	}
}

func pressureWindowContract(window hostmemory.PressureWindow) contracts.PressureWindow {
	return contracts.PressureWindow{
		Average10Seconds:  cloneMetric(window.Average10Seconds),
		Average60Seconds:  cloneMetric(window.Average60Seconds),
		Average300Seconds: cloneMetric(window.Average300Seconds),
		Total:             cloneIntegerMetric(window.Total),
	}
}

func clonePressureContract(window contracts.PressureWindow) contracts.PressureWindow {
	return contracts.PressureWindow{
		Average10Seconds:  cloneMetric(window.Average10Seconds),
		Average60Seconds:  cloneMetric(window.Average60Seconds),
		Average300Seconds: cloneMetric(window.Average300Seconds),
		Total:             cloneIntegerMetric(window.Total),
	}
}

func cloneIntegerMetric(metric domain.Metric[int64]) domain.Metric[int64] {
	return domain.Metric[int64]{
		Availability: metric.Availability, Value: cloneInt64(metric.Value), Unit: metric.Unit,
		ReasonCode: metricReason(metric),
	}
}

func unavailableIntegerMetric(unit domain.Unit, reason domain.UnavailabilityReason) domain.Metric[int64] {
	return domain.Metric[int64]{Availability: domain.AvailabilityUnavailable, Unit: unit, ReasonCode: &reason}
}

func cloneInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func cloneBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func memoryHistoryMetric(metric contracts.MemoryMetric) history.MetricKey {
	switch metric {
	case contracts.MemoryMetricTotal:
		return history.MetricMemoryTotalBytes
	case contracts.MemoryMetricUsed:
		return history.MetricMemoryUsedBytes
	case contracts.MemoryMetricAvailable:
		return history.MetricMemoryAvailableBytes
	case contracts.MemoryMetricFree:
		return history.MetricMemoryFreeBytes
	case contracts.MemoryMetricCached:
		return history.MetricMemoryCachedBytes
	case contracts.MemoryMetricBuffered:
		return history.MetricMemoryBufferedBytes
	case contracts.MemoryMetricSwapTotal:
		return history.MetricSwapTotalBytes
	case contracts.MemoryMetricSwapUsed:
		return history.MetricSwapUsedBytes
	case contracts.MemoryMetricPSISomeAverage10:
		return history.MetricMemoryPSISomeAverage10
	case contracts.MemoryMetricPSIFullAverage10:
		return history.MetricMemoryPSIFullAverage10
	case contracts.MemoryMetricPSISomeTotal:
		return history.MetricMemoryPSISomeTotalUS
	case contracts.MemoryMetricPSIFullTotal:
		return history.MetricMemoryPSIFullTotalUS
	default:
		return history.MetricMemoryUsagePercent
	}
}

func memoryHistoryPointState(state history.MetricPointState) contracts.MemoryHistoryPointState {
	switch state {
	case history.MetricPointObserved:
		return contracts.MemoryHistoryObserved
	case history.MetricPointUnavailable:
		return contracts.MemoryHistoryUnavailable
	default:
		return contracts.MemoryHistoryGap
	}
}

func memoryResource() domain.ResourceRef {
	return domain.ResourceRef{Kind: domain.ResourceMemory, ID: "host-memory", DisplayName: "Memory"}
}
