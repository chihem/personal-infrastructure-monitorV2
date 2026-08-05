package scheduler

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/collector"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

func TestScheduledRunStartsAtNextMinuteBoundary(t *testing.T) {
	start := time.Date(2026, 8, 5, 12, 0, 30, 0, time.UTC)
	clock := newFakeClock(start)
	provider := &providerStub{collect: func(context.Context) collector.Result {
		return collector.Success("snapshot")
	}}
	service := newTestService(t, clock, provider, provider, 10*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- service.Run(ctx) }()
	waitForCondition(t, func() bool { return clock.activeTimerCount() == 1 }, "initial minute timer")

	clock.Advance(29 * time.Second)
	if _, ok := service.LastRun(); ok {
		t.Fatal("collection ran before the minute boundary")
	}

	clock.Advance(time.Second)
	waitForCondition(t, func() bool {
		run, ok := service.LastRun()
		return ok && run.Record.StartedAt.Equal(start.Add(30*time.Second))
	}, "scheduled collection")

	run, _ := service.LastRun()
	if run.Record.Trigger != domain.CollectionTriggerScheduled {
		t.Fatalf("trigger = %q, want scheduled", run.Record.Trigger)
	}
	if provider.calls.Load() != 2 {
		t.Fatalf("provider calls = %d, want 2", provider.calls.Load())
	}

	cancel()
	if err := receive(t, result, "scheduler shutdown"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestMissedPeriodsAreNotBackfilled(t *testing.T) {
	start := time.Date(2026, 8, 5, 12, 0, 30, 0, time.UTC)
	clock := newFakeClock(start)
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	hostProvider := &providerStub{collect: func(context.Context) collector.Result {
		select {
		case started <- struct{}{}:
		default:
		}
		<-release
		return collector.Success("host")
	}}
	dockerProvider := &providerStub{collect: func(context.Context) collector.Result {
		return collector.Success("docker")
	}}
	service := newTestService(t, clock, hostProvider, dockerProvider, 30*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- service.Run(ctx) }()
	waitForCondition(t, func() bool { return clock.activeTimerCount() == 1 }, "initial minute timer")
	clock.Advance(30 * time.Second)
	receive(t, started, "first scheduled run")

	clock.Advance(3 * time.Minute)
	close(release)
	waitForCondition(t, func() bool {
		_, ok := service.LastRun()
		return ok && clock.activeTimerCount() == 1
	}, "next aligned timer")
	if hostProvider.calls.Load() != 1 {
		t.Fatalf("host calls after missed periods = %d, want 1", hostProvider.calls.Load())
	}

	clock.Advance(59 * time.Second)
	if hostProvider.calls.Load() != 1 {
		t.Fatalf("host calls before next boundary = %d, want 1", hostProvider.calls.Load())
	}
	clock.Advance(time.Second)
	waitForCondition(t, func() bool { return hostProvider.calls.Load() == 2 }, "next scheduled run")

	cancel()
	if err := receive(t, result, "scheduler shutdown"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestManualTriggerRejectsOverlap(t *testing.T) {
	clock := newFakeClock(time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC))
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	hostProvider := &providerStub{collect: func(context.Context) collector.Result {
		started <- struct{}{}
		<-release
		return collector.Success("host")
	}}
	dockerProvider := &providerStub{collect: func(context.Context) collector.Result {
		return collector.Success("docker")
	}}
	service := newTestService(t, clock, hostProvider, dockerProvider, 10*time.Second)

	first := make(chan triggerResult, 1)
	go func() {
		run, err := service.Trigger(context.Background())
		first <- triggerResult{run: run, err: err}
	}()
	receive(t, started, "manual collection start")

	if _, err := service.Trigger(context.Background()); !errors.Is(err, ErrRunInProgress) {
		t.Fatalf("overlapping Trigger() error = %v, want ErrRunInProgress", err)
	}
	close(release)
	completed := receive(t, first, "manual collection result")
	if completed.err != nil {
		t.Fatalf("first Trigger() error = %v", completed.err)
	}
	if completed.run.Record.Trigger != domain.CollectionTriggerManual {
		t.Fatalf("trigger = %q, want manual", completed.run.Record.Trigger)
	}
}

func TestCollectorTimeoutKeepsOtherSubsystemResult(t *testing.T) {
	clock := newFakeClock(time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC))
	hostStarted := make(chan struct{}, 1)
	hostProvider := &providerStub{collect: func(ctx context.Context) collector.Result {
		hostStarted <- struct{}{}
		<-ctx.Done()
		return collector.Success("late host result")
	}}
	dockerProvider := &providerStub{collect: func(context.Context) collector.Result {
		return collector.Success("docker snapshot")
	}}
	service := newTestService(t, clock, hostProvider, dockerProvider, 10*time.Second)

	result := make(chan triggerResult, 1)
	go func() {
		run, err := service.Trigger(context.Background())
		result <- triggerResult{run: run, err: err}
	}()
	receive(t, hostStarted, "host collector start")
	waitForCondition(t, func() bool { return clock.activeTimerCount() >= 1 }, "collector deadline timer")
	clock.Advance(10 * time.Second)

	completed := receive(t, result, "timed collection")
	if completed.err != nil {
		t.Fatalf("Trigger() error = %v", completed.err)
	}
	if completed.run.Record.Status != domain.CollectionPartial {
		t.Fatalf("run status = %q, want partial", completed.run.Record.Status)
	}
	if completed.run.Record.HostResult.Status != domain.CollectionFailed ||
		completed.run.Record.HostResult.ErrorCode != errorCollectorTimeout {
		t.Fatalf("host result = %+v, want timeout failure", completed.run.Record.HostResult)
	}
	if completed.run.Record.DockerResult.Status != domain.CollectionSucceeded ||
		completed.run.DockerSnapshot != "docker snapshot" {
		t.Fatalf("Docker result was not retained: %+v", completed.run)
	}
	if completed.run.HostSnapshot != nil {
		t.Fatalf("timed-out host snapshot = %#v, want nil", completed.run.HostSnapshot)
	}
}

func TestPartialProviderResultIsPreserved(t *testing.T) {
	clock := newFakeClock(time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC))
	hostProvider := &providerStub{collect: func(context.Context) collector.Result {
		return collector.Partial("partial host snapshot", "memory_pressure_unavailable")
	}}
	dockerProvider := &providerStub{collect: func(context.Context) collector.Result {
		return collector.Success("docker snapshot")
	}}
	service := newTestService(t, clock, hostProvider, dockerProvider, 10*time.Second)

	run, err := service.Trigger(context.Background())
	if err != nil {
		t.Fatalf("Trigger() error = %v", err)
	}
	if run.Record.Status != domain.CollectionPartial || run.HostSnapshot != "partial host snapshot" {
		t.Fatalf("partial result was not preserved: %+v", run)
	}
}

func TestPartialRunDoesNotReplaceLastSuccessfulRun(t *testing.T) {
	clock := newFakeClock(time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC))
	var partial atomic.Bool
	hostProvider := &providerStub{collect: func(context.Context) collector.Result {
		if partial.Load() {
			return collector.Partial("partial host", "field_unavailable")
		}
		return collector.Success("complete host")
	}}
	dockerProvider := &providerStub{collect: func(context.Context) collector.Result {
		return collector.Success("docker")
	}}
	service := newTestService(t, clock, hostProvider, dockerProvider, 10*time.Second)

	first, err := service.Trigger(context.Background())
	if err != nil {
		t.Fatalf("first Trigger() error = %v", err)
	}
	clock.Advance(time.Second)
	partial.Store(true)
	if _, err := service.Trigger(context.Background()); err != nil {
		t.Fatalf("partial Trigger() error = %v", err)
	}

	lastSuccess, ok := service.LastSuccessfulRun()
	if !ok || !lastSuccess.Record.FinishedAt.Equal(first.Record.FinishedAt) {
		t.Fatalf("LastSuccessfulRun() = %+v, available = %v", lastSuccess, ok)
	}
}

