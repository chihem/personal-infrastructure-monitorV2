package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/api/contracts"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/config"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/observability"
)

const (
	MaxRequestBodyBytes int64 = 1 << 20
	MaxHeaderBytes            = 16 << 10
	RequestIDHeader           = "X-Request-ID"
)

type OperationalStatusProvider func() contracts.OperationalStatus

type HandlerOptions struct {
	WebHandler http.Handler
	Status     OperationalStatusProvider
	Logger     *observability.Logger

	requestIDGenerator func() string
	now                func() time.Time
}

type requestIDContextKey struct{}

var fallbackRequestID atomic.Uint64

// NewHandler combines the versioned API with the embedded web application.
func NewHandler(webHandler http.Handler, status OperationalStatusProvider) http.Handler {
	return NewHandlerWithOptions(HandlerOptions{WebHandler: webHandler, Status: status})
}

func NewHandlerWithOptions(options HandlerOptions) http.Handler {
	if options.WebHandler == nil {
		options.WebHandler = http.NotFoundHandler()
	}
	if options.Status == nil {
		options.Status = unavailableOperationalStatus
	}
	if options.Logger == nil {
		options.Logger = observability.Discard()
	}
	if options.requestIDGenerator == nil {
		options.requestIDGenerator = newRequestID
	}
	if options.now == nil {
		options.now = time.Now
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health/live", func(writer http.ResponseWriter, request *http.Request) {
		if !allowReadMethod(writer, request, options.now) {
			return
		}
		writeData(writer, request, http.StatusOK, options.now, contracts.LivenessStatus{Alive: true})
	})

	readinessHandler := func(writer http.ResponseWriter, request *http.Request) {
		if !allowReadMethod(writer, request, options.now) {
			return
		}
		snapshot := options.Status()
		if err := contracts.ValidateOperationalStatus(snapshot); err != nil {
			writeInternalError(writer, request, options.now)
			return
		}
		readiness := readinessFrom(snapshot)
		statusCode := http.StatusOK
		if !readiness.Ready {
			statusCode = http.StatusServiceUnavailable
		}
		writeData(writer, request, statusCode, options.now, readiness)
	}
	mux.HandleFunc("/api/v1/health/ready", readinessHandler)
	mux.HandleFunc("/api/v1/health", readinessHandler)

	mux.HandleFunc("/api/v1/status", func(writer http.ResponseWriter, request *http.Request) {
		if !allowReadMethod(writer, request, options.now) {
			return
		}
		snapshot := options.Status()
		if err := contracts.ValidateOperationalStatus(snapshot); err != nil {
			writeInternalError(writer, request, options.now)
			return
		}
		writeData(writer, request, http.StatusOK, options.now, snapshot)
	})

	mux.HandleFunc("/api/", func(writer http.ResponseWriter, request *http.Request) {
		writeAPIError(writer, request, http.StatusNotFound, options.now, domain.ErrorNotFound, "errors.notFound", "The requested endpoint was not found.")
	})
	mux.Handle("/", options.WebHandler)

	handler := requestSafeguards(mux, options.now)
	return observeRequests(handler, options.Logger, options.requestIDGenerator, options.now)
}

func RequestIDFromContext(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDContextKey{}).(string)
	return requestID, ok && requestID != ""
}

func requestSafeguards(next http.Handler, now func() time.Time) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Security-Policy", "default-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'; object-src 'none'")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("X-Frame-Options", "DENY")

		removeForwardedHeaders(request.Header)
		request.Header.Del(RequestIDHeader)
		if request.ContentLength > MaxRequestBodyBytes {
			if strings.HasPrefix(request.URL.Path, "/api/") {
				writeAPIError(writer, request, http.StatusRequestEntityTooLarge, now, domain.ErrorValidationFailed, "errors.requestTooLarge", "The request body is too large.")
			} else {
				http.Error(writer, "request body too large", http.StatusRequestEntityTooLarge)
			}
			return
		}
		request.Body = http.MaxBytesReader(writer, request.Body, MaxRequestBodyBytes)
		next.ServeHTTP(writer, request)
	})
}

func observeRequests(
	next http.Handler,
	logger *observability.Logger,
	requestIDGenerator func() string,
	now func() time.Time,
) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		startedAt := now()
		requestID := requestIDGenerator()
		writer.Header().Set(RequestIDHeader, requestID)
		ctx := context.WithValue(request.Context(), requestIDContextKey{}, requestID)
		request = request.WithContext(ctx)
		capture := &responseCapture{ResponseWriter: writer}
		resource := remoteResource(request.RemoteAddr)

		defer func() {
			duration := now().Sub(startedAt)
			if duration < 0 {
				duration = 0
			}
			if recovered := recover(); recovered != nil {
				_ = logger.Error(observability.Event{
					Component: "http", Code: "http.request.panic", RequestID: requestID,
					Duration: &duration, StatusCode: http.StatusInternalServerError, Resource: resource,
				})
				if !capture.wroteHeader {
					if strings.HasPrefix(request.URL.Path, "/api/") {
						writeInternalError(capture, request, now)
					} else {
						http.Error(capture, "internal server error", http.StatusInternalServerError)
					}
				}
				return
			}

			event := observability.Event{
				Component: "http", Code: "http.request.completed", RequestID: requestID,
				Duration: &duration, StatusCode: capture.statusCode(), Resource: resource,
			}
			if request.Context().Err() != nil {
				event.Code = "http.request.cancelled"
				_ = logger.Warning(event)
				return
			}
			_ = logger.Info(event)
		}()

		next.ServeHTTP(capture, request)
	})
}

