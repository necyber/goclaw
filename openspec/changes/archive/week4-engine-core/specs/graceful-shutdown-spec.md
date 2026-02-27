# Graceful Shutdown Spec (Week 4 Archive Backfill)

## Scope

Specify graceful shutdown behavior for in-flight workflow execution.

## Requirements

### FR-1 Stop admission

Shutdown initiation SHALL stop acceptance of new workflow submissions.

### FR-2 In-flight handling

Engine stop SHALL wait for running tasks/workflows to finish within context deadline or cancellation window.

### FR-3 Component close order

Shutdown SHALL close scheduling/queue components in deterministic order to avoid orphan work.

### FR-4 Timeout-respect

Stop operation SHALL honor external context timeout and return timeout/cancel errors accordingly.

## Archive Note

Historical backfill for archived change `week4-engine-core`.

