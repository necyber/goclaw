# 配置系统设计文档

## 架构概览

```
┌─────────────────────────────────────────────────────────────┐
│                     Configuration System                     │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │  YAML    │  │  JSON    │  │   Env    │  │  Flags   │    │
│  │  Files   │  │  Files   │  │   Vars   │  │  (CLI)   │    │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘    │
│       └──────────────┴──────────────┴──────────────┘         │
│                          │                                   │
│                   ┌──────▼──────┐                           │
│                   │   Loader    │  ← Viper/Koanf            │
│                   └──────┬──────┘                           │
│                          │                                   │
│                   ┌──────▼──────┐                           │
│                   │  Validator  │  ← go-playground/validator│
│                   └──────┬──────┘                           │
│                          │                                   │
│                   ┌──────▼──────┐                           │
│                   │    Config   │  ← Strongly typed struct  │
│                   │    Struct   │                           │
│                   └──────┬──────┘                           │
│                          │                                   │
│       ┌──────────────────┼──────────────────┐               │
│       ▼                  ▼                  ▼               │
│  ┌─────────┐       ┌─────────┐       ┌─────────┐           │
│  │ Engine  │       │  Lane   │       │ Logger  │           │
│  │ Config  │       │ Config  │       │ Config  │           │
│  └─────────┘       └─────────┘       └─────────┘           │
└─────────────────────────────────────────────────────────────┘
```

## 配置结构

### 根配置

```go
// Config 是 Goclaw 的全局配置
type Config struct {
    // 应用基础配置
    App AppConfig `mapstructure:"app" validate:"required"`
    
    // 服务器配置
    Server ServerConfig `mapstructure:"server" validate:"required"`
    
    // 日志配置
    Log LogConfig `mapstructure:"log" validate:"required"`
    
    // 编排引擎配置
    Orchestration OrchestrationConfig `mapstructure:"orchestration"`
    
    // 集群配置（Phase 2）
    Cluster ClusterConfig `mapstructure:"cluster"`
    
    // 存储配置
    Storage StorageConfig `mapstructure:"storage"`
    
    // 指标监控配置
    Metrics MetricsConfig `mapstructure:"metrics"`
    
    // 链路追踪配置（Phase 3）
    Tracing TracingConfig `mapstructure:"tracing"`
}
```

### 各模块配置详情

```go
// AppConfig 应用基础配置
type AppConfig struct {
    Name        string `mapstructure:"name" validate:"required"`
    Version     string `mapstructure:"version"`
    Environment string `mapstructure:"environment" validate:"oneof=development staging production"`
    Debug       bool   `mapstructure:"debug"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
    Host string      `mapstructure:"host" default:"0.0.0.0"`
    Port int         `mapstructure:"port" validate:"required,min=1,max=65535"`
    GRPC GRPCConfig  `mapstructure:"grpc"`
    HTTP HTTPConfig  `mapstructure:"http"`
}

// LogConfig 日志配置
type LogConfig struct {
    Level   string `mapstructure:"level" validate:"oneof=debug info warn error" default:"info"`
    Format  string `mapstructure:"format" validate:"oneof=json text" default:"json"`
    Output  string `mapstructure:"output" default:"stdout"`
}

// OrchestrationConfig 编排配置
type OrchestrationConfig struct {
    MaxAgents    int           `mapstructure:"max_agents" validate:"min=1" default:"1000"`
    QueueSize    int           `mapstructure:"queue_size" validate:"min=100" default:"10000"`
    Scheduler    SchedulerType `mapstructure:"scheduler_type" default:"round_robin"`
}
```

## 关键技术决策

### 1. 配置库选择

**选型：Koanf**（替代 Viper）

| 特性 | Viper | Koanf |
|------|-------|-------|
| 依赖大小 | 大（1MB+） | 小（<100KB） |
| 性能 | 一般 | 更好 |
| 模块化 | 一体式 | 按需加载 |
| 验证集成 | 需额外代码 | 原生支持 |
| 热重载 | 支持 | 支持 |

**决策理由**：Koanf 更轻量、模块化，适合 Goclaw 对性能和依赖控制的要求。

### 2. 验证方案

使用 `go-playground/validator`：

```go
import "github.com/go-playground/validator/v10"

var validate = validator.New()

func (c *Config) Validate() error {
    return validate.Struct(c)
}
```

### 3. 环境变量映射

```
GOCLAW_APP_NAME=goclaw
GOCLAW_SERVER_PORT=8080
GOCLAW_LOG_LEVEL=debug
```

## 配置加载优先级（从高到低）

1. CLI 命令行参数 (`--config`, `--port` 等)
2. 环境变量 (`GOCLAW_*`)
3. 配置文件 (`config.yaml`, `config.json`)
4. 默认值

## 热重载机制

```go
// Watcher 接口
type Watcher interface {
    Watch(ctx context.Context, onChange func()) error
    Stop() error
}

// 支持热重载的配置项
- Log.Level   // 日志级别
- Log.Format  // 日志格式
- Metrics.Enabled // 指标开关

// 不支持热重载（需重启）
- Server.Port
- Storage.Type
```

## 文件结构

```
config/
├── config.go           # 配置结构体定义
├── config_test.go      # 单元测试
├── loader.go           # 配置加载器
├── loader_test.go
├── validator.go        # 配置验证
├── watcher.go          # 文件监控
├── defaults.go         # 默认值定义
└── example.yaml        # 配置示例

pkg/logger/
├── logger.go           # 日志接口
├── slog.go             # slog 实现
├── config.go           # 日志配置
└── logger_test.go
```

## 错误处理策略

```go
// ConfigError 配置错误类型
type ConfigError struct {
    Field   string
    Message string
    Value   interface{}
}

// 示例错误输出
// config validation failed:
//   - server.port: must be between 1 and 65535, got 99999
//   - log.level: must be one of [debug info warn error], got "trace"
```

## 测试策略

1. **单元测试**：各配置项的验证逻辑
2. **集成测试**：完整加载流程
3. **模糊测试**：随机配置输入验证
