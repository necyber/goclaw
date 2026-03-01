## Why

Week8 distributed-lane implementation has correctness and concurrency gaps that can cause false-success workflow completion, signal delivery races, and unstable collect behavior. These issues impact production safety in distributed mode and must be fixed before further roadmap phases rely on Redis lane and signal patterns.

## What Changes

- Tighten Redis lane execution semantics so tasks submitted through engine Redis queue mode are actually executed by workers, not only dequeued and counted.
- Strengthen Redis lane distributed safety for capacity accounting and dedup cleanup on ownership/fencing failure paths.
- Harden Redis signal bus unsubscribe/close concurrency behavior to prevent send-on-closed-channel panics.
- Fix collect pattern fan-in behavior to avoid starvation/blocking on per-task channels and reliably return partial results on timeout.
- Clarify fallback degradation classification to avoid treating non-connectivity lane/business errors as Redis health failures.
- Add deterministic regression coverage for the above scenarios (execution path, race-prone unsubscribe, collect timeout partial results).

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `redis-lane`: tighten worker execution contract, distributed backpressure accounting, and dedup/fencing safety requirements.
- `signal-bus`: require race-safe unsubscribe/close behavior under concurrent publish/forward operations.
- `message-patterns`: require collect fan-in fairness and deterministic partial-result behavior under timeout.

## Impact

- Affected code: `pkg/lane/*`, `pkg/engine/*` (Redis lane integration), `pkg/signal/*`, and related tests.
- APIs/contracts: no new public API required; behavior and error semantics are tightened for existing APIs.
- Dependencies/systems: Redis-backed lane and Redis Pub/Sub signal mode behavior in single-node and distributed deployments.
