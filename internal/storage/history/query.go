package history

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

const MaxMetricSamplesPerQuery = 100_000

var ErrQueryLimitExceeded = errors.New("history query exceeded the safe row limit")

type MetricKey string

const (
	MetricOverallCPUPercent      MetricKey = "host.cpu.overall_percent"
	MetricLoad1                  MetricKey = "host.load.1"
	MetricLoad5                  MetricKey = "host.load.5"
	MetricLoad15                 MetricKey = "host.load.15"
	MetricMemoryTotalBytes       MetricKey = "host.memory.total_bytes"
	MetricMemoryUsedBytes        MetricKey = "host.memory.used_bytes"
	MetricMemoryAvailableBytes   MetricKey = "host.memory.available_bytes"
	MetricMemoryFreeBytes        MetricKey = "host.memory.free_bytes"
	MetricMemoryCachedBytes      MetricKey = "host.memory.cached_bytes"
	MetricMemoryBufferedBytes    MetricKey = "host.memory.buffered_bytes"
	MetricMemoryUsagePercent     MetricKey = "host.memory.usage_percent"
	MetricSwapTotalBytes         MetricKey = "host.swap.total_bytes"
	MetricSwapUsedBytes          MetricKey = "host.swap.used_bytes"
	MetricMemoryPSISomeAverage10 MetricKey = "host.memory.psi.some_avg10"
	MetricMemoryPSIFullAverage10 MetricKey = "host.memory.psi.full_avg10"
	MetricMemoryPSISomeTotalUS   MetricKey = "host.memory.psi.some_total_us"
	MetricMemoryPSIFullTotalUS   MetricKey = "host.memory.psi.full_total_us"
	MetricCPUCoreUsagePercent    MetricKey = "cpu.core.usage_percent"
	MetricFilesystemTotalBytes   MetricKey = "filesystem.total_bytes"
	MetricFilesystemUsedBytes    MetricKey = "filesystem.used_bytes"
	MetricFilesystemFreeBytes    MetricKey = "filesystem.free_bytes"
	MetricFilesystemUsagePercent MetricKey = "filesystem.usage_percent"
	MetricBlockReadBytesTotal    MetricKey = "block.read_bytes_total"
	MetricBlockWriteBytesTotal   MetricKey = "block.write_bytes_total"
	MetricBlockReadBytesPerSec   MetricKey = "block.read_bytes_per_second"
	MetricBlockWriteBytesPerSec  MetricKey = "block.write_bytes_per_second"
	MetricContainerCPUPercent    MetricKey = "container.cpu_percent"
	MetricContainerMemoryBytes   MetricKey = "container.memory_used_bytes"
	MetricContainerMemoryLimit   MetricKey = "container.memory_limit_bytes"
	MetricContainerMemoryPercent MetricKey = "container.memory_percent"
	MetricContainerUptimeSeconds MetricKey = "container.uptime_seconds"
	MetricContainerRestartCount  MetricKey = "container.restart_count"
)

type resourceMode uint8

const (
	resourceNone resourceMode = iota
	resourceOpaque
	resourceInteger
)

type metricSpec struct {
	table              string
	column             string
	availabilityColumn string
	reasonColumn       string
	resourceColumn     string
	resourceMode       resourceMode
	unit               domain.Unit
}

