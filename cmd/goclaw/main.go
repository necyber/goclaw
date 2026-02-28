package main

// @title Goclaw API
// @version 1.0
// @description Production-grade, high-performance, distributed-ready multi-agent orchestration engine
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url https://github.com/goclaw/goclaw
// @contact.email support@goclaw.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /
// @schemes http https

import (
	"context"
	"flag"
	"fmt"
	"os"
	ossignal "os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/goclaw/goclaw/config"
	"github.com/goclaw/goclaw/pkg/api"
	"github.com/goclaw/goclaw/pkg/api/events"
	"github.com/goclaw/goclaw/pkg/api/handlers"
	"github.com/goclaw/goclaw/pkg/engine"
	grpcpkg "github.com/goclaw/goclaw/pkg/grpc"
	grpchandlers "github.com/goclaw/goclaw/pkg/grpc/handlers"
	pb "github.com/goclaw/goclaw/pkg/grpc/pb/v1"
	grpcstreaming "github.com/goclaw/goclaw/pkg/grpc/streaming"
	"github.com/goclaw/goclaw/pkg/lane"
	"github.com/goclaw/goclaw/pkg/logger"
	memorypkg "github.com/goclaw/goclaw/pkg/memory"
	"github.com/goclaw/goclaw/pkg/metrics"
	signalpkg "github.com/goclaw/goclaw/pkg/signal"
	"github.com/goclaw/goclaw/pkg/storage"
	badgerstorage "github.com/goclaw/goclaw/pkg/storage/badger"
	memstorage "github.com/goclaw/goclaw/pkg/storage/memory"
	tracingpkg "github.com/goclaw/goclaw/pkg/telemetry/tracing"
	"github.com/goclaw/goclaw/pkg/version"

	dgbadger "github.com/dgraph-io/badger/v4"
	"github.com/redis/go-redis/v9"
)

var (
	configPath  = flag.String("config", "", "Path to configuration file")
	versionFlag = flag.Bool("version", false, "Print version information")
	helpFlag    = flag.Bool("help", false, "Print help information")

	// CLI overrides
	appName    = flag.String("app-name", "", "Override app name")
	serverPort = flag.Int("port", 0, "Override server port")
	logLevel   = flag.String("log-level", "", "Override log level")
	debugMode  = flag.Bool("debug", false, "Enable debug mode")
)

