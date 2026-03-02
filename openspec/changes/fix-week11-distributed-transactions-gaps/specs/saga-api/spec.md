## MODIFIED Requirements

### Requirement: Submit Saga via HTTP

The system SHALL provide an HTTP endpoint to submit a Saga for execution and SHALL persist the submitted definition for restart-safe lifecycle operations.

#### Scenario: Submit Saga successfully
- **WHEN** POST /api/v1/sagas is called with a valid Saga definition
- **THEN** the system creates a Saga instance, persists its executable definition snapshot, and returns the Saga ID with status 201

#### Scenario: Submit invalid Saga
- **WHEN** POST /api/v1/sagas is called with an invalid definition (e.g., cyclic dependencies)
- **THEN** the system returns 400 with validation error details

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

### Requirement: Compensate Saga via gRPC

The system SHALL provide a gRPC method to trigger manual compensation using definition data resolved from durable storage.

#### Scenario: gRPC trigger compensation
- **WHEN** CompensateSaga RPC is called for a pending-compensation Saga
- **THEN** the system resolves the Saga definition snapshot, begins compensation, and returns acknowledgment

#### Scenario: gRPC trigger compensation with missing definition
- **WHEN** CompensateSaga RPC is called and no definition snapshot exists for that Saga
- **THEN** the system returns NOT_FOUND or FAILED_PRECONDITION and does not execute compensation
