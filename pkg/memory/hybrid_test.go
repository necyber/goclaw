package memory

import (
	"context"
	"testing"
)

func TestHybridRetriever_VectorOnly(t *testing.T) {
	vi := NewVectorIndex(3)
	bi := NewBM25Index(1.5, 0.75)
	hr := NewHybridRetriever(vi, bi, 0.7, 0.3)

	entries := map[string]*MemoryEntry{
		"a": {ID: "a", SessionID: "s1", Content: "hello"},
		"b": {ID: "b", SessionID: "s1", Content: "world"},
	}
	vi.AddVector("a", "s1", []float32{1, 0, 0})
	vi.AddVector("b", "s1", []float32{0, 1, 0})

	getEntry := func(id string) *MemoryEntry { return entries[id] }

	results, err := hr.Retrieve(context.Background(), "s1", Query{
		Vector: []float32{1, 0, 0},
		Mode:   ModeVector,
		TopK:   1,
	}, getEntry)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Entry.ID != "a" {
		t.Errorf("expected entry 'a', got %v", results)
	}
}

func TestHybridRetriever_BM25Only(t *testing.T) {
	vi := NewVectorIndex(3)
	bi := NewBM25Index(1.5, 0.75)
	hr := NewHybridRetriever(vi, bi, 0.7, 0.3)

	entries := map[string]*MemoryEntry{
		"a": {ID: "a", SessionID: "s1", Content: "machine learning algorithms"},
		"b": {ID: "b", SessionID: "s1", Content: "cooking recipes pasta"},
	}
	bi.IndexDocument("a", "s1", entries["a"].Content)
	bi.IndexDocument("b", "s1", entries["b"].Content)

	getEntry := func(id string) *MemoryEntry { return entries[id] }

	results, err := hr.Retrieve(context.Background(), "s1", Query{
		Text: "machine learning",
		Mode: ModeBM25,
		TopK: 1,
	}, getEntry)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Entry.ID != "a" {
		t.Errorf("expected entry 'a', got %v", results)
	}
}

func TestHybridRetriever_HybridMode(t *testing.T) {
	vi := NewVectorIndex(3)
	bi := NewBM25Index(1.5, 0.75)
	hr := NewHybridRetriever(vi, bi, 0.7, 0.3)

	entries := map[string]*MemoryEntry{
		"a": {ID: "a", SessionID: "s1", Content: "machine learning"},
		"b": {ID: "b", SessionID: "s1", Content: "deep learning"},
	}
	vi.AddVector("a", "s1", []float32{1, 0, 0})
	vi.AddVector("b", "s1", []float32{0.9, 0.1, 0})
	bi.IndexDocument("a", "s1", entries["a"].Content)
	bi.IndexDocument("b", "s1", entries["b"].Content)

	getEntry := func(id string) *MemoryEntry { return entries[id] }

	results, err := hr.Retrieve(context.Background(), "s1", Query{
		Text:   "machine learning",
		Vector: []float32{1, 0, 0},
		Mode:   ModeHybrid,
		TopK:   2,
	}, getEntry)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected results from hybrid search")
	}
}

func TestHybridRetriever_AutoDetectMode(t *testing.T) {
	vi := NewVectorIndex(2)
	bi := NewBM25Index(1.5, 0.75)
	hr := NewHybridRetriever(vi, bi, 0.7, 0.3)

	entries := map[string]*MemoryEntry{
		"a": {ID: "a", SessionID: "s1", Content: "hello world"},
	}
	bi.IndexDocument("a", "s1", "hello world")

	getEntry := func(id string) *MemoryEntry { return entries[id] }

	// Text only -> should auto-detect BM25
	results, err := hr.Retrieve(context.Background(), "s1", Query{
		Text: "hello",
		TopK: 1,
	}, getEntry)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestHybridRetriever_InvalidQuery(t *testing.T) {
	vi := NewVectorIndex(2)
	bi := NewBM25Index(1.5, 0.75)
	hr := NewHybridRetriever(vi, bi, 0.7, 0.3)

	getEntry := func(id string) *MemoryEntry { return nil }

	_, err := hr.Retrieve(context.Background(), "s1", Query{}, getEntry)
	if err != ErrInvalidQuery {
		t.Errorf("expected ErrInvalidQuery, got %v", err)
	}
}

func TestHybridRetriever_MetadataFilter(t *testing.T) {
	vi := NewVectorIndex(2)
	bi := NewBM25Index(1.5, 0.75)
	hr := NewHybridRetriever(vi, bi, 0.7, 0.3)

	entries := map[string]*MemoryEntry{
		"a": {ID: "a", SessionID: "s1", Content: "hello", Metadata: map[string]string{"type": "chat"}},
		"b": {ID: "b", SessionID: "s1", Content: "hello", Metadata: map[string]string{"type": "task"}},
	}
	bi.IndexDocument("a", "s1", "hello")
	bi.IndexDocument("b", "s1", "hello")

	getEntry := func(id string) *MemoryEntry { return entries[id] }

	results, err := hr.Retrieve(context.Background(), "s1", Query{
		Text:    "hello",
		Mode:    ModeBM25,
		TopK:    10,
		Filters: map[string]string{"type": "chat"},
	}, getEntry)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Entry.ID != "a" {
		t.Errorf("expected only entry 'a' with type=chat, got %v", results)
	}
}
