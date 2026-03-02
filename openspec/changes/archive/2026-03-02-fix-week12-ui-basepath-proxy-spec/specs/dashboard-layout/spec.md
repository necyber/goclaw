## MODIFIED Requirements

### Requirement: Page routing

The system SHALL provide client-side routing for all dashboard pages under the configured UI base path.

#### Scenario: Navigate between pages
- **WHEN** the user clicks a sidebar navigation link
- **THEN** the system renders the corresponding page without a full page reload

#### Scenario: Direct URL access
- **WHEN** the user navigates directly to a deep URL under the configured UI base path (e.g., `/dashboard/workflows/abc-123`)
- **THEN** the system renders the correct page via SPA fallback routing

#### Scenario: 404 page
- **WHEN** the user navigates to an unknown route under the configured UI base path
- **THEN** the system renders a "Page Not Found" view with a link back to the dashboard

#### Scenario: Router base path binding
- **WHEN** the Web UI boots with server base path `/dashboard`
- **THEN** the client router uses `/dashboard` as basename instead of hardcoded `/ui`
