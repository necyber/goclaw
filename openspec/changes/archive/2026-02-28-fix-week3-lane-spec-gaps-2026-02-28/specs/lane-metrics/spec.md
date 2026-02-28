## MODIFIED Requirements

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
