package interceptors

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	// RequestIDKey is the metadata key for request ID
	RequestIDKey = "x-request-id"
)

// RequestIDUnaryInterceptor generates or propagates request ID
func RequestIDUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		requestID := extractOrGenerateRequestID(ctx)
		ctx = withRequestID(ctx, requestID)

		// Add to outgoing metadata
		ctx = metadata.AppendToOutgoingContext(ctx, RequestIDKey, requestID)

		return handler(ctx, req)
	}
}

// RequestIDStreamInterceptor generates or propagates request ID for streams
func RequestIDStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		requestID := extractOrGenerateRequestID(ctx)
		ctx = withRequestID(ctx, requestID)

		// Wrap the stream with new context
		wrapped := &wrappedStream{ServerStream: ss, ctx: ctx}
		return handler(srv, wrapped)
	}
}

// extractOrGenerateRequestID extracts request ID from metadata or generates new one
func extractOrGenerateRequestID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if ids := md.Get(RequestIDKey); len(ids) > 0 {
			return ids[0]
		}
	}
	return uuid.New().String()
}

// wrappedStream wraps grpc.ServerStream with custom context
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}
