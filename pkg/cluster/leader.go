package cluster

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// LeaderElectorConfig configures leader lease acquire/renew behavior.
type LeaderElectorConfig struct {
	LeaseTTL      time.Duration
	RenewInterval time.Duration
	AcquireRetry  time.Duration
}

// DefaultLeaderElectorConfig returns default leader election timings.
func DefaultLeaderElectorConfig() LeaderElectorConfig {
	return LeaderElectorConfig{
		LeaseTTL:      8 * time.Second,
		RenewInterval: 2 * time.Second,
		AcquireRetry:  500 * time.Millisecond,
	}
}

// LeadershipState captures current leadership status.
type LeadershipState struct {
	IsLeader bool
	Lease    LeaderLease
	At       time.Time
	Reason   string
}

// LeaderElector acquires and renews leader lease with failover handling.
type LeaderElector struct {
	coordination Coordinator
	nodeID       string
	cfg          LeaderElectorConfig

	mu          sync.RWMutex
	state       LeadershipState
	running     bool
	cancel      context.CancelFunc
	subscribers map[int]chan LeadershipState
	subSeq      int
}

// NewLeaderElector creates a leader elector for a specific node.
func NewLeaderElector(coordination Coordinator, nodeID string, cfg LeaderElectorConfig) (*LeaderElector, error) {
	if coordination == nil {
		return nil, fmt.Errorf("cluster: coordination cannot be nil")
	}
	if nodeID == "" {
		return nil, fmt.Errorf("cluster: node id cannot be empty")
	}
	if cfg.LeaseTTL <= 0 || cfg.RenewInterval <= 0 || cfg.AcquireRetry <= 0 {
		return nil, fmt.Errorf("cluster: lease ttl/renew interval/acquire retry must be > 0")
	}
	return &LeaderElector{
		coordination: coordination,
		nodeID:       nodeID,
		cfg:          cfg,
		state: LeadershipState{
			IsLeader: false,
			At:       time.Now().UTC(),
			Reason:   "init",
		},
		subscribers: make(map[int]chan LeadershipState),
	}, nil
}

// Start launches the leader acquisition and renewal loop.
func (e *LeaderElector) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return nil
	}
	loopCtx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	e.running = true
	e.mu.Unlock()

	go e.run(loopCtx)
	return ctx.Err()
}

func (e *LeaderElector) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			e.releaseLeadership(context.Background(), "stopped")
			return
		default:
		}

		if !e.isLeader() {
			lease, err := e.coordination.AcquireLeaderLease(ctx, e.nodeID, e.cfg.LeaseTTL)
			if err != nil {
				e.publish(false, LeaderLease{}, "acquire_failed")
				select {
				case <-ctx.Done():
					return
				case <-time.After(e.cfg.AcquireRetry):
				}
				continue
			}
			e.publish(true, lease, "acquired")
			continue
		}

		select {
		case <-ctx.Done():
			e.releaseLeadership(context.Background(), "stopped")
			return
		case <-time.After(e.cfg.RenewInterval):
		}

		current := e.State()
		lease, err := e.coordination.RenewLeaderLease(ctx, current.Lease.LeaseID, e.cfg.LeaseTTL)
		if err != nil {
			e.publish(false, LeaderLease{}, "renew_failed")
			continue
		}
		e.publish(true, lease, "renewed")
	}
}

// Stop stops election loop and releases leader lease if held.
func (e *LeaderElector) Stop(ctx context.Context) error {
	e.mu.Lock()
	if !e.running {
		e.mu.Unlock()
		return nil
	}
	cancel := e.cancel
	e.running = false
	e.cancel = nil
	e.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	e.releaseLeadership(ctx, "stopped")
	return nil
}

// Subscribe returns leadership state updates.
func (e *LeaderElector) Subscribe(ctx context.Context) (<-chan LeadershipState, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ch := make(chan LeadershipState, 16)

	e.mu.Lock()
	id := e.subSeq
	e.subSeq++
	e.subscribers[id] = ch
	current := e.state
	e.mu.Unlock()

	ch <- current

	go func() {
		<-ctx.Done()
		e.mu.Lock()
		if existing, ok := e.subscribers[id]; ok {
			delete(e.subscribers, id)
			close(existing)
		}
		e.mu.Unlock()
	}()

	return ch, nil
}

// State returns current leadership state.
func (e *LeaderElector) State() LeadershipState {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state
}

func (e *LeaderElector) isLeader() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state.IsLeader
}

func (e *LeaderElector) releaseLeadership(ctx context.Context, reason string) {
	current := e.State()
	if current.IsLeader {
		_ = e.coordination.ReleaseLeaderLease(ctx, current.Lease.LeaseID)
	}
	e.publish(false, LeaderLease{}, reason)
}

func (e *LeaderElector) publish(isLeader bool, lease LeaderLease, reason string) {
	e.mu.Lock()
	e.state = LeadershipState{
		IsLeader: isLeader,
		Lease:    lease,
		At:       time.Now().UTC(),
		Reason:   reason,
	}
	state := e.state
	targets := make([]chan LeadershipState, 0, len(e.subscribers))
	for _, ch := range e.subscribers {
		targets = append(targets, ch)
	}
	e.mu.Unlock()

	for _, ch := range targets {
		select {
		case ch <- state:
		default:
			// best-effort updates to avoid blocking the elector loop
		}
	}
}
