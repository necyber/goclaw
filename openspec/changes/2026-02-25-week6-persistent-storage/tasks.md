# Phase 2: 持久化存储实现任务清单

## 1. 存储接口定义

- [x] 1.1 创建 `pkg/storage` 包目录结构
- [x] 1.2 定义 `Storage` 接口（workflow 和 task 操作）
- [x] 1.3 定义 `WorkflowState` 和 `TaskState` 数据结构
- [x] 1.4 定义 `WorkflowFilter` 查询过滤器
- [x] 1.5 定义类型化错误（NotFoundError, DuplicateKeyError, StorageUnavailableError）
- [x] 1.6 添加接口文档注释

## 2. 内存存储实现

- [x] 2.1 创建 `pkg/storage/memory` 包
- [x] 2.2 实现 `MemoryStorage` 结构（重构现有 WorkflowStore）
- [x] 2.3 实现 workflow CRUD 操作
- [x] 2.4 实现 task 持久化操作
- [x] 2.5 实现 ListWorkflows 带过滤和分页
- [x] 2.6 添加并发安全保护（sync.RWMutex）
- [x] 2.7 编写 MemoryStorage 单元测试

## 3. Badger 存储实现

- [x] 3.1 添加 Badger 依赖到 go.mod (`github.com/dgraph-io/badger/v4`)
- [x] 3.2 创建 `pkg/storage/badger` 包
- [x] 3.3 定义 `BadgerStorage` 结构和配置
- [x] 3.4 实现 Open/Close 生命周期方法
- [x] 3.5 实现 key 生成函数（workflow, task, index keys）
- [x] 3.6 实现 JSON 序列化/反序列化辅助函数
- [x] 3.7 实现 SaveWorkflow（带事务）
- [x] 3.8 实现 GetWorkflow
- [x] 3.9 实现 ListWorkflows（带前缀扫描和过滤）
- [x] 3.10 实现 DeleteWorkflow（级联删除 tasks）
- [x] 3.11 实现 SaveTask
- [x] 3.12 实现 GetTask
- [x] 3.13 实现 ListTasks
- [x] 3.14 实现索引维护（status, created time）
- [x] 3.15 添加错误处理和类型转换
- [x] 3.16 编写 BadgerStorage 单元测试

## 4. 存储配置

- [x] 4.1 在 `config/config.go` 添加 `StorageConfig` 结构
- [x] 4.2 添加 `BadgerConfig` 子结构（path, sync_writes, value_log_file_size）
- [x] 4.3 添加 storage type 字段（memory, badger）
- [x] 4.4 更新 `config.example.yaml` 添加 storage 配置示例
- [x] 4.5 添加配置验证规则
- [x] 4.6 添加默认值（type: memory）
- [x] 4.7 编写配置加载测试

## 5. 引擎集成

- [x] 5.1 在 `pkg/engine/engine.go` 添加 `storage Storage` 字段
- [x] 5.2 更新 `NewEngine()` 接受 storage 参数
- [x] 5.3 修改 `pkg/engine/workflow_manager.go` 使用 Storage 接口
- [x] 5.4 替换 WorkflowStore map 为 storage.SaveWorkflow 调用
- [x] 5.5 替换 Get 操作为 storage.GetWorkflow 调用
- [x] 5.6 替换 List 操作为 storage.ListWorkflows 调用
- [x] 5.7 更新 task 状态保存使用 storage.SaveTask
- [x] 5.8 更新 task 结果获取使用 storage.GetTask
- [x] 5.9 移除旧的 WorkflowStore 结构
- [x] 5.10 更新引擎测试使用 MemoryStorage

## 6. 恢复机制

