package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/api/contracts"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/config"
)

const (
	MaxRequestBodyBytes int64 = 1 << 20
	MaxHeaderBytes            = 16 << 10
)

type FoundationHealth struct {
	Maintenance              bool   `json:"maintenance"`
	HistoryDatabaseAvailable bool   `json:"historyDatabaseAvailable"`
	AuditDatabaseAvailable   bool   `json:"auditDatabaseAvailable"`
	ConfigurationState       string `json:"configurationState"`
}

type HealthProvider func() FoundationHealth

type healthResponse struct {
	APIVersion string           `json:"apiVersion"`
	Status     string           `json:"status"`
	Data       FoundationHealth `json:"data"`
}

// NewHandler combines the versioned API with the embedded web application.
func NewHandler(webHandler http.Handler, health HealthProvider) http.Handler {
	if webHandler == nil {
		webHandler = http.NotFoundHandler()
	}
	if health == nil {
		health = func() FoundationHealth { return FoundationHealth{} }
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			writer.Header().Set("Allow", "GET, HEAD")
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		snapshot := health()
		status := "ok"
		if snapshot.Maintenance {
			status = "maintenance"
		} else if !snapshot.HistoryDatabaseAvailable || !snapshot.AuditDatabaseAvailable {
			status = "degraded"
		}

		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		writer.Header().Set("Cache-Control", "no-store")
		writer.WriteHeader(http.StatusOK)
		if request.Method == http.MethodHead {
			return
		}
		_ = json.NewEncoder(writer).Encode(healthResponse{
			APIVersion: contracts.APIVersion,
			Status:     status,
			Data:       snapshot,
		})
	})
	mux.HandleFunc("/api/", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		writer.Header().Set("Cache-Control", "no-store")
		http.Error(writer, `{"apiVersion":"v1","error":{"code":"not_found"}}`, http.StatusNotFound)
	})
	mux.Handle("/", webHandler)

	return requestSafeguards(mux)
}

func requestSafeguards(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Security-Policy", "default-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'; object-src 'none'")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("X-Frame-Options", "DENY")

		removeForwardedHeaders(request.Header)
		if request.ContentLength > MaxRequestBodyBytes {
			http.Error(writer, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		request.Body = http.MaxBytesReader(writer, request.Body, MaxRequestBodyBytes)
		next.ServeHTTP(writer, request)
	})
}

func removeForwardedHeaders(header http.Header) {
	for name := range header {
		canonical := http.CanonicalHeaderKey(name)
		if canonical == "Forwarded" || canonical == "X-Real-Ip" || strings.HasPrefix(canonical, "X-Forwarded-") {
			header.Del(name)
		}
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
