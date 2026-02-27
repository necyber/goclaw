package cluster

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// NodeLifecycleConfig configures membership lifecycle behavior.
type NodeLifecycleConfig struct {
	LeaseTTL          time.Duration
	HeartbeatInterval time.Duration
	FailureThreshold  int
}

// DefaultNodeLifecycleConfig returns defaults suitable for local cluster simulation.
func DefaultNodeLifecycleConfig() NodeLifecycleConfig {
	return NodeLifecycleConfig{
		LeaseTTL:          10 * time.Second,
		HeartbeatInterval: 2 * time.Second,
		FailureThreshold:  3,
	}
}

// NodeLifecycleManager manages join/heartbeat/leave lifecycle and health transitions.
type NodeLifecycleManager struct {
	coordination Coordinator
	registration NodeRegistration
	cfg          NodeLifecycleConfig

	mu      sync.RWMutex
	lease   MembershipLease
	state   HealthState
	running bool

	heartbeatCancel context.CancelFunc
	onStateChange   func(from, to HealthState)
}

// NewNodeLifecycleManager creates a lifecycle manager bound to a coordinator.
func NewNodeLifecycleManager(coordination Coordinator, registration NodeRegistration, cfg NodeLifecycleConfig) (*NodeLifecycleManager, error) {
	if coordination == nil {
		return nil, fmt.Errorf("cluster: coordination cannot be nil")
	}
	if registration.NodeID == "" {
		return nil, fmt.Errorf("cluster: node id cannot be empty")
	}
	if cfg.LeaseTTL <= 0 || cfg.HeartbeatInterval <= 0 {
		return nil, fmt.Errorf("cluster: lease ttl and heartbeat interval must be > 0")
	}
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 1
	}
	return &NodeLifecycleManager{
		coordination: coordination,
		registration: registration,
		cfg:          cfg,
		state:        HealthStateUnknown,
	}, nil
}

// SetStateChangeHook sets a callback invoked on local health-state transitions.
func (m *NodeLifecycleManager) SetStateChangeHook(callback func(from, to HealthState)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onStateChange = callback
}

// Start registers node membership and starts heartbeat loop.
func (m *NodeLifecycleManager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()

	lease, err := m.coordination.Join(ctx, m.registration, m.cfg.LeaseTTL)
	if err != nil {
		return err
	}

	heartbeatCtx, cancel := context.WithCancel(context.Background())

	m.mu.Lock()
	m.lease = lease
	m.running = true
	m.heartbeatCancel = cancel
	m.mu.Unlock()

	m.transition(HealthStateHealthy)
	go m.heartbeatLoop(heartbeatCtx)
	return nil
}

func (m *NodeLifecycleManager) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(m.cfg.HeartbeatInterval)
	defer ticker.Stop()

	failures := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		m.mu.RLock()
		lease := m.lease
		m.mu.RUnlock()

		_, err := m.coordination.Heartbeat(ctx, m.registration.NodeID, lease.LeaseID, m.cfg.LeaseTTL)
		if err != nil {
			failures++
			if failures >= m.cfg.FailureThreshold {
				m.transition(HealthStateUnhealthy)
			}
			continue
		}

		failures = 0
		m.transition(HealthStateHealthy)
	}
}

// Stop stops heartbeat and leaves cluster membership.
func (m *NodeLifecycleManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return nil
	}
	cancel := m.heartbeatCancel
	lease := m.lease
	m.running = false
	m.heartbeatCancel = nil
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	m.transition(HealthStateLeaving)
	return m.coordination.Leave(ctx, m.registration.NodeID, lease.LeaseID)
}

// Lease returns the latest membership lease snapshot.
func (m *NodeLifecycleManager) Lease() MembershipLease {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lease
}

// State returns current local health state.
func (m *NodeLifecycleManager) State() HealthState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

func (m *NodeLifecycleManager) transition(next HealthState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.state == next {
		return
	}
	previous := m.state
	m.state = next
	if m.onStateChange != nil {
		m.onStateChange(previous, next)
	}
}
