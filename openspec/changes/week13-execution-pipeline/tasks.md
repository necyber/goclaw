## 1. Execution Pipeline Core

- [x] 1.1 Introduce runtime submission mode model (sync/async) and request-to-runtime mapping
- [x] 1.2 Implement workflow pipeline transitions `pending -> scheduled -> running -> terminal` as runtime invariants
- [x] 1.3 Add transition validation guards for illegal workflow/task state transitions
- [x] 1.4 Persist workflow/task timestamps and terminal payload fields at each transition boundary

## 2. Scheduler and Lane Integration

- [x] 2.1 Refactor scheduler dispatch path to use `lane.Manager.Submit` for runnable tasks
- [x] 2.2 Implement lane-driven task completion signaling for layer barrier progression
- [x] 2.3 Implement lane submission error mapping to deterministic task/workflow failure outcomes
- [x] 2.4 Add cancellation-aware scheduling behavior for queued/runnable tasks

## 3. Runtime API Semantics (HTTP)

- [x] 3.1 Implement async submit behavior that returns after workflow is persisted as `pending` (without waiting for execution admission)
- [x] 3.2 Implement sync submit behavior: block until terminal state only when executable `TaskFns` are provided; otherwise return non-terminal `pending` response
- [x] 3.3 Align workflow query responses with persisted runtime state and timestamps
- [x] 3.4 Align cancel and task-result endpoint behavior for pending/running/terminal states

## 4. gRPC Runtime Wiring and Consistency

- [x] 4.1 Add gRPC service registration wiring in main runtime startup for enabled services
- [x] 4.2 Introduce concrete engine adapter(s) used by gRPC handlers
- [x] 4.3 Fail startup on missing required adapter wiring for enabled services
- [x] 4.4 Align gRPC status/error semantics with runtime execution lifecycle

## 5. Streaming and Metrics Alignment

- [x] 5.1 Emit workflow/task stream events from persisted transition hooks
- [x] 5.2 Enforce per-workflow transition ordering in stream emission path
- [x] 5.3 Align workflow metrics emission to transition hooks with explicit cancellation/timeout labeling
- [x] 5.4 Align task metrics emission to scheduling/running/retry/terminal transitions with idempotent terminal counting

## 6. Verification and Regression Coverage

- [x] 6.1 Add unit tests for transition guard rules and terminal outcome precedence
- [x] 6.2 Add integration tests for scheduler-lane dispatch and layer barrier behavior
- [x] 6.3 Add API tests for sync/async submit semantics and cancel/result contracts
- [x] 6.4 Add gRPC integration tests for runtime registration and adapter wiring correctness
- [x] 6.5 Add streaming/metrics consistency tests against persisted state transitions
