## Why

GoClaw 已具备完整的 HTTP API、gRPC API 和 Prometheus 指标体系，但缺少可视化管理界面。运维人员和开发者只能通过 curl/gRPC 客户端或 Swagger UI 操作系统，无法直观地监控工作流执行状态、查看 DAG 拓扑、分析性能指标。Web UI 是 Phase 3 的最后一块拼图，将显著提升系统的可用性和可观测性。

## What Changes

- 新增嵌入式 Web UI 前端（React + TypeScript），通过 Go embed 打包到二进制中
- 新增 WebSocket 端点，支持工作流状态实时推送
- 新增 DAG 可视化组件，展示任务依赖关系和执行进度
- 新增指标仪表盘，展示 Prometheus 指标的时序图表
- 新增管理控制台，支持工作流提交、取消、暂停/恢复等操作
- 新增静态资源服务中间件，挂载到 `/ui` 路径
- 修改 HTTP Server 配置，添加 Web UI 相关配置项

## Capabilities

### New Capabilities

- `dashboard-layout`: 主布局框架，包括导航栏、侧边栏、页面路由、主题切换（亮/暗）
- `workflow-management`: 工作流列表页（分页、过滤、搜索）、详情页（状态、任务列表、元数据）、提交/取消操作
- `dag-visualization`: DAG 拓扑图渲染（基于 dagre/ELK 布局算法）、节点状态着色、执行进度动画、缩放/平移交互
- `metrics-dashboard`: 实时指标图表（基于 recharts）、工作流吞吐量、任务执行时间分布、Lane 队列深度、系统资源使用
- `realtime-updates`: WebSocket 连接管理、工作流状态变更推送、自动重连、心跳保活
- `admin-controls`: 引擎状态管理（暂停/恢复）、Lane 统计查看、集群节点信息、调试信息导出
- `static-embedding`: Go embed 静态资源打包、SPA 路由回退、Gzip 压缩、缓存控制头

### Modified Capabilities

- `http-server`: 添加 WebSocket 升级端点和静态资源服务路由

## Impact

- **新增依赖**: React 18、TypeScript、Vite（构建工具）、dagre-d3 或 ELK（DAG 布局）、recharts（图表）、gorilla/websocket（WebSocket 服务端）
- **构建流程**: 需要在 Go 构建前先构建前端资源，Makefile 添加 `build-ui` 目标
- **二进制体积**: 嵌入前端资源预计增加 2-5MB（Gzip 后）
- **API 变更**: 新增 `/ws/workflows/{id}` WebSocket 端点，新增 `/ui/*` 静态资源路由
- **配置变更**: `config.yaml` 新增 `ui` 配置段（启用/禁用、基础路径）
- **受影响代码**: `pkg/api/server.go`、`pkg/api/router.go`、`cmd/goclaw/main.go`、`config/config.go`
