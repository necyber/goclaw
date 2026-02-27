## 1. 项目结构和依赖

- [x] 1.1 创建 `pkg/saga/` 包目录结构
- [x] 1.2 在 `config/config.go` 添加 `SagaConfig` 结构
- [x] 1.3 更新 `config.example.yaml` 添加 `saga` 配置段
- [x] 1.4 添加配置验证规则
- [x] 1.5 编写配置加载测试

## 2. Saga 数据模型

- [x] 2.1 创建 `pkg/saga/step.go` 定义 `Step` 结构（action + compensation）
- [x] 2.2 创建 `pkg/saga/saga.go` 定义 `SagaDefinition` 结构
- [x] 2.3 实现 Builder 模式 DSL（`New().Step().Step().Build()`）
- [x] 2.4 实现步骤依赖验证（DAG 合法性检查）
- [x] 2.5 创建 `pkg/saga/state.go` 定义 `SagaState` 状态机
- [x] 2.6 实现状态转移验证（合法转移检查）
- [x] 2.7 定义 `SagaInstance` 运行时实例结构
- [x] 2.8 编写数据模型单元测试

## 3. WAL 实现

- [x] 3.1 创建 `pkg/saga/wal.go` 定义 WAL 接口
- [x] 3.2 定义 `WALEntry` 结构（SagaID, StepID, Type, Data, Timestamp）
- [x] 3.3 定义 `WALEntryType` 枚举（StepStarted, StepCompleted, StepFailed, CompensationStarted, CompensationCompleted, CompensationFailed）
- [x] 3.4 实现基于 Badger 的 WAL 写入（key: "wal:{sagaID}:{sequence}"）
- [x] 3.5 实现 WAL 读取（按 SagaID 前缀扫描）
- [x] 3.6 实现同步和异步写入模式
- [x] 3.7 实现 WAL 序列号生成（单调递增）
- [x] 3.8 编写 WAL 单元测试
- [x] 3.9 编写 WAL 写入性能基准测试（目标 < 2ms）

## 4. Checkpoint 实现

- [x] 4.1 创建 `pkg/saga/checkpoint.go` 定义 Checkpoint 结构
- [x] 4.2 实现 Checkpoint 序列化/反序列化（JSON）
- [x] 4.3 实现 Checkpoint 写入 Badger（key: "checkpoint:{sagaID}"）
- [x] 4.4 实现 Checkpoint 读取
- [x] 4.5 实现每步完成后自动写 Checkpoint
- [x] 4.6 编写 Checkpoint 单元测试

## 5. Saga 编排器核心

- [x] 5.1 创建 `pkg/saga/orchestrator.go` 定义 `SagaOrchestrator`
- [x] 5.2 实现 `Execute()` 方法（前向执行主循环）
- [x] 5.3 实现步骤拓扑排序（复用 DAG 编译器逻辑）
- [x] 5.4 实现并行步骤执行（同层步骤并发）
- [x] 5.5 实现步骤结果传递（前序步骤结果注入 context）
- [x] 5.6 实现步骤超时控制
- [x] 5.7 实现 Saga 级别超时控制
- [x] 5.8 实现失败检测和补偿触发
- [x] 5.9 实现并发 Saga 限制（可配置上限）
- [x] 5.10 编写编排器单元测试
- [x] 5.11 编写编排器集成测试

## 6. 补偿执行

- [x] 6.1 创建 `pkg/saga/compensation.go` 定义补偿逻辑
- [x] 6.2 实现反向拓扑序补偿执行
- [x] 6.3 实现三种补偿策略（Auto/Manual/Skip）
- [x] 6.4 实现补偿重试（指数退避）
- [x] 6.5 实现补偿超时控制
- [x] 6.6 实现补偿 context 注入（原始输入、结果、失败原因）
- [x] 6.7 实现幂等性检查工具函数
- [x] 6.8 实现补偿失败后状态转移（CompensationFailed）
- [x] 6.9 编写补偿执行单元测试
- [x] 6.10 编写补偿重试测试
- [x] 6.11 编写补偿顺序测试（线性和并行场景）

## 7. 故障恢复

