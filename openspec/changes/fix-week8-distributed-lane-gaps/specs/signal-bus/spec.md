## MODIFIED Requirements

### Requirement: Redis signal bus implementation

The system SHALL provide a distributed signal bus using Redis Pub/Sub with lifecycle-safe subscription teardown.

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

#### Scenario: Unsubscribe during in-flight forward
- **WHEN** unsubscribe is called while a forwarding goroutine is concurrently delivering a signal
- **THEN** the system MUST complete teardown without panic and without send-on-closed-channel failure

### Requirement: Concurrent signal operations

The system SHALL support concurrent publish, subscribe, unsubscribe, and close operations without data races or panics.

#### Scenario: Concurrent publishes
- **WHEN** multiple goroutines publish signals simultaneously
- **THEN** all signals are delivered without data races

#### Scenario: Concurrent subscribe and publish
- **WHEN** one goroutine subscribes while another publishes
- **THEN** both operations complete without deadlock or data races

#### Scenario: Concurrent unsubscribe and publish
- **WHEN** unsubscribe races with publish/forward operations for the same task
- **THEN** the system MUST remain race-safe and MUST NOT panic

### Requirement: Signal bus graceful shutdown

The system SHALL gracefully shut down the signal bus on system shutdown using race-safe channel ownership semantics.

#### Scenario: Shutdown with active subscriptions
- **WHEN** the signal bus is closed while subscriptions are active
- **THEN** all subscriptions are cancelled and subscriber channels are closed in a way that cannot panic concurrent forwarders

#### Scenario: Publish after shutdown
- **WHEN** `Publish` is called after the signal bus is closed
- **THEN** the system returns an ErrBusClosed error
