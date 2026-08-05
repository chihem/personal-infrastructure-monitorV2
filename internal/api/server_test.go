package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/config"
)

func TestHealthPlaceholderAndSecurityHeaders(t *testing.T) {
	t.Parallel()

	handler := NewHandler(http.NotFoundHandler(), func() FoundationHealth {
		return FoundationHealth{
			HistoryDatabaseAvailable: true,
			AuditDatabaseAvailable:   true,
			ConfigurationState:       "valid",
		}
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	for _, header := range []string{"Content-Security-Policy", "Referrer-Policy", "X-Content-Type-Options", "X-Frame-Options"} {
		if response.Header().Get(header) == "" {
			t.Errorf("%s header is empty", header)
		}
	}
	var payload healthResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.APIVersion != "v1" || payload.Status != "ok" {
		t.Fatalf("health response = %+v", payload)
	}
}

func TestHealthReportsMaintenanceAndDegradedState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		health FoundationHealth
		want   string
	}{
		{name: "maintenance", health: FoundationHealth{Maintenance: true, HistoryDatabaseAvailable: true, AuditDatabaseAvailable: true}, want: "maintenance"},
		{name: "audit unavailable", health: FoundationHealth{HistoryDatabaseAvailable: true}, want: "degraded"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			handler := NewHandler(nil, func() FoundationHealth { return test.health })
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
			var payload healthResponse
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if payload.Status != test.want {
				t.Fatalf("status = %q, want %q", payload.Status, test.want)
			}
		})
	}
}

func TestForwardedHeadersAreRemovedBeforeHandling(t *testing.T) {
	t.Parallel()

	seen := make(chan http.Header, 1)
	handler := NewHandler(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		seen <- request.Header.Clone()
		writer.WriteHeader(http.StatusNoContent)
	}), nil)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Forwarded", "for=203.0.113.10")
	request.Header.Set("X-Forwarded-For", "203.0.113.10")
	request.Header.Set("X-Forwarded-Proto", "https")
	request.Header.Set("X-Real-IP", "203.0.113.10")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	headers := <-seen
	if headers.Get("Forwarded") != "" || headers.Get("X-Forwarded-For") != "" || headers.Get("X-Forwarded-Proto") != "" {
		t.Fatalf("forwarded headers reached handler: %+v", headers)
	}
	if headers.Get("X-Real-IP") != "" {
		t.Fatal("X-Real-IP reached the application handler")
	}
}

func TestOversizedRequestBodyIsRejected(t *testing.T) {
	t.Parallel()

	handler := NewHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("oversized request reached application handler")
	}), nil)
	body := bytes.NewReader(make([]byte, MaxRequestBodyBytes+1))
	request := httptest.NewRequest(http.MethodPost, "/", body)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestStreamingRequestBodyIsBounded(t *testing.T) {
	t.Parallel()

	var readErr error
	handler := NewHandler(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, readErr = io.Copy(io.Discard, request.Body)
		writer.WriteHeader(http.StatusNoContent)
	}), nil)
	request := httptest.NewRequest(http.MethodGet, "/", bytes.NewReader(make([]byte, MaxRequestBodyBytes+1)))
	request.ContentLength = -1
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	var maxBytesError *http.MaxBytesError
	if !errors.As(readErr, &maxBytesError) {
		t.Fatalf("body read error = %v, want *http.MaxBytesError", readErr)
	}
}

func TestNewServerAppliesConfiguredTimeoutsAndHeaderLimit(t *testing.T) {
	t.Parallel()

	settings := config.ServerSettings{
		ReadHeaderTimeoutSecs: 2,
		ReadTimeoutSecs:       3,
		WriteTimeoutSecs:      4,
		IdleTimeoutSecs:       5,
	}
	server := NewServer("100.64.0.23:8788", settings, http.NotFoundHandler())

	if server.Addr != "100.64.0.23:8788" ||
		server.ReadHeaderTimeout != 2*time.Second ||
		server.ReadTimeout != 3*time.Second ||
		server.WriteTimeout != 4*time.Second ||
		server.IdleTimeout != 5*time.Second ||
		server.MaxHeaderBytes != MaxHeaderBytes {
		t.Fatalf("server settings = %+v", server)
	}
}
