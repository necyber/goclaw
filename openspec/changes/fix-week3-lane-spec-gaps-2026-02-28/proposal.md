## Why

Week3 lane implementation and the canonical specs have drifted in a few behavior-critical areas: backpressure outcome observability, wait-duration accounting, and redirect outcome classification. These gaps should be corrected now to keep runtime behavior and metrics contracts auditable and consistent.

## What Changes

- Align lane metrics capability with canonical backpressure outcome semantics by requiring explicit metric exposure for `accepted`, `rejected`, `redirected`, and `dropped`.
- Clarify and enforce wait-duration accounting so enqueue-to-execution latency is recorded for normal in-memory lane submissions, not only for task types with custom timestamp fields.
- Tighten redirect outcome accounting so `redirected` is recorded only for successful redirect submissions, and failed redirects are not misclassified as successful redirects.
- Add conformance tests that validate the above semantics for ChannelLane and RedisLane paths where applicable.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `channel-lane-runtime`: refine redirect and wait-accounting behavior so runtime outcomes and accounting semantics match requirement intent.
- `lane-metrics`: require canonical backpressure outcome counters to be exposed by metrics instrumentation and bound to lane submission outcomes.

## Impact

- Affected code: `pkg/lane/*`, `pkg/metrics/*`, and corresponding test files.
- No public API breaking changes; behavior and observability semantics are made stricter.
- Improves runtime/spec conformance and reduces ambiguity for monitoring and operations.
