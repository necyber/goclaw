# Week 4 任务列表

## Phase 1: 状态管理 (Day 1)

### 1.1 TaskResult 和 StateTracker
- [x] 创建 `pkg/engine/state.go`
- [x] 定义 `TaskState` 枚举（Pending/Scheduled/Running/Completed/Failed）
- [x] 定义 `TaskResult` 结构（TaskID, State, Error, StartedAt, EndedAt, Retries）
- [x] 实现 `StateTracker`（mutex 保护的 map）
- [x] 实现 `SetState()`、`GetResult()`、`InitTasks()` 方法

### 1.2 错误类型
- [x] 创建 `pkg/engine/errors.go`
- [x] 定义 `WorkflowCompileError`
- [x] 定义 `TaskExecutionError`（含 TaskID、重试次数）
- [x] 定义 `EngineNotRunningError`

## Phase 2: TaskRunner 适配器 (Day 1-2)

### 2.1 taskRunner 实现
- [x] 创建 `pkg/engine/runner.go`
- [x] 实现 `taskRunner` 结构（包装 `dag.Task`）
- [x] 实现 `lane.Task` 接口（ID/Priority/Lane）
- [x] 实现 `Execute(ctx)` 方法（含重试逻辑）
- [x] 在 Execute 内更新 StateTracker 状态

## Phase 3: Scheduler (Day 2)

### 3.1 逐层调度器
- [x] 创建 `pkg/engine/scheduler.go`
- [x] 实现 `Scheduler` 结构
- [x] 实现 `Schedule(ctx, plan, tasks)` 方法
- [x] 使用 goroutine + WaitGroup 并发提交同层任务
- [x] 等待每层全部完成后进入下一层
- [x] 支持 fail-fast：任一任务失败则停止后续层

## Phase 4: Engine 核心重写 (Day 2-3)

### 4.1 Engine 结构重写
- [x] 重写 `pkg/engine/engine.go`
- [x] 集成 `*config.Config`、`appLogger`、`*lane.Manager`、`*Scheduler`、`*StateTracker`
- [x] 使用 `atomic.Int32` 管理 engineState
- [x] 实现 `New(cfg *config.Config, logger appLogger) (*Engine, error)`

### 4.2 Start/Stop 实现
- [x] 实现 `Start(ctx)`: 初始化默认 Lane，启动 Lane Manager
- [x] 实现 `Stop(ctx)`: 优雅关闭 Lane Manager
- [x] 确保 Stop 感知 ctx 超时

### 4.3 Submit 实现
- [x] 定义 `Workflow` 和 `WorkflowResult` 类型
- [x] 实现 `Submit(ctx, workflow) (*WorkflowResult, error)`
- [x] 构建 dag.Graph → Compile → Schedule → 返回结果

## Phase 5: main.go 更新 (Day 3)

### 5.1 CLI 入口更新
- [x] 更新 `cmd/goclaw/main.go`
- [x] 传入完整 `*config.Config` 和 logger 给 `engine.New()`
- [x] 移除旧的 `engine.Config` 用法

## Phase 6: 测试 (Day 4-5)

### 6.1 单元测试
- [x] 创建 `pkg/engine/engine_test.go`
- [x] 测试 StateTracker 并发安全
- [x] 测试 taskRunner 重试逻辑
- [x] 测试 Scheduler 逐层执行顺序
- [x] 测试 Engine Start/Stop 状态机

### 6.2 集成测试
- [x] 端到端测试：提交 3 层 DAG，验证执行顺序
- [x] 测试任务失败 + 重试场景
- [x] 测试 context 取消中断执行

### 6.3 示例代码
- [x] 创建 `examples/engine/main.go`
- [x] 演示完整工作流提交和结果获取

## 进度跟踪

| Phase | 状态 | 完成度 |
|-------|------|--------|
| Phase 1 | ✅ 完成 | 100% |
| Phase 2 | ✅ 完成 | 100% |
| Phase 3 | ✅ 完成 | 100% |
| Phase 4 | ✅ 完成 | 100% |
| Phase 5 | ✅ 完成 | 100% |
| Phase 6 | ✅ 完成 | 100% |

**总体进度: 100% ✅**

## 已创建文件

```
pkg/engine/
├── engine.go        # Engine 结构、Start/Stop/Submit、appLogger 接口
├── scheduler.go     # Scheduler，逐层调度
├── runner.go        # taskRunner，lane.Task 适配器，含重试
├── state.go         # StateTracker，任务状态管理
├── errors.go        # 引擎错误类型
└── engine_test.go   # 单元 + 集成测试

examples/engine/
└── main.go          # 完整工作流示例
```

## 关键设计决策

1. **appLogger 接口**：engine 包定义最小 logger 接口，避免与 pkg/logger 循环依赖
2. **per-workflow StateTracker**：每次 Submit 创建独立 tracker，支持并发工作流（未来）
3. **无 errgroup 依赖**：使用标准库 sync.WaitGroup + mutex，避免引入外部依赖
4. **默认 Lane**：dag.Task.Lane 为空时自动路由到 "default" lane
