# execution-pipeline Specification

## Purpose
TBD - synced from changes week13-execution-pipeline, week14-cluster-event-bus.

## Requirements

### Requirement: Workflow submission pipeline lifecycle
The runtime MUST process workflow submission through a two-phase lifecycle:
- submission phase: `accepted -> persisted(pending)`
- execution phase: `triggered -> scheduled -> running -> terminal`

#### Scenario: Submission enters persisted pending state before acceptance
- **WHEN** a workflow submission request is accepted by runtime
- **THEN** the workflow state MUST be persisted as `pending` before returning a successful submission response

#### Scenario: Accepted workflow proceeds to execution lifecycle
- **WHEN** a persisted `pending` workflow is triggered for execution
- **THEN** the workflow state MUST transition through `scheduled` and `running` before any terminal state is emitted

### Requirement: Workflow execution triggering policy
Workflow submission MUST define explicit trigger behavior based on executable function availability.

#### Scenario: Auto-trigger execution when TaskFns are provided
- **WHEN** a workflow is submitted with executable `TaskFns` provided
- **THEN** runtime MUST trigger workflow execution automatically after successful submission persistence

#### Scenario: Keep workflow pending when TaskFns are not provided
- **WHEN** a workflow is submitted without executable `TaskFns`
- **THEN** runtime MUST persist the workflow in `pending` state for later execution trigger

### Requirement: Workflow terminal state invariants
The runtime MUST enforce valid terminal state transitions for workflow lifecycle.

#### Scenario: Successful workflow completion
- **WHEN** all executable tasks finish successfully
- **THEN** the workflow state MUST transition to `completed` and persist `completed_at`

#### Scenario: Workflow execution failure
- **WHEN** any required task fails without recoverable continuation
- **THEN** the workflow state MUST transition to `failed` and persist terminal error details

#### Scenario: Workflow cancellation
- **WHEN** cancellation is requested for a non-terminal workflow
- **THEN** the workflow state MUST transition to `cancelled` and persist cancellation metadata

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

### Requirement: Timestamp field naming contract
Persisted and API-facing task timestamp fields MUST use snake_case naming.

#### Scenario: Timestamp field naming is consistent
- **WHEN** task timestamps are persisted or returned by APIs
- **THEN** field names MUST be `started_at` and `completed_at`
- **AND** internal language-specific field names MUST map to these canonical names without semantic change

### Requirement: Cancellation and timeout precedence
The runtime MUST define deterministic precedence for cancellation and timeout outcomes.

#### Scenario: Context cancellation before task terminal outcome
- **WHEN** a task context is cancelled before success/failure is committed
- **THEN** the task terminal outcome MUST be recorded as cancellation-derived state according to runtime policy

#### Scenario: Task timeout during execution
- **WHEN** a task exceeds configured timeout while running
- **THEN** the task terminal outcome MUST be persisted as timeout-derived failure or cancellation according to runtime policy
