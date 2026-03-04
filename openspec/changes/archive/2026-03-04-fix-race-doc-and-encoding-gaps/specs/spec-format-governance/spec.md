## ADDED Requirements

### Requirement: Canonical project docs must reflect implemented project phase
Repository-level canonical project documents MUST describe the current implementation maturity and phase status consistent with shipped code and archived change history.

#### Scenario: Project maturity statement is reviewed
- **WHEN** maintainers update canonical project guides (for example AGENTS or roadmap/status documents)
- **THEN** maturity and phase statements MUST match the current implemented baseline and MUST NOT claim design-only state once implementation exists.

#### Scenario: Status alignment after major scope completion
- **WHEN** a phase baseline is completed and archived in OpenSpec history
- **THEN** canonical project documents MUST update their phase/status statements in the same maintenance window.

### Requirement: Canonical project docs must be UTF-8 readable
Canonical project markdown documents MUST be stored and maintained in UTF-8 readable form to avoid mojibake across supported contributor environments.

#### Scenario: Canonical markdown readability check
- **WHEN** canonical project markdown files are opened using UTF-8 decoding
- **THEN** headings and body text MUST render as intended human-readable content without mojibake artifacts.

#### Scenario: Cross-platform contributor path
- **WHEN** contributors on common Windows/macOS/Linux environments view canonical docs in default Git/editor workflows
- **THEN** the repository baseline MUST preserve readable UTF-8 content without requiring file-by-file manual recoding.

