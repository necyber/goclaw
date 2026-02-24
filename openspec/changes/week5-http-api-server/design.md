# 设计文档：HTTP API 服务器

## 概述

本文档描述 Goclaw HTTP API 服务器的技术设计，整合了四个核心功能模块的规范，提供统一的实现策略。

## 架构设计

### 整体架构

```
┌─────────────────────────────────────────────────────────┐
│                     HTTP 客户端                          │
└─────────────────────┬───────────────────────────────────┘
                      │ HTTP/REST
┌─────────────────────▼───────────────────────────────────┐
│                  HTTP API 层                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │   中间件链    │  │  路由系统     │  │  处理器层     │  │
│  │  - 日志       │  │  - chi路由    │  │  - Workflow  │  │
│  │  - 恢复       │  │  - 版本控制   │  │  - Health    │  │
│  │  - CORS      │  │  - 路径匹配   │  │  - 验证      │  │
│  │  - 请求ID     │  │              │  │              │  │
│  │  - 超时       │  │              │  │              │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
└─────────────────────┬───────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────┐
│                  编排引擎层                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │   Engine     │  │   DAG编译     │  │   Lane队列    │  │
│  │  - 工作流管理 │  │  - 依赖分析   │  │  - 任务执行   │  │
│  │  - 状态跟踪   │  │  - 拓扑排序   │  │  - 并发控制   │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────┘
```

### 分层职责

**API 层**:
- HTTP 请求/响应处理
- 请求验证和错误处理
- 中间件链执行
- 路由分发

**引擎层**:
- 工作流生命周期管理
- DAG 编译和执行计划生成
- 任务调度和状态跟踪
- 存储抽象

## 核心组件设计

### 1. HTTP 服务器 (pkg/api/server.go)

**职责**:
- 管理 HTTP 服务器生命周期
- 配置路由和中间件
- 处理优雅关闭

**接口**:
```go
type Server interface {
    Start() error
    Shutdown(ctx context.Context) error
}

type HTTPServer struct {
    config  *config.ServerConfig
    engine  *engine.Engine
    server  *http.Server
    router  chi.Router
    logger  *logger.Logger
}
```

**初始化流程**:
1. 加载配置
2. 创建 chi 路由器
3. 注册全局中间件
4. 注册路由和处理器
5. 创建 http.Server
6. 启动监听

**关闭流程**:
1. 接收关闭信号
2. 停止接受新连接
3. 等待现有请求完成（30秒超时）
4. 清理资源
5. 记录关闭日志

### 2. 中间件链 (pkg/api/middleware/)

**执行顺序**:
```
请求 → 日志 → 恢复 → 请求ID → CORS → 超时 → 处理器 → 响应
```

**各中间件职责**:

**日志中间件** (logger.go):
- 记录请求开始时间
- 包装 ResponseWriter 捕获状态码
- 记录请求方法、路径、状态码、响应时间、请求ID

**恢复中间件** (recovery.go):
- 捕获 panic
- 记录堆栈跟踪
- 返回 500 错误响应
- 防止服务器崩溃

**请求ID中间件** (request_id.go):
- 生成或提取请求ID（UUID）
- 添加到上下文和响应头
- 用于请求追踪

**CORS中间件** (cors.go):
- 处理预检请求（OPTIONS）
- 设置 CORS 响应头
- 支持配置允许的源、方法、头

**超时中间件** (timeout.go):
- 为请求设置超时上下文
- 超时时取消请求
- 返回 504 Gateway Timeout

### 3. 路由系统 (pkg/api/router.go)

**路由结构**:
```go
/api/v1/
├── workflows/
│   ├── POST   /                    # 提交工作流
│   ├── GET    /                    # 列出工作流
│   ├── GET    /{id}                # 查询工作流
│   ├── POST   /{id}/cancel         # 取消工作流
│   └── GET    /{id}/tasks/{tid}/result  # 获取任务结果
├── /health                         # 健康检查
├── /ready                          # 就绪检查
├── /status                         # 详细状态
└── /docs/*                         # Swagger UI
```

