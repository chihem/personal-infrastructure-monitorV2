package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/collector"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/collector/docker"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/collector/host"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

const CollectionInterval = time.Minute

const (
	errorCollectorTimeout = "collector_timeout"
	errorRunCancelled     = "collection_cancelled"
	errorInvalidResult    = "invalid_collector_result"
)

var (
	ErrRunInProgress    = errors.New("collection run already in progress")
	ErrSchedulerActive  = errors.New("scheduler is already running")
	errCollectorTimeout = errors.New("collector deadline exceeded")
)

type Options struct {
	Host             host.Provider
	Docker           docker.Provider
	CollectorTimeout time.Duration
	Recorder         Recorder

	clock clock
}

type CompletedRun struct {
	Record         domain.CollectionRun
	HostSnapshot   any
	DockerSnapshot any
}

type Recorder interface {
	RecordCollection(context.Context, CompletedRun) error
}

type Service struct {
	host             host.Provider
	docker           docker.Provider
	collectorTimeout time.Duration
	recorder         Recorder
	clock            clock

	mu          sync.RWMutex
	runActive   bool
	loopActive  bool
	lastRun     *CompletedRun
	lastSuccess *CompletedRun
}

func New(options Options) (*Service, error) {
	if options.Host == nil {
		return nil, fmt.Errorf("host provider is required")
	}
	if options.Docker == nil {
		return nil, fmt.Errorf("docker provider is required")
	}
	if options.CollectorTimeout <= 0 || options.CollectorTimeout >= CollectionInterval {
		return nil, fmt.Errorf("collector timeout must be positive and shorter than %s", CollectionInterval)
	}
	if options.clock == nil {
		options.clock = realClock{}
	}

	return &Service{
		host:             options.Host,
		docker:           options.Docker,
		collectorTimeout: options.CollectorTimeout,
		recorder:         options.Recorder,
		clock:            options.clock,
	}, nil
}

// Run executes collection at UTC minute boundaries until ctx is cancelled.
// A period that passes while another run is active is skipped, not replayed.
func (service *Service) Run(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !service.beginLoop() {
		return ErrSchedulerActive
	}
	defer service.endLoop()

	for {
		now := service.clock.Now()
		wait := nextMinuteBoundary(now).Sub(now)
		timer := service.clock.NewTimer(wait)

		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C():
		}

		_, err := service.collect(ctx, domain.CollectionTriggerScheduled)
		if err == nil || errors.Is(err, ErrRunInProgress) {
			continue
		}
		if ctx.Err() != nil {
			return nil
		}
		return err
	}
}

// Trigger starts one manual collection unless another collection is active.
func (service *Service) Trigger(ctx context.Context) (CompletedRun, error) {
	return service.collect(ctx, domain.CollectionTriggerManual)
}

func (service *Service) LastRun() (CompletedRun, bool) {
	service.mu.RLock()
	defer service.mu.RUnlock()
	if service.lastRun == nil {
		return CompletedRun{}, false
	}
	return *service.lastRun, true
}

func (service *Service) LastSuccessfulRun() (CompletedRun, bool) {
	service.mu.RLock()
	defer service.mu.RUnlock()
	if service.lastSuccess == nil {
		return CompletedRun{}, false
	}
	return *service.lastSuccess, true
}

func (service *Service) CollectionInProgress() bool {
	service.mu.RLock()
	defer service.mu.RUnlock()
	return service.runActive
}

