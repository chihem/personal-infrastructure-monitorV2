package storage_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/storage"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/storage/audit"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/storage/history"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/storage/migrations"
)

func TestFreshDatabasesInitializeIndependently(t *testing.T) {
	t.Parallel()

	paths := testPaths(t)
	historyDatabase := openHistory(t, paths.history)
	defer historyDatabase.Close()
	auditDatabase := openAudit(t, paths.audit)
	defer auditDatabase.Close()

	assertTables(t, historyDatabase.SQL(), []string{
		"schema_migrations", "schema_compatibility", "collection_runs", "host_samples",
		"cpu_core_samples", "filesystems", "filesystem_samples", "block_devices",
		"block_device_io_samples", "containers", "container_samples",
		"container_state_events", "incidents",
	})
	assertTables(t, auditDatabase.SQL(), []string{"schema_migrations", "schema_compatibility", "audit_entries"})
	assertCompatibility(t, historyDatabase.SQL(), migrations.History)
	assertCompatibility(t, auditDatabase.SQL(), migrations.Audit)
}

func TestRepeatedStartupIsIdempotent(t *testing.T) {
	t.Parallel()

	path := testPaths(t).history
	first := openHistory(t, path)
	if _, err := first.SQL().Exec(`
		INSERT INTO collection_runs (
			started_at_unix, finished_at_unix, trigger_kind, result, host_result, docker_result
		) VALUES (1, 2, 'scheduled', 'succeeded', 'succeeded', 'succeeded')
	`); err != nil {
		t.Fatalf("insert collection run: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}

	second := openHistory(t, path)
	defer second.Close()
	var count int
	if err := second.SQL().QueryRow("SELECT count(*) FROM collection_runs").Scan(&count); err != nil {
		t.Fatalf("count collection runs: %v", err)
	}
	if count != 1 {
		t.Fatalf("collection run count = %d, want 1", count)
	}
	assertMigrationCount(t, second.SQL(), 1)
}

func TestWALAndConnectionPoliciesApplyToEveryPooledConnection(t *testing.T) {
	t.Parallel()

	database := openHistory(t, testPaths(t).history)
	defer database.Close()

	connections := make([]*sql.Conn, 0, storage.MaxOpenConnections)
	for range storage.MaxOpenConnections {
		connection, err := database.SQL().Conn(context.Background())
		if err != nil {
			t.Fatalf("Conn() error = %v", err)
		}
		connections = append(connections, connection)
	}
	defer func() {
		for _, connection := range connections {
			connection.Close()
		}
	}()

	for _, connection := range connections {
		assertPragma(t, connection, "journal_mode", "wal")
		assertPragma(t, connection, "foreign_keys", "1")
		assertPragma(t, connection, "busy_timeout", "5000")
	}
}

func TestForeignKeysAreEnforced(t *testing.T) {
	t.Parallel()

	database := openHistory(t, testPaths(t).history)
	defer database.Close()

	_, err := database.SQL().Exec(`
		INSERT INTO cpu_core_samples (
			collection_run_id, logical_index, observed_at_unix,
			availability, unavailable_reason, usage_percent
		) VALUES (999, 0, 1, 'available', NULL, 10)
	`)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "foreign key") {
		t.Fatalf("foreign-key insert error = %v", err)
	}
}

func TestWALAllowsWriteWhileReadTransactionIsOpen(t *testing.T) {
	t.Parallel()

	database := openHistory(t, testPaths(t).history)
	defer database.Close()
	if _, err := database.SQL().Exec(`
		INSERT INTO collection_runs (
			started_at_unix, finished_at_unix, trigger_kind, result, host_result, docker_result
		) VALUES (1, 2, 'scheduled', 'succeeded', 'succeeded', 'succeeded')
	`); err != nil {
		t.Fatalf("insert initial collection run: %v", err)
	}

	readTransaction, err := database.SQL().BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		t.Fatalf("BeginTx(read) error = %v", err)
	}
	defer readTransaction.Rollback()
	var count int
	if err := readTransaction.QueryRow("SELECT count(*) FROM collection_runs").Scan(&count); err != nil {
		t.Fatalf("read transaction query error = %v", err)
	}

	if _, err := database.SQL().Exec(`
		INSERT INTO collection_runs (
			started_at_unix, finished_at_unix, trigger_kind, result, host_result, docker_result
		) VALUES (3, 4, 'scheduled', 'succeeded', 'succeeded', 'succeeded')
	`); err != nil {
		t.Fatalf("write during read transaction error = %v", err)
	}
}

func TestConcurrentInitialization(t *testing.T) {
	path := testPaths(t).history

	const workers = 12
	var waitGroup sync.WaitGroup
	errorsChannel := make(chan error, workers)
	for range workers {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			database, err := history.Open(context.Background(), path)
			if err != nil {
				errorsChannel <- err
				return
			}
			if err := database.Close(); err != nil {
				errorsChannel <- err
			}
		}()
	}
	waitGroup.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		t.Errorf("concurrent Open() error = %v", err)
	}

	database := openHistory(t, path)
	defer database.Close()
	assertMigrationCount(t, database.SQL(), 1)
}

