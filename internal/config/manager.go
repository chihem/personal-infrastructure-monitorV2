package config

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
)

const (
	defaultDebounce = 250 * time.Millisecond

	messageSettingsInvalid     = "configuration.settings_invalid"
	messageSettingsUnavailable = "configuration.settings_unavailable"
)

type State string

const (
	StateValid         State = "valid"
	StateUsingPrevious State = "using_previous"
	StateUnavailable   State = "unavailable"
)

type Source string

const (
	SourceActive    Source = "active"
	SourceLastValid Source = "last_valid"
	SourceDefaults  Source = "defaults"
)

type ReloadOutcome string

const (
	ReloadApplied      ReloadOutcome = "applied"
	ReloadRejected     ReloadOutcome = "rejected"
	ReloadRecovered    ReloadOutcome = "recovered"
	ReloadUsedDefaults ReloadOutcome = "used_defaults"
	ReloadUnchanged    ReloadOutcome = "unchanged"
	ReloadUnavailable  ReloadOutcome = "unavailable"
	ReloadWatchError   ReloadOutcome = "watch_error"
)

type Paths struct {
	Active    string
	LastValid string
}

type Status struct {
	State      State
	Source     Source
	LoadedAt   time.Time
	ErrorCode  domain.ErrorCode
	MessageKey string
}

type ReloadEvent struct {
	Outcome    ReloadOutcome
	OccurredAt time.Time
	State      State
	Source     Source
	ErrorCode  domain.ErrorCode
	MessageKey string
}

type EventSink func(ReloadEvent)

type Option func(*Manager)

func WithDebounce(duration time.Duration) Option {
	return func(manager *Manager) {
		if duration > 0 {
			manager.debounce = duration
		}
	}
}

func WithClock(clock func() time.Time) Option {
	return func(manager *Manager) {
		if clock != nil {
			manager.now = clock
		}
	}
}

func WithEventSink(sink EventSink) Option {
	return func(manager *Manager) {
		if sink != nil {
			manager.sink = sink
		}
	}
}

type Manager struct {
	paths    Paths
	debounce time.Duration
	now      func() time.Time
	sink     EventSink

	settings atomic.Pointer[Settings]
	status   atomic.Pointer[Status]

	reloadMu sync.Mutex
	hashSet  bool
	lastHash [sha256.Size]byte

	watchMu  sync.Mutex
	watching bool
}

func NewManager(paths Paths, options ...Option) (*Manager, error) {
	if paths.Active == "" {
		return nil, errors.New("active settings path is required")
	}
	if paths.LastValid == "" {
		return nil, errors.New("last-valid settings path is required")
	}
	if filepath.Clean(paths.Active) == filepath.Clean(paths.LastValid) {
		return nil, errors.New("active and last-valid settings paths must differ")
	}

	manager := &Manager{
		paths:    paths,
		debounce: defaultDebounce,
		now:      time.Now,
		sink:     func(ReloadEvent) {},
	}
	for _, option := range options {
		option(manager)
	}
	return manager, nil
}

// Load initializes the manager from the active file, a last-valid copy, or
// defaults on a first start where neither file exists.
func (manager *Manager) Load() error {
	manager.reloadMu.Lock()
	defer manager.reloadMu.Unlock()

	now := manager.now().UTC()
	activeData, activeReadErr := readSettingsFile(manager.paths.Active)
	if activeReadErr == nil {
		manager.rememberHash(activeData)
		candidate, parseErr := Parse(activeData)
		if parseErr == nil {
			if persistErr := writeLastValid(manager.paths.LastValid, candidate); persistErr == nil {
				manager.commit(candidate, Status{State: StateValid, Source: SourceActive, LoadedAt: now})
				manager.publish(ReloadEvent{Outcome: ReloadApplied, OccurredAt: now, State: StateValid, Source: SourceActive})
				return nil
			} else {
				activeReadErr = fmt.Errorf("persist active settings as last-valid: %w", persistErr)
			}
		} else {
			activeReadErr = parseErr
		}
	}

	lastValidData, lastValidErr := readLastValidSettingsFile(manager.paths.LastValid)
	if lastValidErr == nil {
		candidate, parseErr := Parse(lastValidData)
		if parseErr == nil {
			status := invalidStatus(StateUsingPrevious, SourceLastValid, now)
			manager.commit(candidate, status)
			manager.publish(eventFromStatus(ReloadRecovered, now, status))
			return nil
		}
		lastValidErr = parseErr
	}

	if errors.Is(activeReadErr, os.ErrNotExist) && errors.Is(lastValidErr, os.ErrNotExist) {
		candidate := Defaults()
		if err := writeLastValid(manager.paths.LastValid, candidate); err != nil {
			return manager.failLoad(now, fmt.Errorf("persist default settings as last-valid: %w", err))
		}
		manager.commit(candidate, Status{State: StateValid, Source: SourceDefaults, LoadedAt: now})
		manager.publish(ReloadEvent{Outcome: ReloadUsedDefaults, OccurredAt: now, State: StateValid, Source: SourceDefaults})
		return nil
	}

	return manager.failLoad(now, fmt.Errorf("no usable settings: active: %v; last-valid: %v", activeReadErr, lastValidErr))
}

