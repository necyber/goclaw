## Why

Week6 persistent storage was implemented, but current behavior still has critical gaps: recovery does not resume execution, Badger status indexes can become stale and return incorrect results, and workflow/task records can diverge after restart. These gaps can cause stuck workflows, wrong query results, and inconsistent task visibility.

## What Changes

- Define normative recovery behavior so startup recovery both resets invalid in-flight states and resubmits recoverable workflows for continued execution.
- Define consistency rules between workflow-level `TaskStatus` and task-level persisted records during recovery and status transitions.
- Define Badger index maintenance requirements so status-filtered queries only return workflows whose current persisted status matches filters.
- Define pagination/iteration requirements for recovery scans so large pending/running sets are fully processed.
- Define defensive copy requirements for in-memory storage responses to prevent caller mutation of persisted maps/slices.
- Define configuration wiring requirements so all declared Badger options are actually applied at bootstrap.

## Capabilities

### New Capabilities
- `storage-interface`: Defines storage consistency, mutation isolation, and typed error requirements across workflow/task persistence.
- `badger-storage`: Defines Badger index lifecycle and filtered query correctness requirements.
- `workflow-persistence`: Defines workflow persistence behavior for status transitions and recovery updates.
- `task-persistence`: Defines task persistence synchronization requirements with workflow state.
- `recovery-mechanism`: Defines startup recovery, batching, resubmission, and idempotent recovery semantics.

### Modified Capabilities
- _(none)_

## Impact

- Affected code:
  - `pkg/engine/engine.go`
  - `pkg/engine/workflow_manager.go`
  - `pkg/storage/badger/badger.go`
  - `pkg/storage/memory/memory.go`
  - `cmd/goclaw/main.go`
- Affected tests:
  - `pkg/engine/*test.go`
  - `pkg/storage/badger/*test.go`
  - `pkg/storage/memory/*test.go`
- No external API contract changes expected; behavior alignment is internal correctness and reliability.
