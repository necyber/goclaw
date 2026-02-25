# Goclaw 开发路线图

## 概览

| 阶段 | 时间 | 目标 | 关键交付物 |
|------|------|------|-----------|
| **Phase 1** | Week 1-4 | MVP - 核心骨架 | 可运行的本地编排引擎 |
| **Phase 2** | Week 5-8 | 核心功能 - 分布式支持 | 支持持久化、分布式队列 |
| **Phase 3** | Week 9-12 | 生产就绪 | 完整 API、监控、Web UI |

---

## Phase 1: MVP (Week 1-4)

**目标**：构建最小可用产品，实现单机本地运行的 Agent 编排引擎

### Week 1: 项目基础设施

- [x] 项目初始化（已完成）
  - [x] go.mod, Makefile, Dockerfile
  - [x] 目录结构（cmd/, pkg/, internal/）
  - [x] CI/CD 配置 (GitHub Actions)
  
- [ ] 配置系统
  - [ ] YAML/JSON 配置解析
  - [ ] 环境变量支持
  - [ ] 配置验证

- [ ] 日志系统
  - [ ] 结构化日志 (slog)
  - [ ] 日志级别控制
  - [ ] 文件输出支持

### Week 2: DAG 编译器

- [ ] 任务定义 DSL
  - [ ] Task 结构体设计
  - [ ] 依赖声明语法
  - [ ] 标签/元数据支持

- [ ] DAG 构建与验证
  - [ ] 图结构表示（邻接表/矩阵）
  - [ ] 环检测算法（DFS）
  - [ ] 拓扑排序（Kahn's Algorithm）

- [ ] 执行计划生成
  - [ ] 分层执行计划
  - [ ] 并行组识别
  - [ ] 关键路径分析

### Week 3: Lane 队列系统

- [ ] Lane 定义与管理
  - [ ] Lane 接口设计
  - [ ] 资源配额管理
  - [ ] 优先级队列

- [ ] Channel 实现
  - [ ] 基于 Go Channel 的队列
  - [ ] 背压机制（Backpressure）
  - [ ] 流控策略（令牌桶/漏桶）

- [ ] 调度器集成
  - [ ] 任务分派逻辑
  - [ ] Worker Pool 模式
  - [ ] 动态扩缩容

### Week 4: 内存存储与同步执行

- [ ] 内存存储实现
  - [ ] State 存储（Map + RWMutex）
  - [ ] Task 状态机管理
  - [ ] 事件发布/订阅

- [ ] 同步执行引擎
  - [ ] 执行器接口
  - [ ] 本地任务执行
  - [ ] 错误处理与重试

- [ ] 端到端测试
  - [ ] 简单工作流测试
  - [ ] 复杂依赖测试
  - [ ] 性能基准测试

---

## Phase 2: 核心功能 (Week 5-8)

**目标**：支持持久化存储、分布式队列和消息模式

### Week 5: 持久化存储 (Badger)

- [ ] Badger 集成
  - [ ] KV 存储接口抽象
  - [ ] BadgerStore 实现
  - [ ] 批量写入优化

- [ ] WAL (Write-Ahead Log)
  - [ ] 日志格式设计
  - [ ] 崩溃恢复机制
  - [ ] 日志压缩

- [ ] 状态快照
  - [ ] 定期快照
  - [ ] 增量快照
  - [ ] 快照恢复

### Week 6: 混合 Memory 系统

- [ ] 向量存储基础
  - [ ] Embedding 接口
  - [ ] 本地向量索引（hnswgo）
  - [ ] 向量检索接口

- [ ] BM25 全文检索
  - [ ] 倒排索引构建
  - [ ] 相关性评分
  - [ ] 混合检索策略

- [ ] FSRS-6 记忆算法
  - [ ] 记忆强度计算
  - [ ] 复习调度
  - [ ] 遗忘曲线建模

### Week 7: 分布式 Lane (Redis) 🔖 候选项

- [ ] Redis 队列实现
  - [ ] Redis Lane 适配器
  - [ ] 分布式锁
  - [ ] 任务去重

