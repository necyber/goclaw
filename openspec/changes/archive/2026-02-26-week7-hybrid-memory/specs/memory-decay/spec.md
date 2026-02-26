## ADDED Requirements

### Requirement: FSRS-6 strength calculation

The system SHALL calculate memory strength using FSRS-6 algorithm: S' = S * e^(-t/τ)

#### Scenario: Calculate strength after time elapsed
- **WHEN** a memory entry with strength 1.0 and stability 24h is reviewed after 24h
- **THEN** the system calculates new strength as approximately 0.368 (1/e)

#### Scenario: Calculate strength with high stability
- **WHEN** a memory entry has high stability (long half-life)
- **THEN** the strength decays slowly over time

### Requirement: Stability parameter management

The system SHALL maintain and update stability parameter for each memory entry.

#### Scenario: Initialize stability for new entry
- **WHEN** a new memory entry is created
- **THEN** the system assigns default stability value (e.g., 24 hours)

#### Scenario: Update stability on successful retrieval
- **WHEN** a memory entry is successfully retrieved and used
- **THEN** the system increases the stability parameter

### Requirement: Automatic strength decay

The system SHALL run a background goroutine to periodically update memory strengths.

#### Scenario: Periodic decay update
- **WHEN** the decay interval (e.g., 1 hour) elapses
- **THEN** the system updates strengths for all memory entries

#### Scenario: Decay on system startup
- **WHEN** the system starts up
- **THEN** the system calculates and updates strengths based on time since last update

### Requirement: Forgetting threshold

The system SHALL support configurable forgetting threshold for automatic memory removal.

#### Scenario: Forget below threshold
- **WHEN** a memory entry's strength falls below threshold (e.g., 0.1)
- **THEN** the system marks the entry for deletion

#### Scenario: Retain above threshold
- **WHEN** a memory entry's strength is above threshold
- **THEN** the system retains the entry

### Requirement: Manual strength boost

The system SHALL allow manual strength boosting when a memory is explicitly retrieved.

#### Scenario: Boost on retrieval
- **WHEN** a memory entry is retrieved by a query
- **THEN** the system increases the strength to 1.0 and updates LastReview timestamp

#### Scenario: Boost on explicit memorization
- **WHEN** a memory entry is explicitly re-memorized
- **THEN** the system resets strength to 1.0 and increases stability

### Requirement: Batch decay processing

The system SHALL process memory decay in batches for efficiency.

#### Scenario: Batch update strengths
- **WHEN** processing decay for 10K memory entries
- **THEN** the system updates strengths in batches (e.g., 1K per batch)

#### Scenario: Batch delete forgotten entries
- **WHEN** multiple entries fall below forgetting threshold
- **THEN** the system deletes them in a single batch operation

### Requirement: Session-based decay isolation

The system SHALL apply decay independently per session.

#### Scenario: Decay within session
- **WHEN** processing decay for session "A"
- **THEN** the system updates only entries belonging to session "A"

#### Scenario: Different decay rates per session
- **WHEN** sessions have different decay configurations
- **THEN** the system applies session-specific decay parameters

### Requirement: Decay interval configuration

The system SHALL support configurable decay update interval.

#### Scenario: Hourly decay updates
- **WHEN** decay interval is configured to 1 hour
- **THEN** the system updates strengths every hour

#### Scenario: Daily decay updates
- **WHEN** decay interval is configured to 24 hours
- **THEN** the system updates strengths once per day

### Requirement: Graceful decay shutdown

The system SHALL gracefully stop decay processing on system shutdown.

#### Scenario: Stop decay on shutdown
- **WHEN** the system receives shutdown signal
- **THEN** the decay goroutine completes current batch and exits

#### Scenario: Save decay state on shutdown
- **WHEN** the system shuts down during decay processing
- **THEN** the system saves current decay state for resume on restart

### Requirement: Decay metrics

The system SHALL expose metrics for memory decay operations.

#### Scenario: Track forgotten entries count
- **WHEN** entries are forgotten due to low strength
- **THEN** the system increments forgotten_entries_total metric

#### Scenario: Track decay processing time
- **WHEN** decay processing completes
- **THEN** the system records decay_processing_duration_seconds metric
