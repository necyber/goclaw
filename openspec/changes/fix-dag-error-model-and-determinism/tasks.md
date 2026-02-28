## 1. Deterministic DAG ordering

- [x] 1.1 Update `pkg/dag/toposort.go` so zero in-degree task selection is stable and deterministic across runs
- [x] 1.2 Ensure deterministic processing order for newly-ready tasks during topological sort
- [x] 1.3 Ensure `Levels()` output is deterministic by sorting task IDs within each emitted layer

## 2. Dependency error model alignment

- [x] 2.1 Update dependency-related mutation path(s) in `pkg/dag/dag.go` so unknown dependency references return `DependencyNotFoundError`
- [x] 2.2 Preserve `TaskNotFoundError` for direct lookup/query paths and document/assert this boundary in tests
- [x] 2.3 Add/adjust tests for missing dependency behavior in validation/compile and edge insertion paths

## 3. Regression tests and validation

- [ ] 3.1 Add deterministic regression tests for repeated `TopologicalSort()` and `Levels()` on representative branching graphs
- [ ] 3.2 Run `go test ./pkg/dag` and fix any failures
- [ ] 3.3 Run `openspec validate --changes --strict` and fix any change artifact issues
