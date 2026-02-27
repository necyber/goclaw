# Cycle Detection Spec (Week 2 Archive Backfill)

## Scope

Define cycle-detection behavior for dependency validation.

## Requirements

### FR-1 Cycle detection capability

DAG compiler SHALL detect cyclic dependencies before scheduling.

### FR-2 Complexity target

Cycle detection SHOULD execute in `O(V+E)` complexity.

### FR-3 Diagnostic path reporting

Cycle errors SHALL include at least one concrete loop path to aid debugging.

### FR-4 Compile gate

Compilation SHALL fail when cycle detection reports a loop.

## Archive Note

Historical backfill for archived change `week2-dag-compiler`.

