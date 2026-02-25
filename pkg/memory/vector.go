package memory

import (
	"fmt"
	"math"
	"sort"
	"sync"
)

// VectorIndex provides approximate nearest neighbor search using a simple
// brute-force approach with cosine similarity. For production workloads with
// 100K+ vectors, this can be replaced with an HNSW implementation.
type VectorIndex struct {
	mu        sync.RWMutex
	dimension int
	vectors   map[string][]float32 // entryID -> vector
	sessions  map[string]string    // entryID -> sessionID
}

// NewVectorIndex creates a new vector index with the given dimension.
func NewVectorIndex(dimension int) *VectorIndex {
	return &VectorIndex{
		dimension: dimension,
		vectors:   make(map[string][]float32),
		sessions:  make(map[string]string),
	}
}

// AddVector adds a vector to the index.
func (v *VectorIndex) AddVector(entryID, sessionID string, vector []float32) error {
	if len(vector) != v.dimension {
		return fmt.Errorf("%w: expected %d, got %d", ErrDimensionMismatch, v.dimension, len(vector))
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.vectors[entryID] = vector
	v.sessions[entryID] = sessionID
	return nil
}

// UpdateVector replaces a vector in the index.
func (v *VectorIndex) UpdateVector(entryID, sessionID string, vector []float32) error {
	if len(vector) != v.dimension {
		return fmt.Errorf("%w: expected %d, got %d", ErrDimensionMismatch, v.dimension, len(vector))
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.vectors[entryID] = vector
	v.sessions[entryID] = sessionID
	return nil
}

// DeleteVector removes a vector from the index.
func (v *VectorIndex) DeleteVector(entryID string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	delete(v.vectors, entryID)
	delete(v.sessions, entryID)
}

// Search finds the top-K most similar vectors to the query.
// If sessionID is non-empty, results are filtered to that session.
func (v *VectorIndex) Search(query []float32, topK int, sessionID string) ([]string, []float64, error) {
	if len(query) != v.dimension {
		return nil, nil, fmt.Errorf("%w: expected %d, got %d", ErrDimensionMismatch, v.dimension, len(query))
	}

	v.mu.RLock()
	defer v.mu.RUnlock()

	type scored struct {
		id    string
		score float64
	}

	var results []scored
	for id, vec := range v.vectors {
		if sessionID != "" && v.sessions[id] != sessionID {
			continue
		}
		sim := cosineSimilarity(query, vec)
		results = append(results, scored{id: id, score: sim})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	if topK > len(results) {
		topK = len(results)
	}
	results = results[:topK]

	ids := make([]string, len(results))
	scores := make([]float64, len(results))
	for i, r := range results {
		ids[i] = r.id
		scores[i] = r.score
	}
	return ids, scores, nil
}

// DeleteBySession removes all vectors for a session.
func (v *VectorIndex) DeleteBySession(sessionID string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	for id, sid := range v.sessions {
		if sid == sessionID {
			delete(v.vectors, id)
			delete(v.sessions, id)
		}
	}
}

// Len returns the number of vectors in the index.
func (v *VectorIndex) Len() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.vectors)
}

// cosineSimilarity calculates the cosine similarity between two vectors.
func cosineSimilarity(a []float32, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return dotProduct / denom
}
