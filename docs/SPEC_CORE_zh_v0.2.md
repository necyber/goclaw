# GoClaw Specification v0.2

**声明式多智能体编排引擎（Go 实现版）**

> [!IMPORTANT]
> 本文档是 `v0.2` 的历史快照，已冻结维护。
> 它不再作为当前能力的事实标准。
> 当前规范请以 `openspec/specs/*` 为准。
> 实现进度请参考 `docs/STATUS.md`。
> 状态说明更新于 `2026-02-27`。

## 1. 项目概述 (Overview)

### 1.1 愿景 (Vision)

构建一个**生产级、高性能、可分布式部署**的多 Agent 编排引擎。

**它的本质就是解决"多个 Agent 如何协作"的问题**——通过声明式依赖定义、细粒度并发控制和智能上下文共享，让独立的 AI Agent 能够有序、可控、有记忆地协同工作。

### 1.2 核心目标 (Goals)

- **性能目标**: 支持 100k+ 并发 Agent 实例，任务启动延迟 < 1ms
- **可靠性**: 支持分布式事务、断点续传、优雅降级
- **扩展性**: 插件化 Agent 注册、可插拔存储后端
- **开发者体验**: 类型安全（泛型）、声明式 API、热重载工作流

### 1.3 非目标 (Non-Goals)

- 不实现 LLM 客户端（只编排，不封装模型调用）
- 不做通用工作流（专注于 AI Agent 场景的记忆、消息模式）
- 不支持循环依赖（保持 DAG 纯粹性）

------

## 2. 架构设计 (Architecture)

### 2.1 高层架构图

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

### 2.2 核心组件边界

**基于"解决多 Agent 协作"这一本质，系统拆分为三个核心层：**

表格

复制

| 组件                   | 职责                             | 关键接口                       | 并发模型                             |
| :--------------------- | :------------------------------- | :----------------------------- | :----------------------------------- |
| **Scheduler**<br/>     | DAG 解析、拓扑排序、依赖注入     | `BuildPlan() ExecuteLayer()`   | 无状态，只读图结构                   |
| **Lane Manager**<br/>  | 资源隔离、流量控制、背压         | `Submit() Acquire() Release()` | 每个 Lane 一个 Channel + Worker Pool |
| **Memory Hub**<br/>    | 混合检索、记忆衰减、上下文组装   | `Retrieve() Memorize()`        | 后台 Goroutine 处理向量化            |
| **Agent Runtime**<br/> | 执行用户函数、超时控制、错误处理 | `Invoke() Cancel()`            | 每个任务一个 Goroutine               |

------

## 3. 技术选型 (Tech Stack)

### 3.1 核心依赖

- **编排核心**: 纯 Go 标准库 (`sync`, `context`, `container/heap`)
- **存储层**:
  - 本地: `badger` (LSM-Tree, 高性能 KV)
  - 分布式: `etcd` (一致性状态) + `Redis` (缓存/队列)
- **通信**:
  - 内部: `NATS` (轻量级消息总线)
  - API: `gRPC` + `connectrpc` (支持 gRPC 和 HTTP 双协议)
- **向量化**: `weaviate` 客户端 或本地 `faiss` bindings
- **可观测性**: `opentelemetry` + `prometheus` (内置指标暴露)

### 3.2 代码组织

plain

复制

```plain
goclaw/
├── api/              # Protobuf 定义 + 生成的代码
├── pkg/
│   ├── dag/          # DAG 编译器（拓扑排序、循环检测）
│   ├── lane/         # Lane Queue 实现（Channel 管理）
│   ├── memory/       # 混合检索 + FSRS-6 算法
│   └── runtime/      # Agent 执行沙箱
├── internal/
│   ├── store/        # 存储抽象（接口定义）
│   └── server/       # HTTP/gRPC 服务实现
├── cmd/
│   └── goclaw/       # 主入口
└── docs/
    └── spec.md       # 本文件
```

------

## 4. 核心机制详细设计 (Core Mechanisms)

### 4.1 DAG 编译器 (The Compiler)

**解决协作问题第一步：定义谁依赖谁**

- **输入**: 声明式任务定义（带依赖关系）
- **输出**: `ExecutionPlan`（分层执行计划）
- **关键算法**:
  - Kahn 算法（拓扑排序）
  - 增量更新（支持运行时添加节点）
  - 环检测（DFS 着色法）

go

复制

```go
// 关键数据结构
type ExecutionPlan struct {
    Layers [][]TaskID          // 分层结果
    HotPath []TaskID           // 关键路径（用于优化）
    ParallelGroups []TaskGroup // 可并行组（用于资源预估）
}
```

### 4.2 Lane 调度器 (The Lane Scheduler)

**解决协作问题第二步：控制谁能同时跑**

- **核心抽象**: `Lane` = `Buffered Channel` + `Worker Pool` + `Rate Limiter`
- **Lane 类型**:
  - `Global`: 全局并发限制（如总 CPU 核数）
  - `Named`: 资源分类（io/cpu/memory/gpu）
  - `Session`: 用户级隔离（保证同一用户请求顺序性）
- **背压策略**: 当 Lane 满时，提供 `Block`, `Drop`, `Redirect` 三种策略

### 4.3 混合记忆系统 (Hybrid Memory)

**解决协作问题第三步：共享上下文和历史**

