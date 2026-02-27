## Context

The repository has mixed-generation OpenSpec main specs. Some specs already follow canonical format, while others still use legacy layouts (for example `## ADDED Requirements` at top level, or free-form narrative sections). OpenSpec v1.2.0 validates main specs against canonical sections and fails archive/sync when rebuilt specs are not structurally valid.

## Goals / Non-Goals

**Goals:**
- Normalize all legacy main specs to canonical structure (`## Purpose` + `## Requirements`).
- Preserve existing requirement/scenario semantics wherever possible.
- Ensure repository-level `openspec validate --specs` reaches zero failures.
- Unblock future default archive/sync operations without `--skip-specs`.

**Non-Goals:**
- Redesigning product/runtime requirements.
- Refactoring implementation code.
- Rewriting all requirement narratives for style consistency.

## Decisions

### 1. Use validator-driven migration scope
**Decision:** Use `openspec validate --specs --json` failing list as exact migration target set.
**Rationale:** Avoid touching already valid specs and keep migration auditable.
**Alternative considered:** Rewrite all specs regardless of status. Rejected due unnecessary churn.

### 2. Preserve requirement/scenario blocks verbatim when available
**Decision:** For legacy specs that already contain `### Requirement` blocks, retain those blocks and wrap with canonical top-level sections.
**Rationale:** Minimizes semantic drift and avoids accidental requirement edits.
**Alternative considered:** Re-author all requirements manually. Rejected due high risk and effort.

### 3. Use baseline requirement for narrative-only legacy specs
**Decision:** For specs lacking requirement/scenario blocks, create a baseline normative requirement+scenario and retain historical narrative in `## Notes`.
**Rationale:** Satisfies validator requirements while preserving prior context.
**Alternative considered:** Drop legacy narrative entirely. Rejected due information loss.

### 4. Validate to zero before considering migration complete
**Decision:** Completion gate is `openspec validate --specs` with no failures.
**Rationale:** Directly aligns with the operational problem being solved.
**Alternative considered:** Spot-check only impacted files. Rejected due hidden residual failures.

## Risks / Trade-offs

- **[Risk] Semantic dilution in baseline-converted specs** -> **Mitigation:** keep original narrative under `## Notes` for traceability.
- **[Risk] Large doc diff volume** -> **Mitigation:** deterministic, structure-focused conversion approach.
- **[Risk] Future drift back to legacy format** -> **Mitigation:** keep validator gate in workflow and use this change as reference pattern.
