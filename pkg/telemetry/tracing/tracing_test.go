package tracing

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/goclaw/goclaw/config"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type mockExporter struct {
	shutdownCalled bool
}

func (m *mockExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error {
	return nil
}

func (m *mockExporter) Shutdown(context.Context) error {
	m.shutdownCalled = true
	return nil
}

func TestInitDisabledDoesNotCreateExporter(t *testing.T) {
	origFactory := newOTLPExporter
	t.Cleanup(func() { newOTLPExporter = origFactory })

	called := false
	newOTLPExporter = func(context.Context, config.TracingConfig) (sdktrace.SpanExporter, error) {
		called = true
		return &mockExporter{}, nil
	}

	shutdown, err := Init(context.Background(), config.TracingConfig{
		Enabled: false,
	}, "goclaw", "test")
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if called {
		t.Fatal("expected exporter factory not to be called when tracing is disabled")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
}

func TestInitEnabledRequiresEndpoint(t *testing.T) {
	_, err := Init(context.Background(), config.TracingConfig{
		Enabled:    true,
		Exporter:   "otlpgrpc",
		Endpoint:   "",
		Timeout:    5 * time.Second,
		Sampler:    "always_on",
		SampleRate: 1.0,
	}, "goclaw", "test")
	if err == nil {
		t.Fatal("expected error for missing endpoint")
	}
	if !strings.Contains(err.Error(), "endpoint") {
		t.Fatalf("expected endpoint error, got %v", err)
	}
}

func TestInitEnabledSuccessAndShutdown(t *testing.T) {
	origFactory := newOTLPExporter
	t.Cleanup(func() { newOTLPExporter = origFactory })

	exp := &mockExporter{}
	newOTLPExporter = func(context.Context, config.TracingConfig) (sdktrace.SpanExporter, error) {
		return exp, nil
	}

	shutdown, err := Init(context.Background(), config.TracingConfig{
		Enabled:    true,
		Exporter:   "otlpgrpc",
		Endpoint:   "http://localhost:4317/v1/traces",
		Headers:    map[string]string{"x-test": "1"},
		Timeout:    5 * time.Second,
		Sampler:    "parentbased_traceidratio",
		SampleRate: 0.1,
	}, "goclaw", "test")
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := shutdown(ctx); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
	if !exp.shutdownCalled {
		t.Fatal("expected exporter shutdown to be called")
	}
}

func TestSelectSampler(t *testing.T) {
	if got := selectSampler(config.TracingConfig{Sampler: "always_on"}).Description(); !strings.Contains(got, "AlwaysOnSampler") {
		t.Fatalf("unexpected always_on sampler description: %s", got)
	}
	if got := selectSampler(config.TracingConfig{Sampler: "always_off"}).Description(); !strings.Contains(got, "AlwaysOffSampler") {
		t.Fatalf("unexpected always_off sampler description: %s", got)
	}
	if got := selectSampler(config.TracingConfig{Sampler: "parentbased_traceidratio", SampleRate: 0.25}).Description(); !strings.Contains(strings.ToLower(got), "parentbased") {
		t.Fatalf("unexpected ratio sampler description: %s", got)
	}
}

func TestNormalizeEndpoint(t *testing.T) {
	if got := normalizeEndpoint("localhost:4317"); got != "localhost:4317" {
		t.Fatalf("normalizeEndpoint() = %q, want %q", got, "localhost:4317")
	}
	if got := normalizeEndpoint("http://localhost:4317/v1/traces"); got != "localhost:4317" {
		t.Fatalf("normalizeEndpoint() = %q, want %q", got, "localhost:4317")
	}
}
