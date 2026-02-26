## Context

GoClaw 已实现完整的后端 API 层（HTTP REST + gRPC）和 Prometheus 指标体系。当前用户只能通过 curl、gRPC 客户端或 Swagger UI 与系统交互。Phase 3 路线图要求提供 Web UI 作为最终的用户界面层。

现有后端能力：
- HTTP API：工作流 CRUD、任务结果查询、健康检查（端口 8080）
- gRPC API：WorkflowService + AdminService（端口 9090）
- Prometheus 指标：工作流、任务、Lane、HTTP 四类指标（端口 9091）
- DAG 编译器：拓扑排序、分层执行计划
- Badger 持久化存储

## Goals / Non-Goals

**Goals:**
- 提供嵌入式 Web UI，随 Go 二进制一起分发，零额外部署
- 实时展示工作流执行状态和 DAG 拓扑
- 提供指标仪表盘，替代直接查看 Prometheus 端点
- 支持工作流管理操作（提交、取消、查看详情）
- 支持亮/暗主题切换
- 首屏加载 < 3s，交互响应 < 200ms

**Non-Goals:**
- 不实现用户认证/授权（后续迭代）
- 不实现多租户隔离
- 不实现自定义仪表盘编辑（固定布局）
- 不实现移动端适配（桌面优先）
- 不替代 Grafana（仅提供内置基础指标视图）

## Decisions

### Decision 1: 前端框架 — React 18 + TypeScript + Vite

**选择**: React 18 + TypeScript + Vite 构建工具链

**替代方案**:
- Vue 3 + Vite：生态略小，DAG 可视化库选择较少
- Svelte：社区较小，企业级组件库不够成熟
- 纯 Go 模板（html/template）：交互能力弱，无法实现实时更新和复杂可视化

**理由**: React 生态最成熟，dagre/reactflow 等 DAG 可视化库丰富，TypeScript 提供类型安全，Vite 构建速度快。团队熟悉度高。

### Decision 2: 静态资源分发 — Go embed

**选择**: 使用 `go:embed` 将构建后的前端资源嵌入 Go 二进制

**替代方案**:
- 独立 Nginx 部署：增加运维复杂度
- CDN 分发：需要网络访问，不适合内网部署

**理由**: 单二进制分发是 GoClaw 的核心优势，embed 保持这一特性。生产环境零配置，开发环境支持代理到 Vite dev server。

### Decision 3: DAG 可视化 — React Flow

**选择**: React Flow（基于 dagre 布局）

**替代方案**:
- D3.js 手动绘制：开发量大，需要自行处理布局算法
- vis.js：功能全面但体积大，React 集成不够原生
- Mermaid：静态渲染，无法实现实时状态更新

**理由**: React Flow 原生 React 组件，支持自定义节点、边样式，内置缩放/平移，dagre 自动布局。可以轻松实现节点状态着色和执行动画。

### Decision 4: 实时更新 — WebSocket

**选择**: WebSocket 推送工作流状态变更

**替代方案**:
- SSE（Server-Sent Events）：单向通信，浏览器兼容性好但功能受限
- 轮询：实现简单但延迟高、资源浪费
- gRPC-Web：需要额外代理层

**理由**: WebSocket 双向通信，延迟低。服务端使用 gorilla/websocket，与现有 chi router 集成简单。支持心跳保活和自动重连。

### Decision 5: 图表库 — Recharts

**选择**: Recharts（基于 D3 的 React 图表库）

**替代方案**:
- Chart.js + react-chartjs-2：功能全面但 React 集成不够声明式
- ECharts：功能强大但体积大（500KB+）
- Nivo：美观但自定义能力有限

**理由**: Recharts 纯 React 声明式 API，体积适中（~150KB），支持响应式布局，满足时序图表、柱状图、饼图等需求。

### Decision 6: UI 组件库 — Tailwind CSS + Headless UI

**选择**: Tailwind CSS 4 + Headless UI 组件

**替代方案**:
- Ant Design：功能全面但体积大，样式定制困难
- Material UI：Google 风格，与 GoClaw 品牌不匹配
- Shadcn/ui：基于 Radix，但需要逐个复制组件

**理由**: Tailwind 原子化 CSS 体积小（purge 后 < 20KB），Headless UI 提供无样式可访问组件（Dialog、Menu、Tab），完全可控的视觉风格。支持亮/暗主题通过 CSS 变量切换。

### Decision 7: 状态管理 — Zustand

**选择**: Zustand 轻量状态管理

**替代方案**:
- Redux Toolkit：功能全面但样板代码多
- Jotai/Recoil：原子化状态，适合细粒度但学习曲线陡
- React Context：内置但大规模状态管理性能差

**理由**: Zustand API 极简，无 Provider 包裹，支持中间件（devtools、persist），适合中等规模应用。WebSocket 消息可直接更新 store。

### Decision 8: 构建集成 — Makefile + 条件编译

**选择**: Makefile 添加 `build-ui` 目标，Go 使用 build tag 控制 embed

**方案**:
- `make build-ui`：运行 `npm run build`，输出到 `web/dist/`
- `make build`：自动触发 `build-ui`，然后 `go build -tags embed_ui`
- 开发模式：不带 tag 编译，UI 路由返回提示信息，前端通过 Vite proxy 访问 API

**理由**: 条件编译避免未安装 Node.js 时构建失败。CI/CD 可选择是否包含 UI。

## Risks / Trade-offs

- **二进制体积增加** → 前端资源 Gzip 后预计 2-5MB，可接受。提供不含 UI 的构建选项（不带 embed_ui tag）
- **Node.js 构建依赖** → 仅构建时需要，运行时无依赖。CI 环境需安装 Node.js 18+
- **WebSocket 连接管理** → 大量客户端同时连接可能消耗资源 → 实现连接数限制和心跳超时清理
- **前端安全** → 无认证机制，内网部署可接受 → 后续迭代添加认证
- **浏览器兼容性** → 仅支持现代浏览器（Chrome/Firefox/Edge 最近 2 个版本）
- **指标数据量** → 前端仅展示最近 1 小时数据，避免内存溢出 → 长期数据仍需 Grafana

## Open Questions

- 是否需要支持 i18n（国际化）？当前计划仅支持英文界面
- 是否需要在 Web UI 中集成 Saga 事务管理界面（依赖 Week 11 实现）？
