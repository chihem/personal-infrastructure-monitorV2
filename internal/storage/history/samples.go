package history

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"reflect"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

type HostSampleRecord struct {
	ObservedAt        time.Time
	Availability      domain.Availability
	UnavailableReason *domain.UnavailabilityReason

	OverallCPUPercent      *float64
	Load1                  *float64
	Load5                  *float64
	Load15                 *float64
	MemoryTotalBytes       *int64
	MemoryUsedBytes        *int64
	MemoryAvailableBytes   *int64
	MemoryFreeBytes        *int64
	MemoryCachedBytes      *int64
	MemoryBufferedBytes    *int64
	MemoryUsagePercent     *float64
	SwapTotalBytes         *int64
	SwapUsedBytes          *int64
	PSIAvailability        domain.Availability
	PSIUnavailableReason   *domain.UnavailabilityReason
	MemoryPSISomeAverage10 *float64
	MemoryPSIFullAverage10 *float64
	MemoryPSISomeTotalUS   *int64
	MemoryPSIFullTotalUS   *int64
}

type CPUCoreSampleRecord struct {
	LogicalIndex      int
	ObservedAt        time.Time
	Availability      domain.Availability
	UnavailableReason *domain.UnavailabilityReason
	UsagePercent      *float64
}

func (record CollectionRecord) validate() error {
	if err := record.Run.Validate(); err != nil {
		return err
	}
	if record.Host != nil {
		if err := record.Host.validate(record.Run); err != nil {
			return fmt.Errorf("host sample: %w", err)
		}
	}
	seenCores := make(map[int]struct{}, len(record.CPUCores))
	for index, sample := range record.CPUCores {
		if err := sample.validate(record.Run); err != nil {
			return fmt.Errorf("CPU core sample %d: %w", index, err)
		}
		if _, exists := seenCores[sample.LogicalIndex]; exists {
			return fmt.Errorf("CPU core sample %d duplicates logical index %d", index, sample.LogicalIndex)
		}
		seenCores[sample.LogicalIndex] = struct{}{}
	}
	return nil
}

func (sample HostSampleRecord) validate(run domain.CollectionRun) error {
	if err := validateSampleTime(sample.ObservedAt, run); err != nil {
		return err
	}
	if err := validateAvailability(sample.Availability, sample.UnavailableReason); err != nil {
		return fmt.Errorf("availability: %w", err)
	}
	if err := validateAvailability(sample.PSIAvailability, sample.PSIUnavailableReason); err != nil {
		return fmt.Errorf("PSI availability: %w", err)
	}
	if sample.Availability == domain.AvailabilityUnavailable && anyNonNil(
		sample.OverallCPUPercent, sample.Load1, sample.Load5, sample.Load15,
		sample.MemoryTotalBytes, sample.MemoryUsedBytes, sample.MemoryAvailableBytes,
		sample.MemoryFreeBytes, sample.MemoryCachedBytes, sample.MemoryBufferedBytes,
		sample.MemoryUsagePercent, sample.SwapTotalBytes, sample.SwapUsedBytes,
	) {
		return fmt.Errorf("unavailable host sample cannot contain host values")
	}
	if sample.PSIAvailability == domain.AvailabilityUnavailable && anyNonNil(
		sample.MemoryPSISomeAverage10, sample.MemoryPSIFullAverage10,
		sample.MemoryPSISomeTotalUS, sample.MemoryPSIFullTotalUS,
	) {
		return fmt.Errorf("unavailable PSI sample cannot contain PSI values")
	}
	for name, value := range map[string]*float64{
		"overall CPU percent":  sample.OverallCPUPercent,
		"memory usage percent": sample.MemoryUsagePercent,
		"PSI some average":     sample.MemoryPSISomeAverage10,
		"PSI full average":     sample.MemoryPSIFullAverage10,
	} {
		if err := validateFiniteNonNegative(name, value); err != nil {
			return err
		}
		if value != nil && *value > 100 {
			return fmt.Errorf("%s must not exceed 100", name)
		}
	}
	for name, value := range map[string]*float64{
		"load 1": sample.Load1, "load 5": sample.Load5, "load 15": sample.Load15,
	} {
		if err := validateFiniteNonNegative(name, value); err != nil {
			return err
		}
	}
	for name, value := range map[string]*int64{
		"memory total": sample.MemoryTotalBytes, "memory used": sample.MemoryUsedBytes,
		"memory available": sample.MemoryAvailableBytes, "memory free": sample.MemoryFreeBytes,
		"memory cached": sample.MemoryCachedBytes, "memory buffered": sample.MemoryBufferedBytes,
		"swap total": sample.SwapTotalBytes, "swap used": sample.SwapUsedBytes,
		"PSI some total": sample.MemoryPSISomeTotalUS, "PSI full total": sample.MemoryPSIFullTotalUS,
	} {
		if value != nil && *value < 0 {
			return fmt.Errorf("%s must not be negative", name)
		}
	}
	return nil
}

