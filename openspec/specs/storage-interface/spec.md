# storage-interface Specification

## Purpose
TBD - created by archiving change fix-week6-persistent-storage-gaps. Update Purpose after archive.
## Requirements
### Requirement: Storage reads SHALL provide mutation isolation
Storage implementations MUST return defensive copies for mutable workflow/task data so callers cannot mutate persisted state without explicit save operations.

#### Scenario: Workflow map and slice isolation
- **WHEN** a caller retrieves a workflow and mutates `metadata`, `tasks`, or `task_status`
- **THEN** subsequent reads from storage MUST remain unchanged until `SaveWorkflow` or `SaveTask` is called

#### Scenario: List results isolation
- **WHEN** a caller mutates objects returned from `ListWorkflows` or `ListTasks`
- **THEN** those mutations MUST NOT alter persisted records unless an explicit save operation is executed

### Requirement: Storage updates SHALL keep workflow and task records consistent
When task status is changed as part of lifecycle transitions or recovery normalization, storage MUST persist both workflow-level task snapshots and task-level records to the same logical state.

#### Scenario: Recovery updates running task to pending
- **WHEN** recovery resets a task from `running` to `pending`
- **THEN** both workflow `task_status[task_id]` and the task record returned by `GetTask(workflow_id, task_id)` MUST show `pending`

#### Scenario: Terminal transition sync
- **WHEN** a task transitions to a terminal status
- **THEN** workflow-level task snapshot and task-level persisted record MUST reflect the same status and timestamps

