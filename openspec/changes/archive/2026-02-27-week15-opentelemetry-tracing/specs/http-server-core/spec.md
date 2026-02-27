## ADDED Requirements

### Requirement: HTTP tracing middleware
The HTTP server SHALL create OpenTelemetry server spans for inbound API requests when tracing is enabled.

#### Scenario: Inbound request with traceparent
- **WHEN** an HTTP request includes valid trace context headers
- **THEN** middleware MUST continue that trace and create a server span bound to request context

#### Scenario: Inbound request without trace context
- **WHEN** an HTTP request has no trace headers
- **THEN** middleware MUST create a new root server span for the request

### Requirement: HTTP span attributes and status mapping
The HTTP server SHALL record standard HTTP span attributes and map response outcomes to span status.

#### Scenario: Successful request span
- **WHEN** an HTTP request returns 2xx status
- **THEN** span MUST record method, route, status code, duration, and success status

#### Scenario: Failed request span
- **WHEN** an HTTP request returns 4xx/5xx or panic is recovered
- **THEN** span MUST record error attributes and completion status reflecting failure semantics

### Requirement: Trace context propagation for outbound HTTP
The HTTP runtime SHALL propagate trace context for outbound HTTP requests made from request-scoped context.

#### Scenario: Outbound call from handler context
- **WHEN** handler performs downstream HTTP call using request context
- **THEN** runtime/client wrapper MUST inject current trace context headers into outbound request

### Requirement: Low-value endpoint control
The HTTP server SHALL support tracing controls for low-value endpoints.

#### Scenario: Health endpoint tracing policy
- **WHEN** request targets health/readiness endpoints
- **THEN** runtime MUST apply configured policy to skip or sample these spans at reduced volume
