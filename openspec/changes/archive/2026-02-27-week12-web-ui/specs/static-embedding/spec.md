## ADDED Requirements

### Requirement: Static asset embedding

The system SHALL embed the built frontend assets into the Go binary using go:embed.

#### Scenario: Embed frontend build output
- **WHEN** the Go binary is built with the embed_ui build tag
- **THEN** the binary contains all files from the web/dist/ directory

#### Scenario: Build without UI
- **WHEN** the Go binary is built without the embed_ui build tag
- **THEN** the binary does not contain frontend assets and the /ui route returns a message indicating UI is not included

### Requirement: Static file serving

The system SHALL serve embedded static files at the /ui path prefix.

#### Scenario: Serve index.html
- **WHEN** a GET request is made to /ui or /ui/
- **THEN** the server responds with the embedded index.html file and Content-Type text/html

#### Scenario: Serve static assets
- **WHEN** a GET request is made to /ui/assets/main.js
- **THEN** the server responds with the embedded JavaScript file and correct Content-Type

#### Scenario: SPA fallback routing
- **WHEN** a GET request is made to /ui/workflows/abc-123 (a client-side route)
- **THEN** the server responds with index.html to allow the SPA router to handle the route

### Requirement: Compression

The system SHALL serve static assets with Gzip compression.

#### Scenario: Gzip response
- **WHEN** a client sends Accept-Encoding: gzip header
- **THEN** the server responds with Content-Encoding: gzip for compressible file types (HTML, CSS, JS, JSON, SVG)

#### Scenario: No compression for small files
- **WHEN** a static file is smaller than 1KB
- **THEN** the server serves it without compression

### Requirement: Cache control

The system SHALL set appropriate cache headers for static assets.

#### Scenario: Cache hashed assets
- **WHEN** a static asset has a content hash in its filename (e.g., main.a1b2c3.js)
- **THEN** the server sets Cache-Control: public, max-age=31536000, immutable

#### Scenario: No cache for index.html
- **WHEN** index.html is served
- **THEN** the server sets Cache-Control: no-cache to ensure the latest version is always loaded

### Requirement: Development mode proxy

The system SHALL support proxying to a Vite dev server during development.

#### Scenario: Dev mode proxy
- **WHEN** the server is started with UI dev mode enabled (config ui.dev_proxy: "http://localhost:5173")
- **THEN** all /ui requests are proxied to the Vite dev server instead of serving embedded files

#### Scenario: Production mode
- **WHEN** the server is started without UI dev mode
- **THEN** the server serves embedded static files normally

### Requirement: Build integration

The system SHALL provide Makefile targets for building the frontend.

#### Scenario: Build frontend
- **WHEN** make build-ui is executed
- **THEN** the system runs npm install and npm run build in the web/ directory, outputting to web/dist/

#### Scenario: Build all with UI
- **WHEN** make build is executed
- **THEN** the system builds the frontend first, then builds the Go binary with the embed_ui tag

#### Scenario: Clean frontend build
- **WHEN** make clean is executed
- **THEN** the system removes the web/dist/ and web/node_modules/ directories
