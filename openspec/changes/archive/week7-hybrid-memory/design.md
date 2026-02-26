## Context

Goclaw 当前已实现 DAG 编译、Lane 队列、持久化存储（Badger）和 HTTP/gRPC API，但缺少智能记忆系统。Agent 无法利用历史执行结果、无法基于语义检索相关上下文，限制了多轮对话和长期任务能力。

根据 SPEC v0.2，Memory Hub 是核心组件之一，负责"混合检索、记忆衰减、上下文组装"。本设计实现 Phase 2 的混合记忆系统（向量检索 + BM25 + FSRS-6）。

**当前状态**:
- 已有 Badger 持久化存储（可作为 L2 层）
- 已有配置系统（可扩展 MemoryConfig）
- 已有 Engine 和 API 框架（可集成 Memory Hub）

**约束**:
- 向量检索需要外部依赖（hnswgo 或向量数据库客户端）
- 内存占用需控制（L1 缓存大小、向量维度）
- 需保持向后兼容（记忆系统可选，默认禁用）

## Goals / Non-Goals

**Goals:**
- 实现分层存储架构（L1 内存 LRU + L2 Badger + L3 向量数据库）
- 实现向量检索（HNSW 算法，余弦相似度）
- 实现 BM25 全文检索
- 实现混合检索策略（RRF 融合排序）
- 实现 FSRS-6 记忆衰减算法
- 提供 Memory Hub API（Memorize/Retrieve/Forget）
- 支持 Session 级别隔离
- 配置化启用/禁用，参数可调优

**Non-Goals:**
- 不实现分布式记忆同步（Phase 3）
- 不实现实时向量化（初期使用预计算向量）
- 不实现多模态记忆（仅文本）
- 不实现记忆压缩/归档（Phase 3）

## Decisions

### 1. 分层存储架构

**决策**: 采用三层存储架构（L1/L2/L3）

**理由**:
- L1 (内存 LRU): 热数据快速访问，减少 I/O
- L2 (Badger): 本地持久化，支持单机部署
- L3 (向量数据库): 可选，支持大规模向量检索

**替代方案**:
- 仅使用向量数据库：成本高，依赖外部服务
- 仅使用 Badger：无法高效向量检索

**实现**:
```go
type MemoryStorage interface {
    Store(ctx context.Context, entry *MemoryEntry) error
    Retrieve(ctx context.Context, query Query) ([]*MemoryEntry, error)
    Delete(ctx context.Context, id string) error
}

// L1: LRU Cache
type L1Cache struct {
    cache *lru.Cache
    maxSize int
}

// L2: Badger
type L2Badger struct {
    db *badger.DB
    prefix string
}

// L3: Vector DB (可选)
type L3VectorDB struct {
    client VectorDBClient // weaviate/qdrant
}
```

### 2. 向量检索实现

**决策**: 使用 hnswgo 本地实现 HNSW 算法

**理由**:
- 纯 Go 实现，无 CGO 依赖
- 性能优秀（近似最近邻搜索）
- 支持余弦相似度
- 可选升级到外部向量数据库

**替代方案**:
- faiss-go: 需要 CGO，编译复杂
- 直接使用 Weaviate/Qdrant: 增加部署复杂度

**实现**:
```go
type VectorRetriever struct {
    index *hnsw.Index
    dimension int
    metric string // "cosine"
}

func (v *VectorRetriever) Search(ctx context.Context, vector []float32, topK int) ([]string, []float32, error)
```

### 3. BM25 全文检索

**决策**: 自实现 BM25 算法，使用 Go 标准库 tokenizer

**理由**:
- BM25 算法简单，易于实现
- 避免引入重量级全文检索引擎（Elasticsearch）
- 支持中英文分词

**替代方案**:
- 使用 Bleve: 功能强大但依赖较重
- 使用 Elasticsearch: 部署复杂

**实现**:
```go
type BM25Searcher struct {
    documents map[string]*Document
    idf map[string]float64
    avgDocLen float64
    k1, b float64 // BM25 参数
}

func (b *BM25Searcher) Search(ctx context.Context, query string, topK int) ([]string, []float64, error)
```

### 4. 混合检索策略

**决策**: 使用 RRF (Reciprocal Rank Fusion) 融合排序

**理由**:
- RRF 简单有效，无需调参
- 融合向量检索和 BM25 结果
- 权重可配置（默认向量 0.7，BM25 0.3）

**替代方案**:
- 加权平均: 需要归一化分数
- 学习排序: 复杂度高

