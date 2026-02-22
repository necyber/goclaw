# GoClaw - AI Agent Project Guide

## Project Overview

**GoClaw** is a declarative multi-agent orchestration engine implemented in Go. Its core purpose is to solve the problem of "how multiple AI Agents collaborate" through declarative dependency definition, fine-grained concurrency control, and intelligent context sharing.

### Key Characteristics

- **Language**: Go (Golang)
- **License**: MIT License (Copyright 2026 cosimqq)
- **Current Phase**: Specification/design phase (no implementation code yet)
- **Project Status**: Early stage - architecture and API design defined, awaiting implementation

### Vision & Goals

- Support 100k+ concurrent Agent instances with < 1ms task startup latency
- Distributed transactions, checkpoint resume, and graceful degradation
- Plugin-based Agent registration with pluggable storage backends
- Type safety with generics, declarative APIs, and hot-reload workflows

### Non-Goals

- Does NOT implement LLM clients (orchestration only, not model encapsulation)
- NOT a generic workflow engine (focuses on AI Agent scenarios with memory and message patterns)
- NO support for circular dependencies (maintains DAG purity)

---

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    API Gateway (gRPC/HTTP)                   │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│                 Control Plane (etcd/Consul)                  │
└──────────────┬─────────────────────────────┬────────────────┘
               │                             │
┌──────────────▼──────────┐    ┌─────────────▼──────────┐
│   GoClaw Engine Node 1  │◄──►│   GoClaw Engine Node 2 │
│  ┌──────────┬────────┐  │    │  ┌──────────┬────────┐  │
│  │Scheduler │Lane Mgr│  │    │  │Scheduler │Lane Mgr│  │
│  └────┬─────┴───┬────┘  │    │  └────┬─────┴───┬────┘  │
│       │         │       │    │       │         │       │
│  ┌────▼────┬────▼────┐  │    │  ┌────▼────┬────▼────┐  │
│  │  State  │ Memory  │  │    │  │  State  │ Memory  │  │
│  │ (Badger)│ (Hybrid)│  │    │  │ (Badger)│ (Hybrid)│  │
│  └─────────┴─────────┘  │    │  └─────────┴─────────┘  │
└─────────────────────────┘    └─────────────────────────┘
```

### Core Components

| Component | Responsibility | Key Interface | Concurrency Model |
|-----------|---------------|---------------|-------------------|
| **Scheduler** | DAG parsing, topological sorting, dependency injection | `BuildPlan()` `ExecuteLayer()` | Stateless, read-only graph structure |
| **Lane Manager** | Resource isolation, flow control, backpressure | `Submit()` `Acquire()` `Release()` | One Channel + Worker Pool per Lane |
| **Memory Hub** | Hybrid retrieval, memory decay, context assembly | `Retrieve()` `Memorize()` | Background goroutine for vectorization |
| **Agent Runtime** | Execute user functions, timeout control, error handling | `Invoke()` `Cancel()` | One goroutine per task |

---

## Technology Stack

### Core Dependencies (Planned)

- **Orchestration Core**: Pure Go standard library (`sync`, `context`, `container/heap`)
- **Storage Layer**:
  - Local: `badger` (LSM-Tree, high-performance KV)
  - Distributed: `etcd` (consensus state) + `Redis` (cache/queue)
- **Communication**:
  - Internal: `NATS` (lightweight message bus)
  - API: `gRPC` + `connectrpc` (dual protocol support)
- **Vectorization**: `weaviate` client or local `faiss` bindings
- **Observability**: `opentelemetry` + `prometheus`

### Project Structure (Planned)

```
goclaw/
├── api/              # Protobuf definitions + generated code
├── pkg/
│   ├── dag/          # DAG compiler (topological sorting, cycle detection)
│   ├── lane/         # Lane Queue implementation (Channel management)
│   ├── memory/       # Hybrid retrieval + FSRS-6 algorithm
│   └── runtime/      # Agent execution sandbox
├── internal/
│   ├── store/        # Storage abstraction (interface definitions)
│   └── server/       # HTTP/gRPC service implementation
├── cmd/
│   └── goclaw/       # Main entry point
└── docs/
    ├── SPEC_en_v0.2.md   # English specification
    └── SPEC_zh_v0.2.md   # Chinese specification
```

---

## Development Roadmap

### Phase 1: MVP (Week 1-4)
- [ ] Basic DAG compiler (topological sorting)
- [ ] In-memory Lane Queue (Channel implementation)
- [ ] In-memory storage (Map + RWMutex)
- [ ] Synchronous execution engine (single node)

### Phase 2: Core Features (Week 5-8)
- [ ] Persistent storage (Badger integration)
- [ ] Hybrid Memory (vector retrieval + BM25)
- [ ] Distributed Lane (Redis queue)
- [ ] Message patterns (steer/interrupt/collect)

### Phase 3: Production Ready (Week 9-12)
- [ ] gRPC API + streaming events
- [ ] Monitoring metrics (Prometheus)
- [ ] Distributed transactions (Saga pattern)
- [ ] Web UI workflow visualization

---

## Build and Development

### Prerequisites

- Go 1.21+ (required for generics support)
- Protocol Buffers compiler (`protoc`)
- Optional: Docker (for dependencies like Badger, Redis, etcd)

### Planned Build Commands

```bash
# Build the project
go build ./cmd/goclaw

