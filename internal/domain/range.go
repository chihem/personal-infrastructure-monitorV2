package domain

import (
	"fmt"
	"time"
)

type RangePreset string

const (
	RangeLastMinute   RangePreset = "last_1m"
	RangeLast5Minutes RangePreset = "last_5m"
	RangeLast15Min    RangePreset = "last_15m"
	RangeLast30Min    RangePreset = "last_30m"
	RangeLastHour     RangePreset = "last_1h"
	RangeLast6Hours   RangePreset = "last_6h"
	RangeLast24Hours  RangePreset = "last_24h"
	RangeLast7Days    RangePreset = "last_7d"
	RangeLast14Days   RangePreset = "last_14d"
	RangeCustom       RangePreset = "custom"
)

func (preset RangePreset) Valid() bool {
	switch preset {
	case RangeLastMinute, RangeLast5Minutes, RangeLast15Min, RangeLast30Min,
		RangeLastHour, RangeLast6Hours, RangeLast24Hours, RangeLast7Days,
		RangeLast14Days, RangeCustom:
		return true
	default:
		return false
	}
}

type RangeSelection struct {
	Preset RangePreset `json:"preset"`
	Start  *time.Time  `json:"start"`
	End    *time.Time  `json:"end"`
}

func (selection RangeSelection) Validate() error {
	if !selection.Preset.Valid() {
		return fmt.Errorf("invalid range preset %q", selection.Preset)
	}
	if selection.Preset != RangeCustom {
		if selection.Start != nil || selection.End != nil {
			return fmt.Errorf("preset range cannot contain custom boundaries")
		}
		return nil
	}
	if selection.Start == nil || selection.End == nil {
		return fmt.Errorf("custom range requires start and end")
	}
	if err := ValidateUTC(*selection.Start); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	if err := ValidateUTC(*selection.End); err != nil {
		return fmt.Errorf("end: %w", err)
	}
	if !selection.End.After(*selection.Start) {
		return fmt.Errorf("range end must be after start")
	}
	return nil
}

type ResolvedRange struct {
	Preset RangePreset `json:"preset"`
	Start  time.Time   `json:"start"`
	End    time.Time   `json:"end"`
}

func (resolved ResolvedRange) Validate() error {
	if !resolved.Preset.Valid() {
		return fmt.Errorf("invalid range preset %q", resolved.Preset)
	}
	if err := ValidateUTC(resolved.Start); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	if err := ValidateUTC(resolved.End); err != nil {
		return fmt.Errorf("end: %w", err)
	}
	if !resolved.End.After(resolved.Start) {
		return fmt.Errorf("range end must be after start")
	}
	return nil
}
