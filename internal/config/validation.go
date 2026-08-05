package config

import (
	"fmt"
	"math"
	"slices"
)

type ValidationError struct {
	Field   string
	Problem string
}

func (err *ValidationError) Error() string {
	return fmt.Sprintf("invalid %s: %s", err.Field, err.Problem)
}

func invalid(field, problem string) error {
	return &ValidationError{Field: field, Problem: problem}
}

func (settings Settings) Validate() error {
	if settings.Version != SchemaVersion {
		return invalid("version", fmt.Sprintf("must equal %d", SchemaVersion))
	}

	thresholds := []struct {
		name  string
		value PercentageThreshold
	}{
		{name: "thresholds.cpu", value: settings.Thresholds.CPU},
		{name: "thresholds.ram", value: settings.Thresholds.RAM},
		{name: "thresholds.filesystem", value: settings.Thresholds.Filesystem},
	}
	for _, threshold := range thresholds {
		if err := validateThreshold(threshold.name, threshold.value); err != nil {
			return err
		}
	}

	if settings.Collection.IntervalSeconds != DefaultCollectionIntervalSecs {
		return invalid("collection.interval_seconds", "must remain 60 for one-minute collection")
	}
	if settings.Collection.StaleAfterSecs != DefaultStaleAfterSecs {
		return invalid("collection.stale_after_seconds", "must remain 120 for the approved stale rule")
	}
	if settings.Collection.RetentionDays != DefaultHistoryRetentionDays {
		return invalid("collection.retention_days", "must remain 14")
	}
	if settings.Backup.IntervalHours != DefaultBackupIntervalHours {
		return invalid("backup.interval_hours", "must remain 48")
	}
	if settings.Backup.RetentionDays != DefaultBackupRetentionDays {
		return invalid("backup.retention_days", "must remain 14")
	}
	if settings.Storage.MaxBytes != DefaultStorageMaxBytes {
		return invalid("storage.max_bytes", "must remain the approved 5 GiB ceiling")
	}

	if settings.Server.Port < 1024 || settings.Server.Port > 65535 {
		return invalid("server.port", "must be a non-privileged port from 1024 through 65535")
	}
	if err := validateSeconds("server.read_header_timeout_seconds", settings.Server.ReadHeaderTimeoutSecs, 1, 30); err != nil {
		return err
	}
	if err := validateSeconds("server.read_timeout_seconds", settings.Server.ReadTimeoutSecs, 1, 120); err != nil {
		return err
	}
	if err := validateSeconds("server.write_timeout_seconds", settings.Server.WriteTimeoutSecs, 1, 120); err != nil {
		return err
	}
	if err := validateSeconds("server.idle_timeout_seconds", settings.Server.IdleTimeoutSecs, 1, 300); err != nil {
		return err
	}

	if settings.Display.Timezone != "Africa/Tunis" {
		return invalid("display.timezone", "must remain Africa/Tunis")
	}
	if !sameLanguages(settings.Display.SupportedLanguages) {
		return invalid("display.supported_languages", "must contain en and fr exactly once")
	}
	if !slices.Contains(settings.Display.SupportedLanguages, settings.Display.DefaultLanguage) {
		return invalid("display.default_language", "must be en or fr")
	}

	if settings.Logs.InitialLines != DefaultLogInitialLines {
		return invalid("logs.initial_lines", "must remain 100")
	}
	if settings.Logs.BrowserMaxBytes != DefaultBrowserLogMaxBytes {
		return invalid("logs.browser_max_bytes", "must remain the approved 5 MiB ceiling")
	}

	return nil
}

func validateThreshold(name string, threshold PercentageThreshold) error {
	if math.IsNaN(threshold.WarningPercent) || math.IsInf(threshold.WarningPercent, 0) ||
		threshold.WarningPercent <= 0 || threshold.WarningPercent >= 100 {
		return invalid(name+".warning_percent", "must be greater than 0 and less than 100")
	}
	if math.IsNaN(threshold.CriticalPercent) || math.IsInf(threshold.CriticalPercent, 0) ||
		threshold.CriticalPercent <= 0 || threshold.CriticalPercent > 100 {
		return invalid(name+".critical_percent", "must be greater than 0 and no greater than 100")
	}
	if threshold.WarningPercent >= threshold.CriticalPercent {
		return invalid(name, "warning_percent must be lower than critical_percent")
	}
	return nil
}

func validateSeconds(field string, value, minimum, maximum int) error {
	if value < minimum || value > maximum {
		return invalid(field, fmt.Sprintf("must be from %d through %d", minimum, maximum))
	}
	return nil
}

func sameLanguages(languages []string) bool {
	if len(languages) != 2 {
		return false
	}
	return slices.Contains(languages, "en") && slices.Contains(languages, "fr")
}
