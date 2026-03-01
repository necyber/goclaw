# Goclaw 馃

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white" alt="Go Version">
  <img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License">
  <img src="https://img.shields.io/github/actions/workflow/status/goclaw/goclaw/ci.yml?branch=main" alt="CI">
  <img src="https://img.shields.io/codecov/c/github/goclaw/goclaw" alt="Coverage">
</p>

<p align="center">
  <a href="#english">English</a> | 
  <a href="#chinese">涓枃</a>
</p>

---

<a name="english"></a>
## 馃専 Overview

**Goclaw** is a production-grade, high-performance, distributed-ready multi-Agent orchestration engine written in Go.

It provides a robust framework for building, deploying, and managing intelligent agents that can collaborate seamlessly in distributed environments.

### Key Features

- 馃殌 **High Performance** - Built with Go's concurrency model for maximum throughput
- 馃彈锔?**Distributed Architecture** - Native support for cluster deployment and service discovery
- 馃攧 **Agent Orchestration** - Advanced workflow management and task scheduling
- 馃寪 **RESTful API** - Complete HTTP API with Swagger documentation
- 馃搳 **Observability** - Built-in metrics, logging, and tracing support
- 馃攲 **Extensible** - Plugin architecture for custom agent behaviors
- 馃洝锔?**Production Ready** - Comprehensive error handling and fault tolerance

### Quick Start

```bash
# Clone the repository
git clone https://github.com/goclaw/goclaw.git
cd goclaw

# Build the project
make build

# Run tests
make test

# Start the server
make run
```

### Web UI

The built-in Web UI is served by the same Go binary and is enabled by default.

- Open `http://localhost:8080/ui` after starting the server.
- Real-time workflow updates are pushed through `GET /ws/events`.
- In local frontend development, set `ui.dev_proxy` to your Vite dev server (for example `http://localhost:5173`).

**Screenshot (placeholder):**

![Web UI screenshot placeholder](docs/images/web-ui-screenshot-placeholder.svg)

### Installation

```bash
go get github.com/goclaw/goclaw
```

### Usage Example

```go
package main

import (
    "context"
    "github.com/goclaw/goclaw/pkg/engine"
    "github.com/goclaw/goclaw/config"
    "github.com/goclaw/goclaw/pkg/logger"
)

func main() {
    // Load configuration
    cfg, err := config.Load("config.yaml", nil)
    if err != nil {
        panic(err)
    }

    // Initialize logger
    log := logger.New(&logger.Config{
        Level:  logger.InfoLevel,
        Format: "json",
        Output: "stdout",
    })

    // Create orchestration engine
    eng, err := engine.New(cfg, log)
    if err != nil {
        panic(err)
    }

    // Start the engine
    ctx := context.Background()
    if err := eng.Start(ctx); err != nil {
        panic(err)
    }
    defer eng.Stop(ctx)

    // HTTP API server is now running on port 8080
    // Access API docs at http://localhost:8080/docs
}
```

### Configuration

Goclaw uses a flexible configuration system that supports multiple sources:

```yaml
# config.yaml
app:
  name: goclaw
  environment: production

server:
  host: 0.0.0.0
  port: 8080

storage:
  type: badger  # or "memory" for testing
  badger:
    path: ./data/badger
    sync_writes: true
    value_log_file_size: 1073741824  # 1GB

metrics:
  enabled: true
  port: 9091
  path: /metrics

tracing:
  enabled: false
  exporter: otlpgrpc
  endpoint: "localhost:4317"
  sampler: parentbased_traceidratio
  sample_rate: 0.1

  # optional legacy compatibility alias:
  # type: jaeger|zipkin (maps to exporter=otlpgrpc)

orchestration:
  max_agents: 10
  queue:
    type: memory
    size: 1000
```

**Storage Options:**
- `memory` - In-memory storage (for development/testing)
- `badger` - Persistent embedded database (for production)

**Metrics Configuration:**
- `enabled` - Enable/disable Prometheus metrics collection
- `port` - Metrics server port (default: 9091)
- `path` - Metrics endpoint path (default: /metrics)

