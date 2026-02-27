## ADDED Requirements

### Requirement: Workflow metrics are transition-driven
Workflow metric counters and histograms MUST be recorded from workflow lifecycle transition hooks.

#### Scenario: Pending to running transition updates active metrics
- **WHEN** workflow transitions from `pending` to `running`
- **THEN** active workflow gauges and submission counters MUST be updated from that transition event

#### Scenario: Running to terminal transition records duration
- **WHEN** workflow transitions from `running` to terminal state
- **THEN** duration and terminal status counters MUST be recorded exactly once

### Requirement: Cancellation and timeout outcomes are labeled explicitly
Workflow terminal metrics MUST distinguish cancellation-derived outcomes from other failures.

#### Scenario: User cancellation terminal outcome
- **WHEN** workflow reaches terminal cancellation due to cancel request
- **THEN** metrics MUST label outcome as cancellation-derived terminal status

#### Scenario: Timeout-derived terminal outcome
- **WHEN** workflow reaches terminal status due to timeout policy
- **THEN** metrics MUST label timeout-derived outcome consistently with runtime policy