- [x] 6.1 在 `pkg/engine/engine.go` 添加 `RecoverWorkflows()` 方法
- [x] 6.2 实现加载 pending/running 工作流逻辑
- [x] 6.3 实现 `resubmitWorkflow()` 辅助方法
- [x] 6.4 添加任务状态判断逻辑（completed 跳过，running 重置）
- [x] 6.5 添加恢复日志记录
- [x] 6.6 实现错误处理（单个失败不阻塞整体）
- [x] 6.7 在 `Start()` 方法中调用 RecoverWorkflows
- [x] 6.8 编写恢复机制单元测试
- [x] 6.9 编写恢复机制集成测试

## 7. 主程序集成

- [x] 7.1 修改 `cmd/goclaw/main.go` 初始化 storage
- [x] 7.2 根据配置创建 MemoryStorage 或 BadgerStorage
- [x] 7.3 传递 storage 到 engine.NewEngine()
- [x] 7.4 在 shutdown 时调用 storage.Close()
- [x] 7.5 添加 storage 初始化错误处理
- [x] 7.6 添加 storage 相关日志
- [x] 7.7 测试启动流程

## 8. 存储测试

- [x] 8.1 创建 `pkg/storage/testing.go` 接口测试套件
- [x] 8.2 测试 workflow CRUD 操作
- [x] 8.3 测试 task 持久化操作
- [x] 8.4 测试并发访问安全性
- [x] 8.5 测试事务原子性
- [x] 8.6 测试错误处理（not found, duplicate key）
- [x] 8.7 测试过滤和分页
- [x] 8.8 测试级联删除
- [x] 8.9 为 MemoryStorage 运行测试套件
- [x] 8.10 为 BadgerStorage 运行测试套件

## 9. 集成测试

- [x] 9.1 创建 `pkg/engine/persistence_test.go`
- [x] 9.2 测试工作流提交后持久化
- [x] 9.3 测试工作流状态更新持久化
- [x] 9.4 测试任务结果持久化
- [x] 9.5 测试服务重启后恢复（模拟重启）
- [x] 9.6 测试恢复后工作流继续执行
- [x] 9.7 测试并发工作流提交和查询
- [x] 9.8 测试存储失败场景
- [x] 9.9 测试数据一致性

## 10. 端到端测试

- [ ] 10.1 创建测试脚本 `test-persistence.sh`
- [ ] 10.2 测试：提交工作流 → 停止服务 → 启动服务 → 验证恢复
- [ ] 10.3 测试：运行中工作流 → 崩溃 → 恢复 → 继续执行
- [ ] 10.4 测试：Badger 数据目录权限
- [ ] 10.5 测试：配置切换（memory ↔ badger）
- [ ] 10.6 测试：大量工作流持久化性能
- [ ] 10.7 验证数据文件正确创建

## 11. 性能测试

- [ ] 11.1 创建 `pkg/storage/benchmark_test.go`
- [ ] 11.2 基准测试：SaveWorkflow 延迟
- [ ] 11.3 基准测试：GetWorkflow 延迟
- [ ] 11.4 基准测试：ListWorkflows 吞吐量
- [ ] 11.5 基准测试：并发写入性能
- [ ] 11.6 基准测试：并发读取性能
- [ ] 11.7 验证 P99 延迟 < 10ms (write), < 5ms (read)
- [ ] 11.8 对比 MemoryStorage 和 BadgerStorage 性能

## 12. 文档更新

- [x] 12.1 更新 `README.md` 添加存储配置说明
- [x] 12.2 添加 Badger 数据目录要求
- [x] 12.3 添加配置示例（memory vs badger）
- [x] 12.4 更新 `CLAUDE.md` 添加存储架构说明
- [x] 12.5 更新 Phase 2 完成状态
- [ ] 12.6 创建 `docs/storage-guide.md` 部署指南
- [ ] 12.7 添加备份和恢复说明
- [ ] 12.8 添加故障排查指南
- [ ] 12.9 更新 API 文档（无变化，但需确认）

## 13. Docker 和部署

- [x] 13.1 更新 `Dockerfile` 创建数据目录
- [x] 13.2 添加 VOLUME 声明用于持久化
- [x] 13.3 更新 `docker-compose.yml` 添加 volume 映射
- [x] 13.4 测试 Docker 容器持久化
- [x] 13.5 测试容器重启后数据恢复
- [x] 13.6 更新部署文档