**Tracing Configuration:**
- `tracing.enabled` - Enable/disable OpenTelemetry tracing lifecycle and middleware/interceptors
- `tracing.exporter` - Exporter backend (`otlpgrpc`)
- `tracing.endpoint` - OTLP collector endpoint (for example `localhost:4317`)
- `tracing.sampler` / `tracing.sample_rate` - Sampling policy
- `server.grpc.enable_tracing` - Enables gRPC tracing interceptors (effective only when `tracing.enabled=true`)

**Environment Variables:**
All config values can be overridden with `GOCLAW_` prefix:
```bash
export GOCLAW_SERVER_PORT=9090
export GOCLAW_STORAGE_TYPE=badger
```

For a complete configuration example, see [config/config.example.yaml](config/config.example.yaml).

### HTTP API

Goclaw provides a complete RESTful API for workflow management:

#### API Endpoints

**Workflow Management:**
- `POST /api/v1/workflows` - Submit a new workflow
- `GET /api/v1/workflows` - List all workflows (with pagination)
- `GET /api/v1/workflows/{id}` - Get workflow status
- `POST /api/v1/workflows/{id}/cancel` - Cancel a workflow
- `GET /api/v1/workflows/{id}/tasks/{tid}/result` - Get task result

**Saga Management:**
- `POST /api/v1/sagas` - Submit a saga
- `GET /api/v1/sagas` - List sagas (with state filter and pagination)
- `GET /api/v1/sagas/{id}` - Get saga status
- `POST /api/v1/sagas/{id}/compensate` - Trigger manual compensation
- `POST /api/v1/sagas/{id}/recover` - Recover from latest checkpoint

**Health Checks:**
- `GET /health` - Liveness probe
- `GET /ready` - Readiness probe
- `GET /status` - Detailed status information

**Metrics:**
- `GET /metrics` - Prometheus metrics endpoint (port 9091)

**Documentation:**
- `GET /docs` - Interactive API documentation (primary)
- `GET /swagger/index.html` - Interactive API documentation (compatibility alias)

### gRPC API

Goclaw also provides a high-performance gRPC API (default port: 9090):

#### Services

**WorkflowService** - Core workflow operations
- `SubmitWorkflow` - Submit new workflows
- `ListWorkflows` - List workflows with pagination
- `GetWorkflowStatus` - Get detailed workflow status
- `CancelWorkflow` - Cancel running workflows
- `GetTaskResult` - Retrieve task results

**StreamingService** - Real-time updates
- `WatchWorkflow` - Stream workflow state changes
- `WatchTasks` - Stream task execution events
- `StreamLogs` - Bidirectional log streaming

**BatchService** - Bulk operations
- `SubmitWorkflows` - Submit multiple workflows in parallel
- `GetWorkflowStatuses` - Get statuses for multiple workflows
- `CancelWorkflows` - Cancel multiple workflows
- `GetTaskResults` - Get results for multiple tasks

**AdminService** - Administrative operations
- `GetEngineStatus` - Engine health and metrics
- `UpdateConfig` - Dynamic configuration updates
- `ManageCluster` - Cluster node management
- `PauseWorkflows` / `ResumeWorkflows` - Workflow control
- `PurgeWorkflows` - Clean up old workflows
- `GetLaneStats` - Lane queue statistics
- `ExportMetrics` - Export metrics in various formats
- `GetDebugInfo` - Runtime profiling data

#### Features

- **TLS/mTLS Support** - Secure communication with certificate-based authentication
- **Server Reflection** - Dynamic service discovery for tools like grpcurl
- **Health Checks** - Standard gRPC health check protocol
- **Interceptors** - Authentication, rate limiting, logging, metrics, tracing
- **Connection Pooling** - Efficient connection management
- **Automatic Retry** - Built-in retry logic with exponential backoff

#### Go Client SDK

