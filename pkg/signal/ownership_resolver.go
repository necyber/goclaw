package signal

import (
	"context"
	"fmt"
	"sync"
)

// StaticOwnershipResolver is a simple task->owner mapping resolver.
type StaticOwnershipResolver struct {
	mu         sync.RWMutex
	localNode  string
	taskOwners map[string]string
}

// NewStaticOwnershipResolver creates a static resolver.
func NewStaticOwnershipResolver(localNodeID string) *StaticOwnershipResolver {
	return &StaticOwnershipResolver{
		localNode:  localNodeID,
		taskOwners: make(map[string]string),
	}
}

// SetOwner sets owner mapping for a task.
func (r *StaticOwnershipResolver) SetOwner(taskID, nodeID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.taskOwners[taskID] = nodeID
}

// ResolveTaskOwner resolves task owner and indicates whether owner is local.
func (r *StaticOwnershipResolver) ResolveTaskOwner(ctx context.Context, taskID string) (string, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	owner, ok := r.taskOwners[taskID]
	if !ok {
		return "", false, fmt.Errorf("signal: no owner for task %q", taskID)
	}
	return owner, owner == r.localNode, nil
}
