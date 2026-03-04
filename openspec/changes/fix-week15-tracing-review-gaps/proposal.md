## Why

The week15 tracing review found observability gaps where runtime behavior does not fully satisfy tracing requirements for correlation and failure semantics. Fixing these now prevents silent telemetry regressions and keeps tracing behavior consistent across HTTP and gRPC paths.

## What Changes

- Require request logging paths to include trace correlation fields (`trace_id`, `span_id`) when an active span exists.
- Require gRPC tracing interceptors to preserve error status mapping even when handler panics are converted to `Internal` by recovery.
- Require gRPC metrics interceptor paths to preserve trace correlation metadata for exemplar-capable backends.
- Require HTTP tracing middleware to attach explicit error attributes for failed outcomes (4xx/5xx and recovered panic paths).
- Add verification scenarios and tests for panic handling, log correlation, and exemplar-aware metrics correlation.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `interceptors`: tighten tracing, logging correlation, and metrics exemplar correlation requirements for gRPC interceptor paths.
- `http-server-core`: tighten HTTP tracing failure semantics to require explicit error attributes on failed requests.

## Impact

- Affected specs: `openspec/specs/interceptors/spec.md`, `openspec/specs/http-server-core/spec.md`
- Affected code (expected): `pkg/grpc/interceptors/{tracing.go,logging.go,metrics.go,interceptors_test.go}`, `pkg/api/middleware/{tracing.go,logger.go,tracing_test.go}`
- Affected test scope: targeted interceptor/middleware tracing correlation and panic-path status mapping tests
- No public API shape changes; behavior and observability semantics are tightened.
