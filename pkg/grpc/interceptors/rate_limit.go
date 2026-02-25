package interceptors

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// RateLimiter manages rate limiting per client
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerSecond float64, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(requestsPerSecond),
		burst:    burst,
	}
}

// getLimiter gets or creates a limiter for a client
func (rl *RateLimiter) getLimiter(clientID string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[clientID]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[clientID] = limiter
	}

	return limiter
}

// RateLimitUnaryInterceptor enforces rate limiting per client
func RateLimitUnaryInterceptor(rl *RateLimiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip rate limiting for health check
		if info.FullMethod == "/grpc.health.v1.Health/Check" ||
			info.FullMethod == "/grpc.health.v1.Health/Watch" {
			return handler(ctx, req)
		}

		// Get client ID (use user_id from auth, or IP address)
		clientID := getClientID(ctx)

		// Get limiter for this client
		limiter := rl.getLimiter(clientID)

		// Check if request is allowed
		if !limiter.Allow() {
			// Calculate retry-after duration
			reservation := limiter.Reserve()
			retryAfter := reservation.Delay()
			reservation.Cancel()

			// Add retry-after to metadata
			md := metadata.Pairs("retry-after", retryAfter.String())
			grpc.SetHeader(ctx, md)

			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(ctx, req)
	}
}

// RateLimitStreamInterceptor enforces rate limiting for streams
func RateLimitStreamInterceptor(rl *RateLimiter) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Skip rate limiting for health check
		if info.FullMethod == "/grpc.health.v1.Health/Check" ||
			info.FullMethod == "/grpc.health.v1.Health/Watch" {
			return handler(srv, ss)
		}

		ctx := ss.Context()

		// Get client ID
		clientID := getClientID(ctx)

		// Get limiter for this client
		limiter := rl.getLimiter(clientID)

		// Check if request is allowed
		if !limiter.Allow() {
			return status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(srv, ss)
	}
}

// getClientID extracts client identifier from context
func getClientID(ctx context.Context) string {
	// Try to get user ID from context (set by auth interceptor)
	if userID := ctx.Value("user_id"); userID != nil {
		return userID.(string)
	}

	// Fall back to request ID
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		return requestID.(string)
	}

	// Default to "anonymous"
	return "anonymous"
}