```go
import "github.com/goclaw/goclaw/pkg/grpc/client"

// Create client
c, err := client.NewClient("localhost:9090",
    client.WithTimeout(30*time.Second),
    client.WithTLS("./certs/ca.crt", "", ""),
)
defer c.Close()

// Submit workflow
workflowID, err := c.SubmitWorkflow(ctx, "my-workflow", tasks)

// Watch workflow progress
eventChan, errChan, err := c.WatchWorkflow(ctx, workflowID, 0)
```

For detailed examples, see:
- [gRPC API Examples](docs/examples/grpc-examples.md)
- [Client SDK Examples](docs/examples/client-sdk-examples.md)
- [TLS/mTLS Setup](docs/examples/tls-setup.md)

#### Quick API Example

```bash
# Submit a workflow
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "data-processing",
    "description": "Process customer data",
    "tasks": [
      {
        "id": "task-1",
        "name": "Fetch data",
        "type": "http"
      },
      {
        "id": "task-2",
        "name": "Process data",
        "type": "script",
        "dependencies": ["task-1"]
      }
    ]
  }'

# Get workflow status
curl http://localhost:8080/api/v1/workflows/{workflow-id}

# List all workflows
curl http://localhost:8080/api/v1/workflows?limit=50&offset=0
```

For more examples, see [docs/examples/curl-examples.md](docs/examples/curl-examples.md).
For compatibility and deprecation guidance, see [docs/api-compatibility.md](docs/api-compatibility.md).

#### API Compatibility Notes

- Canonical workflow identifier field in API responses is `workflow_id` (legacy `id` is still included during compatibility window).
- Canonical task dependency field is `dependencies`; `depends_on` is accepted as a legacy request alias.
- `GET /api/v1/workflows/{id}/tasks/{tid}/result` returns `409` until the task reaches a terminal state.

### Monitoring and Observability

Goclaw provides production-grade monitoring with Prometheus metrics:

For tracing operations and configuration:
- [OpenTelemetry Tracing Guide](docs/opentelemetry-tracing-guide.md)
- [Tracing Architecture](docs/tracing-architecture.md)
- [Tracing Runbook](docs/tracing-runbook.md)

#### Metrics Endpoint

```bash
# Access metrics
curl http://localhost:9091/metrics
```

#### Available Metrics

**Workflow Metrics:**
- `workflow_submissions_total` - Total workflow submissions by status
- `workflow_duration_seconds` - Workflow execution duration histogram
- `workflow_active_count` - Current active workflows by status

**Saga Metrics:**
- `saga_executions_total` - Total saga executions by status
- `saga_duration_seconds` - Saga execution duration histogram
- `saga_active_count` - Current active saga executions
- `saga_compensations_total` - Compensation phases by status
- `saga_compensation_duration_seconds` - Compensation phase duration histogram
- `saga_compensation_retries_total` - Compensation retry count
- `saga_recovery_total` - Recovery attempts by status

**Task Metrics:**
- `task_executions_total` - Total task executions by status
- `task_duration_seconds` - Task execution duration histogram
- `task_retries_total` - Total task retry attempts

**Lane Queue Metrics:**
- `lane_queue_depth` - Current queue depth by lane
- `lane_wait_duration_seconds` - Task wait time in queue histogram
- `lane_throughput_total` - Total tasks processed by lane

**HTTP API Metrics:**
- `http_requests_total` - Total HTTP requests by method/path/status
- `http_request_duration_seconds` - HTTP request latency histogram
- `http_active_connections` - Current active HTTP connections

**System Metrics:**
- `go_goroutines` - Number of goroutines
- `go_memstats_alloc_bytes` - Memory allocated
- `process_cpu_seconds_total` - CPU time
- `process_open_fds` - Open file descriptors

#### Docker Compose with Monitoring Stack

```bash
# Start Goclaw with Prometheus and Grafana
docker-compose up -d

# Access services
# - Goclaw API: http://localhost:8080
# - Metrics: http://localhost:9091/metrics
# - Prometheus: http://localhost:9092
# - Grafana: http://localhost:3000 (admin/admin)
```

