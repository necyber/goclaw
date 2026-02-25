# 任务清单：HTTP API 服务器

本文档列出实现 HTTP API 服务器所需的所有任务，按实现顺序分组。

## 阶段 1: 基础设施搭建

### 1.1 项目依赖管理
- [x] 添加 `github.com/go-chi/chi/v5` 到 go.mod
- [x] 添加 `github.com/go-chi/cors` 到 go.mod
- [x] 添加 `github.com/go-playground/validator/v10` 到 go.mod
- [x] 添加 `github.com/google/uuid` 到 go.mod
- [x] 运行 `go mod tidy` 确保依赖正确

### 1.2 配置扩展
- [x] 在 `config/config.go` 中添加 `ServerConfig` 结构
- [x] 添加 HTTP 服务器配置字段（host, port, timeouts）
- [x] 添加 CORS 配置字段
- [x] 更新 `config.example.yaml` 添加服务器配置示例
- [x] 添加配置验证规则

### 1.3 响应辅助函数
- [x] 创建 `pkg/api/response/json.go`
- [x] 实现 `JSON()` 函数（序列化并发送 JSON 响应）
- [x] 实现 `Error()` 函数（格式化错误响应）
- [x] 创建 `pkg/api/response/error.go`
- [x] 定义 `ErrorResponse` 和 `ErrorDetail` 结构
- [x] 实现错误类型到 HTTP 状态码的映射
- [x] 编写单元测试

### 1.4 中间件实现
- [x] 创建 `pkg/api/middleware/logger.go`
  - [x] 实现请求日志记录
  - [x] 包装 ResponseWriter 捕获状态码
  - [x] 记录请求方法、路径、状态码、响应时间
- [x] 创建 `pkg/api/middleware/recovery.go`
  - [x] 实现 panic 恢复
  - [x] 记录堆栈跟踪
  - [x] 返回 500 错误
- [x] 创建 `pkg/api/middleware/request_id.go`
  - [x] 生成或提取请求 ID
  - [x] 添加到上下文和响应头
- [x] 创建 `pkg/api/middleware/cors.go`
  - [x] 实现 CORS 处理
  - [x] 支持预检请求
  - [x] 从配置读取 CORS 设置
- [x] 创建 `pkg/api/middleware/timeout.go`
  - [x] 实现请求超时控制
  - [x] 返回 504 超时错误
- [x] 为每个中间件编写单元测试

### 1.5 路由系统
- [x] 创建 `pkg/api/router.go`
- [x] 实现 `NewRouter()` 函数创建 chi 路由器
- [x] 实现 `RegisterRoutes()` 函数注册所有路由
- [x] 定义路由结构（/api/v1/workflows, /health, /ready, /status）
- [x] 编写路由测试

### 1.6 HTTP 服务器核心
- [x] 创建 `pkg/api/server.go`
- [x] 定义 `Server` 接口
- [x] 实现 `HTTPServer` 结构
- [x] 实现 `NewHTTPServer()` 构造函数
- [x] 实现 `Start()` 方法
- [x] 实现 `Shutdown()` 方法（优雅关闭）
- [x] 编写服务器生命周期测试

## 阶段 2: 引擎集成

### 2.1 引擎接口扩展
- [x] 在 `pkg/engine/engine.go` 中添加工作流管理方法
  - [x] `SubmitWorkflow(ctx, req) (string, error)`
  - [x] `GetWorkflowStatus(ctx, id) (*WorkflowStatus, error)`
  - [x] `ListWorkflows(ctx, filter) ([]*WorkflowStatus, int, error)`
  - [x] `CancelWorkflow(ctx, id) error`
  - [x] `GetTaskResult(ctx, workflowID, taskID) (*TaskResult, error)`
- [x] 添加健康检查方法
  - [x] `IsHealthy() bool`
  - [x] `IsReady() bool`
  - [x] `GetStatus() *EngineStatus`

### 2.2 工作流状态管理
- [x] 创建 `pkg/engine/workflow.go`
- [x] 定义 `Workflow` 结构（ID, Name, Status, Tasks, Metadata）
- [x] 定义 `WorkflowStatus` 结构
- [x] 定义 `TaskStatus` 结构
- [x] 实现工作流状态转换逻辑
- [x] 添加线程安全的工作流存储（使用 sync.RWMutex）
- [x] 实现工作流 CRUD 操作

