package signal

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// OwnershipResolver resolves task ownership for distributed signal routing.
type OwnershipResolver interface {
	ResolveTaskOwner(ctx context.Context, taskID string) (ownerNodeID string, isLocal bool, err error)
}

// RemotePublisher routes a signal to a remote node.
type RemotePublisher interface {
	PublishRemote(ctx context.Context, nodeID string, sig *Signal) error
}

// DistributedBus routes signals according to distributed ownership resolution.
type DistributedBus struct {
	local    Bus
	resolver OwnershipResolver
	remote   RemotePublisher

	mu   sync.Mutex
	seen map[string]time.Time
}

// NewDistributedBus creates a distributed signal routing bus.
func NewDistributedBus(local Bus, resolver OwnershipResolver, remote RemotePublisher) (*DistributedBus, error) {
	if local == nil {
		return nil, fmt.Errorf("signal: local bus cannot be nil")
	}
	if resolver == nil {
		return nil, fmt.Errorf("signal: ownership resolver cannot be nil")
	}
	if remote == nil {
		return nil, fmt.Errorf("signal: remote publisher cannot be nil")
	}
	return &DistributedBus{
		local:    local,
		resolver: resolver,
		remote:   remote,
		seen:     make(map[string]time.Time),
	}, nil
}

// Publish routes signals to local fast-path or remote owner path.
func (b *DistributedBus) Publish(ctx context.Context, sig *Signal) error {
	if sig == nil {
		return fmt.Errorf("signal: signal cannot be nil")
	}
	if sig.TaskID == "" {
		return fmt.Errorf("signal: task id cannot be empty")
	}

	key := signalDedupeKey(sig)
	if b.markSeen(key) {
		// Duplicate route input is suppressed to avoid double delivery.
		return nil
	}

	ownerNode, isLocal, err := b.resolver.ResolveTaskOwner(ctx, sig.TaskID)
	if err != nil {
		return err
	}
	if isLocal {
		return b.local.Publish(ctx, sig)
	}
	return b.remote.PublishRemote(ctx, ownerNode, sig)
}

// Subscribe delegates to local bus.
func (b *DistributedBus) Subscribe(ctx context.Context, taskID string) (<-chan *Signal, error) {
	return b.local.Subscribe(ctx, taskID)
}

// Unsubscribe delegates to local bus.
func (b *DistributedBus) Unsubscribe(taskID string) error {
	return b.local.Unsubscribe(taskID)
}

// Close delegates to local bus.
func (b *DistributedBus) Close() error {
	return b.local.Close()
}

// Healthy reports local bus health.
func (b *DistributedBus) Healthy() bool {
	return b.local.Healthy()
}

func (b *DistributedBus) markSeen(key string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now().UTC()
	for seenKey, seenAt := range b.seen {
		if now.Sub(seenAt) > 2*time.Minute {
			delete(b.seen, seenKey)
		}
	}
	if _, exists := b.seen[key]; exists {
		return true
	}
	b.seen[key] = now
	return false
}

func signalDedupeKey(sig *Signal) string {
	payload := ""
	if len(sig.Payload) > 0 {
		var anyPayload any
		if err := json.Unmarshal(sig.Payload, &anyPayload); err == nil {
			if normalized, nErr := json.Marshal(anyPayload); nErr == nil {
				payload = string(normalized)
			}
		}
	}
	sum := sha1.Sum([]byte(string(sig.Type) + "|" + sig.TaskID + "|" + payload + "|" + sig.SentAt.UTC().Format(time.RFC3339Nano)))
	return hex.EncodeToString(sum[:])
}