var metricSpecs = map[MetricKey]metricSpec{
	MetricOverallCPUPercent:      hostMetric("overall_cpu_percent", domain.UnitPercent),
	MetricLoad1:                  hostMetric("load_1", domain.UnitLoad),
	MetricLoad5:                  hostMetric("load_5", domain.UnitLoad),
	MetricLoad15:                 hostMetric("load_15", domain.UnitLoad),
	MetricMemoryTotalBytes:       hostMetric("memory_total_bytes", domain.UnitBytes),
	MetricMemoryUsedBytes:        hostMetric("memory_used_bytes", domain.UnitBytes),
	MetricMemoryAvailableBytes:   hostMetric("memory_available_bytes", domain.UnitBytes),
	MetricMemoryFreeBytes:        hostMetric("memory_free_bytes", domain.UnitBytes),
	MetricMemoryCachedBytes:      hostMetric("memory_cached_bytes", domain.UnitBytes),
	MetricMemoryBufferedBytes:    hostMetric("memory_buffered_bytes", domain.UnitBytes),
	MetricMemoryUsagePercent:     hostMetric("memory_usage_percent", domain.UnitPercent),
	MetricSwapTotalBytes:         hostMetric("swap_total_bytes", domain.UnitBytes),
	MetricSwapUsedBytes:          hostMetric("swap_used_bytes", domain.UnitBytes),
	MetricMemoryPSISomeAverage10: psiMetric("memory_psi_some_avg10", domain.UnitPercent),
	MetricMemoryPSIFullAverage10: psiMetric("memory_psi_full_avg10", domain.UnitPercent),
	MetricMemoryPSISomeTotalUS:   psiMetric("memory_psi_some_total_us", domain.UnitMicroseconds),
	MetricMemoryPSIFullTotalUS:   psiMetric("memory_psi_full_total_us", domain.UnitMicroseconds),
	MetricCPUCoreUsagePercent: {
		table: "cpu_core_samples", column: "usage_percent", availabilityColumn: "availability",
		reasonColumn: "unavailable_reason", resourceColumn: "logical_index", resourceMode: resourceInteger,
		unit: domain.UnitPercent,
	},
	MetricFilesystemTotalBytes:   resourceMetric("filesystem_samples", "total_bytes", "filesystem_id", domain.UnitBytes),
	MetricFilesystemUsedBytes:    resourceMetric("filesystem_samples", "used_bytes", "filesystem_id", domain.UnitBytes),
	MetricFilesystemFreeBytes:    resourceMetric("filesystem_samples", "free_bytes", "filesystem_id", domain.UnitBytes),
	MetricFilesystemUsagePercent: resourceMetric("filesystem_samples", "usage_percent", "filesystem_id", domain.UnitPercent),
	MetricBlockReadBytesTotal:    resourceMetric("block_device_io_samples", "read_bytes_total", "block_device_id", domain.UnitBytes),
	MetricBlockWriteBytesTotal:   resourceMetric("block_device_io_samples", "write_bytes_total", "block_device_id", domain.UnitBytes),
	MetricBlockReadBytesPerSec:   resourceMetric("block_device_io_samples", "read_bytes_per_second", "block_device_id", domain.UnitBytesPerSecond),
	MetricBlockWriteBytesPerSec:  resourceMetric("block_device_io_samples", "write_bytes_per_second", "block_device_id", domain.UnitBytesPerSecond),
	MetricContainerCPUPercent:    resourceMetric("container_samples", "cpu_percent", "container_id", domain.UnitPercent),
	MetricContainerMemoryBytes:   resourceMetric("container_samples", "memory_used_bytes", "container_id", domain.UnitBytes),
	MetricContainerMemoryLimit:   resourceMetric("container_samples", "memory_limit_bytes", "container_id", domain.UnitBytes),
	MetricContainerMemoryPercent: resourceMetric("container_samples", "memory_percent", "container_id", domain.UnitPercent),
	MetricContainerUptimeSeconds: resourceMetric("container_samples", "uptime_seconds", "container_id", domain.UnitSeconds),
	MetricContainerRestartCount:  resourceMetric("container_samples", "restart_count", "container_id", domain.UnitCount),
}

type MetricQuery struct {
	Metric     MetricKey
	ResourceID string
	Range      domain.ResolvedRange
}

type MetricSample struct {
	ObservedAt        time.Time
	Availability      domain.Availability
	UnavailableReason *domain.UnavailabilityReason
	Value             *float64
}

type MetricPointState string

const (
	MetricPointObserved    MetricPointState = "observed"
	MetricPointUnavailable MetricPointState = "unavailable"
	MetricPointGap         MetricPointState = "gap"
)

