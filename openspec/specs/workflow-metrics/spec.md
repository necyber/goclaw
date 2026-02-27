# workflow-metrics Specification

## Purpose
Migrated from legacy OpenSpec format while preserving existing requirement and scenario content.

## Requirements

### Requirement: Workflow submission metrics
The metrics system SHALL track workflow submission events with status labels.

#### Scenario: Record workflow submission
- **WHEN** engine receives a new workflow submission request
- **THEN** system increments workflow_submissions_total counter with status="pending"

#### Scenario: Track workflow completion
- **WHEN** workflow execution completes successfully
- **THEN** system increments workflow_submissions_total counter with status="completed"

#### Scenario: Track workflow failure
- **WHEN** workflow execution fails
- **THEN** system increments workflow_submissions_total counter with status="failed"

#### Scenario: Track workflow cancellation
- **WHEN** workflow is cancelled by user
- **THEN** system increments workflow_submissions_total counter with status="cancelled"

### Requirement: Workflow duration metrics
The metrics system SHALL measure workflow execution duration from submission to completion.

#### Scenario: Record successful workflow duration
- **WHEN** workflow completes successfully
- **THEN** system records duration in workflow_duration_seconds histogram with status="completed"

#### Scenario: Record failed workflow duration
- **WHEN** workflow fails during execution
- **THEN** system records duration in workflow_duration_seconds histogram with status="failed"

#### Scenario: Duration histogram buckets
- **WHEN** recording workflow duration
- **THEN** system uses buckets [0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300] seconds

### Requirement: Active workflow count metrics
The metrics system SHALL track the current number of active workflows by status.

#### Scenario: Increment active workflow count
- **WHEN** workflow transitions to running state
- **THEN** system increments workflow_active_count gauge with status="running"

#### Scenario: Decrement active workflow count
- **WHEN** workflow completes or fails
- **THEN** system decrements workflow_active_count gauge for previous status

#### Scenario: Track pending workflows
- **WHEN** workflow is submitted but not yet running
- **THEN** system reflects count in workflow_active_count gauge with status="pending"

### Requirement: Workflow metrics integration
The metrics system SHALL integrate with engine workflow lifecycle hooks.

#### Scenario: Hook into workflow submission
- **WHEN** engine SubmitWorkflowRequest method is called
- **THEN** metrics are recorded before returning to caller

#### Scenario: Hook into workflow execution
- **WHEN** engine starts workflow execution
- **THEN** metrics manager is notified to update active count

#### Scenario: Hook into workflow completion
- **WHEN** engine completes workflow execution
- **THEN** metrics manager records duration and updates counters atomically

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

