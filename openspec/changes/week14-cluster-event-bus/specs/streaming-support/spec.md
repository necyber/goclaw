## ADDED Requirements

### Requirement: Streaming bridge consumes canonical distributed events
Streaming services in distributed mode MUST source workflow/task updates from canonical cluster event streams.

#### Scenario: Cross-node workflow update
- **WHEN** workflow transition occurs on node A
- **THEN** subscribers connected to node B MUST receive the update through event-bus-backed streaming bridge

#### Scenario: Task update fan-out
- **WHEN** task transitions are published to canonical event bus
- **THEN** streaming service MUST transform and deliver updates to matching workflow/task subscriptions

### Requirement: Stream consistency across node boundaries
Streaming bridge MUST preserve scoped ordering and deduplicate duplicate bus deliveries.

#### Scenario: Ordered per-workflow delivery
- **WHEN** multiple updates for one workflow are consumed from event bus
- **THEN** streaming output MUST preserve per-workflow transition order

#### Scenario: Duplicate event consumed
- **WHEN** duplicate lifecycle events are consumed from event bus
- **THEN** streaming service MUST suppress duplicates using event identity and ordering metadata

