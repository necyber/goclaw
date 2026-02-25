package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true

	m := NewManager(cfg)
	if m == nil {
		t.Fatal("NewManager returned nil")
	}

	if !m.Enabled() {
		t.Error("Expected metrics to be enabled")
	}
}

func TestNewManager_Disabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false

	m := NewManager(cfg)
	if m == nil {
		t.Fatal("NewManager returned nil")
	}

	if m.Enabled() {
		t.Error("Expected metrics to be disabled")
	}
}

func TestMetricsHandler(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true

	m := NewManager(cfg)

	// Record some metrics
	m.RecordWorkflowSubmission("pending")
	m.RecordWorkflowSubmission("completed")
	m.RecordWorkflowDuration("completed", 5*time.Second)

	// Create test request
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	// Serve metrics
	m.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("Expected non-empty metrics output")
	}

	// Check for expected metrics
	expectedMetrics := []string{
		"workflow_submissions_total",
		"workflow_duration_seconds",
	}

	for _, metric := range expectedMetrics {
		if !contains(body, metric) {
			t.Errorf("Expected metric %s not found in output", metric)
		}
	}
}

func TestMetricsHandler_Disabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false

	m := NewManager(cfg)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	m.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 when disabled, got %d", w.Code)
	}
}

func TestStartServer(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Port = 19091 // Use different port for testing

	m := NewManager(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server in background
	errCh := make(chan error, 1)
	go func() {
		err := m.StartServer(ctx, cfg.Port, cfg.Path)
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Try to fetch metrics
	resp, err := http.Get("http://localhost:19091/metrics")
	if err != nil {
		t.Fatalf("Failed to fetch metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Cancel context to stop server
	cancel()

	// Check for errors
	select {
	case err := <-errCh:
		t.Errorf("Server error: %v", err)
	case <-time.After(1 * time.Second):
		// Server stopped cleanly
	}
}

func TestNoOpManager(t *testing.T) {
	m := NoOpManager()

	if m.Enabled() {
		t.Error("NoOpManager should not be enabled")
	}

	// These should not panic
	m.RecordWorkflowSubmission("test")
	m.RecordWorkflowDuration("test", time.Second)
	m.IncActiveWorkflows("test")
	m.DecActiveWorkflows("test")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || contains(s[1:], substr)))
}
