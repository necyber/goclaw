# DAG 编译器设计文档

## 架构概览

```
┌─────────────────────────────────────────────────────────────┐
│                      DAG Compiler                            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐             │
│  │   Task   │───▶│   DAG    │───▶│  Cycle   │             │
│  │  Builder │    │  Graph   │    │ Detector │             │
│  └──────────┘    └────┬─────┘    └────┬─────┘             │
│                       │               │                     │
│                       ▼               │                     │
│              ┌──────────────┐         │                     │
│              │ Toposort     │◀────────┘                     │
│              │ (Kahn's)     │                               │
│              └──────┬───────┘                               │
│                     │                                       │
│                     ▼                                       │
│         ┌─────────────────────┐                            │
│         │   Execution Plan    │                            │
│         │ ┌─────────────────┐ │                            │
│         │ │ Layers          │ │                            │
│         │ │ Parallel Groups │ │                            │
│         │ │ Critical Path   │ │                            │
│         │ └─────────────────┘ │                            │
│         └─────────────────────┘                            │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 范围边界（归档修正）

- Week 2 范围仅包含 DAG-core：Task/Graph/Cycle/Toposort/ExecutionPlan。
- Workflow 定义接口与调度器集成为 Deferred，放在后续变更中处理。

## 核心数据结构

### Task 定义

```go
// Task 表示一个可执行的任务单元
type Task struct {
    ID          string            // 唯一标识
    Name        string            // 人类可读名称
    Agent       string            // 执行 Agent 类型
    Lane        string            // 所属 Lane
    Deps        []string          // 依赖任务ID列表
    Timeout     time.Duration     // 执行超时
    Retries     int               // 重试次数
    Metadata    map[string]string // 扩展元数据
    Input       interface{}       // 输入数据模式
    Output      interface{}       // 输出数据模式
}

// TaskStatus 表示任务状态
type TaskStatus int

const (
    TaskPending TaskStatus = iota
    TaskScheduled
    TaskRunning
    TaskCompleted
    TaskFailed
    TaskCancelled
)
```

### DAG 图结构

```go
// Graph 表示有向无环图
type Graph struct {
    tasks     map[string]*Task    // 任务ID -> 任务
    edges     map[string][]string // 邻接表: 任务 -> 依赖它的任务
    inDegree  map[string]int      // 入度计数
    outDegree map[string]int      // 出度计数
}

// NewGraph 创建新的 DAG
func NewGraph() *Graph

// AddTask 添加任务节点
func (g *Graph) AddTask(task *Task) error

// AddEdge 添加依赖边 (from -> to 表示 to 依赖 from)
func (g *Graph) AddEdge(from, to string) error

// RemoveTask 移除任务及其边
func (g *Graph) RemoveTask(id string) error

// GetTask 获取任务
func (g *Graph) GetTask(id string) (*Task, bool)

// Dependencies 获取任务的依赖
func (g *Graph) Dependencies(id string) ([]*Task, error)

// Dependents 获取依赖此任务的任务
func (g *Graph) Dependents(id string) ([]*Task, error)
```

## 算法实现

### 1. 环检测（DFS）

```go
// CyclicDependencyError 表示检测到循环依赖
type CyclicDependencyError struct {
    Path []string // 循环路径
}

func (e *CyclicDependencyError) Error() string

// DetectCycle 使用 DFS 检测环
// 时间复杂度: O(V+E)
// 空间复杂度: O(V)
func (g *Graph) DetectCycle() (*CyclicDependencyError, bool)

// 实现细节:
// - 使用三色标记法 (white/gray/black)
// - gray 节点表示当前 DFS 路径上
// - 遇到 gray 节点说明有环
// - 记录路径以便报告具体循环
```

### 2. 拓扑排序（Kahn's Algorithm）

```go
// TopologicalSort 返回拓扑排序后的任务ID列表
// 时间复杂度: O(V+E)
// 空间复杂度: O(V)
func (g *Graph) TopologicalSort() ([]string, error)

