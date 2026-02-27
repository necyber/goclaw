package cluster

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type chaosCoordinator struct {
	*MemoryCoordinator
	failHeartbeats atomic.Bool
}

func (c *chaosCoordinator) Heartbeat(ctx context.Context, nodeID, leaseID string, ttl time.Duration) (NodeState, error) {
	if c.failHeartbeats.Load() {
		return NodeState{}, errors.New("simulated coordination outage")
	}
	return c.MemoryCoordinator.Heartbeat(ctx, nodeID, leaseID, ttl)
}

func TestChaos_NodeLifecycleTransitionsOnCoordinationOutage(t *testing.T) {
	coord := &chaosCoordinator{MemoryCoordinator: NewMemoryCoordinator("memory")}

	manager, err := NewNodeLifecycleManager(coord, NodeRegistration{NodeID: "node-1"}, NodeLifecycleConfig{
		LeaseTTL:          200 * time.Millisecond,
		HeartbeatInterval: 50 * time.Millisecond,
		FailureThreshold:  1,
	})
	if err != nil {
		t.Fatalf("NewNodeLifecycleManager() error = %v", err)
	}

	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer manager.Stop(context.Background())

	coord.failHeartbeats.Store(true)
	if !waitState(manager, HealthStateUnhealthy, 2*time.Second) {
		t.Fatalf("expected unhealthy state, got %s", manager.State())
	}

	coord.failHeartbeats.Store(false)
	if !waitState(manager, HealthStateHealthy, 2*time.Second) {
		t.Fatalf("expected recovered healthy state, got %s", manager.State())
	}
}

func waitState(manager *NodeLifecycleManager, target HealthState, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if manager.State() == target {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}
