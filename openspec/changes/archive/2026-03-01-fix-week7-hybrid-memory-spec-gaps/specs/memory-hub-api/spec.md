## ADDED Requirements

### Requirement: Session-safe forget behavior
The memory hub MUST enforce session ownership when forgetting entries by ID.

#### Scenario: Forget own-session entry
- **WHEN** `Forget(session="A", ids=[x])` is called and entry `x` belongs to session `A`
- **THEN** the hub deletes entry `x` from storage and all retrieval indexes

#### Scenario: Forget cross-session entry
- **WHEN** `Forget(session="A", ids=[y])` is called and entry `y` belongs to session `B`
- **THEN** the hub MUST NOT delete entry `y`

### Requirement: Canonical query mode validation
The memory hub MUST support canonical query modes (`hybrid`, `vector-only`, `bm25-only`) and reject unknown modes.

#### Scenario: Canonical mode accepted
- **WHEN** `Retrieve` is called with mode `vector-only`
- **THEN** the hub executes vector retrieval only

#### Scenario: Unknown mode rejected
- **WHEN** `Retrieve` is called with mode `foobar`
- **THEN** the hub returns a validation error and MUST NOT silently fallback to another mode

### Requirement: Startup index bootstrap
The memory hub MUST rebuild in-memory retrieval indexes from persisted memory entries during startup.

#### Scenario: Rebuild after restart
- **WHEN** persisted entries exist and the process restarts
- **THEN** `Start` rebuilds vector/BM25 indexes before serving retrieval requests

