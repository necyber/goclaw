# saga-compensation Specification

## Purpose
TBD - created by archiving change week11-distributed-transactions. Update Purpose after archive.
## Requirements
### Requirement: Reverse-order compensation execution

The system SHALL execute compensation operations in reverse topological order of completed steps.

#### Scenario: Linear compensation order
- **WHEN** steps A → B → C were executed and C fails
- **THEN** the system compensates B then A (reverse order)

#### Scenario: Parallel step compensation
- **WHEN** steps B and C (both depending on A) were executed in parallel and a later step fails
- **THEN** the system compensates B and C in parallel, then compensates A

#### Scenario: Skip steps without compensation
- **WHEN** a completed step has no compensation function defined
- **THEN** the system skips that step during compensation

### Requirement: Compensation policies

The system SHALL support three compensation policies: AutoCompensate, ManualCompensate, SkipCompensate.

#### Scenario: Auto compensation
- **WHEN** a step fails and the Saga policy is AutoCompensate
- **THEN** the system immediately begins reverse compensation

#### Scenario: Manual compensation
- **WHEN** a step fails and the Saga policy is ManualCompensate
- **THEN** the system marks the Saga as "pending-compensation" and waits for manual trigger

#### Scenario: Skip compensation
- **WHEN** a step is configured with SkipCompensate policy
- **THEN** the system skips compensation for that step even during reverse compensation

### Requirement: Compensation retry

The system SHALL retry failed compensation operations with configurable backoff.

#### Scenario: Retry on transient failure
- **WHEN** a compensation operation fails with a transient error
- **THEN** the system retries up to MaxRetries times with exponential backoff

#### Scenario: Exhaust retries
- **WHEN** a compensation operation fails after all retries
- **THEN** the Saga transitions to CompensationFailed state

#### Scenario: Custom retry configuration
- **WHEN** a Saga is configured with MaxRetries=5, InitialBackoff=100ms, BackoffFactor=2.0
- **THEN** the system retries with delays: 100ms, 200ms, 400ms, 800ms, 1600ms

### Requirement: Compensation idempotency support

The system SHALL provide utilities to help users write idempotent compensation operations.

#### Scenario: Idempotency key check
- **WHEN** a compensation operation is executed with an idempotency key
- **THEN** the system checks if the compensation was already applied and skips if so

#### Scenario: Record compensation execution
- **WHEN** a compensation operation completes successfully
- **THEN** the system records the idempotency key to prevent re-execution

### Requirement: Compensation context

The system SHALL provide compensation operations with the original step's input and result.

#### Scenario: Access original result in compensation
- **WHEN** step A's compensation is executed
- **THEN** the compensation function receives A's original input and result

#### Scenario: Access Saga context in compensation
- **WHEN** a compensation function executes
- **THEN** it has access to the Saga ID, failed step ID, and failure reason

### Requirement: Compensation timeout

The system SHALL support configurable timeout for individual compensation operations.

#### Scenario: Compensation within timeout
- **WHEN** a compensation operation completes within its timeout
- **THEN** the compensation is marked as successful

#### Scenario: Compensation exceeds timeout
- **WHEN** a compensation operation exceeds its timeout
- **THEN** the operation is cancelled and counted as a failed attempt (subject to retry)

### Requirement: Manual compensation trigger

The system SHALL support manually triggering compensation for Sagas in pending-compensation state.

#### Scenario: Trigger manual compensation
- **WHEN** a manual compensation is triggered for a Saga in pending-compensation state
- **THEN** the system begins reverse compensation execution

#### Scenario: Trigger compensation for wrong state
- **WHEN** manual compensation is triggered for a Saga in Completed state
- **THEN** the system returns an invalid-state error

### Requirement: Compensation metrics

The system SHALL expose metrics for compensation operations.

#### Scenario: Track compensation executions
- **WHEN** compensation operations execute
- **THEN** the system increments `saga_compensations_total` with status label (success/failure)

#### Scenario: Track compensation duration
- **WHEN** a compensation phase completes
- **THEN** the system records `saga_compensation_duration_seconds` histogram

#### Scenario: Track compensation retries
- **WHEN** compensation operations are retried
- **THEN** the system increments `saga_compensation_retries_total`

