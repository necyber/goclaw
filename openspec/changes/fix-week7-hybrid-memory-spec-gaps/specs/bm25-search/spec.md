## ADDED Requirements

### Requirement: Persistent bootstrap of BM25 index
BM25 search MUST rebuild its in-memory index from persisted memory entries at startup.

#### Scenario: Rebuild BM25 corpus on startup
- **WHEN** startup runs with persisted entries containing textual content
- **THEN** BM25 indexes all eligible entries before serving search requests

#### Scenario: Preserve corpus statistics on rebuild
- **WHEN** bootstrap completes
- **THEN** BM25 corpus statistics (document count and average length inputs) reflect rebuilt entries