func (service *Service) collect(ctx context.Context, trigger domain.CollectionTrigger) (CompletedRun, error) {
	if err := ctx.Err(); err != nil {
		return CompletedRun{}, err
	}
	if !trigger.Valid() {
		return CompletedRun{}, fmt.Errorf("invalid collection trigger %q", trigger)
	}
	if !service.beginRun() {
		return CompletedRun{}, ErrRunInProgress
	}
	defer service.endRun()

	startedAt := service.clock.Now().UTC()
	hostResult := make(chan providerExecution, 1)
	dockerResult := make(chan providerExecution, 1)

	go func() {
		hostResult <- service.executeProvider(ctx, domain.CollectionSubsystemHost, service.host)
	}()
	go func() {
		dockerResult <- service.executeProvider(ctx, domain.CollectionSubsystemDocker, service.docker)
	}()

	hostExecution := <-hostResult
	dockerExecution := <-dockerResult
	finishedAt := service.clock.Now().UTC()
	if finishedAt.Before(startedAt) {
		finishedAt = startedAt
	}

	record := domain.CollectionRun{
		StartedAt:    startedAt,
		FinishedAt:   finishedAt,
		Trigger:      trigger,
		Status:       domain.AggregateCollectionStatus(hostExecution.outcome.Status, dockerExecution.outcome.Status),
		HostResult:   hostExecution.outcome,
		DockerResult: dockerExecution.outcome,
	}
	if err := record.Validate(); err != nil {
		return CompletedRun{}, fmt.Errorf("build collection run: %w", err)
	}

	completed := CompletedRun{
		Record:         record,
		HostSnapshot:   hostExecution.snapshot,
		DockerSnapshot: dockerExecution.snapshot,
	}
	service.storeLastRun(completed)
	if service.recorder != nil {
		if err := service.recorder.RecordCollection(ctx, completed); err != nil {
			return completed, fmt.Errorf("record completed collection: %w", err)
		}
	}
	return completed, nil
}

type collectingProvider interface {
	Collect(context.Context) collector.Result
}

type providerExecution struct {
	outcome  domain.CollectionOutcome
	snapshot any
}

func (service *Service) executeProvider(
	ctx context.Context,
	subsystem domain.CollectionSubsystem,
	provider collectingProvider,
) providerExecution {
	startedAt := service.clock.Now().UTC()
	collectorContext, cancel := context.WithCancelCause(ctx)
	timeout := service.clock.NewTimer(service.collectorTimeout)
	monitorDone := make(chan struct{})

	go func() {
		select {
		case <-timeout.C():
			cancel(errCollectorTimeout)
		case <-collectorContext.Done():
		case <-monitorDone:
		}
	}()

	result := provider.Collect(collectorContext)
	cause := context.Cause(collectorContext)
	close(monitorDone)
	timeout.Stop()
	cancel(context.Canceled)

	finishedAt := service.clock.Now().UTC()
	if finishedAt.Before(startedAt) {
		finishedAt = startedAt
	}

	switch {
	case errors.Is(cause, errCollectorTimeout):
		return failedExecution(subsystem, startedAt, finishedAt, errorCollectorTimeout)
	case cause != nil:
		return failedExecution(subsystem, startedAt, finishedAt, errorRunCancelled)
	case result.Validate() != nil:
		return failedExecution(subsystem, startedAt, finishedAt, errorInvalidResult)
	default:
		return providerExecution{
			outcome: domain.CollectionOutcome{
				Subsystem:  subsystem,
				Status:     result.Status,
				StartedAt:  startedAt,
				FinishedAt: finishedAt,
				ErrorCode:  result.ErrorCode,
			},
			snapshot: result.Snapshot,
		}
	}
}

func failedExecution(
	subsystem domain.CollectionSubsystem,
	startedAt time.Time,
	finishedAt time.Time,
	errorCode string,
) providerExecution {
	return providerExecution{outcome: domain.CollectionOutcome{
		Subsystem:  subsystem,
		Status:     domain.CollectionFailed,
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		ErrorCode:  errorCode,
	}}
}

func nextMinuteBoundary(now time.Time) time.Time {
	return now.UTC().Truncate(CollectionInterval).Add(CollectionInterval)
}

func (service *Service) beginRun() bool {
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.runActive {
		return false
	}
	service.runActive = true
	return true
}

func (service *Service) endRun() {
	service.mu.Lock()
	service.runActive = false
	service.mu.Unlock()
}

func (service *Service) beginLoop() bool {
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.loopActive {
		return false
	}
	service.loopActive = true
	return true
}

func (service *Service) endLoop() {
	service.mu.Lock()
	service.loopActive = false
	service.mu.Unlock()
}

func (service *Service) storeLastRun(run CompletedRun) {
	service.mu.Lock()
	service.lastRun = &run
	if run.Record.Status == domain.CollectionSucceeded {
		service.lastSuccess = &run
	}
	service.mu.Unlock()
}
