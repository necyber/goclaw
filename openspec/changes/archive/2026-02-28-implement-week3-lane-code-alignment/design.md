## Context

Week3 lane documentation has been normalized, but several runtime guarantees remain only partially enforced or not explicitly verified by tests. Current `pkg/lane` code already includes most primitives (channel lane, worker pool, priority queue, manager, token bucket, leaky bucket), so this change is primarily semantic alignment and verification hardening rather than greenfield implementation.

The implementation must preserve existing public API compatibility while making canonical behavior explicit in code paths and test coverage.

## Goals / Non-Goals

**Goals:**
- Enforce canonical runtime semantics for channel-lane execution.
- Ensure deterministic ordering for equal-priority tasks.
- Enforce idempotent lane close lifecycle behavior.
- Ensure manager operations remain safe under concurrent register/get/submit/close usage.
- Normalize backpressure outcome accounting (`accepted/rejected/redirected/dropped`) and expose it for metrics integration.
- Add focused tests that lock in these semantics.

**Non-Goals:**
- No redesign of scheduler architecture or workflow DAG behavior.
- No distributed-lane protocol or ownership model changes.
- No new external dependencies.

## Decisions

1. Keep Token Bucket as the admission-control baseline in channel-lane submission path, and treat Leaky Bucket as optional extension only.
Rationale: This matches the normative scope in updated week3 docs and avoids retroactive baseline expansion.
Alternative: make both algorithms baseline-equal. Rejected because it blurs acceptance criteria.

2. Use explicit tie-breaker key (enqueue sequence) for equal-priority ordering.
Rationale: Heap priority alone is insufficient for deterministic behavior among equals; sequence key provides stable ordering.
Alternative: rely on heap incidental ordering. Rejected due to non-determinism across runs.

3. Treat close as idempotent and safe for repeated calls across lane/manager boundaries.
Rationale: Lifecycle safety is required by lane-interface acceptance notes and prevents shutdown race hazards.
Alternative: return error on repeated close. Rejected because it complicates orchestration shutdown logic.

4. Introduce explicit backpressure outcome accounting update points in submission paths.
Rationale: Outcome counters are needed to align runtime and metrics semantics (`accepted/rejected/redirected/dropped`).
Alternative: infer metrics indirectly from pending/completed/dropped counters. Rejected due to ambiguity (especially redirect vs reject).

5. Validate semantics through targeted unit tests instead of broad integration rewrites.
Rationale: This keeps changes small, localized, and suitable for regression prevention.
Alternative: end-to-end only verification. Rejected because failures become harder to isolate.

## Risks / Trade-offs

- [Risk] Additional counters may increase Stats surface complexity.  
  -> Mitigation: keep naming explicit and maintain backward compatibility for existing fields.

- [Risk] Deterministic tie-breaking may slightly alter task execution order in previously unspecified edge cases.  
  -> Mitigation: document canonical behavior and add tests to make the change intentional.

- [Risk] Concurrency-focused tests can be flaky under timing variance.  
  -> Mitigation: prefer deterministic synchronization (channels/waitgroups/atomic checks) over sleep-based assertions.

## Migration Plan

1. Add/adjust lane data structures for explicit outcome accounting and deterministic tie-breaker support.
2. Update submission and queue paths (`Submit`, `TrySubmit`, redirect/drop/block branches) to consistently record outcomes.
3. Add/adjust tests for:
   - equal-priority deterministic order
   - repeated close idempotency
   - manager concurrent safety invariants
   - backpressure outcome accounting correctness
4. Run focused tests (`go test ./pkg/lane/...`) and full change validation.

Rollback strategy: revert this change set; API compatibility is preserved, so rollback is low risk.

## Open Questions

- Should backpressure outcome counters be exposed directly in `lane.Stats` now, or first via metrics hooks only?  
  Current plan: expose in `Stats` to keep accounting auditable and testable.
