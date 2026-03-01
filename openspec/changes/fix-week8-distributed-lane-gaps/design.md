## Context

Week8 introduced Redis-backed lane processing, fallback behavior, and distributed signal/message patterns. The current implementation passes baseline tests but still has high-impact correctness gaps:
- Engine Redis queue mode can mark tasks completed without executing the task function.
- Redis signal bus unsubscribe/close may race with forwarding and panic.
- Collect fan-in can block on one channel and miss available partial results.
- Distributed queue capacity checks rely too heavily on local counters.
- Some fallback downgrade paths over-classify non-connectivity errors as Redis failures.
- Fencing failure paths can skip dedup cleanup and leave stale dedup keys.

Constraints:
- Keep existing public APIs stable where possible.
- Preserve OpenSpec week8 capabilities while tightening behavioral guarantees.
- Keep changes incremental and test-backed.

## Goals / Non-Goals

**Goals:**
- Guarantee Redis-lane tasks submitted through engine execution path run the bound task function before terminal success.
- Make signal bus close/unsubscribe concurrency-safe under publish/forward pressure.
- Ensure collect returns deterministic partial results on timeout without starvation.
- Improve distributed queue backpressure correctness and dedup/fencing cleanup semantics.
- Add regression tests that catch previous false-success, race, and timeout-flake cases.

**Non-Goals:**
- Redesign lane APIs or introduce new user-facing RPCs.
- Build a fully new distributed scheduler/ownership protocol.
- Replace Redis Pub/Sub transport or change signal message schema.

## Decisions

### 1) Redis lane execution binding must use executable payload contract
- Decision: Add an execution payload path that preserves or resolves executable behavior at worker side (for same-process path used by engine), and fail-fast when executable payload cannot be resolved.
- Rationale: Current metadata-only payload allows false positive completion.
- Alternative considered: Keep Redis lane as transport-only and force external workers. Rejected because engine currently exposes Redis queue mode as first-class runtime path.

### 2) Redis bus channel lifecycle must be writer-owned
- Decision: Forwarding goroutine becomes the single closer/writer boundary for subscriber channels; unsubscribe/close cancels context and removes subscription map entry without directly closing channels from management path.
- Rationale: Eliminates send-on-closed-channel races.
- Alternative considered: Add broad locking around every send/close. Rejected due to deadlock risk and higher contention.

### 3) Collect fan-in should use multiplexed non-blocking aggregation
- Decision: Replace per-channel blocking loop with fan-in multiplexer (single result stream / select-based dispatch) that tracks completion per task and honors timeout while emitting/recording partial results.
- Rationale: Prevent starvation and flaky timeout behavior.
- Alternative considered: Keep polling loop with reduced per-channel wait time. Rejected as still fairness-sensitive and timing-fragile.

### 4) Distributed backpressure should consult authoritative queue depth when near/at limit
- Decision: Use Redis queue length as authoritative source before final admission decision (with optional local fast-path only as hint).
- Rationale: Local `pending` is node-local and underestimates shared depth in multi-producer deployments.
- Alternative considered: Strictly local counters only. Rejected due to correctness loss in distributed mode.

### 5) Fallback degrade classification should be connectivity-focused
- Decision: Narrow `isRedisError` to connectivity/transport failures and explicitly exclude known lane-domain errors (duplicate/full/closed/drop/validation/context).
- Rationale: Prevent accidental mode switches on business errors.
- Alternative considered: Keep broad classification for aggressiveness. Rejected because it breaks error semantics and observability.

### 6) Dedup cleanup must run on all terminal dequeue outcomes
- Decision: Ensure dedup key cleanup happens on fencing validation failure and other early terminal paths that consume a queued item.
- Rationale: Avoid stale dedup preventing legitimate resubmits.
- Alternative considered: rely on TTL only. Rejected due to long duplicate suppression windows.

## Risks / Trade-offs

- [Behavior change in Redis lane execution path] ¡ú Add explicit tests for engine Redis-mode workflow execution and document expected same-process execution contract.
- [Race fix may affect close timing semantics] ¡ú Add deterministic concurrent unsubscribe/publish tests and run `go test -race` in touched packages.
- [Queue-depth authoritative checks add Redis round-trips] ¡ú Only perform strict check at admission boundary and keep local hints to reduce overhead.
- [Collect refactor complexity] ¡ú Keep implementation small, with task-indexed completion map and timeout path tests.

## Migration Plan

1. Update specs deltas for redis-lane, signal-bus, and message-patterns.
2. Implement fixes behind existing APIs.
3. Add regression tests first for each bug class, then adjust implementation until all pass.
4. Validate with unit/integration + race tests for `pkg/lane` and `pkg/signal`.
5. Rollback strategy: revert the change set as a unit; no schema migration or persisted data format change required.

## Open Questions

- Should Redis lane explicitly reject non-serializable executable tasks when configured for cross-process consumption mode?
- Do we want a dedicated metric for collect timeout partial-count to improve operability?
- Should fallback lane expose current mode in lane stats as a first-class field (beyond external metrics/logging)?
