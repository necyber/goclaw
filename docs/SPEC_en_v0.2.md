# GoClaw Specification v0.2

**Declarative Multi-Agent Orchestration Engine (Go Implementation)**

> [!IMPORTANT]
> This document is a historical snapshot for version `v0.2` and is now frozen.
> It is no longer the source of truth for current capabilities.
> For current canonical specifications, see `openspec/specs/*`.
> For implementation progress, see `docs/STATUS.md`.
> Status note updated on `2026-02-27`.

## 1. Overview

### 1.1 Vision

Build a **production-grade, high-performance, distributed-ready** multi-Agent orchestration engine.

**Its essence is to solve the problem of "how multiple Agents collaborate"**—through declarative dependency definition, fine-grained concurrency control, and intelligent context sharing, enabling independent AI Agents to work together in an orderly, controllable, and memory-assisted manner.

### 1.2 Goals

- **Performance**: Support 100k+ concurrent Agent instances with < 1ms task startup latency
- **Reliability**: Support distributed transactions, checkpoint resume, and graceful degradation
- **Extensibility**: Plugin-based Agent registration and pluggable storage backends
- **Developer Experience**: Type safety (generics), declarative APIs, and hot-reload workflows

### 1.3 Non-Goals

- Implement LLM clients (we orchestrate only, do not encapsulate model calls)
- Generic workflow engine (focus on AI Agent scenarios with memory and message patterns)
- Support for circular dependencies (keep DAG purity)

------

## 2. Architecture Design

### 2.1 High-Level Architecture

plain

复制

```plain
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

### 2.2 Core Component Boundaries

**Based on the essence of "solving multi-Agent collaboration", the system is divided into three core layers:**

表格

复制

| Component              | Responsibility                                          | Key Interface                  | Concurrency Model                      |
| :--------------------- | :------------------------------------------------------ | :----------------------------- | :------------------------------------- |
| **Scheduler**<br/>     | DAG parsing, topological sorting, dependency injection  | `BuildPlan() ExecuteLayer()`   | Stateless, read-only graph structure   |
| **Lane Manager**<br/>  | Resource isolation, flow control, backpressure          | `Submit() Acquire() Release()` | One Channel + Worker Pool per Lane     |
| **Memory Hub**<br/>    | Hybrid retrieval, memory decay, context assembly        | `Retrieve() Memorize()`        | Background goroutine for vectorization |
| **Agent Runtime**<br/> | Execute user functions, timeout control, error handling | `Invoke() Cancel()`            | One goroutine per task                 |

------

## 3. Technology Stack

### 3.1 Core Dependencies

- **Orchestration Core**: Pure Go standard library (`sync`, `context`, `container/heap`)
- **Storage Layer**:
  - Local: `badger` (LSM-Tree, high-performance KV)
  - Distributed: `etcd` (consensus state) + `Redis` (cache/queue)
- **Communication**:
  - Internal: `NATS` (lightweight message bus)
  - API: `gRPC` + `connectrpc` (dual protocol support for gRPC and HTTP)
- **Vectorization**: `weaviate` client or local `faiss` bindings
- **Observability**: `opentelemetry` + `prometheus` (built-in metrics exposure)

### 3.2 Code Organization

plain

复制

```plain
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
    └── spec.md       # This document
```

------

## 4. Core Mechanisms Detailed Design

### 4.1 DAG Compiler (The Compiler)

**Step 1 of solving collaboration: Define who depends on whom**

- **Input**: Declarative task definitions (with dependency relationships)
- **Output**: `ExecutionPlan` (layered execution plan)
- **Key Algorithms**:
  - Kahn's algorithm (topological sorting)
  - Incremental updates (support runtime node addition)
  - Cycle detection (DFS coloring method)

go

复制

```go
// Key data structures
type ExecutionPlan struct {
    Layers [][]TaskID          // Layered results
    HotPath []TaskID           // Critical path (for optimization)
    ParallelGroups []TaskGroup // Parallelizable groups (for resource estimation)
}
```

### 4.2 Lane Scheduler (The Lane Scheduler)

**Step 2 of solving collaboration: Control who can run simultaneously**

- **Core Abstraction**: `Lane` = `Buffered Channel` + `Worker Pool` + `Rate Limiter`
- **Lane Types**:
  - `Global`: Global concurrency limit (e.g., total CPU cores)
  - `Named`: Resource classification (io/cpu/memory/gpu)
  - `Session`: User-level isolation (guarantee sequential execution for same user)
- **Backpressure Strategies**: When Lane is full, provide three strategies: `Block`, `Drop`, `Redirect`

### 4.3 Hybrid Memory System (Hybrid Memory)

**Step 3 of solving collaboration: Share context and history**

- **Storage Tiering**:
  - **L1 (Hot)**: In-process LRU Cache (recent task context)
  - **L2 (Warm)**: Badger (local persistence)
  - **L3 (Cold)**: Weaviate/Pinecone (vector database)
- **Retrieval Strategy**:
  - **Vector Retrieval**: HNSW algorithm, cosine similarity, weight 0.7
  - **BM25**: Full-text retrieval, weight 0.3
  - **Fusion**: RRF (Reciprocal Rank Fusion) ranking
- **Forgetting Mechanism**: FSRS-6 algorithm implementation, background goroutine for periodic cleanup

------

## 5. Public API Design

### 5.1 Programmatic API (Go SDK)

go

复制

```go
// Workflow definition (compile time)
wf := goclaw.NewWorkflow("data_pipeline").
    WithTask("fetch", FetchAgent{}, 
        goclaw.WithLane("io"),
        goclaw.WithTimeout(5*time.Second)).
    WithTask("analyze", AnalyzeAgent{},
        goclaw.WithDeps("fetch"),
        goclaw.WithLane("cpu")).
    WithMemoryPolicy(
        goclaw.RetrieveLastN(3),      // Retrieve last 3 memories
        goclaw.ForgetThreshold(0.3),  // Forget if strength < 0.3
    )

