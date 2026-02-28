## Context

The current lane runtime mostly aligns with Week3 behavior, but three contract-level gaps remain:

1. Canonical backpressure outcomes are counted inside lane structs but are not exposed as first-class Prometheus metrics in `pkg/metrics`.
2. Channel-lane wait duration recording depends on an optional `EnqueuedAt()` task method, so common `TaskFunc` submissions do not reliably produce wait observations.
3. Redirect outcome accounting is recorded before redirect submission result is known, which can classify failed redirects as successful `redirected`.

These are cross-cutting across runtime (`pkg/lane`) and observability (`pkg/metrics`) and require explicit spec deltas plus coordinated code/test updates.

## Goals / Non-Goals

**Goals:**
- Expose canonical lane submission outcomes (`accepted`, `rejected`, `redirected`, `dropped`) as metrics with lane-level labels.
- Ensure wait duration is measured from enqueue to dequeue for standard in-memory lane submissions.
- Ensure redirect accounting semantics reflect redirect success/failure correctly.
- Add regression tests that validate behavior and metrics coupling.

**Non-Goals:**
- No redesign of lane public interfaces (`Lane`, `Task`) beyond minimal internal compatibility extensions.
- No change to broader workflow/scheduler architecture.
- No changes to Redis queue data model beyond accounting timing adjustments.

## Decisions

### Decision 1: Add explicit outcome metrics in `pkg/metrics`
- Add a dedicated counter vec for submission outcomes keyed by `lane_name` and `outcome`.
- Implement `RecordSubmissionOutcome(laneName, outcome)` on metrics manager so existing lane hooks can write canonical outcomes.
- Rationale: lane runtime already emits canonical outcome events; wiring a concrete metrics surface closes the spec gap with minimal API churn.
- Alternative considered: rely only on lane-local `Stats` fields. Rejected because spec requires metrics exposure and operational scrapeability.

### Decision 2: Record enqueue timestamp in ChannelLane path
- Wrap queued tasks in an internal envelope (task + enqueue time) or equivalent internal timestamp mapping.
- On worker dequeue, compute wait duration from stored enqueue time and always record for accepted submissions.
- Keep support for externally supplied `EnqueuedAt()` tasks as optional compatibility behavior, but internal timestamp is the baseline.
- Rationale: avoids modifying external `Task` contract while guaranteeing consistent wait accounting.
- Alternative considered: extend `Task` interface with `EnqueuedAt()`. Rejected because it is a breaking interface change.

### Decision 3: Record `redirected` only on successful redirect submission
- Move redirected accounting to execute only after `targetLane.Submit(...)` returns nil.
- On redirect failure, classify outcome as `dropped` or `rejected` according to path semantics; never count as successful redirect.
- Apply same rule to both ChannelLane and RedisLane.
- Rationale: preserves one-event-one-outcome semantics and prevents inflated redirect success metrics.
- Alternative considered: dual-count failed redirect attempts (`redirected_attempt` + terminal outcome). Rejected for now because spec canonical set is fixed and simpler.

## Risks / Trade-offs

- [Risk] Additional metrics cardinality (`outcome` label) increases time series count.  
  Mitigation: bounded four-value label set (`accepted|rejected|redirected|dropped`).

- [Risk] Internal enqueue timestamp wrapper can add minor allocation overhead.  
  Mitigation: keep wrapper lightweight and lane-local; avoid map contention by carrying timestamp with payload.

- [Risk] Redirect failure reclassification may change dashboards that implicitly treated attempts as success.  
  Mitigation: document semantic correction in change notes and adjust alerts to use canonical definitions.

## Migration Plan

1. Add spec deltas for `lane-metrics` and `channel-lane-runtime`.
2. Implement metrics manager outcome counter and recorder method.
3. Implement ChannelLane enqueue-time tracking for wait metrics.
4. Adjust redirect accounting timing in ChannelLane and RedisLane.
5. Add/adjust unit tests for outcome metrics, wait duration recording, and redirect failure accounting.
6. Run `go test ./pkg/lane ./pkg/metrics` and verify metric names/labels.

Rollback:
- Revert the change commit to restore prior behavior; no schema/data migration required.

## Open Questions

- Should failed redirect-to-target submission be uniformly `rejected` vs policy-dependent terminal outcome? This change keeps current policy-specific semantics and only fixes misclassification as successful redirect.
