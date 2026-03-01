## ADDED Requirements

### Requirement: Workflow HTTP payload contract is week5-canonical
Workflow management endpoints SHALL expose the Week 5 canonical payload contract for workflow resources.

#### Scenario: Submit workflow returns canonical identifier field
- **WHEN** a client calls `POST /api/v1/workflows` with a valid workflow payload
- **THEN** the response MUST include `workflow_id`, `status`, and `created_at`

#### Scenario: Query workflow returns canonical workflow fields
- **WHEN** a client calls `GET /api/v1/workflows/{workflow_id}` for an existing workflow
- **THEN** the response MUST use `workflow_id` as the workflow identifier field
- **AND** the response MUST include task status entries for known tasks

#### Scenario: List workflows uses canonical item shape
- **WHEN** a client calls `GET /api/v1/workflows`
- **THEN** each workflow item in `workflows` MUST include `workflow_id`, `name`, `status`, and `created_at`

### Requirement: Workflow submit request compatibility is explicit
Workflow submit handlers SHALL support the Week 5 dependency field naming and a compatibility alias for currently deployed clients.

#### Scenario: Week5 dependency field is accepted
- **WHEN** a client submits tasks using `dependencies`
- **THEN** the workflow MUST be accepted and dependencies MUST be applied to runtime planning

#### Scenario: Compatibility alias is accepted
- **WHEN** a client submits tasks using `depends_on`
- **THEN** the workflow MUST be accepted and mapped to canonical dependency semantics

### Requirement: Task result endpoint enforces terminal-state semantics
Task result retrieval SHALL return conflict semantics for non-terminal task states.

#### Scenario: Non-terminal task returns conflict
- **WHEN** `GET /api/v1/workflows/{workflow_id}/tasks/{task_id}/result` is called for a pending, scheduled, or running task
- **THEN** the endpoint MUST return HTTP `409`

#### Scenario: Terminal task returns result payload
- **WHEN** `GET /api/v1/workflows/{workflow_id}/tasks/{task_id}/result` is called for a completed, failed, or cancelled task
- **THEN** the endpoint MUST return HTTP `200` with terminal status and persisted result/error fields

### Requirement: List pagination defaults and bounds are deterministic
Workflow list pagination SHALL enforce Week 5 defaults and bounds.

#### Scenario: Default list limit
- **WHEN** `GET /api/v1/workflows` is called without `limit`
- **THEN** `limit` MUST default to `50`

#### Scenario: Maximum list limit
- **WHEN** `GET /api/v1/workflows` is called with `limit` greater than `100`
- **THEN** the API MUST cap effective limit at `100`

#### Scenario: Invalid pagination is rejected
- **WHEN** `GET /api/v1/workflows` is called with non-numeric or negative pagination values
- **THEN** the API MUST return HTTP `400`
