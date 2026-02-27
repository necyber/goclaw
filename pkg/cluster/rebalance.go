package cluster

import (
	"sort"
)

// RebalanceReason describes why ownership transfer is being planned.
type RebalanceReason string

const (
	RebalanceReasonNodeJoin    RebalanceReason = "node_join"
	RebalanceReasonNodeLeave   RebalanceReason = "node_leave"
	RebalanceReasonNodeFailure RebalanceReason = "node_failure"
)

// OwnershipTransfer describes a shard ownership movement between nodes.
type OwnershipTransfer struct {
	ShardKey string
	FromNode string
	ToNode   string
	Reason   RebalanceReason
}

// PlanRebalance computes deterministic ownership transfers from old/new assignments.
func PlanRebalance(previousAssignments, currentAssignments map[string]string, reason RebalanceReason) []OwnershipTransfer {
	transfers := make([]OwnershipTransfer, 0)

	keys := make([]string, 0, len(currentAssignments))
	for shardKey := range currentAssignments {
		keys = append(keys, shardKey)
	}
	sort.Strings(keys)

	for _, shardKey := range keys {
		oldOwner := previousAssignments[shardKey]
		newOwner := currentAssignments[shardKey]
		if oldOwner == newOwner {
			continue
		}
		if newOwner == "" {
			continue
		}
		transfers = append(transfers, OwnershipTransfer{
			ShardKey: shardKey,
			FromNode: oldOwner,
			ToNode:   newOwner,
			Reason:   reason,
		})
	}

	return transfers
}

// PlanRebalanceFromMembership computes transfers using consistent hashing over healthy nodes.
func PlanRebalanceFromMembership(previousNodes, currentNodes []NodeState, shardKeys []string, replicas int, reason RebalanceReason) []OwnershipTransfer {
	previousRing := NewHashRing(replicas)
	currentRing := NewHashRing(replicas)

	_ = previousRing.SetNodes(healthyNodeIDs(previousNodes))
	_ = currentRing.SetNodes(healthyNodeIDs(currentNodes))

	return PlanRebalance(previousRing.Assign(shardKeys), currentRing.Assign(shardKeys), reason)
}

func healthyNodeIDs(nodes []NodeState) []string {
	out := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if node.Health == HealthStateHealthy {
			out = append(out, node.NodeID)
		}
	}
	sort.Strings(out)
	return out
}