The monitoring stack includes:
- **Prometheus** - Metrics collection and storage
- **Grafana** - Visualization dashboards
- **Alert Rules** - Pre-configured alerts for failures, latency, and resource usage

For detailed monitoring setup, see [config/prometheus.yml](config/prometheus.yml) and [config/grafana/](config/grafana/).

### Distributed Lane and Signal Bus

Goclaw supports Redis-backed queueing and signal delivery for distributed deployment.

#### Distributed Runtime Config

```yaml
orchestration:
  queue:
    type: redis

redis:
  enabled: true
  address: "localhost:6379"

signal:
  mode: redis
  channel_prefix: "goclaw:signal:"
```

When Redis is unavailable, startup falls back automatically to local mode and reports:

- effective queue mode (`redis` or `memory(fallback)`)
- effective signal mode (`redis` or `local(fallback)`)
- redis connection status (`redis_connected`)

See [docs/distributed-lane-guide.md](docs/distributed-lane-guide.md) for configuration details, signal patterns (steer/interrupt/collect), and deployment steps.

### Saga Distributed Transactions

GoClaw includes orchestration-based Saga support for eventual consistency across multi-step workflows.

Highlights:
- Declarative Saga DSL with dependency validation
- Auto/manual/skip compensation policies
- WAL + checkpoint recovery on restart
- HTTP + gRPC Saga management APIs

See [docs/saga-guide.md](docs/saga-guide.md) for usage, configuration, troubleshooting, and metrics.

### Hybrid Memory System

Goclaw includes a hybrid memory system for intelligent agent memory management, combining vector-based semantic search, BM25 full-text retrieval, and FSRS-6 spaced-repetition decay.

#### Architecture

- **Tiered Storage** 鈥?L1 LRU cache + L2 Badger persistence
- **Vector Index** 鈥?Cosine similarity search over embedding vectors
- **BM25 Index** 鈥?Full-text search with TF-IDF scoring (CJK support)
- **Hybrid Retriever** 鈥?Reciprocal Rank Fusion (RRF) combining both indexes
- **FSRS-6 Decay** 鈥?Automatic memory strength decay with spaced repetition

#### Memory API Endpoints

- `POST /api/v1/memory/{sessionID}` - Store a memory entry
- `GET /api/v1/memory/{sessionID}` - Query memories (text/vector/hybrid)
- `DELETE /api/v1/memory/{sessionID}` - Delete specific entries
- `GET /api/v1/memory/{sessionID}/list` - List entries (paginated)
- `GET /api/v1/memory/{sessionID}/stats` - Get session statistics
- `DELETE /api/v1/memory/{sessionID}/all` - Delete entire session
- `DELETE /api/v1/memory/{sessionID}/weak` - Delete weak memories

#### Quick Example

```bash
# Store a memory
curl -X POST http://localhost:8080/api/v1/memory/session-1 \
  -H "Content-Type: application/json" \
  -d '{"content": "Go is a compiled language", "metadata": {"type": "fact"}}'

# Query memories
curl "http://localhost:8080/api/v1/memory/session-1?query=compiled+language&limit=5"
```

#### Configuration

```yaml
memory:
  enabled: true
  vector_dimension: 768
  vector_weight: 0.7
  bm25_weight: 0.3
  l1_cache_size: 1000
  forget_threshold: 0.1
  decay_interval: 1h
  default_stability: 24.0
  storage_path: "./data/memory"
```

For detailed documentation, see [docs/memory-system-guide.md](docs/memory-system-guide.md).

---

<a name="chinese"></a>
## 馃専 椤圭洰绠€浠?

**Goclaw** 鏄竴涓熀浜?Go 璇█鏋勫缓鐨勭敓浜х骇銆侀珮鎬ц兘銆佸垎甯冨紡澶?Agent 缂栨帓寮曟搸銆?

