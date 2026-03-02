## ADDED Requirements

### Requirement: Monitoring behavior conformance for Week10 metrics
The system MUST provide requirement-traceable conformance checks for Week10 monitoring metrics behavior across lane, task, workflow, HTTP, and Prometheus integration paths.

#### Scenario: Conformance checks cover identified monitoring gaps
- **WHEN** monitoring conformance verification is executed for this change
- **THEN** checks MUST include lane redirect wait-duration behavior, task_type labels, workflow pending/running active gauges, HTTP status/path normalization, and label-cardinality safeguards

#### Scenario: Conformance checks are automated
- **WHEN** CI or local verification runs for this change
- **THEN** conformance checks MUST be executable as automated tests instead of manual inspection only

### Requirement: Monitoring compatibility notes are explicit
The system MUST document observable metric-shape changes that can affect dashboards or alerts.

#### Scenario: Metric label semantics change
- **WHEN** a metric label semantic is changed by this fix
- **THEN** documentation MUST include migration guidance for existing PromQL queries
