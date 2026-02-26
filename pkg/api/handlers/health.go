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
// @Summary Health check
// @Description Check if the service is alive and running
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string "Service is healthy"
// @Failure 503 {object} map[string]string "Service is unhealthy"
// @Router /health [get]
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
// @Summary Readiness check
// @Description Check if the service is ready to accept requests
// @Tags health
// @Produce json
// @Success 200 {object} map[string]bool "Service is ready"
// @Failure 503 {object} map[string]bool "Service is not ready"
// @Router /ready [get]
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
// @Summary Detailed status
// @Description Get detailed status information about the service and engine
// @Tags health
// @Produce json
// @Success 200 {object} engine.EngineStatus "Detailed status information"
// @Router /status [get]
func (h *HealthHandler) Status(w http.ResponseWriter, r *http.Request) {
	status := h.engine.GetStatus()
	response.JSON(w, http.StatusOK, status)
}
