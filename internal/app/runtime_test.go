package app

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/api"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/config"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/observability"
)

func TestRunStartsServesAndStops(t *testing.T) {
	root := t.TempDir()
	settingsDirectory := filepath.Join(root, "etc")
	if err := os.MkdirAll(settingsDirectory, 0o700); err != nil {
		t.Fatalf("create settings directory: %v", err)
	}

	listenerReady := make(chan net.Listener, 1)
	options := Options{
		SettingsPath:    filepath.Join(settingsDirectory, "settings.toml"),
		LastValidPath:   filepath.Join(root, "state", "last-valid-settings.toml"),
		HistoryPath:     filepath.Join(root, "state", "history.db"),
		AuditPath:       filepath.Join(root, "state", "audit.db"),
		ShutdownTimeout: 2 * time.Second,
		Logger:          observability.Discard(),
		listen: func(context.Context, int) (net.Listener, error) {
			listener, err := net.Listen("tcp4", "127.0.0.1:0")
			if err == nil {
				listenerReady <- listener
			}
			return listener, err
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- Run(ctx, options) }()
	listener := waitFor(t, listenerReady, "listener")

	client := &http.Client{Timeout: 2 * time.Second}
	baseURL := "http://" + listener.Addr().String()
	response := requestUntilReady(t, client, baseURL+"/")
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatalf("read embedded UI: %v", err)
	}
	if response.StatusCode != http.StatusOK || len(body) == 0 {
		t.Fatalf("UI response status = %d, body length = %d", response.StatusCode, len(body))
	}

	healthResponse := requestUntilReady(t, client, baseURL+"/api/v1/health")
	if healthResponse.StatusCode != http.StatusOK {
		healthResponse.Body.Close()
		t.Fatalf("health status = %d", healthResponse.StatusCode)
	}
	healthResponse.Body.Close()

	cpuResponse := requestUntilReady(t, client, baseURL+"/api/v1/cpu")
	cpuBody, err := io.ReadAll(cpuResponse.Body)
	cpuResponse.Body.Close()
	if err != nil {
		t.Fatalf("read CPU response: %v", err)
	}
	if cpuResponse.StatusCode != http.StatusOK || !strings.Contains(string(cpuBody), `"freshness":{"state":"unavailable"`) {
		t.Fatalf("CPU response status = %d, body = %s", cpuResponse.StatusCode, cpuBody)
	}

	cancel()
	if err := waitFor(t, result, "Run result"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	for _, databasePath := range []string{options.HistoryPath, options.AuditPath} {
		renamed := databasePath + ".closed"
		if err := os.Rename(databasePath, renamed); err != nil {
			t.Fatalf("database %s was not released: %v", databasePath, err)
		}
	}
}

func TestRuntimeGracefullyDrainsRequestBeforeClosingResources(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	handler := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		close(requestStarted)
		<-releaseRequest
		writer.WriteHeader(http.StatusNoContent)
	})
	server := api.NewServer(listener.Addr().String(), config.Defaults().Server, handler)
	closer := &countingCloser{}
	runtime := &runtime{
		server:          server,
		listener:        listener,
		closers:         []io.Closer{closer},
		stopWatcher:     func() {},
		shutdownTimeout: 2 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	serveResult := make(chan error, 1)
	go func() { serveResult <- runtime.serve(ctx) }()
	requestResult := make(chan error, 1)
	go func() {
		response, requestErr := http.Get("http://" + listener.Addr().String())
		if requestErr == nil {
			response.Body.Close()
		}
		requestResult <- requestErr
	}()
	waitFor(t, requestStarted, "request start")
	cancel()

	select {
	case err := <-serveResult:
		t.Fatalf("runtime stopped before request drained: %v", err)
	case <-time.After(75 * time.Millisecond):
	}
	if closer.count.Load() != 0 {
		t.Fatal("resources closed while request was active")
	}

	close(releaseRequest)
	if err := waitFor(t, requestResult, "request result"); err != nil {
		t.Fatalf("request error = %v", err)
	}
	if err := waitFor(t, serveResult, "serve result"); err != nil {
		t.Fatalf("serve error = %v", err)
	}
	if closer.count.Load() != 1 {
		t.Fatalf("resource close count = %d, want 1", closer.count.Load())
	}
}

func TestRunContinuesWhenAuditDatabaseIsUnavailable(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc"), 0o700); err != nil {
		t.Fatalf("create settings directory: %v", err)
	}

	listenerReady := make(chan net.Listener, 1)
	options := Options{
		SettingsPath:    filepath.Join(root, "etc", "settings.toml"),
		LastValidPath:   filepath.Join(root, "state", "last-valid-settings.toml"),
		HistoryPath:     filepath.Join(root, "state", "history.db"),
		AuditPath:       filepath.Join(root, "state", "audit.db"),
		ShutdownTimeout: 2 * time.Second,
		Logger:          observability.Discard(),
		listen: func(context.Context, int) (net.Listener, error) {
			listener, err := net.Listen("tcp4", "127.0.0.1:0")
			if err == nil {
				listenerReady <- listener
			}
			return listener, err
		},
		openAudit: func(context.Context, string) (io.Closer, error) { return nil, errors.New("audit unavailable") },
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- Run(ctx, options) }()
	listener := waitFor(t, listenerReady, "listener")

	client := &http.Client{Timeout: 2 * time.Second}
	response := requestUntilReady(t, client, "http://"+listener.Addr().String()+"/api/v1/status")
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatalf("read health response: %v", err)
	}
	if !strings.Contains(string(body), `"state":"degraded"`) || !strings.Contains(string(body), `"auditDatabase":{"state":"unavailable"`) {
		t.Fatalf("health response = %s", body)
	}

	cancel()
	if err := waitFor(t, result, "Run result"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if err := os.Rename(options.HistoryPath, options.HistoryPath+".closed"); err != nil {
		t.Fatalf("history database was not released: %v", err)
	}
}

type countingCloser struct {
	count atomic.Int32
}

func (closer *countingCloser) Close() error {
	closer.count.Add(1)
	return nil
}

func requestUntilReady(t *testing.T, client *http.Client, url string) *http.Response {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		response, err := client.Get(url)
		if err == nil {
			return response
		}
		if time.Now().After(deadline) {
			t.Fatalf("GET %s did not become ready: %v", url, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func waitFor[T any](t *testing.T, channel <-chan T, label string) T {
	t.Helper()
	select {
	case value := <-channel:
		return value
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for %s", label)
		var zero T
		return zero
	}
}
