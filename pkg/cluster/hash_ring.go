package cluster

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strconv"
	"sync"
)

// HashRing provides deterministic shard-to-node mapping using consistent hashing.
type HashRing struct {
	mu sync.RWMutex

	replicas int
	keys     []uint64
	owners   map[uint64]string
	nodes    map[string]struct{}
}

// NewHashRing creates a consistent hash ring.
func NewHashRing(replicas int) *HashRing {
	if replicas <= 0 {
		replicas = 64
	}
	return &HashRing{
		replicas: replicas,
		keys:     make([]uint64, 0),
		owners:   make(map[uint64]string),
		nodes:    make(map[string]struct{}),
	}
}

// AddNode adds a node to the ring.
func (r *HashRing) AddNode(nodeID string) error {
	if nodeID == "" {
		return fmt.Errorf("cluster: node id cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.nodes[nodeID]; exists {
		return nil
	}
	r.nodes[nodeID] = struct{}{}
	for i := 0; i < r.replicas; i++ {
		key := hashKey(nodeID + "#" + strconv.Itoa(i))
		r.keys = append(r.keys, key)
		r.owners[key] = nodeID
	}
	sort.Slice(r.keys, func(i, j int) bool { return r.keys[i] < r.keys[j] })
	return nil
}

// RemoveNode removes a node and all of its virtual replicas.
func (r *HashRing) RemoveNode(nodeID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.nodes[nodeID]; !exists {
		return
	}
	delete(r.nodes, nodeID)

	filtered := r.keys[:0]
	for _, key := range r.keys {
		if r.owners[key] == nodeID {
			delete(r.owners, key)
			continue
		}
		filtered = append(filtered, key)
	}
	r.keys = filtered
}

// SetNodes replaces ring membership using deterministic sorted input.
func (r *HashRing) SetNodes(nodeIDs []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.keys = r.keys[:0]
	r.owners = make(map[uint64]string)
	r.nodes = make(map[string]struct{})

	sorted := append([]string(nil), nodeIDs...)
	sort.Strings(sorted)
	for _, nodeID := range sorted {
		if nodeID == "" {
			return fmt.Errorf("cluster: node id cannot be empty")
		}
		r.nodes[nodeID] = struct{}{}
		for i := 0; i < r.replicas; i++ {
			key := hashKey(nodeID + "#" + strconv.Itoa(i))
			r.keys = append(r.keys, key)
			r.owners[key] = nodeID
		}
	}
	sort.Slice(r.keys, func(i, j int) bool { return r.keys[i] < r.keys[j] })
	return nil
}

// Owner resolves a shard key to its owner node.
func (r *HashRing) Owner(shardKey string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.keys) == 0 {
		return "", false
	}

	key := hashKey(shardKey)
	index := sort.Search(len(r.keys), func(i int) bool { return r.keys[i] >= key })
	if index == len(r.keys) {
		index = 0
	}
	node := r.owners[r.keys[index]]
	return node, node != ""
}

// Assign returns deterministic owners for the provided shard keys.
func (r *HashRing) Assign(shardKeys []string) map[string]string {
	assignments := make(map[string]string, len(shardKeys))
	for _, shardKey := range shardKeys {
		owner, ok := r.Owner(shardKey)
		if !ok {
			assignments[shardKey] = ""
			continue
		}
		assignments[shardKey] = owner
	}
	return assignments
}

func hashKey(raw string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(raw))
	return h.Sum64()
}
