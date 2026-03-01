## Context

The Week7 hybrid memory stack is now integrated into runtime and HTTP APIs, but post-implementation review identified correctness and contract drifts: `MemoryHub` can panic after restart cycles, decay can repeatedly apply the same elapsed window, and the memory API still misses or partially implements required behavior (`vector-only` mode usability over HTTP, list sorting semantics, and session stats storage size field).

This is a cross-module fix touching memory runtime internals (`pkg/memory`) and API handlers (`pkg/api/handlers`). The main constraints are: preserve backward compatibility for existing clients, keep changes incremental, and ensure behavior is enforced by tests to avoid regression.

## Goals / Non-Goals

**Goals:**
- Make decay loop lifecycle safe across repeated `Start/Stop` cycles on the same `MemoryHub` instance.
- Ensure periodic decay applies only newly elapsed time windows (no repeated over-decay of prior intervals).
- Align HTTP query mode behavior with canonical mode requirements and make vector-only mode usable through API contract.
- Implement list sorting contract for memory listing endpoints.
- Include storage size in session memory statistics response and corresponding model/spec alignment.
- Add focused tests that reproduce prior failures and lock in corrected behavior.

**Non-Goals:**
- Replacing brute-force vector retrieval with HNSW/ANN in this change.
- Changing persistence backend or key schema.
- Introducing breaking API route changes.
- Implementing unrelated memory features (multi-modal memory, distributed memory sync).

## Decisions

### 1) Decay loop restart safety via per-start lifecycle primitives

- Decision: Reinitialize loop lifecycle state (`done` channel and cancellable context ownership) each time the decay loop starts, and guard start/stop transitions to avoid double-close and stale waiter state.
- Rationale: Current one-time channel allocation causes `close of closed channel` on second stop cycle. Per-run lifecycle primitives make `Start/Stop` idempotent and restart-safe.
- Alternatives considered:
  - Keep a single persistent `done` channel and avoid closing it: harder to reason about wait semantics, risks goroutine leaks.
  - Allocate a brand-new `DecayManager` on every `MemoryHub.Start`: larger object churn and hidden ownership complexity.

### 2) Decay time progression should advance LastReview after decay updates

- Decision: When decay is applied to an entry as part of periodic processing, persist both updated `Strength` and advanced reference timestamp (`LastReview`) so subsequent cycles only apply newly elapsed time.
- Rationale: FSRS decay is based on elapsed time from last update/review; not advancing timestamp causes repeated decay over identical interval and underestimates memory strength.
- Alternatives considered:
  - Keep current timestamp unchanged and infer last decay externally: adds hidden state and increases coupling.
  - Store separate `LastDecayAt` field: possible, but unnecessary for current scope and increases data model surface.

### 3) API contract alignment for query mode, list sorting, and stats payload

- Decision: Extend memory query HTTP contract so canonical mode requests are validated consistently and vector-only mode can be executed using request-supplied vector input (while preserving text-query path). Implement `sort` + `order` for list endpoint and include `storage_size` in session stats response.
- Rationale: Existing behavior rejects valid canonical requests in practice and leaves required list/stats behavior unimplemented.
- Alternatives considered:
  - Keep GET query endpoint text-only and de-scope vector-only over HTTP: violates existing spec contract.
  - Add a separate endpoint for vector query now: potentially cleaner, but unnecessary scope expansion for this bugfix.

### 4) Regression-first tests for each identified gap

- Decision: Add dedicated tests for restart panic regression, non-overlapping decay progression, API vector-only mode path, list sorting semantics, and session stats storage size field.
- Rationale: All discovered gaps were behavior-level regressions not caught by existing tests; fixes must be locked with specific failure-repro tests.
- Alternatives considered:
  - Rely only on integration tests: slower feedback and weaker fault localization.

## Risks / Trade-offs

- [Risk] Decay timestamp advancement may change retrieval rankings compared with previous buggy behavior.
  - Mitigation: update specs/tests to define intended semantics and communicate behavior correction in change notes.
- [Risk] Extending query input parsing for vector-only mode can create validation ambiguity (missing/invalid vector).
  - Mitigation: explicit validation rules and deterministic `400 VALIDATION_FAILED` responses.
- [Risk] Sorting implementation over paginated list may increase per-request work if done post-scan.
  - Mitigation: keep scope minimal now, validate behavior correctness first, optimize ordering path later if needed.

## Migration Plan

1. Implement runtime safety and decay progression fixes in `pkg/memory`.
2. Update API request/response handling for query mode, list sorting, and stats payload.
3. Add and run focused unit/API tests covering all corrected contracts.
4. Roll out with no config migration required.

Rollback strategy:
- Revert this change set if unexpected behavior appears; storage format remains unchanged, so rollback is code-only.

## Open Questions

- Should vector-only HTTP queries accept vectors only in request body (POST) or also via encoded query params for GET compatibility? (This change will choose one consistent path and document it.)
- Should list sorting be limited to known fields (`created_at`, `strength`) with strict validation, or silently fallback on unknown fields?
