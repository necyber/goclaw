## 1. 依赖与配置模型

- [x] 1.1 审查现有 `TracingConfig` 与默认值，确认与 OTLP 目标模型的差异
- [x] 1.2 在 `config/config.go` 扩展 tracing 配置（exporter、endpoint、headers、timeout、sampler）
- [x] 1.3 在 `config/defaults.go` 更新 tracing 默认值（保持默认关闭）
- [x] 1.4 在 `config/config.example.yaml` 和 `config/config.example.json` 更新 tracing 示例配置
- [x] 1.5 在 `config/validator.go` 添加 tracing 配置校验（非法配置 fail fast）
- [x] 1.6 添加/更新 tracing 配置单元测试（有效与无效场景）
- [x] 1.7 设计并实现 legacy tracing 字段兼容映射（如 `type` 老字段）

## 2. TracerProvider 生命周期模块

- [x] 2.1 新建 tracing 生命周期管理模块（如 `pkg/telemetry/tracing`）
- [x] 2.2 实现 `Init`：根据配置初始化 TracerProvider、Resource、Sampler、Exporter
- [x] 2.3 实现全局 propagator 注册（TraceContext + Baggage）
- [x] 2.4 实现 `Shutdown/Flush`：支持超时控制与错误返回
- [x] 2.5 tracing 禁用时实现 no-op 路径（不启动 exporter 后台开销）
- [x] 2.6 为生命周期模块补充单元测试（启用、禁用、无效配置、关闭）

## 3. gRPC tracing 集成

- [x] 3.1 评估并调整 `pkg/grpc/interceptors/tracing.go` 的 span 语义与属性映射
- [x] 3.2 完善 unary tracing 拦截器（状态码、错误、方法属性）
- [x] 3.3 完善 stream tracing 拦截器（流生命周期与错误语义）
- [x] 3.4 验证 metadata 提取/注入链路（入站提取、出站注入）
- [x] 3.5 在拦截器链构建中支持 tracing 按配置启停
- [x] 3.6 在 gRPC 服务器启动流程中接入 tracing provider 初始化
- [x] 3.7 在 gRPC 关闭流程中接入 tracing flush/shutdown
- [x] 3.8 增加 gRPC tracing 启动诊断日志（不泄漏敏感头）
- [x] 3.9 增加 gRPC tracing 集成测试（启用、禁用、无效配置）

## 4. HTTP tracing 集成

- [x] 4.1 新增 `pkg/api/middleware/tracing.go` 实现 HTTP tracing 中间件
- [x] 4.2 实现入站 trace context 提取与新 root span 创建逻辑
- [x] 4.3 实现 HTTP span 属性与状态码映射（2xx/4xx/5xx）
- [x] 4.4 在 HTTP 路由/中间件链中接入 tracing 中间件（按配置启停）
- [x] 4.5 实现 health/readiness 低价值端点 tracing 策略（跳过或低采样）
- [x] 4.6 实现 request-scoped outbound HTTP context 注入辅助
- [x] 4.7 增加 HTTP tracing 单元/集成测试（traceparent 继承、无 header、新 root）

## 5. 运行时核心 span 覆盖

- [x] 5.1 梳理 workflow 执行主链路埋点位置并定义稳定 span 命名
- [x] 5.2 在 workflow/任务调度关键路径补充 runtime spans
- [x] 5.3 在 lane 调度与等待路径补充 spans（含关键属性）
- [x] 5.4 在 saga 前向执行路径补充 spans
- [x] 5.5 在 saga 补偿路径补充 spans
- [x] 5.6 在 saga 恢复路径补充 spans
- [x] 5.7 增加 runtime span 覆盖测试（至少 workflow + saga 关键场景）

## 6. 降级与可靠性

- [x] 6.1 实现 exporter 不可用时的错误隔离策略（不影响业务请求）
- [x] 6.2 对 tracing 失败路径增加告警级日志与调试上下文字段
- [x] 6.3 增加 exporter 故障注入测试，验证请求路径可用性
- [x] 6.4 增加关闭阶段 flush 超时测试，验证进程可有界退出

## 7. 可观测性关联

- [x] 7.1 在日志上下文中补充 trace_id/span_id 关联字段（存在 span 时）
- [x] 7.2 评估并补充 metrics exemplar/trace 关联点（后端支持时启用）
- [x] 7.3 补充日志与指标关联测试（trace 上下文存在/缺失）

## 8. 文档更新

- [x] 8.1 新增/更新 OpenTelemetry tracing 使用指南文档
- [x] 8.2 更新配置说明文档（字段含义、默认值、迁移说明）
- [x] 8.3 更新运行手册（collector 连接、故障排查、关闭语义）
- [x] 8.4 在 README 或架构文档补充 tracing 架构图/流程说明

## 9. 质量门禁

- [ ] 9.1 运行 `go fmt` 格式化受影响代码
- [ ] 9.2 运行 `go vet ./...`
- [ ] 9.3 运行 `golangci-lint run`（或等效命令）
- [ ] 9.4 运行 `go test ./...` 全量回归
- [ ] 9.5 运行 tracing 相关包的 targeted tests 与 benchmark（如有）

## 10. 验收清单

- [ ] 10.1 tracing 启用时可完成 provider 初始化与全局 propagator 注册
- [ ] 10.2 tracing 禁用时业务路径正常且无 exporter 初始化开销
- [ ] 10.3 gRPC unary/stream tracing span 生成与状态映射正确
- [ ] 10.4 HTTP tracing span 生成、状态映射、上下文提取正确
- [ ] 10.5 gRPC/HTTP 出站调用可注入 trace context
- [ ] 10.6 workflow/lane/saga 核心路径 span 覆盖满足规范
- [ ] 10.7 exporter 不可用时业务请求不受影响
- [ ] 10.8 shutdown 时 tracing flush 在超时窗口内完成
- [ ] 10.9 tracing 配置校验严格且错误信息清晰
- [ ] 10.10 文档、示例配置、测试与实现保持一致
