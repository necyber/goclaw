## Context

Goclaw 当前的 Lane 系统基于 Go Channel 实现（`channel_lane.go`），仅支持单节点内存队列。现有 `Lane` 接口定义了 `Submit`、`TrySubmit`、`Stats`、`Close` 等方法，`Manager` 协调多个命名 Lane。

本设计需要：
1. 实现 Redis Lane 适配器，复用现有 `Lane` 接口
2. 实现三种消息模式（steer/interrupt/collect），用于 Agent 运行时通信
3. 实现信号总线，支持本地和分布式两种模式

**当前状态**:
- `pkg/lane/lane.go` 定义了 `Lane` 接口和 `Task` 接口
- `pkg/lane/channel_lane.go` 是内存 Channel 实现
- `pkg/lane/manager.go` 管理多个命名 Lane
- `pkg/lane/priority_queue.go` 支持优先级排序
- `pkg/lane/rate_limiter.go` 支持限流
- gRPC proto 已定义 `SignalTask` RPC

**约束**:
- Redis Lane 必须实现现有 `Lane` 接口，保持向后兼容
- 消息模式不能引入循环依赖（保持 DAG 纯粹性）
- 本地模式（无 Redis）必须继续正常工作

## Goals / Non-Goals

**Goals:**
- 实现 Redis Lane 适配器，支持跨节点任务分发
- 实现 Redis 队列的优先级排序和背压策略
- 实现 steer 消息模式（运行时修改任务参数）
- 实现 interrupt 消息模式（运行时中断任务）
- 实现 collect 消息模式（聚合多任务输出）
- 实现信号总线（本地 + Redis Pub/Sub）
- 扩展 Lane Manager 支持混合模式
- 支持任务去重（幂等提交）

**Non-Goals:**
- 不实现完整的分布式协调（etcd/Consul，Phase 3）
- 不实现分布式事务（Saga 模式，Week 11）
- 不实现任务迁移（节点间任务转移）
- 不实现 Redis Cluster 模式（初期单 Redis 实例）

## Decisions

### 1. Redis Lane 实现方案

**决策**: 使用 Redis Sorted Set + List 实现优先级队列

**理由**:
- Sorted Set 天然支持优先级排序（score = priority）
- List 用于 FIFO 队列（无优先级场景）
- Redis 原子操作保证并发安全
- BRPOP/BLPOP 支持阻塞等待

**替代方案**:
- Redis Stream: 功能强大但复杂度高，消费者组管理开销大
- Redis List only: 不支持优先级

**实现**:
```go
type RedisLane struct {
    client    *redis.Client
    name      string
    config    *Config
    queueKey  string // "goclaw:lane:{name}:queue"
    pendingKey string // "goclaw:lane:{name}:pending" (sorted set)
    dedupeKey string // "goclaw:lane:{name}:dedupe" (set)
}

func (r *RedisLane) Submit(ctx context.Context, task Task) error {
    // 1. 去重检查
    // 2. 根据配置选择 List 或 Sorted Set
    // 3. 应用背压策略
}
```

### 2. 任务序列化

**决策**: 使用 JSON 序列化任务到 Redis

**理由**:
- JSON 可读性好，便于调试
- Go 标准库支持
- 跨语言兼容

**替代方案**:
- Protocol Buffers: 性能更好但增加复杂度
- MessagePack: 更紧凑但可读性差

**实现**:
```go
type RedisTaskPayload struct {
    ID       string            `json:"id"`
    Lane     string            `json:"lane"`
    Priority int               `json:"priority"`
    Payload  json.RawMessage   `json:"payload"`
    Metadata map[string]string `json:"metadata"`
    EnqueuedAt time.Time       `json:"enqueued_at"`
}
```

### 3. 背压策略（Redis）

**决策**: 复用现有三种策略（Block/Drop/Redirect），通过 Redis 原子操作实现

**理由**:
- 保持与 Channel Lane 一致的行为
- Lua 脚本保证原子性

**实现**:
- Block: 使用 BRPOPLPUSH 阻塞等待
- Drop: 检查队列长度，超过容量则丢弃
- Redirect: 检查队列长度，超过容量则推送到另一个 Lane 的 key

### 4. 消息模式架构

**决策**: 通过 Signal Bus 实现三种消息模式，Signal Bus 支持本地（channel）和分布式（Redis Pub/Sub）两种后端

**理由**:
- 统一的消息传递抽象
- 本地模式零依赖，适合单节点
- Redis Pub/Sub 支持跨节点通信
- 与 gRPC `SignalTask` RPC 对接

**实现**:
```go
type SignalBus interface {
    // Publish 发送信号到指定任务
    Publish(ctx context.Context, signal *Signal) error
    // Subscribe 订阅指定任务的信号
    Subscribe(ctx context.Context, taskID string) (<-chan *Signal, error)
    // Unsubscribe 取消订阅
    Unsubscribe(taskID string) error
    // Close 关闭信号总线
    Close() error
}

type Signal struct {
    Type    SignalType         // Steer, Interrupt, Collect
    TaskID  string
    Payload map[string]any
    SentAt  time.Time
}

type SignalType int
const (
    SignalSteer SignalType = iota
    SignalInterrupt
    SignalCollect
)
```

