## 1. Metrics Core and Interface Updates

- [x] 1.1 Extend task metrics interfaces to accept `task_type` for execution, duration, and retry recording.
- [x] 1.2 Implement bounded label-cardinality guard utilities in `pkg/metrics` with dropped-label accounting metric.
- [x] 1.3 Add metrics-path validation rules to config validation for empty/invalid path values.
- [x] 1.4 Update Prometheus metrics manager wiring to use validated metrics path and cardinality guard.

## 2. Lane and Engine Lifecycle Instrumentation Fixes

- [x] 2.1 Update `ChannelLane` redirect acceptance path to use the same enqueue wrapper contract as other accepted paths.
- [x] 2.2 Add/adjust lane metrics recording to ensure wait duration is emitted for redirect fast-path accepted tasks.
- [x] 2.3 Update workflow lifecycle instrumentation to track `workflow_active_count{status="pending"}` transitions.
- [x] 2.4 Ensure pending/running active gauges remain balanced for success, failure, cancellation, and pre-run failure paths.
- [x] 2.5 Resolve task type from runtime task metadata and pass normalized `task_type` to task metrics calls.

## 3. HTTP Metrics Normalization and Safety

- [ ] 3.1 Normalize HTTP status labels to status classes (`2xx|3xx|4xx|5xx`) before metrics emission.
- [ ] 3.2 Expand HTTP path normalization to cover UUID, ULID, numeric IDs, and long opaque token segments.
- [ ] 3.3 Keep metrics-endpoint recursion exclusion behavior aligned with configured metrics path semantics.
- [ ] 3.4 Route normalized HTTP labels through cardinality guard to drop unsafe new values.

## 4. Tests and Conformance Coverage

- [ ] 4.1 Add lane tests proving redirect fast-path submissions still produce wait duration metrics.
- [ ] 4.2 Add task metrics tests proving `task_type` labels are emitted and bounded fallback is deterministic.
- [ ] 4.3 Add workflow metrics tests for pending/running gauge increments and decrements across terminal branches.
- [ ] 4.4 Add HTTP middleware tests for status-class normalization and extended dynamic-path normalization.
- [ ] 4.5 Add metrics manager tests for cardinality-limit behavior, warning/drop behavior, and dropped-label counters.
- [ ] 4.6 Add config validation tests for invalid metrics path values.

## 5. Documentation and Compatibility Notes

- [ ] 5.1 Update `docs/monitoring-guide.md` with new label semantics (`task_type`, HTTP status class, path normalization rules).
- [ ] 5.2 Add PromQL migration examples for dashboards/alerts impacted by label-semantic changes.
- [ ] 5.3 Update README monitoring section with compatibility caveats and recommended queries.
- [ ] 5.4 Run `openspec validate --changes --strict` and `go test ./...` before implementation handoff.
