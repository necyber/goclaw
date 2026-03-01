## MODIFIED Requirements

### Requirement: Backpressure strategies for Redis Lane

The system SHALL support Block, Drop, and Redirect backpressure strategies with authoritative Redis queue-length checks at admission boundaries.

#### Scenario: Block strategy when full
- **WHEN** the Redis queue is at capacity and backpressure is Block
- **THEN** the Submit call MUST block until capacity is available, lane closes, or context is cancelled

#### Scenario: Drop strategy when full
- **WHEN** the Redis queue is at capacity and backpressure is Drop
- **THEN** the task MUST be dropped and an error MUST be returned

#### Scenario: Redirect strategy when full
- **WHEN** the Redis queue is at capacity and backpressure is Redirect
- **THEN** the task MUST be submitted to the configured redirect lane; if redirect submission fails, the source lane MUST return a non-success outcome without counting a redirect success

### Requirement: Worker pool for Redis Lane

The system SHALL run a worker pool that dequeues and executes tasks from Redis, and MUST NOT report task success unless the bound task execution path completes successfully.

#### Scenario: Worker dequeues task
- **WHEN** a worker is idle and a task is available in Redis
- **THEN** the worker dequeues the task and executes the bound task logic

#### Scenario: Worker pool concurrency limit
- **WHEN** all workers are busy (at MaxConcurrency)
- **THEN** no additional tasks are dequeued until a worker becomes available

#### Scenario: Worker handles task failure
- **WHEN** a task execution returns an error
- **THEN** the worker records the failure and proceeds to the next task

#### Scenario: Missing execution binding in same-process runtime
- **WHEN** a dequeued task cannot be resolved to an executable binding in the same-process runtime path
- **THEN** the worker MUST treat the task as failed and MUST NOT increment completed success counters

### Requirement: Redis Lane fallback

The system SHALL support fallback to local Channel Lane when Redis is unavailable due to connectivity or transport health failures.

#### Scenario: Redis unavailable at startup
- **WHEN** Redis Lane is configured but Redis is unreachable at startup
- **THEN** the system falls back to local Channel Lane with a warning log

#### Scenario: Redis becomes unavailable during operation
- **WHEN** Redis connectivity fails during operation
- **THEN** the system switches to local Channel Lane and retries Redis connection in background

#### Scenario: Non-connectivity lane errors do not trigger fallback
- **WHEN** submission fails with lane-domain errors (for example duplicate, full, dropped, or validation errors)
- **THEN** the system MUST preserve current mode and MUST NOT mark Redis as degraded solely from that error

### Requirement: Redis lane failover safety

Redis lane execution MUST avoid duplicate terminal execution during node failover and MUST clean dedup state for dequeued tasks on terminal failure paths.

#### Scenario: Failover with pending queue
- **WHEN** owner node fails and ownership is reassigned
- **THEN** reassigned owner MUST continue pending queue processing without violating deduplication guarantees

#### Scenario: Stale consumer after lease loss
- **WHEN** a node loses ownership lease but still has local worker activity
- **THEN** runtime MUST fence stale consumer execution for new dequeue operations

#### Scenario: Fencing failure after dequeue
- **WHEN** fencing validation fails for a dequeued task
- **THEN** runtime MUST mark execution as failed and MUST release dedup state for that task ID
