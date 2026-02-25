package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/goclaw/goclaw/config"
	"github.com/goclaw/goclaw/pkg/api"
	"github.com/goclaw/goclaw/pkg/api/handlers"
	"github.com/goclaw/goclaw/pkg/engine"
	"github.com/goclaw/goclaw/pkg/logger"
	"github.com/goclaw/goclaw/pkg/version"
)

var (
	configPath = flag.String("config", "", "Path to configuration file")
	versionFlag = flag.Bool("version", false, "Print version information")
	helpFlag    = flag.Bool("help", false, "Print help information")

	// CLI overrides
	appName     = flag.String("app-name", "", "Override app name")
	serverPort  = flag.Int("port", 0, "Override server port")
	logLevel    = flag.String("log-level", "", "Override log level")
	debugMode   = flag.Bool("debug", false, "Enable debug mode")
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

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Initialize and start the orchestration engine.
	eng, err := engine.New(cfg, log)
	if err != nil {
		log.Error("Failed to create engine", "error", err)
		os.Exit(1)
	}
	if err := eng.Start(ctx); err != nil {
		log.Error("Failed to start engine", "error", err)
		os.Exit(1)
	}

	// Initialize HTTP server with handlers
	workflowHandler := handlers.NewWorkflowHandler(eng, log)
	healthHandler := handlers.NewHealthHandler(eng)

	apiHandlers := &api.Handlers{
		Workflow: workflowHandler,
		Health:   healthHandler,
	}

	httpServer := api.NewHTTPServer(cfg, log, apiHandlers)

	// Start HTTP server in a separate goroutine
	serverErrChan := make(chan error, 1)
	go func() {
		log.Info("Starting HTTP server", "address", fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port))
		if err := httpServer.Start(); err != nil {
			serverErrChan <- err
		}
	}()

	log.Info("Goclaw is running",
		"http_port", cfg.Server.Port,
		"grpc_port", cfg.Server.GRPC.Port,
		"metrics_port", cfg.Metrics.Port,
	)
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
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("Error shutting down HTTP server", "error", err)
	}

	// Stop the engine gracefully.
	log.Info("Stopping engine")
	if err := eng.Stop(shutdownCtx); err != nil {
		log.Error("Error during engine shutdown", "error", err)
	}

	log.Info("Goclaw stopped gracefully")
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
