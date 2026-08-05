package migrations

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sync"

	"github.com/pressly/goose/v3"
)

const migrationTable = "schema_migrations"

type Kind string

const (
	History Kind = "history"
	Audit   Kind = "audit"
)

var (
	ErrSchemaTooNew       = errors.New("database schema is newer than this application")
	ErrSchemaIncomplete   = errors.New("database schema is incomplete")
	ErrSchemaIncompatible = errors.New("database schema compatibility metadata is invalid")
)

//go:embed history/*.sql audit/*.sql
var files embed.FS

var pathLocks sync.Map

type Compatibility struct {
	Kind                          Kind
	CurrentVersion                int64
	MinimumReaderVersion          int64
	BackwardCompatibleFromVersion int64
	AppliedAtUnix                 int64
}

func Apply(ctx context.Context, database *sql.DB, kind Kind, databasePath string) error {
	unlock := lockPath(databasePath)
	defer unlock()

	provider, err := newProvider(database, kind, files)
	if err != nil {
		return err
	}
	current, target, err := provider.GetVersions(ctx)
	if err != nil {
		return fmt.Errorf("read %s schema version: %w", kind, err)
	}
	if current > target {
		return fmt.Errorf("%w: %s database is at version %d, application supports %d", ErrSchemaTooNew, kind, current, target)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply %s migrations: %w", kind, err)
	}
	return Validate(ctx, database, kind)
}

func Validate(ctx context.Context, database *sql.DB, kind Kind) error {
	var migrationTableExists int
	if err := database.QueryRowContext(ctx, `
		SELECT count(*)
		FROM sqlite_schema
		WHERE type = 'table' AND name = 'schema_migrations'
	`).Scan(&migrationTableExists); err != nil {
		return fmt.Errorf("inspect %s migration metadata: %w", kind, err)
	}
	if migrationTableExists != 1 {
		return fmt.Errorf("%w: %s database has no migration metadata", ErrSchemaIncomplete, kind)
	}

	provider, err := newProvider(database, kind, files)
	if err != nil {
		return err
	}
	current, target, err := provider.GetVersions(ctx)
	if err != nil {
		return fmt.Errorf("read %s schema version: %w", kind, err)
	}
	if current > target {
		return fmt.Errorf("%w: %s database is at version %d, application supports %d", ErrSchemaTooNew, kind, current, target)
	}
	if current != target {
		return fmt.Errorf("%w: %s database is at version %d, expected %d", ErrSchemaIncomplete, kind, current, target)
	}

	compatibility, err := ReadCompatibility(ctx, database)
	if err != nil {
		return err
	}
	if compatibility.Kind != kind || compatibility.CurrentVersion != target ||
		compatibility.MinimumReaderVersion > target ||
		compatibility.BackwardCompatibleFromVersion > target {
		return fmt.Errorf("%w: got %+v for %s version %d", ErrSchemaIncompatible, compatibility, kind, target)
	}
	return nil
}

func ReadCompatibility(ctx context.Context, database *sql.DB) (Compatibility, error) {
	var compatibility Compatibility
	err := database.QueryRowContext(ctx, `
		SELECT schema_kind, current_version, minimum_reader_version,
		       backward_compatible_from_version, applied_at_unix
		FROM schema_compatibility
		WHERE singleton_id = 1
	`).Scan(
		&compatibility.Kind,
		&compatibility.CurrentVersion,
		&compatibility.MinimumReaderVersion,
		&compatibility.BackwardCompatibleFromVersion,
		&compatibility.AppliedAtUnix,
	)
	if err != nil {
		return Compatibility{}, fmt.Errorf("read schema compatibility metadata: %w", err)
	}
	return compatibility, nil
}

func newProvider(database *sql.DB, kind Kind, filesystem fs.FS) (*goose.Provider, error) {
	if database == nil {
		return nil, errors.New("database is required")
	}
	if kind != History && kind != Audit {
		return nil, fmt.Errorf("unsupported database kind %q", kind)
	}
	subtree, err := fs.Sub(filesystem, string(kind))
	if err != nil {
		return nil, fmt.Errorf("open embedded %s migrations: %w", kind, err)
	}
	provider, err := goose.NewProvider(
		goose.DialectSQLite3,
		database,
		subtree,
		goose.WithTableName(migrationTable),
		goose.WithDisableGlobalRegistry(true),
	)
	if err != nil {
		return nil, fmt.Errorf("create %s migration provider: %w", kind, err)
	}
	return provider, nil
}

func lockPath(path string) func() {
	value, _ := pathLocks.LoadOrStore(path, &sync.Mutex{})
	mutex := value.(*sync.Mutex)
	mutex.Lock()
	return mutex.Unlock
}
