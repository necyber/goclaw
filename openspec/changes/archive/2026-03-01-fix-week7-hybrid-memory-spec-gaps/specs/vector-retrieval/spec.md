## ADDED Requirements

### Requirement: Persistent bootstrap of vector index
Vector retrieval MUST rebuild its in-memory index from persisted memory entries at startup.

#### Scenario: Rebuild vectors on startup
- **WHEN** startup runs with persisted entries containing vectors
- **THEN** the vector index is repopulated with entry IDs, vectors, and session mappings

#### Scenario: Skip entries without vectors
- **WHEN** startup scans an entry without vector payload
- **THEN** the entry is skipped for vector indexing without failing bootstrap

