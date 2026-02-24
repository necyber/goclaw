package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestRequestID(t *testing.T) {
	tests := []struct {
		name              string
		existingRequestID string
		wantGenerated     bool
	}{
		{
			name:              "generate new request ID",
			existingRequestID: "",
			wantGenerated:     true,
		},
		{
			name:              "use existing request ID",
			existingRequestID: "existing-123",
			wantGenerated:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler
			var capturedRequestID string
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedRequestID = GetRequestID(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with request ID middleware
			middleware := RequestID()
			wrappedHandler := middleware(handler)

			// Create test request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.existingRequestID != "" {
				req.Header.Set("X-Request-ID", tt.existingRequestID)
			}
			w := httptest.NewRecorder()

			// Execute request
			wrappedHandler.ServeHTTP(w, req)

			// Verify request ID in response header
			responseID := w.Header().Get("X-Request-ID")
			if responseID == "" {
				t.Error("X-Request-ID header not set in response")
			}

			// Verify request ID in context
			if capturedRequestID == "" {
				t.Error("Request ID not set in context")
			}

			// Verify request ID matches
			if responseID != capturedRequestID {
				t.Errorf("Response ID %v != Context ID %v", responseID, capturedRequestID)
			}

			// Verify generation behavior
			if tt.wantGenerated {
				// Should be a valid UUID
				if _, err := uuid.Parse(capturedRequestID); err != nil {
					t.Errorf("Generated request ID is not a valid UUID: %v", err)
				}
			} else {
				// Should match existing ID
				if capturedRequestID != tt.existingRequestID {
					t.Errorf("Request ID = %v, want %v", capturedRequestID, tt.existingRequestID)
				}
			}
		})
	}
}
