## Why

The archived change `week3-lane-queue-system` contains internal drift across `proposal.md`, `design.md`, `tasks.md`, and backfilled specs, specifically on limiter scope, worker model wording, requirement traceability, and backpressure metrics semantics. We need a traceable correction change now so archive records remain reliable for audits and future maintenance.

## What Changes

- Align rate-limiter scope language across week3 artifacts so Token Bucket is the normative requirement, and Leaky Bucket is clearly marked as optional extension/non-blocking enhancement.
- Align worker-pool terminology across week3 artifacts to remove fixed-vs-dynamic contradiction and define a single canonical statement.
- Add explicit requirement-traceability statements in week3 design/tasks for currently implicit spec requirements:
  - priority queue deterministic tie-breaking
  - lane manager concurrent read/write safety
  - lane lifecycle repeated-close safety
- Align backpressure metrics wording so accepted/rejected/redirected/dropped counters are consistently defined across artifacts.
- Keep this change documentation-only for archived artifacts; no runtime code changes.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `spec-format-governance`: Extend archived-change consistency governance to require requirement-traceable alignment when repairing archived artifact drift.

## Impact

- Documentation only; no API or runtime behavior changes.
- Affects archived files under `openspec/changes/archive/week3-lane-queue-system/`.
- Adds a governance delta spec under this change for consistency-repair traceability.
