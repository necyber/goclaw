## 1. Redis Lane Execution Correctness

- [ ] 1.1 Add a Redis lane executable-binding path so worker execution in engine Redis mode runs actual task logic instead of metadata-only completion.
- [ ] 1.2 Update Redis worker success/failure accounting to prevent completed++ on unresolved executable bindings.
- [ ] 1.3 Add regression test covering engine workflow submission with Redis queue mode to prove task function side effects occur before success.

## 2. Distributed Admission and Dedup Safety

- [ ] 2.1 Refine Redis lane admission checks to use authoritative Redis queue depth at capacity boundary in distributed producer scenarios.
- [ ] 2.2 Ensure dedup key cleanup is executed on fencing-validation failure and other terminal dequeue-failure paths.
- [ ] 2.3 Add/extend unit tests for redirect/drop/block outcomes and fencing-failure dedup cleanup behavior.

## 3. Fallback Degradation Classification

- [ ] 3.1 Narrow fallback Redis error classification to connectivity/transport failures and exclude lane-domain/business errors.
- [ ] 3.2 Add tests proving duplicate/full/dropped/validation errors do not trigger degraded mode while connectivity failures do.

## 4. Signal Bus Concurrency Hardening

- [ ] 4.1 Refactor Redis signal subscription lifecycle so channel close ownership is race-safe under concurrent forward/unsubscribe/close.
- [ ] 4.2 Add regression tests that concurrently publish/unsubscribe/close and assert no panic and correct teardown behavior.
- [ ] 4.3 Validate touched signal packages with `go test -race` and fix any race reports in production code paths.

## 5. Collect Fan-in Determinism

- [ ] 5.1 Replace collector per-channel blocking loop with fair fan-in aggregation that does not starve ready task results.
- [ ] 5.2 Ensure timeout returns deterministic partial results for completed tasks plus an explicit incomplete/timeout error.
- [ ] 5.3 Add deterministic tests for timeout partial results and slow-channel fairness (including repeat runs to catch flakiness).

## 6. Verification and Documentation

- [ ] 6.1 Run focused test suites for `pkg/lane`, `pkg/signal`, and relevant engine integration tests after fixes.
- [ ] 6.2 Update distributed-lane and signal behavior docs if runtime semantics changed (especially Redis execution and collect timeout behavior).
- [ ] 6.3 Confirm OpenSpec change validates cleanly and mark completed tasks in this file during implementation.
