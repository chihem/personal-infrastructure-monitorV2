package history

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/storage"
)

func TestRecordCollectionRunPreservesSubsystemOutcomes(t *testing.T) {
	t.Parallel()
	repository, database := openTestRepository(t)
	defer database.Close()
	run := validCollectionRun(time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC))
	run.Status = domain.CollectionPartial
	run.DockerResult.Status = domain.CollectionFailed
	run.DockerResult.ErrorCode = "docker_unavailable"

	id, err := repository.RecordCollectionRun(context.Background(), run)
	if err != nil {
		t.Fatalf("RecordCollectionRun() error = %v", err)
	}
	var result, hostResult, dockerResult string
	var hostError, dockerError *string
	if err := database.SQL().QueryRow(`
		SELECT result, host_result, docker_result, host_error_code, docker_error_code
		FROM collection_runs WHERE id = ?
	`, id).Scan(&result, &hostResult, &dockerResult, &hostError, &dockerError); err != nil {
		t.Fatalf("query collection run: %v", err)
	}
	if result != "partial" || hostResult != "succeeded" || dockerResult != "failed" || hostError != nil || dockerError == nil || *dockerError != "docker_unavailable" {
		t.Fatalf("stored collection outcome = %q %q %q %#v %#v", result, hostResult, dockerResult, hostError, dockerError)
	}
}

func TestRecordCollectionRunRejectsInvalidOrReadOnlyWrites(t *testing.T) {
	t.Parallel()
	repository, database := openTestRepository(t)
	path := database.Path()
	invalid := validCollectionRun(time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC))
	invalid.Status = domain.CollectionFailed
	if _, err := repository.RecordCollectionRun(context.Background(), invalid); err == nil {
		t.Fatal("invalid collection run was stored")
	}
	if err := database.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	readOnlyDatabase, err := OpenReadOnly(context.Background(), path)
	if err != nil {
		t.Fatalf("OpenReadOnly() error = %v", err)
	}
	defer readOnlyDatabase.Close()
	readOnlyRepository, err := New(readOnlyDatabase)
	if err != nil {
		t.Fatalf("New(read-only) error = %v", err)
	}
	if _, err := readOnlyRepository.RecordCollectionRun(context.Background(), validCollectionRun(time.Date(2026, 8, 5, 12, 1, 0, 0, time.UTC))); err == nil {
		t.Fatal("read-only repository accepted a collection run")
	}
}

func TestRecordCollectionStoresRunHostAndCPUSamplesTogether(t *testing.T) {
	t.Parallel()
	repository, database := openTestRepository(t)
	defer database.Close()
	start := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	cpu, memory, core := 25.0, 60.0, 30.0
	psiReason := domain.ReasonNotSupported

	runID, err := repository.RecordCollection(context.Background(), CollectionRecord{
		Run: validCollectionRun(start),
		Host: &HostSampleRecord{
			ObservedAt: start, Availability: domain.AvailabilityAvailable,
			OverallCPUPercent: &cpu, MemoryUsagePercent: &memory,
			PSIAvailability: domain.AvailabilityUnavailable, PSIUnavailableReason: &psiReason,
		},
		CPUCores: []CPUCoreSampleRecord{
			{LogicalIndex: 0, ObservedAt: start, Availability: domain.AvailabilityAvailable, UsagePercent: &core},
		},
	})
	if err != nil {
		t.Fatalf("RecordCollection() error = %v", err)
	}
	var storedCPU, storedMemory, storedCore float64
	if err := database.SQL().QueryRow(
		"SELECT host.overall_cpu_percent, host.memory_usage_percent, core.usage_percent "+
			"FROM host_samples AS host "+
			"JOIN cpu_core_samples AS core ON core.collection_run_id = host.collection_run_id "+
			"WHERE host.collection_run_id = ? AND core.logical_index = 0",
		runID,
	).Scan(&storedCPU, &storedMemory, &storedCore); err != nil {
		t.Fatalf("query stored collection: %v", err)
	}
	if storedCPU != cpu || storedMemory != memory || storedCore != core {
		t.Fatalf("stored metrics = %v, %v, %v", storedCPU, storedMemory, storedCore)
	}

	invalidStart := start.Add(time.Minute)
	invalidCore := CPUCoreSampleRecord{
		LogicalIndex: 0, ObservedAt: invalidStart,
		Availability: domain.AvailabilityAvailable, UsagePercent: &core,
	}
	if _, err := repository.RecordCollection(context.Background(), CollectionRecord{
		Run: validCollectionRun(invalidStart), CPUCores: []CPUCoreSampleRecord{invalidCore, invalidCore},
	}); err == nil {
		t.Fatal("duplicate logical CPU samples were accepted")
	}
	var runCount int
	if err := database.SQL().QueryRow("SELECT count(*) FROM collection_runs").Scan(&runCount); err != nil || runCount != 1 {
		t.Fatalf("collection transaction count = %d, error = %v", runCount, err)
	}
}

func TestConcurrentCollectionRunWrites(t *testing.T) {
	repository, database := openTestRepository(t)
	defer database.Close()
	start := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	const workers = 24
	errorsChannel := make(chan error, workers)
	var waitGroup sync.WaitGroup
	for index := range workers {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			_, err := repository.RecordCollectionRun(context.Background(), validCollectionRun(start.Add(time.Duration(index)*time.Minute)))
			errorsChannel <- err
		}()
	}
	waitGroup.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		if err != nil {
			t.Errorf("concurrent RecordCollectionRun() error = %v", err)
		}
	}
	var count int
	if err := database.SQL().QueryRow("SELECT count(*) FROM collection_runs").Scan(&count); err != nil || count != workers {
		t.Fatalf("collection run count = %d, error = %v", count, err)
	}
}

func openTestRepository(t *testing.T) (*Repository, *storage.Database) {
	t.Helper()
	database, err := Open(context.Background(), filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	repository, err := New(database)
	if err != nil {
		database.Close()
		t.Fatalf("New() error = %v", err)
	}
	return repository, database
}

func validCollectionRun(start time.Time) domain.CollectionRun {
	finish := start.Add(time.Second)
	return domain.CollectionRun{
		StartedAt: start, FinishedAt: finish, Trigger: domain.CollectionTriggerScheduled,
		Status: domain.CollectionSucceeded,
		HostResult: domain.CollectionOutcome{
			Subsystem: domain.CollectionSubsystemHost, Status: domain.CollectionSucceeded,
			StartedAt: start, FinishedAt: finish,
		},
		DockerResult: domain.CollectionOutcome{
			Subsystem: domain.CollectionSubsystemDocker, Status: domain.CollectionSucceeded,
			StartedAt: start, FinishedAt: finish,
		},
	}
}
