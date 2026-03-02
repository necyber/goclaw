## MODIFIED Requirements

### Requirement: Streaming events follow persisted lifecycle transitions
Workflow and task streaming updates MUST be emitted from persisted lifecycle transitions.

#### Scenario: Persisted transition emits stream event
- **WHEN** workflow or task state change is persisted
- **THEN** streaming service MUST emit a corresponding update event that reflects the persisted state

#### Scenario: Initial workflow stream message reflects persisted state
- **WHEN** a client subscribes to a workflow stream
- **THEN** the first state-bearing message MUST be derived from the latest persisted workflow state
- **AND** streaming service MUST NOT emit a synthetic fixed `pending` state that conflicts with persisted state

### Requirement: Terminal transition stream guarantees
Streaming MUST expose terminal state visibility for workflow and task streams.

#### Scenario: Workflow reaches terminal state
- **WHEN** workflow transitions to `completed`, `failed`, or `cancelled`
- **THEN** streaming service MUST emit terminal update before stream closure or idle state

#### Scenario: Slow consumer during terminal transition
- **WHEN** stream backpressure is present while terminal transition event is being published
- **THEN** streaming service MUST prioritize terminal visibility over intermediate non-terminal updates according to runtime policy
- **AND** if delivery cannot be completed, stream MUST close with explicit backpressure/resource error semantics

## ADDED Requirements

### Requirement: Backpressure handling preserves per-workflow transition correctness
Streaming backpressure handling MUST preserve per-workflow transition correctness guarantees without blocking runtime persistence.

#### Scenario: Intermediate non-terminal updates are shed under pressure
- **WHEN** a subscriber cannot keep up and stream buffer pressure exceeds limits
- **THEN** streaming implementation MAY coalesce or shed intermediate non-terminal updates according to policy
- **AND** emitted updates MUST remain transition-order-consistent for each delivered workflow stream

#### Scenario: Terminal visibility remains deterministic
- **WHEN** update shedding/coalescing occurs under pressure
- **THEN** terminal workflow/task transition visibility MUST still be deterministic to subscribers
- **AND** the client MUST be informed through terminal update delivery or explicit terminal stream error semantics
