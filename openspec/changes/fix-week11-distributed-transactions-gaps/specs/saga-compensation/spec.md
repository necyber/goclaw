## MODIFIED Requirements

### Requirement: Reverse-order compensation execution

The system SHALL execute compensation operations in reverse topological order of completed steps and SHALL keep compensation bookkeeping race-free under parallel execution.

#### Scenario: Linear compensation order
- **WHEN** steps A -> B -> C were executed and C fails
- **THEN** the system compensates B then A (reverse order)

#### Scenario: Parallel step compensation
- **WHEN** steps B and C (both depending on A) were executed in parallel and a later step fails
- **THEN** the system compensates B and C in parallel, then compensates A

#### Scenario: Parallel compensation bookkeeping safety
- **WHEN** multiple compensation functions in one reverse layer complete concurrently
- **THEN** the system records all compensated step IDs exactly once without data races or lost updates

#### Scenario: Skip steps without compensation
- **WHEN** a completed step has no compensation function defined
- **THEN** the system skips that step during compensation
