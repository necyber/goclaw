// Package handlers provides HTTP request handlers.
package handlers

import (
	"net/http"
	"runtime"
	"time"

	"github.com/goclaw/goclaw/pkg/api/models"
	"github.com/goclaw/goclaw/pkg/api/response"
	"github.com/goclaw/goclaw/pkg/engine"
)

// HealthHandler handles health check endpoints.
type HealthHandler struct {
	engine    *engine.Engine
	startedAt time.Time
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler(eng *engine.Engine) *HealthHandler {
	return &HealthHandler{
		engine:    eng,
		startedAt: time.Now().UTC(),
	}
}

// Health handles the /health endpoint (liveness probe).
// @Summary Health check
// @Description Check if the service is alive and running
// @Tags health
// @Produce json
// @Success 200 {object} models.HealthResponse "Service is healthy"
// @Failure 503 {object} models.HealthResponse "Service is unhealthy"
// @Router /health [get]
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	if h.engine.IsHealthy() {
		response.JSON(w, http.StatusOK, models.HealthResponse{
			Status:    "healthy",
			Timestamp: now,
		})
	} else {
		response.JSON(w, http.StatusServiceUnavailable, models.HealthResponse{
			Status:    "unhealthy",
			Timestamp: now,
			Error:     "engine is not healthy",
		})
	}
}

// Ready handles the /ready endpoint (readiness probe).
// @Summary Readiness check
// @Description Check if the service is ready to accept requests
// @Tags health
// @Produce json
// @Success 200 {object} models.ReadyResponse "Service is ready"
// @Failure 503 {object} models.ReadyResponse "Service is not ready"
// @Router /ready [get]
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	ready := h.engine.IsReady()
	checks := map[string]models.ComponentCheck{
		"engine":  {Status: "not_ready"},
		"storage": {Status: "not_ready"},
	}
	if ready {
		checks["engine"] = models.ComponentCheck{Status: "ready"}
		checks["storage"] = models.ComponentCheck{Status: "ready"}
		response.JSON(w, http.StatusOK, models.ReadyResponse{
			Status:    "ready",
			Timestamp: now,
			Checks:    checks,
		})
		return
	}

	response.JSON(w, http.StatusServiceUnavailable, models.ReadyResponse{
		Status:    "not_ready",
		Timestamp: now,
		Checks:    checks,
		Error:     "one or more dependencies are not ready",
	})
}

// Status handles the /status endpoint (detailed status).
// @Summary Detailed status
// @Description Get detailed status information about the service and engine
// @Tags health
// @Produce json
// @Success 200 {object} models.StatusResponse "Detailed status information"
// @Router /status [get]
func (h *HealthHandler) Status(w http.ResponseWriter, r *http.Request) {
	status := h.engine.GetStatus()
	now := time.Now().UTC()
	uptime := now.Sub(h.startedAt)

	memStats := runtime.MemStats{}
	runtime.ReadMemStats(&memStats)

	overall := "degraded"
	if h.engine.IsReady() {
		overall = "ready"
	}

	resp := models.StatusResponse{
		Status:    overall,
		Version:   status.Version,
		Uptime:    uptime.String(),
		Timestamp: now,
		Runtime: models.StatusRuntime{
			State:      status.State,
			GoVersion:  runtime.Version(),
			NumCPU:     runtime.NumCPU(),
			Goroutines: runtime.NumGoroutine(),
		},
		Components: map[string]models.ComponentCheck{
			"engine":  {Status: componentStatus(h.engine.IsHealthy(), "running", "not running")},
			"storage": {Status: componentStatus(h.engine.IsReady(), "ready", "not_ready")},
		},
		System: models.StatusSystem{
			MemoryAllocBytes: memStats.Alloc,
			MemorySysBytes:   memStats.Sys,
		},
	}

	response.JSON(w, http.StatusOK, resp)
}

func componentStatus(ok bool, healthyValue, unhealthyValue string) string {
	if ok {
		return healthyValue
	}
	return unhealthyValue
}
