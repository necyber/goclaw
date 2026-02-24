package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/goclaw/goclaw/pkg/api/response"
	"github.com/goclaw/goclaw/pkg/logger"
)

// Recovery returns a middleware that recovers from panics.
func Recovery(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Log the panic with stack trace
					stack := debug.Stack()
					log.Error("Panic recovered",
						"error", err,
						"path", r.URL.Path,
						"method", r.Method,
						"stack", string(stack),
					)

					// Get request ID from context if available
					requestID := r.Header.Get("X-Request-ID")
					if requestID == "" {
						requestID = "unknown"
					}

					// Return 500 error
					response.Error(w,
						http.StatusInternalServerError,
						response.ErrCodeInternalServer,
						fmt.Sprintf("Internal server error: %v", err),
						requestID,
					)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
