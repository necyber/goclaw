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
- `redis_lane.go` provides Redis-backed lane execution with optional fallback

**`pkg/signal/`** — Inter-task signal bus and patterns:
- `bus.go` defines the signal bus contract
- `local_bus.go` is in-process channel delivery
- `redis_bus.go` is Redis Pub/Sub delivery for distributed mode
- `steer.go`, `interrupt.go`, `collect.go` implement the message patterns
- `message.go` defines signal payload and context injection helpers

**`pkg/engine/`** — Orchestration engine: wires DAG + Lane together; manages state (Idle → Running → Stopped/Error); provides workflow management API.

**`pkg/api/`** — HTTP API server:
- `server.go` defines HTTP server with graceful shutdown
- `router.go` sets up chi router with middleware chain
- `handlers/` contains workflow and health check handlers
- `middleware/` provides RequestID, Logger, Recovery, CORS, Timeout
- `response/` contains JSON response helpers and error formatting
- `models/` defines API request/response data structures

**`pkg/memory/`** — Hybrid memory system:
- `memory.go` defines the `Hub` interface; `hub.go` is the concrete `MemoryHub`
- `entry.go` defines `MemoryEntry`, `Query`, `RetrievalResult`, `MemoryStats`
- `vector.go` implements cosine similarity vector search
- `bm25.go` implements BM25 full-text search with CJK tokenization
- `hybrid.go` implements RRF (Reciprocal Rank Fusion) combining vector + BM25
- `storage.go` implements tiered storage (L1 LRU cache → L2 Badger)
- `fsrs.go` implements FSRS-6 memory decay with background loop

**`pkg/metrics/`** — Prometheus metrics:
- `metrics.go` defines `Manager` with registry and HTTP server (port 9091)
- `workflow.go`, `task.go`, `lane.go`, `http.go` define domain-specific metrics
- No-op mode when disabled for zero overhead

**`config/`** — Multi-source config loading via [koanf](https://github.com/knadh/koanf): defaults → YAML/JSON file → env vars (`GOCLAW_` prefix) → CLI flags. Validated with `go-playground/validator`.

**`pkg/logger/`** — Thin wrapper around Go's `log/slog` with JSON/text format and file/stdout output.

**`cmd/goclaw/main.go`** — CLI entry point: loads config, initializes logger, starts engine and HTTP server, handles graceful shutdown on SIGINT/SIGTERM.

### Distributed lane + signal integration notes

- Redis is initialized when `redis.enabled=true`, or queue/signal mode requires Redis.
- Queue mode: `orchestration.queue.type` supports `memory` and `redis`.
- Signal mode: `signal.mode` supports `local` and `redis`.
- Startup auto-falls back to local queue/signal when Redis is unavailable.
- Runtime mode is logged with `queue_type`, `signal_mode`, and `redis_connected`.

### Configuration

Copy `config/config.example.yaml` as your config file. Key sections: `app`, `server` (HTTP :8080, gRPC :9090), `log`, `orchestration`, `cluster`, `storage`, `metrics`, `tracing`, `memory`.

### Development phases

- **Phase 1:** DAG, Lane, in-memory storage, CLI, HTTP API — implemented ✅
- **Phase 2:** Persistent storage (Badger/Redis), distributed mode (Consul/etcd)
- **Phase 3:** gRPC API, Prometheus metrics, Web UI
- **Memory system:** Hybrid retrieval (vector + BM25), FSRS-6 decay, tiered storage — implemented ✅
- **Monitoring:** Prometheus metrics, Grafana dashboards, alert rules — implemented ✅

### Monitoring

Prometheus metrics on port 9091 (configurable). Workflow, task, lane, HTTP, and system metrics. Grafana dashboard in `config/grafana/`. Alert rules in `config/prometheus/alerts.yml`. See `docs/monitoring-guide.md` for full reference.

### HTTP API

The HTTP API server runs on port 8080 (configurable) and provides:

**Workflow Management:**
- `POST /api/v1/workflows` - Submit workflow
- `GET /api/v1/workflows` - List workflows (pagination supported)
- `GET /api/v1/workflows/{id}` - Get workflow status
- `POST /api/v1/workflows/{id}/cancel` - Cancel workflow
- `GET /api/v1/workflows/{id}/tasks/{tid}/result` - Get task result

**Memory API:**
- `POST /api/v1/memory/{sessionID}` - Store memory entry
- `GET /api/v1/memory/{sessionID}` - Query memories
- `DELETE /api/v1/memory/{sessionID}` - Delete entries
- `GET /api/v1/memory/{sessionID}/list` - List entries (paginated)
- `GET /api/v1/memory/{sessionID}/stats` - Session statistics
- `DELETE /api/v1/memory/{sessionID}/all` - Delete session
- `DELETE /api/v1/memory/{sessionID}/weak` - Delete weak memories

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
