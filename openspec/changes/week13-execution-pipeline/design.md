## Context

The codebase already has most runtime building blocks (HTTP handlers, engine submit path, scheduler, lane manager, storage persistence, metrics, gRPC handlers), but the critical execution path is fragmented.

Current gaps:
- HTTP workflow submission persists workflow state but does not trigger execution.
- Scheduler still executes task runners directly in goroutines instead of dispatching through Lane Manager.
- Runtime status persistence and API-visible execution semantics are not strictly defined as one lifecycle contract.
- gRPC handlers exist, but runtime registration and engine adapter wiring are not treated as a first-class runtime requirement.

Constraints:
- Keep DAG execution semantics (layer-based dependency ordering) intact.
- Reuse existing engine/storage/lane abstractions without introducing a new workflow runtime framework.
- Preserve backward-compatible API shapes where possible; tighten semantics instead of redesigning endpoints.

## Goals / Non-Goals

**Goals:**
- Define one authoritative execution lifecycle from submission to terminal state.
- Make Lane Manager the required dispatch boundary between scheduler and task execution.
- Define deterministic workflow/task state transitions and persistence expectations.
- Align HTTP/gRPC status responses and streaming events with persisted runtime state.
- Ensure workflow/task metrics map to lifecycle transitions instead of ad-hoc call sites.

**Non-Goals:**
- No cluster coordination or sharding design in this change (handled separately).
- No NATS event-bus design in this change (handled separately).
- No Saga/distributed transaction implementation in this change.
- No redesign of UI/UX workflows beyond runtime semantics they consume.

## Decisions

### 1. Execution lifecycle as a strict pipeline contract

Decision:
- Define workflow runtime as a pipeline with explicit gates:
  `accepted -> persisted(pending) -> scheduled -> running -> terminal(completed/failed/cancelled)`.

Rationale:
- Removes ambiguity between API acceptance and actual execution.
- Provides a stable contract for status queries, cancellation semantics, streaming, and metrics.

Alternatives considered:
- Keep current loosely coupled flow and patch each module independently.
- Rejected because it preserves inconsistent behavior and fragile status semantics.

### 2. Submission mode semantics are explicit and observable

Decision:
- Support both sync and async submission modes with explicit response semantics.
- Async mode returns once workflow state is persisted as `pending` (without waiting for execution admission or terminal completion).
- Sync mode blocks until terminal workflow state or request cancellation/timeout when executable `TaskFns` are provided.
- Sync mode without executable `TaskFns` returns a non-terminal `pending` response after persistence (no blocking for terminal completion).

Rationale:
- Supports operational APIs while retaining deterministic behavior for callers.
- Avoids overloading a single submit endpoint with implicit mode behavior.

Alternatives considered:
- Async-only submission.
- Rejected because tests/integration tooling and some server-side operations need synchronous completion semantics.

### 3. Scheduler dispatches via Lane Manager, never direct task goroutine execution

Decision:
- Replace direct task runner execution in scheduler with lane submission and completion signaling.
- Scheduler remains responsible for DAG ordering; lane subsystem is responsible for execution resource control.

Rationale:
- Enforces one resource-control path.
- Makes backpressure/rate-limit/retry accounting coherent.

Alternatives considered:
- Hybrid direct/lane execution based on task type.
- Rejected because it fragments runtime semantics and metric interpretation.

### 4. State transition invariants are persisted at each transition boundary

Decision:
- Persist workflow/task state updates at each meaningful transition with timestamps and terminal payloads.
- Define forbidden transitions and cancellation precedence rules.

Rationale:
- Enables reliable status query, restart recovery, and stream replay consistency.

Alternatives considered:
- Persist only terminal states.
- Rejected because intermediate observability and cancellation/debugging semantics become unreliable.

### 5. gRPC runtime wiring is part of execution pipeline, not optional integration detail

Decision:
- Add explicit requirements for service registration completeness and adapter wiring between gRPC handlers and engine runtime interfaces.

Rationale:
- Existing code has handler implementations, but runtime viability depends on registration + wiring.
- Treating wiring as spec-level behavior prevents “implemented but unreachable” drift.

Alternatives considered:
- Keep wiring concerns in implementation tasks only.
- Rejected because it repeatedly regresses without a formal contract.

### 6. Lifecycle metrics and streaming events are derived from the same transition source

Decision:
- Record workflow/task metrics and emit workflow/task stream events from transition hooks driven by persisted state changes.

Rationale:
- Prevents disagreement between metrics, stream events, and queried state.

Alternatives considered:
- Independent instrumentation per subsystem.
- Rejected due to drift risk and harder incident diagnosis.

## Risks / Trade-offs

- [Risk] Tighter transition enforcement may expose existing edge-case bugs during rollout.
  → Mitigation: introduce transition validation logs first, then enforce hard failures after validation period.

- [Risk] Scheduler-to-lane refactor can change throughput/latency characteristics.
  → Mitigation: preserve current concurrency defaults; add benchmark and guardrail metrics before rollout.

- [Risk] Sync submit mode can hold server resources for long workflows.
  → Mitigation: require explicit sync mode usage and enforce request timeout boundaries.

- [Risk] Adapter interfaces for HTTP/gRPC may introduce temporary duplication.
  → Mitigation: define one runtime-facing adapter contract and reuse across transports.

## Migration Plan

1. Define and merge execution-pipeline and scheduler-lane integration specs.
2. Add adapter contracts for runtime submit/status/cancel/task-result operations.
3. Refactor scheduler dispatch path to lane submission + completion signaling.
4. Add transition persistence hooks and validation for illegal transitions.
5. Align HTTP and gRPC handlers to shared runtime semantics.
6. Move metrics/stream emission to transition-driven hooks.
7. Roll out with transition validation logs enabled, then enforce strict transition guards.

## Open Questions

- Should sync/async mode be endpoint-level or request-flag-level in HTTP API?
- For cancellation, should queued (not yet running) tasks be marked `cancelled` immediately or `failed` with cancellation reason?
- Should streaming guarantee at-least-once or best-effort delivery for transition events in this phase?
