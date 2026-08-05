package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/api/contracts"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/config"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/observability"
)

func TestLivenessReadinessStatusAndSecurityHeaders(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, HandlerOptions{
		Status: func() contracts.OperationalStatus { return testOperationalStatus() },
	})

	tests := []struct {
		path string
		code int
	}{
		{path: "/api/v1/health/live", code: http.StatusOK},
		{path: "/api/v1/health/ready", code: http.StatusOK},
		{path: "/api/v1/health", code: http.StatusOK},
		{path: "/api/v1/status", code: http.StatusOK},
	}
	for _, test := range tests {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
		if response.Code != test.code {
			t.Fatalf("GET %s status = %d, body = %s", test.path, response.Code, response.Body.String())
		}
		for _, header := range []string{"Content-Security-Policy", "Referrer-Policy", "X-Content-Type-Options", "X-Frame-Options", RequestIDHeader} {
			if response.Header().Get(header) == "" {
				t.Errorf("GET %s header %s is empty", test.path, header)
			}
		}
	}
}

func TestReadinessDistinguishesDegradedFromUnavailable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status contracts.OperationalStatus
		code   int
		ready  bool
	}{
		{name: "audit degraded remains ready", status: testOperationalStatus(), code: http.StatusOK, ready: true},
		{name: "maintenance", status: withOperationalMutation(func(status *contracts.OperationalStatus) {
			status.State = contracts.OperationalMaintenance
			status.Maintenance = true
		}), code: http.StatusServiceUnavailable},
		{name: "history unavailable", status: withOperationalMutation(func(status *contracts.OperationalStatus) {
			status.State = contracts.OperationalNotReady
			status.HistoryDatabase = contracts.DatabaseOperationalStatus{State: contracts.DependencyUnavailable}
		}), code: http.StatusServiceUnavailable},
		{name: "previous config remains ready", status: withOperationalMutation(func(status *contracts.OperationalStatus) {
			status.ConfigurationState = contracts.ConfigurationUsingPrevious
		}), code: http.StatusOK, ready: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := newTestHandler(t, HandlerOptions{Status: func() contracts.OperationalStatus { return test.status }})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/health/ready", nil))
			if response.Code != test.code {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			var envelope contracts.Envelope[contracts.ReadinessStatus]
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if envelope.Data == nil || envelope.Data.Ready != test.ready {
				t.Fatalf("readiness = %+v, want %v", envelope.Data, test.ready)
			}
		})
	}
}

func TestRequestIDPropagatesThroughContextResponseEnvelopeAndLog(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	seen := make(chan string, 1)
	handler := newTestHandler(t, HandlerOptions{
		WebHandler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			requestID, _ := RequestIDFromContext(request.Context())
			seen <- requestID
			writer.WriteHeader(http.StatusNoContent)
		}),
		Logger: observability.New(&stdout, &stderr),
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(RequestIDHeader, "client-controlled-id")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	requestID := response.Header().Get(RequestIDHeader)
	if requestID != "request-test-001" || <-seen != requestID {
		t.Fatalf("request ID was not propagated: header=%q", requestID)
	}
	if request.Header.Get(RequestIDHeader) != "" {
		t.Fatal("client request ID reached the application handler")
	}
	if !strings.Contains(stdout.String(), `"requestId":"request-test-001"`) || stderr.Len() != 0 {
		t.Fatalf("request log stdout=%s stderr=%s", stdout.String(), stderr.String())
	}

	apiHandler := newTestHandler(t, HandlerOptions{Status: func() contracts.OperationalStatus { return testOperationalStatus() }})
	apiResponse := httptest.NewRecorder()
	apiHandler.ServeHTTP(apiResponse, httptest.NewRequest(http.MethodGet, "/api/v1/status", nil))
	var envelope contracts.Envelope[contracts.OperationalStatus]
	if err := json.Unmarshal(apiResponse.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode status envelope: %v", err)
	}
	if envelope.RequestID != apiResponse.Header().Get(RequestIDHeader) {
		t.Fatalf("envelope request ID = %q, header = %q", envelope.RequestID, apiResponse.Header().Get(RequestIDHeader))
	}
}

