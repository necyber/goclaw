## 1. 项目结构和构建基础

- [x] 1.1 创建 `web/` 前端项目目录，初始化 Vite + React 18 + TypeScript 项目
- [x] 1.2 安装核心依赖：react-router-dom、zustand、tailwindcss、@headlessui/react
- [x] 1.3 安装可视化依赖：@xyflow/react（React Flow）、dagre、recharts
- [x] 1.4 配置 Tailwind CSS 4 和亮/暗主题 CSS 变量
- [x] 1.5 配置 Vite 构建输出到 `web/dist/`，配置 API 代理到 localhost:8080
- [x] 1.6 在 Makefile 添加 `build-ui` 目标（npm install + npm run build）
- [x] 1.7 在 Makefile 修改 `build` 目标，条件触发 `build-ui`
- [x] 1.8 在 Makefile 添加 `clean` 目标清理 web/dist/ 和 web/node_modules/
- [x] 1.9 添加 `.gitignore` 规则忽略 web/node_modules/ 和 web/dist/

## 2. Go embed 和静态资源服务

- [x] 2.1 创建 `pkg/api/ui_embed.go`（build tag: embed_ui），使用 go:embed 嵌入 web/dist/
- [x] 2.2 创建 `pkg/api/ui_noembed.go`（build tag: !embed_ui），返回 "UI not included" 提示
- [x] 2.3 实现 SPA 回退路由（非静态文件路径返回 index.html）
- [x] 2.4 实现 Gzip 压缩中间件（对 HTML/CSS/JS/JSON/SVG 压缩，< 1KB 跳过）
- [x] 2.5 实现缓存控制（带 hash 文件名: max-age=31536000; index.html: no-cache）
- [x] 2.6 编写静态资源服务单元测试
- [x] 2.7 编写 SPA 回退路由测试

## 3. UI 配置

- [x] 3.1 在 `config/config.go` 添加 `UIConfig` 结构（Enabled, BasePath, DevProxy）
- [x] 3.2 更新 `config.example.yaml` 添加 `ui` 配置段
- [x] 3.3 添加配置验证规则（BasePath 必须以 / 开头）
- [x] 3.4 编写配置加载测试

## 4. 路由注册

