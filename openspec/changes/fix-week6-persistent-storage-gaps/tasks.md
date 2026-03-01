## 1. Recovery Execution Semantics

- [x] 1.1 Refactor `Engine.RecoverWorkflows` to iterate through all recoverable workflows using paginated scans instead of a single fixed page.
- [x] 1.2 Implement recovery resubmission flow that normalizes state, persists updates, and starts execution for recoverable workflows.
- [x] 1.3 Add duplicate-execution protection so recovery skips workflows already present in the active execution registry.
- [x] 1.4 Ensure recovery errors are aggregated and logged without blocking engine startup.

## 2. Workflow/Task Persistence Consistency

- [ ] 2.1 Update recovery normalization to persist task-level state updates via storage task APIs alongside workflow snapshot updates.
- [ ] 2.2 Ensure task runtime fields (started/completed/error) are cleared consistently when resetting running tasks to pending.
- [ ] 2.3 Add regression coverage for workflow/task consistency after simulated restart and recovery.

## 3. Badger Index Correctness

- [ ] 3.1 Update Badger `SaveWorkflow` to remove stale status index keys during status transitions within the same transaction.
- [ ] 3.2 Update Badger `DeleteWorkflow` to delete associated status/created index keys in addition to workflow/task records.
- [ ] 3.3 Harden status-filtered `ListWorkflows` to verify current workflow status and deduplicate by workflow ID.
- [ ] 3.4 Add Badger tests covering status churn, stale index keys, and duplicate prevention.

## 4. Memory Storage Isolation

- [ ] 4.1 Implement deep-copy helpers for mutable workflow/task fields (maps, slices, nested task state) in memory storage.
- [ ] 4.2 Apply deep-copy helpers consistently in `SaveWorkflow`, `GetWorkflow`, `ListWorkflows`, and task list/read paths.
- [ ] 4.3 Add mutation-isolation tests proving caller-side mutations do not alter persisted state without explicit saves.

## 5. Configuration Wiring and Bootstrap

- [ ] 5.1 Propagate `storage.badger.num_versions_to_keep` from config into `badgerstorage.Config` in `cmd/goclaw/main.go`.
- [ ] 5.2 Add/adjust bootstrap tests validating all declared Badger config fields are wired into storage initialization.

## 6. Verification and Regression Safety

- [ ] 6.1 Extend engine tests for recovery resubmission and full-batch recovery coverage (>1 page of recoverable workflows).
- [ ] 6.2 Extend API/runtime tests to confirm task-result reads are consistent with normalized post-recovery task records.
- [ ] 6.3 Run `go test ./pkg/storage/... ./pkg/engine/... ./cmd/goclaw/...` and resolve new regressions introduced by this change.