func (sample CPUCoreSampleRecord) validate(run domain.CollectionRun) error {
	if sample.LogicalIndex < 0 {
		return fmt.Errorf("logical index must not be negative")
	}
	if err := validateSampleTime(sample.ObservedAt, run); err != nil {
		return err
	}
	if err := validateAvailability(sample.Availability, sample.UnavailableReason); err != nil {
		return err
	}
	if sample.Availability == domain.AvailabilityAvailable && sample.UsagePercent == nil {
		return fmt.Errorf("available CPU core sample requires usage")
	}
	if sample.Availability == domain.AvailabilityUnavailable && sample.UsagePercent != nil {
		return fmt.Errorf("unavailable CPU core sample cannot contain usage")
	}
	if err := validateFiniteNonNegative("CPU core usage", sample.UsagePercent); err != nil {
		return err
	}
	if sample.UsagePercent != nil && *sample.UsagePercent > 100 {
		return fmt.Errorf("CPU core usage must not exceed 100")
	}
	return nil
}

const insertHostSampleSQL = "INSERT INTO host_samples (" +
	"collection_run_id, observed_at_unix, availability, unavailable_reason, " +
	"overall_cpu_percent, load_1, load_5, load_15, " +
	"memory_total_bytes, memory_used_bytes, memory_available_bytes, " +
	"memory_free_bytes, memory_cached_bytes, memory_buffered_bytes, " +
	"memory_usage_percent, swap_total_bytes, swap_used_bytes, " +
	"psi_availability, psi_unavailable_reason, " +
	"memory_psi_some_avg10, memory_psi_full_avg10, " +
	"memory_psi_some_total_us, memory_psi_full_total_us" +
	") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"

const insertCPUCoreSampleSQL = "INSERT INTO cpu_core_samples (" +
	"collection_run_id, logical_index, observed_at_unix, " +
	"availability, unavailable_reason, usage_percent" +
	") VALUES (?, ?, ?, ?, ?, ?)"

func insertCollectionSamples(ctx context.Context, transaction *sql.Tx, runID int64, record CollectionRecord) error {
	if record.Host != nil {
		sample := record.Host
		_, err := transaction.ExecContext(ctx, insertHostSampleSQL,
			runID, sample.ObservedAt.Unix(), sample.Availability, nullableReason(sample.UnavailableReason),
			sample.OverallCPUPercent, sample.Load1, sample.Load5, sample.Load15,
			sample.MemoryTotalBytes, sample.MemoryUsedBytes, sample.MemoryAvailableBytes,
			sample.MemoryFreeBytes, sample.MemoryCachedBytes, sample.MemoryBufferedBytes,
			sample.MemoryUsagePercent, sample.SwapTotalBytes, sample.SwapUsedBytes,
			sample.PSIAvailability, nullableReason(sample.PSIUnavailableReason),
			sample.MemoryPSISomeAverage10, sample.MemoryPSIFullAverage10,
			sample.MemoryPSISomeTotalUS, sample.MemoryPSIFullTotalUS,
		)
		if err != nil {
			return fmt.Errorf("insert host sample: %w", err)
		}
	}
	for _, sample := range record.CPUCores {
		if _, err := transaction.ExecContext(ctx, insertCPUCoreSampleSQL,
			runID, sample.LogicalIndex, sample.ObservedAt.Unix(), sample.Availability,
			nullableReason(sample.UnavailableReason), sample.UsagePercent,
		); err != nil {
			return fmt.Errorf("insert CPU core sample %d: %w", sample.LogicalIndex, err)
		}
	}
	return nil
}

func validateSampleTime(observedAt time.Time, run domain.CollectionRun) error {
	if err := domain.ValidateUTC(observedAt); err != nil {
		return fmt.Errorf("observed time: %w", err)
	}
	if observedAt.Before(run.StartedAt) || observedAt.After(run.FinishedAt) {
		return fmt.Errorf("observed time must be within collection run")
	}
	return nil
}

func validateAvailability(availability domain.Availability, reason *domain.UnavailabilityReason) error {
	if !availability.Valid() {
		return fmt.Errorf("invalid availability %q", availability)
	}
	if availability == domain.AvailabilityAvailable && reason != nil {
		return fmt.Errorf("available sample cannot contain an unavailable reason")
	}
	if availability == domain.AvailabilityUnavailable && (reason == nil || !reason.Valid()) {
		return fmt.Errorf("unavailable sample requires a valid reason")
	}
	return nil
}

func validateFiniteNonNegative(name string, value *float64) error {
	if value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0) || *value < 0) {
		return fmt.Errorf("%s must be finite and non-negative", name)
	}
	return nil
}

func nullableReason(reason *domain.UnavailabilityReason) any {
	if reason == nil {
		return nil
	}
	return string(*reason)
}

func anyNonNil(values ...any) bool {
	for _, value := range values {
		if value != nil && !reflect.ValueOf(value).IsNil() {
			return true
		}
	}
	return false
}
