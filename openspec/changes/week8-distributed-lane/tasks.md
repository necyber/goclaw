## 1. 项目结构和依赖

- [x] 1.1 添加 `github.com/redis/go-redis/v9` 依赖到 `go.mod`
- [x] 1.2 创建 `pkg/signal/` 包目录结构
- [x] 1.3 在 `config/config.go` 添加 `RedisConfig` 结构
- [x] 1.4 在 `config/config.go` 添加 `SignalConfig` 结构
- [x] 1.5 更新 `config.example.yaml` 添加 `redis` 和 `signal` 配置段
- [x] 1.6 添加配置验证规则
- [x] 1.7 编写配置加载测试

## 2. Redis 连接管理

- [x] 2.1 创建 `pkg/lane/redis_config.go` Redis 连接配置
- [x] 2.2 实现 Redis 客户端初始化（支持单实例和 Sentinel）
- [x] 2.3 实现连接健康检查 `Ping()`
- [x] 2.4 实现自动重连逻辑（指数退避）
- [x] 2.5 实现连接池配置（MaxConns, MinIdleConns）
- [x] 2.6 编写 Redis 连接管理测试

## 3. Redis Lane 核心实现

- [x] 3.1 创建 `pkg/lane/redis_lane.go`
- [x] 3.2 实现 `RedisLane` 结构体，实现 `Lane` 接口
- [x] 3.3 实现 `Submit()` 方法（JSON 序列化 + 入队）
- [x] 3.4 实现 `TrySubmit()` 方法（非阻塞入队）
- [x] 3.5 实现 `Stats()` 方法（从 Redis 获取队列状态）
- [x] 3.6 实现 `Close()` 方法（优雅关闭）
- [x] 3.7 实现 `IsClosed()` 方法
- [x] 3.8 定义 `RedisTaskPayload` JSON 序列化结构

## 4. Redis 队列和优先级

- [x] 4.1 实现 FIFO 队列（Redis List: LPUSH/BRPOP）
- [x] 4.2 实现优先级队列（Redis Sorted Set: ZADD/ZPOPMIN）
- [x] 4.3 实现任务去重（Redis Set: SADD/SISMEMBER）
- [x] 4.4 实现去重 key 过期清理
- [x] 4.5 实现 Lua 脚本保证原子操作
- [x] 4.6 编写队列操作单元测试

## 5. Redis Lane 背压策略

- [x] 5.1 实现 Block 策略（轮询等待空间）
- [x] 5.2 实现 Drop 策略（检查容量后丢弃）
- [x] 5.3 实现 Redirect 策略（推送到其他 Lane key）
- [x] 5.4 实现容量检查 Lua 脚本
- [x] 5.5 编写背压策略测试

## 6. Redis Lane Worker Pool

- [x] 6.1 实现 Redis Worker Pool（从 Redis 队列消费）
- [x] 6.2 实现 BRPOP 阻塞消费循环
- [x] 6.3 实现 ZPOPMIN 优先级消费循环
- [x] 6.4 实现任务反序列化和执行
- [x] 6.5 实现执行结果回写（成功/失败计数）
- [x] 6.6 实现 Worker 优雅关闭
- [x] 6.7 编写 Worker Pool 测试

## 7. Redis Lane 降级和容错

- [x] 7.1 实现 Redis 不可用时降级到本地 Channel Lane
- [x] 7.2 实现后台 Redis 重连检测
- [x] 7.3 实现降级恢复（Redis 恢复后切回）
- [x] 7.4 添加降级日志和指标
- [x] 7.5 编写降级和容错测试

## 8. Lane Manager 混合模式

- [x] 8.1 扩展 `pkg/lane/manager.go` 支持 Lane 类型配置
- [x] 8.2 实现按名称创建不同类型 Lane（memory/redis）
- [x] 8.3 实现混合模式下的 Stats 聚合
- [x] 8.4 实现混合模式下的 Close 逻辑
- [x] 8.5 编写混合模式测试

## 9. Signal Bus 核心

