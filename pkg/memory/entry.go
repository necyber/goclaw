// Package memory provides a hybrid memory system with vector retrieval,
// BM25 full-text search, and FSRS-6 memory decay for the Goclaw orchestration engine.
package memory

import (
	"time"
)

// MemoryEntry represents a single memory entry stored in the system.
type MemoryEntry struct {
	// ID is the unique identifier for this memory entry.
	ID string `json:"id"`

	// TaskID is the optional task that produced this memory.
	TaskID string `json:"task_id,omitempty"`

	// SessionID isolates memories by session.
	SessionID string `json:"session_id"`

	// Content is the raw text content of the memory.
	Content string `json:"content"`

	// Vector is the embedding vector for semantic retrieval.
	// May be nil if the entry was stored without a vector.
	Vector []float32 `json:"vector,omitempty"`

	// Metadata holds arbitrary key-value pairs for filtering.
	Metadata map[string]string `json:"metadata,omitempty"`

	// Strength is the current memory strength (0.0 to 1.0).
	// Managed by the FSRS-6 decay algorithm.
	Strength float64 `json:"strength"`

	// Stability is the FSRS-6 stability parameter (in hours).
	// Higher stability means slower decay.
	Stability float64 `json:"stability"`

	// LastReview is the timestamp of the last retrieval or boost.
	LastReview time.Time `json:"last_review"`

	// CreatedAt is the creation timestamp.
	CreatedAt time.Time `json:"created_at"`
}

// Query represents a retrieval query against the memory system.
type Query struct {
	// Text is the text query for BM25 search.
	Text string `json:"text,omitempty"`

	// Vector is the query vector for semantic search.
	Vector []float32 `json:"vector,omitempty"`

	// Filters are metadata key-value pairs for filtering results.
	Filters map[string]string `json:"filters,omitempty"`

	// Mode selects the retrieval strategy.
	// Valid values: "hybrid", "vector", "bm25". Default is "hybrid".
	Mode string `json:"mode,omitempty"`

	// TopK limits the number of results returned.
	TopK int `json:"top_k,omitempty"`
}

// RetrievalResult wraps a memory entry with its relevance score.
type RetrievalResult struct {
	// Entry is the matched memory entry.
	Entry *MemoryEntry `json:"entry"`

	// Score is the relevance score (higher is better).
	Score float64 `json:"score"`
}

// MemoryStats holds statistics about memory usage.
type MemoryStats struct {
	// TotalEntries is the total number of memory entries.
	TotalEntries int `json:"total_entries"`

	// AverageStrength is the mean strength across all entries.
	AverageStrength float64 `json:"average_strength"`

	// SessionCount is the number of distinct sessions.
	SessionCount int `json:"session_count,omitempty"`
}
