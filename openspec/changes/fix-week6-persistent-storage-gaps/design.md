## Context

Week6 persistent storage behavior exists in code but is partially divergent from its intended operational contract. Current gaps include: recovery only resetting state without resubmission, stale Badger status indexes that can corrupt filtered listing semantics, workflow/task persistence divergence during recovery, partial recovery due to fixed batch limits, and mutable references leaking from in-memory storage reads.

The affected logic spans `pkg/engine`, `pkg/storage/badger`, `pkg/storage/memory`, and bootstrap wiring in `cmd/goclaw/main.go`, so this is a cross-cutting reliability fix rather than a localized patch.

## Goals / Non-Goals

**Goals:**
- Define startup recovery semantics that actually resume recoverable workflows.
- Define strict consistency rules between workflow-level task snapshots and task-level persisted rows.
- Define Badger index update/delete behavior and status-filter correctness.
- Define complete recovery scanning behavior for datasets larger than one batch.
- Define mutation-isolation requirements for memory storage reads/lists.
- Define configuration wiring requirements so declared Badger options are honored at runtime.

**Non-Goals:**
- Introduce new external APIs or change HTTP/gRPC response shapes.
- Redesign scheduler/lane execution model.
- Add distributed recovery coordination across nodes.
- Add new storage backends beyond existing memory/Badger.

## Decisions

1. Recovery is two-phase: normalize persisted state, then resubmit.
- Decision: Recovery MUST (a) load pending/running workflows, (b) reset invalid in-flight task/workflow markers, (c) persist normalized state, and (d) explicitly resubmit eligible workflows.
- Rationale: Reset-only recovery leaves workflows permanently pending and violates operational expectations.
- Alternative considered: Keep reset-only behavior and rely on external re-triggering.
  - Rejected because it creates hidden liveness failures and operator burden.

2. Workflow/task persistence must be synchronized during recovery transitions.
- Decision: Any recovery normalization that changes task statuses MUST persist both workflow `TaskStatus` and task records.
- Rationale: Read paths use task-level persistence for task result/status retrieval; workflow-only updates can create contradictory views.
- Alternative considered: Treat workflow `TaskStatus` as source of truth and ignore task rows.
  - Rejected because existing read APIs already use task rows and changing that is out of scope.

3. Badger status index entries are maintained as first-class state.
- Decision: On workflow status change and delete, stale status index entries MUST be removed; filtered listing MUST only return workflows whose current status matches filter and MUST deduplicate by workflow ID.
- Rationale: Append-only index maintenance causes stale reads and duplicate rows.
- Alternative considered: Leave stale indexes and rely on best-effort filtering by caller.
  - Rejected due to correctness drift and unbounded index bloat.

4. Recovery scan must be exhaustive and bounded.
- Decision: Recovery MUST paginate through all pending/running workflows until exhaustion; a single fixed page is insufficient.
- Rationale: Fixed-limit scans silently skip recoverable workloads.
- Alternative considered: Use very large one-shot limit.
  - Rejected because it is still non-exhaustive in worst-case deployments and risks memory spikes.

5. Memory storage returns defensive copies.
- Decision: `GetWorkflow`/`ListWorkflows` MUST deep-copy mutable fields (maps/slices and nested task structures).
- Rationale: Shared references allow callers to mutate persisted state without Save operations.
- Alternative considered: Document mutation caveat.
  - Rejected because it violates storage abstraction guarantees.

6. Bootstrap must wire declared Badger options completely.
- Decision: Startup wiring MUST pass all configured Badger options, including `num_versions_to_keep`, into storage construction.
- Rationale: Unwired config keys are misleading and produce non-deterministic operational behavior.
- Alternative considered: Remove unsupported config field.
  - Rejected; field is already declared and should be honored.

## Risks / Trade-offs

- [Recovery resubmission duplicates execution if already active] -> Mitigation: recovery path checks active execution map and skips already-running workflows.
- [Stricter index cleanup increases write cost] -> Mitigation: perform cleanup within same transaction; prefer correctness over minimal write amplification.
- [Deep copy in memory storage adds CPU/memory overhead] -> Mitigation: scope copies to mutable fields only and keep objects compact.
- [Recovery pagination can increase startup time in very large datasets] -> Mitigation: keep bounded page size and emit progress logs.

## Migration Plan

1. Add/adjust specs for recovery, storage consistency, and Badger index semantics.
2. Implement code changes behind existing execution and storage abstractions.
3. Add regression tests for:
- recovery resubmission,
- exhaustive recovery pagination,
- workflow/task synchronization,
- status-filter correctness with index churn,
- memory copy isolation,
- config option propagation.
4. Rollout as a normal patch release; no API migration required.
5. Rollback by reverting the patch if unexpected startup behavior appears.

## Open Questions

- Should recovery retry failed tasks with remaining retries in this patch, or keep current behavior focused on pending/running only?
- Should Badger index cleanup include opportunistic compaction metrics in this change, or defer to a follow-up observability change?
