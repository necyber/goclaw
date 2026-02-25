package memory

import (
	"context"
	"errors"
)

// Sentinel errors for the memory system.
var (
	ErrInvalidSessionID   = errors.New("memory: invalid session ID")
	ErrInvalidQuery       = errors.New("memory: invalid query (no text and no vector)")
	ErrInvalidEntryID     = errors.New("memory: invalid entry ID")
	ErrDimensionMismatch  = errors.New("memory: vector dimension mismatch")
	ErrStorageUnavailable = errors.New("memory: storage unavailable")
	ErrNotFound           = errors.New("memory: entry not found")
)

// Hub is the main interface for the hybrid memory system.
type Hub interface {
	// Memorize stores a new memory entry and returns its ID.
	Memorize(ctx context.Context, sessionID string, content string, vector []float32, metadata map[string]string) (string, error)

	// BatchMemorize stores multiple memory entries in one call.
	BatchMemorize(ctx context.Context, sessionID string, entries []BatchEntry) ([]string, error)

	// Retrieve searches for memory entries matching the query.
	Retrieve(ctx context.Context, sessionID string, query Query) ([]*RetrievalResult, error)

	// Forget deletes specific memory entries by ID.
	Forget(ctx context.Context, sessionID string, ids []string) error

	// ForgetByThreshold deletes entries with strength below the threshold.
	// Returns the number of deleted entries.
	ForgetByThreshold(ctx context.Context, sessionID string, threshold float64) (int, error)

	// List returns all memory entries for a session with pagination.
	List(ctx context.Context, sessionID string, limit, offset int) ([]*MemoryEntry, int, error)

	// Count returns the number of memory entries for a session.
	Count(ctx context.Context, sessionID string) (int, error)

	// GetStats returns memory statistics for a session.
	GetStats(ctx context.Context, sessionID string) (*MemoryStats, error)

	// DeleteSession removes all memory entries for a session.
	DeleteSession(ctx context.Context, sessionID string) (int, error)

	// Start initializes the memory system and starts background processes.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the memory system.
	Stop(ctx context.Context) error
}

// BatchEntry is a single entry in a batch memorize call.
type BatchEntry struct {
	Content  string
	Vector   []float32
	Metadata map[string]string
}
