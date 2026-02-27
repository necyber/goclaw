package middleware

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const httpTracerName = "goclaw.http"

// TracingOptions defines HTTP tracing middleware behavior.
type TracingOptions struct {
	// SkipPaths are low-value endpoints that should not create spans.
	SkipPaths map[string]struct{}
}

// DefaultTracingOptions returns default HTTP tracing middleware options.
func DefaultTracingOptions() TracingOptions {
	return TracingOptions{
		SkipPaths: map[string]struct{}{
			"/health": {},
			"/ready":  {},
		},
	}
}

// Tracing creates HTTP server spans from incoming requests.
func Tracing(opts TracingOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if shouldSkipTracing(r.URL.Path, opts) {
				next.ServeHTTP(w, r)
				return
			}

			ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
			tracer := otel.Tracer(httpTracerName)

			ctx, span := tracer.Start(ctx, "HTTP "+r.Method, trace.WithSpanKind(trace.SpanKindServer))
			defer span.End()

			span.SetAttributes(
				attribute.String("http.request.method", r.Method),
				attribute.String("url.path", r.URL.Path),
			)

			wrapped := &tracingResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(wrapped, r.WithContext(ctx))

			route := routePattern(r.WithContext(ctx))
			span.SetAttributes(
				attribute.String("http.route", route),
				attribute.Int("http.response.status_code", wrapped.statusCode),
			)
			recordHTTPSpanStatus(span, wrapped.statusCode)
		})
	}
}

// InjectOutboundTraceContext injects trace context into outbound HTTP request headers.
func InjectOutboundTraceContext(req *http.Request) *http.Request {
	if req == nil {
		return nil
	}

	otel.GetTextMapPropagator().Inject(req.Context(), propagation.HeaderCarrier(req.Header))
	return req
}

// NewTracingRequest creates an outbound request and injects trace context headers.
func NewTracingRequest(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	return InjectOutboundTraceContext(req), nil
}

func shouldSkipTracing(path string, opts TracingOptions) bool {
	if len(opts.SkipPaths) == 0 {
		return false
	}
	normalized := strings.TrimSpace(path)
	_, found := opts.SkipPaths[normalized]
	return found
}

func routePattern(r *http.Request) string {
	if r == nil {
		return ""
	}

	if rc := chi.RouteContext(r.Context()); rc != nil {
		if pattern := strings.TrimSpace(rc.RoutePattern()); pattern != "" {
			return pattern
		}
	}

	return r.URL.Path
}

func recordHTTPSpanStatus(span trace.Span, statusCode int) {
	if statusCode >= http.StatusBadRequest {
		span.SetStatus(otelcodes.Error, http.StatusText(statusCode))
		return
	}
	span.SetStatus(otelcodes.Ok, http.StatusText(statusCode))
}

type tracingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *tracingResponseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *tracingResponseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}
