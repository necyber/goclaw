## Purpose

Define and enforce canonical OpenSpec main spec structure across the repository.

## Requirements

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

### Requirement: Archived change semantic corrections require a new change record
The repository SHALL apply semantic corrections to archived OpenSpec changes through a new change directory, not by undocumented direct edits.

For this requirement, semantic correction means changes that alter requirement meaning, scope boundary, acceptance criteria, or error-model terminology.

#### Scenario: Semantic inconsistency found in archived artifacts
- **WHEN** a reviewer identifies contradictory terminology, scope, or acceptance semantics in an archived change
- **THEN** the team MUST create a new change that documents rationale and planned corrections before applying those edits

### Requirement: Archived change non-semantic errata must be traceable
The repository SHALL allow direct archived edits only for non-semantic errata (for example encoding repair, typo, or broken link), and each such edit MUST include an explicit errata note in the affected artifact.

The errata note MUST use this format:
`[Errata YYYY-MM-DD] Type=<encoding|typo|link> Reason=<text> Scope=<text>`.

#### Scenario: Encoding or typo fix in archived file
- **WHEN** a maintainer fixes a non-semantic issue in an archived artifact
- **THEN** the maintainer MUST add an errata note with date and reason in the same artifact

#### Scenario: Errata format is validated
- **WHEN** a non-semantic errata entry is added to an archived artifact
- **THEN** the entry MUST include date, type, reason, and scope fields using the required format

### Requirement: Archived change artifacts must remain internally consistent
For an archived change, proposal, design, tasks, and delta specs SHALL not contain conflicting naming, scope, or acceptance criteria statements.

#### Scenario: Archive consistency review
- **WHEN** an archived change is reviewed for maintenance
- **THEN** reviewers MUST be able to map naming, scope, and acceptance language across artifacts without contradiction

### Requirement: Archived consistency repairs must provide requirement-traceable alignment
For semantic consistency repairs on archived changes, the correction change SHALL explicitly map each resolved inconsistency to the affected artifact(s) and canonical statement.

The mapping MUST identify at least:
- canonical terminology decisions
- canonical scope boundary decisions
- canonical acceptance/metrics semantics when applicable

#### Scenario: Repairing implicit requirement coverage in archived artifacts
- **WHEN** a reviewer finds that archived proposal/design/tasks do not explicitly reflect an existing archived spec requirement
- **THEN** the correction change MUST add explicit traceability language so the requirement can be mapped across artifacts without inference

#### Scenario: Repairing normative-vs-optional ambiguity in archived artifacts
- **WHEN** an archived artifact mixes normative requirements and optional extensions without clear distinction
- **THEN** the correction change MUST mark the normative baseline explicitly and identify optional behavior as non-blocking extension
