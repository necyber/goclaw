## MODIFIED Requirements

### Requirement: Checkpoint creation

The system SHALL create and persist checkpoints after each step completion and after each Saga lifecycle state transition that changes recovery behavior.

#### Scenario: Create checkpoint after step
- **WHEN** step A completes successfully
- **THEN** the system writes a checkpoint with completed steps ["A"] and A's result

#### Scenario: Create checkpoint on transition to compensating
- **WHEN** a running Saga transitions to Compensating due to a step failure
- **THEN** the system persists a checkpoint whose state is Compensating and includes failed step metadata

#### Scenario: Create checkpoint on terminal transition
- **WHEN** a Saga transitions to Completed, Compensated, or CompensationFailed
- **THEN** the system persists a checkpoint reflecting that terminal state

### Requirement: Recovery from checkpoint

The system SHALL recover incomplete Sagas from their last checkpoint on startup using the persisted definition snapshot for each Saga.

#### Scenario: Recover running Saga
- **WHEN** the system starts and finds a Saga in Running state with checkpoint
- **THEN** the system resumes execution from the next uncompleted step using the stored definition snapshot

#### Scenario: Recover compensating Saga
- **WHEN** the system starts and finds a Saga in Compensating state
- **THEN** the system resumes compensation from the next uncompensated step using the stored definition snapshot

#### Scenario: Definition snapshot missing during recovery
- **WHEN** an incomplete checkpoint exists but no corresponding definition snapshot can be loaded
- **THEN** the system skips execution, emits explicit recovery failure telemetry, and leaves checkpoint data for operator intervention
