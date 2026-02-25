## Context

Goclaw 当前支持 DAG 工作流执行，但缺乏事务保障。任务失败后，已完成的步骤无法回滚，工作流无法从断点恢复。在分布式多节点场景下，这会导致数据不一致。

**当前状态**:
- 已有 Badger 持久化存储（可用于 WAL 和 Checkpoint）
- 已有任务状态机（Pending → Scheduled → Running → Completed/Failed/Cancelled/Retrying）
- 已有 Signal Bus（可用于补偿信号传递）
- 已有 Engine 管理工作流生命周期

**约束**:
- Saga 是最终一致性模型，不提供 ACID 隔离性
- 补偿操作必须是幂等的
- WAL 写入不能显著影响任务执行延迟

## Goals / Non-Goals

**Goals:**
- 实现 Saga 编排模式（Orchestration-based Saga）
- 支持声明式定义 Saga 步骤和补偿操作
- 实现前向执行：按 DAG 依赖顺序执行步骤
- 实现反向补偿：失败时按逆序执行补偿操作
- 实现 WAL 持久化 Saga 状态变更
- 实现断点续传：进程重启后从最后 checkpoint 恢复
- 实现 Saga 状态机和生命周期管理
- 提供 HTTP 和 gRPC API 管理 Saga
- 暴露 Saga 相关 Prometheus 指标

**Non-Goals:**
- 不实现 Choreography-based Saga（事件驱动编排）
- 不实现两阶段提交（2PC）
- 不实现嵌套 Saga（Saga 内嵌套 Saga）
- 不实现跨集群 Saga 协调（初期单集群）
- 不实现自动补偿操作生成（由用户定义）

## Decisions

### 1. Saga 模式选择

**决策**: 采用 Orchestration-based Saga（编排式）

**理由**:
- 中心化编排器，逻辑清晰，易于调试
- 与 Goclaw 的 DAG 调度器天然契合
- 补偿顺序由编排器控制，确保正确性

**替代方案**:
- Choreography-based: 去中心化，通过事件驱动，但调试困难、补偿顺序难以保证

**实现**:
```go
type SagaOrchestrator struct {
    store     SagaStore       // WAL + Checkpoint
    engine    *engine.Engine  // 任务执行
    signalBus signal.SignalBus
}

func (o *SagaOrchestrator) Execute(ctx context.Context, saga *SagaDefinition) (*SagaResult, error) {
    // 1. 创建 Saga 实例，写入 WAL
    // 2. 按 DAG 顺序前向执行步骤
    // 3. 每步完成后写 checkpoint
    // 4. 失败时触发反向补偿
    // 5. 返回最终结果
}
```

### 2. Saga 定义 DSL

**决策**: 使用 Go 结构体 + Builder 模式定义 Saga

**理由**:
- 类型安全，编译时检查
- 与现有 Workflow 定义风格一致
- 支持声明式和编程式两种方式

**实现**:
```go
saga := saga.New("order-processing").
    Step("reserve-inventory",
        saga.Action(reserveInventory),
        saga.Compensate(releaseInventory),
    ).
    Step("charge-payment",
        saga.Action(chargePayment),
        saga.Compensate(refundPayment),
        saga.DependsOn("reserve-inventory"),
    ).
    Step("ship-order",
        saga.Action(shipOrder),
        saga.Compensate(cancelShipment),
        saga.DependsOn("charge-payment"),
    ).
    Build()
```

### 3. Saga 状态机

**决策**: 六状态状态机

**理由**:
- 覆盖 Saga 完整生命周期
- 支持补偿和恢复场景

**实现**:
```
Created → Running → Completed
                  → Compensating → Compensated
                                 → CompensationFailed
         → Recovering (从 checkpoint 恢复)
```

```go
type SagaState int
const (
    SagaCreated SagaState = iota
    SagaRunning
    SagaCompleted
    SagaCompensating
    SagaCompensated
    SagaCompensationFailed
)
```

### 4. WAL 实现

**决策**: 基于 Badger 实现 WAL，每个状态变更写入一条日志

**理由**:
- 复用现有 Badger 存储，无新依赖
- Badger 的 LSM-Tree 写入性能优秀
- 支持按 Saga ID 前缀扫描恢复

