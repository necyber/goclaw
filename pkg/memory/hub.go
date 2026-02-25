package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/goclaw/goclaw/config"
	"github.com/google/uuid"
)

// MemoryHub is the concrete implementation of the Hub interface.
type MemoryHub struct {
	mu sync.RWMutex

	cfg     *config.MemoryConfig
	storage *TieredStorage
	vector  *VectorIndex
	bm25    *BM25Index
	hybrid  *HybridRetriever
	decay   *DecayManager
	logger  hubLogger
	started bool
}

// hubLogger is the minimal logger interface used by MemoryHub.
type hubLogger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// nopHubLogger is a no-op logger.
type nopHubLogger struct{}

func (n *nopHubLogger) Debug(msg string, args ...any) {}
func (n *nopHubLogger) Info(msg string, args ...any)  {}
func (n *nopHubLogger) Warn(msg string, args ...any)  {}
func (n *nopHubLogger) Error(msg string, args ...any) {}

// NewMemoryHub creates a new MemoryHub from configuration and storage.
func NewMemoryHub(cfg *config.MemoryConfig, storage *TieredStorage, logger hubLogger) *MemoryHub {
	if logger == nil {
		logger = &nopHubLogger{}
	}

	vectorIdx := NewVectorIndex(cfg.VectorDimension)
	bm25Idx := NewBM25Index(cfg.BM25.K1, cfg.BM25.B)
	hybridRetriever := NewHybridRetriever(vectorIdx, bm25Idx, cfg.VectorWeight, cfg.BM25Weight)
	decayMgr := NewDecayManager(cfg.ForgetThreshold, cfg.DefaultStability, cfg.DecayInterval)

	return &MemoryHub{
		cfg:     cfg,
		storage: storage,
		vector:  vectorIdx,
		bm25:    bm25Idx,
		hybrid:  hybridRetriever,
		decay:   decayMgr,
		logger:  logger,
	}
}

// Start initializes the memory system and starts the decay loop.
func (h *MemoryHub) Start(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.started {
		return fmt.Errorf("memory hub already started")
	}

	h.logger.Info("starting memory hub",
		"vector_dimension", h.cfg.VectorDimension,
		"l1_cache_size", h.cfg.L1CacheSize,
		"decay_interval", h.cfg.DecayInterval,
	)

	// Start the decay loop
	h.decay.StartDecayLoop(ctx, h.processDecay)
	h.started = true

	h.logger.Info("memory hub started")
	return nil
}

// Stop gracefully shuts down the memory system.
func (h *MemoryHub) Stop(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.started {
		return nil
	}

	h.logger.Info("stopping memory hub")
	h.decay.Stop()
	h.started = false
	h.logger.Info("memory hub stopped")
	return nil
}

// Memorize stores a new memory entry.
func (h *MemoryHub) Memorize(ctx context.Context, sessionID string, content string, vector []float32, metadata map[string]string) (string, error) {
	if sessionID == "" {
		return "", ErrInvalidSessionID
	}

	entryID := uuid.New().String()
	now := time.Now()

	entry := &MemoryEntry{
		ID:        entryID,
		SessionID: sessionID,
		Content:   content,
		Vector:    vector,
		Metadata:  metadata,
		CreatedAt: now,
	}

	// Initialize decay parameters
	h.decay.InitEntry(entry)

	// Store in tiered storage
	if err := h.storage.Store(ctx, entry); err != nil {
		return "", fmt.Errorf("memory: store failed: %w", err)
	}

	// Index for vector search
	if len(vector) > 0 {
		if err := h.vector.AddVector(entryID, sessionID, vector); err != nil {
			h.logger.Warn("failed to index vector", "entry_id", entryID, "error", err)
		}
	}

	// Index for BM25 search
	if content != "" {
		h.bm25.IndexDocument(entryID, sessionID, content)
	}

	return entryID, nil
}

