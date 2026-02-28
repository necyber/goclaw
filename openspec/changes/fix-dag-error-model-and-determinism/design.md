## Context

`pkg/dag` currently works for basic DAG compilation, but two behaviors are underspecified and inconsistent: dependency-related error typing and deterministic ordering for identical graph input. This creates flaky tests, unstable execution plans, and ambiguous caller-side error handling.

## Goals / Non-Goals

**Goals:**
- Make `TopologicalSort` and layer generation deterministic for identical graph input.
- Ensure unknown dependency references are surfaced as `DependencyNotFoundError` in dependency-related code paths.
- Preserve current DAG APIs while tightening behavior contracts and tests.

**Non-Goals:**
- No changes to external transport APIs (HTTP/gRPC).
- No new runtime scheduling features beyond DAG compile behavior.
- No broad refactor outside `pkg/dag`.

## Decisions

1. Deterministic topological ordering
- Decision: Use stable ordering for queue seeding and processing in Kahn-style sort.
- Rationale: Map iteration order is non-deterministic; stable ordering removes run-to-run output variance.
- Alternatives considered:
  - Keep current behavior and relax tests: rejected because downstream planning remains unstable.
  - Use randomization plus canonicalization at output: rejected as unnecessary complexity.

2. Deterministic layer output
- Decision: Sort task IDs within each layer before returning levels/plan layers.
- Rationale: Layer membership may be correct while order fluctuates; sorting gives stable plan representation.
- Alternatives considered:
  - Preserve insertion order heuristics: rejected because insertion order is not consistently available.

3. Dependency error model boundary
- Decision: Treat unknown dependency references as `DependencyNotFoundError` in dependency declaration or dependency-edge insertion paths; keep `TaskNotFoundError` for direct lookup/query by ID.
- Rationale: Callers need semantic distinction between “missing dependency” and “query target not found.”
- Alternatives considered:
  - Collapse everything into `TaskNotFoundError`: rejected because it loses actionable error semantics.

4. Test hardening
- Decision: Add explicit tests for deterministic order and missing dependency error typing.
- Rationale: Behavior contract should be enforced via tests to prevent regression.

## Risks / Trade-offs

- [Risk] Stronger deterministic expectations can break tests that assume flexible order.
  → Mitigation: Update tests to check exact stable order only where behavior is explicitly deterministic.

- [Risk] Error-type changes may affect existing internal callers.
  → Mitigation: Keep messages clear and constrain type changes to dependency-specific paths.

- [Risk] Sorting introduces minor overhead.
  → Mitigation: Overhead is small and bounded; correctness/stability benefit outweighs cost.

## Migration Plan

1. Add/adjust requirements and implementation tasks for DAG core behavior.
2. Implement deterministic sorting in `pkg/dag/toposort.go` and layer assembly paths.
3. Implement dependency error typing adjustments in relevant DAG mutation/validation paths.
4. Add/adjust unit tests for deterministic behavior and missing dependency typing.
5. Run `go test ./pkg/dag` and OpenSpec validation.

## Open Questions

- Should deterministic ordering be lexicographic by task ID globally, or only within same readiness layer?
  - Proposed for this change: lexicographic by task ID in each readiness step and within each emitted layer.
