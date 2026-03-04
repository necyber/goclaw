## ADDED Requirements

### Requirement: Trace-correlated request logging in interceptor paths
The system MUST include trace correlation fields in gRPC request logs when an active span context exists in interceptor execution context.

#### Scenario: Unary request log includes trace correlation
- **WHEN** a unary gRPC request is logged during an active span
- **THEN** the logging path MUST include `trace_id` and `span_id` fields derived from request context.

#### Scenario: Stream request log includes trace correlation
- **WHEN** a stream gRPC request lifecycle is logged during an active span
- **THEN** the logging path MUST include `trace_id` and `span_id` fields derived from stream context.

### Requirement: Panic-aware tracing status mapping
The tracing interceptor MUST preserve failure semantics when handler panics are recovered by outer recovery interceptors.

#### Scenario: Unary panic is traced as failure
- **WHEN** a unary handler panics and recovery converts it to gRPC `Internal`
- **THEN** the tracing span MUST record error attributes and error completion status before panic is re-thrown to recovery.

#### Scenario: Stream panic is traced as failure
- **WHEN** a stream handler panics and recovery converts it to gRPC `Internal`
- **THEN** the tracing span MUST record error attributes and error completion status before panic is re-thrown to recovery.

### Requirement: gRPC metrics trace correlation metadata
The metrics interceptor MUST preserve trace correlation metadata for metrics backends that support exemplars.

#### Scenario: Unary metrics recorded during active span
- **WHEN** unary request metrics are recorded with a valid span context
- **THEN** implementation MUST attach trace correlation metadata where collector APIs support exemplars.

#### Scenario: Stream metrics recorded during active span
- **WHEN** stream lifecycle metrics are recorded with a valid span context
- **THEN** implementation MUST attach trace correlation metadata where collector APIs support exemplars.

#### Scenario: Metrics backend without exemplar support
- **WHEN** metrics collector does not support exemplar APIs
- **THEN** interceptor MUST continue recording baseline metrics without failing request handling.
