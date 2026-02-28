## 1. Baseline Alignment Artifacts

- [x] 1.1 Build a mismatch inventory covering archived `proposal.md`, `design.md`, `tasks.md`, and `specs/*` statements for Week4.
- [x] 1.2 Record one canonical resolution for each mismatch, including status (`baseline` vs `deferred`) and rationale.
- [x] 1.3 Ensure all alignment outputs are stored only under `openspec/changes/align-week4-archive-baseline/*` and do not edit archive files.

## 2. Traceability and Reviewability

- [x] 2.1 Add source traceability references for each mismatch/resolution entry so reviewers can locate original archive statements.
- [x] 2.2 Verify that all required known mismatches are explicitly covered (`cancelled` state, timeout semantics, scheduler wording, CLI signal handling, completion-status wording).
- [x] 2.3 Run a consistency pass to ensure proposal/design/spec/tasks use the same scope boundary and terminology.

## 3. Follow-up Boundary and Handoff

- [x] 3.1 Document deferred implementation items that require code changes and confirm they are out of scope for this change.
- [x] 3.2 Prepare a handoff note for the next implementation-focused change to address deferred Week4 behavior gaps.
- [x] 3.3 Re-run change status and verification checks to confirm this change is apply-ready with no archive-file mutations.