### 5. Steer 消息模式

**决策**: Steer 通过 Signal Bus 发送参数修改信号，任务通过 context 或 channel 接收

**理由**:
- 非阻塞，不中断任务执行
- 任务可以选择性地响应 steer 信号
- 支持动态修改任务行为

**实现**:
```go
// 发送 steer 信号
bus.Publish(ctx, &Signal{
    Type:   SignalSteer,
    TaskID: "task-1",
    Payload: map[string]any{
        "temperature": 0.8,
        "max_tokens": 2000,
    },
})

// 任务内接收 steer 信号
func myTask(ctx context.Context) error {
    signals := signal.FromContext(ctx)
    for {
        select {
        case sig := <-signals:
            if sig.Type == SignalSteer {
                // 更新参数
            }
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}
```

### 6. Interrupt 消息模式

**决策**: Interrupt 通过取消 task context 实现，支持优雅中断和强制中断

**理由**:
- 利用 Go context 的取消机制
- 优雅中断允许任务清理资源
- 强制中断设置超时后强制终止

**实现**:
```go
// 发送 interrupt 信号
bus.Publish(ctx, &Signal{
    Type:   SignalInterrupt,
    TaskID: "task-1",
    Payload: map[string]any{
        "reason": "user_cancelled",
        "graceful": true,
        "timeout": "5s",
    },
})
```

### 7. Collect 消息模式

**决策**: Collect 通过 Signal Bus 收集多个任务的输出，使用 fan-in 模式聚合

**理由**:
- 支持等待所有任务完成后聚合
- 支持流式收集（边执行边收集）
- 支持超时和部分结果

**实现**:
```go
type Collector struct {
    bus      SignalBus
    taskIDs  []string
    results  map[string]any
    timeout  time.Duration
}

func (c *Collector) Collect(ctx context.Context) (map[string]any, error) {
    // 订阅所有任务的完成信号
    // 等待所有结果或超时
    // 返回聚合结果
}
```

### 8. Lane Manager 混合模式

**决策**: 扩展 Manager 支持按 Lane 名称配置不同后端（memory/redis）

**理由**:
- 灵活性：CPU 密集型任务用本地 Lane，IO 密集型用 Redis Lane
- 向后兼容：默认使用内存 Lane
- 渐进迁移：可以逐步将 Lane 迁移到 Redis

**实现**:
```go
// 配置示例
lanes:
  cpu:
    type: memory
    capacity: 100
    max_concurrency: 8
  io:
    type: redis
    capacity: 1000
    max_concurrency: 50
  gpu:
    type: redis
    capacity: 10
    max_concurrency: 2
```

## Risks / Trade-offs

### 1. Redis 单点故障

**风险**: Redis 不可用导致分布式 Lane 失效

**缓解**:
- 支持降级到本地 Lane（配置 fallback）
- 支持 Redis Sentinel 高可用
- 健康检查和自动重连

### 2. 网络延迟

**风险**: Redis 操作增加 ~1ms 延迟

**缓解**:
- 批量操作减少 RTT
- Pipeline 模式
- 本地缓存热数据

### 3. 任务序列化开销

**风险**: JSON 序列化/反序列化增加 CPU 开销

**缓解**:
- 任务 payload 保持精简
- 后续可切换到 MessagePack/Protobuf

### 4. 消息丢失

**风险**: Redis Pub/Sub 不保证消息持久化

**缓解**:
- 关键信号（interrupt）使用 Redis Stream 或直接写入
- 非关键信号（steer）允许丢失
- 本地模式使用 buffered channel

### 5. 向后兼容

**风险**: 新增功能可能影响现有 Lane 行为

**缓解**:
- Redis Lane 是新增实现，不修改 Channel Lane
- Signal Bus 默认禁用
- 所有新功能通过配置开关控制

## Migration Plan

### 部署步骤

1. **配置更新**:
   - 更新 `config.example.yaml` 添加 `redis` 和 `signal` 配置段
   - 默认 Lane 类型保持 `memory`

2. **依赖安装**:
   - `go get github.com/redis/go-redis/v9`

3. **渐进迁移**:
   - 先部署代码，保持所有 Lane 为 memory 模式
   - 逐步将 IO Lane 切换到 redis 模式
   - 验证稳定后切换更多 Lane

### 回滚策略

- 将 Lane 类型改回 `memory` 即可回滚
- Redis 中的队列数据可安全丢弃（任务会重新提交）
- Signal Bus 改回 `local` 模式

## Open Questions

1. **Redis Cluster 支持**: 是否需要支持 Redis Cluster？
   - 建议: Phase 3 考虑，初期单实例 + Sentinel

2. **任务持久化**: Redis Lane 中的任务是否需要持久化到 Badger？
   - 建议: 是，作为备份，防止 Redis 重启丢失

3. **消息顺序保证**: steer 信号是否需要严格顺序？
   - 建议: 不需要，使用最新值覆盖

4. **Collect 超时策略**: 部分任务超时时如何处理？
   - 建议: 返回已收集的部分结果 + 超时错误
