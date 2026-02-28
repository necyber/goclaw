## ADDED Requirements

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
