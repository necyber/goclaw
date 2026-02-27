## Why

Current runtime behavior is partially implemented but not fully connected: workflow submission, execution scheduling, state persistence, and query/cancel semantics are not yet a single closed loop. This causes mismatch between API contract, runtime behavior, and observability.

## What Changes

- Define and implement an execution pipeline that guarantees `submit -> schedule -> run -> persist state -> query/cancel` continuity.
- Add explicit workflow/task state transition rules, including cancellation and timeout paths.
- Integrate Scheduler with Lane Manager as the required dispatch path (instead of direct goroutine execution in scheduler flow).
- Define runtime-facing API behavior for synchronous/asynchronous submission modes and terminal/non-terminal status responses.
- Complete gRPC runtime wiring requirements so service registration and engine adapter integration are first-class runtime requirements.
- Align lifecycle metrics requirements with actual persisted transitions and runtime outcomes.

## Capabilities

### New Capabilities
- `execution-pipeline`: End-to-end workflow runtime pipeline and invariants from submission to terminal states.
- `scheduler-lane-integration`: Scheduler dispatch contract that routes executable tasks through Lane Manager with defined completion signaling.
- `workflow-runtime-api`: Runtime API behavior for submission mode, execution visibility, cancellation semantics, and status consistency.

### Modified Capabilities
- `grpc-server`: Strengthen requirements for concrete service registration and runtime wiring with engine adapters.
- `streaming-support`: Clarify event stream behavior against real execution lifecycle and persisted state changes.
- `workflow-metrics`: Align workflow metric triggers with persisted lifecycle transitions and terminal outcomes.
- `task-metrics`: Align task metric triggers with scheduling/running/retry/cancel/timeout transitions.

## Impact

- Affected runtime modules: engine, scheduler, lane integration, state tracking/persistence path.
- Affected API surfaces: HTTP workflow endpoints, gRPC service runtime behavior, streaming semantics.
- Affected observability: workflow/task metrics and event stream consistency.
- Affected docs/specs: new capability specs above plus delta updates for modified capabilities.
