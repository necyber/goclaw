## Context

`week2-dag-compiler` is already archived, but its documents are inconsistent in naming, scope, and acceptance language. Because archived changes are used as historical evidence for audits and backfill reviews, inconsistent artifacts reduce trust and create conflicting interpretations.

## Goals / Non-Goals

**Goals:**
- Make `proposal.md`, `design.md`, `tasks.md`, and the 7 week2 spec files internally consistent.
- Define a clear correction policy for archived changes to preserve traceability.
- Keep fixes minimal and documentation-only.

**Non-Goals:**
- No code changes in `pkg/dag` or runtime behavior changes.
- No re-implementation of week2 features.
- No broad rewrite of unrelated archived changes.

## Decisions

1. Canonicalize cycle error terminology to `CyclicDependencyError` across week2 artifacts.
Rationale: Current mixed naming (`CycleError` vs `CyclicDependencyError`) creates direct contradiction.
Alternative considered: keep both names and add alias notes. Rejected because it preserves ambiguity.

2. Treat week2 as DAG-core-only scope in archived docs.
Rationale: Week2 specs cover DAG compiler behavior; workflow/scheduler integration references are out of scope and should be marked deferred.
Alternative considered: expand week2 specs to include workflow/scheduler. Rejected as unnecessary scope expansion.

3. Preserve archive integrity by recording semantic corrections through a new change.
Rationale: Editing archived artifacts without change records weakens historical traceability.
Alternative considered: directly patch archived files without governance rules. Rejected due to auditability risk.

4. Normalize acceptance language from hard environment thresholds to reproducible benchmark reporting where needed.
Rationale: Historic hard thresholds often lack fixed hardware context and cause contradictory pass/fail interpretation.
Alternative considered: keep strict numeric thresholds unchanged. Rejected for low reproducibility.

## Risks / Trade-offs

- [Risk] Governance wording may be interpreted as a blanket rule for all historical archives.  
  → Mitigation: Scope the new requirements to archived change correction process and preserve non-semantic errata allowance.

- [Risk] Minimal edits may leave some legacy phrasing imperfect.  
  → Mitigation: Prioritize contradiction removal (naming/scope/acceptance) and defer stylistic cleanup.

- [Risk] Teams may still patch archived files directly out of habit.  
  → Mitigation: Add explicit requirement and scenario for semantic-fix workflow in governance spec.
