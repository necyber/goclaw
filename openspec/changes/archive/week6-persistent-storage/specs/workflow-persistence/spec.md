## ADDED Requirements

### Requirement: Workflow state is persisted on creation
The system SHALL persist workflow state immediately when a workflow is submitted.

#### Scenario: Persist new workflow
- **WHEN** user submits workflow via API
- **THEN** system saves workflow state to storage before returning workflow ID

#### Scenario: Persist workflow metadata
- **WHEN** workflow includes name, description, and custom metadata
- **THEN** system persists all metadata fields

#### Scenario: Handle persistence failure on creation
- **WHEN** storage fails during workflow creation
- **THEN** system returns error and does not create workflow

### Requirement: Workflow status updates are persisted
The system SHALL persist workflow status changes as execution progresses.

#### Scenario: Persist status transition to running
- **WHEN** workflow execution starts
- **THEN** system updates status to "running" and persists started timestamp

#### Scenario: Persist status transition to completed
- **WHEN** all workflow tasks complete successfully
- **THEN** system updates status to "completed" and persists completed timestamp

#### Scenario: Persist status transition to failed
- **WHEN** workflow execution fails
- **THEN** system updates status to "failed" and persists error message

### Requirement: Workflow task list is persisted
The system SHALL persist the complete task list and dependencies for each workflow.

#### Scenario: Persist task definitions
- **WHEN** workflow is created with task list
- **THEN** system persists all task IDs, names, types, and dependencies

#### Scenario: Retrieve task list
- **WHEN** querying workflow status
- **THEN** system returns complete task list from storage

### Requirement: Workflow queries use persisted data
The system SHALL serve all workflow queries from persistent storage.

#### Scenario: Get workflow by ID
- **WHEN** API requests workflow status by ID
- **THEN** system retrieves workflow from storage

#### Scenario: List workflows with filter
- **WHEN** API requests workflows filtered by status
- **THEN** system queries storage with filter and returns results

#### Scenario: Pagination uses storage
- **WHEN** API requests workflows with limit and offset
- **THEN** system applies pagination at storage layer

### Requirement: Workflow deletion removes persisted data
The system SHALL remove workflow data from storage when deleted.

#### Scenario: Delete workflow
- **WHEN** workflow is deleted via API or cleanup
- **THEN** system removes workflow and all associated tasks from storage

#### Scenario: Verify deletion
- **WHEN** querying deleted workflow
- **THEN** system returns not found error
