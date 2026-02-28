## 1. Metrics Capability Alignment

- [x] 1.1 Add canonical backpressure outcome counters (`accepted/rejected/redirected/dropped`) to `pkg/metrics` lane instrumentation with `lane_name` and `outcome` labels.
- [x] 1.2 Expose and wire `RecordSubmissionOutcome` in metrics manager so lane runtime outcome hooks are exported through Prometheus.

## 2. Lane Runtime Accounting Corrections

- [ ] 2.1 Implement enqueue-time tracking in `ChannelLane` so wait duration is recorded for standard lane submissions even when tasks do not implement `EnqueuedAt()`.
- [ ] 2.2 Update redirect accounting in `ChannelLane` and `RedisLane` so `redirected` is counted only on successful redirect submission and failed redirects are not misclassified.

## 3. Conformance Tests and Validation

- [ ] 3.1 Add/update tests in `pkg/lane` and `pkg/metrics` for outcome metrics exposure, wait-duration recording, and redirect-failure accounting.
- [ ] 3.2 Run targeted Go tests (`./pkg/lane`, `./pkg/metrics`) and ensure all new behaviors are covered and passing.
