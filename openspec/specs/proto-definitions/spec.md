# proto-definitions Specification

## Purpose
Migrated from legacy OpenSpec format while preserving existing requirement and scenario content.

## Requirements

### Requirement: Protocol Buffer service definitions
The system SHALL define Protocol Buffer service definitions for workflow management, task operations, and admin functions.

#### Scenario: Workflow service definition
- **WHEN** generating gRPC code from proto files
- **THEN** WorkflowService MUST include SubmitWorkflow, ListWorkflows, GetWorkflowStatus, CancelWorkflow, and GetTaskResult RPCs

#### Scenario: Streaming service definition
- **WHEN** generating gRPC code from proto files
- **THEN** StreamingService MUST include WatchWorkflow (server streaming) and WatchTasks (server streaming) RPCs

#### Scenario: Batch service definition
- **WHEN** generating gRPC code from proto files
- **THEN** BatchService MUST include SubmitWorkflows, GetWorkflowStatuses, and CancelWorkflows RPCs

#### Scenario: Admin service definition
- **WHEN** generating gRPC code from proto files
- **THEN** AdminService MUST include GetEngineStatus, UpdateConfig, and ManageCluster RPCs

### Requirement: Message type definitions
The system SHALL define Protocol Buffer message types for all request and response payloads.

#### Scenario: Workflow message types
- **WHEN** defining workflow-related messages
- **THEN** messages MUST include WorkflowRequest, WorkflowResponse, WorkflowStatus, TaskDefinition, and TaskResult

#### Scenario: Pagination support
- **WHEN** defining list operation messages
- **THEN** messages MUST include page_size, page_token fields for pagination

#### Scenario: Error handling
- **WHEN** defining response messages
- **THEN** messages MUST include error field with code, message, and details

### Requirement: Proto file organization
The system SHALL organize proto files by service domain with proper package namespacing.

#### Scenario: File structure
- **WHEN** organizing proto files
- **THEN** files MUST be organized as api/proto/goclaw/v1/workflow.proto, streaming.proto, batch.proto, admin.proto

#### Scenario: Package naming
- **WHEN** defining proto packages
- **THEN** package MUST be named goclaw.v1 for version 1 API

#### Scenario: Go package option
- **WHEN** generating Go code
- **THEN** proto files MUST specify go_package option as "github.com/goclaw/goclaw/pkg/grpc/pb/v1"

### Requirement: Field naming conventions
The system SHALL use snake_case for proto field names following Protocol Buffer style guide.

#### Scenario: Field naming
- **WHEN** defining message fields
- **THEN** field names MUST use snake_case (e.g., workflow_id, created_at, task_result)

#### Scenario: Enum naming
- **WHEN** defining enum types
- **THEN** enum values MUST use UPPER_SNAKE_CASE with type prefix (e.g., WORKFLOW_STATUS_PENDING)

### Requirement: Backward compatibility
The system SHALL maintain backward compatibility by using field numbers and reserved fields.

#### Scenario: Field number assignment
- **WHEN** adding new fields
- **THEN** new fields MUST use previously unused field numbers

#### Scenario: Deprecated fields
- **WHEN** removing fields
- **THEN** field numbers MUST be marked as reserved to prevent reuse

#### Scenario: Version management
- **WHEN** making breaking changes
- **THEN** changes MUST be introduced in a new package version (e.g., goclaw.v2)

