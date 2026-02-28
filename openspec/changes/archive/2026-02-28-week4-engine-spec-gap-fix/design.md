## Context

Review against week4 engine specs found behavior mismatches in runtime shutdown admission and cancellation outcome mapping. The implementation currently has most week4 capabilities, but these gaps create inconsistent behavior under shutdown and timeout-driven cancellation paths.

## Goals / Non-Goals

**Goals:**
- Reject new workflow submissions once engine shutdown starts.
- Ensure timeout/cancellation-driven task termination maps to cancellation-oriented workflow outcome semantics.
- Add deterministic regression tests for shutdown admission and cancellation mapping.

**Non-Goals:**
- No broad engine architecture refactor.
- No changes to archived change artifacts.
- No changes to unrelated scheduler/lane runtime behavior beyond required conformance.

## Decisions

### Decision 1: Introduce explicit stopping state admission guard
Add a distinct lifecycle state for shutdown in progress and block `Submit` while in this state.

Alternative considered:
- Keep only `running/stopped` and rely on lane close timing.
- Rejected because it allows a race where submissions are accepted after shutdown starts.

### Decision 2: Normalize timeout-derived failures to cancellation semantics
When task execution ends due to context cancellation/deadline, propagate cancellation semantics to workflow terminal status selection.

Alternative considered:
- Keep current rule based only on outer workflow context cancellation.
- Rejected because per-task timeout cancellation can be mislabeled as generic failure.

### Decision 3: Cover gaps with focused regression tests
Use targeted tests in `pkg/engine` for admission and cancellation mapping rather than broad end-to-end suites.

Alternative considered:
- Rely on existing integration tests without dedicated gap tests.
- Rejected because prior tests did not catch these edge-case semantics.

## Risks / Trade-offs

- [Risk] Adding a stopping state may affect callers that assume only idle/running/stopped/error.
  -> Mitigation: keep external `State()` mapping stable unless explicitly needed; document transition behavior.

- [Risk] Cancellation normalization may alter expected historical failure metrics.
  -> Mitigation: scope change to cancellation/deadline-specific error classes and preserve generic failures.

- [Risk] Race-sensitive shutdown tests may become flaky.
  -> Mitigation: use deterministic synchronization and bounded waits.

## Migration Plan

1. Implement engine stop-admission guard and transition ordering.
2. Adjust workflow status mapping for cancellation/deadline-driven task termination.
3. Add regression tests for both behaviors.
4. Run package-level compile/tests for `pkg/engine` and affected entrypoints.
5. Roll back by reverting change set if behavior regressions appear.

## Open Questions

- Should `State()` expose `stopping` explicitly or keep backward-compatible values only?
- Should cancellation normalization include lane submission cancellation failures beyond task runtime deadlines?
