## Context

The archive alignment change established a canonical baseline and deferred implementation work into a dedicated coding change. This change delivers concrete runtime conformance in engine/CLI paths with tests as the source of truth.

## Goals / Non-Goals

**Goals:**
- Ensure task cancellation and timeout behavior conforms to Week4 conformance expectations.
- Ensure scheduler behavior is deterministic for fail-fast and layer barriers when cancellation/errors occur.
- Ensure CLI process termination signals trigger controlled shutdown behavior.
- Add/adjust tests to make conformance verifiable.

**Non-Goals:**
- No archive document edits.
- No unrelated feature expansion outside Week4 conformance scope.
- No broad refactor of engine architecture unless required by conformance fixes.

## Decisions

### Decision 1: Treat conformance as behavior + tests
Implementation is considered complete only when behavior is enforced and covered by focused tests.

Alternative considered:
- Code-only tweaks without new/updated tests.
- Rejected because conformance would remain unverifiable.

### Decision 2: Preserve existing public contracts where possible
Prefer internal logic updates and additive tests over API-shape changes.

Alternative considered:
- Redesign runtime interfaces.
- Rejected because it increases migration risk and scope.

### Decision 3: Stage implementation by tasks phases
Tasks are grouped into execution phases so each phase can be compiled/tested and committed independently.

Alternative considered:
- One large implementation batch.
- Rejected because it reduces isolation and rollback clarity.

## Risks / Trade-offs

- [Risk] Existing tests may assert old semantics.
  -> Mitigation: update tests deliberately with clear scenario intent.

- [Risk] Signal-handling tests can be flaky due to timing.
  -> Mitigation: use deterministic synchronization and bounded timeouts.

- [Risk] Timeout/cancellation edges may vary by call path.
  -> Mitigation: cover task-runner and scheduler interactions in integration tests.
