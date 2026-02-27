# workflow-api-endpoints Specification

## Purpose
Migrated from legacy OpenSpec format while preserving existing requirement and scenario content.

## Requirements

### Requirement: Query memory entries

The system SHALL provide an API endpoint to query memory entries for a session.

#### Scenario: Query memories by text
- **WHEN** GET /api/v1/memory/{sessionID}?query=text&limit=10 is called
- **THEN** the system returns top 10 matching memory entries

#### Scenario: Query memories with metadata filter
- **WHEN** GET /api/v1/memory/{sessionID}?query=text&metadata.type=conversation is called
- **THEN** the system returns only entries matching the metadata filter

### Requirement: Store memory entry

The system SHALL provide an API endpoint to store new memory entries.

#### Scenario: Store memory with content
- **WHEN** POST /api/v1/memory/{sessionID} is called with content and metadata
- **THEN** the system stores the entry and returns the entry ID

#### Scenario: Store memory with vector
- **WHEN** POST /api/v1/memory/{sessionID} is called with content, vector, and metadata
- **THEN** the system stores the entry with vector for semantic search

### Requirement: Delete memory entries

The system SHALL provide an API endpoint to delete memory entries.

#### Scenario: Delete by entry IDs
- **WHEN** DELETE /api/v1/memory/{sessionID} is called with entry IDs in request body
- **THEN** the system deletes the specified entries

#### Scenario: Delete by strength threshold
- **WHEN** DELETE /api/v1/memory/{sessionID}/weak?threshold=0.1 is called
- **THEN** the system deletes all entries with strength below 0.1

### Requirement: List memory entries

The system SHALL provide an API endpoint to list all memory entries for a session.

#### Scenario: List with pagination
- **WHEN** GET /api/v1/memory/{sessionID}/list?limit=20&offset=0 is called
- **THEN** the system returns paginated list of memory entries

#### Scenario: List with sorting
- **WHEN** GET /api/v1/memory/{sessionID}/list?sort=strength&order=desc is called
- **THEN** the system returns entries sorted by strength in descending order

### Requirement: Get memory statistics

The system SHALL provide an API endpoint to retrieve memory statistics.

#### Scenario: Get session statistics
- **WHEN** GET /api/v1/memory/{sessionID}/stats is called
- **THEN** the system returns total entries, average strength, and storage size

#### Scenario: Get global statistics
- **WHEN** GET /api/v1/memory/stats is called
- **THEN** the system returns statistics across all sessions

### Requirement: Delete session memories

The system SHALL provide an API endpoint to delete all memories for a session.

#### Scenario: Delete entire session
- **WHEN** DELETE /api/v1/memory/{sessionID}/all is called
- **THEN** the system deletes all memory entries for that session

#### Scenario: Confirm deletion count
- **WHEN** DELETE /api/v1/memory/{sessionID}/all is called
- **THEN** the system returns the count of deleted entries

