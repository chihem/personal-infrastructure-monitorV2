package cpu

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	gopsutilcpu "github.com/shirou/gopsutil/v4/cpu"
	gopsutilload "github.com/shirou/gopsutil/v4/load"
)

type Counter struct {
	LogicalIndex int
	User         float64
	System       float64
	Idle         float64
	Nice         float64
	IOWait       float64
	IRQ          float64
	SoftIRQ      float64
	Steal        float64
}

type LoadAverage struct {
	One     float64
	Five    float64
	Fifteen float64
}

type Source interface {
	// Counters returns one cumulative counter per currently visible logical CPU.
	Counters(context.Context) ([]Counter, error)
	// LoadAverage returns only the host-wide 1, 5, and 15 minute averages.
	LoadAverage(context.Context) (LoadAverage, error)
}

type gopsutilSource struct{}

func (gopsutilSource) Counters(ctx context.Context) ([]Counter, error) {
	times, err := gopsutilcpu.TimesWithContext(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("read CPU counters: %w", err)
	}
	counters := make([]Counter, 0, len(times))
	seen := make(map[int]struct{}, len(times))
	for _, value := range times {
		indexText, ok := strings.CutPrefix(strings.ToLower(value.CPU), "cpu")
		if !ok || indexText == "" {
			return nil, fmt.Errorf("unexpected logical CPU name")
		}
		index, err := strconv.Atoi(indexText)
		if err != nil || index < 0 {
			return nil, fmt.Errorf("unexpected logical CPU name")
		}
		if _, duplicate := seen[index]; duplicate {
			return nil, fmt.Errorf("duplicate logical CPU index %d", index)
		}
		seen[index] = struct{}{}
		counter := Counter{
			LogicalIndex: index,
			User:         value.User, System: value.System, Idle: value.Idle, Nice: value.Nice,
			IOWait: value.Iowait, IRQ: value.Irq, SoftIRQ: value.Softirq, Steal: value.Steal,
		}
		if err := counter.validate(); err != nil {
			return nil, fmt.Errorf("logical CPU %d: %w", index, err)
		}
		counters = append(counters, counter)
	}
	return normalizeCounters(counters)
}

func normalizeCounters(counters []Counter) ([]Counter, error) {
	if len(counters) == 0 {
		return nil, fmt.Errorf("no logical CPU counters were returned")
	}
	normalized := cloneCounters(counters)
	sort.Slice(normalized, func(left, right int) bool {
		return normalized[left].LogicalIndex < normalized[right].LogicalIndex
	})
	previousIndex := -1
	for _, counter := range normalized {
		if counter.LogicalIndex < 0 || counter.LogicalIndex == previousIndex {
			return nil, fmt.Errorf("logical CPU indexes must be unique and non-negative")
		}
		if err := counter.validate(); err != nil {
			return nil, fmt.Errorf("logical CPU %d: %w", counter.LogicalIndex, err)
		}
		previousIndex = counter.LogicalIndex
	}
	return normalized, nil
}

func (gopsutilSource) LoadAverage(ctx context.Context) (LoadAverage, error) {
	average, err := gopsutilload.AvgWithContext(ctx)
	if err != nil {
		return LoadAverage{}, fmt.Errorf("read load averages: %w", err)
	}
	value := LoadAverage{One: average.Load1, Five: average.Load5, Fifteen: average.Load15}
	if err := value.validate(); err != nil {
		return LoadAverage{}, err
	}
	return value, nil
}

func (counter Counter) validate() error {
	for _, value := range []float64{
		counter.User, counter.System, counter.Idle, counter.Nice,
		counter.IOWait, counter.IRQ, counter.SoftIRQ, counter.Steal,
	} {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			return fmt.Errorf("counter values must be finite and non-negative")
		}
	}
	return nil
}

func (average LoadAverage) validate() error {
	for _, value := range []float64{average.One, average.Five, average.Fifteen} {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			return fmt.Errorf("load averages must be finite and non-negative")
		}
	}
	return nil
}
