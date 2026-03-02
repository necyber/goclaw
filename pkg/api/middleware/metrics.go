package middleware

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// MetricsRecorder defines the interface for recording HTTP metrics.
type MetricsRecorder interface {
	RecordHTTPRequest(method, path, status string, duration time.Duration)
	IncActiveConnections()
	DecActiveConnections()
}

// TraceAwareMetricsRecorder extends MetricsRecorder with context-aware recording.
// Implementations can use trace context for exemplar correlation when supported.
type TraceAwareMetricsRecorder interface {
	MetricsRecorder
	RecordHTTPRequestWithContext(ctx context.Context, method, path, status string, duration time.Duration)
}

// MetricsOptions configures HTTP metrics middleware behavior.
type MetricsOptions struct {
	MetricsPath string
}

// Metrics returns a middleware that records HTTP metrics.
func Metrics(recorder MetricsRecorder) func(http.Handler) http.Handler {
	return MetricsWithOptions(recorder, MetricsOptions{MetricsPath: "/metrics"})
}

// MetricsWithOptions returns a middleware that records HTTP metrics with explicit options.
func MetricsWithOptions(recorder MetricsRecorder, opts MetricsOptions) func(http.Handler) http.Handler {
	metricsPath := normalizeMetricsPathOption(opts.MetricsPath)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip metrics endpoint to avoid recursion
			if isMetricsEndpointRequest(r.URL.Path, metricsPath) {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			recorder.IncActiveConnections()
			defer recorder.DecActiveConnections()

			// Wrap response writer to capture status code
			wrapped := &metricsResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Handle panics to ensure metrics are recorded
			defer func() {
				if err := recover(); err != nil {
					wrapped.statusCode = http.StatusInternalServerError
					duration := time.Since(start)
					path := normalizePath(r.URL.Path)
					recordHTTPRequest(recorder, r.Context(), r.Method, path, statusClass(wrapped.statusCode), duration)
					panic(err) // Re-panic after recording
				}
			}()

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)
			path := normalizePath(r.URL.Path)
			recordHTTPRequest(recorder, r.Context(), r.Method, path, statusClass(wrapped.statusCode), duration)
		})
	}
}

func recordHTTPRequest(recorder MetricsRecorder, ctx context.Context, method, path, status string, duration time.Duration) {
	if traceAwareRecorder, ok := recorder.(TraceAwareMetricsRecorder); ok {
		traceAwareRecorder.RecordHTTPRequestWithContext(ctx, method, path, status, duration)
		return
	}
	recorder.RecordHTTPRequest(method, path, status, duration)
}

// metricsResponseWriter wraps http.ResponseWriter to capture the status code.
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *metricsResponseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *metricsResponseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

// normalizePath normalizes URL paths to reduce cardinality.
// Replaces UUIDs and numeric IDs with placeholders.
func normalizePath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		// Replace UUIDs (8-4-4-4-12 format)
		if uuidPattern.MatchString(part) {
			parts[i] = ":id"
			continue
		}
		// Replace ULIDs
		if ulidPattern.MatchString(strings.ToUpper(part)) {
			parts[i] = ":id"
			continue
		}
		// Replace numeric IDs
		if _, err := strconv.Atoi(part); err == nil && len(part) > 0 {
			parts[i] = ":id"
			continue
		}
		// Replace long opaque tokens
		if opaqueTokenPattern.MatchString(part) {
			parts[i] = ":id"
		}
	}
	return strings.Join(parts, "/")
}

func statusClass(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	default:
		return "5xx"
	}
}

func normalizeMetricsPathOption(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/metrics"
	}
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}

func isMetricsEndpointRequest(requestPath, metricsPath string) bool {
	if requestPath == metricsPath {
		return true
	}
	return strings.HasPrefix(requestPath, metricsPath+"/")
}

var (
	uuidPattern       = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	ulidPattern       = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)
	opaqueTokenPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{16,}$`)
)
