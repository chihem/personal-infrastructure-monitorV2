package app

import (
	"context"
	"fmt"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/collector/host"
	hostcpu "github.com/chihem/personal-infrastructure-monitorV2/internal/collector/host/cpu"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/scheduler"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/storage/history"
)

type collectionRecorder struct {
	history *history.Repository
}

func (recorder collectionRecorder) RecordCollection(ctx context.Context, completed scheduler.CompletedRun) error {
	record := history.CollectionRecord{Run: completed.Record}
	if completed.Record.HostResult.Status != domain.CollectionFailed {
		snapshot, ok := completed.HostSnapshot.(host.Snapshot)
		if !ok {
			return fmt.Errorf("host collector returned unexpected snapshot type %T", completed.HostSnapshot)
		}
		if err := snapshot.Validate(); err != nil {
			return fmt.Errorf("validate host snapshot: %w", err)
		}
		record.Host = hostHistorySample(snapshot)
		if snapshot.CPU != nil {
			record.CPUCores = coreHistorySamples(*snapshot.CPU)
		}
	}
	_, err := recorder.history.RecordCollection(ctx, record)
	return err
}

func hostHistorySample(snapshot host.Snapshot) *history.HostSampleRecord {
	notCollected := domain.ReasonNotCollected
	sample := &history.HostSampleRecord{
		PSIAvailability:      domain.AvailabilityUnavailable,
		PSIUnavailableReason: &notCollected,
	}
	if snapshot.CPU != nil {
		sample.ObservedAt = snapshot.CPU.ObservedAt
		sample.OverallCPUPercent = metricValue(snapshot.CPU.Overall)
		sample.Load1 = metricValue(snapshot.CPU.Load.OneMinute)
		sample.Load5 = metricValue(snapshot.CPU.Load.FiveMinutes)
		sample.Load15 = metricValue(snapshot.CPU.Load.FifteenMinutes)
	}
	if snapshot.Memory != nil {
		memory := snapshot.Memory
		if sample.ObservedAt.IsZero() || memory.ObservedAt.After(sample.ObservedAt) {
			sample.ObservedAt = memory.ObservedAt
		}
		sample.MemoryTotalBytes = integerMetricValue(memory.Total)
		sample.MemoryUsedBytes = integerMetricValue(memory.Used)
		sample.MemoryAvailableBytes = integerMetricValue(memory.Available)
		sample.MemoryFreeBytes = integerMetricValue(memory.Free)
		sample.MemoryCachedBytes = integerMetricValue(memory.Cached)
		sample.MemoryBufferedBytes = integerMetricValue(memory.Buffered)
		sample.MemoryUsagePercent = metricValue(memory.Usage)
		sample.SwapTotalBytes = integerMetricValue(memory.Swap.Total)
		sample.SwapUsedBytes = integerMetricValue(memory.Swap.Used)
		sample.PSIAvailability = memory.Pressure.Availability
		sample.PSIUnavailableReason = cloneReason(memory.Pressure.ReasonCode)
		sample.MemoryPSISomeAverage10 = metricValue(memory.Pressure.Some.Average10Seconds)
		sample.MemoryPSIFullAverage10 = metricValue(memory.Pressure.Full.Average10Seconds)
		sample.MemoryPSISomeTotalUS = integerMetricValue(memory.Pressure.Some.Total)
		sample.MemoryPSIFullTotalUS = integerMetricValue(memory.Pressure.Full.Total)
	}
	if anyHostValue(sample) {
		sample.Availability = domain.AvailabilityAvailable
		return sample
	}
	sample.Availability = domain.AvailabilityUnavailable
	sample.UnavailableReason = &notCollected
	return sample
}

func anyHostValue(sample *history.HostSampleRecord) bool {
	return sample.OverallCPUPercent != nil || sample.Load1 != nil || sample.Load5 != nil || sample.Load15 != nil ||
		sample.MemoryTotalBytes != nil || sample.MemoryUsedBytes != nil || sample.MemoryAvailableBytes != nil ||
		sample.MemoryFreeBytes != nil || sample.MemoryCachedBytes != nil || sample.MemoryBufferedBytes != nil ||
		sample.MemoryUsagePercent != nil || sample.SwapTotalBytes != nil || sample.SwapUsedBytes != nil
}

func coreHistorySamples(snapshot hostcpu.Snapshot) []history.CPUCoreSampleRecord {
	samples := make([]history.CPUCoreSampleRecord, 0, len(snapshot.Cores))
	for _, core := range snapshot.Cores {
		samples = append(samples, history.CPUCoreSampleRecord{
			LogicalIndex:      core.LogicalIndex,
			ObservedAt:        snapshot.ObservedAt,
			Availability:      core.Usage.Availability,
			UnavailableReason: metricReason(core.Usage),
			UsagePercent:      metricValue(core.Usage),
		})
	}
	return samples
}

func metricValue(metric domain.Metric[float64]) *float64 {
	if metric.Value == nil {
		return nil
	}
	value := *metric.Value
	return &value
}

func metricReason[T any](metric domain.Metric[T]) *domain.UnavailabilityReason {
	if metric.ReasonCode == nil {
		return nil
	}
	reason := *metric.ReasonCode
	return &reason
}

func integerMetricValue(metric domain.Metric[int64]) *int64 {
	if metric.Value == nil {
		return nil
	}
	value := *metric.Value
	return &value
}

func cloneReason(reason *domain.UnavailabilityReason) *domain.UnavailabilityReason {
	if reason == nil {
		return nil
	}
	value := *reason
	return &value
}

var _ scheduler.Recorder = collectionRecorder{}
