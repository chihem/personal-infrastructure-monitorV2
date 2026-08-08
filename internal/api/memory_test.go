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

func TestMemoryCurrentAndHistoryEndpoints(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	source := &memorySourceStub{current: testMemorySnapshot(now), history: testMemoryHistory(now)}
	handler := newTestHandler(t, HandlerOptions{Memory: source})

	currentResponse := httptest.NewRecorder()
	handler.ServeHTTP(currentResponse, httptest.NewRequest(http.MethodGet, "/api/v1/memory", nil))
	if currentResponse.Code != http.StatusOK {
		t.Fatalf("current status = %d, body = %s", currentResponse.Code, currentResponse.Body.String())
	}
	var current contracts.Envelope[contracts.MemorySnapshot]
	if err := json.Unmarshal(currentResponse.Body.Bytes(), &current); err != nil || current.Data == nil || current.Data.Used.Value == nil {
		t.Fatalf("current response = %+v, error = %v", current, err)
	}

	historyResponse := httptest.NewRecorder()
	handler.ServeHTTP(historyResponse, httptest.NewRequest(http.MethodGet, "/api/v1/memory/history?metric=pressure_full_avg10&range=last_1h", nil))
	if historyResponse.Code != http.StatusOK {
		t.Fatalf("history status = %d, body = %s", historyResponse.Code, historyResponse.Body.String())
	}
	if source.request.Metric != contracts.MemoryMetricPSIFullAverage10 {
		t.Fatalf("parsed history request = %+v", source.request)
	}
}

func TestMemoryHistoryEndpointRejectsUnsafeOrAmbiguousQueries(t *testing.T) {
	t.Parallel()
	handler := newTestHandler(t, HandlerOptions{Memory: &memorySourceStub{}})
	paths := []string{
		"/api/v1/memory/history",
		"/api/v1/memory/history?metric=unknown&range=last_1h",
		"/api/v1/memory/history?metric=usage&range=last_1h&range=last_14d",
		"/api/v1/memory/history?metric=usage&range=last_1h&unknown=value",
		"/api/v1/memory/history?metric=usage&range=custom&start=2026-08-05T10:00:00%2B01:00&end=2026-08-05T11:00:00Z",
	}
	for _, path := range paths {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusBadRequest {
			t.Errorf("GET %s status = %d, body = %s", path, response.Code, response.Body.String())
		}
	}
}

func TestMemoryHistoryEndpointDoesNotLeakRepositoryErrors(t *testing.T) {
	t.Parallel()
	source := &memorySourceStub{err: errors.New("synthetic database path and SQL detail")}
	handler := newTestHandler(t, HandlerOptions{Memory: source})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/memory/history?metric=usage&range=last_1h", nil))
	if response.Code != http.StatusServiceUnavailable || contains(response.Body.String(), "synthetic") {
		t.Fatalf("response status = %d, body = %s", response.Code, response.Body.String())
	}
}

type memorySourceStub struct {
	current contracts.MemorySnapshot
	history contracts.MemoryHistorySeries
	request contracts.MemoryHistoryRequest
	err     error
}

func (source *memorySourceStub) CurrentMemory() contracts.MemorySnapshot { return source.current }

func (source *memorySourceStub) MemoryHistory(_ context.Context, request contracts.MemoryHistoryRequest) (contracts.MemoryHistorySeries, error) {
	source.request = request
	return source.history, source.err
}

func testMemorySnapshot(now time.Time) contracts.MemorySnapshot {
	total, used, available, free, cached, buffered := int64(11<<30), int64(3<<30), int64(8<<30), int64(2<<30), int64(5<<30), int64(256<<20)
	usage, pressure := 27.27, 0.1
	notConfigured := domain.ReasonNotConfigured
	swapConfigured := false
	window := contracts.PressureWindow{
		Average10Seconds:  availableCPUMetric(pressure, domain.UnitPercent),
		Average60Seconds:  availableCPUMetric(pressure, domain.UnitPercent),
		Average300Seconds: availableCPUMetric(pressure, domain.UnitPercent),
		Total:             availableAPIIntegerMetric(100, domain.UnitMicroseconds),
	}
	return contracts.MemorySnapshot{
		Resource:  domain.ResourceRef{Kind: domain.ResourceMemory, ID: "host-memory", DisplayName: "Memory"},
		Freshness: domain.Freshness{State: domain.FreshnessFresh, ObservedAt: &now, LastSuccessfulAt: &now},
		Total:     availableAPIIntegerMetric(total, domain.UnitBytes), Used: availableAPIIntegerMetric(used, domain.UnitBytes),
		Available: availableAPIIntegerMetric(available, domain.UnitBytes), Free: availableAPIIntegerMetric(free, domain.UnitBytes),
		Cached: availableAPIIntegerMetric(cached, domain.UnitBytes), Buffered: availableAPIIntegerMetric(buffered, domain.UnitBytes),
		Usage: availableCPUMetric(usage, domain.UnitPercent),
		Swap: contracts.SwapSnapshot{
			Configured: &swapConfigured,
			Total:      unavailableAPIIntegerMetric(domain.UnitBytes, notConfigured), Used: unavailableAPIIntegerMetric(domain.UnitBytes, notConfigured),
			Free: unavailableAPIIntegerMetric(domain.UnitBytes, notConfigured),
		},
		Pressure: contracts.MemoryPressure{Some: window, Full: window},
	}
}

func testMemoryHistory(now time.Time) contracts.MemoryHistorySeries {
	value := 0.1
	return contracts.MemoryHistorySeries{
		Resource: domain.ResourceRef{Kind: domain.ResourceMemory, ID: "host-memory", DisplayName: "Memory"},
		Metric:   contracts.MemoryMetricPSIFullAverage10, Unit: domain.UnitPercent,
		Range:                 domain.ResolvedRange{Preset: domain.RangeLastHour, Start: now.Add(-time.Hour), End: now},
		BucketDurationSeconds: 60,
		Points: []contracts.MemoryHistoryPoint{{
			Start: now.Add(-time.Minute), End: now, State: contracts.MemoryHistoryObserved,
			ObservedSamples: 1, AvailableSamples: 1, Minimum: &value, Average: &value, Maximum: &value,
		}},
	}
}

func availableAPIIntegerMetric(value int64, unit domain.Unit) domain.Metric[int64] {
	return domain.Metric[int64]{Availability: domain.AvailabilityAvailable, Value: &value, Unit: unit}
}

func unavailableAPIIntegerMetric(unit domain.Unit, reason domain.UnavailabilityReason) domain.Metric[int64] {
	return domain.Metric[int64]{Availability: domain.AvailabilityUnavailable, Unit: unit, ReasonCode: &reason}
}
