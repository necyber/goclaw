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

	m.registry.MustRegister(m.laneQueueDepth)
	m.registry.MustRegister(m.laneWaitDuration)
	m.registry.MustRegister(m.laneThroughput)
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
