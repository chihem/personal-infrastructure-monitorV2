package contracts

import (
	"fmt"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

func (request ExportRequest) Validate() error {
	if request.Format != ExportCSV && request.Format != ExportJSON {
		return fmt.Errorf("invalid export format %q", request.Format)
	}
	if len(request.Datasets) == 0 {
		return fmt.Errorf("at least one export dataset is required")
	}
	seen := make(map[ExportDataset]struct{}, len(request.Datasets))
	for _, dataset := range request.Datasets {
		if !dataset.Valid() {
			return fmt.Errorf("invalid export dataset %q", dataset)
		}
		if _, duplicate := seen[dataset]; duplicate {
			return fmt.Errorf("duplicate export dataset %q", dataset)
		}
		seen[dataset] = struct{}{}
	}
	if err := request.Range.Validate(); err != nil {
		return err
	}
	switch request.Range.Preset {
	case domain.RangeLastHour, domain.RangeLast24Hours, domain.RangeLast7Days,
		domain.RangeLast14Days, domain.RangeCustom:
		return nil
	default:
		return fmt.Errorf("range preset %q is not available for exports", request.Range.Preset)
	}
}

func (dataset ExportDataset) Valid() bool {
	switch dataset {
	case ExportCPU, ExportMemory, ExportFilesystems, ExportContainerUsage,
		ExportContainerEvents, ExportIncidents, ExportAudit:
		return true
	default:
		return false
	}
}

func (request AuditDeleteRequest) Validate() error {
	switch request.Scope {
	case AuditDeleteSelected:
		if len(request.IDs) == 0 || request.Range != nil {
			return fmt.Errorf("selected deletion requires ids and no range")
		}
		for _, id := range request.IDs {
			if err := domain.ValidateOpaqueID(id); err != nil {
				return fmt.Errorf("audit id: %w", err)
			}
		}
	case AuditDeleteRange:
		if len(request.IDs) != 0 || request.Range == nil || request.Range.Preset != domain.RangeCustom {
			return fmt.Errorf("range deletion requires one custom range and no ids")
		}
		if err := request.Range.Validate(); err != nil {
			return fmt.Errorf("range: %w", err)
		}
	case AuditDeleteAll:
		if len(request.IDs) != 0 || request.Range != nil {
			return fmt.Errorf("all deletion cannot contain ids or range")
		}
	default:
		return fmt.Errorf("invalid audit deletion scope %q", request.Scope)
	}
	return nil
}

func (request ConfirmationRequest) Validate() error {
	if !request.Action.Valid() {
		return fmt.Errorf("invalid action %q", request.Action)
	}
	if err := request.Target.Validate(); err != nil {
		return fmt.Errorf("target: %w", err)
	}
	return validateActionTarget(request.Action, request.Target.Kind)
}

func (intent ConfirmationIntent) Validate() error {
	if err := domain.ValidateOpaqueID(intent.ID); err != nil {
		return fmt.Errorf("id: %w", err)
	}
	if err := (ConfirmationRequest{Action: intent.Action, Target: intent.Target}).Validate(); err != nil {
		return err
	}
	if err := domain.ValidateUTC(intent.ExpiresAt); err != nil {
		return fmt.Errorf("expiresAt: %w", err)
	}
	return nil
}

func (request ExecuteActionRequest) Validate() error {
	if err := domain.ValidateOpaqueID(request.ConfirmationID); err != nil {
		return fmt.Errorf("confirmationId: %w", err)
	}
	return (ConfirmationRequest{Action: request.Action, Target: request.Target}).Validate()
}

func (event DockerLogEvent) Validate() error {
	switch event.Type {
	case DockerLogLineEvent:
		if event.Line == nil || event.Error != nil {
			return fmt.Errorf("line event requires line and no error")
		}
		if event.Line.Stream != LogStdout && event.Line.Stream != LogStderr {
			return fmt.Errorf("invalid log stream %q", event.Line.Stream)
		}
		if err := domain.ValidateOptionalUTC(event.Line.Timestamp); err != nil {
			return fmt.Errorf("line timestamp: %w", err)
		}
	case DockerLogErrorEvent:
		if event.Line != nil || event.Error == nil {
			return fmt.Errorf("error event requires error and no line")
		}
		if err := event.Error.Validate(); err != nil {
			return fmt.Errorf("error: %w", err)
		}
	case DockerLogEndEvent:
		if event.Line != nil || event.Error != nil {
			return fmt.Errorf("end event cannot contain line or error")
		}
	default:
		return fmt.Errorf("invalid Docker log event type %q", event.Type)
	}
	return nil
}

func validateActionTarget(action domain.ActionKind, kind domain.ResourceKind) error {
	var expected domain.ResourceKind
	switch action {
	case domain.ActionDockerStart, domain.ActionDockerStop, domain.ActionDockerRestart:
		expected = domain.ResourceContainer
	case domain.ActionBackupCreate, domain.ActionBackupRestore:
		expected = domain.ResourceBackup
	case domain.ActionAuditDelete:
		expected = domain.ResourceAuditDB
	default:
		return fmt.Errorf("invalid action %q", action)
	}
	if kind != expected {
		return fmt.Errorf("action %q requires target kind %q", action, expected)
	}
	return nil
}
