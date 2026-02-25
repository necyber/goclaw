package memory

import (
	"container/list"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/dgraph-io/badger/v4"
)

// MemoryStorage is the interface for memory persistence backends.
type MemoryStorage interface {
	Store(ctx context.Context, entry *MemoryEntry) error
	Get(ctx context.Context, id string) (*MemoryEntry, error)
	Delete(ctx context.Context, id string) error
	ListBySession(ctx context.Context, sessionID string, limit, offset int) ([]*MemoryEntry, int, error)
	CountBySession(ctx context.Context, sessionID string) (int, error)
	DeleteBySession(ctx context.Context, sessionID string) (int, error)
	AllBySession(ctx context.Context, sessionID string) ([]*MemoryEntry, error)
	Close() error
}

// --- L1 LRU Cache ---

// L1Cache is an in-memory LRU cache for hot memory entries.
type L1Cache struct {
	mu       sync.RWMutex
	maxSize  int
	items    map[string]*list.Element
	eviction *list.List
	hits     int64
	misses   int64
}

type l1Item struct {
	key   string
	entry *MemoryEntry
}

// NewL1Cache creates a new L1 LRU cache with the given max size.
func NewL1Cache(maxSize int) *L1Cache {
	return &L1Cache{
		maxSize:  maxSize,
		items:    make(map[string]*list.Element),
		eviction: list.New(),
	}
}

// Get retrieves an entry from the cache, promoting it to the front.
func (c *L1Cache) Get(key string) (*MemoryEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.eviction.MoveToFront(elem)
		c.hits++
		return elem.Value.(*l1Item).entry, true
	}
	c.misses++
	return nil, false
}

// Put adds or updates an entry in the cache.
func (c *L1Cache) Put(key string, entry *MemoryEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.eviction.MoveToFront(elem)
		elem.Value.(*l1Item).entry = entry
		return
	}

	if c.eviction.Len() >= c.maxSize {
		c.evictOldest()
	}

	elem := c.eviction.PushFront(&l1Item{key: key, entry: entry})
	c.items[key] = elem
}

// Delete removes an entry from the cache.
func (c *L1Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.eviction.Remove(elem)
		delete(c.items, key)
	}
}

// Len returns the number of items in the cache.
func (c *L1Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// HitRate returns the cache hit rate (0.0-1.0) and total accesses.
func (c *L1Cache) HitRate() (rate float64, total int64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	total = c.hits + c.misses
	if total == 0 {
		return 0, 0
	}
	return float64(c.hits) / float64(total), total
}

func (c *L1Cache) evictOldest() {
	back := c.eviction.Back()
	if back == nil {
		return
	}
	c.eviction.Remove(back)
	delete(c.items, back.Value.(*l1Item).key)
}

// --- L2 Badger Storage ---

const memoryKeyPrefix = "memory:"

// L2Badger is a Badger-backed persistent storage for memory entries.
type L2Badger struct {
	db *badger.DB
}

// NewL2Badger creates a new L2 Badger storage.
func NewL2Badger(db *badger.DB) *L2Badger {
	return &L2Badger{db: db}
}

func sessionKey(sessionID, entryID string) []byte {
	return []byte(fmt.Sprintf("%s%s:%s", memoryKeyPrefix, sessionID, entryID))
}

func sessionPrefix(sessionID string) []byte {
	return []byte(fmt.Sprintf("%s%s:", memoryKeyPrefix, sessionID))
}

// Store persists a memory entry to Badger.
func (s *L2Badger) Store(ctx context.Context, entry *MemoryEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("memory: marshal entry: %w", err)
	}
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(sessionKey(entry.SessionID, entry.ID), data)
	})
}

// Get retrieves a memory entry by ID. The caller must know the sessionID.
func (s *L2Badger) Get(ctx context.Context, id string) (*MemoryEntry, error) {
	var entry MemoryEntry
	err := s.db.View(func(txn *badger.Txn) error {
		// We need to scan all sessions to find by ID since we don't know the session.
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(memoryKeyPrefix)
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := string(item.Key())
			// Key format: memory:{sessionID}:{entryID}
			parts := strings.SplitN(key, ":", 3)
			if len(parts) == 3 && parts[2] == id {
				return item.Value(func(val []byte) error {
					return json.Unmarshal(val, &entry)
				})
			}
		}
		return ErrNotFound
	})
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// Delete removes a memory entry from Badger.
func (s *L2Badger) Delete(ctx context.Context, id string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		// Scan to find the full key
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(memoryKeyPrefix)
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			key := string(it.Item().Key())
			parts := strings.SplitN(key, ":", 3)
			if len(parts) == 3 && parts[2] == id {
				return txn.Delete(it.Item().KeyCopy(nil))
			}
		}
		return nil // Not found is not an error for delete
	})
}

