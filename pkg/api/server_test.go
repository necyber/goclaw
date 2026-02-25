package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/goclaw/goclaw/config"
	"github.com/goclaw/goclaw/pkg/api/handlers"
	"github.com/goclaw/goclaw/pkg/engine"
	"github.com/goclaw/goclaw/pkg/logger"
)

func TestNewHTTPServer(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
			HTTP: config.HTTPConfig{
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
				IdleTimeout:  120 * time.Second,
			},
			CORS: config.CORSConfig{
				Enabled: false,
			},
		},
	}

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})

	// Create engine and handlers
	eng, _ := engine.New(cfg, log)
	ctx := context.Background()
	eng.Start(ctx)
	defer eng.Stop(ctx)

	testHandlers := &Handlers{
		Workflow: handlers.NewWorkflowHandler(eng, log),
		Health:   handlers.NewHealthHandler(eng),
	}

	server := NewHTTPServer(cfg, log, testHandlers)

	if server == nil {
		t.Fatal("NewHTTPServer returned nil")
	}

	if server.server == nil {
		t.Error("HTTP server not initialized")
	}

	if server.router == nil {
		t.Error("Router not initialized")
	}
}

func TestHTTPServer_StartAndShutdown(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 18080, // Use different port to avoid conflicts
			HTTP: config.HTTPConfig{
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 5 * time.Second,
				IdleTimeout:  10 * time.Second,
			},
			CORS: config.CORSConfig{
				Enabled: false,
			},
		},
	}

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})

	// Create engine and handlers
	eng, _ := engine.New(cfg, log)
	ctx := context.Background()
	eng.Start(ctx)
	defer eng.Stop(ctx)

	testHandlers := &Handlers{
		Workflow: handlers.NewWorkflowHandler(eng, log),
		Health:   handlers.NewHealthHandler(eng),
	}

	server := NewHTTPServer(cfg, log, testHandlers)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start()
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Test if server is responding
	resp, err := http.Get("http://localhost:18080/health")
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Health check status = %v, want %v", resp.StatusCode, http.StatusOK)
	}

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	// Check that Start() returned without error
	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("Start() returned error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Start() did not return after shutdown")
	}
}

