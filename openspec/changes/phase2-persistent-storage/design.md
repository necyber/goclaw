## Context

Goclaw currently uses in-memory storage for all workflow and task state, implemented as Go maps with mutex protection in `pkg/engine/workflow_manager.go`. This works for development and testing but prevents production deployment because:

1. All data is lost on service restart or crash
2. No way to recover running workflows after failure
3. Cannot scale beyond single instance (no shared state)
4. No audit trail or historical data

The system needs persistent storage to become production-ready. This design adds a storage abstraction layer with Badger as the default embedded database, enabling service restarts while maintaining backward compatibility with in-memory storage for testing.

**Current State:**
- `WorkflowStore` in `pkg/engine/workflow_manager.go` uses `map[string]*WorkflowState`
- Protected by `sync.RWMutex` for concurrent access
- No persistence, no recovery mechanism
- HTTP API in `pkg/api` calls engine methods that use in-memory storage

**Constraints:**
- Must maintain API compatibility (no breaking changes)
- Must support both embedded (Badger) and in-memory storage
- Must handle concurrent access safely
- Performance overhead must be acceptable (<10ms write latency)
- Must work on Windows, Linux, macOS

**Stakeholders:**
- Developers: Need reliable testing with in-memory storage
- Operators: Need production deployment with persistence
- End users: Expect workflows to survive service restarts

## Goals / Non-Goals

**Goals:**
- Add storage abstraction layer with pluggable backends
- Implement Badger embedded storage as default production backend
- Persist workflow state (metadata, status, tasks) automatically
- Persist task execution results and errors
- Implement automatic recovery on service startup
- Maintain backward compatibility with in-memory storage
- Achieve <10ms write latency, <5ms read latency (P99)
- Support concurrent access from multiple goroutines
- Provide clear error messages for storage failures

**Non-Goals:**
- Distributed storage (Redis) - deferred to Phase 2.2
- Data encryption at rest - can be added later
- Storage migration tools - manual migration acceptable for now
- Real-time replication - single-node only in this phase
- Query optimization beyond basic indexing
- Storage metrics and monitoring - deferred to Phase 3

## Decisions

### Decision 1: Storage Abstraction Layer

**Choice:** Create `pkg/storage` package with `Storage` interface

**Rationale:**
- Allows switching between in-memory, Badger, and future backends (Redis)
- Enables testing with mock storage
- Isolates storage concerns from business logic
- Standard Go pattern for dependency injection

**Alternatives Considered:**
- Direct Badger integration in engine: Rejected - tight coupling, hard to test
- Generic KV interface: Rejected - too low-level, loses domain semantics

**Implementation:**
```go
type Storage interface {
    SaveWorkflow(ctx context.Context, wf *WorkflowState) error
    GetWorkflow(ctx context.Context, id string) (*WorkflowState, error)
    ListWorkflows(ctx context.Context, filter *WorkflowFilter) ([]*WorkflowState, int, error)
    DeleteWorkflow(ctx context.Context, id string) error

    SaveTask(ctx context.Context, workflowID string, task *TaskState) error
    GetTask(ctx context.Context, workflowID, taskID string) (*TaskState, error)
    ListTasks(ctx context.Context, workflowID string) ([]*TaskState, error)

    Close() error
}
```

### Decision 2: Badger as Default Storage Backend

**Choice:** Use `github.com/dgraph-io/badger/v4` for embedded storage

**Rationale:**
- Pure Go implementation (no CGO, easy cross-platform builds)
- Embedded database (no external dependencies)
- High performance (LSM-tree architecture)
- ACID transactions
- Active maintenance and good documentation
- Used in production by Dgraph and other projects

**Alternatives Considered:**
- BoltDB: Rejected - archived project, no longer maintained
- LevelDB: Rejected - requires CGO, harder to build
- SQLite: Rejected - overkill for KV storage, CGO dependency
- Redis: Rejected - requires external service, deferred to Phase 2.2

**Trade-offs:**
- Pros: Fast, reliable, no external dependencies
- Cons: Single-node only, no built-in replication

### Decision 3: Key Structure and Data Model

**Choice:** Hierarchical key structure with JSON values

**Key Format:**
```
workflow:{id}                          -> WorkflowState (JSON)
workflow:{id}:task:{tid}               -> TaskState (JSON)
workflow:index:status:{status}:{id}    -> "" (empty value, for indexing)
workflow:index:created:{timestamp}:{id} -> "" (for time-based queries)
```

**Rationale:**
- Prefix-based scanning enables efficient queries
- JSON serialization is simple and debuggable
- Index keys enable filtering without full scans
- Hierarchical structure groups related data

**Alternatives Considered:**
- Protobuf serialization: Rejected - adds complexity, JSON is sufficient
- Flat key structure: Rejected - harder to query and maintain
- Separate index tables: Rejected - more complex, same performance

### Decision 4: Transaction Strategy

**Choice:** Use Badger transactions for atomic operations

**Approach:**
- Single workflow save: Single transaction
- Workflow + tasks save: Single transaction with multiple writes
- Batch operations: Single transaction for consistency

**Rationale:**
- Ensures data consistency
- Prevents partial writes on failure
- Badger transactions are efficient

**Trade-offs:**
- Larger transactions may increase latency
- Acceptable for typical workflow sizes (<100 tasks)

### Decision 5: Recovery Mechanism

**Choice:** Automatic recovery on engine startup

