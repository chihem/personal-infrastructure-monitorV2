package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDefaultsAreApprovedAndValid(t *testing.T) {
	t.Parallel()

	settings := Defaults()
	if err := settings.Validate(); err != nil {
		t.Fatalf("Defaults().Validate() error = %v", err)
	}

	if settings.Thresholds.CPU.WarningPercent != 85 || settings.Thresholds.CPU.CriticalPercent != 95 {
		t.Fatalf("CPU thresholds = %+v", settings.Thresholds.CPU)
	}
	if settings.Thresholds.RAM.WarningPercent != 85 || settings.Thresholds.RAM.CriticalPercent != 95 {
		t.Fatalf("RAM thresholds = %+v", settings.Thresholds.RAM)
	}
	if settings.Thresholds.Filesystem.WarningPercent != 90 || settings.Thresholds.Filesystem.CriticalPercent != 95 {
		t.Fatalf("filesystem thresholds = %+v", settings.Thresholds.Filesystem)
	}
	if settings.Collection.IntervalSeconds != 60 || settings.Collection.StaleAfterSecs != 120 || settings.Collection.RetentionDays != 14 {
		t.Fatalf("collection settings = %+v", settings.Collection)
	}
	if settings.Storage.MaxBytes != 5*1024*1024*1024 {
		t.Fatalf("storage maximum = %d", settings.Storage.MaxBytes)
	}
	if settings.Logs.BrowserMaxBytes != 5*1024*1024 {
		t.Fatalf("browser log maximum = %d", settings.Logs.BrowserMaxBytes)
	}
}

