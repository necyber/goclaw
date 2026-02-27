## ADDED Requirements

### Requirement: Signal routing honors distributed ownership
In distributed mode, signal delivery MUST route according to current task ownership.

#### Scenario: Signal to remotely owned task
- **WHEN** a signal is published for a task owned by another node
- **THEN** runtime MUST route the signal to the owner node via distributed signal transport

#### Scenario: Signal to local owned task
- **WHEN** a signal is published for a task owned by the local node
- **THEN** runtime MAY deliver through local fast-path while preserving signal contract

### Requirement: Signal delivery during ownership changes
Signal routing MUST remain deterministic while ownership changes are in progress.

#### Scenario: Ownership changes during signal send
- **WHEN** ownership changes between signal publish and consume
- **THEN** runtime MUST deliver to current owner or return explicit ownership-change error based on policy

#### Scenario: Duplicate signal path prevention
- **WHEN** both local and distributed routing paths are available
- **THEN** runtime MUST avoid duplicate signal delivery for the same signal identifier

