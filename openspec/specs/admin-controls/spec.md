# admin-controls Specification

## Purpose
Migrated from legacy OpenSpec format while preserving existing requirement and scenario content.

## Requirements

### Requirement: Engine status display

The system SHALL display the current engine status and health information.

#### Scenario: Display engine status
- **WHEN** the user navigates to the Admin page
- **THEN** the system displays: engine state (idle/running/stopped/error), uptime, version, active workflow count, goroutine count, memory usage

#### Scenario: Engine state indicator
- **WHEN** the engine state changes
- **THEN** the Admin page updates the state indicator: running (green), idle (yellow), stopped (gray), error (red)

### Requirement: Pause and resume workflows

The system SHALL allow administrators to pause and resume workflow processing.

#### Scenario: Pause workflow processing
- **WHEN** the admin clicks the Pause button
- **THEN** the system calls the AdminService PauseWorkflows API and the engine stops accepting new workflow executions

#### Scenario: Resume workflow processing
- **WHEN** the admin clicks the Resume button while paused
- **THEN** the system calls the AdminService ResumeWorkflows API and the engine resumes accepting workflows

#### Scenario: Pause confirmation
- **WHEN** the admin clicks the Pause button
- **THEN** the system displays a confirmation dialog warning that running workflows will continue but no new ones will start

### Requirement: Lane statistics display

The system SHALL display statistics for each configured Lane.

#### Scenario: Display lane list
- **WHEN** the Admin page renders
- **THEN** the system displays a table of all lanes with: name, queue depth, worker count, throughput/sec, error rate

#### Scenario: Lane detail view
- **WHEN** the admin clicks a lane name
- **THEN** the system displays detailed lane statistics including queue depth history chart and recent task executions

#### Scenario: Refresh lane stats
- **WHEN** the Admin page is visible
- **THEN** the system refreshes lane statistics every 5 seconds

### Requirement: Purge workflows

The system SHALL allow administrators to purge completed workflows.

#### Scenario: Purge completed workflows
- **WHEN** the admin clicks Purge and confirms the action
- **THEN** the system calls the AdminService PurgeWorkflows API to remove completed/failed workflows older than the specified retention period

#### Scenario: Purge confirmation
- **WHEN** the admin clicks the Purge button
- **THEN** the system displays a confirmation dialog showing the number of workflows that will be removed

### Requirement: Debug information export

The system SHALL allow administrators to export debug information.

#### Scenario: Export debug info
- **WHEN** the admin clicks "Export Debug Info"
- **THEN** the system calls the AdminService GetDebugInfo API and downloads a JSON file containing goroutine dump, heap profile summary, and system info

#### Scenario: Export metrics
- **WHEN** the admin clicks "Export Metrics"
- **THEN** the system downloads the current Prometheus metrics in text format

### Requirement: Cluster information display

The system SHALL display cluster node information when running in distributed mode.

#### Scenario: Display cluster nodes
- **WHEN** the system is running in cluster mode and the admin views the Admin page
- **THEN** the system displays a list of cluster nodes with: node ID, address, status (healthy/unhealthy), last heartbeat

#### Scenario: Single node mode
- **WHEN** the system is running in standalone mode
- **THEN** the Admin page displays "Standalone mode" instead of cluster information

