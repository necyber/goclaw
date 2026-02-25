# Phase 2: 持久化存储架构设计

## 概述

为 Goclaw 添加持久化存储支持，使工作流和任务状态能够在服务重启后恢复。

## 目标

1. **持久化工作流状态** - 工作流元数据、状态、任务列表
2. **持久化任务结果** - 任务执行结果和错误信息
3. **支持服务重启恢复** - 服务重启后自动恢复未完成的工作流
4. **高性能** - 存储操作不应成为性能瓶颈
5. **可扩展** - 支持多种存储后端（Badger, Redis）

## 架构设计

### 1. 存储层抽象

```
pkg/storage/
├── storage.go          # Storage 接口定义
├── badger/
│   ├── badger.go      # Badger 实现
│   └── badger_test.go
├── redis/
│   ├── redis.go       # Redis 实现（Phase 2.2）
│   └── redis_test.go
└── memory/
    ├── memory.go      # 内存实现（用于测试）
    └── memory_test.go
```

### 2. Storage 接口

```go
package storage

import (
    "context"
    "time"
)

// Storage 定义持久化存储接口
type Storage interface {
    // Workflow operations
    SaveWorkflow(ctx context.Context, wf *WorkflowState) error
    GetWorkflow(ctx context.Context, id string) (*WorkflowState, error)
    ListWorkflows(ctx context.Context, filter *WorkflowFilter) ([]*WorkflowState, int, error)
    DeleteWorkflow(ctx context.Context, id string) error

    // Task operations
    SaveTask(ctx context.Context, workflowID string, task *TaskState) error
    GetTask(ctx context.Context, workflowID, taskID string) (*TaskState, error)
    ListTasks(ctx context.Context, workflowID string) ([]*TaskState, error)

    // Lifecycle
    Close() error
}

// WorkflowState 工作流状态
type WorkflowState struct {
    ID          string
    Name        string
    Description string
    Status      string
    Tasks       []TaskDefinition
    Metadata    map[string]string
    CreatedAt   time.Time
    StartedAt   *time.Time
    CompletedAt *time.Time
    Error       string
}

// TaskState 任务状态
type TaskState struct {
    ID          string
    Name        string
    Status      string
    StartedAt   *time.Time
    CompletedAt *time.Time
    Error       string
    Result      interface{}
}

// WorkflowFilter 工作流过滤器
type WorkflowFilter struct {
    Status string
    Limit  int
    Offset int
}
```

### 3. Badger 存储实现

**选择 Badger 的原因**:
- 嵌入式 KV 存储，无需外部依赖
- 高性能（LSM-tree 架构）
- 支持事务
- Go 原生实现
- 适合单机部署

**数据模型**:
```
Key 格式:
- workflow:{id}           -> WorkflowState (JSON)
- workflow:{id}:task:{tid} -> TaskState (JSON)
- workflow:index:status:{status}:{id} -> "" (用于按状态查询)
- workflow:index:created:{timestamp}:{id} -> "" (用于按时间排序)
```

**实现要点**:
- 使用 JSON 序列化存储数据
- 使用前缀扫描实现列表查询
- 使用事务保证原子性
- 定期 GC 清理旧数据

### 4. 配置扩展

```yaml
storage:
  # 存储类型: memory, badger, redis
  type: badger

  # Badger 配置
  badger:
    path: ./data/badger
    sync_writes: true
    value_log_file_size: 1073741824  # 1GB

  # Redis 配置（Phase 2.2）
  redis:
    addr: localhost:6379
    password: ""
    db: 0
    pool_size: 10
```

### 5. 引擎集成

**修改点**:
1. `pkg/engine/engine.go` - 添加 Storage 字段
2. `pkg/engine/workflow_manager.go` - 使用 Storage 替代内存 map
3. `cmd/goclaw/main.go` - 初始化 Storage

**恢复机制**:
```go
// 启动时恢复未完成的工作流
func (e *Engine) RecoverWorkflows(ctx context.Context) error {
    // 1. 从存储加载所有 pending/running 工作流
    workflows, err := e.storage.ListWorkflows(ctx, &storage.WorkflowFilter{
        Status: "pending,running",
    })

    // 2. 重新提交到执行队列
    for _, wf := range workflows {
        if err := e.resubmitWorkflow(ctx, wf); err != nil {
            e.logger.Error("failed to recover workflow", "id", wf.ID, "error", err)
        }
    }

    return nil
}
```

## 实现计划

### Phase 2.1: Badger 存储（本阶段）

1. ✅ **设计存储架构** - 定义接口和数据模型
2. ⏭️ **实现 Storage 接口** - 创建 pkg/storage 包
3. ⏭️ **添加存储配置** - 扩展 config 包
4. ⏭️ **实现 Badger 存储** - BadgerStorage 实现
5. ⏭️ **编写存储测试** - 单元测试和集成测试
6. ⏭️ **集成到引擎** - 替换内存存储
7. ⏭️ **端到端测试** - 测试重启恢复

### Phase 2.2: Redis 存储（后续）

1. 实现 RedisStorage
2. 支持分布式部署
3. 添加 Redis 集群支持

### Phase 2.3: 高级特性（后续）

1. 数据压缩
2. 自动清理过期数据
3. 存储迁移工具
4. 备份和恢复

## 性能目标

- **写入延迟**: < 10ms (P99)
- **读取延迟**: < 5ms (P99)
- **吞吐量**: > 1000 ops/s
- **存储开销**: < 1KB per workflow

## 测试策略

### 单元测试
- Storage 接口的每个方法
- 边界条件和错误处理
- 并发安全性

### 集成测试
- 完整的工作流生命周期
- 服务重启恢复
- 数据一致性

### 性能测试
- 基准测试（go test -bench）
- 负载测试（大量工作流）
- 并发测试（多客户端）

## 风险和缓解

### 风险 1: 数据损坏
**缓解**:
- 使用 Badger 的事务保证原子性
- 定期备份
- 添加数据校验

### 风险 2: 性能下降
**缓解**:
- 使用索引优化查询
- 批量操作
- 异步写入（可选）

### 风险 3: 存储空间增长
**缓解**:
- 自动清理完成的工作流（可配置保留期）
- 数据压缩
- 监控存储使用

## 兼容性

- **向后兼容**: 保留内存存储选项（type: memory）
- **数据迁移**: 提供工具从内存迁移到 Badger
- **API 不变**: Storage 层对上层透明

## 文档

- 更新 README.md 添加存储配置说明
- 更新 CLAUDE.md 添加存储架构说明
- 创建存储配置示例
- 编写故障排查指南

## 验收标准

- [ ] Storage 接口定义完整
- [ ] BadgerStorage 实现所有方法
- [ ] 单元测试覆盖率 > 80%
- [ ] 集成测试通过
- [ ] 性能测试达标
- [ ] 服务重启恢复正常
- [ ] 文档完整

## 时间估算

- 设计架构: 0.5 天 ✅
- 实现接口: 0.5 天
- Badger 实现: 1 天
- 配置扩展: 0.5 天
- 引擎集成: 1 天
- 测试: 1 天
- 文档: 0.5 天

**总计**: 约 5 天
