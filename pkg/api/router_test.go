package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goclaw/goclaw/config"
	"github.com/goclaw/goclaw/pkg/logger"
)

func TestNewRouter(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		Server: config.ServerConfig{
			HTTP: config.HTTPConfig{
				ReadTimeout: 30,
			},
			CORS: config.CORSConfig{
				Enabled: false,
			},
		},
	}

	// Create test logger
	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})

	// Create router
	handlers := &Handlers{}
	router := NewRouter(cfg, log, handlers)

	if router == nil {
		t.Fatal("NewRouter returned nil")
	}
}

func TestRegisterRoutes_HealthEndpoints(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		method     string
		wantStatus int
	}{
		{
			name:       "health check",
			path:       "/health",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
		},
		{
			name:       "ready check",
			path:       "/ready",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
		},
		{
			name:       "status check",
			path:       "/status",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
		},
	}

	// Create test config and router
	cfg := &config.Config{
		Server: config.ServerConfig{
			HTTP: config.HTTPConfig{
				ReadTimeout: 30,
			},
			CORS: config.CORSConfig{
				Enabled: false,
			},
		},
	}

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})

	handlers := &Handlers{}
	router := NewRouter(cfg, log, handlers)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %v, want %v", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestRegisterRoutes_WorkflowEndpoints(t *testing.T) {
	// Create test config and router
	cfg := &config.Config{
		Server: config.ServerConfig{
			HTTP: config.HTTPConfig{
				ReadTimeout: 30,
			},
			CORS: config.CORSConfig{
				Enabled: false,
			},
		},
	}

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})

	handlers := &Handlers{}
	router := NewRouter(cfg, log, handlers)

	// Test workflow list endpoint (placeholder)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should return 501 Not Implemented for now
	if w.Code != http.StatusNotImplemented {
		t.Errorf("workflow endpoint status = %v, want %v", w.Code, http.StatusNotImplemented)
	}
}

