# week8-distributed-lane TODO

Last updated: 2026-02-26

## Remaining Task Items

- [ ] 19.7 Verify test coverage > 80%
  - Current check result: not met.
  - `go test ./... -coverprofile=coverage.out` showed multiple packages below 80%.
  - Coverage run also has a flaky cleanup failure in `config/TestWatcher_Watch` under cover mode.

## Remaining Acceptance Risks

- [ ] Quality threshold: unit test coverage > 80% (overall) is not yet satisfied.
- [ ] Performance targets not yet validated against acceptance thresholds:
  - Redis lane submit latency `< 5ms`
  - Redis lane throughput `> 10K tasks/s`
  - Signal bus latency `< 2ms (local)` / `< 5ms (redis)`

## Suggested Next Actions

1. Stabilize `config` watcher tests under coverage mode.
2. Add focused coverage tests for low-coverage packages (`pkg/lane`, `pkg/signal`, `pkg/engine`, `pkg/grpc/client`).
3. Re-run coverage and confirm aggregate > 80%.
4. Run controlled benchmarks (larger `-benchtime`, fixed input size) and compare with acceptance thresholds.