func TestInvalidProviderResultBecomesBoundedFailure(t *testing.T) {
	clock := newFakeClock(time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC))
	invalidProvider := &providerStub{collect: func(context.Context) collector.Result {
		return collector.Result{Status: domain.CollectionSucceeded, ErrorCode: "raw failure"}
	}}
	validProvider := &providerStub{collect: func(context.Context) collector.Result {
		return collector.Success("docker snapshot")
	}}
	service := newTestService(t, clock, invalidProvider, validProvider, 10*time.Second)

	run, err := service.Trigger(context.Background())
	if err != nil {
		t.Fatalf("Trigger() error = %v", err)
	}
	if run.Record.Status != domain.CollectionPartial ||
		run.Record.HostResult.ErrorCode != errorInvalidResult ||
		run.HostSnapshot != nil {
		t.Fatalf("invalid provider result was not contained: %+v", run)
	}
}

func TestCancellationStopsCollectorsAndRecordsFailure(t *testing.T) {
	clock := newFakeClock(time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC))
	started := make(chan struct{}, 2)
	provider := &providerStub{collect: func(ctx context.Context) collector.Result {
		started <- struct{}{}
		<-ctx.Done()
		return collector.Failure("provider_cancelled")
	}}
	service := newTestService(t, clock, provider, provider, 10*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan triggerResult, 1)
	go func() {
		run, err := service.Trigger(ctx)
		result <- triggerResult{run: run, err: err}
	}()
	receive(t, started, "first collector start")
	receive(t, started, "second collector start")
	cancel()

	completed := receive(t, result, "cancelled collection")
	if completed.err != nil {
		t.Fatalf("Trigger() error = %v", completed.err)
	}
	if completed.run.Record.Status != domain.CollectionFailed ||
		completed.run.Record.HostResult.ErrorCode != errorRunCancelled ||
		completed.run.Record.DockerResult.ErrorCode != errorRunCancelled {
		t.Fatalf("cancelled run = %+v", completed.run.Record)
	}
}

