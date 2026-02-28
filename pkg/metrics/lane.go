package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// initLaneMetrics initializes lane queue metrics.
func (m *Manager) initLaneMetrics(cfg Config) {
	m.laneQueueDepth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lane_queue_depth",
			Help: "Current depth of lane queue",
		},
		[]string{"lane_name"},
	)

	m.laneWaitDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "lane_wait_duration_seconds",
			Help:    "Time tasks spend waiting in queue",
			Buckets: cfg.LaneWaitBuckets,
		},
		[]string{"lane_name"},
	)

	m.laneThroughput = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lane_throughput_total",
			Help: "Total number of tasks processed by lane",
		},
		[]string{"lane_name"},
	)

	m.laneSubmission = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lane_submission_outcomes_total",
			Help: "Total number of lane submissions by canonical outcome",
		},
		[]string{"lane_name", "outcome"},
	)

	m.redisQueueDepth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "redis_lane_queue_depth",
			Help: "Current depth of Redis-backed lane queue",
		},
		[]string{"lane_name"},
	)

	m.redisSubmitDur = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "redis_lane_submit_duration_seconds",
			Help:    "Redis lane submit duration in seconds",
			Buckets: cfg.LaneWaitBuckets,
		},
		[]string{"lane_name"},
	)

	m.redisThroughput = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "redis_lane_throughput_total",
			Help: "Total number of tasks processed by Redis-backed lanes",
		},
		[]string{"lane_name"},
	)

	m.registry.MustRegister(m.laneQueueDepth)
	m.registry.MustRegister(m.laneWaitDuration)
	m.registry.MustRegister(m.laneThroughput)
	m.registry.MustRegister(m.laneSubmission)
	m.registry.MustRegister(m.redisQueueDepth)
	m.registry.MustRegister(m.redisSubmitDur)
	m.registry.MustRegister(m.redisThroughput)
}

// SetQueueDepth sets the current queue depth for a lane.
func (m *Manager) SetQueueDepth(laneName string, depth float64) {
	if !m.enabled {
		return
	}
	m.laneQueueDepth.WithLabelValues(laneName).Set(depth)
}

// IncQueueDepth increments the queue depth for a lane.
func (m *Manager) IncQueueDepth(laneName string) {
	if !m.enabled {
		return
	}
	m.laneQueueDepth.WithLabelValues(laneName).Inc()
}

// DecQueueDepth decrements the queue depth for a lane.
func (m *Manager) DecQueueDepth(laneName string) {
	if !m.enabled {
		return
	}
	m.laneQueueDepth.WithLabelValues(laneName).Dec()
}

// RecordWaitDuration records the time a task spent waiting in queue.
func (m *Manager) RecordWaitDuration(laneName string, duration time.Duration) {
	if !m.enabled {
		return
	}
	m.laneWaitDuration.WithLabelValues(laneName).Observe(duration.Seconds())
}

// RecordThroughput records a task being processed by a lane.
func (m *Manager) RecordThroughput(laneName string) {
	if !m.enabled {
		return
	}
	m.laneThroughput.WithLabelValues(laneName).Inc()
}

// RecordSubmissionOutcome records canonical lane submission outcomes.
func (m *Manager) RecordSubmissionOutcome(laneName string, outcome string) {
	if !m.enabled {
		return
	}

	switch outcome {
	case "accepted", "rejected", "redirected", "dropped":
		m.laneSubmission.WithLabelValues(laneName, outcome).Inc()
	default:
		// Ignore unknown outcomes to keep label cardinality bounded.
	}
}

// SetRedisQueueDepth sets the current queue depth for a Redis-backed lane.
func (m *Manager) SetRedisQueueDepth(laneName string, depth float64) {
	if !m.enabled {
		return
	}
	m.redisQueueDepth.WithLabelValues(laneName).Set(depth)
}

// RecordRedisSubmitDuration records Redis submit latency.
func (m *Manager) RecordRedisSubmitDuration(laneName string, duration time.Duration) {
	if !m.enabled {
		return
	}
	m.redisSubmitDur.WithLabelValues(laneName).Observe(duration.Seconds())
}

// RecordRedisThroughput records a processed task for a Redis lane.
func (m *Manager) RecordRedisThroughput(laneName string) {
	if !m.enabled {
		return
	}
	m.redisThroughput.WithLabelValues(laneName).Inc()
}
