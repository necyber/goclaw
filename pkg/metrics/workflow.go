package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// initWorkflowMetrics initializes workflow-related metrics.
func (m *Manager) initWorkflowMetrics(cfg Config) {
	m.workflowSubmissions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "workflow_submissions_total",
			Help: "Total number of workflow submissions by status",
		},
		[]string{"status"},
	)

	m.workflowDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "workflow_duration_seconds",
			Help:    "Workflow execution duration in seconds",
			Buckets: cfg.WorkflowDurationBuckets,
		},
		[]string{"status"},
	)

	m.workflowActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "workflow_active_count",
			Help: "Current number of active workflows by status",
		},
		[]string{"status"},
	)

	m.registry.MustRegister(m.workflowSubmissions)
	m.registry.MustRegister(m.workflowDuration)
	m.registry.MustRegister(m.workflowActive)
}

// RecordWorkflowSubmission records a workflow submission event.
func (m *Manager) RecordWorkflowSubmission(status string) {
	if !m.enabled {
		return
	}
	m.workflowSubmissions.WithLabelValues(status).Inc()
}

// RecordWorkflowDuration records workflow execution duration.
func (m *Manager) RecordWorkflowDuration(status string, duration time.Duration) {
	if !m.enabled {
		return
	}
	m.workflowDuration.WithLabelValues(status).Observe(duration.Seconds())
}

// SetActiveWorkflows sets the current number of active workflows.
func (m *Manager) SetActiveWorkflows(status string, count float64) {
	if !m.enabled {
		return
	}
	m.workflowActive.WithLabelValues(status).Set(count)
}

// IncActiveWorkflows increments the active workflow count.
func (m *Manager) IncActiveWorkflows(status string) {
	if !m.enabled {
		return
	}
	m.workflowActive.WithLabelValues(status).Inc()
}

// DecActiveWorkflows decrements the active workflow count.
func (m *Manager) DecActiveWorkflows(status string) {
	if !m.enabled {
		return
	}
	m.workflowActive.WithLabelValues(status).Dec()
}
