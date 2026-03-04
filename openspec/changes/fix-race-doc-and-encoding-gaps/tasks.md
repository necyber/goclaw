## 1. Streaming Race-Safety Fix

- [x] 1.1 Make `pkg/grpc/handlers/streaming_test.go` mock stream update buffers concurrency-safe (synchronized append/read helpers).
- [x] 1.2 Update race-prone assertions to use synchronized snapshots instead of direct shared-slice reads.
- [x] 1.3 Run `go test -race ./pkg/grpc/handlers` and fix any remaining race findings.

## 2. Documentation Consistency and Encoding Repair

- [x] 2.1 Update `AGENTS.md` project phase/status statements to match implemented baseline.
- [x] 2.2 Repair UTF-8 readability issues in canonical docs (`README.md`, `ROADMAP.md`) without altering intended meaning.
- [x] 2.3 Normalize `openspec/specs/api-documentation/spec.md` maintained baseline text to readable UTF-8 where scoped by this change.

## 3. Verification and Governance Closure

- [ ] 3.1 Run `go test ./...` and `go vet ./...` after fixes.
- [ ] 3.2 Run `openspec validate --change fix-race-doc-and-encoding-gaps --strict`.
- [ ] 3.3 Ensure this change artifacts and modified files are committed together with a traceable message.
