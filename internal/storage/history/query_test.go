package history

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

func TestQueryMetricSeriesPreservesObservedUnavailableAndGapMinutes(t *testing.T) {
	t.Parallel()
	repository, database := openTestRepository(t)
	defer database.Close()
	start := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	insertHostMetric(t, repository, start, domain.AvailabilityAvailable, nil, floatPointer(10))
	insertHostMetric(t, repository, start.Add(time.Minute), domain.AvailabilityUnavailable, reasonPointer(domain.ReasonCollectorError), nil)
	insertHostMetric(t, repository, start.Add(3*time.Minute), domain.AvailabilityAvailable, nil, floatPointer(30))
	insertHostMetric(t, repository, start.Add(3*time.Minute+10*time.Second), domain.AvailabilityAvailable, nil, floatPointer(50))

	query := MetricQuery{
		Metric: MetricOverallCPUPercent,
		Range:  domain.ResolvedRange{Preset: domain.RangeLast5Minutes, Start: start, End: start.Add(5 * time.Minute)},
	}
	series, err := repository.QueryMetricSeries(context.Background(), query)
	if err != nil {
		t.Fatalf("QueryMetricSeries() error = %v", err)
	}
	if series.BucketDuration != time.Minute || len(series.Points) != 5 {
		t.Fatalf("series shape = duration %s, points %d", series.BucketDuration, len(series.Points))
	}
	states := []MetricPointState{
		MetricPointObserved, MetricPointUnavailable, MetricPointGap, MetricPointObserved, MetricPointGap,
	}
	for index, state := range states {
		if series.Points[index].State != state {
			t.Errorf("point[%d].State = %q, want %q", index, series.Points[index].State, state)
		}
	}
	point := series.Points[3]
	if point.ObservedSamples != 2 || point.AvailableSamples != 2 ||
		point.Minimum == nil || *point.Minimum != 30 ||
		point.Average == nil || *point.Average != 40 ||
		point.Maximum == nil || *point.Maximum != 50 {
		t.Fatalf("aggregated minute point = %+v", point)
	}
	raw, err := repository.QueryMetricSamples(context.Background(), query)
	if err != nil || len(raw) != 4 {
		t.Fatalf("QueryMetricSamples() rows = %d, error = %v", len(raw), err)
	}
}

func TestLongRangeAggregationKeepsMinimumAverageAndMaximum(t *testing.T) {
	t.Parallel()
	repository, database := openTestRepository(t)
	defer database.Close()
	start := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	insertHostMetric(t, repository, start, domain.AvailabilityAvailable, nil, floatPointer(5))
	insertHostMetric(t, repository, start.Add(time.Minute), domain.AvailabilityAvailable, nil, floatPointer(95))
	insertHostMetric(t, repository, start.Add(2*time.Minute), domain.AvailabilityUnavailable, reasonPointer(domain.ReasonCollectorError), nil)

	series, err := repository.QueryMetricSeries(context.Background(), MetricQuery{
		Metric: MetricOverallCPUPercent,
		Range:  domain.ResolvedRange{Preset: domain.RangeLast24Hours, Start: start, End: start.Add(24 * time.Hour)},
	})
	if err != nil {
		t.Fatalf("QueryMetricSeries() error = %v", err)
	}
	if series.BucketDuration != 3*time.Minute || len(series.Points) != 480 {
		t.Fatalf("series shape = duration %s, points %d", series.BucketDuration, len(series.Points))
	}
	first := series.Points[0]
	if first.State != MetricPointObserved || first.ObservedSamples != 3 || first.AvailableSamples != 2 ||
		first.Minimum == nil || *first.Minimum != 5 ||
		first.Average == nil || *first.Average != 50 ||
		first.Maximum == nil || *first.Maximum != 95 {
		t.Fatalf("first aggregate = %+v", first)
	}
}

func TestEmptyMetricRangeReturnsOnlyHonestGaps(t *testing.T) {
	t.Parallel()
	repository, database := openTestRepository(t)
	defer database.Close()
	start := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	series, err := repository.QueryMetricSeries(context.Background(), MetricQuery{
		Metric: MetricMemoryUsagePercent,
		Range: domain.ResolvedRange{
			Preset: domain.RangeLast5Minutes, Start: start, End: start.Add(5 * time.Minute),
		},
	})
	if err != nil {
		t.Fatalf("QueryMetricSeries() error = %v", err)
	}
	if len(series.Points) != 5 {
		t.Fatalf("point count = %d, want 5", len(series.Points))
	}
	for index, point := range series.Points {
		if point.State != MetricPointGap || point.ObservedSamples != 0 ||
			point.Minimum != nil || point.Average != nil || point.Maximum != nil {
			t.Fatalf("point[%d] fabricated evidence: %+v", index, point)
		}
	}
}

