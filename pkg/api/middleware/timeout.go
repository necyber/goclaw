package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/goclaw/goclaw/pkg/api/response"
)

// Timeout returns a middleware that enforces request timeouts.
func Timeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create context with timeout
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// Create a channel to signal completion
			done := make(chan struct{})

			// Run the handler in a goroutine
			go func() {
				next.ServeHTTP(w, r.WithContext(ctx))
				close(done)
			}()

			// Wait for completion or timeout
			select {
			case <-done:
				// Request completed successfully
				return
			case <-ctx.Done():
				// Timeout occurred
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
