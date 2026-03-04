package interceptors

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

type fakeCounter struct {
	incCalls      int
	addTotal      float64
	exemplarCalls int
	lastExemplar  prometheus.Labels
	lastExemplarV float64
}

func (f *fakeCounter) Inc() { f.incCalls++ }
func (f *fakeCounter) Add(v float64) {
	f.addTotal += v
}
func (f *fakeCounter) AddWithExemplar(v float64, exemplar prometheus.Labels) {
	f.exemplarCalls++
	f.lastExemplarV = v
	f.lastExemplar = exemplar
}

type fakeCounterNoExemplar struct {
	incCalls int
	addTotal float64
}

func (f *fakeCounterNoExemplar) Inc() { f.incCalls++ }
func (f *fakeCounterNoExemplar) Add(v float64) {
	f.addTotal += v
}

type fakeHistogram struct {
	observed      []float64
	exemplarCalls int
	lastExemplar  prometheus.Labels
}

func (f *fakeHistogram) Observe(v float64) {
	f.observed = append(f.observed, v)
}
func (f *fakeHistogram) ObserveWithExemplar(v float64, exemplar prometheus.Labels) {
	f.exemplarCalls++
	f.observed = append(f.observed, v)
	f.lastExemplar = exemplar
}

type fakeHistogramNoExemplar struct {
	observed []float64
}

func (f *fakeHistogramNoExemplar) Observe(v float64) {
	f.observed = append(f.observed, v)
}

func TestTraceExemplarLabels_WithSpan(t *testing.T) {
	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		SpanID:     trace.SpanID{2, 2, 2, 2, 2, 2, 2, 2},
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)

	labels, ok := traceExemplarLabels(ctx)
	if !ok {
		t.Fatal("expected trace exemplar labels for valid span context")
	}
	if labels["trace_id"] != spanCtx.TraceID().String() {
		t.Fatalf("trace_id = %s, want %s", labels["trace_id"], spanCtx.TraceID().String())
	}
	if labels["span_id"] != spanCtx.SpanID().String() {
		t.Fatalf("span_id = %s, want %s", labels["span_id"], spanCtx.SpanID().String())
	}
}

func TestTraceExemplarLabels_WithoutSpan(t *testing.T) {
	labels, ok := traceExemplarLabels(context.Background())
	if ok {
		t.Fatalf("expected no exemplar labels, got %v", labels)
	}
}

func TestIncrementCounter_ExemplarAndFallback(t *testing.T) {
	exemplar := prometheus.Labels{"trace_id": "trace", "span_id": "span"}

	withExemplar := &fakeCounter{}
	usedExemplar := incrementCounter(withExemplar, true, exemplar)
	if !usedExemplar {
		t.Fatal("expected exemplar path for exemplar-capable counter")
	}
	if withExemplar.exemplarCalls != 1 {
		t.Fatalf("expected 1 exemplar call, got %d", withExemplar.exemplarCalls)
	}

	withoutExemplar := &fakeCounterNoExemplar{}
	usedExemplar = incrementCounter(withoutExemplar, true, exemplar)
	if usedExemplar {
		t.Fatal("expected fallback path for non-exemplar counter")
	}
	if withoutExemplar.incCalls != 1 {
		t.Fatalf("expected fallback Inc() call, got %d", withoutExemplar.incCalls)
	}
}

func TestObserveHistogram_ExemplarAndFallback(t *testing.T) {
	exemplar := prometheus.Labels{"trace_id": "trace", "span_id": "span"}

	withExemplar := &fakeHistogram{}
	usedExemplar := observeHistogram(withExemplar, 0.25, true, exemplar)
	if !usedExemplar {
		t.Fatal("expected exemplar path for exemplar-capable histogram")
	}
	if withExemplar.exemplarCalls != 1 {
		t.Fatalf("expected 1 exemplar call, got %d", withExemplar.exemplarCalls)
	}

	withoutExemplar := &fakeHistogramNoExemplar{}
	usedExemplar = observeHistogram(withoutExemplar, 0.5, true, exemplar)
	if usedExemplar {
		t.Fatal("expected fallback path for non-exemplar histogram")
	}
	if len(withoutExemplar.observed) != 1 {
		t.Fatalf("expected 1 observed value, got %d", len(withoutExemplar.observed))
	}
}

func TestAddCounter_ExemplarAndFallback(t *testing.T) {
	exemplar := prometheus.Labels{"trace_id": "trace", "span_id": "span"}

	withExemplar := &fakeCounter{}
	usedExemplar := addCounter(withExemplar, 2, true, exemplar)
	if !usedExemplar {
		t.Fatal("expected exemplar path for exemplar-capable add counter")
	}
	if withExemplar.exemplarCalls != 1 {
		t.Fatalf("expected 1 exemplar add call, got %d", withExemplar.exemplarCalls)
	}

	withoutExemplar := &fakeCounterNoExemplar{}
	usedExemplar = addCounter(withoutExemplar, 3, true, exemplar)
	if usedExemplar {
		t.Fatal("expected fallback add path for non-exemplar counter")
	}
	if withoutExemplar.addTotal != 3 {
		t.Fatalf("expected addTotal=3, got %f", withoutExemplar.addTotal)
	}
}
