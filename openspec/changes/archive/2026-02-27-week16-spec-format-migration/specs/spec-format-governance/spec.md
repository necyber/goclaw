## ADDED Requirements

### Requirement: Canonical top-level section structure
The repository SHALL keep every main spec document in canonical OpenSpec structure.

#### Scenario: Main spec structure is validated
- **WHEN** a main spec at `openspec/specs/<capability>/spec.md` is evaluated
- **THEN** it MUST contain `## Purpose` and `## Requirements` sections

### Requirement: Preservation-first migration for structured legacy specs
Legacy specs with existing requirement/scenario blocks SHALL be migrated by structural wrapping instead of semantic rewrite.

#### Scenario: Legacy requirement blocks already exist
- **WHEN** a legacy spec contains `### Requirement:` and `#### Scenario:` blocks
- **THEN** migration MUST preserve those blocks and only normalize top-level sections

### Requirement: Baseline normalization for narrative-only legacy specs
Legacy specs without requirement/scenario blocks SHALL still become validator-compliant while retaining historical text.

#### Scenario: Legacy spec has no requirement/scenario block
- **WHEN** a legacy spec has no `### Requirement:` section
- **THEN** migration MUST add at least one normative requirement and one scenario, and retain prior narrative under `## Notes`

### Requirement: Repository-level validation gate
Spec migration SHALL be accepted only after repository-level spec validation is fully green.

#### Scenario: Validation result for main specs
- **WHEN** `openspec validate --specs` is executed after migration
- **THEN** total failed main specs MUST be `0`
