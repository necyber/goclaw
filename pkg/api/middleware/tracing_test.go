package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestTracing_ContinueInboundTraceContext(t *testing.T) {
	recorder, shutdown := setTracingTestProvider(t)
	defer shutdown()

	parent := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		SpanID:     trace.SpanID{2, 2, 2, 2, 2, 2, 2, 2},
		TraceFlags: trace.FlagsSampled,
	})
	parentCtx := trace.ContextWithSpanContext(context.Background(), parent)
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(parentCtx, carrier)

	handler := Tracing(DefaultTracingOptions())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows", nil)
	for k, v := range carrier {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	spans := waitForHTTPSpans(recorder, 1, 500*time.Millisecond)
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if got, want := spans[0].Parent().TraceID(), parent.TraceID(); got != want {
		t.Fatalf("continued trace id = %s, want %s", got, want)
	}
}

func TestTracing_CreateRootWhenNoHeaders(t *testing.T) {
	recorder, shutdown := setTracingTestProvider(t)
	defer shutdown()

	handler := Tracing(DefaultTracingOptions())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	spans := waitForHTTPSpans(recorder, 1, 500*time.Millisecond)
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Parent().IsValid() {
		t.Fatal("expected root span when no inbound trace headers are present")
	}
}

func TestTracing_HTTPStatusMapping(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		wantStatusCode otelcodes.Code
	}{
		{name: "2xx is ok", statusCode: http.StatusOK, wantStatusCode: otelcodes.Ok},
		{name: "4xx is error", statusCode: http.StatusNotFound, wantStatusCode: otelcodes.Error},
		{name: "5xx is error", statusCode: http.StatusInternalServerError, wantStatusCode: otelcodes.Error},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder, shutdown := setTracingTestProvider(t)
			defer shutdown()

			handler := Tracing(DefaultTracingOptions())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))

			req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			spans := waitForHTTPSpans(recorder, 1, 500*time.Millisecond)
			if len(spans) != 1 {
				t.Fatalf("expected 1 span, got %d", len(spans))
			}
			if got, want := spans[0].Status().Code, tt.wantStatusCode; got != want {
				t.Fatalf("span status = %v, want %v", got, want)
			}
			if !hasAttributeValue(spans[0].Attributes(), "http.response.status_code", int64(tt.statusCode)) {
				t.Fatalf("missing http.response.status_code=%d", tt.statusCode)
			}
		})
	}
}

func TestTracing_SkipHealthEndpoints(t *testing.T) {
	recorder, shutdown := setTracingTestProvider(t)
	defer shutdown()

	handler := Tracing(DefaultTracingOptions())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	spans := waitForHTTPSpans(recorder, 1, 200*time.Millisecond)
	if len(spans) != 0 {
		t.Fatalf("expected no spans for /health, got %d", len(spans))
	}
}

func TestInjectOutboundTraceContext(t *testing.T) {
	_, shutdown := setTracingTestProvider(t)
	defer shutdown()

	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "outbound")
	defer span.End()

	req := httptest.NewRequest(http.MethodGet, "http://example.test/path", nil).WithContext(ctx)
	req.Header.Set("x-custom", "1")
	injected := InjectOutboundTraceContext(req)
	if injected == nil {
		t.Fatal("expected non-nil request")
	}
	if injected.Header.Get("traceparent") == "" {
		t.Fatal("expected traceparent header to be injected")
	}
	if injected.Header.Get("x-custom") != "1" {
		t.Fatal("expected existing headers to be preserved")
	}
}

func TestNewTracingRequest(t *testing.T) {
	_, shutdown := setTracingTestProvider(t)
	defer shutdown()

	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "new-request")
	defer span.End()

	req, err := NewTracingRequest(ctx, http.MethodGet, "http://example.test/items", nil)
	if err != nil {
		t.Fatalf("NewTracingRequest() error = %v", err)
	}
	if req.Header.Get("traceparent") == "" {
		t.Fatal("expected traceparent header on new request")
	}
}

func setTracingTestProvider(t *testing.T) (*tracetest.SpanRecorder, func()) {
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

func waitForHTTPSpans(recorder *tracetest.SpanRecorder, minCount int, timeout time.Duration) []sdktrace.ReadOnlySpan {
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

func hasAttributeValue(attrs []attribute.KeyValue, key string, want int64) bool {
	for _, attr := range attrs {
		if string(attr.Key) == key && attr.Value.AsInt64() == want {
			return true
		}
	}
	return false
}
