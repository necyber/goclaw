# memory-hub-api Specification

## Purpose
Migrated from legacy OpenSpec format while preserving existing requirement and scenario content.

## Requirements

### Requirement: Memorize operation

The system SHALL provide a Memorize operation to store new memory entries.

#### Scenario: Memorize with text content
- **WHEN** Memorize is called with sessionID, text content, and metadata
- **THEN** the system stores the entry and returns a unique entry ID

#### Scenario: Memorize with vector
- **WHEN** Memorize is called with sessionID, content, vector, and metadata
- **THEN** the system stores the entry with vector for semantic retrieval

#### Scenario: Memorize without vector
- **WHEN** Memorize is called without a vector
- **THEN** the system stores the entry for BM25 search only

### Requirement: Retrieve operation

The system SHALL provide a Retrieve operation to search memory entries.

#### Scenario: Retrieve by text query
- **WHEN** Retrieve is called with sessionID and text query
- **THEN** the system performs hybrid search and returns top-K matching entries

#### Scenario: Retrieve by vector query
- **WHEN** Retrieve is called with sessionID and query vector
- **THEN** the system performs vector search and returns top-K similar entries

#### Scenario: Retrieve with metadata filters
- **WHEN** Retrieve is called with metadata filters
- **THEN** the system returns only entries matching the filters

### Requirement: Forget operation

The system SHALL provide a Forget operation to delete memory entries.

#### Scenario: Forget by entry IDs
- **WHEN** Forget is called with sessionID and list of entry IDs
- **THEN** the system deletes the specified entries from all storage tiers

#### Scenario: Forget non-existent entry
- **WHEN** Forget is called with a non-existent entry ID
- **THEN** the system returns success without error

### Requirement: ForgetByThreshold operation

The system SHALL provide a ForgetByThreshold operation to delete entries below a strength threshold.

#### Scenario: Forget weak memories
- **WHEN** ForgetByThreshold is called with sessionID and threshold 0.1
- **THEN** the system deletes all entries with strength < 0.1 and returns count

#### Scenario: No entries below threshold
- **WHEN** ForgetByThreshold is called but no entries are below threshold
- **THEN** the system returns count of 0

### Requirement: List operation

The system SHALL provide a List operation to retrieve all memory entries for a session.

#### Scenario: List all memories
- **WHEN** List is called with sessionID
- **THEN** the system returns all memory entries for that session

#### Scenario: List with pagination
- **WHEN** List is called with sessionID, limit, and offset
- **THEN** the system returns paginated results

### Requirement: Count operation

The system SHALL provide a Count operation to get the number of memory entries.

#### Scenario: Count memories in session
- **WHEN** Count is called with sessionID
- **THEN** the system returns the total number of entries for that session

#### Scenario: Count with metadata filter
- **WHEN** Count is called with sessionID and metadata filter
- **THEN** the system returns the count of entries matching the filter

### Requirement: Context-aware retrieval

The system SHALL support context-aware retrieval using conversation history.

#### Scenario: Retrieve with context
- **WHEN** Retrieve is called with recent conversation context
- **THEN** the system uses context to improve retrieval relevance

#### Scenario: Retrieve without context
- **WHEN** Retrieve is called without context
- **THEN** the system performs standard retrieval based on query only

### Requirement: Batch memorize operation

The system SHALL support batch memorization of multiple entries.

#### Scenario: Batch memorize
- **WHEN** BatchMemorize is called with multiple entries
- **THEN** the system stores all entries in a single transaction and returns all IDs

#### Scenario: Batch memorize partial failure
- **WHEN** BatchMemorize is called and some entries fail validation
- **THEN** the system stores valid entries and returns errors for invalid ones

### Requirement: Memory statistics

The system SHALL provide statistics about memory usage per session.

#### Scenario: Get memory stats
- **WHEN** GetStats is called with sessionID
- **THEN** the system returns total entries, average strength, storage size

#### Scenario: Get global stats
- **WHEN** GetStats is called without sessionID
- **THEN** the system returns statistics across all sessions

### Requirement: Session management

The system SHALL support session lifecycle operations.

#### Scenario: Create session
- **WHEN** a new sessionID is used for the first time
- **THEN** the system automatically creates the session

#### Scenario: Delete session
- **WHEN** DeleteSession is called with sessionID
- **THEN** the system deletes all memory entries for that session

### Requirement: API error handling

The system SHALL return appropriate errors for invalid operations.

#### Scenario: Invalid session ID
- **WHEN** an operation is called with empty sessionID
- **THEN** the system returns ErrInvalidSessionID

#### Scenario: Invalid query
- **WHEN** Retrieve is called with empty query (no text and no vector)
- **THEN** the system returns ErrInvalidQuery

#### Scenario: Storage unavailable
- **WHEN** an operation is called but storage is unavailable
- **THEN** the system returns ErrStorageUnavailable

### Requirement: Concurrent operation support

The system SHALL support concurrent operations from multiple goroutines.

#### Scenario: Concurrent memorize
- **WHEN** multiple goroutines call Memorize simultaneously
- **THEN** all operations complete successfully without data races

#### Scenario: Concurrent retrieve
- **WHEN** multiple goroutines call Retrieve simultaneously
- **THEN** all operations return correct results without blocking each other

