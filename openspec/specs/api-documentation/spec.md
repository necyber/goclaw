# 规范：API 文档

## 概述

为 HTTP API 提供完整的文档，包括 OpenAPI/Swagger 规范、交互式文档界面和使用示例。

## OpenAPI 规范

### 规范版本
- 使用 OpenAPI 3.0.3 或更高版本
- 遵循 OpenAPI 规范标准

### 规范内容

**基本信息**:
```yaml
openapi: 3.0.3
info:
  title: Goclaw API
  description: 分布式多代理编排引擎 HTTP API
  version: 0.1.0
  contact:
    name: Goclaw Team
  license:
    name: MIT
servers:
  - url: http://localhost:8080/api/v1
    description: 本地开发服务器
```

**标签分组**:
- `workflows`: 工作流管理
- `health`: 健康监控

### 端点文档要求

每个端点必须包含：
- 完整的描述
- 请求参数（路径、查询、请求体）
- 响应示例（成功和错误）
- HTTP 状态码说明
- 数据模型定义

### 数据模型（Schemas）

定义所有请求和响应的数据结构：
- `WorkflowRequest`
- `WorkflowResponse`
- `TaskDefinition`
- `TaskStatus`
- `ErrorResponse`
- `HealthResponse`
- `ReadyResponse`

## 交互式文档

### Swagger UI

**端点**: `GET /docs`

**功能**:
- 提供交互式 API 文档界面
- 支持在线测试 API 端点
- 显示请求/响应示例
- 自动从 OpenAPI 规范生成

**实现方式**:
- 使用 `swaggo/swag` 生成 OpenAPI 规范
- 使用 `swaggo/http-swagger` 提供 Swagger UI
- 或使用静态文件托管 Swagger UI

### ReDoc（可选）

**端点**: `GET /redoc`

**功能**:
- 提供另一种文档界面选择
- 更适合阅读和打印
- 响应式设计

## 代码注解

### Swag 注解示例

在处理器函数上添加注解：

```go
// SubmitWorkflow godoc
// @Summary 提交工作流
// @Description 提交新的工作流进行执行
// @Tags workflows
// @Accept json
// @Produce json
// @Param workflow body WorkflowRequest true "工作流定义"
// @Success 201 {object} WorkflowResponse
// @Failure 400 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /workflows [post]
func (h *WorkflowHandler) SubmitWorkflow(w http.ResponseWriter, r *http.Request) {
    // 实现
}
```

### 数据模型注解

```go
// WorkflowRequest 工作流提交请求
type WorkflowRequest struct {
    // 工作流名称
    Name string `json:"name" example:"my-workflow"`
    // 任务列表
    Tasks []TaskDefinition `json:"tasks"`
    // 元数据
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}
```

## 文档生成

### 生成命令

```bash
# 安装 swag
go install github.com/swaggo/swag/cmd/swag@latest

# 生成文档
swag init -g cmd/goclaw/main.go -o docs/swagger

# 输出文件
# - docs/swagger/swagger.json
# - docs/swagger/swagger.yaml
# - docs/swagger/docs.go
```

### 集成到服务器

```go
import (
    httpSwagger "github.com/swaggo/http-swagger"
    _ "goclaw/docs/swagger" // 导入生成的文档
)

// 注册路由
router.Get("/docs/*", httpSwagger.WrapHandler)
```

## 文件结构

```
docs/
├── swagger/
│   ├── swagger.json    # OpenAPI JSON 规范
│   ├── swagger.yaml    # OpenAPI YAML 规范
│   └── docs.go         # 生成的 Go 代码
└── examples/
    ├── submit_workflow.json
    └── workflow_response.json
```

## 使用示例

### cURL 示例

```bash
# 提交工作流
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "example-workflow",
    "tasks": [
      {"id": "task-1", "dependencies": []},
      {"id": "task-2", "dependencies": ["task-1"]}
    ]
  }'

# 查询工作流状态
curl http://localhost:8080/api/v1/workflows/{workflow_id}
```

### Go 客户端示例

```go
// 提交工作流
req := &WorkflowRequest{
    Name: "example-workflow",
    Tasks: []TaskDefinition{
        {ID: "task-1", Dependencies: []string{}},
        {ID: "task-2", Dependencies: []string{"task-1"}},
    },
}

resp, err := http.Post(
    "http://localhost:8080/api/v1/workflows",
    "application/json",
    bytes.NewBuffer(jsonData),
)
```

## 依赖

- `github.com/swaggo/swag` - OpenAPI 规范生成
- `github.com/swaggo/http-swagger` - Swagger UI 集成

## 测试要求

### 文档验证
- OpenAPI 规范有效性
- 所有端点都有文档
- 示例数据正确
- 数据模型完整

### 可访问性测试
- `/docs` 端点可访问
- Swagger UI 正常加载
- 可以执行测试请求

## 验收标准

- [ ] OpenAPI 规范完整且有效
- [ ] 所有端点都有详细文档
- [ ] Swagger UI 可访问并正常工作
- [ ] 包含请求/响应示例
- [ ] 数据模型定义完整
- [ ] 提供使用示例（cURL、Go）
- [ ] 文档自动生成流程正常
- [ ] 文档与实际 API 一致
