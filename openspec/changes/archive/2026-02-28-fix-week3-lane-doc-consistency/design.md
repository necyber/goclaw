## Context

The archived change `week3-lane-queue-system` currently has cross-artifact drift between `proposal.md`, `design.md`, `tasks.md`, and its archived backfill specs. The inconsistencies are narrow but material for auditability: rate-limiter scope mismatch, worker model wording conflict, implicit (non-traceable) spec requirements, and inconsistent backpressure metrics semantics.

This change is a documentation correction change only. It must preserve archive traceability by recording semantic alignment through a new change, not undocumented direct edits.

## Goals / Non-Goals

**Goals:**
- Remove contradiction between Token Bucket normative scope and Leaky Bucket extension wording.
- Normalize worker-pool wording so all week3 artifacts describe one consistent model.
- Make requirement traceability explicit for three previously implicit requirements:
  - deterministic tie-breaking for equal priorities
  - concurrent read/write safety in lane manager
  - repeated-close safety in lane lifecycle
- Normalize backpressure metric semantics (`accepted`, `rejected`, `redirected`, `dropped`) across artifacts.
- Keep corrections minimal, auditable, and documentation-only.

**Non-Goals:**
- No changes to runtime code under `pkg/lane`.
- No new lane features beyond existing week3 scope.
- No broad rewrite of unrelated archived changes.

## Decisions

1. Canonical rate-limiter scope: Token Bucket is normative; Leaky Bucket is optional extension.
Rationale: The archived week3 rate-limiter spec is Token Bucket-centric, while some docs mention Leaky Bucket as if required. Canonicalizing this avoids scope drift.
Alternative considered: Promote Leaky Bucket to mandatory week3 requirement. Rejected because it retroactively changes archived scope.

2. Canonical worker model wording: configurable worker concurrency with optional dynamic scaling support.
Rationale: Current wording conflicts between "fixed" and "dynamic". A single canonical statement keeps historical docs internally coherent without changing implementation intent.
Alternative considered: Force "fixed only" wording. Rejected because tasks/proposal already claim dynamic behavior.

3. Require explicit requirement-traceability annotations in archived consistency repairs.
Rationale: Some week3 spec requirements are present but only implied in proposal/design/tasks. Explicit traceability reduces ambiguity during archive audits.
Alternative considered: Keep implicit alignment and rely on reviewer interpretation. Rejected due to recurring drift risk.

4. Canonical backpressure metrics vocabulary across all week3 artifacts.
Rationale: Backpressure spec requires consistent counters, but task/design wording is incomplete. Standardized terms prevent reporting mismatches.
Alternative considered: Keep minimal metrics language. Rejected because it fails traceability expectation.

## Risks / Trade-offs

- [Risk] Additional governance wording could increase maintenance overhead for small archive edits.  
  -> Mitigation: scope traceability requirement to semantic consistency repairs, not all edits.

- [Risk] Historical language cleanup may still leave minor stylistic differences.  
  -> Mitigation: prioritize semantic consistency (scope, terminology, metrics), defer stylistic normalization.

- [Risk] Reviewers may interpret "optional extension" differently.  
  -> Mitigation: explicitly state that optional Leaky Bucket does not expand week3 acceptance baseline.

## Migration Plan

1. Add a delta spec under this change for `spec-format-governance` to codify traceable consistency repair expectations.
2. Update archived week3 `proposal.md`, `design.md`, and `tasks.md` language to align the four warning areas.
3. Re-run consistency review for week3 artifacts and verify no contradiction remains across proposal/design/tasks/specs.
4. Validate OpenSpec artifacts and keep this change as the audit record for semantic corrections.

## Open Questions

- None at this time; the correction scope is bounded to the four identified consistency warnings.