### 2.3 数据模型定义
- [x] 创建 `pkg/api/models/workflow.go`
- [x] 定义 `WorkflowRequest` 结构（带验证标签）
- [x] 定义 `WorkflowResponse` 结构
- [x] 定义 `TaskDefinition` 结构
- [x] 定义 `WorkflowFilter` 结构（用于列表查询）
- [x] 添加 JSON 序列化标签

## 阶段 3: API 处理器实现

### 3.1 工作流处理器
- [x] 创建 `pkg/api/handlers/workflow.go`
- [x] 定义 `WorkflowHandler` 结构
- [x] 实现 `NewWorkflowHandler()` 构造函数
- [x] 实现 `SubmitWorkflow()` 处理器
  - [x] 解析请求体
  - [x] 验证请求数据
  - [x] 调用引擎提交工作流
  - [x] 返回 201 Created 响应
- [x] 实现 `GetWorkflow()` 处理器
  - [x] 提取路径参数
  - [x] 验证工作流 ID
  - [x] 查询工作流状态
  - [x] 返回 200 OK 响应
- [x] 实现 `ListWorkflows()` 处理器
  - [x] 解析查询参数（status, limit, offset）
  - [x] 调用引擎列出工作流
  - [x] 返回分页响应
- [x] 实现 `CancelWorkflow()` 处理器
  - [x] 提取工作流 ID
  - [x] 调用引擎取消工作流
  - [x] 处理状态冲突（409）
- [x] 实现 `GetTaskResult()` 处理器
  - [x] 提取工作流 ID 和任务 ID
  - [x] 查询任务结果
  - [x] 返回结果或 404

### 3.2 请求验证
- [x] 初始化 validator 实例（在 WorkflowHandler 中）
- [x] 实现验证错误格式化（在处理器中）
- [ ] 创建 `pkg/api/handlers/validation.go`（可选，当前在处理器中实现）
- [ ] 实现自定义验证器（循环依赖检测）（可选）
- [ ] 实现 UUID 格式验证（可选）
- [ ] 编写验证测试（已在 workflow_test.go 中测试）

### 3.3 健康检查处理器
- [x] 创建 `pkg/api/handlers/health.go`
- [x] 定义 `HealthHandler` 结构
- [x] 实现 `NewHealthHandler()` 构造函数
- [x] 实现 `Health()` 处理器（简单存活检查）
- [x] 实现 `Ready()` 处理器（依赖项检查）
- [x] 实现 `Status()` 处理器（详细状态）
- [ ] 创建 `pkg/api/handlers/checkers/engine.go`（引擎检查器）（可选）
- [ ] 创建 `pkg/api/handlers/checkers/storage.go`（存储检查器，可选）

### 3.4 处理器测试
- [x] 为 `WorkflowHandler` 编写单元测试
  - [x] 测试成功场景
  - [x] 测试验证失败
  - [x] 测试引擎错误
  - [ ] 测试并发请求（可选）
- [x] 为 `HealthHandler` 编写单元测试
  - [x] 测试健康检查
  - [x] 测试就绪检查
  - [ ] 测试依赖项失败场景（可选）

## 阶段 4: 主程序集成

### 4.1 主程序修改
- [ ] 修改 `cmd/goclaw/main.go`
- [ ] 在配置加载后初始化 HTTP 服务器
- [ ] 创建处理器实例
- [ ] 启动 HTTP 服务器（在单独的 goroutine）
- [ ] 修改信号处理，同时关闭引擎和 HTTP 服务器
- [ ] 确保优雅关闭顺序正确

### 4.2 启动流程测试
- [ ] 测试服务器启动
- [ ] 测试配置加载
- [ ] 测试优雅关闭
- [ ] 测试信号处理

## 阶段 5: API 文档

### 5.1 Swagger 集成
- [ ] 添加 `github.com/swaggo/swag` 依赖
- [ ] 添加 `github.com/swaggo/http-swagger` 依赖
- [ ] 在 `main.go` 中添加 Swagger 注解
  - [ ] @title, @version, @description
  - [ ] @host, @BasePath
  - [ ] @contact, @license
- [ ] 为每个处理器函数添加 Swagger 注解
  - [ ] @Summary, @Description
  - [ ] @Tags, @Accept, @Produce
  - [ ] @Param, @Success, @Failure
  - [ ] @Router
- [ ] 为数据模型添加注解和示例

