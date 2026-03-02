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
		[]string{"status", "task_type"},
	)

	m.taskDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "task_duration_seconds",
			Help:    "Task execution duration in seconds",
			Buckets: cfg.TaskDurationBuckets,
		},
		[]string{"task_type"},
	)

	m.taskRetries = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "task_retries_total",
			Help: "Total number of task retries",
		},
		[]string{"task_type"},
	)

	m.registry.MustRegister(m.taskExecutions)
	m.registry.MustRegister(m.taskDuration)
	m.registry.MustRegister(m.taskRetries)
}

// RecordTaskExecution records a task execution event.
func (m *Manager) RecordTaskExecution(status string, taskType string) {
	if !m.enabled {
		return
	}
	taskType, ok := m.cardinalityGuard.admit("task_executions_total", "task_type", taskType)
	if !ok {
		m.labelValuesDropped.WithLabelValues("task_executions_total", "task_type").Inc()
		return
	}
	m.taskExecutions.WithLabelValues(status, taskType).Inc()
}

// RecordTaskDuration records task execution duration.
func (m *Manager) RecordTaskDuration(taskType string, duration time.Duration) {
	if !m.enabled {
		return
	}
	taskType, ok := m.cardinalityGuard.admit("task_duration_seconds", "task_type", taskType)
	if !ok {
		m.labelValuesDropped.WithLabelValues("task_duration_seconds", "task_type").Inc()
		return
	}
	m.taskDuration.WithLabelValues(taskType).Observe(duration.Seconds())
}

// RecordTaskRetry records a task retry event.
func (m *Manager) RecordTaskRetry(taskType string) {
	if !m.enabled {
		return
	}
	taskType, ok := m.cardinalityGuard.admit("task_retries_total", "task_type", taskType)
	if !ok {
		m.labelValuesDropped.WithLabelValues("task_retries_total", "task_type").Inc()
		return
	}
	m.taskRetries.WithLabelValues(taskType).Inc()
}
