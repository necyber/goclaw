## ADDED Requirements

### Requirement: OpenTelemetry tracing interceptor semantics
The system SHALL implement tracing interceptors using OpenTelemetry span semantics for unary and streaming RPCs.

#### Scenario: Unary RPC tracing span
- **WHEN** a unary gRPC call is handled with tracing enabled
- **THEN** interceptor MUST create a server span with RPC method attributes and close it on completion with mapped status/error fields

#### Scenario: Streaming RPC tracing span
- **WHEN** a streaming gRPC call is handled with tracing enabled
- **THEN** interceptor MUST create a stream-lifecycle span that records stream completion status and error information

### Requirement: gRPC trace context extraction and injection
The system SHALL propagate OpenTelemetry trace context through gRPC metadata.

#### Scenario: Extract incoming trace metadata
- **WHEN** request metadata includes W3C trace context
- **THEN** tracing interceptor MUST extract and continue that trace in handler context

#### Scenario: Inject outgoing trace metadata
- **WHEN** runtime performs downstream gRPC calls from request context
- **THEN** interceptor/client path MUST inject current trace context into outgoing metadata

### Requirement: Trace correlation for observability signals
The system SHALL expose trace correlation fields to other observability components.

#### Scenario: Correlate logs with trace context
- **WHEN** logging occurs during an active span
- **THEN** log context MUST include trace ID and span ID

#### Scenario: Correlate metrics with trace exemplar context
- **WHEN** request metrics are recorded during an active span
- **THEN** implementation MUST preserve trace correlation metadata where the metrics backend supports exemplars

### Requirement: Tracing disabled behavior
The system SHALL support lightweight behavior when tracing is disabled.

#### Scenario: Disabled tracing path
- **WHEN** tracing is disabled by configuration
- **THEN** tracing interceptor path MUST avoid exporter initialization and MUST add only minimal overhead
