package middleware

import (
	"bytes"
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/goclaw/goclaw/pkg/api/response"
)

type timeoutWriter struct {
	mu          sync.Mutex
	header      http.Header
	buf         bytes.Buffer
	code        int
	wroteHeader bool
	timedOut    bool
}

func newTimeoutWriter() *timeoutWriter {
	return &timeoutWriter{
		header: make(http.Header),
		code:   http.StatusOK,
	}
}

func (tw *timeoutWriter) Header() http.Header {
	return tw.header
}

func (tw *timeoutWriter) WriteHeader(statusCode int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut || tw.wroteHeader {
		return
	}
	tw.code = statusCode
	tw.wroteHeader = true
}

func (tw *timeoutWriter) Write(p []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut {
		return 0, http.ErrHandlerTimeout
	}
	if !tw.wroteHeader {
		tw.code = http.StatusOK
		tw.wroteHeader = true
	}
	return tw.buf.Write(p)
}

func (tw *timeoutWriter) timeout() {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.timedOut = true
}

func (tw *timeoutWriter) writeTo(w http.ResponseWriter) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut {
		return
	}
	for key, values := range tw.header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(tw.code)
	_, _ = w.Write(tw.buf.Bytes())
}

// Timeout returns a middleware that enforces request timeouts.
func Timeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create context with timeout
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// Create a channel to signal completion
			done := make(chan struct{})

			tw := newTimeoutWriter()

			// Run the handler in a goroutine
			go func() {
				next.ServeHTTP(tw, r.WithContext(ctx))
				close(done)
			}()

			// Wait for completion or timeout
			select {
			case <-done:
				// Request completed successfully
				tw.writeTo(w)
				return
			case <-ctx.Done():
				// Timeout occurred
				tw.timeout()
				requestID := GetRequestID(r.Context())
				if requestID == "" {
					requestID = "unknown"
				}

				response.Error(w,
					http.StatusGatewayTimeout,
					response.ErrCodeGatewayTimeout,
					"Request timeout",
					requestID,
				)
			}
		})
	}
}
