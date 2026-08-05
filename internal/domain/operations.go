package domain

type ActionKind string

const (
	ActionDockerStart   ActionKind = "docker.start"
	ActionDockerStop    ActionKind = "docker.stop"
	ActionDockerRestart ActionKind = "docker.restart"
	ActionBackupCreate  ActionKind = "backup.create"
	ActionBackupRestore ActionKind = "backup.restore"
	ActionExportCreate  ActionKind = "export.create"
	ActionAuditDelete   ActionKind = "audit.delete"
	ActionConfigReload  ActionKind = "configuration.reload"
)

func (kind ActionKind) Valid() bool {
	switch kind {
	case ActionDockerStart, ActionDockerStop, ActionDockerRestart,
		ActionBackupCreate, ActionBackupRestore, ActionExportCreate,
		ActionAuditDelete, ActionConfigReload:
		return true
	default:
		return false
	}
}

type ActionStatus string

const (
	ActionPending   ActionStatus = "pending"
	ActionSucceeded ActionStatus = "succeeded"
	ActionFailed    ActionStatus = "failed"
	ActionRejected  ActionStatus = "rejected"
)

func (status ActionStatus) Valid() bool {
	switch status {
	case ActionPending, ActionSucceeded, ActionFailed, ActionRejected:
		return true
	default:
		return false
	}
}

type ErrorCode string

const (
	ErrorValidationFailed    ErrorCode = "validation_failed"
	ErrorNotFound            ErrorCode = "not_found"
	ErrorUnavailable         ErrorCode = "unavailable"
	ErrorConflict            ErrorCode = "conflict"
	ErrorConfirmationNeeded  ErrorCode = "confirmation_required"
	ErrorConfirmationExpired ErrorCode = "confirmation_expired"
	ErrorRateLimited         ErrorCode = "rate_limited"
	ErrorInternal            ErrorCode = "internal_error"
	ErrorDockerUnavailable   ErrorCode = "docker_unavailable"
	ErrorDockerAction        ErrorCode = "docker_action_failed"
	ErrorHistoryUnavailable  ErrorCode = "history_unavailable"
	ErrorAuditUnavailable    ErrorCode = "audit_unavailable"
	ErrorSettingsInvalid     ErrorCode = "settings_invalid"
	ErrorStorageLimit        ErrorCode = "storage_limit_reached"
	ErrorBackupFailed        ErrorCode = "backup_failed"
	ErrorRestoreFailed       ErrorCode = "restore_failed"
	ErrorExportFailed        ErrorCode = "export_failed"
)

func (code ErrorCode) Valid() bool {
	switch code {
	case ErrorValidationFailed, ErrorNotFound, ErrorUnavailable, ErrorConflict,
		ErrorConfirmationNeeded, ErrorConfirmationExpired, ErrorRateLimited,
		ErrorInternal, ErrorDockerUnavailable, ErrorDockerAction,
		ErrorHistoryUnavailable, ErrorAuditUnavailable, ErrorSettingsInvalid,
		ErrorStorageLimit, ErrorBackupFailed, ErrorRestoreFailed, ErrorExportFailed:
		return true
	default:
		return false
	}
}
