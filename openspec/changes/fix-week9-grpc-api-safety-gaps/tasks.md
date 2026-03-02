## 1. Server And Interceptor Hardening

- [x] 1.1 Update `pkg/grpc/server.go` to always mount the default interceptor chain and keep tracing as a conditional extension.
- [x] 1.2 Ensure forced shutdown timeout path in `Server.Stop` transitions runtime state to stopped and keeps state consistent for subsequent lifecycle calls.
- [x] 1.3 Align gRPC server health initialization to set both global and service-specific serving statuses for registered services.
- [x] 1.4 Add or update unit tests for interceptor-chain activation and server stop state transitions.

## 2. Authentication And Authorization Behavior

- [x] 2.1 Replace placeholder token-validation behavior in `pkg/grpc/interceptors/auth.go` with enforceable validation semantics and explicit identity extraction.
- [x] 2.2 Update `pkg/grpc/interceptors/authorization.go` to derive and enforce admin role from authenticated identity context.
- [x] 2.3 Add interceptor tests covering valid auth, invalid auth, admin authorization pass/fail, and health-check bypass behavior.

## 3. Batch Pagination Safety

- [x] 3.1 Harden page-token parsing and bounds checks in `pkg/grpc/handlers/batch.go` for all paginated batch handlers before slice operations.
- [x] 3.2 Return `InvalidArgument` for malformed, negative, or out-of-range offsets instead of allowing panic paths.
- [x] 3.3 Add regression tests for invalid tokens and out-of-range offsets to verify panic-free behavior.

## 4. Streaming Concurrency Safety

- [ ] 4.1 Refactor `StreamLogs` in `pkg/grpc/handlers/streaming.go` to synchronize dynamic filter updates between recv/send goroutines.
- [ ] 4.2 Refactor `CleanupStaleSubscribers` in `pkg/grpc/streaming/registry.go` to avoid lock re-entry and deadlock risk.
- [ ] 4.3 Add streaming and registry tests that cover dynamic filter updates and stale-subscriber cleanup under concurrent activity.

## 5. Client Health Alignment And Validation

- [ ] 5.1 Update `pkg/grpc/client/client.go` health-check behavior to align with service-scoped server health registration.
- [ ] 5.2 Add client tests to verify healthy service checks succeed with service-specific queries.
- [ ] 5.3 Run verification suites: `go test ./pkg/grpc/...` and targeted race suites for hardened paths (`handlers`, `streaming`, `interceptors`).
