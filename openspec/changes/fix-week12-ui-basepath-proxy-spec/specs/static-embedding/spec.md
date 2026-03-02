## MODIFIED Requirements

### Requirement: Static asset embedding

The system SHALL embed the built frontend assets into the Go binary using go:embed.

#### Scenario: Embed frontend build output
- **WHEN** the Go binary is built with the `embed_ui` build tag
- **THEN** the binary contains all files from the `web/dist/` directory

#### Scenario: Build without UI
- **WHEN** the Go binary is built without the `embed_ui` build tag and UI routes are enabled
- **THEN** the configured UI base path returns a message indicating UI is not included

### Requirement: Static file serving

The system SHALL serve embedded static files at the configured UI base path prefix (default `/ui`).

#### Scenario: Serve index.html
- **WHEN** a GET request is made to `<ui.base_path>` or `<ui.base_path>/`
- **THEN** the server responds with the embedded `index.html` file and `Content-Type: text/html`

#### Scenario: Serve static assets
- **WHEN** a GET request is made to `<ui.base_path>/assets/main.js`
- **THEN** the server responds with the embedded JavaScript file and correct content type

#### Scenario: SPA fallback routing
- **WHEN** a GET request is made to `<ui.base_path>/workflows/abc-123` (a client-side route)
- **THEN** the server responds with `index.html` to allow the SPA router to handle the route

### Requirement: Development mode proxy

The system SHALL support proxying to a Vite dev server during development.

#### Scenario: Dev mode proxy
- **WHEN** the server is started with UI dev mode enabled (`ui.dev_proxy: "http://localhost:5173"`)
- **THEN** all requests under `<ui.base_path>` are proxied to the Vite dev server instead of serving embedded files

#### Scenario: Dev mode proxy request rewrite
- **WHEN** `<ui.base_path>` is `/dashboard` and a request is made to `/dashboard/metrics`
- **THEN** the upstream dev server receives path `/metrics`

#### Scenario: Production mode
- **WHEN** the server is started without UI dev mode
- **THEN** the server serves embedded static files normally
