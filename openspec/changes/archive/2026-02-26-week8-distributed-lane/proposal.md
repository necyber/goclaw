## Why

当前 Goclaw 的 Lane 队列系统仅支持单节点内存 Channel 实现，无法在多节点环境下分发任务。同时，Agent 之间缺乏运行时通信机制（如转向、中断、收集），限制了复杂协作场景的实现。分布式 Lane 和消息模式是 Phase 2 的核心功能，为后续集群部署和高级 Agent 协作奠定基础。

## What Changes

- 新增 Redis Lane 适配器，实现 `Lane` 接口，支持跨节点任务分发
- 新增 Redis 连接管理和配置
- 实现 steer 消息模式：运行时动态修改任务参数或行为
- 实现 interrupt 消息模式：运行时中断正在执行的任务
- 实现 collect 消息模式：聚合多个任务的输出结果
- 新增 Signal 机制，支持通过 gRPC `SignalTask` 发送消息
- 扩展 Lane Manager 支持混合模式（本地 + Redis Lane 共存）
- 新增 Redis 相关配置项

## Capabilities

### New Capabilities

- `redis-lane`: 基于 Redis 的分布式 Lane 队列实现，支持跨节点任务提交、去重、优先级排序和背压策略
- `message-patterns`: Agent 间运行时消息模式，包括 steer（转向）、interrupt（中断）、collect（收集）三种模式
- `signal-bus`: 信号总线，用于在 Agent/Task 之间传递运行时消息，支持本地和分布式（Redis Pub/Sub）两种模式

### Modified Capabilities

无

## Impact

**新增代码**:
- `pkg/lane/redis_lane.go` - Redis Lane 实现
- `pkg/lane/redis_config.go` - Redis 连接配置和管理
- `pkg/signal/` - 新包，信号总线和消息模式实现
  - `bus.go` - 信号总线核心
  - `steer.go` - steer 消息模式
  - `interrupt.go` - interrupt 消息模式
  - `collect.go` - collect 消息模式
  - `message.go` - 消息类型定义

**修改代码**:
- `pkg/lane/manager.go` - 扩展支持混合 Lane 模式（本地 + Redis）
- `config/config.go` - 新增 RedisConfig 和 SignalConfig
- `pkg/engine/engine.go` - 集成 Signal Bus
- `cmd/goclaw/main.go` - 初始化 Redis 连接和 Signal Bus

**新增依赖**:
- `github.com/redis/go-redis/v9` - Redis 客户端

**配置影响**:
- `config.example.yaml` 新增 `redis` 和 `signal` 配置段
- 支持 Lane 类型选择（memory / redis）
- 支持 Signal Bus 模式选择（local / redis）

**性能影响**:
- Redis Lane 增加网络延迟（~1ms per operation）
- Redis Pub/Sub 用于信号传递，增加少量网络开销
- 本地模式无额外开销
