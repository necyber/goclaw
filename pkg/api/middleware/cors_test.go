package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goclaw/goclaw/config"
)

func TestCORS(t *testing.T) {
	tests := []struct {
		name           string
		config         *config.CORSConfig
		method         string
		origin         string
		wantStatus     int
		wantCORSHeader bool
	}{
		{
			name: "CORS enabled with allowed origin",
			config: &config.CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"http://localhost:3000"},
				AllowedMethods: []string{"GET", "POST"},
				AllowedHeaders: []string{"Content-Type"},
				MaxAge:         3600,
			},
			method:         http.MethodGet,
			origin:         "http://localhost:3000",
			wantStatus:     http.StatusOK,
			wantCORSHeader: true,
		},
		{
			name: "CORS enabled with wildcard origin",
			config: &config.CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST"},
			},
			method:         http.MethodGet,
			origin:         "http://example.com",
			wantStatus:     http.StatusOK,
			wantCORSHeader: true,
		},
		{
			name: "CORS disabled",
			config: &config.CORSConfig{
				Enabled: false,
			},
			method:         http.MethodGet,
			origin:         "http://localhost:3000",
			wantStatus:     http.StatusOK,
			wantCORSHeader: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with CORS middleware
			middleware := CORS(tt.config)
			wrappedHandler := middleware(handler)

			// Create test request
			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			w := httptest.NewRecorder()

			// Execute request
			wrappedHandler.ServeHTTP(w, req)

			// Verify status code
			if w.Code != tt.wantStatus {
				t.Errorf("CORS middleware status = %v, want %v", w.Code, tt.wantStatus)
			}

			// Verify CORS headers
			corsHeader := w.Header().Get("Access-Control-Allow-Origin")
			if tt.wantCORSHeader && corsHeader == "" {
				t.Error("Expected CORS header but not found")
			}
			if !tt.wantCORSHeader && corsHeader != "" {
				t.Error("Unexpected CORS header found")
			}
		})
	}
}
