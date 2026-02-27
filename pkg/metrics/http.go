package metrics

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
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
	m.recordHTTPRequest(context.Background(), method, path, status, duration)
}

// RecordHTTPRequestWithContext records an HTTP request and attaches exemplar trace
// labels when the current span context is valid and the backend supports exemplars.
func (m *Manager) RecordHTTPRequestWithContext(ctx context.Context, method, path, status string, duration time.Duration) {
	m.recordHTTPRequest(ctx, method, path, status, duration)
}

func (m *Manager) recordHTTPRequest(ctx context.Context, method, path, status string, duration time.Duration) {
	if !m.enabled {
		return
	}

	exemplar, hasExemplar := traceExemplarLabels(ctx)

	requestCounter := m.httpRequests.WithLabelValues(method, path, status)
	if hasExemplar {
		if exemplarAdder, ok := requestCounter.(prometheus.ExemplarAdder); ok {
			exemplarAdder.AddWithExemplar(1, exemplar)
		} else {
			requestCounter.Inc()
		}
	} else {
		requestCounter.Inc()
	}

	requestDuration := m.httpDuration.WithLabelValues(method, path)
	if hasExemplar {
		if exemplarObserver, ok := requestDuration.(prometheus.ExemplarObserver); ok {
			exemplarObserver.ObserveWithExemplar(duration.Seconds(), exemplar)
		} else {
			requestDuration.Observe(duration.Seconds())
		}
	} else {
		requestDuration.Observe(duration.Seconds())
	}
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

func traceExemplarLabels(ctx context.Context) (prometheus.Labels, bool) {
	if ctx == nil {
		return nil, false
	}

	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return nil, false
	}

	return prometheus.Labels{
		"trace_id": spanCtx.TraceID().String(),
		"span_id":  spanCtx.SpanID().String(),
	}, true
}
