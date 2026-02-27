# Engine Error Model Spec (Week 4 Archive Backfill)

## Scope

Define error categories emitted by the week4 engine core path.

## Requirements

### FR-1 Compile errors

Engine SHALL expose workflow compile errors separately from execution errors.

### FR-2 Execution errors

Engine SHALL expose task execution errors with task identity and retry context.

### FR-3 Lifecycle errors

Engine SHALL return explicit not-running/lifecycle errors when APIs are called in invalid engine states.

### FR-4 Cancellation semantics

Context cancellation SHALL be represented distinctly from generic execution failures when possible.

## Archive Note

Historical backfill for archived change `week4-engine-core`.

