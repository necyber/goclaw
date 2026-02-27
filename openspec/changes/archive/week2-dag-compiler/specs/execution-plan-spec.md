# Execution Plan Spec (Week 2 Archive Backfill)

## Scope

Define plan generation from DAG to layer-based scheduling structure.

## Requirements

### FR-1 Layer generation

Compiler SHALL produce layer sets where tasks in the same layer can run concurrently.

### FR-2 Plan metadata

Execution plan SHALL include total task count and maximum observed parallelism.

### FR-3 Parallel groups

Execution plan SHOULD expose parallel groups suitable for scheduler batching.

### FR-4 Critical path signal

Execution plan SHOULD expose critical-path information for future optimization and observability.

## Archive Note

Historical backfill for archived change `week2-dag-compiler`.

