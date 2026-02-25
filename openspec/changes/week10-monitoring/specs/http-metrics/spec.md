## ADDED Requirements

### Requirement: HTTP request count metrics
The metrics system SHALL track HTTP request counts by method, path, and status code.

#### Scenario: Record successful request
- **WHEN** HTTP request completes with 2xx status code
- **THEN** system increments http_requests_total counter with method, path, and status labels

#### Scenario: Record client error request
- **WHEN** HTTP request completes with 4xx status code
- **THEN** system increments http_requests_total counter with method, path, and status="4xx" labels

#### Scenario: Record server error request
- **WHEN** HTTP request completes with 5xx status code
- **THEN** system increments http_requests_total counter with method, path, and status="5xx" labels

#### Scenario: Normalize request paths
- **WHEN** recording HTTP request metrics
- **THEN** system normalizes path parameters (e.g., /api/v1/workflows/:id instead of /api/v1/workflows/abc123)

### Requirement: HTTP request duration metrics
The metrics system SHALL measure HTTP request latency from receipt to response.

#### Scenario: Record request duration
- **WHEN** HTTP request completes
- **THEN** system records duration in http_request_duration_seconds histogram with method and path labels

#### Scenario: Duration histogram buckets
- **WHEN** recording request duration
- **THEN** system uses buckets [0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5] seconds

#### Scenario: Exclude metrics endpoint from metrics
- **WHEN** /metrics endpoint is accessed
- **THEN** system does not record metrics for this request to avoid recursion

### Requirement: HTTP active connections metrics
The metrics system SHALL track the current number of active HTTP connections.

#### Scenario: Increment active connections
- **WHEN** new HTTP connection is established
- **THEN** system increments http_active_connections gauge

#### Scenario: Decrement active connections
- **WHEN** HTTP connection is closed
- **THEN** system decrements http_active_connections gauge

### Requirement: HTTP metrics middleware integration
The metrics system SHALL provide middleware for automatic HTTP metrics collection.

#### Scenario: Middleware wraps request handler
- **WHEN** HTTP request enters middleware chain
- **THEN** middleware records start time and increments active connections

#### Scenario: Middleware records on completion
- **WHEN** HTTP request completes (success or error)
- **THEN** middleware calculates duration, records metrics, and decrements active connections

#### Scenario: Middleware handles panics
- **WHEN** HTTP handler panics during request processing
- **THEN** middleware still records metrics with status=500 before propagating panic

#### Scenario: Middleware integration with chi router
- **WHEN** metrics middleware is added to chi router
- **THEN** middleware is positioned after recovery middleware but before business logic handlers
