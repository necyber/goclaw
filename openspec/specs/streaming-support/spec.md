# streaming-support Specification

## Purpose
Migrated from legacy OpenSpec format while preserving existing requirement and scenario content.

## Requirements

### Requirement: Server-side streaming for workflow updates
The system SHALL provide server-side streaming to push real-time workflow status updates to clients.

#### Scenario: Watch workflow stream
- **WHEN** client calls WatchWorkflow with workflow ID
- **THEN** server MUST stream WorkflowStatusUpdate messages whenever workflow state changes

#### Scenario: Stream initial state
- **WHEN** client subscribes to workflow updates
- **THEN** server MUST send current workflow state as first message

#### Scenario: Stream completion
- **WHEN** workflow reaches terminal state (completed, failed, cancelled)
- **THEN** server MUST send final status update and close stream

#### Scenario: Stream error handling
- **WHEN** error occurs during streaming
- **THEN** server MUST send error status and close stream

### Requirement: Server-side streaming for task updates
The system SHALL provide server-side streaming to push real-time task progress updates to clients.

#### Scenario: Watch tasks stream
- **WHEN** client calls WatchTasks with workflow ID
- **THEN** server MUST stream TaskProgressUpdate messages for all tasks in workflow

#### Scenario: Task state changes
- **WHEN** task transitions between states (pending 鈫?running 鈫?completed)
- **THEN** server MUST send update with new state, timestamps, and progress percentage

#### Scenario: Task result streaming
- **WHEN** task completes with result
- **THEN** server MUST include result data in final task update

#### Scenario: Partial task updates
- **WHEN** long-running task reports progress
- **THEN** server MUST stream incremental progress updates

### Requirement: Stream lifecycle management
The system SHALL manage stream lifecycle including subscription, updates, and cleanup.

#### Scenario: Stream subscription
- **WHEN** client initiates stream
- **THEN** server MUST register subscriber and begin sending updates

#### Scenario: Stream unsubscription
- **WHEN** client closes stream
- **THEN** server MUST remove subscriber and stop sending updates

#### Scenario: Multiple subscribers
- **WHEN** multiple clients watch same workflow
- **THEN** server MUST maintain separate streams for each client

#### Scenario: Stream timeout
- **WHEN** no updates occur within keepalive interval
- **THEN** server MUST send keepalive message to prevent connection timeout

#### Scenario: Stale subscriber cleanup
- **WHEN** stale slow-consumer subscribers are cleaned up
- **THEN** cleanup MUST complete without deadlock and MUST leave registry state consistent

### Requirement: Bidirectional streaming for log streaming
The system SHALL provide bidirectional streaming for real-time log delivery.

#### Scenario: Log stream initialization
- **WHEN** client calls StreamLogs with workflow ID and log level filter
- **THEN** server MUST stream log entries matching the filter

#### Scenario: Dynamic filter updates
- **WHEN** client sends filter update message
- **THEN** server MUST apply new filter to subsequent log entries using concurrency-safe state synchronization

#### Scenario: Log buffering
- **WHEN** logs are generated faster than client can consume
- **THEN** server MUST buffer logs up to configured limit and drop oldest if exceeded

#### Scenario: Log stream completion
- **WHEN** workflow completes
- **THEN** server MUST flush remaining logs and close stream

### Requirement: Stream backpressure handling
The system SHALL handle slow consumers without blocking workflow execution.

#### Scenario: Slow consumer detection
- **WHEN** client cannot keep up with update rate
- **THEN** server MUST detect backpressure via send buffer size

#### Scenario: Update coalescing
- **WHEN** multiple updates occur before client reads
- **THEN** server MUST coalesce intermediate states and send only latest

#### Scenario: Buffer overflow
- **WHEN** client is too slow and buffer is full
- **THEN** server MUST close stream with ResourceExhausted status

### Requirement: Stream filtering
The system SHALL support filtering to reduce unnecessary updates.

#### Scenario: Task filter by ID
- **WHEN** client watches specific tasks
- **THEN** server MUST only stream updates for requested task IDs

#### Scenario: State filter
- **WHEN** client requests only terminal state updates
- **THEN** server MUST only stream when tasks reach completed/failed/cancelled states

#### Scenario: Log level filter
- **WHEN** client specifies log level
- **THEN** server MUST only stream logs at or above specified level

### Requirement: Stream reconnection
The system SHALL support stream resumption after disconnection.

#### Scenario: Resume from last update
- **WHEN** client reconnects with last received sequence number
- **THEN** server MUST resume stream from next update after that sequence

#### Scenario: Sequence number validation
- **WHEN** client provides invalid or expired sequence number
- **THEN** server MUST return InvalidArgument status

#### Scenario: Full resync
- **WHEN** client reconnects without sequence number
- **THEN** server MUST send current state and continue with new updates

### Requirement: Stream metadata
The system SHALL include metadata in stream messages for ordering and deduplication.

#### Scenario: Sequence numbers
- **WHEN** server sends update
- **THEN** message MUST include monotonically increasing sequence number

#### Scenario: Timestamps
- **WHEN** server sends update
- **THEN** message MUST include server timestamp for event ordering

