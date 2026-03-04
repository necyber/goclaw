## Context

The week15 OpenTelemetry tracing implementation is mostly complete, but review identified four behavior gaps against spec intent: request logs are not consistently trace-correlated, gRPC panic paths can lose tracing error semantics, gRPC metrics do not preserve exemplar trace correlation metadata, and HTTP failed spans do not emit explicit error attributes. Current code already has tracing bootstrap and middleware/interceptor plumbing, so this change focuses on tightening observability semantics without expanding API surface.

## Goals / Non-Goals

**Goals:**
- Ensure HTTP and gRPC request logging paths emit `trace_id` and `span_id` when span context is available.
- Ensure gRPC tracing spans capture failure semantics for panic-to-Internal recovery paths.
- Ensure gRPC metrics paths preserve trace correlation metadata for exemplar-capable backends while remaining compatible with non-exemplar collectors.
- Ensure HTTP tracing spans set explicit error attributes on failure outcomes (4xx/5xx and recovered panic paths).
- Add deterministic tests for the above behaviors.

**Non-Goals:**
- Changing external HTTP/gRPC APIs or config schema.
- Reworking full interceptor architecture beyond what is required for tracing correctness.
- Introducing new tracing backends or exporter types.

## Decisions

1. Use context-aware logger entrypoints in middleware/interceptors.
- Decision: switch request log call sites from non-context logger methods to context-aware methods so logger-level trace enrichment is automatically applied.
- Rationale: minimal code change, reuses existing `logger.appendTraceContextFields` behavior.
- Alternatives considered:
  - Inject trace IDs manually at each call site: duplicates logic and increases drift risk.
  - Build a dedicated logging interceptor wrapper: over-scoped for this fix.

2. Preserve gRPC tracing failure semantics on panic without changing global interceptor order.
- Decision: add panic-aware handling inside tracing interceptors that records span error/status and re-panics so recovery interceptor still owns response translation.
- Rationale: fixes tracing fidelity while keeping existing interceptor ordering contract intact.
- Alternatives considered:
  - Reorder chain to place tracing outside recovery: broader behavior change with higher regression risk.

3. Add optional exemplar-aware recording path for gRPC metrics.
- Decision: extend metrics recording path to use context-aware/exemplar-capable interfaces where available, with safe fallback to current counters/histograms.
- Rationale: satisfies trace-metrics correlation requirement while preserving compatibility with existing Prometheus collectors.
- Alternatives considered:
  - Force exemplar-only implementation: would break backends/collectors without exemplar support.

4. Add explicit HTTP span error attributes for failure outcomes.
- Decision: enrich HTTP tracing middleware with error attributes/events for 4xx/5xx and recovered panic paths, in addition to status code mapping.
- Rationale: aligns with OTel failure semantics and makes trace backends queryable by error details.
- Alternatives considered:
  - Keep status-only mapping: insufficient for the requirement and weak for diagnostics.

## Risks / Trade-offs

- [Risk] Additional logging/tracing attribute writes increase hot-path overhead slightly -> Mitigation: only add fields when span context is valid; reuse existing middleware wrappers.
- [Risk] Panic instrumentation could accidentally swallow panics -> Mitigation: tracing panic handler must always re-panic after recording span failure.
- [Risk] Exemplar behavior differs by metrics backend -> Mitigation: implement capability detection with no-op fallback and cover both paths in tests.
- [Risk] Assertion-heavy tracing tests can be flaky -> Mitigation: use in-memory OTel span recorder and bounded wait helpers.

## Migration Plan

1. Update interceptor and middleware implementations with the scoped behavior changes.
2. Add/adjust unit tests for logging correlation, panic status mapping, exemplar correlation, and HTTP error attributes.
3. Run targeted package tests for changed areas.
4. Mark tasks complete and proceed to archive after verification.

## Open Questions

- Should gRPC metrics exemplar correlation be limited to latency histograms only, or also include request counters when collector supports both?
- For HTTP 4xx responses, should all be tagged as span errors, or should specific client-cancel/status cases be downgraded in future tuning?
