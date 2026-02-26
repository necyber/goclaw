## ADDED Requirements

### Requirement: HNSW index initialization

The system SHALL initialize an HNSW index with configurable dimension and distance metric.

#### Scenario: Initialize with cosine similarity
- **WHEN** vector retrieval is initialized with dimension 768 and metric "cosine"
- **THEN** the system creates an HNSW index supporting 768-dimensional vectors with cosine similarity

#### Scenario: Initialize with custom parameters
- **WHEN** vector retrieval is initialized with custom M and efConstruction parameters
- **THEN** the system creates an HNSW index with the specified parameters

### Requirement: Vector indexing

The system SHALL add vectors to the HNSW index with associated memory entry IDs.

#### Scenario: Add vector to index
- **WHEN** a memory entry with vector is stored
- **THEN** the system adds the vector to HNSW index with the entry ID

#### Scenario: Update vector in index
- **WHEN** a memory entry's vector is updated
- **THEN** the system removes the old vector and adds the new vector to the index

### Requirement: Approximate nearest neighbor search

The system SHALL perform approximate nearest neighbor search using HNSW algorithm.

#### Scenario: Search top-K similar vectors
- **WHEN** searching for top 10 similar vectors to a query vector
- **THEN** the system returns up to 10 memory entry IDs with similarity scores in descending order

#### Scenario: Search with minimum similarity threshold
- **WHEN** searching with minimum similarity threshold 0.7
- **THEN** the system returns only entries with similarity score >= 0.7

### Requirement: Cosine similarity calculation

The system SHALL calculate cosine similarity between query vector and indexed vectors.

#### Scenario: Calculate similarity for identical vectors
- **WHEN** query vector is identical to an indexed vector
- **THEN** the system returns similarity score of 1.0

#### Scenario: Calculate similarity for orthogonal vectors
- **WHEN** query vector is orthogonal to an indexed vector
- **THEN** the system returns similarity score of 0.0

### Requirement: Vector dimension validation

The system SHALL validate that all vectors have the same dimension as the index.

#### Scenario: Add vector with correct dimension
- **WHEN** adding a vector with dimension matching the index
- **THEN** the system successfully adds the vector

#### Scenario: Reject vector with incorrect dimension
- **WHEN** adding a vector with dimension not matching the index
- **THEN** the system returns a dimension mismatch error

### Requirement: Session-based vector filtering

The system SHALL filter vector search results by SessionID.

#### Scenario: Search within session
- **WHEN** searching vectors for session "A"
- **THEN** the system returns only entries belonging to session "A"

#### Scenario: Search across all sessions
- **WHEN** searching vectors with empty session filter
- **THEN** the system returns entries from all sessions

### Requirement: Index persistence

The system SHALL support saving and loading HNSW index to/from disk.

#### Scenario: Save index to disk
- **WHEN** the system shuts down gracefully
- **THEN** the HNSW index is saved to disk

#### Scenario: Load index from disk
- **WHEN** the system starts up
- **THEN** the HNSW index is loaded from disk if it exists

### Requirement: Concurrent search support

The system SHALL support concurrent vector searches without blocking.

#### Scenario: Multiple concurrent searches
- **WHEN** multiple goroutines perform vector searches simultaneously
- **THEN** all searches complete successfully without data races

### Requirement: Vector retrieval performance

The system SHALL perform vector search with sub-millisecond latency for indexes up to 100K vectors.

#### Scenario: Search in small index
- **WHEN** searching in an index with 1K vectors
- **THEN** the search completes in less than 1ms

#### Scenario: Search in large index
- **WHEN** searching in an index with 100K vectors
- **THEN** the search completes in less than 10ms
