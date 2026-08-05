package host

import (
	"context"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/collector"
)

// Provider collects one opaque host snapshot. Implementations must honor
// context cancellation and must not execute commands supplied by users or
// configuration.
type Provider interface {
	Collect(context.Context) collector.Result
}
