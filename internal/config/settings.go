package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/pelletier/go-toml/v2"
)

const (
	SchemaVersion = 1

	MaxSettingsBytes = 64 * 1024

	DefaultHTTPPort                     = 8788
	DefaultCollectionIntervalSecs       = 60
	DefaultStaleAfterSecs               = 120
	DefaultHistoryRetentionDays         = 14
	DefaultBackupIntervalHours          = 48
	DefaultBackupRetentionDays          = 14
	DefaultStorageMaxBytes        int64 = 5 * 1024 * 1024 * 1024
	DefaultLogInitialLines              = 100
	DefaultBrowserLogMaxBytes     int64 = 5 * 1024 * 1024
)

type Settings struct {
	Version    int                `toml:"version"`
	Thresholds ThresholdSettings  `toml:"thresholds"`
	Collection CollectionSettings `toml:"collection"`
	Backup     BackupSettings     `toml:"backup"`
	Storage    StorageSettings    `toml:"storage"`
	Server     ServerSettings     `toml:"server"`
	Display    DisplaySettings    `toml:"display"`
	Logs       LogSettings        `toml:"logs"`
}

type ThresholdSettings struct {
	CPU        PercentageThreshold `toml:"cpu"`
	RAM        PercentageThreshold `toml:"ram"`
	Filesystem PercentageThreshold `toml:"filesystem"`
}

type PercentageThreshold struct {
	WarningPercent  float64 `toml:"warning_percent"`
	CriticalPercent float64 `toml:"critical_percent"`
}

type CollectionSettings struct {
	IntervalSeconds int `toml:"interval_seconds"`
	StaleAfterSecs  int `toml:"stale_after_seconds"`
	RetentionDays   int `toml:"retention_days"`
}

type BackupSettings struct {
	IntervalHours int `toml:"interval_hours"`
	RetentionDays int `toml:"retention_days"`
}

type StorageSettings struct {
	MaxBytes int64 `toml:"max_bytes"`
}

type ServerSettings struct {
	Port                  int `toml:"port"`
	ReadHeaderTimeoutSecs int `toml:"read_header_timeout_seconds"`
	ReadTimeoutSecs       int `toml:"read_timeout_seconds"`
	WriteTimeoutSecs      int `toml:"write_timeout_seconds"`
	IdleTimeoutSecs       int `toml:"idle_timeout_seconds"`
}

type DisplaySettings struct {
	Timezone           string   `toml:"timezone"`
	SupportedLanguages []string `toml:"supported_languages"`
	DefaultLanguage    string   `toml:"default_language"`
}

type LogSettings struct {
	InitialLines    int   `toml:"initial_lines"`
	BrowserMaxBytes int64 `toml:"browser_max_bytes"`
}

func Defaults() Settings {
	return Settings{
		Version: SchemaVersion,
		Thresholds: ThresholdSettings{
			CPU:        PercentageThreshold{WarningPercent: 85, CriticalPercent: 95},
			RAM:        PercentageThreshold{WarningPercent: 85, CriticalPercent: 95},
			Filesystem: PercentageThreshold{WarningPercent: 90, CriticalPercent: 95},
		},
		Collection: CollectionSettings{
			IntervalSeconds: DefaultCollectionIntervalSecs,
			StaleAfterSecs:  DefaultStaleAfterSecs,
			RetentionDays:   DefaultHistoryRetentionDays,
		},
		Backup: BackupSettings{
			IntervalHours: DefaultBackupIntervalHours,
			RetentionDays: DefaultBackupRetentionDays,
		},
		Storage: StorageSettings{MaxBytes: DefaultStorageMaxBytes},
		Server: ServerSettings{
			Port:                  DefaultHTTPPort,
			ReadHeaderTimeoutSecs: 5,
			ReadTimeoutSecs:       15,
			WriteTimeoutSecs:      30,
			IdleTimeoutSecs:       60,
		},
		Display: DisplaySettings{
			Timezone:           "Africa/Tunis",
			SupportedLanguages: []string{"en", "fr"},
			DefaultLanguage:    "en",
		},
		Logs: LogSettings{
			InitialLines:    DefaultLogInitialLines,
			BrowserMaxBytes: DefaultBrowserLogMaxBytes,
		},
	}
}

// Parse overlays a TOML document on the safe defaults and validates the whole
// resulting candidate. Unknown fields are rejected instead of being ignored.
func Parse(data []byte) (Settings, error) {
	if len(data) > MaxSettingsBytes {
		return Settings{}, fmt.Errorf("settings file exceeds %d bytes", MaxSettingsBytes)
	}

	candidate := Defaults()
	decoder := toml.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&candidate); err != nil && !errors.Is(err, io.EOF) {
		return Settings{}, fmt.Errorf("parse settings TOML: %w", err)
	}
	if err := candidate.Validate(); err != nil {
		return Settings{}, err
	}
	return candidate.clone(), nil
}

func encode(settings Settings) ([]byte, error) {
	data, err := toml.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("encode validated settings: %w", err)
	}
	return data, nil
}

func (settings Settings) clone() Settings {
	settings.Display.SupportedLanguages = append(
		[]string(nil),
		settings.Display.SupportedLanguages...,
	)
	return settings
}
