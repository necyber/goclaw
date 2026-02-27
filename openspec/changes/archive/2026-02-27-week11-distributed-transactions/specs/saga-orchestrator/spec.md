## ADDED Requirements

### Requirement: Saga definition

The system SHALL support declarative Saga definitions with ordered steps, each containing an action and an optional compensation.

#### Scenario: Define Saga with steps
- **WHEN** a Saga is defined with steps "reserve-inventory", "charge-payment", "ship-order"
- **THEN** the system creates a valid Saga definition with three steps in dependency order

#### Scenario: Define step with action and compensation
- **WHEN** a step is defined with both action and compensation functions
- **THEN** the step stores both functions for forward execution and backward compensation

#### Scenario: Define step without compensation
- **WHEN** a step is defined with action only (no compensation)
- **THEN** the step is valid and skipped during compensation phase

#### Scenario: Define step with dependencies
- **WHEN** a step declares dependencies on other steps
- **THEN** the system validates dependencies exist and builds execution order

### Requirement: Saga state machine

The system SHALL manage Saga lifecycle through a state machine: Created → Running → Completed | Compensating → Compensated | CompensationFailed.

#### Scenario: Transition from Created to Running
- **WHEN** a Saga execution begins
- **THEN** the state transitions from Created to Running

#### Scenario: Transition from Running to Completed
- **WHEN** all steps complete successfully
- **THEN** the state transitions from Running to Completed

#### Scenario: Transition from Running to Compensating
- **WHEN** a step fails during forward execution
- **THEN** the state transitions from Running to Compensating

#### Scenario: Transition from Compensating to Compensated
- **WHEN** all compensation operations complete successfully
- **THEN** the state transitions from Compensating to Compensated

#### Scenario: Transition from Compensating to CompensationFailed
- **WHEN** a compensation operation fails after all retries
- **THEN** the state transitions from Compensating to CompensationFailed

#### Scenario: Invalid state transition
- **WHEN** an invalid state transition is attempted (e.g., Completed → Running)
- **THEN** the system returns a state transition error

### Requirement: Forward execution

The system SHALL execute Saga steps in dependency order (topological order of the step DAG).

#### Scenario: Execute steps sequentially
- **WHEN** steps A → B → C are defined with linear dependencies
- **THEN** the system executes A, then B, then C in order

#### Scenario: Execute steps in parallel
- **WHEN** steps B and C both depend only on A
- **THEN** the system executes B and C in parallel after A completes

#### Scenario: Step execution with context
- **WHEN** a step is executed
- **THEN** the step receives a context with Saga ID, step results from previous steps, and cancellation support

### Requirement: Step result passing

The system SHALL pass results from completed steps to dependent steps.

#### Scenario: Access previous step result
- **WHEN** step B depends on step A and A completed with result data
- **THEN** step B can access A's result via the Saga context

#### Scenario: Access multiple step results
- **WHEN** step C depends on steps A and B
- **THEN** step C can access both A's and B's results

### Requirement: Saga timeout

The system SHALL support configurable timeout for the entire Saga execution.

#### Scenario: Saga completes within timeout
- **WHEN** a Saga with 30s timeout completes in 10s
- **THEN** the Saga transitions to Completed state

#### Scenario: Saga exceeds timeout
- **WHEN** a Saga with 30s timeout does not complete in 30s
- **THEN** the system cancels running steps and triggers compensation

### Requirement: Step timeout

The system SHALL support configurable timeout per step.

#### Scenario: Step completes within timeout
- **WHEN** a step with 5s timeout completes in 2s
- **THEN** the step is marked as completed

#### Scenario: Step exceeds timeout
- **WHEN** a step with 5s timeout does not complete in 5s
- **THEN** the step is marked as failed and triggers Saga compensation

### Requirement: Saga instance management

The system SHALL create and track Saga instances with unique IDs.

#### Scenario: Create Saga instance
- **WHEN** a Saga definition is submitted for execution
- **THEN** the system creates a Saga instance with a unique ID and initial state Created

#### Scenario: Query Saga instance
- **WHEN** a Saga instance ID is queried
- **THEN** the system returns the current state, completed steps, and step results

#### Scenario: List Saga instances
- **WHEN** listing Saga instances with optional state filter
- **THEN** the system returns matching instances with pagination support

### Requirement: Concurrent Saga limit

The system SHALL support configurable maximum concurrent Saga executions.

#### Scenario: Within concurrent limit
- **WHEN** a new Saga is submitted and current running count is below the limit
- **THEN** the Saga begins execution immediately

#### Scenario: Exceeds concurrent limit
- **WHEN** a new Saga is submitted and current running count equals the limit
- **THEN** the Saga is queued and begins when a slot becomes available

### Requirement: Saga metrics

The system SHALL expose Prometheus metrics for Saga operations.

#### Scenario: Track Saga completions
- **WHEN** a Saga completes (success or compensation)
- **THEN** the system increments `saga_executions_total` with status label

#### Scenario: Track Saga duration
- **WHEN** a Saga completes
- **THEN** the system records `saga_duration_seconds` histogram

#### Scenario: Track active Sagas
- **WHEN** Sagas are running
- **THEN** the system updates `saga_active_count` gauge
