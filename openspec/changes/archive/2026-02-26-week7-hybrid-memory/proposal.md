## Why

AI Agent 协作的核心挑战之一是上下文共享和历史记忆。当前 Goclaw 引擎缺乏智能记忆系统，导致 Agent 无法有效利用历史执行结果、无法基于语义相似度检索相关上下文，限制了复杂多轮对话和长期任务的能力。混合记忆系统通过向量检索、全文检索和记忆衰减机制，为 Agent 提供智能的上下文管理能力。

## What Changes

- 新增 `pkg/memory/` 包，实现混合记忆系统核心功能
- 实现向量检索能力（HNSW 算法，支持余弦相似度）
- 实现 BM25 全文检索能力
- 实现 FSRS-6 记忆衰减算法，自动管理记忆强度
- 实现混合检索策略（RRF - Reciprocal Rank Fusion）
- 实现分层存储架构（L1 内存缓存 + L2 Badger + L3 向量数据库）
- 提供 Memory Hub API 供 Agent Runtime 调用
- 支持 Session 级别的记忆隔离
- 新增配置项支持记忆系统的启用/禁用和参数调优

## Capabilities

### New Capabilities

- `memory-storage`: 记忆条目的存储和持久化，支持分层存储架构（L1/L2/L3）
- `vector-retrieval`: 基于向量嵌入的语义检索，使用 HNSW 算法实现高性能近似最近邻搜索
- `bm25-search`: 基于 BM25 算法的全文检索，支持关键词匹配和相关性排序
- `hybrid-retrieval`: 混合检索策略，融合向量检索和 BM25 结果，使用 RRF 算法排序
- `memory-decay`: FSRS-6 记忆衰减算法，自动计算和更新记忆强度，支持遗忘机制
- `memory-hub-api`: Memory Hub 对外接口，提供 Memorize/Retrieve/Forget 等操作

### Modified Capabilities

- `workflow-api-endpoints`: 新增记忆相关的 API 端点（查询记忆、清理记忆等）

## Impact

**新增代码**:
- `pkg/memory/` - 新包，包含所有记忆系统实现
  - `memory.go` - Memory Hub 核心接口和实现
  - `storage.go` - 分层存储管理
  - `vector.go` - 向量检索实现
  - `bm25.go` - BM25 全文检索实现
  - `hybrid.go` - 混合检索策略
  - `fsrs.go` - FSRS-6 记忆衰减算法
  - `entry.go` - MemoryEntry 数据结构定义

**修改代码**:
- `config/config.go` - 新增 MemoryConfig 配置结构
- `pkg/engine/engine.go` - 集成 Memory Hub
- `pkg/api/handlers/` - 新增记忆管理 API 端点
- `cmd/goclaw/main.go` - 初始化 Memory Hub

**新增依赖**:
- 向量数据库客户端（可选）：`weaviate-go-client` 或 `qdrant-go`
- 向量索引库：`hnswgo` 或 `faiss-go` (本地实现)
- 文本处理：`go-tokenizer` (用于 BM25)

**配置影响**:
- `config.example.yaml` 新增 `memory` 配置段
- 支持启用/禁用记忆系统
- 支持配置向量维度、检索权重、遗忘阈值等参数

**性能影响**:
- 向量检索增加内存占用（取决于向量维度和条目数量）
- 后台 Goroutine 定期执行记忆衰减计算
- L1 缓存提升热数据访问性能
