# Tracing Architecture

This document describes the OpenTelemetry tracing architecture used by GoClaw.

## High-Level Diagram

```text
HTTP Request / gRPC Call
          |
          v
  +--------------------+
  | W3C Propagator     |  TraceContext + Baggage
  +--------------------+
          |
          v
  +--------------------+        +-----------------------+
  | Ingress Spans      |        | Runtime Spans         |
  | - HTTP middleware  | -----> | - workflow.*          |
  | - gRPC interceptors|        | - lane.wait           |
  +--------------------+        | - saga.*              |
          |                     +-----------------------+
          v
  +--------------------+
  | TracerProvider     |
  | BatchSpanProcessor |
  +--------------------+
          |
          v
  +--------------------+
  | Isolating Exporter |
  | (error isolation)  |
  +--------------------+
          |
          v
  OTLP gRPC Collector
```

## End-to-End Flow

1. `cmd/goclaw/main.go` initializes tracing via `pkg/telemetry/tracing.Init`.
2. Global tracer provider and propagator are registered.
3. Inbound HTTP/gRPC requests extract trace context and start server spans.
4. Runtime execution produces workflow/lane/saga spans in the same trace context.
5. Logs and HTTP metrics can correlate to active trace context.
6. Spans are batched and exported through OTLP gRPC.
7. Export failures are logged and isolated from business-path execution.

## Components

- `pkg/telemetry/tracing`
  - Provider lifecycle
  - Sampler selection
  - OTLP exporter setup
  - Flush/shutdown contract
- `pkg/api/middleware/tracing.go`
  - HTTP trace extraction and span creation
  - Outbound HTTP trace injection helpers
- `pkg/grpc/interceptors/tracing.go`
  - gRPC unary/stream tracing
  - gRPC metadata extraction/injection
- `pkg/engine` + `pkg/saga`
  - Runtime and transaction span coverage

## Correlation Signals

- Logs:
  - Context-aware logging adds `trace_id` and `span_id` when span context exists.
- Metrics:
  - HTTP request metrics use exemplar trace labels when backend supports exemplars.
