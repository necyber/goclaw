package cluster

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// MemoryCoordinator provides a deterministic in-memory implementation of Coordinator.
// It is suitable for unit tests and local development modes.
type MemoryCoordinator struct {
	mu sync.RWMutex

	backend string
	nowFn   func() time.Time

	nodes      map[string]NodeState
	leader     *LeaderLease
	ownerships map[string]OwnershipClaim

	watchers       map[int]chan MembershipEvent
	watcherSeq     int
	fencingCounter atomic.Uint64
}

// NewMemoryCoordinator creates a memory-backed coordinator with the given backend name.
func NewMemoryCoordinator(backend string) *MemoryCoordinator {
	if backend == "" {
		backend = "memory"
	}
	return &MemoryCoordinator{
		backend:    backend,
		nowFn:      time.Now,
		nodes:      make(map[string]NodeState),
		ownerships: make(map[string]OwnershipClaim),
		watchers:   make(map[int]chan MembershipEvent),
	}
}

// Join registers a node and creates/renews a membership lease.
func (c *MemoryCoordinator) Join(ctx context.Context, registration NodeRegistration, ttl time.Duration) (MembershipLease, error) {
	if err := ctx.Err(); err != nil {
		return MembershipLease{}, err
	}
	if registration.NodeID == "" {
		return MembershipLease{}, fmt.Errorf("cluster: node id cannot be empty")
	}
	if ttl <= 0 {
		return MembershipLease{}, fmt.Errorf("cluster: ttl must be > 0")
	}

	now := c.now()
	lease := MembershipLease{
		LeaseID:   c.newLeaseID(),
		NodeID:    registration.NodeID,
		ExpiresAt: now.Add(ttl),
	}

	c.mu.Lock()
	node := NodeState{
		NodeID:         registration.NodeID,
		Address:        registration.Address,
		Metadata:       cloneMap(registration.Metadata),
		Health:         HealthStateHealthy,
		LeaseID:        lease.LeaseID,
		LastHeartbeat:  now,
		LeaseExpiresAt: lease.ExpiresAt,
	}
	c.nodes[registration.NodeID] = node
	c.mu.Unlock()

	c.notify(MembershipEvent{
		Type:      MembershipEventJoined,
		Node:      node,
		Timestamp: now,
		Reason:    c.backend,
	})

	return lease, nil
}

// Heartbeat updates a node lease and marks the node healthy.
func (c *MemoryCoordinator) Heartbeat(ctx context.Context, nodeID, leaseID string, ttl time.Duration) (NodeState, error) {
	if err := ctx.Err(); err != nil {
		return NodeState{}, err
	}
	if ttl <= 0 {
		return NodeState{}, fmt.Errorf("cluster: ttl must be > 0")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	node, ok := c.nodes[nodeID]
	if !ok {
		return NodeState{}, ErrNodeNotFound
	}
	if node.LeaseID != leaseID {
		return NodeState{}, ErrLeaseMismatch
	}

	now := c.now()
	if now.After(node.LeaseExpiresAt) {
		node.Health = HealthStateUnhealthy
		c.nodes[nodeID] = node
		return NodeState{}, ErrLeaseExpired
	}

	previousHealth := node.Health
	node.Health = HealthStateHealthy
	node.LastHeartbeat = now
	node.LeaseExpiresAt = now.Add(ttl)
	c.nodes[nodeID] = node

	evtType := MembershipEventHeartbeat
	if previousHealth != HealthStateHealthy {
		evtType = MembershipEventStateChange
	}
	go c.notify(MembershipEvent{
		Type:      evtType,
		Node:      node,
		Timestamp: now,
		Reason:    c.backend,
	})

	return node, nil
}

// Leave removes node membership and any associated ownership.
func (c *MemoryCoordinator) Leave(ctx context.Context, nodeID, leaseID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	c.mu.Lock()
	node, ok := c.nodes[nodeID]
	if !ok {
		c.mu.Unlock()
		return ErrNodeNotFound
	}
	if node.LeaseID != leaseID {
		c.mu.Unlock()
		return ErrLeaseMismatch
	}

	now := c.now()
	node.Health = HealthStateLeaving
	c.nodes[nodeID] = node
	delete(c.nodes, nodeID)

	for shardKey, claim := range c.ownerships {
		if claim.NodeID == nodeID {
			delete(c.ownerships, shardKey)
		}
	}

	if c.leader != nil && c.leader.NodeID == nodeID {
		c.leader = nil
	}
	c.mu.Unlock()

	c.notify(MembershipEvent{
		Type:      MembershipEventLeft,
		Node:      node,
		Timestamp: now,
		Reason:    c.backend,
	})
	c.notify(MembershipEvent{
		Type:       MembershipEventLeader,
		LeaderNode: "",
		Timestamp:  now,
		Reason:     c.backend,
	})

	return nil
}

// ListNodes returns current nodes sorted by node ID for deterministic callers.
func (c *MemoryCoordinator) ListNodes(ctx context.Context) ([]NodeState, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]NodeState, 0, len(c.nodes))
	for _, node := range c.nodes {
		out = append(out, cloneNode(node))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NodeID < out[j].NodeID })
	return out, nil
}

