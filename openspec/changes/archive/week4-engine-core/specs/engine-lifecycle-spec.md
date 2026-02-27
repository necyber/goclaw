# Engine Lifecycle Spec (Week 4 Archive Backfill)

## Scope

Define core engine lifecycle behavior for start, run-state management, and stop transitions.

## Requirements

### FR-1 Engine state machine

Engine SHALL expose lifecycle states equivalent to `Idle -> Running -> Stopped` with explicit error-state signaling.

### FR-2 Start semantics

Engine start SHALL initialize required runtime components (scheduler context, lane manager dependencies, default lane) before accepting workflow submissions.

### FR-3 Stop semantics

Engine stop SHALL transition engine to non-running state and coordinate component shutdown in deterministic order.

### FR-4 Running-state guard

Workflow submission SHALL reject requests when engine is not in running state.

## Archive Note

Historical backfill for archived change `week4-engine-core`.  
Does not alter canonical specs in `openspec/specs/*`.

