package history

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/storage"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/storage/migrations"
)

type Repository struct {
	database *sql.DB
	writable bool
}

type CollectionRecord struct {
	Run      domain.CollectionRun
	Host     *HostSampleRecord
	CPUCores []CPUCoreSampleRecord
}

func New(database *storage.Database) (*Repository, error) {
	if database == nil || database.SQL() == nil {
		return nil, errors.New("history database is required")
	}
	if database.Kind() != migrations.History {
		return nil, errors.New("history repository requires a history database")
	}
	return &Repository{database: database.SQL(), writable: database.Mode() == storage.ReadWrite}, nil
}

func (repository *Repository) RecordCollectionRun(ctx context.Context, run domain.CollectionRun) (int64, error) {
	return repository.RecordCollection(ctx, CollectionRecord{Run: run})
}

func (repository *Repository) RecordCollection(ctx context.Context, record CollectionRecord) (int64, error) {
	if repository == nil || repository.database == nil {
		return 0, errors.New("history repository is unavailable")
	}
	if !repository.writable {
		return 0, errors.New("history repository is read-only")
	}
	if err := record.validate(); err != nil {
		return 0, fmt.Errorf("validate collection run: %w", err)
	}

	transaction, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin collection transaction: %w", err)
	}
	defer transaction.Rollback()

	runID, err := insertCollectionRun(ctx, transaction, record.Run)
	if err != nil {
		return 0, err
	}
	if err := insertCollectionSamples(ctx, transaction, runID, record); err != nil {
		return 0, err
	}
	if err := transaction.Commit(); err != nil {
		return 0, fmt.Errorf("commit collection transaction: %w", err)
	}
	return runID, nil
}

func insertCollectionRun(ctx context.Context, transaction *sql.Tx, run domain.CollectionRun) (int64, error) {
	result, err := transaction.ExecContext(ctx, `
		INSERT INTO collection_runs (
			started_at_unix, finished_at_unix, trigger_kind, result,
			host_result, docker_result, error_code,
			host_error_code, docker_error_code
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		run.StartedAt.Unix(), run.FinishedAt.Unix(), run.Trigger, run.Status,
		run.HostResult.Status, run.DockerResult.Status, primaryCollectionError(run),
		nullableString(run.HostResult.ErrorCode), nullableString(run.DockerResult.ErrorCode),
	)
	if err != nil {
		return 0, fmt.Errorf("insert collection run: %w", err)
	}
	runID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read collection run ID: %w", err)
	}
	return runID, nil
}

func primaryCollectionError(run domain.CollectionRun) any {
	hostCode := run.HostResult.ErrorCode
	dockerCode := run.DockerResult.ErrorCode
	switch {
	case hostCode == "":
		return nullableString(dockerCode)
	case dockerCode == "" || dockerCode == hostCode:
		return hostCode
	default:
		return "multiple_failures"
	}
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
