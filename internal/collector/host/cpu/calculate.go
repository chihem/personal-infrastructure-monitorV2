package cpu

import (
	"errors"
	"math"
)

var (
	errCounterReset      = errors.New("CPU counter decreased")
	errCounterNoProgress = errors.New("CPU counter did not advance")
)

func calculateUsage(previous, current Counter) (float64, error) {
	if err := previous.validate(); err != nil {
		return 0, err
	}
	if err := current.validate(); err != nil {
		return 0, err
	}
	previousValues := counterValues(previous)
	currentValues := counterValues(current)
	for index := range previousValues {
		if currentValues[index] < previousValues[index] {
			return 0, errCounterReset
		}
	}

	previousTotal, previousIdle := counterTotals(previous)
	currentTotal, currentIdle := counterTotals(current)
	deltaTotal := currentTotal - previousTotal
	deltaIdle := currentIdle - previousIdle
	if deltaTotal <= 0 {
		return 0, errCounterNoProgress
	}
	if deltaIdle < 0 || deltaIdle > deltaTotal {
		return 0, errCounterReset
	}
	percentage := (deltaTotal - deltaIdle) / deltaTotal * 100
	if math.IsNaN(percentage) || math.IsInf(percentage, 0) || percentage < 0 || percentage > 100 {
		return 0, errCounterReset
	}
	return percentage, nil
}

func counterTotals(counter Counter) (total float64, idle float64) {
	total = counter.User + counter.Nice + counter.System + counter.Idle +
		counter.IOWait + counter.IRQ + counter.SoftIRQ + counter.Steal
	idle = counter.Idle + counter.IOWait
	return total, idle
}

func counterValues(counter Counter) []float64 {
	return []float64{
		counter.User, counter.Nice, counter.System, counter.Idle,
		counter.IOWait, counter.IRQ, counter.SoftIRQ, counter.Steal,
	}
}

func aggregateCounters(counters []Counter) Counter {
	var total Counter
	for _, counter := range counters {
		total.User += counter.User
		total.System += counter.System
		total.Idle += counter.Idle
		total.Nice += counter.Nice
		total.IOWait += counter.IOWait
		total.IRQ += counter.IRQ
		total.SoftIRQ += counter.SoftIRQ
		total.Steal += counter.Steal
	}
	return total
}
