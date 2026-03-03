## Why

`week14-cluster-event-bus` defines distributed coordination and canonical event-bus contracts, but key runtime integration paths are not actually connected. This creates a false-conformance state where tests pass locally while cross-node event visibility, stream bridging, and ownership correctness are not guaranteed in production paths.

## What Changes

- Wire canonical lifecycle event publication into runtime workflow/task transition hooks using the existing `eventbus.Publisher` abstraction.
- Add startup/runtime wiring so streaming service can consume canonical event-bus updates in production (not only in isolated tests).
- Align streaming bridge payload conversion with gRPC streaming handlers so bridge-delivered events are interpreted as workflow/task lifecycle updates instead of being silently skipped.
- Tighten coordination backend semantics: eliminate implicit in-memory behavior for `etcd`/`consul` modes in non-test runtime paths, with explicit degraded/unsupported handling.
- Fix ownership transfer idempotency scope to avoid cross-shard suppression collisions for identical workload IDs.
- Add regression coverage for end-to-end publish->bridge->stream flow, degraded-mode telemetry transitions, and shard-scoped transfer idempotency.

## Capabilities

### New Capabilities
- `runtime-eventbus-integration`: Runtime transition-to-canonical-event publication contract and startup wiring expectations.

### Modified Capabilities
- `nats-event-bus`: Require runtime transition hooks to publish canonical workflow/task lifecycle events in distributed mode.
- `streaming-support`: Require production bridge wiring and bridge-to-handler payload compatibility for cross-node workflow/task updates.
- `cluster-coordination`: Require backend identity correctness (no silent memory emulation for etcd/consul distributed mode).
- `workflow-sharding`: Require shard-scoped transfer idempotency and duplicate suppression semantics.

## Impact

- Affected code: `cmd/goclaw/main.go`, runtime broadcaster wiring, `pkg/eventbus/*`, `pkg/grpc/streaming/eventbus_bridge.go`, `pkg/grpc/handlers/streaming.go`, `pkg/cluster/adapters.go`, `pkg/cluster/transfer.go`.
- Affected tests: event-bus publish/consume integration, streaming bridge integration, cluster transfer unit/integration tests, startup wiring tests.
- Affected observability: event-bus degraded/recovery metrics and ownership-change instrumentation.
- Dependencies: coordination backend behavior checks and event transport wiring in distributed mode.
