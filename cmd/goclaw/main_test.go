package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/goclaw/goclaw/config"
	"github.com/goclaw/goclaw/pkg/api"
	"github.com/goclaw/goclaw/pkg/api/handlers"
	"github.com/goclaw/goclaw/pkg/engine"
	"github.com/goclaw/goclaw/pkg/logger"
)

func TestServerStartup(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		App: config.AppConfig{
			Name:        "test",
			Environment: "test",
		},
		Server: config.ServerConfig{
			Host: "127.0.0.1",
			Port: 18080, // Use different port for testing
			HTTP: config.HTTPConfig{
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
				IdleTimeout:  60 * time.Second,
			},
			CORS: config.CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowedHeaders: []string{"*"},
			},
		},
		Orchestration: config.OrchestrationConfig{
			MaxAgents: 10,
		},
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
	}

	// Initialize logger
	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})

	// Create and start engine
	ctx := context.Background()
	eng, err := engine.New(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	if err := eng.Start(ctx); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	defer eng.Stop(ctx)

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
		if err := httpServer.Start(); err != nil && err != http.ErrServerClosed {
			serverErrChan <- err
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Check if server started without errors
	select {
	case err := <-serverErrChan:
		t.Fatalf("Server failed to start: %v", err)
	default:
		// Server started successfully
	}

	// Test health endpoint
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", cfg.Server.Port))
	if err != nil {
		t.Fatalf("Failed to call health endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Health endpoint returned status %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Test ready endpoint
	resp, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/ready", cfg.Server.Port))
	if err != nil {
		t.Fatalf("Failed to call ready endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Ready endpoint returned status %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Test status endpoint
	resp, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/status", cfg.Server.Port))
	if err != nil {
		t.Fatalf("Failed to call status endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status endpoint returned status %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		t.Errorf("Failed to shutdown server: %v", err)
	}
}

func TestBuildOverrides(t *testing.T) {
	// Save original values
	origAppName := *appName
	origServerPort := *serverPort
	origLogLevel := *logLevel
	origDebugMode := *debugMode

	// Restore original values after test
	defer func() {
		*appName = origAppName
		*serverPort = origServerPort
		*logLevel = origLogLevel
		*debugMode = origDebugMode
	}()

	// Test with no overrides
	*appName = ""
	*serverPort = 0
	*logLevel = ""
	*debugMode = false

	overrides := buildOverrides()
	if len(overrides) != 0 {
		t.Errorf("Expected empty overrides, got %d items", len(overrides))
	}

	// Test with all overrides
	*appName = "test-app"
	*serverPort = 9090
	*logLevel = "debug"
	*debugMode = true

	overrides = buildOverrides()
	if len(overrides) != 4 {
		t.Errorf("Expected 4 overrides, got %d", len(overrides))
	}

	if overrides["app.name"] != "test-app" {
		t.Errorf("Expected app.name=test-app, got %v", overrides["app.name"])
	}
	if overrides["server.port"] != 9090 {
		t.Errorf("Expected server.port=9090, got %v", overrides["server.port"])
	}
	if overrides["log.level"] != "debug" {
		t.Errorf("Expected log.level=debug, got %v", overrides["log.level"])
	}
	if overrides["app.debug"] != true {
		t.Errorf("Expected app.debug=true, got %v", overrides["app.debug"])
	}
}

func TestPrintVersion(t *testing.T) {
	// Redirect stdout to capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printVersion()

	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Check if output contains expected strings
	expectedStrings := []string{"Goclaw", "Version:", "Build Time:", "Git Commit:", "Go Version:"}
	for _, expected := range expectedStrings {
		if !contains(output, expected) {
			t.Errorf("Expected output to contain %q, but it didn't. Output: %s", expected, output)
		}
	}
}

func TestPrintHelp(t *testing.T) {
	// Redirect stdout to capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printHelp()

	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	buf := make([]byte, 2048)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Check if output contains expected strings
	expectedStrings := []string{"Goclaw", "Usage:", "Options:", "Examples:"}
	for _, expected := range expectedStrings {
		if !contains(output, expected) {
			t.Errorf("Expected output to contain %q, but it didn't. Output: %s", expected, output)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
