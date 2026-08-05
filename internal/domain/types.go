package domain

import (
	"fmt"
	"strings"
	"time"
)

type HealthState string

const (
	HealthHealthy  HealthState = "healthy"
	HealthWarning  HealthState = "warning"
	HealthCritical HealthState = "critical"
	HealthUnknown  HealthState = "unknown"
)

func (state HealthState) Valid() bool {
	switch state {
	case HealthHealthy, HealthWarning, HealthCritical, HealthUnknown:
		return true
	default:
		return false
	}
}

type Availability string

const (
	AvailabilityAvailable   Availability = "available"
	AvailabilityUnavailable Availability = "unavailable"
)

func (availability Availability) Valid() bool {
	return availability == AvailabilityAvailable || availability == AvailabilityUnavailable
}

type FreshnessState string

const (
	FreshnessFresh       FreshnessState = "fresh"
	FreshnessStale       FreshnessState = "stale"
	FreshnessUnavailable FreshnessState = "unavailable"
)

func (state FreshnessState) Valid() bool {
	switch state {
	case FreshnessFresh, FreshnessStale, FreshnessUnavailable:
		return true
	default:
		return false
	}
}

type Unit string

const (
	UnitNone           Unit = "none"
	UnitPercent        Unit = "percent"
	UnitBytes          Unit = "bytes"
	UnitBytesPerSecond Unit = "bytes_per_second"
	UnitCount          Unit = "count"
	UnitSeconds        Unit = "seconds"
	UnitLoad           Unit = "load"
	UnitMicroseconds   Unit = "microseconds"
)

func (unit Unit) Valid() bool {
	switch unit {
	case UnitNone, UnitPercent, UnitBytes, UnitBytesPerSecond, UnitCount,
		UnitSeconds, UnitLoad, UnitMicroseconds:
		return true
	default:
		return false
	}
}

type UnavailabilityReason string

const (
	ReasonNotCollected   UnavailabilityReason = "not_collected"
	ReasonNotSupported   UnavailabilityReason = "not_supported"
	ReasonNotConfigured  UnavailabilityReason = "not_configured"
	ReasonCollectorError UnavailabilityReason = "collector_error"
	ReasonPermission     UnavailabilityReason = "permission_denied"
	ReasonDependencyDown UnavailabilityReason = "dependency_unavailable"
)

func (reason UnavailabilityReason) Valid() bool {
	switch reason {
	case ReasonNotCollected, ReasonNotSupported, ReasonNotConfigured,
		ReasonCollectorError, ReasonPermission, ReasonDependencyDown:
		return true
	default:
		return false
	}
}

type ResourceKind string

const (
	ResourceHost          ResourceKind = "host"
	ResourceCPU           ResourceKind = "cpu"
	ResourceMemory        ResourceKind = "memory"
	ResourceFilesystem    ResourceKind = "filesystem"
	ResourceBlockDevice   ResourceKind = "block_device"
	ResourceDocker        ResourceKind = "docker"
	ResourceContainer     ResourceKind = "container"
	ResourceHistoryDB     ResourceKind = "history_database"
	ResourceAuditDB       ResourceKind = "audit_database"
	ResourceConfiguration ResourceKind = "configuration"
	ResourceBackup        ResourceKind = "backup"
	ResourceExport        ResourceKind = "export"
)

func (kind ResourceKind) Valid() bool {
	switch kind {
	case ResourceHost, ResourceCPU, ResourceMemory, ResourceFilesystem,
		ResourceBlockDevice, ResourceDocker, ResourceContainer, ResourceHistoryDB,
		ResourceAuditDB, ResourceConfiguration, ResourceBackup, ResourceExport:
		return true
	default:
		return false
	}
}

type CauseCode string

const (
	CauseCPUWarning                CauseCode = "cpu_warning"
	CauseCPUCritical               CauseCode = "cpu_critical"
	CauseMemoryWarning             CauseCode = "memory_warning"
	CauseMemoryCritical            CauseCode = "memory_critical"
	CauseFilesystemWarning         CauseCode = "filesystem_warning"
	CauseFilesystemCritical        CauseCode = "filesystem_critical"
	CauseContainerUnhealthy        CauseCode = "container_unhealthy"
	CauseDockerUnavailable         CauseCode = "docker_unavailable"
	CauseHistoryUnavailable        CauseCode = "history_unavailable"
	CauseAuditUnavailable          CauseCode = "audit_unavailable"
	CauseConfigurationInvalid      CauseCode = "configuration_invalid"
	CauseCollectionStale           CauseCode = "collection_stale"
	CauseHostMeasurementsMissing   CauseCode = "host_measurements_unavailable"
	CauseMemoryPressureUnavailable CauseCode = "memory_pressure_unavailable"
	CauseStorageLimit              CauseCode = "storage_limit_reached"
)

