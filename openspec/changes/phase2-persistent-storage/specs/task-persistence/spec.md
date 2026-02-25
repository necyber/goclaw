## ADDED Requirements

### Requirement: Task state is persisted on status change
The system SHALL persist task state whenever task status changes.

#### Scenario: Persist task start
- **WHEN** task execution begins
- **THEN** system updates task status to "running" and persists started timestamp

#### Scenario: Persist task completion
- **WHEN** task completes successfully
- **THEN** system updates task status to "completed" and persists completed timestamp

#### Scenario: Persist task failure
- **WHEN** task execution fails
- **THEN** system updates task status to "failed" and persists error message

### Requirement: Task results are persisted
The system SHALL persist task execution results for later retrieval.

#### Scenario: Persist task output
- **WHEN** task completes with result data
- **THEN** system persists result data linked to task

#### Scenario: Retrieve task result
- **WHEN** API requests task result by workflow ID and task ID
- **THEN** system returns persisted result from storage

#### Scenario: Handle large results
- **WHEN** task result exceeds size limit
- **THEN** system truncates or references external storage

### Requirement: Task errors are persisted
The system SHALL persist detailed error information when tasks fail.

#### Scenario: Persist error message
- **WHEN** task fails with error
- **THEN** system persists error message and error type

#### Scenario: Persist stack trace
- **WHEN** task panics or crashes
- **THEN** system persists stack trace for debugging

#### Scenario: Retrieve error details
- **WHEN** querying failed task
- **THEN** system returns complete error information

### Requirement: Task state is queryable
The system SHALL allow querying task state by workflow and task ID.

#### Scenario: Get task by ID
- **WHEN** API requests task status
- **THEN** system retrieves task state from storage

#### Scenario: List all tasks for workflow
- **WHEN** API requests all tasks for a workflow
- **THEN** system returns all task states from storage

#### Scenario: Filter tasks by status
- **WHEN** querying tasks with status filter
- **THEN** system returns only tasks matching the status

### Requirement: Task state is linked to workflow
The system SHALL maintain referential integrity between tasks and workflows.

#### Scenario: Task belongs to workflow
- **WHEN** task is persisted
- **THEN** system links task to parent workflow ID

#### Scenario: Delete workflow cascades to tasks
- **WHEN** workflow is deleted
- **THEN** system deletes all associated tasks

#### Scenario: Orphaned task detection
- **WHEN** task exists without parent workflow
- **THEN** system detects and reports orphaned task
