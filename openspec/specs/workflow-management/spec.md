# workflow-management Specification

## Purpose
Migrated from legacy OpenSpec format while preserving existing requirement and scenario content.

## Requirements

### Requirement: Workflow list page

The system SHALL display a paginated list of workflows with filtering and search capabilities.

#### Scenario: Display workflow list
- **WHEN** the user navigates to the Workflows page
- **THEN** the system displays a table of workflows showing ID, name, status, created time, and task count

#### Scenario: Paginate workflow list
- **WHEN** the workflow list exceeds the page size (default 20)
- **THEN** the system displays pagination controls to navigate between pages

#### Scenario: Filter by status
- **WHEN** the user selects a status filter (e.g., running, completed, failed)
- **THEN** the system displays only workflows matching the selected status

#### Scenario: Search by name
- **WHEN** the user types in the search input
- **THEN** the system filters the workflow list by name (client-side for current page, server-side for full search)

### Requirement: Workflow status indicators

The system SHALL display color-coded status badges for workflow and task states.

#### Scenario: Display workflow status badge
- **WHEN** a workflow is displayed in the list or detail view
- **THEN** the system shows a color-coded badge: pending (gray), running (blue), completed (green), failed (red), cancelled (yellow)

#### Scenario: Display task status badge
- **WHEN** a task is displayed in the workflow detail view
- **THEN** the system shows a color-coded badge matching the task state

### Requirement: Workflow detail page

The system SHALL display comprehensive workflow details including status, metadata, and task list.

#### Scenario: Display workflow details
- **WHEN** the user clicks a workflow in the list
- **THEN** the system navigates to the detail page showing: workflow ID, name, status, created/started/completed timestamps, metadata, and task list

#### Scenario: Display task list in workflow
- **WHEN** the workflow detail page renders
- **THEN** the system displays all tasks with their ID, name, status, duration, and error (if any)

#### Scenario: View task result
- **WHEN** the user clicks a completed task in the workflow detail
- **THEN** the system displays the task result data in a formatted JSON viewer

### Requirement: Submit workflow

The system SHALL allow users to submit new workflows via a form or JSON editor.

#### Scenario: Submit workflow via JSON
- **WHEN** the user enters a valid workflow JSON definition and clicks Submit
- **THEN** the system sends POST /api/v1/workflows and navigates to the new workflow's detail page

#### Scenario: Validate workflow input
- **WHEN** the user enters invalid JSON in the workflow submission form
- **THEN** the system displays a validation error before submission

#### Scenario: Submit workflow error
- **WHEN** the workflow submission API returns an error
- **THEN** the system displays the error message without navigating away

### Requirement: Cancel workflow

The system SHALL allow users to cancel running workflows from the detail page.

#### Scenario: Cancel running workflow
- **WHEN** the user clicks the Cancel button on a running workflow's detail page
- **THEN** the system sends POST /api/v1/workflows/{id}/cancel and updates the status display

#### Scenario: Cancel button visibility
- **WHEN** a workflow is in a terminal state (completed, failed, cancelled)
- **THEN** the Cancel button is not displayed

#### Scenario: Cancel confirmation
- **WHEN** the user clicks the Cancel button
- **THEN** the system displays a confirmation dialog before sending the cancel request

### Requirement: Workflow auto-refresh

The system SHALL automatically refresh workflow data for non-terminal workflows.

#### Scenario: Auto-refresh running workflow
- **WHEN** the user is viewing a running workflow's detail page
- **THEN** the system refreshes the workflow data every 2 seconds (or via WebSocket if connected)

#### Scenario: Stop auto-refresh for terminal workflow
- **WHEN** the workflow reaches a terminal state
- **THEN** the system stops auto-refreshing the data

