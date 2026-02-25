package interceptors

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// TracingUnaryInterceptor adds distributed tracing for unary RPCs.
func TracingUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = extractTraceContext(ctx)

		tracer := otel.Tracer("goclaw.grpc")
		ctx, span := tracer.Start(ctx, info.FullMethod, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		span.SetAttributes(methodAttributes(info.FullMethod)...)
		ctx = injectTraceContext(ctx)

		resp, err := handler(ctx, req)
		recordSpanResult(span, err)
		return resp, err
	}
}

// TracingStreamInterceptor adds distributed tracing for streaming RPCs.
func TracingStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := extractTraceContext(ss.Context())
		tracer := otel.Tracer("goclaw.grpc")
		ctx, span := tracer.Start(ctx, info.FullMethod, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		span.SetAttributes(methodAttributes(info.FullMethod)...)
		ctx = injectTraceContext(ctx)

		wrapped := &wrappedStream{ServerStream: ss, ctx: ctx}
		err := handler(srv, wrapped)
		recordSpanResult(span, err)
		return err
	}
}

func extractTraceContext(ctx context.Context) context.Context {
	md, _ := metadata.FromIncomingContext(ctx)
	return otel.GetTextMapPropagator().Extract(ctx, metadataCarrier(md))
}

func injectTraceContext(ctx context.Context) context.Context {
	md := metadata.New(nil)
	otel.GetTextMapPropagator().Inject(ctx, metadataCarrier(md))
	return metadata.NewOutgoingContext(ctx, md)
}

func recordSpanResult(span trace.Span, err error) {
	if err == nil {
		span.SetStatus(otelcodes.Ok, "ok")
		return
	}
	span.RecordError(err)
	st := status.Code(err)
	span.SetStatus(otelcodes.Error, st.String())
}

func methodAttributes(fullMethod string) []attribute.KeyValue {
	service, method := splitMethod(fullMethod)
	return []attribute.KeyValue{
		attribute.String("rpc.system", "grpc"),
		attribute.String("rpc.service", service),
		attribute.String("rpc.method", method),
	}
}

func splitMethod(fullMethod string) (string, string) {
	if fullMethod == "" {
		return "unknown", "unknown"
	}
	fullMethod = strings.TrimPrefix(fullMethod, "/")
	parts := strings.SplitN(fullMethod, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return fullMethod, "unknown"
}

type metadataCarrier metadata.MD

func (c metadataCarrier) Get(key string) string {
	values := metadata.MD(c).Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (c metadataCarrier) Set(key string, value string) {
	metadata.MD(c).Set(key, value)
}

func (c metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(metadata.MD(c)))
	for k := range metadata.MD(c) {
		keys = append(keys, k)
	}
	return keys
}

var _ propagation.TextMapCarrier = metadataCarrier{}