func main() {
	flag.Parse()

	// Print help
	if *helpFlag {
		printHelp()
		os.Exit(0)
	}

	// Print version
	if *versionFlag {
		printVersion()
		os.Exit(0)
	}

	// Build CLI overrides map
	overrides := buildOverrides()

	// Load configuration
	cfg, err := config.Load(*configPath, overrides)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration:\n%s\n", err)
		os.Exit(1)
	}

	// Initialize logger with configuration
	logCfg := &logger.Config{
		Level:  logger.ParseLevel(cfg.Log.Level),
		Format: cfg.Log.Format,
		Output: cfg.Log.Output,
	}
	if cfg.App.Debug || *debugMode {
		logCfg.Level = logger.DebugLevel
	}
	log := logger.New(logCfg)
	logger.SetGlobal(log)

	log.Info("Starting Goclaw",
		"version", version.Version,
		"buildTime", version.BuildTime,
		"gitCommit", version.GitCommit,
		"app", cfg.App.Name,
		"environment", cfg.App.Environment,
	)

	log.Debug("Configuration loaded", "config", cfg.String())

	// Create root context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tracingShutdown, err := initTracing(ctx, cfg, log)
	if err != nil {
		log.Error("Failed to initialize gRPC tracing", "error", err)
		os.Exit(1)
	}

	// Setup graceful shutdown
	sigChan := setupShutdownSignals()
	defer stopShutdownSignals(sigChan)

	// Initialize storage backend
	var store storage.Storage
	switch cfg.Storage.Type {
	case "badger":
		badgerCfg := &badgerstorage.Config{
			Path:             cfg.Storage.Badger.Path,
			SyncWrites:       cfg.Storage.Badger.SyncWrites,
			ValueLogFileSize: cfg.Storage.Badger.ValueLogFileSize,
		}
		store, err = badgerstorage.NewBadgerStorage(badgerCfg)
		if err != nil {
			log.Error("Failed to create Badger storage", "error", err)
			os.Exit(1)
		}
		log.Info("Initialized Badger storage", "path", badgerCfg.Path)
	case "memory":
		store = memstorage.NewMemoryStorage()
		log.Info("Initialized memory storage")
	default:
		store = memstorage.NewMemoryStorage()
		log.Warn("Unknown storage type, using memory storage", "type", cfg.Storage.Type)
	}
	defer func() {
		if err := store.Close(); err != nil {
			log.Error("Error closing storage", "error", err)
		}
	}()

	// Initialize metrics manager
	metricsCfg := metrics.Config{
		Enabled:                 cfg.Metrics.Enabled,
		Port:                    cfg.Metrics.Port,
		Path:                    cfg.Metrics.Path,
		WorkflowDurationBuckets: metrics.DefaultConfig().WorkflowDurationBuckets,
		TaskDurationBuckets:     metrics.DefaultConfig().TaskDurationBuckets,
		LaneWaitBuckets:         metrics.DefaultConfig().LaneWaitBuckets,
		HTTPDurationBuckets:     metrics.DefaultConfig().HTTPDurationBuckets,
	}
	metricsManager := metrics.NewManager(metricsCfg)
	signalpkg.SetMetricsRecorder(metricsManager)

	// Start metrics server if enabled
	if metricsManager.Enabled() {
		go func() {
			log.Info("Starting metrics server", "port", cfg.Metrics.Port, "path", cfg.Metrics.Path)
			if err := metricsManager.StartServer(ctx, cfg.Metrics.Port, cfg.Metrics.Path); err != nil {
				log.Error("Metrics server error", "error", err)
			}
		}()
	}

	// Initialize and start the orchestration engine.
	eventBroadcaster := events.NewBroadcaster()
	var streamingRegistry *grpcstreaming.SubscriberRegistry
	var streamObserver *grpcstreaming.WorkflowStreamObserver
	if cfg.Server.GRPC.Enabled {
		streamingRegistry = grpcstreaming.NewSubscriberRegistry()
		streamObserver = grpcstreaming.NewWorkflowStreamObserver(streamingRegistry)
	}
	runtimeBroadcaster := newRuntimeEventBroadcaster(eventBroadcaster, streamObserver)
	wsHandler := handlers.NewWebSocketHandler(log, handlers.WebSocketConfig{
		AllowedOrigins: cfg.Server.CORS.AllowedOrigins,
		MaxConnections: cfg.UI.MaxWebSocketConnections,
		PingInterval:   30 * time.Second,
		PongTimeout:    10 * time.Second,
	})
	eventSubscription := eventBroadcaster.Subscribe(256)
	defer eventBroadcaster.Unsubscribe(eventSubscription)
	go func() {
		for event := range eventSubscription {
			_ = wsHandler.Broadcast(handlers.EventMessage{
				Type:      event.Type,
				Timestamp: event.Timestamp,
				Payload:   event.Payload,
			})
		}
	}()

	engineOpts := []engine.Option{
		engine.WithMetrics(metricsManager),
		engine.WithEventBroadcaster(runtimeBroadcaster),
	}

	needsRedis := cfg.Redis.Enabled || cfg.Orchestration.Queue.Type == "redis" || cfg.Signal.Mode == "redis"
	var redisClient *redis.Client
	if needsRedis {
		redisClient, err = initializeRedisClient(ctx, cfg)
		if err != nil {
			log.Warn("Redis initialization failed; distributed Redis features will fall back to local mode", "error", err)
		} else {
			engineOpts = append(engineOpts, engine.WithRedisClient(redisClient))
			log.Info("Redis client initialized", "address", cfg.Redis.Address, "db", cfg.Redis.DB, "sentinel", cfg.Redis.Sentinel.Enabled)
		}
	}

	signalBus, effectiveSignalMode := initializeSignalBus(cfg, redisClient, log)
	engineOpts = append(engineOpts, engine.WithSignalBus(signalBus))

	// Initialize memory hub if enabled
	var memoryHub *memorypkg.MemoryHub
	var memoryHandler *handlers.MemoryHandler
	if cfg.Memory.Enabled {
		// Memory system needs its own Badger instance for storage
		memoryBadgerOpts := dgbadger.DefaultOptions(cfg.Memory.StoragePath)
		memoryBadgerOpts.Logger = nil
		memoryDB, err := dgbadger.Open(memoryBadgerOpts)
		if err != nil {
			log.Error("Failed to open memory Badger DB", "error", err)
			os.Exit(1)
		}
		defer func() {
			if err := memoryDB.Close(); err != nil {
				log.Error("Error closing memory Badger DB", "error", err)
			}
		}()

		l1Cache := memorypkg.NewL1Cache(cfg.Memory.L1CacheSize)
		l2Storage := memorypkg.NewL2Badger(memoryDB)
		tieredStorage := memorypkg.NewTieredStorage(l1Cache, l2Storage)

		memoryHub = memorypkg.NewMemoryHub(&cfg.Memory, tieredStorage, log)
		engineOpts = append(engineOpts, engine.WithMemoryHub(memoryHub))
		memoryHandler = handlers.NewMemoryHandler(memoryHub, log)

		log.Info("Memory hub initialized",
			"vector_dimension", cfg.Memory.VectorDimension,
			"l1_cache_size", cfg.Memory.L1CacheSize,
		)
	} else {
		log.Info("Memory hub disabled")
	}

	effectiveQueueType := cfg.Orchestration.Queue.Type
	if effectiveQueueType == "redis" && redisClient == nil {
		effectiveQueueType = "memory(fallback)"
	}
	log.Info("Distributed runtime configured",
		"queue_type", effectiveQueueType,
		"signal_mode", effectiveSignalMode,
		"redis_connected", redisClient != nil,
	)

	eng, err := engine.New(cfg, log, store, engineOpts...)
	if err != nil {
		log.Error("Failed to create engine", "error", err)
		os.Exit(1)
	}
	if err := eng.Start(ctx); err != nil {
		log.Error("Failed to start engine", "error", err)
		os.Exit(1)
	}

	var sagaHandler *handlers.SagaHandler
	var sagaGRPCService *grpchandlers.SagaServiceServer
	if cfg.Saga.Enabled {
		sagaOrchestrator := eng.GetSagaOrchestrator()
		if sagaOrchestrator == nil {
			log.Warn("Saga is enabled but orchestrator is unavailable")
		} else {
			sagaCheckpointStore := eng.GetSagaCheckpointStore()
			sagaRecoveryManager := eng.GetSagaRecoveryManager()
			sagaHandler = handlers.NewSagaHandler(sagaOrchestrator, sagaCheckpointStore, sagaRecoveryManager, log)
			sagaGRPCService = grpchandlers.NewSagaServiceServer(sagaOrchestrator, sagaCheckpointStore)
			log.Info("Saga orchestrator initialized",
				"max_concurrent", cfg.Saga.MaxConcurrent,
				"wal_sync_mode", cfg.Saga.WALSyncMode,
				"compensation_policy", cfg.Saga.CompensationPolicy,
			)
		}
	} else {
		log.Info("Saga orchestrator disabled")
	}

	// Initialize HTTP server with handlers
	workflowHandler := handlers.NewWorkflowHandler(eng, log)
	healthHandler := handlers.NewHealthHandler(eng)

	apiHandlers := &api.Handlers{
		Workflow:  workflowHandler,
		Health:    healthHandler,
		Memory:    memoryHandler,
		Saga:      sagaHandler,
		Metrics:   metricsManager,
		WebSocket: wsHandler,
	}

	httpServer := api.NewHTTPServer(cfg, log, apiHandlers)

	// Start HTTP server in a separate goroutine
	serverErrChan := make(chan error, 2) // Increased buffer for both HTTP and gRPC
	go func() {
		log.Info("Starting HTTP server", "address", fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port))
		if err := httpServer.Start(); err != nil {
			serverErrChan <- err
		}
	}()

	// Initialize and start gRPC server if enabled
	var grpcServer *grpcpkg.Server
	if cfg.Server.GRPC.Enabled {
		grpcCfg := cfg.Server.GRPC.ToGRPCConfig()
		grpcCfg.EnableTracing = cfg.Server.GRPC.EnableTracing && cfg.Tracing.Enabled
		grpcServer, err = grpcpkg.New(grpcCfg)
		if err != nil {
			log.Error("Failed to create gRPC server", "error", err)
			os.Exit(1)
		}
		if err := registerGRPCServices(grpcServer, eng, signalBus, streamingRegistry, sagaGRPCService); err != nil {
			log.Error("Failed to register gRPC services", "error", err)
			os.Exit(1)
		}

		// Start gRPC server in a separate goroutine
		go func() {
			log.Info("Starting gRPC server", "address", grpcCfg.Address)
			if err := grpcServer.Start(); err != nil {
				serverErrChan <- fmt.Errorf("gRPC server error: %w", err)
			}
		}()
	} else {
		log.Info("gRPC server disabled")
	}

	log.Info("Goclaw is running",
		"http_port", cfg.Server.Port,
		"grpc_port", cfg.Server.GRPC.Port,
		"grpc_enabled", cfg.Server.GRPC.Enabled,
		"metrics_port", cfg.Metrics.Port,
	)
	if cfg.UI.Enabled {
		basePath := strings.TrimSpace(cfg.UI.BasePath)
		if basePath == "" {
			basePath = "/ui"
		}
		log.Info(fmt.Sprintf("Web UI available at http://localhost:%d%s", cfg.Server.Port, strings.TrimRight(basePath, "/")))
	}
	log.Info("Press Ctrl+C to stop")

	// Wait for shutdown signal or server error
	select {
	case sig := <-sigChan:
		log.Info("Received shutdown signal", "signal", sig)
	case err := <-serverErrChan:
		log.Error("HTTP server error", "error", err)
	case <-ctx.Done():
		log.Info("Context cancelled")
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server first
	log.Info("Shutting down HTTP server")
	log.Info("Closing websocket connections")
	wsHandler.Close()
	eventBroadcaster.Close()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("Error shutting down HTTP server", "error", err)
	}

	// Shutdown gRPC server if it was started
	if grpcServer != nil && grpcServer.IsRunning() {
		log.Info("Shutting down gRPC server")
		if err := grpcServer.Stop(shutdownCtx); err != nil {
			log.Error("Error shutting down gRPC server", "error", err)
		}
	}
	if err := shutdownTracing(tracingShutdown, cfg.Tracing.Timeout, log); err != nil {
		log.Error("Error shutting down gRPC tracing", "error", err)
	}

	// Stop the engine gracefully.
	log.Info("Stopping engine")
	if err := eng.Stop(shutdownCtx); err != nil {
		log.Error("Error during engine shutdown", "error", err)
	}

	log.Info("Closing signal bus")
	if err := signalBus.Close(); err != nil {
		log.Error("Error closing signal bus", "error", err)
	}
	if redisClient != nil {
		log.Info("Closing Redis client")
		if err := redisClient.Close(); err != nil {
			log.Error("Error closing Redis client", "error", err)
		}
	}

	log.Info("Goclaw stopped gracefully")
}

