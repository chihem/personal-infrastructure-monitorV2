package contracts

import (
	"fmt"
	"strings"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

const APIVersion = "v1"

type APIError struct {
	Code            domain.ErrorCode `json:"code"`
	MessageKey      string           `json:"messageKey"`
	FallbackMessage string           `json:"fallbackMessage"`
	TechnicalDetail *string          `json:"technicalDetail"`
	FieldErrors     []FieldError     `json:"fieldErrors"`
}

type FieldError struct {
	Field       string `json:"field"`
	Code        string `json:"code"`
	MessageKey  string `json:"messageKey"`
	FallbackMsg string `json:"fallbackMessage"`
}

func (apiError APIError) Validate() error {
	if !apiError.Code.Valid() {
		return fmt.Errorf("invalid error code %q", apiError.Code)
	}
	if err := validateMessage(apiError.MessageKey, 160, "messageKey"); err != nil {
		return err
	}
	if err := validateMessage(apiError.FallbackMessage, 512, "fallbackMessage"); err != nil {
		return err
	}
	if apiError.TechnicalDetail != nil && len(*apiError.TechnicalDetail) > 4096 {
		return fmt.Errorf("technicalDetail exceeds 4096 characters")
	}
	if apiError.FieldErrors == nil {
		return fmt.Errorf("fieldErrors must be an array")
	}
	for index, fieldError := range apiError.FieldErrors {
		if err := fieldError.Validate(); err != nil {
			return fmt.Errorf("fieldErrors[%d]: %w", index, err)
		}
	}
	return nil
}

func (fieldError FieldError) Validate() error {
	if err := validateMessage(fieldError.Field, 160, "field"); err != nil {
		return err
	}
	if err := validateMessage(fieldError.Code, 160, "code"); err != nil {
		return err
	}
	if err := validateMessage(fieldError.MessageKey, 160, "messageKey"); err != nil {
		return err
	}
	return validateMessage(fieldError.FallbackMsg, 512, "fallbackMessage")
}

type Envelope[T any] struct {
	APIVersion  string    `json:"apiVersion"`
	RequestID   string    `json:"requestId"`
	GeneratedAt time.Time `json:"generatedAt"`
	Data        *T        `json:"data"`
	Error       *APIError `json:"error"`
}

func (envelope Envelope[T]) Validate(validateData func(T) error) error {
	if envelope.APIVersion != APIVersion {
		return fmt.Errorf("apiVersion must be %q", APIVersion)
	}
	if err := domain.ValidateOpaqueID(envelope.RequestID); err != nil {
		return fmt.Errorf("requestId: %w", err)
	}
	if err := domain.ValidateUTC(envelope.GeneratedAt); err != nil {
		return fmt.Errorf("generatedAt: %w", err)
	}
	if (envelope.Data == nil) == (envelope.Error == nil) {
		return fmt.Errorf("exactly one of data or error must be present")
	}
	if envelope.Error != nil {
		return envelope.Error.Validate()
	}
	if validateData != nil {
		return validateData(*envelope.Data)
	}
	return nil
}

type PageInfo struct {
	Limit      int     `json:"limit"`
	HasMore    bool    `json:"hasMore"`
	NextCursor *string `json:"nextCursor"`
}

func (page PageInfo) Validate() error {
	if page.Limit < 1 || page.Limit > 200 {
		return fmt.Errorf("page limit must be between 1 and 200")
	}
	if page.HasMore && (page.NextCursor == nil || *page.NextCursor == "") {
		return fmt.Errorf("hasMore page requires nextCursor")
	}
	if !page.HasMore && page.NextCursor != nil {
		return fmt.Errorf("final page cannot contain nextCursor")
	}
	if page.NextCursor != nil {
		if err := domain.ValidateOpaqueID(*page.NextCursor); err != nil {
			return fmt.Errorf("nextCursor: %w", err)
		}
	}
	return nil
}

func validateMessage(value string, maximum int, field string) error {
	if strings.TrimSpace(value) == "" || len(value) > maximum {
		return fmt.Errorf("%s must contain 1 to %d characters", field, maximum)
	}
	return nil
}
