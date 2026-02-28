## ADDED Requirements

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
