# lane-metrics Specification

## Purpose
Migrated from legacy OpenSpec format while preserving existing requirement and scenario content.

## Requirements

### Requirement: Lane queue depth metrics
The metrics system SHALL track the current depth of each lane queue.

#### Scenario: Update queue depth on enqueue
- **WHEN** task is added to lane queue
- **THEN** system increments lane_queue_depth gauge for that lane

#### Scenario: Update queue depth on dequeue
- **WHEN** task is removed from lane queue for execution
- **THEN** system decrements lane_queue_depth gauge for that lane

#### Scenario: Track multiple lanes
- **WHEN** system has multiple named lanes
- **THEN** each lane has separate lane_queue_depth gauge with lane_name label

### Requirement: Lane wait duration metrics
The metrics system SHALL measure time tasks spend waiting in queue before execution.

#### Scenario: Record task wait time
- **WHEN** task is dequeued and begins execution
- **THEN** system calculates wait duration and records in lane_wait_duration_seconds histogram

#### Scenario: Wait duration histogram buckets
- **WHEN** recording wait duration
- **THEN** system uses buckets [0.001, 0.01, 0.1, 0.5, 1, 5, 10, 30] seconds

#### Scenario: Track wait time by lane
- **WHEN** recording wait duration
- **THEN** system includes lane_name label for filtering

### Requirement: Lane throughput metrics
The metrics system SHALL track the total number of tasks processed by each lane.

#### Scenario: Record task processing
- **WHEN** lane completes processing a task (success or failure)
- **THEN** system increments lane_throughput_total counter for that lane

#### Scenario: Track throughput by lane
- **WHEN** recording throughput
- **THEN** system includes lane_name label to distinguish lanes

### Requirement: Lane metrics integration
The metrics system SHALL integrate with lane queue operations.

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

### Requirement: Backpressure outcome metrics
Lane metrics SHALL expose canonical backpressure outcomes using `accepted`, `rejected`, `redirected`, and `dropped` counters.

#### Scenario: Record accepted submissions
- **WHEN** a task submission is admitted into a lane queue
- **THEN** metrics MUST increment `accepted` for that lane

#### Scenario: Record rejected submissions
- **WHEN** a task submission fails before admission
- **THEN** metrics MUST increment `rejected` for that lane

#### Scenario: Record redirected submissions
- **WHEN** a task submission is redirected to another lane
- **THEN** metrics MUST increment `redirected` for source lane and MUST NOT classify the same event as direct accepted

#### Scenario: Record dropped submissions
- **WHEN** a task submission is dropped due to backpressure policy
- **THEN** metrics MUST increment `dropped` for that lane

#### Scenario: Outcome counters are exported via metrics manager
- **WHEN** runtime records canonical submission outcomes
- **THEN** metrics manager MUST expose a scrapeable counter metric keyed by `lane_name` and `outcome`
- **AND** `outcome` label values MUST be limited to `accepted`, `rejected`, `redirected`, and `dropped`

