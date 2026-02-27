# OpenTelemetry Tracing Guide

This guide describes how to enable and operate OpenTelemetry tracing in GoClaw.

## Scope

GoClaw tracing currently covers:

- TracerProvider lifecycle bootstrap and shutdown
- HTTP ingress tracing middleware
- gRPC unary and stream tracing interceptors
- Runtime spans for workflow, lane scheduling, and saga execution
- Log correlation fields (`trace_id`, `span_id`)
- HTTP metrics exemplar correlation when backend supports exemplars

## Enable Tracing

Set tracing in your config file:

```yaml
tracing:
  enabled: true
  exporter: otlpgrpc
  endpoint: "localhost:4317"
  headers: {}
  timeout: 5s
  sampler: parentbased_traceidratio
  sample_rate: 0.1
  type: "" # deprecated legacy alias: jaeger, zipkin

server:
  grpc:
    enabled: true
    enable_tracing: true
```

Notes:

- `tracing.enabled` controls global tracing lifecycle and provider registration.
- gRPC tracing interceptors run only when both `tracing.enabled=true` and `server.grpc.enable_tracing=true`.
- HTTP tracing middleware is enabled by `tracing.enabled=true`.

## Configuration Reference

| Field | Required When Enabled | Default | Description |
| --- | --- | --- | --- |
| `tracing.enabled` | yes | `false` | Enables tracing lifecycle and instrumentation hooks. |
| `tracing.exporter` | yes | `otlpgrpc` | Exporter backend (`otlpgrpc`). |
| `tracing.endpoint` | yes | `localhost:4317` | OTLP gRPC collector endpoint. URLs are normalized to host:port. |
| `tracing.headers` | no | `{}` | Optional OTLP request headers (for auth, tenant tags, etc.). |
| `tracing.timeout` | yes | `5s` | Exporter timeout and shutdown flush timeout fallback. |
| `tracing.sampler` | no | `parentbased_traceidratio` | `always_on`, `always_off`, or `parentbased_traceidratio`. |
| `tracing.sample_rate` | no | `0.1` | Ratio used by `parentbased_traceidratio` (0.0-1.0). |
| `tracing.type` | no (deprecated) | `""` | Legacy alias; `jaeger`/`zipkin` map to `otlpgrpc`. |

Validation behavior:

- If `tracing.enabled=true`, `exporter`, `endpoint`, and positive `timeout` are required.
- Invalid values fail fast during config validation.

## Trace Propagation and Span Coverage

Ingress and propagation:

- HTTP: extracts W3C `traceparent`/`baggage`, starts server span, propagates request context.
- gRPC: extracts metadata context, starts server spans for unary/stream calls, injects outgoing metadata.
- Outbound HTTP helpers:
  - `middleware.InjectOutboundTraceContext(req)`
  - `middleware.NewTracingRequest(ctx, method, url, body)`

Runtime span taxonomy:

- `workflow.execute`
- `workflow.layer`
- `workflow.task.schedule`
- `workflow.task.run`
- `lane.wait`
- `saga.execute.forward`
- `saga.step.forward`
- `saga.execute.compensation`
- `saga.step.compensate`
- `saga.recovery.resume`

## Observability Correlation

- Context-aware logger methods append:
  - `trace_id`
  - `span_id`
- HTTP request metrics use exemplar labels from active span context when supported by backend/export path.

## Related Documents

- [Tracing Architecture](./tracing-architecture.md)
- [Tracing Runbook](./tracing-runbook.md)
- [Monitoring Guide](./monitoring-guide.md)
