## Why

Goclaw currently stores all workflow and task state in memory, which means all data is lost when the service restarts. This prevents production deployment and makes the system unreliable. We need persistent storage to enable service restarts, crash recovery, and production-grade reliability.

## What Changes

- Add storage abstraction layer with pluggable backends
- Implement Badger embedded database as the default storage backend
- Persist workflow state (metadata, status, tasks) to disk
- Persist task execution results and errors
- Add automatic recovery mechanism to restore workflows on startup
- Extend configuration to support storage backend selection
- Maintain backward compatibility with in-memory storage for testing

## Capabilities

### New Capabilities
- `storage-interface`: Abstract storage layer defining operations for workflow and task persistence
- `badger-storage`: Embedded key-value storage implementation using Badger database
- `workflow-persistence`: Workflow state serialization and persistence to storage backend
- `task-persistence`: Task state and result persistence to storage backend
- `recovery-mechanism`: Automatic workflow recovery on service restart

### Modified Capabilities
- `http-server-core`: No requirement changes (implementation uses storage transparently)
- `workflow-api-endpoints`: No requirement changes (API behavior unchanged)

## Impact

**Affected Code:**
- `pkg/engine/engine.go` - Add storage dependency
- `pkg/engine/workflow_manager.go` - Replace in-memory map with storage calls
- `cmd/goclaw/main.go` - Initialize storage backend on startup
- `config/config.go` - Add storage configuration section

**New Dependencies:**
- `github.com/dgraph-io/badger/v4` - Embedded database

**Configuration:**
- New `storage` section in config.yaml
- Storage type selection (memory, badger)
- Badger-specific settings (path, sync mode, etc.)

**Deployment:**
- Requires persistent volume for data directory
- Data directory must be writable by goclaw process
- Backup strategy needed for production

**Performance:**
- Write latency: +5-10ms per workflow operation (acceptable)
- Read latency: +2-5ms per query (acceptable)
- No impact on task execution performance

**Breaking Changes:**
- None - in-memory storage remains available for testing
