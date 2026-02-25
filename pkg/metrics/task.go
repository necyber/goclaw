package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// initTaskMetrics initializes task-related metrics.
func (m *Manager) initTaskMetrics(cfg Config) {
	m.taskExecutions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "task_executions_total",
			Help: "Total number of task executions by status",
		},
		[]string{"status"},
	)

	m.taskDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "task_duration_seconds",
			Help:    "Task execution duration in seconds",
			Buckets: cfg.TaskDurationBuckets,
		},
		[]string{},
	)

	m.taskRetries = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "task_retries_total",
			Help: "Total number of task retries",
		},
		[]string{},
	)

	m.registry.MustRegister(m.taskExecutions)
	m.registry.MustRegister(m.taskDuration)
	m.registry.MustRegister(m.taskRetries)
}

// RecordTaskExecution records a task execution event.
func (m *Manager) RecordTaskExecution(status string) {
	if !m.enabled {
		return
	}
	m.taskExecutions.WithLabelValues(status).Inc()
}

// RecordTaskDuration records task execution duration.
func (m *Manager) RecordTaskDuration(duration time.Duration) {
	if !m.enabled {
		return
	}
	m.taskDuration.WithLabelValues().Observe(duration.Seconds())
}

// RecordTaskRetry records a task retry event.
func (m *Manager) RecordTaskRetry() {
	if !m.enabled {
		return
	}
	m.taskRetries.WithLabelValues().Inc()
}
