# Week 10: 监控与可观测性 - 任务清单

## 1. Metrics 包基础设施

- [x] 1.1 创建 `pkg/metrics` 包目录结构
- [x] 1.2 定义 `MetricsManager` 结构
- [x] 1.3 实现 Prometheus registry 初始化
- [x] 1.4 添加 metrics 配置结构到 `config/config.go`
- [x] 1.5 实现 metrics server (HTTP endpoint)
- [x] 1.6 添加 metrics 启用/禁用开关

## 2. Workflow 指标

- [x] 2.1 定义 workflow 指标（Counter, Histogram, Gauge）
- [x] 2.2 实现 `RecordWorkflowSubmission()` 方法
- [x] 2.3 实现 `RecordWorkflowDuration()` 方法
- [x] 2.4 实现 `SetActiveWorkflows()` 方法
- [x] 2.5 在 `engine.SubmitWorkflowRequest()` 中添加指标记录
- [x] 2.6 在 workflow 执行完成时记录时长
- [x] 2.7 编写 workflow metrics 单元测试

## 3. Task 指标

- [x] 3.1 定义 task 指标（Counter, Histogram）
- [x] 3.2 实现 `RecordTaskExecution()` 方法
- [x] 3.3 实现 `RecordTaskDuration()` 方法
- [x] 3.4 实现 `RecordTaskRetry()` 方法
- [x] 3.5 在 task runner 中添加指标记录
- [x] 3.6 在 task 重试逻辑中添加计数
- [x] 3.7 编写 task metrics 单元测试

## 4. Lane 队列指标

- [x] 4.1 定义 lane 指标（Gauge, Histogram, Counter）
- [x] 4.2 实现 `SetQueueDepth()` 方法
- [x] 4.3 实现 `RecordWaitDuration()` 方法
- [x] 4.4 实现 `RecordThroughput()` 方法
- [x] 4.5 在 `ChannelLane.Submit()` 中添加队列深度记录
- [x] 4.6 在 task 出队时记录等待时长
- [x] 4.7 在 worker 处理完成时记录吞吐量
- [x] 4.8 编写 lane metrics 单元测试

## 5. HTTP API 指标

- [x] 5.1 定义 HTTP 指标（Counter, Histogram, Gauge）
- [x] 5.2 创建 `pkg/api/middleware/metrics.go`
- [x] 5.3 实现 metrics middleware
- [x] 5.4 实现 `RecordHTTPRequest()` 方法
- [x] 5.5 实现活跃连接计数（Inc/Dec）
- [x] 5.6 包装 ResponseWriter 以捕获状态码
- [x] 5.7 在 router 中注册 metrics middleware
- [x] 5.8 编写 HTTP metrics 单元测试

## 6. 系统指标

- [x] 6.1 添加 Go runtime 指标（goroutines, memory, GC）
- [x] 6.2 实现自定义 collector（可选）
- [x] 6.3 添加进程指标（CPU, 文件描述符）
- [x] 6.4 配置指标收集间隔

## 7. 配置集成

- [x] 7.1 在 `config/config.go` 添加 `MetricsConfig` 结构
- [x] 7.2 添加 metrics 端口配置（默认 9091）
- [x] 7.3 添加 metrics 路径配置（默认 /metrics）
- [x] 7.4 添加各类指标的启用开关
- [x] 7.5 添加 histogram buckets 配置
- [x] 7.6 更新 `config.example.yaml` 添加 metrics 配置
- [x] 7.7 添加配置验证规则
- [x] 7.8 编写配置加载测试

## 8. 主程序集成

- [x] 8.1 在 `cmd/goclaw/main.go` 初始化 MetricsManager
- [x] 8.2 启动 metrics HTTP server（独立端口）
- [x] 8.3 传递 metrics 到 engine
- [x] 8.4 传递 metrics 到 API server
- [x] 8.5 在 shutdown 时优雅关闭 metrics server
- [x] 8.6 添加 metrics 初始化日志
- [x] 8.7 测试 metrics endpoint 可访问性

## 9. Prometheus 配置

- [x] 9.1 创建 `config/prometheus.yml` 配置文件
- [x] 9.2 配置 scrape job for goclaw
- [x] 9.3 设置 scrape interval（15s）
- [x] 9.4 添加 service labels
- [x] 9.5 配置 relabeling rules（可选）
- [x] 9.6 测试 Prometheus 抓取

## 10. Grafana 仪表板

- [x] 10.1 创建 `config/grafana/` 目录
- [x] 10.2 设计工作流概览仪表板
- [x] 10.3 设计任务执行仪表板
- [x] 10.4 设计 Lane 队列仪表板
- [x] 10.5 设计 HTTP API 仪表板
- [x] 10.6 导出仪表板 JSON 文件
- [x] 10.7 编写仪表板导入说明

## 11. 告警规则