func TestParseOverlaysDefaults(t *testing.T) {
	t.Parallel()

	settings, err := Parse([]byte(`
[thresholds.cpu]
warning_percent = 80
critical_percent = 92

[server]
port = 9123

[display]
default_language = "fr"
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if settings.Thresholds.CPU.WarningPercent != 80 || settings.Thresholds.CPU.CriticalPercent != 92 {
		t.Fatalf("CPU thresholds = %+v", settings.Thresholds.CPU)
	}
	if settings.Thresholds.RAM != Defaults().Thresholds.RAM {
		t.Fatalf("RAM defaults changed: %+v", settings.Thresholds.RAM)
	}
	if settings.Server.Port != 9123 {
		t.Fatalf("server port = %d", settings.Server.Port)
	}
	if settings.Display.DefaultLanguage != "fr" {
		t.Fatalf("default language = %q", settings.Display.DefaultLanguage)
	}
}

func TestThresholdBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		threshold PercentageThreshold
		wantError bool
	}{
		{name: "valid lower boundary", threshold: PercentageThreshold{WarningPercent: 0.1, CriticalPercent: 0.2}},
		{name: "valid upper boundary", threshold: PercentageThreshold{WarningPercent: 99.9, CriticalPercent: 100}},
		{name: "zero warning", threshold: PercentageThreshold{WarningPercent: 0, CriticalPercent: 95}, wantError: true},
		{name: "warning at one hundred", threshold: PercentageThreshold{WarningPercent: 100, CriticalPercent: 100}, wantError: true},
		{name: "critical above one hundred", threshold: PercentageThreshold{WarningPercent: 85, CriticalPercent: 100.1}, wantError: true},
		{name: "equal values", threshold: PercentageThreshold{WarningPercent: 95, CriticalPercent: 95}, wantError: true},
		{name: "warning above critical", threshold: PercentageThreshold{WarningPercent: 96, CriticalPercent: 95}, wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			settings := Defaults()
			settings.Thresholds.CPU = test.threshold
			err := settings.Validate()
			if (err != nil) != test.wantError {
				t.Fatalf("Validate() error = %v, wantError = %v", err, test.wantError)
			}
		})
	}
}

func TestValidationRejectsChangedConfirmedLimits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		change func(*Settings)
	}{
		{name: "collection interval", change: func(settings *Settings) { settings.Collection.IntervalSeconds = 30 }},
		{name: "stale time", change: func(settings *Settings) { settings.Collection.StaleAfterSecs = 180 }},
		{name: "history retention", change: func(settings *Settings) { settings.Collection.RetentionDays = 7 }},
		{name: "backup interval", change: func(settings *Settings) { settings.Backup.IntervalHours = 24 }},
		{name: "backup retention", change: func(settings *Settings) { settings.Backup.RetentionDays = 7 }},
		{name: "storage ceiling", change: func(settings *Settings) { settings.Storage.MaxBytes++ }},
		{name: "timezone", change: func(settings *Settings) { settings.Display.Timezone = "UTC" }},
		{name: "languages missing French", change: func(settings *Settings) { settings.Display.SupportedLanguages = []string{"en"} }},
		{name: "extra language", change: func(settings *Settings) { settings.Display.SupportedLanguages = []string{"en", "fr", "de"} }},
		{name: "initial log lines", change: func(settings *Settings) { settings.Logs.InitialLines = 200 }},
		{name: "browser log ceiling", change: func(settings *Settings) { settings.Logs.BrowserMaxBytes++ }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			settings := Defaults()
			test.change(&settings)
			var validationError *ValidationError
			if err := settings.Validate(); !errors.As(err, &validationError) {
				t.Fatalf("Validate() error = %v, want *ValidationError", err)
			}
		})
	}
}

func TestServerBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		change    func(*Settings)
		wantError bool
	}{
		{name: "lowest non-privileged port", change: func(settings *Settings) { settings.Server.Port = 1024 }},
		{name: "highest port", change: func(settings *Settings) { settings.Server.Port = 65535 }},
		{name: "privileged port", change: func(settings *Settings) { settings.Server.Port = 443 }, wantError: true},
		{name: "port too high", change: func(settings *Settings) { settings.Server.Port = 65536 }, wantError: true},
		{name: "header timeout too low", change: func(settings *Settings) { settings.Server.ReadHeaderTimeoutSecs = 0 }, wantError: true},
		{name: "idle timeout too high", change: func(settings *Settings) { settings.Server.IdleTimeoutSecs = 301 }, wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			settings := Defaults()
			test.change(&settings)
			err := settings.Validate()
			if (err != nil) != test.wantError {
				t.Fatalf("Validate() error = %v, wantError = %v", err, test.wantError)
			}
		})
	}
}

func TestParseRejectsMalformedUnknownAndDangerousFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		toml string
	}{
		{name: "malformed", toml: `[server`},
		{name: "unknown", toml: `surprise = true`},
		{name: "shell command", toml: `shell_command = "rm -rf /"`},
		{name: "Docker endpoint", toml: `docker_endpoint = "tcp://127.0.0.1:2375"`},
		{name: "bind address", toml: "[server]\nbind_address = \"0.0.0.0\""},
		{name: "network interface", toml: "[server]\ninterface = \"eth0\""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Parse([]byte(test.toml)); err == nil {
				t.Fatal("Parse() error = nil")
			}
		})
	}
}

func TestParseRejectsOversizedFile(t *testing.T) {
	t.Parallel()

	data := []byte(strings.Repeat("#", MaxSettingsBytes+1))
	if _, err := Parse(data); err == nil {
		t.Fatal("Parse() error = nil")
	}
}

func TestEncodeRoundTrip(t *testing.T) {
	t.Parallel()

	want := Defaults()
	want.Thresholds.Filesystem.WarningPercent = 88
	want.Server.Port = 9999

	data, err := encode(want)
	if err != nil {
		t.Fatalf("encode() error = %v", err)
	}
	got, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse(encoded) error = %v\n%s", err, data)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip = %+v, want %+v", got, want)
	}
}

func TestExampleSettingsAreValid(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "..", "configs", "settings.example.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	settings, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse(example) error = %v", err)
	}
	if !reflect.DeepEqual(settings, Defaults()) {
		t.Fatalf("example settings = %+v, want defaults %+v", settings, Defaults())
	}
}
