//go:build !windows

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManagerRejectsSymlinkedLastValidFile(t *testing.T) {
	t.Parallel()

	manager, paths := newTestManager(t)
	writeSettings(t, paths.Active, Defaults())
	target := filepath.Join(t.TempDir(), "target.toml")
	writeSettings(t, target, Defaults())
	if err := os.MkdirAll(filepath.Dir(paths.LastValid), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.Symlink(target, paths.LastValid); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	if err := manager.Load(); err == nil {
		t.Fatal("Load() error = nil")
	}
	if manager.Status().State != StateUnavailable {
		t.Fatalf("Status() = %+v", manager.Status())
	}
}
