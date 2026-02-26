package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func (m *Manager) initSignalMetrics() {
	m.signalSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "signal_sent_total",
			Help: "Total number of signals sent",
		},
		[]string{"mode", "type"},
	)

	m.signalReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "signal_received_total",
			Help: "Total number of signals delivered to subscribers",
		},
		[]string{"mode", "type"},
	)

	m.signalFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "signal_failures_total",
			Help: "Total number of signal delivery failures",
		},
		[]string{"mode", "type", "reason"},
	)

	m.signalPatternOps = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "signal_pattern_total",
			Help: "Total number of message pattern operations by pattern and status",
		},
		[]string{"pattern", "status"},
	)

	m.signalPatternDur = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "signal_pattern_duration_seconds",
			Help:    "Message pattern operation duration in seconds",
			Buckets: []float64{0.0005, 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5},
		},
		[]string{"pattern", "status"},
	)

	m.registry.MustRegister(m.signalSent)
	m.registry.MustRegister(m.signalReceived)
	m.registry.MustRegister(m.signalFailures)
	m.registry.MustRegister(m.signalPatternOps)
	m.registry.MustRegister(m.signalPatternDur)
}

// RecordSignalSent records a signal sent event.
func (m *Manager) RecordSignalSent(mode string, signalType string) {
	if !m.enabled {
		return
	}
	m.signalSent.WithLabelValues(mode, signalType).Inc()
}

// RecordSignalReceived records a signal received event.
func (m *Manager) RecordSignalReceived(mode string, signalType string) {
	if !m.enabled {
		return
	}
	m.signalReceived.WithLabelValues(mode, signalType).Inc()
}

// RecordSignalFailed records a failed signal operation.
func (m *Manager) RecordSignalFailed(mode string, signalType string, reason string) {
	if !m.enabled {
		return
	}
	m.signalFailures.WithLabelValues(mode, signalType, reason).Inc()
}

// RecordSignalPattern records message-pattern counters and latency.
func (m *Manager) RecordSignalPattern(pattern string, status string, duration time.Duration) {
	if !m.enabled {
		return
	}
	m.signalPatternOps.WithLabelValues(pattern, status).Inc()
	m.signalPatternDur.WithLabelValues(pattern, status).Observe(duration.Seconds())
}
