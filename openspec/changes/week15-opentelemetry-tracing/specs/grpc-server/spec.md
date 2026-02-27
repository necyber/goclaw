## ADDED Requirements

### Requirement: Tracing provider lifecycle integration
The gRPC server runtime SHALL integrate OpenTelemetry provider lifecycle into startup and shutdown.

#### Scenario: Startup with tracing enabled
- **WHEN** gRPC server starts with tracing enabled and valid config
- **THEN** server bootstrap MUST initialize tracing provider before serving requests

#### Scenario: Startup with invalid tracing config
- **WHEN** tracing configuration is invalid
- **THEN** gRPC server startup MUST fail with explicit configuration error and MUST NOT enter serving state

### Requirement: Global propagator registration
The gRPC runtime SHALL register supported text-map propagators at startup.

#### Scenario: Register propagators
- **WHEN** tracing is enabled
- **THEN** runtime MUST register W3C Trace Context and Baggage propagators for inbound and outbound propagation

### Requirement: Tracing interceptors in server chains
The gRPC server SHALL include tracing interceptors in unary and stream interceptor chains when tracing is enabled.

#### Scenario: Unary chain includes tracing
- **WHEN** unary interceptor chain is assembled with tracing enabled
- **THEN** chain MUST include tracing interceptor so handlers execute with span context

#### Scenario: Stream chain includes tracing
- **WHEN** stream interceptor chain is assembled with tracing enabled
- **THEN** chain MUST include tracing interceptor so stream handlers execute with span context

### Requirement: Graceful tracing shutdown on server stop
The gRPC runtime SHALL flush tracing pipeline during shutdown.

#### Scenario: Server shutdown flush
- **WHEN** gRPC server receives shutdown signal
- **THEN** runtime MUST attempt tracing flush/shutdown with timeout as part of graceful stop sequence

### Requirement: Tracing startup diagnostics
The gRPC runtime SHALL emit startup diagnostics for tracing state.

#### Scenario: Tracing enabled diagnostics
- **WHEN** server starts with tracing enabled
- **THEN** startup logs MUST include tracing enabled state, exporter type, and endpoint summary without leaking secrets
