## ADDED Requirements

### Requirement: Versioned event envelope
All distributed lifecycle events MUST use a versioned common envelope.

#### Scenario: Required envelope fields
- **WHEN** runtime emits a distributed lifecycle event
- **THEN** the envelope MUST include `event_id`, `event_type`, `timestamp`, `schema_version`, and source node identity

#### Scenario: Workflow and task identity fields
- **WHEN** event relates to workflow or task lifecycle
- **THEN** envelope or payload MUST include workflow/task identifiers sufficient for consumer correlation

### Requirement: Backward-compatible schema evolution
Event schema evolution MUST preserve compatibility guarantees across supported versions.

#### Scenario: Additive schema change
- **WHEN** a new optional field is introduced
- **THEN** older consumers MUST continue processing events without parse failure

#### Scenario: Breaking schema change
- **WHEN** an incompatible schema change is required
- **THEN** runtime MUST publish with a new schema version and provide explicit compatibility window

### Requirement: Ordering metadata for consumers
Events MUST include ordering metadata for scoped consumer sequencing.

#### Scenario: Per-workflow ordering metadata
- **WHEN** multiple events are emitted for the same workflow
- **THEN** events MUST carry ordering metadata that allows consumers to reconstruct per-workflow transition order