- [x] 4.1 在 `pkg/api/router.go` 注册 /ui/* 静态资源路由（根据 UIConfig.Enabled 条件注册）
- [x] 4.2 实现开发模式代理（UIConfig.DevProxy 非空时代理到 Vite dev server）
- [x] 4.3 编写路由注册测试（启用/禁用 UI）

## 5. WebSocket 服务端

- [x] 5.1 添加 gorilla/websocket 依赖
- [x] 5.2 创建 `pkg/api/handlers/websocket.go` 定义 WebSocket handler
- [x] 5.3 实现 WebSocket 升级（/ws/events 端点）
- [x] 5.4 实现连接管理器（注册、注销、广播）
- [x] 5.5 实现连接数限制（默认 100，可配置）
- [x] 5.6 实现心跳检测（ping/pong，30s 间隔，10s 超时）
- [x] 5.7 实现工作流订阅（客户端发送 subscribe 消息指定 workflow ID）
- [x] 5.8 定义事件消息格式（type, timestamp, payload）
- [x] 5.9 实现 CORS 检查（WebSocket origin 验证）
- [x] 5.10 在 `pkg/api/router.go` 注册 /ws/events 路由
- [x] 5.11 编写 WebSocket handler 单元测试
- [x] 5.12 编写连接管理器测试（注册、注销、广播、连接限制）

## 6. 事件广播集成

- [x] 6.1 创建 `pkg/api/events/broadcaster.go` 定义事件广播器
- [x] 6.2 实现工作流状态变更事件（workflow.state_changed）
- [x] 6.3 实现任务状态变更事件（task.state_changed）
- [x] 6.4 在 Engine 中注入事件广播器，状态变更时触发事件
- [x] 6.5 编写事件广播器单元测试

## 7. 前端应用框架（Dashboard Layout）

- [x] 7.1 创建 `web/src/App.tsx` 应用入口，配置 React Router
- [x] 7.2 创建 `web/src/layouts/AppShell.tsx` 主布局（顶部导航 + 侧边栏 + 内容区）
- [x] 7.3 实现侧边栏导航（Dashboard、Workflows、Metrics、Admin）
- [x] 7.4 实现侧边栏折叠/展开切换
- [x] 7.5 实现当前页面高亮
- [x] 7.6 实现 404 页面
- [x] 7.7 创建 `web/src/stores/theme.ts` Zustand 主题 store
- [x] 7.8 实现亮/暗主题切换按钮，持久化到 localStorage
- [x] 7.9 实现系统主题偏好检测（prefers-color-scheme）
- [x] 7.10 创建通用 Loading 组件（spinner + skeleton）
- [x] 7.11 创建通用 Error 组件（错误消息 + 重试按钮）
- [x] 7.12 创建通用 Empty 组件（空状态插图 + 提示文字）
- [x] 7.13 实现响应式布局（>= 1280px 展开侧边栏，1024-1279px 自动折叠）

## 8. API 客户端层

- [x] 8.1 创建 `web/src/api/client.ts` HTTP 客户端（基于 fetch，统一错误处理）
- [x] 8.2 创建 `web/src/api/workflows.ts` 工作流 API（list, get, submit, cancel）
- [x] 8.3 创建 `web/src/api/admin.ts` 管理 API（engine status, lane stats, pause, resume, purge）
- [x] 8.4 创建 `web/src/api/metrics.ts` 指标 API（fetch /metrics 端点并解析）
- [x] 8.5 定义 TypeScript 类型（Workflow, Task, EngineStatus, LaneStats 等）

## 9. WebSocket 客户端

- [x] 9.1 创建 `web/src/lib/websocket.ts` WebSocket 客户端类
- [x] 9.2 实现自动重连（指数退避：1s → 2s → 4s → ... → 30s，最多 10 次）
- [x] 9.3 实现心跳保活（30s ping）
- [x] 9.4 实现连接状态管理（connected/disconnected/reconnecting）
- [x] 9.5 创建 `web/src/stores/websocket.ts` Zustand WebSocket store
- [x] 9.6 实现连接状态指示器组件（绿/红/黄点）
- [x] 9.7 实现工作流订阅/取消订阅
- [x] 9.8 实现事件分发到对应 store

## 10. 工作流管理页面

- [x] 10.1 创建 `web/src/pages/Workflows.tsx` 工作流列表页
- [x] 10.2 实现工作流表格（ID、名称、状态、创建时间、任务数）
- [x] 10.3 实现状态徽章组件（pending=灰、running=蓝、completed=绿、failed=红、cancelled=黄）
- [x] 10.4 实现分页控件（上一页/下一页，每页 20 条）
- [x] 10.5 实现状态过滤下拉框
- [x] 10.6 实现名称搜索输入框
- [x] 10.7 创建 `web/src/pages/WorkflowDetail.tsx` 工作流详情页
- [x] 10.8 实现详情头部（ID、名称、状态、时间戳、元数据）
- [x] 10.9 实现任务列表表格（ID、名称、状态、耗时、错误信息）
- [x] 10.10 实现任务结果 JSON 查看器（点击任务展开结果）
- [x] 10.11 实现取消工作流按钮（带确认对话框，仅 running 状态显示）
- [x] 10.12 创建 `web/src/components/SubmitWorkflowDialog.tsx` 提交工作流对话框
- [x] 10.13 实现 JSON 编辑器输入（带语法校验）
- [x] 10.14 实现提交和错误处理
- [x] 10.15 创建 `web/src/stores/workflows.ts` Zustand 工作流 store
- [x] 10.16 实现自动刷新（非终态工作流每 2s 刷新，或通过 WebSocket 更新）

## 11. DAG 可视化

- [x] 11.1 创建 `web/src/components/DagView.tsx` DAG 可视化组件
- [x] 11.2 实现 dagre 布局算法集成（自上而下层次布局）
- [x] 11.3 实现自定义任务节点组件（显示任务名称和状态图标）
- [x] 11.4 实现节点状态着色（pending=灰虚线、running=蓝脉冲、completed=绿✓、failed=红✗、cancelled=黄⊘）
- [x] 11.5 实现依赖边渲染（已完成=绿实线、待执行=灰虚线）
- [x] 11.6 实现缩放/平移交互
- [x] 11.7 实现 Fit to View 按钮
- [x] 11.8 实现节点点击选中，侧面板显示任务详情
- [x] 11.9 实现 minimap（> 10 节点时显示）
- [x] 11.10 实现实时状态更新（WebSocket 事件触发节点状态变更，无需重渲染整图）
- [x] 11.11 在工作流详情页添加 DAG 标签页
- [x] 11.12 处理无依赖场景（所有任务单行排列）

## 12. 指标仪表盘

- [x] 12.1 创建 `web/src/pages/Metrics.tsx` 指标页面
- [x] 12.2 实现概览卡片（活跃工作流数、24h 完成数、24h 失败数、平均执行时间）
- [x] 12.3 创建 `web/src/components/ThroughputChart.tsx` 吞吐量折线图（提交/完成，1 分钟粒度，最近 1 小时）
- [x] 12.4 创建 `web/src/components/DurationHistogram.tsx` 任务执行时间分布柱状图
- [x] 12.5 创建 `web/src/components/QueueDepthChart.tsx` Lane 队列深度堆叠面积图
- [x] 12.6 创建 `web/src/components/ResourceGauges.tsx` 系统资源仪表盘（内存、goroutine、CPU）
- [x] 12.7 创建 `web/src/components/ErrorRateChart.tsx` 错误率折线图（含 > 10% 红色高亮带）
- [x] 12.8 实现时间范围选择器（15m、1h、6h、24h）
- [x] 12.9 实现图表 hover tooltip
- [x] 12.10 实现图表图例点击切换数据系列可见性
- [x] 12.11 实现 Prometheus 文本格式解析工具函数
- [x] 12.12 实现 10s 自动刷新
- [x] 12.13 实现指标不可用时的降级显示

## 13. 管理控制台

- [x] 13.1 创建 `web/src/pages/Admin.tsx` 管理页面
- [x] 13.2 实现引擎状态卡片（状态指示灯、uptime、version、活跃工作流数、goroutine 数、内存使用）
- [x] 13.3 实现暂停/恢复按钮（带确认对话框）
- [x] 13.4 实现 Lane 统计表格（名称、队列深度、worker 数、吞吐量、错误率）
- [x] 13.5 实现 Lane 详情展开（队列深度历史图表）
- [x] 13.6 实现 Lane 统计 5s 自动刷新
- [x] 13.7 实现清理工作流按钮（带确认对话框，显示将删除数量）
- [x] 13.8 实现导出调试信息按钮（下载 JSON 文件）
- [x] 13.9 实现导出指标按钮（下载 Prometheus 文本格式）
- [x] 13.10 实现集群节点信息展示（集群模式：节点列表；单机模式：显示 "Standalone mode"）

## 14. Engine 集成

- [x] 14.1 在 `pkg/engine/engine.go` 添加事件广播器字段
- [x] 14.2 在工作流状态变更时触发 workflow.state_changed 事件
- [x] 14.3 在任务状态变更时触发 task.state_changed 事件
- [x] 14.4 编写事件触发集成测试

## 15. 主程序集成

- [x] 15.1 在 `cmd/goclaw/main.go` 初始化 WebSocket handler 和事件广播器
- [x] 15.2 传递事件广播器到 Engine
- [x] 15.3 传递 WebSocket handler 到 API Server
- [x] 15.4 在 shutdown 时关闭所有 WebSocket 连接
- [x] 15.5 添加 UI 启动日志（"Web UI available at http://localhost:8080/ui"）
- [x] 15.6 测试完整启动和关闭流程

## 16. 测试

- [x] 16.1 编写前端单元测试（API 客户端、WebSocket 客户端、store）
- [x] 16.2 编写前端组件测试（状态徽章、DAG 节点、图表组件）
- [x] 16.3 编写 Go WebSocket handler 单元测试
- [x] 16.4 编写 Go 静态资源服务测试（embed、SPA 回退、缓存头、Gzip）
- [x] 16.5 编写 Go 事件广播器测试
- [x] 16.6 编写端到端集成测试（提交工作流 → WebSocket 接收状态变更 → UI 更新）
- [x] 16.7 编写 WebSocket 重连测试
- [x] 16.8 运行完整测试套件确认无回归

## 17. 文档和清理

- [x] 17.1 更新 `README.md` 添加 Web UI 说明和截图占位
- [x] 17.2 更新 `CLAUDE.md` 添加前端构建命令和 Web UI 架构说明
- [x] 17.3 更新 `config.example.yaml` 添加 UI 配置注释
- [x] 17.4 运行 `go fmt` 和 `go vet`
- [x] 17.5 运行前端 lint（eslint + prettier）
- [ ] 17.6 运行 `golangci-lint`
- [ ] 17.7 修复所有 lint 警告
- [ ] 17.8 更新 go.mod 和 go.sum
- [ ] 17.9 验证 Go 测试覆盖率 > 80%
- [ ] 17.10 验证前端构建产物体积 < 5MB（Gzip 前）
