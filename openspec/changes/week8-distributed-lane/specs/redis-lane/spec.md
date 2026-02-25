## ADDED Requirements

### Requirement: Redis Lane implements Lane interface

The Redis Lane SHALL implement the existing `Lane` interface (`Submit`, `TrySubmit`, `Stats`, `Close`, `IsClosed`).

#### Scenario: Submit task to Redis Lane
- **WHEN** `Submit` is called with a valid task
- **THEN** the task is serialized and enqueued to the Redis queue

#### Scenario: TrySubmit when queue has capacity
- **WHEN** `TrySubmit` is called and the queue is not full
- **THEN** the task is enqueued and returns true

#### Scenario: TrySubmit when queue is full
- **WHEN** `TrySubmit` is called and the queue is at capacity
- **THEN** the task is not enqueued and returns false

### Requirement: Redis queue with priority support

The system SHALL use Redis Sorted Set for priority queues and Redis List for FIFO queues.

#### Scenario: Submit with priority enabled
- **WHEN** a task with priority 10 is submitted to a priority-enabled Redis Lane
- **THEN** the task is added to a Sorted Set with score equal to the priority

#### Scenario: Dequeue highest priority task
- **WHEN** a worker dequeues from a priority-enabled Redis Lane
- **THEN** the task with the highest priority score is returned first

#### Scenario: Submit without priority
- **WHEN** a task is submitted to a non-priority Redis Lane
- **THEN** the task is appended to a Redis List (FIFO order)

### Requirement: Task serialization

The system SHALL serialize tasks to JSON for storage in Redis.

#### Scenario: Serialize task to Redis
- **WHEN** a task is submitted to Redis Lane
- **THEN** the task is serialized as JSON with fields: id, lane, priority, payload, metadata, enqueued_at

#### Scenario: Deserialize task from Redis
- **WHEN** a worker dequeues a task from Redis
- **THEN** the JSON payload is deserialized back to a task object

### Requirement: Task deduplication

The system SHALL prevent duplicate task submissions using a Redis Set.

#### Scenario: Submit unique task
- **WHEN** a task with ID "task-1" is submitted for the first time
- **THEN** the task is enqueued and the ID is added to the dedup set

#### Scenario: Submit duplicate task
- **WHEN** a task with ID "task-1" is submitted again while still pending
- **THEN** the submission is rejected with a duplicate error

#### Scenario: Resubmit after completion
- **WHEN** a task with ID "task-1" is submitted after it was completed and removed from dedup set
- **THEN** the task is enqueued successfully

### Requirement: Backpressure strategies for Redis Lane

The system SHALL support Block, Drop, and Redirect backpressure strategies.

#### Scenario: Block strategy when full
- **WHEN** the Redis queue is at capacity and backpressure is Block
- **THEN** the Submit call blocks until space is available (using BRPOP polling)

#### Scenario: Drop strategy when full
- **WHEN** the Redis queue is at capacity and backpressure is Drop
- **THEN** the task is dropped and an error is returned

#### Scenario: Redirect strategy when full
- **WHEN** the Redis queue is at capacity and backpressure is Redirect
- **THEN** the task is enqueued to the configured redirect Lane's Redis key

### Requirement: Worker pool for Redis Lane

The system SHALL run a worker pool that dequeues and executes tasks from Redis.

#### Scenario: Worker dequeues task
- **WHEN** a worker is idle and a task is available in Redis
- **THEN** the worker dequeues the task and executes it

#### Scenario: Worker pool concurrency limit
- **WHEN** all workers are busy (at MaxConcurrency)
- **THEN** no additional tasks are dequeued until a worker becomes available

#### Scenario: Worker handles task failure
- **WHEN** a task execution returns an error
- **THEN** the worker records the failure and proceeds to the next task

### Requirement: Redis Lane statistics

The system SHALL provide accurate statistics for Redis Lane operations.

#### Scenario: Get pending count
- **WHEN** `Stats()` is called
- **THEN** the system returns the current Redis queue length as Pending count

#### Scenario: Get running count
- **WHEN** `Stats()` is called
- **THEN** the system returns the number of currently executing tasks as Running count

#### Scenario: Track completed and failed counts
- **WHEN** tasks complete or fail
- **THEN** the system increments Completed or Failed counters atomically

### Requirement: Redis connection management

The system SHALL manage Redis connections with automatic reconnection.

#### Scenario: Connect to Redis
- **WHEN** Redis Lane is initialized with valid Redis configuration
- **THEN** the system establishes a connection to Redis

#### Scenario: Reconnect on connection loss
- **WHEN** the Redis connection is lost
- **THEN** the system automatically attempts to reconnect with exponential backoff

#### Scenario: Health check
- **WHEN** the system performs a health check on Redis Lane
- **THEN** the system pings Redis and reports connection status

### Requirement: Graceful shutdown

The system SHALL gracefully shut down Redis Lane operations.

#### Scenario: Close with pending tasks
- **WHEN** `Close` is called with pending tasks in Redis
- **THEN** the system stops accepting new tasks and waits for running tasks to complete

#### Scenario: Close with timeout
- **WHEN** `Close` is called and running tasks don't complete within context deadline
- **THEN** the system cancels running tasks and returns

### Requirement: Redis Lane fallback

The system SHALL support fallback to local Channel Lane when Redis is unavailable.

#### Scenario: Redis unavailable at startup
- **WHEN** Redis Lane is configured but Redis is unreachable at startup
- **THEN** the system falls back to local Channel Lane with a warning log

#### Scenario: Redis becomes unavailable during operation
- **WHEN** Redis becomes unreachable during operation
- **THEN** the system switches to local Channel Lane and retries Redis connection in background

### Requirement: Redis Lane performance

The system SHALL achieve acceptable performance for distributed operations.

#### Scenario: Submit latency
- **WHEN** submitting a task to Redis Lane
- **THEN** the operation completes in less than 5ms (excluding network latency)

#### Scenario: Throughput
- **WHEN** submitting tasks at high rate
- **THEN** the system sustains at least 10,000 tasks/second per Lane
