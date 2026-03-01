## ADDED Requirements

### Requirement: Decay loop full-corpus coverage
The decay loop MUST process all persisted memory entries across all sessions on each interval.

#### Scenario: Global decay sweep
- **WHEN** a decay interval elapses
- **THEN** the decay process scans and evaluates entries from every session

#### Scenario: Empty corpus no-op
- **WHEN** no persisted memory entries exist
- **THEN** the decay process completes without updates or deletions

### Requirement: Session-safe forgetting in decay
Decay-triggered forgetting MUST delete only the entries selected for each session.

#### Scenario: Per-session forget set
- **WHEN** decay computes forgotten IDs for session `A`
- **THEN** only entries in that computed set are deleted for session `A`

