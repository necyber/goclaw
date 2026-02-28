## Purpose

Define deterministic DAG compilation behavior and dependency error semantics for `pkg/dag`.

## Requirements

### Requirement: Deterministic topological order for identical input
The DAG compiler SHALL return deterministic topological ordering for identical graph input.

#### Scenario: Repeated sort on same graph
- **WHEN** topological sort is executed multiple times on the same acyclic graph
- **THEN** each run MUST return the same ordered task ID sequence

#### Scenario: Multiple zero in-degree candidates
- **WHEN** more than one task is ready at the same step
- **THEN** the compiler MUST choose tasks in a stable deterministic order

### Requirement: Deterministic layer representation
The DAG compiler SHALL return deterministic layer content ordering for identical graph input.

#### Scenario: Repeated level generation
- **WHEN** levels are generated multiple times for the same acyclic graph
- **THEN** both layer membership and task order within each layer MUST remain identical across runs

### Requirement: Dependency reference failures use dependency-not-found errors
The DAG compiler SHALL return `DependencyNotFoundError` when a dependency reference points to an unknown task ID in dependency-related paths.

#### Scenario: Task declaration references unknown dependency
- **WHEN** a task includes a dependency ID that does not exist in the graph
- **THEN** validation or compile MUST fail with `DependencyNotFoundError`

#### Scenario: Edge insertion references unknown dependency source
- **WHEN** dependency edge insertion declares `from -> to` and `from` does not exist
- **THEN** the operation MUST fail with `DependencyNotFoundError`

### Requirement: DAG compiler tests cover determinism and dependency typing
The DAG test suite SHALL include explicit tests for deterministic ordering and missing-dependency error typing.

#### Scenario: Determinism regression guard
- **WHEN** DAG unit tests are executed
- **THEN** tests MUST verify stable sort and layer output for representative multi-branch graphs

#### Scenario: Missing-dependency error typing guard
- **WHEN** DAG unit tests are executed
- **THEN** tests MUST assert `DependencyNotFoundError` for unknown dependency references in dependency-related paths
