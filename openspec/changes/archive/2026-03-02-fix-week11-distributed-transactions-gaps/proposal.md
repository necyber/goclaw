## Why

`2026-02-27-week11-distributed-transactions` landed the Saga feature set, but code review and race checks still show correctness gaps in compensation concurrency, checkpoint state freshness, and restart recovery. These gaps can cause duplicate execution, silent recovery skips, or undefined runtime behavior during failures.

## What Changes

- Persist executable Saga definitions by `saga_id` so recovery and manual operations do not depend on process-local memory maps.
- Make compensation bookkeeping concurrency-safe for parallel reverse compensation paths.
- Persist checkpoints on lifecycle state transitions (not only step completion) so restart recovery uses accurate Saga state.
- Enforce per-definition step concurrency (`MaxConcurrent`) during forward and resumed execution.
- Align HTTP and gRPC Saga operations with persistent-definition lookup and explicit failure behavior when definition data is unavailable.
- Add race-focused and crash-recovery-focused conformance tests for the above behavior.

## Capabilities

### New Capabilities
- `saga-definition-persistence`: Durable storage and retrieval contract for executable Saga definitions keyed by `saga_id`.

### Modified Capabilities
- `saga-orchestrator`: Enforce per-definition execution concurrency and use persisted definitions during recovery/manual operations.
- `saga-checkpoint`: Persist checkpoints on state transitions so recovery decisions are based on current lifecycle state.
- `saga-compensation`: Require concurrency-safe in-memory compensation bookkeeping for parallel compensation layers.
- `saga-api`: Resolve definitions from durable storage for compensation/recovery APIs instead of in-memory-only maps.

## Impact

- Affected code: `pkg/saga/*`, `pkg/engine/engine.go`, `pkg/api/handlers/saga.go`, `pkg/grpc/handlers/saga*.go`, `config/*`, and related tests.
- Affected behavior: startup recovery reliability, manual compensation/recovery behavior after restarts, and compensation runtime safety under parallel execution.
- Compatibility: additive at API contract level; error messages/codes for missing definitions may become stricter and more explicit.
- Operational impact: improved crash recovery determinism and reduced risk of silent Saga drift after node restarts.
