## MODIFIED Requirements

### Requirement: gRPC server initialization
The system SHALL initialize a gRPC server on the configured port with proper lifecycle management.

#### Scenario: Server startup
- **WHEN** application starts
- **THEN** gRPC server MUST start on the configured port (default 9090) and register all service handlers

#### Scenario: Graceful shutdown
- **WHEN** shutdown signal is received
- **THEN** gRPC server MUST call `GracefulStop()` to drain in-flight requests before terminating

#### Scenario: Forced shutdown timeout
- **WHEN** graceful shutdown times out and server falls back to `Stop()`
- **THEN** server runtime state MUST transition to stopped before returning

#### Scenario: Concurrent operation with HTTP
- **WHEN** both HTTP and gRPC servers are enabled
- **THEN** both servers MUST run concurrently without port conflicts

### Requirement: Interceptor chain
The system SHALL apply interceptors for cross-cutting concerns in the correct order.

#### Scenario: Unary interceptor chain
- **WHEN** processing unary RPCs
- **THEN** interceptors MUST execute in order: recovery -> request_id -> auth -> authorization -> rate_limit -> validation -> logging -> metrics -> tracing -> handler

#### Scenario: Stream interceptor chain
- **WHEN** processing streaming RPCs
- **THEN** stream interceptors MUST execute in order: recovery -> request_id -> auth -> authorization -> rate_limit -> validation -> logging -> metrics -> tracing -> handler

#### Scenario: Tracing disabled still enforces base chain
- **WHEN** tracing is disabled by configuration
- **THEN** all non-tracing interceptors in the default chain MUST still be active

### Requirement: Health check service
The system SHALL implement the gRPC health check protocol.

#### Scenario: Health check endpoint
- **WHEN** client calls Check RPC
- **THEN** server MUST return `SERVING` status if engine is healthy

#### Scenario: Service-specific health
- **WHEN** client requests health of specific service
- **THEN** server MUST return per-service health status for registered gRPC services

#### Scenario: Watch health changes
- **WHEN** client calls Watch RPC
- **THEN** server MUST stream health status changes

### Requirement: Connection management
The system SHALL handle connection lifecycle and resource cleanup.

#### Scenario: Connection limits
- **WHEN** max connections is configured
- **THEN** server MUST enforce the configured limit semantics explicitly and consistently in runtime behavior

#### Scenario: Idle timeout
- **WHEN** connection is idle beyond configured timeout
- **THEN** server MUST close the connection

#### Scenario: Keepalive settings
- **WHEN** keepalive is configured
- **THEN** server MUST send keepalive pings and enforce client keepalive policy
