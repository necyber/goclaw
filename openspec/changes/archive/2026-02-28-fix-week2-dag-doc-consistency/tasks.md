## 1. Align week2 archived artifacts

- [x] 1.1 Normalize cycle error naming in `openspec/changes/archive/week2-dag-compiler/design.md` and `tasks.md` to `CyclicDependencyError`
- [x] 1.2 Align week2 scope statements across `proposal.md`, `design.md`, and `tasks.md` to DAG-core-only and mark workflow/scheduler integration as deferred
- [x] 1.3 Align week2 API naming and acceptance wording (toposort function names, benchmark language) to remove internal contradictions

## 2. Add archived-correction governance

- [x] 2.1 Update delta spec `specs/spec-format-governance/spec.md` if needed after review feedback to keep requirements precise and testable
- [x] 2.2 Add errata note format guidance to affected archived files when applying non-semantic fixes

## 3. Validate and prepare apply

- [x] 3.1 Run `openspec validate --changes --strict` and fix any schema/format issues
- [x] 3.2 Re-run consistency review on week2 artifacts and confirm no naming/scope/acceptance conflicts remain
- [x] 3.3 Capture final change summary in this change before implementation/apply

## 4. Final Summary (2026-02-28)

- Updated archived week2 `proposal.md`, `design.md`, and `tasks.md` to remove naming/scope/acceptance contradictions.
- Canonicalized cycle error naming to `CyclicDependencyError` in archived design/tasks artifacts.
- Marked workflow/scheduler-related items as Deferred and clarified DAG-core-only scope for week2.
- Aligned task API naming (`TopologicalSort`, `TopologicalSortDFS`) and benchmark wording to reproducible reporting.
- Added non-semantic errata record format guidance to affected archived files.
- Refined delta governance spec for semantic/non-semantic distinction and errata format testability.
- Validation completed with `openspec validate --changes --strict`.
