# batch-operations Specification

## Purpose
Migrated from legacy OpenSpec format while preserving existing requirement and scenario content.

## Requirements

### Requirement: Batch workflow submission
The system SHALL provide batch workflow submission to create multiple workflows in single RPC.

#### Scenario: Submit multiple workflows
- **WHEN** SubmitWorkflows is called with list of workflow definitions
- **THEN** server MUST create all workflows and return list of workflow IDs

#### Scenario: Partial success handling
- **WHEN** some workflows fail validation during batch submission
- **THEN** server MUST return results with success/failure status for each workflow

#### Scenario: Atomic batch submission
- **WHEN** atomic flag is set in batch request
- **THEN** server MUST create all workflows or none if any validation fails

#### Scenario: Batch size limit
- **WHEN** batch exceeds maximum size
- **THEN** server MUST return InvalidArgument status with max batch size

### Requirement: Batch status queries
The system SHALL provide batch status queries to retrieve multiple workflow statuses in single RPC.

#### Scenario: Get multiple workflow statuses
- **WHEN** GetWorkflowStatuses is called with list of workflow IDs
- **THEN** server MUST return status for each workflow in single response

#### Scenario: Missing workflow handling
- **WHEN** some workflow IDs do not exist
- **THEN** server MUST return NotFound status for missing workflows and valid status for existing ones

#### Scenario: Status query limit
- **WHEN** status query exceeds maximum IDs
- **THEN** server MUST return InvalidArgument status with max query size

#### Scenario: Efficient status retrieval
- **WHEN** retrieving multiple statuses
- **THEN** server MUST fetch statuses in parallel to minimize latency

### Requirement: Batch workflow cancellation
The system SHALL provide batch cancellation to cancel multiple workflows in single RPC.

#### Scenario: Cancel multiple workflows
- **WHEN** CancelWorkflows is called with list of workflow IDs
- **THEN** server MUST cancel all workflows and return cancellation results

#### Scenario: Partial cancellation
- **WHEN** some workflows are already completed
- **THEN** server MUST return results indicating which workflows were cancelled and which were already terminal

#### Scenario: Force cancellation
- **WHEN** force flag is set in batch cancellation
- **THEN** server MUST immediately terminate running tasks without waiting for graceful shutdown

#### Scenario: Cancellation timeout
- **WHEN** cancellation takes longer than timeout
- **THEN** server MUST return partial results with timeout status

### Requirement: Batch task result retrieval
The system SHALL provide batch task result retrieval to get multiple task results in single RPC.

#### Scenario: Get multiple task results
- **WHEN** GetTaskResults is called with workflow ID and task ID list
- **THEN** server MUST return results for all requested tasks

#### Scenario: Incomplete task handling
- **WHEN** some tasks are not yet completed
- **THEN** server MUST return current status for incomplete tasks and results for completed ones

#### Scenario: Result size limit
- **WHEN** task results exceed response size limit
- **THEN** server MUST return truncated results with continuation token

### Requirement: Batch operation performance
The system SHALL optimize batch operations for high throughput.

#### Scenario: Parallel processing
- **WHEN** processing batch operations
- **THEN** server MUST process items in parallel using worker pool

#### Scenario: Connection pooling
- **WHEN** batch operations access storage
- **THEN** server MUST use connection pooling to avoid connection exhaustion

#### Scenario: Batch timeout
- **WHEN** batch operation exceeds deadline
- **THEN** server MUST return partial results with DeadlineExceeded status

### Requirement: Batch operation idempotency
The system SHALL support idempotent batch operations using request IDs.

#### Scenario: Duplicate batch submission
- **WHEN** batch submission is retried with same request ID
- **THEN** server MUST return cached results without creating duplicate workflows

#### Scenario: Idempotency key validation
- **WHEN** batch request includes idempotency key
- **THEN** server MUST validate key format and reject invalid keys

#### Scenario: Idempotency cache expiration
- **WHEN** idempotency cache entry expires
- **THEN** server MUST treat retry as new request

### Requirement: Batch operation ordering
The system SHALL preserve ordering guarantees for batch operations when requested.

#### Scenario: Sequential batch submission
- **WHEN** ordered flag is set in batch submission
- **THEN** server MUST create workflows in request order

#### Scenario: Unordered batch submission
- **WHEN** ordered flag is not set
- **THEN** server MUST process workflows in parallel for better performance

#### Scenario: Dependency-aware batching
- **WHEN** workflows in batch have dependencies
- **THEN** server MUST respect dependency order during creation

### Requirement: Batch operation error handling
The system SHALL provide detailed error information for batch operation failures.

#### Scenario: Per-item error details
- **WHEN** batch operation fails for some items
- **THEN** response MUST include error code and message for each failed item

#### Scenario: Error aggregation
- **WHEN** multiple items fail with same error
- **THEN** response MUST aggregate errors to reduce payload size

#### Scenario: Rollback on failure
- **WHEN** atomic batch fails
- **THEN** server MUST rollback all changes and return detailed failure reason

### Requirement: Batch operation pagination
The system SHALL support pagination for large batch responses.

#### Scenario: Paginated batch results
- **WHEN** batch response exceeds page size
- **THEN** server MUST return first page with continuation token

#### Scenario: Continuation token
- **WHEN** client requests next page
- **THEN** server MUST return next batch of results using continuation token

#### Scenario: Page size configuration
- **WHEN** client specifies page size
- **THEN** server MUST respect page size up to maximum limit

