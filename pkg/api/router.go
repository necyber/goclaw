// Package api provides HTTP API server components.
package api

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/goclaw/goclaw/config"
	"github.com/goclaw/goclaw/pkg/api/handlers"
	"github.com/goclaw/goclaw/pkg/api/middleware"
	"github.com/goclaw/goclaw/pkg/logger"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/goclaw/goclaw/docs/swagger" // Import generated docs
)

// Handlers holds all HTTP handlers.
type Handlers struct {
	// Workflow handles workflow-related endpoints
	Workflow *handlers.WorkflowHandler

	// Health handles health check endpoints
	Health *handlers.HealthHandler

	// Memory handles memory-related endpoints
	Memory *handlers.MemoryHandler

	// Saga handles saga-related endpoints
	Saga *handlers.SagaHandler

	// Metrics is the optional metrics recorder
	Metrics middleware.MetricsRecorder

	// WebSocket handles websocket events endpoint
	WebSocket http.Handler
}

// NewRouter creates a new chi router with middleware and routes.
func NewRouter(cfg *config.Config, log logger.Logger, handlers *Handlers) chi.Router {
	r := chi.NewRouter()

	// Register global middleware
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(log))
	r.Use(middleware.Recovery(log))

	// Add metrics middleware if provided
	if handlers.Metrics != nil {
		r.Use(middleware.Metrics(handlers.Metrics))
	}

	r.Use(middleware.CORS(&cfg.Server.CORS))
	r.Use(middleware.Timeout(cfg.Server.HTTP.ReadTimeout))

	// Register routes
	RegisterRoutes(r, cfg, log, handlers)

	return r
}

// RegisterRoutes registers all API routes.
func RegisterRoutes(r chi.Router, cfg *config.Config, log logger.Logger, handlers *Handlers) {
	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Workflow routes
		if handlers.Workflow != nil {
			r.Route("/workflows", func(r chi.Router) {
				r.Post("/", handlers.Workflow.SubmitWorkflow)
				r.Get("/", handlers.Workflow.ListWorkflows)
				r.Get("/{id}", handlers.Workflow.GetWorkflow)
				r.Post("/{id}/cancel", handlers.Workflow.CancelWorkflow)
				r.Get("/{id}/tasks/{tid}/result", handlers.Workflow.GetTaskResult)
			})
		}

		// Memory routes
		if handlers.Memory != nil {
			r.Route("/memory/{sessionID}", func(r chi.Router) {
				r.Post("/", handlers.Memory.StoreMemory)
				r.Get("/", handlers.Memory.QueryMemory)
				r.Delete("/", handlers.Memory.DeleteMemory)
				r.Get("/list", handlers.Memory.ListMemory)
				r.Get("/stats", handlers.Memory.GetStats)
				r.Delete("/all", handlers.Memory.DeleteSession)
				r.Delete("/weak", handlers.Memory.DeleteWeakMemories)
			})
		}

		// Saga routes
		if handlers.Saga != nil {
			r.Route("/sagas", func(r chi.Router) {
				r.Post("/", handlers.Saga.SubmitSaga)
				r.Get("/", handlers.Saga.ListSagas)
				r.Get("/{id}", handlers.Saga.GetSaga)
				r.Post("/{id}/compensate", handlers.Saga.CompensateSaga)
				r.Post("/{id}/recover", handlers.Saga.RecoverSaga)
			})
		}
	})

	// Health check routes (not versioned)
	if handlers.Health != nil {
		r.Get("/health", handlers.Health.Health)
		r.Get("/ready", handlers.Health.Ready)
		r.Get("/status", handlers.Health.Status)
	}

	// WebSocket events
	if handlers.WebSocket != nil {
		r.Handle("/ws/events", handlers.WebSocket)
	}

	// Swagger documentation
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	registerUIRoutes(r, cfg, log)
}

func registerUIRoutes(r chi.Router, cfg *config.Config, log logger.Logger) {
	if cfg == nil || !cfg.UI.Enabled {
		return
	}

	basePath := normalizeUIBasePath(cfg.UI.BasePath)
	handler := newUIHandler(log)

	if cfg.UI.DevProxy != "" {
		proxy, err := newUIDevProxy(cfg.UI.DevProxy, log)
		if err != nil {
			log.Error("invalid ui.dev_proxy, falling back to embedded UI", "value", cfg.UI.DevProxy, "error", err)
			handler = http.StripPrefix(basePath, handler)
		} else {
			handler = proxy
		}
	} else {
		handler = http.StripPrefix(basePath, handler)
	}

	r.Handle(basePath, handler)
	r.Handle(basePath+"/", handler)
	r.Handle(basePath+"/*", handler)
}

func normalizeUIBasePath(basePath string) string {
	normalized := strings.TrimSpace(basePath)
	if normalized == "" {
		return "/ui"
	}
	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}
	return strings.TrimRight(normalized, "/")
}

func newUIDevProxy(rawURL string, log logger.Logger) (http.Handler, error) {
	target, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, proxyErr error) {
		if log != nil {
			log.Error("ui dev proxy request failed", "target", rawURL, "path", req.URL.Path, "error", proxyErr)
		}
		http.Error(w, "UI dev proxy unavailable", http.StatusBadGateway)
	}

	return proxy, nil
}
