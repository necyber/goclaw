# Task Runner Spec (Week 4 Archive Backfill)

## Scope

Specify adapter behavior that maps DAG task definitions into lane-executable units.

## Requirements

### FR-1 Lane task adapter

Task runner SHALL implement the lane task contract (`ID`, `Lane`, `Priority`, `Execute`).

### FR-2 Retry support

Task runner SHALL execute with bounded retry attempts based on task retry policy.

### FR-3 Timeout and cancellation propagation

Task runner SHALL honor context cancellation and per-task timeout boundaries.

### FR-4 State hooks

Task runner SHALL update state tracker for scheduled/running/completed/failed/cancelled transitions.

## Archive Note

Historical backfill for archived change `week4-engine-core`.

