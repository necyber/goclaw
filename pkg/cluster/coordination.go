package cluster

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNodeNotFound indicates the requested node does not exist.
	ErrNodeNotFound = errors.New("cluster: node not found")
	// ErrLeaseMismatch indicates the provided lease identifier does not match.
	ErrLeaseMismatch = errors.New("cluster: lease mismatch")
	// ErrLeaseExpired indicates a lease is expired and cannot be used.
	ErrLeaseExpired = errors.New("cluster: lease expired")
	// ErrLeaderLeaseHeld indicates there is an active leader lease holder.
	ErrLeaderLeaseHeld = errors.New("cluster: leader lease already held")
	// ErrOwnershipConflict indicates an ownership claim conflicts with an active owner.
	ErrOwnershipConflict = errors.New("cluster: ownership conflict")
	// ErrFencingTokenInvalid indicates the supplied fencing token is stale or invalid.
	ErrFencingTokenInvalid = errors.New("cluster: invalid fencing token")
)

// HealthState represents runtime health for a node in cluster membership.
type HealthState string

const (
	HealthStateUnknown   HealthState = "unknown"
	HealthStateHealthy   HealthState = "healthy"
	HealthStateUnhealthy HealthState = "unhealthy"
	HealthStateLeaving   HealthState = "leaving"
)

// MembershipEventType represents cluster membership/watch event types.
type MembershipEventType string

const (
	MembershipEventJoined      MembershipEventType = "joined"
	MembershipEventHeartbeat   MembershipEventType = "heartbeat"
	MembershipEventStateChange MembershipEventType = "state_changed"
	MembershipEventLeft        MembershipEventType = "left"
	MembershipEventLeader      MembershipEventType = "leader_changed"
)

// NodeRegistration describes a node joining the cluster.
type NodeRegistration struct {
	NodeID   string
	Address  string
	Metadata map[string]string
}

// NodeState describes a node as observed from coordination storage.
type NodeState struct {
	NodeID         string
	Address        string
	Metadata       map[string]string
	Health         HealthState
	LeaseID        string
	LastHeartbeat  time.Time
	LeaseExpiresAt time.Time
}

// MembershipLease is a lease result returned by join/leader claim operations.
type MembershipLease struct {
	LeaseID   string
	NodeID    string
	ExpiresAt time.Time
}

// LeaderLease represents a leader lease claim.
type LeaderLease struct {
	LeaseID   string
	NodeID    string
	ExpiresAt time.Time
}

// MembershipEvent is emitted for membership and leader changes.
type MembershipEvent struct {
	Type       MembershipEventType
	Node       NodeState
	LeaderNode string
	Timestamp  time.Time
	Reason     string
}

// OwnershipClaimRequest requests shard ownership under a node lease.
type OwnershipClaimRequest struct {
	ShardKey      string
	NodeID        string
	NodeLeaseID   string
	TTL           time.Duration
	ExpectedToken uint64
}

// OwnershipClaim records active shard ownership and fencing token.
type OwnershipClaim struct {
	ShardKey     string
	NodeID       string
	NodeLeaseID  string
	FencingToken uint64
	LeaseExpires time.Time
	UpdatedAt    time.Time
}

// OwnershipReleaseRequest releases ownership for a shard.
type OwnershipReleaseRequest struct {
	ShardKey      string
	NodeID        string
	NodeLeaseID   string
	ExpectedToken uint64
}

// Coordinator is the unified coordination abstraction for distributed runtime state.
type Coordinator interface {
	Join(ctx context.Context, registration NodeRegistration, ttl time.Duration) (MembershipLease, error)
	Heartbeat(ctx context.Context, nodeID, leaseID string, ttl time.Duration) (NodeState, error)
	Leave(ctx context.Context, nodeID, leaseID string) error
	ListNodes(ctx context.Context) ([]NodeState, error)
	WatchMembership(ctx context.Context) (<-chan MembershipEvent, error)

	AcquireLeaderLease(ctx context.Context, nodeID string, ttl time.Duration) (LeaderLease, error)
	RenewLeaderLease(ctx context.Context, leaseID string, ttl time.Duration) (LeaderLease, error)
	ReleaseLeaderLease(ctx context.Context, leaseID string) error
	CurrentLeader(ctx context.Context) (LeaderLease, bool, error)

	ClaimOwnership(ctx context.Context, request OwnershipClaimRequest) (OwnershipClaim, error)
	GetOwnership(ctx context.Context, shardKey string) (OwnershipClaim, bool, error)
	ReleaseOwnership(ctx context.Context, request OwnershipReleaseRequest) error
	ValidateFencingToken(ctx context.Context, shardKey, nodeID string, token uint64) error
}
