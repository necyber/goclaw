## Context

The archived change `openspec/changes/archive/week4-engine-core` contains internally inconsistent statements across `proposal.md`, `design.md`, `tasks.md`, and archive backfill specs. Reviewers cannot derive a single canonical interpretation for Week4 scope and acceptance from those files alone.  
This follow-up change is documentation-only and must not rewrite archive history files.

## Goals / Non-Goals

**Goals:**
- Establish a canonical, reviewable baseline for Week4 archive interpretation.
- Resolve previously identified contradictions in a deterministic way.
- Keep resolution traceable to exact source files/lines in archive artifacts.
- Preserve archive immutability while enabling future implementation follow-ups.

**Non-Goals:**
- No code changes.
- No edits to archived files under `openspec/changes/archive/week4-engine-core/*`.
- No edits to canonical specs under `openspec/specs/*`.
- No retroactive claim that unresolved Week4 items were implemented.

## Decisions

### Decision 1: Archive artifacts are immutable
We will not edit any file under `openspec/changes/archive/week4-engine-core/*`.

Rationale:
- Archive content is historical evidence and should remain auditable.
- Baseline alignment should be additive and traceable.

Alternative considered:
- Edit archived files directly to make them consistent.
- Rejected because it rewrites history and blurs provenance.

### Decision 2: Canonical baseline is produced as additive artifacts in this change
This change will define a superseding baseline using its own `proposal/design/specs/tasks`.

Rationale:
- Keeps correction context explicit and reviewable.
- Supports incremental handoff to follow-up implementation changes.

Alternative considered:
- Encode corrections in ad-hoc notes outside OpenSpec artifacts.
- Rejected because it weakens traceability and workflow discipline.

### Decision 3: Conflict resolution is normative and requirement-driven
The new spec for this change will define MUST/SHALL-level requirements for:
- Conflict inventory completeness.
- Canonical resolution record.
- Traceability mapping.
- Explicit deferred-work declarations.

Rationale:
- Converts prior review findings into testable artifact requirements.
- Prevents future ambiguity about what is baseline vs deferred.

Alternative considered:
- Keep guidance as non-normative recommendations only.
- Rejected because ambiguity already caused inconsistency.

### Decision 4: Deferred implementation is explicit
Any behavior gap requiring code changes (for example, Week4 spec conformance fixes) is declared as deferred to separate implementation changes.

Rationale:
- Matches stakeholder instruction for this change boundary.
- Keeps this change tightly scoped and low-risk.

Alternative considered:
- Mix baseline cleanup and implementation in one change.
- Rejected to avoid scope coupling and review noise.

## Risks / Trade-offs

- [Risk] Additive correction docs may diverge from future implementation intent.  
  -> Mitigation: tasks include a traceability checklist and explicit follow-up linkage requirements.

- [Risk] Reviewers may treat this baseline as replacing canonical platform specs.  
  -> Mitigation: explicitly document that `openspec/specs/*` is unchanged in this change.

- [Risk] Incomplete conflict inventory could leave hidden contradictions.  
  -> Mitigation: require a full mismatch matrix and source citation before marking tasks done.

## Migration Plan

1. Create this change's spec defining alignment requirements.
2. Produce tasks to generate and verify alignment artifacts.
3. Review and approve this change as the Week4 archive interpretation baseline.
4. Open separate implementation change(s) to satisfy unresolved behavior gaps.
5. Optionally archive this alignment change after verification.

Rollback strategy:
- If alignment is disputed, reject/close this change without touching archived files.
- Since no production code/spec roots are modified, rollback is document-level only.

## Open Questions

- Which concrete follow-up change name will carry Week4 code conformance fixes?
- Should post-alignment verification become a standard checklist for all archived backfills?
