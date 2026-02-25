# 提案：第五周 — HTTP API 服务器

## 为什么

Goclaw 目前拥有 CLI 接口和核心编排引擎，但缺少供外部客户端与系统交互的 HTTP API。HTTP API 服务器对于以下方面至关重要：

- 实现对工作流编排的程序化访问
- 支持 Web UI 和外部集成
- 提供用于任务提交、状态查询和结果检索的 RESTful 端点
- 促进客户端通过 HTTP 通信的分布式部署

这与 CLAUDE.md 中提到的第三阶段开发路线图相符。

## 变更内容

为 Goclaw 添加完整的 HTTP API 服务器，包含以下组件：

1. 具有优雅关闭功能的 HTTP 服务器基础设施
2. 用于工作流操作的 RESTful API 端点
3. 请求验证和错误处理
4. API 文档（OpenAPI/Swagger）
5. 与现有引擎和 Lane 组件的集成

## 功能模块

### http-server-core
核心 HTTP 服务器设置，包括路由、中间件和生命周期管理。

### workflow-api-endpoints
用于工作流提交、状态查询、取消和结果检索的 RESTful 端点。

### health-monitoring-endpoints
用于部署编排的健康检查和就绪状态端点。

### api-documentation
OpenAPI/Swagger 规范和文档生成。

## 影响范围

### 新增
- 新增 `pkg/api/` 包，用于 HTTP 服务器和处理器
- `config/config.yaml` 中的 HTTP 服务器配置
- API 路由定义和中间件
- OpenAPI 规范文件

### 修改
- `cmd/goclaw/main.go` - 在引擎旁初始化并启动 HTTP 服务器
- `pkg/engine/` - 添加支持 API 操作的方法（提交、查询、取消）
- 配置加载以包含 HTTP 服务器设置

### 删除
- 无

## 备注

- HTTP 服务器应在可配置端口上运行（默认 :8080，根据配置）
- 使用标准库 `net/http` 或轻量级路由器如 `chi` 或 `gorilla/mux`
- 确保对引擎状态的线程安全访问
- 遵循 RESTful 约定进行端点设计
- 包含适当的 CORS 处理以支持 Web UI 集成
