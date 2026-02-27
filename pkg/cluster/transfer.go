package cluster

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// QueuedWork represents queued workload bound to a shard.
type QueuedWork struct {
	ID      string
	Payload any
}

// InFlightWork represents currently executing workload bound to a shard.
type InFlightWork struct {
	ID         string
	OwnerToken uint64
	StartedAt  time.Time
	Completed  bool
}

// TransferSnapshot captures transferable shard state.
type TransferSnapshot struct {
	ShardKey      string
	NewOwnerToken uint64
	Queued        []QueuedWork
	InFlightIDs   []string
}

// OwnershipTransferManager handles queued/in-flight state during ownership transfer.
type OwnershipTransferManager struct {
	mu sync.Mutex

	activeToken map[string]uint64
	queued      map[string][]QueuedWork
	inFlight    map[string]map[string]InFlightWork
	completed   map[string]struct{}
}

// NewOwnershipTransferManager creates a transfer manager.
func NewOwnershipTransferManager() *OwnershipTransferManager {
	return &OwnershipTransferManager{
		activeToken: make(map[string]uint64),
		queued:      make(map[string][]QueuedWork),
		inFlight:    make(map[string]map[string]InFlightWork),
		completed:   make(map[string]struct{}),
	}
}

// SetActiveToken updates the active owner fencing token for a shard.
func (m *OwnershipTransferManager) SetActiveToken(shardKey string, token uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeToken[shardKey] = token
}

// QueueWork records queued workload for a shard.
func (m *OwnershipTransferManager) QueueWork(shardKey, workloadID string, payload any) error {
	if shardKey == "" || workloadID == "" {
		return fmt.Errorf("cluster: shard key and workload id are required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.completed[workloadID]; exists {
		return nil
	}
	m.queued[shardKey] = append(m.queued[shardKey], QueuedWork{ID: workloadID, Payload: payload})
	return nil
}

// StartInFlight marks workload as in-flight under the current owner token.
func (m *OwnershipTransferManager) StartInFlight(shardKey, workloadID string, ownerToken uint64) error {
	if shardKey == "" || workloadID == "" {
		return fmt.Errorf("cluster: shard key and workload id are required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if token := m.activeToken[shardKey]; token > 0 && token != ownerToken {
		return ErrFencingTokenInvalid
	}
	if _, done := m.completed[workloadID]; done {
		return nil
	}

	if _, ok := m.inFlight[shardKey]; !ok {
		m.inFlight[shardKey] = make(map[string]InFlightWork)
	}
	m.inFlight[shardKey][workloadID] = InFlightWork{
		ID:         workloadID,
		OwnerToken: ownerToken,
		StartedAt:  time.Now().UTC(),
	}
	return nil
}

// CompleteInFlight marks workload terminal and suppresses duplicate terminal outcomes.
func (m *OwnershipTransferManager) CompleteInFlight(shardKey, workloadID string, ownerToken uint64) (bool, error) {
	if shardKey == "" || workloadID == "" {
		return false, fmt.Errorf("cluster: shard key and workload id are required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, alreadyDone := m.completed[workloadID]; alreadyDone {
		return false, nil
	}
	if token := m.activeToken[shardKey]; token > 0 && token != ownerToken {
		return false, ErrFencingTokenInvalid
	}

	if entries, ok := m.inFlight[shardKey]; ok {
		delete(entries, workloadID)
		if len(entries) == 0 {
			delete(m.inFlight, shardKey)
		}
	}
	m.completed[workloadID] = struct{}{}
	return true, nil
}

// AdoptInFlight rebinds in-flight workload to a new owner token after transfer.
func (m *OwnershipTransferManager) AdoptInFlight(shardKey string, ownerToken uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeToken[shardKey] = ownerToken
	if entries, ok := m.inFlight[shardKey]; ok {
		for workID, work := range entries {
			work.OwnerToken = ownerToken
			entries[workID] = work
		}
	}
}

// TransferShard updates active token and returns queued + in-flight snapshot for handoff.
func (m *OwnershipTransferManager) TransferShard(shardKey string, newOwnerToken uint64) TransferSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.activeToken[shardKey] = newOwnerToken

	queued := append([]QueuedWork(nil), m.queued[shardKey]...)
	delete(m.queued, shardKey)

	inFlightIDs := make([]string, 0)
	if entries, ok := m.inFlight[shardKey]; ok {
		for workID := range entries {
			inFlightIDs = append(inFlightIDs, workID)
		}
		sort.Strings(inFlightIDs)
	}

	return TransferSnapshot{
		ShardKey:      shardKey,
		NewOwnerToken: newOwnerToken,
		Queued:        queued,
		InFlightIDs:   inFlightIDs,
	}
}