### 5.2 文档生成
- [ ] 运行 `swag init` 生成文档
- [ ] 创建 `docs/swagger/` 目录
- [ ] 验证生成的 `swagger.json` 和 `swagger.yaml`
- [ ] 在路由中注册 `/docs/*` 端点
- [ ] 测试 Swagger UI 访问

### 5.3 使用示例
- [ ] 创建 `docs/examples/` 目录
- [ ] 编写 cURL 示例
- [ ] 编写 Go 客户端示例
- [ ] 编写 Postman 集合（可选）

## 阶段 6: 集成测试

### 6.1 端到端测试
- [ ] 创建 `pkg/api/integration_test.go`
- [ ] 实现测试服务器启动辅助函数
- [ ] 测试完整的工作流生命周期
  - [ ] 提交工作流
  - [ ] 查询工作流状态
  - [ ] 列出工作流
  - [ ] 取消工作流
  - [ ] 获取任务结果
- [ ] 测试健康检查端点
- [ ] 测试错误场景
  - [ ] 无效请求
  - [ ] 不存在的资源
  - [ ] 状态冲突

### 6.2 并发测试
- [ ] 测试多个客户端同时提交工作流
- [ ] 测试并发查询和修改
- [ ] 验证线程安全性
- [ ] 检查数据一致性

### 6.3 性能测试
- [ ] 使用 `go test -bench` 进行基准测试
- [ ] 测试健康检查响应时间
- [ ] 测试工作流提交响应时间
- [ ] 测试状态查询响应时间
- [ ] 使用 `wrk` 或 `ab` 进行负载测试（可选）

## 阶段 7: 文档和示例

### 7.1 README 更新
- [ ] 更新项目 README.md
- [ ] 添加 HTTP API 使用说明
- [ ] 添加 API 端点列表
- [ ] 添加配置示例
- [ ] 添加快速开始指南

### 7.2 CLAUDE.md 更新
- [ ] 更新架构说明
- [ ] 添加 API 服务器相关命令
- [ ] 更新开发阶段说明

### 7.3 示例和教程
- [ ] 创建基本使用示例
- [ ] 创建高级用例示例
- [ ] 编写故障排查指南

## 阶段 8: 部署准备

### 8.1 Docker 支持
- [ ] 更新 Dockerfile 暴露 HTTP 端口
- [ ] 更新 docker-compose.yml（如果有）
- [ ] 测试容器化部署

### 8.2 Kubernetes 配置
- [ ] 创建 Deployment YAML
- [ ] 配置 liveness 和 readiness 探测
- [ ] 创建 Service YAML
- [ ] 创建 ConfigMap（可选）
- [ ] 测试 Kubernetes 部署（可选）

## 验收标准

完成所有任务后，验证以下标准：

### 功能性
- [ ] 所有 API 端点正常工作
- [ ] 请求验证按预期工作
- [ ] 错误响应格式一致
- [ ] 健康检查端点正常
- [ ] Swagger UI 可访问

### 质量
- [ ] 单元测试覆盖率 > 80%
- [ ] 所有集成测试通过
- [ ] 无已知的严重 bug
- [ ] 代码通过 `go vet` 和 `golangci-lint`

### 性能
- [ ] 健康检查响应时间 < 10ms
- [ ] 工作流提交响应时间 < 100ms
- [ ] 状态查询响应时间 < 50ms

### 文档
- [ ] API 文档完整
- [ ] 使用示例清晰
- [ ] README 更新
- [ ] 代码注释充分

### 部署
- [ ] Docker 镜像可构建
- [ ] 服务可在容器中运行
- [ ] 配置文件示例完整

## 估算

- **阶段 1**: 基础设施搭建 - 2-3 天
- **阶段 2**: 引擎集成 - 1-2 天
- **阶段 3**: API 处理器实现 - 2-3 天
- **阶段 4**: 主程序集成 - 0.5 天
- **阶段 5**: API 文档 - 1 天
- **阶段 6**: 集成测试 - 1-2 天
- **阶段 7**: 文档和示例 - 1 天
- **阶段 8**: 部署准备 - 0.5-1 天

**总计**: 约 9-13 天

## 注意事项

1. **按顺序实现**: 每个阶段依赖前一阶段的完成
2. **测试驱动**: 为每个组件编写测试
3. **增量提交**: 完成每个子任务后提交代码
4. **代码审查**: 关键组件需要审查
5. **文档同步**: 代码和文档同步更新
