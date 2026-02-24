package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goclaw/goclaw/pkg/logger"
)

func TestLogger(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		handlerStatus  int
		handlerBody    string
		wantStatusCode int
	}{
		{
			name:           "successful GET request",
			method:         http.MethodGet,
			path:           "/api/v1/workflows",
			handlerStatus:  http.StatusOK,
			handlerBody:    `{"status":"ok"}`,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "POST request with 201",
			method:         http.MethodPost,
			path:           "/api/v1/workflows",
			handlerStatus:  http.StatusCreated,
			handlerBody:    `{"id":"123"}`,
			wantStatusCode: http.StatusCreated,
		},
		{
			name:           "not found request",
			method:         http.MethodGet,
			path:           "/api/v1/notfound",
			handlerStatus:  http.StatusNotFound,
			handlerBody:    `{"error":"not found"}`,
			wantStatusCode: http.StatusNotFound,
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

			// Create test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.handlerStatus)
				w.Write([]byte(tt.handlerBody))
			})

			// Wrap with logger middleware
			middleware := Logger(log)
			wrappedHandler := middleware(handler)

			// Create test request
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			// Execute request
			wrappedHandler.ServeHTTP(w, req)

			// Verify status code
			if w.Code != tt.wantStatusCode {
				t.Errorf("Logger middleware status = %v, want %v", w.Code, tt.wantStatusCode)
			}

			// Verify body
			if w.Body.String() != tt.handlerBody {
				t.Errorf("Logger middleware body = %v, want %v", w.Body.String(), tt.handlerBody)
			}
		})
	}
}