- [ ] 集群协调 🔖 候选项
  - [ ] 节点发现（Consul/etcd）
  - [ ] 任务分片
  - [ ] 负载均衡

- [ ] 消息传递
  - [ ] NATS 集成
  - [ ] 发布订阅模式
  - [ ] 消息持久化

### Week 8: 消息模式

- [ ] Steer 模式
  - [ ] 运行时任务调整
  - [ ] 动态参数修改
  - [ ] 执行路径切换

- [ ] Interrupt 模式
  - [ ] 任务中断信号
  - [ ] 优雅停止
  - [ ] 状态保存

- [ ] Collect 模式
  - [ ] 结果收集器
  - [ ] 聚合策略
  - [ ] 流式输出

---

## Phase 3: 生产就绪 (Week 9-12)

**目标**：完整的 API、监控、分布式事务和可视化界面

### Week 9: gRPC API 与事件流

- [ ] Protocol Buffers 定义
  - [ ] 服务接口定义
  - [ ] 消息类型定义
  - [ ] API 版本管理

- [ ] gRPC 服务实现
  - [ ] Workflow 提交接口
  - [ ] 状态查询接口
  - [ ] 信号控制接口

- [ ] 事件流 (SSE/gRPC Streaming)
  - [ ] 任务事件推送
  - [ ] 实时日志流
  - [ ] 客户端订阅管理

### Week 10: 监控与可观测性

- [ ] Prometheus 指标
  - [ ] 任务计数器
  - [ ] Lane 等待时长
  - [ ] 执行延迟分布

- [ ] OpenTelemetry 链路追踪
  - [ ] Trace ID 传播
  - [ ] Span 上下文
  - [ ] 分布式追踪

- [ ] 健康检查
  - [ ] Liveness Probe
  - [ ] Readiness Probe
  - [ ] 依赖健康检查

### Week 11: 分布式事务 (Saga)

- [ ] Saga 协调器
  - [ ] 补偿事务定义
  - [ ] 正向/逆向流程
  - [ ] 事务状态机

- [ ] 故障恢复
  - [ ] 超时处理
  - [ ] 重试策略（指数退避）
  - [ ] 死信队列

- [ ] 一致性保证
  - [ ] 最终一致性
  - [ ] 幂等性设计
  - [ ] 并发控制

### Week 12: Web UI 与工作流可视化

- [ ] 后端 API
  - [ ] RESTful API 封装
  - [ ] WebSocket 实时推送
  - [ ] 认证授权

- [ ] 工作流可视化
  - [ ] DAG 图形渲染
  - [ ] 实时状态展示
  - [ ] 执行路径高亮

- [ ] 管理界面
  - [ ] 任务管理面板
  - [ ] Lane 监控
  - [ ] 系统配置

---

## 技术选型

| 组件 | 选型 | 备选 |
|------|------|------|
| 配置解析 | Viper | Koanf |
| 结构化日志 | slog | zap |
| KV 存储 | Badger | BoltDB |
| 分布式存储 | etcd | Consul |
| 缓存/队列 | Redis | NATS JetStream |
| 消息总线 | NATS | RabbitMQ |
| 向量检索 | hnswgo | faiss-go |
| gRPC | google.golang.org/grpc | connectrpc |
| 指标监控 | Prometheus + Grafana | |
| 链路追踪 | OpenTelemetry | Jaeger |

---

## 开发规范

### 代码提交规范

```
<type>(<scope>): <subject>

<body>

<footer>
```

Type: `feat|fix|docs|style|refactor|test|chore`

### 测试要求

- 单元测试覆盖率 > 70%
- 核心模块覆盖率 > 85%
- 每个 PR 必须通过 CI

### 文档要求

- 所有导出函数必须有 GoDoc
- 复杂算法必须引用论文/实现
- API 变更必须更新文档

---

## 里程碑

| 日期 | 里程碑 | 验收标准 |
|------|--------|----------|
| Week 4 | Alpha 版本 | 本地运行简单工作流 |
| Week 8 | Beta 版本 | 支持分布式部署 |
| Week 12 | v1.0.0 发布 | 生产可用，完整文档 |