瀹冩彁渚涗簡涓€涓仴澹殑妗嗘灦锛岀敤浜庢瀯寤恒€侀儴缃插拰绠＄悊鑳藉鍦ㄥ垎甯冨紡鐜涓棤缂濆崗浣滅殑鏅鸿兘浠ｇ悊銆?

### 鏍稿績鐗规€?

- 馃殌 **楂樻€ц兘** - 鍩轰簬 Go 鐨勫苟鍙戞ā鍨嬶紝瀹炵幇鏈€澶у悶鍚愰噺
- 馃彈锔?**鍒嗗竷寮忔灦鏋?* - 鍘熺敓鏀寔闆嗙兢閮ㄧ讲鍜屾湇鍔″彂鐜?
- 馃攧 **Agent 缂栨帓** - 楂樼骇宸ヤ綔娴佺鐞嗗拰浠诲姟璋冨害
- 馃寪 **RESTful API** - 瀹屾暣鐨?HTTP API 鍜?Swagger 鏂囨。
- 馃搳 **鍙娴嬫€?* - 鍐呯疆鎸囨爣銆佹棩蹇楀拰閾捐矾杩借釜鏀寔
- 馃攲 **鍙墿灞?* - 鎻掍欢鍖栨灦鏋勶紝鏀寔鑷畾涔?Agent 琛屼负
- 馃洝锔?**鐢熶骇灏辩华** - 瀹屽杽鐨勯敊璇鐞嗗拰瀹归敊鏈哄埗

### 蹇€熷紑濮?

```bash
# 鍏嬮殕浠撳簱
git clone https://github.com/goclaw/goclaw.git
cd goclaw

# 鏋勫缓椤圭洰
make build

# 杩愯娴嬭瘯
make test

# 鍚姩鏈嶅姟
make run
```

### 瀹夎

```bash
go get github.com/goclaw/goclaw
```

### 浣跨敤绀轰緥

```go
package main

import (
    "context"
    "github.com/goclaw/goclaw/pkg/engine"
    "github.com/goclaw/goclaw/config"
    "github.com/goclaw/goclaw/pkg/logger"
)

func main() {
    // 鍔犺浇閰嶇疆
    cfg, err := config.Load("config.yaml", nil)
    if err != nil {
        panic(err)
    }

    // 鍒濆鍖栨棩蹇?
    log := logger.New(&logger.Config{
        Level:  logger.InfoLevel,
        Format: "json",
        Output: "stdout",
    })

    // 鍒涘缓缂栨帓寮曟搸
    eng, err := engine.New(cfg, log)
    if err != nil {
        panic(err)
    }

    // 鍚姩寮曟搸
    ctx := context.Background()
    if err := eng.Start(ctx); err != nil {
        panic(err)
    }
    defer eng.Stop(ctx)

    // HTTP API 鏈嶅姟鍣ㄧ幇鍦ㄨ繍琛屽湪 8080 绔彛
    // 璁块棶 API 文档: http://localhost:8080/docs
}
```

### HTTP API

Goclaw 鎻愪緵瀹屾暣鐨?RESTful API 鐢ㄤ簬宸ヤ綔娴佺鐞嗭細

#### API 绔偣

**宸ヤ綔娴佺鐞嗭細**
- `POST /api/v1/workflows` - 鎻愪氦鏂板伐浣滄祦
- `GET /api/v1/workflows` - 鍒楀嚭鎵€鏈夊伐浣滄祦锛堟敮鎸佸垎椤碉級
- `GET /api/v1/workflows/{id}` - 鑾峰彇宸ヤ綔娴佺姸鎬?
- `POST /api/v1/workflows/{id}/cancel` - 鍙栨秷宸ヤ綔娴?
- `GET /api/v1/workflows/{id}/tasks/{tid}/result` - 鑾峰彇浠诲姟缁撴灉

**鍋ュ悍妫€鏌ワ細**
- `GET /health` - 瀛樻椿鎺㈤拡
- `GET /ready` - 灏辩华鎺㈤拡
- `GET /status` - 璇︾粏鐘舵€佷俊鎭?

