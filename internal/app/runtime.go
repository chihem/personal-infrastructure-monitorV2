package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/chihem/personal-infrastructure-monitorV2/internal/api"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/collector/docker"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/collector/host"
	hostcpu "github.com/chihem/personal-infrastructure-monitorV2/internal/collector/host/cpu"
	hostmemory "github.com/chihem/personal-infrastructure-monitorV2/internal/collector/host/memory"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/config"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/observability"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/scheduler"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/storage"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/storage/audit"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/storage/history"
	"github.com/chihem/personal-infrastructure-monitorV2/internal/web"
)

const (
	DefaultSettingsPath  = "/etc/pim/settings.toml"
	DefaultLastValidPath = "/var/lib/pim/last-valid-settings.toml"
	defaultShutdownTime  = 15 * time.Second
	defaultCollectorTime = 10 * time.Second
)

type listenerFactory func(context.Context, int) (net.Listener, error)
type databaseOpener func(context.Context, string) (io.Closer, error)
type historyDatabaseOpener func(context.Context, string) (*storage.Database, error)

type Options struct {
	SettingsPath  string
	LastValidPath string
	HistoryPath   string
	AuditPath     string

	ShutdownTimeout time.Duration
	Logger          *observability.Logger

	listen           listenerFactory
	openHistory      historyDatabaseOpener
	openAudit        databaseOpener
	now              func() time.Time
	databaseSize     databaseSizer
	collectionStatus collectionStatusProvider
}

func DefaultOptions() Options {
	return Options{
		SettingsPath:    DefaultSettingsPath,
		LastValidPath:   DefaultLastValidPath,
		HistoryPath:     storage.DefaultHistoryPath,
		AuditPath:       storage.DefaultAuditPath,
		ShutdownTimeout: defaultShutdownTime,
		Logger:          observability.New(os.Stdout, os.Stderr),
		now:             time.Now,
		databaseSize:    sizeDatabaseFiles,
		listen:          ListenTailscaleIPv4,
		openHistory: func(ctx context.Context, path string) (*storage.Database, error) {
			return history.Open(ctx, path)
		},
		openAudit: func(ctx context.Context, path string) (io.Closer, error) {
			return audit.Open(ctx, path)
		},
	}
}

// Run wires the production application and blocks until the context is
// cancelled or the HTTP server exits unexpectedly.
func Run(ctx context.Context, options Options) (runErr error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := options.prepare(); err != nil {
		return err
	}
	startedAt := options.now().UTC()
	_ = options.Logger.Info(observability.Event{Component: "app", Code: "app.starting"})
	defer func() {
		duration := options.now().Sub(startedAt)
		if duration < 0 {
			duration = 0
		}
		if runErr != nil {
			_ = options.Logger.Error(observability.Event{Component: "app", Code: "app.run.failed", Duration: &duration})
			return
		}
		_ = options.Logger.Info(observability.Event{Component: "app", Code: "app.stopped", Duration: &duration})
	}()

	settingsManager, err := config.NewManager(config.Paths{
		Active:    options.SettingsPath,
		LastValid: options.LastValidPath,
	}, config.WithEventSink(options.configurationEventSink()))
	if err != nil {
		return fmt.Errorf("create settings manager: %w", err)
	}
	if err := settingsManager.Load(); err != nil {
		return fmt.Errorf("load settings: %w", err)
	}
	settings, ok := settingsManager.Current()
	if !ok {
		return errors.New("settings manager returned no active settings")
	}

	listener, err := options.listen(ctx, settings.Server.Port)
	if err != nil {
		return fmt.Errorf("open private HTTP listener: %w", err)
	}

	historyDatabase, err := options.openHistory(ctx, options.HistoryPath)
	if err != nil {
		return errors.Join(fmt.Errorf("open history database: %w", err), listener.Close())
	}
	closers := []io.Closer{historyDatabase}
	historyRepository, err := history.New(historyDatabase)
	if err != nil {
		return errors.Join(fmt.Errorf("create history repository: %w", err), listener.Close(), historyDatabase.Close())
	}
	cpuCollector, err := hostcpu.NewWithOptions(hostcpu.Options{Now: options.now})
	if err != nil {
		return errors.Join(fmt.Errorf("create CPU collector: %w", err), listener.Close(), historyDatabase.Close())
	}
	memoryCollector, err := hostmemory.NewWithOptions(hostmemory.Options{Now: options.now})
	if err != nil {
		return errors.Join(fmt.Errorf("create memory collector: %w", err), listener.Close(), historyDatabase.Close())
	}
	hostCollector, err := host.NewCollector(cpuCollector, memoryCollector)
	if err != nil {
		return errors.Join(fmt.Errorf("create host collector: %w", err), listener.Close(), historyDatabase.Close())
	}
	collectionScheduler, err := scheduler.New(scheduler.Options{
		Host: hostCollector, Docker: docker.UnavailableProvider{}, CollectorTimeout: defaultCollectorTime,
		Recorder: collectionRecorder{history: historyRepository},
	})
	if err != nil {
		return errors.Join(fmt.Errorf("create collection scheduler: %w", err), listener.Close(), historyDatabase.Close())
	}

	auditDatabase, auditErr := options.openAudit(ctx, options.AuditPath)
	if auditErr == nil {
		closers = append(closers, auditDatabase)
	}

	maintenance := &MaintenanceState{}
	collectionStatus := options.collectionStatus
	if collectionStatus == nil {
		collectionStatus = collectionScheduler
	}
	status := statusSource{
		startedAt: startedAt, now: options.now, maintenance: maintenance,
		configuration: settingsManager.Status,
		historyPath:   options.HistoryPath, auditPath: options.AuditPath,
		historyAvailable: true, auditAvailable: auditErr == nil,
		collection: collectionStatus, databaseSize: options.databaseSize,
	}
	cpuSource := cpuDataSource{
		collector: cpuCollector, history: historyRepository, now: options.now,
		staleAfter: time.Duration(settings.Collection.StaleAfterSecs) * time.Second,
	}
	memorySource := memoryDataSource{
		collector: memoryCollector, history: historyRepository, now: options.now,
		staleAfter: time.Duration(settings.Collection.StaleAfterSecs) * time.Second,
	}
	handler := api.NewHandlerWithOptions(api.HandlerOptions{
		WebHandler: web.Handler(), Status: status.snapshot, CPU: cpuSource, Memory: memorySource, Logger: options.Logger,
	})
	server := api.NewServer(listener.Addr().String(), settings.Server, handler)

	watchContext, stopWatcher := context.WithCancel(ctx)
	if err := settingsManager.Start(watchContext); err != nil {
		stopWatcher()
		return errors.Join(fmt.Errorf("start settings watcher: %w", err), listener.Close(), closeAll(closers))
	}
	workerContext, stopWorkers := context.WithCancel(ctx)
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		runCollectionScheduler(workerContext, collectionScheduler, options.Logger)
	}()
	go func() {
		defer workers.Done()
		runHistoryRetention(workerContext, historyRepository, options.now, options.Logger)
	}()

	runtime := &runtime{
		server:          server,
		listener:        listener,
		closers:         closers,
		stopWatcher:     stopWatcher,
		stopWorkers:     stopWorkers,
		workers:         &workers,
		shutdownTimeout: options.ShutdownTimeout,
	}
	_ = options.Logger.Info(observability.Event{Component: "app", Code: "app.ready"})
	return runtime.serve(ctx)
}

