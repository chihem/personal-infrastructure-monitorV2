package app

import (
	"context"
	"fmt"

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
		snapshot, ok := completed.HostSnapshot.(hostcpu.Snapshot)
		if !ok {
			return fmt.Errorf("host collector returned unexpected snapshot type %T", completed.HostSnapshot)
		}
		if err := snapshot.Validate(); err != nil {
			return fmt.Errorf("validate CPU snapshot: %w", err)
		}
		record.Host = hostHistorySample(snapshot)
		record.CPUCores = coreHistorySamples(snapshot)
	}
	_, err := recorder.history.RecordCollection(ctx, record)
	return err
}

func hostHistorySample(snapshot hostcpu.Snapshot) *history.HostSampleRecord {
	notCollected := domain.ReasonNotCollected
	sample := &history.HostSampleRecord{
		ObservedAt:           snapshot.ObservedAt,
		OverallCPUPercent:    metricValue(snapshot.Overall),
		Load1:                metricValue(snapshot.Load.OneMinute),
		Load5:                metricValue(snapshot.Load.FiveMinutes),
		Load15:               metricValue(snapshot.Load.FifteenMinutes),
		PSIAvailability:      domain.AvailabilityUnavailable,
		PSIUnavailableReason: &notCollected,
	}
	if sample.OverallCPUPercent != nil || sample.Load1 != nil || sample.Load5 != nil || sample.Load15 != nil {
		sample.Availability = domain.AvailabilityAvailable
		return sample
	}
	sample.Availability = domain.AvailabilityUnavailable
	sample.UnavailableReason = metricReason(snapshot.Overall)
	if sample.UnavailableReason == nil {
		sample.UnavailableReason = &notCollected
	}
	return sample
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

func metricReason(metric domain.Metric[float64]) *domain.UnavailabilityReason {
	if metric.ReasonCode == nil {
		return nil
	}
	reason := *metric.ReasonCode
	return &reason
}

var _ scheduler.Recorder = collectionRecorder{}
