# Week 4: Engine 核心实现

## 背景

前三周已完成配置系统、DAG 编译器和 Lane 队列系统。`pkg/engine/engine.go` 目前是一个空壳，仅有状态机骨架和 TODO 注释。Week 4 的目标是将 DAG 和 Lane 真正串联起来，让 Engine 能够接收工作流定义、编译执行计划并驱动 Lane 执行任务。

## 目标

实现可运行的编排引擎核心，具备以下能力：

1. **Engine 初始化**：集成 `dag.Graph`、`lane.Manager`、`config.Config`、`logger`
2. **工作流提交**：接收 DAG 定义，编译为 `ExecutionPlan`
3. **执行调度**：按层（Layer）驱动 Lane 并发执行任务
4. **状态追踪**：追踪每个任务的运行状态（Pending → Running → Completed/Failed）
5. **优雅关闭**：等待运行中任务完成后再退出
6. **错误处理**：任务失败时支持重试，超出重试次数后标记工作流失败

## 非目标

- 持久化存储（Phase 2）
- 分布式调度（Phase 2）
- gRPC API（Phase 3）
- Agent 注册与发现（Phase 3）

## 详细设计

### 核心抽象

```
Engine = DAG Compiler + Execution Scheduler + Lane Manager + State Tracker
```

### 组件设计

| 组件 | 职责 | 关键接口 |
|------|------|----------|
| Engine | 顶层编排器 | Submit(), Start(), Stop(), State() |
| Scheduler | 按层调度执行计划 | Schedule(plan, ctx) |
| TaskRunner | 将 dag.Task 包装为 lane.Task | Run(task) |
| StateTracker | 追踪任务/工作流状态 | Update(), Get(), Watch() |

### 执行流程

```
Submit(workflow) → Compile DAG → ExecutionPlan
                                      ↓
                              For each Layer:
                                Submit all tasks to Lane concurrently
                                Wait for all tasks in layer to complete
                                      ↓
                              Next Layer (or Done/Error)
```

### 状态机

Engine 状态：`Idle → Running → Stopped | Error`
任务状态：`Pending → Scheduled → Running → Completed | Failed`

## 验收标准

- [ ] Engine 能成功初始化并集成 DAG + Lane
- [ ] 提交工作流后按层并发执行任务
- [ ] 任务失败时支持重试（可配置次数）
- [ ] 优雅关闭等待运行中任务完成
- [ ] 单元测试覆盖率 > 80%
- [ ] 集成测试：端到端执行一个 3 层 DAG

## 影响范围

- 重写 `pkg/engine/engine.go`
- 新增 `pkg/engine/scheduler.go`
- 新增 `pkg/engine/runner.go`
- 新增 `pkg/engine/state.go`
- 新增 `pkg/engine/engine_test.go`
- 更新 `cmd/goclaw/main.go`（传入完整 config）

## 风险与缓解

| 风险 | 缓解措施 |
|------|----------|
| 层间同步复杂 | 使用 `sync.WaitGroup` + `errgroup` |
| 任务 panic 泄漏 | Lane WorkerPool 已有 panic 恢复 |
| 状态竞争 | 使用 `sync/atomic` 或 mutex 保护状态 |
