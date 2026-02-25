package memory

import (
	"testing"
)

func TestBM25Index_IndexAndSearch(t *testing.T) {
	idx := NewBM25Index(1.5, 0.75)

	idx.IndexDocument("d1", "s1", "the quick brown fox jumps over the lazy dog")
	idx.IndexDocument("d2", "s1", "machine learning and artificial intelligence")
	idx.IndexDocument("d3", "s1", "the fox is quick and brown")

	ids, scores := idx.Search("quick fox", 10, "")
	if len(ids) == 0 {
		t.Fatal("expected results")
	}
	// d1 and d3 should match, d2 should not
	for i, id := range ids {
		if id == "d2" {
			t.Errorf("d2 should not match 'quick fox', score=%f", scores[i])
		}
	}
}

func TestBM25Index_SessionFilter(t *testing.T) {
	idx := NewBM25Index(1.5, 0.75)

	idx.IndexDocument("d1", "s1", "hello world")
	idx.IndexDocument("d2", "s2", "hello universe")

	ids, _ := idx.Search("hello", 10, "s1")
	if len(ids) != 1 || ids[0] != "d1" {
		t.Errorf("expected only d1 from session s1, got %v", ids)
	}
}

func TestBM25Index_RemoveDocument(t *testing.T) {
	idx := NewBM25Index(1.5, 0.75)

	idx.IndexDocument("d1", "s1", "hello world")
	idx.RemoveDocument("d1")

	ids, _ := idx.Search("hello", 10, "")
	if len(ids) != 0 {
		t.Errorf("expected no results after removal, got %v", ids)
	}
	if idx.Len() != 0 {
		t.Errorf("expected 0 docs, got %d", idx.Len())
	}
}

func TestBM25Index_UpdateDocument(t *testing.T) {
	idx := NewBM25Index(1.5, 0.75)

	idx.IndexDocument("d1", "s1", "hello world")
	idx.IndexDocument("d1", "s1", "goodbye universe")

	ids, _ := idx.Search("hello", 10, "")
	if len(ids) != 0 {
		t.Errorf("expected no results for old content, got %v", ids)
	}

	ids, _ = idx.Search("goodbye", 10, "")
	if len(ids) != 1 || ids[0] != "d1" {
		t.Errorf("expected d1 for updated content, got %v", ids)
	}
}

func TestBM25Index_DeleteBySession(t *testing.T) {
	idx := NewBM25Index(1.5, 0.75)

	idx.IndexDocument("d1", "s1", "hello world")
	idx.IndexDocument("d2", "s1", "foo bar")
	idx.IndexDocument("d3", "s2", "hello there")

	idx.DeleteBySession("s1")
	if idx.Len() != 1 {
		t.Errorf("expected 1 doc, got %d", idx.Len())
	}
}

func TestBM25Index_EmptyQuery(t *testing.T) {
	idx := NewBM25Index(1.5, 0.75)
	idx.IndexDocument("d1", "s1", "hello world")

	ids, _ := idx.Search("", 10, "")
	if len(ids) != 0 {
		t.Errorf("expected no results for empty query, got %v", ids)
	}
}

func TestBM25Index_EmptyCorpus(t *testing.T) {
	idx := NewBM25Index(1.5, 0.75)
	ids, _ := idx.Search("hello", 10, "")
	if len(ids) != 0 {
		t.Errorf("expected no results for empty corpus, got %v", ids)
	}
}
