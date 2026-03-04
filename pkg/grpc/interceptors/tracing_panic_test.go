package interceptors

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestTracingUnaryInterceptor_PanicMapsToErrorSpan(t *testing.T) {
	recorder, shutdown := setTracingPanicTestProvider(t)
	defer shutdown()

	recovery := RecoveryUnaryInterceptor()
	tracing := TracingUnaryInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: "/svc/m"}

	_, err := recovery(context.Background(), nil, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return tracing(ctx, req, info, func(ctx context.Context, req interface{}) (interface{}, error) {
			panic("boom")
		})
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected Internal after panic recovery, got %v", status.Code(err))
	}

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	span := spans[0]
	if got, want := span.Status().Code, otelcodes.Error; got != want {
		t.Fatalf("span status code = %v, want %v", got, want)
	}
	if !hasIntAttribute(span.Attributes(), "rpc.grpc.status_code", int(codes.Internal)) {
		t.Fatalf("expected rpc.grpc.status_code=%d", codes.Internal)
	}
	if !hasStringAttribute(span.Attributes(), "error.type", "panic") {
		t.Fatal("expected panic error.type attribute")
	}
}

func TestTracingStreamInterceptor_PanicMapsToErrorSpan(t *testing.T) {
	recorder, shutdown := setTracingPanicTestProvider(t)
	defer shutdown()

	recovery := RecoveryStreamInterceptor()
	tracing := TracingStreamInterceptor()
	info := &grpc.StreamServerInfo{FullMethod: "/svc/stream"}
	stream := &testServerStream{ctx: context.Background()}

	err := recovery(nil, stream, info, func(srv interface{}, ss grpc.ServerStream) error {
		return tracing(srv, ss, info, func(srv interface{}, ss grpc.ServerStream) error {
			panic("stream boom")
		})
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected Internal after panic recovery, got %v", status.Code(err))
	}

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	span := spans[0]
	if got, want := span.Status().Code, otelcodes.Error; got != want {
		t.Fatalf("span status code = %v, want %v", got, want)
	}
	if !hasIntAttribute(span.Attributes(), "rpc.grpc.status_code", int(codes.Internal)) {
		t.Fatalf("expected rpc.grpc.status_code=%d", codes.Internal)
	}
	if !hasStringAttribute(span.Attributes(), "error.type", "panic") {
		t.Fatal("expected panic error.type attribute")
	}
}

func setTracingPanicTestProvider(t *testing.T) (*tracetest.SpanRecorder, func()) {
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

func hasStringAttribute(attrs []attribute.KeyValue, key, value string) bool {
	for _, attr := range attrs {
		if string(attr.Key) == key && attr.Value.AsString() == value {
			return true
		}
	}
	return false
}