**实现**:
```go
type HybridRetriever struct {
    vectorRetriever *VectorRetriever
    bm25Searcher *BM25Searcher
    vectorWeight float64
    bm25Weight float64
}

func (h *HybridRetriever) Retrieve(ctx context.Context, query Query) ([]*MemoryEntry, error) {
    // 1. 向量检索
    vectorResults := h.vectorRetriever.Search(...)
    // 2. BM25 检索
    bm25Results := h.bm25Searcher.Search(...)
    // 3. RRF 融合
    return h.fuseResults(vectorResults, bm25Results)
}
```

### 5. FSRS-6 记忆衰减

**决策**: 实现 FSRS-6 算法核心公式，后台 Goroutine 定期更新

**理由**:
- FSRS-6 是科学的记忆模型
- 自动管理记忆强度，支持遗忘
- 后台异步更新，不阻塞主流程

**实现**:
```go
type MemoryDecay struct {
    entries map[string]*MemoryEntry
    threshold float64 // 遗忘阈值
}

func (m *MemoryDecay) UpdateStrength(entry *MemoryEntry) {
    // FSRS-6 公式: S' = S * e^(-t/τ)
    elapsed := time.Since(entry.LastReview)
    entry.Strength *= math.Exp(-elapsed.Hours() / entry.Stability)
}

func (m *MemoryDecay) StartDecayLoop(ctx context.Context, interval time.Duration) {
    ticker := time.NewTicker(interval)
    go func() {
        for {
            select {
            case <-ticker.C:
                m.decayAll()
            case <-ctx.Done():
                return
            }
        }
    }()
}
```

### 6. Memory Hub API 设计

**决策**: 提供简洁的 Memorize/Retrieve/Forget 接口

**理由**:
- 符合直觉的 API 命名
- 支持 Session 隔离
- 支持批量操作

**实现**:
```go
type MemoryHub interface {
    Memorize(ctx context.Context, sessionID string, content []byte, metadata map[string]string) (string, error)
    Retrieve(ctx context.Context, sessionID string, query Query, topK int) ([]*MemoryEntry, error)
    Forget(ctx context.Context, sessionID string, ids []string) error
    ForgetByThreshold(ctx context.Context, sessionID string, threshold float64) (int, error)
}

type Query struct {
    Text string
    Vector []float32
    Filters map[string]string
}
```

## Risks / Trade-offs

### 1. 向量维度与内存占用

**风险**: 高维向量（如 1536 维）占用大量内存

**缓解**:
- L1 缓存限制大小（默认 1000 条）
- 支持配置向量维度（默认 768 维）
- 提供内存监控指标

### 2. 向量化性能

**风险**: 实时向量化可能成为瓶颈

**缓解**:
- 初期使用预计算向量（由调用方提供）
- 后台异步向量化（Phase 3）
- 支持批量向量化

### 3. BM25 分词准确性

**风险**: 简单分词可能影响检索质量

**缓解**:
- 支持自定义 tokenizer
- 提供中英文分词器
- 后续可集成专业分词库

### 4. FSRS-6 参数调优

**风险**: 默认参数可能不适合所有场景

**缓解**:
- 提供配置化参数（stability, threshold）
- 支持 Session 级别参数覆盖
- 提供调优指南文档

### 5. 向后兼容性

**风险**: 新增记忆系统可能影响现有功能

**缓解**:
- 记忆系统默认禁用
- 配置项 `memory.enabled: false`
- 不修改现有 API 行为

## Migration Plan

### 部署步骤

1. **配置更新**:
   - 更新 `config.example.yaml` 添加 `memory` 配置段
   - 默认 `enabled: false`

2. **依赖安装**:
   - `go get github.com/Bithack/go-hnsw`
   - 可选: `go get github.com/weaviate/weaviate-go-client`

3. **初始化**:
   - 在 `cmd/goclaw/main.go` 初始化 Memory Hub
   - 传递给 Engine

4. **API 集成**:
   - 新增 `/api/v1/memory/*` 端点
   - 不影响现有端点

### 回滚策略

- 设置 `memory.enabled: false` 禁用记忆系统
- 删除 `pkg/memory/` 包不影响其他功能
- Badger 数据独立存储，可单独清理

## Open Questions

1. **向量化方案**: 是否需要内置向量化能力（如集成 sentence-transformers）？
   - 建议: Phase 3 考虑，初期由调用方提供向量

2. **分布式记忆同步**: 多节点如何同步记忆？
   - 建议: Phase 3 通过 NATS 或 Redis Pub/Sub 实现

3. **记忆压缩**: 长期记忆如何压缩归档？
   - 建议: Phase 3 实现，支持导出到对象存储

4. **多模态支持**: 是否支持图片、音频记忆？
   - 建议: Phase 3 考虑，需要多模态向量模型
