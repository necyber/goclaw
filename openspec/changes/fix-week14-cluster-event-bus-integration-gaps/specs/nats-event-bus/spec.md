## MODIFIED Requirements

### Requirement: Canonical lifecycle event publication
The runtime MUST publish workflow and task lifecycle events to NATS in distributed mode from persisted lifecycle transition hooks.

#### Scenario: Publish workflow lifecycle event
- **WHEN** a workflow lifecycle transition is committed
- **THEN** runtime MUST publish a workflow event to canonical NATS subject namespace
- **AND** publication MUST be invoked from the runtime transition publication path instead of test-only wiring

#### Scenario: Publish task lifecycle event
- **WHEN** a task lifecycle transition is committed
- **THEN** runtime MUST publish a task event to canonical NATS subject namespace
- **AND** publication MUST include canonical event identity metadata required for downstream deduplication

### Requirement: NATS outage degraded behavior
Runtime behavior MUST be deterministic when NATS is unavailable.

#### Scenario: Bus outage during execution
- **WHEN** NATS connectivity is unavailable
- **THEN** runtime MUST continue local execution under degraded-mode policy and record bus outage telemetry
- **AND** runtime MUST NOT block transition persistence waiting for NATS recovery

#### Scenario: Bus recovery
- **WHEN** NATS connectivity is restored
- **THEN** runtime MUST resume canonical publication and clear degraded-mode indicators
