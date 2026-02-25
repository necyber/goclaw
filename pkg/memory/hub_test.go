package memory

import (
	"context"
	"os"
	"testing"
	"time"

	dgbadger "github.com/dgraph-io/badger/v4"
	"github.com/goclaw/goclaw/config"
)

func setupTestHub(t *testing.T) (*MemoryHub, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "goclaw-memory-test-*")
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

	cfg := &config.MemoryConfig{
		Enabled:          true,
		VectorDimension:  3,
		VectorWeight:     0.7,
		BM25Weight:       0.3,
		L1CacheSize:      100,
		ForgetThreshold:  0.1,
		DecayInterval:    time.Hour,
		DefaultStability: 24.0,
		BM25:             config.BM25Config{K1: 1.5, B: 0.75},
	}

	l1 := NewL1Cache(cfg.L1CacheSize)
	l2 := NewL2Badger(db)
	ts := NewTieredStorage(l1, l2)
	hub := NewMemoryHub(cfg, ts, nil)

	cleanup := func() {
		hub.Stop(context.Background()) //nolint:errcheck
		db.Close()                     //nolint:errcheck
		os.RemoveAll(dir)              //nolint:errcheck
	}

	return hub, cleanup
}

func TestHub_MemorizeAndRetrieve(t *testing.T) {
	hub, cleanup := setupTestHub(t)
	defer cleanup()

	ctx := context.Background()
	hub.Start(ctx)

	// Store entries
	id1, err := hub.Memorize(ctx, "s1", "machine learning algorithms", []float32{1, 0, 0}, map[string]string{"type": "tech"})
	if err != nil {
		t.Fatal(err)
	}
	if id1 == "" {
		t.Fatal("expected non-empty ID")
	}

	_, err = hub.Memorize(ctx, "s1", "cooking pasta recipes", []float32{0, 1, 0}, map[string]string{"type": "food"})
	if err != nil {
		t.Fatal(err)
	}

	// Retrieve by text (BM25)
	results, err := hub.Retrieve(ctx, "s1", Query{Text: "machine learning", TopK: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Entry.ID != id1 {
		t.Errorf("expected entry %s, got %s", id1, results[0].Entry.ID)
	}
}

func TestHub_MemorizeAndRetrieveByVector(t *testing.T) {
	hub, cleanup := setupTestHub(t)
	defer cleanup()

	ctx := context.Background()
	hub.Start(ctx)

	hub.Memorize(ctx, "s1", "entry a", []float32{1, 0, 0}, nil)
	hub.Memorize(ctx, "s1", "entry b", []float32{0, 1, 0}, nil)

	results, err := hub.Retrieve(ctx, "s1", Query{Vector: []float32{1, 0, 0}, Mode: ModeVector, TopK: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Entry.Content != "entry a" {
		t.Errorf("expected 'entry a', got %v", results)
	}
}

func TestHub_Forget(t *testing.T) {
	hub, cleanup := setupTestHub(t)
	defer cleanup()

	ctx := context.Background()
	hub.Start(ctx)

	id, _ := hub.Memorize(ctx, "s1", "to be forgotten", nil, nil)
	err := hub.Forget(ctx, "s1", []string{id})
	if err != nil {
		t.Fatal(err)
	}

	count, _ := hub.Count(ctx, "s1")
	if count != 0 {
		t.Errorf("expected 0 entries, got %d", count)
	}
}

func TestHub_ForgetByThreshold(t *testing.T) {
	hub, cleanup := setupTestHub(t)
	defer cleanup()

	ctx := context.Background()
	hub.Start(ctx)

	// Store an entry and manually weaken it
	id, _ := hub.Memorize(ctx, "s1", "weak memory", nil, nil)

	// Get the entry and set low strength
	entry, _ := hub.storage.Get(ctx, id)
	entry.Strength = 0.05
	if err := hub.storage.Store(ctx, entry); err != nil {
		t.Fatal(err)
	}

	// Also store a strong entry
	hub.Memorize(ctx, "s1", "strong memory", nil, nil)

	deleted, err := hub.ForgetByThreshold(ctx, "s1", 0.1)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}

	count, _ := hub.Count(ctx, "s1")
	if count != 1 {
		t.Errorf("expected 1 remaining, got %d", count)
	}
}

func TestHub_ListAndCount(t *testing.T) {
	hub, cleanup := setupTestHub(t)
	defer cleanup()

	ctx := context.Background()
	hub.Start(ctx)

	hub.Memorize(ctx, "s1", "entry 1", nil, nil)
	hub.Memorize(ctx, "s1", "entry 2", nil, nil)
	hub.Memorize(ctx, "s1", "entry 3", nil, nil)

	count, err := hub.Count(ctx, "s1")
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}

	entries, total, err := hub.List(ctx, "s1", 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries (limit), got %d", len(entries))
	}
}

func TestHub_GetStats(t *testing.T) {
	hub, cleanup := setupTestHub(t)
	defer cleanup()

	ctx := context.Background()
	hub.Start(ctx)

	hub.Memorize(ctx, "s1", "entry 1", nil, nil)
	hub.Memorize(ctx, "s1", "entry 2", nil, nil)

	stats, err := hub.GetStats(ctx, "s1")
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalEntries != 2 {
		t.Errorf("expected 2 entries, got %d", stats.TotalEntries)
	}
	if stats.AverageStrength < 0.9 {
		t.Errorf("expected high average strength for new entries, got %f", stats.AverageStrength)
	}
}

func TestHub_DeleteSession(t *testing.T) {
	hub, cleanup := setupTestHub(t)
	defer cleanup()

	ctx := context.Background()
	hub.Start(ctx)

	hub.Memorize(ctx, "s1", "entry 1", []float32{1, 0, 0}, nil)
	hub.Memorize(ctx, "s1", "entry 2", []float32{0, 1, 0}, nil)
	hub.Memorize(ctx, "s2", "other session", []float32{0, 0, 1}, nil)

	deleted, err := hub.DeleteSession(ctx, "s1")
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 2 {
		t.Errorf("expected 2 deleted, got %d", deleted)
	}

	count, _ := hub.Count(ctx, "s2")
	if count != 1 {
		t.Errorf("expected 1 entry in s2, got %d", count)
	}
}

func TestHub_BatchMemorize(t *testing.T) {
	hub, cleanup := setupTestHub(t)
	defer cleanup()

	ctx := context.Background()
	hub.Start(ctx)

	entries := []BatchEntry{
		{Content: "entry 1", Vector: []float32{1, 0, 0}},
		{Content: "entry 2", Vector: []float32{0, 1, 0}},
		{Content: "entry 3", Vector: []float32{0, 0, 1}},
	}

	ids, err := hub.BatchMemorize(ctx, "s1", entries)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 3 {
		t.Errorf("expected 3 IDs, got %d", len(ids))
	}

	count, _ := hub.Count(ctx, "s1")
	if count != 3 {
		t.Errorf("expected 3 entries, got %d", count)
	}
}

func TestHub_SessionIsolation(t *testing.T) {
	hub, cleanup := setupTestHub(t)
	defer cleanup()

	ctx := context.Background()
	hub.Start(ctx)

	hub.Memorize(ctx, "s1", "session one data", []float32{1, 0, 0}, nil)
	hub.Memorize(ctx, "s2", "session two data", []float32{0, 1, 0}, nil)

	// BM25 search should be isolated
	results, err := hub.Retrieve(ctx, "s1", Query{Text: "session data", TopK: 10})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.Entry.SessionID != "s1" {
			t.Errorf("expected only s1 results, got session %s", r.Entry.SessionID)
		}
	}
}

func TestHub_InvalidSessionID(t *testing.T) {
	hub, cleanup := setupTestHub(t)
	defer cleanup()

	ctx := context.Background()

	_, err := hub.Memorize(ctx, "", "content", nil, nil)
	if err != ErrInvalidSessionID {
		t.Errorf("expected ErrInvalidSessionID, got %v", err)
	}

	_, err = hub.Retrieve(ctx, "", Query{Text: "test"})
	if err != ErrInvalidSessionID {
		t.Errorf("expected ErrInvalidSessionID, got %v", err)
	}
}

func TestHub_InvalidQuery(t *testing.T) {
	hub, cleanup := setupTestHub(t)
	defer cleanup()

	ctx := context.Background()

	_, err := hub.Retrieve(ctx, "s1", Query{})
	if err != ErrInvalidQuery {
		t.Errorf("expected ErrInvalidQuery, got %v", err)
	}
}
