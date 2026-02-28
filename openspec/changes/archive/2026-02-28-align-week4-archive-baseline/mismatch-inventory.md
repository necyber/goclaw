# Week4 Archive Mismatch Inventory and Canonical Resolution

## Scope

This document reconciles inconsistencies inside the archived change `week4-engine-core` without editing archive files.  
All evidence references point to files under `openspec/changes/archive/week4-engine-core/*`.

## Mismatch Matrix

| ID | Mismatch | Evidence (archive) | Canonical Resolution | Status | Rationale |
|---|---|---|---|---|---|
| M1 | Task lifecycle state set excludes `cancelled` in proposal/design/tasks but archive specs require it. | `proposal.md:57`; `design.md:90-96`; `tasks.md:7`; `specs/state-tracker-spec.md:11`; `specs/task-runner-spec.md:23` | Baseline interpretation keeps `cancelled` as required behavior. Missing alignment in proposal/design/tasks is treated as documentation drift. | `deferred` | Requirement is normative in specs; conformance requires follow-up code/tests and possibly archive-side explanatory notes in a separate change. |
| M2 | Per-task timeout is required in archive specs but design marks it as not integrated. | `specs/task-runner-spec.md:19`; `design.md:218` | Baseline interpretation keeps timeout propagation as required behavior. Week4 archive reflects an acknowledged implementation gap. | `deferred` | Normative requirement exists; delivery evidence is explicitly missing in design, so code conformance must be addressed later. |
| M3 | Scheduler concurrency mechanism is inconsistent (`errgroup` in proposal/design vs "no errgroup dependency" in tasks). | `proposal.md:81`; `design.md:145,152`; `tasks.md:112`; `tasks.md:33` | Baseline requirement is semantic (intra-layer concurrency + layer barrier + fail-fast), not a specific primitive. `WaitGroup` and `errgroup` are both acceptable implementation strategies. | `baseline` | Archive specs define behavior, not library choice. Primitive-level conflict is normalized as non-normative wording drift. |
| M4 | CLI signal-handling requirement is explicit in archive specs but not explicit in archive tasks/proposal acceptance items. | `specs/cli-engine-bootstrap-spec.md:15,19`; `tasks.md:57-60`; `proposal.md:75` | Baseline interpretation keeps signal handling as required CLI behavior. Missing explicit task wording is treated as coverage gap in planning docs. | `deferred` | Requirement-level statement exists; implementation/test evidence should be tracked in next implementation-focused change. |
| M5 | Completion claims conflict (`proposal` acceptance checklist unchecked vs `tasks` reports 100% complete). | `proposal.md:61-66`; `tasks.md:84-91` | Baseline interpretation: archive completion status is non-authoritative due to contradiction; compliance must be derived from explicit verification in follow-up changes. | `deferred` | Contradictory completion signals cannot both be authoritative; verification is required before claiming conformance. |

## Required Mismatch Coverage Check

The required known mismatches are fully covered by this inventory:

- `cancelled` state mismatch -> `M1`
- timeout semantics mismatch -> `M2`
- scheduler wording mismatch -> `M3`
- CLI signal handling mismatch -> `M4`
- completion-status wording mismatch -> `M5`

## Boundary Statement

- This change defines canonical interpretation only.
- This change does not edit archive files.
- This change does not change runtime code or canonical platform specs.
