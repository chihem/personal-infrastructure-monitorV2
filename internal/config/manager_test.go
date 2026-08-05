package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestManagerUsesDefaultsOnFirstStart(t *testing.T) {
	t.Parallel()

	manager, paths := newTestManager(t)
	if err := manager.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	settings, ok := manager.Current()
	if !ok {
		t.Fatal("Current() available = false")
	}
	if settings.Server.Port != DefaultHTTPPort {
		t.Fatalf("port = %d, want %d", settings.Server.Port, DefaultHTTPPort)
	}
	status := manager.Status()
	if status.State != StateValid || status.Source != SourceDefaults {
		t.Fatalf("Status() = %+v", status)
	}
	assertProtectedFile(t, paths.LastValid)
}

func TestManagerValidReloadIsAtomicAndPersisted(t *testing.T) {
	t.Parallel()

	manager, paths := newTestManager(t)
	writeSettings(t, paths.Active, Defaults())
	if err := manager.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	updated := Defaults()
	updated.Thresholds.CPU.WarningPercent = 82
	updated.Server.Port = 9123
	writeSettings(t, paths.Active, updated)
	event, err := manager.Reload()
	if err != nil {
		t.Fatalf("Reload() error = %v", err)
	}
	if event.Outcome != ReloadApplied {
		t.Fatalf("Reload() outcome = %q", event.Outcome)
	}
	got, ok := manager.Current()
	if !ok || got.Thresholds.CPU.WarningPercent != 82 || got.Server.Port != 9123 {
		t.Fatalf("Current() = %+v, available = %v", got, ok)
	}

	lastValidData, err := os.ReadFile(paths.LastValid)
	if err != nil {
		t.Fatalf("ReadFile(last-valid) error = %v", err)
	}
	lastValid, err := Parse(lastValidData)
	if err != nil {
		t.Fatalf("Parse(last-valid) error = %v", err)
	}
	if lastValid.Server.Port != 9123 {
		t.Fatalf("last-valid port = %d", lastValid.Server.Port)
	}
	assertProtectedFile(t, paths.LastValid)
}

func TestManagerInvalidReloadKeepsEntirePreviousSnapshot(t *testing.T) {
	t.Parallel()

	manager, paths := newTestManager(t)
	initial := Defaults()
	initial.Server.Port = 9000
	writeSettings(t, paths.Active, initial)
	if err := manager.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	invalid := initial
	invalid.Server.Port = 9001
	invalid.Collection.RetentionDays = 1
	writeSettings(t, paths.Active, invalid)
	event, err := manager.Reload()
	if err == nil {
		t.Fatal("Reload() error = nil")
	}
	if event.Outcome != ReloadRejected || event.State != StateUsingPrevious {
		t.Fatalf("Reload() event = %+v", event)
	}
	current, ok := manager.Current()
	if !ok || current.Server.Port != 9000 || current.Collection.RetentionDays != 14 {
		t.Fatalf("Current() = %+v, available = %v", current, ok)
	}
	status := manager.Status()
	if status.State != StateUsingPrevious || status.Source != SourceLastValid || status.MessageKey != messageSettingsInvalid {
		t.Fatalf("Status() = %+v", status)
	}

	validAgain := initial
	validAgain.Server.Port = 9002
	writeSettings(t, paths.Active, validAgain)
	if _, err := manager.Reload(); err != nil {
		t.Fatalf("Reload(valid again) error = %v", err)
	}
	status = manager.Status()
	if status.State != StateValid || status.ErrorCode != "" || status.MessageKey != "" {
		t.Fatalf("Status() after valid reload = %+v", status)
	}
}

func TestManagerRecoversLastValidAtStartup(t *testing.T) {
	t.Parallel()

	manager, paths := newTestManager(t)
	initial := Defaults()
	initial.Thresholds.RAM.WarningPercent = 81
	writeSettings(t, paths.Active, initial)
	if err := manager.Load(); err != nil {
		t.Fatalf("first Load() error = %v", err)
	}
	if err := os.WriteFile(paths.Active, []byte("[broken"), 0o600); err != nil {
		t.Fatalf("WriteFile(broken active) error = %v", err)
	}

	restarted, err := NewManager(paths)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	if err := restarted.Load(); err != nil {
		t.Fatalf("restarted Load() error = %v", err)
	}
	current, ok := restarted.Current()
	if !ok || current.Thresholds.RAM.WarningPercent != 81 {
		t.Fatalf("Current() = %+v, available = %v", current, ok)
	}
	status := restarted.Status()
	if status.State != StateUsingPrevious || status.Source != SourceLastValid {
		t.Fatalf("Status() = %+v", status)
	}
}

