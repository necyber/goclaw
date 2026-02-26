## ADDED Requirements

### Requirement: Task metrics cover scheduling to terminal lifecycle
Task metrics MUST cover task transitions from scheduling through terminal completion.

#### Scenario: Scheduled to running transition
- **WHEN** a task transitions from `scheduled` to `running`
- **THEN** task execution and wait-duration metrics MUST record scheduling latency

#### Scenario: Running to terminal transition
- **WHEN** a task transitions from `running` to terminal state
- **THEN** terminal counters and task duration metrics MUST be recorded from that transition

### Requirement: Retry and cancellation metrics are explicit
Task metrics MUST explicitly track retries and cancellation/timeout-derived terminal outcomes.

#### Scenario: Task retry attempt
- **WHEN** a task enters retry attempt after failure
- **THEN** retry metrics MUST increment with lane and task labels

#### Scenario: Task cancellation or timeout outcome
- **WHEN** a task terminates due to cancellation or timeout policy
- **THEN** terminal metrics MUST capture cancellation/timeout outcome labels consistently

### Requirement: Terminal metrics are idempotent
Task terminal metric emission MUST be idempotent per task attempt.

#### Scenario: Duplicate terminal callback prevented
- **WHEN** terminal transition callback is triggered more than once for the same attempt
- **THEN** metrics subsystem MUST count terminal outcome only once for that attempt

