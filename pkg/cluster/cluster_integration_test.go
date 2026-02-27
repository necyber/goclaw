package cluster

import (
	"context"
	"testing"
	"time"
)

func TestIntegration_OwnershipTransferOnNodeFailure(t *testing.T) {
	coord := NewMemoryCoordinator("memory")
	ctx := context.Background()

	leaseA, err := coord.Join(ctx, NodeRegistration{NodeID: "node-a"}, 2*time.Second)
	if err != nil {
		t.Fatalf("Join(node-a) error = %v", err)
	}
	leaseB, err := coord.Join(ctx, NodeRegistration{NodeID: "node-b"}, 2*time.Second)
	if err != nil {
		t.Fatalf("Join(node-b) error = %v", err)
	}

	manager, err := NewOwnershipManager(coord, 2*time.Second)
	if err != nil {
		t.Fatalf("NewOwnershipManager() error = %v", err)
	}

	claimA, err := manager.ClaimShard(ctx, "workflow-shard-1", "node-a", leaseA.LeaseID)
	if err != nil {
		t.Fatalf("ClaimShard(node-a) error = %v", err)
	}

	if err := coord.Leave(ctx, "node-a", leaseA.LeaseID); err != nil {
		t.Fatalf("Leave(node-a) error = %v", err)
	}

	claimB, err := manager.ClaimShard(ctx, "workflow-shard-1", "node-b", leaseB.LeaseID)
	if err != nil {
		t.Fatalf("ClaimShard(node-b) error = %v", err)
	}
	if claimB.NodeID != "node-b" {
		t.Fatalf("expected node-b ownership, got %s", claimB.NodeID)
	}
	if claimB.FencingToken <= claimA.FencingToken {
		t.Fatalf("expected fencing token increase, old=%d new=%d", claimA.FencingToken, claimB.FencingToken)
	}
}

func TestIntegration_LeaderFailoverByLeaseExpiry(t *testing.T) {
	coord := NewMemoryCoordinator("memory")
	ctx := context.Background()

	if _, err := coord.Join(ctx, NodeRegistration{NodeID: "node-a"}, 2*time.Second); err != nil {
		t.Fatalf("Join(node-a) error = %v", err)
	}
	if _, err := coord.Join(ctx, NodeRegistration{NodeID: "node-b"}, 2*time.Second); err != nil {
		t.Fatalf("Join(node-b) error = %v", err)
	}

	leaderA, err := coord.AcquireLeaderLease(ctx, "node-a", 80*time.Millisecond)
	if err != nil {
		t.Fatalf("AcquireLeaderLease(node-a) error = %v", err)
	}
	if leaderA.NodeID != "node-a" {
		t.Fatalf("expected node-a leader, got %s", leaderA.NodeID)
	}

	time.Sleep(120 * time.Millisecond)

	leaderB, err := coord.AcquireLeaderLease(ctx, "node-b", 80*time.Millisecond)
	if err != nil {
		t.Fatalf("AcquireLeaderLease(node-b) error = %v", err)
	}
	if leaderB.NodeID != "node-b" {
		t.Fatalf("expected node-b leader, got %s", leaderB.NodeID)
	}
}
