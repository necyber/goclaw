package interceptors

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goclaw/goclaw/pkg/logger"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

func TestLoggingUnaryInterceptor_IncludesTraceCorrelation(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "grpc-logging.json")
	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: logPath,
	})
	t.Cleanup(func() { _ = log.Close() })

	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		SpanID:     trace.SpanID{2, 2, 2, 2, 2, 2, 2, 2},
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)
	ctx = log.WithContext(ctx)
	ctx = withRequestID(ctx, "req-test")

	interceptor := LoggingUnaryInterceptor()
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/m"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log output: %v", err)
	}
	logText := string(content)
	if !strings.Contains(logText, spanCtx.TraceID().String()) {
		t.Fatalf("expected trace_id in logs, got: %s", logText)
	}
	if !strings.Contains(logText, spanCtx.SpanID().String()) {
		t.Fatalf("expected span_id in logs, got: %s", logText)
	}
}
