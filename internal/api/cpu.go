package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/api/contracts"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

type CPUDataSource interface {
	CurrentCPU() contracts.CPUSnapshot
	CPUHistory(context.Context, contracts.CPUHistoryRequest) (contracts.CPUHistorySeries, error)
}

func registerCPURoutes(mux *http.ServeMux, source CPUDataSource, now func() time.Time) {
	mux.HandleFunc("/api/v1/cpu", func(writer http.ResponseWriter, request *http.Request) {
		if !allowReadMethod(writer, request, now) {
			return
		}
		if source == nil {
			writeAPIError(writer, request, http.StatusServiceUnavailable, now, domain.ErrorUnavailable, "errors.cpuUnavailable", "CPU information is unavailable.")
			return
		}
		snapshot := source.CurrentCPU()
		if err := snapshot.Validate(); err != nil {
			writeInternalError(writer, request, now)
			return
		}
		writeData(writer, request, http.StatusOK, now, snapshot)
	})

	mux.HandleFunc("/api/v1/cpu/history", func(writer http.ResponseWriter, request *http.Request) {
		if !allowReadMethod(writer, request, now) {
			return
		}
		query, err := parseCPUHistoryRequest(request)
		if err != nil {
			writeAPIError(writer, request, http.StatusBadRequest, now, domain.ErrorValidationFailed, "errors.invalidCPUHistoryQuery", "The CPU history query is invalid.")
			return
		}
		if source == nil {
			writeAPIError(writer, request, http.StatusServiceUnavailable, now, domain.ErrorHistoryUnavailable, "errors.historyUnavailable", "CPU history is unavailable.")
			return
		}
		series, err := source.CPUHistory(request.Context(), query)
		if err != nil {
			if errors.Is(err, contracts.ErrInvalidCPUHistoryRange) {
				writeAPIError(writer, request, http.StatusBadRequest, now, domain.ErrorValidationFailed, "errors.invalidCPUHistoryQuery", "The CPU history query is invalid.")
				return
			}
			writeAPIError(writer, request, http.StatusServiceUnavailable, now, domain.ErrorHistoryUnavailable, "errors.historyUnavailable", "CPU history is unavailable.")
			return
		}
		if err := series.Validate(); err != nil {
			writeInternalError(writer, request, now)
			return
		}
		writeData(writer, request, http.StatusOK, now, series)
	})
}

func parseCPUHistoryRequest(request *http.Request) (contracts.CPUHistoryRequest, error) {
	values := request.URL.Query()
	allowed := map[string]struct{}{"metric": {}, "range": {}, "core": {}, "start": {}, "end": {}}
	for key, list := range values {
		if _, ok := allowed[key]; !ok {
			return contracts.CPUHistoryRequest{}, fmt.Errorf("unsupported query parameter")
		}
		if len(list) != 1 {
			return contracts.CPUHistoryRequest{}, fmt.Errorf("query parameters cannot be repeated")
		}
	}
	metricValues, metricPresent := values["metric"]
	rangeValues, rangePresent := values["range"]
	if !metricPresent || !rangePresent || metricValues[0] == "" || rangeValues[0] == "" {
		return contracts.CPUHistoryRequest{}, fmt.Errorf("metric and range are required")
	}

	parsed := contracts.CPUHistoryRequest{
		Metric: contracts.CPUMetric(metricValues[0]),
		Range:  domain.RangeSelection{Preset: domain.RangePreset(rangeValues[0])},
	}
	if coreValues, present := values["core"]; present {
		index, err := strconv.Atoi(coreValues[0])
		if err != nil || index < 0 {
			return contracts.CPUHistoryRequest{}, fmt.Errorf("core must be a non-negative integer")
		}
		parsed.CoreIndex = &index
	}
	if parsed.Range.Preset == domain.RangeCustom {
		start, err := parseUTCQueryTime(values, "start")
		if err != nil {
			return contracts.CPUHistoryRequest{}, err
		}
		end, err := parseUTCQueryTime(values, "end")
		if err != nil {
			return contracts.CPUHistoryRequest{}, err
		}
		parsed.Range.Start = &start
		parsed.Range.End = &end
	} else if _, hasStart := values["start"]; hasStart {
		return contracts.CPUHistoryRequest{}, fmt.Errorf("preset range cannot contain start")
	} else if _, hasEnd := values["end"]; hasEnd {
		return contracts.CPUHistoryRequest{}, fmt.Errorf("preset range cannot contain end")
	}
	if err := parsed.Validate(); err != nil {
		return contracts.CPUHistoryRequest{}, err
	}
	return parsed, nil
}

func parseUTCQueryTime(values map[string][]string, name string) (time.Time, error) {
	items, present := values[name]
	if !present || items[0] == "" {
		return time.Time{}, fmt.Errorf("custom range requires %s", name)
	}
	value, err := time.Parse(time.RFC3339, items[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be RFC 3339", name)
	}
	if err := domain.ValidateUTC(value); err != nil {
		return time.Time{}, err
	}
	return value, nil
}
