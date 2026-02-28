## Why

Week3 archived documents were aligned, but runtime behavior and tests still need explicit code-level alignment with the canonical semantics. This change closes the documentation-to-implementation gap so lane behavior, metrics, and lifecycle guarantees are enforced by code and tests.

## What Changes

- Implement canonical ChannelLane runtime semantics in code:
  - configurable worker concurrency (fixed-by-default, optional dynamic scaling support)
  - Token Bucket as normative admission baseline
  - Leaky Bucket as optional extension behavior, not baseline gate
  - deterministic tie-breaking for equal-priority tasks
  - idempotent lane lifecycle close semantics
  - concurrent-safe lane manager operations
- Normalize backpressure accounting to explicit counters for `accepted`, `rejected`, `redirected`, and `dropped`.
- Add/adjust tests to verify the above semantics and prevent regression.
- Keep API-compatible behavior unless a spec delta explicitly requires a contract change.

## Capabilities

### New Capabilities
- `channel-lane-runtime`: Define normative in-memory channel-lane execution semantics, worker model, priority ordering behavior, lifecycle safety, and lane-manager concurrency guarantees.

### Modified Capabilities
- `lane-metrics`: Extend lane metrics requirements to include canonical backpressure outcome counters (`accepted/rejected/redirected/dropped`) with consistent update semantics.

## Impact

- Primary code impact under `pkg/lane/` and related tests.
- Potential minor integration touch points where scheduler/runtime expects lane submission behavior.
- No external dependency changes expected.
