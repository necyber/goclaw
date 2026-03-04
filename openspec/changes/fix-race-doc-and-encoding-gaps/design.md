## Context

This change spans runtime-adjacent test reliability and repository documentation governance. The current baseline passes regular tests, but `go test -race ./...` reports a data race in gRPC streaming handler tests where shared update slices are appended and read concurrently. In parallel, project-facing markdown documents have two governance issues: some files drift from actual implementation maturity, and UTF-8 text can render as mojibake in common Windows terminal/editor paths when encoding handling is inconsistent.

The goal is to lock down these quality gaps without changing public runtime APIs.

## Goals / Non-Goals

**Goals:**
- Eliminate race detector failures in streaming handler test paths with deterministic synchronization.
- Add explicit spec requirements that concurrent streaming paths must remain race-safe under targeted race checks.
- Add explicit governance requirements that canonical project documents remain phase-accurate and UTF-8 readable.
- Normalize currently affected documentation/spec artifacts to readable UTF-8 content.

**Non-Goals:**
- Re-architecting gRPC streaming runtime behavior or subscriber registry internals.
- Introducing new external documentation tooling stacks.
- Redesigning all historical archived content beyond scoped errata/normalization needed for readability and consistency.

## Decisions

1. Harden streaming test fixtures with synchronization primitives.
- Decision: update streaming test mock streams to guard mutable shared slices/counters with `sync.Mutex` and expose snapshot/read helpers for assertions.
- Rationale: race is in test fixture shared state, not core handler logic; localized synchronization resolves detector failures with minimal behavior change.
- Alternatives considered:
  - Rewriting tests to fully channel-driven assertions: cleaner concurrency model but larger refactor and slower delivery.
  - Ignoring package in race runs: hides regressions and conflicts with quality objective.

2. Encode race-safety as a capability requirement in `streaming-support`.
- Decision: add requirement/scenarios that targeted streaming package tests must pass under `-race`, including event-bus bridge integration paths.
- Rationale: converts implicit quality expectation into enforceable spec contract.
- Alternatives considered:
  - Keep as implementation-only note in tasks: weaker long-term governance.

3. Add repository documentation governance rules in `spec-format-governance`.
- Decision: require canonical project docs (README, AGENTS guide, roadmap-like status docs) to remain implementation-phase accurate and UTF-8 readable.
- Rationale: documented project state is part of system usability and contributor correctness.
- Alternatives considered:
  - Track only in CONTRIBUTING checklist: not machine-verifiable and easy to drift.

4. Repair `api-documentation` readable baseline through requirement-level normalization.
- Decision: add requirement that normative API documentation text must remain readable UTF-8 and not contain mojibake in maintained baseline docs/spec notes.
- Rationale: current spec notes contain corrupted text and undermine API-documentation utility.
- Alternatives considered:
  - Leave legacy notes untouched forever: preserves history but keeps current baseline unusable.

## Risks / Trade-offs

- [Risk] Added test synchronization could mask timing bugs if over-serialized. -> Mitigation: lock only around shared structure mutation/read, keep stream lifecycle async.
- [Risk] UTF-8 governance rules may be interpreted inconsistently across editors. -> Mitigation: define explicit scope in spec scenarios and normalize key files in this change.
- [Risk] Touching documentation files can create large diffs. -> Mitigation: constrain edits to affected sections and preserve semantics.
- [Risk] Race checks across all packages can be costly in CI. -> Mitigation: require targeted race checks for streaming-concurrency scope first; expand later if needed.

## Migration Plan

1. Update specs for `streaming-support`, `spec-format-governance`, and `api-documentation`.
2. Implement race-safe updates in `pkg/grpc/handlers/streaming_test.go`.
3. Normalize and align project docs (`README.md`, `ROADMAP.md`, `AGENTS.md`) and any scoped spec-note encoding drift.
4. Validate with targeted race tests and repository checks:
   - `go test -race ./pkg/grpc/handlers`
   - `go test ./...`
   - `openspec validate --change fix-race-doc-and-encoding-gaps --strict`

Rollback strategy: revert this change commit; no persistent data migration is involved.

## Open Questions

- Should we introduce `.gitattributes`/`.editorconfig` in this same change to harden encoding defaults, or keep this change scoped to behavior/spec/document fixes only?
- For legacy archived markdown with mojibake, should normalization be done incrementally under errata policy or in one dedicated maintenance change?