// Reload applies the active file only when its bytes changed since the last
// attempted load. Invalid candidates leave the existing snapshot untouched.
func (manager *Manager) Reload() (ReloadEvent, error) {
	manager.reloadMu.Lock()
	defer manager.reloadMu.Unlock()

	now := manager.now().UTC()
	data, err := readSettingsFile(manager.paths.Active)
	if err != nil {
		return manager.rejectReload(now, err)
	}
	hash := sha256.Sum256(data)
	if manager.hashSet && hash == manager.lastHash {
		status := manager.Status()
		return eventFromStatus(ReloadUnchanged, now, status), nil
	}

	manager.hashSet = true
	manager.lastHash = hash
	candidate, err := Parse(data)
	if err != nil {
		return manager.rejectReload(now, err)
	}
	if err := writeLastValid(manager.paths.LastValid, candidate); err != nil {
		// A persistence failure is retryable even when file contents do not change.
		manager.hashSet = false
		return manager.rejectReload(now, fmt.Errorf("persist last-valid settings: %w", err))
	}

	status := Status{State: StateValid, Source: SourceActive, LoadedAt: now}
	manager.commit(candidate, status)
	event := eventFromStatus(ReloadApplied, now, status)
	manager.publish(event)
	return event, nil
}

func (manager *Manager) Current() (Settings, bool) {
	settings := manager.settings.Load()
	if settings == nil {
		return Settings{}, false
	}
	return settings.clone(), true
}

func (manager *Manager) Status() Status {
	status := manager.status.Load()
	if status == nil {
		return Status{State: StateUnavailable, ErrorCode: domain.ErrorSettingsInvalid, MessageKey: messageSettingsUnavailable}
	}
	return *status
}

func (manager *Manager) commit(settings Settings, status Status) {
	candidate := settings.clone()
	manager.settings.Store(&candidate)
	manager.status.Store(&status)
}

func (manager *Manager) rejectReload(now time.Time, err error) (ReloadEvent, error) {
	status := invalidStatus(StateUnavailable, "", now)
	if _, ok := manager.Current(); ok {
		previous := manager.Status()
		status.State = StateUsingPrevious
		status.Source = SourceLastValid
		status.LoadedAt = previous.LoadedAt
	}
	manager.status.Store(&status)
	event := eventFromStatus(ReloadRejected, now, status)
	manager.publish(event)
	return event, fmt.Errorf("reload settings: %w", err)
}

func (manager *Manager) failLoad(now time.Time, err error) error {
	status := invalidStatus(StateUnavailable, "", now)
	status.MessageKey = messageSettingsUnavailable
	manager.status.Store(&status)
	manager.publish(eventFromStatus(ReloadUnavailable, now, status))
	return err
}

func (manager *Manager) rememberHash(data []byte) {
	manager.hashSet = true
	manager.lastHash = sha256.Sum256(data)
}

func (manager *Manager) publish(event ReloadEvent) {
	manager.sink(event)
}

func invalidStatus(state State, source Source, loadedAt time.Time) Status {
	return Status{
		State:      state,
		Source:     source,
		LoadedAt:   loadedAt,
		ErrorCode:  domain.ErrorSettingsInvalid,
		MessageKey: messageSettingsInvalid,
	}
}

func eventFromStatus(outcome ReloadOutcome, occurredAt time.Time, status Status) ReloadEvent {
	return ReloadEvent{
		Outcome:    outcome,
		OccurredAt: occurredAt,
		State:      status.State,
		Source:     status.Source,
		ErrorCode:  status.ErrorCode,
		MessageKey: status.MessageKey,
	}
}

func readSettingsFile(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("settings path is not a regular file")
	}
	if info.Size() > MaxSettingsBytes {
		return nil, fmt.Errorf("settings file exceeds %d bytes", MaxSettingsBytes)
	}
	return os.ReadFile(path)
}

func readLastValidSettingsFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("last-valid settings path must not be a symbolic link")
	}
	return readSettingsFile(path)
}

func writeLastValid(path string, settings Settings) error {
	data, err := encode(settings)
	if err != nil {
		return err
	}
	return writeProtectedFile(path, data)
}

func writeProtectedFile(path string, data []byte) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create last-valid directory: %w", err)
	}
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return errors.New("last-valid settings path must not be a symbolic link")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect last-valid settings path: %w", err)
	}

	temporary, err := os.CreateTemp(directory, ".last-valid-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary last-valid settings: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("protect temporary last-valid settings: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return fmt.Errorf("write temporary last-valid settings: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync temporary last-valid settings: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary last-valid settings: %w", err)
	}
	if err := replaceFile(temporaryPath, path); err != nil {
		return fmt.Errorf("publish last-valid settings: %w", err)
	}
	return nil
}
