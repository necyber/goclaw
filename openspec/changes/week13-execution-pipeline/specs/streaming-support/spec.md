## ADDED Requirements

### Requirement: Streaming events follow persisted lifecycle transitions
Workflow and task streaming updates MUST be emitted from persisted lifecycle transitions.

#### Scenario: Persisted transition emits stream event
- **WHEN** workflow or task state change is persisted
- **THEN** streaming service MUST emit a corresponding update event that reflects the persisted state

### Requirement: Transition-consistent stream ordering per workflow
Streaming for a single workflow MUST preserve transition order consistency.

#### Scenario: Ordered task transition stream
- **WHEN** a task transitions from `scheduled` to `running` to terminal state
- **THEN** subscribers for that workflow MUST receive updates in the same transition order

### Requirement: Terminal transition stream guarantees
Streaming MUST expose terminal state visibility for workflow and task streams.

#### Scenario: Workflow reaches terminal state
- **WHEN** workflow transitions to `completed`, `failed`, or `cancelled`
- **THEN** streaming service MUST emit terminal update before stream closure or idle state