// ListBySession returns paginated entries for a session.
func (s *L2Badger) ListBySession(ctx context.Context, sessionID string, limit, offset int) ([]*MemoryEntry, int, error) {
	all, err := s.AllBySession(ctx, sessionID)
	if err != nil {
		return nil, 0, err
	}
	total := len(all)
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

// CountBySession returns the number of entries for a session.
func (s *L2Badger) CountBySession(ctx context.Context, sessionID string) (int, error) {
	count := 0
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = sessionPrefix(sessionID)
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			count++
		}
		return nil
	})
	return count, err
}

// DeleteBySession removes all entries for a session and returns the count.
func (s *L2Badger) DeleteBySession(ctx context.Context, sessionID string) (int, error) {
	count := 0
	err := s.db.Update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = sessionPrefix(sessionID)
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		var keys [][]byte
		for it.Rewind(); it.Valid(); it.Next() {
			keys = append(keys, it.Item().KeyCopy(nil))
		}
		for _, key := range keys {
			if err := txn.Delete(key); err != nil {
				return err
			}
			count++
		}
		return nil
	})
	return count, err
}

// AllBySession returns all entries for a session.
func (s *L2Badger) AllBySession(ctx context.Context, sessionID string) ([]*MemoryEntry, error) {
	var entries []*MemoryEntry
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = sessionPrefix(sessionID)
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			var entry MemoryEntry
			if err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &entry)
			}); err != nil {
				return err
			}
			entries = append(entries, &entry)
		}
		return nil
	})
	return entries, err
}

// Close is a no-op since the Badger DB lifecycle is managed externally.
func (s *L2Badger) Close() error {
	return nil
}

// --- Tiered Storage Coordinator ---

// TieredStorage coordinates L1 cache and L2 Badger storage.
type TieredStorage struct {
	l1 *L1Cache
	l2 *L2Badger
}

// NewTieredStorage creates a new tiered storage coordinator.
func NewTieredStorage(l1 *L1Cache, l2 *L2Badger) *TieredStorage {
	return &TieredStorage{l1: l1, l2: l2}
}

// Store writes to both L2 (persistent) and L1 (cache).
func (t *TieredStorage) Store(ctx context.Context, entry *MemoryEntry) error {
	if err := t.l2.Store(ctx, entry); err != nil {
		return err
	}
	t.l1.Put(entry.ID, entry)
	return nil
}

// Get retrieves from L1 first, then L2 with promotion.
func (t *TieredStorage) Get(ctx context.Context, id string) (*MemoryEntry, error) {
	// L1 check
	if entry, ok := t.l1.Get(id); ok {
		return entry, nil
	}
	// L2 check with promotion
	entry, err := t.l2.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	t.l1.Put(entry.ID, entry)
	return entry, nil
}

// Delete removes from both L1 and L2.
func (t *TieredStorage) Delete(ctx context.Context, id string) error {
	t.l1.Delete(id)
	return t.l2.Delete(ctx, id)
}

// ListBySession delegates to L2 (L1 is a subset).
func (t *TieredStorage) ListBySession(ctx context.Context, sessionID string, limit, offset int) ([]*MemoryEntry, int, error) {
	return t.l2.ListBySession(ctx, sessionID, limit, offset)
}

// CountBySession delegates to L2.
func (t *TieredStorage) CountBySession(ctx context.Context, sessionID string) (int, error) {
	return t.l2.CountBySession(ctx, sessionID)
}

// DeleteBySession removes all entries for a session from both tiers.
func (t *TieredStorage) DeleteBySession(ctx context.Context, sessionID string) (int, error) {
	// Get all entries to clear L1
	entries, err := t.l2.AllBySession(ctx, sessionID)
	if err != nil {
		return 0, err
	}
	for _, e := range entries {
		t.l1.Delete(e.ID)
	}
	return t.l2.DeleteBySession(ctx, sessionID)
}

// AllBySession delegates to L2.
func (t *TieredStorage) AllBySession(ctx context.Context, sessionID string) ([]*MemoryEntry, error) {
	return t.l2.AllBySession(ctx, sessionID)
}

// Close delegates to L2.
func (t *TieredStorage) Close() error {
	return t.l2.Close()
}
