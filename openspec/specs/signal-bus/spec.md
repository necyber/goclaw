# signal-bus Specification

## Purpose
TBD - created by archiving change week8-distributed-lane. Update Purpose after archive.
## Requirements
### Requirement: Signal Bus interface

The system SHALL provide a SignalBus interface with Publish, Subscribe, Unsubscribe, and Close operations.

#### Scenario: Publish signal
- **WHEN** `Publish` is called with a valid signal
- **THEN** the signal is delivered to all subscribers of the target task ID

#### Scenario: Subscribe to task signals
- **WHEN** `Subscribe` is called with a task ID
- **THEN** the system returns a channel that receives signals for that task

#### Scenario: Unsubscribe from task signals
- **WHEN** `Unsubscribe` is called with a task ID
- **THEN** the system stops delivering signals for that task and closes the channel

#### Scenario: Close signal bus
- **WHEN** `Close` is called
- **THEN** all subscriptions are cancelled and resources are released

### Requirement: Local signal bus implementation

The system SHALL provide a local (in-memory) signal bus using Go channels.

#### Scenario: Local publish and subscribe
- **WHEN** a signal is published locally and a subscriber exists
- **THEN** the subscriber receives the signal via its channel

#### Scenario: Local publish without subscriber
- **WHEN** a signal is published locally but no subscriber exists for the task
- **THEN** the signal is discarded silently

#### Scenario: Local buffered delivery
- **WHEN** a subscriber's channel buffer is full
- **THEN** the system drops the oldest signal and delivers the new one (ring buffer behavior)

### Requirement: Redis signal bus implementation

The system SHALL provide a distributed signal bus using Redis Pub/Sub.

#### Scenario: Publish signal via Redis
- **WHEN** a signal is published via Redis signal bus
- **THEN** the signal is serialized and published to Redis channel "goclaw:signal:{taskID}"

#### Scenario: Subscribe via Redis
- **WHEN** `Subscribe` is called on Redis signal bus
- **THEN** the system subscribes to Redis channel "goclaw:signal:{taskID}"

#### Scenario: Cross-node signal delivery
- **WHEN** node A publishes a signal and node B has a subscriber for the same task
- **THEN** node B receives the signal via Redis Pub/Sub

#### Scenario: Redis Pub/Sub reconnection
- **WHEN** Redis connection is lost during subscription
- **THEN** the system automatically resubscribes after reconnection

### Requirement: Signal bus mode selection

The system SHALL support configurable signal bus mode: local or redis.

#### Scenario: Local mode configuration
- **WHEN** signal bus is configured with mode "local"
- **THEN** the system uses in-memory channels for signal delivery

#### Scenario: Redis mode configuration
- **WHEN** signal bus is configured with mode "redis"
- **THEN** the system uses Redis Pub/Sub for signal delivery

#### Scenario: Default mode
- **WHEN** signal bus mode is not configured
- **THEN** the system defaults to "local" mode

### Requirement: Signal type definitions

The system SHALL define signal types: Steer, Interrupt, and Collect.

#### Scenario: Create steer signal
- **WHEN** a signal is created with type Steer
- **THEN** the signal contains a payload map with parameter key-value pairs

#### Scenario: Create interrupt signal
- **WHEN** a signal is created with type Interrupt
- **THEN** the signal contains reason, graceful flag, and optional timeout

#### Scenario: Create collect signal
- **WHEN** a signal is created with type Collect
- **THEN** the signal contains the task result data

### Requirement: Signal serialization for Redis

The system SHALL serialize signals to JSON for Redis Pub/Sub transport.

#### Scenario: Serialize signal
- **WHEN** a signal is published via Redis
- **THEN** the signal is serialized to JSON with fields: type, taskID, payload, sentAt

#### Scenario: Deserialize signal
- **WHEN** a signal is received from Redis Pub/Sub
- **THEN** the JSON is deserialized back to a Signal struct

### Requirement: Concurrent signal operations

The system SHALL support concurrent publish and subscribe operations without data races.

#### Scenario: Concurrent publishes
- **WHEN** multiple goroutines publish signals simultaneously
- **THEN** all signals are delivered without data races

#### Scenario: Concurrent subscribe and publish
- **WHEN** one goroutine subscribes while another publishes
- **THEN** both operations complete without deadlock or data races

### Requirement: Signal buffer configuration

The system SHALL support configurable signal channel buffer size.

#### Scenario: Custom buffer size
- **WHEN** signal bus is configured with buffer size 100
- **THEN** each subscription channel has a buffer of 100 signals

#### Scenario: Default buffer size
- **WHEN** buffer size is not configured
- **THEN** the system uses a default buffer size of 16

### Requirement: Signal bus graceful shutdown

The system SHALL gracefully shut down the signal bus on system shutdown.

#### Scenario: Shutdown with active subscriptions
- **WHEN** the signal bus is closed while subscriptions are active
- **THEN** all subscription channels are closed and subscribers receive channel close notification

#### Scenario: Publish after shutdown
- **WHEN** `Publish` is called after the signal bus is closed
- **THEN** the system returns an ErrBusClosed error

### Requirement: Signal bus health check

The system SHALL provide health check for the signal bus.

#### Scenario: Local bus health check
- **WHEN** health check is performed on local signal bus
- **THEN** the system returns healthy status

#### Scenario: Redis bus health check
- **WHEN** health check is performed on Redis signal bus
- **THEN** the system pings Redis and returns connection status


### Requirement: Signal routing honors distributed ownership
In distributed mode, signal delivery MUST route according to current task ownership.

#### Scenario: Signal to remotely owned task
- **WHEN** a signal is published for a task owned by another node
- **THEN** runtime MUST route the signal to the owner node via distributed signal transport

#### Scenario: Signal to local owned task
- **WHEN** a signal is published for a task owned by the local node
- **THEN** runtime MAY deliver through local fast-path while preserving signal contract

### Requirement: Signal delivery during ownership changes
Signal routing MUST remain deterministic while ownership changes are in progress.

#### Scenario: Ownership changes during signal send
- **WHEN** ownership changes between signal publish and consume
- **THEN** runtime MUST deliver to current owner or return explicit ownership-change error based on policy

#### Scenario: Duplicate signal path prevention
- **WHEN** both local and distributed routing paths are available
- **THEN** runtime MUST avoid duplicate signal delivery for the same signal identifier
