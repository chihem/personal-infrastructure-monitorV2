package contracts

import (
	"fmt"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

type OperationalState string

const (
	OperationalOK          OperationalState = "ok"
	OperationalDegraded    OperationalState = "degraded"
	OperationalMaintenance OperationalState = "maintenance"
	OperationalNotReady    OperationalState = "not_ready"
)

func (state OperationalState) Valid() bool {
	switch state {
	case OperationalOK, OperationalDegraded, OperationalMaintenance, OperationalNotReady:
		return true
	default:
		return false
	}
}

type DependencyState string

const (
	DependencyAvailable      DependencyState = "available"
	DependencyUnavailable    DependencyState = "unavailable"
	DependencyNotStarted     DependencyState = "not_started"
	DependencyNotImplemented DependencyState = "not_implemented"
	DependencyNotChecked     DependencyState = "not_checked"
)

func (state DependencyState) Valid() bool {
	switch state {
	case DependencyAvailable, DependencyUnavailable, DependencyNotStarted,
		DependencyNotImplemented, DependencyNotChecked:
		return true
	default:
		return false
	}
}

type LivenessStatus struct {
	Alive bool `json:"alive"`
}

func ValidateLivenessStatus(status LivenessStatus) error {
	return status.Validate()
}

func (status LivenessStatus) Validate() error {
	if !status.Alive {
		return fmt.Errorf("a served liveness response must report alive")
	}
	return nil
}

type ReadinessStatus struct {
	Ready                    bool               `json:"ready"`
	Maintenance              bool               `json:"maintenance"`
	ConfigurationState       ConfigurationState `json:"configurationState"`
	HistoryDatabaseAvailable bool               `json:"historyDatabaseAvailable"`
}

func ValidateReadinessStatus(status ReadinessStatus) error {
	return status.Validate()
}

func (status ReadinessStatus) Validate() error {
	if !status.ConfigurationState.Valid() {
		return fmt.Errorf("invalid configuration state %q", status.ConfigurationState)
	}
	expected := !status.Maintenance && status.HistoryDatabaseAvailable && status.ConfigurationState != ConfigurationUnavailable
	if status.Ready != expected {
		return fmt.Errorf("ready does not match required dependencies")
	}
	return nil
}

type DatabaseOperationalStatus struct {
	State     DependencyState `json:"state"`
	SizeBytes *int64          `json:"sizeBytes"`
}

func (status DatabaseOperationalStatus) Validate() error {
	if status.State != DependencyAvailable && status.State != DependencyUnavailable {
		return fmt.Errorf("database state must be available or unavailable")
	}
	if status.SizeBytes != nil && *status.SizeBytes < 0 {
		return fmt.Errorf("database size must not be negative")
	}
	if status.State == DependencyUnavailable && status.SizeBytes != nil {
		return fmt.Errorf("unavailable database cannot report a size")
	}
	return nil
}

type CollectionRunOperationalStatus struct {
	StartedAt  time.Time               `json:"startedAt"`
	FinishedAt time.Time               `json:"finishedAt"`
	DurationMS int64                   `json:"durationMs"`
	Status     domain.CollectionStatus `json:"status"`
}

func (status CollectionRunOperationalStatus) Validate() error {
	if err := domain.ValidateUTC(status.StartedAt); err != nil {
		return fmt.Errorf("startedAt: %w", err)
	}
	if err := domain.ValidateUTC(status.FinishedAt); err != nil {
		return fmt.Errorf("finishedAt: %w", err)
	}
	if status.FinishedAt.Before(status.StartedAt) {
		return fmt.Errorf("finishedAt must not be before startedAt")
	}
	if status.DurationMS < 0 || status.DurationMS != status.FinishedAt.Sub(status.StartedAt).Milliseconds() {
		return fmt.Errorf("durationMs must match collection timestamps")
	}
	if !status.Status.Valid() || status.Status == domain.CollectionNotAttempted {
		return fmt.Errorf("invalid completed collection status %q", status.Status)
	}
	return nil
}

type CollectionOperationalStatus struct {
	State            DependencyState                 `json:"state"`
	InProgress       bool                            `json:"inProgress"`
	LastRun          *CollectionRunOperationalStatus `json:"lastRun"`
	LastSuccessfulAt *time.Time                      `json:"lastSuccessfulAt"`
}

func (status CollectionOperationalStatus) Validate() error {
	if status.State != DependencyAvailable && status.State != DependencyNotStarted && status.State != DependencyUnavailable {
		return fmt.Errorf("invalid collection state %q", status.State)
	}
	if status.State == DependencyNotStarted && (status.InProgress || status.LastRun != nil || status.LastSuccessfulAt != nil) {
		return fmt.Errorf("not-started collection cannot contain run state")
	}
	if status.LastRun != nil {
		if err := status.LastRun.Validate(); err != nil {
			return fmt.Errorf("lastRun: %w", err)
		}
	}
	if err := domain.ValidateOptionalUTC(status.LastSuccessfulAt); err != nil {
		return fmt.Errorf("lastSuccessfulAt: %w", err)
	}
	return nil
}

type OperationalStatus struct {
	State              OperationalState            `json:"state"`
	UptimeSeconds      int64                       `json:"uptimeSeconds"`
	Maintenance        bool                        `json:"maintenance"`
	ConfigurationState ConfigurationState          `json:"configurationState"`
	HistoryDatabase    DatabaseOperationalStatus   `json:"historyDatabase"`
	AuditDatabase      DatabaseOperationalStatus   `json:"auditDatabase"`
	Collection         CollectionOperationalStatus `json:"collection"`
	BackupState        DependencyState             `json:"backupState"`
	DockerConnectivity DependencyState             `json:"dockerConnectivity"`
}

func ValidateOperationalStatus(status OperationalStatus) error {
	return status.Validate()
}

func (status OperationalStatus) Validate() error {
	if !status.State.Valid() {
		return fmt.Errorf("invalid operational state %q", status.State)
	}
	if status.UptimeSeconds < 0 {
		return fmt.Errorf("uptimeSeconds must not be negative")
	}
	if !status.ConfigurationState.Valid() {
		return fmt.Errorf("invalid configuration state %q", status.ConfigurationState)
	}
	if err := status.HistoryDatabase.Validate(); err != nil {
		return fmt.Errorf("historyDatabase: %w", err)
	}
	if err := status.AuditDatabase.Validate(); err != nil {
		return fmt.Errorf("auditDatabase: %w", err)
	}
	if err := status.Collection.Validate(); err != nil {
		return fmt.Errorf("collection: %w", err)
	}
	if status.BackupState != DependencyAvailable && status.BackupState != DependencyUnavailable && status.BackupState != DependencyNotImplemented {
		return fmt.Errorf("invalid backup state %q", status.BackupState)
	}
	if status.DockerConnectivity != DependencyAvailable && status.DockerConnectivity != DependencyUnavailable && status.DockerConnectivity != DependencyNotChecked {
		return fmt.Errorf("invalid Docker connectivity state %q", status.DockerConnectivity)
	}
	requiredUnavailable := status.ConfigurationState == ConfigurationUnavailable || status.HistoryDatabase.State == DependencyUnavailable
	expectedState := OperationalOK
	switch {
	case status.Maintenance:
		expectedState = OperationalMaintenance
	case requiredUnavailable:
		expectedState = OperationalNotReady
	case status.ConfigurationState != ConfigurationValid ||
		status.AuditDatabase.State != DependencyAvailable ||
		status.Collection.State != DependencyAvailable ||
		status.BackupState != DependencyAvailable ||
		status.DockerConnectivity != DependencyAvailable:
		expectedState = OperationalDegraded
	}
	if status.State != expectedState {
		return fmt.Errorf("operational state %q does not match dependency state %q", status.State, expectedState)
	}
	return nil
}
