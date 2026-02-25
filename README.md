# Goclaw 🦀

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go&logoColor=white" alt="Go Version">
  <img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg" alt="License">
  <img src="https://img.shields.io/github/actions/workflow/status/goclaw/goclaw/ci.yml?branch=main" alt="CI">
  <img src="https://img.shields.io/codecov/c/github/goclaw/goclaw" alt="Coverage">
</p>

<p align="center">
  <a href="#english">English</a> | 
  <a href="#chinese">中文</a>
</p>

---

<a name="english"></a>
## 🌟 Overview

**Goclaw** is a production-grade, high-performance, distributed-ready multi-Agent orchestration engine written in Go.

It provides a robust framework for building, deploying, and managing intelligent agents that can collaborate seamlessly in distributed environments.

### Key Features

- 🚀 **High Performance** - Built with Go's concurrency model for maximum throughput
- 🏗️ **Distributed Architecture** - Native support for cluster deployment and service discovery
- 🔄 **Agent Orchestration** - Advanced workflow management and task scheduling
- 🌐 **RESTful API** - Complete HTTP API with Swagger documentation
- 📊 **Observability** - Built-in metrics, logging, and tracing support
- 🔌 **Extensible** - Plugin architecture for custom agent behaviors
- 🛡️ **Production Ready** - Comprehensive error handling and fault tolerance

### Quick Start

```bash
# Clone the repository
git clone https://github.com/goclaw/goclaw.git
cd goclaw

# Build the project
make build

# Run tests
make test

# Start the server
make run
```

### Installation

```bash
go get github.com/goclaw/goclaw
```

### Usage Example

```go
package main

import (
    "context"
    "github.com/goclaw/goclaw/pkg/engine"
    "github.com/goclaw/goclaw/config"
    "github.com/goclaw/goclaw/pkg/logger"
)

func main() {
    // Load configuration
    cfg, err := config.Load("config.yaml", nil)
    if err != nil {
        panic(err)
    }

    // Initialize logger
    log := logger.New(&logger.Config{
        Level:  logger.InfoLevel,
        Format: "json",
        Output: "stdout",
    })

    // Create orchestration engine
    eng, err := engine.New(cfg, log)
    if err != nil {
        panic(err)
    }

    // Start the engine
    ctx := context.Background()
    if err := eng.Start(ctx); err != nil {
        panic(err)
    }
    defer eng.Stop(ctx)

    // HTTP API server is now running on port 8080
    // Access Swagger UI at http://localhost:8080/swagger/index.html
}
```

### HTTP API

Goclaw provides a complete RESTful API for workflow management:

#### API Endpoints

**Workflow Management:**
- `POST /api/v1/workflows` - Submit a new workflow
- `GET /api/v1/workflows` - List all workflows (with pagination)
- `GET /api/v1/workflows/{id}` - Get workflow status
- `POST /api/v1/workflows/{id}/cancel` - Cancel a workflow
- `GET /api/v1/workflows/{id}/tasks/{tid}/result` - Get task result

**Health Checks:**
- `GET /health` - Liveness probe
- `GET /ready` - Readiness probe
- `GET /status` - Detailed status information

**Documentation:**
- `GET /swagger/index.html` - Interactive API documentation

#### Quick API Example

```bash
# Submit a workflow
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "data-processing",
    "description": "Process customer data",
    "tasks": [
      {
        "id": "task-1",
        "name": "Fetch data",
        "type": "http"
      },
      {
        "id": "task-2",
        "name": "Process data",
        "type": "script",
        "depends_on": ["task-1"]
      }
    ]
  }'

# Get workflow status
curl http://localhost:8080/api/v1/workflows/{workflow-id}

# List all workflows
curl http://localhost:8080/api/v1/workflows?limit=10&offset=0
```

For more examples, see [docs/examples/curl-examples.md](docs/examples/curl-examples.md).

---

<a name="chinese"></a>
## 🌟 项目简介

**Goclaw** 是一个基于 Go 语言构建的生产级、高性能、分布式多 Agent 编排引擎。

它提供了一个健壮的框架，用于构建、部署和管理能够在分布式环境中无缝协作的智能代理。

### 核心特性

- 🚀 **高性能** - 基于 Go 的并发模型，实现最大吞吐量
- 🏗️ **分布式架构** - 原生支持集群部署和服务发现
- 🔄 **Agent 编排** - 高级工作流管理和任务调度
- 🌐 **RESTful API** - 完整的 HTTP API 和 Swagger 文档
- 📊 **可观测性** - 内置指标、日志和链路追踪支持
- 🔌 **可扩展** - 插件化架构，支持自定义 Agent 行为
- 🛡️ **生产就绪** - 完善的错误处理和容错机制

### 快速开始

```bash
# 克隆仓库
git clone https://github.com/goclaw/goclaw.git
cd goclaw

# 构建项目
make build

# 运行测试
make test

# 启动服务
make run
```

### 安装

```bash
go get github.com/goclaw/goclaw
```

### 使用示例

```go
package main

import (
    "context"
    "github.com/goclaw/goclaw/pkg/engine"
    "github.com/goclaw/goclaw/config"
    "github.com/goclaw/goclaw/pkg/logger"
)

func main() {
    // 加载配置
    cfg, err := config.Load("config.yaml", nil)
    if err != nil {
        panic(err)
    }

    // 初始化日志
    log := logger.New(&logger.Config{
        Level:  logger.InfoLevel,
        Format: "json",
        Output: "stdout",
    })

    // 创建编排引擎
    eng, err := engine.New(cfg, log)
    if err != nil {
        panic(err)
    }

    // 启动引擎
    ctx := context.Background()
    if err := eng.Start(ctx); err != nil {
        panic(err)
    }
    defer eng.Stop(ctx)

    // HTTP API 服务器现在运行在 8080 端口
    // 访问 Swagger UI: http://localhost:8080/swagger/index.html
}
```

### HTTP API

Goclaw 提供完整的 RESTful API 用于工作流管理：

#### API 端点

**工作流管理：**
- `POST /api/v1/workflows` - 提交新工作流
- `GET /api/v1/workflows` - 列出所有工作流（支持分页）
- `GET /api/v1/workflows/{id}` - 获取工作流状态
- `POST /api/v1/workflows/{id}/cancel` - 取消工作流
- `GET /api/v1/workflows/{id}/tasks/{tid}/result` - 获取任务结果

**健康检查：**
- `GET /health` - 存活探针
- `GET /ready` - 就绪探针
- `GET /status` - 详细状态信息

**文档：**
- `GET /swagger/index.html` - 交互式 API 文档

#### 快速 API 示例

```bash
# 提交工作流
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "数据处理",
    "description": "处理客户数据",
    "tasks": [
      {
        "id": "task-1",
        "name": "获取数据",
        "type": "http"
      },
      {
        "id": "task-2",
        "name": "处理数据",
        "type": "script",
        "depends_on": ["task-1"]
      }
    ]
  }'

# 获取工作流状态
curl http://localhost:8080/api/v1/workflows/{workflow-id}

# 列出所有工作流
curl http://localhost:8080/api/v1/workflows?limit=10&offset=0
```

更多示例请参见 [docs/examples/curl-examples.md](docs/examples/curl-examples.md)。

---

## 📚 Documentation

- [English Specification](docs/SPEC_en_v0.2.md)
- [中文规格说明](docs/SPEC_zh_v0.2.md)

## 🤝 Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details.

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.
