## MODIFIED Requirements

### Requirement: Memory statistics

The system SHALL provide statistics about memory usage per session.

#### Scenario: Get memory stats
- **WHEN** GetStats is called with sessionID
- **THEN** the system returns total entries, average strength, and storage size in bytes for that session

#### Scenario: Get global stats
- **WHEN** GetGlobalStats is called
- **THEN** the system returns statistics across all sessions

### Requirement: Concurrent operation support

The system SHALL support concurrent operations and lifecycle transitions from multiple goroutines.

#### Scenario: Concurrent memorize
- **WHEN** multiple goroutines call Memorize simultaneously
- **THEN** all operations complete successfully without data races

#### Scenario: Concurrent retrieve
- **WHEN** multiple goroutines call Retrieve simultaneously
- **THEN** all operations return correct results without blocking each other

#### Scenario: Repeated lifecycle transitions
- **WHEN** callers perform repeated `Start`/`Stop` cycles on the same MemoryHub instance
- **THEN** lifecycle transitions are safe and do not panic due to decay loop coordination
