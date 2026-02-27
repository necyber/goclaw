package saga

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestSagaTracing_ForwardAndCompensationSpans(t *testing.T) {
	recorder, shutdown := setSagaTracingProvider(t)
	defer shutdown()

	def, err := New("trace-saga").
		Step("a",
			Action(func(context.Context, *StepContext) (any, error) { return "ok", nil }),
			Compensate(func(context.Context, *CompensationContext) error { return nil }),
		).
		Step("b",
			Action(func(context.Context, *StepContext) (any, error) { return nil, errors.New("boom") }),
			DependsOn("a"),
		).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	orchestrator := NewSagaOrchestrator()
	if _, err := orchestrator.ExecuteWithID(context.Background(), "saga-trace-1", def, nil); err == nil {
		t.Fatal("expected ExecuteWithID() to return step failure")
	}

	spans := waitSagaSpans(recorder, 4, 1*time.Second)
	if !containsSagaSpan(spans, spanSagaExecuteForward) {
		t.Fatalf("expected span %q", spanSagaExecuteForward)
	}
	if !containsSagaSpan(spans, spanSagaStepForward) {
		t.Fatalf("expected span %q", spanSagaStepForward)
	}
	if !containsSagaSpan(spans, spanSagaExecuteCompensate) {
		t.Fatalf("expected span %q", spanSagaExecuteCompensate)
	}
	if !containsSagaSpan(spans, spanSagaStepCompensate) {
		t.Fatalf("expected span %q", spanSagaStepCompensate)
	}
}

func TestSagaTracing_RecoverySpan(t *testing.T) {
	recorder, shutdown := setSagaTracingProvider(t)
	defer shutdown()

	def, err := New("trace-recovery").
		Step("a", Action(func(context.Context, *StepContext) (any, error) { return "ok", nil })).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	orchestrator := NewSagaOrchestrator()
	checkpoint := &Checkpoint{
		SagaID:         "recover-trace-1",
		DefinitionName: def.Name,
		State:          SagaStateRunning,
		CompletedSteps: []string{},
		StepResults:    map[string]any{},
		LastUpdated:    time.Now().UTC(),
	}

	instance, err := orchestrator.ResumeFromCheckpoint(context.Background(), def, checkpoint, nil)
	if err != nil {
		t.Fatalf("ResumeFromCheckpoint() error = %v", err)
	}
	if instance.State != SagaStateCompleted {
		t.Fatalf("expected completed state, got %s", instance.State)
	}

	spans := waitSagaSpans(recorder, 1, 1*time.Second)
	if !containsSagaSpan(spans, spanSagaRecoveryResume) {
		t.Fatalf("expected span %q", spanSagaRecoveryResume)
	}
}

func setSagaTracingProvider(t *testing.T) (*tracetest.SpanRecorder, func()) {
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

func waitSagaSpans(recorder *tracetest.SpanRecorder, minCount int, timeout time.Duration) []sdktrace.ReadOnlySpan {
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

func containsSagaSpan(spans []sdktrace.ReadOnlySpan, name string) bool {
	for _, span := range spans {
		if span.Name() == name {
			return true
		}
	}
	return false
}
