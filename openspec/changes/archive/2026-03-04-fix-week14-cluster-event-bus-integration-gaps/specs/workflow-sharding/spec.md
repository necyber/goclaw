## MODIFIED Requirements

### Requirement: In-flight task handling on ownership transfer
Ownership transfer MUST define behavior for in-flight and queued work with duplicate suppression scoped by shard and workload identity.

#### Scenario: Transfer with queued work
- **WHEN** ownership transfers for a shard with queued tasks
- **THEN** queued tasks MUST be processed by the new owner without duplicate execution
- **AND** duplicate tracking MUST use shard-scoped identity instead of global workload-only keys

#### Scenario: Transfer with in-flight task
- **WHEN** ownership transfers while an in-flight task exists
- **THEN** runtime MUST enforce one terminal outcome through fencing or idempotent completion rules

#### Scenario: Same workload ID appears on different shards
- **WHEN** transfer completion is recorded for workload ID `X` on shard `A`
- **THEN** transfer processing for workload ID `X` on shard `B` MUST remain independently eligible
- **AND** cross-shard duplicate suppression collisions MUST NOT occur
