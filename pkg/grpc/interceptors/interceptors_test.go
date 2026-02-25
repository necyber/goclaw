package interceptors

import (
	"context"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type testServerStream struct {
	ctx      context.Context
	recvMsgs []interface{}
	sendErr  error
}

func (t *testServerStream) Context() context.Context { return t.ctx }
func (t *testServerStream) SetHeader(md metadata.MD) error {
	return nil
}
func (t *testServerStream) SendHeader(md metadata.MD) error {
	return nil
}
func (t *testServerStream) SetTrailer(md metadata.MD) {}
func (t *testServerStream) SendMsg(m interface{}) error {
	return t.sendErr
}
func (t *testServerStream) RecvMsg(m interface{}) error {
	if len(t.recvMsgs) == 0 {
		return io.EOF
	}
	next := t.recvMsgs[0]
	t.recvMsgs = t.recvMsgs[1:]
	val := reflect.ValueOf(m)
	if val.Kind() == reflect.Ptr && val.Elem().CanSet() {
		val.Elem().Set(reflect.ValueOf(next))
	}
	return nil
}

type validatableReq struct {
	Name string
}

func (v validatableReq) Validate() error {
	if v.Name == "" {
		return FieldErrors{{Field: "name", Message: "required"}}
	}
	return nil
}

type businessRuleReq struct{}

func (b businessRuleReq) Validate() error {
	return BusinessRuleError{Message: "rule violated"}
}

func TestRecoveryUnaryInterceptor_Panic(t *testing.T) {
	interceptor := RecoveryUnaryInterceptor()
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc/m"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		panic("boom")
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected Internal, got %v", status.Code(err))
	}
}

func TestRequestIDUnaryInterceptor_Generates(t *testing.T) {
	interceptor := RequestIDUnaryInterceptor()
	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/m"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		if ctx.Value(RequestIDKey) == nil {
			t.Fatal("request id not set in context")
		}
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok || len(md.Get(RequestIDKey)) == 0 {
			t.Fatal("request id not in outgoing metadata")
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthenticationUnaryInterceptor_MissingToken(t *testing.T) {
	interceptor := AuthenticationUnaryInterceptor()
	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/m"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", status.Code(err))
	}
}

func TestAuthenticationUnaryInterceptor_HealthCheckBypass(t *testing.T) {
	interceptor := AuthenticationUnaryInterceptor()
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/grpc.health.v1.Health/Check"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthorizationUnaryInterceptor_AdminDenied(t *testing.T) {
	interceptor := AuthorizationUnaryInterceptor()
	ctx := context.WithValue(context.Background(), "user_id", "user-123")
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/goclaw.v1.AdminService/GetEngineStatus"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", status.Code(err))
	}
}

func TestRateLimitUnaryInterceptor_Exceeded(t *testing.T) {
	rl := NewRateLimiter(1, 1)
	interceptor := RateLimitUnaryInterceptor(rl)
	ctx := context.Background()
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/m"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/m"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	})
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("expected ResourceExhausted, got %v", status.Code(err))
	}
}

