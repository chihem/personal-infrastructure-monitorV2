package migrations

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"testing/fstest"

	_ "modernc.org/sqlite"
)

func TestGooseUpgradeAppliesOnlyPendingMigration(t *testing.T) {
	t.Parallel()

	database := openMemoryDatabase(t)
	defer database.Close()

	versionOne := fstest.MapFS{
		"history/00001_first.sql": {Data: []byte("-- +goose Up\nCREATE TABLE example (id INTEGER PRIMARY KEY) STRICT;")},
	}
	provider, err := newProvider(database, History, versionOne)
	if err != nil {
		t.Fatalf("newProvider(version one) error = %v", err)
	}
	results, err := provider.Up(context.Background())
	if err != nil {
		t.Fatalf("Up(version one) error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("version-one results = %d, want 1", len(results))
	}

	versionTwo := fstest.MapFS{
		"history/00001_first.sql":  {Data: []byte("-- +goose Up\nCREATE TABLE example (id INTEGER PRIMARY KEY) STRICT;")},
		"history/00002_second.sql": {Data: []byte("-- +goose Up\nALTER TABLE example ADD COLUMN name TEXT;")},
	}
	provider, err = newProvider(database, History, versionTwo)
	if err != nil {
		t.Fatalf("newProvider(version two) error = %v", err)
	}
	results, err = provider.Up(context.Background())
	if err != nil {
		t.Fatalf("Up(version two) error = %v", err)
	}
	if len(results) != 1 || results[0].Source.Version != 2 {
		t.Fatalf("version-two results = %+v", results)
	}
	assertColumnExists(t, database, "example", "name")
}

func TestFailedMigrationRollsBackSchemaAndVersion(t *testing.T) {
	t.Parallel()

	database := openMemoryDatabase(t)
	defer database.Close()
	filesystem := fstest.MapFS{
		"history/00001_broken.sql": {Data: []byte(`
-- +goose Up
CREATE TABLE must_roll_back (id INTEGER PRIMARY KEY) STRICT;
THIS IS NOT VALID SQL;
`)},
	}
	provider, err := newProvider(database, History, filesystem)
	if err != nil {
		t.Fatalf("newProvider() error = %v", err)
	}
	if _, err := provider.Up(context.Background()); err == nil {
		t.Fatal("Up() error = nil")
	}

	var tableCount int
	if err := database.QueryRow(`SELECT count(*) FROM sqlite_schema WHERE type = 'table' AND name = 'must_roll_back'`).Scan(&tableCount); err != nil {
		t.Fatalf("query rolled-back table: %v", err)
	}
	if tableCount != 0 {
		t.Fatalf("rolled-back table count = %d", tableCount)
	}
	current, err := provider.GetDBVersion(context.Background())
	if err != nil {
		t.Fatalf("GetDBVersion() error = %v", err)
	}
	if current != 0 {
		t.Fatalf("database version = %d, want 0", current)
	}
}

func TestValidateRejectsNewerSchema(t *testing.T) {
	t.Parallel()

	database := openMemoryDatabase(t)
	defer database.Close()
	provider, err := newProvider(database, History, files)
	if err != nil {
		t.Fatalf("newProvider() error = %v", err)
	}
	if _, err := provider.Up(context.Background()); err != nil {
		t.Fatalf("Up() error = %v", err)
	}
	if _, err := database.Exec(`
		INSERT INTO schema_migrations (version_id, is_applied, tstamp)
		VALUES (999, 1, CURRENT_TIMESTAMP)
	`); err != nil {
		t.Fatalf("insert future migration: %v", err)
	}
	if err := Validate(context.Background(), database, History); !errors.Is(err, ErrSchemaTooNew) {
		t.Fatalf("Validate() error = %v, want ErrSchemaTooNew", err)
	}
}

func openMemoryDatabase(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	database.SetMaxOpenConns(1)
	if err := database.Ping(); err != nil {
		database.Close()
		t.Fatalf("Ping() error = %v", err)
	}
	return database
}

func assertColumnExists(t *testing.T, database *sql.DB, table, column string) {
	t.Helper()
	rows, err := database.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatalf("table_info(%s): %v", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var sequence, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&sequence, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatalf("scan table_info(%s): %v", table, err)
		}
		if name == column {
			return
		}
	}
	t.Fatalf("column %s.%s not found", table, column)
}
