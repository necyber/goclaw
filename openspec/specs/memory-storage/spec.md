# memory-storage Specification

## Purpose
Migrated from legacy OpenSpec format while preserving existing requirement and scenario content.

## Requirements

### Requirement: Three-tier storage architecture

The system SHALL implement a three-tier storage architecture (L1/L2/L3) for memory entries.

#### Scenario: L1 cache hit
- **WHEN** a memory entry is requested and exists in L1 cache
- **THEN** the system returns the entry from L1 without accessing L2 or L3

#### Scenario: L1 miss, L2 hit
- **WHEN** a memory entry is not in L1 but exists in L2 (Badger)
- **THEN** the system retrieves from L2 and promotes to L1 cache

#### Scenario: L1 and L2 miss, L3 hit
- **WHEN** a memory entry is not in L1 or L2 but exists in L3 (vector database)
- **THEN** the system retrieves from L3 and promotes to L2 and L1

### Requirement: L1 cache management

The system SHALL maintain an LRU cache in memory with configurable maximum size.

#### Scenario: Cache eviction on size limit
- **WHEN** L1 cache reaches maximum size and a new entry is added
- **THEN** the least recently used entry is evicted from L1

#### Scenario: Cache invalidation
- **WHEN** a memory entry is deleted or updated
- **THEN** the system invalidates the corresponding L1 cache entry

### Requirement: L2 Badger persistence

The system SHALL persist memory entries to Badger database with session-based key prefixing.

#### Scenario: Store entry to Badger
- **WHEN** a memory entry is stored
- **THEN** the system writes to Badger with key format "memory:{sessionID}:{entryID}"

#### Scenario: Retrieve entries by session
- **WHEN** retrieving memories for a session
- **THEN** the system scans Badger with prefix "memory:{sessionID}:"

### Requirement: L3 vector database integration

The system SHALL support optional L3 vector database integration (Weaviate or Qdrant).

#### Scenario: Vector database disabled
- **WHEN** L3 vector database is not configured
- **THEN** the system operates with L1 and L2 only

#### Scenario: Vector database enabled
- **WHEN** L3 vector database is configured and enabled
- **THEN** the system stores vectors in L3 and uses it for vector retrieval

### Requirement: Memory entry structure

The system SHALL store memory entries with the following fields: ID, TaskID, SessionID, Content, Vector, Metadata, Strength, LastReview, CreatedAt.

#### Scenario: Store complete entry
- **WHEN** a memory entry is created with all fields
- **THEN** the system persists all fields to storage

#### Scenario: Retrieve entry with metadata
- **WHEN** a memory entry is retrieved
- **THEN** the system returns all fields including metadata

### Requirement: Session isolation

The system SHALL isolate memory entries by SessionID to prevent cross-session data leakage.

#### Scenario: Query within session
- **WHEN** retrieving memories for session "A"
- **THEN** the system returns only entries with SessionID "A"

#### Scenario: Delete within session
- **WHEN** deleting memories for session "A"
- **THEN** the system deletes only entries with SessionID "A"

### Requirement: Batch operations

The system SHALL support batch store and retrieve operations for efficiency.

#### Scenario: Batch store
- **WHEN** storing multiple memory entries in one call
- **THEN** the system writes all entries in a single transaction

#### Scenario: Batch retrieve
- **WHEN** retrieving multiple memory entries by IDs
- **THEN** the system returns all entries in one operation

### Requirement: Storage error handling

The system SHALL handle storage errors gracefully and return appropriate error codes.

#### Scenario: Badger write failure
- **WHEN** Badger write operation fails
- **THEN** the system returns an error and does not update L1 cache

#### Scenario: Vector database unavailable
- **WHEN** L3 vector database is unavailable
- **THEN** the system falls back to L1 and L2 without failing the operation