## 14. 清理和优化

- [x] 14.1 移除未使用的代码（旧 WorkflowStore）
- [x] 14.2 优化导入语句
- [x] 14.3 运行 `go fmt` 格式化代码
- [x] 14.4 运行 `go vet` 检查问题
- [ ] 14.5 运行 `golangci-lint` 检查代码质量
- [ ] 14.6 修复所有 lint 警告
- [x] 14.7 更新 go.mod 和 go.sum
- [x] 14.8 运行完整测试套件确认无回归
- [ ] 5.9 移除旧的 WorkflowStore 结构
- [ ] 5.10 更新引擎测试使用 MemoryStorage

## 6. 恢复机制

- [ ] 6.1 在 `pkg/engine/engine.go` 添加 `RecoverWorkflows()` 方法
- [ ] 6.2 实现加载 pending/running 工作流逻辑
- [ ] 6.3 实现 `resubmitWorkflow()` 辅助方法
- [ ] 6.4 添加任务状态判断逻辑（completed 跳过，running 重置）
- [ ] 6.5 添加恢复日志记录
- [ ] 6.6 实现错误处理（单个失败不阻塞整体）
- [ ] 6.7 在 `Start()` 方法中调用 RecoverWorkflows
- [ ] 6.8 编写恢复机制单元测试
- [ ] 6.9 编写恢复机制集成测试

## 7. 主程序集成

- [ ] 7.1 修改 `cmd/goclaw/main.go` 初始化 storage
- [ ] 7.2 根据配置创建 MemoryStorage 或 BadgerStorage
- [ ] 7.3 传递 storage 到 engine.NewEngine()
- [ ] 7.4 在 shutdown 时调用 storage.Close()
- [ ] 7.5 添加 storage 初始化错误处理
- [ ] 7.6 添加 storage 相关日志
- [ ] 7.7 测试启动流程

## 8. 存储测试

- [ ] 8.1 创建 `pkg/storage/storage_test.go` 接口测试套件
- [ ] 8.2 测试 workflow CRUD 操作
- [ ] 8.3 测试 task 持久化操作
- [ ] 8.4 测试并发访问安全性
- [ ] 8.5 测试事务原子性
- [ ] 8.6 测试错误处理（not found, duplicate key）
- [ ] 8.7 测试过滤和分页
- [ ] 8.8 测试级联删除
- [ ] 8.9 为 MemoryStorage 运行测试套件
- [ ] 8.10 为 BadgerStorage 运行测试套件

## 9. 集成测试

- [ ] 9.1 创建 `pkg/engine/persistence_test.go`
- [ ] 9.2 测试工作流提交后持久化
- [ ] 9.3 测试工作流状态更新持久化
- [ ] 9.4 测试任务结果持久化
- [ ] 9.5 测试服务重启后恢复（模拟重启）
- [ ] 9.6 测试恢复后工作流继续执行
- [ ] 9.7 测试并发工作流提交和查询
- [ ] 9.8 测试存储失败场景
- [ ] 9.9 测试数据一致性

## 10. 端到端测试

- [ ] 10.1 创建测试脚本 `test-persistence.sh`
- [ ] 10.2 测试：提交工作流 → 停止服务 → 启动服务 → 验证恢复
- [ ] 10.3 测试：运行中工作流 → 崩溃 → 恢复 → 继续执行
- [ ] 10.4 测试：Badger 数据目录权限
- [ ] 10.5 测试：配置切换（memory ↔ badger）
- [ ] 10.6 测试：大量工作流持久化性能
- [ ] 10.7 验证数据文件正确创建

## 11. 性能测试

