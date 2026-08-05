package collector

import (
	"fmt"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

// Result is the scheduler-facing envelope returned by a collector. Snapshot is
// intentionally opaque: host and Docker feature tasks will define their own
// validated snapshot types without coupling the scheduler to those details.
type Result struct {
	Status    domain.CollectionStatus
	Snapshot  any
	ErrorCode string
}

func Success(snapshot any) Result {
	return Result{Status: domain.CollectionSucceeded, Snapshot: snapshot}
}

func Partial(snapshot any, errorCode string) Result {
	return Result{Status: domain.CollectionPartial, Snapshot: snapshot, ErrorCode: errorCode}
}

func Failure(errorCode string) Result {
	return Result{Status: domain.CollectionFailed, ErrorCode: errorCode}
}

func (result Result) Validate() error {
	if !result.Status.Valid() || result.Status == domain.CollectionNotAttempted {
		return fmt.Errorf("collector returned invalid status %q", result.Status)
	}
	if err := domain.ValidateCollectionErrorCode(result.ErrorCode); err != nil {
		return err
	}

	switch result.Status {
	case domain.CollectionSucceeded:
		if result.ErrorCode != "" {
			return fmt.Errorf("successful collector result cannot contain an error code")
		}
	case domain.CollectionFailed:
		if result.ErrorCode == "" {
			return fmt.Errorf("failed collector result requires an error code")
		}
	}
	return nil
}