func TestRunStopsBeforeFirstCollection(t *testing.T) {
	clock := newFakeClock(time.Date(2026, 8, 5, 12, 0, 30, 0, time.UTC))
	provider := &providerStub{collect: func(context.Context) collector.Result {
		return collector.Success(nil)
	}}
	service := newTestService(t, clock, provider, provider, 10*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- service.Run(ctx) }()
	waitForCondition(t, func() bool { return clock.activeTimerCount() == 1 }, "initial minute timer")
	cancel()
	if err := receive(t, result, "scheduler shutdown"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if provider.calls.Load() != 0 {
		t.Fatalf("provider calls = %d, want 0", provider.calls.Load())
	}
}

func TestSecondSchedulerLoopIsRejected(t *testing.T) {
	clock := newFakeClock(time.Date(2026, 8, 5, 12, 0, 30, 0, time.UTC))
	provider := &providerStub{collect: func(context.Context) collector.Result {
		return collector.Success(nil)
	}}
	service := newTestService(t, clock, provider, provider, 10*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	first := make(chan error, 1)
	go func() { first <- service.Run(ctx) }()
	waitForCondition(t, func() bool { return clock.activeTimerCount() == 1 }, "first scheduler loop")
	if err := service.Run(context.Background()); !errors.Is(err, ErrSchedulerActive) {
		t.Fatalf("second Run() error = %v, want ErrSchedulerActive", err)
	}
	cancel()
	if err := receive(t, first, "first scheduler shutdown"); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
}

func TestNewRejectsUnsafeOptions(t *testing.T) {
	provider := &providerStub{collect: func(context.Context) collector.Result {
		return collector.Success(nil)
	}}

	tests := []struct {
		name    string
		options Options
	}{
		{name: "missing host", options: Options{Docker: provider, CollectorTimeout: time.Second}},
		{name: "missing Docker", options: Options{Host: provider, CollectorTimeout: time.Second}},
		{name: "zero timeout", options: Options{Host: provider, Docker: provider}},
		{name: "timeout reaches interval", options: Options{Host: provider, Docker: provider, CollectorTimeout: CollectionInterval}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := New(test.options); err == nil {
				t.Fatal("New() accepted unsafe options")
			}
		})
	}
}

type triggerResult struct {
	run CompletedRun
	err error
}

type providerStub struct {
	collect func(context.Context) collector.Result
	calls   atomic.Int32
}

func (provider *providerStub) Collect(ctx context.Context) collector.Result {
	provider.calls.Add(1)
	return provider.collect(ctx)
}

func newTestService(
	t *testing.T,
	clock *fakeClock,
	hostProvider *providerStub,
	dockerProvider *providerStub,
	timeout time.Duration,
) *Service {
	t.Helper()
	service, err := New(Options{
		Host: hostProvider, Docker: dockerProvider, CollectorTimeout: timeout, clock: clock,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return service
}

func receive[T any](t *testing.T, values <-chan T, label string) T {
	t.Helper()
	select {
	case value := <-values:
		return value
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", label)
		var zero T
		return zero
	}
}

func waitForCondition(t *testing.T, condition func() bool, label string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", label)
		}
		time.Sleep(time.Millisecond)
	}
}

type fakeClock struct {
	mu     sync.Mutex
	now    time.Time
	timers []*fakeTimer
}

func newFakeClock(now time.Time) *fakeClock {
	return &fakeClock{now: now}
}

func (clock *fakeClock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	return clock.now
}

func (clock *fakeClock) NewTimer(duration time.Duration) timer {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	timer := &fakeTimer{
		clock: clock, deadline: clock.now.Add(duration), channel: make(chan time.Time, 1),
	}
	clock.timers = append(clock.timers, timer)
	if duration <= 0 {
		timer.fired = true
		timer.channel <- clock.now
	}
	return timer
}

func (clock *fakeClock) Advance(duration time.Duration) {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	clock.now = clock.now.Add(duration)
	for _, timer := range clock.timers {
		if timer.stopped || timer.fired || timer.deadline.After(clock.now) {
			continue
		}
		timer.fired = true
		timer.channel <- clock.now
	}
}

func (clock *fakeClock) activeTimerCount() int {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	count := 0
	for _, timer := range clock.timers {
		if !timer.stopped && !timer.fired {
			count++
		}
	}
	return count
}

type fakeTimer struct {
	clock    *fakeClock
	deadline time.Time
	channel  chan time.Time
	stopped  bool
	fired    bool
}

func (timer *fakeTimer) C() <-chan time.Time {
	return timer.channel
}

func (timer *fakeTimer) Stop() bool {
	timer.clock.mu.Lock()
	defer timer.clock.mu.Unlock()
	if timer.stopped || timer.fired {
		return false
	}
	timer.stopped = true
	return true
}
