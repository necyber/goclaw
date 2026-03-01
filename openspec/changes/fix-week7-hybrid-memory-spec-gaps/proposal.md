## Why

The current Week7 hybrid-memory implementation has several behavior gaps against its specs, including cross-session deletion risk, ineffective decay processing, non-resilient retrieval after restart, and API/query-mode contract drift. These issues affect correctness and data isolation, so they must be fixed before further memory features are built on top.

## What Changes

- Enforce strict session isolation for delete/forget flows so a session cannot delete entries owned by another session.
- Define full-scan decay behavior so background decay processes all persisted entries across sessions.
- Require startup index bootstrap/rebuild for vector and BM25 indexes from persisted memory entries.
- Align retrieval mode contract and error semantics with spec-defined modes and invalid mode handling.
- Align deletion/result reporting semantics so APIs return actual deletion outcomes instead of optimistic counts.
- Add missing global memory statistics API behavior.
- Add regression tests for cross-session safety, restart rebuild, decay coverage, query-mode validation, and API behavior.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `memory-hub-api`: tighten session-safe forget semantics, mode/error handling, and stats/query behavior.
- `memory-storage`: require full-entry scan support and session-safe delete behavior for lookup-by-id paths.
- `hybrid-retrieval`: standardize query mode names and invalid-mode behavior.
- `vector-retrieval`: require index bootstrap/rebuild behavior on startup from persisted entries.
- `bm25-search`: require index bootstrap/rebuild behavior on startup from persisted entries.
- `memory-decay`: require periodic decay to cover all entries/sessions, not empty-session no-op scans.
- `workflow-api-endpoints`: add/align global memory stats endpoint and memory query parameter contract.

## Impact

- Affected code:
  - `pkg/memory/hub.go`
  - `pkg/memory/storage.go`
  - `pkg/memory/hybrid.go`
  - `pkg/memory/vector.go`
  - `pkg/memory/bm25.go`
  - `pkg/api/handlers/memory.go`
  - `pkg/api/router.go`
  - related tests in `pkg/memory/*_test.go` and `pkg/api/handlers/memory_test.go`
- Behavioral impact:
  - safer session isolation and deletion semantics
  - reliable retrieval behavior after restart
  - decay loop performs real work on persisted datasets
  - API contract consistency for query modes and global stats
