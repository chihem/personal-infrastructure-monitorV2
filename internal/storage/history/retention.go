package history

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const (
	DefaultRetentionBatchSize  = 250
	MaximumRetentionBatchSize  = 1000
	DefaultRetentionMaxBatches = 20
	RetentionRunInterval       = 24 * time.Hour
)

type RetentionResult struct {
	Cutoff                time.Time
	CollectionRunsDeleted int64
	StateEventsDeleted    int64
	IncidentsDeleted      int64
	FilesystemsDeleted    int64
	BlockDevicesDeleted   int64
	ContainersDeleted     int64
	More                  bool
}

func (result *RetentionResult) add(other RetentionResult) {
	result.Cutoff = other.Cutoff
	result.CollectionRunsDeleted += other.CollectionRunsDeleted
	result.StateEventsDeleted += other.StateEventsDeleted
	result.IncidentsDeleted += other.IncidentsDeleted
	result.FilesystemsDeleted += other.FilesystemsDeleted
	result.BlockDevicesDeleted += other.BlockDevicesDeleted
	result.ContainersDeleted += other.ContainersDeleted
	result.More = other.More
}

func (repository *Repository) CleanupExpired(ctx context.Context, now time.Time, batchSize int) (RetentionResult, error) {
	if repository == nil || repository.database == nil {
		return RetentionResult{}, errors.New("history repository is unavailable")
	}
	if !repository.writable {
		return RetentionResult{}, errors.New("history repository is read-only")
	}
	if now.IsZero() {
		return RetentionResult{}, errors.New("current time is required")
	}
	if batchSize < 1 || batchSize > MaximumRetentionBatchSize {
		return RetentionResult{}, fmt.Errorf("retention batch size must be between 1 and %d", MaximumRetentionBatchSize)
	}

	cutoff := now.UTC().Add(-RetentionDuration)
	transaction, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return RetentionResult{}, fmt.Errorf("begin retention transaction: %w", err)
	}
	defer transaction.Rollback()

	result := RetentionResult{Cutoff: cutoff}
	result.StateEventsDeleted, err = deleteBatch(ctx, transaction, `
		DELETE FROM container_state_events
		WHERE id IN (
			SELECT id FROM container_state_events
			WHERE observed_at_unix < ? ORDER BY observed_at_unix, id LIMIT ?
		)
	`, cutoff.Unix(), batchSize)
	if err != nil {
		return RetentionResult{}, fmt.Errorf("delete expired container events: %w", err)
	}
	result.IncidentsDeleted, err = deleteBatch(ctx, transaction, `
		DELETE FROM incidents
		WHERE id IN (
			SELECT id FROM incidents
			WHERE ended_at_unix IS NOT NULL AND ended_at_unix < ?
			ORDER BY ended_at_unix, id LIMIT ?
		)
	`, cutoff.Unix(), batchSize)
	if err != nil {
		return RetentionResult{}, fmt.Errorf("delete expired incidents: %w", err)
	}
	result.CollectionRunsDeleted, err = deleteBatch(ctx, transaction, `
		DELETE FROM collection_runs
		WHERE id IN (
			SELECT id FROM collection_runs
			WHERE started_at_unix < ? ORDER BY started_at_unix, id LIMIT ?
		)
	`, cutoff.Unix(), batchSize)
	if err != nil {
		return RetentionResult{}, fmt.Errorf("delete expired collection runs: %w", err)
	}

	result.FilesystemsDeleted, err = deleteBatch(ctx, transaction, `
		DELETE FROM filesystems
		WHERE id IN (
			SELECT resources.id FROM filesystems AS resources
			WHERE resources.removed_at_unix IS NOT NULL
			  AND resources.removed_at_unix < ?
			  AND NOT EXISTS (
				SELECT 1 FROM filesystem_samples AS samples
				WHERE samples.filesystem_id = resources.id
			  )
			ORDER BY resources.removed_at_unix, resources.id LIMIT ?
		)
	`, cutoff.Unix(), batchSize)
	if err != nil {
		return RetentionResult{}, fmt.Errorf("delete expired filesystem resources: %w", err)
	}
	result.BlockDevicesDeleted, err = deleteBatch(ctx, transaction, `
		DELETE FROM block_devices
		WHERE id IN (
			SELECT resources.id FROM block_devices AS resources
			WHERE resources.removed_at_unix IS NOT NULL
			  AND resources.removed_at_unix < ?
			  AND NOT EXISTS (
				SELECT 1 FROM block_device_io_samples AS samples
				WHERE samples.block_device_id = resources.id
			  )
			ORDER BY resources.removed_at_unix, resources.id LIMIT ?
		)
	`, cutoff.Unix(), batchSize)
	if err != nil {
		return RetentionResult{}, fmt.Errorf("delete expired block-device resources: %w", err)
	}
	result.ContainersDeleted, err = deleteBatch(ctx, transaction, `
		DELETE FROM containers
		WHERE docker_id IN (
			SELECT resources.docker_id FROM containers AS resources
			WHERE resources.deleted_at_unix IS NOT NULL
			  AND resources.deleted_at_unix < ?
			  AND NOT EXISTS (
				SELECT 1 FROM container_samples AS samples
				WHERE samples.container_id = resources.docker_id
			  )
			  AND NOT EXISTS (
				SELECT 1 FROM container_state_events AS events
				WHERE events.container_id = resources.docker_id
			  )
			ORDER BY resources.deleted_at_unix, resources.docker_id LIMIT ?
		)
	`, cutoff.Unix(), batchSize)
	if err != nil {
		return RetentionResult{}, fmt.Errorf("delete expired container resources: %w", err)
	}

	result.More = result.CollectionRunsDeleted == int64(batchSize) ||
		result.StateEventsDeleted == int64(batchSize) ||
		result.IncidentsDeleted == int64(batchSize) ||
		result.FilesystemsDeleted == int64(batchSize) ||
		result.BlockDevicesDeleted == int64(batchSize) ||
		result.ContainersDeleted == int64(batchSize)
	if err := transaction.Commit(); err != nil {
		return RetentionResult{}, fmt.Errorf("commit retention transaction: %w", err)
	}
	return result, nil
}

func (repository *Repository) CleanupExpiredBatches(
	ctx context.Context,
	now time.Time,
	batchSize int,
	maximumBatches int,
) (RetentionResult, error) {
	if maximumBatches < 1 || maximumBatches > DefaultRetentionMaxBatches {
		return RetentionResult{}, fmt.Errorf("maximum retention batches must be between 1 and %d", DefaultRetentionMaxBatches)
	}
	var total RetentionResult
	for range maximumBatches {
		result, err := repository.CleanupExpired(ctx, now, batchSize)
		if err != nil {
			return RetentionResult{}, err
		}
		total.add(result)
		if !result.More {
			return total, nil
		}
		if err := ctx.Err(); err != nil {
			return RetentionResult{}, err
		}
	}
	total.More = true
	return total, nil
}

func deleteBatch(ctx context.Context, transaction *sql.Tx, statement string, cutoffUnix int64, batchSize int) (int64, error) {
	result, err := transaction.ExecContext(ctx, statement, cutoffUnix, batchSize)
	if err != nil {
		return 0, err
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return deleted, nil
}
