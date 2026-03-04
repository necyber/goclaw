package interceptors

import (
	"context"
	"time"

	"github.com/goclaw/goclaw/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LoggingUnaryInterceptor logs request and response for unary RPCs
func LoggingUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Extract request ID from context
		requestID, ok := requestIDFromContext(ctx)
		if !ok {
			requestID = "unknown"
		}

		// Log request with context so trace correlation fields are emitted when available.
		log := logger.FromContext(ctx)
		log.InfoContext(ctx, "gRPC request started",
			"request_id", requestID,
			"method", info.FullMethod,
		)

		// Call handler
		resp, err := handler(ctx, req)

		// Log response
		duration := time.Since(start)
		statusCode := codes.OK
		if err != nil {
			statusCode = status.Code(err)
		}

		log.InfoContext(ctx, "gRPC request completed",
			"request_id", requestID,
			"method", info.FullMethod,
			"status", statusCode.String(),
			"duration_ms", duration.Milliseconds(),
		)

		return resp, err
	}
}

// LoggingStreamInterceptor logs stream lifecycle for streaming RPCs
func LoggingStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		// Extract request ID from context
		ctx := ss.Context()
		requestID, ok := requestIDFromContext(ctx)
		if !ok {
			requestID = "unknown"
		}

		log := logger.FromContext(ctx)
		log.InfoContext(ctx, "gRPC stream started",
			"request_id", requestID,
			"method", info.FullMethod,
			"is_client_stream", info.IsClientStream,
			"is_server_stream", info.IsServerStream,
		)

		// Call handler
		err := handler(srv, ss)

		// Log stream end
		duration := time.Since(start)
		statusCode := codes.OK
		if err != nil {
			statusCode = status.Code(err)
		}

		log.InfoContext(ctx, "gRPC stream completed",
			"request_id", requestID,
			"method", info.FullMethod,
			"status", statusCode.String(),
			"duration_ms", duration.Milliseconds(),
		)

		return err
	}
}