func runCollectionScheduler(ctx context.Context, service *scheduler.Service, logger *observability.Logger) {
	for ctx.Err() == nil {
		err := service.Run(ctx)
		if err == nil || ctx.Err() != nil {
			return
		}
		_ = logger.Warning(observability.Event{Component: "collection", Code: "collection.scheduler.failed"})
	}
}

func (options *Options) prepare() error {
	defaults := DefaultOptions()
	if options.SettingsPath == "" || options.LastValidPath == "" ||
		options.HistoryPath == "" || options.AuditPath == "" {
		return errors.New("settings, last-valid, history, and audit paths are required")
	}
	if options.ShutdownTimeout <= 0 {
		return errors.New("shutdown timeout must be positive")
	}
	if options.listen == nil {
		options.listen = defaults.listen
	}
	if options.openHistory == nil {
		options.openHistory = defaults.openHistory
	}
	if options.openAudit == nil {
		options.openAudit = defaults.openAudit
	}
	if options.Logger == nil {
		options.Logger = defaults.Logger
	}
	if options.now == nil {
		options.now = defaults.now
	}
	if options.databaseSize == nil {
		options.databaseSize = defaults.databaseSize
	}
	return nil
}

func (options Options) configurationEventSink() config.EventSink {
	return func(event config.ReloadEvent) {
		logEvent := observability.Event{Component: "config", Code: "config.reload." + string(event.Outcome)}
		switch event.Outcome {
		case config.ReloadRejected, config.ReloadUnavailable, config.ReloadWatchError:
			_ = options.Logger.Warning(logEvent)
		default:
			_ = options.Logger.Info(logEvent)
		}
	}
}

var _ collectionStatusProvider = (*scheduler.Service)(nil)

type runtime struct {
	server          *http.Server
	listener        net.Listener
	closers         []io.Closer
	stopWatcher     context.CancelFunc
	stopWorkers     context.CancelFunc
	workers         *sync.WaitGroup
	shutdownTimeout time.Duration
	closeOnce       sync.Once
	closeErr        error
}

func (runtime *runtime) serve(ctx context.Context) error {
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- runtime.server.Serve(runtime.listener)
	}()

	select {
	case serveErr := <-serveErrors:
		runtime.stopServices()
		closeErr := runtime.closeResources()
		if errors.Is(serveErr, http.ErrServerClosed) {
			return closeErr
		}
		return errors.Join(fmt.Errorf("serve private HTTP: %w", serveErr), closeErr)
	case <-ctx.Done():
		runtime.stopServices()
		shutdownContext, cancel := context.WithTimeout(context.Background(), runtime.shutdownTimeout)
		shutdownErr := runtime.server.Shutdown(shutdownContext)
		cancel()
		if shutdownErr != nil {
			shutdownErr = errors.Join(shutdownErr, runtime.server.Close())
		}

		serveErr := <-serveErrors
		if errors.Is(serveErr, http.ErrServerClosed) {
			serveErr = nil
		}
		return errors.Join(shutdownErr, serveErr, runtime.closeResources())
	}
}

func (runtime *runtime) stopServices() {
	runtime.stopWatcher()
	if runtime.stopWorkers != nil {
		runtime.stopWorkers()
	}
	if runtime.workers != nil {
		runtime.workers.Wait()
	}
}

func (runtime *runtime) closeResources() error {
	runtime.closeOnce.Do(func() {
		runtime.closeErr = closeAll(runtime.closers)
	})
	return runtime.closeErr
}

func closeAll(closers []io.Closer) error {
	var combined error
	for index := len(closers) - 1; index >= 0; index-- {
		if closers[index] != nil {
			combined = errors.Join(combined, closers[index].Close())
		}
	}
	return combined
}
