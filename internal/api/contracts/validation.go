package contracts

import (
	"fmt"
	"math"
	"strings"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

func ValidateMonitoringEnvelope(envelope Envelope[MonitoringSnapshot]) error {
	return envelope.Validate(func(snapshot MonitoringSnapshot) error {
		return snapshot.Validate()
	})
}

func (snapshot MonitoringSnapshot) Validate() error {
	if err := snapshot.Freshness.Validate(); err != nil {
		return fmt.Errorf("freshness: %w", err)
	}
	if err := snapshot.Health.Validate(); err != nil {
		return fmt.Errorf("health: %w", err)
	}
	if err := snapshot.CPU.Validate(); err != nil {
		return fmt.Errorf("cpu: %w", err)
	}
	if err := snapshot.Memory.Validate(); err != nil {
		return fmt.Errorf("memory: %w", err)
	}
	if snapshot.Filesystems == nil {
		return fmt.Errorf("filesystems must be an array")
	}
	filesystemIDs := make(map[string]struct{}, len(snapshot.Filesystems))
	for index, filesystem := range snapshot.Filesystems {
		if err := filesystem.Validate(); err != nil {
			return fmt.Errorf("filesystems[%d]: %w", index, err)
		}
		if _, duplicate := filesystemIDs[filesystem.Resource.ID]; duplicate {
			return fmt.Errorf("duplicate filesystem id %q", filesystem.Resource.ID)
		}
		filesystemIDs[filesystem.Resource.ID] = struct{}{}
	}
	if err := snapshot.Docker.Validate(); err != nil {
		return fmt.Errorf("docker: %w", err)
	}
	if err := snapshot.Config.Validate(); err != nil {
		return fmt.Errorf("configuration: %w", err)
	}
	return nil
}

func (snapshot CPUSnapshot) Validate() error {
	if err := validateResourceKind(snapshot.Resource, domain.ResourceCPU); err != nil {
		return err
	}
	if err := snapshot.Freshness.Validate(); err != nil {
		return fmt.Errorf("freshness: %w", err)
	}
	if err := validatePercent(snapshot.Overall); err != nil {
		return fmt.Errorf("overall: %w", err)
	}
	if snapshot.LogicalCPU < 0 || len(snapshot.Cores) > snapshot.LogicalCPU {
		return fmt.Errorf("logicalCpuCount is inconsistent with cores")
	}
	if snapshot.Cores == nil {
		return fmt.Errorf("cores must be an array")
	}
	seen := make(map[int]struct{}, len(snapshot.Cores))
	for index, core := range snapshot.Cores {
		if core.Index < 0 {
			return fmt.Errorf("cores[%d].index cannot be negative", index)
		}
		if _, duplicate := seen[core.Index]; duplicate {
			return fmt.Errorf("duplicate core index %d", core.Index)
		}
		seen[core.Index] = struct{}{}
		if err := validatePercent(core.Usage); err != nil {
			return fmt.Errorf("cores[%d].usage: %w", index, err)
		}
	}
	for name, metric := range map[string]domain.Metric[float64]{
		"oneMinute":      snapshot.Load.OneMinute,
		"fiveMinutes":    snapshot.Load.FiveMinutes,
		"fifteenMinutes": snapshot.Load.FifteenMinute,
	} {
		if err := validateMetric(metric, domain.UnitLoad); err != nil {
			return fmt.Errorf("load.%s: %w", name, err)
		}
	}
	return nil
}

func (snapshot MemorySnapshot) Validate() error {
	if err := validateResourceKind(snapshot.Resource, domain.ResourceMemory); err != nil {
		return err
	}
	if err := snapshot.Freshness.Validate(); err != nil {
		return fmt.Errorf("freshness: %w", err)
	}
	for name, metric := range map[string]domain.Metric[int64]{
		"total":      snapshot.Total,
		"used":       snapshot.Used,
		"available":  snapshot.Available,
		"free":       snapshot.Free,
		"cached":     snapshot.Cached,
		"buffered":   snapshot.Buffered,
		"swap.total": snapshot.Swap.Total,
		"swap.used":  snapshot.Swap.Used,
		"swap.free":  snapshot.Swap.Free,
	} {
		if err := validateIntegerMetric(metric, domain.UnitBytes); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	if err := validatePercent(snapshot.Usage); err != nil {
		return fmt.Errorf("usage: %w", err)
	}
	for name, window := range map[string]PressureWindow{
		"some": snapshot.Pressure.Some,
		"full": snapshot.Pressure.Full,
	} {
		for field, metric := range map[string]domain.Metric[float64]{
			"average10Seconds":  window.Average10Seconds,
			"average60Seconds":  window.Average60Seconds,
			"average300Seconds": window.Average300Seconds,
		} {
			if err := validatePercent(metric); err != nil {
				return fmt.Errorf("pressure.%s.%s: %w", name, field, err)
			}
		}
		if err := validateIntegerMetric(window.Total, domain.UnitMicroseconds); err != nil {
			return fmt.Errorf("pressure.%s.total: %w", name, err)
		}
	}
	return nil
}

func (snapshot FilesystemSnapshot) Validate() error {
	if err := validateResourceKind(snapshot.Resource, domain.ResourceFilesystem); err != nil {
		return err
	}
	if err := snapshot.Freshness.Validate(); err != nil {
		return fmt.Errorf("freshness: %w", err)
	}
	if strings.TrimSpace(snapshot.MountPath) == "" || len(snapshot.MountPath) > 4096 {
		return fmt.Errorf("mountPath must contain 1 to 4096 characters")
	}
	if len(snapshot.DeviceName) > 512 || strings.TrimSpace(snapshot.FilesystemType) == "" {
		return fmt.Errorf("invalid filesystem identity")
	}
	for name, metric := range map[string]domain.Metric[int64]{
		"total": snapshot.Total,
		"used":  snapshot.Used,
		"free":  snapshot.Free,
	} {
		if err := validateIntegerMetric(metric, domain.UnitBytes); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	if err := validatePercent(snapshot.Usage); err != nil {
		return fmt.Errorf("usage: %w", err)
	}
	for name, metric := range map[string]domain.Metric[float64]{
		"readRate":  snapshot.IO.ReadRate,
		"writeRate": snapshot.IO.WriteRate,
	} {
		if err := validateMetric(metric, domain.UnitBytesPerSecond); err != nil {
			return fmt.Errorf("io.%s: %w", name, err)
		}
	}
	return nil
}

func (snapshot DockerSnapshot) Validate() error {
	if err := validateResourceKind(snapshot.Resource, domain.ResourceDocker); err != nil {
		return err
	}
	if err := snapshot.Freshness.Validate(); err != nil {
		return fmt.Errorf("freshness: %w", err)
	}
	if err := snapshot.Communication.Validate(); err != nil {
		return fmt.Errorf("communication: %w", err)
	}
	if snapshot.Containers == nil {
		return fmt.Errorf("containers must be an array")
	}
	seen := make(map[string]struct{}, len(snapshot.Containers))
	for index, container := range snapshot.Containers {
		if err := container.Validate(); err != nil {
			return fmt.Errorf("containers[%d]: %w", index, err)
		}
		if _, duplicate := seen[container.Resource.ID]; duplicate {
			return fmt.Errorf("duplicate container id %q", container.Resource.ID)
		}
		seen[container.Resource.ID] = struct{}{}
	}
	return nil
}

func (snapshot ContainerSnapshot) Validate() error {
	if err := validateResourceKind(snapshot.Resource, domain.ResourceContainer); err != nil {
		return err
	}
	if err := snapshot.Freshness.Validate(); err != nil {
		return fmt.Errorf("freshness: %w", err)
	}
	if strings.TrimSpace(snapshot.Name) == "" || len(snapshot.Name) > 256 {
		return fmt.Errorf("name must contain 1 to 256 characters")
	}
	if !snapshot.State.Valid() {
		return fmt.Errorf("invalid container state %q", snapshot.State)
	}
	if !snapshot.Health.Valid() {
		return fmt.Errorf("invalid container health %q", snapshot.Health)
	}
	if err := validateIntegerMetric(snapshot.Uptime, domain.UnitSeconds); err != nil {
		return fmt.Errorf("uptime: %w", err)
	}
	if err := validateIntegerMetric(snapshot.RestartCount, domain.UnitCount); err != nil {
		return fmt.Errorf("restartCount: %w", err)
	}
	if err := validateMetric(snapshot.CPUUsage, domain.UnitPercent); err != nil {
		return fmt.Errorf("cpuUsage: %w", err)
	}
	if err := validateIntegerMetric(snapshot.MemoryUsage, domain.UnitBytes); err != nil {
		return fmt.Errorf("memoryUsage: %w", err)
	}
	if snapshot.Ports == nil {
		return fmt.Errorf("ports must be an array")
	}
	for index, port := range snapshot.Ports {
		if port.Protocol != "tcp" && port.Protocol != "udp" && port.Protocol != "sctp" {
			return fmt.Errorf("ports[%d] has invalid protocol", index)
		}
		if port.ContainerPort == 0 {
			return fmt.Errorf("ports[%d] requires containerPort", index)
		}
	}
	return nil
}

func (status ConfigurationStatus) Validate() error {
	if err := validateResourceKind(status.Resource, domain.ResourceConfiguration); err != nil {
		return err
	}
	if !status.State.Valid() {
		return fmt.Errorf("invalid configuration state %q", status.State)
	}
	if err := domain.ValidateOptionalUTC(status.LoadedAt); err != nil {
		return fmt.Errorf("loadedAt: %w", err)
	}
	if status.State == ConfigurationValid {
		if status.LoadedAt == nil || status.ErrorCode != nil || status.ErrorMessageKey != nil {
			return fmt.Errorf("valid configuration requires loadedAt and no error")
		}
		return nil
	}
	if status.ErrorCode == nil || !status.ErrorCode.Valid() || status.ErrorMessageKey == nil {
		return fmt.Errorf("non-valid configuration requires errorCode and errorMessageKey")
	}
	return validateMessage(*status.ErrorMessageKey, 160, "errorMessageKey")
}

func (series ChartSeries) Validate() error {
	if err := series.Resource.Validate(); err != nil {
		return fmt.Errorf("resource: %w", err)
	}
	if err := validateMessage(series.Metric, 160, "metric"); err != nil {
		return err
	}
	if err := series.Range.Validate(); err != nil {
		return fmt.Errorf("range: %w", err)
	}
	for index, point := range series.Points {
		if err := point.Validate(); err != nil {
			return fmt.Errorf("points[%d]: %w", index, err)
		}
	}
	return nil
}

func (point ChartPoint) Validate() error {
	if err := domain.ValidateUTC(point.Timestamp); err != nil {
		return fmt.Errorf("timestamp: %w", err)
	}
	switch point.State {
	case ChartObserved:
		if point.Measurement == nil {
			return fmt.Errorf("observed point requires measurement")
		}
		return point.Measurement.Validate()
	case ChartGap:
		if point.Measurement != nil {
			return fmt.Errorf("gap point cannot contain measurement")
		}
		return nil
	default:
		return fmt.Errorf("invalid chart point state %q", point.State)
	}
}

func validateResourceKind(resource domain.ResourceRef, expected domain.ResourceKind) error {
	if err := resource.Validate(); err != nil {
		return fmt.Errorf("resource: %w", err)
	}
	if resource.Kind != expected {
		return fmt.Errorf("resource kind must be %q", expected)
	}
	return nil
}

func validatePercent(metric domain.Metric[float64]) error {
	if err := validateMetric(metric, domain.UnitPercent); err != nil {
		return err
	}
	if metric.Value != nil && (math.IsNaN(*metric.Value) || math.IsInf(*metric.Value, 0) || *metric.Value < 0 || *metric.Value > 100) {
		return fmt.Errorf("percentage must be finite and between 0 and 100")
	}
	return nil
}

func validateMetric(metric domain.Metric[float64], unit domain.Unit) error {
	if err := metric.Validate(); err != nil {
		return err
	}
	if metric.Unit != unit {
		return fmt.Errorf("unit must be %q", unit)
	}
	if metric.Value != nil && (math.IsNaN(*metric.Value) || math.IsInf(*metric.Value, 0) || *metric.Value < 0) {
		return fmt.Errorf("metric value must be finite and non-negative")
	}
	return nil
}

func validateIntegerMetric(metric domain.Metric[int64], unit domain.Unit) error {
	if err := metric.Validate(); err != nil {
		return err
	}
	if metric.Unit != unit {
		return fmt.Errorf("unit must be %q", unit)
	}
	if metric.Value != nil && *metric.Value < 0 {
		return fmt.Errorf("metric value cannot be negative")
	}
	return nil
}
