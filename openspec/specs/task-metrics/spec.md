## ADDED Requirements

### Requirement: Task execution metrics
The metrics system SHALL track task execution events with status and type labels.

#### Scenario: Record successful task execution
- **WHEN** task completes successfully
- **THEN** system increments task_executions_total counter with status="completed"

#### Scenario: Record failed task execution
- **WHEN** task execution fails
- **THEN** system increments task_executions_total counter with status="failed"

#### Scenario: Track task type
- **WHEN** recording task execution
- **THEN** system includes task_type label if available

### Requirement: Task duration metrics
The metrics system SHALL measure individual task execution duration.

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
The metrics system SHALL track task retry attempts.

#### Scenario: Record task retry
- **WHEN** task is retried after failure
- **THEN** system increments task_retries_total counter

#### Scenario: Track retry by task type
- **WHEN** recording task retry
- **THEN** system includes task_type label if available

### Requirement: Task metrics integration
The metrics system SHALL integrate with lane task execution hooks.

#### Scenario: Hook into task start
- **WHEN** lane begins executing a task
- **THEN** metrics manager records start timestamp

#### Scenario: Hook into task completion
- **WHEN** lane completes task execution
- **THEN** metrics manager calculates duration and records metrics

#### Scenario: Hook into task retry
- **WHEN** lane retries a failed task
- **THEN** metrics manager increments retry counter before retry attempt
