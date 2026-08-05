package domain

import (
	"testing"
	"time"
)

func TestAggregateCollectionStatus(t *testing.T) {
	tests := []struct {
		name     string
		statuses []CollectionStatus
		want     CollectionStatus
	}{
		{name: "all succeeded", statuses: []CollectionStatus{CollectionSucceeded, CollectionSucceeded}, want: CollectionSucceeded},
		{name: "successful and failed", statuses: []CollectionStatus{CollectionSucceeded, CollectionFailed}, want: CollectionPartial},
		{name: "partial and failed", statuses: []CollectionStatus{CollectionPartial, CollectionFailed}, want: CollectionPartial},
		{name: "all failed", statuses: []CollectionStatus{CollectionFailed, CollectionFailed}, want: CollectionFailed},
		{name: "nothing attempted", statuses: []CollectionStatus{CollectionNotAttempted, CollectionNotAttempted}, want: CollectionFailed},
		{name: "empty", want: CollectionFailed},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := AggregateCollectionStatus(test.statuses...); got != test.want {
				t.Fatalf("AggregateCollectionStatus() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestCollectionRunValidation(t *testing.T) {
	start := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	finish := start.Add(time.Second)
	run := CollectionRun{
		StartedAt:  start,
		FinishedAt: finish,
		Trigger:    CollectionTriggerScheduled,
		Status:     CollectionPartial,
		HostResult: CollectionOutcome{
			Subsystem: CollectionSubsystemHost,
			Status:    CollectionSucceeded, StartedAt: start, FinishedAt: finish,
		},
		DockerResult: CollectionOutcome{
			Subsystem: CollectionSubsystemDocker,
			Status:    CollectionFailed, StartedAt: start, FinishedAt: finish,
			ErrorCode: "docker_unavailable",
		},
	}

	if err := run.Validate(); err != nil {
		t.Fatalf("valid run rejected: %v", err)
	}

	invalidStatus := run
	invalidStatus.Status = CollectionSucceeded
	if err := invalidStatus.Validate(); err == nil {
		t.Fatal("run accepted an aggregate status that contradicts its subsystem results")
	}

	invalidSubsystem := run
	invalidSubsystem.HostResult.Subsystem = CollectionSubsystemDocker
	if err := invalidSubsystem.Validate(); err == nil {
		t.Fatal("run accepted a Docker result in the host field")
	}

	invalidCode := run
	invalidCode.DockerResult.ErrorCode = "raw error: connection refused"
	if err := invalidCode.Validate(); err == nil {
		t.Fatal("run accepted an unbounded human-readable error as an error code")
	}
}
