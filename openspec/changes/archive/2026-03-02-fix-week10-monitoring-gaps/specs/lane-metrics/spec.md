## MODIFIED Requirements

### Requirement: Lane wait duration metrics
The metrics system SHALL measure time tasks spend waiting in queue before execution for all submission paths, including redirect backpressure paths.

#### Scenario: Record task wait time
- **WHEN** task is dequeued and begins execution
- **THEN** system calculates wait duration and records in lane_wait_duration_seconds histogram

#### Scenario: Wait duration histogram buckets
- **WHEN** recording wait duration
- **THEN** system uses buckets [0.001, 0.01, 0.1, 0.5, 1, 5, 10, 30] seconds

#### Scenario: Track wait time by lane
- **WHEN** recording wait duration
- **THEN** system includes lane_name label for filtering

#### Scenario: Wait duration for redirect fast-path submission
- **WHEN** a task is accepted via lane Redirect fast path and later dequeued
- **THEN** wait duration MUST still be recorded using enqueue timestamp from the accepted source-lane submission event

### Requirement: Lane metrics integration
The metrics system SHALL integrate with lane queue operations using a single enqueue timestamp contract for block, drop-accepted, try-submit-accepted, and redirect-accepted paths.

#### Scenario: Hook into Enqueue operation
- **WHEN** lane Enqueue method is called
- **THEN** metrics manager records enqueue timestamp and updates queue depth

#### Scenario: Hook into Dequeue operation
- **WHEN** lane worker dequeues task for execution
- **THEN** metrics manager calculates wait time and updates queue depth

#### Scenario: Hook into task completion
- **WHEN** lane completes task processing
- **THEN** metrics manager increments throughput counter

#### Scenario: Wait duration is recorded for standard lane submissions
- **WHEN** a task is accepted through normal lane submission path and later dequeued
- **THEN** wait duration observation MUST be recorded even if task does not implement custom enqueue-time interfaces

#### Scenario: Redirect submission uses the same enqueue wrapper contract
- **WHEN** lane accepts a task on redirect fast path
- **THEN** the runtime MUST wrap the task with the same enqueue timestamp contract used by non-redirect accepted paths
