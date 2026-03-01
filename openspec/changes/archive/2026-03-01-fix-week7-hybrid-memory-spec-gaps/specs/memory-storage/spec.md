## ADDED Requirements

### Requirement: Global memory entry iteration
Memory storage MUST provide iteration over all persisted memory entries across sessions for maintenance workflows.

#### Scenario: Iterate all sessions
- **WHEN** decay or bootstrap requests a global scan
- **THEN** storage returns entries from all sessions, not just a single session prefix

### Requirement: Session-scoped delete by entry ID
Memory storage MUST support deleting an entry by `(sessionID, entryID)` and MUST NOT delete entries from other sessions.

#### Scenario: Delete with matching session
- **WHEN** delete is called with `(session="A", entry="x")` and `x` belongs to `A`
- **THEN** storage deletes `x`

#### Scenario: Delete with mismatched session
- **WHEN** delete is called with `(session="A", entry="y")` and `y` belongs to `B`
- **THEN** storage MUST leave `y` unchanged

