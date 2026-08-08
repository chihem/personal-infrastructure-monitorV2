package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/api/contracts"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

type MemoryDataSource interface {
	CurrentMemory() contracts.MemorySnapshot
	MemoryHistory(context.Context, contracts.MemoryHistoryRequest) (contracts.MemoryHistorySeries, error)
}

func registerMemoryRoutes(mux *http.ServeMux, source MemoryDataSource, now func() time.Time) {
	mux.HandleFunc("/api/v1/memory", func(writer http.ResponseWriter, request *http.Request) {
		if !allowReadMethod(writer, request, now) {
			return
		}
		if source == nil {
			writeAPIError(writer, request, http.StatusServiceUnavailable, now, domain.ErrorUnavailable, "errors.memoryUnavailable", "Memory information is unavailable.")
			return
		}
		snapshot := source.CurrentMemory()
		if err := snapshot.Validate(); err != nil {
			writeInternalError(writer, request, now)
			return
		}
		writeData(writer, request, http.StatusOK, now, snapshot)
	})

	mux.HandleFunc("/api/v1/memory/history", func(writer http.ResponseWriter, request *http.Request) {
		if !allowReadMethod(writer, request, now) {
			return
		}
		query, err := parseMemoryHistoryRequest(request)
		if err != nil {
			writeAPIError(writer, request, http.StatusBadRequest, now, domain.ErrorValidationFailed, "errors.invalidMemoryHistoryQuery", "The memory history query is invalid.")
			return
		}
		if source == nil {
			writeAPIError(writer, request, http.StatusServiceUnavailable, now, domain.ErrorHistoryUnavailable, "errors.historyUnavailable", "Memory history is unavailable.")
			return
		}
		series, err := source.MemoryHistory(request.Context(), query)
		if err != nil {
			if errors.Is(err, contracts.ErrInvalidMemoryHistoryRange) {
				writeAPIError(writer, request, http.StatusBadRequest, now, domain.ErrorValidationFailed, "errors.invalidMemoryHistoryQuery", "The memory history query is invalid.")
				return
			}
			writeAPIError(writer, request, http.StatusServiceUnavailable, now, domain.ErrorHistoryUnavailable, "errors.historyUnavailable", "Memory history is unavailable.")
			return
		}
		if err := series.Validate(); err != nil {
			writeInternalError(writer, request, now)
			return
		}
		writeData(writer, request, http.StatusOK, now, series)
	})
}

func parseMemoryHistoryRequest(request *http.Request) (contracts.MemoryHistoryRequest, error) {
	values := request.URL.Query()
	allowed := map[string]struct{}{"metric": {}, "range": {}, "start": {}, "end": {}}
	for key, list := range values {
		if _, ok := allowed[key]; !ok {
			return contracts.MemoryHistoryRequest{}, fmt.Errorf("unsupported query parameter")
		}
		if len(list) != 1 {
			return contracts.MemoryHistoryRequest{}, fmt.Errorf("query parameters cannot be repeated")
		}
	}
	metricValues, metricPresent := values["metric"]
	rangeValues, rangePresent := values["range"]
	if !metricPresent || !rangePresent || metricValues[0] == "" || rangeValues[0] == "" {
		return contracts.MemoryHistoryRequest{}, fmt.Errorf("metric and range are required")
	}
	parsed := contracts.MemoryHistoryRequest{
		Metric: contracts.MemoryMetric(metricValues[0]),
		Range:  domain.RangeSelection{Preset: domain.RangePreset(rangeValues[0])},
	}
	if parsed.Range.Preset == domain.RangeCustom {
		start, err := parseUTCQueryTime(values, "start")
		if err != nil {
			return contracts.MemoryHistoryRequest{}, err
		}
		end, err := parseUTCQueryTime(values, "end")
		if err != nil {
			return contracts.MemoryHistoryRequest{}, err
		}
		parsed.Range.Start = &start
		parsed.Range.End = &end
	} else if _, hasStart := values["start"]; hasStart {
		return contracts.MemoryHistoryRequest{}, fmt.Errorf("preset range cannot contain start")
	} else if _, hasEnd := values["end"]; hasEnd {
		return contracts.MemoryHistoryRequest{}, fmt.Errorf("preset range cannot contain end")
	}
	if err := parsed.Validate(); err != nil {
		return contracts.MemoryHistoryRequest{}, err
	}
	return parsed, nil
}
