## 1. gRPC Observability Corrections

- [x] 1.1 Update gRPC logging interceptors to use context-aware logger entrypoints so active spans emit `trace_id`/`span_id`.
- [x] 1.2 Harden gRPC tracing interceptors for panic paths: record span failure semantics, then re-panic to preserve recovery behavior.
- [x] 1.3 Add/adjust gRPC tracing tests to cover panic-to-Internal mapping with span error assertions.
- [x] 1.4 Extend gRPC metrics interceptors to preserve trace correlation metadata for exemplar-capable backends with non-exemplar fallback.
- [x] 1.5 Add/adjust gRPC metrics tests for exemplar-capable and fallback collector paths.

## 2. HTTP Tracing Failure Semantics

- [x] 2.1 Extend HTTP tracing middleware to emit explicit error attributes for 4xx/5xx outcomes.
- [x] 2.2 Ensure recovered panic paths emit failure attributes/status on tracing spans.
- [x] 2.3 Add/adjust HTTP tracing tests for failed status and recovered panic error-attribute assertions.

## 3. Verification and Task Closure

- [x] 3.1 Run targeted tests for changed packages (`pkg/grpc/interceptors`, `pkg/api/middleware`, and related telemetry paths).
- [x] 3.2 Mark all completed items in this tasks file and verify `openspec instructions apply` reports ready/all_done.
