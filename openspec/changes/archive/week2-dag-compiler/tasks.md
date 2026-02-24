# Week 2: DAG 编译器 - 任务清单

## 1. 基础数据结构

### 1.1 Task 定义
- [ ] 创建 `pkg/dag/task.go`，定义 Task 结构体
- [ ] 定义 TaskStatus 枚举
- [ ] 实现 Task 验证方法
- [ ] 添加 Task String() 方法

### 1.2 DAG 图结构
- [ ] 创建 `pkg/dag/dag.go`，定义 Graph 结构体
- [ ] 实现 NewGraph() 构造函数
- [ ] 实现 AddTask() 方法
- [ ] 实现 AddEdge() 方法
- [ ] 实现 GetTask() 方法
- [ ] 实现 Dependencies() 和 Dependents() 方法

## 2. 环检测

### 2.1 DFS 实现
- [ ] 创建 `pkg/dag/cycle.go`
- [ ] 实现三色标记 DFS
- [ ] 实现 DetectCycle() 方法
- [ ] 实现 CycleError 类型及详细错误信息

### 2.2 错误类型
- [ ] 创建 `pkg/dag/errors.go`
- [ ] 定义 DAGError 接口
- [ ] 实现 TaskNotFoundError
- [ ] 实现 DuplicateTaskError
- [ ] 实现 DependencyNotFoundError
- [ ] 实现 CyclicDependencyError

## 3. 拓扑排序

### 3.1 Kahn's Algorithm
- [ ] 创建 `pkg/dag/toposort.go`
- [ ] 实现 KahnTopologicalSort()
- [ ] 处理入度计算
- [ ] 实现队列处理逻辑

### 3.2 备选方案
- [ ] 实现 DFSTopologicalSort()（作为对比）
- [ ] 性能对比测试

## 4. 执行计划生成

### 4.1 执行计划结构
- [ ] 创建 `pkg/dag/plan.go`
- [ ] 定义 ExecutionPlan 结构体
- [ ] 定义 TaskGroup 结构体

### 4.2 编译器实现
- [ ] 创建 `pkg/dag/compiler.go`
- [ ] 实现 Compile() 方法
- [ ] 实现分层算法
- [ ] 实现并行组识别
- [ ] 实现关键路径分析

## 5. 单元测试

### 5.1 基础测试
- [ ] 测试空图
- [ ] 测试单节点
- [ ] 测试线性链（A→B→C）
- [ ] 测试星型结构
- [ ] 测试菱形结构（A→B,C→D）

### 5.2 环检测测试
- [ ] 测试自环（A→A）
- [ ] 测试两节点环（A↔B）
- [ ] 测试三节点环（A→B→C→A）
- [ ] 测试复杂环
- [ ] 验证错误路径准确性

### 5.3 性能测试
- [ ] 测试 100 节点图
- [ ] 测试 1000 节点图
- [ ] 测试 10000 节点图
- [ ] 记录编译时间

## 6. 集成与示例

### 6.1 Workflow 集成
- [ ] 创建 Workflow 构建器
- [ ] 实现从 Workflow 创建 DAG
- [ ] 添加便捷方法

### 6.2 示例代码
- [ ] 创建简单 DAG 示例
- [ ] 创建复杂工作流示例
- [ ] 添加性能测试示例

---

## 任务优先级

| 任务 | 优先级 | 依赖 | 预估时间 |
|------|--------|------|----------|
| 1.1 Task 定义 | P0 | 无 | 1h |
| 1.2 Graph 结构 | P0 | 1.1 | 2h |
| 2.1 环检测 | P0 | 1.2 | 3h |
| 2.2 错误类型 | P0 | 2.1 | 1h |
| 3.1 拓扑排序 | P0 | 2.1 | 2h |
| 4.1 执行计划结构 | P0 | 3.1 | 1h |
| 4.2 编译器 | P0 | 4.1 | 3h |
| 5.1-5.3 测试 | P1 | 全部 | 4h |
| 6.1-6.2 集成示例 | P2 | 全部 | 3h |

**总计：约 20 小时**

---

## 验收标准

- [ ] 所有 P0 任务完成
- [ ] 单元测试覆盖率 > 85%
- [ ] 1000 节点图编译时间 < 10ms
- [ ] 循环依赖错误信息准确指出路径
- [ ] 代码通过 `make check`
