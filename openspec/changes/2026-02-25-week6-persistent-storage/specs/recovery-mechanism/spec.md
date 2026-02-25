## ADDED Requirements

### Requirement: System recovers workflows on startup
The system SHALL automatically recover incomplete workflows when service starts.

#### Scenario: Recover pending workflows
- **WHEN** service starts after restart
- **THEN** system loads all workflows with status "pending" from storage

#### Scenario: Recover running workflows
- **WHEN** service starts after crash
- **THEN** system loads all workflows with status "running" from storage

#### Scenario: Skip completed workflows
- **WHEN** service starts
- **THEN** system does not load workflows with status "completed" or "failed"

### Requirement: Recovered workflows resume execution
The system SHALL resume execution of recovered workflows from their last known state.

#### Scenario: Resume pending workflow
- **WHEN** pending workflow is recovered
- **THEN** system submits workflow to execution queue

#### Scenario: Resume running workflow
- **WHEN** running workflow is recovered
- **THEN** system determines which tasks completed and resumes from next task

#### Scenario: Retry failed tasks
- **WHEN** recovered workflow has failed tasks with retry count remaining
- **THEN** system retries those tasks

### Requirement: Recovery handles task state correctly
The system SHALL correctly interpret task state during recovery.

#### Scenario: Completed tasks are not re-executed
- **WHEN** recovering workflow with completed tasks
- **THEN** system marks those tasks as done and skips execution

#### Scenario: Running tasks are restarted
- **WHEN** recovering workflow with tasks in "running" state
- **THEN** system resets those tasks to "pending" and re-executes

#### Scenario: Failed tasks follow retry policy
- **WHEN** recovering workflow with failed tasks
- **THEN** system applies retry policy (retry or mark as failed)

### Requirement: Recovery logs are generated
The system SHALL log recovery operations for observability.

#### Scenario: Log recovery start
- **WHEN** recovery process begins
- **THEN** system logs number of workflows to recover

#### Scenario: Log recovered workflow
- **WHEN** each workflow is recovered
- **THEN** system logs workflow ID, status, and task count

#### Scenario: Log recovery completion
- **WHEN** recovery process completes
- **THEN** system logs total recovered workflows and any errors

### Requirement: Recovery handles errors gracefully
The system SHALL handle recovery errors without blocking service startup.

#### Scenario: Skip corrupted workflow
- **WHEN** workflow data is corrupted during recovery
- **THEN** system logs error and continues with next workflow

#### Scenario: Partial recovery on storage error
- **WHEN** storage is partially unavailable during recovery
- **THEN** system recovers accessible workflows and logs unavailable ones

#### Scenario: Service starts despite recovery failures
- **WHEN** some workflows fail to recover
- **THEN** system completes startup and serves new requests

### Requirement: Recovery is idempotent
The system SHALL handle multiple recovery attempts safely.

#### Scenario: Duplicate recovery attempt
- **WHEN** recovery runs multiple times for same workflow
- **THEN** system detects duplicate and skips re-recovery

#### Scenario: Recovery during execution
- **WHEN** workflow is already executing when recovery runs
- **THEN** system detects active execution and skips recovery
