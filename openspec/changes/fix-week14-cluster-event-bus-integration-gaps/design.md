## Context

`week14-cluster-event-bus` added specs for distributed coordination and canonical lifecycle events, but current runtime wiring remains mostly local-only. Lifecycle transitions are emitted to local observers without canonical event publication, streaming bridge wiring exists but is not attached in startup, and bridge payloads are not translated into the event types consumed by gRPC streaming handlers. In addition, coordination adapters currently emulate etcd/consul with in-memory behavior, and transfer deduplication state is globally keyed by workload ID, which can collide across shards.

Constraints:
- Keep compatibility with existing local runtime behavior.
- Preserve non-blocking transition hooks and avoid introducing hot-path blocking on event publication.
- Keep event consumer semantics at-least-once with explicit deduplication.

## Goals / Non-Goals

**Goals:**
- Ensure workflow/task persisted transitions publish canonical lifecycle events in distributed mode.
- Ensure gRPC streaming can consume canonical event-bus updates in production startup paths.
- Ensure bridge-delivered messages are decoded/translated into handler-compatible lifecycle event types.
- Prevent silent backend emulation for etcd/consul in distributed mode.
- Scope ownership-transfer duplicate suppression by shard to avoid cross-shard collisions.

**Non-Goals:**
- Full JetStream persistence/replay redesign.
- Redesign of workflow/task state machines from week13.
- Exactly-once delivery guarantees across the distributed stack.

## Decisions

### 1) Introduce composite runtime broadcaster with canonical publisher path
Decision:
- Add a composite broadcaster that preserves existing local broadcast behavior and additionally publishes canonical lifecycle events via `eventbus.Publisher`.
- Publish is asynchronous with bounded retry (existing publisher policy) and degraded-mode telemetry updates.

Rationale:
- Keeps current runtime event hooks intact while enforcing week14 canonical publication contracts.
- Avoids invasive engine transition rewrites.

Alternatives considered:
- Replace local broadcaster entirely with event-bus-only path.
- Rejected due to immediate regression risk for existing local websocket/stream consumers.

### 2) Bridge must output typed workflow/task lifecycle events for gRPC handlers
Decision:
- Extend bridge decode path to map envelope payloads into `engine.WorkflowEvent`/`engine.TaskEvent` before registry broadcast.
- Validate schema version routing via `SchemaRouter`; unknown schema versions surface explicit bridge errors/metrics rather than silent skip.

Rationale:
- gRPC handlers currently filter by concrete engine event types; map-based payload broadcast silently drops updates.

Alternatives considered:
- Modify handlers to parse generic maps.
- Rejected because it spreads schema logic across handlers and weakens type safety.

### 3) Wire bridge attachment in startup when distributed event bus is configured
Decision:
- Add startup wiring in server bootstrap to create/attach bridge when event transport is available.
- Keep graceful no-op when transport disabled in local-only mode.

Rationale:
- Existing attach API is never called; this leaves distributed streaming behavior dead code.

Alternatives considered:
- Lazy attach on first stream subscription.
- Rejected due to race/observability complexity and inconsistent startup diagnostics.

### 4) Coordination adapter semantics must be explicit, not silently emulated
Decision:
- For `etcd`/`consul` modes without real backend implementation, fail fast with explicit unsupported/degraded error path (or require explicit dev/test override).
- Keep `memory` backend for local/testing.

Rationale:
- Silent memory emulation in distributed mode violates ownership/leader semantics and can hide production risk.

Alternatives considered:
- Continue emulation with warning logs.
- Rejected because it still advertises unsupported correctness guarantees.

### 5) Transfer dedupe state keyed by `(shard, workload)`
Decision:
- Change ownership transfer completion/duplicate tracking keys from global `workloadID` to scoped `(shardKey, workloadID)`.

Rationale:
- Prevents cross-shard suppression collisions and aligns with shard ownership semantics.

Alternatives considered:
- Require globally unique workload IDs everywhere.
- Rejected due to unnecessary coupling across independent shard domains.

## Risks / Trade-offs

- [Risk] Additional publish path may add latency under persistent bus failure.
  -> Mitigation: keep retries bounded and non-blocking; rely on degraded telemetry and backpressure-safe fallback.

- [Risk] Schema decode strictness could drop events for unsupported versions.
  -> Mitigation: explicit metrics/error surfacing and compatibility-window policy tests.

- [Risk] Fail-fast coordination backend behavior can break deployments relying on implicit emulation.
  -> Mitigation: explicit config gate for test/dev emulation and migration note in rollout plan.

## Migration Plan

1. Add composite broadcaster and event publisher wiring behind a runtime config gate.
2. Add typed bridge decode/translation and startup attach path.
3. Add cluster adapter guardrails (unsupported backend behavior without explicit emulation flag).
4. Update transfer manager dedupe scoping.
5. Roll out with staging flags enabled, verify event publish/bridge/stream metrics.
6. Rollback: disable distributed event publish/bridge flags and revert to local-only broadcaster path.

## Open Questions

- Should unsupported etcd/consul mode be hard error at startup or soft degraded mode with read-only local execution?
- Should bridge schema decode failures be surfaced to stream clients directly or only through telemetry/logging?
- Do we need explicit envelope-to-engine mapping version registry per schema version beyond v1 in this change?
