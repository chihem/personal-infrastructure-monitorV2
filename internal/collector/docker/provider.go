package docker

import (
	"context"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/collector"
)

// Provider collects one opaque Docker snapshot. Implementations must honor
// context cancellation and use only the approved read operations.
type Provider interface {
	Collect(context.Context) collector.Result
}
