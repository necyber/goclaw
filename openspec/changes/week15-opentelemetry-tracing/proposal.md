## Why

GoClaw currently has generic tracing requirements but no OpenTelemetry-specific contract for tracer provider lifecycle, exporter configuration, or propagation behavior. As distributed execution and Saga orchestration are now in place, adding explicit OpenTelemetry specs is needed to make observability implementation consistent and production-ready.

## What Changes

- Add a dedicated OpenTelemetry tracing capability with explicit requirements for provider setup, resource attributes, sampling, exporter support, and graceful shutdown flushing.
- Define end-to-end trace context propagation across HTTP ingress, gRPC ingress/egress, and internal execution context handoff.
- Standardize span model for core operations (request handling, workflow execution, lane scheduling, saga execution/compensation/recovery).
- Define failure/degradation behavior when telemetry backend is unavailable (service remains available; telemetry errors are isolated).
- Add configuration requirements for tracing enablement and exporter settings.

## Capabilities

### New Capabilities
- `opentelemetry-tracing`: OpenTelemetry provider lifecycle, propagation, span model, sampling/exporter configuration, and graceful degradation requirements.

### Modified Capabilities
- `interceptors`: tracing interceptor requirements are tightened to align with OpenTelemetry context extraction/injection and span semantics.
- `grpc-server`: server startup/shutdown requirements are extended to include tracing provider wiring and flush-on-shutdown behavior.
- `http-server-core`: HTTP middleware chain requirements are extended to include OpenTelemetry HTTP span creation and context propagation.

## Impact

- Affected code:
  - `config/config.go`, `config/config.example.yaml` (tracing config schema)
  - `cmd/goclaw/main.go` (provider initialization and lifecycle wiring)
  - `pkg/grpc/interceptors/*` and gRPC server bootstrap code
  - `pkg/api/middleware/*` and HTTP server bootstrap code
  - runtime/engine and saga execution paths for span instrumentation
- External dependencies:
  - OpenTelemetry Go SDK packages and OTLP exporter packages
- Operational impact:
  - New telemetry configuration options and runtime behavior documentation
  - Additional tracing validation in tests and startup diagnostics
