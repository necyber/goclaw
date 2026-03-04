## 1. Runtime Canonical Publish Wiring

- [x] 1.1 Add composite runtime broadcaster path that keeps local broadcast behavior and publishes canonical workflow/task lifecycle events via `eventbus.Publisher`
- [x] 1.2 Wire canonical publication from persisted workflow/task transition hooks in distributed mode without blocking persistence completion
- [x] 1.3 Add runtime tests that assert persisted transitions trigger canonical workflow/task publish and preserve local subscriber delivery

## 2. Streaming Bridge Startup and Typed Translation

- [x] 2.1 Attach eventbus bridge in production startup/bootstrap when distributed event transport is enabled
- [x] 2.2 Implement envelope decode and translation to handler-compatible `engine.WorkflowEvent`/`engine.TaskEvent` payloads before registry broadcast
- [x] 2.3 Add bridge/streaming integration tests for cross-node publish->bridge->stream delivery and unsupported schema decode telemetry

## 3. NATS Degraded-Mode Semantics

- [x] 3.1 Ensure publish failures follow bounded retry and degraded-mode telemetry policy while execution continues locally
- [x] 3.2 Ensure recovery path clears degraded indicators and resumes canonical publication after transport restoration
- [x] 3.3 Add tests for transient publish failure, retry behavior, and degraded-to-recovered transitions

## 4. Coordination Backend Guardrails

- [x] 4.1 Enforce explicit unsupported handling for `etcd`/`consul` modes when real backend implementations are unavailable
- [x] 4.2 Remove silent in-memory coordinator emulation from non-test distributed runtime paths unless explicitly gated for dev/test override
- [x] 4.3 Add adapter/bootstrap tests covering supported memory mode, unsupported backend fail-fast, and explicit override behavior

## 5. Shard-Scoped Transfer Idempotency

- [x] 5.1 Change transfer duplicate suppression/completion keying from global `workloadID` to `(shardKey, workloadID)`
- [x] 5.2 Update ownership transfer flow to preserve one-terminal-outcome guarantees with shard-scoped dedupe semantics
- [x] 5.3 Add tests proving same workload IDs on different shards do not collide while same-shard duplicates remain suppressed

## 6. End-to-End Regression Coverage

- [ ] 6.1 Add end-to-end regression tests validating canonical lifecycle publication reaches remote streaming subscribers in distributed mode
- [ ] 6.2 Add observability assertions for bridge decode failures, degraded mode transitions, and recovery signals
- [ ] 6.3 Run focused test suites (`eventbus`, `grpc streaming`, `cluster adapters`, `transfer`, `startup wiring`) and resolve regressions