// BatchMemorize stores multiple entries in one call.
func (h *MemoryHub) BatchMemorize(ctx context.Context, sessionID string, entries []BatchEntry) ([]string, error) {
	if sessionID == "" {
		return nil, ErrInvalidSessionID
	}

	ids := make([]string, 0, len(entries))
	for _, be := range entries {
		id, err := h.Memorize(ctx, sessionID, be.Content, be.Vector, be.Metadata)
		if err != nil {
			return ids, fmt.Errorf("memory: batch memorize failed at entry %d: %w", len(ids), err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// Retrieve searches for memory entries matching the query.
func (h *MemoryHub) Retrieve(ctx context.Context, sessionID string, query Query) ([]*RetrievalResult, error) {
	if sessionID == "" {
		return nil, ErrInvalidSessionID
	}
	if query.Text == "" && len(query.Vector) == 0 {
		return nil, ErrInvalidQuery
	}

	getEntry := func(id string) *MemoryEntry {
		entry, err := h.storage.Get(ctx, id)
		if err != nil {
			return nil
		}
		return entry
	}

	results, err := h.hybrid.Retrieve(ctx, sessionID, query, getEntry)
	if err != nil {
		return nil, err
	}

	// Boost strength for retrieved entries
	for _, r := range results {
		h.decay.BoostStrength(r.Entry)
		// Update storage with boosted strength
		if err := h.storage.Store(ctx, r.Entry); err != nil {
			h.logger.Warn("failed to update entry strength", "entry_id", r.Entry.ID, "error", err)
		}
	}

	return results, nil
}

// Forget deletes specific memory entries by ID.
func (h *MemoryHub) Forget(ctx context.Context, sessionID string, ids []string) error {
	if sessionID == "" {
		return ErrInvalidSessionID
	}

	for _, id := range ids {
		h.vector.DeleteVector(id)
		h.bm25.RemoveDocument(id)
		if err := h.storage.Delete(ctx, id); err != nil {
			h.logger.Warn("failed to delete entry", "entry_id", id, "error", err)
		}
	}
	return nil
}

// ForgetByThreshold deletes entries with strength below the threshold.
func (h *MemoryHub) ForgetByThreshold(ctx context.Context, sessionID string, threshold float64) (int, error) {
	if sessionID == "" {
		return 0, ErrInvalidSessionID
	}

	entries, err := h.storage.AllBySession(ctx, sessionID)
	if err != nil {
		return 0, fmt.Errorf("memory: list entries failed: %w", err)
	}

	count := 0
	for _, entry := range entries {
		// Recalculate strength before checking
		h.decay.UpdateStrength(entry)
		if entry.Strength < threshold {
			h.vector.DeleteVector(entry.ID)
			h.bm25.RemoveDocument(entry.ID)
			if err := h.storage.Delete(ctx, entry.ID); err != nil {
				h.logger.Warn("failed to delete weak entry", "entry_id", entry.ID, "error", err)
				continue
			}
			count++
		}
	}
	return count, nil
}

// List returns paginated memory entries for a session.
func (h *MemoryHub) List(ctx context.Context, sessionID string, limit, offset int) ([]*MemoryEntry, int, error) {
	if sessionID == "" {
		return nil, 0, ErrInvalidSessionID
	}
	if limit <= 0 {
		limit = 20
	}
	return h.storage.ListBySession(ctx, sessionID, limit, offset)
}

// Count returns the number of memory entries for a session.
func (h *MemoryHub) Count(ctx context.Context, sessionID string) (int, error) {
	if sessionID == "" {
		return 0, ErrInvalidSessionID
	}
	return h.storage.CountBySession(ctx, sessionID)
}

// GetStats returns memory statistics for a session.
func (h *MemoryHub) GetStats(ctx context.Context, sessionID string) (*MemoryStats, error) {
	if sessionID == "" {
		return nil, ErrInvalidSessionID
	}

	entries, err := h.storage.AllBySession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("memory: get stats failed: %w", err)
	}

	stats := &MemoryStats{
		TotalEntries: len(entries),
	}

	if len(entries) > 0 {
		totalStrength := 0.0
		for _, e := range entries {
			totalStrength += e.Strength
		}
		stats.AverageStrength = totalStrength / float64(len(entries))
	}

	return stats, nil
}

// DeleteSession removes all memory entries for a session.
func (h *MemoryHub) DeleteSession(ctx context.Context, sessionID string) (int, error) {
	if sessionID == "" {
		return 0, ErrInvalidSessionID
	}

	// Clean up indexes
	h.vector.DeleteBySession(sessionID)
	h.bm25.DeleteBySession(sessionID)

	return h.storage.DeleteBySession(ctx, sessionID)
}

// processDecay is the callback for the decay loop.
func (h *MemoryHub) processDecay(ctx context.Context) error {
	h.logger.Debug("running memory decay cycle")

	// We need to iterate all sessions. Since we don't track sessions explicitly,
	// we scan all entries via Badger prefix scan.
	// For now, we process all entries in the L2 store.
	entries, err := h.storage.l2.AllBySession(ctx, "")
	if err != nil {
		// If empty session returns nothing, that's fine
		return nil
	}

	// Group by session for isolation
	sessionEntries := make(map[string][]*MemoryEntry)
	for _, e := range entries {
		sessionEntries[e.SessionID] = append(sessionEntries[e.SessionID], e)
	}

	for sessionID, entries := range sessionEntries {
		updated, forgotten := h.decay.DecayEntries(entries)

		// Update surviving entries
		for _, entry := range updated {
			if err := h.storage.Store(ctx, entry); err != nil {
				h.logger.Warn("failed to update decayed entry", "entry_id", entry.ID, "error", err)
			}
		}

		// Delete forgotten entries
		if len(forgotten) > 0 {
			if err := h.Forget(ctx, sessionID, forgotten); err != nil {
				h.logger.Warn("failed to forget entries", "session_id", sessionID, "error", err)
			}
			h.logger.Info("memory decay: forgotten entries",
				"session_id", sessionID,
				"count", len(forgotten),
			)
		}
	}

	return nil
}
