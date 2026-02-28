## Why

Week4 archive baseline has been clarified, but several runtime behaviors still need explicit code-level conformance work. We need a dedicated implementation change to close those gaps with tests before claiming Week4 spec compliance.

## What Changes

- Implement and verify engine/task runtime behavior against Week4 conformance targets for cancellation and timeout semantics.
- Ensure scheduler/task-runner state transitions remain deterministic under cancellation, timeout, and fail-fast paths.
- Validate CLI bootstrap shutdown path for termination signals with explicit tests.
- Add focused unit/integration coverage for all conformance items.

## Capabilities

### New Capabilities
- `week4-engine-spec-conformance`: Defines implementation conformance requirements and testable scenarios for Week4 engine-core behavior.

### Modified Capabilities
- None.

## Impact

- Code paths: `pkg/engine/*`, `cmd/goclaw/*` (as needed by implementation).
- Tests: engine and CLI runtime tests for cancellation/timeout/signal shutdown semantics.
- No changes to archived change artifacts.
