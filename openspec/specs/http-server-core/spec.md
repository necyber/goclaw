# 规范：HTTP 服务器核心

## 概述

实现 Goclaw 的核心 HTTP 服务器基础设施，包括服务器生命周期管理、路由系统、中间件链和优雅关闭机制。

## 功能需求

### 1. HTTP 服务器初始化

**输入**:
- 配置对象（端口、超时、TLS 设置等）
- 引擎实例引用

**输出**:
- 已配置的 HTTP 服务器实例

**行为**:
- 从配置加载服务器参数（监听地址、端口、超时）
- 初始化路由器
- 注册中间件链
- 绑定到指定端口
- 支持 HTTP 和 HTTPS（可选）

### 2. 路由系统

**需求**:
- 支持 RESTful 路由模式（GET、POST、PUT、DELETE、PATCH）
- 路径参数提取（如 `/workflows/{id}`）
- 查询参数解析
- 路由分组（如 `/api/v1/...`）
- 404 和 405 处理

**推荐实现**:
- 使用 `chi` 路由器（轻量、标准库兼容）
- 或使用标准库 `net/http` 的 `ServeMux`（Go 1.22+）

### 3. 中间件链

**必需中间件**:
- **日志中间件**: 记录请求方法、路径、状态码、响应时间
- **恢复中间件**: 捕获 panic，返回 500 错误
- **CORS 中间件**: 配置跨域资源共享策略
- **请求 ID 中间件**: 为每个请求生成唯一 ID
- **超时中间件**: 设置请求处理超时

**可选中间件**:
- 速率限制
- 认证/授权（为未来扩展预留）
- 压缩（gzip）

### 4. 优雅关闭

**需求**:
- 监听系统信号（SIGINT、SIGTERM）
- 停止接受新连接
- 等待现有请求完成（带超时）
- 清理资源
- 记录关闭事件

**实现**:
```go
// 伪代码
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
server.Shutdown(ctx)
```

### 5. 错误处理

**标准错误响应格式**:
```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "人类可读的错误消息",
    "details": {},
    "request_id": "uuid"
  }
}
```

**HTTP 状态码映射**:
- 400: 请求参数错误
- 404: 资源未找到
- 405: 方法不允许
- 500: 内部服务器错误
- 503: 服务不可用

## 配置

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

### 配置验证

- `port` 必须在 1-65535 范围内
- 超时值必须为正数
- CORS 配置必须有效

## 接口设计

### Server 接口

```go
type Server interface {
    Start() error
    Shutdown(ctx context.Context) error
    RegisterRoutes(router chi.Router)
}
```

### HTTPServer 实现

```go
type HTTPServer struct {
    config *config.ServerConfig
    engine *engine.Engine
    server *http.Server
    router chi.Router
    logger *logger.Logger
}

func NewHTTPServer(cfg *config.ServerConfig, eng *engine.Engine, log *logger.Logger) *HTTPServer
```

## 文件结构

```
pkg/api/
├── server.go           # HTTPServer 实现
├── middleware/
│   ├── logger.go       # 日志中间件
│   ├── recovery.go     # 恢复中间件
│   ├── cors.go         # CORS 中间件
│   ├── request_id.go   # 请求 ID 中间件
│   └── timeout.go      # 超时中间件
├── response/
│   ├── json.go         # JSON 响应辅助函数
│   └── error.go        # 错误响应格式化
└── router.go           # 路由注册
```

## 依赖

- `github.com/go-chi/chi/v5` - HTTP 路由器
- `github.com/go-chi/cors` - CORS 中间件
- 现有的 `pkg/logger` - 日志记录
- 现有的 `config` - 配置管理
- 现有的 `pkg/engine` - 引擎接口

## 测试要求

### 单元测试
- 服务器初始化
- 中间件功能
- 错误响应格式化
- 配置验证

### 集成测试
- 服务器启动和关闭
- 优雅关闭流程
- 中间件链执行顺序
- 基本路由功能

## 验收标准

- [ ] HTTP 服务器可以在配置的端口上启动
- [ ] 所有中间件正确应用到请求链
- [ ] 优雅关闭在超时内完成
- [ ] 错误响应格式一致
- [ ] CORS 头正确设置
- [ ] 请求日志包含所有必需字段
- [ ] 单元测试覆盖率 > 80%
- [ ] 集成测试通过