func TestLoggingUnaryInterceptor(t *testing.T) {
	interceptor := LoggingUnaryInterceptor()
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc/m"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidationUnaryInterceptor_InvalidArgument(t *testing.T) {
	interceptor := ValidationUnaryInterceptor()
	called := false
	_, err := interceptor(context.Background(), validatableReq{}, &grpc.UnaryServerInfo{FullMethod: "/svc/m"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		return nil, nil
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
	if called {
		t.Fatal("handler should not be called on validation error")
	}
}

func TestValidationUnaryInterceptor_BusinessRule(t *testing.T) {
	interceptor := ValidationUnaryInterceptor()
	_, err := interceptor(context.Background(), businessRuleReq{}, &grpc.UnaryServerInfo{FullMethod: "/svc/m"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", status.Code(err))
	}
}

func TestValidationStreamInterceptor_InvalidArgument(t *testing.T) {
	interceptor := ValidationStreamInterceptor()
	stream := &testServerStream{
		ctx:      context.Background(),
		recvMsgs: []interface{}{validatableReq{}},
	}
	err := interceptor(nil, stream, &grpc.StreamServerInfo{FullMethod: "/svc/stream"}, func(srv interface{}, ss grpc.ServerStream) error {
		var req validatableReq
		if err := ss.RecvMsg(&req); err != nil {
			return err
		}
		return nil
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestMetricsUnaryInterceptor_Records(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)
	interceptor := MetricsUnaryInterceptor(metrics)
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc/m"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := testutil.ToFloat64(metrics.requests.WithLabelValues("/svc/m", "OK"))
	if got != 1 {
		t.Fatalf("expected request count 1, got %v", got)
	}
	inflight := testutil.ToFloat64(metrics.inflight.WithLabelValues("/svc/m"))
	if inflight != 0 {
		t.Fatalf("expected inflight 0, got %v", inflight)
	}
}

func TestMetricsStreamInterceptor_RecordsMessages(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)
	interceptor := MetricsStreamInterceptor(metrics)

	stream := &testServerStream{
		ctx:      context.Background(),
		recvMsgs: []interface{}{validatableReq{Name: "ok"}, validatableReq{Name: "ok"}},
	}
	err := interceptor(nil, stream, &grpc.StreamServerInfo{FullMethod: "/svc/stream"}, func(srv interface{}, ss grpc.ServerStream) error {
		var req validatableReq
		_ = ss.RecvMsg(&req)
		_ = ss.RecvMsg(&req)
		if err := ss.SendMsg(&req); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recv := testutil.ToFloat64(metrics.streamMessages.WithLabelValues("/svc/stream", "recv"))
	if recv != 2 {
		t.Fatalf("expected recv count 2, got %v", recv)
	}
	sent := testutil.ToFloat64(metrics.streamMessages.WithLabelValues("/svc/stream", "sent"))
	if sent != 1 {
		t.Fatalf("expected sent count 1, got %v", sent)
	}
}

func TestTracingUnaryInterceptor_StartsSpanAndInjects(t *testing.T) {
	prevProvider := otel.GetTracerProvider()
	prevProp := otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(prevProvider)
		otel.SetTextMapPropagator(prevProp)
	})

	otel.SetTracerProvider(noop.NewTracerProvider())
	otel.SetTextMapPropagator(propagation.TraceContext{})

	parent := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SpanID:     trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		TraceFlags: trace.FlagsSampled,
	})
	parentCtx := trace.ContextWithSpanContext(context.Background(), parent)
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(parentCtx, carrier)
	incoming := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string(carrier)))

	interceptor := TracingUnaryInterceptor()
	_, err := interceptor(incoming, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/m"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		if !trace.SpanContextFromContext(ctx).IsValid() {
			return nil, errors.New("span not set")
		}
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok || len(md.Get("traceparent")) == 0 {
			return nil, errors.New("traceparent not injected")
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTracingStreamInterceptor_StartsSpan(t *testing.T) {
	prevProvider := otel.GetTracerProvider()
	prevProp := otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(prevProvider)
		otel.SetTextMapPropagator(prevProp)
	})

	otel.SetTracerProvider(noop.NewTracerProvider())
	otel.SetTextMapPropagator(propagation.TraceContext{})

	parent := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{9, 8, 7, 6, 5, 4, 3, 2, 1, 10, 11, 12, 13, 14, 15, 16},
		SpanID:     trace.SpanID{8, 7, 6, 5, 4, 3, 2, 1},
		TraceFlags: trace.FlagsSampled,
	})
	parentCtx := trace.ContextWithSpanContext(context.Background(), parent)
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(parentCtx, carrier)
	incoming := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string(carrier)))

	stream := &testServerStream{
		ctx: incoming,
	}
	interceptor := TracingStreamInterceptor()
	err := interceptor(nil, stream, &grpc.StreamServerInfo{FullMethod: "/svc/stream"}, func(srv interface{}, ss grpc.ServerStream) error {
		if !trace.SpanContextFromContext(ss.Context()).IsValid() {
			return errors.New("span not set")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
