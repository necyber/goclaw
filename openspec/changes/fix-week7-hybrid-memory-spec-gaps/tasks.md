## 1. Storage Safety And Enumeration

- [x] 1.1 Add session-scoped delete API in memory storage (`sessionID + entryID`) and keep ID-only path internal-only.
- [x] 1.2 Implement global entry enumeration API for all sessions (for decay/bootstrap flows).
- [x] 1.3 Add storage tests proving cross-session delete is blocked and global enumeration returns all sessions.

## 2. Memory Hub Correctness

- [x] 2.1 Update `MemoryHub.Forget` to enforce session ownership and return actual delete outcomes.
- [x] 2.2 Update decay processing to use global entry enumeration and perform per-session safe forget.
- [x] 2.3 Add startup bootstrap to rebuild vector and BM25 indexes from persisted entries before retrieval.
- [x] 2.4 Add hub tests for cross-session forget safety, full-corpus decay coverage, and restart index rebuild.

## 3. Retrieval Mode Contract

- [x] 3.1 Normalize mode aliases (`vector`/`bm25`) to canonical modes (`vector-only`/`bm25-only`).
- [x] 3.2 Reject unsupported mode values with validation errors (no silent fallback).
- [x] 3.3 Add hybrid retriever tests for alias normalization and unknown-mode rejection.

## 4. API Contract Alignment

- [ ] 4.1 Update memory query handler to validate/propagate canonical mode contract and invalid-mode errors.
- [ ] 4.2 Update delete-memory endpoint response to return actual deleted count from hub.
- [ ] 4.3 Add `GET /api/v1/memory/stats` global statistics endpoint and route wiring.
- [ ] 4.4 Add handler/router tests for global stats endpoint, invalid mode request, and accurate delete count.

## 5. Verification And Regression Safety

- [ ] 5.1 Run targeted suites: `go test ./pkg/memory/... ./pkg/api/handlers/... -race -v`.
- [ ] 5.2 Run full impacted suites: `go test ./pkg/memory/... ./pkg/api/... ./cmd/goclaw/...`.
- [ ] 5.3 Resolve regressions and keep behavior aligned with updated specs/design.