- [x] 9.1 创建 `pkg/signal/bus.go` 定义 `SignalBus` 接口
- [x] 9.2 创建 `pkg/signal/message.go` 定义 `Signal` 和 `SignalType`
- [x] 9.3 实现本地 Signal Bus（Go channel 实现）
- [x] 9.4 实现 `Publish()` 方法
- [x] 9.5 实现 `Subscribe()` 方法（返回 channel）
- [x] 9.6 实现 `Unsubscribe()` 方法
- [x] 9.7 实现 `Close()` 方法
- [x] 9.8 实现信号缓冲区管理（可配置大小）
- [x] 9.9 实现并发安全保护
- [x] 9.10 编写本地 Signal Bus 单元测试

## 10. Redis Signal Bus

- [x] 10.1 创建 `pkg/signal/redis_bus.go`
- [x] 10.2 实现 Redis Pub/Sub 发布
- [x] 10.3 实现 Redis Pub/Sub 订阅
- [x] 10.4 实现信号 JSON 序列化/反序列化
- [x] 10.5 实现 Redis 频道命名（"goclaw:signal:{taskID}"）
- [x] 10.6 实现断线重连和自动重订阅
- [x] 10.7 实现健康检查
- [x] 10.8 编写 Redis Signal Bus 测试

## 11. Steer 消息模式

- [x] 11.1 创建 `pkg/signal/steer.go`
- [x] 11.2 实现 `SendSteer()` 方法（发送参数修改信号）
- [x] 11.3 实现 steer 信号验证（payload 格式检查）
- [x] 11.4 实现 `signal.FromContext(ctx)` 获取信号 channel
- [x] 11.5 实现 context 注入（在任务执行前注入信号 channel）
- [x] 11.6 编写 steer 消息模式测试

## 12. Interrupt 消息模式

- [x] 12.1 创建 `pkg/signal/interrupt.go`
- [x] 12.2 实现 `SendInterrupt()` 方法
- [x] 12.3 实现优雅中断（cancel context + 等待超时）
- [x] 12.4 实现强制中断（立即 cancel context）
- [x] 12.5 实现中断原因记录
- [x] 12.6 实现对 pending 任务的中断（从队列移除）
- [x] 12.7 编写 interrupt 消息模式测试

## 13. Collect 消息模式

- [x] 13.1 创建 `pkg/signal/collect.go`
- [x] 13.2 实现 `Collector` 结构体
- [x] 13.3 实现 `Collect()` 方法（等待所有任务完成）
- [x] 13.4 实现超时处理（返回部分结果）
- [x] 13.5 实现流式收集模式（fan-in）
- [x] 13.6 实现部分失败处理
- [x] 13.7 编写 collect 消息模式测试

## 14. Engine 和 API 集成

- [x] 14.1 在 `pkg/engine/engine.go` 添加 `signalBus` 字段
- [x] 14.2 在 `New()` 中初始化 Signal Bus
- [x] 14.3 在 `Start()` 中启动 Signal Bus
- [x] 14.4 在 `Stop()` 中关闭 Signal Bus
- [x] 14.5 在任务执行时注入信号 context
- [x] 14.6 实现 gRPC `SignalTask` RPC 与 Signal Bus 对接
- [x] 14.7 编写集成测试

## 15. 主程序集成

- [x] 15.1 在 `cmd/goclaw/main.go` 初始化 Redis 客户端（如果配置）
- [x] 15.2 初始化 Signal Bus（根据配置选择 local/redis）
- [x] 15.3 传递 Redis 客户端到 Lane Manager
- [x] 15.4 传递 Signal Bus 到 Engine
- [x] 15.5 在 shutdown 时关闭 Redis 连接和 Signal Bus
- [x] 15.6 添加启动日志
- [x] 15.7 测试完整启动和关闭流程

## 16. 指标集成

- [ ] 16.1 添加 Redis Lane 指标（队列深度、延迟、吞吐量）
- [ ] 16.2 添加 Signal Bus 指标（信号发送/接收/失败计数）
- [ ] 16.3 添加消息模式指标（steer/interrupt/collect 计数和延迟）
- [ ] 16.4 在 Prometheus 中注册新指标
- [ ] 16.5 编写指标测试

