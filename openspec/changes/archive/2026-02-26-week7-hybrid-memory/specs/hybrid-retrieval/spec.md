## ADDED Requirements

### Requirement: Dual retrieval execution

The system SHALL execute both vector retrieval and BM25 search in parallel for hybrid queries.

#### Scenario: Parallel retrieval
- **WHEN** a hybrid search is performed
- **THEN** the system executes vector retrieval and BM25 search concurrently

#### Scenario: One retriever fails
- **WHEN** vector retrieval fails but BM25 succeeds
- **THEN** the system returns BM25 results only with a warning

### Requirement: RRF score fusion

The system SHALL fuse results using Reciprocal Rank Fusion (RRF) algorithm: RRF(d) = Σ 1/(k + rank(d))

#### Scenario: Fuse results with RRF
- **WHEN** vector retrieval returns [A, B, C] and BM25 returns [B, A, D]
- **THEN** the system calculates RRF scores and returns fused ranking

#### Scenario: RRF with custom k parameter
- **WHEN** RRF is configured with k=60
- **THEN** the system uses k=60 in the RRF formula

### Requirement: Configurable retrieval weights

The system SHALL support configurable weights for vector and BM25 results.

#### Scenario: Default weights
- **WHEN** hybrid retrieval is initialized without custom weights
- **THEN** the system uses vector weight 0.7 and BM25 weight 0.3

#### Scenario: Custom weights
- **WHEN** hybrid retrieval is configured with vector weight 0.5 and BM25 weight 0.5
- **THEN** the system applies equal weights to both retrievers

### Requirement: Result deduplication

The system SHALL deduplicate results that appear in both vector and BM25 results.

#### Scenario: Deduplicate common results
- **WHEN** entry "A" appears in both vector and BM25 results
- **THEN** the system includes "A" only once with combined RRF score

#### Scenario: Preserve unique results
- **WHEN** entry "B" appears only in vector results
- **THEN** the system includes "B" with its RRF score from vector retrieval

### Requirement: Query mode selection

The system SHALL support three query modes: vector-only, BM25-only, and hybrid.

#### Scenario: Vector-only mode
- **WHEN** query mode is set to "vector-only"
- **THEN** the system performs only vector retrieval

#### Scenario: BM25-only mode
- **WHEN** query mode is set to "BM25-only"
- **THEN** the system performs only BM25 search

#### Scenario: Hybrid mode
- **WHEN** query mode is set to "hybrid"
- **THEN** the system performs both retrievals and fuses results

### Requirement: Top-K result limiting

The system SHALL return top-K results after fusion and ranking.

#### Scenario: Limit fused results
- **WHEN** hybrid search returns 50 fused results and topK=10
- **THEN** the system returns the top 10 results by RRF score

#### Scenario: Fewer results than K
- **WHEN** hybrid search returns 5 results and topK=10
- **THEN** the system returns all 5 results

### Requirement: Query with text and vector

The system SHALL support queries with both text and vector components.

#### Scenario: Query with text only
- **WHEN** query contains text but no vector
- **THEN** the system performs BM25 search only

#### Scenario: Query with vector only
- **WHEN** query contains vector but no text
- **THEN** the system performs vector retrieval only

#### Scenario: Query with both text and vector
- **WHEN** query contains both text and vector
- **THEN** the system performs hybrid retrieval

### Requirement: Metadata filtering

The system SHALL support filtering results by metadata fields.

#### Scenario: Filter by metadata
- **WHEN** query includes metadata filter "type=conversation"
- **THEN** the system returns only entries with metadata type=conversation

#### Scenario: Multiple metadata filters
- **WHEN** query includes multiple metadata filters
- **THEN** the system returns only entries matching all filters (AND logic)

### Requirement: Hybrid retrieval performance

The system SHALL perform hybrid retrieval with latency less than 20ms for corpora up to 100K entries.

#### Scenario: Hybrid search in small corpus
- **WHEN** performing hybrid search in a corpus with 1K entries
- **THEN** the search completes in less than 10ms

#### Scenario: Hybrid search in large corpus
- **WHEN** performing hybrid search in a corpus with 100K entries
- **THEN** the search completes in less than 20ms
