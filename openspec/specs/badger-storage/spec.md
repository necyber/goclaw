# badger-storage Specification

## Purpose
TBD - created by archiving change fix-week6-persistent-storage-gaps. Update Purpose after archive.
## Requirements
### Requirement: Badger status indexes SHALL be maintained on workflow mutation
Badger storage MUST remove stale status index entries when workflow status changes or workflow data is deleted.

#### Scenario: Status transition updates index
- **WHEN** a workflow status changes from `pending` to `running`
- **THEN** the old status index entry for `pending` MUST be removed and the `running` index entry MUST be present

#### Scenario: Workflow deletion cleans indexes
- **WHEN** a workflow is deleted
- **THEN** all status index entries for that workflow MUST be removed

### Requirement: Status-filtered listing SHALL return only current matches
Badger status-filtered listing MUST validate each candidate against the current persisted workflow status and MUST deduplicate by workflow ID.

#### Scenario: Stale index key exists
- **WHEN** a stale status index key remains due to previous state
- **THEN** `ListWorkflows` with status filters MUST NOT return the workflow unless its current status matches the filter

#### Scenario: Multi-index duplication
- **WHEN** a workflow is reachable through multiple index entries
- **THEN** `ListWorkflows` MUST return the workflow at most once