func initializeRedisClient(ctx context.Context, cfg *config.Config) (*redis.Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if cfg.Redis.Sentinel.Enabled {
		client := lane.NewRedisSentinelClient(&redis.FailoverOptions{
			MasterName:    cfg.Redis.Sentinel.MasterName,
			SentinelAddrs: cfg.Redis.Sentinel.Addresses,
			Password:      cfg.Redis.Password,
			DB:            cfg.Redis.DB,
			MaxRetries:    cfg.Redis.MaxRetries,
			PoolSize:      cfg.Redis.PoolSize,
			MinIdleConns:  cfg.Redis.MinIdleConns,
			DialTimeout:   cfg.Redis.DialTimeout,
			ReadTimeout:   cfg.Redis.ReadTimeout,
			WriteTimeout:  cfg.Redis.WriteTimeout,
		})
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := lane.PingRedis(pingCtx, client); err != nil {
			_ = client.Close()
			return nil, err
		}
		return client, nil
	}

	client := lane.NewRedisClient(&redis.Options{
		Addr:         cfg.Redis.Address,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		MaxRetries:   cfg.Redis.MaxRetries,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
	})
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := lane.PingRedis(pingCtx, client); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}

func initializeSignalBus(cfg *config.Config, redisClient redis.UniversalClient, log logger.Logger) (signalpkg.Bus, string) {
	if cfg != nil && cfg.Signal.Mode == "redis" {
		if redisClient == nil {
			if log != nil {
				log.Warn("Signal bus redis mode requested but Redis client unavailable; falling back to local bus")
			}
			return signalpkg.NewLocalBus(cfg.Signal.BufferSize), "local(fallback)"
		}

		bus := signalpkg.NewRedisBus(redisClient, cfg.Signal.ChannelPrefix, cfg.Signal.BufferSize)
		if !bus.Healthy() {
			if log != nil {
				log.Warn("Redis signal bus health check failed; falling back to local bus")
			}
			_ = bus.Close()
			return signalpkg.NewLocalBus(cfg.Signal.BufferSize), "local(fallback)"
		}
		return bus, "redis"
	}

	bufferSize := 16
	if cfg != nil {
		bufferSize = cfg.Signal.BufferSize
	}
	return signalpkg.NewLocalBus(bufferSize), "local"
}

