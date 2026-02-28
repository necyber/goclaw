## 1. Align channel-lane runtime semantics

- [x] 1.1 Update `pkg/lane/channel_lane.go` submission paths to enforce canonical outcome accounting (`accepted/rejected/redirected/dropped`)
- [x] 1.2 Ensure rate-limiter admission path keeps Token Bucket as baseline behavior and does not make Leaky Bucket a baseline gate
- [x] 1.3 Implement deterministic tie-breaking for equal-priority tasks in priority queue execution path
- [x] 1.4 Verify lane close lifecycle remains idempotent under repeated close calls

## 2. Harden manager/runtime safety behavior

- [x] 2.1 Review and adjust `pkg/lane/manager.go` for concurrent-safe register/get/submit/close invariants where needed
- [x] 2.2 Ensure worker concurrency behavior is explicit and testable for fixed-default and optional dynamic scaling modes
- [x] 2.3 Add/adjust runtime stats structures and plumbing to expose canonical backpressure outcomes for metrics integration

## 3. Expand test coverage for canonical semantics

- [x] 3.1 Add unit tests for equal-priority deterministic ordering
- [x] 3.2 Add unit tests for backpressure outcome accounting (`accepted/rejected/redirected/dropped`)
- [x] 3.3 Add unit tests for repeated close idempotency and manager concurrency safety edge cases
- [x] 3.4 Update or add lane-metrics tests/assertions to validate backpressure outcome counters

## 4. Validate and finalize

- [x] 4.1 Run `go test ./pkg/lane/...` and fix regressions
- [x] 4.2 Run `openspec validate --changes --strict` and fix spec/task formatting issues
- [x] 4.3 Record implementation summary and mark completed tasks in this file

## 5. Final Summary (2026-02-28)

- Implemented canonical channel-lane submission outcome accounting (`accepted/rejected/redirected/dropped`) and exposed counters in runtime stats.
- Kept Token Bucket as the normative channel-lane admission baseline and documented that path in submission code.
- Implemented deterministic equal-priority tie-breaking in priority queue via enqueue sequence ordering.
- Hardened manager lifecycle/concurrency invariants with explicit closed-state checks and lock-safe close flow.
- Added optional dynamic worker configuration (`EnableDynamicWorkers`, `MinConcurrency`) with fixed-size default behavior.
- Expanded regression tests for priority tie-break determinism, outcome accounting, idempotent close, manager concurrent close invariants, and dynamic worker selection.
- Validation complete:
  - `go test ./pkg/lane/... -race` => pass
  - `openspec validate --changes --strict --json` => 1 change passed, 0 failed
