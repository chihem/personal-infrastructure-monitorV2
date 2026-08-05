package contracts

import (
	"fmt"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

type Page[T any] struct {
	Items []T      `json:"items"`
	Page  PageInfo `json:"page"`
}

func (page Page[T]) Validate(validateItem func(T) error) error {
	if page.Items == nil {
		return fmt.Errorf("items must be an array")
	}
	if err := page.Page.Validate(); err != nil {
		return fmt.Errorf("page: %w", err)
	}
	if validateItem != nil {
		for index, item := range page.Items {
			if err := validateItem(item); err != nil {
				return fmt.Errorf("items[%d]: %w", index, err)
			}
		}
	}
	return nil
}

type LogStream string

const (
	LogStdout LogStream = "stdout"
	LogStderr LogStream = "stderr"
)

type DockerLogLine struct {
	Sequence  uint64     `json:"sequence"`
	Stream    LogStream  `json:"stream"`
	Timestamp *time.Time `json:"timestamp"`
	Content   string     `json:"content"`
}

type DockerLogEventType string

const (
	DockerLogLineEvent  DockerLogEventType = "line"
	DockerLogErrorEvent DockerLogEventType = "error"
	DockerLogEndEvent   DockerLogEventType = "end"
)

type DockerLogEvent struct {
	Type  DockerLogEventType `json:"type"`
	Line  *DockerLogLine     `json:"line"`
	Error *APIError          `json:"error"`
}

type ExportFormat string

const (
	ExportCSV  ExportFormat = "csv"
	ExportJSON ExportFormat = "json"
)

type ExportDataset string

const (
	ExportCPU             ExportDataset = "cpu"
	ExportMemory          ExportDataset = "memory"
	ExportFilesystems     ExportDataset = "filesystems"
	ExportContainerUsage  ExportDataset = "container_usage"
	ExportContainerEvents ExportDataset = "container_events"
	ExportIncidents       ExportDataset = "incidents"
	ExportAudit           ExportDataset = "audit"
)

type ExportRequest struct {
	Format   ExportFormat          `json:"format"`
	Datasets []ExportDataset       `json:"datasets"`
	Range    domain.RangeSelection `json:"range"`
}

type ExportJob struct {
	ID          string               `json:"id"`
	Status      domain.ActionStatus  `json:"status"`
	Format      ExportFormat         `json:"format"`
	Datasets    []ExportDataset      `json:"datasets"`
	Range       domain.ResolvedRange `json:"range"`
	RequestedAt time.Time            `json:"requestedAt"`
	CompletedAt *time.Time           `json:"completedAt"`
	DownloadURL *string              `json:"downloadUrl"`
	Error       *APIError            `json:"error"`
}

type AuditOutcome string

const (
	AuditSucceeded AuditOutcome = "succeeded"
	AuditFailed    AuditOutcome = "failed"
	AuditRejected  AuditOutcome = "rejected"
)

type AuditEntry struct {
	ID          string              `json:"id"`
	RequestedAt time.Time           `json:"requestedAt"`
	CompletedAt *time.Time          `json:"completedAt"`
	SourceIP    string              `json:"sourceIp"`
	Action      domain.ActionKind   `json:"action"`
	Target      *domain.ResourceRef `json:"target"`
	Parameters  map[string]string   `json:"parameters"`
	Outcome     AuditOutcome        `json:"outcome"`
	ErrorCode   *domain.ErrorCode   `json:"errorCode"`
	ErrorDetail *string             `json:"errorDetail"`
}

type AuditDeleteScope string

const (
	AuditDeleteSelected AuditDeleteScope = "selected"
	AuditDeleteRange    AuditDeleteScope = "range"
	AuditDeleteAll      AuditDeleteScope = "all"
)

type AuditDeleteRequest struct {
	Scope AuditDeleteScope       `json:"scope"`
	IDs   []string               `json:"ids"`
	Range *domain.RangeSelection `json:"range"`
}

type BackupKind string

const (
	BackupScheduled BackupKind = "scheduled"
	BackupManual    BackupKind = "manual"
	BackupSafety    BackupKind = "safety"
)

type BackupStatus string

const (
	BackupPending   BackupStatus = "pending"
	BackupAvailable BackupStatus = "available"
	BackupInvalid   BackupStatus = "invalid"
	BackupFailed    BackupStatus = "failed"
)

type BackupRecord struct {
	Resource      domain.ResourceRef `json:"resource"`
	Kind          BackupKind         `json:"kind"`
	Status        BackupStatus       `json:"status"`
	CreatedAt     time.Time          `json:"createdAt"`
	SizeBytes     int64              `json:"sizeBytes"`
	FormatVersion int                `json:"formatVersion"`
	Checksum      string             `json:"checksum"`
	ErrorCode     *domain.ErrorCode  `json:"errorCode"`
}

type RecoveryMode string

const (
	RecoveryNormal      RecoveryMode = "normal"
	RecoveryRecommended RecoveryMode = "restore_recommended"
	RecoveryMaintenance RecoveryMode = "maintenance"
)

type RecoveryStatus struct {
	Mode                RecoveryMode            `json:"mode"`
	HistoryAvailability domain.AvailabilityInfo `json:"historyAvailability"`
	RecommendedBackup   *domain.ResourceRef     `json:"recommendedBackup"`
	ReasonCode          *domain.ErrorCode       `json:"reasonCode"`
}

type ConfirmationRequest struct {
	Action domain.ActionKind  `json:"action"`
	Target domain.ResourceRef `json:"target"`
}

type ConfirmationIntent struct {
	ID        string             `json:"id"`
	Action    domain.ActionKind  `json:"action"`
	Target    domain.ResourceRef `json:"target"`
	ExpiresAt time.Time          `json:"expiresAt"`
}

type ExecuteActionRequest struct {
	ConfirmationID string             `json:"confirmationId"`
	Action         domain.ActionKind  `json:"action"`
	Target         domain.ResourceRef `json:"target"`
}

type ActionResult struct {
	Action      domain.ActionKind   `json:"action"`
	Target      domain.ResourceRef  `json:"target"`
	Status      domain.ActionStatus `json:"status"`
	RequestedAt time.Time           `json:"requestedAt"`
	CompletedAt *time.Time          `json:"completedAt"`
	Error       *APIError           `json:"error"`
}
