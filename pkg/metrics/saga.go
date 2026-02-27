package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func (m *Manager) initSagaMetrics(cfg Config) {
	m.sagaExecutions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "saga_executions_total",
			Help: "Total number of saga executions by terminal status",
		},
		[]string{"status"},
	)

	m.sagaDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "saga_duration_seconds",
			Help:    "Saga execution duration in seconds",
			Buckets: cfg.WorkflowDurationBuckets,
		},
		[]string{"status"},
	)

	m.sagaActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "saga_active_count",
			Help: "Current number of active saga executions",
		},
	)

	m.sagaCompensations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "saga_compensations_total",
			Help: "Total number of compensation phases by status",
		},
		[]string{"status"},
	)

	m.sagaCompensationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "saga_compensation_duration_seconds",
			Help:    "Compensation phase duration in seconds",
			Buckets: cfg.TaskDurationBuckets,
		},
		[]string{},
	)

	m.sagaCompensationRetries = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "saga_compensation_retries_total",
			Help: "Total number of compensation retries",
		},
		[]string{},
	)

	m.sagaRecovery = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "saga_recovery_total",
			Help: "Total number of saga recovery attempts by status",
		},
		[]string{"status"},
	)

	m.registry.MustRegister(m.sagaExecutions)
	m.registry.MustRegister(m.sagaDuration)
	m.registry.MustRegister(m.sagaActive)
	m.registry.MustRegister(m.sagaCompensations)
	m.registry.MustRegister(m.sagaCompensationDuration)
	m.registry.MustRegister(m.sagaCompensationRetries)
	m.registry.MustRegister(m.sagaRecovery)
}

// RecordSagaExecution records one saga execution outcome.
func (m *Manager) RecordSagaExecution(status string) {
	if !m.enabled {
		return
	}
	m.sagaExecutions.WithLabelValues(status).Inc()
}

// RecordSagaDuration records saga execution latency.
func (m *Manager) RecordSagaDuration(status string, duration time.Duration) {
	if !m.enabled {
		return
	}
	m.sagaDuration.WithLabelValues(status).Observe(duration.Seconds())
}

// IncActiveSagas increments current active saga count.
func (m *Manager) IncActiveSagas() {
	if !m.enabled {
		return
	}
	m.sagaActive.Inc()
}

// DecActiveSagas decrements current active saga count.
func (m *Manager) DecActiveSagas() {
	if !m.enabled {
		return
	}
	m.sagaActive.Dec()
}

// RecordCompensation records one compensation phase outcome.
func (m *Manager) RecordCompensation(status string) {
	if !m.enabled {
		return
	}
	m.sagaCompensations.WithLabelValues(status).Inc()
}

// RecordCompensationDuration records compensation phase duration.
func (m *Manager) RecordCompensationDuration(duration time.Duration) {
	if !m.enabled {
		return
	}
	m.sagaCompensationDuration.WithLabelValues().Observe(duration.Seconds())
}

// RecordCompensationRetry records one compensation retry.
func (m *Manager) RecordCompensationRetry() {
	if !m.enabled {
		return
	}
	m.sagaCompensationRetries.WithLabelValues().Inc()
}

// RecordSagaRecovery records one recovery operation outcome.
func (m *Manager) RecordSagaRecovery(status string) {
	if !m.enabled {
		return
	}
	m.sagaRecovery.WithLabelValues(status).Inc()
}
