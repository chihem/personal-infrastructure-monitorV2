package domain

import "fmt"

type Metric[T any] struct {
	Availability Availability          `json:"availability"`
	Value        *T                    `json:"value"`
	Unit         Unit                  `json:"unit"`
	ReasonCode   *UnavailabilityReason `json:"reasonCode"`
}

func (metric Metric[T]) Validate() error {
	if !metric.Availability.Valid() {
		return fmt.Errorf("invalid metric availability %q", metric.Availability)
	}
	if !metric.Unit.Valid() {
		return fmt.Errorf("invalid metric unit %q", metric.Unit)
	}

	switch metric.Availability {
	case AvailabilityAvailable:
		if metric.Value == nil {
			return fmt.Errorf("available metric requires a value")
		}
		if metric.ReasonCode != nil {
			return fmt.Errorf("available metric cannot contain a reason code")
		}
	case AvailabilityUnavailable:
		if metric.Value != nil {
			return fmt.Errorf("unavailable metric cannot contain a value")
		}
		if metric.ReasonCode == nil || !metric.ReasonCode.Valid() {
			return fmt.Errorf("unavailable metric requires a valid reason code")
		}
	}

	return nil
}