type MetricPoint struct {
	Start            time.Time
	End              time.Time
	State            MetricPointState
	ObservedSamples  int
	AvailableSamples int
	Minimum          *float64
	Average          *float64
	Maximum          *float64
}

type MetricSeries struct {
	Metric         MetricKey
	ResourceID     string
	Unit           domain.Unit
	Range          domain.ResolvedRange
	BucketDuration time.Duration
	Points         []MetricPoint
}

func (repository *Repository) QueryMetricSamples(ctx context.Context, query MetricQuery) ([]MetricSample, error) {
	spec, resourceValue, err := validateMetricQuery(query)
	if err != nil {
		return nil, err
	}
	if repository == nil || repository.database == nil {
		return nil, errors.New("history repository is unavailable")
	}

	statement := "SELECT observed_at_unix, " + spec.availabilityColumn + ", " + spec.reasonColumn +
		", CAST(" + spec.column + " AS REAL) FROM " + spec.table +
		" WHERE observed_at_unix >= ? AND observed_at_unix < ?"
	arguments := []any{query.Range.Start.Unix(), query.Range.End.Unix()}
	if spec.resourceMode != resourceNone {
		statement += " AND " + spec.resourceColumn + " = ?"
		arguments = append(arguments, resourceValue)
	}
	statement += " ORDER BY observed_at_unix ASC LIMIT ?"
	arguments = append(arguments, MaxMetricSamplesPerQuery+1)

	rows, err := repository.database.QueryContext(ctx, statement, arguments...)
	if err != nil {
		return nil, fmt.Errorf("query metric samples %q: %w", query.Metric, err)
	}
	defer rows.Close()

	samples := make([]MetricSample, 0)
	for rows.Next() {
		if len(samples) == MaxMetricSamplesPerQuery {
			return nil, ErrQueryLimitExceeded
		}
		var observedUnix int64
		var availability domain.Availability
		var reason sql.NullString
		var value sql.NullFloat64
		if err := rows.Scan(&observedUnix, &availability, &reason, &value); err != nil {
			return nil, fmt.Errorf("scan metric sample %q: %w", query.Metric, err)
		}
		if !availability.Valid() {
			return nil, fmt.Errorf("metric sample %q contains invalid availability %q", query.Metric, availability)
		}
		sample := MetricSample{ObservedAt: time.Unix(observedUnix, 0).UTC(), Availability: availability}
		if reason.Valid {
			reasonCode := domain.UnavailabilityReason(reason.String)
			if !reasonCode.Valid() {
				return nil, fmt.Errorf("metric sample %q contains invalid unavailable reason", query.Metric)
			}
			sample.UnavailableReason = &reasonCode
		}
		if availability == domain.AvailabilityUnavailable && sample.UnavailableReason == nil {
			return nil, fmt.Errorf("metric sample %q is unavailable without a reason", query.Metric)
		}
		if availability == domain.AvailabilityAvailable && sample.UnavailableReason != nil {
			return nil, fmt.Errorf("metric sample %q is available with an unavailable reason", query.Metric)
		}
		if value.Valid && availability == domain.AvailabilityAvailable {
			if math.IsNaN(value.Float64) || math.IsInf(value.Float64, 0) {
				return nil, fmt.Errorf("metric sample %q contains a non-finite value", query.Metric)
			}
			sample.Value = &value.Float64
		}
		samples = append(samples, sample)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate metric samples %q: %w", query.Metric, err)
	}
	return samples, nil
}

