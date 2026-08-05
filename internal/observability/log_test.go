package observability

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestLoggerWritesAllowlistedStructuredFieldsToSeverityStream(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	logger := New(&stdout, &stderr, WithClock(func() time.Time { return now }))
	duration := 1250 * time.Millisecond
	secretBearingIdentity := "container-token-value"

	if err := logger.Info(Event{
		Component: "http", Code: "http.request.completed", RequestID: "request-001",
		Duration: &duration, StatusCode: 204,
		Resource: &ResourceIdentity{Kind: "client", ID: secretBearingIdentity},
	}); err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("Info event reached stderr: %s", stderr.String())
	}
	if strings.Contains(stdout.String(), secretBearingIdentity) {
		t.Fatal("raw resource identity appeared in operational log")
	}

	var record map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &record); err != nil {
		t.Fatalf("decode JSON log: %v", err)
	}
	for _, field := range []string{"timestamp", "severity", "component", "eventCode", "requestId", "durationMs", "statusCode", "resourceKind", "resourceKey"} {
		if _, ok := record[field]; !ok {
			t.Errorf("log field %q is missing", field)
		}
	}
	for _, forbidden := range []string{"message", "error", "headers", "token", "containerLogs", "environment"} {
		if _, ok := record[forbidden]; ok {
			t.Errorf("forbidden arbitrary field %q is present", forbidden)
		}
	}

	if err := logger.Error(Event{Component: "app", Code: "app.run.failed"}); err != nil {
		t.Fatalf("Error() error = %v", err)
	}
	if !strings.Contains(stderr.String(), `"severity":"error"`) {
		t.Fatalf("Error event did not reach stderr: %s", stderr.String())
	}
}

func TestLoggerRejectsUnsafeMetadata(t *testing.T) {
	logger := Discard()
	tests := []Event{
		{Severity: SeverityInfo, Component: "http headers", Code: "request.completed"},
		{Severity: SeverityInfo, Component: "http", Code: "request/../../secret"},
		{Severity: SeverityInfo, Component: "http", Code: "request.completed", RequestID: "line\nbreak"},
		{Severity: SeverityInfo, Component: "http", Code: "request.completed", Resource: &ResourceIdentity{Kind: "client", ID: ""}},
	}
	for _, event := range tests {
		if err := logger.Log(event); err == nil {
			t.Fatalf("Log() accepted unsafe event: %+v", event)
		}
	}
}