func initTracing(
	ctx context.Context,
	cfg *config.Config,
	log logger.Logger,
) (func(context.Context) error, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	tracingCfg := cfg.Tracing

	shutdown, err := tracingpkg.Init(ctx, tracingCfg, cfg.App.Name, cfg.App.Version)
	if err != nil {
		return nil, fmt.Errorf("initialize tracing provider: %w", err)
	}

	if log != nil {
		logTracingStartup(log, cfg, tracingCfg)
	}

	return shutdown, nil
}

func shutdownTracing(
	shutdown func(context.Context) error,
	timeout time.Duration,
	log logger.Logger,
) error {
	if shutdown == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), resolveTracingShutdownTimeout(timeout))
	defer cancel()

	if log != nil {
		log.Info("Shutting down tracing provider", "timeout", resolveTracingShutdownTimeout(timeout).String())
	}

	return shutdown(ctx)
}

func resolveTracingShutdownTimeout(timeout time.Duration) time.Duration {
	if timeout > 0 {
		return timeout
	}
	return 5 * time.Second
}

func logTracingStartup(log logger.Logger, cfg *config.Config, tracingCfg config.TracingConfig) {
	if log == nil || cfg == nil {
		return
	}

	if !tracingCfg.Enabled {
		log.Info("OpenTelemetry tracing disabled",
			"enabled", false,
		)
		return
	}

	log.Info("OpenTelemetry tracing enabled",
		"enabled", true,
		"exporter", tracingCfg.Exporter,
		"endpoint", summarizeTracingEndpoint(tracingCfg.Endpoint),
		"sampler", tracingCfg.Sampler,
		"sample_rate", tracingCfg.SampleRate,
		"grpc_interceptor_enabled", cfg.Server.GRPC.Enabled && cfg.Server.GRPC.EnableTracing,
	)
}

