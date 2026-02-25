package memory

import (
	"context"
	"sort"
	"sync"
)

// QueryMode defines the retrieval strategy.
const (
	ModeHybrid = "hybrid"
	ModeVector = "vector"
	ModeBM25   = "bm25"
)

// HybridRetriever combines vector and BM25 retrieval with RRF fusion.
type HybridRetriever struct {
	vector       *VectorIndex
	bm25         *BM25Index
	vectorWeight float64
	bm25Weight   float64
	rrfK         float64 // RRF constant, typically 60
}

// NewHybridRetriever creates a new hybrid retriever.
func NewHybridRetriever(vector *VectorIndex, bm25 *BM25Index, vectorWeight, bm25Weight float64) *HybridRetriever {
	return &HybridRetriever{
		vector:       vector,
		bm25:         bm25,
		vectorWeight: vectorWeight,
		bm25Weight:   bm25Weight,
		rrfK:         60.0,
	}
}

// Retrieve performs hybrid retrieval based on the query mode.
func (h *HybridRetriever) Retrieve(ctx context.Context, sessionID string, query Query, getEntry func(string) *MemoryEntry) ([]*RetrievalResult, error) {
	mode := query.Mode
	if mode == "" {
		// Auto-detect mode based on query contents
		hasText := query.Text != ""
		hasVector := len(query.Vector) > 0
		switch {
		case hasText && hasVector:
			mode = ModeHybrid
		case hasVector:
			mode = ModeVector
		case hasText:
			mode = ModeBM25
		default:
			return nil, ErrInvalidQuery
		}
	}

	topK := query.TopK
	if topK <= 0 {
		topK = 10
	}

	// Fetch more candidates from each retriever for better fusion
	fetchK := topK * 3
	if fetchK < 30 {
		fetchK = 30
	}

	switch mode {
	case ModeVector:
		return h.vectorOnly(ctx, sessionID, query, topK, getEntry)
	case ModeBM25:
		return h.bm25Only(ctx, sessionID, query, topK, getEntry)
	case ModeHybrid:
		return h.hybrid(ctx, sessionID, query, topK, fetchK, getEntry)
	default:
		return h.hybrid(ctx, sessionID, query, topK, fetchK, getEntry)
	}
}

func (h *HybridRetriever) vectorOnly(ctx context.Context, sessionID string, query Query, topK int, getEntry func(string) *MemoryEntry) ([]*RetrievalResult, error) {
	if len(query.Vector) == 0 {
		return nil, ErrInvalidQuery
	}
	ids, scores, err := h.vector.Search(query.Vector, topK, sessionID)
	if err != nil {
		return nil, err
	}
	return h.buildResults(ids, scores, query.Filters, getEntry), nil
}

func (h *HybridRetriever) bm25Only(ctx context.Context, sessionID string, query Query, topK int, getEntry func(string) *MemoryEntry) ([]*RetrievalResult, error) {
	if query.Text == "" {
		return nil, ErrInvalidQuery
	}
	ids, scores := h.bm25.Search(query.Text, topK, sessionID)
	return h.buildResults(ids, scores, query.Filters, getEntry), nil
}

func (h *HybridRetriever) hybrid(ctx context.Context, sessionID string, query Query, topK, fetchK int, getEntry func(string) *MemoryEntry) ([]*RetrievalResult, error) {
	type result struct {
		ids    []string
		scores []float64
		err    error
	}

	var wg sync.WaitGroup
	var vectorRes, bm25Res result

	// Run vector and BM25 in parallel
	if len(query.Vector) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			vectorRes.ids, vectorRes.scores, vectorRes.err = h.vector.Search(query.Vector, fetchK, sessionID)
		}()
	}

	if query.Text != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bm25Res.ids, bm25Res.scores = h.bm25.Search(query.Text, fetchK, sessionID)
		}()
	}

	wg.Wait()

	// Graceful degradation: if one fails, use the other
	if vectorRes.err != nil && len(bm25Res.ids) > 0 {
		return h.buildResults(bm25Res.ids, bm25Res.scores, query.Filters, getEntry), nil
	}
	if vectorRes.err != nil {
		return nil, vectorRes.err
	}
	if len(vectorRes.ids) == 0 && len(bm25Res.ids) == 0 {
		return nil, nil
	}

	// RRF fusion
	fused := h.fuseRRF(vectorRes.ids, bm25Res.ids)

	if topK > len(fused) {
		topK = len(fused)
	}
	fused = fused[:topK]

	// Build results
	var results []*RetrievalResult
	for _, f := range fused {
		entry := getEntry(f.id)
		if entry == nil {
			continue
		}
		if !matchesFilters(entry, query.Filters) {
			continue
		}
		results = append(results, &RetrievalResult{
			Entry: entry,
			Score: f.score,
		})
	}
	return results, nil
}

type fusedResult struct {
	id    string
	score float64
}

// fuseRRF applies Reciprocal Rank Fusion: RRF(d) = Σ weight/(k + rank(d))
func (h *HybridRetriever) fuseRRF(vectorIDs, bm25IDs []string) []fusedResult {
	scores := make(map[string]float64)

	for rank, id := range vectorIDs {
		scores[id] += h.vectorWeight / (h.rrfK + float64(rank+1))
	}
	for rank, id := range bm25IDs {
		scores[id] += h.bm25Weight / (h.rrfK + float64(rank+1))
	}

	results := make([]fusedResult, 0, len(scores))
	for id, score := range scores {
		results = append(results, fusedResult{id: id, score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	return results
}

func (h *HybridRetriever) buildResults(ids []string, scores []float64, filters map[string]string, getEntry func(string) *MemoryEntry) []*RetrievalResult {
	var results []*RetrievalResult
	for i, id := range ids {
		entry := getEntry(id)
		if entry == nil {
			continue
		}
		if !matchesFilters(entry, filters) {
			continue
		}
		score := 0.0
		if i < len(scores) {
			score = scores[i]
		}
		results = append(results, &RetrievalResult{
			Entry: entry,
			Score: score,
		})
	}
	return results
}

// matchesFilters checks if an entry matches all metadata filters (AND logic).
func matchesFilters(entry *MemoryEntry, filters map[string]string) bool {
	if len(filters) == 0 {
		return true
	}
	for k, v := range filters {
		if entry.Metadata == nil || entry.Metadata[k] != v {
			return false
		}
	}
	return true
}