// 实现步骤:
// 1. 计算所有节点入度
// 2. 将入度为0的节点加入队列
// 3. 依次取出节点，将其邻接节点入度减1
// 4. 若邻接节点入度为0，加入队列
// 5. 重复直到队列为空
// 6. 检查是否所有节点都被处理（否则有环）
```

### 3. 执行计划生成

```go
// ExecutionPlan 表示编译后的执行计划
type ExecutionPlan struct {
    Layers         [][]string    // 分层执行，每层内部可并行
    ParallelGroups []TaskGroup   // 最大并行组
    CriticalPath   []string      // 关键路径（决定总执行时间）
    TotalTasks     int           // 总任务数
    MaxParallel    int           // 最大并行度
}

// TaskGroup 表示一组可并行执行的任务
type TaskGroup struct {
    Tasks   []string
    Layer   int
}

// Compile 将 DAG 编译为执行计划
func (g *Graph) Compile() (*ExecutionPlan, error)

// 实现步骤:
// 1. 检测环
// 2. 拓扑排序
// 3. 分层：相同深度的任务在同一层
// 4. 识别并行组
// 5. 计算关键路径
```

## 关键路径分析

```go
// CriticalPath 计算关键路径（最长路径）
// 使用动态规划: dist[v] = max(dist[u] + weight(u,v))
func (g *Graph) CriticalPath() ([]string, int)

// 关键路径决定:
// - 最短可能执行时间
// - 哪些任务延迟会影响总体
```

## 错误处理

```go
// DAGError 是 DAG 相关错误的基接口
type DAGError interface {
    error
    TaskID() string
}

// TaskNotFoundError 任务不存在
type TaskNotFoundError struct {
    ID string
}

// DuplicateTaskError 重复任务ID
type DuplicateTaskError struct {
    ID string
}

// DependencyNotFoundError 依赖任务不存在
type DependencyNotFoundError struct {
    TaskID string
    DepID  string
}

// CyclicDependencyError 循环依赖
type CyclicDependencyError struct {
    Path []string
}
```

## 性能优化

1. **延迟加载** - 图结构修改时标记 dirty，查询时才计算
2. **缓存拓扑序** - 图不变时复用已计算的排序
3. **并行环检测** - 对不连通子图并行检测

## 测试策略

1. **单元测试**
   - 空图、单节点图
   - 线性依赖链
   - 星型依赖
   - 复杂 DAG
   - 含环图（应报错）

2. **性能测试**
   - 1000 节点图
   - 10000 节点图
   - 测量编译时间

3. **模糊测试**
   - 随机生成图结构
   - 验证环检测正确性

## 文件结构

```
pkg/dag/
├── dag.go              # Graph 结构和方法
├── dag_test.go         # 单元测试
├── task.go             # Task 定义
├── compiler.go         # 编译器逻辑
├── compiler_test.go    # 编译器测试
├── cycle.go            # 环检测
├── cycle_test.go       # 环检测测试
├── toposort.go         # 拓扑排序
├── toposort_test.go    # 拓扑排序测试
└── plan.go             # ExecutionPlan
```

## 使用示例

```go
// 构建 DAG
g := dag.NewGraph()

// 添加任务
g.AddTask(&dag.Task{ID: "fetch", Agent: "Fetcher"})
g.AddTask(&dag.Task{ID: "parse", Agent: "Parser", Deps: []string{"fetch"}})
g.AddTask(&dag.Task{ID: "analyze", Agent: "Analyzer", Deps: []string{"parse"}})
g.AddTask(&dag.Task{ID: "report", Agent: "Reporter", Deps: []string{"analyze"}})

// 编译
plan, err := g.Compile()
if err != nil {
    // 处理错误，如循环依赖
    if cycleErr, ok := err.(*dag.CyclicDependencyError); ok {
        fmt.Printf("检测到循环: %v\n", cycleErr.Path)
    }
}

// 使用执行计划
for layerIdx, layer := range plan.Layers {
    fmt.Printf("Layer %d: %v\n", layerIdx, layer)
    // 并行执行本层所有任务
}
```

## Errata 记录格式（仅用于非语义修复）

当仅修复编码、错别字、失效链接等非语义问题时，使用如下格式追加记录：

`[Errata YYYY-MM-DD] Type=<encoding|typo|link> Reason=<原因> Scope=<影响段落/文件>`
