package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goclaw/goclaw/pkg/api/response"
)

func TestTimeout(t *testing.T) {
	tests := []struct {
		name         string
		timeout      time.Duration
		handlerDelay time.Duration
		wantStatus   int
		wantTimeout  bool
	}{
		{
			name:         "request completes before timeout",
			timeout:      100 * time.Millisecond,
			handlerDelay: 10 * time.Millisecond,
			wantStatus:   http.StatusOK,
			wantTimeout:  false,
		},
		{
			name:         "request times out",
			timeout:      50 * time.Millisecond,
			handlerDelay: 200 * time.Millisecond,
			wantStatus:   http.StatusGatewayTimeout,
			wantTimeout:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler with delay
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(tt.handlerDelay)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("ok"))
			})

			// Wrap with timeout middleware
			middleware := Timeout(tt.timeout)
			wrappedHandler := middleware(handler)

			// Create test request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("X-Request-ID", "test-123")
			w := httptest.NewRecorder()

			// Execute request
			wrappedHandler.ServeHTTP(w, req)

			// Verify status code
			if w.Code != tt.wantStatus {
				t.Errorf("Timeout middleware status = %v, want %v", w.Code, tt.wantStatus)
			}

			// If timeout expected, verify error response
			if tt.wantTimeout {
				var errResp response.ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
					t.Fatalf("failed to unmarshal error response: %v", err)
				}

				if errResp.Error.Code != response.ErrCodeGatewayTimeout {
					t.Errorf("error code = %v, want %v", errResp.Error.Code, response.ErrCodeGatewayTimeout)
				}
			}
		})
	}
}
