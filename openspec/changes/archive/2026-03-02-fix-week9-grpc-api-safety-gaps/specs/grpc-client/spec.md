## MODIFIED Requirements

### Requirement: Connection pooling
The system SHALL manage connection pooling for efficient resource usage.

#### Scenario: Connection reuse
- **WHEN** multiple RPCs are made
- **THEN** client MUST reuse existing connection instead of creating new ones

#### Scenario: Connection health check
- **WHEN** connection becomes unhealthy
- **THEN** client MUST detect health failure using service-scoped health checks and automatically reconnect when transport state is recoverable

#### Scenario: Service health status alignment
- **WHEN** client checks health for a specific gRPC service
- **THEN** client MUST receive `SERVING` when server has registered serving status for that service

#### Scenario: Graceful close
- **WHEN** client is closed
- **THEN** client MUST drain in-flight requests and close connection cleanly
