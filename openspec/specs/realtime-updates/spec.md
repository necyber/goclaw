## ADDED Requirements

### Requirement: WebSocket connection management

The system SHALL establish and maintain a WebSocket connection for real-time updates.

#### Scenario: Establish WebSocket connection
- **WHEN** the user opens the Web UI
- **THEN** the frontend establishes a WebSocket connection to ws://{host}/ws/events

#### Scenario: Connection indicator
- **WHEN** the WebSocket connection state changes
- **THEN** the UI displays a connection status indicator: connected (green dot), disconnected (red dot), reconnecting (yellow dot)

### Requirement: Automatic reconnection

The system SHALL automatically reconnect WebSocket connections with exponential backoff.

#### Scenario: Reconnect on disconnect
- **WHEN** the WebSocket connection is lost
- **THEN** the system attempts to reconnect with exponential backoff starting at 1s, doubling up to 30s max

#### Scenario: Reset backoff on success
- **WHEN** a reconnection attempt succeeds
- **THEN** the system resets the backoff timer to the initial 1s delay

#### Scenario: Max reconnection attempts
- **WHEN** reconnection fails 10 consecutive times
- **THEN** the system stops reconnecting and displays a "Connection lost" banner with a manual reconnect button

### Requirement: Heartbeat keepalive

The system SHALL send periodic heartbeat messages to keep the WebSocket connection alive.

#### Scenario: Send heartbeat
- **WHEN** the WebSocket connection is established
- **THEN** the client sends a ping message every 30 seconds

#### Scenario: Detect stale connection
- **WHEN** no pong response is received within 10 seconds of a ping
- **THEN** the system closes the connection and initiates reconnection

### Requirement: Workflow state change events

The system SHALL push workflow state change events via WebSocket.

#### Scenario: Receive workflow state change
- **WHEN** a workflow transitions to a new state (e.g., running → completed)
- **THEN** the WebSocket sends a message with type "workflow.state_changed", workflow ID, old state, and new state

#### Scenario: Update UI on state change
- **WHEN** a workflow state change event is received
- **THEN** the UI updates the workflow status in all visible views (list, detail, DAG) without a full page refresh

### Requirement: Task state change events

The system SHALL push task state change events via WebSocket.

#### Scenario: Receive task state change
- **WHEN** a task transitions to a new state
- **THEN** the WebSocket sends a message with type "task.state_changed", workflow ID, task ID, old state, new state, and result (if completed)

#### Scenario: Update DAG on task change
- **WHEN** a task state change event is received while viewing the workflow's DAG
- **THEN** the DAG node updates its visual state without re-rendering the entire graph

### Requirement: WebSocket server endpoint

The system SHALL provide a WebSocket server endpoint that broadcasts Saga and workflow events.

#### Scenario: WebSocket upgrade
- **WHEN** a client sends a WebSocket upgrade request to /ws/events
- **THEN** the server upgrades the HTTP connection to WebSocket protocol

#### Scenario: Subscribe to workflow events
- **WHEN** a client sends a subscribe message with a workflow ID
- **THEN** the server streams only events for that specific workflow

#### Scenario: Broadcast global events
- **WHEN** a client connects without subscribing to a specific workflow
- **THEN** the server broadcasts all workflow and task state change events

#### Scenario: Connection limit
- **WHEN** the number of active WebSocket connections reaches the configured limit (default 100)
- **THEN** the server rejects new connections with HTTP 503

### Requirement: Event message format

The system SHALL use a consistent JSON message format for all WebSocket events.

#### Scenario: Event message structure
- **WHEN** a WebSocket event is sent
- **THEN** the message contains fields: type (string), timestamp (ISO 8601), payload (object)

#### Scenario: Workflow event payload
- **WHEN** a workflow state change event is sent
- **THEN** the payload contains: workflow_id, name, old_state, new_state, updated_at

#### Scenario: Task event payload
- **WHEN** a task state change event is sent
- **THEN** the payload contains: workflow_id, task_id, task_name, old_state, new_state, error (optional), result (optional)