- [ ] 11.1 创建 `pkg/storage/benchmark_test.go`
- [ ] 11.2 基准测试：SaveWorkflow 延迟
- [ ] 11.3 基准测试：GetWorkflow 延迟
- [ ] 11.4 基准测试：ListWorkflows 吞吐量
- [ ] 11.5 基准测试：并发写入性能
- [ ] 11.6 基准测试：并发读取性能
- [ ] 11.7 验证 P99 延迟 < 10ms (write), < 5ms (read)
- [ ] 11.8 对比 MemoryStorage 和 BadgerStorage 性能

## 12. 文档更新

- [ ] 12.1 更新 `README.md` 添加存储配置说明
- [ ] 12.2 添加 Badger 数据目录要求
- [ ] 12.3 添加配置示例（memory vs badger）
- [ ] 12.4 更新 `CLAUDE.md` 添加存储架构说明
- [ ] 12.5 更新 Phase 2 完成状态
- [ ] 12.6 创建 `docs/storage-guide.md` 部署指南
- [ ] 12.7 添加备份和恢复说明
- [ ] 12.8 添加故障排查指南
- [ ] 12.9 更新 API 文档（无变化，但需确认）

## 13. Docker 和部署

- [ ] 13.1 更新 `Dockerfile` 创建数据目录
- [ ] 13.2 添加 VOLUME 声明用于持久化
- [ ] 13.3 更新 `docker-compose.yml` 添加 volume 映射
- [ ] 13.4 测试 Docker 容器持久化
- [ ] 13.5 测试容器重启后数据恢复
- [ ] 13.6 更新部署文档

## 14. 清理和优化

- [ ] 14.1 移除未使用的代码（旧 WorkflowStore）
- [ ] 14.2 优化导入语句
- [ ] 14.3 运行 `go fmt` 格式化代码
- [ ] 14.4 运行 `go vet` 检查问题
- [ ] 14.5 运行 `golangci-lint` 检查代码质量
- [ ] 14.6 修复所有 lint 警告
- [ ] 14.7 更新 go.mod 和 go.sum
- [ ] 14.8 运行完整测试套件确认无回归

## 验收标准

完成所有任务后，验证以下标准：

### 功能性
- [ ] 工作流状态在服务重启后保持
- [ ] 任务结果正确持久化和恢复
- [ ] 内存和 Badger 存储都正常工作
- [ ] 配置切换无问题
- [ ] 恢复机制正确处理各种状态

### 质量
- [ ] 单元测试覆盖率 > 80%
- [ ] 所有集成测试通过
- [ ] 端到端测试通过
- [ ] 无已知严重 bug
- [ ] 代码通过 lint 检查

### 性能
- [ ] 写入延迟 P99 < 10ms
- [ ] 读取延迟 P99 < 5ms
- [ ] 吞吐量 > 1000 ops/s
- [ ] 恢复时间 < 5s (100 workflows)

### 文档
- [ ] 配置文档完整
- [ ] 部署指南清晰
- [ ] API 文档更新
- [ ] 故障排查指南可用

### 部署
- [ ] Docker 镜像可构建
- [ ] 容器持久化正常
- [ ] 配置示例完整
- [ ] 数据目录权限正确

## 估算

- **阶段 1-2**: 存储接口和内存实现 - 1 天
- **阶段 3**: Badger 实现 - 2 天
- **阶段 4-5**: 配置和引擎集成 - 1 天
- **阶段 6-7**: 恢复机制和主程序 - 1 天
- **阶段 8-11**: 测试（单元、集成、端到端、性能）- 2 天
- **阶段 12-13**: 文档和部署 - 1 天
- **阶段 14**: 清理和优化 - 0.5 天

**总计**: 约 8.5 天

## 注意事项

1. **按顺序实现**: 每个阶段依赖前一阶段完成
2. **测试驱动**: 为每个组件编写测试后再实现下一个
3. **增量提交**: 完成每个阶段后提交代码
4. **性能监控**: 持续监控存储操作延迟
5. **错误处理**: 确保所有错误场景都有适当处理
6. **向后兼容**: 保持内存存储选项可用
7. **数据安全**: 使用事务保证数据一致性
8. **文档同步**: 代码和文档同步更新