- [x] 7.1 创建 `pkg/saga/recovery.go` 定义恢复逻辑
- [x] 7.2 实现启动时扫描未完成 Saga
- [x] 7.3 实现从 Checkpoint 恢复 Running 状态 Saga
- [x] 7.4 实现从 Checkpoint 恢复 Compensating 状态 Saga
- [x] 7.5 实现恢复幂等性保证
- [x] 7.6 实现恢复日志记录
- [x] 7.7 编写恢复单元测试
- [x] 7.8 编写恢复集成测试（模拟进程崩溃）

## 8. WAL 清理

- [x] 8.1 实现 WAL 保留策略配置（默认 7 天）
- [x] 8.2 实现后台 WAL 清理 Goroutine
- [x] 8.3 实现批量删除过期 WAL 条目
- [x] 8.4 实现 Checkpoint 清理（终态 Saga）
- [x] 8.5 编写 WAL 清理测试

## 9. Saga 存储层

- [x] 9.1 定义 `SagaStore` 接口（CRUD for Saga instances）
- [x] 9.2 实现基于 Badger 的 SagaStore
- [x] 9.3 实现 Saga 实例持久化（key: "saga:{sagaID}"）
- [x] 9.4 实现按状态查询 Saga 列表
- [x] 9.5 实现分页查询
- [x] 9.6 编写 SagaStore 单元测试

## 10. HTTP API 端点

- [x] 10.1 创建 `pkg/api/handlers/saga.go`
- [x] 10.2 实现 `POST /api/v1/sagas` 提交 Saga
- [x] 10.3 实现 `GET /api/v1/sagas/{id}` 查询 Saga 状态
- [x] 10.4 实现 `GET /api/v1/sagas` 列出 Saga（分页 + 状态过滤）
- [x] 10.5 实现 `POST /api/v1/sagas/{id}/compensate` 手动触发补偿
- [x] 10.6 实现 `POST /api/v1/sagas/{id}/recover` 手动触发恢复
- [x] 10.7 在 `pkg/api/router.go` 注册 Saga 路由
- [x] 10.8 定义请求/响应模型
- [x] 10.9 添加请求验证和错误处理
- [x] 10.10 编写 API 端点单元测试
- [x] 10.11 编写 API 端点集成测试

## 11. gRPC 服务

- [x] 11.1 定义 Saga proto 消息和服务（SubmitSaga, GetSagaStatus, ListSagas, CompensateSaga, WatchSaga）
- [x] 11.2 生成 Go 代码
- [x] 11.3 实现 `SubmitSaga` RPC
- [x] 11.4 实现 `GetSagaStatus` RPC
- [x] 11.5 实现 `ListSagas` RPC
- [x] 11.6 实现 `CompensateSaga` RPC
- [x] 11.7 实现 `WatchSaga` 流式 RPC
- [x] 11.8 注册 Saga 服务到 gRPC server
- [x] 11.9 实现 proto ↔ 内部模型转换
- [x] 11.10 编写 gRPC 服务测试

## 12. Engine 集成

- [x] 12.1 在 `pkg/engine/engine.go` 添加 `sagaOrchestrator` 字段
- [x] 12.2 在 `New()` 中初始化 Saga 编排器（如果启用）
- [x] 12.3 在 `Start()` 中启动 Saga 恢复和 WAL 清理
- [x] 12.4 在 `Stop()` 中优雅关闭 Saga 编排器
- [x] 12.5 提供 `GetSagaOrchestrator()` 方法
- [x] 12.6 编写 Engine 集成测试

## 13. 主程序集成

- [x] 13.1 在 `cmd/goclaw/main.go` 初始化 Saga 编排器
- [x] 13.2 传递 Saga 编排器到 Engine
- [x] 13.3 传递 Saga 编排器到 API Server 和 gRPC Server
- [x] 13.4 在 shutdown 时优雅关闭 Saga 编排器
- [x] 13.5 添加 Saga 启动日志
- [x] 13.6 测试完整启动和关闭流程

## 14. 指标集成

- [x] 14.1 添加 `saga_executions_total` Counter（按状态标签）
- [x] 14.2 添加 `saga_duration_seconds` Histogram
- [x] 14.3 添加 `saga_active_count` Gauge
- [x] 14.4 添加 `saga_compensations_total` Counter
- [x] 14.5 添加 `saga_compensation_duration_seconds` Histogram
- [x] 14.6 添加 `saga_compensation_retries_total` Counter
- [x] 14.7 添加 `saga_recovery_total` Counter
- [x] 14.8 在 Prometheus 中注册指标
- [x] 14.9 编写指标测试

