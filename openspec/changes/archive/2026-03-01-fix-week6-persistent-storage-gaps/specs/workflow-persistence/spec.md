## ADDED Requirements

### Requirement: Recovery workflow normalization SHALL persist canonical workflow state
During startup recovery, workflows selected for recovery MUST be normalized to canonical pre-execution state before resubmission.

#### Scenario: Recover running workflow
- **WHEN** startup recovery loads a workflow with status `running`
- **THEN** storage MUST persist the workflow as `pending` with cleared runtime-only fields before resubmission

#### Scenario: Preserve completed and failed workflows
- **WHEN** startup recovery processes persisted workflows
- **THEN** workflows already in terminal states MUST remain unchanged and MUST NOT be normalized

### Requirement: Recovery scan SHALL be exhaustive
Recovery listing MUST process all recoverable workflows through paginated or equivalent iteration until no more matching records remain.

#### Scenario: Recoverable set exceeds one page
- **WHEN** there are more recoverable workflows than a single batch limit
- **THEN** recovery MUST continue scanning subsequent pages until all recoverable workflows are processed

#### Scenario: Empty next page ends scan
- **WHEN** a recovery page returns no additional workflows
- **THEN** recovery MUST stop iteration and report completion
