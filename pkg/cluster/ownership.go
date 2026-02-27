package cluster

import (
	"context"
	"fmt"
	"time"
)

// OwnershipManager wraps claim/renew/release operations and enforces fencing checks.
type OwnershipManager struct {
	coordination Coordinator
	claimTTL     time.Duration
}

// NewOwnershipManager creates an ownership manager.
func NewOwnershipManager(coordination Coordinator, claimTTL time.Duration) (*OwnershipManager, error) {
	if coordination == nil {
		return nil, fmt.Errorf("cluster: coordination cannot be nil")
	}
	if claimTTL <= 0 {
		return nil, fmt.Errorf("cluster: claim ttl must be > 0")
	}
	return &OwnershipManager{
		coordination: coordination,
		claimTTL:     claimTTL,
	}, nil
}

// ClaimShard claims ownership for a shard with fencing token semantics.
func (m *OwnershipManager) ClaimShard(ctx context.Context, shardKey, nodeID, nodeLeaseID string) (OwnershipClaim, error) {
	existing, ok, err := m.coordination.GetOwnership(ctx, shardKey)
	if err != nil {
		return OwnershipClaim{}, err
	}

	request := OwnershipClaimRequest{
		ShardKey:    shardKey,
		NodeID:      nodeID,
		NodeLeaseID: nodeLeaseID,
		TTL:         m.claimTTL,
	}
	// Carry the latest known token when renewing existing ownership.
	if ok {
		request.ExpectedToken = existing.FencingToken
	}

	return m.coordination.ClaimOwnership(ctx, request)
}

// RenewShard renews an existing claim and validates the provided fencing token.
func (m *OwnershipManager) RenewShard(ctx context.Context, shardKey, nodeID, nodeLeaseID string, fencingToken uint64) (OwnershipClaim, error) {
	return m.coordination.ClaimOwnership(ctx, OwnershipClaimRequest{
		ShardKey:      shardKey,
		NodeID:        nodeID,
		NodeLeaseID:   nodeLeaseID,
		TTL:           m.claimTTL,
		ExpectedToken: fencingToken,
	})
}

// ValidateOperation validates fencing token for ownership-sensitive operations.
func (m *OwnershipManager) ValidateOperation(ctx context.Context, shardKey, nodeID string, fencingToken uint64) error {
	return m.coordination.ValidateFencingToken(ctx, shardKey, nodeID, fencingToken)
}

// ReleaseShard releases ownership for a shard.
func (m *OwnershipManager) ReleaseShard(ctx context.Context, shardKey, nodeID, nodeLeaseID string, fencingToken uint64) error {
	return m.coordination.ReleaseOwnership(ctx, OwnershipReleaseRequest{
		ShardKey:      shardKey,
		NodeID:        nodeID,
		NodeLeaseID:   nodeLeaseID,
		ExpectedToken: fencingToken,
	})
}
