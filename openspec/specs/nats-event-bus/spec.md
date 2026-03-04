# nats-event-bus Specification

## Purpose
TBD - synced from changes week13-execution-pipeline, week14-cluster-event-bus.

## Requirements

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

### Requirement: At-least-once publish intent
Event publication MUST provide at-least-once delivery intent with retry policy.

#### Scenario: Transient publish failure
- **WHEN** NATS publish fails due to transient transport error
- **THEN** runtime MUST retry publication according to configured backoff policy

#### Scenario: Duplicate delivery tolerance
- **WHEN** a retried event is published more than once
- **THEN** event consumers MUST be able to deduplicate by event identifier

### Requirement: NATS outage degraded behavior
Runtime behavior MUST be deterministic when NATS is unavailable.

#### Scenario: Bus outage during execution
- **WHEN** NATS connectivity is unavailable
- **THEN** runtime MUST continue local execution under degraded-mode policy and record bus outage telemetry
- **AND** runtime MUST NOT block transition persistence waiting for NATS recovery

#### Scenario: Bus recovery
- **WHEN** NATS connectivity is restored
- **THEN** runtime MUST resume canonical publication and clear degraded-mode indicators