**鎸囨爣鐩戞帶锛?*
- `GET /metrics` - Prometheus 鎸囨爣绔偣锛堢鍙?9091锛?

**鏂囨。锛?*
- `GET /swagger/index.html` - 浜や簰寮?API 鏂囨。

#### 蹇€?API 绀轰緥

```bash
# 鎻愪氦宸ヤ綔娴?
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "鏁版嵁澶勭悊",
    "description": "澶勭悊瀹㈡埛鏁版嵁",
    "tasks": [
      {
        "id": "task-1",
        "name": "鑾峰彇鏁版嵁",
        "type": "http"
      },
      {
        "id": "task-2",
        "name": "澶勭悊鏁版嵁",
        "type": "script",
        "dependencies": ["task-1"]
      }
    ]
  }'

# 鑾峰彇宸ヤ綔娴佺姸鎬?
curl http://localhost:8080/api/v1/workflows/{workflow-id}

# 鍒楀嚭鎵€鏈夊伐浣滄祦
curl http://localhost:8080/api/v1/workflows?limit=50&offset=0
```

鏇村绀轰緥璇峰弬瑙?[docs/examples/curl-examples.md](docs/examples/curl-examples.md)銆?

### 鐩戞帶涓庡彲瑙傛祴鎬?

Goclaw 鎻愪緵鐢熶骇绾х殑 Prometheus 鎸囨爣鐩戞帶锛?

#### 鎸囨爣绔偣

```bash
# 璁块棶鎸囨爣
curl http://localhost:9091/metrics
```

#### 鍙敤鎸囨爣

**宸ヤ綔娴佹寚鏍囷細**
- `workflow_submissions_total` - 鎸夌姸鎬佺粺璁＄殑宸ヤ綔娴佹彁浜ゆ€绘暟
- `workflow_duration_seconds` - 宸ヤ綔娴佹墽琛屾椂闀跨洿鏂瑰浘
- `workflow_active_count` - 鎸夌姸鎬佺粺璁＄殑褰撳墠娲昏穬宸ヤ綔娴佹暟

**浠诲姟鎸囨爣锛?*
- `task_executions_total` - 鎸夌姸鎬佺粺璁＄殑浠诲姟鎵ц鎬绘暟
- `task_duration_seconds` - 浠诲姟鎵ц鏃堕暱鐩存柟鍥?
- `task_retries_total` - 浠诲姟閲嶈瘯鎬绘鏁?

**闃熷垪鎸囨爣锛?*
- `lane_queue_depth` - 鎸?lane 缁熻鐨勫綋鍓嶉槦鍒楁繁搴?
- `lane_wait_duration_seconds` - 浠诲姟鍦ㄩ槦鍒椾腑鐨勭瓑寰呮椂闀跨洿鏂瑰浘
- `lane_throughput_total` - 鎸?lane 缁熻鐨勫凡澶勭悊浠诲姟鎬绘暟

**HTTP API 鎸囨爣锛?*
- `http_requests_total` - 鎸夋柟娉?璺緞/鐘舵€佺粺璁＄殑 HTTP 璇锋眰鎬绘暟
- `http_request_duration_seconds` - HTTP 璇锋眰寤惰繜鐩存柟鍥?
- `http_active_connections` - 褰撳墠娲昏穬 HTTP 杩炴帴鏁?

**绯荤粺鎸囨爣锛?*
- `go_goroutines` - Goroutine 鏁伴噺
- `go_memstats_alloc_bytes` - 宸插垎閰嶅唴瀛?
- `process_cpu_seconds_total` - CPU 鏃堕棿
- `process_open_fds` - 鎵撳紑鐨勬枃浠舵弿杩扮鏁?

#### Docker Compose 鐩戞帶鏍?

```bash
# 鍚姩 Goclaw 鍙?Prometheus 鍜?Grafana
docker-compose up -d

# 璁块棶鏈嶅姟
# - Goclaw API: http://localhost:8080
# - 鎸囨爣绔偣: http://localhost:9091/metrics
# - Prometheus: http://localhost:9092
# - Grafana: http://localhost:3000 (admin/admin)
```

