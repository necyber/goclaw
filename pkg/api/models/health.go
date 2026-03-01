package models

import "time"

// HealthResponse defines the liveness payload.
type HealthResponse struct {
	// Status is the liveness state.
	Status string `json:"status"`

	// Timestamp is when the response was generated.
	Timestamp time.Time `json:"timestamp"`

	// Error is present when status is unhealthy.
	Error string `json:"error,omitempty"`
}

// ComponentCheck describes a component readiness/status check.
type ComponentCheck struct {
	// Status is the component status value.
	Status string `json:"status"`

	// Message contains optional detail for failures.
	Message string `json:"message,omitempty"`
}

// ReadyResponse defines the readiness payload.
type ReadyResponse struct {
	// Status is the readiness state.
	Status string `json:"status"`

	// Timestamp is when the response was generated.
	Timestamp time.Time `json:"timestamp"`

	// Checks contains component readiness checks.
	Checks map[string]ComponentCheck `json:"checks"`

	// Error is present when status is not_ready.
	Error string `json:"error,omitempty"`
}

// StatusRuntime contains runtime metadata.
type StatusRuntime struct {
	// State is the engine lifecycle state.
	State string `json:"state"`

	// GoVersion is the runtime Go version.
	GoVersion string `json:"go_version"`

	// NumCPU is the number of logical CPUs.
	NumCPU int `json:"num_cpu"`

	// Goroutines is the number of active goroutines.
	Goroutines int `json:"goroutines"`
}

// StatusSystem contains process/system details.
type StatusSystem struct {
	// MemoryAllocBytes is current allocated memory.
	MemoryAllocBytes uint64 `json:"memory_alloc_bytes"`

	// MemorySysBytes is total bytes obtained from OS.
	MemorySysBytes uint64 `json:"memory_sys_bytes"`
}

// StatusResponse defines the detailed status envelope.
type StatusResponse struct {
	// Status is the overall service status.
	Status string `json:"status"`

	// Version is the application version.
	Version string `json:"version"`

	// Uptime is process uptime in duration string format.
	Uptime string `json:"uptime"`

	// Timestamp is when the response was generated.
	Timestamp time.Time `json:"timestamp"`

	// Runtime contains runtime metadata.
	Runtime StatusRuntime `json:"runtime"`

	// Components contains component-level status values.
	Components map[string]ComponentCheck `json:"components"`

	// System contains process/system details.
	System StatusSystem `json:"system"`
}
