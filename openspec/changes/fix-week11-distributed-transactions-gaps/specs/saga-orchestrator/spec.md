## MODIFIED Requirements

### Requirement: Forward execution

The system SHALL execute Saga steps in dependency order (topological order of the step DAG) while enforcing per-definition step concurrency limits.

#### Scenario: Execute steps sequentially
- **WHEN** steps A -> B -> C are defined with linear dependencies
- **THEN** the system executes A, then B, then C in order

#### Scenario: Execute steps in parallel
- **WHEN** steps B and C both depend only on A
- **THEN** the system executes B and C in parallel after A completes

#### Scenario: Respect per-definition max concurrency
- **WHEN** a Saga definition sets `MaxConcurrent=2` and one layer has four runnable steps
- **THEN** the orchestrator runs at most two steps from that Saga concurrently until the layer completes

#### Scenario: Step execution with context
- **WHEN** a step is executed
- **THEN** the step receives a context with Saga ID, step results from previous steps, and cancellation support

### Requirement: Saga instance management

The system SHALL create and track Saga instances with unique IDs and maintain durable linkage to their executable definition snapshots.

#### Scenario: Create Saga instance
- **WHEN** a Saga definition is submitted for execution
- **THEN** the system creates a Saga instance with a unique ID and initial state Created

#### Scenario: Persist definition linkage
- **WHEN** a Saga instance is created
- **THEN** the system stores or references a durable definition snapshot retrievable by Saga ID for later recovery and manual lifecycle operations

#### Scenario: Query Saga instance
- **WHEN** a Saga instance ID is queried
- **THEN** the system returns the current state, completed steps, and step results

#### Scenario: List Saga instances
- **WHEN** listing Saga instances with optional state filter
- **THEN** the system returns matching instances with pagination support