**路由注册**:
```go
func RegisterRoutes(r chi.Router, handlers *Handlers) {
    r.Route("/api/v1", func(r chi.Router) {
        // 工作流路由
        r.Route("/workflows", func(r chi.Router) {
            r.Post("/", handlers.Workflow.SubmitWorkflow)
            r.Get("/", handlers.Workflow.ListWorkflows)
            r.Get("/{id}", handlers.Workflow.GetWorkflow)
            r.Post("/{id}/cancel", handlers.Workflow.CancelWorkflow)
            r.Get("/{id}/tasks/{tid}/result", handlers.Workflow.GetTaskResult)
        })
    })

    // 健康检查路由（不带版本前缀）
    r.Get("/health", handlers.Health.Health)
    r.Get("/ready", handlers.Health.Ready)
    r.Get("/status", handlers.Health.Status)

    // 文档路由
    r.Get("/docs/*", httpSwagger.WrapHandler)
}
```

### 4. 处理器层 (pkg/api/handlers/)

**WorkflowHandler** (workflow.go):
- 处理工作流相关的所有端点
- 请求验证和数据转换
- 调用引擎方法
- 格式化响应

**HealthHandler** (health.go):
- 执行健康检查器
- 聚合检查结果
- 返回健康状态

**响应辅助函数** (response/):
- JSON 序列化
- 错误响应格式化
- 状态码设置

### 5. 引擎集成

**需要在 Engine 中添加的方法**:

```go
// 工作流管理
type WorkflowManager interface {
    SubmitWorkflow(ctx context.Context, req *WorkflowRequest) (string, error)
    GetWorkflowStatus(ctx context.Context, id string) (*WorkflowStatus, error)
    ListWorkflows(ctx context.Context, filter WorkflowFilter) ([]*WorkflowStatus, int, error)
    CancelWorkflow(ctx context.Context, id string) error
    GetTaskResult(ctx context.Context, workflowID, taskID string) (*TaskResult, error)
}

// 健康检查
type HealthChecker interface {
    IsHealthy() bool
    IsReady() bool
    GetStatus() *EngineStatus
}
```

**工作流状态管理**:
- 使用内存存储（Phase 1）
- 工作流ID使用UUID
- 状态：pending → running → completed/failed/cancelled
- 线程安全的状态访问（使用 sync.RWMutex）

## 数据流设计

### 提交工作流流程

```
1. 客户端 POST /api/v1/workflows
   ↓
2. 中间件链处理（日志、验证等）
   ↓
3. WorkflowHandler.SubmitWorkflow
   ↓
4. 请求验证（结构、循环依赖）
   ↓
5. Engine.SubmitWorkflow
   ↓
6. DAG 编译和验证
   ↓
7. 生成执行计划
   ↓
8. 保存工作流状态
   ↓
9. 提交到 Lane 队列
   ↓
10. 返回工作流ID和状态
```

### 查询工作流流程

```
1. 客户端 GET /api/v1/workflows/{id}
   ↓
2. 中间件链处理
   ↓
3. WorkflowHandler.GetWorkflow
   ↓
4. 验证工作流ID格式
   ↓
5. Engine.GetWorkflowStatus
   ↓
6. 从存储读取状态
   ↓
7. 格式化响应
   ↓
8. 返回工作流详情
```

## 错误处理策略

### 错误分类

**客户端错误 (4xx)**:
- 400: 请求格式错误、验证失败
- 404: 资源不存在
- 405: 方法不允许
- 409: 状态冲突（如取消已完成的工作流）

**服务器错误 (5xx)**:
- 500: 内部错误、panic
- 503: 服务不可用（引擎未就绪）
- 504: 请求超时

### 错误响应格式

```go
type ErrorResponse struct {
    Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
    Code      string                 `json:"code"`
    Message   string                 `json:"message"`
    Details   map[string]interface{} `json:"details,omitempty"`
    RequestID string                 `json:"request_id"`
}
```

### 错误处理流程

1. 捕获错误
2. 分类错误类型
3. 记录错误日志
4. 格式化错误响应
5. 设置适当的状态码
6. 返回给客户端

## 并发和线程安全

### 并发场景

1. **多个客户端同时提交工作流**
   - Engine 使用互斥锁保护工作流映射
   - 每个工作流独立执行

2. **查询和修改同一工作流**
   - 使用 RWMutex：读多写少
   - 读操作使用 RLock
   - 写操作使用 Lock

3. **优雅关闭期间的请求**
   - 使用 context 传播取消信号
   - 等待现有请求完成
   - 拒绝新请求

### 线程安全设计

```go
type Engine struct {
    mu        sync.RWMutex
    workflows map[string]*Workflow
    // ...
}

func (e *Engine) GetWorkflowStatus(ctx context.Context, id string) (*WorkflowStatus, error) {
    e.mu.RLock()
    defer e.mu.RUnlock()

    wf, exists := e.workflows[id]
    if !exists {
        return nil, ErrWorkflowNotFound
    }
    return wf.Status(), nil
}
```

