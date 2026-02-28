## ADDED Requirements

### Requirement: Task cancellation terminal state conformance
Engine runtime SHALL mark task executions as `cancelled` when execution context is cancelled or deadline is exceeded, and SHALL avoid reporting such cases as generic task failures.

#### Scenario: Context cancelled during task execution
- **WHEN** a running task observes `context.Canceled`
- **THEN** the task terminal state is recorded as `cancelled`
- **AND** workflow status reflects cancellation semantics

#### Scenario: Deadline exceeded during task execution
- **WHEN** a task exceeds its execution context deadline
- **THEN** the task terminal state is recorded as `cancelled`
- **AND** timeout outcome is observable in returned error context

### Requirement: Per-task timeout boundary enforcement
Task runner SHALL enforce per-task timeout boundaries when task timeout is configured and SHALL propagate timeout cancellation to task execution.

#### Scenario: Configured task timeout is exceeded
- **WHEN** a task has `Timeout > 0` and execution exceeds that duration
- **THEN** task runner terminates task execution via context deadline
- **AND** terminal state transitions to `cancelled`

### Requirement: Scheduler layer fail-fast determinism
Scheduler SHALL preserve layer-by-layer barrier behavior and SHALL stop subsequent layer scheduling when an unrecoverable task error occurs.

#### Scenario: Unrecoverable task failure in a layer
- **WHEN** any task in current layer ends with unrecoverable error
- **THEN** scheduler returns error for workflow execution
- **AND** later layers are not dispatched

### Requirement: CLI signal-triggered graceful shutdown
CLI bootstrap SHALL handle `SIGINT` and `SIGTERM` by triggering controlled runtime shutdown for HTTP/gRPC/engine components.

#### Scenario: Process receives SIGTERM
- **WHEN** running process receives `SIGTERM`
- **THEN** shutdown path invokes graceful stop sequence
- **AND** process exits without abrupt termination of active components
