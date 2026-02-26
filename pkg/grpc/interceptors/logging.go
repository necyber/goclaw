package interceptors

import (
	"context"
	"fmt"
	"time"

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

		// Log request
		fmt.Printf("[%s] --> %s\n", requestID, info.FullMethod)

		// Call handler
		resp, err := handler(ctx, req)

		// Log response
		duration := time.Since(start)
		statusCode := codes.OK
		if err != nil {
			statusCode = status.Code(err)
		}

		fmt.Printf("[%s] <-- %s [%s] %v\n", requestID, info.FullMethod, statusCode, duration)

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

		// Log stream start
		fmt.Printf("[%s] --> STREAM %s (client=%v, server=%v)\n",
			requestID, info.FullMethod, info.IsClientStream, info.IsServerStream)

		// Call handler
		err := handler(srv, ss)

		// Log stream end
		duration := time.Since(start)
		statusCode := codes.OK
		if err != nil {
			statusCode = status.Code(err)
		}

		fmt.Printf("[%s] <-- STREAM %s [%s] %v\n", requestID, info.FullMethod, statusCode, duration)

		return err
	}
}
