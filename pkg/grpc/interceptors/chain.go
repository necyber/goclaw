package interceptors

import (
	"google.golang.org/grpc"
)

// ChainBuilder helps build interceptor chains in the correct order
type ChainBuilder struct {
	unaryInterceptors  []grpc.UnaryServerInterceptor
	streamInterceptors []grpc.StreamServerInterceptor
}

// NewChainBuilder creates a new interceptor chain builder
func NewChainBuilder() *ChainBuilder {
	return &ChainBuilder{
		unaryInterceptors:  make([]grpc.UnaryServerInterceptor, 0),
		streamInterceptors: make([]grpc.StreamServerInterceptor, 0),
	}
}

// WithRecovery adds recovery interceptor (should be first)
func (b *ChainBuilder) WithRecovery() *ChainBuilder {
	b.unaryInterceptors = append(b.unaryInterceptors, RecoveryUnaryInterceptor())
	b.streamInterceptors = append(b.streamInterceptors, RecoveryStreamInterceptor())
	return b
}

// WithRequestID adds request ID interceptor
func (b *ChainBuilder) WithRequestID() *ChainBuilder {
	b.unaryInterceptors = append(b.unaryInterceptors, RequestIDUnaryInterceptor())
	b.streamInterceptors = append(b.streamInterceptors, RequestIDStreamInterceptor())
	return b
}

// WithAuthentication adds authentication interceptor
func (b *ChainBuilder) WithAuthentication() *ChainBuilder {
	b.unaryInterceptors = append(b.unaryInterceptors, AuthenticationUnaryInterceptor())
	b.streamInterceptors = append(b.streamInterceptors, AuthenticationStreamInterceptor())
	return b
}

// WithAuthorization adds authorization interceptor
func (b *ChainBuilder) WithAuthorization() *ChainBuilder {
	b.unaryInterceptors = append(b.unaryInterceptors, AuthorizationUnaryInterceptor())
	b.streamInterceptors = append(b.streamInterceptors, AuthorizationStreamInterceptor())
	return b
}

// WithRateLimit adds rate limiting interceptor
func (b *ChainBuilder) WithRateLimit(requestsPerSecond float64, burst int) *ChainBuilder {
	rl := NewRateLimiter(requestsPerSecond, burst)
	b.unaryInterceptors = append(b.unaryInterceptors, RateLimitUnaryInterceptor(rl))
	b.streamInterceptors = append(b.streamInterceptors, RateLimitStreamInterceptor(rl))
	return b
}

// WithValidation adds validation interceptor
func (b *ChainBuilder) WithValidation() *ChainBuilder {
	b.unaryInterceptors = append(b.unaryInterceptors, ValidationUnaryInterceptor())
	b.streamInterceptors = append(b.streamInterceptors, ValidationStreamInterceptor())
	return b
}

// WithLogging adds logging interceptor
func (b *ChainBuilder) WithLogging() *ChainBuilder {
	b.unaryInterceptors = append(b.unaryInterceptors, LoggingUnaryInterceptor())
	b.streamInterceptors = append(b.streamInterceptors, LoggingStreamInterceptor())
	return b
}

// WithMetrics adds metrics interceptor
func (b *ChainBuilder) WithMetrics(m *Metrics) *ChainBuilder {
	if m == nil {
		// Create default metrics with default registerer
		m = NewMetrics(nil)
	}
	b.unaryInterceptors = append(b.unaryInterceptors, MetricsUnaryInterceptor(m))
	b.streamInterceptors = append(b.streamInterceptors, MetricsStreamInterceptor(m))
	return b
}

// WithTracing adds tracing interceptor
func (b *ChainBuilder) WithTracing() *ChainBuilder {
	b.unaryInterceptors = append(b.unaryInterceptors, TracingUnaryInterceptor())
	b.streamInterceptors = append(b.streamInterceptors, TracingStreamInterceptor())
	return b
}

// Build returns the configured interceptors as server options
func (b *ChainBuilder) Build() []grpc.ServerOption {
	opts := make([]grpc.ServerOption, 0, 2)

	if len(b.unaryInterceptors) > 0 {
		opts = append(opts, grpc.ChainUnaryInterceptor(b.unaryInterceptors...))
	}

	if len(b.streamInterceptors) > 0 {
		opts = append(opts, grpc.ChainStreamInterceptor(b.streamInterceptors...))
	}

	return opts
}

// DefaultChain returns a chain with recommended interceptors in correct order:
// recovery -> request_id -> auth -> authorization -> rate_limit -> validation -> logging -> metrics -> tracing
func DefaultChain() *ChainBuilder {
	return NewChainBuilder().
		WithRecovery().
		WithRequestID().
		WithAuthentication().
		WithAuthorization().
		WithRateLimit(100, 200). // 100 req/s, burst of 200
		WithValidation().
		WithLogging().
		WithMetrics(nil). // nil will create default metrics
		WithTracing()
}
