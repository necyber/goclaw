## Context

The `week13-execution-pipeline` change established lifecycle semantics, but runtime behavior still has conformance gaps discovered by review:
- task terminal result payload is not persisted in the runtime transition path;
- cancel flow does not enforce a configurable graceful-stop timeout window;
- streaming may emit a synthetic initial pending state that does not match persisted state;
- streaming may drop transition events under backpressure, including terminal visibility risk;
- task terminal metrics do not consistently distinguish timeout-derived outcomes from user cancellation.

These issues span engine runtime transitions, API semantics, streaming infrastructure, and metrics labels. A coordinated design is needed to avoid piecemeal fixes that reintroduce divergence.

## Goals / Non-Goals

**Goals:**
- Make task terminal payload persistence deterministic and queryable from the existing task-result APIs.
- Enforce cancellation graceful-timeout semantics with explicit terminal outcome precedence.
- Ensure streaming updates are sourced from persisted state semantics, including first message correctness.
- Preserve per-workflow transition ordering and terminal visibility under backpressure.
- Emit task terminal metrics with explicit cancellation vs timeout labels.

**Non-Goals:**
- No redesign of workflow DAG scheduling or lane resource model.
- No introduction of durable stream replay storage in this change.
- No changes to unrelated Saga/distributed-transaction semantics.

## Decisions

### 1. Persist terminal task payload in transition hook

Decision:
- Extend runtime task terminal transition handling so `completed` transitions persist terminal payload into `storage.TaskState.Result`.
- Keep non-terminal transitions result-free to avoid stale payload leakage.

Rationale:
- Task result APIs already read persisted task state; persisting payload at terminal transition is the single-source-of-truth path.

Alternatives considered:
- Persist payload only in transport handlers.
- Rejected because it breaks storage-driven consistency and recovery behavior.

### 2. Add bounded graceful cancellation window

Decision:
- Introduce configurable cancellation graceful timeout in runtime cancel flow.
- `CancelWorkflow` must: signal cancellation, wait for in-flight task settlement up to timeout, then force timeout-derived terminal mapping for remaining active tasks.

Rationale:
- This enforces spec-required deterministic cancel behavior and avoids immediate-return race semantics.

Alternatives considered:
- Immediate cancel return without wait.
- Rejected because it cannot guarantee graceful-stop semantics.

### 3. Stream initial message reflects persisted workflow state

Decision:
- Replace fixed initial `pending` stream update with persisted current workflow state snapshot at subscription time.

Rationale:
- Prevents first-event drift and aligns streaming with persisted state source-of-truth guarantees.

Alternatives considered:
- Keep synthetic initial state for simplicity.
- Rejected because clients can observe impossible regressions (e.g., running workflow reported as pending).

### 4. Backpressure policy preserves terminal visibility

Decision:
- Keep non-blocking streaming fanout, but enforce terminal-priority delivery: terminal workflow/task events must not be silently dropped for subscribed consumers.
- For sustained overload, close stream with explicit resource/backpressure error after terminal delivery attempt.

Rationale:
- Balances engine safety (no blocking runtime transitions) with lifecycle visibility guarantees.

Alternatives considered:
- Unbounded buffering.
- Rejected due to memory safety and head-of-line blocking risks.

### 5. Explicit timeout vs cancel metric labels

Decision:
- Define deterministic terminal label mapping:
  - user cancellation -> `cancelled`
  - timeout-derived termination -> `failed_timeout` (or dedicated timeout terminal label, consistently applied)
- Apply mapping at task terminal transition emission point.

Rationale:
- Enables operational distinction between user intent and runtime timeout policy.

Alternatives considered:
- Treat all cancellations/timeouts as `cancelled`.
- Rejected due to loss of incident diagnosability and inconsistency with workflow timeout labeling.

## Risks / Trade-offs

- [Risk] Waiting during cancel can increase request latency.
  -> Mitigation: enforce bounded timeout with clear config defaults and timeout telemetry.

- [Risk] Terminal-priority stream delivery may increase stream-side complexity.
  -> Mitigation: keep policy scoped to terminal events and add targeted tests for ordering/drop behavior.

- [Risk] Result payload persistence may increase storage footprint.
  -> Mitigation: persist existing payload contract only for terminal completed tasks; no replay log introduced.

## Migration Plan

1. Update runtime transition helpers for task result persistence and terminal label mapping.
2. Add cancellation graceful-timeout configuration and bounded wait flow in cancel path.
3. Update streaming subscription bootstrap to load and emit persisted current state.
4. Implement terminal-priority backpressure behavior and stream error signaling.
5. Add regression tests for each review finding and verify no behavior regressions in existing runtime paths.

Rollback:
- Revert to previous cancel/stream behavior by disabling new cancellation timeout setting and reverting the transition/stream patch set if operational issues appear.

## Open Questions

- Should timeout-derived task terminal label remain `failed_timeout` for backward compatibility, or move to an explicit `timeout` label in a follow-up change?
- For backpressure-triggered stream closure, should error code be `ResourceExhausted` uniformly across all streaming RPCs?
