package config

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
	"github.com/fsnotify/fsnotify"
)

// Start watches the active settings directory. Watching the directory rather
// than only the file also detects editors that save through atomic renames.
func (manager *Manager) Start(ctx context.Context) error {
	manager.watchMu.Lock()
	defer manager.watchMu.Unlock()
	if manager.watching {
		return fmt.Errorf("settings watcher is already running")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create settings watcher: %w", err)
	}
	if err := watcher.Add(filepath.Dir(manager.paths.Active)); err != nil {
		watcher.Close()
		return fmt.Errorf("watch settings directory: %w", err)
	}
	manager.watching = true

	go manager.watch(ctx, watcher)
	return nil
}

func (manager *Manager) watch(ctx context.Context, watcher *fsnotify.Watcher) {
	defer func() {
		watcher.Close()
		manager.watchMu.Lock()
		manager.watching = false
		manager.watchMu.Unlock()
	}()

	var timer *time.Timer
	var timerChannel <-chan time.Time
	stopTimer := func() {
		if timer != nil && !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}
	defer stopTimer()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if !manager.relevant(event) {
				continue
			}
			stopTimer()
			if timer == nil {
				timer = time.NewTimer(manager.debounce)
			} else {
				timer.Reset(manager.debounce)
			}
			timerChannel = timer.C
		case <-timerChannel:
			timerChannel = nil
			_, _ = manager.Reload()
		case _, ok := <-watcher.Errors:
			if !ok {
				return
			}
			status := manager.Status()
			manager.publish(ReloadEvent{
				Outcome:    ReloadWatchError,
				OccurredAt: manager.now().UTC(),
				State:      status.State,
				Source:     status.Source,
				ErrorCode:  domain.ErrorInternal,
				MessageKey: "configuration.watch_failed",
			})
		}
	}
}

func (manager *Manager) relevant(event fsnotify.Event) bool {
	if filepath.Clean(event.Name) != filepath.Clean(manager.paths.Active) {
		return false
	}
	return event.Has(fsnotify.Create) || event.Has(fsnotify.Write) ||
		event.Has(fsnotify.Rename) || event.Has(fsnotify.Remove)
}
