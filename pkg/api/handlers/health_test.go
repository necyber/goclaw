package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goclaw/goclaw/config"
	"github.com/goclaw/goclaw/pkg/engine"
	"github.com/goclaw/goclaw/pkg/logger"
)

func TestHealthHandler_Health(t *testing.T) {
	// Create test engine
	cfg := &config.Config{
		App: config.AppConfig{
			Name:        "test",
			Environment: "development",
		},
		Orchestration: config.OrchestrationConfig{
			MaxAgents: 10,
		},
	}
	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})

	eng, err := engine.New(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Start engine
	ctx := context.Background()
	if err := eng.Start(ctx); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	defer eng.Stop(ctx)

	handler := NewHealthHandler(eng)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Health() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestHealthHandler_Ready(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			Name:        "test",
			Environment: "development",
		},
		Orchestration: config.OrchestrationConfig{
			MaxAgents: 10,
		},
	}
	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})

	eng, _ := engine.New(cfg, log)
	ctx := context.Background()
	eng.Start(ctx)
	defer eng.Stop(ctx)

	handler := NewHealthHandler(eng)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	handler.Ready(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Ready() status = %v, want %v", w.Code, http.StatusOK)
	}
}