# Run tests
go test ./...

# Generate protobuf code
protoc --go_out=. --go-grpc_out=. api/*.proto

# Run with hot reload (when implemented)
go run ./cmd/goclaw --config=config.yaml
```

### Configuration

The project will support YAML/JSON configuration files:

```yaml
name: customer_support
tasks:
  - id: intake
    agent: IntakeAgent
    lane: io

  - id: classify
    agent: ClassifyAgent
    deps: [intake]
    lane: cpu
```

---

## API Design

### Go SDK (Programmatic API)

```go
// Workflow definition (compile time)
wf := goclaw.NewWorkflow("data_pipeline").
    WithTask("fetch", FetchAgent{}, 
        goclaw.WithLane("io"),
        goclaw.WithTimeout(5*time.Second)).
    WithTask("analyze", AnalyzeAgent{},
        goclaw.WithDeps("fetch"),
        goclaw.WithLane("cpu"))

// Execution (runtime)
engine := goclaw.NewEngine(
    goclaw.WithStore(badgerStore),
    goclaw.WithLaneLimit("cpu", 8),
)
result, err := engine.Execute(ctx, wf, initialInput)
```

### gRPC Service API

```protobuf
service GoClawEngine {
  rpc SubmitWorkflow(WorkflowSpec) returns (stream TaskEvent);
  rpc GetTaskStatus(TaskID) returns (TaskStatus);
  rpc SignalTask(SignalRequest) returns (Empty); // For steer/interrupt
}
```

---

## Code Style Guidelines

### Go Conventions

- Follow [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use Go generics for type safety where appropriate
- Use `any` for Agent input/output in early phases, then migrate to generics

### Naming Conventions

- Interfaces should be named with descriptive suffixes (e.g., `Store`, `Scheduler`)
- Concrete implementations should include the technology name (e.g., `BadgerStore`)
- Acronyms should be all caps (e.g., `DAG`, `HTTP`, `gRPC`)

### Documentation

- All exported functions must have GoDoc comments
- Comments should be in English
- Complex algorithms should reference papers/implementations

---

## Testing Strategy

### Planned Test Structure

```
*_test.go          # Unit tests
*_integration_test.go  # Integration tests (require external deps)
```

### Testing Principles

- Unit tests should not require external dependencies
- Integration tests should use Docker Compose for dependencies
- Benchmark tests for performance-critical paths (DAG compilation, Lane scheduling)
- Fuzz testing for input validation

### Key Test Scenarios

- DAG cycle detection
- Lane backpressure handling
- Memory retrieval and decay
- Task state machine transitions
- Distributed consensus (Phase 2+)

---

## Non-Functional Requirements

### Performance Metrics

- **Latency**: P99 task startup latency < 5ms (local storage)
- **Throughput**: Single node 10k TPS (simple tasks)
- **Memory**: Support 1 million pending tasks without OOM

### Reliability

- **Fault Tolerance**: Agent panic does not crash engine, auto-retry 3 times
- **Persistence**: Task state changes immediately write to WAL (Write-Ahead Log)
- **Graceful Shutdown**: Wait for in-progress tasks on SIGTERM (configurable timeout)

### Observability

- **Tracing**: Automatic Trace ID injection per task, distributed tracing support
- **Metrics**: Prometheus metrics (`goclaw_tasks_total`, `goclaw_lane_wait_duration`)
- **Logging**: Structured logging with `zap`, context injection support

---

## Security Considerations

### Input Validation

- All user-provided Agent code runs in a sandbox (planned)
- Task timeout and resource limits enforced by runtime
- No direct execution of user input as code

### Dependencies

- Regular vulnerability scanning with `govulncheck`
- Pin dependency versions in `go.mod`
- Prefer well-maintained, widely-used libraries

### Data Protection

- Sensitive data in memory should be encrypted at rest (Phase 2+)
- Session isolation enforced by Memory Hub
- No logging of sensitive task content

---

## Key Data Structures

### Task State Machine

```go
type TaskState int

const (
    StatePending TaskState = iota
    StateScheduled    // Enqueued in Lane
    StateRunning      // Currently executing
    StateCompleted
    StateFailed
    StateCancelled
    StateRetrying     // Auto-retrying
)
```

### Memory Entry Structure

```go
type MemoryEntry struct {
    ID         ulid.ULID         // Sortable ID
    TaskID     string
    SessionID  string            // Isolate different users/sessions
    Content    []byte            // Serialized content
    Vector     []float32         // Vector embedding
    Metadata   map[string]string
    Strength   float64           // FSRS-6 memory strength [0,1]
    LastReview time.Time
}
```

### Execution Plan

```go
type ExecutionPlan struct {
    Layers         [][]TaskID   // Layered results
    HotPath        []TaskID     // Critical path (for optimization)
    ParallelGroups []TaskGroup  // Parallelizable groups
}
```

---

## References

- **Inspiration**: OpenClaw (TypeScript), Temporal, Cadence, Prefect
- **Algorithm References**: FSRS-6 Paper, HNSW Paper, Kahn's Algorithm
- **Go Best Practices**: Uber Go Style Guide, Go Code Review Comments

---

## Contributing Guidelines

1. All code changes should align with the current phase of the roadmap
2. Follow the planned architecture and component boundaries
3. Maintain compatibility with the specification documents
4. Update documentation when adding new features
5. Ensure tests pass before submitting changes
