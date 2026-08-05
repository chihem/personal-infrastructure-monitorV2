package history

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func TestCleanupExpiredBatchesPreservesBoundaryAndActiveEvidence(t *testing.T) {
	t.Parallel()
	repository, database := openTestRepository(t)
	defer database.Close()
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	cutoff := now.Add(-RetentionDuration)

	oldRunOne := insertRunAt(t, repository, cutoff.Add(-2*time.Minute))
	oldRunTwo := insertRunAt(t, repository, cutoff.Add(-time.Minute))
	boundaryRun := insertRunAt(t, repository, cutoff)
	insertMinimalHostSample(t, database.SQL(), oldRunOne, cutoff.Add(-2*time.Minute))
	insertMinimalHostSample(t, database.SQL(), oldRunTwo, cutoff.Add(-time.Minute))
	insertMinimalHostSample(t, database.SQL(), boundaryRun, cutoff)

	if _, err := database.SQL().Exec(`
		INSERT INTO filesystems (
			id, mount_path, device_name, filesystem_type,
			first_seen_at_unix, last_seen_at_unix, removed_at_unix
		) VALUES ('fs-old', '/old', 'old', 'ext4', ?, ?, ?)
	`, cutoff.Add(-time.Hour).Unix(), cutoff.Add(-time.Minute).Unix(), cutoff.Add(-time.Minute).Unix()); err != nil {
		t.Fatalf("insert removed filesystem: %v", err)
	}
	if _, err := database.SQL().Exec(`
		INSERT INTO incidents (
			health_state, cause_code, subject_kind, subject_id, subject_display_name,
			started_at_unix, updated_at_unix, ended_at_unix
		) VALUES
		('warning', 'cpu_warning', 'cpu', 'overall', 'Overall CPU', ?, ?, ?),
		('warning', 'cpu_warning', 'cpu', 'active', 'Active CPU', ?, ?, NULL),
		('warning', 'cpu_warning', 'cpu', 'boundary', 'Boundary CPU', ?, ?, ?)
	`,
		cutoff.Add(-time.Hour).Unix(), cutoff.Add(-time.Minute).Unix(), cutoff.Add(-time.Minute).Unix(),
		cutoff.Add(-time.Hour).Unix(), now.Unix(),
		cutoff.Add(-time.Hour).Unix(), cutoff.Unix(), cutoff.Unix(),
	); err != nil {
		t.Fatalf("insert incidents: %v", err)
	}

	first, err := repository.CleanupExpired(context.Background(), now, 1)
	if err != nil {
		t.Fatalf("CleanupExpired() error = %v", err)
	}
	if first.CollectionRunsDeleted != 1 || first.IncidentsDeleted != 1 || !first.More {
		t.Fatalf("first retention batch = %+v", first)
	}
	total, err := repository.CleanupExpiredBatches(context.Background(), now, 1, DefaultRetentionMaxBatches)
	if err != nil {
		t.Fatalf("CleanupExpiredBatches() error = %v", err)
	}
	if total.CollectionRunsDeleted != 1 || total.More {
		t.Fatalf("remaining retention result = %+v", total)
	}
	assertCount(t, database.SQL(), "collection_runs", 1)
	assertCount(t, database.SQL(), "host_samples", 1)
	assertCount(t, database.SQL(), "filesystems", 0)
	assertCount(t, database.SQL(), "incidents", 2)
}

func TestCleanupExpiredRejectsUnsafeBoundsAndCancellation(t *testing.T) {
	t.Parallel()
	repository, database := openTestRepository(t)
	defer database.Close()
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	if _, err := repository.CleanupExpired(context.Background(), now, 0); err == nil {
		t.Fatal("zero retention batch was accepted")
	}
	if _, err := repository.CleanupExpired(context.Background(), now, MaximumRetentionBatchSize+1); err == nil {
		t.Fatal("oversized retention batch was accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := repository.CleanupExpired(ctx, now, 1); err == nil {
		t.Fatal("cancelled retention context was accepted")
	}
}

func insertRunAt(t *testing.T, repository *Repository, at time.Time) int64 {
	t.Helper()
	id, err := repository.RecordCollectionRun(context.Background(), validCollectionRun(at))
	if err != nil {
		t.Fatalf("RecordCollectionRun() error = %v", err)
	}
	return id
}

func insertMinimalHostSample(t *testing.T, database sqlExecutor, runID int64, observedAt time.Time) {
	t.Helper()
	if _, err := database.Exec(`
		INSERT INTO host_samples (
			collection_run_id, observed_at_unix, availability, unavailable_reason,
			psi_availability, psi_unavailable_reason
		) VALUES (?, ?, 'unavailable', 'not_collected', 'unavailable', 'not_collected')
	`, runID, observedAt.Unix()); err != nil {
		t.Fatalf("insert host sample: %v", err)
	}
}

func assertCount(t *testing.T, database rowQuerier, table string, want int) {
	t.Helper()
	var count int
	if err := database.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if count != want {
		t.Fatalf("%s count = %d, want %d", table, count, want)
	}
}

type rowQuerier interface {
	QueryRow(string, ...any) *sql.Row
}
