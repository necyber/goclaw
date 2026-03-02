## Context

`2026-02-27-week10-monitoring` 已归档，但当前实现与归档需求存在偏差，主要集中在指标语义一致性与标签基数控制：
- `ChannelLane` 的 `Redirect` 快速路径未统一入队包装，导致等待时长统计不稳定。
- Task 指标未携带 `task_type` 维度，无法按任务类型过滤与聚合。
- `workflow_active_count` 仅跟踪 `running`，缺少 `pending` 生命周期计数。
- HTTP 指标状态标签与路径归一化语义不完全符合监控需求。
- 缺少通用的标签基数保护与配置硬化规则。

该改动横跨 `engine`、`lane`、`api/middleware`、`metrics`、`config` 多模块，属于跨切面监控一致性修复。

## Goals / Non-Goals

**Goals:**
- 让监控行为与 Week10 监控规格可追溯一致。
- 保持 Prometheus 指标命名稳定前提下，修复标签语义和关键缺失指标。
- 为潜在高基数字段增加明确、可测试的边界保护。
- 提供低风险迁移方案，避免业务执行路径回归。

**Non-Goals:**
- 不引入新的监控后端或替换 Prometheus 客户端。
- 不重构现有引擎执行模型与 Lane 运行时架构。
- 不在本次变更中做全量 dashboard 重设计，仅做查询兼容建议。

## Decisions

1. Lane redirect 统一采用 `queuedTask` 封装入队  
理由：等待时长依赖 `EnqueuedAt()`；统一封装可避免路径分叉导致的漏记。  
备选：仅在 worker 端兜底缺失时间戳。  
取舍：worker 兜底会引入估算误差，无法反映真实排队时间，故不采用。

2. Task 指标接口升级为带 `task_type` 维度  
理由：规格要求按任务类型统计 execution/duration/retry。  
备选：在 metric label 中复用 task ID 或 name。  
取舍：task ID/name 容易带来高基数，且语义不稳定，故仅保留受控 `task_type`。

3. Workflow 活跃数状态机扩展为 `pending` + `running`  
理由：提交后到运行前存在真实排队窗口，需可观测。  
备选：仅统计 running 并通过 submissions 推断 pending。  
取舍：推断不准确且滞后，直接维护 gauge 更可靠。

4. HTTP 状态与路径标签标准化  
决策：
- `status` 标签使用分组值：`2xx|3xx|4xx|5xx`（保留必要兼容窗口见迁移计划）。  
- `path` 使用归一化模板路径，替换动态段（数字、UUID、ULID 及长随机 token）。  
备选：保留原始状态码/路径，仅在文档提示风险。  
取舍：文档约束无法防止线上爆炸，必须在采集端收敛。

5. 引入通用标签基数保护器（bounded label cardinality guard）  
决策：对高风险维度维护“允许值集合 + 上限计数”，超过阈值时丢弃新值并记录告警计数。  
备选：完全依赖 `normalizePath`。  
取舍：单点归一化不足以覆盖所有标签来源，需要统一保护层。

## Risks / Trade-offs

- [指标语义变化导致历史看板查询失效] → 提供兼容期双写或兼容查询模板，并在文档中列出迁移 PromQL。
- [标签收敛过度损失排障信息] → 保留结构化日志字段用于细粒度定位，不将高基数信息直接入 metric label。
- [跨模块接口变更导致编译影响面扩大] → 先扩展接口再逐处迁移调用，最后收敛旧签名。
- [active gauge 计数不平衡] → 增加状态迁移单测，覆盖取消、失败、启动失败等分支。

## Migration Plan

1. 扩展 `pkg/metrics` 接口与内部结构，保留短期兼容入口。
2. 迁移 `engine` 与 `lane` 调用点，补齐 `task_type` 与 pending/running 生命周期更新。
3. 调整 HTTP middleware 的状态与路径标准化策略，并加基数保护。
4. 增补/更新测试：单元测试、并发测试、指标暴露测试。
5. 更新 `docs/monitoring-guide.md` 与 README 指标示例；标注 PromQL 兼容写法。
6. 通过 `go test ./...` 与 `openspec validate --changes --strict` 后合入。

回滚策略：保留兼容开关或兼容入口，必要时可回退到旧标签模式。

## Open Questions

- 是否需要为 HTTP `status` 保留原始状态码作为可选附加指标（避免 breaking dashboard）？
- `task_type` 的来源优先级如何定义（task.Agent、task.Name、自定义配置字段）？
- 标签上限阈值是否固定为 1000，还是做成可配置并给出默认值？
