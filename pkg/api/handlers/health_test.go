package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goclaw/goclaw/config"
	"github.com/goclaw/goclaw/pkg/api/models"
	"github.com/goclaw/goclaw/pkg/engine"
	"github.com/goclaw/goclaw/pkg/logger"
	"github.com/goclaw/goclaw/pkg/storage/memory"
)

func createHealthTestEngine(t *testing.T, start bool) (*engine.Engine, context.Context, func()) {
	t.Helper()

	cfg := &config.Config{
		App: config.AppConfig{
			Name:        "test",
			Environment: "development",
			Version:     "test-version",
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

	eng, err := engine.New(cfg, log, memory.NewMemoryStorage())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	ctx := context.Background()
	if start {
		if err := eng.Start(ctx); err != nil {
			t.Fatalf("Failed to start engine: %v", err)
		}
	}

	cleanup := func() {
		_ = eng.Stop(ctx)
	}

	return eng, ctx, cleanup
}

func TestHealthHandler_Health_Healthy(t *testing.T) {
	eng, _, cleanup := createHealthTestEngine(t, true)
	defer cleanup()

	handler := NewHealthHandler(eng)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Health() status = %v, want %v", w.Code, http.StatusOK)
	}

	var resp models.HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "healthy" {
		t.Fatalf("status = %s, want healthy", resp.Status)
	}
	if resp.Timestamp.IsZero() {
		t.Fatal("expected timestamp to be set")
	}
	if resp.Error != "" {
		t.Fatalf("error = %q, want empty", resp.Error)
	}
}

func TestHealthHandler_Health_Unhealthy(t *testing.T) {
	eng, _, cleanup := createHealthTestEngine(t, false)
	defer cleanup()

	handler := NewHealthHandler(eng)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("Health() status = %v, want %v", w.Code, http.StatusServiceUnavailable)
	}

	var resp models.HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "unhealthy" {
		t.Fatalf("status = %s, want unhealthy", resp.Status)
	}
	if resp.Timestamp.IsZero() {
		t.Fatal("expected timestamp to be set")
	}
	if resp.Error == "" {
		t.Fatal("expected error detail for unhealthy response")
	}
}

func TestHealthHandler_Ready(t *testing.T) {
	eng, _, cleanup := createHealthTestEngine(t, true)
	defer cleanup()

	handler := NewHealthHandler(eng)
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	handler.Ready(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Ready() status = %v, want %v", w.Code, http.StatusOK)
	}

	var resp models.ReadyResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "ready" {
		t.Fatalf("status = %s, want ready", resp.Status)
	}
	if resp.Timestamp.IsZero() {
		t.Fatal("expected timestamp to be set")
	}
	if _, ok := resp.Checks["engine"]; !ok {
		t.Fatal("expected engine check")
	}
	if _, ok := resp.Checks["storage"]; !ok {
		t.Fatal("expected storage check")
	}
}

func TestHealthHandler_Ready_NotReady(t *testing.T) {
	eng, _, cleanup := createHealthTestEngine(t, false)
	defer cleanup()

	handler := NewHealthHandler(eng)
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	handler.Ready(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("Ready() status = %v, want %v", w.Code, http.StatusServiceUnavailable)
	}

	var resp models.ReadyResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "not_ready" {
		t.Fatalf("status = %s, want not_ready", resp.Status)
	}
	if resp.Timestamp.IsZero() {
		t.Fatal("expected timestamp to be set")
	}
	if resp.Error == "" {
		t.Fatal("expected error detail")
	}
}

func TestHealthHandler_Status(t *testing.T) {
	eng, _, cleanup := createHealthTestEngine(t, true)
	defer cleanup()

	handler := NewHealthHandler(eng)
	time.Sleep(10 * time.Millisecond)

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()

	handler.Status(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Status() status = %v, want %v", w.Code, http.StatusOK)
	}

	var resp models.StatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status == "" {
		t.Fatal("expected status field")
	}
	if resp.Version != "test-version" {
		t.Fatalf("version = %q, want test-version", resp.Version)
	}
	if resp.Uptime == "" {
		t.Fatal("expected uptime field")
	}
	if resp.Timestamp.IsZero() {
		t.Fatal("expected timestamp to be set")
	}
	if resp.Runtime.State == "" {
		t.Fatal("expected runtime.state")
	}
	if resp.Runtime.GoVersion == "" {
		t.Fatal("expected runtime.go_version")
	}
	if _, ok := resp.Components["engine"]; !ok {
		t.Fatal("expected components.engine")
	}
	if _, ok := resp.Components["storage"]; !ok {
		t.Fatal("expected components.storage")
	}
}
