package docker

import (
	"context"
	"testing"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

func TestUnavailableProviderIsHonestAndReadOnly(t *testing.T) {
	t.Parallel()
	result := (UnavailableProvider{}).Collect(context.Background())
	if result.Status != domain.CollectionFailed || result.ErrorCode != ErrorNotImplemented || result.Snapshot != nil {
		t.Fatalf("result = %+v", result)
	}
	if err := result.Validate(); err != nil {
		t.Fatalf("result validation error = %v", err)
	}
}