**Approach:**
```go
func (e *Engine) Start(ctx context.Context) error {
    // 1. Initialize storage
    if err := e.storage.Open(); err != nil {
        return err
    }

    // 2. Recover incomplete workflows
    if err := e.RecoverWorkflows(ctx); err != nil {
        e.logger.Warn("recovery failed", "error", err)
        // Continue startup - don't block on recovery failures
    }

    // 3. Start execution engine
    return e.startExecution()
}

func (e *Engine) RecoverWorkflows(ctx context.Context) error {
    // Load workflows with status: pending, running
    workflows, err := e.storage.ListWorkflows(ctx, &storage.WorkflowFilter{
        Status: []string{"pending", "running"},
    })

    for _, wf := range workflows {
        // Resubmit to execution queue
        if err := e.resubmitWorkflow(ctx, wf); err != nil {
            e.logger.Error("failed to recover workflow", "id", wf.ID, "error", err)
            // Continue with next workflow
        }
    }

    return nil
}
```

**Rationale:**
- Automatic recovery improves reliability
- Non-blocking recovery allows service to start even if some workflows fail
- Logging provides visibility into recovery process

**Alternatives Considered:**
- Manual recovery via API: Rejected - requires operator intervention
- Background recovery: Rejected - delays workflow execution
- Skip recovery: Rejected - defeats purpose of persistence

### Decision 6: Configuration Structure

**Choice:** Add `storage` section to config.yaml

```yaml
storage:
  type: badger  # or "memory" for testing

  badger:
    path: ./data/badger
    sync_writes: true
    value_log_file_size: 1073741824  # 1GB
    num_versions_to_keep: 1
```

**Rationale:**
- Clear separation of storage config
- Type field enables backend selection
- Backend-specific config in nested sections
- Sensible defaults for production

### Decision 7: Error Handling

**Choice:** Define typed errors for different failure scenarios

```go
type NotFoundError struct { ID string }
type DuplicateKeyError struct { ID string }
type StorageUnavailableError struct { Cause error }
type SerializationError struct { Cause error }
```

**Rationale:**
- Enables proper error handling in API layer
- Allows mapping to HTTP status codes
- Provides context for debugging

## Risks / Trade-offs

### Risk 1: Storage Corruption
**Risk:** Badger database files could become corrupted due to crashes or disk failures

**Mitigation:**
- Enable sync writes in production (durability over performance)
- Implement health checks that detect corruption
- Provide recovery guidance in documentation
- Consider periodic backups (manual for now)

### Risk 2: Performance Degradation
**Risk:** Storage operations add latency to workflow operations

**Mitigation:**
- Target <10ms write latency (acceptable for workflow operations)
- Use async writes for non-critical updates (future optimization)
- Benchmark and monitor storage performance
- Keep in-memory storage option for performance-critical testing

### Risk 3: Storage Space Growth
**Risk:** Workflow data accumulates over time, consuming disk space

**Mitigation:**
- Implement cleanup policy for completed workflows (configurable retention)
- Document storage requirements in deployment guide
- Add storage usage monitoring (Phase 3)
- Badger GC reclaims space from deleted data

### Risk 4: Migration Complexity
**Risk:** Existing deployments need to migrate from in-memory to persistent storage

**Mitigation:**
- In-memory storage remains default for backward compatibility
- Provide clear migration guide
- No data migration needed (fresh start acceptable for Phase 2)
- Future: Add export/import tools if needed

### Risk 5: Concurrent Access Issues
**Risk:** Race conditions between storage operations and in-memory caching

**Mitigation:**
- Remove in-memory caching - storage is source of truth
- Rely on Badger's concurrency safety
- Add integration tests for concurrent scenarios
- Use context for cancellation and timeouts

## Migration Plan

### Phase 1: Implementation (Week 1-2)
1. Create `pkg/storage` package with interface
2. Implement `MemoryStorage` (refactor existing code)
3. Implement `BadgerStorage` with tests
4. Add storage configuration
5. Update engine to use storage interface

### Phase 2: Integration (Week 2)
1. Replace `WorkflowStore` with `Storage` interface
2. Implement recovery mechanism
3. Update main.go to initialize storage
4. Add integration tests

### Phase 3: Testing (Week 3)
1. Unit tests for storage implementations
2. Integration tests for recovery
3. Performance benchmarks
4. End-to-end testing with restarts

### Phase 4: Documentation (Week 3)
1. Update README with storage configuration
2. Add deployment guide with storage requirements
3. Document backup and recovery procedures
4. Update CLAUDE.md with architecture changes

### Rollback Strategy
- If issues found: Set `storage.type: memory` in config
- No data loss risk (fresh start acceptable)
- Can revert code changes without data migration

### Deployment Checklist
- [ ] Create data directory with write permissions
- [ ] Configure storage type in config.yaml
- [ ] Set appropriate value_log_file_size for workload
- [ ] Enable sync_writes for production
- [ ] Monitor storage space usage
- [ ] Test recovery by restarting service

## Open Questions

### Q1: Cleanup Policy
**Question:** How long should completed workflows be retained?

**Options:**
- A: Keep forever (until manual deletion)
- B: Auto-delete after N days (configurable)
- C: Keep last N workflows per status

**Recommendation:** Start with A (keep forever), add B in Phase 2.3 based on user feedback

### Q2: Large Task Results
**Question:** How to handle task results that exceed reasonable size (>1MB)?

**Options:**
- A: Truncate and log warning
- B: Store reference to external blob storage
- C: Reject task with error

**Recommendation:** Start with A (truncate at 1MB), add B if needed

### Q3: Backup Strategy
**Question:** Should we provide built-in backup functionality?

**Options:**
- A: Document manual backup (copy data directory)
- B: Add backup API endpoint
- C: Integrate with external backup tools

**Recommendation:** Start with A (documentation), evaluate B based on user needs

### Q4: Storage Metrics
**Question:** What storage metrics should be exposed?

**Deferred to Phase 3** (Prometheus metrics):
- Storage size
- Operation latency (P50, P99)
- Error rates
- Recovery statistics
