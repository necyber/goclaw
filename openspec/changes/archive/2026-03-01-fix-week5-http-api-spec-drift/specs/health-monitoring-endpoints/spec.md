## ADDED Requirements

### Requirement: Health endpoint payload schema is canonical
`GET /health` SHALL return the canonical liveness payload structure.

#### Scenario: Healthy liveness response
- **WHEN** the service is alive
- **THEN** the response MUST be HTTP `200` and include `status: healthy` and `timestamp`

#### Scenario: Unhealthy liveness response
- **WHEN** the service is not healthy
- **THEN** the response MUST be HTTP `503` and include `status: unhealthy`, `timestamp`, and `error`

### Requirement: Readiness endpoint payload includes dependency checks
`GET /ready` SHALL report readiness status and dependency check details.

#### Scenario: Ready response includes checks
- **WHEN** all required dependencies are ready
- **THEN** the response MUST be HTTP `200` with `status: ready`, `timestamp`, and `checks` map including `engine` and `storage`

#### Scenario: Not-ready response includes failure detail
- **WHEN** one or more required dependencies are not ready
- **THEN** the response MUST be HTTP `503` with `status: not_ready`, `timestamp`, `checks`, and `error`

### Requirement: Status endpoint exposes structured runtime detail
`GET /status` SHALL expose a detailed status envelope for diagnostics and monitoring.

#### Scenario: Status response includes core metadata
- **WHEN** a client calls `GET /status`
- **THEN** the response MUST include `status`, `version`, `uptime`, and `timestamp`

#### Scenario: Status response includes component and system sections
- **WHEN** a client calls `GET /status`
- **THEN** the response MUST include `components` and `system` objects with runtime detail fields
