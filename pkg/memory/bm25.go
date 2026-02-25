package memory

import (
	"math"
	"sort"
	"strings"
	"sync"
	"unicode"
)

// BM25Index provides full-text search using the BM25 scoring algorithm.
type BM25Index struct {
	mu sync.RWMutex

	// BM25 parameters
	k1 float64
	b  float64

	// Inverted index: term -> set of entryIDs
	invertedIndex map[string]map[string]struct{}

	// Forward index: entryID -> term frequencies
	termFreqs map[string]map[string]int

	// Document lengths (in tokens)
	docLengths map[string]int

	// Session mapping
	sessions map[string]string // entryID -> sessionID

	// Corpus stats
	totalDocs int
	totalLen  int

	// Stop words (optional)
	stopWords map[string]struct{}
}

// NewBM25Index creates a new BM25 index with the given parameters.
func NewBM25Index(k1, b float64) *BM25Index {
	return &BM25Index{
		k1:            k1,
		b:             b,
		invertedIndex: make(map[string]map[string]struct{}),
		termFreqs:     make(map[string]map[string]int),
		docLengths:    make(map[string]int),
		sessions:      make(map[string]string),
		stopWords:     defaultStopWords(),
	}
}

// IndexDocument adds or updates a document in the index.
func (idx *BM25Index) IndexDocument(entryID, sessionID, content string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Remove old index if updating
	if _, exists := idx.termFreqs[entryID]; exists {
		idx.removeDocLocked(entryID)
	}

	tokens := idx.tokenize(content)
	freqs := make(map[string]int)
	for _, token := range tokens {
		freqs[token]++
	}

	idx.termFreqs[entryID] = freqs
	idx.docLengths[entryID] = len(tokens)
	idx.sessions[entryID] = sessionID
	idx.totalDocs++
	idx.totalLen += len(tokens)

	for term := range freqs {
		if idx.invertedIndex[term] == nil {
			idx.invertedIndex[term] = make(map[string]struct{})
		}
		idx.invertedIndex[term][entryID] = struct{}{}
	}
}

// RemoveDocument removes a document from the index.
func (idx *BM25Index) RemoveDocument(entryID string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.removeDocLocked(entryID)
}

func (idx *BM25Index) removeDocLocked(entryID string) {
	freqs, exists := idx.termFreqs[entryID]
	if !exists {
		return
	}

	for term := range freqs {
		if docs, ok := idx.invertedIndex[term]; ok {
			delete(docs, entryID)
			if len(docs) == 0 {
				delete(idx.invertedIndex, term)
			}
		}
	}

	idx.totalLen -= idx.docLengths[entryID]
	idx.totalDocs--
	delete(idx.termFreqs, entryID)
	delete(idx.docLengths, entryID)
	delete(idx.sessions, entryID)
}

// Search performs a BM25 search and returns the top-K results.
// If sessionID is non-empty, results are filtered to that session.
func (idx *BM25Index) Search(query string, topK int, sessionID string) ([]string, []float64) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if idx.totalDocs == 0 {
		return nil, nil
	}

	queryTokens := idx.tokenize(query)
	if len(queryTokens) == 0 {
		return nil, nil
	}

	avgDL := float64(idx.totalLen) / float64(idx.totalDocs)

	// Collect candidate documents using pre-allocated map
	candidates := make(map[string]struct{}, len(idx.invertedIndex))
	for _, token := range queryTokens {
		if docs, ok := idx.invertedIndex[token]; ok {
			for id := range docs {
				if sessionID != "" && idx.sessions[id] != sessionID {
					continue
				}
				candidates[id] = struct{}{}
			}
		}
	}

	type scored struct {
		id    string
		score float64
	}

	results := make([]scored, 0, len(candidates))
	for id := range candidates {
		score := idx.scoreLocked(id, queryTokens, avgDL)
		if score > 0 {
			results = append(results, scored{id: id, score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	if topK > len(results) {
		topK = len(results)
	}
	results = results[:topK]

	ids := make([]string, topK)
	scores := make([]float64, topK)
	for i, r := range results {
		ids[i] = r.id
		scores[i] = r.score
	}
	return ids, scores
}

// DeleteBySession removes all documents for a session.
func (idx *BM25Index) DeleteBySession(sessionID string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	var toRemove []string
	for id, sid := range idx.sessions {
		if sid == sessionID {
			toRemove = append(toRemove, id)
		}
	}
	for _, id := range toRemove {
		idx.removeDocLocked(id)
	}
}

// Len returns the number of indexed documents.
func (idx *BM25Index) Len() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.totalDocs
}

// scoreLocked calculates the BM25 score for a document. Must be called with read lock held.
func (idx *BM25Index) scoreLocked(docID string, queryTokens []string, avgDL float64) float64 {
	docLen := float64(idx.docLengths[docID])
	freqs := idx.termFreqs[docID]
	score := 0.0

	for _, term := range queryTokens {
		tf := float64(freqs[term])
		if tf == 0 {
			continue
		}

		// IDF: log((N - n + 0.5) / (n + 0.5) + 1)
		n := float64(len(idx.invertedIndex[term]))
		idf := math.Log((float64(idx.totalDocs)-n+0.5)/(n+0.5) + 1.0)

		// BM25 term score
		numerator := tf * (idx.k1 + 1)
		denominator := tf + idx.k1*(1-idx.b+idx.b*docLen/avgDL)
		score += idf * numerator / denominator
	}

	return score
}

// tokenize splits text into lowercase tokens, removing punctuation and stop words.
func (idx *BM25Index) tokenize(text string) []string {
	text = strings.ToLower(text)

	// Pre-allocate with estimated capacity
	tokens := make([]string, 0, len(text)/4)
	var current strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				token := current.String()
				if _, isStop := idx.stopWords[token]; !isStop {
					tokens = append(tokens, token)
				}
				current.Reset()
			}
			// Handle CJK characters as individual tokens
			if unicode.Is(unicode.Han, r) {
				tokens = append(tokens, string(r))
			}
		}
	}
	if current.Len() > 0 {
		token := current.String()
		if _, isStop := idx.stopWords[token]; !isStop {
			tokens = append(tokens, token)
		}
	}

	return tokens
}

func defaultStopWords() map[string]struct{} {
	words := []string{
		"a", "an", "the", "is", "are", "was", "were", "be", "been", "being",
		"have", "has", "had", "do", "does", "did", "will", "would", "could",
		"should", "may", "might", "shall", "can", "need", "dare", "ought",
		"used", "to", "of", "in", "for", "on", "with", "at", "by", "from",
		"as", "into", "through", "during", "before", "after", "above", "below",
		"between", "out", "off", "over", "under", "again", "further", "then",
		"once", "and", "but", "or", "nor", "not", "so", "yet", "both",
		"either", "neither", "each", "every", "all", "any", "few", "more",
		"most", "other", "some", "such", "no", "only", "own", "same", "than",
		"too", "very", "just", "because", "if", "when", "where", "how", "what",
		"which", "who", "whom", "this", "that", "these", "those", "i", "me",
		"my", "myself", "we", "our", "ours", "ourselves", "you", "your",
		"yours", "yourself", "yourselves", "he", "him", "his", "himself",
		"she", "her", "hers", "herself", "it", "its", "itself", "they",
		"them", "their", "theirs", "themselves",
	}
	m := make(map[string]struct{}, len(words))
	for _, w := range words {
		m[w] = struct{}{}
	}
	return m
}
