package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/api/contracts"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/config"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/scheduler"
)

func TestStatusSourceDistinguishesReadyDegradedAndMaintenance(t *testing.T) {
	start := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	now := start.Add(15 * time.Second)
	maintenance := &MaintenanceState{}
	configurationState := config.StateValid
	source := statusSource{
		startedAt:     start,
		now:           func() time.Time { return now },
		maintenance:   maintenance,
		configuration: func() config.Status { return config.Status{State: configurationState} },
		historyPath:   "history.db", auditPath: "audit.db",
		historyAvailable: true, auditAvailable: true,
		databaseSize: func(string) (int64, bool) { return 4096, true },
	}

	status := source.snapshot()
	if status.State != contracts.OperationalDegraded || status.UptimeSeconds != 15 {
		t.Fatalf("foundation status = %+v", status)
	}
	if readiness := source.readiness(); !readiness.Ready {
		t.Fatalf("foundation readiness = %+v, want ready", readiness)
	}

	configurationState = config.StateUsingPrevious
	if readiness := source.readiness(); !readiness.Ready {
		t.Fatalf("previous-valid configuration should remain ready: %+v", readiness)
	}

	maintenance.Enter()
	if status := source.snapshot(); status.State != contracts.OperationalMaintenance {
		t.Fatalf("maintenance status = %+v", status)
	}
	if readiness := source.readiness(); readiness.Ready {
		t.Fatalf("maintenance readiness = %+v, want not ready", readiness)
	}

	maintenance.Exit()
	configurationState = config.StateUnavailable
	if status := source.snapshot(); status.State != contracts.OperationalNotReady {
		t.Fatalf("unavailable configuration status = %+v", status)
	}
}

func TestStatusSourceReportsCollectionTimingAndDockerConnectivity(t *testing.T) {
	start := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	finished := start.Add(1250 * time.Millisecond)
	run := scheduler.CompletedRun{Record: domain.CollectionRun{
		StartedAt: start, FinishedAt: finished, Trigger: domain.CollectionTriggerScheduled,
		Status:       domain.CollectionPartial,
		HostResult:   domain.CollectionOutcome{Subsystem: domain.CollectionSubsystemHost, Status: domain.CollectionSucceeded, StartedAt: start, FinishedAt: finished},
		DockerResult: domain.CollectionOutcome{Subsystem: domain.CollectionSubsystemDocker, Status: domain.CollectionFailed, StartedAt: start, FinishedAt: finished, ErrorCode: "docker_unavailable"},
	}}
	provider := &collectionStatusStub{lastRun: run, hasLastRun: true}
	source := statusSource{
		startedAt: start, now: func() time.Time { return finished },
		maintenance: &MaintenanceState{}, configuration: func() config.Status { return config.Status{State: config.StateValid} },
		historyAvailable: true, auditAvailable: true, collection: provider,
		databaseSize: func(string) (int64, bool) { return 1, true },
	}

	status := source.snapshot()
	if status.Collection.LastRun == nil || status.Collection.LastRun.DurationMS != 1250 {
		t.Fatalf("collection timing = %+v", status.Collection)
	}
	if status.DockerConnectivity != contracts.DependencyUnavailable {
		t.Fatalf("Docker connectivity = %q, want unavailable", status.DockerConnectivity)
	}
}

func TestSizeDatabaseFilesIncludesWALAndSHMAndRejectsUnsafePaths(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "history.db")
	for name, size := range map[string]int{path: 10, path + "-wal": 20, path + "-shm": 30} {
		if err := os.WriteFile(name, make([]byte, size), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if size, ok := sizeDatabaseFiles(path); !ok || size != 60 {
		t.Fatalf("sizeDatabaseFiles() = %d, %v, want 60, true", size, ok)
	}
	if _, ok := sizeDatabaseFiles(filepath.Join(root, "missing.db")); ok {
		t.Fatal("missing database reported a size")
	}
	if _, ok := sizeDatabaseFiles(root); ok {
		t.Fatal("directory reported a database size")
	}
}

type collectionStatusStub struct {
	inProgress     bool
	lastRun        scheduler.CompletedRun
	hasLastRun     bool
	lastSuccess    scheduler.CompletedRun
	hasLastSuccess bool
}

func (stub *collectionStatusStub) CollectionInProgress() bool { return stub.inProgress }
func (stub *collectionStatusStub) LastRun() (scheduler.CompletedRun, bool) {
	return stub.lastRun, stub.hasLastRun
}
func (stub *collectionStatusStub) LastSuccessfulRun() (scheduler.CompletedRun, bool) {
	return stub.lastSuccess, stub.hasLastSuccess
}
