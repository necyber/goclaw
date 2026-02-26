## ADDED Requirements

### Requirement: Scheduler dispatches runnable tasks through Lane Manager
The scheduler MUST submit runnable tasks to `lane.Manager` instead of executing task functions directly.

#### Scenario: Runnable task dispatch
- **WHEN** a task becomes runnable in a workflow layer
- **THEN** the scheduler MUST dispatch it via `lane.Manager.Submit`

### Requirement: Layer barrier waits for lane-driven completion
Layer progression MUST be based on completion signals from lane-executed tasks.

#### Scenario: Advance to next layer only after current layer completion
- **WHEN** tasks in a layer are dispatched through lanes
- **THEN** the scheduler MUST wait until all tasks in that layer reach terminal state before starting the next layer

### Requirement: Lane submission failure handling
The scheduler MUST handle lane submission failures as deterministic workflow/task failures.

#### Scenario: Lane submit returns error
- **WHEN** `lane.Manager.Submit` fails for a task
- **THEN** the task MUST be marked failed with submit error context and workflow failure policy MUST be applied

### Requirement: Backpressure and cancellation behavior
Lane backpressure and context cancellation MUST be reflected in scheduler control flow.

#### Scenario: Backpressure delays submission
- **WHEN** lane backpressure delays task admission
- **THEN** the scheduler MUST treat the task as non-running until lane admission succeeds

#### Scenario: Workflow context cancelled during scheduling
- **WHEN** workflow context is cancelled while tasks are pending lane admission
- **THEN** the scheduler MUST stop scheduling new tasks and mark unscheduled runnable tasks according to cancellation policy

