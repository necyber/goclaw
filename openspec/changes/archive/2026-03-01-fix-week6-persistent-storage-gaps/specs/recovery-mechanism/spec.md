## ADDED Requirements

### Requirement: Startup recovery SHALL resubmit recoverable workflows
Engine startup recovery MUST actively resubmit workflows that remain recoverable after normalization, instead of only resetting persisted status.

#### Scenario: Pending workflow on startup
- **WHEN** startup recovery finds a workflow in `pending`
- **THEN** the engine MUST enqueue or start execution for that workflow after validation

#### Scenario: Running workflow after crash
- **WHEN** startup recovery finds a workflow previously in `running`
- **THEN** the engine MUST normalize in-flight task/workflow state and then resubmit execution

### Requirement: Recovery SHALL be safe and idempotent
Recovery MUST avoid duplicate execution and MUST continue startup even when some workflows fail to recover.

#### Scenario: Workflow already executing
- **WHEN** recovery encounters a workflow that is already active in the execution registry
- **THEN** recovery MUST skip duplicate resubmission for that workflow

#### Scenario: Partial recovery failure
- **WHEN** one workflow fails normalization or resubmission
- **THEN** recovery MUST log the failure and continue processing remaining workflows without blocking service startup
