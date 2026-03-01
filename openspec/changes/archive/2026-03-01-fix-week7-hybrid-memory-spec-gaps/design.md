## Context

Week7 hybrid-memory is already implemented and covered by tests, but several runtime behaviors are unsafe or incomplete:
- `Forget(sessionID, ids)` can delete entries outside the caller's session.
- Background decay uses an empty-session scan path that does not enumerate all entries.
- Vector/BM25 indexes are in-memory only at startup and are not rebuilt from persisted entries.
- Retrieval mode and API semantics diverge from the intended contract.
- Delete APIs can report optimistic counts rather than actual deletes.

The change spans `pkg/memory` and HTTP handlers/routes, so it is cross-cutting and requires explicit design decisions before implementation.

## Goals / Non-Goals

**Goals:**
- Enforce session-safe deletion semantics across hub/storage/API.
- Ensure decay processing covers all persisted entries across all sessions.
- Make retrieval resilient after restart by rebuilding in-memory indexes from persisted entries.
- Normalize query mode handling and invalid-mode behavior.
- Return deletion/statistics API responses that reflect real persisted outcomes.
- Add targeted regression tests for each fixed behavior.

**Non-Goals:**
- Replacing brute-force vector search with HNSW in this patch.
- Introducing a new external vector DB dependency.
- Redesigning memory APIs beyond compatibility-safe fixes.
- Changing unrelated workflow/saga runtime behavior.

## Decisions

1. Add explicit session-scoped delete paths and make hub use them.
- Decision: introduce/delete via storage methods that require both `sessionID` and `entryID`, and have `Forget` enforce session ownership.
- Rationale: prevents cross-session deletion when entry IDs are known but session is mismatched.
- Alternative considered: keep ID-only delete and pre-check with `Get`.
  - Rejected: still relies on global scans and is easier to misuse in future code paths.

2. Introduce full-entry iteration API for decay and startup rebuild.
- Decision: add storage iteration that returns all entries (or all sessions) without requiring a specific session prefix.
- Rationale: decay and index bootstrap are global maintenance tasks.
- Alternative considered: maintain a separate session registry.
  - Rejected: extra consistency surface and migration overhead for this fix scope.

3. Rebuild vector/BM25 indexes from persisted memory entries at hub startup.
- Decision: during `MemoryHub.Start`, load persisted entries and index them before serving retrieve requests.
- Rationale: keeps persisted source-of-truth and avoids empty indexes after restart.
- Alternative considered: persist index snapshots and restore directly.
  - Rejected: useful future optimization, but not required to restore correctness now.

4. Standardize query mode contract and validation.
- Decision: accept canonical mode values (`hybrid`, `vector-only`, `bm25-only`) and reject unknown values with validation error.
- Rationale: explicit errors are safer than silent fallback; route behavior becomes predictable.
- Alternative considered: continue permissive fallback to hybrid.
  - Rejected: hides caller errors and creates hard-to-debug behavior drift.

5. Return actual deletion outcomes in hub/API.
- Decision: `Forget` and delete endpoints should report real deleted count and surface storage failures consistently.
- Rationale: operational correctness and reliable observability.
- Alternative considered: keep best-effort delete with optimistic response count.
  - Rejected: misleading API contract and weakens incident diagnosis.

6. Add global memory stats endpoint.
- Decision: support `/api/v1/memory/stats` for cross-session aggregate stats, while preserving per-session stats.
- Rationale: closes documented API gap with minimal interface expansion.
- Alternative considered: only update docs/specs.
  - Rejected: leaves user-visible contract incomplete.

## Risks / Trade-offs

- [Startup index rebuild increases boot latency on large datasets] → Mitigation: bounded batched scan with progress logs and future snapshot optimization.
- [New storage APIs may overlap existing behavior and confuse callers] → Mitigation: deprecate ID-only delete internally and migrate hub usage immediately.
- [Stricter mode validation can surface previously hidden client bugs] → Mitigation: keep backward-compatible aliases for one patch cycle where possible and document canonical values.
- [Real delete count computation may require extra reads] → Mitigation: perform counting within storage transaction/scan where feasible.

## Migration Plan

1. Add/adjust storage APIs for session-scoped delete and global iteration.
2. Update MemoryHub forget/decay/startup-bootstrap logic to use new APIs.
3. Update hybrid mode parsing and API handler validation/response semantics.
4. Add global stats route and handler coverage.
5. Add regression tests:
   - cross-session forget safety
   - decay global scan coverage
   - restart index rebuild
   - invalid mode handling
   - delete count correctness and global stats endpoint
6. Run full memory + handler + related engine tests; rollback by reverting patch if regression appears.

## Open Questions

- Should legacy aliases (`vector`, `bm25`) be accepted indefinitely or sunset in a follow-up deprecation cycle?
- Should index rebuild run synchronously before readiness, or asynchronously with temporary degraded retrieval?
- Do we want to expose per-session deletion skipped count (not-owned IDs) as part of delete response?
