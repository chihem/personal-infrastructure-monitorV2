package memory

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const kibibyte = int64(1024)

var requiredMemoryFields = []string{
	"MemTotal", "MemFree", "MemAvailable", "Buffers", "Cached", "SwapTotal", "SwapFree",
}

func parseMemInfo(data []byte) (map[string]int64, error) {
	values := make(map[string]int64, len(requiredMemoryFields))
	wanted := make(map[string]struct{}, len(requiredMemoryFields))
	seen := make(map[string]bool, len(requiredMemoryFields))
	for _, field := range requiredMemoryFields {
		wanted[field] = struct{}{}
	}
	var problems error
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		name, remainder, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		if _, required := wanted[name]; !required {
			continue
		}
		if seen[name] {
			problems = errors.Join(problems, fmt.Errorf("%s is duplicated", name))
			delete(values, name)
			delete(wanted, name)
			continue
		}
		seen[name] = true
		fields := strings.Fields(remainder)
		if len(fields) != 2 || fields[1] != "kB" {
			problems = errors.Join(problems, fmt.Errorf("%s must use a kB value", name))
			delete(wanted, name)
			continue
		}
		value, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil || value < 0 || value > math.MaxInt64/kibibyte {
			problems = errors.Join(problems, fmt.Errorf("%s has an invalid value", name))
			delete(wanted, name)
			continue
		}
		values[name] = value * kibibyte
	}
	if err := scanner.Err(); err != nil {
		problems = errors.Join(problems, fmt.Errorf("scan meminfo: %w", err))
	}
	for _, field := range requiredMemoryFields {
		if _, present := values[field]; !present {
			problems = errors.Join(problems, fmt.Errorf("%s is missing", field))
		}
	}
	if total, present := values["MemTotal"]; present && total == 0 {
		delete(values, "MemTotal")
		problems = errors.Join(problems, fmt.Errorf("MemTotal must be positive"))
	}
	return values, problems
}

type pressureValues struct {
	Some pressureWindowValues
	Full pressureWindowValues
}

type pressureWindowValues struct {
	Average10  float64
	Average60  float64
	Average300 float64
	Total      int64
}

func parsePressure(data []byte) (pressureValues, error) {
	var result pressureValues
	seen := map[string]bool{}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 1024), 64*1024)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 5 || (fields[0] != "some" && fields[0] != "full") || seen[fields[0]] {
			return pressureValues{}, fmt.Errorf("pressure data has an invalid line")
		}
		seen[fields[0]] = true
		window, err := parsePressureWindow(fields[1:])
		if err != nil {
			return pressureValues{}, fmt.Errorf("pressure %s: %w", fields[0], err)
		}
		if fields[0] == "some" {
			result.Some = window
		} else {
			result.Full = window
		}
	}
	if err := scanner.Err(); err != nil {
		return pressureValues{}, fmt.Errorf("scan pressure data: %w", err)
	}
	if !seen["some"] || !seen["full"] {
		return pressureValues{}, fmt.Errorf("pressure data requires some and full lines")
	}
	return result, nil
}

func parsePressureWindow(fields []string) (pressureWindowValues, error) {
	values := map[string]string{}
	for _, field := range fields {
		name, value, found := strings.Cut(field, "=")
		if !found || value == "" || values[name] != "" {
			return pressureWindowValues{}, fmt.Errorf("invalid pressure field")
		}
		values[name] = value
	}
	if len(values) != 4 {
		return pressureWindowValues{}, fmt.Errorf("unexpected pressure fields")
	}
	average10, err := parsePressureAverage(values["avg10"])
	if err != nil {
		return pressureWindowValues{}, fmt.Errorf("avg10: %w", err)
	}
	average60, err := parsePressureAverage(values["avg60"])
	if err != nil {
		return pressureWindowValues{}, fmt.Errorf("avg60: %w", err)
	}
	average300, err := parsePressureAverage(values["avg300"])
	if err != nil {
		return pressureWindowValues{}, fmt.Errorf("avg300: %w", err)
	}
	total, err := strconv.ParseInt(values["total"], 10, 64)
	if err != nil || total < 0 {
		return pressureWindowValues{}, fmt.Errorf("total must be a non-negative integer")
	}
	return pressureWindowValues{Average10: average10, Average60: average60, Average300: average300, Total: total}, nil
}

func parsePressureAverage(value string) (float64, error) {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) || parsed < 0 || parsed > 100 {
		return 0, fmt.Errorf("value must be between 0 and 100")
	}
	return parsed, nil
}
