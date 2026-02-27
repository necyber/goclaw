# Graph Construction Spec (Week 2 Archive Backfill)

## Scope

Specify DAG graph creation and mutation behaviors.

## Requirements

### FR-1 Graph initialization

Compiler module SHALL provide a constructor for an empty DAG graph instance.

### FR-2 Node insertion

Graph SHALL support task node insertion with duplicate ID rejection.

### FR-3 Edge insertion

Graph SHALL support dependency edge insertion and reject edges referencing unknown tasks.

### FR-4 Query operations

Graph SHALL provide dependency/dependent lookup operations for scheduler and diagnostics usage.

## Archive Note

Historical backfill for archived change `week2-dag-compiler`.

