## Context

Goclaw currently provides an HTTP/JSON API for workflow orchestration. As the system evolves toward distributed multi-agent orchestration (Phase 2), internal service-to-service communication needs a more efficient protocol. The HTTP API will remain for external clients, while gRPC will handle internal cluster communication.

Current state:
- HTTP API on port 8080 with REST endpoints
- Engine supports workflow submission, status queries, and cancellation
- No streaming support for real-time updates
- No batch operations for bulk workflow management
- No admin API for runtime configuration or cluster management

Constraints:
- Must maintain backward compatibility with existing HTTP API
- Must support both TLS and mTLS for security
- Must integrate with existing metrics and logging infrastructure
- Must handle high throughput (>2,000 req/s per endpoint)
- Must support graceful shutdown without losing in-flight requests

Stakeholders:
- Internal services requiring efficient inter-service communication
- Operators needing admin APIs for cluster management
- Developers building on Goclaw requiring streaming updates

## Goals / Non-Goals

**Goals:**
- Add gRPC server on port 9090 for internal service-to-service communication
- Implement full workflow API parity with HTTP (submit, list, status, cancel, task results)
- Provide streaming APIs for real-time workflow and task updates
- Support batch operations for bulk workflow management
- Implement admin APIs for engine control and cluster coordination
- Create Go client SDK with connection pooling and retry logic
- Integrate authentication, logging, metrics, and tracing via interceptors

**Non-Goals:**
- Replace or deprecate existing HTTP API (both will coexist)
- Implement gRPC-Web for browser clients (HTTP API serves this use case)
- Support languages other than Go for client SDK (focus on Go first)
- Implement persistent streaming (streams are session-based, not durable)
- Provide GraphQL or other query languages over gRPC

## Decisions

### Decision 1: Dual-protocol architecture (HTTP + gRPC)

**Choice:** Run HTTP and gRPC servers concurrently on different ports.

**Rationale:**
- HTTP API serves external clients and browser-based tools
- gRPC serves internal services with better performance and streaming
- Allows gradual migration without breaking existing integrations
- Each protocol optimized for its use case

**Alternatives considered:**
- gRPC-only: Would break existing HTTP clients and require migration
- HTTP-only: Cannot provide efficient streaming or type-safe contracts
- gRPC-Web: Adds complexity and doesn't match HTTP API feature parity

### Decision 2: Protocol Buffer organization

**Choice:** Organize proto files by service domain (workflow.proto, streaming.proto, batch.proto, admin.proto) under `api/proto/goclaw/v1/`.

**Rationale:**
- Clear separation of concerns by service type
- Easier to maintain and version independently
- Follows gRPC best practices for API organization
- Package versioning (v1) enables future breaking changes via v2

**Alternatives considered:**
- Single monolithic proto file: Harder to maintain and navigate
- Per-RPC proto files: Too granular, creates file sprawl
- No versioning: Makes breaking changes difficult to manage

### Decision 3: Interceptor chain architecture

**Choice:** Use grpc-ecosystem/go-grpc-middleware for chaining interceptors in order: recovery → request_id → auth → authorization → rate_limit → validation → logging → metrics → tracing.

**Rationale:**
- Recovery must be outermost to catch all panics
- Auth/authz before business logic to fail fast
- Logging/metrics/tracing after auth to include identity context
- Established library with community support

**Alternatives considered:**
- Custom interceptor chaining: Reinvents the wheel, more maintenance
- Middleware per-service: Inconsistent behavior across services
- No interceptors: Cross-cutting concerns scattered in handlers

### Decision 4: Streaming architecture

**Choice:** Server-side streaming for workflow/task updates, bidirectional streaming for logs. Use in-memory pub/sub for update distribution.

**Rationale:**
- Server-side streaming sufficient for status updates (client only receives)
- Bidirectional streaming for logs allows dynamic filter updates
- In-memory pub/sub is fast and simple for session-based streams
- No persistence needed (streams are real-time, not durable)

**Alternatives considered:**
- Persistent streaming via message queue: Overkill for real-time updates
- Polling-based updates: Inefficient, defeats purpose of streaming
- Client-side streaming: No use case for client pushing data streams

### Decision 5: Authentication strategy

**Choice:** Support both token-based auth (bearer tokens) and mTLS. Token auth for development, mTLS for production clusters.

**Rationale:**
- Token auth easier for development and testing
- mTLS provides stronger security for production service-to-service
- Both methods integrate with same authorization interceptor
- Flexibility for different deployment environments

**Alternatives considered:**
- mTLS-only: Too complex for development environments
- Token-only: Insufficient security for production clusters
- OAuth2: Overkill for internal service-to-service communication

### Decision 6: Client SDK design

**Choice:** Provide high-level Go client with connection pooling, automatic retry, and helper methods wrapping generated gRPC stubs.

**Rationale:**
- Generated stubs are low-level and verbose
- Connection pooling improves performance
- Automatic retry handles transient failures transparently
- Helper methods provide idiomatic Go API

