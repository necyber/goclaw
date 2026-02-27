package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type mockMetricsRecorder struct {
	requests    int
	activeConns int
}

func (m *mockMetricsRecorder) RecordHTTPRequest(method, path, status string, duration time.Duration) {
	m.requests++
}

func (m *mockMetricsRecorder) IncActiveConnections() {
	m.activeConns++
}

func (m *mockMetricsRecorder) DecActiveConnections() {
	m.activeConns--
}

type traceAwareMockMetricsRecorder struct {
	records     int
	baseRecords int
	traceID     string
	spanID      string
	activeConns int
}

func (m *traceAwareMockMetricsRecorder) RecordHTTPRequest(method, path, status string, duration time.Duration) {
	m.baseRecords++
}

func (m *traceAwareMockMetricsRecorder) RecordHTTPRequestWithContext(ctx context.Context, method, path, status string, duration time.Duration) {
	m.records++
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		m.traceID = spanCtx.TraceID().String()
		m.spanID = spanCtx.SpanID().String()
	}
}

func (m *traceAwareMockMetricsRecorder) IncActiveConnections() {
	m.activeConns++
}

func (m *traceAwareMockMetricsRecorder) DecActiveConnections() {
	m.activeConns--
}

func TestMetrics_Success(t *testing.T) {
	mock := &mockMetricsRecorder{}

	handler := Metrics(mock)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if mock.requests != 1 {
		t.Errorf("Expected 1 request recorded, got %d", mock.requests)
	}

	if mock.activeConns != 0 {
		t.Errorf("Expected active connections to be 0 after request, got %d", mock.activeConns)
	}
}

func TestMetrics_SkipMetricsEndpoint(t *testing.T) {
	mock := &mockMetricsRecorder{}

	handler := Metrics(mock)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if mock.requests != 0 {
		t.Errorf("Expected 0 requests recorded for /metrics endpoint, got %d", mock.requests)
	}
}

func TestMetrics_CaptureStatusCode(t *testing.T) {
	mock := &mockMetricsRecorder{}

	handler := Metrics(mock)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest("GET", "/api/v1/notfound", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	if mock.requests != 1 {
		t.Errorf("Expected 1 request recorded, got %d", mock.requests)
	}
}

func TestMetrics_HandlePanic(t *testing.T) {
	mock := &mockMetricsRecorder{}

	handler := Metrics(mock)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest("GET", "/api/v1/panic", nil)
	w := httptest.NewRecorder()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic to be propagated")
		}
	}()

	handler.ServeHTTP(w, req)

	// Should record metrics even on panic
	if mock.requests != 1 {
		t.Errorf("Expected 1 request recorded after panic, got %d", mock.requests)
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/api/v1/workflows/123", "/api/v1/workflows/:id"},
		{"/api/v1/workflows/550e8400-e29b-41d4-a716-446655440000", "/api/v1/workflows/:id"},
		{"/api/v1/workflows/123/tasks/456", "/api/v1/workflows/:id/tasks/:id"},
		{"/api/v1/workflows", "/api/v1/workflows"},
		{"/health", "/health"},
	}

	for _, tt := range tests {
		result := normalizePath(tt.input)
		if result != tt.expected {
			t.Errorf("normalizePath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestMetricsResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	mw := &metricsResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	mw.WriteHeader(http.StatusCreated)

	if mw.statusCode != http.StatusCreated {
		t.Errorf("Expected status code 201, got %d", mw.statusCode)
	}

	if !mw.written {
		t.Error("Expected written flag to be true")
	}

	// Second call should not change status
	mw.WriteHeader(http.StatusBadRequest)
	if mw.statusCode != http.StatusCreated {
		t.Errorf("Expected status code to remain 201, got %d", mw.statusCode)
	}
}

func TestMetricsResponseWriter_Write(t *testing.T) {
	w := httptest.NewRecorder()
	mw := &metricsResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	data := []byte("test data")
	n, err := mw.Write(data)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected %d bytes written, got %d", len(data), n)
	}

	if !mw.written {
		t.Error("Expected written flag to be true")
	}
}

func TestMetrics_TraceAwareRecorderWithTraceContext(t *testing.T) {
	mock := &traceAwareMockMetricsRecorder{}
	handler := Metrics(mock)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		SpanID:     trace.SpanID{2, 2, 2, 2, 2, 2, 2, 2},
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)
	req := httptest.NewRequest("GET", "/api/v1/workflows/123", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if mock.records != 1 {
		t.Fatalf("expected context-aware recorder to be called once, got %d", mock.records)
	}
	if mock.baseRecords != 0 {
		t.Fatalf("expected base recorder path not called, got %d", mock.baseRecords)
	}
	if mock.traceID != spanCtx.TraceID().String() {
		t.Fatalf("expected trace_id %s, got %s", spanCtx.TraceID().String(), mock.traceID)
	}
	if mock.spanID != spanCtx.SpanID().String() {
		t.Fatalf("expected span_id %s, got %s", spanCtx.SpanID().String(), mock.spanID)
	}
}

func TestMetrics_TraceAwareRecorderWithoutTraceContext(t *testing.T) {
	mock := &traceAwareMockMetricsRecorder{}
	handler := Metrics(mock)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/workflows/123", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if mock.records != 1 {
		t.Fatalf("expected context-aware recorder to be called once, got %d", mock.records)
	}
	if mock.traceID != "" || mock.spanID != "" {
		t.Fatalf("expected empty trace correlation without span, got trace_id=%q span_id=%q", mock.traceID, mock.spanID)
	}
}
