## Why

The execution pipeline implementation currently passes tests but still diverges from spec-level runtime guarantees in several critical paths. We need a focused conformance fix so runtime state, streaming, and metrics semantics remain deterministic and externally trustworthy.

## What Changes

- Persist task terminal result payloads in the runtime transition path so completed task result APIs return stored data instead of empty payloads.
- Add explicit cancellation graceful-timeout handling for running workflows, including bounded wait semantics and deterministic timeout-derived terminal outcomes.
- Align streaming initial message behavior with persisted runtime state rather than emitting a synthetic fixed pending state.
- Tighten streaming delivery guarantees for per-workflow transitions under backpressure, especially terminal visibility and ordered transition consistency.
- Distinguish task timeout-derived outcomes from user-cancelled outcomes in terminal metrics labels.
- Add regression tests covering all above behaviors end-to-end across engine, HTTP, gRPC streaming, and metrics hooks.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `execution-pipeline`: enforce persisted terminal result payload contract and cancellation-timeout outcome behavior.
- `workflow-runtime-api`: enforce graceful cancellation timeout semantics and deterministic terminal result retrieval behavior.
- `streaming-support`: align initial stream state and backpressure behavior with persisted transition guarantees.
- `task-metrics`: enforce explicit timeout vs cancellation terminal labeling consistency.

## Impact

- Affected runtime modules: `pkg/engine/workflow_manager.go`, cancellation and transition helpers, runtime test suites.
- Affected APIs: HTTP task result and cancel semantics, gRPC streaming initial/update behavior.
- Affected observability: task terminal label semantics and stream delivery consistency under load.
- Affected tests: engine runtime execution tests, API handler tests, streaming registry/handler tests, metrics assertions.
