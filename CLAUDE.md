# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
make build          # Build binary to ./bin/goclaw
make build-race     # Build with race detector

# Test
make test           # Run all tests with race detector
make test-short     # Run short tests only
make test-coverage  # Generate HTML coverage report
go test ./pkg/dag/... -run TestName  # Run a single test

# Quality
make fmt            # Format code
make vet            # Run go vet
make lint           # Run golangci-lint (requires golangci-lint installed)
make check          # Run fmt + vet + test + lint

# Run
make run            # Build and run
make run-dev        # Hot reload (requires air)
```

## Architecture

Goclaw is a distributed multi-agent orchestration engine. The core execution model is: tasks are defined as a DAG, compiled into an execution plan with layered topological ordering, then dispatched through Lane queues for concurrent execution.

### Key packages

**`pkg/dag/`** — DAG compiler pipeline:
- `task.go` defines the `Task` type with dependencies
- `dag.go` builds the graph from tasks
- `cycle.go` detects cycles via DFS; `toposort.go` sorts via Kahn's algorithm
- `plan.go` generates a layered `ExecutionPlan` (tasks grouped by execution wave)

**`pkg/lane/`** — Resource-constrained execution queues:
- `lane.go` defines the `Lane` interface and `TaskFunc`
- `channel_lane.go` is the channel-based implementation
- `worker_pool.go` manages concurrent workers; `priority_queue.go` orders tasks
- `rate_limiter.go` handles backpressure (Block / Drop / Redirect strategies)
- `manager.go` coordinates multiple named lanes

**`pkg/engine/`** — Orchestration engine (stub, Phase 1): wires DAG + Lane together; manages state (Idle → Running → Stopped/Error).

**`config/`** — Multi-source config loading via [koanf](https://github.com/knadh/koanf): defaults → YAML/JSON file → env vars (`GOCLAW_` prefix) → CLI flags. Validated with `go-playground/validator`.

**`pkg/logger/`** — Thin wrapper around Go's `log/slog` with JSON/text format and file/stdout output.

**`cmd/goclaw/main.go`** — CLI entry point: loads config, initializes logger, starts engine, handles graceful shutdown on SIGINT/SIGTERM.

### Configuration

Copy `config/config.example.yaml` as your config file. Key sections: `app`, `server` (HTTP :8080, gRPC :9090), `log`, `orchestration`, `cluster`, `storage`, `metrics`, `tracing`.

### Development phases

- **Phase 1 (current):** DAG, Lane, in-memory storage, CLI — implemented
- **Phase 2:** Persistent storage (Badger/Redis), distributed mode (Consul/etcd)
- **Phase 3:** gRPC API, Prometheus metrics, Web UI
