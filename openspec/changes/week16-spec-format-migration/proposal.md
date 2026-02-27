## Why

OpenSpec CLI v1.2.0 enforces a canonical spec structure (`## Purpose` + `## Requirements`), while part of this repository still used legacy spec formats. This blocked normal archive flows that sync delta specs into main specs.

## What Changes

- Migrate all legacy-formatted main specs in `openspec/specs/*/spec.md` to the canonical structure required by current OpenSpec validation.
- Preserve existing `### Requirement` and `#### Scenario` content when already present.
- For legacy narrative-only specs without requirement/scenario blocks, add a baseline requirement+scenario and retain original content in `## Notes`.
- Make `openspec validate --specs` pass with zero failures.

## Capabilities

### New Capabilities
- `spec-format-governance`: Define and enforce a canonical OpenSpec spec structure migration strategy for the repository.

### Modified Capabilities
- None. This change standardizes specification document structure and does not introduce behavioral requirement changes for runtime features.

## Impact

- Affected assets: `openspec/specs/*/spec.md` (legacy formatted specs).
- Operational impact: unblocks default archive/sync workflows that validate rebuilt main specs.
- No runtime code, API surface, or dependency behavior changes.
