package metrics

import "github.com/prometheus/client_golang/prometheus"

func (m *Manager) initDistributedMetrics() {
	m.ownershipChanges = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cluster_ownership_changes_total",
			Help: "Total number of ownership changes by reason",
		},
		[]string{"reason"},
	)

	m.redisOwnershipDecision = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "redis_lane_ownership_decision_total",
			Help: "Redis lane ownership decisions in distributed mode",
		},
		[]string{"lane_name", "decision"},
	)

	m.eventBusPublish = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "event_bus_publish_total",
			Help: "Total event bus publish attempts by status",
		},
		[]string{"status"},
	)

	m.eventBusRetries = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "event_bus_publish_retries_total",
			Help: "Total number of event-bus publish retries",
		},
	)

	m.eventBusDegraded = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "event_bus_degraded",
			Help: "Whether event-bus path is currently in degraded mode (1=degraded)",
		},
	)

	m.eventBusOutages = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "event_bus_outages_total",
			Help: "Total event-bus outage transitions",
		},
	)

	m.eventBusRecoveries = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "event_bus_recoveries_total",
			Help: "Total event-bus recovery transitions",
		},
	)

	m.registry.MustRegister(m.ownershipChanges)
	m.registry.MustRegister(m.redisOwnershipDecision)
	m.registry.MustRegister(m.eventBusPublish)
	m.registry.MustRegister(m.eventBusRetries)
	m.registry.MustRegister(m.eventBusDegraded)
	m.registry.MustRegister(m.eventBusOutages)
	m.registry.MustRegister(m.eventBusRecoveries)
}

// RecordOwnershipChange records distributed ownership transfer/change reason.
func (m *Manager) RecordOwnershipChange(reason string) {
	if !m.enabled {
		return
	}
	m.ownershipChanges.WithLabelValues(reason).Inc()
}

// RecordRedisOwnershipDecision records dequeue decision under ownership enforcement.
func (m *Manager) RecordRedisOwnershipDecision(laneName string, decision string) {
	if !m.enabled {
		return
	}
	m.redisOwnershipDecision.WithLabelValues(laneName, decision).Inc()
}

// RecordPublish records event-bus publish status.
func (m *Manager) RecordPublish(status string) {
	if !m.enabled {
		return
	}
	m.eventBusPublish.WithLabelValues(status).Inc()
}

// RecordRetry records event-bus publish retry.
func (m *Manager) RecordRetry() {
	if !m.enabled {
		return
	}
	m.eventBusRetries.Inc()
}

// SetDegradedMode sets event-bus degraded state gauge.
func (m *Manager) SetDegradedMode(active bool) {
	if !m.enabled {
		return
	}
	if active {
		m.eventBusDegraded.Set(1)
		return
	}
	m.eventBusDegraded.Set(0)
}

// RecordOutage records a degraded-mode transition into outage state.
func (m *Manager) RecordOutage() {
	if !m.enabled {
		return
	}
	m.eventBusOutages.Inc()
}

// RecordRecovery records a degraded-mode recovery transition.
func (m *Manager) RecordRecovery() {
	if !m.enabled {
		return
	}
	m.eventBusRecoveries.Inc()
}
