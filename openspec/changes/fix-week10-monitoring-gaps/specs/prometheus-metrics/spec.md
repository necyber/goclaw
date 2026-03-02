## MODIFIED Requirements

### Requirement: Metrics HTTP endpoint exposition
The metrics system SHALL expose a configured metrics endpoint for Prometheus scraping and reject invalid endpoint configuration at startup.

#### Scenario: Expose metrics endpoint
- **WHEN** metrics server starts on configured port
- **THEN** configured metrics path endpoint serves Prometheus text format metrics

#### Scenario: Metrics endpoint authentication
- **WHEN** metrics endpoint receives request without valid credentials (if auth enabled)
- **THEN** endpoint returns 401 Unauthorized

#### Scenario: Concurrent scrape requests
- **WHEN** multiple Prometheus instances scrape metrics simultaneously
- **THEN** endpoint handles requests concurrently without blocking

#### Scenario: Invalid metrics path configuration
- **WHEN** metrics path configuration is empty or does not start with "/"
- **THEN** application MUST fail validation before server startup

### Requirement: Metric label cardinality control
The metrics system SHALL limit label cardinality to prevent memory exhaustion using bounded label-value admission.

#### Scenario: High cardinality label detection
- **WHEN** metric label has more than 1000 unique values
- **THEN** system logs warning and drops new label values

#### Scenario: Label value sanitization
- **WHEN** recording metric with user-provided label value
- **THEN** system sanitizes value to prevent cardinality explosion

#### Scenario: Dropped label value accounting
- **WHEN** system drops a new label value because the cardinality limit is reached
- **THEN** system increments an internal dropped-label counter metric for operational visibility
