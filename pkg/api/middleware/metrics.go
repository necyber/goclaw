package middleware

import (
	"net/http"
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

// Metrics returns a middleware that records HTTP metrics.
func Metrics(recorder MetricsRecorder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip metrics endpoint to avoid recursion
			if strings.HasPrefix(r.URL.Path, "/metrics") {
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
					recorder.RecordHTTPRequest(r.Method, path, strconv.Itoa(wrapped.statusCode), duration)
					panic(err) // Re-panic after recording
				}
			}()

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)
			path := normalizePath(r.URL.Path)
			recorder.RecordHTTPRequest(r.Method, path, strconv.Itoa(wrapped.statusCode), duration)
		})
	}
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
		if len(part) == 36 && strings.Count(part, "-") == 4 {
			parts[i] = ":id"
			continue
		}
		// Replace numeric IDs
		if _, err := strconv.Atoi(part); err == nil && len(part) > 0 {
			parts[i] = ":id"
		}
	}
	return strings.Join(parts, "/")
}