**Alternatives considered:**
- Expose generated stubs directly: Poor developer experience
- No client SDK: Every consumer reimplements connection management
- Multi-language SDKs: Premature, focus on Go first

### Decision 7: Batch operation implementation

**Choice:** Process batch operations in parallel using worker pools, with optional atomic mode for all-or-nothing semantics.

**Rationale:**
- Parallel processing maximizes throughput
- Atomic mode needed for workflows with dependencies
- Worker pools prevent resource exhaustion
- Idempotency keys prevent duplicate submissions on retry

**Alternatives considered:**
- Sequential processing: Too slow for large batches
- Always atomic: Reduces throughput for independent workflows
- No idempotency: Retries create duplicate workflows

### Decision 8: Engine integration for streaming

**Choice:** Add observer pattern to engine for state change notifications. Streaming handlers subscribe to updates and push to gRPC streams.

**Rationale:**
- Decouples engine from gRPC layer
- Multiple subscribers can watch same workflow
- Clean separation of concerns
- Minimal changes to existing engine code

**Alternatives considered:**
- Polling engine state: Inefficient, adds latency
- Direct engine-to-gRPC coupling: Violates separation of concerns
- External message queue: Adds complexity and dependencies

## Risks / Trade-offs

**[Risk] gRPC and HTTP servers compete for resources**
→ Mitigation: Configure separate worker pools and connection limits per server. Monitor resource usage and adjust limits.

**[Risk] Streaming connections consume memory at scale**
→ Mitigation: Implement connection limits, idle timeouts, and backpressure handling. Close slow consumer streams.

**[Risk] Protocol Buffer breaking changes break clients**
→ Mitigation: Use field numbers carefully, mark deprecated fields as reserved, version packages (v1, v2).

**[Risk] Interceptor chain adds latency**
→ Mitigation: Keep interceptors lightweight, measure per-interceptor latency, optimize hot paths.

**[Risk] Batch operations can overwhelm engine**
→ Mitigation: Enforce batch size limits, use worker pools, implement rate limiting per client.

**[Risk] mTLS certificate management complexity**
→ Mitigation: Document certificate setup, provide example configs, support cert rotation without restart.

**[Risk] Client SDK retry logic can amplify failures**
→ Mitigation: Use exponential backoff, limit max retries, don't retry non-idempotent operations.

**[Trade-off] Dual protocol increases maintenance burden**
→ Accepted: Both protocols serve different use cases and will coexist long-term.

**[Trade-off] In-memory streaming doesn't survive restarts**
→ Accepted: Streams are session-based and real-time. Clients reconnect after restart.

**[Trade-off] Go-only client SDK limits adoption**
→ Accepted: Focus on Go first, add other languages based on demand.

## Migration Plan

**Phase 1: Core gRPC infrastructure (Week 9)**
1. Add proto definitions and generate Go code
2. Implement gRPC server with basic interceptors
3. Add workflow service handlers (submit, list, status, cancel)
4. Update config to support gRPC settings
5. Modify main.go to start both HTTP and gRPC servers
6. Add integration tests for gRPC endpoints

**Phase 2: Streaming and batch operations (Week 10)**
1. Add observer pattern to engine for state notifications
2. Implement streaming service handlers
3. Add batch operation handlers
4. Implement backpressure and connection management
5. Add streaming integration tests

**Phase 3: Admin API and client SDK (Week 11)**
1. Implement admin service handlers
2. Add admin authentication and authorization
3. Create Go client SDK with connection pooling
4. Add retry logic and helper methods
5. Write client SDK examples and documentation

**Rollback strategy:**
- gRPC server can be disabled via config without affecting HTTP API
- Feature flag for gRPC endpoints allows gradual rollout
- Clients can fall back to HTTP API if gRPC unavailable
- No database schema changes, so rollback is safe

**Deployment:**
1. Deploy with gRPC disabled initially
2. Enable gRPC in staging environment
3. Test internal services against gRPC API
4. Enable gRPC in production with monitoring
5. Gradually migrate internal services from HTTP to gRPC

## Open Questions

1. **Should we support gRPC reflection in production?**
   - Pro: Easier debugging with grpcurl
   - Con: Security risk exposing service definitions
   - Proposal: Disable by default, enable via config flag

2. **What should be the default batch size limit?**
   - Need to benchmark engine throughput with batch operations
   - Proposal: Start with 100, adjust based on performance testing

3. **Should admin API require separate authentication?**
   - Pro: Defense in depth, separate admin credentials
   - Con: More complex auth setup
   - Proposal: Use same auth mechanism but require admin role claim

4. **How to handle streaming backpressure?**
   - Drop updates vs buffer vs close stream
   - Proposal: Buffer up to limit, then close stream with ResourceExhausted

5. **Should we support HTTP/2 for HTTP API?**
   - Would enable HTTP/2 streaming as alternative to gRPC
   - Proposal: Defer to future, focus on gRPC for streaming
