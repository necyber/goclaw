## Why

The current memory implementation has production-impacting correctness gaps: restarting a `MemoryHub` instance can panic, decay can over-apply elapsed time, and several memory API behaviors diverge from spec (query mode, list sorting, stats payload). This change aligns runtime behavior and API contracts with the published OpenSpec requirements before further feature work builds on unstable semantics.

## What Changes

- Fix memory decay lifecycle so repeated `Start/Stop` cycles are safe and do not panic.
- Correct decay time progression semantics so periodic decay uses non-overlapping elapsed intervals instead of repeatedly decaying the same time window.
- Align memory query API mode handling so canonical modes are accepted and vector-only flow is usable via HTTP contract.
- Implement list sorting behavior for memory listing endpoint (`sort` and `order`).
- Extend memory stats to include storage size in session statistics responses.
- Add/adjust unit and API tests for lifecycle restart safety, decay progression, query mode handling, list sorting, and stats payload.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `memory-decay`: tighten graceful shutdown/restart behavior and decay progression requirements to prevent double-close panics and repeated interval decay.
- `memory-hub-api`: require stats payload to include storage size and clarify lifecycle robustness expectations.
- `workflow-api-endpoints`: refine memory query/list/stats endpoint contracts (vector-only mode usability, sorting semantics, stats fields).

## Impact

- Affected code: `pkg/memory/fsrs.go`, `pkg/memory/hub.go`, `pkg/memory/entry.go`, `pkg/api/handlers/memory.go`, plus related tests in `pkg/memory/*_test.go` and `pkg/api/handlers/memory_test.go`.
- Affected API surface: `/api/v1/memory/{sessionID}` query mode behavior, `/api/v1/memory/{sessionID}/list` sorting parameters, `/api/v1/memory/{sessionID}/stats` response schema.
- Compatibility: non-breaking intent for existing clients; adds missing spec-compliant behavior and fields.
