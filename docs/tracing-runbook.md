# OpenTelemetry Tracing Runbook

This runbook is for on-call and production operations of GoClaw tracing.

## 1. Preflight Checklist

- Collector is reachable from GoClaw host (`<collector-host>:4317`).
- `tracing.enabled=true` in effective config.
- `tracing.endpoint` is set and non-empty.
- `tracing.timeout` is greater than `0`.
- If gRPC tracing is expected: `server.grpc.enable_tracing=true`.

## 2. Startup Verification

On startup, check logs for one of the following:

- `OpenTelemetry tracing enabled` with `exporter`, `endpoint`, `sampler`, and `sample_rate`
- `OpenTelemetry tracing disabled` when tracing is intentionally off

Expected endpoint logging is sanitized to host:port summary.

## 3. Collector Connectivity Checks

Basic network check from GoClaw host:

```powershell
Test-NetConnection <collector-host> -Port 4317
```

If headers/auth are required, verify `tracing.headers` contains expected keys.

## 4. Failure Modes and Actions

### A. Startup fails with tracing config error

Symptoms:

- Process exits during startup.
- Error includes `tracing endpoint cannot be empty`, `tracing exporter cannot be empty`, or timeout validation failures.

Actions:

1. Fix config fields under `tracing`.
2. Re-run with the corrected config.
3. If urgent recovery is needed, temporarily set `tracing.enabled=false`.

### B. Collector unavailable at runtime

Symptoms:

- Service remains healthy.
- Warning logs appear: `tracing exporter failed`.

Behavior:

- Export failures are isolated by design and do not fail business requests.

Actions:

1. Restore collector availability.
2. Validate warnings stop after collector recovery.
3. Confirm traces resume in backend.

### C. No traces visible in backend

Checks:

1. Confirm incoming traffic exists.
2. Confirm sampling is not too low (`sample_rate`, `sampler`).
3. Confirm collector pipeline routes traces from this service.
4. Confirm trace context reaches ingress (HTTP/gRPC metadata).

## 5. Shutdown Semantics

GoClaw tracing shutdown behavior:

1. `ForceFlush` is called.
2. Provider `Shutdown` is called.
3. Both run within a timeout window from `tracing.timeout`.
4. If timeout is unset or invalid at shutdown path, fallback timeout is `5s`.

Operational guidance:

- Keep `tracing.timeout` small but sufficient for your collector RTT.
- During controlled restarts, wait for clean process exit to maximize span delivery.

## 6. Rollback / Disable Strategy

If tracing contributes to incident noise or collector outage:

1. Set `tracing.enabled=false`.
2. Redeploy.
3. Verify core workflow/saga traffic is healthy.
4. Re-enable tracing after collector stability is restored.
