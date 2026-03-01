## ADDED Requirements

### Requirement: Global memory statistics endpoint
The API MUST expose global memory statistics at `GET /api/v1/memory/stats`.

#### Scenario: Request global stats
- **WHEN** `GET /api/v1/memory/stats` is called
- **THEN** the system returns aggregated memory statistics across all sessions

### Requirement: Accurate delete result reporting
Memory delete endpoints MUST return actual deletion outcomes, not optimistic request counts.

#### Scenario: Delete with mixed ownership
- **WHEN** `DELETE /api/v1/memory/{sessionID}` is called with IDs from multiple sessions
- **THEN** the response deleted count includes only entries actually deleted for `{sessionID}`

### Requirement: Query mode API contract
Memory query API MUST accept canonical query mode values and reject unsupported values.

#### Scenario: Canonical mode query
- **WHEN** `GET /api/v1/memory/{sessionID}?query=x&mode=vector-only` is called
- **THEN** the request is accepted and processed in vector-only mode

#### Scenario: Unsupported mode query
- **WHEN** `GET /api/v1/memory/{sessionID}?query=x&mode=foobar` is called
- **THEN** the API returns a validation error response

