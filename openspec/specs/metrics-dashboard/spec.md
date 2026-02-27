# metrics-dashboard Specification

## Purpose
Migrated from legacy OpenSpec format while preserving existing requirement and scenario content.

## Requirements

### Requirement: Overview dashboard

The system SHALL display a dashboard overview page with key system metrics.

#### Scenario: Display overview cards
- **WHEN** the user navigates to the Dashboard page
- **THEN** the system displays summary cards: active workflows, completed workflows (last 24h), failed workflows (last 24h), average execution time

#### Scenario: Refresh overview data
- **WHEN** the dashboard page is visible
- **THEN** the system refreshes the overview data every 10 seconds

### Requirement: Workflow throughput chart

The system SHALL display a time-series chart of workflow submissions and completions.

#### Scenario: Display throughput chart
- **WHEN** the Metrics page renders
- **THEN** the system displays a line chart showing workflow submissions and completions over the last hour with 1-minute granularity

#### Scenario: Hover tooltip
- **WHEN** the user hovers over a data point on the chart
- **THEN** the system displays a tooltip with the exact timestamp and value

### Requirement: Task execution time distribution

The system SHALL display a histogram of task execution durations.

#### Scenario: Display duration histogram
- **WHEN** the Metrics page renders
- **THEN** the system displays a bar chart showing the distribution of task execution times in configurable buckets

#### Scenario: Filter by time range
- **WHEN** the user selects a time range (last 15m, 1h, 6h, 24h)
- **THEN** the chart updates to show data for the selected period

### Requirement: Lane queue depth chart

The system SHALL display real-time queue depth for each Lane.

#### Scenario: Display lane queue depths
- **WHEN** the Metrics page renders
- **THEN** the system displays a stacked area chart showing queue depth per lane over time

#### Scenario: Lane selection
- **WHEN** the user clicks a lane name in the chart legend
- **THEN** the chart toggles visibility of that lane's data series

### Requirement: System resource metrics

The system SHALL display system resource usage metrics.

#### Scenario: Display resource metrics
- **WHEN** the Metrics page renders
- **THEN** the system displays gauges for: memory usage, goroutine count, and CPU usage (if available from AdminService)

#### Scenario: Memory usage gauge
- **WHEN** memory usage data is available
- **THEN** the system displays a gauge showing current heap usage with a color indicator (green < 70%, yellow < 90%, red >= 90%)

### Requirement: Error rate chart

The system SHALL display workflow and task error rates over time.

#### Scenario: Display error rate
- **WHEN** the Metrics page renders
- **THEN** the system displays a line chart showing the percentage of failed workflows and tasks over the last hour

#### Scenario: Error spike indicator
- **WHEN** the error rate exceeds 10% in any 1-minute window
- **THEN** the chart highlights that period with a red background band

### Requirement: Metrics data source

The system SHALL fetch metrics data from the Prometheus endpoint and AdminService gRPC.

#### Scenario: Fetch from Prometheus
- **WHEN** the metrics dashboard loads
- **THEN** the system queries the /metrics endpoint and parses Prometheus text format for chart data

#### Scenario: Fetch from AdminService
- **WHEN** the metrics dashboard loads
- **THEN** the system calls the HTTP API proxy for AdminService data (engine status, lane stats)

#### Scenario: Metrics unavailable
- **WHEN** the metrics endpoint is unreachable
- **THEN** the system displays a "Metrics unavailable" message with the last known data timestamp