func TestManagerFailsWhenActiveAndLastValidAreInvalid(t *testing.T) {
	t.Parallel()

	manager, paths := newTestManager(t)
	if err := os.WriteFile(paths.Active, []byte("[broken"), 0o600); err != nil {
		t.Fatalf("WriteFile(active) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(paths.LastValid), 0o700); err != nil {
		t.Fatalf("MkdirAll(last-valid parent) error = %v", err)
	}
	if err := os.WriteFile(paths.LastValid, []byte("[also-broken"), 0o600); err != nil {
		t.Fatalf("WriteFile(last-valid) error = %v", err)
	}
	if err := manager.Load(); err == nil {
		t.Fatal("Load() error = nil")
	}
	if _, ok := manager.Current(); ok {
		t.Fatal("Current() available = true")
	}
	if manager.Status().State != StateUnavailable {
		t.Fatalf("Status() = %+v", manager.Status())
	}
	if manager.Status().MessageKey != messageSettingsUnavailable {
		t.Fatalf("Status() message key = %q", manager.Status().MessageKey)
	}
}

func TestManagerSuppressesUnchangedReload(t *testing.T) {
	t.Parallel()

	manager, paths := newTestManager(t)
	writeSettings(t, paths.Active, Defaults())
	if err := manager.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	event, err := manager.Reload()
	if err != nil {
		t.Fatalf("Reload() error = %v", err)
	}
	if event.Outcome != ReloadUnchanged {
		t.Fatalf("Reload() outcome = %q", event.Outcome)
	}
}

func TestManagerConcurrentReloadsExposeWholeSnapshots(t *testing.T) {
	t.Parallel()

	manager, paths := newTestManager(t)
	first := Defaults()
	first.Server.Port = 9000
	first.Thresholds.CPU.WarningPercent = 80
	writeSettings(t, paths.Active, first)
	if err := manager.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	second := Defaults()
	second.Server.Port = 9100
	second.Thresholds.CPU.WarningPercent = 81
	writeSettings(t, paths.Active, second)

	const workers = 32
	var waitGroup sync.WaitGroup
	errorsChannel := make(chan error, workers)
	for range workers {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			_, err := manager.Reload()
			if err != nil {
				errorsChannel <- err
			}
			settings, ok := manager.Current()
			if !ok {
				errorsChannel <- errors.New("settings unavailable")
				return
			}
			firstSnapshot := settings.Server.Port == 9000 && settings.Thresholds.CPU.WarningPercent == 80
			secondSnapshot := settings.Server.Port == 9100 && settings.Thresholds.CPU.WarningPercent == 81
			if !firstSnapshot && !secondSnapshot {
				errorsChannel <- errors.New("observed partial snapshot")
			}
		}()
	}
	waitGroup.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		t.Error(err)
	}
}

func TestWatcherDebouncesDuplicateFileEvents(t *testing.T) {
	manager, paths := newTestManager(t)
	writeSettings(t, paths.Active, Defaults())

	var eventMu sync.Mutex
	var events []ReloadEvent
	manager.sink = func(event ReloadEvent) {
		eventMu.Lock()
		defer eventMu.Unlock()
		events = append(events, event)
	}
	if err := manager.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	eventMu.Lock()
	events = nil
	eventMu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := manager.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	updated := Defaults()
	updated.Server.Port = 9100
	data, err := encode(updated)
	if err != nil {
		t.Fatalf("encode() error = %v", err)
	}
	for range 3 {
		if err := os.WriteFile(paths.Active, data, 0o600); err != nil {
			t.Fatalf("WriteFile(active) error = %v", err)
		}
	}

	waitFor(t, 3*time.Second, func() bool {
		settings, ok := manager.Current()
		return ok && settings.Server.Port == 9100
	})
	time.Sleep(4 * manager.debounce)

	eventMu.Lock()
	defer eventMu.Unlock()
	applied := 0
	for _, event := range events {
		if event.Outcome == ReloadApplied {
			applied++
		}
	}
	if applied != 1 {
		t.Fatalf("applied events = %d, all events = %+v", applied, events)
	}
}

func TestCurrentReturnsAnImmutableCopy(t *testing.T) {
	t.Parallel()

	manager, _ := newTestManager(t)
	if err := manager.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	copyOne, _ := manager.Current()
	copyOne.Display.SupportedLanguages[0] = "changed"
	copyTwo, _ := manager.Current()
	if copyTwo.Display.SupportedLanguages[0] != "en" {
		t.Fatalf("stored languages changed: %v", copyTwo.Display.SupportedLanguages)
	}
}

func newTestManager(t *testing.T) (*Manager, Paths) {
	t.Helper()
	directory := t.TempDir()
	paths := Paths{
		Active:    filepath.Join(directory, "settings.toml"),
		LastValid: filepath.Join(directory, "state", "last-valid-settings.toml"),
	}
	manager, err := NewManager(paths, WithDebounce(25*time.Millisecond))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	return manager, paths
}

func writeSettings(t *testing.T, path string, settings Settings) {
	t.Helper()
	data, err := encode(settings)
	if err != nil {
		t.Fatalf("encode() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func assertProtectedFile(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("permissions = %o, want 600", info.Mode().Perm())
	}
}

func waitFor(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
}
