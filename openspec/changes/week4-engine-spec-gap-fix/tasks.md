## 1. Shutdown Admission Guard

- [x] 1.1 Add explicit engine shutdown-transition admission control in `pkg/engine` lifecycle state handling.
- [x] 1.2 Ensure `Submit` rejects requests once stop has begun and before final stopped state is reached.
- [x] 1.3 Add regression tests for submit rejection during shutdown transition.
- [x] 1.4 Run focused `pkg/engine` tests for lifecycle admission behavior.

## 2. Cancellation Outcome Mapping

- [ ] 2.1 Update workflow terminal-status mapping so cancellation/deadline-driven task outcomes resolve to cancellation semantics.
- [ ] 2.2 Keep generic non-cancellation task failures mapped to failed semantics.
- [ ] 2.3 Add regression tests for per-task timeout/cancellation workflow outcome mapping.
- [ ] 2.4 Run focused `pkg/engine` tests for timeout/cancellation outcome behavior.

## 3. Conformance Verification

- [ ] 3.1 Run package-level compile/tests for affected paths (`pkg/engine`, and `cmd/goclaw` if impacted).
- [ ] 3.2 Verify updated behaviors satisfy week4-engine-spec-conformance delta scenarios.
- [ ] 3.3 Prepare sync/archive steps after implementation completion.
