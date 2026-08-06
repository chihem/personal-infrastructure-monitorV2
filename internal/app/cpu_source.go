package app

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/api/contracts"
	hostcpu "github.com/chihem/personal-infrastructure-monitorV2/internal/collector/host/cpu"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/storage/history"
)

type cpuDataSource struct {
	collector  *hostcpu.Collector
	history    *history.Repository
	now        func() time.Time
	staleAfter time.Duration
}

func (source cpuDataSource) CurrentCPU() contracts.CPUSnapshot {
	snapshot, ok := source.collector.Latest()
	if !ok {
		return unavailableCPUSnapshot()
	}
	observedAt := snapshot.ObservedAt
	freshness := domain.Freshness{
		State: domain.FreshnessFresh, ObservedAt: &observedAt, LastSuccessfulAt: &observedAt,
	}
	if !source.now().UTC().Before(observedAt.Add(source.staleAfter)) {
		freshness.State = domain.FreshnessStale
	}
	cores := make([]contracts.CPUCore, 0, len(snapshot.Cores))
	for _, core := range snapshot.Cores {
		cores = append(cores, contracts.CPUCore{Index: core.LogicalIndex, Usage: cloneMetric(core.Usage)})
	}
	return contracts.CPUSnapshot{
		Resource: cpuResource(nil), Freshness: freshness, Overall: cloneMetric(snapshot.Overall), Cores: cores,
		Load: contracts.LoadAverages{
			OneMinute: cloneMetric(snapshot.Load.OneMinute), FiveMinutes: cloneMetric(snapshot.Load.FiveMinutes),
			FifteenMinute: cloneMetric(snapshot.Load.FifteenMinutes),
		},
		LogicalCPU: snapshot.LogicalCPUCount,
	}
}

func (source cpuDataSource) CPUHistory(ctx context.Context, request contracts.CPUHistoryRequest) (contracts.CPUHistorySeries, error) {
	if err := request.Validate(); err != nil {
		return contracts.CPUHistorySeries{}, err
	}
	resolved, err := history.ResolveRange(request.Range, source.now().UTC())
	if err != nil {
		return contracts.CPUHistorySeries{}, fmt.Errorf("%w: %v", contracts.ErrInvalidCPUHistoryRange, err)
	}
	metric, resourceID := historyMetric(request)
	series, err := source.history.QueryMetricSeries(ctx, history.MetricQuery{
		Metric: metric, ResourceID: resourceID, Range: resolved,
	})
	if err != nil {
		return contracts.CPUHistorySeries{}, err
	}
	points := make([]contracts.CPUHistoryPoint, 0, len(series.Points))
	for _, point := range series.Points {
		points = append(points, contracts.CPUHistoryPoint{
			Start: point.Start, End: point.End, State: historyPointState(point.State),
			ObservedSamples: point.ObservedSamples, AvailableSamples: point.AvailableSamples,
			Minimum: cloneFloat(point.Minimum), Average: cloneFloat(point.Average), Maximum: cloneFloat(point.Maximum),
		})
	}
	response := contracts.CPUHistorySeries{
		Resource: cpuResource(request.CoreIndex), Metric: request.Metric, CoreIndex: cloneInt(request.CoreIndex),
		Unit: series.Unit, Range: series.Range, BucketDurationSeconds: int64(series.BucketDuration / time.Second), Points: points,
	}
	if err := response.Validate(); err != nil {
		return contracts.CPUHistorySeries{}, fmt.Errorf("build CPU history response: %w", err)
	}
	return response, nil
}

func unavailableCPUSnapshot() contracts.CPUSnapshot {
	notCollected := domain.ReasonNotCollected
	return contracts.CPUSnapshot{
		Resource: cpuResource(nil), Freshness: domain.Freshness{State: domain.FreshnessUnavailable},
		Overall: unavailableMetric(domain.UnitPercent, notCollected), Cores: []contracts.CPUCore{},
		Load: contracts.LoadAverages{
			OneMinute: unavailableMetric(domain.UnitLoad, notCollected), FiveMinutes: unavailableMetric(domain.UnitLoad, notCollected),
			FifteenMinute: unavailableMetric(domain.UnitLoad, notCollected),
		},
	}
}

func historyMetric(request contracts.CPUHistoryRequest) (history.MetricKey, string) {
	switch request.Metric {
	case contracts.CPUMetricCore:
		return history.MetricCPUCoreUsagePercent, strconv.Itoa(*request.CoreIndex)
	case contracts.CPUMetricLoad1:
		return history.MetricLoad1, ""
	case contracts.CPUMetricLoad5:
		return history.MetricLoad5, ""
	case contracts.CPUMetricLoad15:
		return history.MetricLoad15, ""
	default:
		return history.MetricOverallCPUPercent, ""
	}
}

func historyPointState(state history.MetricPointState) contracts.CPUHistoryPointState {
	switch state {
	case history.MetricPointObserved:
		return contracts.CPUHistoryObserved
	case history.MetricPointUnavailable:
		return contracts.CPUHistoryUnavailable
	default:
		return contracts.CPUHistoryGap
	}
}

func cpuResource(coreIndex *int) domain.ResourceRef {
	if coreIndex == nil {
		return domain.ResourceRef{Kind: domain.ResourceCPU, ID: "host-cpu", DisplayName: "Overall CPU"}
	}
	return domain.ResourceRef{
		Kind: domain.ResourceCPU, ID: "cpu-core-" + strconv.Itoa(*coreIndex),
		DisplayName: "vCPU " + strconv.Itoa(*coreIndex),
	}
}

func cloneMetric(metric domain.Metric[float64]) domain.Metric[float64] {
	return domain.Metric[float64]{
		Availability: metric.Availability, Value: cloneFloat(metric.Value), Unit: metric.Unit,
		ReasonCode: metricReason(metric),
	}
}

func unavailableMetric(unit domain.Unit, reason domain.UnavailabilityReason) domain.Metric[float64] {
	return domain.Metric[float64]{Availability: domain.AvailabilityUnavailable, Unit: unit, ReasonCode: &reason}
}

func cloneFloat(value *float64) *float64 {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}