## 17. 测试

- [ ] 17.1 编写 Redis Lane 单元测试（mock Redis）
- [ ] 17.2 编写 Redis Lane 集成测试（需要 Redis 实例）
- [x] 17.3 编写 Signal Bus 单元测试
- [x] 17.4 编写消息模式端到端测试
- [ ] 17.5 编写混合 Lane 模式集成测试
- [ ] 17.6 编写降级和容错场景测试
- [x] 17.7 编写并发安全测试
- [ ] 17.8 编写性能基准测试（Redis Lane 吞吐量）
- [ ] 17.9 编写性能基准测试（Signal Bus 延迟）
- [ ] 17.10 运行完整测试套件确认无回归

## 18. 文档

- [ ] 18.1 创建 `docs/distributed-lane-guide.md` 使用指南
- [ ] 18.2 文档化 Redis Lane 配置选项
- [ ] 18.3 文档化消息模式使用方法和示例
- [ ] 18.4 文档化 Signal Bus 配置
- [ ] 18.5 添加 Redis 部署和配置指南
- [ ] 18.6 更新 `README.md` 添加分布式 Lane 说明
- [ ] 18.7 更新 `CLAUDE.md` 添加 signal 架构说明
- [ ] 18.8 更新 `docker-compose.yml` 添加 Redis 服务

## 19. 清理和验收

- [ ] 19.1 运行 `go fmt` 格式化代码
- [ ] 19.2 运行 `go vet` 检查问题
- [ ] 19.3 运行 `golangci-lint` 检查代码质量
- [ ] 19.4 修复所有 lint 警告
- [ ] 19.5 更新 go.mod 和 go.sum
- [ ] 19.6 运行完整测试套件
- [ ] 19.7 验证测试覆盖率 > 80%

## 验收标准

### 功能性
- [ ] Redis Lane 正确实现 Lane 接口
- [ ] 优先级队列和 FIFO 队列正常工作
- [ ] 三种背压策略正常工作
- [ ] 任务去重正常工作
- [ ] Steer 消息模式正常工作
- [ ] Interrupt 消息模式正常工作
- [ ] Collect 消息模式正常工作
- [ ] Signal Bus 本地和 Redis 模式正常工作
- [ ] 混合 Lane 模式正常工作
- [ ] Redis 不可用时降级正常

### 质量
- [ ] 单元测试覆盖率 > 80%
- [ ] 所有集成测试通过
- [ ] 无已知严重 bug
- [ ] 代码通过 lint 检查

### 性能
- [ ] Redis Lane Submit 延迟 < 5ms
- [ ] Redis Lane 吞吐量 > 10K tasks/s
- [ ] Signal Bus 信号延迟 < 2ms（本地）/ < 5ms（Redis）
- [ ] 降级切换时间 < 1s

## 估算

- **阶段 1-2**: 项目结构和 Redis 连接 - 1 天
- **阶段 3-6**: Redis Lane 核心实现 - 3 天
- **阶段 7-8**: 降级容错和混合模式 - 1 天
- **阶段 9-10**: Signal Bus 实现 - 2 天
- **阶段 11-13**: 消息模式实现 - 2 天
- **阶段 14-16**: 集成和指标 - 2 天
- **阶段 17-19**: 测试、文档和清理 - 2 天

**总计**: 约 13 天

## 注意事项

1. **Redis 可选**: Redis 是可选依赖，不配置时使用本地模式
2. **接口兼容**: Redis Lane 必须完全实现现有 Lane 接口
3. **测试隔离**: 集成测试需要 Redis 实例，单元测试使用 mock
4. **原子操作**: Redis 操作使用 Lua 脚本保证原子性
5. **序列化**: 任务 payload 使用 JSON，保持可读性
6. **降级策略**: Redis 不可用时自动降级，恢复后自动切回
7. **信号缓冲**: 信号 channel 有缓冲区，防止阻塞
8. **向后兼容**: 所有新功能通过配置开关控制
