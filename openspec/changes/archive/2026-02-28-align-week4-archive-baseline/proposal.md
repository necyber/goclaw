## Why

`week4-engine-core` archive artifacts currently describe conflicting requirements and completion status across `proposal.md`, `design.md`, `tasks.md`, and `specs/*`. This inconsistency blocks reliable review, and it also creates ambiguity for follow-up implementation changes that need a single baseline.

## What Changes

- Create a documentation-only baseline alignment package under this new change to reconcile `week4-engine-core` archive intent and requirement interpretation.
- Define an explicit conflict-resolution record for known mismatches (`cancelled` state, per-task timeout semantics, scheduler concurrency wording, CLI signal wiring, and completion-status wording).
- Produce superseding alignment artifacts that state the canonical interpretation without editing archived files.
- Keep `openspec/specs/*` and production code unchanged in this change; implementation work is deferred to separate follow-up changes.

## Capabilities

### New Capabilities

- `week4-archive-baseline-alignment`: Define how archived Week4 engine-core documents are reconciled into a single reviewable baseline, including conflict resolution rules and traceability mapping.

### Modified Capabilities

- None.

## Impact

- Affected scope is limited to `openspec/changes/align-week4-archive-baseline/*`.
- No direct edits to `openspec/changes/archive/week4-engine-core/*`.
- No edits to canonical specs under `openspec/specs/*`.
- No runtime/code-path changes in this change.
