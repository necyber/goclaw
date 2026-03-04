## Why

Recent project-wide assessment found three gaps that can cause maintainability and quality regression: a reproducible Go race detector failure in gRPC streaming tests, documentation truth drift about project maturity, and recurring UTF-8 rendering issues in key markdown documents. Addressing them now prevents hidden concurrency defects, reduces onboarding confusion, and stabilizes documentation tooling behavior across environments.

## What Changes

- Tighten streaming-related quality requirements so concurrency-sensitive paths are verified under race detection in targeted test scope.
- Add repository documentation governance requirements for phase/status accuracy and UTF-8 consistency in top-level and roadmap documents.
- Repair current document drift and encoding artifacts in affected files (README/roadmap/agent guide and related spec notes).
- Add verification tasks to ensure quality gates (`go test -race` target scope and spec validation) remain green.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `streaming-support`: add race-safety verification requirements for concurrent streaming fan-out and bridge integration paths.
- `spec-format-governance`: add repository documentation consistency and UTF-8 encoding governance requirements for canonical project docs.
- `api-documentation`: require readable UTF-8 documentation baseline for normative API documentation content.

## Impact

- Affected specs:
  - `openspec/specs/streaming-support/spec.md`
  - `openspec/specs/spec-format-governance/spec.md`
  - `openspec/specs/api-documentation/spec.md`
- Affected code/docs (expected):
  - `pkg/grpc/handlers/streaming_test.go` (race-safe test fixture/synchronization)
  - `README.md`, `ROADMAP.md`, `AGENTS.md`
  - `openspec/specs/api-documentation/spec.md` (legacy notes encoding normalization where needed)
- Affected validation:
  - Targeted `go test -race` for gRPC streaming handlers
  - `openspec validate --change fix-race-doc-and-encoding-gaps --strict`
