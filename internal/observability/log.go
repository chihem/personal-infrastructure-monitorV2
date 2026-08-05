// Package observability provides bounded operational logging primitives.
package observability

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

type ResourceIdentity struct {
	Kind string
	ID   string
}

type Event struct {
	Severity   Severity
	Component  string
	Code       string
	RequestID  string
	Duration   *time.Duration
	StatusCode int
	Resource   *ResourceIdentity
}

type Logger struct {
	stdout io.Writer
	stderr io.Writer
	now    func() time.Time
	mu     sync.Mutex
}

type Option func(*Logger)

func WithClock(now func() time.Time) Option {
	return func(logger *Logger) {
		if now != nil {
			logger.now = now
		}
	}
}

func New(stdout, stderr io.Writer, options ...Option) *Logger {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	logger := &Logger{stdout: stdout, stderr: stderr, now: time.Now}
	for _, option := range options {
		option(logger)
	}
	return logger
}

func Discard() *Logger {
	return New(io.Discard, io.Discard)
}

func (logger *Logger) Info(event Event) error {
	event.Severity = SeverityInfo
	return logger.Log(event)
}

func (logger *Logger) Warning(event Event) error {
	event.Severity = SeverityWarning
	return logger.Log(event)
}

func (logger *Logger) Error(event Event) error {
	event.Severity = SeverityError
	return logger.Log(event)
}

func (logger *Logger) Log(event Event) error {
	if err := event.validate(); err != nil {
		return err
	}

	record := logRecord{
		Timestamp:  logger.now().UTC(),
		Severity:   event.Severity,
		Component:  event.Component,
		EventCode:  event.Code,
		RequestID:  event.RequestID,
		StatusCode: event.StatusCode,
	}
	if event.Duration != nil {
		durationMilliseconds := event.Duration.Milliseconds()
		record.DurationMilliseconds = &durationMilliseconds
	}
	if event.Resource != nil {
		record.ResourceKind = event.Resource.Kind
		record.ResourceKey = hashedResourceKey(*event.Resource)
	}

	destination := logger.stdout
	if event.Severity == SeverityWarning || event.Severity == SeverityError {
		destination = logger.stderr
	}

	logger.mu.Lock()
	defer logger.mu.Unlock()
	if err := json.NewEncoder(destination).Encode(record); err != nil {
		return fmt.Errorf("encode operational log record: %w", err)
	}
	return nil
}

type logRecord struct {
	Timestamp            time.Time `json:"timestamp"`
	Severity             Severity  `json:"severity"`
	Component            string    `json:"component"`
	EventCode            string    `json:"eventCode"`
	RequestID            string    `json:"requestId,omitempty"`
	DurationMilliseconds *int64    `json:"durationMs,omitempty"`
	StatusCode           int       `json:"statusCode,omitempty"`
	ResourceKind         string    `json:"resourceKind,omitempty"`
	ResourceKey          string    `json:"resourceKey,omitempty"`
}

func (event Event) validate() error {
	if event.Severity != SeverityInfo && event.Severity != SeverityWarning && event.Severity != SeverityError {
		return fmt.Errorf("invalid operational log severity %q", event.Severity)
	}
	if !validCode(event.Component) {
		return fmt.Errorf("invalid operational log component %q", event.Component)
	}
	if !validCode(event.Code) {
		return fmt.Errorf("invalid operational log event code %q", event.Code)
	}
	if event.RequestID != "" {
		if err := domain.ValidateOpaqueID(event.RequestID); err != nil {
			return fmt.Errorf("request ID: %w", err)
		}
	}
	if event.Duration != nil && *event.Duration < 0 {
		return fmt.Errorf("operational log duration must not be negative")
	}
	if event.StatusCode != 0 && (event.StatusCode < 100 || event.StatusCode > 599) {
		return fmt.Errorf("invalid HTTP status code %d", event.StatusCode)
	}
	if event.Resource != nil {
		if !validCode(event.Resource.Kind) {
			return fmt.Errorf("invalid resource kind %q", event.Resource.Kind)
		}
		if err := domain.ValidateOpaqueID(event.Resource.ID); err != nil {
			return fmt.Errorf("resource ID: %w", err)
		}
	}
	return nil
}

func validCode(value string) bool {
	if value == "" || len(value) > 96 {
		return false
	}
	for _, character := range value {
		valid := character >= 'a' && character <= 'z' ||
			character >= '0' && character <= '9' ||
			character == '.' || character == '-' || character == '_'
		if !valid {
			return false
		}
	}
	return true
}

func hashedResourceKey(resource ResourceIdentity) string {
	digest := sha256.Sum256([]byte(resource.Kind + "\x00" + resource.ID))
	return "sha256:" + hex.EncodeToString(digest[:12])
}
