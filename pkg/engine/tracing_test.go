package engine

import (
	"context"
	"testing"
	"time"

	"github.com/goclaw/goclaw/pkg/dag"
	"github.com/goclaw/goclaw/pkg/storage/memory"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestRuntimeTracing_WorkflowAndLaneSpans(t *testing.T) {
	recorder, shutdown := setEngineTracingProvider(t)
	defer shutdown()

	eng, err := New(minConfig(), nil, memory.NewMemoryStorage())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	ctx := context.Background()
	if err := eng.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = eng.Stop(stopCtx)
	}()

	wf := &Workflow{
		ID: "trace-workflow",
		Tasks: []*dag.Task{
			{ID: "task-1", Name: "task-1", Agent: "test", Lane: "default"},
		},
		TaskFns: map[string]func(context.Context) error{
			"task-1": func(context.Context) error { return nil },
		},
	}

	if _, err := eng.Submit(context.Background(), wf); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}

	spans := waitEngineSpans(recorder, 5, 1*time.Second)
	if !containsEngineSpan(spans, spanWorkflowExecute) {
		t.Fatalf("expected span %q", spanWorkflowExecute)
	}
	if !containsEngineSpan(spans, spanWorkflowLayer) {
		t.Fatalf("expected span %q", spanWorkflowLayer)
	}
	if !containsEngineSpan(spans, spanTaskSchedule) {
		t.Fatalf("expected span %q", spanTaskSchedule)
	}
	if !containsEngineSpan(spans, spanTaskRun) {
		t.Fatalf("expected span %q", spanTaskRun)
	}
	if !containsEngineSpan(spans, spanLaneWait) {
		t.Fatalf("expected span %q", spanLaneWait)
	}
}

func setEngineTracingProvider(t *testing.T) (*tracetest.SpanRecorder, func()) {
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

func waitEngineSpans(recorder *tracetest.SpanRecorder, minCount int, timeout time.Duration) []sdktrace.ReadOnlySpan {
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

func containsEngineSpan(spans []sdktrace.ReadOnlySpan, name string) bool {
	for _, span := range spans {
		if span.Name() == name {
			return true
		}
	}
	return false
}
