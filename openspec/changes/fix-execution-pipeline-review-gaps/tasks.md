## 1. Runtime Transition Conformance

- [x] 1.1 Persist `storage.TaskState.Result` on `completed` terminal transition in runtime execution path
- [x] 1.2 Ensure non-terminal and non-completed terminal transitions do not leak stale task result payloads
- [x] 1.3 Add/adjust engine tests validating completed task result payload persistence and retrieval

## 2. Cancellation Graceful Timeout Semantics

- [ ] 2.1 Add configurable cancellation graceful-timeout setting for workflow runtime cancel path
- [ ] 2.2 Implement bounded cancel flow: signal cancellation, wait for task settlement, enforce timeout-derived terminal mapping on expiry
- [ ] 2.3 Add integration tests for running-workflow cancel behavior within timeout and timeout-expiry cases

## 3. Streaming State Consistency and Backpressure

- [ ] 3.1 Replace fixed initial workflow stream state with persisted-state-derived initial snapshot
- [ ] 3.2 Implement terminal-priority behavior under streaming backpressure and explicit stream error on undeliverable terminal visibility
- [ ] 3.3 Add streaming tests for initial-state correctness, transition ordering, and terminal visibility under slow-consumer pressure

## 4. Task Metrics Outcome Labeling

- [ ] 4.1 Implement deterministic task terminal metric label mapping that distinguishes user cancellation from timeout-derived outcomes
- [ ] 4.2 Align timeout detection logic in task terminal metric emission path with runtime terminal policy
- [ ] 4.3 Add metrics tests asserting cancellation and timeout outcomes are query-distinguishable and idempotent per attempt

## 5. Regression and Documentation Hygiene

- [ ] 5.1 Add cross-module regression tests for HTTP task-result endpoint and workflow cancel contract alignment
- [ ] 5.2 Update runtime/streaming/metrics docs to reflect new conformance semantics and labels
- [ ] 5.3 Run focused test suites (`engine`, `api handlers`, `grpc handlers`, `grpc streaming`, `metrics`) and fix regressions
