# 规范：健康监控端点

## 概述

实现用于系统健康检查和就绪状态监控的端点，支持 Kubernetes 等容器编排平台的健康探测。

## API 端点

### 1. 健康检查

**端点**: `GET /health`

**用途**: 检查服务是否存活（liveness probe）

**响应** (200 OK):
```json
{
  "status": "healthy",
  "timestamp": "2026-02-24T10:00:00Z"
}
```

**响应** (503 Service Unavailable):
```json
{
  "status": "unhealthy",
  "timestamp": "2026-02-24T10:00:00Z",
  "error": "服务不可用"
}
```

**检查项**:
- HTTP 服务器正在运行
- 基本响应能力

**特点**:
- 轻量级，快速响应
- 不检查依赖项
- 用于判断是否需要重启服务

### 2. 就绪检查

**端点**: `GET /ready`

**用途**: 检查服务是否准备好接收流量（readiness probe）

**响应** (200 OK):
```json
{
  "status": "ready",
  "timestamp": "2026-02-24T10:00:00Z",
  "checks": {
    "engine": "ok",
    "storage": "ok"
  }
}
```

**响应** (503 Service Unavailable):
```json
{
  "status": "not_ready",
  "timestamp": "2026-02-24T10:00:00Z",
  "checks": {
    "engine": "ok",
    "storage": "failed"
  },
  "error": "存储未就绪"
}
```

**检查项**:
- 引擎状态（是否已初始化）
- 存储连接（如果已配置）
- 关键依赖项可用性

**特点**:
- 检查依赖项状态
- 失败时服务不接收流量
- 用于滚动更新和负载均衡

### 3. 详细状态（可选）

**端点**: `GET /status`

**用途**: 获取详细的系统状态信息

**响应** (200 OK):
```json
{
  "status": "running",
  "version": "0.1.0",
  "uptime": "2h30m15s",
  "timestamp": "2026-02-24T10:00:00Z",
  "components": {
    "engine": {
      "status": "running",
      "workflows_active": 5,
      "workflows_total": 100
    },
    "lanes": {
      "status": "ok",
      "total_lanes": 3,
      "active_workers": 10
    },
    "storage": {
      "status": "connected",
      "type": "memory"
    }
  },
  "system": {
    "goroutines": 50,
    "memory_mb": 128
  }
}
```

**特点**:
- 提供详细的运行时信息
- 用于监控和调试
- 可能包含敏感信息，考虑访问控制

## 实现设计

### HealthChecker 接口

```go
type HealthChecker interface {
    Check(ctx context.Context) error
    Name() string
}
```

### 内置检查器

```go
// 引擎健康检查
type EngineHealthChecker struct {
    engine *engine.Engine
}

// 存储健康检查
type StorageHealthChecker struct {
    storage storage.Storage
}
```

### HealthHandler 实现

```go
type HealthHandler struct {
    checkers []HealthChecker
    startTime time.Time
    version string
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request)
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request)
func (h *HealthHandler) Status(w http.ResponseWriter, r *http.Request)
```

## 文件结构

```
pkg/api/handlers/
├── health.go           # 健康检查处理器
├── health_test.go      # 单元测试
└── checkers/
    ├── engine.go       # 引擎检查器
    └── storage.go      # 存储检查器
```

## Kubernetes 集成示例

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 2
```

## 测试要求

### 单元测试
- 健康检查逻辑
- 就绪检查逻辑
- 检查器接口实现
- 响应格式

### 集成测试
- 端点可访问性
- 状态码正确性
- 依赖项失败场景
- 超时处理

## 验收标准

- [ ] `/health` 端点始终快速响应
- [ ] `/ready` 端点正确反映依赖项状态
- [ ] `/status` 端点提供详细信息
- [ ] 响应格式一致
- [ ] 支持超时控制
- [ ] 单元测试覆盖率 > 80%
- [ ] 与 Kubernetes 探测兼容
