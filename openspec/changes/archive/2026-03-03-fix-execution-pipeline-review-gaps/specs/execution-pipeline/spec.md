## MODIFIED Requirements

### Requirement: Task transition persistence contract
The runtime MUST persist task-level state transitions with timestamps and terminal payload fields.

#### Scenario: Pending to scheduled transition is persisted
- **WHEN** a task transitions from `pending` to `scheduled`
- **THEN** runtime MUST persist the `scheduled` state before task execution begins

#### Scenario: Task starts execution
- **WHEN** a scheduled task starts execution
- **THEN** the task state MUST persist as `running` with `started_at`

#### Scenario: Task completes execution
- **WHEN** a running task succeeds
- **THEN** the task state MUST persist as `completed` with `completed_at` and result payload

#### Scenario: Task fails execution
- **WHEN** a running task fails
- **THEN** the task state MUST persist as `failed` with `completed_at` and error payload

#### Scenario: started_at and completed_at fields are recorded
- **WHEN** task transitions to `running`
- **THEN** runtime MUST record `started_at`
- **AND** when task transitions to `completed` or `failed`, runtime MUST record `completed_at`

#### Scenario: Completed task payload is persisted for terminal result retrieval
- **WHEN** a task reaches `completed`
- **THEN** runtime MUST persist the terminal result payload in task state storage
- **AND** subsequent task-result queries MUST read that persisted payload without synthetic reconstruction

### Requirement: Cancellation and timeout precedence
The runtime MUST define deterministic precedence for cancellation and timeout outcomes.

#### Scenario: Context cancellation before task terminal outcome
- **WHEN** a task context is cancelled before success/failure is committed
- **THEN** the task terminal outcome MUST be recorded as cancellation-derived state according to runtime policy

#### Scenario: Task timeout during execution
- **WHEN** a task exceeds configured timeout while running
- **THEN** the task terminal outcome MUST be persisted as timeout-derived failure or cancellation according to runtime policy

#### Scenario: Graceful cancel timeout expires with in-flight task
- **WHEN** workflow cancellation is requested and a task does not stop within configured graceful cancellation timeout
- **THEN** runtime MUST persist a deterministic timeout-derived terminal outcome for that task
- **AND** the workflow terminal state MUST reflect timeout/cancellation policy without remaining in non-terminal state
