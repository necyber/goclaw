## Why

The archived change `week2-dag-compiler` has internal documentation drift across `proposal.md`, `design.md`, `tasks.md`, and its 7 spec files. This makes archival records hard to trust and creates ambiguity for later audits and backfill verification.

## What Changes

- Align terminology for cycle-related errors across artifacts (single canonical name).
- Align documented scope so archived week2 artifacts consistently describe DAG-core-only behavior.
- Align naming and acceptance criteria wording to remove contradictory API and performance expectations.
- Add governance requirements for how archived changes may be corrected (semantic fixes via new changes, non-semantic errata policy).

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `spec-format-governance`: Extend governance to cover consistency and correction policy for archived change artifacts.

## Impact

- Documentation only; no runtime behavior or API changes.
- Affects archived files under `openspec/changes/archive/week2-dag-compiler/`.
- Adds delta spec requirements under this change for `spec-format-governance`.
