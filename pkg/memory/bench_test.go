package memory

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	dgbadger "github.com/dgraph-io/badger/v4"
	"github.com/goclaw/goclaw/config"
)

func setupBenchHub(b *testing.B) (*MemoryHub, func()) {
	b.Helper()
	dir, err := os.MkdirTemp("", "goclaw-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	opts := dgbadger.DefaultOptions(dir)
	opts.Logger = nil
	db, err := dgbadger.Open(opts)
	if err != nil {
		os.RemoveAll(dir) //nolint:errcheck
		b.Fatal(err)
	}
	cfg := &config.MemoryConfig{
		Enabled: true, VectorDimension: 128, VectorWeight: 0.7, BM25Weight: 0.3,
		L1CacheSize: 5000, ForgetThreshold: 0.1, DecayInterval: 1<<63 - 1, DefaultStability: 24.0,
		BM25: config.BM25Config{K1: 1.5, B: 0.75},
	}
	l1 := NewL1Cache(cfg.L1CacheSize)
	l2 := NewL2Badger(db)
	ts := NewTieredStorage(l1, l2)
	hub := NewMemoryHub(cfg, ts, nil)
	if err := hub.Start(context.Background()); err != nil {
		b.Fatal(err)
	}
	return hub, func() {
		hub.Stop(context.Background()) //nolint:errcheck
		db.Close()                     //nolint:errcheck
		os.RemoveAll(dir)              //nolint:errcheck
	}
}

func makeVec(dim int, seed float32) []float32 {
	v := make([]float32, dim)
	for i := range v {
		v[i] = seed + float32(i)*0.001
	}
	return v
}

// --- 13.7 并发安全测试 ---

func TestHub_ConcurrentMemorize(t *testing.T) {
	dir, err := os.MkdirTemp("", "goclaw-conc-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir) //nolint:errcheck
	opts := dgbadger.DefaultOptions(dir)
	opts.Logger = nil
	db, err := dgbadger.Open(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	cfg := &config.MemoryConfig{
		Enabled: true, VectorDimension: 3, VectorWeight: 0.7, BM25Weight: 0.3,
		L1CacheSize: 100, ForgetThreshold: 0.1, DecayInterval: 1<<63 - 1, DefaultStability: 24.0,
		BM25: config.BM25Config{K1: 1.5, B: 0.75},
	}
	hub := NewMemoryHub(cfg, NewTieredStorage(NewL1Cache(100), NewL2Badger(db)), nil)
	if err := hub.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer hub.Stop(context.Background()) //nolint:errcheck

	ctx := context.Background()
	var wg sync.WaitGroup
	errs := make(chan error, 50)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, err := hub.Memorize(ctx, "s1", fmt.Sprintf("content %d", n), []float32{float32(n), 0, 0}, nil)
			if err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent memorize error: %v", err)
	}

	count, _ := hub.Count(ctx, "s1")
	if count != 50 {
		t.Errorf("expected 50 entries, got %d", count)
	}
}

func TestHub_ConcurrentRetrieve(t *testing.T) {
	dir, err := os.MkdirTemp("", "goclaw-conc-r-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir) //nolint:errcheck
	opts := dgbadger.DefaultOptions(dir)
	opts.Logger = nil
	db, err := dgbadger.Open(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	cfg := &config.MemoryConfig{
		Enabled: true, VectorDimension: 3, VectorWeight: 0.7, BM25Weight: 0.3,
		L1CacheSize: 100, ForgetThreshold: 0.1, DecayInterval: 1<<63 - 1, DefaultStability: 24.0,
		BM25: config.BM25Config{K1: 1.5, B: 0.75},
	}
	hub := NewMemoryHub(cfg, NewTieredStorage(NewL1Cache(100), NewL2Badger(db)), nil)
	if err := hub.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer hub.Stop(context.Background()) //nolint:errcheck

	ctx := context.Background()
	for i := 0; i < 20; i++ {
		if _, err := hub.Memorize(ctx, "s1", fmt.Sprintf("document about topic %d", i), []float32{float32(i), 0, 0}, nil); err != nil {
			t.Fatal(err)
		}
	}

	var wg sync.WaitGroup
	errs := make(chan error, 30)
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := hub.Retrieve(ctx, "s1", Query{Text: "document topic", TopK: 5})
			if err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent retrieve error: %v", err)
	}
}

// --- 13.10 性能基准测试 ---

func BenchmarkVectorSearch_1K(b *testing.B) {
	idx := NewVectorIndex(128)
	for i := 0; i < 1000; i++ {
		_ = idx.AddVector(fmt.Sprintf("e%d", i), "s1", makeVec(128, float32(i)))
	}
	query := makeVec(128, 500)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = idx.Search(query, 10, "")
	}
}

func BenchmarkVectorSearch_10K(b *testing.B) {
	idx := NewVectorIndex(128)
	for i := 0; i < 10000; i++ {
		_ = idx.AddVector(fmt.Sprintf("e%d", i), "s1", makeVec(128, float32(i)))
	}
	query := makeVec(128, 5000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = idx.Search(query, 10, "")
	}
}

func BenchmarkBM25Search_1K(b *testing.B) {
	idx := NewBM25Index(1.5, 0.75)
	for i := 0; i < 1000; i++ {
		idx.IndexDocument(fmt.Sprintf("e%d", i), "s1", fmt.Sprintf("document about topic %d with various keywords and content", i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx.Search("topic keywords content", 10, "")
	}
}

func BenchmarkBM25Search_10K(b *testing.B) {
	idx := NewBM25Index(1.5, 0.75)
	for i := 0; i < 10000; i++ {
		idx.IndexDocument(fmt.Sprintf("e%d", i), "s1", fmt.Sprintf("document about topic %d with various keywords and content", i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx.Search("topic keywords content", 10, "")
	}
}

func BenchmarkHubMemorize(b *testing.B) {
	hub, cleanup := setupBenchHub(b)
	defer cleanup()
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = hub.Memorize(ctx, "s1", fmt.Sprintf("content %d", i), makeVec(128, float32(i)), nil)
	}
}

func BenchmarkHubRetrieve(b *testing.B) {
	hub, cleanup := setupBenchHub(b)
	defer cleanup()
	ctx := context.Background()
	for i := 0; i < 1000; i++ {
		_, _ = hub.Memorize(ctx, "s1", fmt.Sprintf("document about topic %d", i), makeVec(128, float32(i)), nil)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = hub.Retrieve(ctx, "s1", Query{Text: "document topic", TopK: 10})
	}
}

// --- 13.11 内存占用测试 ---

func TestMemoryFootprint_10K(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory footprint test in short mode")
	}
	idx := NewVectorIndex(128)
	bm := NewBM25Index(1.5, 0.75)
	for i := 0; i < 10000; i++ {
		id := fmt.Sprintf("e%d", i)
		_ = idx.AddVector(id, "s1", makeVec(128, float32(i)))
		bm.IndexDocument(id, "s1", fmt.Sprintf("document about topic %d with content", i))
	}
	if idx.Len() != 10000 {
		t.Errorf("expected 10000 vectors, got %d", idx.Len())
	}
	if bm.Len() != 10000 {
		t.Errorf("expected 10000 docs, got %d", bm.Len())
	}
}

// --- 3.11 向量检索性能基准测试 ---

func BenchmarkVectorSearch_100(b *testing.B) {
	idx := NewVectorIndex(128)
	for i := 0; i < 100; i++ {
		_ = idx.AddVector(fmt.Sprintf("e%d", i), "s1", makeVec(128, float32(i)))
	}
	query := makeVec(128, 50)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = idx.Search(query, 10, "s1")
	}
}

func BenchmarkVectorAdd(b *testing.B) {
	idx := NewVectorIndex(128)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = idx.AddVector(fmt.Sprintf("e%d", i), "s1", makeVec(128, float32(i)))
	}
}

func BenchmarkVectorDelete(b *testing.B) {
	idx := NewVectorIndex(128)
	for i := 0; i < b.N; i++ {
		_ = idx.AddVector(fmt.Sprintf("e%d", i), "s1", makeVec(128, float32(i)))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx.DeleteVector(fmt.Sprintf("e%d", i))
	}
}

func BenchmarkVectorSaveLoad(b *testing.B) {
	idx := NewVectorIndex(128)
	for i := 0; i < 1000; i++ {
		_ = idx.AddVector(fmt.Sprintf("e%d", i), "s1", makeVec(128, float32(i)))
	}
	path := b.TempDir() + "/vectors.bin"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = idx.Save(path)
		idx2 := NewVectorIndex(128)
		_ = idx2.Load(path)
	}
}

// --- 4.12 BM25 性能基准测试 ---

func BenchmarkBM25IndexDocument(b *testing.B) {
	idx := NewBM25Index(1.5, 0.75)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx.IndexDocument(fmt.Sprintf("e%d", i), "s1", fmt.Sprintf("document about topic %d with various keywords and content for benchmarking", i))
	}
}

