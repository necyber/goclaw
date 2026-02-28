## ADDED Requirements

### Requirement: Shutdown stop-admission guard
Engine runtime SHALL stop accepting new workflow submissions immediately after shutdown initiation and before queue/lane component closure completes.

#### Scenario: Submit during shutdown transition
- **WHEN** shutdown has started and engine is transitioning to stop
- **THEN** new workflow submission is rejected with lifecycle/not-running semantics
- **AND** no new task dispatch is admitted to scheduling lanes

## MODIFIED Requirements

### Requirement: Task cancellation terminal state conformance
Engine runtime SHALL mark task executions as `cancelled` when execution context is cancelled or deadline is exceeded, SHALL avoid reporting such cases as generic task failures, and SHALL map workflow terminal status to cancellation semantics when terminal task outcomes are cancellation-driven.

#### Scenario: Context cancelled during task execution
- **WHEN** a running task observes `context.Canceled`
- **THEN** the task terminal state is recorded as `cancelled`
- **AND** workflow status reflects cancellation semantics

#### Scenario: Deadline exceeded during task execution
- **WHEN** a task exceeds its execution context deadline
- **THEN** the task terminal state is recorded as `cancelled`
- **AND** timeout outcome is observable in returned error context

### Requirement: Per-task timeout boundary enforcement
Task runner SHALL enforce per-task timeout boundaries when task timeout is configured, SHALL propagate timeout cancellation to task execution, and SHALL treat timeout completion as cancellation even if task logic returns nil after context deadline.

#### Scenario: Configured task timeout is exceeded
- **WHEN** a task has `Timeout > 0` and execution exceeds that duration
- **THEN** task runner terminates task execution via context deadline
- **AND** terminal state transitions to `cancelled`
