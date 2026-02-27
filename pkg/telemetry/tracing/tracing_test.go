package tracing

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/goclaw/goclaw/config"
	"go.opentelemetry.io/otel"
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

type failingExporter struct {
	shutdownCalled bool
	exportCalls    int
}

func (f *failingExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error {
	f.exportCalls++
	return errors.New("export unavailable")
}

func (f *failingExporter) Shutdown(context.Context) error {
	f.shutdownCalled = true
	return nil
}

type blockingShutdownExporter struct{}

func (b *blockingShutdownExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error {
	return nil
}

func (b *blockingShutdownExporter) Shutdown(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
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

func TestInitEnabled_ExporterFailureIsIsolated(t *testing.T) {
	origFactory := newOTLPExporter
	origReporter := reportExporterFailure
	t.Cleanup(func() {
		newOTLPExporter = origFactory
		reportExporterFailure = origReporter
	})

	exporter := &failingExporter{}
	newOTLPExporter = func(context.Context, config.TracingConfig) (sdktrace.SpanExporter, error) {
		return exporter, nil
	}

	reported := 0
	reportExporterFailure = func(err error, exporterKind, endpoint string, spanCount int) {
		reported++
		if exporterKind == "" {
			t.Fatal("expected exporter kind in failure report")
		}
		if endpoint == "" {
			t.Fatal("expected endpoint in failure report")
		}
		if spanCount <= 0 {
			t.Fatalf("expected positive span_count, got %d", spanCount)
		}
		if err == nil {
			t.Fatal("expected non-nil export error in report")
		}
	}

	shutdown, err := Init(context.Background(), config.TracingConfig{
		Enabled:    true,
		Exporter:   "otlpgrpc",
		Endpoint:   "localhost:4317",
		Timeout:    200 * time.Millisecond,
		Sampler:    "always_on",
		SampleRate: 1.0,
	}, "goclaw", "test")
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	_, span := otel.Tracer("test").Start(context.Background(), "request-path")
	span.End()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := shutdown(ctx); err != nil {
		t.Fatalf("shutdown() should not fail on exporter delivery failure: %v", err)
	}
	if exporter.exportCalls == 0 {
		t.Fatal("expected exporter to receive export calls")
	}
	if reported == 0 {
		t.Fatal("expected exporter failure to be reported")
	}
}

func TestShutdown_TimeoutIsBounded(t *testing.T) {
	origFactory := newOTLPExporter
	t.Cleanup(func() { newOTLPExporter = origFactory })

	newOTLPExporter = func(context.Context, config.TracingConfig) (sdktrace.SpanExporter, error) {
		return &blockingShutdownExporter{}, nil
	}

	shutdown, err := Init(context.Background(), config.TracingConfig{
		Enabled:    true,
		Exporter:   "otlpgrpc",
		Endpoint:   "localhost:4317",
		Timeout:    100 * time.Millisecond,
		Sampler:    "always_on",
		SampleRate: 1.0,
	}, "goclaw", "test")
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err = shutdown(ctx)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected shutdown() to return timeout-related error")
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("shutdown exceeded bounded timeout, elapsed=%v", elapsed)
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
