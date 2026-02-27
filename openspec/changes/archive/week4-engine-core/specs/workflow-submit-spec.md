# Workflow Submit Spec (Week 4 Archive Backfill)

## Scope

Specify workflow submission path from DAG construction to execution-plan scheduling.

## Requirements

### FR-1 Workflow ingestion

Engine SHALL accept workflow definitions containing task graph data and execution functions.

### FR-2 Compilation pipeline

Submission SHALL build a DAG graph and compile it into an execution plan before any task dispatch.

### FR-3 Compile-failure behavior

If graph compilation fails (for example cycle/dependency errors), engine SHALL return a compile error and MUST NOT dispatch tasks.

### FR-4 Result contract

Submission SHALL return workflow result containing workflow ID, terminal status, task results, and terminal error (if any).

## Archive Note

Historical backfill for archived change `week4-engine-core`.

