## MODIFIED Requirements

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
- **WHEN** client calls `StreamLogs` with workflow ID and log level filter
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
