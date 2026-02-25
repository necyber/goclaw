package memory

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
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

// Save persists the vector index to a file.
// Format: [dimension:uint32][count:uint32] then for each entry:
// [idLen:uint16][id:bytes][sidLen:uint16][sid:bytes][vector:float32*dim]
func (v *VectorIndex) Save(path string) error {
	v.mu.RLock()
	defer v.mu.RUnlock()

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("vector: save failed: %w", err)
	}
	defer f.Close()

	// Header
	if err := binary.Write(f, binary.LittleEndian, uint32(v.dimension)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(len(v.vectors))); err != nil {
		return err
	}

	for id, vec := range v.vectors {
		sid := v.sessions[id]
		// Write entry ID
		if err := binary.Write(f, binary.LittleEndian, uint16(len(id))); err != nil {
			return err
		}
		if _, err := f.Write([]byte(id)); err != nil {
			return err
		}
		// Write session ID
		if err := binary.Write(f, binary.LittleEndian, uint16(len(sid))); err != nil {
			return err
		}
		if _, err := f.Write([]byte(sid)); err != nil {
			return err
		}
		// Write vector
		if err := binary.Write(f, binary.LittleEndian, vec); err != nil {
			return err
		}
	}
	return nil
}

// Load restores the vector index from a file.
func (v *VectorIndex) Load(path string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("vector: load failed: %w", err)
	}
	defer f.Close()

	var dim, count uint32
	if err := binary.Read(f, binary.LittleEndian, &dim); err != nil {
		return err
	}
	if err := binary.Read(f, binary.LittleEndian, &count); err != nil {
		return err
	}

	if int(dim) != v.dimension {
		return fmt.Errorf("%w: file has %d, index expects %d", ErrDimensionMismatch, dim, v.dimension)
	}

	vectors := make(map[string][]float32, count)
	sessions := make(map[string]string, count)

	for i := uint32(0); i < count; i++ {
		// Read entry ID
		var idLen uint16
		if err := binary.Read(f, binary.LittleEndian, &idLen); err != nil {
			return err
		}
		idBuf := make([]byte, idLen)
		if _, err := io.ReadFull(f, idBuf); err != nil {
			return err
		}
		id := string(idBuf)

		// Read session ID
		var sidLen uint16
		if err := binary.Read(f, binary.LittleEndian, &sidLen); err != nil {
			return err
		}
		sidBuf := make([]byte, sidLen)
		if _, err := io.ReadFull(f, sidBuf); err != nil {
			return err
		}
		sid := string(sidBuf)

		// Read vector
		vec := make([]float32, dim)
		if err := binary.Read(f, binary.LittleEndian, vec); err != nil {
			return err
		}

		vectors[id] = vec
		sessions[id] = sid
	}

	v.vectors = vectors
	v.sessions = sessions
	return nil
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
