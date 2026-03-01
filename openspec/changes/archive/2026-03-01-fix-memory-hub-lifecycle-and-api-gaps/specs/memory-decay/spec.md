## MODIFIED Requirements

### Requirement: FSRS-6 strength calculation

The system SHALL calculate memory strength using FSRS-6 algorithm: `S' = S * e^(-t/τ)` where `t` is elapsed hours since the last effective decay/review timestamp.

#### Scenario: Calculate strength after time elapsed
- **WHEN** a memory entry with strength 1.0 and stability 24h is reviewed after 24h
- **THEN** the system calculates new strength as approximately 0.368 (1/e)

#### Scenario: Calculate strength with high stability
- **WHEN** a memory entry has high stability (long half-life)
- **THEN** the strength decays slowly over time

#### Scenario: Periodic decay uses non-overlapping elapsed windows
- **WHEN** periodic decay runs in consecutive intervals for the same entry
- **THEN** each run uses elapsed time since the last persisted decay update rather than reapplying previously elapsed time

### Requirement: Graceful decay shutdown

The system SHALL gracefully stop and restart decay processing across repeated lifecycle transitions.

#### Scenario: Stop decay on shutdown
- **WHEN** the system receives shutdown signal
- **THEN** the decay goroutine completes current batch and exits

#### Scenario: Save decay state on shutdown
- **WHEN** the system shuts down during decay processing
- **THEN** the system saves current decay state for resume on restart

#### Scenario: Restart decay after prior stop
- **WHEN** MemoryHub is started, stopped, and started again on the same process instance
- **THEN** decay loop startup and shutdown complete without panic, including repeated stop calls
