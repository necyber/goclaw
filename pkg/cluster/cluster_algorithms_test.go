package cluster

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryCoordinator_ClaimAndFencing(t *testing.T) {
	coord := NewMemoryCoordinator("memory")
	ctx := context.Background()

	lease, err := coord.Join(ctx, NodeRegistration{NodeID: "node-a"}, time.Second)
	if err != nil {
		t.Fatalf("Join() error = %v", err)
	}

	claim, err := coord.ClaimOwnership(ctx, OwnershipClaimRequest{
		ShardKey:    "lane:cpu:0",
		NodeID:      "node-a",
		NodeLeaseID: lease.LeaseID,
		TTL:         time.Second,
	})
	if err != nil {
		t.Fatalf("ClaimOwnership() error = %v", err)
	}
	if claim.FencingToken == 0 {
		t.Fatal("expected fencing token to be generated")
	}

	if err := coord.ValidateFencingToken(ctx, "lane:cpu:0", "node-a", claim.FencingToken); err != nil {
		t.Fatalf("ValidateFencingToken() error = %v", err)
	}
	if err := coord.ValidateFencingToken(ctx, "lane:cpu:0", "node-a", claim.FencingToken+1); !errors.Is(err, ErrFencingTokenInvalid) {
		t.Fatalf("expected ErrFencingTokenInvalid, got %v", err)
	}
}

func TestHashRing_StableRoutingAndRebalance(t *testing.T) {
	ring := NewHashRing(32)
	if err := ring.SetNodes([]string{"node-a", "node-b"}); err != nil {
		t.Fatalf("SetNodes() error = %v", err)
	}

	first, ok := ring.Owner("workflow:123")
	if !ok {
		t.Fatal("expected owner for shard key")
	}
	second, ok := ring.Owner("workflow:123")
	if !ok {
		t.Fatal("expected owner for shard key")
	}
	if first != second {
		t.Fatalf("expected stable owner, got %s and %s", first, second)
	}

	transfers := PlanRebalance(
		map[string]string{"s1": "node-a", "s2": "node-a", "s3": "node-b"},
		map[string]string{"s1": "node-a", "s2": "node-c", "s3": "node-b"},
		RebalanceReasonNodeJoin,
	)
	if len(transfers) != 1 {
		t.Fatalf("expected one rebalance transfer, got %d", len(transfers))
	}
	if transfers[0].ShardKey != "s2" || transfers[0].FromNode != "node-a" || transfers[0].ToNode != "node-c" {
		t.Fatalf("unexpected transfer plan: %+v", transfers[0])
	}
}
