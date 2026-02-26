# Week 10: 监控与可观测性 - 设计文档

## 概述

为 Goclaw 添加 Prometheus 指标监控，提供生产级可观测性。

## 架构设计

### 1. 指标分层

```
┌─────────────────────────────────────────┐
│         Prometheus Metrics              │
├─────────────────────────────────────────┤
│  HTTP API Metrics  │  Engine Metrics    │
│  - Request rate    │  - Workflow count  │
│  - Latency         │  - Task execution  │
│  - Error rate      │  - Success/Failure │
├────────────────────┴────────────────────┤
│  Lane Metrics      │  System Metrics    │
│  - Queue depth     │  - Goroutines      │
│  - Wait time       │  - Memory usage    │
│  - Throughput      │  - GC stats        │
└─────────────────────────────────────────┘
```

### 2. 核心指标

#### Workflow 指标
```go
// Counter: 工作流提交总数
workflow_submissions_total{status="pending|running|completed|failed"}

// Histogram: 工作流执行时长
workflow_duration_seconds{status="completed|failed"}

// Gauge: 当前活跃工作流数
workflow_active_count{status="pending|running"}
```

#### Task 指标
```go
// Counter: 任务执行总数
task_executions_total{status="completed|failed"}

// Histogram: 任务执行时长
task_duration_seconds{task_name="..."}

// Counter: 任务重试次数
task_retries_total{task_name="..."}
```

#### Lane 指标
```go
// Gauge: 队列深度
lane_queue_depth{lane_name="default"}

// Histogram: 任务等待时长
lane_wait_duration_seconds{lane_name="default"}

// Counter: 队列吞吐量
lane_throughput_total{lane_name="default"}
```

#### HTTP API 指标
```go
// Counter: HTTP 请求总数
http_requests_total{method="GET|POST",path="/api/v1/workflows",status="200|404|500"}

// Histogram: HTTP 请求延迟
http_request_duration_seconds{method="GET|POST",path="/api/v1/workflows"}

// Gauge: 当前活跃连接数
http_active_connections
```

### 3. 组件设计

#### 3.1 Metrics 包结构

```
pkg/metrics/
├── metrics.go          # 指标注册和管理
├── workflow.go         # 工作流指标
├── task.go            # 任务指标
├── lane.go            # Lane 指标
├── http.go            # HTTP 指标
└── collector.go       # 自定义收集器
```

#### 3.2 Metrics Manager

```go
type MetricsManager struct {
    registry *prometheus.Registry

    // Workflow metrics
    workflowSubmissions *prometheus.CounterVec
    workflowDuration    *prometheus.HistogramVec
    workflowActive      *prometheus.GaugeVec

    // Task metrics
    taskExecutions      *prometheus.CounterVec
    taskDuration        *prometheus.HistogramVec
    taskRetries         *prometheus.CounterVec

    // Lane metrics
    laneQueueDepth      *prometheus.GaugeVec
    laneWaitDuration    *prometheus.HistogramVec
    laneThroughput      *prometheus.CounterVec

    // HTTP metrics
    httpRequests        *prometheus.CounterVec
    httpDuration        *prometheus.HistogramVec
    httpConnections     prometheus.Gauge
}
```

### 4. 集成点

#### 4.1 Engine 集成

```go
// pkg/engine/engine.go
func (e *Engine) SubmitWorkflowRequest(ctx context.Context, req *models.WorkflowRequest) (string, error) {
    // 记录工作流提交
    e.metrics.RecordWorkflowSubmission("pending")

    // 执行业务逻辑
    id, err := e.submitWorkflow(ctx, req)

    return id, err
}

func (e *Engine) executeWorkflow(ctx context.Context, wf *Workflow) error {
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        status := "completed"
        if err != nil {
            status = "failed"
        }
        e.metrics.RecordWorkflowDuration(status, duration)
    }()

    // 执行工作流
    return e.runWorkflow(ctx, wf)
}
```

#### 4.2 Lane 集成

