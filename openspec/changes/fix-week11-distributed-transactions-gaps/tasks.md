## 1. Saga Definition Persistence

- [ ] 1.1 Add a durable Saga definition store in `pkg/saga` keyed by `saga_id`.
- [ ] 1.2 Persist definition snapshots from HTTP submit path (`pkg/api/handlers/saga.go`).
- [ ] 1.3 Persist definition snapshots from gRPC submit path (`pkg/grpc/handlers/saga.go`).
- [ ] 1.4 Wire engine startup recovery to build/load definition map from durable storage instead of passing an empty map.
- [ ] 1.5 Add unit tests for definition store save/load/delete/list behavior and error handling.

## 2. Orchestrator and Checkpoint Correctness

- [ ] 2.1 Introduce a centralized checkpoint persistence helper for lifecycle transitions in `pkg/saga/orchestrator.go`.
- [ ] 2.2 Persist checkpoints on transitions to `pending-compensation`, `compensating`, and terminal states.
- [ ] 2.3 Ensure failure metadata updates are reflected in persisted checkpoints before returning errors.
- [ ] 2.4 Enforce `SagaDefinition.MaxConcurrent` with per-execution step concurrency control in execute/resume paths.
- [ ] 2.5 Add integration tests proving resumed execution uses updated checkpoint state and respects `MaxConcurrent`.

## 3. Compensation Concurrency Safety

- [ ] 3.1 Add synchronization for shared compensation bookkeeping (`Compensated`, failure fields, timestamps) under parallel compensation.
- [ ] 3.2 Ensure compensation bookkeeping remains correct with retries and mixed success/failure outcomes.
- [ ] 3.3 Add/extend tests for parallel compensation layers and deterministic compensated-step recording.
- [ ] 3.4 Run `go test -race ./pkg/saga/...` and fix remaining race issues in Saga paths.

## 4. API and gRPC Lifecycle Behavior

- [ ] 4.1 Update HTTP compensate/recover handlers to resolve definitions from durable store before fallback cache.
- [ ] 4.2 Update gRPC compensate flow to resolve definitions from durable store before fallback cache.
- [ ] 4.3 Return explicit not-found/precondition errors when definition snapshots are missing.
- [ ] 4.4 Add HTTP and gRPC tests for post-restart manual compensate/recover scenarios.

## 5. Validation and Documentation

- [ ] 5.1 Update `docs/saga-guide.md` with restart-safe recovery guarantees and missing-definition behavior.
- [ ] 5.2 Add regression tests for startup recovery with persisted definitions and incomplete checkpoints.
- [ ] 5.3 Run `go test ./...` and `openspec validate --changes --strict`.
- [ ] 5.4 Record final verification notes in change artifacts before apply/archive.
