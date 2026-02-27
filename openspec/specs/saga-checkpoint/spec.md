# saga-checkpoint Specification

## Purpose
TBD - created by archiving change week11-distributed-transactions. Update Purpose after archive.
## Requirements
### Requirement: WAL entry persistence

The system SHALL write a WAL entry to Badger for every Saga state change.

#### Scenario: Write WAL on step start
- **WHEN** a Saga step begins execution
- **THEN** the system writes a WAL entry with type StepStarted before executing the step

#### Scenario: Write WAL on step completion
- **WHEN** a Saga step completes successfully
- **THEN** the system writes a WAL entry with type StepCompleted and the step result

#### Scenario: Write WAL on step failure
- **WHEN** a Saga step fails
- **THEN** the system writes a WAL entry with type StepFailed and the error details

#### Scenario: Write WAL on compensation
- **WHEN** a compensation operation starts or completes
- **THEN** the system writes a WAL entry with the compensation event type

### Requirement: Checkpoint creation

The system SHALL create a checkpoint after each step completion containing the full Saga state.

#### Scenario: Create checkpoint after step
- **WHEN** step A completes successfully
- **THEN** the system writes a checkpoint with completed steps ["A"] and A's result

#### Scenario: Checkpoint contains all completed results
- **WHEN** steps A and B have completed
- **THEN** the checkpoint contains completed steps ["A", "B"] and both results

### Requirement: Recovery from checkpoint

The system SHALL recover incomplete Sagas from their last checkpoint on startup.

#### Scenario: Recover running Saga
- **WHEN** the system starts and finds a Saga in Running state with checkpoint
- **THEN** the system resumes execution from the next uncompleted step

#### Scenario: Recover compensating Saga
- **WHEN** the system starts and finds a Saga in Compensating state
- **THEN** the system resumes compensation from the next uncompensated step

#### Scenario: No incomplete Sagas
- **WHEN** the system starts and all Sagas are in terminal states
- **THEN** no recovery is performed

### Requirement: WAL key format

The system SHALL use key format "wal:{sagaID}:{sequence}" for WAL entries in Badger.

#### Scenario: Sequential WAL entries
- **WHEN** multiple WAL entries are written for the same Saga
- **THEN** each entry has a monotonically increasing sequence number

#### Scenario: Scan WAL by Saga ID
- **WHEN** recovering a Saga
- **THEN** the system scans all WAL entries with prefix "wal:{sagaID}:" in sequence order

### Requirement: Checkpoint key format

The system SHALL use key format "checkpoint:{sagaID}" for checkpoints in Badger.

#### Scenario: Write checkpoint
- **WHEN** a checkpoint is created
- **THEN** it is stored at key "checkpoint:{sagaID}" overwriting the previous checkpoint

#### Scenario: Read checkpoint
- **WHEN** recovering a Saga
- **THEN** the system reads the latest checkpoint from "checkpoint:{sagaID}"

### Requirement: WAL cleanup

The system SHALL support configurable WAL retention and cleanup.

#### Scenario: Clean WAL for completed Saga
- **WHEN** a Saga reaches a terminal state and retention period expires
- **THEN** the system deletes all WAL entries for that Saga

#### Scenario: Retain WAL within period
- **WHEN** a Saga completed within the retention period
- **THEN** the WAL entries are preserved

#### Scenario: Background cleanup
- **WHEN** the WAL cleanup interval elapses
- **THEN** the system scans for expired WAL entries and deletes them in batch

### Requirement: WAL write performance

The system SHALL write WAL entries with minimal latency impact.

#### Scenario: Synchronous WAL write
- **WHEN** WAL sync mode is "sync"
- **THEN** the WAL entry is flushed to disk before returning (durability guarantee)

#### Scenario: Asynchronous WAL write
- **WHEN** WAL sync mode is "async"
- **THEN** the WAL entry is buffered and flushed periodically (lower latency, risk of loss)

#### Scenario: WAL write latency
- **WHEN** writing a WAL entry in sync mode
- **THEN** the write completes in less than 2ms

### Requirement: Checkpoint serialization

The system SHALL serialize checkpoints to JSON for storage.

#### Scenario: Serialize checkpoint
- **WHEN** a checkpoint is created
- **THEN** it is serialized to JSON with fields: sagaID, state, completedSteps, failedStep, stepResults, lastUpdated

#### Scenario: Deserialize checkpoint
- **WHEN** a checkpoint is loaded for recovery
- **THEN** the JSON is deserialized back to a Checkpoint struct with all fields intact

### Requirement: Recovery idempotency

The system SHALL ensure recovery operations are idempotent.

#### Scenario: Recover already completed step
- **WHEN** recovery replays a step that was already completed (per checkpoint)
- **THEN** the system skips the step and proceeds to the next

#### Scenario: Multiple recovery attempts
- **WHEN** recovery is triggered multiple times for the same Saga
- **THEN** each recovery produces the same result without side effects

