# workflow-runtime-api Specification

## Purpose
TBD - synced from changes week13-execution-pipeline, week14-cluster-event-bus.

## Requirements

### Requirement: Submission mode semantics are explicit
The runtime API MUST define explicit synchronous and asynchronous submission modes.

#### Scenario: Asynchronous submit response
- **WHEN** a workflow is submitted in asynchronous mode and accepted
- **THEN** the API MUST return once workflow state is persisted as `pending`, without waiting for execution admission or terminal workflow completion

#### Scenario: Asynchronous submit flag in request payload
- **WHEN** workflow is submitted with `async: true`
- **THEN** the runtime MUST treat submission as asynchronous mode and return immediately after `pending` persistence semantics are satisfied

#### Scenario: Synchronous submit response
- **WHEN** a workflow is submitted in synchronous mode with executable `TaskFns`
- **THEN** the API MUST block until workflow terminal state or caller context cancellation/timeout

#### Scenario: Synchronous submit without executable task functions
- **WHEN** a workflow is submitted in synchronous mode without executable `TaskFns`
- **THEN** the runtime MUST persist the workflow in `pending` state for later execution trigger
- **AND** the API MUST return a non-terminal pending response without blocking for terminal completion

### Requirement: Status query reflects persisted runtime state
Workflow and task query APIs MUST return persisted state as the single source of truth.

#### Scenario: Query running workflow
- **WHEN** a workflow query is issued during active execution
- **THEN** returned workflow and task states MUST match persisted runtime state and timestamps

#### Scenario: Query terminal workflow
- **WHEN** a workflow query is issued after terminal completion
- **THEN** response MUST include terminal workflow state and task terminal details

#### Scenario: Poll workflow status via HTTP endpoint
- **WHEN** a client polls `GET /api/v1/workflows/{id}` after async submission
- **THEN** the endpoint MUST return the persisted latest workflow status and task status details

### Requirement: Cancellation semantics are deterministic
Cancellation APIs MUST define behavior for pending, running, and terminal workflows.

#### Scenario: Cancel pending workflow
- **WHEN** cancellation is requested for a pending workflow
- **THEN** workflow MUST transition to `cancelled` without starting new task execution

#### Scenario: Cancel running workflow
- **WHEN** cancellation is requested for a running workflow
- **THEN** runtime MUST propagate cancellation to in-flight tasks and persist cancellation-derived terminal states

#### Scenario: Cancel propagates context cancellation signal to running tasks
- **WHEN** `CancelWorkflow` is called for a running workflow
- **THEN** runtime MUST cancel the execution context of all currently running tasks
- **AND** running tasks MUST be able to observe context cancellation signal through their execution context

#### Scenario: Running tasks stop gracefully within cancellation timeout
- **WHEN** running tasks receive cancellation signal from workflow cancel operation
- **THEN** runtime MUST enforce graceful-stop behavior within configured cancellation timeout
- **AND** tasks exceeding graceful timeout MUST transition according to timeout/cancellation terminal policy

#### Scenario: Cancel terminal workflow
- **WHEN** cancellation is requested for a terminal workflow
- **THEN** API MUST return a conflict-style error indicating terminal immutability

### Requirement: Task result endpoint terminal behavior
Task result retrieval MUST provide deterministic behavior for terminal and non-terminal tasks.

#### Scenario: Get result for completed task
- **WHEN** task result is requested for a completed task
- **THEN** API MUST return persisted result payload

#### Scenario: Get result for non-terminal task
- **WHEN** task result is requested for a task that is pending, scheduled, or running
- **THEN** API MUST return non-terminal status without fabricating result payload
