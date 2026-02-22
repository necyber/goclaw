package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/goclaw/goclaw/pkg/version"
)

var (
	configPath  = flag.String("config", "config.yaml", "Path to configuration file")
	versionFlag = flag.Bool("version", false, "Print version information")
	helpFlag    = flag.Bool("help", false, "Print help information")
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

	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting Goclaw",
		"version", version.Version,
		"buildTime", version.BuildTime,
		"gitCommit", version.GitCommit,
	)

	// Create root context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// TODO: Initialize and start the orchestration engine
	// engine := engine.New(engine.Config{
	//     ConfigPath: *configPath,
	// })
	// if err := engine.Start(ctx); err != nil {
	//     logger.Error("Failed to start engine", "error", err)
	//     os.Exit(1)
	// }

	logger.Info("Goclaw is running", "config", *configPath)
	logger.Info("Press Ctrl+C to stop")

	// Wait for shutdown signal
	select {
	case sig := <-sigChan:
		logger.Info("Received shutdown signal", "signal", sig)
	case <-ctx.Done():
		logger.Info("Context cancelled")
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// TODO: Stop the engine gracefully
	// if err := engine.Stop(shutdownCtx); err != nil {
	//     logger.Error("Error during shutdown", "error", err)
	// }

	logger.Info("Goclaw stopped gracefully")
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
}
