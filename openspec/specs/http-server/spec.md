# http-server Specification

## Purpose
Migrated from legacy OpenSpec format while preserving existing requirement and scenario content.

## Requirements

### Requirement: WebSocket upgrade endpoint

The system SHALL provide a WebSocket upgrade endpoint at /ws/events for real-time event streaming.

#### Scenario: WebSocket handshake
- **WHEN** a client sends a WebSocket upgrade request to /ws/events
- **THEN** the server upgrades the connection using gorilla/websocket and begins event streaming

#### Scenario: Reject non-WebSocket request
- **WHEN** a regular HTTP GET request is made to /ws/events
- **THEN** the server returns 400 Bad Request

### Requirement: UI route registration

The system SHALL register a UI route group in the HTTP router using the configured `ui.base_path` (default `/ui`) for serving the Web UI.

#### Scenario: Register UI routes with default base path
- **WHEN** the HTTP server starts with UI enabled and `ui.base_path` is empty
- **THEN** the router registers /ui/* to serve static files or proxy to dev server

#### Scenario: Register UI routes with custom base path
- **WHEN** the HTTP server starts with UI enabled and `ui.base_path` is `/dashboard`
- **THEN** the router registers `/dashboard/*` to serve static files or proxy to dev server

#### Scenario: UI disabled
- **WHEN** the HTTP server starts with UI disabled (`ui.enabled: false`)
- **THEN** the router does not register UI routes

### Requirement: UI configuration

The system SHALL support UI-related configuration options in the server config and apply them consistently for serving and proxying.

#### Scenario: Enable UI
- **WHEN** the config contains `ui.enabled: true`
- **THEN** the server serves the Web UI at the configured base path (default `/ui`)

#### Scenario: Custom base path
- **WHEN** the config contains `ui.base_path: "/dashboard"`
- **THEN** the server serves the Web UI at `/dashboard` instead of `/ui`

#### Scenario: Dev proxy configuration
- **WHEN** the config contains `ui.dev_proxy: "http://localhost:5173"`
- **THEN** the server proxies UI requests to the specified Vite dev server

#### Scenario: Dev proxy path rewrite
- **WHEN** `ui.base_path` is `/dashboard`, `ui.dev_proxy` is configured, and a request is made to `/dashboard/workflows/abc`
- **THEN** the server proxies the request to upstream path `/workflows/abc`

### Requirement: CORS for WebSocket

The system SHALL apply CORS policy to WebSocket connections.

#### Scenario: Allow WebSocket from same origin
- **WHEN** a WebSocket upgrade request comes from the same origin as the server
- **THEN** the server accepts the connection

#### Scenario: Allow WebSocket from configured origins
- **WHEN** a WebSocket upgrade request comes from an origin listed in the CORS allowed origins
- **THEN** the server accepts the connection

#### Scenario: Reject WebSocket from unknown origin
- **WHEN** a WebSocket upgrade request comes from an unlisted origin and CORS is not set to allow all
- **THEN** the server rejects the connection with 403 Forbidden

