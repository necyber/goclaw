# Backpressure Spec (Week 3 Archive Backfill)

## Scope

Define queue-full handling policy for lane submissions.

## Requirements

### FR-1 Block strategy

When queue is full, block strategy SHALL wait for capacity and honor context cancellation or timeout.

### FR-2 Drop strategy

When queue is full, drop strategy SHALL reject incoming tasks and return an explicit dropped/full error.

### FR-3 Redirect strategy

When queue is full, redirect strategy SHALL route tasks to a configured alternate lane with loop prevention safeguards.

### FR-4 Consistent metrics

All strategies SHALL update counters consistently for accepted, rejected, redirected, and dropped tasks.

## Archive Note

Historical backfill for archived change `week3-lane-queue-system`.

