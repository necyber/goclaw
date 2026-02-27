# DAG Error Model Spec (Week 2 Archive Backfill)

## Scope

Define structured error categories surfaced by DAG compiler operations.

## Requirements

### FR-1 Duplicate task errors

Adding an existing task ID SHALL return an explicit duplicate-task error.

### FR-2 Missing dependency errors

Referencing unknown dependency task IDs SHALL return dependency-not-found errors.

### FR-3 Cycle errors

Cycle detection/compile SHALL return dedicated cyclic-dependency errors.

### FR-4 Task lookup errors

Task queries against unknown IDs SHOULD return explicit task-not-found errors.

## Archive Note

Historical backfill for archived change `week2-dag-compiler`.