func TestAuditFailureDoesNotPreventReadOnlyHistory(t *testing.T) {
	t.Parallel()

	paths := testPaths(t)
	historyDatabase := openHistory(t, paths.history)
	if err := historyDatabase.Close(); err != nil {
		t.Fatalf("history Close() error = %v", err)
	}

	brokenAuditPath := filepath.Join(filepath.Dir(paths.audit), "broken.db")
	if err := os.Mkdir(brokenAuditPath, 0o700); err != nil {
		t.Fatalf("Mkdir(broken audit path) error = %v", err)
	}
	if database, err := audit.Open(context.Background(), brokenAuditPath); err == nil {
		database.Close()
		t.Fatal("audit Open() error = nil")
	}

	readOnly, err := history.OpenReadOnly(context.Background(), paths.history)
	if err != nil {
		t.Fatalf("history OpenReadOnly() error = %v", err)
	}
	defer readOnly.Close()
	if readOnly.Mode() != storage.ReadOnly {
		t.Fatalf("history mode = %q", readOnly.Mode())
	}
	var one int
	if err := readOnly.SQL().QueryRow("SELECT 1").Scan(&one); err != nil || one != 1 {
		t.Fatalf("read-only history query = %d, error = %v", one, err)
	}
	if _, err := readOnly.SQL().Exec("DELETE FROM collection_runs"); err == nil {
		t.Fatal("read-only DELETE error = nil")
	}
}

func TestReadOnlyOpenRejectsIncompleteSchema(t *testing.T) {
	t.Parallel()

	path := testPaths(t).history
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("WriteFile(empty database) error = %v", err)
	}
	database, err := history.OpenReadOnly(context.Background(), path)
	if err == nil {
		database.Close()
		t.Fatal("OpenReadOnly() error = nil")
	}
}

func TestDatabaseFilesAreProtected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose Unix permission bits")
	}

	database := openHistory(t, testPaths(t).history)
	defer database.Close()
	for _, path := range []string{database.Path(), database.Path() + "-wal", database.Path() + "-shm"} {
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			t.Fatalf("Stat(%q) error = %v", path, err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("permissions for %q = %o, want 600", path, info.Mode().Perm())
		}
	}
}

func TestSymlinkedDatabaseIsRejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows developer accounts may not have symlink permission")
	}

	paths := testPaths(t)
	target := filepath.Join(filepath.Dir(paths.history), "target.db")
	if err := os.WriteFile(target, nil, 0o600); err != nil {
		t.Fatalf("WriteFile(target) error = %v", err)
	}
	if err := os.Symlink(target, paths.history); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}
	database, err := history.Open(context.Background(), paths.history)
	if err == nil {
		database.Close()
		t.Fatal("history Open() error = nil")
	}
}

func TestAuditSchemaHasNoRawSecretOrLogFields(t *testing.T) {
	t.Parallel()

	database := openAudit(t, testPaths(t).audit)
	defer database.Close()
	columns := tableColumns(t, database.SQL(), "audit_entries")
	for _, prohibited := range []string{"token", "secret", "password", "header", "environment", "container_log", "exported_content"} {
		if slices.Contains(columns, prohibited) {
			t.Fatalf("audit_entries contains prohibited column %q", prohibited)
		}
	}
}

type databasePaths struct {
	history string
	audit   string
}

func testPaths(t *testing.T) databasePaths {
	t.Helper()
	directory := t.TempDir()
	return databasePaths{
		history: filepath.Join(directory, "history.db"),
		audit:   filepath.Join(directory, "audit.db"),
	}
}

func openHistory(t *testing.T, path string) *storage.Database {
	t.Helper()
	database, err := history.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("history Open() error = %v", err)
	}
	return database
}

func openAudit(t *testing.T, path string) *storage.Database {
	t.Helper()
	database, err := audit.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("audit Open() error = %v", err)
	}
	return database
}

func assertTables(t *testing.T, database *sql.DB, expected []string) {
	t.Helper()
	rows, err := database.Query(`SELECT name FROM sqlite_schema WHERE type = 'table'`)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table: %v", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate tables: %v", err)
	}
	for _, name := range expected {
		if !slices.Contains(names, name) {
			t.Errorf("missing table %q; got %v", name, names)
		}
	}
}

func assertCompatibility(t *testing.T, database *sql.DB, kind migrations.Kind) {
	t.Helper()
	compatibility, err := migrations.ReadCompatibility(context.Background(), database)
	if err != nil {
		t.Fatalf("ReadCompatibility() error = %v", err)
	}
	if compatibility.Kind != kind || compatibility.CurrentVersion != 1 || compatibility.MinimumReaderVersion != 1 {
		t.Fatalf("compatibility = %+v", compatibility)
	}
}

func assertMigrationCount(t *testing.T, database *sql.DB, want int) {
	t.Helper()
	var count int
	if err := database.QueryRow(`SELECT count(*) FROM schema_migrations WHERE is_applied = 1 AND version_id > 0`).Scan(&count); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if count != want {
		t.Fatalf("migration count = %d, want %d", count, want)
	}
}

type pragmaQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func assertPragma(t *testing.T, connection pragmaQuerier, name, want string) {
	t.Helper()
	var got string
	if err := connection.QueryRowContext(context.Background(), "PRAGMA "+name).Scan(&got); err != nil {
		t.Fatalf("PRAGMA %s error = %v", name, err)
	}
	if !strings.EqualFold(got, want) {
		t.Fatalf("PRAGMA %s = %q, want %q", name, got, want)
	}
}

func tableColumns(t *testing.T, database *sql.DB, table string) []string {
	t.Helper()
	rows, err := database.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatalf("table_info(%s): %v", table, err)
	}
	defer rows.Close()
	var columns []string
	for rows.Next() {
		var sequence, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&sequence, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatalf("scan table_info(%s): %v", table, err)
		}
		columns = append(columns, name)
	}
	return columns
}