## 配置设计

### 配置结构

```yaml
server:
  http:
    enabled: true
    host: "0.0.0.0"
    port: 8080
    read_timeout: 30s
    write_timeout: 30s
    idle_timeout: 120s
    shutdown_timeout: 30s
  cors:
    enabled: true
    allowed_origins: ["*"]
    allowed_methods: ["GET", "POST", "PUT", "DELETE", "PATCH"]
    allowed_headers: ["Content-Type", "Authorization"]
    max_age: 3600
```

### 配置加载优先级

1. 默认值
2. 配置文件 (config.yaml)
3. 环境变量 (GOCLAW_SERVER_HTTP_PORT)
4. 命令行参数 (--server.http.port)

## 测试策略

### 单元测试

**中间件测试**:
- 使用 httptest.NewRecorder
- 验证中间件行为
- 测试错误场景

**处理器测试**:
- Mock Engine 接口
- 测试请求验证
- 测试响应格式

**示例**:
```go
func TestSubmitWorkflow(t *testing.T) {
    mockEngine := &MockEngine{}
    handler := NewWorkflowHandler(mockEngine, logger)

    req := httptest.NewRequest("POST", "/api/v1/workflows", body)
    w := httptest.NewRecorder()

    handler.SubmitWorkflow(w, req)

    assert.Equal(t, 201, w.Code)
}
```

### 集成测试

**完整流程测试**:
- 启动真实的 HTTP 服务器
- 使用真实的 Engine
- 测试端到端流程

**并发测试**:
- 多个 goroutine 同时发送请求
- 验证线程安全性
- 检查数据一致性

### API 测试

**使用工具**:
- Postman/Insomnia 集合
- 自动化 API 测试脚本
- 性能测试（wrk、ab）

## 部署考虑

### 容器化

**Dockerfile**:
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o goclaw cmd/goclaw/main.go

FROM alpine:latest
COPY --from=builder /app/goclaw /usr/local/bin/
EXPOSE 8080 9090
CMD ["goclaw", "start"]
```

### Kubernetes 部署

**Deployment**:
- 配置 liveness 和 readiness 探测
- 设置资源限制
- 配置环境变量

**Service**:
- ClusterIP 用于内部访问
- LoadBalancer 用于外部访问

### 监控和日志

**日志**:
- 结构化日志（JSON 格式）
- 包含请求ID用于追踪
- 记录关键操作和错误

**指标**（未来）:
- 请求计数和延迟
- 工作流提交/完成率
- 错误率

## 安全考虑

### 当前阶段（Phase 1）

- CORS 配置
- 请求大小限制
- 超时保护
- Panic 恢复

### 未来增强（Phase 2+）

- 认证（JWT、API Key）
- 授权（RBAC）
- 速率限制
- TLS/HTTPS
- 输入清理和验证

## 性能优化

### 响应时间目标

- 健康检查: < 10ms
- 工作流提交: < 100ms
- 状态查询: < 50ms

### 优化策略

1. **连接池**: 复用 HTTP 连接
2. **缓存**: 缓存频繁查询的数据
3. **批量操作**: 支持批量提交（未来）
4. **异步处理**: 工作流提交立即返回
5. **索引**: 优化状态查询（使用存储索引）

## 实现顺序

### 阶段 1: 基础设施
1. HTTP 服务器框架
2. 中间件链
3. 路由系统
4. 响应辅助函数

### 阶段 2: 核心功能
1. 工作流处理器
2. 引擎集成方法
3. 请求验证

### 阶段 3: 监控和文档
1. 健康检查端点
2. Swagger 文档
3. 使用示例

### 阶段 4: 测试和优化
1. 单元测试
2. 集成测试
3. 性能测试
4. 文档完善

## 依赖管理

### 外部依赖

```go
require (
    github.com/go-chi/chi/v5 v5.0.10
    github.com/go-chi/cors v1.2.1
    github.com/go-playground/validator/v10 v10.15.5
    github.com/google/uuid v1.4.0
    github.com/swaggo/swag v1.16.2
    github.com/swaggo/http-swagger v1.3.4
)
```

### 版本控制

- 使用 Go modules
- 固定主要依赖版本
- 定期更新安全补丁

## 总结

本设计文档提供了 HTTP API 服务器的完整技术方案，涵盖架构、组件、数据流、错误处理、并发、测试和部署等方面。实现将分阶段进行，优先完成核心功能，然后逐步添加监控、文档和优化。
