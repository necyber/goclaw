# task-metrics Specification

## Purpose
Migrated from legacy OpenSpec format while preserving existing requirement and scenario content.

## Requirements

### Requirement: Task execution metrics
The metrics system SHALL track task execution events with status and task_type labels.

#### Scenario: Record successful task execution
- **WHEN** task completes successfully
- **THEN** system increments task_executions_total counter with status="completed"

#### Scenario: Record failed task execution
- **WHEN** task execution fails
- **THEN** system increments task_executions_total counter with status="failed"

#### Scenario: Track task type
- **WHEN** recording task execution
- **THEN** system includes task_type label if available

#### Scenario: Fallback task type is deterministic
- **WHEN** runtime cannot resolve an explicit task type
- **THEN** system MUST emit a bounded fallback label value instead of an unbounded dynamic identifier

### Requirement: Task duration metrics
The metrics system SHALL measure individual task execution duration with task_type labels.

#### Scenario: Record task execution time
- **WHEN** task completes (success or failure)
- **THEN** system records duration in task_duration_seconds histogram

#### Scenario: Duration histogram buckets
- **WHEN** recording task duration
- **THEN** system uses buckets [0.01, 0.05, 0.1, 0.5, 1, 5, 10, 30] seconds

#### Scenario: Include task type in duration
- **WHEN** recording task duration
- **THEN** system includes task_type label for filtering

### Requirement: Task retry metrics
The metrics system SHALL track task retry attempts with task_type labels.

#### Scenario: Record task retry
- **WHEN** task is retried after failure
- **THEN** system increments task_retries_total counter

#### Scenario: Track retry by task type
- **WHEN** recording task retry
- **THEN** system includes task_type label if available

### Requirement: Task metrics integration
The metrics system SHALL integrate with lane task execution hooks and emit task_type labels consistently for start/completion/retry related metrics.

#### Scenario: Hook into task start
- **WHEN** lane begins executing a task
- **THEN** metrics manager records start timestamp

#### Scenario: Hook into task completion
- **WHEN** lane completes task execution
- **THEN** metrics manager calculates duration and records metrics

#### Scenario: Hook into task retry
- **WHEN** lane retries a failed task
- **THEN** metrics manager increments retry counter before retry attempt

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

