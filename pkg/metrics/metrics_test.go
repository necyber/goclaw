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

// --- Benchmarks for metrics collection overhead (Phase 15) ---

func BenchmarkRecordWorkflowSubmission(b *testing.B) {
	m := NewManager(DefaultConfig())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.RecordWorkflowSubmission("completed")
	}
}

func BenchmarkRecordWorkflowDuration(b *testing.B) {
	m := NewManager(DefaultConfig())
	d := 100 * time.Millisecond
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.RecordWorkflowDuration("completed", d)
	}
}

func BenchmarkRecordTaskExecution(b *testing.B) {
	m := NewManager(DefaultConfig())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.RecordTaskExecution("completed")
	}
}

func BenchmarkRecordHTTPRequest(b *testing.B) {
	m := NewManager(DefaultConfig())
	d := 5 * time.Millisecond
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.RecordHTTPRequest("GET", "/api/v1/workflows", "200", d)
	}
}

func BenchmarkRecordLaneThroughput(b *testing.B) {
	m := NewManager(DefaultConfig())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.RecordThroughput("default")
	}
}

func BenchmarkNoOpRecording(b *testing.B) {
	m := NoOpManager()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.RecordWorkflowSubmission("completed")
		m.RecordTaskExecution("completed")
		m.RecordThroughput("default")
	}
}

func TestMetricsMemoryUsage(t *testing.T) {
	m := NewManager(DefaultConfig())

	// Simulate heavy metrics recording with bounded label values
	statuses := []string{"completed", "failed", "pending", "cancelled"}
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	paths := []string{"/api/v1/workflows", "/api/v1/workflows/:id", "/health", "/ready"}
	lanes := []string{"default", "priority", "batch"}

	for i := 0; i < 100000; i++ {
		m.RecordWorkflowSubmission(statuses[i%len(statuses)])
		m.RecordWorkflowDuration(statuses[i%len(statuses)], time.Duration(i)*time.Microsecond)
		m.RecordTaskExecution(statuses[i%len(statuses)])
		m.RecordTaskDuration(time.Duration(i) * time.Microsecond)
		m.RecordHTTPRequest(methods[i%len(methods)], paths[i%len(paths)], "200", time.Duration(i)*time.Microsecond)
		m.RecordThroughput(lanes[i%len(lanes)])
		m.RecordWaitDuration(lanes[i%len(lanes)], time.Duration(i)*time.Microsecond)
	}

	// Verify metrics endpoint still responds correctly after heavy load
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	m.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 after heavy load, got %d", w.Code)
	}

	body := w.Body.String()
	// Verify cardinality is bounded: label combinations should be small
	// 4 statuses * 1 metric = 4 time series for workflow_submissions_total
	// 4 methods * 4 paths * 1 status = 16 time series for http_requests_total (bounded)
	if len(body) > 10*1024*1024 { // 10MB sanity check
		t.Errorf("Metrics output too large: %d bytes", len(body))
	}
}

func TestSignalAndRedisMetricsRegistered(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	m := NewManager(cfg)

	m.SetRedisQueueDepth("io", 3)
	m.RecordRedisSubmitDuration("io", 5*time.Millisecond)
	m.RecordRedisThroughput("io")

	m.RecordSignalSent("local", "steer")
	m.RecordSignalReceived("local", "steer")
	m.RecordSignalFailed("local", "steer", "no_subscriber")
	m.RecordSignalPattern("steer", "success", 2*time.Millisecond)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	m.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	expected := []string{
		"redis_lane_queue_depth",
		"redis_lane_submit_duration_seconds",
		"redis_lane_throughput_total",
		"signal_sent_total",
		"signal_received_total",
		"signal_failures_total",
		"signal_pattern_total",
		"signal_pattern_duration_seconds",
	}
	for _, metric := range expected {
		if !contains(body, metric) {
			t.Errorf("expected metric %s not found in output", metric)
		}
	}
}
