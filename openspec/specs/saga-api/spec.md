# saga-api Specification

## Purpose
TBD - created by archiving change week11-distributed-transactions. Update Purpose after archive.
## Requirements
### Requirement: Submit Saga via HTTP

The system SHALL provide an HTTP endpoint to submit a Saga for execution and SHALL persist the submitted definition for restart-safe lifecycle operations.

#### Scenario: Submit Saga successfully
- **WHEN** POST /api/v1/sagas is called with a valid Saga definition
- **THEN** the system creates a Saga instance, persists its executable definition snapshot, and returns the Saga ID with status 201

#### Scenario: Submit invalid Saga
- **WHEN** POST /api/v1/sagas is called with an invalid definition (e.g., cyclic dependencies)
- **THEN** the system returns 400 with validation error details

### Requirement: Query Saga status via HTTP

The system SHALL provide an HTTP endpoint to query Saga instance status.

#### Scenario: Get Saga status
- **WHEN** GET /api/v1/sagas/{id} is called with a valid Saga ID
- **THEN** the system returns the Saga state, completed steps, failed step, and step results

#### Scenario: Get non-existent Saga
- **WHEN** GET /api/v1/sagas/{id} is called with an unknown ID
- **THEN** the system returns 404

### Requirement: List Sagas via HTTP

The system SHALL provide an HTTP endpoint to list Saga instances with filtering and pagination.

#### Scenario: List all Sagas
- **WHEN** GET /api/v1/sagas?limit=20&offset=0 is called
- **THEN** the system returns paginated list of Saga instances

#### Scenario: List Sagas by state
- **WHEN** GET /api/v1/sagas?state=compensating is called
- **THEN** the system returns only Sagas in Compensating state

### Requirement: Trigger compensation via HTTP

The system SHALL provide an HTTP endpoint to manually trigger compensation using definition data resolved from durable storage.

#### Scenario: Trigger compensation
- **WHEN** POST /api/v1/sagas/{id}/compensate is called for a Saga in pending-compensation state
- **THEN** the system resolves the definition snapshot for that Saga, begins compensation, and returns 202 Accepted

#### Scenario: Trigger compensation for invalid state
- **WHEN** POST /api/v1/sagas/{id}/compensate is called for a Saga in Completed state
- **THEN** the system returns 409 Conflict

#### Scenario: Trigger compensation with missing definition
- **WHEN** POST /api/v1/sagas/{id}/compensate is called and no definition snapshot exists for that Saga
- **THEN** the system returns 404/FAILED_PRECONDITION equivalent and does not start compensation

### Requirement: Recover Saga via HTTP

The system SHALL provide an HTTP endpoint to manually trigger recovery for a stuck Saga using checkpoint and durable definition data.

#### Scenario: Recover stuck Saga
- **WHEN** POST /api/v1/sagas/{id}/recover is called for a Saga in Running state
- **THEN** the system loads the latest checkpoint and definition snapshot, attempts recovery, and returns 202 Accepted

#### Scenario: Recover terminal Saga
- **WHEN** POST /api/v1/sagas/{id}/recover is called for a Saga in Completed state
- **THEN** the system returns 409 Conflict (already terminal)

#### Scenario: Recover Saga with missing definition
- **WHEN** POST /api/v1/sagas/{id}/recover is called and a checkpoint exists but definition snapshot is unavailable
- **THEN** the system returns a not-found or precondition error and does not execute recovery steps

### Requirement: Submit Saga via gRPC

The system SHALL provide a gRPC method to submit a Saga for execution.

#### Scenario: gRPC submit Saga
- **WHEN** SubmitSaga RPC is called with a valid Saga definition
- **THEN** the system creates a Saga instance and returns the Saga ID

#### Scenario: gRPC submit invalid Saga
- **WHEN** SubmitSaga RPC is called with an invalid definition
- **THEN** the system returns INVALID_ARGUMENT status with details

### Requirement: Query Saga via gRPC

The system SHALL provide a gRPC method to query Saga status.

#### Scenario: gRPC get Saga status
- **WHEN** GetSagaStatus RPC is called with a valid Saga ID
- **THEN** the system returns the full Saga state

#### Scenario: gRPC get non-existent Saga
- **WHEN** GetSagaStatus RPC is called with an unknown ID
- **THEN** the system returns NOT_FOUND status

### Requirement: Watch Saga via gRPC streaming

The system SHALL provide a gRPC streaming method to watch Saga state changes in real-time.

#### Scenario: Watch Saga events
- **WHEN** WatchSaga server-streaming RPC is called with a Saga ID
- **THEN** the system streams state change events as they occur

#### Scenario: Watch completed Saga
- **WHEN** WatchSaga is called for a Saga that reaches a terminal state
- **THEN** the stream sends the terminal event and closes

### Requirement: Compensate Saga via gRPC

The system SHALL provide a gRPC method to trigger manual compensation using definition data resolved from durable storage.

#### Scenario: gRPC trigger compensation
- **WHEN** CompensateSaga RPC is called for a pending-compensation Saga
- **THEN** the system resolves the Saga definition snapshot, begins compensation, and returns acknowledgment

#### Scenario: gRPC trigger compensation with missing definition
- **WHEN** CompensateSaga RPC is called and no definition snapshot exists for that Saga
- **THEN** the system returns NOT_FOUND or FAILED_PRECONDITION and does not execute compensation

### Requirement: Saga API error codes

The system SHALL return consistent error codes across HTTP and gRPC APIs.

#### Scenario: Saga not found
- **WHEN** a non-existent Saga ID is referenced
- **THEN** HTTP returns 404, gRPC returns NOT_FOUND

#### Scenario: Invalid state transition
- **WHEN** an operation is invalid for the current Saga state
- **THEN** HTTP returns 409, gRPC returns FAILED_PRECONDITION

#### Scenario: Validation error
- **WHEN** a Saga definition fails validation
- **THEN** HTTP returns 400, gRPC returns INVALID_ARGUMENT
