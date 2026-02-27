## Why

Current distributed capabilities are fragmented: Redis lane and signal bus exist, but there is no formal cluster coordination contract (node discovery, ownership, sharding) and no runtime NATS event-bus contract for cross-node workflow/task events. This blocks reliable multi-node operation and consistent event propagation.

## What Changes

- Define cluster coordination requirements for node membership, liveness, and lane/workflow ownership.
- Define deterministic task/workflow sharding and reassignment behavior for node join/leave/failure.
- Define NATS event-bus requirements for workflow/task lifecycle events, delivery semantics, and replay boundaries.
- Define event schema/versioning requirements for cross-node consumers (streaming, metrics, audit pipelines).
- Define failure handling and degraded-mode behavior when coordination backend or NATS is unavailable.

## Capabilities

### New Capabilities
- `cluster-coordination`: Cluster membership, liveness, ownership, and failover contracts.
- `workflow-sharding`: Task/workflow partitioning, routing, rebalance, and ownership transfer rules.
- `nats-event-bus`: NATS-based event publication/consumption contract for workflow/task lifecycle events.
- `event-schema-contract`: Event envelope, schema versioning, compatibility, and consumer-facing guarantees.

### Modified Capabilities
- `redis-lane`: Define how Redis lane behavior integrates with sharded multi-node ownership and failover.
- `signal-bus`: Define interoperability between signal delivery and cluster ownership boundaries in distributed mode.
- `streaming-support`: Define how runtime streams consume/bridge cluster-level event bus updates.

## Impact

- Affected runtime modules: lane manager, distributed queue behavior, signal routing, event publishing/consumption.
- Affected infrastructure dependencies: etcd/Consul (coordination backend), NATS (event backbone).
- Affected API/ops surfaces: streaming consistency, admin observability, failure/degradation semantics.
- Affected specs/docs: new distributed and event-bus capabilities plus deltas for redis-lane/signal-bus/streaming-support.
