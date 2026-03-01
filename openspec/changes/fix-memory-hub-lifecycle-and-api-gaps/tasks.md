## 1. Decay Lifecycle Safety

- [x] 1.1 Refactor `DecayManager` loop lifecycle state so each `StartDecayLoop` initializes per-run coordination channels/state safely.
- [x] 1.2 Make repeated `Stop` calls safe and ensure no double-close panic occurs across `Start -> Stop -> Start -> Stop` cycles.
- [x] 1.3 Add/adjust `MemoryHub.Start`/`Stop` integration paths to preserve restart behavior and avoid goroutine leaks.

## 2. Decay Progression Correctness

- [x] 2.1 Update decay strength progression logic so periodic decay uses non-overlapping elapsed time windows.
- [x] 2.2 Persist decay reference timestamp updates (`LastReview` or equivalent) when decay is applied.
- [x] 2.3 Verify threshold-based forgetting still works correctly after timestamp progression changes.

## 3. Query API Contract Alignment

- [x] 3.1 Extend memory query endpoint parsing/validation so canonical `mode=vector-only` requests can execute with vector input and without mandatory text query.
- [x] 3.2 Keep strict validation for unsupported modes and malformed vector input with deterministic 4xx responses.
- [x] 3.3 Ensure hub retrieval mode handling remains consistent between HTTP layer and `memory.Query` semantics.

## 4. List and Stats Contract Alignment

- [x] 4.1 Implement `/api/v1/memory/{sessionID}/list` sorting support for `sort` and `order` semantics before pagination.
- [x] 4.2 Add storage size (bytes) to session memory statistics model and response payload.
- [x] 4.3 Ensure global/session stats behavior remains backward-compatible except for additive fields.

## 5. Regression Tests and Verification

- [x] 5.1 Add regression test that reproduces prior `close of closed channel` panic and passes with lifecycle fix.
- [x] 5.2 Add tests for non-overlapping decay progression across consecutive decay runs.
- [x] 5.3 Add API tests for vector-only query mode success path and invalid mode/vector validation paths.
- [x] 5.4 Add API tests for list sorting behavior and stats `storage size` field presence/value.
- [x] 5.5 Run focused test suites for `pkg/memory` and `pkg/api/handlers` and confirm all pass.