type responseCapture struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (capture *responseCapture) WriteHeader(status int) {
	if capture.wroteHeader {
		return
	}
	capture.status = status
	capture.wroteHeader = true
	capture.ResponseWriter.WriteHeader(status)
}

func (capture *responseCapture) Write(data []byte) (int, error) {
	if !capture.wroteHeader {
		capture.WriteHeader(http.StatusOK)
	}
	return capture.ResponseWriter.Write(data)
}

func (capture *responseCapture) Unwrap() http.ResponseWriter {
	return capture.ResponseWriter
}

func (capture *responseCapture) statusCode() int {
	if capture.status == 0 {
		return http.StatusOK
	}
	return capture.status
}

func readinessFrom(snapshot contracts.OperationalStatus) contracts.ReadinessStatus {
	return contracts.ReadinessStatus{
		Ready:                    !snapshot.Maintenance && snapshot.HistoryDatabase.State == contracts.DependencyAvailable && snapshot.ConfigurationState != contracts.ConfigurationUnavailable,
		Maintenance:              snapshot.Maintenance,
		ConfigurationState:       snapshot.ConfigurationState,
		HistoryDatabaseAvailable: snapshot.HistoryDatabase.State == contracts.DependencyAvailable,
	}
}

func allowReadMethod(writer http.ResponseWriter, request *http.Request, now func() time.Time) bool {
	if request.Method == http.MethodGet || request.Method == http.MethodHead {
		return true
	}
	writer.Header().Set("Allow", "GET, HEAD")
	writeAPIError(writer, request, http.StatusMethodNotAllowed, now, domain.ErrorValidationFailed, "errors.methodNotAllowed", "The request method is not allowed.")
	return false
}

func writeData[T any](writer http.ResponseWriter, request *http.Request, status int, now func() time.Time, data T) {
	requestID, _ := RequestIDFromContext(request.Context())
	envelope := contracts.Envelope[T]{
		APIVersion: contracts.APIVersion, RequestID: requestID, GeneratedAt: now().UTC(), Data: &data,
	}
	writeJSON(writer, request, status, envelope)
}

func writeAPIError(
	writer http.ResponseWriter,
	request *http.Request,
	status int,
	now func() time.Time,
	code domain.ErrorCode,
	messageKey string,
	fallback string,
) {
	requestID, _ := RequestIDFromContext(request.Context())
	apiError := contracts.APIError{
		Code: code, MessageKey: messageKey, FallbackMessage: fallback, FieldErrors: []contracts.FieldError{},
	}
	envelope := contracts.Envelope[any]{
		APIVersion: contracts.APIVersion, RequestID: requestID, GeneratedAt: now().UTC(), Error: &apiError,
	}
	writeJSON(writer, request, status, envelope)
}

func writeInternalError(writer http.ResponseWriter, request *http.Request, now func() time.Time) {
	writeAPIError(writer, request, http.StatusInternalServerError, now, domain.ErrorInternal, "errors.internal", "An internal error occurred.")
}

func writeJSON(writer http.ResponseWriter, request *http.Request, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(status)
	if request.Method == http.MethodHead {
		return
	}
	_ = json.NewEncoder(writer).Encode(payload)
}

func removeForwardedHeaders(header http.Header) {
	for name := range header {
		canonical := http.CanonicalHeaderKey(name)
		if canonical == "Forwarded" || canonical == "X-Real-Ip" || strings.HasPrefix(canonical, "X-Forwarded-") {
			header.Del(name)
		}
	}
}

func remoteResource(remoteAddress string) *observability.ResourceIdentity {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err != nil || net.ParseIP(host) == nil {
		return nil
	}
	return &observability.ResourceIdentity{Kind: "client", ID: host}
}

func newRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err == nil {
		return hex.EncodeToString(bytes)
	}
	return "fallback-" + time.Now().UTC().Format("20060102t150405.000000000") + "-" +
		strconv.FormatUint(fallbackRequestID.Add(1), 10)
}

func unavailableOperationalStatus() contracts.OperationalStatus {
	return contracts.OperationalStatus{
		State:              contracts.OperationalNotReady,
		ConfigurationState: contracts.ConfigurationUnavailable,
		HistoryDatabase:    contracts.DatabaseOperationalStatus{State: contracts.DependencyUnavailable},
		AuditDatabase:      contracts.DatabaseOperationalStatus{State: contracts.DependencyUnavailable},
		Collection:         contracts.CollectionOperationalStatus{State: contracts.DependencyNotStarted},
		BackupState:        contracts.DependencyNotImplemented,
		DockerConnectivity: contracts.DependencyNotChecked,
	}
}

func NewServer(address string, settings config.ServerSettings, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: time.Duration(settings.ReadHeaderTimeoutSecs) * time.Second,
		ReadTimeout:       time.Duration(settings.ReadTimeoutSecs) * time.Second,
		WriteTimeout:      time.Duration(settings.WriteTimeoutSecs) * time.Second,
		IdleTimeout:       time.Duration(settings.IdleTimeoutSecs) * time.Second,
		MaxHeaderBytes:    MaxHeaderBytes,
	}
}
