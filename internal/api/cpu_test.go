package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/api/contracts"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

func TestCPUCurrentAndHistoryEndpoints(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	source := &cpuSourceStub{current: testCPUSnapshot(now), history: testCPUHistory(now)}
	handler := newTestHandler(t, HandlerOptions{CPU: source})

	currentResponse := httptest.NewRecorder()
	handler.ServeHTTP(currentResponse, httptest.NewRequest(http.MethodGet, "/api/v1/cpu", nil))
	if currentResponse.Code != http.StatusOK {
		t.Fatalf("current status = %d, body = %s", currentResponse.Code, currentResponse.Body.String())
	}
	var current contracts.Envelope[contracts.CPUSnapshot]
	if err := json.Unmarshal(currentResponse.Body.Bytes(), &current); err != nil || current.Data == nil || current.Data.LogicalCPU != 2 {
		t.Fatalf("current response = %+v, error = %v", current, err)
	}

	historyResponse := httptest.NewRecorder()
	handler.ServeHTTP(historyResponse, httptest.NewRequest(http.MethodGet, "/api/v1/cpu/history?metric=core&core=1&range=last_1h", nil))
	if historyResponse.Code != http.StatusOK {
		t.Fatalf("history status = %d, body = %s", historyResponse.Code, historyResponse.Body.String())
	}
	if source.request.Metric != contracts.CPUMetricCore || source.request.CoreIndex == nil || *source.request.CoreIndex != 1 {
		t.Fatalf("parsed history request = %+v", source.request)
	}
}

func TestCPUHistoryEndpointRejectsUnsafeOrAmbiguousQueries(t *testing.T) {
	t.Parallel()
	handler := newTestHandler(t, HandlerOptions{CPU: &cpuSourceStub{}})
	paths := []string{
		"/api/v1/cpu/history",
		"/api/v1/cpu/history?metric=overall&range=last_1h&core=0",
		"/api/v1/cpu/history?metric=core&range=last_1h&core=0%20OR%201=1",
		"/api/v1/cpu/history?metric=overall&range=last_1h&range=last_14d",
		"/api/v1/cpu/history?metric=overall&range=last_1h&unknown=value",
		"/api/v1/cpu/history?metric=overall&range=custom&start=2026-08-05T10:00:00%2B01:00&end=2026-08-05T11:00:00Z",
	}
	for _, path := range paths {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusBadRequest {
			t.Errorf("GET %s status = %d, body = %s", path, response.Code, response.Body.String())
		}
	}
}

func TestCPUHistoryEndpointReturnsSafeUnavailableError(t *testing.T) {
	t.Parallel()
	source := &cpuSourceStub{err: errors.New("synthetic database detail")}
	handler := newTestHandler(t, HandlerOptions{CPU: source})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/cpu/history?metric=overall&range=last_1h", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if body := response.Body.String(); contains(body, "synthetic database detail") {
		t.Fatalf("internal detail leaked: %s", body)
	}
}

func TestCPUHistoryEndpointClassifiesResolvedRangeErrorsAsValidation(t *testing.T) {
	t.Parallel()
	source := &cpuSourceStub{err: contracts.ErrInvalidCPUHistoryRange}
	handler := newTestHandler(t, HandlerOptions{CPU: source})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/cpu/history?metric=overall&range=last_1h", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

type cpuSourceStub struct {
	current contracts.CPUSnapshot
	history contracts.CPUHistorySeries
	request contracts.CPUHistoryRequest
	err     error
}

func (source *cpuSourceStub) CurrentCPU() contracts.CPUSnapshot { return source.current }

func (source *cpuSourceStub) CPUHistory(_ context.Context, request contracts.CPUHistoryRequest) (contracts.CPUHistorySeries, error) {
	source.request = request
	return source.history, source.err
}

func testCPUSnapshot(now time.Time) contracts.CPUSnapshot {
	overall, coreZero, coreOne, load := 35.0, 20.0, 50.0, 0.4
	return contracts.CPUSnapshot{
		Resource:  domain.ResourceRef{Kind: domain.ResourceCPU, ID: "host-cpu", DisplayName: "CPU"},
		Freshness: domain.Freshness{State: domain.FreshnessFresh, ObservedAt: &now, LastSuccessfulAt: &now},
		Overall:   availableCPUMetric(overall, domain.UnitPercent),
		Cores: []contracts.CPUCore{
			{Index: 0, Usage: availableCPUMetric(coreZero, domain.UnitPercent)},
			{Index: 1, Usage: availableCPUMetric(coreOne, domain.UnitPercent)},
		},
		Load: contracts.LoadAverages{
			OneMinute: availableCPUMetric(load, domain.UnitLoad), FiveMinutes: availableCPUMetric(load, domain.UnitLoad),
			FifteenMinute: availableCPUMetric(load, domain.UnitLoad),
		},
		LogicalCPU: 2,
	}
}

func testCPUHistory(now time.Time) contracts.CPUHistorySeries {
	value := 50.0
	core := 1
	return contracts.CPUHistorySeries{
		Resource: domain.ResourceRef{Kind: domain.ResourceCPU, ID: "cpu-core-1", DisplayName: "CPU 1"},
		Metric:   contracts.CPUMetricCore, CoreIndex: &core, Unit: domain.UnitPercent,
		Range:                 domain.ResolvedRange{Preset: domain.RangeLastHour, Start: now.Add(-time.Hour), End: now},
		BucketDurationSeconds: 60,
		Points: []contracts.CPUHistoryPoint{{
			Start: now.Add(-time.Minute), End: now, State: contracts.CPUHistoryObserved,
			ObservedSamples: 1, AvailableSamples: 1, Minimum: &value, Average: &value, Maximum: &value,
		}},
	}
}

func availableCPUMetric(value float64, unit domain.Unit) domain.Metric[float64] {
	return domain.Metric[float64]{Availability: domain.AvailabilityAvailable, Value: &value, Unit: unit}
}

func contains(value, substring string) bool {
	for index := 0; index+len(substring) <= len(value); index++ {
		if value[index:index+len(substring)] == substring {
			return true
		}
	}
	return false
}