#### Scenario: Update type
- **WHEN** server sends update
- **THEN** message MUST indicate update type (state_change, progress, log, etc.)

### Requirement: Stream performance
The system SHALL optimize streaming performance for high-throughput scenarios.

#### Scenario: Batch updates
- **WHEN** multiple tasks update simultaneously
- **THEN** server MUST batch updates in single stream message when possible

#### Scenario: Compression
- **WHEN** streaming large payloads
- **THEN** server MUST use gRPC compression to reduce bandwidth

#### Scenario: Stream concurrency
- **WHEN** handling multiple concurrent streams
- **THEN** server MUST use goroutines per stream without blocking others

### Requirement: Streaming events follow persisted lifecycle transitions
Workflow and task streaming updates MUST be emitted from persisted lifecycle transitions.

#### Scenario: Persisted transition emits stream event
- **WHEN** workflow or task state change is persisted
- **THEN** streaming service MUST emit a corresponding update event that reflects the persisted state

#### Scenario: Initial workflow stream message reflects persisted state
- **WHEN** a client subscribes to a workflow stream
- **THEN** the first state-bearing message MUST be derived from the latest persisted workflow state
- **AND** streaming service MUST NOT emit a synthetic fixed `pending` state that conflicts with persisted state

### Requirement: Transition-consistent stream ordering per workflow
Streaming for a single workflow MUST preserve transition order consistency.

#### Scenario: Ordered task transition stream
- **WHEN** a task transitions from `scheduled` to `running` to terminal state
- **THEN** subscribers for that workflow MUST receive updates in the same transition order

### Requirement: Terminal transition stream guarantees
Streaming MUST expose terminal state visibility for workflow and task streams.

#### Scenario: Workflow reaches terminal state
- **WHEN** workflow transitions to `completed`, `failed`, or `cancelled`
- **THEN** streaming service MUST emit terminal update before stream closure or idle state

#### Scenario: Slow consumer during terminal transition
- **WHEN** stream backpressure is present while terminal transition event is being published
- **THEN** streaming service MUST prioritize terminal visibility over intermediate non-terminal updates according to runtime policy
- **AND** if delivery cannot be completed, stream MUST close with explicit backpressure/resource error semantics

### Requirement: Backpressure handling preserves per-workflow transition correctness
Streaming backpressure handling MUST preserve per-workflow transition correctness guarantees without blocking runtime persistence.

#### Scenario: Intermediate non-terminal updates are shed under pressure
- **WHEN** a subscriber cannot keep up and stream buffer pressure exceeds limits
- **THEN** streaming implementation MAY coalesce or shed intermediate non-terminal updates according to policy
- **AND** emitted updates MUST remain transition-order-consistent for each delivered workflow stream

#### Scenario: Terminal visibility remains deterministic
- **WHEN** update shedding/coalescing occurs under pressure
- **THEN** terminal workflow/task transition visibility MUST still be deterministic to subscribers
- **AND** the client MUST be informed through terminal update delivery or explicit terminal stream error semantics

### Requirement: Streaming bridge consumes canonical distributed events
Streaming services in distributed mode MUST source workflow/task updates from canonical cluster event streams through startup-attached bridge wiring.

#### Scenario: Cross-node workflow update
- **WHEN** workflow transition occurs on node A
- **THEN** subscribers connected to node B MUST receive the update through event-bus-backed streaming bridge
- **AND** bridge attachment MUST be active from runtime startup path in distributed mode

#### Scenario: Task update fan-out
- **WHEN** task transitions are published to canonical event bus
- **THEN** streaming service MUST transform and deliver updates to matching workflow/task subscriptions
- **AND** bridge output MUST be decoded into handler-compatible lifecycle event types before fan-out

### Requirement: Stream consistency across node boundaries
Streaming bridge MUST preserve scoped ordering and deduplicate duplicate bus deliveries.

#### Scenario: Ordered per-workflow delivery
- **WHEN** multiple updates for one workflow are consumed from event bus
- **THEN** streaming output MUST preserve per-workflow transition order

#### Scenario: Duplicate event consumed
- **WHEN** duplicate lifecycle events are consumed from event bus
- **THEN** streaming service MUST suppress duplicates using event identity and ordering metadata

#### Scenario: Unsupported bridge schema version
- **WHEN** bridge receives an event payload with unsupported schema version
- **THEN** bridge MUST surface explicit decode failure telemetry
- **AND** bridge MUST NOT silently broadcast untyped payloads that bypass lifecycle handlers

### Requirement: Race-safe streaming concurrency verification
Streaming workflow/task delivery paths and their concurrency test fixtures MUST remain race-safe under Go race detector execution.

#### Scenario: Workflow streaming bridge path is race-safe
- **WHEN** targeted race checks run for gRPC streaming handlers (including event-bus bridge integration paths)
- **THEN** the test execution MUST complete without data race findings.

#### Scenario: Shared stream buffers are synchronized
- **WHEN** streaming tests append and read shared update collections from multiple goroutines
- **THEN** the implementation MUST use explicit synchronization so concurrent reads/writes do not race.

