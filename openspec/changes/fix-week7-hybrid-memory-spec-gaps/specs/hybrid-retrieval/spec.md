## ADDED Requirements

### Requirement: Query mode alias normalization
Hybrid retrieval MUST normalize supported aliases to canonical modes before execution.

#### Scenario: Normalize vector alias
- **WHEN** mode is provided as `vector`
- **THEN** the retriever treats it as `vector-only`

#### Scenario: Normalize BM25 alias
- **WHEN** mode is provided as `bm25`
- **THEN** the retriever treats it as `bm25-only`

### Requirement: Unknown mode error semantics
Hybrid retrieval MUST return a validation error for unknown mode values.

#### Scenario: Reject unknown mode
- **WHEN** mode is provided as an unsupported value
- **THEN** the retriever returns an error and does not fallback to `hybrid`

