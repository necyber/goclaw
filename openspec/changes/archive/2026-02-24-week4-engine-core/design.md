# Week 4: Engine 核心设计文档

## 架构概览

```
┌─────────────────────────────────────────────────────────────┐
│                        Engine                                │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Submit(Workflow)                                            │
│       │                                                      │
│       ▼                                                      │
│  ┌──────────┐    ┌──────────────┐    ┌──────────────────┐  │
│  │   DAG    │───▶│  Execution   │───▶│   Scheduler      │  │
│  │ Compiler │    │    Plan      │    │  (layer-by-layer) │  │
│  └──────────┘    └──────────────┘    └────────┬─────────┘  │
│                                               │              │
│                                               ▼              │
│                                    ┌──────────────────┐     │
│                                    │   Lane Manager   │     │
│                                    │  ┌────┐ ┌────┐  │     │
│                                    │  │cpu │ │io  │  │     │
│                                    │  └────┘ └────┘  │     │
│                                    └──────────────────┘     │
│                                               │              │
│                                               ▼              │
│                                    ┌──────────────────┐     │
│                                    │  State Tracker   │     │
│                                    │ task/workflow    │     │
│                                    │ status map       │     │
│                                    └──────────────────┘     │
└─────────────────────────────────────────────────────────────┘
```

## 核心数据结构

### Engine

```go
type Engine struct {
    config      *config.Config
    logger      *slog.Logger
    dagGraph    *dag.Graph
    laneManager *lane.Manager
    scheduler   *Scheduler
    tracker     *StateTracker
    state       atomic.Int32   // EngineState
    mu          sync.Mutex
}
```

### Workflow（提交单元）

```go
// Workflow 是提交给 Engine 的工作流定义
type Workflow struct {
    ID    string
    Tasks []*dag.Task
}

// WorkflowResult 是工作流执行结果
type WorkflowResult struct {
    WorkflowID string
    Status     WorkflowStatus
    TaskResults map[string]*TaskResult
    Error      error
}
```

### Scheduler

```go
// Scheduler 按层驱动执行计划
type Scheduler struct {
    laneManager *lane.Manager
    tracker     *StateTracker
    logger      *slog.Logger
}

// Schedule 执行一个编译好的计划，按层并发提交任务
// 每层内所有任务并发提交到对应 Lane，等待全部完成后进入下一层
func (s *Scheduler) Schedule(ctx context.Context, plan *dag.ExecutionPlan, tasks map[string]*dag.Task) error
```

### StateTracker

```go
type TaskState int

const (
    TaskStatePending TaskState = iota
    TaskStateScheduled
    TaskStateRunning
    TaskStateCompleted
    TaskStateFailed
)

type TaskResult struct {
    TaskID    string
    State     TaskState
    Error     error
    StartedAt time.Time
    EndedAt   time.Time
    Retries   int
}

type StateTracker struct {
    mu      sync.RWMutex
    results map[string]*TaskResult
}
```

### TaskRunner（lane.Task 适配器）

```go
// taskRunner 将 dag.Task 包装为 lane.Task，供 Lane 执行
type taskRunner struct {
    task    *dag.Task
    tracker *StateTracker
    retries int
    fn      func(ctx context.Context) error
}

func (r *taskRunner) ID() string       { return r.task.ID }
func (r *taskRunner) Priority() int    { return 1 }
func (r *taskRunner) Lane() string     { return r.task.Lane }
```

## 关键流程

### Submit 流程

```
1. 构建 dag.Graph（AddTask for each task）
2. graph.Compile() → ExecutionPlan
3. 初始化 StateTracker（所有任务设为 Pending）
4. scheduler.Schedule(ctx, plan, tasks)
5. 返回 WorkflowResult
```

### Schedule 流程（逐层执行）

```
for each layer in plan.Layers:
    errgroup.Go for each taskID in layer:
        tracker.SetState(taskID, Scheduled)
        laneManager.Submit(ctx, taskRunner{task})
        // taskRunner.Execute() 内部:
        //   tracker.SetState(taskID, Running)
        //   执行任务逻辑（重试）
        //   tracker.SetState(taskID, Completed/Failed)
    errgroup.Wait()  // 等待本层全部完成
    if any task failed and no retry left: return error
```

### 优雅关闭

```
Stop(ctx):
    1. 设置 state = Stopping（不再接受新 Workflow）
    2. 等待当前 Workflow 完成（或 ctx 超时）
    3. laneManager.Close(ctx)
    4. 设置 state = Stopped
```

## Engine Config 扩展

在现有 `config.Config` 的 `orchestration` 节中使用：

```yaml
orchestration:
  max_agents: 10
  queue_type: "channel"
  queue_size: 1000
  scheduler_type: "layered"
  default_lane: "default"
  default_concurrency: 4
  task_timeout: "30s"
  max_retries: 3
```

## 默认 Lane 策略

Engine 启动时根据 config 自动创建默认 Lane：

```go
// 若 dag.Task.Lane 为空，使用 "default" lane
defaultLaneConfig := &lane.Config{
    Name:           "default",
    Capacity:       cfg.Orchestration.QueueSize,
    MaxConcurrency: cfg.Orchestration.DefaultConcurrency,
    Backpressure:   lane.Block,
}
```

## 错误处理

- **编译错误**（循环依赖等）：立即返回，不进入调度
- **任务执行失败**：按 `dag.Task.Retries` 重试，超出后标记 Failed
- **层执行失败**：默认 fail-fast（整个 Workflow 失败），后续层不执行
- **context 取消**：所有 Lane Submit 感知 ctx，立即停止调度

## 文件结构

```
pkg/engine/
├── engine.go        # Engine 结构、Start/Stop/Submit
├── scheduler.go     # Scheduler，逐层调度
├── runner.go        # taskRunner，lane.Task 适配器
├── state.go         # StateTracker，任务状态管理
├── errors.go        # 引擎错误类型
└── engine_test.go   # 单元 + 集成测试
```

## 待优化项

1. **并行工作流**：当前每次 Submit 串行等待完成，后续支持并发多工作流
2. **任务超时**：per-task timeout（dag.Task.Timeout）尚未接入
3. **事件总线**：任务状态变更事件通知（Phase 3 Web UI 需要）
