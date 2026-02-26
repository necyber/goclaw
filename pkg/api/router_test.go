package api

import (
	"context"
	"github.com/goclaw/goclaw/pkg/storage/memory"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/goclaw/goclaw/config"
	"github.com/goclaw/goclaw/pkg/api/handlers"
	"github.com/goclaw/goclaw/pkg/engine"
	"github.com/goclaw/goclaw/pkg/logger"
)

// createTestHandlers creates test handlers with a running engine
func createTestHandlers(t *testing.T) (*Handlers, func()) {
	cfg := &config.Config{
		App: config.AppConfig{
			Name:        "test",
			Environment: "development",
		},
		Orchestration: config.OrchestrationConfig{
			MaxAgents: 10,
		},
	}
	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})

	eng, err := engine.New(cfg, log, memory.NewMemoryStorage())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	ctx := context.Background()
	if err := eng.Start(ctx); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}

	cleanup := func() {
		eng.Stop(ctx)
	}

	return &Handlers{
		Workflow: handlers.NewWorkflowHandler(eng, log),
		Health:   handlers.NewHealthHandler(eng),
	}, cleanup
}

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
				ReadTimeout: 30 * time.Second,
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

	testHandlers, cleanup := createTestHandlers(t)
	defer cleanup()

	router := NewRouter(cfg, log, testHandlers)

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
				ReadTimeout: 30 * time.Second,
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

	testHandlers, cleanup := createTestHandlers(t)
	defer cleanup()

	router := NewRouter(cfg, log, testHandlers)

	// Test workflow list endpoint - should now work with real handler
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should return 200 OK with empty workflow list
	if w.Code != http.StatusOK {
		t.Errorf("workflow endpoint status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestRegisterRoutes_UIEnabled(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			HTTP: config.HTTPConfig{
				ReadTimeout: 30 * time.Second,
			},
			CORS: config.CORSConfig{
				Enabled: false,
			},
		},
		UI: config.UIConfig{
			Enabled:  true,
			BasePath: "/ui",
		},
	}

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})

	router := NewRouter(cfg, log, &Handlers{})

	req := httptest.NewRequest(http.MethodGet, "/ui", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotImplemented)
	}
	if !strings.Contains(w.Body.String(), "UI not included") {
		t.Fatalf("unexpected response body: %q", w.Body.String())
	}
}

func TestRegisterRoutes_UIDisabled(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			HTTP: config.HTTPConfig{
				ReadTimeout: 30 * time.Second,
			},
			CORS: config.CORSConfig{
				Enabled: false,
			},
		},
		UI: config.UIConfig{
			Enabled: false,
		},
	}

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})

	router := NewRouter(cfg, log, &Handlers{})

	req := httptest.NewRequest(http.MethodGet, "/ui", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestRegisterRoutes_UIDevProxy(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("proxied:" + r.URL.Path))
	}))
	defer upstream.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{
			HTTP: config.HTTPConfig{
				ReadTimeout: 30 * time.Second,
			},
			CORS: config.CORSConfig{
				Enabled: false,
			},
		},
		UI: config.UIConfig{
			Enabled:  true,
			BasePath: "/ui",
			DevProxy: upstream.URL,
		},
	}

	log := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: "json",
		Output: "stdout",
	})

	router := NewRouter(cfg, log, &Handlers{})

	req := httptest.NewRequest(http.MethodGet, "/ui/workflows/abc", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "proxied:/ui/workflows/abc") {
		t.Fatalf("unexpected proxy body: %q", w.Body.String())
	}
}

func TestRegisterRoutes_WebSocket(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			HTTP: config.HTTPConfig{
				ReadTimeout: 30 * time.Second,
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

	router := NewRouter(cfg, log, &Handlers{
		WebSocket: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusSwitchingProtocols)
		}),
	})

	req := httptest.NewRequest(http.MethodGet, "/ws/events", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSwitchingProtocols)
	}
}
