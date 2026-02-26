## ADDED Requirements

### Requirement: Deterministic shard assignment
Workflow and task ownership MUST be assigned deterministically from active membership.

#### Scenario: Assign workflow shard on submit
- **WHEN** a workflow is accepted in distributed mode
- **THEN** runtime MUST assign it to a shard key and route ownership using a deterministic hash strategy

#### Scenario: Stable routing without membership changes
- **WHEN** cluster membership remains unchanged
- **THEN** identical shard keys MUST resolve to the same owner node

### Requirement: Rebalance on membership changes
The runtime MUST rebalance shard ownership when nodes join, leave, or fail.

#### Scenario: Node joins cluster
- **WHEN** a new healthy node joins
- **THEN** rebalance MUST move only necessary shard ownerships according to consistent hashing rules

#### Scenario: Node failure
- **WHEN** an owner node becomes unhealthy
- **THEN** affected shard ownership MUST be reassigned to healthy nodes

### Requirement: In-flight task handling on ownership transfer
Ownership transfer MUST define behavior for in-flight and queued work.

#### Scenario: Transfer with queued work
- **WHEN** ownership transfers for a shard with queued tasks
- **THEN** queued tasks MUST be processed by the new owner without duplicate execution

#### Scenario: Transfer with in-flight task
- **WHEN** ownership transfers while an in-flight task exists
- **THEN** runtime MUST enforce one terminal outcome through fencing or idempotent completion rules

