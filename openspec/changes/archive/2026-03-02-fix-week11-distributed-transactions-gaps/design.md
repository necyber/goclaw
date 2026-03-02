## Context

Week11 introduced the Saga runtime, API, and recovery pipeline. Current behavior still has four production risks:
- Parallel compensation mutates shared instance fields without synchronization.
- Startup recovery is wired with an empty definition map, so incomplete Sagas are skipped.
- Checkpoints are persisted after step completion only, so lifecycle transitions can be missing from durable state.
- `SagaDefinition.MaxConcurrent` is validated but not enforced during step execution.

These are cross-cutting issues across `pkg/saga`, `pkg/engine`, and API transport layers, and they affect high-stakes failure handling.

## Goals / Non-Goals

**Goals:**
- Ensure compensation state bookkeeping is race-free under parallel compensation.
- Make automatic/manual recovery independent from process-local definition caches.
- Persist checkpoint snapshots whenever Saga lifecycle state changes.
- Enforce per-definition step concurrency limits consistently in normal and resumed execution.
- Add regression tests that fail on race/recovery regressions.

**Non-Goals:**
- No choreography-style Saga redesign.
- No new external service dependency beyond current Badger-based runtime persistence.
- No breaking redesign of the existing Saga DSL surface.

## Decisions

1. Persist executable definitions by `saga_id`
- Decision: add a durable definition store in `pkg/saga` and write definition snapshots at submit time; read by recovery and manual operations.
- Why: recovery/manual operations must survive process restarts.
- Alternative considered: keep in-memory map plus warning logs.
- Why not: does not satisfy restart recovery requirements.

2. Centralize checkpoint persistence on state transitions
- Decision: introduce a single orchestrator helper that persists checkpoint snapshots after every successful lifecycle transition and failure-field update.
- Why: eliminates branch-specific omissions and stale checkpoint state.
- Alternative considered: add ad-hoc persistence calls in each branch.
- Why not: high drift risk and easy to regress.

3. Synchronize compensation bookkeeping mutations
- Decision: guard mutable `SagaInstance` compensation fields with synchronization during parallel compensation execution.
- Why: removes data races confirmed by `go test -race`.
- Alternative considered: force serial compensation per layer.
- Why not: unnecessary throughput regression and not required by spec.

4. Enforce definition-level step concurrency
- Decision: use a per-execution step semaphore bounded by `definition.MaxConcurrent` for forward and resumed step execution.
- Why: makes DSL concurrency setting effective and predictable.
- Alternative considered: rely only on global saga-level semaphore.
- Why not: does not constrain fan-out within one Saga.

5. API/gRPC lookup order for definitions
- Decision: resolve definitions by `saga_id` from durable store first, then optional in-memory cache fallback.
- Why: consistent post-restart behavior while preserving in-process fast path.
- Alternative considered: remove in-memory cache entirely.
- Why not: adds avoidable read overhead for hot-path operations.

## Risks / Trade-offs

- [Definition snapshot size growth] -> Mitigation: store one immutable definition per `saga_id` and clean up with terminal Saga retention policy.
- [More checkpoint writes] -> Mitigation: write compact snapshots only on state transitions and critical updates, and keep Badger batch-friendly usage.
- [Lock contention in compensation bookkeeping] -> Mitigation: lock only around shared state mutation, not around user compensation function execution.
- [Behavior differences for missing definitions] -> Mitigation: return explicit not-found/precondition errors and document migration/ops behavior.

## Migration Plan

1. Add definition store types, persistence keys, and serialization tests.
2. Wire definition persistence into HTTP/gRPC submit paths and engine-owned startup recovery flow.
3. Add orchestrator checkpoint persistence helper and replace scattered state-only updates.
4. Add synchronization for compensation bookkeeping and run `go test -race ./pkg/saga/...`.
5. Add per-definition concurrency semaphore in execute/resume paths and cover with fan-out tests.
6. Update Saga API handlers to use durable definition lookup for compensate/recover operations.
7. Update docs (`docs/saga-guide.md`) to reflect recovery prerequisites and guarantees.

Rollback strategy:
- Keep existing in-memory definition cache path for compatibility.
- If new persistent lookup causes operational issues, gate fallback behavior by config while preserving old execution behavior.

## Open Questions

- Should definition snapshots be deleted immediately at terminal state, or retained for an operator-configurable period?
- Should missing-definition recovery count as `failed` or `skipped` in recovery metrics for alerting semantics?
