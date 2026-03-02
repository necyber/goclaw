## Why

Week 10 monitoring code currently has several behavior gaps versus its own archived monitoring requirements. These gaps reduce observability correctness and can hide queueing latency or create avoidable metric-cardinality risk in production.

## What Changes

- Align lane metrics behavior so redirected submissions preserve enqueue timestamp semantics and lane wait duration is recorded consistently.
- Align task metrics schema and runtime integration to include `task_type` dimensions where required.
- Align workflow active-count behavior to track `pending` and `running` states consistently during lifecycle transitions.
- Align HTTP metrics semantics for status grouping and path normalization, and add bounded-cardinality safeguards for labels.
- Add explicit metrics endpoint configuration hardening (validation and startup guardrails) for invalid paths.

## Capabilities

### New Capabilities

- `monitoring-spec-conformance`: Cross-capability conformance contract and acceptance checks for monitoring metrics behavior.

### Modified Capabilities

- `lane-metrics`: Fix redirect-path queueing instrumentation so wait duration is captured consistently.
- `task-metrics`: Add and enforce `task_type` label semantics for execution, duration, and retry metrics.
- `workflow-metrics`: Enforce pending/running active workflow gauge transitions.
- `http-metrics`: Normalize request paths and status-label semantics to match monitoring requirements.
- `prometheus-metrics`: Add bounded cardinality controls and metrics endpoint safety constraints.

## Impact

- Affected code: `pkg/metrics/*`, `pkg/lane/channel_lane.go`, `pkg/engine/workflow_manager.go`, `pkg/api/middleware/metrics.go`, `config/*`, and related tests.
- Affected behavior: Prometheus time-series shape and label sets for task/workflow/http/lane metrics.
- Compatibility: additive/minor semantic adjustments; existing dashboards may require query updates for normalized status/task_type labels.
- Risk: low-to-medium; most changes are instrumentation and validation, but metric names/labels must be migrated carefully.