// WatchMembership returns a watch channel for membership and leader events.
func (c *MemoryCoordinator) WatchMembership(ctx context.Context) (<-chan MembershipEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ch := make(chan MembershipEvent, 32)

	c.mu.Lock()
	id := c.watcherSeq
	c.watcherSeq++
	c.watchers[id] = ch
	c.mu.Unlock()

	go func() {
		<-ctx.Done()
		c.mu.Lock()
		if existing, ok := c.watchers[id]; ok {
			delete(c.watchers, id)
			close(existing)
		}
		c.mu.Unlock()
	}()

	return ch, nil
}

// AcquireLeaderLease acquires the cluster leader lease for a node.
func (c *MemoryCoordinator) AcquireLeaderLease(ctx context.Context, nodeID string, ttl time.Duration) (LeaderLease, error) {
	if err := ctx.Err(); err != nil {
		return LeaderLease{}, err
	}
	if ttl <= 0 {
		return LeaderLease{}, fmt.Errorf("cluster: ttl must be > 0")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	node, ok := c.nodes[nodeID]
	if !ok {
		return LeaderLease{}, ErrNodeNotFound
	}
	now := c.now()
	if now.After(node.LeaseExpiresAt) {
		return LeaderLease{}, ErrLeaseExpired
	}

	if c.leader != nil && now.Before(c.leader.ExpiresAt) && c.leader.NodeID != nodeID {
		return LeaderLease{}, ErrLeaderLeaseHeld
	}

	lease := LeaderLease{
		LeaseID:   c.newLeaseID(),
		NodeID:    nodeID,
		ExpiresAt: now.Add(ttl),
	}
	c.leader = &lease

	go c.notify(MembershipEvent{
		Type:       MembershipEventLeader,
		LeaderNode: nodeID,
		Node:       node,
		Timestamp:  now,
		Reason:     c.backend,
	})

	return lease, nil
}

// RenewLeaderLease renews the current leader lease.
func (c *MemoryCoordinator) RenewLeaderLease(ctx context.Context, leaseID string, ttl time.Duration) (LeaderLease, error) {
	if err := ctx.Err(); err != nil {
		return LeaderLease{}, err
	}
	if ttl <= 0 {
		return LeaderLease{}, fmt.Errorf("cluster: ttl must be > 0")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.leader == nil {
		return LeaderLease{}, ErrNodeNotFound
	}
	if c.leader.LeaseID != leaseID {
		return LeaderLease{}, ErrLeaseMismatch
	}
	now := c.now()
	if now.After(c.leader.ExpiresAt) {
		return LeaderLease{}, ErrLeaseExpired
	}

	c.leader.ExpiresAt = now.Add(ttl)
	return *c.leader, nil
}

// ReleaseLeaderLease releases the current leader lease.
func (c *MemoryCoordinator) ReleaseLeaderLease(ctx context.Context, leaseID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.leader == nil {
		return nil
	}
	if c.leader.LeaseID != leaseID {
		return ErrLeaseMismatch
	}
	c.leader = nil
	go c.notify(MembershipEvent{
		Type:       MembershipEventLeader,
		LeaderNode: "",
		Timestamp:  c.now(),
		Reason:     c.backend,
	})
	return nil
}

// CurrentLeader returns the current leader lease if present and unexpired.
func (c *MemoryCoordinator) CurrentLeader(ctx context.Context) (LeaderLease, bool, error) {
	if err := ctx.Err(); err != nil {
		return LeaderLease{}, false, err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.leader == nil {
		return LeaderLease{}, false, nil
	}
	now := c.now()
	if now.After(c.leader.ExpiresAt) {
		return LeaderLease{}, false, nil
	}
	return *c.leader, true, nil
}

// ClaimOwnership creates or renews shard ownership with fencing semantics.
func (c *MemoryCoordinator) ClaimOwnership(ctx context.Context, request OwnershipClaimRequest) (OwnershipClaim, error) {
	if err := ctx.Err(); err != nil {
		return OwnershipClaim{}, err
	}
	if request.ShardKey == "" || request.NodeID == "" || request.NodeLeaseID == "" {
		return OwnershipClaim{}, fmt.Errorf("cluster: shard key, node id and lease id are required")
	}
	if request.TTL <= 0 {
		return OwnershipClaim{}, fmt.Errorf("cluster: ttl must be > 0")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	node, ok := c.nodes[request.NodeID]
	if !ok {
		return OwnershipClaim{}, ErrNodeNotFound
	}
	if node.LeaseID != request.NodeLeaseID {
		return OwnershipClaim{}, ErrLeaseMismatch
	}
	if now.After(node.LeaseExpiresAt) {
		return OwnershipClaim{}, ErrLeaseExpired
	}

	current, exists := c.ownerships[request.ShardKey]
	if exists && now.Before(current.LeaseExpires) {
		if request.ExpectedToken > 0 && current.FencingToken != request.ExpectedToken {
			return OwnershipClaim{}, ErrFencingTokenInvalid
		}
		if current.NodeID != request.NodeID {
			return OwnershipClaim{}, ErrOwnershipConflict
		}
		// same owner renews the claim without changing token
		current.LeaseExpires = now.Add(request.TTL)
		current.UpdatedAt = now
		c.ownerships[request.ShardKey] = current
		return current, nil
	}

	token := c.fencingCounter.Add(1)
	claim := OwnershipClaim{
		ShardKey:     request.ShardKey,
		NodeID:       request.NodeID,
		NodeLeaseID:  request.NodeLeaseID,
		FencingToken: token,
		LeaseExpires: now.Add(request.TTL),
		UpdatedAt:    now,
	}
	c.ownerships[request.ShardKey] = claim
	return claim, nil
}

// GetOwnership returns ownership for a shard if a valid claim exists.
func (c *MemoryCoordinator) GetOwnership(ctx context.Context, shardKey string) (OwnershipClaim, bool, error) {
	if err := ctx.Err(); err != nil {
		return OwnershipClaim{}, false, err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	claim, ok := c.ownerships[shardKey]
	if !ok {
		return OwnershipClaim{}, false, nil
	}
	if c.now().After(claim.LeaseExpires) {
		return OwnershipClaim{}, false, nil
	}
	return claim, true, nil
}

// ReleaseOwnership releases ownership when lease and fencing token are valid.
func (c *MemoryCoordinator) ReleaseOwnership(ctx context.Context, request OwnershipReleaseRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	claim, ok := c.ownerships[request.ShardKey]
	if !ok {
		return nil
	}
	if claim.NodeID != request.NodeID || claim.NodeLeaseID != request.NodeLeaseID {
		return ErrOwnershipConflict
	}
	if request.ExpectedToken > 0 && claim.FencingToken != request.ExpectedToken {
		return ErrFencingTokenInvalid
	}
	delete(c.ownerships, request.ShardKey)
	return nil
}

// ValidateFencingToken validates the current fencing token for ownership-sensitive operations.
func (c *MemoryCoordinator) ValidateFencingToken(ctx context.Context, shardKey, nodeID string, token uint64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	claim, ok := c.ownerships[shardKey]
	if !ok {
		return ErrOwnershipConflict
	}
	if c.now().After(claim.LeaseExpires) {
		return ErrLeaseExpired
	}
	if claim.NodeID != nodeID {
		return ErrOwnershipConflict
	}
	if claim.FencingToken != token {
		return ErrFencingTokenInvalid
	}
	return nil
}

func (c *MemoryCoordinator) notify(event MembershipEvent) {
	c.mu.RLock()
	targets := make([]chan MembershipEvent, 0, len(c.watchers))
	for _, ch := range c.watchers {
		targets = append(targets, ch)
	}
	c.mu.RUnlock()

	for _, ch := range targets {
		select {
		case ch <- event:
		default:
			// Watchers are best-effort and intentionally non-blocking.
		}
	}
}

func (c *MemoryCoordinator) now() time.Time {
	if c.nowFn == nil {
		return time.Now()
	}
	return c.nowFn()
}

func (c *MemoryCoordinator) newLeaseID() string {
	return uuid.NewString()
}

func cloneMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneNode(in NodeState) NodeState {
	out := in
	out.Metadata = cloneMap(in.Metadata)
	return out
}
