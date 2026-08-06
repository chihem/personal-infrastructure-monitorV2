package docker

import (
	"context"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/collector"
)

const ErrorNotImplemented = "docker_not_implemented"

// UnavailableProvider is the explicit pre-CORE-08 boundary. It performs no
// Docker operation and prevents the scheduler from presenting Docker as
// healthy before the real read-only provider exists.
type UnavailableProvider struct{}

func (UnavailableProvider) Collect(ctx context.Context) collector.Result {
	if ctx.Err() != nil {
		return collector.Failure("docker_collection_cancelled")
	}
	return collector.Failure(ErrorNotImplemented)
}

var _ Provider = UnavailableProvider{}
