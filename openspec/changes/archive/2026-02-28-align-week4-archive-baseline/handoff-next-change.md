# Handoff: Next Implementation-Focused Change

## Purpose

This note defines what must be implemented in a separate code-focused change after baseline alignment approval.

## Deferred Implementation Items

The following items require runtime/code updates and are out of scope for this change:

1. Add and validate `cancelled` task-state support end-to-end where required by Week4 archive specs.
2. Implement per-task timeout propagation in task runner execution path (`dag.Task.Timeout` or equivalent contract).
3. Ensure CLI bootstrap path includes explicit process signal handling (`SIGINT`/`SIGTERM`) and controlled engine shutdown wiring.
4. Reconcile/verify scheduler behavior documentation and implementation evidence for intra-layer concurrency, barrier, and fail-fast semantics.
5. Resolve completion-status ambiguity through code/test verification rather than archive checklist text.

## Suggested Next Change Scope

Suggested new change theme: `week4-engine-spec-conformance` (name can be adjusted by owner).

Suggested artifacts for the next change:

- Proposal:
  - Target: bring implementation into conformance with Week4 archive specs.
- Specs:
  - Either adopt existing Week4 archive backfill specs as implementation source of truth, or publish clarified canonical delta specs before coding.
- Design:
  - Include engine state model updates for `cancelled` and timeout/cancel propagation.
- Tasks:
  - Include implementation tasks plus unit/integration test coverage for all deferred items.

## Acceptance Handoff Checklist

- Baseline mismatch inventory approved.
- Canonical baseline/deferred tags approved for all mismatch entries.
- No archive files mutated.
- Implementation work explicitly tracked in a separate change.
