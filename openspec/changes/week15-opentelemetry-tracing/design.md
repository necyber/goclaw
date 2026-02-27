## Context

GoClaw already includes gRPC tracing interceptors built on OpenTelemetry APIs, but tracing is not end-to-end operational yet:
- no unified TracerProvider bootstrap/shutdown lifecycle;
- no explicit OTLP exporter contract in runtime wiring;
- no HTTP tracing middleware coverage;
- tracing config model is legacy (`jaeger/zipkin`) and not aligned to current OTLP-first deployment patterns.

This change is cross-cutting (config, bootstrap, gRPC, HTTP, runtime instrumentation), so design decisions should be captured before implementation.

## Goals / Non-Goals

**Goals:**
- Define a production-ready OpenTelemetry tracing architecture for GoClaw.
- Standardize tracing lifecycle: init, propagation, instrumentation, graceful flush.
- Cover ingress paths (HTTP + gRPC) and key runtime spans (workflow/task/saga/lane phases).
- Keep business-path availability when telemetry backend is degraded.
- Provide clear config and migration path from current tracing settings.

**Non-Goals:**
- Building a custom tracing backend or UI.
- Introducing logs/metrics backend migration in this change.
- Full auto-instrumentation of every third-party dependency.
- Cross-cluster trace analytics features.

## Decisions

### 1. Centralize tracing lifecycle in a dedicated package
**Decision:** Add `pkg/telemetry/tracing` (or `pkg/observability/tracing`) with:
- `Init(ctx, cfg, appMeta) (ShutdownFunc, error)`
- provider/exporter/sampler creation
- global `otel.SetTracerProvider(...)` and propagator registration

**Rationale:** Avoid scattered initialization in server packages and ensure one lifecycle owner.

**Alternatives considered:**
- Initialize tracing directly in `cmd/goclaw/main.go`: simple but hard to test and reuse.
- Let each server (HTTP/gRPC) create its own provider: risks duplicate providers and broken propagation.

### 2. Use OTLP gRPC as the primary exporter contract
**Decision:** Normalize runtime tracing exporter behavior around OTLP gRPC, with timeout and header support.

**Rationale:** OTLP is the de-facto standard for OTel collectors and reduces backend-specific branching.

**Alternatives considered:**
- Keep jaeger/zipkin exporter-specific branches as first-class config: increases maintenance complexity.
- OTLP HTTP only: valid, but current gRPC stack and existing interceptor flow favor OTLP gRPC first.

### 3. Register global W3C propagators (TraceContext + Baggage)
**Decision:** Configure composite text-map propagator at startup and use it for both HTTP and gRPC paths.

**Rationale:** Ensures consistent cross-protocol trace continuation and baggage propagation.

**Alternatives considered:**
- Per-protocol propagators only: leads to inconsistent behavior and duplicate code paths.

### 4. Keep existing gRPC tracing interceptors and harden semantics
**Decision:** Reuse `pkg/grpc/interceptors/tracing.go`, but align span attributes/status handling with updated spec requirements and ensure chain wiring is config-aware.

**Rationale:** Existing code and tests already provide a base; incremental hardening lowers migration risk.

**Alternatives considered:**
- Replace with third-party interceptor package directly: faster but reduces control over project-specific metadata and ordering.

### 5. Add HTTP tracing middleware in API middleware chain
**Decision:** Introduce `pkg/api/middleware/tracing.go` to extract/inject context and create HTTP server spans.

**Rationale:** HTTP path currently lacks equivalent trace continuity, creating broken traces across API entrypoints.

**Alternatives considered:**
- Only trace gRPC: leaves partial observability and fails roadmap intent.
- Use net/http auto-instrumentation without middleware integration: harder to preserve project-specific context and exclusion policy.

### 6. Define explicit span taxonomy for runtime phases
**Decision:** Introduce stable span naming/attributes for:
- request ingress (`http.*`, `rpc.*`)
- workflow execution phases
- lane scheduling/wait
- saga forward/compensation/recovery phases

**Rationale:** Stable naming improves queryability and alert/runbook consistency.

**Alternatives considered:**
- Ad-hoc span names from each module: quick but yields inconsistent telemetry data.

### 7. Telemetry failure isolation by default
**Decision:** Export failures should be non-fatal to request/workflow execution; tracing errors are recorded as diagnostics/log warnings.

**Rationale:** Observability must not become an availability dependency.

**Alternatives considered:**
- Fail request on exporter failure: improves strict observability but unacceptable for runtime reliability.

## Risks / Trade-offs

- **[Risk] Increased runtime overhead from tracing** -> **Mitigation:** sampling config, health-endpoint suppression, targeted span coverage.
- **[Risk] Misconfigured exporter prevents startup** -> **Mitigation:** strict config validation + clear startup diagnostics.
- **[Risk] Duplicate/inconsistent context propagation across HTTP and gRPC** -> **Mitigation:** shared propagation helpers and integration tests covering both paths.
- **[Risk] Backward compatibility with existing tracing config fields** -> **Mitigation:** staged migration with compatibility mapping and deprecation notices.
- **[Risk] Excessive cardinality in span attributes** -> **Mitigation:** restrict dynamic tags, document safe attribute conventions.

## Migration Plan

1. Extend tracing config model and defaults (maintain compatibility with existing tracing fields during transition).
2. Add centralized tracing initializer and hook into `cmd/goclaw/main.go` lifecycle.
3. Wire gRPC and HTTP tracing paths to shared provider/propagator setup.
4. Add runtime phase instrumentation (workflow/lane/saga) with stable span names.
5. Add tests for propagation, lifecycle flush, degraded exporter behavior, and config validation.
6. Rollout plan:
   - default `tracing.enabled=false`
   - enable in staging with low sample rate
   - validate collector pipeline before production enablement

**Rollback strategy:**
- Set `tracing.enabled=false` to disable tracing without affecting core runtime paths.
- Revert to previous config defaults if startup validation issues are found.

## Open Questions

1. Should we keep legacy `tracing.type` values (`jaeger`, `zipkin`) as compatibility aliases, and for how long?
2. Should health/readiness endpoints be fully excluded from traces by default or sampled at a low fixed rate?
3. Do we need a first-class tenant/workspace trace attribute policy now, or defer to a later multi-tenant observability change?