func BenchmarkBM25RemoveDocument(b *testing.B) {
	idx := NewBM25Index(1.5, 0.75)
	for i := 0; i < b.N; i++ {
		idx.IndexDocument(fmt.Sprintf("e%d", i), "s1", fmt.Sprintf("document about topic %d with content", i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx.RemoveDocument(fmt.Sprintf("e%d", i))
	}
}

func BenchmarkBM25Search_100(b *testing.B) {
	idx := NewBM25Index(1.5, 0.75)
	for i := 0; i < 100; i++ {
		idx.IndexDocument(fmt.Sprintf("e%d", i), "s1", fmt.Sprintf("document about topic %d with various keywords and content", i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx.Search("topic keywords content", 10, "s1")
	}
}

// --- 6.12 衰减性能测试 ---

func BenchmarkDecayEntries_100(b *testing.B) {
	dm := NewDecayManager(0.1, 24.0, 1<<63-1)
	entries := make([]*MemoryEntry, 100)
	for i := range entries {
		entries[i] = &MemoryEntry{
			ID:         fmt.Sprintf("e%d", i),
			SessionID:  "s1",
			Strength:   1.0,
			Stability:  24.0,
			LastReview: time.Now().Add(-time.Duration(i) * time.Hour),
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, e := range entries {
			e.Strength = 1.0
		}
		dm.DecayEntries(entries)
	}
}

func BenchmarkDecayEntries_10K(b *testing.B) {
	dm := NewDecayManager(0.1, 24.0, 1<<63-1)
	entries := make([]*MemoryEntry, 10000)
	for i := range entries {
		entries[i] = &MemoryEntry{
			ID:         fmt.Sprintf("e%d", i),
			SessionID:  "s1",
			Strength:   1.0,
			Stability:  24.0,
			LastReview: time.Now().Add(-time.Duration(i%48) * time.Hour),
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, e := range entries {
			e.Strength = 1.0
		}
		dm.DecayEntries(entries)
	}
}

func BenchmarkBoostStrength(b *testing.B) {
	dm := NewDecayManager(0.1, 24.0, 1<<63-1)
	entry := &MemoryEntry{
		ID: "e1", SessionID: "s1", Strength: 0.5, Stability: 24.0,
		LastReview: time.Now().Add(-12 * time.Hour),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dm.BoostStrength(entry)
	}
}
