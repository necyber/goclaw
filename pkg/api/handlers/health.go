// Package handlers provides HTTP request handlers.
package handlers

import (
	"net/http"

	"github.com/goclaw/goclaw/pkg/api/response"
	"github.com/goclaw/goclaw/pkg/engine"
)

// HealthHandler handles health check endpoints.
type HealthHandler struct {
	engine *engine.Engine
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler(eng *engine.Engine) *HealthHandler {
	return &HealthHandler{
		engine: eng,
	}
}

// Health handles the /health endpoint (liveness probe).
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	if h.engine.IsHealthy() {
		response.JSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
	} else {
		response.JSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unhealthy",
		})
	}
}

// Ready handles the /ready endpoint (readiness probe).
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if h.engine.IsReady() {
		response.JSON(w, http.StatusOK, map[string]bool{
			"ready": true,
		})
	} else {
		response.JSON(w, http.StatusServiceUnavailable, map[string]bool{
			"ready": false,
		})
	}
}

// Status handles the /status endpoint (detailed status).
func (h *HealthHandler) Status(w http.ResponseWriter, r *http.Request) {
	status := h.engine.GetStatus()
	response.JSON(w, http.StatusOK, status)
}


