## 1. Engine cancellation and timeout conformance

- [x] 1.1 Audit current engine/task-runner cancellation and timeout paths against this change spec scenarios.
- [x] 1.2 Implement or adjust runtime logic in `pkg/engine` so cancellation/deadline outcomes converge on `cancelled` terminal semantics.
- [x] 1.3 Add or update unit tests for task-runner cancellation and per-task timeout enforcement.
- [x] 1.4 Run package compile/tests for engine conformance scope.

## 2. Scheduler fail-fast and layer barrier verification

- [x] 2.1 Verify scheduler behavior for unrecoverable layer failure and subsequent-layer blocking.
- [x] 2.2 Implement or adjust scheduler logic only if current behavior diverges from conformance requirement.
- [x] 2.3 Add or update scheduler-focused tests for fail-fast and layer barrier determinism.
- [x] 2.4 Run package compile/tests for scheduler conformance scope.

## 3. CLI signal shutdown conformance

- [x] 3.1 Verify current CLI signal handling path (`SIGINT`/`SIGTERM`) and identify any missing controlled shutdown guarantees.
- [x] 3.2 Implement or adjust `cmd/goclaw` shutdown wiring as needed for graceful signal-triggered stop.
- [x] 3.3 Add or update CLI/runtime tests for signal-driven shutdown behavior.
- [x] 3.4 Run relevant compile/tests for CLI + integrated engine shutdown path.