## 15. 测试

- [x] 15.1 编写 Saga 定义和 Builder 测试
- [x] 15.2 编写状态机转移测试
- [x] 15.3 编写前向执行测试（线性、并行、混合）
- [x] 15.4 编写补偿执行测试（线性、并行、跳过）
- [x] 15.5 编写 WAL 持久化和恢复测试
- [x] 15.6 编写 Checkpoint 创建和恢复测试
- [x] 15.7 编写超时测试（步骤超时、Saga 超时）
- [x] 15.8 编写并发 Saga 执行测试
- [x] 15.9 编写端到端集成测试（提交 → 执行 → 完成/补偿）
- [x] 15.10 编写故障恢复集成测试
- [x] 15.11 编写性能基准测试
- [x] 15.12 运行完整测试套件确认无回归

## 16. 文档

- [x] 16.1 创建 `docs/saga-guide.md` 使用指南
- [x] 16.2 文档化 Saga 定义 DSL 和示例
- [x] 16.3 文档化补偿策略和最佳实践
- [x] 16.4 文档化断点续传机制
- [x] 16.5 文档化 API 端点（HTTP + gRPC）
- [x] 16.6 文档化配置选项
- [x] 16.7 添加故障排查指南
- [x] 16.8 更新 `README.md` 添加 Saga 说明
- [x] 16.9 更新 `CLAUDE.md` 添加 Saga 架构说明
- [x] 16.10 生成 Swagger 文档（Saga API）

## 17. 清理和验收

- [x] 17.1 运行 `go fmt` 格式化代码
- [x] 17.2 运行 `go vet` 检查问题
- [x] 17.3 运行 `golangci-lint` 检查代码质量
- [x] 17.4 修复所有 lint 警告
- [x] 17.5 更新 go.mod 和 go.sum
- [x] 17.6 运行完整测试套件
- [x] 17.7 验证测试覆盖率 > 80%

## 验收标准

### 功能性
- [x] Saga 定义 DSL 正常工作
- [x] 前向执行按拓扑序正确执行
- [x] 步骤结果正确传递给依赖步骤
- [x] 失败时自动触发反向补偿
- [x] 补偿按逆序正确执行
- [x] 补偿重试正常工作
- [x] WAL 正确记录所有状态变更
- [x] Checkpoint 正确保存和恢复
- [x] 进程重启后未完成 Saga 自动恢复
- [x] HTTP 和 gRPC API 正常响应
- [x] 手动补偿和恢复正常工作

### 质量
- [x] 单元测试覆盖率 > 80%
- [x] 所有集成测试通过
- [x] 无已知严重 bug
- [x] 代码通过 lint 检查

### 性能
- [x] WAL 写入延迟 < 2ms（同步模式）
- [x] Saga 启动延迟 < 5ms
- [x] 支持 100 并发 Saga 执行
- [x] 恢复扫描时间 < 1s（1000 个 Saga）

### 文档
- [ ] 使用指南完整
- [ ] API 文档完整
- [ ] 补偿最佳实践文档
- [ ] 配置说明清晰

## 估算

- **阶段 1-2**: 项目结构和数据模型 - 1 天
- **阶段 3-4**: WAL 和 Checkpoint - 2 天
- **阶段 5-6**: 编排器和补偿 - 3 天
- **阶段 7-8**: 恢复和清理 - 1 天
- **阶段 9-11**: 存储层和 API - 3 天
- **阶段 12-14**: 集成和指标 - 2 天
- **阶段 15-17**: 测试、文档和清理 - 2 天

**总计**: 约 14 天

## 注意事项

1. **补偿幂等性**: 文档强调用户定义的补偿操作必须幂等
2. **WAL 先写**: 状态变更前先写 WAL，保证持久性
3. **测试驱动**: 为每个组件编写测试，特别是状态机和恢复逻辑
4. **增量提交**: 完成每个阶段后提交代码
5. **向后兼容**: Saga 功能默认禁用，不影响现有工作流
6. **复用组件**: 复用 DAG 编译器的拓扑排序、Badger 存储、Signal Bus
7. **错误处理**: 补偿失败是关键路径，必须有完善的错误处理和告警
8. **日志详尽**: Saga 生命周期的每个阶段都要有详细日志
