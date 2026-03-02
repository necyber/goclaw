## MODIFIED Requirements

### Requirement: Retry and cancellation metrics are explicit
Task metrics MUST explicitly track retries and cancellation/timeout-derived terminal outcomes.

#### Scenario: Task retry attempt
- **WHEN** a task enters retry attempt after failure
- **THEN** retry metrics MUST increment with lane and task labels

#### Scenario: Task cancellation or timeout outcome
- **WHEN** a task terminates due to cancellation or timeout policy
- **THEN** terminal metrics MUST capture cancellation/timeout outcome labels consistently

#### Scenario: User cancellation label is distinct from timeout label
- **WHEN** a task terminates because workflow/user cancellation is observed before timeout policy fires
- **THEN** terminal task metric label MUST be cancellation-derived label
- **AND** it MUST NOT be recorded as timeout-derived terminal label

#### Scenario: Timeout-derived terminal label is explicit
- **WHEN** a task terminal outcome is produced by deadline/timeout policy
- **THEN** terminal task metric label MUST be timeout-derived label
- **AND** it MUST be distinguishable from user-cancelled terminal label in metrics queries

### Requirement: Terminal metrics are idempotent
Task terminal metric emission MUST be idempotent per task attempt.

#### Scenario: Duplicate terminal callback prevented
- **WHEN** terminal transition callback is triggered more than once for the same attempt
- **THEN** metrics subsystem MUST count terminal outcome only once for that attempt
