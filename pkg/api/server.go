// Package api provides HTTP API server components.
package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/goclaw/goclaw/config"
	"github.com/goclaw/goclaw/pkg/logger"
)

// Server defines the interface for HTTP server lifecycle management.
type Server interface {
	Start() error
	Shutdown(ctx context.Context) error
}

// HTTPServer implements the Server interface.
type HTTPServer struct {
	config *config.Config
	server *http.Server
	router chi.Router
	logger logger.Logger
}

// NewHTTPServer creates a new HTTP server instance.
func NewHTTPServer(cfg *config.Config, log logger.Logger, handlers *Handlers) *HTTPServer {
	// Create router with middleware and routes
	router := NewRouter(cfg, log, handlers)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.HTTP.ReadTimeout,
		WriteTimeout: cfg.Server.HTTP.WriteTimeout,
		IdleTimeout:  cfg.Server.HTTP.IdleTimeout,
	}

	return &HTTPServer{
		config: cfg,
		server: srv,
		router: router,
		logger: log,
	}
}

// Start starts the HTTP server.
func (s *HTTPServer) Start() error {
	s.logger.Info("Starting HTTP server",
		"addr", s.server.Addr,
		"read_timeout", s.config.Server.HTTP.ReadTimeout,
		"write_timeout", s.config.Server.HTTP.WriteTimeout,
	)

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.logger.Error("HTTP server failed", "error", err)
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the HTTP server.
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server")

	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.Error("HTTP server shutdown failed", "error", err)
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	s.logger.Info("HTTP server stopped")
	return nil
}

