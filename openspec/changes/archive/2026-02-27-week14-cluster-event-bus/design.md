## Context

The project has local and Redis-based runtime primitives, but lacks a formal distributed control contract:
- No authoritative node membership and ownership model for multi-node execution.
- No deterministic sharding/rebalance rules for workflow and lane ownership.
- No NATS event backbone contract for cross-node workflow/task lifecycle propagation.

As a result, distributed behavior is implementation-fragile: failover semantics, stream consistency, and cross-node observability are not guaranteed by spec.

Constraints:
- Keep this change focused on distributed coordination + event propagation.
- Do not duplicate week13 execution-pipeline semantics; this change consumes those lifecycle transitions.
- Support degraded mode when etcd/Consul or NATS dependencies are unavailable.

## Goals / Non-Goals

**Goals:**
- Define membership and ownership contracts for cluster nodes.
- Define deterministic sharding and rebalance rules.
- Define NATS event publication/consumption semantics for workflow/task lifecycle events.
- Define event schema/versioning guarantees for internal consumers.
- Define failure and degradation behavior for coordination and bus outages.

**Non-Goals:**
- Not implementing distributed transactions/Saga logic in this change.
- Not redesigning user-facing workflow APIs.
- Not replacing Redis lane or signal bus; only defining their distributed integration behavior.
- Not enforcing exactly-once semantics across the entire distributed stack.

## Decisions

### 1. Coordination backend is abstracted, with etcd/Consul as implementations

Decision:
- Define a coordination interface with two implementation targets (etcd and Consul).
- Core operations: join/heartbeat/leave, leader lease, ownership claims, watch membership changes.

Rationale:
- Keeps runtime portable while supporting the planned control-plane options.
- Prevents hard-coding distributed behavior to one backend.

Alternatives considered:
- Backend-specific behavior embedded in engine.
- Rejected because it duplicates logic and complicates failover guarantees.

### 2. Ownership model is lease-based with deterministic shard assignment

Decision:
- Use lease-backed ownership for shard keys.
- Assign shard ownership using consistent hashing ring over healthy nodes.
- Rebalance on membership change with minimal key movement.

Rationale:
- Deterministic routing across nodes.
- Graceful behavior on node churn with bounded reassignment.

Alternatives considered:
- Randomized assignment or broadcast execution.
- Rejected due to duplicate execution risk and unstable routing.

### 3. NATS is the canonical cross-node lifecycle event backbone

Decision:
- Publish workflow/task lifecycle events to NATS subjects by domain and shard.
- Consumers (streaming adapters, metrics pipeline, audit components) subscribe to canonical subjects.

Rationale:
- Decouples execution from downstream event consumers.
- Enables scalable fan-out and cross-node observability.

Alternatives considered:
- Redis Pub/Sub as sole lifecycle backbone.
- Rejected due to weaker stream durability/replay patterns for this use case.

### 4. Event envelope uses versioned schema contract

Decision:
- Define required event envelope fields (`event_id`, `event_type`, `workflow_id`, `task_id`, `timestamp`, `node_id`, `schema_version`, `payload`).
- Require backward-compatible schema evolution with explicit version handling.

Rationale:
- Prevents consumer breakage as event payload evolves.
- Enables side-by-side readers during migrations.

Alternatives considered:
- Ad-hoc per-publisher payloads without global envelope.
- Rejected because compatibility and operational debugging degrade quickly.

### 5. Delivery semantics are at-least-once with idempotent consumption expectations

Decision:
- Runtime guarantees at-least-once event publish intent.
- Consumers MUST handle duplicates using idempotency keys (`event_id`).

Rationale:
- Practical for distributed reliability without heavy exactly-once coordination overhead.

Alternatives considered:
- Exactly-once end-to-end guarantee.
- Rejected due to complexity/cost not justified for this phase.

### 6. Degraded mode behavior is explicit

Decision:
- If coordination backend is unavailable: node enters standalone-safe mode (no new distributed ownership claims).
- If NATS is unavailable: runtime keeps local execution and records bus outage metrics; configurable buffering/retry policy is applied.

Rationale:
- Prefer controlled degradation over undefined partial operation.

Alternatives considered:
- Hard-stop entire runtime on dependency outage.
- Rejected because local progress and recovery flexibility are needed.

## Risks / Trade-offs

- [Risk] Split-brain ownership during coordination instability.
  → Mitigation: lease TTL + fencing token checks on ownership-sensitive operations.

- [Risk] Event duplication and out-of-order arrival at consumers.
  → Mitigation: idempotent consumer contract + per-workflow sequence metadata.

- [Risk] Rebalance storms on frequent node churn.
  → Mitigation: rebalance cooldown window and capped movement per cycle.

- [Risk] Operational complexity from two external dependencies.
  → Mitigation: health probes, clear degraded-mode telemetry, and dependency-specific runbooks.

## Migration Plan

1. Introduce coordination abstraction and backend adapters (feature-flagged).
2. Introduce shard assignment + ownership claim flow in shadow mode (observe only).
3. Introduce NATS publishers for lifecycle events while keeping current local paths.
4. Enable consumers (streaming/metrics bridges) to read canonical NATS envelope.
5. Turn on active ownership enforcement and rebalance in staged environments.
6. Enable degraded-mode policy controls and alerting thresholds in production.

## Open Questions

- Should ownership claims be per-workflow, per-lane, or dual-level (lane + workflow shard)?
- Do we need JetStream persistence from day one, or core NATS with bounded replay bridge?
- How should sequence ordering be scoped: global, per-workflow, or per-shard?
