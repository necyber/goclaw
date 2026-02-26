## 1. Execution Pipeline Core

- [ ] 1.1 Introduce runtime submission mode model (sync/async) and request-to-runtime mapping
- [ ] 1.2 Implement workflow pipeline transitions `pending -> scheduled -> running -> terminal` as runtime invariants
- [ ] 1.3 Add transition validation guards for illegal workflow/task state transitions
- [ ] 1.4 Persist workflow/task timestamps and terminal payload fields at each transition boundary

## 2. Scheduler and Lane Integration

- [ ] 2.1 Refactor scheduler dispatch path to use `lane.Manager.Submit` for runnable tasks
- [ ] 2.2 Implement lane-driven task completion signaling for layer barrier progression
- [ ] 2.3 Implement lane submission error mapping to deterministic task/workflow failure outcomes
- [ ] 2.4 Add cancellation-aware scheduling behavior for queued/runnable tasks

## 3. Runtime API Semantics (HTTP)

- [ ] 3.1 Implement async submit behavior that returns after workflow is persisted as `pending` (without waiting for execution admission)
- [ ] 3.2 Implement sync submit behavior: block until terminal state only when executable `TaskFns` are provided; otherwise return non-terminal `pending` response
- [ ] 3.3 Align workflow query responses with persisted runtime state and timestamps
- [ ] 3.4 Align cancel and task-result endpoint behavior for pending/running/terminal states

## 4. gRPC Runtime Wiring and Consistency

- [ ] 4.1 Add gRPC service registration wiring in main runtime startup for enabled services
- [ ] 4.2 Introduce concrete engine adapter(s) used by gRPC handlers
- [ ] 4.3 Fail startup on missing required adapter wiring for enabled services
- [ ] 4.4 Align gRPC status/error semantics with runtime execution lifecycle

## 5. Streaming and Metrics Alignment

- [ ] 5.1 Emit workflow/task stream events from persisted transition hooks
- [ ] 5.2 Enforce per-workflow transition ordering in stream emission path
- [ ] 5.3 Align workflow metrics emission to transition hooks with explicit cancellation/timeout labeling
- [ ] 5.4 Align task metrics emission to scheduling/running/retry/terminal transitions with idempotent terminal counting

## 6. Verification and Regression Coverage

- [ ] 6.1 Add unit tests for transition guard rules and terminal outcome precedence
- [ ] 6.2 Add integration tests for scheduler-lane dispatch and layer barrier behavior
- [ ] 6.3 Add API tests for sync/async submit semantics and cancel/result contracts
- [ ] 6.4 Add gRPC integration tests for runtime registration and adapter wiring correctness
- [ ] 6.5 Add streaming/metrics consistency tests against persisted state transitions
