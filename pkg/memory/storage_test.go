package memory

import (
	"context"
	"os"
	"testing"

	dgbadger "github.com/dgraph-io/badger/v4"
)

func setupTestStorage(t *testing.T) (*TieredStorage, *dgbadger.DB, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "goclaw-storage-test-*")
	if err != nil {
		t.Fatal(err)
	}

	opts := dgbadger.DefaultOptions(dir)
	opts.Logger = nil
	db, err := dgbadger.Open(opts)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatal(err)
	}

	l1 := NewL1Cache(10)
	l2 := NewL2Badger(db)
	ts := NewTieredStorage(l1, l2)

	cleanup := func() {
		db.Close()
		os.RemoveAll(dir)
	}

	return ts, db, cleanup
}

func TestL1Cache_PutAndGet(t *testing.T) {
	cache := NewL1Cache(3)

	entry := &MemoryEntry{ID: "a", Content: "hello"}
	cache.Put("a", entry)

	got, ok := cache.Get("a")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got.Content != "hello" {
		t.Errorf("expected 'hello', got %q", got.Content)
	}
}

func TestL1Cache_Eviction(t *testing.T) {
	cache := NewL1Cache(2)

	cache.Put("a", &MemoryEntry{ID: "a"})
	cache.Put("b", &MemoryEntry{ID: "b"})
	cache.Put("c", &MemoryEntry{ID: "c"}) // Should evict "a"

	if _, ok := cache.Get("a"); ok {
		t.Error("expected 'a' to be evicted")
	}
	if _, ok := cache.Get("b"); !ok {
		t.Error("expected 'b' to still be in cache")
	}
	if _, ok := cache.Get("c"); !ok {
		t.Error("expected 'c' to still be in cache")
	}
}

func TestL1Cache_LRUOrder(t *testing.T) {
	cache := NewL1Cache(2)

	cache.Put("a", &MemoryEntry{ID: "a"})
	cache.Put("b", &MemoryEntry{ID: "b"})
	cache.Get("a")                        // Promote "a"
	cache.Put("c", &MemoryEntry{ID: "c"}) // Should evict "b" (least recently used)

	if _, ok := cache.Get("a"); !ok {
		t.Error("expected 'a' to still be in cache (was promoted)")
	}
	if _, ok := cache.Get("b"); ok {
		t.Error("expected 'b' to be evicted")
	}
}

func TestL1Cache_Delete(t *testing.T) {
	cache := NewL1Cache(10)
	cache.Put("a", &MemoryEntry{ID: "a"})
	cache.Delete("a")

	if _, ok := cache.Get("a"); ok {
		t.Error("expected 'a' to be deleted")
	}
	if cache.Len() != 0 {
		t.Errorf("expected 0 items, got %d", cache.Len())
	}
}

func TestTieredStorage_StoreAndGet(t *testing.T) {
	ts, _, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	entry := &MemoryEntry{ID: "e1", SessionID: "s1", Content: "hello"}

	if err := ts.Store(ctx, entry); err != nil {
		t.Fatal(err)
	}

	got, err := ts.Get(ctx, "e1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "hello" {
		t.Errorf("expected 'hello', got %q", got.Content)
	}
}

func TestTieredStorage_L1CachePromotion(t *testing.T) {
	ts, _, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	entry := &MemoryEntry{ID: "e1", SessionID: "s1", Content: "hello"}

	// Store (goes to both L1 and L2)
	if err := ts.Store(ctx, entry); err != nil {
		t.Fatal(err)
	}

	// Clear L1 manually
	ts.l1.Delete("e1")

	// Get should fetch from L2 and promote to L1
	got, err := ts.Get(ctx, "e1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "hello" {
		t.Errorf("expected 'hello', got %q", got.Content)
	}

	// Now it should be in L1
	cached, ok := ts.l1.Get("e1")
	if !ok {
		t.Error("expected entry to be promoted to L1")
	}
	if cached.Content != "hello" {
		t.Errorf("expected 'hello' in L1, got %q", cached.Content)
	}
}

func TestTieredStorage_Delete(t *testing.T) {
	ts, _, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	entry := &MemoryEntry{ID: "e1", SessionID: "s1", Content: "hello"}
	if err := ts.Store(ctx, entry); err != nil {
		t.Fatal(err)
	}

	if err := ts.Delete(ctx, "e1"); err != nil {
		t.Fatal(err)
	}

	_, err := ts.Get(ctx, "e1")
	if err == nil {
		t.Error("expected error after deletion")
	}
}

func TestTieredStorage_ListBySession(t *testing.T) {
	ts, _, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	ts.Store(ctx, &MemoryEntry{ID: "e1", SessionID: "s1", Content: "a"}) //nolint:errcheck
	ts.Store(ctx, &MemoryEntry{ID: "e2", SessionID: "s1", Content: "b"}) //nolint:errcheck
	ts.Store(ctx, &MemoryEntry{ID: "e3", SessionID: "s2", Content: "c"}) //nolint:errcheck

	entries, total, err := ts.ListBySession(ctx, "s1", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

func TestTieredStorage_DeleteBySession(t *testing.T) {
	ts, _, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	ts.Store(ctx, &MemoryEntry{ID: "e1", SessionID: "s1", Content: "a"})
	ts.Store(ctx, &MemoryEntry{ID: "e2", SessionID: "s1", Content: "b"})
	ts.Store(ctx, &MemoryEntry{ID: "e3", SessionID: "s2", Content: "c"})

	count, err := ts.DeleteBySession(ctx, "s1")
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("expected 2 deleted, got %d", count)
	}

	remaining, _ := ts.CountBySession(ctx, "s2")
	if remaining != 1 {
		t.Errorf("expected 1 remaining in s2, got %d", remaining)
	}
}
