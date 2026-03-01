# task-persistence Specification

## Purpose
TBD - created by archiving change fix-week6-persistent-storage-gaps. Update Purpose after archive.
## Requirements
### Requirement: Recovery SHALL synchronize task records with workflow task snapshots
When recovery normalizes in-flight task states, task-level persisted records MUST be updated to match the normalized workflow snapshot.

#### Scenario: Running task reset during recovery
- **WHEN** recovery resets a task from `running` to `pending`
- **THEN** `SaveTask` equivalent persistence MUST be applied so `GetTask` returns `pending`

#### Scenario: Cleared runtime fields
- **WHEN** recovery resets an in-flight task
- **THEN** task runtime fields such as started/completed timestamps and transient errors MUST be cleared in persisted task records

### Requirement: Task result reads SHALL reflect latest persisted terminal state
Task result queries MUST read task persistence that is synchronized with lifecycle transitions and recovery updates.

#### Scenario: Read after recovery normalization
- **WHEN** `GetTaskResult` is called after recovery normalization but before re-execution
- **THEN** the task response MUST reflect normalized non-terminal state and MUST NOT expose stale terminal data

#### Scenario: Read after terminal completion
- **WHEN** task execution reaches terminal state and is persisted
- **THEN** task result queries MUST return the terminal status and associated result/error data from the latest persisted record

