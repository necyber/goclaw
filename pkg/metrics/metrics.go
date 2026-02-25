// Package metrics provides Prometheus metrics instrumentation for Goclaw.
package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Manager manages all Prometheus metrics for Goclaw.
type Manager struct {
	registry *prometheus.Registry
	enabled  bool

	// Workflow metrics
	workflowSubmissions *prometheus.CounterVec
	workflowDuration    *prometheus.HistogramVec
	workflowActive      *prometheus.GaugeVec

	// Task metrics
	taskExecutions *prometheus.CounterVec
	taskDuration   *prometheus.HistogramVec
	taskRetries    *prometheus.CounterVec

	// Lane metrics
	laneQueueDepth   *prometheus.GaugeVec
	laneWaitDuration *prometheus.HistogramVec
	laneThroughput   *prometheus.CounterVec

	// HTTP metrics
	httpRequests    *prometheus.CounterVec
	httpDuration    *prometheus.HistogramVec
	httpConnections prometheus.Gauge
}

// Config holds metrics configuration.
type Config struct {
	Enabled bool
	Port    int
	Path    string

	// Histogram bucket configurations
	WorkflowDurationBuckets []float64
	TaskDurationBuckets     []float64
	LaneWaitBuckets         []float64
	HTTPDurationBuckets     []float64
}

// DefaultConfig returns default metrics configuration.
func DefaultConfig() Config {
	return Config{
		Enabled: true,
		Port:    9091,
		Path:    "/metrics",
		WorkflowDurationBuckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300},
		TaskDurationBuckets:     []float64{0.01, 0.05, 0.1, 0.5, 1, 5, 10, 30},
		LaneWaitBuckets:         []float64{0.001, 0.01, 0.1, 0.5, 1, 5, 10, 30},
		HTTPDurationBuckets:     []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5},
	}
}

// NewManager creates a new metrics manager.
func NewManager(cfg Config) *Manager {
	if !cfg.Enabled {
		return &Manager{enabled: false}
	}

	registry := prometheus.NewRegistry()

	// Register Go runtime metrics
	registry.MustRegister(prometheus.NewGoCollector())
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	m := &Manager{
		registry: registry,
		enabled:  true,
	}

	m.initWorkflowMetrics(cfg)
	m.initTaskMetrics(cfg)
	m.initLaneMetrics(cfg)
	m.initHTTPMetrics(cfg)

	return m
}

// Enabled returns whether metrics collection is enabled.
func (m *Manager) Enabled() bool {
	return m.enabled
}

// Handler returns the HTTP handler for the metrics endpoint.
func (m *Manager) Handler() http.Handler {
	if !m.enabled {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
	}
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// StartServer starts the metrics HTTP server on the configured port.
func (m *Manager) StartServer(ctx context.Context, port int, path string) error {
	if !m.enabled {
		return nil
	}

	mux := http.NewServeMux()
	mux.Handle(path, m.Handler())

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	return server.ListenAndServe()
}

// NoOpManager returns a no-op metrics manager for when metrics are disabled.
func NoOpManager() *Manager {
	return &Manager{enabled: false}
}