**替代方案**:
- 独立 WAL 文件: 需要自实现日志格式和恢复逻辑
- Redis Stream: 增加外部依赖

**实现**:
```go
type WALEntry struct {
    SagaID    string
    StepID    string
    Type      WALEntryType // StepStarted, StepCompleted, StepFailed, CompensationStarted, ...
    Data      []byte
    Timestamp time.Time
}

// Key format: "wal:{sagaID}:{sequence}"
```

### 5. Checkpoint 机制

**决策**: 每个步骤完成后写入 checkpoint，包含已完成步骤列表和中间结果

**理由**:
- 粒度适中（步骤级别）
- 恢复时只需重放未完成的步骤
- checkpoint 数据量可控

**实现**:
```go
type Checkpoint struct {
    SagaID         string
    State          SagaState
    CompletedSteps []string
    FailedStep     string
    StepResults    map[string][]byte
    LastUpdated    time.Time
}
```

### 6. 补偿策略

**决策**: 支持三种补偿策略：自动补偿、手动补偿、跳过补偿

**理由**:
- 自动补偿适合大多数场景
- 手动补偿用于需要人工介入的场景
- 跳过补偿用于幂等操作

**实现**:
```go
type CompensationPolicy int
const (
    AutoCompensate   CompensationPolicy = iota // 自动执行补偿
    ManualCompensate                           // 标记待补偿，等待人工触发
    SkipCompensate                             // 跳过补偿（幂等操作）
)
```

### 7. 补偿执行顺序

**决策**: 反向拓扑序执行补偿（已完成步骤的逆序）

**理由**:
- 保证依赖关系的逆向解除
- 与 DAG 执行顺序对称

**实现**:
- 步骤执行顺序: A → B → C（C 失败）
- 补偿执行顺序: B.compensate → A.compensate

### 8. 补偿重试

**决策**: 补偿操作支持配置重试次数和退避策略

**理由**:
- 补偿操作也可能暂时失败（网络抖动等）
- 重试提高补偿成功率
- 最终失败进入 CompensationFailed 状态，需人工介入

**实现**:
```go
type CompensationRetryConfig struct {
    MaxRetries     int
    InitialBackoff time.Duration
    MaxBackoff     time.Duration
    BackoffFactor  float64
}
```

## Risks / Trade-offs

### 1. 补偿操作幂等性

**风险**: 用户定义的补偿操作不幂等，重复执行导致副作用

**缓解**:
- 文档强调补偿操作必须幂等
- 提供幂等性检查工具函数
- WAL 记录补偿执行状态，避免重复触发

### 2. WAL 写入延迟

**风险**: 每步写 WAL 增加延迟

**缓解**:
- Badger 异步写入模式（可配置）
- 批量写入优化
- 基准测试目标: WAL 写入 < 1ms

### 3. 长时间运行的 Saga

**风险**: Saga 执行时间过长，占用资源

**缓解**:
- 支持 Saga 级别超时配置
- 超时后自动触发补偿
- 提供 Saga 执行时长指标

### 4. CompensationFailed 状态

**风险**: 补偿失败后系统处于不一致状态

**缓解**:
- 告警通知（集成现有告警规则）
- 提供手动重试补偿 API
- 提供 Saga 状态查询和诊断 API

### 5. 进程崩溃恢复

**风险**: 进程崩溃时 Saga 处于中间状态

**缓解**:
- WAL 保证状态持久化
- 启动时扫描未完成 Saga 并恢复
- Checkpoint 减少恢复时的重放量

## Migration Plan

### 部署步骤

1. 更新 `config.example.yaml` 添加 `saga` 配置段
2. Saga 功能默认禁用（`saga.enabled: false`）
3. 启用后，启动时自动扫描并恢复未完成 Saga

### 回滚策略

- 设置 `saga.enabled: false` 禁用
- WAL 数据保留在 Badger 中，不影响其他功能
- 已完成的 Saga 数据可通过 API 清理

## Open Questions

1. **Saga 超时后的行为**: 超时后是立即补偿还是标记为待处理？
   - 建议: 自动补偿，可配置

2. **并发 Saga 限制**: 是否限制同时运行的 Saga 数量？
   - 建议: 可配置上限，默认 100

3. **Saga 日志保留策略**: WAL 日志保留多久？
   - 建议: 已完成 Saga 的 WAL 保留 7 天后自动清理