func summarizeTracingEndpoint(endpoint string) string {
	raw := strings.TrimSpace(endpoint)
	if raw == "" {
		return ""
	}
	if parts := strings.SplitN(raw, "://", 2); len(parts) == 2 {
		raw = parts[1]
	}
	if idx := strings.Index(raw, "/"); idx >= 0 {
		raw = raw[:idx]
	}
	return raw
}

type runtimeEventBroadcaster struct {
	web      *events.Broadcaster
	observer *grpcstreaming.WorkflowStreamObserver
}

func newRuntimeEventBroadcaster(web *events.Broadcaster, observer *grpcstreaming.WorkflowStreamObserver) *runtimeEventBroadcaster {
	return &runtimeEventBroadcaster{
		web:      web,
		observer: observer,
	}
}

func (b *runtimeEventBroadcaster) BroadcastWorkflowStateChanged(workflowID, name, oldState, newState string, updatedAt time.Time) {
	if b.web != nil {
		b.web.BroadcastWorkflowStateChanged(workflowID, name, oldState, newState, updatedAt)
	}
	if b.observer != nil {
		b.observer.OnWorkflowEvent(engine.WorkflowEvent{
			WorkflowID: workflowID,
			EventType:  mapWorkflowEventType(newState),
			Status:     strings.ToUpper(newState),
			Message:    "workflow state changed",
			Timestamp:  updatedAt.Unix(),
		})
	}
}

func (b *runtimeEventBroadcaster) BroadcastTaskStateChanged(
	workflowID, taskID, taskName, oldState, newState, errorMessage string,
	result any,
	updatedAt time.Time,
) {
	if b.web != nil {
		b.web.BroadcastTaskStateChanged(workflowID, taskID, taskName, oldState, newState, errorMessage, result, updatedAt)
	}
	if b.observer != nil {
		message := "task state changed"
		if errorMessage != "" {
			message = errorMessage
		}
		b.observer.OnTaskEvent(engine.TaskEvent{
			WorkflowID: workflowID,
			TaskID:     taskID,
			EventType:  mapTaskEventType(newState),
			Status:     strings.ToUpper(newState),
			Message:    message,
			Timestamp:  updatedAt.Unix(),
		})
	}
}