- **存储分层**:
  - **L1 (Hot)**: 进程内 LRU Cache（最近任务上下文）
  - **L2 (Warm)**: Badger（本地持久化）
  - **L3 (Cold)**: Weaviate/Pinecone（向量数据库）
- **检索策略**:
  - **向量检索**: HNSW 算法，余弦相似度，权重 0.7
  - **BM25**: 全文检索，权重 0.3
  - **融合**: RRF (Reciprocal Rank Fusion) 排序
- **遗忘机制**: FSRS-6 算法实现，后台 Goroutine 定期清理

------

## 5. 公共 API 设计 (Public API)

### 5.1 编程式 API (Go SDK)

go

复制

```go
// 工作流定义（编译期）
wf := goclaw.NewWorkflow("data_pipeline").
    WithTask("fetch", FetchAgent{}, 
        goclaw.WithLane("io"),
        goclaw.WithTimeout(5*time.Second)).
    WithTask("analyze", AnalyzeAgent{},
        goclaw.WithDeps("fetch"),
        goclaw.WithLane("cpu")).
    WithMemoryPolicy(
        goclaw.RetrieveLastN(3),      // 检索最近3条记忆
        goclaw.ForgetThreshold(0.3),  // 强度<0.3遗忘
    )

// 执行（运行时）
engine := goclaw.NewEngine(
    goclaw.WithStore(badgerStore),
    goclaw.WithLaneLimit("cpu", 8),
)
result, err := engine.Execute(ctx, wf, initialInput)
```

### 5.2 声明式 API (YAML/JSON)

支持配置文件定义工作流（便于 CI/CD 和可视化编辑器）：

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

### 5.3 gRPC 服务 API

protobuf

复制

```protobuf
service GoClawEngine {
  rpc SubmitWorkflow(WorkflowSpec) returns (stream TaskEvent);
  rpc GetTaskStatus(TaskID) returns (TaskStatus);
  rpc SignalTask(SignalRequest) returns (Empty); // 用于 steer/interrupt
}
```

------

## 6. 数据模型 (Data Models)

### 6.1 任务状态机

go

复制

```go
type TaskState int

const (
    StatePending TaskState = iota
    StateScheduled    // 已入 Lane 队列
    StateRunning      // 正在执行
    StateCompleted
    StateFailed
    StateCancelled
    StateRetrying     // 自动重试中
)
```

### 6.2 记忆条目结构

go

复制

```go
type MemoryEntry struct {
    ID        ulid.ULID     // 排序友好ID
    TaskID    string
    SessionID string        // 隔离不同用户/会话
    Content   []byte        // 序列化内容
    Vector    []float32     // 向量嵌入
    Metadata  map[string]string
    Strength  float64       // FSRS-6 记忆强度 [0,1]
    LastReview time.Time
}
```

------

## 7. 开发路线图 (Roadmap)

### Phase 1: MVP (Week 1-4)

- [ ] 基础 DAG 编译器（拓扑排序）
- [ ] 内存 Lane Queue（Channel 实现）
- [ ] 本地内存存储（Map + RWMutex）
- [ ] 同步执行引擎（单节点）

### Phase 2: 核心功能 (Week 5-8)

- [ ] 持久化存储（Badger 集成）
- [ ] Hybrid Memory（向量检索 + BM25）
- [ ] 分布式 Lane（Redis 队列）
- [ ] 消息模式（steer/interrupt/collect）

### Phase 3: 生产化 (Week 9-12)

- [ ] gRPC API + 流式事件
- [ ] 监控指标（Prometheus）
- [ ] 分布式事务（Saga 模式）
- [ ] Web UI 工作流可视化

------

## 8. 非功能性需求 (NFRs)

### 8.1 性能指标

- **延迟**: P99 任务启动延迟 < 5ms（本地存储）
- **吞吐**: 单节点 10k TPS（简单任务）
- **内存**: 支持 100 万个待执行任务不 OOM

### 8.2 可靠性

- **容错**: Agent panic 不影响引擎，自动重试 3 次
- **持久化**: 任务状态变更立即写 WAL（Write-Ahead Log）
- **优雅关闭**: 收到 SIGTERM 后等待进行中的任务完成（可配置超时）

### 8.3 可观测性

- **追踪**: 每个任务自动注入 Trace ID，支持分布式追踪
- **指标**: 暴露 `goclaw_tasks_total`, `goclaw_lane_wait_duration` 等 Prometheus 指标
- **日志**: 结构化日志（zap），支持上下文注入

------

## 9. 风险评估与缓解 (Risks)

表格

复制

| 风险              | 影响 | 缓解措施                                                 |
| :---------------- | :--- | :------------------------------------------------------- |
| **Go 泛型复杂性** | 中   | 初期使用 `any` 作为 Agent 输入输出，后续封装泛型工具函数 |
| **分布式一致性**  | 高   | Phase 1 专注单节点，Phase 2 引入 etcd 做 Leader 选举     |
| **向量检索性能**  | 中   | 支持多种后端（本地 HNSW/远程 Weaviate），可插拔切换      |
| **社区接受度**    | 低   | 保持与 OpenClaw 概念兼容，提供 Python 到 Go 的迁移指南   |

------

## 10. 参考与资源 (References)

- **灵感来源**: OpenClaw (TypeScript), Temporal, Cadence, Prefect
- **算法参考**: FSRS-6 Paper, HNSW Paper, Kahn's Algorithm
- **Go 最佳实践**: Uber Go Style Guide, Go Code Review Comments