```go
// pkg/lane/channel_lane.go
func (l *ChannelLane) Submit(ctx context.Context, task TaskFunc) error {
    // 记录队列深度
    l.metrics.SetQueueDepth(l.name, len(l.queue))

    start := time.Now()

    // 提交任务
    select {
    case l.queue <- task:
        // 记录等待时长
        waitDuration := time.Since(start)
        l.metrics.RecordWaitDuration(l.name, waitDuration)
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

#### 4.3 HTTP Middleware

```go
// pkg/api/middleware/metrics.go
func Metrics(metrics *metrics.MetricsManager) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()

            // 记录活跃连接
            metrics.IncActiveConnections()
            defer metrics.DecActiveConnections()

            // 包装 ResponseWriter 以捕获状态码
            wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

            // 执行请求
            next.ServeHTTP(wrapped, r)

            // 记录指标
            duration := time.Since(start)
            metrics.RecordHTTPRequest(r.Method, r.URL.Path, wrapped.statusCode, duration)
        })
    }
}
```

### 5. 配置

```yaml
# config.yaml
metrics:
  enabled: true
  port: 9091
  path: /metrics

  # 指标配置
  workflow:
    enabled: true
    duration_buckets: [0.1, 0.5, 1, 2, 5, 10, 30, 60]

  task:
    enabled: true
    duration_buckets: [0.01, 0.05, 0.1, 0.5, 1, 5, 10]

  lane:
    enabled: true
    queue_depth_limit: 10000

  http:
    enabled: true
    duration_buckets: [0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1]
```

### 6. Prometheus 配置

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'goclaw'
    scrape_interval: 15s
    static_configs:
      - targets: ['localhost:9091']
        labels:
          service: 'goclaw'
          environment: 'production'
```

### 7. Grafana 仪表板

提供预配置的 Grafana 仪表板模板：

**仪表板 1: 工作流概览**
- 工作流提交率（QPS）
- 工作流成功率
- 工作流执行时长（P50, P95, P99）
- 活跃工作流数量

**仪表板 2: 任务执行**
- 任务执行率
- 任务失败率
- 任务重试次数
- 任务执行时长分布

**仪表板 3: Lane 队列**
- 队列深度趋势
- 任务等待时长
- 队列吞吐量
- 队列饱和度

**仪表板 4: HTTP API**
- 请求率（按端点）
- 请求延迟（P50, P95, P99）
- 错误率（4xx, 5xx）
- 活跃连接数

### 8. 告警规则

```yaml
# alerts.yml
groups:
  - name: goclaw
    interval: 30s
    rules:
      # 工作流失败率过高
      - alert: HighWorkflowFailureRate
        expr: |
          rate(workflow_submissions_total{status="failed"}[5m])
          /
          rate(workflow_submissions_total[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "工作流失败率超过 10%"

      # Lane 队列积压
      - alert: LaneQueueBacklog
        expr: lane_queue_depth > 1000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Lane 队列深度超过 1000"

      # HTTP API 高延迟
      - alert: HighAPILatency
        expr: |
          histogram_quantile(0.95,
            rate(http_request_duration_seconds_bucket[5m])
          ) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "API P95 延迟超过 1 秒"

      # HTTP 错误率过高
      - alert: HighHTTPErrorRate
        expr: |
          rate(http_requests_total{status=~"5.."}[5m])
          /
          rate(http_requests_total[5m]) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "HTTP 5xx 错误率超过 5%"
```

## 实现计划

### Phase 1: 基础指标（2 天）
1. 创建 metrics 包
2. 实现 MetricsManager
3. 添加 Workflow 指标
4. 添加 Task 指标

### Phase 2: 队列和 HTTP 指标（1 天）
1. 添加 Lane 指标
2. 实现 HTTP metrics middleware
3. 集成到 API server

### Phase 3: 配置和部署（1 天）
1. 添加 metrics 配置
2. 启动 metrics server
3. 编写 Prometheus 配置
4. 创建 Grafana 仪表板

### Phase 4: 测试和文档（1 天）
1. 单元测试
2. 集成测试
3. 性能测试
4. 文档编写

## 性能考虑

1. **低开销**：指标收集使用原子操作，开销 <1%
2. **非阻塞**：指标记录不阻塞业务逻辑
3. **内存控制**：限制指标基数，避免内存泄漏
4. **可配置**：支持禁用特定指标以降低开销

## 安全考虑

1. **端口隔离**：metrics 端口与 API 端口分离
2. **访问控制**：可配置 metrics 端点的访问限制
3. **敏感信息**：不在指标标签中包含敏感数据

## 兼容性

- 向后兼容：metrics 是可选功能，不影响现有 API
- Prometheus 兼容：遵循 Prometheus 最佳实践
- Grafana 兼容：提供标准 JSON 仪表板格式
