// Package api provides HTTP API server components.
package api

import (
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

	// Metrics is the optional metrics recorder
	Metrics middleware.MetricsRecorder
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
	RegisterRoutes(r, handlers)

	return r
}

// RegisterRoutes registers all API routes.
func RegisterRoutes(r chi.Router, handlers *Handlers) {
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
	})

	// Health check routes (not versioned)
	if handlers.Health != nil {
		r.Get("/health", handlers.Health.Health)
		r.Get("/ready", handlers.Health.Ready)
		r.Get("/status", handlers.Health.Status)
	}

	// Swagger documentation
	r.Get("/swagger/*", httpSwagger.WrapHandler)
}
