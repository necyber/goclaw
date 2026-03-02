## Why

The archived Week9 gRPC implementation contains multiple correctness and safety gaps that can cause missing security controls, panic crashes, deadlocks, data races, and false health failures in production paths. We need a focused hardening change now because these gaps affect core runtime behavior even when existing unit tests pass.

## What Changes

- Wire the full default gRPC interceptor chain into server startup, while keeping tracing toggle behavior compatible with existing config.
- Harden batch pagination and token handling to prevent out-of-range panics and return `InvalidArgument` for invalid offsets.
- Fix streaming subscriber lifecycle concurrency issues, including deadlock-prone cleanup logic and unsafe concurrent filter mutation in bidirectional streams.
- Align client/server health check semantics so service-specific health checks are reliable.
- Tighten operational robustness for shutdown and connection-limit semantics (state transitions and configuration mapping).
- Replace placeholder authn/authz behavior with explicit non-placeholder behavior compatible with admin access requirements.
- Add race-focused and failure-mode regression tests for the above paths.

## Capabilities

### New Capabilities

- `grpc-runtime-safety-hardening`: Cross-cutting hardening for concurrency/race/deadlock and panic-proof request handling in gRPC runtime paths.

### Modified Capabilities

- `grpc-server`: Interceptor chain wiring, shutdown state correctness, connection-limit semantics, and health serving behavior.
- `interceptors`: Authentication/authorization behavior and effective enforcement through active chain registration.
- `batch-operations`: Pagination token validation and panic-free bounds handling.
- `streaming-support`: Subscriber cleanup safety and concurrent stream filter update correctness.
- `grpc-client`: Health check behavior against service-scoped health statuses.

## Impact

- Affected code: `pkg/grpc/server.go`, `pkg/grpc/interceptors/*`, `pkg/grpc/handlers/batch.go`, `pkg/grpc/handlers/streaming.go`, `pkg/grpc/streaming/registry.go`, `pkg/grpc/client/client.go`, and related tests.
- API behavior: stricter validation and safer error responses for invalid batch pagination tokens; reliable health status checks.
- Runtime behavior: security interceptors and observability interceptors become effective in normal startup; reduced risk of panic/deadlock/data race under load.
- Dependencies: no new external dependency required; existing gRPC/Prometheus/OpenTelemetry stack reused.
