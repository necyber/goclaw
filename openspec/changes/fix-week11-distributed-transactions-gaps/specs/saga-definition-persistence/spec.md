## ADDED Requirements

### Requirement: Persist Saga definition snapshot by saga ID
The system SHALL persist a complete executable Saga definition snapshot keyed by `saga_id` when a Saga instance is submitted.

#### Scenario: Store definition at submission
- **WHEN** a Saga is accepted for execution
- **THEN** the system stores the full definition snapshot using the created `saga_id`

#### Scenario: Read definition for lifecycle operation
- **WHEN** compensation or recovery is requested for an existing `saga_id`
- **THEN** the system resolves the definition snapshot from durable storage without requiring process-local caches

### Requirement: Definition snapshot lookup semantics
The system SHALL return explicit errors when a required definition snapshot is missing or unreadable.

#### Scenario: Missing definition snapshot
- **WHEN** a lifecycle operation references a `saga_id` with no stored definition
- **THEN** the system returns a not-found or precondition error and does not execute forward or reverse steps

#### Scenario: Corrupted definition snapshot
- **WHEN** definition snapshot deserialization fails
- **THEN** the system surfaces an internal error and records recovery failure telemetry