func mapWorkflowEventType(state string) engine.WorkflowEventType {
	switch strings.ToLower(state) {
	case "pending":
		return engine.WorkflowEventSubmitted
	case "running":
		return engine.WorkflowEventStarted
	case "completed":
		return engine.WorkflowEventCompleted
	case "failed":
		return engine.WorkflowEventFailed
	case "cancelled":
		return engine.WorkflowEventCancelled
	default:
		return engine.WorkflowEventStarted
	}
}

func mapTaskEventType(state string) engine.TaskEventType {
	switch strings.ToLower(state) {
	case "running":
		return engine.TaskEventStarted
	case "completed":
		return engine.TaskEventCompleted
	case "failed":
		return engine.TaskEventFailed
	case "cancelled":
		return engine.TaskEventCancelled
	default:
		return engine.TaskEventProgress
	}
}

func setupShutdownSignals() chan os.Signal {
	sigChan := make(chan os.Signal, 1)
	ossignal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	return sigChan
}

func stopShutdownSignals(sigChan chan os.Signal) {
	if sigChan == nil {
		return
	}
	ossignal.Stop(sigChan)
}

func registerGRPCServices(
	grpcServer *grpcpkg.Server,
	eng *engine.Engine,
	signalBus signalpkg.Bus,
	streamingRegistry *grpcstreaming.SubscriberRegistry,
	sagaSvc *grpchandlers.SagaServiceServer,
) error {
	if grpcServer == nil {
		return fmt.Errorf("grpc server is nil")
	}
	if eng == nil {
		return fmt.Errorf("engine adapter wiring is missing")
	}
	if streamingRegistry == nil {
		return fmt.Errorf("streaming registry wiring is missing")
	}
	if signalBus == nil {
		return fmt.Errorf("signal bus wiring is missing")
	}

	engineAdapter := grpchandlers.NewEngineAdapter(eng)
	if engineAdapter == nil {
		return fmt.Errorf("engine adapter wiring is invalid")
	}

	workflowSvc := grpchandlers.NewWorkflowServiceServer(engineAdapter)
	batchSvc := grpchandlers.NewBatchServiceServer(engineAdapter)
	streamingSvc := grpchandlers.NewStreamingServiceServer(streamingRegistry)
	adminSvc := grpchandlers.NewAdminServiceServer(engineAdapter)
	signalSvc := grpchandlers.NewSignalServiceServer(signalBus)
	if sagaSvc == nil {
		sagaSvc = grpchandlers.NewSagaServiceServer(nil, nil)
	}

	grpcServer.RegisterService(&pb.WorkflowService_ServiceDesc, workflowSvc)
	grpcServer.RegisterService(&pb.BatchService_ServiceDesc, batchSvc)
	grpcServer.RegisterService(&pb.StreamingService_ServiceDesc, streamingSvc)
	grpcServer.RegisterService(&pb.AdminService_ServiceDesc, adminSvc)
	grpcServer.RegisterService(&pb.SignalService_ServiceDesc, signalSvc)
	grpcServer.RegisterService(&pb.SagaService_ServiceDesc, sagaSvc)

	return nil
}

func buildOverrides() map[string]interface{} {
	overrides := make(map[string]interface{})

	if *appName != "" {
		overrides["app.name"] = *appName
	}
	if *serverPort != 0 {
		overrides["server.port"] = *serverPort
	}
	if *logLevel != "" {
		overrides["log.level"] = *logLevel
	}
	if *debugMode {
		overrides["app.debug"] = true
	}

	return overrides
}

func printVersion() {
	fmt.Printf("Goclaw - Multi-Agent Orchestration Engine\n")
	fmt.Printf("Version:    %s\n", version.Version)
	fmt.Printf("Build Time: %s\n", version.BuildTime)
	fmt.Printf("Git Commit: %s\n", version.GitCommit)
	fmt.Printf("Go Version: %s\n", version.GoVersion)
}

func printHelp() {
	fmt.Printf("Goclaw - Production-grade, high-performance, distributed-ready multi-Agent orchestration engine\n\n")
	fmt.Printf("Usage: goclaw [options]\n\n")
	fmt.Printf("Options:\n")
	flag.PrintDefaults()
	fmt.Printf("\nExamples:\n")
	fmt.Printf("  goclaw                                    # Run with default config\n")
	fmt.Printf("  goclaw -config config.yaml                # Use specific config file\n")
	fmt.Printf("  goclaw -port 9090 -log-level debug        # Override specific options\n")
	fmt.Printf("  goclaw -version                           # Print version info\n")
}
