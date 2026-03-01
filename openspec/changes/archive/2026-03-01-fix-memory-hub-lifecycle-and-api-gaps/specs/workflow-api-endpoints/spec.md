## MODIFIED Requirements

### Requirement: Query memory entries

The system SHALL provide an API endpoint to query memory entries for a session.

#### Scenario: Query memories by text
- **WHEN** GET /api/v1/memory/{sessionID}?query=text&limit=10 is called
- **THEN** the system returns top 10 matching memory entries

#### Scenario: Query memories with metadata filter
- **WHEN** GET /api/v1/memory/{sessionID}?query=text&metadata.type=conversation is called
- **THEN** the system returns only entries matching the metadata filter

#### Scenario: Canonical mode query
- **WHEN** GET /api/v1/memory/{sessionID}?mode=vector-only&vector=v1,v2,...,vn is called
- **THEN** the request is accepted and processed in vector-only mode without requiring `query` text

#### Scenario: Unsupported mode query
- **WHEN** GET /api/v1/memory/{sessionID}?query=x&mode=foobar is called
- **THEN** the API returns a validation error response

### Requirement: List memory entries

The system SHALL provide an API endpoint to list all memory entries for a session.

#### Scenario: List with pagination
- **WHEN** GET /api/v1/memory/{sessionID}/list?limit=20&offset=0 is called
- **THEN** the system returns paginated list of memory entries

#### Scenario: List with sorting
- **WHEN** GET /api/v1/memory/{sessionID}/list?sort=strength&order=desc is called
- **THEN** the system returns entries sorted by `strength` in descending order before pagination is applied

### Requirement: Get memory statistics

The system SHALL provide an API endpoint to retrieve memory statistics.

#### Scenario: Get session statistics
- **WHEN** GET /api/v1/memory/{sessionID}/stats is called
- **THEN** the system returns total entries, average strength, and storage size in bytes for the session

#### Scenario: Get global statistics
- **WHEN** GET /api/v1/memory/stats is called
- **THEN** the system returns statistics across all sessions