鐩戞帶鏍堝寘鎷細
- **Prometheus** - 鎸囨爣鏀堕泦鍜屽瓨鍌?
- **Grafana** - 鍙鍖栦华琛ㄦ澘
- **鍛婅瑙勫垯** - 棰勯厤缃殑澶辫触銆佸欢杩熷拰璧勬簮浣跨敤鍛婅

璇︾粏鐨勭洃鎺ч厤缃鍙傝 [config/prometheus.yml](config/prometheus.yml) 鍜?[config/grafana/](config/grafana/)銆?

### 娣峰悎璁板繂绯荤粺

Goclaw 鍐呯疆娣峰悎璁板繂绯荤粺锛岀粨鍚堝悜閲忚涔夋悳绱€丅M25 鍏ㄦ枃妫€绱㈠拰 FSRS-6 闂撮殧閲嶅琛板噺绠楁硶锛屼负 Agent 鎻愪緵鏅鸿兘璁板繂绠＄悊銆?

#### 鏋舵瀯

- **鍒嗗眰瀛樺偍** 鈥?L1 LRU 缂撳瓨 + L2 Badger 鎸佷箙鍖?
- **鍚戦噺绱㈠紩** 鈥?鍩轰簬浣欏鸡鐩镐技搴︾殑宓屽叆鍚戦噺鎼滅储
- **BM25 绱㈠紩** 鈥?鏀寔涓嫳鏂囩殑鍏ㄦ枃妫€绱?
- **娣峰悎妫€绱?* 鈥?RRF (Reciprocal Rank Fusion) 铻嶅悎涓ょ妫€绱㈢粨鏋?
- **FSRS-6 琛板噺** 鈥?鍩轰簬闂撮殧閲嶅鐨勮嚜鍔ㄨ蹇嗗己搴﹁“鍑?

#### 璁板繂 API 绔偣

- `POST /api/v1/memory/{sessionID}` - 瀛樺偍璁板繂
- `GET /api/v1/memory/{sessionID}` - 鏌ヨ璁板繂锛堟枃鏈?鍚戦噺/娣峰悎锛?
- `DELETE /api/v1/memory/{sessionID}` - 鍒犻櫎鎸囧畾璁板繂
- `GET /api/v1/memory/{sessionID}/list` - 鍒楀嚭璁板繂锛堝垎椤碉級
- `GET /api/v1/memory/{sessionID}/stats` - 鑾峰彇浼氳瘽缁熻
- `DELETE /api/v1/memory/{sessionID}/all` - 鍒犻櫎鏁翠釜浼氳瘽
- `DELETE /api/v1/memory/{sessionID}/weak` - 鍒犻櫎寮辫蹇?

#### 蹇€熺ず渚?

```bash
# 瀛樺偍璁板繂
curl -X POST http://localhost:8080/api/v1/memory/session-1 \
  -H "Content-Type: application/json" \
  -d '{"content": "Go 鏄紪璇戝瀷璇█", "metadata": {"type": "fact"}}'

# 鏌ヨ璁板繂
curl "http://localhost:8080/api/v1/memory/session-1?query=缂栬瘧鍨嬭瑷€&limit=5"
```

璇︾粏鏂囨。璇峰弬瑙?[docs/memory-system-guide.md](docs/memory-system-guide.md)銆?

---

## 馃摎 Documentation

- [Current Canonical Specs (OpenSpec)](openspec/specs/)
- [Implementation Status Snapshot](docs/STATUS.md)
- [Historical Specification v0.2 (English)](docs/SPEC_en_v0.2.md)
- [OpenTelemetry Tracing Guide](docs/opentelemetry-tracing-guide.md)
- [Tracing Architecture](docs/tracing-architecture.md)
- [Tracing Runbook](docs/tracing-runbook.md)
- [Historical Specification v0.2 (Chinese)](docs/SPEC_zh_v0.2.md)

## 馃 Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details.

## 馃搫 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

