## ADDED Requirements

### Requirement: Documentation endpoint contract includes `/docs`
HTTP routing SHALL expose an interactive API documentation endpoint at `/docs`.

#### Scenario: Docs endpoint is reachable
- **WHEN** a client calls `GET /docs` or `GET /docs/`
- **THEN** the server MUST return documentation UI content instead of `404`

#### Scenario: Backward-compatible documentation route
- **WHEN** a deployment still uses `/swagger/*`
- **THEN** documentation routing MAY keep `/swagger/*` as a compatibility alias

### Requirement: Generated API specification uses OpenAPI 3.0.3 or newer
Generated API docs SHALL publish an OpenAPI 3.x document aligned with Week 5 documentation baseline.

#### Scenario: OpenAPI major version check
- **WHEN** generated spec artifacts are produced for release
- **THEN** the document version MUST be `3.0.3` or later

#### Scenario: Core metadata contract
- **WHEN** generated spec artifacts are produced
- **THEN** spec metadata MUST include title, version, description, and a valid server/base path mapping for HTTP endpoints

### Requirement: Documentation and runtime endpoint maps stay synchronized
Documentation SHALL describe implemented HTTP endpoints and their effective request/response contracts.

#### Scenario: Workflow endpoint documentation parity
- **WHEN** workflow endpoints are available in runtime router
- **THEN** docs MUST include matching path, parameters, and status-code contracts

#### Scenario: Health endpoint documentation parity
- **WHEN** health endpoints are available in runtime router
- **THEN** docs MUST include `/health`, `/ready`, and `/status` with matching response schemas
