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
)

func main() {
    // Create a new orchestration engine
    eng := engine.New(engine.Config{
        Name: "my-agent-cluster",
    })

    // Start the engine
    ctx := context.Background()
    if err := eng.Start(ctx); err != nil {
        panic(err)
    }
    defer eng.Stop()

    // Your agent orchestration logic here
}
```

---

<a name="chinese"></a>
## 🌟 项目简介

**Goclaw** 是一个基于 Go 语言构建的生产级、高性能、分布式多 Agent 编排引擎。

它提供了一个健壮的框架，用于构建、部署和管理能够在分布式环境中无缝协作的智能代理。

### 核心特性

- 🚀 **高性能** - 基于 Go 的并发模型，实现最大吞吐量
- 🏗️ **分布式架构** - 原生支持集群部署和服务发现
- 🔄 **Agent 编排** - 高级工作流管理和任务调度
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
)

func main() {
    // 创建新的编排引擎
    eng := engine.New(engine.Config{
        Name: "my-agent-cluster",
    })

    // 启动引擎
    ctx := context.Background()
    if err := eng.Start(ctx); err != nil {
        panic(err)
    }
    defer eng.Stop()

    // 在此编写您的 Agent 编排逻辑
}
```

---

## 📚 Documentation

- [English Specification](docs/SPEC_en_v0.2.md)
- [中文规格说明](docs/SPEC_zh_v0.2.md)

## 🤝 Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details.

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.
