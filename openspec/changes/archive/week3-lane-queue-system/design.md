# Week 3: Lane 队列系统设计

## 概述

Week 3 实现了 GoClaw 的 Lane 队列系统，这是一个基于 Go Channel 的内存任务调度系统，支持背压策略、Worker Pool、优先级队列和速率限制。

## 核心组件

### 1. Lane 接口

```go
type Lane interface {
    Name() string
    Submit(ctx context.Context, task Task) error
    TrySubmit(task Task) bool
    Stats() Stats
    Close(ctx context.Context) error
    IsClosed() bool
}
```

### 2. ChannelLane 实现

- **任务队列**: 使用 buffered channel 存储待执行的任务
- **Worker Pool**: 每个 Lane 拥有独立的 goroutine 池
- **背压策略**: 支持 Block、Drop、Redirect 三种模式
- **速率限制**: 可选的令牌桶流控

### 3. Worker Pool

- 固定数量的 worker goroutine
- 任务分发到空闲 worker
- 支持优雅关闭，等待正在执行的任务完成
- panic 恢复机制

### 4. 优先级队列

- 基于 `container/heap` 实现
- 高优先级任务优先执行
- 线程安全（使用互斥锁）
- 支持阻塞和非阻塞操作

### 5. Lane Manager

- 统一管理多个 Lane
- 支持动态注册/注销
- 全局统计信息
- 任务自动路由到对应 Lane

### 6. 速率限制

**令牌桶 (Token Bucket)**:
- 支持突发流量
- 可配置产生速率和桶容量
- 支持等待或立即失败

**漏桶 (Leaky Bucket)**:
- 匀速输出
- 适合严格限速场景

## 背压策略

| 策略 | 行为 | 适用场景 |
|------|------|----------|
| Block | 队列满时阻塞提交者 | 不允许丢任务 |
| Drop | 队列满时丢弃新任务 | 允许丢任务 |
| Redirect | 重定向到其他 Lane | 负载均衡 |

## 使用示例

```go
// 创建 Lane
config := &lane.Config{
    Name:           "cpu",
    Capacity:       100,
    MaxConcurrency: 8,
    Backpressure:   lane.Block,
    RateLimit:      100, // 100 tasks/sec
}

l, _ := lane.New(config)
defer l.Close(context.Background())
l.Run()

// 提交任务
task := lane.NewTaskFunc("task-1", "cpu", 1, func(ctx context.Context) error {
    // Do work
    return nil
})

err := l.Submit(context.Background(), task)
```

## 性能特性

- **无锁设计**: 利用 Channel 的原子性
- **水平扩展**: 多个 Lane 并行运行
- **内存高效**: 固定容量，避免 OOM
- **低延迟**: < 1ms 任务调度延迟（本地测试）

## 实现文件

```
pkg/lane/
├── lane.go           # 接口定义
├── errors.go         # 错误类型
├── channel_lane.go   # Lane 实现
├── worker_pool.go    # Worker Pool
├── priority_queue.go # 优先级队列
├── rate_limiter.go   # 速率限制
├── manager.go        # Lane Manager
└── lane_test.go      # 单元测试
```

## 待优化项

1. **优先级抢占**: 高优先级任务无法中断低优先级任务
2. **分布式支持**: 当前仅支持单节点
3. **持久化**: 任务队列非持久化

> 注：动态扩缩容已通过 `DynamicWorkerPool`（`worker_pool.go:136`）实现。以上剩余优化将在 Phase 2 (Week 5-8) 中实现。
