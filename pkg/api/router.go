// Package api provides HTTP API server components.
package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/goclaw/goclaw/config"
	"github.com/goclaw/goclaw/pkg/api/handlers"
	"github.com/goclaw/goclaw/pkg/api/middleware"
	"github.com/goclaw/goclaw/pkg/logger"
)

// Handlers holds all HTTP handlers.
type Handlers struct {
	// Workflow handles workflow-related endpoints
	Workflow *handlers.WorkflowHandler

	// Health handles health check endpoints
	Health *handlers.HealthHandler
}

// NewRouter creates a new chi router with middleware and routes.
func NewRouter(cfg *config.Config, log logger.Logger, handlers *Handlers) chi.Router {
	r := chi.NewRouter()

	// Register global middleware
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(log))
	r.Use(middleware.Recovery(log))
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
}
