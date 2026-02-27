# Goclaw 开发路线图

> 更新日期：2026-02-27  
> 说明：本路线图反映“已实现 + 下一步计划”。  
> 历史变更记录见 `openspec/changes/archive/*`，当前规范基线见 `openspec/specs/*`。

---

## 概览

| 阶段 | 周次 | 目标 | 当前状态 |
|------|------|------|----------|
| **Phase 1** | Week 1-4 | MVP 核心骨架 | ✅ 完成 |
| **Phase 2** | Week 5-8 | 核心能力（存储/记忆/分布式 Lane） | ✅ 完成 |
| **Phase 3** | Week 9-12 | 生产能力（gRPC/监控/Saga/Web UI） | ✅ 完成 |
| **Phase 4** | Week 13-15 | 执行闭环 + 集群事件总线 + OTel 链路追踪 | ✅ 完成 |
| **Phase 5** | Week 16+ | 稳定性、性能与发布治理 | 🚧 进行中 |

---

## 已完成（Week 1-15）

### Phase 1: MVP（Week 1-4）✅

- [x] **Week 1 - 配置与基础设施**
  - [x] 项目初始化（`go.mod`、`Makefile`、`Dockerfile`）
  - [x] 配置加载与校验（YAML/JSON + 环境变量覆盖）
  - [x] 基础日志体系与启动流程
- [x] **Week 2 - DAG 编译器**
  - [x] DAG 构建、环检测、拓扑排序
  - [x] 分层执行计划生成
  - [x] 关键路径能力（HotPath）基础支持
- [x] **Week 3 - Lane 队列系统**
  - [x] Lane 抽象与资源隔离
  - [x] 背压与流控机制
  - [x] Worker Pool 并发执行模型
- [x] **Week 4 - 引擎核心**
  - [x] 工作流执行核心
  - [x] 状态机与任务生命周期管理
  - [x] 基础端到端执行路径

### Phase 2: 核心能力（Week 5-8）✅

- [x] **Week 5 - HTTP API 与持久化存储**
  - [x] HTTP Server / Workflow API
  - [x] Badger 存储集成
  - [x] WAL 与快照机制
- [x] **Week 6 - Hybrid Memory**
  - [x] 向量检索（Vector Retrieval）
  - [x] BM25 全文检索
  - [x] RRF 混合检索
  - [x] FSRS-6 记忆衰减
- [x] **Week 7 - 混合记忆与运行时能力补齐**
  - [x] Memory Hub API 与存储接口整合
  - [x] 记忆读写与衰减闭环
  - [x] 相关可观测指标接入
- [x] **Week 8 - 分布式 Lane 与消息模式**
  - [x] Redis Lane（队列/去重/退化）
  - [x] Signal Bus（steer/interrupt/collect）
  - [x] 分布式运行模式下的兼容与降级

### Phase 3: 生产能力（Week 9-12）✅

- [x] **Week 9 - gRPC API 与流式能力**
  - [x] Proto 定义与服务实现
  - [x] gRPC 客户端与服务端集成
  - [x] Streaming/Realtime 能力
- [x] **Week 10 - 监控与可观测**
  - [x] Prometheus 指标
  - [x] HTTP/gRPC 指标采集
  - [x] 健康检查（Liveness/Readiness）
- [x] **Week 11 - 分布式事务（Saga）**
  - [x] Saga 编排与补偿策略
  - [x] Checkpoint/恢复机制
  - [x] Saga API 与管理能力
- [x] **Week 12 - Web UI 与可视化**
  - [x] UI 路由与静态资源集成
  - [x] DAG 可视化与状态展示
  - [x] 事件推送与控制台交互

### Phase 4: 增强能力（Week 13-15）✅

- [x] **Week 13 - 执行流水线闭环**
  - [x] 提交 -> 调度 -> 执行 -> 状态回写全链路
  - [x] 调度与 Lane 的一致性集成
  - [x] 运行时行为与 API 语义对齐
- [x] **Week 14 - 集群事件总线**
  - [x] NATS 事件总线能力
  - [x] 事件契约与消费者规范
  - [x] 集群协调与分片相关能力补齐
- [x] **Week 15 - OpenTelemetry 链路追踪**
  - [x] Tracing 生命周期管理
  - [x] HTTP/gRPC 中间件/拦截器接入
  - [x] Tracing 文档、架构与运维 Runbook

---

## 下一阶段（Week 16+）🚧

### Phase 5: 稳定性与发布治理

- [ ] **性能与容量验证**
  - [ ] P95/P99 启动延迟与吞吐基准
  - [ ] 压测场景标准化（单节点/集群）
  - [ ] 资源成本画像（CPU/内存/IO）
- [ ] **可靠性强化**
  - [ ] 故障注入与恢复演练（网络抖动、存储不可用、消息堆积）
  - [ ] Saga/消息链路混沌测试
  - [ ] 降级与回退策略验收
- [ ] **安全与治理**
  - [ ] API 认证鉴权策略收敛
  - [ ] 依赖漏洞扫描与版本治理（含 `govulncheck`）
  - [ ] 审计日志与敏感信息处理策略
- [ ] **发布工程化**
  - [ ] 版本节奏与发布清单模板
  - [ ] 升级/回滚手册
  - [ ] 文档站点与变更日志自动化

---

## 里程碑

| 日期 | 里程碑 | 状态 |
|------|--------|------|
| 2026-02-26 | 完成 Week 8（分布式 Lane / Signal Bus）归档 | ✅ |
| 2026-02-27 | 完成 Week 9-12（gRPC / 监控 / Saga / Web UI）归档 | ✅ |
| 2026-02-27 | 完成 Week 13-15（执行闭环 / 事件总线 / OTel Tracing）归档 | ✅ |
| TBD | Phase 5 稳定性与发布治理验收 | 🚧 |