// Execution (runtime)
engine := goclaw.NewEngine(
    goclaw.WithStore(badgerStore),
    goclaw.WithLaneLimit("cpu", 8),
)
result, err := engine.Execute(ctx, wf, initialInput)
```

### 5.2 Declarative API (YAML/JSON)

Support workflow definition via configuration files (facilitates CI/CD and visual editors):

yaml

复制

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

  - id: resolve
    agent: ResolveAgent
    deps: [classify]
    lane: cpu
    memory:
      retrieve: 
        vector_query: "{{classify.category}}"
        top_k: 5
```

### 5.3 gRPC Service API

protobuf

复制

```protobuf
service GoClawEngine {
  rpc SubmitWorkflow(WorkflowSpec) returns (stream TaskEvent);
  rpc GetTaskStatus(TaskID) returns (TaskStatus);
  rpc SignalTask(SignalRequest) returns (Empty); // For steer/interrupt
}
```

------

## 6. Data Models

### 6.1 Task State Machine

go

复制

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

### 6.2 Memory Entry Structure

go

复制

```go
type MemoryEntry struct {
    ID        ulid.ULID     // Sortable ID
    TaskID    string
    SessionID string        // Isolate different users/sessions
    Content   []byte        // Serialized content
    Vector    []float32     // Vector embedding
    Metadata  map[string]string
    Strength  float64       // FSRS-6 memory strength [0,1]
    LastReview time.Time
}
```

------

## 7. Development Roadmap

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

------

## 8. Non-Functional Requirements (NFRs)

### 8.1 Performance Metrics

- **Latency**: P99 task startup latency < 5ms (local storage)
- **Throughput**: Single node 10k TPS (simple tasks)
- **Memory**: Support 1 million pending tasks without OOM

### 8.2 Reliability

- **Fault Tolerance**: Agent panic does not crash engine, auto-retry 3 times
- **Persistence**: Task state changes immediately write to WAL (Write-Ahead Log)
- **Graceful Shutdown**: Wait for in-progress tasks to complete upon SIGTERM (configurable timeout)

### 8.3 Observability

- **Tracing**: Automatic Trace ID injection per task, support distributed tracing
- **Metrics**: Expose `goclaw_tasks_total`, `goclaw_lane_wait_duration`, etc. Prometheus metrics
- **Logging**: Structured logging (zap), support context injection

------

## 9. Risk Assessment & Mitigation

表格

复制

| Risk                             | Impact | Mitigation                                                   |
| :------------------------------- | :----- | :----------------------------------------------------------- |
| **Go Generics Complexity**       | Medium | Use `any` for Agent input/output initially, then encapsulate generic utility functions |
| **Distributed Consistency**      | High   | Focus on single-node in Phase 1, introduce etcd for leader election in Phase 2 |
| **Vector Retrieval Performance** | Medium | Support multiple backends (local HNSW/remote Weaviate), pluggable switching |
| **Community Adoption**           | Low    | Maintain conceptual compatibility with OpenClaw, provide Python-to-Go migration guide |

------

## 10. References

- **Inspiration**: OpenClaw (TypeScript), Temporal, Cadence, Prefect
- **Algorithm References**: FSRS-6 Paper, HNSW Paper, Kahn's Algorithm
- **Go Best Practices**: Uber Go Style Guide, Go Code Review Comments
