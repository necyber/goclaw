## Why

The week4 engine implementation still has behavior gaps against declared specs, mainly around shutdown admission control and cancellation outcome consistency. A focused fix change is needed to align runtime behavior with spec intent and remove ambiguity before further engine evolution.

## What Changes

- Enforce stop-admission behavior so new submissions are rejected once shutdown begins.
- Align workflow cancellation semantics when task execution ends due to cancellation/deadline signals.
- Add deterministic regression coverage for shutdown admission and cancellation outcome paths.
- Clarify conformance expectations in change-level delta specs for the existing week4 engine capability.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `week4-engine-spec-conformance`: Tighten requirements for stop admission and cancellation outcome mapping during timeout/cancel paths.

## Impact

- Engine runtime state/submit/stop handling in `pkg/engine/*`.
- Related tests in `pkg/engine/*_test.go` and, if required, `cmd/goclaw/*_test.go`.
- Delta spec updates only for `week4-engine-spec-conformance`.
