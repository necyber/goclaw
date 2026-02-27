package grpc

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	ggrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func TestServerStart_TracingEnabledCreatesSpan(t *testing.T) {
	recorder, shutdownOTel := setTestTracerProvider(t)
	defer shutdownOTel()

	srv := startTestGRPCServer(t, true)
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Stop(stopCtx)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := ggrpc.NewClient(srv.Address(), ggrpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer conn.Close()

	healthClient := grpc_health_v1.NewHealthClient(conn)
	if _, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{}); err != nil {
		t.Fatalf("health check error = %v", err)
	}

	spans := waitForEndedSpans(recorder, 1, 2*time.Second)
	if len(spans) == 0 {
		t.Fatal("expected at least one span when tracing is enabled")
	}

	found := false
	for _, span := range spans {
		if span.Name() == "/grpc.health.v1.Health/Check" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected health check span, got %d spans", len(spans))
	}
}

func TestServerStart_TracingDisabledNoSpan(t *testing.T) {
	recorder, shutdownOTel := setTestTracerProvider(t)
	defer shutdownOTel()

	srv := startTestGRPCServer(t, false)
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Stop(stopCtx)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := ggrpc.NewClient(srv.Address(), ggrpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer conn.Close()

	healthClient := grpc_health_v1.NewHealthClient(conn)
	if _, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{}); err != nil {
		t.Fatalf("health check error = %v", err)
	}

	spans := waitForEndedSpans(recorder, 1, 200*time.Millisecond)
	if len(spans) != 0 {
		t.Fatalf("expected no spans when tracing is disabled, got %d", len(spans))
	}
}

func startTestGRPCServer(t *testing.T, tracingEnabled bool) *Server {
	t.Helper()

	cfg := DefaultConfig()
	cfg.Address = "127.0.0.1:0"
	cfg.EnableTracing = tracingEnabled
	cfg.EnableHealthCheck = true

	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	return srv
}

func setTestTracerProvider(t *testing.T) (*tracetest.SpanRecorder, func()) {
	t.Helper()

	prevProvider := otel.GetTracerProvider()
	prevPropagator := otel.GetTextMapPropagator()
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return recorder, func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(prevProvider)
		otel.SetTextMapPropagator(prevPropagator)
	}
}

func waitForEndedSpans(recorder *tracetest.SpanRecorder, minCount int, timeout time.Duration) []sdktrace.ReadOnlySpan {
	deadline := time.Now().Add(timeout)
	for {
		spans := recorder.Ended()
		if len(spans) >= minCount {
			return spans
		}
		if time.Now().After(deadline) {
			return spans
		}
		time.Sleep(10 * time.Millisecond)
	}
}