func (repository *Repository) QueryMetricSeries(ctx context.Context, query MetricQuery) (MetricSeries, error) {
	spec, _, err := validateMetricQuery(query)
	if err != nil {
		return MetricSeries{}, err
	}
	samples, err := repository.QueryMetricSamples(ctx, query)
	if err != nil {
		return MetricSeries{}, err
	}

	interval := bucketDuration(query.Range)
	duration := query.Range.End.Sub(query.Range.Start)
	pointCount := int((duration + interval - 1) / interval)
	if pointCount > MaxChartPoints && interval > time.Minute {
		return MetricSeries{}, errors.New("aggregated history query exceeded chart point limit")
	}
	points := make([]MetricPoint, pointCount)
	sums := make([]float64, pointCount)
	for index := range points {
		start := query.Range.Start.Add(time.Duration(index) * interval)
		end := start.Add(interval)
		if end.After(query.Range.End) {
			end = query.Range.End
		}
		points[index] = MetricPoint{Start: start, End: end, State: MetricPointGap}
	}

	for _, sample := range samples {
		index := int(sample.ObservedAt.Sub(query.Range.Start) / interval)
		if index < 0 || index >= len(points) {
			continue
		}
		point := &points[index]
		point.ObservedSamples++
		if sample.Value == nil {
			continue
		}
		value := *sample.Value
		if point.AvailableSamples == 0 {
			minimum, maximum := value, value
			point.Minimum = &minimum
			point.Maximum = &maximum
		} else {
			if value < *point.Minimum {
				*point.Minimum = value
			}
			if value > *point.Maximum {
				*point.Maximum = value
			}
		}
		point.AvailableSamples++
		sums[index] += value
	}

	for index := range points {
		point := &points[index]
		switch {
		case point.ObservedSamples == 0:
			point.State = MetricPointGap
		case point.AvailableSamples == 0:
			point.State = MetricPointUnavailable
		default:
			point.State = MetricPointObserved
			average := sums[index] / float64(point.AvailableSamples)
			point.Average = &average
		}
	}

	return MetricSeries{
		Metric: query.Metric, ResourceID: query.ResourceID, Unit: spec.unit,
		Range: query.Range, BucketDuration: interval, Points: points,
	}, nil
}

func validateMetricQuery(query MetricQuery) (metricSpec, any, error) {
	spec, ok := metricSpecs[query.Metric]
	if !ok {
		return metricSpec{}, nil, fmt.Errorf("unsupported history metric %q", query.Metric)
	}
	if err := validateResolvedRange(query.Range); err != nil {
		return metricSpec{}, nil, fmt.Errorf("range: %w", err)
	}
	switch spec.resourceMode {
	case resourceNone:
		if query.ResourceID != "" {
			return metricSpec{}, nil, errors.New("host metric cannot contain a resource ID")
		}
		return spec, nil, nil
	case resourceOpaque:
		if err := domain.ValidateOpaqueID(query.ResourceID); err != nil {
			return metricSpec{}, nil, fmt.Errorf("resource ID: %w", err)
		}
		return spec, query.ResourceID, nil
	case resourceInteger:
		if strings.TrimSpace(query.ResourceID) != query.ResourceID {
			return metricSpec{}, nil, errors.New("logical CPU index must not contain whitespace")
		}
		index, err := strconv.ParseInt(query.ResourceID, 10, 32)
		if err != nil || index < 0 {
			return metricSpec{}, nil, errors.New("logical CPU index must be a non-negative integer")
		}
		return spec, index, nil
	default:
		return metricSpec{}, nil, errors.New("history metric has an invalid resource mode")
	}
}

func hostMetric(column string, unit domain.Unit) metricSpec {
	return metricSpec{
		table: "host_samples", column: column, availabilityColumn: "availability",
		reasonColumn: "unavailable_reason", resourceMode: resourceNone, unit: unit,
	}
}

func psiMetric(column string, unit domain.Unit) metricSpec {
	return metricSpec{
		table: "host_samples", column: column, availabilityColumn: "psi_availability",
		reasonColumn: "psi_unavailable_reason", resourceMode: resourceNone, unit: unit,
	}
}

func resourceMetric(table, column, resourceColumn string, unit domain.Unit) metricSpec {
	return metricSpec{
		table: table, column: column, availabilityColumn: "availability",
		reasonColumn: "unavailable_reason", resourceColumn: resourceColumn,
		resourceMode: resourceOpaque, unit: unit,
	}
}
