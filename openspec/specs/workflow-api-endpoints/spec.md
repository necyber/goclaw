# 规范：工作流 API 端点

## 概述

实现用于工作流管理的 RESTful API 端点，包括工作流提交、状态查询、取消操作和结果检索。

## API 端点

### 1. 提交工作流

**端点**: `POST /api/v1/workflows`

**请求体**:
```json
{
  "name": "workflow-name",
  "tasks": [
    {
      "id": "task-1",
      "dependencies": []
    },
    {
      "id": "task-2",
      "dependencies": ["task-1"]
    }
  ],
  "metadata": {
    "user": "username",
    "priority": "high"
  }
}
```

**响应** (201 Created):
```json
{
  "workflow_id": "uuid",
  "status": "pending",
  "created_at": "2026-02-24T10:00:00Z",
  "message": "工作流已提交"
}
```

**错误响应**:
- 400: 请求体格式错误或任务定义无效
- 503: 引擎不可用

**验证规则**:
- `name` 必填，长度 1-100 字符
- `tasks` 必填，至少包含一个任务
- 每个任务必须有唯一的 `id`
- 依赖关系不能形成循环

### 2. 查询工作流状态

**端点**: `GET /api/v1/workflows/{workflow_id}`

**路径参数**:
- `workflow_id`: 工作流 UUID

**响应** (200 OK):
```json
{
  "workflow_id": "uuid",
  "name": "workflow-name",
  "status": "running",
  "created_at": "2026-02-24T10:00:00Z",
  "started_at": "2026-02-24T10:00:01Z",
  "completed_at": null,
  "tasks": [
    {
      "id": "task-1",
      "status": "completed",
      "started_at": "2026-02-24T10:00:01Z",
      "completed_at": "2026-02-24T10:00:05Z"
    },
    {
      "id": "task-2",
      "status": "running",
      "started_at": "2026-02-24T10:00:05Z",
      "completed_at": null
    }
  ],
  "metadata": {}
}
```

**状态值**:
- `pending`: 等待执行
- `running`: 执行中
- `completed`: 已完成
- `failed`: 失败
- `cancelled`: 已取消

**错误响应**:
- 404: 工作流不存在

### 3. 列出工作流

**端点**: `GET /api/v1/workflows`

**查询参数**:
- `status`: 按状态过滤（可选）
- `limit`: 返回数量限制（默认 50，最大 100）
- `offset`: 分页偏移量（默认 0）

**响应** (200 OK):
```json
{
  "workflows": [
    {
      "workflow_id": "uuid",
      "name": "workflow-name",
      "status": "completed",
      "created_at": "2026-02-24T10:00:00Z",
      "completed_at": "2026-02-24T10:05:00Z"
    }
  ],
  "total": 100,
  "limit": 50,
  "offset": 0
}
```

### 4. 取消工作流

**端点**: `POST /api/v1/workflows/{workflow_id}/cancel`

**路径参数**:
- `workflow_id`: 工作流 UUID

**响应** (200 OK):
```json
{
  "workflow_id": "uuid",
  "status": "cancelled",
  "message": "工作流已取消"
}
```

**错误响应**:
- 404: 工作流不存在
- 409: 工作流已完成或已取消，无法取消

### 5. 获取任务结果

**端点**: `GET /api/v1/workflows/{workflow_id}/tasks/{task_id}/result`

**路径参数**:
- `workflow_id`: 工作流 UUID
- `task_id`: 任务 ID

**响应** (200 OK):
```json
{
  "task_id": "task-1",
  "status": "completed",
  "result": {
    "output": "任务输出数据",
    "metrics": {}
  },
  "error": null
}
```

**错误响应**:
- 404: 工作流或任务不存在
- 409: 任务尚未完成

## 数据模型

### WorkflowRequest
```go
type WorkflowRequest struct {
    Name     string                 `json:"name" validate:"required,min=1,max=100"`
    Tasks    []TaskDefinition       `json:"tasks" validate:"required,min=1,dive"`
    Metadata map[string]interface{} `json:"metadata"`
}

type TaskDefinition struct {
    ID           string   `json:"id" validate:"required"`
    Dependencies []string `json:"dependencies"`
}
```

### WorkflowResponse
```go
type WorkflowResponse struct {
    WorkflowID  string                 `json:"workflow_id"`
    Name        string                 `json:"name"`
    Status      string                 `json:"status"`
    CreatedAt   time.Time              `json:"created_at"`
    StartedAt   *time.Time             `json:"started_at,omitempty"`
    CompletedAt *time.Time             `json:"completed_at,omitempty"`
    Tasks       []TaskStatus           `json:"tasks,omitempty"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
```

## 处理器实现

### 文件结构
```
pkg/api/handlers/
├── workflow.go         # 工作流处理器
├── workflow_test.go    # 单元测试
└── validation.go       # 请求验证
```

### WorkflowHandler 接口
```go
type WorkflowHandler struct {
    engine *engine.Engine
    logger *logger.Logger
}

func NewWorkflowHandler(eng *engine.Engine, log *logger.Logger) *WorkflowHandler
func (h *WorkflowHandler) SubmitWorkflow(w http.ResponseWriter, r *http.Request)
func (h *WorkflowHandler) GetWorkflow(w http.ResponseWriter, r *http.Request)
func (h *WorkflowHandler) ListWorkflows(w http.ResponseWriter, r *http.Request)
func (h *WorkflowHandler) CancelWorkflow(w http.ResponseWriter, r *http.Request)
func (h *WorkflowHandler) GetTaskResult(w http.ResponseWriter, r *http.Request)
```

## 与引擎集成

需要在 `pkg/engine/engine.go` 中添加以下方法：

```go
// 提交工作流
func (e *Engine) SubmitWorkflow(ctx context.Context, req *WorkflowRequest) (string, error)

// 查询工作流状态
func (e *Engine) GetWorkflowStatus(ctx context.Context, workflowID string) (*WorkflowStatus, error)

// 列出工作流
func (e *Engine) ListWorkflows(ctx context.Context, filter WorkflowFilter) ([]*WorkflowStatus, int, error)

// 取消工作流
func (e *Engine) CancelWorkflow(ctx context.Context, workflowID string) error

// 获取任务结果
func (e *Engine) GetTaskResult(ctx context.Context, workflowID, taskID string) (*TaskResult, error)
```

## 验证

### 请求验证
- 使用 `go-playground/validator` 进行结构体验证
- 自定义验证器检查循环依赖
- 验证工作流 ID 格式（UUID）

### 业务逻辑验证
- 工作流名称唯一性（可选）
- 任务 ID 在工作流内唯一
- 依赖的任务必须存在
- 状态转换合法性

## 测试要求

### 单元测试
- 每个端点的处理器逻辑
- 请求验证规则
- 错误响应格式
- 数据模型序列化/反序列化

### 集成测试
- 完整的工作流生命周期（提交 → 查询 → 取消）
- 并发请求处理
- 错误场景（无效输入、不存在的资源）
- 与引擎的集成

### API 测试
- 使用 `httptest` 进行端到端测试
- 验证 HTTP 状态码
- 验证响应头（Content-Type）
- 验证响应体结构

## 验收标准

- [ ] 所有 5 个端点正确实现
- [ ] 请求验证按规则工作
- [ ] 错误响应格式一致
- [ ] 与引擎集成无缝
- [ ] 支持并发请求
- [ ] 单元测试覆盖率 > 80%
- [ ] 集成测试通过
- [ ] API 文档完整