func TestPanicRecoveryDoesNotExposePanicValue(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	sensitiveValue := "synthetic-sensitive-panic-value"
	panicHandler := observeRequests(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(sensitiveValue)
	}), observability.New(&stdout, &stderr), func() string { return "request-test-001" }, time.Now)
	panicResponse := httptest.NewRecorder()
	panicHandler.ServeHTTP(panicResponse, httptest.NewRequest(http.MethodGet, "/api/v1/panic-test", nil))
	if panicResponse.Code != http.StatusInternalServerError {
		t.Fatalf("panic status = %d, body = %s", panicResponse.Code, panicResponse.Body.String())
	}
	combined := panicResponse.Body.String() + stdout.String() + stderr.String()
	if strings.Contains(combined, sensitiveValue) || strings.Contains(combined, "goroutine") {
		t.Fatalf("panic detail escaped recovery boundary: %s", combined)
	}
	if !strings.Contains(stderr.String(), `"eventCode":"http.request.panic"`) {
		t.Fatalf("panic event missing from stderr: %s", stderr.String())
	}
}

func TestCancelledRequestProducesBoundedWarning(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	started := make(chan struct{})
	handler := newTestHandler(t, HandlerOptions{
		WebHandler: http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			close(started)
			<-request.Context().Done()
		}),
		Logger: observability.New(&stdout, &stderr),
	})
	ctx, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	finished := make(chan struct{})
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), request)
		close(finished)
	}()
	<-started
	cancel()
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled request did not finish")
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), `"eventCode":"http.request.cancelled"`) {
		t.Fatalf("cancellation logs stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestForwardedHeadersAreRemovedBeforeHandling(t *testing.T) {
	t.Parallel()

	seen := make(chan http.Header, 1)
	handler := newTestHandler(t, HandlerOptions{WebHandler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		seen <- request.Header.Clone()
		writer.WriteHeader(http.StatusNoContent)
	})})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Forwarded", "for=203.0.113.10")
	request.Header.Set("X-Forwarded-For", "203.0.113.10")
	request.Header.Set("X-Forwarded-Proto", "https")
	request.Header.Set("X-Real-IP", "203.0.113.10")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	headers := <-seen
	if headers.Get("Forwarded") != "" || headers.Get("X-Forwarded-For") != "" || headers.Get("X-Forwarded-Proto") != "" || headers.Get("X-Real-IP") != "" {
		t.Fatalf("forwarded headers reached handler: %+v", headers)
	}
}

func TestOversizedRequestBodyIsRejected(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, HandlerOptions{WebHandler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("oversized request reached application handler")
	})})
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
	handler := newTestHandler(t, HandlerOptions{WebHandler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, readErr = io.Copy(io.Discard, request.Body)
		writer.WriteHeader(http.StatusNoContent)
	})})
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
		ReadHeaderTimeoutSecs: 2, ReadTimeoutSecs: 3, WriteTimeoutSecs: 4, IdleTimeoutSecs: 5,
	}
	server := NewServer("192.0.2.23:8788", settings, http.NotFoundHandler())

	if server.Addr != "192.0.2.23:8788" ||
		server.ReadHeaderTimeout != 2*time.Second ||
		server.ReadTimeout != 3*time.Second ||
		server.WriteTimeout != 4*time.Second ||
		server.IdleTimeout != 5*time.Second ||
		server.MaxHeaderBytes != MaxHeaderBytes {
		t.Fatalf("server settings = %+v", server)
	}
}

func newTestHandler(t *testing.T, options HandlerOptions) http.Handler {
	t.Helper()
	options.requestIDGenerator = func() string { return "request-test-001" }
	options.now = func() time.Time { return time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC) }
	return NewHandlerWithOptions(options)
}

func testOperationalStatus() contracts.OperationalStatus {
	size := int64(4096)
	return contracts.OperationalStatus{
		State:              contracts.OperationalDegraded,
		UptimeSeconds:      15,
		ConfigurationState: contracts.ConfigurationValid,
		HistoryDatabase:    contracts.DatabaseOperationalStatus{State: contracts.DependencyAvailable, SizeBytes: &size},
		AuditDatabase:      contracts.DatabaseOperationalStatus{State: contracts.DependencyUnavailable},
		Collection:         contracts.CollectionOperationalStatus{State: contracts.DependencyNotStarted},
		BackupState:        contracts.DependencyNotImplemented,
		DockerConnectivity: contracts.DependencyNotChecked,
	}
}

func withOperationalMutation(mutate func(*contracts.OperationalStatus)) contracts.OperationalStatus {
	status := testOperationalStatus()
	mutate(&status)
	return status
}
