package memory

import (
	"math"
	"testing"
)

func TestVectorIndex_AddAndSearch(t *testing.T) {
	idx := NewVectorIndex(3)

	// Add vectors
	if err := idx.AddVector("a", "s1", []float32{1, 0, 0}); err != nil {
		t.Fatal(err)
	}
	if err := idx.AddVector("b", "s1", []float32{0, 1, 0}); err != nil {
		t.Fatal(err)
	}
	if err := idx.AddVector("c", "s1", []float32{0.9, 0.1, 0}); err != nil {
		t.Fatal(err)
	}

	// Search for vector similar to [1,0,0]
	ids, scores, err := idx.Search([]float32{1, 0, 0}, 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 results, got %d", len(ids))
	}
	if ids[0] != "a" {
		t.Errorf("expected first result 'a', got %q", ids[0])
	}
	if math.Abs(scores[0]-1.0) > 0.001 {
		t.Errorf("expected score ~1.0, got %f", scores[0])
	}
}

func TestVectorIndex_DimensionMismatch(t *testing.T) {
	idx := NewVectorIndex(3)
	err := idx.AddVector("a", "s1", []float32{1, 0})
	if err == nil {
		t.Fatal("expected dimension mismatch error")
	}
}

func TestVectorIndex_SessionFilter(t *testing.T) {
	idx := NewVectorIndex(2)
	idx.AddVector("a", "s1", []float32{1, 0})
	idx.AddVector("b", "s2", []float32{0.9, 0.1})

	ids, _, err := idx.Search([]float32{1, 0}, 10, "s1")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "a" {
		t.Errorf("expected only 'a' from session s1, got %v", ids)
	}
}

func TestVectorIndex_DeleteVector(t *testing.T) {
	idx := NewVectorIndex(2)
	idx.AddVector("a", "s1", []float32{1, 0})
	idx.DeleteVector("a")
	if idx.Len() != 0 {
		t.Errorf("expected 0 vectors, got %d", idx.Len())
	}
}

func TestVectorIndex_DeleteBySession(t *testing.T) {
	idx := NewVectorIndex(2)
	idx.AddVector("a", "s1", []float32{1, 0})
	idx.AddVector("b", "s1", []float32{0, 1})
	idx.AddVector("c", "s2", []float32{1, 1})
	idx.DeleteBySession("s1")
	if idx.Len() != 1 {
		t.Errorf("expected 1 vector, got %d", idx.Len())
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a, b []float32
		want float64
	}{
		{"identical", []float32{1, 0, 0}, []float32{1, 0, 0}, 1.0},
		{"orthogonal", []float32{1, 0, 0}, []float32{0, 1, 0}, 0.0},
		{"opposite", []float32{1, 0}, []float32{-1, 0}, -1.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosineSimilarity(tt.a, tt.b)
			if math.Abs(got-tt.want) > 0.001 {
				t.Errorf("cosineSimilarity = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestVectorIndex_SaveLoad(t *testing.T) {
	idx := NewVectorIndex(3)
	idx.AddVector("a", "s1", []float32{1, 0, 0})
	idx.AddVector("b", "s2", []float32{0, 1, 0})
	idx.AddVector("c", "s1", []float32{0.5, 0.5, 0})

	tmpFile := t.TempDir() + "/vectors.bin"

	// Save
	if err := idx.Save(tmpFile); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load into new index
	idx2 := NewVectorIndex(3)
	if err := idx2.Load(tmpFile); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if idx2.Len() != 3 {
		t.Errorf("expected 3 vectors after load, got %d", idx2.Len())
	}

	// Verify search still works
	ids, scores, err := idx2.Search([]float32{1, 0, 0}, 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "a" {
		t.Errorf("expected 'a' as top result, got %v", ids)
	}
	if math.Abs(scores[0]-1.0) > 0.001 {
		t.Errorf("expected score ~1.0, got %f", scores[0])
	}

	// Verify session filter works after load
	ids, _, err = idx2.Search([]float32{1, 0, 0}, 10, "s2")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "b" {
		t.Errorf("expected only 'b' from session s2, got %v", ids)
	}
}

func TestVectorIndex_LoadDimensionMismatch(t *testing.T) {
	idx := NewVectorIndex(3)
	idx.AddVector("a", "s1", []float32{1, 0, 0})

	tmpFile := t.TempDir() + "/vectors.bin"
	idx.Save(tmpFile)

	idx2 := NewVectorIndex(5) // different dimension
	err := idx2.Load(tmpFile)
	if err == nil {
		t.Fatal("expected dimension mismatch error on load")
	}
}
