# Project Status (2026-02-25)

本文件用于记录**当前实现进度评估**与**下一阶段任务优先级**。与 `ROADMAP.md` 的计划性不同，此处聚焦“已实现/未打通/下一步”。

## 进度评估（当前结论）

结论：项目已进入“实现阶段”，MVP 主要骨架已具备；Phase 2 部分完成（持久化/指标），Phase 3 仅有骨架（HTTP/gRPC）。

### 已实现/可用（核心能力已落地）
- DAG 构建与执行计划（层级并行、关键路径）
- 调度器按 layer 并行执行
- Lane 队列（背压/限流/worker pool）
- 存储：内存 + Badger
- 配置与日志
- HTTP API 与指标

### 未打通/仅骨架
- HTTP 提交流程只落库未触发执行（提交与执行未闭环）
- Scheduler 仍直接 goroutine 执行，Lane Manager 暂未接入
- gRPC 只有 proto/生成代码与 server 骨架，业务服务实现未见

### 路线图映射（粗略）
- Phase 1：大体完成，但“执行-API-存储”链路尚未贯通
- Phase 2：Badger 已落地；混合记忆/分布式 Lane/消息模式未开始
- Phase 3：HTTP + Metrics 有基础；gRPC/Streaming/事务/UI 仍处于骨架或未实现

---

## 下一阶段优先级（执行闭环 / gRPC / 分布式 / 记忆系统）

### Priority 1 — 执行闭环（必须先打通）
目标：HTTP 提交 -> 实际执行 -> 状态落库 -> 可查询/可取消
- 接口与执行打通
  - 在 `SubmitWorkflowRequest` 后触发 `Engine.Submit`（同步或异步模式二选一）
  - 增加异步执行模式：提交即返回，后台执行
- 状态持久化
  - 任务/工作流状态转移：pending → scheduled → running → completed/failed/cancelled
  - 将 `StateTracker` 状态写入 `storage`（含 StartedAt/CompletedAt/Result/Error）
- Lane 接入调度
  - Scheduler 不直接 goroutine 执行，改为通过 `lane.Manager.Submit`
  - Worker 执行后回写状态、完成信号
- 取消与超时
  - CancelWorkflow：支持取消正在执行的任务（上下文 cancel）
  - per-task timeout 已有，加入取消后状态处理
- 指标/日志闭环
  - 任务/工作流生命周期统一计数与时延
  - API 返回含执行状态（不只是“已提交”）

### Priority 2 — gRPC 服务落地
目标：gRPC 跟 HTTP 同功能、支持流式事件
- 服务实现与注册
  - 实现 `WorkflowServiceServer`（Submit/List/Get/Cancel/GetTaskResult）
  - 在 gRPC server 启动处注册 service
- 数据模型映射
  - API models <-> proto 消息转换
  - 统一校验与错误码（gRPC status + details）
- 事件流
  - 任务状态变化通过 streaming 推送（至少 workflow/task 事件流）
- 安全与可观测性
  - 接入已有拦截器链（auth/rate limit/log/metrics/tracing）

### Priority 3 — 分布式能力（最小集群可用）
目标：多节点运行、任务分发与状态一致
- Redis Lane
  - 实现 Redis Lane 适配器（队列 + 去重）
  - Backpressure / 重试策略
- 集群协调
  - etcd/Consul 节点发现
  - 任务分片策略（hash/consistent）
- 事件总线
  - NATS 集成（任务事件、状态变更）

### Priority 4 — 记忆系统（混合检索 + 记忆衰减）
目标：面向 Agent 的可检索记忆与衰减机制
- 向量存储基础
  - Embedding 接口定义
  - 本地向量索引（hnswgo 或 faiss-go）
- 文本检索
  - BM25 索引与检索
  - 混合检索策略（向量 + BM25）
- 记忆强度模型
  - FSRS-6 计算与调度
  - 记忆强度写入/更新策略

