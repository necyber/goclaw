## MODIFIED Requirements

### Requirement: Authentication interceptor
The system SHALL implement authentication interceptor to verify client identity.

#### Scenario: Token-based authentication
- **WHEN** request includes bearer token in metadata
- **THEN** interceptor MUST validate token integrity and claims and extract user identity from validated token content

#### Scenario: mTLS authentication
- **WHEN** request uses mutual TLS
- **THEN** interceptor MUST verify client certificate and extract identity from certificate subject information

#### Scenario: Authentication failure
- **WHEN** authentication fails
- **THEN** interceptor MUST return `Unauthenticated` status and reject request

#### Scenario: Skip authentication for health checks
- **WHEN** request is for health check service
- **THEN** interceptor MUST allow request without authentication

### Requirement: Authorization interceptor
The system SHALL implement authorization interceptor to enforce access control.

#### Scenario: Role-based access control
- **WHEN** authenticated user makes request
- **THEN** interceptor MUST verify user has required role for the operation

#### Scenario: Workflow ownership check
- **WHEN** user requests workflow operation
- **THEN** interceptor MUST verify user owns or has access to the workflow

#### Scenario: Admin operation authorization
- **WHEN** user calls admin API
- **THEN** interceptor MUST verify user has admin role derived from validated authentication identity

#### Scenario: Authorization failure
- **WHEN** authorization check fails
- **THEN** interceptor MUST return `PermissionDenied` status
