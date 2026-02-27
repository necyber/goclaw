# Saga Guide

## Overview

GoClaw Saga provides orchestration-based distributed transactions for workflows that need eventual consistency.

Core capabilities:
- Declarative Saga DSL (`New().Step().Build()`)
- Forward execution by DAG dependency order
- Reverse compensation on failure
- WAL + checkpoint persistence for crash recovery
- HTTP and gRPC management APIs
- Prometheus metrics for execution, compensation, and recovery

## Quick Start

Enable Saga in `config/config.example.yaml`:

```yaml
saga:
  enabled: true
  max_concurrent: 100
  default_timeout: 2m
  default_step_timeout: 30s
  wal_sync_mode: sync
  wal_retention: 168h
  wal_cleanup_interval: 1h
  compensation_policy: auto
  compensation_max_retries: 3
  compensation_initial_backoff: 100ms
  compensation_max_backoff: 5s
  compensation_backoff_factor: 2.0
```

## Defining Sagas (DSL)

```go
orderSaga, err := saga.New("order-processing").
    WithCompensationPolicy(saga.AutoCompensate).
    WithTimeout(2*time.Minute).
    WithDefaultStepTimeout(30*time.Second).
    Step("reserve-inventory",
        saga.Action(reserveInventory),
        saga.Compensate(releaseInventory),
    ).
    Step("charge-payment",
        saga.Action(chargePayment),
        saga.Compensate(refundPayment),
        saga.DependsOn("reserve-inventory"),
    ).
    Step("ship-order",
        saga.Action(shipOrder),
        saga.Compensate(cancelShipment),
        saga.DependsOn("charge-payment"),
    ).
    Build()
```

Design rules:
- Step IDs must be unique.
- Dependencies must reference existing steps.
- Cycles are rejected at build time.

## Compensation Strategies

Saga-level strategies:
- `AutoCompensate`: trigger compensation immediately on failure.
- `ManualCompensate`: move to `pending-compensation`, then trigger manually.
- `SkipCompensate`: skip reverse compensation.

Step-level strategy:
- `WithStepCompensationPolicy(saga.SkipCompensate)`: skip compensation for one step.

Best practices:
- Keep compensation operations idempotent.
- Use external idempotency keys when touching external systems.
- Use bounded retries with exponential backoff for transient failures.
- Alert on `compensation-failed` state and keep manual runbook ready.

## Recovery and Resume

Persistence model:
- WAL key format: `wal:{sagaID}:{sequence}`
- Checkpoint key format: `checkpoint:{sagaID}`

Recovery flow:
1. On startup, scan non-terminal checkpoints.
2. Resume `running` Sagas from the next uncompleted step.
3. Resume `compensating` Sagas from remaining compensation work.
4. Persist the latest snapshot after recovery attempt.

Operational notes:
- `wal_sync_mode=sync` favors durability.
- `wal_sync_mode=async` reduces latency but risks buffered loss on crash.

## API Reference

HTTP endpoints:
- `POST /api/v1/sagas`
- `GET /api/v1/sagas/{id}`
- `GET /api/v1/sagas`
- `POST /api/v1/sagas/{id}/compensate`
- `POST /api/v1/sagas/{id}/recover`

gRPC service (`goclaw.v1.SagaService`):
- `SubmitSaga`
- `GetSagaStatus`
- `ListSagas`
- `CompensateSaga`
- `WatchSaga`

## Metrics

Saga Prometheus metrics:
- `saga_executions_total{status=...}`
- `saga_duration_seconds{status=...}`
- `saga_active_count`
- `saga_compensations_total{status=...}`
- `saga_compensation_duration_seconds`
- `saga_compensation_retries_total`
- `saga_recovery_total{status=...}`

## Troubleshooting

`saga orchestrator unavailable`:
- Ensure `saga.enabled=true`.
- Verify engine startup logs for Saga initialization failures.

`saga definition not found` on manual operations:
- The running process must still have the submitted in-memory definition.
- Re-submit the Saga definition or extend persistence for definitions.

`checkpoint not found` during recover:
- Verify checkpoint persistence path and permissions.
- Confirm the Saga reached at least one completed step before crash.

Frequent `compensation-failed`:
- Validate compensation idempotency.
- Increase retry/backoff for transient downstream errors.
- Add targeted alerting on `saga_compensations_total{status="failure"}`.
