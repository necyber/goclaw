package grpc

import (
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// HealthServer wraps the gRPC health check server
type HealthServer struct {
	server *health.Server
}

// NewHealthServer creates a new health check server
func NewHealthServer() *HealthServer {
	return &HealthServer{
		server: health.NewServer(),
	}
}

// SetServingStatus sets the serving status for a service
func (h *HealthServer) SetServingStatus(service string, status grpc_health_v1.HealthCheckResponse_ServingStatus) {
	h.server.SetServingStatus(service, status)
}

// SetServingStatusAll sets the serving status for all services
func (h *HealthServer) SetServingStatusAll(status grpc_health_v1.HealthCheckResponse_ServingStatus) {
	h.server.SetServingStatus("", status)
}

// Shutdown gracefully shuts down the health server
func (h *HealthServer) Shutdown() {
	h.server.Shutdown()
}

// Resume resumes the health server
func (h *HealthServer) Resume() {
	h.server.Resume()
}

// GetServer returns the underlying health server for registration
func (h *HealthServer) GetServer() *health.Server {
	return h.server
}
