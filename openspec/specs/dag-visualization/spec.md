## ADDED Requirements

### Requirement: DAG graph rendering

The system SHALL render workflow task dependencies as a directed acyclic graph using automatic layout.

#### Scenario: Render DAG for workflow
- **WHEN** the user views a workflow detail page with the DAG tab selected
- **THEN** the system renders a directed graph where each node represents a task and edges represent dependencies

#### Scenario: Automatic layout
- **WHEN** the DAG is rendered
- **THEN** the system uses dagre layout algorithm to position nodes in a top-to-bottom hierarchical arrangement

#### Scenario: Empty DAG
- **WHEN** a workflow has no task dependencies (all tasks are independent)
- **THEN** the system renders all task nodes in a single horizontal row

### Requirement: Node status visualization

The system SHALL color-code DAG nodes based on task execution status.

#### Scenario: Pending task node
- **WHEN** a task is in pending state
- **THEN** the node is rendered with a gray background and dashed border

#### Scenario: Running task node
- **WHEN** a task is in running state
- **THEN** the node is rendered with a blue background and a pulsing animation

#### Scenario: Completed task node
- **WHEN** a task is in completed state
- **THEN** the node is rendered with a green background and a checkmark icon

#### Scenario: Failed task node
- **WHEN** a task is in failed state
- **THEN** the node is rendered with a red background and an error icon

#### Scenario: Cancelled task node
- **WHEN** a task is in cancelled state
- **THEN** the node is rendered with a yellow background and a stop icon

### Requirement: DAG interaction

The system SHALL support zoom, pan, and node selection interactions on the DAG view.

#### Scenario: Zoom in/out
- **WHEN** the user scrolls the mouse wheel over the DAG view
- **THEN** the graph zooms in or out centered on the cursor position

#### Scenario: Pan the graph
- **WHEN** the user clicks and drags on the DAG background
- **THEN** the graph pans in the drag direction

#### Scenario: Fit to view
- **WHEN** the user clicks the "Fit" button
- **THEN** the graph zooms and pans to fit all nodes within the visible area

#### Scenario: Select node
- **WHEN** the user clicks a task node in the DAG
- **THEN** the system highlights the node and displays task details in a side panel

### Requirement: DAG real-time updates

The system SHALL update DAG node states in real-time as tasks execute.

#### Scenario: Update node on task start
- **WHEN** a task transitions from pending to running
- **THEN** the corresponding DAG node updates its color and animation without re-rendering the entire graph

#### Scenario: Update node on task completion
- **WHEN** a task transitions to completed or failed
- **THEN** the corresponding DAG node updates its color and icon

### Requirement: Edge visualization

The system SHALL render edges between DAG nodes to indicate dependencies.

#### Scenario: Render dependency edges
- **WHEN** task B depends on task A
- **THEN** the system renders a directed edge from node A to node B with an arrowhead

#### Scenario: Completed edge styling
- **WHEN** both the source and target tasks of an edge are completed
- **THEN** the edge is rendered with a solid green line

#### Scenario: Pending edge styling
- **WHEN** the target task of an edge is pending
- **THEN** the edge is rendered with a dashed gray line

### Requirement: DAG minimap

The system SHALL display a minimap for large DAG graphs.

#### Scenario: Show minimap for large graphs
- **WHEN** the DAG contains more than 10 nodes
- **THEN** the system displays a minimap in the bottom-right corner showing the full graph overview

#### Scenario: Navigate via minimap
- **WHEN** the user clicks a position on the minimap
- **THEN** the main view pans to center on that position
