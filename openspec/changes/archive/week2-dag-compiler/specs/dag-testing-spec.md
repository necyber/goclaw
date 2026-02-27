# DAG Testing Spec (Week 2 Archive Backfill)

## Scope

Define minimum test scenarios for DAG compiler correctness and resilience.

## Requirements

### FR-1 Shape coverage

Tests SHALL cover empty graph, single-node, chain, fan-out, and diamond graph shapes.

### FR-2 Cycle coverage

Tests SHALL cover self-cycle and multi-node cycles and validate returned cycle diagnostics.

### FR-3 Compile path coverage

Tests SHALL validate compile outputs (layers/order/parallelism metadata) for representative acyclic graphs.

### FR-4 Failure coverage

Tests SHALL cover duplicate-node, missing-dependency, and invalid-edge cases.

## Archive Note

Historical backfill for archived change `week2-dag-compiler`.

