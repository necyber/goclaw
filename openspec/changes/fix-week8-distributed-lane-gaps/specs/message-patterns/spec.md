## MODIFIED Requirements

### Requirement: Collect message pattern

The system SHALL support a collect message pattern that aggregates outputs from multiple tasks with fair fan-in behavior and deterministic timeout partial results.

#### Scenario: Collect all task results
- **WHEN** a collector is created for tasks ["task-1", "task-2", "task-3"]
- **THEN** the collector waits for all tasks to complete and returns aggregated results

#### Scenario: Collect with timeout
- **WHEN** a collector is created with a 30s timeout and not all tasks complete in time
- **THEN** the collector returns available partial results and a timeout error that indicates collection was incomplete

#### Scenario: Collect with streaming
- **WHEN** a collector is created in streaming mode
- **THEN** the collector emits results as each task completes (fan-in pattern)

#### Scenario: Collect with all tasks failed
- **WHEN** all tasks in a collector fail
- **THEN** the collector returns an error with all individual task errors

#### Scenario: Collect with partial failure
- **WHEN** some tasks succeed and some fail
- **THEN** the collector returns successful results and errors for failed tasks

#### Scenario: Slow channel does not block ready results
- **WHEN** one task channel is idle or delayed while another task result is already available
- **THEN** collector fan-in MUST continue processing ready results without being blocked by the idle channel