func (code CauseCode) Valid() bool {
	switch code {
	case CauseCPUWarning, CauseCPUCritical, CauseMemoryWarning, CauseMemoryCritical,
		CauseFilesystemWarning, CauseFilesystemCritical, CauseContainerUnhealthy,
		CauseDockerUnavailable, CauseHistoryUnavailable, CauseAuditUnavailable,
		CauseConfigurationInvalid, CauseCollectionStale, CauseHostMeasurementsMissing,
		CauseMemoryPressureUnavailable, CauseStorageLimit:
		return true
	default:
		return false
	}
}

type ResourceRef struct {
	Kind        ResourceKind `json:"kind"`
	ID          string       `json:"id"`
	DisplayName string       `json:"displayName"`
}

func (resource ResourceRef) Validate() error {
	if !resource.Kind.Valid() {
		return fmt.Errorf("invalid resource kind %q", resource.Kind)
	}
	if err := ValidateOpaqueID(resource.ID); err != nil {
		return fmt.Errorf("resource id: %w", err)
	}
	if strings.TrimSpace(resource.DisplayName) == "" || len(resource.DisplayName) > 256 {
		return fmt.Errorf("resource display name must contain 1 to 256 characters")
	}
	return nil
}

type Freshness struct {
	State            FreshnessState `json:"state"`
	ObservedAt       *time.Time     `json:"observedAt"`
	LastSuccessfulAt *time.Time     `json:"lastSuccessfulAt"`
}

func (freshness Freshness) Validate() error {
	if !freshness.State.Valid() {
		return fmt.Errorf("invalid freshness state %q", freshness.State)
	}
	if err := ValidateOptionalUTC(freshness.ObservedAt); err != nil {
		return fmt.Errorf("observedAt: %w", err)
	}
	if err := ValidateOptionalUTC(freshness.LastSuccessfulAt); err != nil {
		return fmt.Errorf("lastSuccessfulAt: %w", err)
	}

	switch freshness.State {
	case FreshnessFresh:
		if freshness.ObservedAt == nil || freshness.LastSuccessfulAt == nil {
			return fmt.Errorf("fresh data requires observedAt and lastSuccessfulAt")
		}
	case FreshnessStale:
		if freshness.LastSuccessfulAt == nil {
			return fmt.Errorf("stale data requires lastSuccessfulAt")
		}
	case FreshnessUnavailable:
		if freshness.ObservedAt != nil || freshness.LastSuccessfulAt != nil {
			return fmt.Errorf("never-available data cannot contain observation timestamps")
		}
	}

	return nil
}

type AvailabilityInfo struct {
	State      Availability          `json:"state"`
	ReasonCode *UnavailabilityReason `json:"reasonCode"`
}

func (info AvailabilityInfo) Validate() error {
	if !info.State.Valid() {
		return fmt.Errorf("invalid availability state %q", info.State)
	}
	if info.State == AvailabilityAvailable && info.ReasonCode != nil {
		return fmt.Errorf("available state cannot contain a reason code")
	}
	if info.State == AvailabilityUnavailable {
		if info.ReasonCode == nil || !info.ReasonCode.Valid() {
			return fmt.Errorf("unavailable state requires a valid reason code")
		}
	}
	return nil
}

func ValidateOpaqueID(id string) error {
	if id == "" || len(id) > 128 {
		return fmt.Errorf("must contain 1 to 128 characters")
	}
	for _, character := range id {
		if character <= 0x20 || character == 0x7f {
			return fmt.Errorf("must not contain whitespace or control characters")
		}
	}
	return nil
}

func ValidateUTC(value time.Time) error {
	if value.IsZero() {
		return fmt.Errorf("timestamp must not be zero")
	}
	_, offset := value.Zone()
	if offset != 0 {
		return fmt.Errorf("timestamp must use UTC")
	}
	return nil
}

func ValidateOptionalUTC(value *time.Time) error {
	if value == nil {
		return nil
	}
	return ValidateUTC(*value)
}
