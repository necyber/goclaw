# Topological Sort Spec (Week 2 Archive Backfill)

## Scope

Define topological ordering behavior for valid DAG execution sequence.

## Requirements

### FR-1 Ordering for acyclic graphs

Compiler SHALL return task order where each task appears after all its dependencies.

### FR-2 Complexity target

Topological sorting SHOULD execute in `O(V+E)` complexity.

### FR-3 Deterministic output

For identical graph input, topological sort output SHOULD be deterministic.

### FR-4 Error propagation

Topological sort SHALL return explicit error if ordering is impossible due to unresolved cyclic structure.

## Archive Note

Historical backfill for archived change `week2-dag-compiler`.

