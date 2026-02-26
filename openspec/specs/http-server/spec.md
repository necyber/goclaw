## ADDED Requirements

### Requirement: WebSocket upgrade endpoint

The system SHALL provide a WebSocket upgrade endpoint at /ws/events for real-time event streaming.

#### Scenario: WebSocket handshake
- **WHEN** a client sends a WebSocket upgrade request to /ws/events
- **THEN** the server upgrades the connection using gorilla/websocket and begins event streaming

#### Scenario: Reject non-WebSocket request
- **WHEN** a regular HTTP GET request is made to /ws/events
- **THEN** the server returns 400 Bad Request

### Requirement: UI route registration

The system SHALL register the /ui/* route group in the HTTP router for serving the Web UI.

#### Scenario: Register UI routes
- **WHEN** the HTTP server starts with UI enabled
- **THEN** the router registers /ui/* to serve static files or proxy to dev server

#### Scenario: UI disabled
- **WHEN** the HTTP server starts with UI disabled (config ui.enabled: false)
- **THEN** the router does not register /ui/* routes

### Requirement: UI configuration

The system SHALL support UI-related configuration options in the server config.

#### Scenario: Enable UI
- **WHEN** the config contains ui.enabled: true
- **THEN** the server serves the Web UI at /ui

#### Scenario: Custom base path
- **WHEN** the config contains ui.base_path: "/dashboard"
- **THEN** the server serves the Web UI at /dashboard instead of /ui

#### Scenario: Dev proxy configuration
- **WHEN** the config contains ui.dev_proxy: "http://localhost:5173"
- **THEN** the server proxies UI requests to the specified Vite dev server

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
