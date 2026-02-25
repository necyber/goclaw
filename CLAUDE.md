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

# Integration & Benchmark Tests
go test ./pkg/api -run TestIntegration  # Run integration tests
go test ./pkg/api -bench=. -benchtime=1s  # Run performance benchmarks

# Quality
make fmt            # Format code
make vet            # Run go vet
make lint           # Run golangci-lint (requires golangci-lint installed)
make check          # Run fmt + vet + test + lint

# Run
make run            # Build and run
make run-dev        # Hot reload (requires air)

# API Documentation
# Generate Swagger docs after modifying API handlers
go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/goclaw/main.go -o docs/swagger
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

**`pkg/engine/`** — Orchestration engine: wires DAG + Lane together; manages state (Idle → Running → Stopped/Error); provides workflow management API.

**`pkg/api/`** — HTTP API server:
- `server.go` defines HTTP server with graceful shutdown
- `router.go` sets up chi router with middleware chain
- `handlers/` contains workflow and health check handlers
- `middleware/` provides RequestID, Logger, Recovery, CORS, Timeout
- `response/` contains JSON response helpers and error formatting
- `models/` defines API request/response data structures

**`config/`** — Multi-source config loading via [koanf](https://github.com/knadh/koanf): defaults → YAML/JSON file → env vars (`GOCLAW_` prefix) → CLI flags. Validated with `go-playground/validator`.

**`pkg/logger/`** — Thin wrapper around Go's `log/slog` with JSON/text format and file/stdout output.

**`cmd/goclaw/main.go`** — CLI entry point: loads config, initializes logger, starts engine and HTTP server, handles graceful shutdown on SIGINT/SIGTERM.

### Configuration

Copy `config/config.example.yaml` as your config file. Key sections: `app`, `server` (HTTP :8080, gRPC :9090), `log`, `orchestration`, `cluster`, `storage`, `metrics`, `tracing`.

### Development phases

- **Phase 1:** DAG, Lane, in-memory storage, CLI, HTTP API — implemented ✅
- **Phase 2:** Persistent storage (Badger/Redis), distributed mode (Consul/etcd)
- **Phase 3:** gRPC API, Prometheus metrics, Web UI

### HTTP API

The HTTP API server runs on port 8080 (configurable) and provides:

**Workflow Management:**
- `POST /api/v1/workflows` - Submit workflow
- `GET /api/v1/workflows` - List workflows (pagination supported)
- `GET /api/v1/workflows/{id}` - Get workflow status
- `POST /api/v1/workflows/{id}/cancel` - Cancel workflow
- `GET /api/v1/workflows/{id}/tasks/{tid}/result` - Get task result

**Health Checks:**
- `GET /health` - Liveness probe (Kubernetes compatible)
- `GET /ready` - Readiness probe (Kubernetes compatible)
- `GET /status` - Detailed engine status

**Documentation:**
- `GET /swagger/index.html` - Interactive Swagger UI

**Performance:**
- Health checks: <500µs response time
- Workflow operations: <500µs response time
- Throughput: >2,000 req/s per endpoint
- Test coverage: 88.9% (pkg/api)

See `docs/examples/curl-examples.md` for API usage examples.
