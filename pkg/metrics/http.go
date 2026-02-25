package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// initHTTPMetrics initializes HTTP API metrics.
func (m *Manager) initHTTPMetrics(cfg Config) {
	m.httpRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	m.httpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: cfg.HTTPDurationBuckets,
		},
		[]string{"method", "path"},
	)

	m.httpConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_active_connections",
			Help: "Current number of active HTTP connections",
		},
	)

	m.registry.MustRegister(m.httpRequests)
	m.registry.MustRegister(m.httpDuration)
	m.registry.MustRegister(m.httpConnections)
}

// RecordHTTPRequest records an HTTP request with method, path, and status.
func (m *Manager) RecordHTTPRequest(method, path, status string, duration time.Duration) {
	if !m.enabled {
		return
	}
	m.httpRequests.WithLabelValues(method, path, status).Inc()
	m.httpDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

// IncActiveConnections increments the active HTTP connections count.
func (m *Manager) IncActiveConnections() {
	if !m.enabled {
		return
	}
	m.httpConnections.Inc()
}

// DecActiveConnections decrements the active HTTP connections count.
func (m *Manager) DecActiveConnections() {
	if !m.enabled {
		return
	}
	m.httpConnections.Dec()
}
