## 1. Saga Definition Persistence

- [x] 1.1 Add a durable Saga definition store in `pkg/saga` keyed by `saga_id`.
- [x] 1.2 Persist definition snapshots from HTTP submit path (`pkg/api/handlers/saga.go`).
- [x] 1.3 Persist definition snapshots from gRPC submit path (`pkg/grpc/handlers/saga.go`).
- [x] 1.4 Wire engine startup recovery to build/load definition map from durable storage instead of passing an empty map.
- [x] 1.5 Add unit tests for definition store save/load/delete/list behavior and error handling.

## 2. Orchestrator and Checkpoint Correctness

- [x] 2.1 Introduce a centralized checkpoint persistence helper for lifecycle transitions in `pkg/saga/orchestrator.go`.
- [x] 2.2 Persist checkpoints on transitions to `pending-compensation`, `compensating`, and terminal states.
- [x] 2.3 Ensure failure metadata updates are reflected in persisted checkpoints before returning errors.
- [x] 2.4 Enforce `SagaDefinition.MaxConcurrent` with per-execution step concurrency control in execute/resume paths.
- [x] 2.5 Add integration tests proving resumed execution uses updated checkpoint state and respects `MaxConcurrent`.

## 3. Compensation Concurrency Safety

- [x] 3.1 Add synchronization for shared compensation bookkeeping (`Compensated`, failure fields, timestamps) under parallel compensation.
- [x] 3.2 Ensure compensation bookkeeping remains correct with retries and mixed success/failure outcomes.
- [x] 3.3 Add/extend tests for parallel compensation layers and deterministic compensated-step recording.
- [x] 3.4 Run `go test -race ./pkg/saga/...` and fix remaining race issues in Saga paths.

## 4. API and gRPC Lifecycle Behavior

- [x] 4.1 Update HTTP compensate/recover handlers to resolve definitions from durable store before fallback cache.
- [x] 4.2 Update gRPC compensate flow to resolve definitions from durable store before fallback cache.
- [x] 4.3 Return explicit not-found/precondition errors when definition snapshots are missing.
- [x] 4.4 Add HTTP and gRPC tests for post-restart manual compensate/recover scenarios.

## 5. Validation and Documentation

- [x] 5.1 Update `docs/saga-guide.md` with restart-safe recovery guarantees and missing-definition behavior.
- [x] 5.2 Add regression tests for startup recovery with persisted definitions and incomplete checkpoints.
- [x] 5.3 Run `go test ./...` and `openspec validate --changes --strict`.
- [x] 5.4 Record final verification notes in change artifacts before apply/archive.

## 6. Final Verification Notes (2026-03-02)

- `go test ./...` passed after all phase updates.
- `go test -race ./pkg/saga/...` passed after compensation bookkeeping and synchronization updates.
- `openspec validate --changes --strict` passed:
  - `change/fix-week11-distributed-transactions-gaps`
  - totals: `1 passed, 0 failed`
