## Why

在多 Agent 协作场景中，一个工作流可能跨越多个节点、多个 Lane 执行，任何一步失败都可能导致整个流程处于不一致状态。当前 Goclaw 缺乏事务保障机制——任务失败后无法自动补偿已完成的步骤，也无法从断点恢复执行。Saga 模式提供了分布式环境下的最终一致性保障，是生产级编排引擎的核心能力。

## What Changes

- 新增 `pkg/saga/` 包，实现 Saga 编排器和补偿机制
- 实现 Saga 定义 DSL，支持声明式定义步骤和补偿操作
- 实现前向执行和反向补偿逻辑
- 实现断点续传（Checkpoint），支持从失败点恢复执行
- 实现 WAL（Write-Ahead Log）保证状态变更持久化
- 实现 Saga 状态机（Running → Compensating → Completed / Failed）
- 新增 Saga 相关 API 端点（提交、查询、恢复）
- 新增 Saga 相关 gRPC 服务方法
- 新增 Saga 指标（成功率、补偿次数、恢复次数）

## Capabilities

### New Capabilities

- `saga-orchestrator`: Saga 编排器核心，管理 Saga 生命周期、前向执行和反向补偿
- `saga-checkpoint`: 断点续传机制，基于 WAL 持久化 Saga 状态，支持从任意步骤恢复
- `saga-compensation`: 补偿操作定义和执行，支持自动和手动补偿策略
- `saga-api`: Saga 管理 API，提供 HTTP 和 gRPC 端点用于提交、查询、恢复 Saga

### Modified Capabilities

无

## Impact

**新增代码**:
- `pkg/saga/` - 新包
  - `saga.go` - Saga 定义和编排器
  - `step.go` - Saga 步骤定义（action + compensation）
  - `state.go` - Saga 状态机
  - `checkpoint.go` - 断点续传和 WAL
  - `compensation.go` - 补偿执行逻辑
  - `recovery.go` - 故障恢复

**修改代码**:
- `config/config.go` - 新增 SagaConfig
- `pkg/engine/engine.go` - 集成 Saga 编排器
- `pkg/api/handlers/` - 新增 Saga API 端点
- `pkg/api/router.go` - 注册 Saga 路由
- `cmd/goclaw/main.go` - 初始化 Saga 编排器

**依赖**:
- 复用现有 Badger 存储（WAL 和 Checkpoint）
- 复用现有 Signal Bus（补偿信号）

**配置影响**:
- `config.example.yaml` 新增 `saga` 配置段
- 支持配置补偿策略、重试次数、WAL 路径
