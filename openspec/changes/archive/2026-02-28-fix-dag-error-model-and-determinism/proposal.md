## Why

Current DAG behavior has two correctness gaps: dependency-related errors are not consistently surfaced with the expected error category, and topological/layer outputs are not guaranteed deterministic for identical graph input. This change is needed now to make scheduling behavior predictable and to align runtime behavior with week2 archived expectations.

## What Changes

- Define and implement deterministic DAG ordering behavior for topological sort and layer outputs.
- Standardize DAG dependency error semantics so unknown dependency references return dependency-not-found errors in the relevant paths.
- Add/strengthen test coverage for missing-dependency failure paths and deterministic output guarantees.
- Clarify behavior boundaries for DAG compilation errors vs lookup errors.

## Capabilities

### New Capabilities
- `dag-compiler-core`: Specifies deterministic ordering, dependency error model consistency, and minimum DAG test coverage for compiler/runtime planning behavior.

### Modified Capabilities
- None.

## Impact

- Affected code: `pkg/dag/*.go`, `pkg/dag/*_test.go`.
- Affected behavior: DAG compile/sort error semantics and ordering stability.
- No external dependencies added.
- No wire/API surface changes expected; behavior and tests are tightened.