func TestConcurrentMetricReadsAndCollectionWrites(t *testing.T) {
	repository, database := openTestRepository(t)
	defer database.Close()
	start := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	query := MetricQuery{
		Metric: MetricOverallCPUPercent,
		Range: domain.ResolvedRange{
			Preset: domain.RangeLastHour, Start: start, End: start.Add(time.Hour),
		},
	}

	const writes = 24
	errorsChannel := make(chan error, writes*2)
	var waitGroup sync.WaitGroup
	for index := range writes {
		waitGroup.Add(2)
		go func() {
			defer waitGroup.Done()
			value := float64(index)
			notCollected := domain.ReasonNotCollected
			_, err := repository.RecordCollection(context.Background(), CollectionRecord{
				Run: validCollectionRun(start.Add(time.Duration(index) * time.Minute)),
				Host: &HostSampleRecord{
					ObservedAt:   start.Add(time.Duration(index) * time.Minute),
					Availability: domain.AvailabilityAvailable, OverallCPUPercent: &value,
					PSIAvailability: domain.AvailabilityUnavailable, PSIUnavailableReason: &notCollected,
				},
			})
			errorsChannel <- err
		}()
		go func() {
			defer waitGroup.Done()
			_, err := repository.QueryMetricSeries(context.Background(), query)
			errorsChannel <- err
		}()
	}
	waitGroup.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		if err != nil {
			t.Errorf("concurrent history operation error = %v", err)
		}
	}
}

func TestMetricQueryUsesAllowlistedIdentifiersAndResourceParameters(t *testing.T) {
	t.Parallel()
	repository, database := openTestRepository(t)
	defer database.Close()
	start := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	run := validCollectionRun(start)
	usageZero, usageOne := 10.0, 90.0
	_, err := repository.RecordCollection(context.Background(), CollectionRecord{
		Run: run,
		CPUCores: []CPUCoreSampleRecord{
			{LogicalIndex: 0, ObservedAt: start, Availability: domain.AvailabilityAvailable, UsagePercent: &usageZero},
			{LogicalIndex: 1, ObservedAt: start, Availability: domain.AvailabilityAvailable, UsagePercent: &usageOne},
		},
	})
	if err != nil {
		t.Fatalf("RecordCollection() error = %v", err)
	}
	rangeValue := domain.ResolvedRange{Preset: domain.RangeLastMinute, Start: start, End: start.Add(time.Minute)}
	rows, err := repository.QueryMetricSamples(context.Background(), MetricQuery{
		Metric: MetricCPUCoreUsagePercent, ResourceID: "0", Range: rangeValue,
	})
	if err != nil || len(rows) != 1 || rows[0].Value == nil || *rows[0].Value != 10 {
		t.Fatalf("core query rows = %+v, error = %v", rows, err)
	}
	if _, err := repository.QueryMetricSamples(context.Background(), MetricQuery{
		Metric: MetricCPUCoreUsagePercent, ResourceID: "0 OR 1=1", Range: rangeValue,
	}); err == nil {
		t.Fatal("unsafe logical CPU resource was accepted")
	}
	if _, err := repository.QueryMetricSamples(context.Background(), MetricQuery{
		Metric: MetricKey("host_samples; DROP TABLE collection_runs"), Range: rangeValue,
	}); err == nil {
		t.Fatal("unallowlisted metric was accepted")
	}

	planRows, err := database.SQL().Query(`
		EXPLAIN QUERY PLAN
		SELECT observed_at_unix, overall_cpu_percent
		FROM host_samples
		WHERE observed_at_unix >= ? AND observed_at_unix < ?
		ORDER BY observed_at_unix
	`, start.Unix(), start.Add(time.Hour).Unix())
	if err != nil {
		t.Fatalf("EXPLAIN QUERY PLAN error = %v", err)
	}
	defer planRows.Close()
	var plan strings.Builder
	for planRows.Next() {
		var id, parent, notUsed int
		var detail string
		if err := planRows.Scan(&id, &parent, &notUsed, &detail); err != nil {
			t.Fatalf("scan query plan: %v", err)
		}
		plan.WriteString(detail)
	}
	if !strings.Contains(plan.String(), "host_samples_observed_at_idx") {
		t.Fatalf("history query did not use time index: %s", plan.String())
	}
}

func insertHostMetric(
	t *testing.T,
	repository *Repository,
	observedAt time.Time,
	availability domain.Availability,
	reason *domain.UnavailabilityReason,
	value *float64,
) {
	t.Helper()
	notCollected := domain.ReasonNotCollected
	_, err := repository.RecordCollection(context.Background(), CollectionRecord{
		Run: validCollectionRun(observedAt),
		Host: &HostSampleRecord{
			ObservedAt: observedAt, Availability: availability, UnavailableReason: reason,
			OverallCPUPercent: value,
			PSIAvailability:   domain.AvailabilityUnavailable, PSIUnavailableReason: &notCollected,
		},
	})
	if err != nil {
		t.Fatalf("RecordCollection() error = %v", err)
	}
}

type sqlExecutor interface {
	Exec(string, ...any) (sql.Result, error)
}

func floatPointer(value float64) *float64 { return &value }

func reasonPointer(value domain.UnavailabilityReason) *domain.UnavailabilityReason { return &value }
