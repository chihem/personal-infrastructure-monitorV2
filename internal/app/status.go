package app

import (
	"os"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/api/contracts"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/config"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/domain"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/scheduler"
)

type collectionStatusProvider interface {
	CollectionInProgress() bool
	LastRun() (scheduler.CompletedRun, bool)
	LastSuccessfulRun() (scheduler.CompletedRun, bool)
}

type databaseSizer func(string) (int64, bool)

type statusSource struct {
	startedAt        time.Time
	now              func() time.Time
	maintenance      *MaintenanceState
	configuration    func() config.Status
	historyPath      string
	auditPath        string
	historyAvailable bool
	auditAvailable   bool
	collection       collectionStatusProvider
	databaseSize     databaseSizer
}

func (source statusSource) snapshot() contracts.OperationalStatus {
	now := source.now().UTC()
	uptime := now.Sub(source.startedAt).Seconds()
	if uptime < 0 {
		uptime = 0
	}

	configurationState := mapConfigurationState(source.configuration().State)
	history := source.databaseStatus(source.historyPath, source.historyAvailable)
	audit := source.databaseStatus(source.auditPath, source.auditAvailable)
	collection, dockerConnectivity := source.collectionStatus()
	backupState := contracts.DependencyNotImplemented
	maintenance := source.maintenance.Active()
	ready := !maintenance && history.State == contracts.DependencyAvailable &&
		configurationState != contracts.ConfigurationUnavailable

	state := contracts.OperationalOK
	switch {
	case maintenance:
		state = contracts.OperationalMaintenance
	case !ready:
		state = contracts.OperationalNotReady
	case configurationState != contracts.ConfigurationValid ||
		audit.State != contracts.DependencyAvailable ||
		collection.State != contracts.DependencyAvailable ||
		backupState != contracts.DependencyAvailable ||
		dockerConnectivity != contracts.DependencyAvailable:
		state = contracts.OperationalDegraded
	}

	return contracts.OperationalStatus{
		State:              state,
		UptimeSeconds:      int64(uptime),
		Maintenance:        maintenance,
		ConfigurationState: configurationState,
		HistoryDatabase:    history,
		AuditDatabase:      audit,
		Collection:         collection,
		BackupState:        backupState,
		DockerConnectivity: dockerConnectivity,
	}
}

func (source statusSource) readiness() contracts.ReadinessStatus {
	snapshot := source.snapshot()
	return contracts.ReadinessStatus{
		Ready:                    snapshot.State != contracts.OperationalMaintenance && snapshot.State != contracts.OperationalNotReady,
		Maintenance:              snapshot.Maintenance,
		ConfigurationState:       snapshot.ConfigurationState,
		HistoryDatabaseAvailable: snapshot.HistoryDatabase.State == contracts.DependencyAvailable,
	}
}

func (source statusSource) databaseStatus(path string, available bool) contracts.DatabaseOperationalStatus {
	if !available {
		return contracts.DatabaseOperationalStatus{State: contracts.DependencyUnavailable}
	}
	status := contracts.DatabaseOperationalStatus{State: contracts.DependencyAvailable}
	if size, ok := source.databaseSize(path); ok {
		status.SizeBytes = &size
	}
	return status
}

func (source statusSource) collectionStatus() (contracts.CollectionOperationalStatus, contracts.DependencyState) {
	if source.collection == nil {
		return contracts.CollectionOperationalStatus{State: contracts.DependencyNotStarted}, contracts.DependencyNotChecked
	}

	status := contracts.CollectionOperationalStatus{
		State:      contracts.DependencyAvailable,
		InProgress: source.collection.CollectionInProgress(),
	}
	lastRun, hasLastRun := source.collection.LastRun()
	if hasLastRun {
		status.LastRun = collectionRunStatus(lastRun)
		if lastRun.Record.Status == domain.CollectionFailed {
			status.State = contracts.DependencyUnavailable
		}
	}
	if lastSuccess, ok := source.collection.LastSuccessfulRun(); ok {
		finishedAt := lastSuccess.Record.FinishedAt
		status.LastSuccessfulAt = &finishedAt
	}

	dockerConnectivity := contracts.DependencyNotChecked
	if hasLastRun {
		dockerConnectivity = contracts.DependencyAvailable
		if lastRun.Record.DockerResult.Status == domain.CollectionFailed ||
			lastRun.Record.DockerResult.Status == domain.CollectionNotAttempted {
			dockerConnectivity = contracts.DependencyUnavailable
		}
	}
	return status, dockerConnectivity
}

func collectionRunStatus(run scheduler.CompletedRun) *contracts.CollectionRunOperationalStatus {
	return &contracts.CollectionRunOperationalStatus{
		StartedAt:  run.Record.StartedAt,
		FinishedAt: run.Record.FinishedAt,
		DurationMS: run.Record.FinishedAt.Sub(run.Record.StartedAt).Milliseconds(),
		Status:     run.Record.Status,
	}
}

func mapConfigurationState(state config.State) contracts.ConfigurationState {
	switch state {
	case config.StateValid:
		return contracts.ConfigurationValid
	case config.StateUsingPrevious:
		return contracts.ConfigurationUsingPrevious
	default:
		return contracts.ConfigurationUnavailable
	}
}

func sizeDatabaseFiles(path string) (int64, bool) {
	var total int64
	for index, candidate := range []string{path, path + "-wal", path + "-shm"} {
		info, err := os.Lstat(candidate)
		if err != nil {
			if os.IsNotExist(err) && index > 0 {
				continue
			}
			return 0, false
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() < 0 {
			return 0, false
		}
		total += info.Size()
	}
	return total, true
}
