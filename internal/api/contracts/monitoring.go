package contracts

import (
	"fmt"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

type HealthCause struct {
	Code       domain.CauseCode   `json:"code"`
	State      domain.HealthState `json:"state"`
	Resource   domain.ResourceRef `json:"resource"`
	StartedAt  time.Time          `json:"startedAt"`
	MessageKey string             `json:"messageKey"`
}

func (cause HealthCause) Validate() error {
	if cause.State == domain.HealthHealthy || !cause.State.Valid() {
		return fmt.Errorf("cause state must be warning, critical, or unknown")
	}
	if !cause.Code.Valid() {
		return fmt.Errorf("invalid cause code %q", cause.Code)
	}
	if err := cause.Resource.Validate(); err != nil {
		return fmt.Errorf("resource: %w", err)
	}
	if err := domain.ValidateUTC(cause.StartedAt); err != nil {
		return fmt.Errorf("startedAt: %w", err)
	}
	return validateMessage(cause.MessageKey, 160, "messageKey")
}

type HealthSummary struct {
	States             []domain.HealthState `json:"states"`
	ActiveWarningCount int                  `json:"activeWarningCount"`
	Causes             []HealthCause        `json:"causes"`
}

func (summary HealthSummary) Validate() error {
	if len(summary.States) == 0 {
		return fmt.Errorf("health states cannot be empty")
	}
	if summary.Causes == nil {
		return fmt.Errorf("health causes must be an array")
	}
	seen := make(map[domain.HealthState]struct{}, len(summary.States))
	for _, state := range summary.States {
		if !state.Valid() {
			return fmt.Errorf("invalid health state %q", state)
		}
		if _, duplicate := seen[state]; duplicate {
			return fmt.Errorf("duplicate health state %q", state)
		}
		seen[state] = struct{}{}
	}
	if _, healthy := seen[domain.HealthHealthy]; healthy && len(seen) != 1 {
		return fmt.Errorf("healthy cannot be combined with problem states")
	}
	if summary.ActiveWarningCount < 0 {
		return fmt.Errorf("activeWarningCount cannot be negative")
	}
	for index, cause := range summary.Causes {
		if err := cause.Validate(); err != nil {
			return fmt.Errorf("causes[%d]: %w", index, err)
		}
		if _, present := seen[cause.State]; !present {
			return fmt.Errorf("cause state %q is absent from health states", cause.State)
		}
	}
	for state := range seen {
		if state == domain.HealthHealthy {
			continue
		}
		found := false
		for _, cause := range summary.Causes {
			if cause.State == state {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("health state %q requires a matching cause", state)
		}
	}
	warningCount := 0
	for _, cause := range summary.Causes {
		if cause.State == domain.HealthWarning {
			warningCount++
		}
	}
	if summary.ActiveWarningCount != warningCount {
		return fmt.Errorf("activeWarningCount does not match warning causes")
	}
	if len(summary.Causes) == 0 {
		if _, healthy := seen[domain.HealthHealthy]; !healthy {
			return fmt.Errorf("problem health requires at least one cause")
		}
	}
	return nil
}

type CPUCore struct {
	Index int                    `json:"index"`
	Usage domain.Metric[float64] `json:"usage"`
}

type LoadAverages struct {
	OneMinute     domain.Metric[float64] `json:"oneMinute"`
	FiveMinutes   domain.Metric[float64] `json:"fiveMinutes"`
	FifteenMinute domain.Metric[float64] `json:"fifteenMinutes"`
}

type CPUSnapshot struct {
	Resource   domain.ResourceRef     `json:"resource"`
	Freshness  domain.Freshness       `json:"freshness"`
	Overall    domain.Metric[float64] `json:"overall"`
	Cores      []CPUCore              `json:"cores"`
	Load       LoadAverages           `json:"load"`
	LogicalCPU int                    `json:"logicalCpuCount"`
}

type PressureWindow struct {
	Average10Seconds  domain.Metric[float64] `json:"average10Seconds"`
	Average60Seconds  domain.Metric[float64] `json:"average60Seconds"`
	Average300Seconds domain.Metric[float64] `json:"average300Seconds"`
	Total             domain.Metric[int64]   `json:"total"`
}

type MemoryPressure struct {
	Some PressureWindow `json:"some"`
	Full PressureWindow `json:"full"`
}

type SwapSnapshot struct {
	Configured *bool                `json:"configured"`
	Total      domain.Metric[int64] `json:"total"`
	Used       domain.Metric[int64] `json:"used"`
	Free       domain.Metric[int64] `json:"free"`
}

type MemorySnapshot struct {
	Resource  domain.ResourceRef     `json:"resource"`
	Freshness domain.Freshness       `json:"freshness"`
	Total     domain.Metric[int64]   `json:"total"`
	Used      domain.Metric[int64]   `json:"used"`
	Available domain.Metric[int64]   `json:"available"`
	Free      domain.Metric[int64]   `json:"free"`
	Cached    domain.Metric[int64]   `json:"cached"`
	Buffered  domain.Metric[int64]   `json:"buffered"`
	Usage     domain.Metric[float64] `json:"usage"`
	Swap      SwapSnapshot           `json:"swap"`
	Pressure  MemoryPressure         `json:"pressure"`
}

type BlockIO struct {
	ReadRate  domain.Metric[float64] `json:"readRate"`
	WriteRate domain.Metric[float64] `json:"writeRate"`
}

type FilesystemSnapshot struct {
	Resource       domain.ResourceRef     `json:"resource"`
	Freshness      domain.Freshness       `json:"freshness"`
	MountPath      string                 `json:"mountPath"`
	DeviceName     string                 `json:"deviceName"`
	FilesystemType string                 `json:"filesystemType"`
	Mounted        bool                   `json:"mounted"`
	ReadOnly       bool                   `json:"readOnly"`
	Total          domain.Metric[int64]   `json:"total"`
	Used           domain.Metric[int64]   `json:"used"`
	Free           domain.Metric[int64]   `json:"free"`
	Usage          domain.Metric[float64] `json:"usage"`
	IO             BlockIO                `json:"io"`
}

type ContainerState string

const (
	ContainerRunning    ContainerState = "running"
	ContainerStopped    ContainerState = "stopped"
	ContainerPaused     ContainerState = "paused"
	ContainerRestarting ContainerState = "restarting"
	ContainerOther      ContainerState = "other"
)

func (state ContainerState) Valid() bool {
	switch state {
	case ContainerRunning, ContainerStopped, ContainerPaused, ContainerRestarting, ContainerOther:
		return true
	default:
		return false
	}
}

type ContainerHealth string

const (
	ContainerHealthy       ContainerHealth = "healthy"
	ContainerUnhealthy     ContainerHealth = "unhealthy"
	ContainerStarting      ContainerHealth = "starting"
	ContainerNotConfigured ContainerHealth = "not_configured"
	ContainerHealthUnknown ContainerHealth = "unavailable"
)

func (health ContainerHealth) Valid() bool {
	switch health {
	case ContainerHealthy, ContainerUnhealthy, ContainerStarting,
		ContainerNotConfigured, ContainerHealthUnknown:
		return true
	default:
		return false
	}
}

type PublishedPort struct {
	Protocol      string `json:"protocol"`
	ContainerPort uint16 `json:"containerPort"`
	HostIP        string `json:"hostIp"`
	HostPort      uint16 `json:"hostPort"`
}

type ContainerSnapshot struct {
	Resource     domain.ResourceRef     `json:"resource"`
	Freshness    domain.Freshness       `json:"freshness"`
	Name         string                 `json:"name"`
	State        ContainerState         `json:"state"`
	Health       ContainerHealth        `json:"health"`
	Deleted      bool                   `json:"deleted"`
	Uptime       domain.Metric[int64]   `json:"uptime"`
	RestartCount domain.Metric[int64]   `json:"restartCount"`
	CPUUsage     domain.Metric[float64] `json:"cpuUsage"`
	MemoryUsage  domain.Metric[int64]   `json:"memoryUsage"`
	Ports        []PublishedPort        `json:"ports"`
}

type DockerSnapshot struct {
	Resource      domain.ResourceRef      `json:"resource"`
	Freshness     domain.Freshness        `json:"freshness"`
	Communication domain.AvailabilityInfo `json:"communication"`
	Containers    []ContainerSnapshot     `json:"containers"`
}

type ConfigurationState string

const (
	ConfigurationValid         ConfigurationState = "valid"
	ConfigurationUsingPrevious ConfigurationState = "using_previous"
	ConfigurationUnavailable   ConfigurationState = "unavailable"
)

func (state ConfigurationState) Valid() bool {
	switch state {
	case ConfigurationValid, ConfigurationUsingPrevious, ConfigurationUnavailable:
		return true
	default:
		return false
	}
}

type ConfigurationStatus struct {
	Resource        domain.ResourceRef `json:"resource"`
	State           ConfigurationState `json:"state"`
	LoadedAt        *time.Time         `json:"loadedAt"`
	ErrorCode       *domain.ErrorCode  `json:"errorCode"`
	ErrorMessageKey *string            `json:"errorMessageKey"`
}

type MonitoringSnapshot struct {
	Freshness   domain.Freshness     `json:"freshness"`
	Health      HealthSummary        `json:"health"`
	CPU         CPUSnapshot          `json:"cpu"`
	Memory      MemorySnapshot       `json:"memory"`
	Filesystems []FilesystemSnapshot `json:"filesystems"`
	Docker      DockerSnapshot       `json:"docker"`
	Config      ConfigurationStatus  `json:"configuration"`
}

type ChartPointState string

const (
	ChartObserved ChartPointState = "observed"
	ChartGap      ChartPointState = "gap"
)

type ChartPoint struct {
	Timestamp   time.Time               `json:"timestamp"`
	State       ChartPointState         `json:"state"`
	Measurement *domain.Metric[float64] `json:"measurement"`
}

type ChartSeries struct {
	Resource domain.ResourceRef   `json:"resource"`
	Metric   string               `json:"metric"`
	Range    domain.ResolvedRange `json:"range"`
	Points   []ChartPoint         `json:"points"`
}

type MetricStatistics struct {
	Minimum domain.Metric[float64] `json:"minimum"`
	Average domain.Metric[float64] `json:"average"`
	Maximum domain.Metric[float64] `json:"maximum"`
}

type Incident struct {
	ID        string             `json:"id"`
	Severity  domain.HealthState `json:"severity"`
	CauseCode domain.CauseCode   `json:"causeCode"`
	Resource  domain.ResourceRef `json:"resource"`
	StartedAt time.Time          `json:"startedAt"`
	EndedAt   *time.Time         `json:"endedAt"`
	Active    bool               `json:"active"`
}

type ContainerStateEvent struct {
	ID        string             `json:"id"`
	Container domain.ResourceRef `json:"container"`
	Timestamp time.Time          `json:"timestamp"`
	State     ContainerState     `json:"state"`
	Health    ContainerHealth    `json:"health"`
}
