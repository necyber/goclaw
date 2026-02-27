package signal

import (
	"context"
	"fmt"
	"sync"
)

// RemoteBusRegistry routes remote signals to node-local bus instances.
// It is useful for in-process integration tests and local multi-node simulation.
type RemoteBusRegistry struct {
	mu    sync.RWMutex
	buses map[string]Bus
}

// NewRemoteBusRegistry creates an empty remote registry.
func NewRemoteBusRegistry() *RemoteBusRegistry {
	return &RemoteBusRegistry{
		buses: make(map[string]Bus),
	}
}

// RegisterNodeBus registers node -> local signal bus mapping.
func (r *RemoteBusRegistry) RegisterNodeBus(nodeID string, bus Bus) error {
	if nodeID == "" {
		return fmt.Errorf("signal: node id cannot be empty")
	}
	if bus == nil {
		return fmt.Errorf("signal: bus cannot be nil")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buses[nodeID] = bus
	return nil
}

// PublishRemote implements RemotePublisher.
func (r *RemoteBusRegistry) PublishRemote(ctx context.Context, nodeID string, sig *Signal) error {
	r.mu.RLock()
	bus, ok := r.buses[nodeID]
	r.mu.RUnlock()
	if !ok {
		return fmt.Errorf("signal: remote node bus %q not found", nodeID)
	}
	return bus.Publish(ctx, sig)
}
