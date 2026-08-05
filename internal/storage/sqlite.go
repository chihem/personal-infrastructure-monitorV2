// Package storage provides the shared SQLite connection policy used by the
// independent history and audit stores.
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/storage/migrations"
	_ "modernc.org/sqlite"
)

const (
	DefaultHistoryPath = "/var/lib/pim/history.db"
	DefaultAuditPath   = "/var/lib/pim/audit.db"

	BusyTimeoutMilliseconds = 5000
	MaxOpenConnections      = 4
)

type Mode string

const (
	ReadWrite Mode = "read_write"
	ReadOnly  Mode = "read_only"
)

type Database struct {
	database *sql.DB
	path     string
	kind     migrations.Kind
	mode     Mode
}

var openLocks sync.Map

func Open(ctx context.Context, path string, kind migrations.Kind, mode Mode) (*Database, error) {
	if kind != migrations.History && kind != migrations.Audit {
		return nil, fmt.Errorf("unsupported SQLite database kind %q", kind)
	}
	lockKey, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve SQLite database path: %w", err)
	}
	unlock := lockOpenPath(lockKey)
	defer unlock()

	absolutePath, err := preparePath(path, mode)
	if err != nil {
		return nil, err
	}

	database, err := sql.Open("sqlite", dataSourceName(absolutePath, mode))
	if err != nil {
		return nil, fmt.Errorf("open %s SQLite database: %w", kind, err)
	}
	database.SetMaxOpenConns(MaxOpenConnections)
	database.SetMaxIdleConns(MaxOpenConnections)
	database.SetConnMaxIdleTime(0)
	database.SetConnMaxLifetime(0)

	closeWithError := func(openErr error) (*Database, error) {
		if closeErr := database.Close(); closeErr != nil {
			return nil, errors.Join(openErr, fmt.Errorf("close database after open failure: %w", closeErr))
		}
		return nil, openErr
	}
	if err := database.PingContext(ctx); err != nil {
		return closeWithError(fmt.Errorf("connect to %s SQLite database: %w", kind, err))
	}

	if mode == ReadWrite {
		if err := migrations.Apply(ctx, database, kind, absolutePath); err != nil {
			return closeWithError(err)
		}
	} else {
		if err := migrations.Validate(ctx, database, kind); err != nil {
			return closeWithError(err)
		}
	}
	if err := verifyConnectionPolicy(ctx, database); err != nil {
		return closeWithError(err)
	}
	if err := protectDatabaseFiles(absolutePath); err != nil {
		return closeWithError(err)
	}

	return &Database{database: database, path: absolutePath, kind: kind, mode: mode}, nil
}

func lockOpenPath(path string) func() {
	value, _ := openLocks.LoadOrStore(filepath.Clean(path), &sync.Mutex{})
	mutex := value.(*sync.Mutex)
	mutex.Lock()
	return mutex.Unlock
}

func (database *Database) SQL() *sql.DB {
	return database.database
}

func (database *Database) Path() string {
	return database.path
}

func (database *Database) Kind() migrations.Kind {
	return database.kind
}

func (database *Database) Mode() Mode {
	return database.mode
}

func (database *Database) Close() error {
	return database.database.Close()
}

func preparePath(path string, mode Mode) (string, error) {
	if mode != ReadWrite && mode != ReadOnly {
		return "", fmt.Errorf("unsupported SQLite mode %q", mode)
	}
	if strings.TrimSpace(path) == "" {
		return "", errors.New("SQLite database path is required")
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve SQLite database path: %w", err)
	}
	if !strings.EqualFold(filepath.Ext(absolutePath), ".db") {
		return "", errors.New("SQLite database path must end in .db")
	}

	directory := filepath.Dir(absolutePath)
	if mode == ReadWrite {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return "", fmt.Errorf("create SQLite data directory: %w", err)
		}
	}
	if info, err := os.Lstat(absolutePath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("SQLite database path must not be a symbolic link")
		}
		if !info.Mode().IsRegular() {
			return "", errors.New("SQLite database path is not a regular file")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect SQLite database path: %w", err)
	} else if mode == ReadOnly {
		return "", fmt.Errorf("open read-only SQLite database: %w", err)
	} else {
		file, createErr := os.OpenFile(absolutePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if createErr != nil {
			if !errors.Is(createErr, os.ErrExist) {
				return "", fmt.Errorf("create protected SQLite database file: %w", createErr)
			}
			concurrentInfo, inspectErr := os.Lstat(absolutePath)
			if inspectErr != nil {
				return "", fmt.Errorf("inspect concurrently created SQLite database file: %w", inspectErr)
			}
			if concurrentInfo.Mode()&os.ModeSymlink != 0 || !concurrentInfo.Mode().IsRegular() {
				return "", errors.New("concurrently created SQLite database path is not a regular file")
			}
		} else if closeErr := file.Close(); closeErr != nil {
			return "", fmt.Errorf("close new SQLite database file: %w", closeErr)
		}
	}
	if err := os.Chmod(absolutePath, 0o600); err != nil {
		return "", fmt.Errorf("protect SQLite database file: %w", err)
	}
	return absolutePath, nil
}

func dataSourceName(path string, mode Mode) string {
	escapedPath := (&url.URL{Path: filepath.ToSlash(path)}).EscapedPath()
	query := make(url.Values)
	if mode == ReadOnly {
		query.Set("mode", "ro")
		query.Add("_pragma", "query_only(1)")
	} else {
		query.Set("mode", "rwc")
		query.Add("_pragma", "journal_mode(WAL)")
		query.Add("_pragma", "synchronous(NORMAL)")
		query.Add("_pragma", "wal_autocheckpoint(1000)")
	}
	query.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", BusyTimeoutMilliseconds))
	query.Add("_pragma", "foreign_keys(1)")
	query.Add("_pragma", "trusted_schema(0)")
	query.Set("_dqs", "0")
	return "file:" + escapedPath + "?" + query.Encode()
}

func verifyConnectionPolicy(ctx context.Context, database *sql.DB) error {
	checks := []struct {
		pragma string
		want   string
	}{
		{pragma: "journal_mode", want: "wal"},
		{pragma: "foreign_keys", want: "1"},
		{pragma: "busy_timeout", want: fmt.Sprint(BusyTimeoutMilliseconds)},
	}
	for _, check := range checks {
		var got string
		if err := database.QueryRowContext(ctx, "PRAGMA "+check.pragma).Scan(&got); err != nil {
			return fmt.Errorf("verify SQLite %s: %w", check.pragma, err)
		}
		if !strings.EqualFold(got, check.want) {
			return fmt.Errorf("unsafe SQLite %s: got %q, want %q", check.pragma, got, check.want)
		}
	}
	return nil
}

func protectDatabaseFiles(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	for _, candidate := range []string{path, path + "-wal", path + "-shm"} {
		if err := os.Chmod(candidate, 0o600); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("protect SQLite file %q: %w", candidate, err)
		}
	}
	return nil
}
