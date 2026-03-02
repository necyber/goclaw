## MODIFIED Requirements

### Requirement: Active workflow count metrics
The metrics system SHALL track the current number of active workflows by lifecycle status, including pending and running states with balanced transitions.

#### Scenario: Increment active workflow count
- **WHEN** workflow transitions to running state
- **THEN** system increments workflow_active_count gauge with status="running"

#### Scenario: Decrement active workflow count
- **WHEN** workflow completes or fails
- **THEN** system decrements workflow_active_count gauge for previous status

#### Scenario: Track pending workflows
- **WHEN** workflow is submitted but not yet running
- **THEN** system reflects count in workflow_active_count gauge with status="pending"

#### Scenario: Pending gauge decrements on run transition
- **WHEN** workflow transitions from pending to running
- **THEN** system decrements workflow_active_count gauge with status="pending" before or atomically with running increment

#### Scenario: Pending gauge decrements on pre-run terminal failure
- **WHEN** workflow terminates from pending state without entering running
- **THEN** system decrements workflow_active_count gauge with status="pending"

### Requirement: Workflow metrics integration
The metrics system SHALL integrate with engine workflow lifecycle hooks and keep submission, active-count, and duration metrics transition-consistent.

#### Scenario: Hook into workflow submission
- **WHEN** engine SubmitWorkflowRequest method is called
- **THEN** metrics are recorded before returning to caller

#### Scenario: Hook into workflow execution
- **WHEN** engine starts workflow execution
- **THEN** metrics manager is notified to update active count

#### Scenario: Hook into workflow completion
- **WHEN** engine completes workflow execution
- **THEN** metrics manager records duration and updates counters atomically
