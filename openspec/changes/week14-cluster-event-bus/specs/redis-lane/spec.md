## ADDED Requirements

### Requirement: Redis lane ownership in distributed mode
Redis lane consumption MUST respect cluster ownership boundaries in distributed mode.

#### Scenario: Owner-only consumption
- **WHEN** a node does not own a shard/lane partition
- **THEN** that node MUST NOT consume tasks for that ownership scope

#### Scenario: Ownership transfer
- **WHEN** shard/lane ownership transfers to another node
- **THEN** previous owner MUST stop new dequeues for that scope and new owner MUST begin consuming

### Requirement: Redis lane failover safety
Redis lane execution MUST avoid duplicate terminal execution during node failover.

#### Scenario: Failover with pending queue
- **WHEN** owner node fails and ownership is reassigned
- **THEN** reassigned owner MUST continue pending queue processing without violating deduplication guarantees

#### Scenario: Stale consumer after lease loss
- **WHEN** a node loses ownership lease but still has local worker activity
- **THEN** runtime MUST fence stale consumer execution for new dequeue operations

