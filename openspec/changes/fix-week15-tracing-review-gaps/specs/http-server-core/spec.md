## ADDED Requirements

### Requirement: HTTP failed span error attributes
HTTP tracing middleware MUST add explicit error attributes for failed request outcomes.

#### Scenario: Error attributes for 4xx response
- **WHEN** an HTTP request completes with a 4xx status code
- **THEN** the span MUST include explicit error attributes describing failure class and response status.

#### Scenario: Error attributes for 5xx response
- **WHEN** an HTTP request completes with a 5xx status code
- **THEN** the span MUST include explicit error attributes describing failure class and response status.

#### Scenario: Error attributes for recovered panic
- **WHEN** request handling panics and panic is recovered by middleware chain
- **THEN** the tracing span MUST include explicit panic-related error attributes and error completion status.
