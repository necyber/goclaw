package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goclaw/goclaw/pkg/api/response"
	"github.com/goclaw/goclaw/pkg/logger"
)

func TestRecovery(t *testing.T) {
	tests := []struct {
		name        string
		handler     http.HandlerFunc
		shouldPanic bool
		wantStatus  int
	}{
		{
			name: "no panic",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("ok"))
			},
			shouldPanic: false,
			wantStatus:  http.StatusOK,
		},
		{
			name: "panic with string",
			handler: func(w http.ResponseWriter, r *http.Request) {
				panic("something went wrong")
			},
			shouldPanic: true,
			wantStatus:  http.StatusInternalServerError,
		},
		{
			name: "panic with error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				panic(response.ErrInternalServer)
			},
			shouldPanic: true,
			wantStatus:  http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test logger
			log := logger.New(&logger.Config{
				Level:  logger.InfoLevel,
				Format: "json",
				Output: "stdout",
			})

			// Wrap with recovery middleware
			middleware := Recovery(log)
			wrappedHandler := middleware(tt.handler)

			// Create test request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("X-Request-ID", "test-123")
			w := httptest.NewRecorder()

			// Execute request
			wrappedHandler.ServeHTTP(w, req)

			// Verify status code
			if w.Code != tt.wantStatus {
				t.Errorf("Recovery middleware status = %v, want %v", w.Code, tt.wantStatus)
			}

			// If panic expected, verify error response
			if tt.shouldPanic {
				var errResp response.ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
					t.Fatalf("failed to unmarshal error response: %v", err)
				}

				if errResp.Error.Code != response.ErrCodeInternalServer {
					t.Errorf("error code = %v, want %v", errResp.Error.Code, response.ErrCodeInternalServer)
				}
			}
		})
	}
}
