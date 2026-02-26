## ADDED Requirements

### Requirement: Application shell layout

The system SHALL provide a responsive application shell with a top navigation bar, collapsible sidebar, and main content area.

#### Scenario: Render application shell
- **WHEN** the user navigates to the Web UI root path
- **THEN** the system renders a top navigation bar with the GoClaw logo, a collapsible sidebar with navigation links, and a main content area

#### Scenario: Collapse sidebar
- **WHEN** the user clicks the sidebar toggle button
- **THEN** the sidebar collapses to icon-only mode and the main content area expands

#### Scenario: Expand sidebar
- **WHEN** the user clicks the sidebar toggle button while collapsed
- **THEN** the sidebar expands to full width showing icons and labels

### Requirement: Page routing

The system SHALL provide client-side routing for all dashboard pages.

#### Scenario: Navigate between pages
- **WHEN** the user clicks a sidebar navigation link
- **THEN** the system renders the corresponding page without a full page reload

#### Scenario: Direct URL access
- **WHEN** the user navigates directly to a deep URL (e.g., /ui/workflows/abc-123)
- **THEN** the system renders the correct page via SPA fallback routing

#### Scenario: 404 page
- **WHEN** the user navigates to an unknown route under /ui
- **THEN** the system renders a "Page Not Found" view with a link back to the dashboard

### Requirement: Navigation structure

The system SHALL provide navigation links to all major sections: Dashboard, Workflows, Metrics, Admin.

#### Scenario: Display navigation items
- **WHEN** the application shell renders
- **THEN** the sidebar displays navigation items: Dashboard (overview), Workflows (list/manage), Metrics (charts), Admin (controls)

#### Scenario: Highlight active page
- **WHEN** the user is on a specific page
- **THEN** the corresponding sidebar navigation item is visually highlighted

### Requirement: Theme switching

The system SHALL support light and dark themes with user preference persistence.

#### Scenario: Toggle to dark theme
- **WHEN** the user clicks the theme toggle in the navigation bar
- **THEN** the UI switches to dark theme and the preference is saved to localStorage

#### Scenario: Toggle to light theme
- **WHEN** the user clicks the theme toggle while in dark theme
- **THEN** the UI switches to light theme and the preference is saved to localStorage

#### Scenario: Restore theme preference
- **WHEN** the user loads the Web UI
- **THEN** the system applies the previously saved theme preference, defaulting to system preference if none saved

### Requirement: Loading and error states

The system SHALL display appropriate loading indicators and error messages for all data-fetching operations.

#### Scenario: Display loading state
- **WHEN** a page is fetching data from the API
- **THEN** the system displays a loading spinner or skeleton placeholder in the content area

#### Scenario: Display error state
- **WHEN** an API request fails
- **THEN** the system displays an error message with a retry button

#### Scenario: Display empty state
- **WHEN** a list page has no data to display
- **THEN** the system displays an empty state illustration with a helpful message

### Requirement: Responsive layout

The system SHALL adapt the layout for different screen widths down to 1024px minimum.

#### Scenario: Wide screen layout
- **WHEN** the viewport width is 1280px or greater
- **THEN** the sidebar is expanded and the content area uses the remaining width

#### Scenario: Medium screen layout
- **WHEN** the viewport width is between 1024px and 1279px
- **THEN** the sidebar auto-collapses to icon-only mode
