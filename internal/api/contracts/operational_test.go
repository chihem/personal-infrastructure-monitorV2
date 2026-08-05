package contracts

import (
	"testing"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

func TestReadinessStatusMatchesRequiredDependencies(t *testing.T) {
	tests := []struct {
		name   string
		status ReadinessStatus
		valid  bool
	}{
		{name: "ready", status: ReadinessStatus{Ready: true, ConfigurationState: ConfigurationValid, HistoryDatabaseAvailable: true}, valid: true},
		{name: "previous config remains ready", status: ReadinessStatus{Ready: true, ConfigurationState: ConfigurationUsingPrevious, HistoryDatabaseAvailable: true}, valid: true},
		{name: "maintenance", status: ReadinessStatus{Maintenance: true, ConfigurationState: ConfigurationValid, HistoryDatabaseAvailable: true}, valid: true},
		{name: "unavailable config", status: ReadinessStatus{ConfigurationState: ConfigurationUnavailable, HistoryDatabaseAvailable: true}, valid: true},
		{name: "contradiction", status: ReadinessStatus{Ready: true, Maintenance: true, ConfigurationState: ConfigurationValid, HistoryDatabaseAvailable: true}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.status.Validate()
			if (err == nil) != test.valid {
				t.Fatalf("Validate() error = %v, valid = %v", err, test.valid)
			}
		})
	}
}

func TestOperationalStatusAllowsHonestFoundationPlaceholders(t *testing.T) {
	size := int64(4096)
	status := OperationalStatus{
		State:              OperationalDegraded,
		UptimeSeconds:      10,
		ConfigurationState: ConfigurationValid,
		HistoryDatabase:    DatabaseOperationalStatus{State: DependencyAvailable, SizeBytes: &size},
		AuditDatabase:      DatabaseOperationalStatus{State: DependencyUnavailable},
		Collection:         CollectionOperationalStatus{State: DependencyNotStarted},
		BackupState:        DependencyNotImplemented,
		DockerConnectivity: DependencyNotChecked,
	}
	if err := status.Validate(); err != nil {
		t.Fatalf("valid foundation status rejected: %v", err)
	}
}

func TestOperationalStatusRejectsOverallStateThatContradictsDependencies(t *testing.T) {
	status := OperationalStatus{
		State:              OperationalOK,
		ConfigurationState: ConfigurationValid,
		HistoryDatabase:    DatabaseOperationalStatus{State: DependencyAvailable},
		AuditDatabase:      DatabaseOperationalStatus{State: DependencyAvailable},
		Collection:         CollectionOperationalStatus{State: DependencyNotStarted},
		BackupState:        DependencyNotImplemented,
		DockerConnectivity: DependencyNotChecked,
	}
	if err := status.Validate(); err == nil {
		t.Fatal("operational status accepted ok with unfinished dependencies")
	}
}

func TestCollectionOperationalDurationMustMatchTimestamps(t *testing.T) {
	start := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	status := CollectionRunOperationalStatus{
		StartedAt: start, FinishedAt: start.Add(time.Second), DurationMS: 999,
		Status: domain.CollectionSucceeded,
	}
	if err := status.Validate(); err == nil {
		t.Fatal("collection status accepted a fabricated duration")
	}
}