- [x] 11.1 创建 `config/prometheus/alerts.yml`
- [x] 11.2 定义工作流失败率告警
- [x] 11.3 定义队列积压告警
- [x] 11.4 定义 API 高延迟告警
- [x] 11.5 定义 HTTP 错误率告警
- [x] 11.6 配置告警阈值和持续时间
- [x] 11.7 添加告警注释和标签
- [x] 11.8 测试告警规则

## 12. 测试

- [ ] 12.1 编写 MetricsManager 单元测试
- [ ] 12.2 编写各类指标的单元测试
- [ ] 12.3 编写 metrics middleware 单元测试
- [ ] 12.4 编写 metrics server 集成测试
- [ ] 12.5 测试指标准确性（计数、时长）
- [ ] 12.6 测试并发场景下的指标收集
- [ ] 12.7 性能测试（开销 <1%）
- [ ] 12.8 测试 metrics 禁用功能

## 13. 文档

- [ ] 13.1 更新 `README.md` 添加 metrics 说明
- [ ] 13.2 创建 `docs/monitoring-guide.md` 监控指南
- [ ] 13.3 文档化所有指标及其含义
- [ ] 13.4 添加 Prometheus 配置示例
- [ ] 13.5 添加 Grafana 仪表板使用说明
- [ ] 13.6 添加告警规则配置指南
- [ ] 13.7 添加故障排查指南
- [ ] 13.8 更新 `CLAUDE.md` 添加 metrics 架构说明

## 14. Docker 和部署

- [x] 14.1 更新 `Dockerfile` 暴露 metrics 端口
- [x] 14.2 更新 `docker-compose.yml` 添加 Prometheus 服务
- [x] 14.3 更新 `docker-compose.yml` 添加 Grafana 服务
- [x] 14.4 配置 Prometheus 数据卷
- [x] 14.5 配置 Grafana 数据卷
- [x] 14.6 测试 Docker 环境下的 metrics 收集
- [x] 14.7 测试 Grafana 仪表板导入

## 15. 性能优化

- [ ] 15.1 优化指标收集性能（使用原子操作）
- [ ] 15.2 限制指标基数（避免高基数标签）
- [ ] 15.3 实现指标采样（可选）
- [ ] 15.4 添加指标缓存（可选）
- [ ] 15.5 基准测试指标收集开销
- [ ] 15.6 优化 histogram buckets 配置
- [ ] 15.7 验证内存使用 <10MB

## 16. 清理和优化

- [ ] 16.1 运行 `go fmt` 格式化代码
- [ ] 16.2 运行 `go vet` 检查问题
- [ ] 16.3 运行 `golangci-lint` 检查代码质量
- [ ] 16.4 修复所有 lint 警告
- [ ] 16.5 更新 go.mod 和 go.sum
- [ ] 16.6 运行完整测试套件确认无回归
- [ ] 16.7 更新 ROADMAP.md 标记完成状态

## 验收标准

完成所有任务后，验证以下标准：

### 功能性
- [ ] Metrics endpoint 可访问（http://localhost:9091/metrics）
- [ ] 所有核心指标正确收集和暴露
- [ ] Prometheus 可以成功抓取指标
- [ ] Grafana 仪表板正确显示数据
- [ ] 告警规则正确触发

### 质量
- [ ] 单元测试覆盖率 > 80%
- [ ] 所有集成测试通过
- [ ] 性能测试通过（开销 <1%）
- [ ] 无已知严重 bug
- [ ] 代码通过 lint 检查

### 性能
- [ ] Metrics 收集开销 <1% CPU
- [ ] Metrics 内存占用 <10MB
- [ ] Metrics endpoint 响应时间 <100ms
- [ ] 不影响业务逻辑性能

### 文档
- [ ] 监控指南完整
- [ ] 所有指标有文档说明
- [ ] Prometheus 配置示例可用
- [ ] Grafana 仪表板可导入
- [ ] 告警规则配置清晰

### 部署
- [ ] Docker 镜像可构建
- [ ] docker-compose 可正常启动
- [ ] Prometheus 和 Grafana 集成正常
- [ ] 配置示例完整

## 估算

- **阶段 1-2**: Metrics 包和 Workflow/Task 指标 - 2 天
- **阶段 3-5**: Lane 和 HTTP 指标 - 1 天
- **阶段 6-8**: 系统指标和主程序集成 - 1 天
- **阶段 9-11**: Prometheus/Grafana/告警配置 - 1 天
- **阶段 12-14**: 测试、文档和部署 - 1 天
- **阶段 15-16**: 性能优化和清理 - 0.5 天

**总计**: 约 6.5 天

## 注意事项

1. **按顺序实现**: 先基础设施，再具体指标
2. **测试驱动**: 为每个指标编写测试
3. **增量提交**: 完成每个阶段后提交代码
4. **性能监控**: 持续监控指标收集开销
5. **标签控制**: 避免高基数标签导致内存问题
6. **向后兼容**: Metrics 是可选功能，不影响现有 API
7. **安全考虑**: Metrics 端口与 API 端口分离
8. **文档同步**: 代码和文档同步更新
