## MODIFIED Requirements

### Requirement: WebSocket connection management

The system SHALL establish and maintain a WebSocket connection for real-time updates through the canonical endpoint `/ws/events`.

#### Scenario: Establish WebSocket connection
- **WHEN** the user opens the Web UI
- **THEN** the frontend establishes a WebSocket connection to `ws://{host}/ws/events`

#### Scenario: Endpoint independent from UI base path
- **WHEN** the Web UI is mounted under a custom base path such as `/dashboard`
- **THEN** the frontend still connects to `/ws/events` on the same host

#### Scenario: Connection indicator
- **WHEN** the WebSocket connection state changes
- **THEN** the UI displays a connection status indicator: connected (green dot), disconnected (red dot), reconnecting (yellow dot)
