package metrics

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestTraceExemplarLabels_WithSpan(t *testing.T) {
	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SpanID:     trace.SpanID{9, 8, 7, 6, 5, 4, 3, 2},
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)

	labels, ok := traceExemplarLabels(ctx)
	if !ok {
		t.Fatal("expected exemplar labels from valid span context")
	}
	if labels["trace_id"] != spanCtx.TraceID().String() {
		t.Fatalf("expected trace_id %s, got %s", spanCtx.TraceID().String(), labels["trace_id"])
	}
	if labels["span_id"] != spanCtx.SpanID().String() {
		t.Fatalf("expected span_id %s, got %s", spanCtx.SpanID().String(), labels["span_id"])
	}
}

func TestTraceExemplarLabels_WithoutSpan(t *testing.T) {
	labels, ok := traceExemplarLabels(context.Background())
	if ok {
		t.Fatalf("expected no exemplar labels without span, got %v", labels)
	}
}
