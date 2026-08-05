package domain

import (
	"fmt"
	"strings"
	"time"
)

type CollectionTrigger string

const (
	CollectionTriggerScheduled CollectionTrigger = "scheduled"
	CollectionTriggerManual    CollectionTrigger = "manual"
)

func (trigger CollectionTrigger) Valid() bool {
	return trigger == CollectionTriggerScheduled || trigger == CollectionTriggerManual
}

type CollectionStatus string

const (
	CollectionSucceeded    CollectionStatus = "succeeded"
	CollectionPartial      CollectionStatus = "partial"
	CollectionFailed       CollectionStatus = "failed"
	CollectionNotAttempted CollectionStatus = "not_attempted"
)

func (status CollectionStatus) Valid() bool {
	switch status {
	case CollectionSucceeded, CollectionPartial, CollectionFailed, CollectionNotAttempted:
		return true
	default:
		return false
	}
}

type CollectionSubsystem string

const (
	CollectionSubsystemHost   CollectionSubsystem = "host"
	CollectionSubsystemDocker CollectionSubsystem = "docker"
)

func (subsystem CollectionSubsystem) Valid() bool {
	return subsystem == CollectionSubsystemHost || subsystem == CollectionSubsystemDocker
}

type CollectionOutcome struct {
	Subsystem  CollectionSubsystem `json:"subsystem"`
	Status     CollectionStatus    `json:"status"`
	StartedAt  time.Time           `json:"startedAt"`
	FinishedAt time.Time           `json:"finishedAt"`
	ErrorCode  string              `json:"errorCode,omitempty"`
}

func (outcome CollectionOutcome) Validate() error {
	if !outcome.Subsystem.Valid() {
		return fmt.Errorf("invalid collection subsystem %q", outcome.Subsystem)
	}
	if !outcome.Status.Valid() {
		return fmt.Errorf("invalid collection status %q", outcome.Status)
	}
	if err := ValidateUTC(outcome.StartedAt); err != nil {
		return fmt.Errorf("startedAt: %w", err)
	}
	if err := ValidateUTC(outcome.FinishedAt); err != nil {
		return fmt.Errorf("finishedAt: %w", err)
	}
	if outcome.FinishedAt.Before(outcome.StartedAt) {
		return fmt.Errorf("finishedAt must not be before startedAt")
	}
	if err := ValidateCollectionErrorCode(outcome.ErrorCode); err != nil {
		return err
	}

	switch outcome.Status {
	case CollectionSucceeded:
		if outcome.ErrorCode != "" {
			return fmt.Errorf("successful collection cannot contain an error code")
		}
	case CollectionFailed, CollectionNotAttempted:
		if outcome.ErrorCode == "" {
			return fmt.Errorf("%s collection requires an error code", outcome.Status)
		}
	}

	return nil
}

type CollectionRun struct {
	StartedAt    time.Time         `json:"startedAt"`
	FinishedAt   time.Time         `json:"finishedAt"`
	Trigger      CollectionTrigger `json:"trigger"`
	Status       CollectionStatus  `json:"status"`
	HostResult   CollectionOutcome `json:"hostResult"`
	DockerResult CollectionOutcome `json:"dockerResult"`
}

func (run CollectionRun) Validate() error {
	if err := ValidateUTC(run.StartedAt); err != nil {
		return fmt.Errorf("startedAt: %w", err)
	}
	if err := ValidateUTC(run.FinishedAt); err != nil {
		return fmt.Errorf("finishedAt: %w", err)
	}
	if run.FinishedAt.Before(run.StartedAt) {
		return fmt.Errorf("finishedAt must not be before startedAt")
	}
	if !run.Trigger.Valid() {
		return fmt.Errorf("invalid collection trigger %q", run.Trigger)
	}
	if !run.Status.Valid() || run.Status == CollectionNotAttempted {
		return fmt.Errorf("invalid collection run status %q", run.Status)
	}

	if err := run.HostResult.Validate(); err != nil {
		return fmt.Errorf("hostResult: %w", err)
	}
	if run.HostResult.Subsystem != CollectionSubsystemHost {
		return fmt.Errorf("hostResult must describe the host subsystem")
	}
	if err := run.DockerResult.Validate(); err != nil {
		return fmt.Errorf("dockerResult: %w", err)
	}
	if run.DockerResult.Subsystem != CollectionSubsystemDocker {
		return fmt.Errorf("dockerResult must describe the docker subsystem")
	}

	for name, outcome := range map[string]CollectionOutcome{
		"hostResult":   run.HostResult,
		"dockerResult": run.DockerResult,
	} {
		if outcome.StartedAt.Before(run.StartedAt) || outcome.FinishedAt.After(run.FinishedAt) {
			return fmt.Errorf("%s timestamps must be within the collection run", name)
		}
	}

	expected := AggregateCollectionStatus(run.HostResult.Status, run.DockerResult.Status)
	if run.Status != expected {
		return fmt.Errorf("collection run status %q does not match subsystem status %q", run.Status, expected)
	}
	return nil
}

func AggregateCollectionStatus(statuses ...CollectionStatus) CollectionStatus {
	if len(statuses) == 0 {
		return CollectionFailed
	}

	succeeded := 0
	partial := 0
	for _, status := range statuses {
		switch status {
		case CollectionSucceeded:
			succeeded++
		case CollectionPartial:
			partial++
		}
	}

	if succeeded == len(statuses) {
		return CollectionSucceeded
	}
	if succeeded > 0 || partial > 0 {
		return CollectionPartial
	}
	return CollectionFailed
}

func ValidateCollectionErrorCode(code string) error {
	if code == "" {
		return nil
	}
	if len(code) > 64 {
		return fmt.Errorf("collection error code must not exceed 64 characters")
	}
	if strings.TrimSpace(code) != code {
		return fmt.Errorf("collection error code must not contain surrounding whitespace")
	}
	for _, character := range code {
		valid := character >= 'a' && character <= 'z' ||
			character >= '0' && character <= '9' ||
			character == '_' || character == '-' || character == '.'
		if !valid {
			return fmt.Errorf("collection error code must use lowercase letters, digits, dots, hyphens, or underscores")
		}
	}
	return nil
}
