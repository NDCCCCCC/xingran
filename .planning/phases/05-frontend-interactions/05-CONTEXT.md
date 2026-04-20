# Phase 5: 前端交互完善 - Context

**Gathered:** 2026-04-20
**Status:** Ready for planning

<domain>
## Phase Boundary

实现仪表盘编辑所需的前端交互组件，让用户可以配置和管理仪表盘。

**In scope:**
- Widget 选择器组件（添加 Widget 到仪表盘）
- 仪表盘设置面板（名称、描述、权限、刷新间隔）
- 模板预览功能
- Widget 编辑/删除交互

**Out of scope:**
- 后端数据获取（Phase 4 已完成）
- WebSocket 实时刷新（Phase 6）
- 响应式布局优化（Phase 7）
- 拖拽布局优化（Phase 7）

</domain>

<decisions>
## Implementation Decisions

### Widget 选择器设计
- **D-01:** 选择器位置 — 模态框
  - 点击添加按钮后弹出模态框
  - 包含完整的 Widget 配置流程
  - 模态框宽度适中，支持滚动

- **D-02:** 配置流程 — 单页表单
  - 所有配置项在一个表单中
  - 适合高级用户快速配置
  - 表单分区：基本信息、数据源、显示配置

- **D-03:** API 端点选择 — 分类树 + 自动过滤
  - 按模块/分类分组显示端点
  - 用户选择端点后自动过滤支持的 Widget 类型
  - 复用 `EndpointCategory` 类型和 `getAvailableEndpoints` API

### 仪表盘设置面板
- **D-04:** 面板位置 — 侧边抽屉
  - 从右侧滑出
  - 可同时看到仪表盘内容变化
  - 实时预览修改效果

- **D-05:** 权限范围选择 — 下拉 + 条件字段
  - 下拉选择 private/dept/global
  - 部门选择器在 scope=dept 时显示
  - 使用 Ant Design Select 组件

- **D-06:** 刷新间隔配置 — 预设 + 自定义
  - 预设选项：30s/1m/5m/15m/30m/1h
  - 支持自定义输入（数字 + 单位）
  - 使用 Select + InputNumber 组合

### 模板预览功能
- **D-07:** 预览方式 — 全屏覆盖
  - 在当前页面覆盖显示预览
  - 点击外部或按钮关闭
  - 显示完整的仪表盘布局

- **D-08:** 预览数据 — 真实 API 数据
  - 实际调用后端 API 获取真实数据
  - 让用户看到真实效果
  - 失败时显示错误提示

- **D-09:** 应用流程 — 直接创建并编辑
  - 预览中点击"使用此模板"
  - 直接创建仪表盘并跳转编辑页面
  - 用户可以在编辑页修改名称等

### Widget 编辑/删除交互
- **D-10:** 编辑方式 — 侧边抽屉
  - 点击 Widget 后从右侧滑出配置面板
  - 可同时看到 Widget 变化
  - 支持实时预览

- **D-11:** 数据源配置 — 动态表单
  - 根据数据源类型（API/WebSocket/Static）显示不同的配置表单
  - 类型切换时保留已填写的公共字段
  - 使用条件渲染

- **D-12:** 删除确认 — 确认对话框
  - 弹出确认对话框
  - 显示 Widget 名称
  - 需要用户确认后才删除

### Claude's Discretion
- 表单验证规则：必填项、格式校验
- 错误提示位置：字段下方
- 加载状态：按钮 loading + 骨架屏
- 组件库：Ant Design 6（与项目一致）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 前端类型定义
- `ruoyi-react-frontend/src/types/dashboard.ts` — Dashboard、WidgetConfig、DataSourceConfig 等类型
- `ruoyi-react-frontend/src/types/dashboard.ts` — EndpointDetail、EndpointCategory 类型

### 状态管理
- `ruoyi-react-frontend/src/store/dashboardStore.ts` — Zustand store，已有 addWidget/updateWidget/removeWidget 等 actions

### 服务层
- `ruoyi-react-frontend/src/services/dashboardService.ts` — API 调用方法

### Widget 配置
- `ruoyi-react-frontend/src/components/dashboard/widgets/configs/widgetRegistry.ts` — Widget 注册表

### 后端 API
- `internal/api/v1/system/dashboard_handler.go` — GetAvailableEndpoints、GetTemplates 等 Handler
- `internal/services/system/dashboard_service.go` — DashboardService 接口

### Phase 4 决策
- `.planning/phases/04-widget-data-fetching/04-CONTEXT.md` — 后端数据获取决策

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **dashboardStore**: 已有完整的 CRUD actions，可直接使用
- **dashboardService**: 已有 getAvailableEndpoints、getTemplates 等 API 方法
- **widgetRegistry**: Widget 类型注册表，包含默认配置
- **EndpointCategory 类型**: 已定义，可直接用于分类树

### Established Patterns
- **Zustand 状态管理**: 使用 create + persist 中间件
- **Ant Design 组件**: Modal、Drawer、Select、InputNumber、Form
- **API 调用**: 使用 `@/lib/api` 封装的 post/get 方法
- **类型定义**: 完整的 TypeScript 类型

### Integration Points
- **Widget 选择器**: 调用 `getAvailableEndpoints` 获取端点列表
- **设置面板**: 调用 `updateDashboard` 保存修改
- **模板预览**: 调用 `getTemplates` 和 `createFromTemplate`
- **Widget 编辑**: 调用 `updateWidget` action

### 待实现组件
```
src/components/dashboard/
├── WidgetSelector/
│   ├── index.tsx           # 模态框入口
│   ├── WidgetTypeSelect.tsx # Widget 类型选择
│   ├── EndpointSelector.tsx # API 端点选择（分类树）
│   └── DataSourceForm.tsx  # 数据源配置表单
├── DashboardSettings/
│   ├── index.tsx           # 侧边抽屉
│   └── ScopeSelector.tsx   # 权限范围选择器
├── TemplatePreview/
│   └── index.tsx           # 全屏预览覆盖层
└── WidgetEditor/
    └── index.tsx           # Widget 编辑抽屉
```

</code_context>

<specifics>
## Specific Ideas

- 参考 Zabbix 的 Widget 配置界面：分类清晰、配置直观
- API 端点选择器参考 VS Code 的命令面板：分类 + 搜索
- 模板预览参考 Notion 的模板预览：全屏覆盖 + 快速应用

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 05-frontend-interactions*
*Context gathered: 2026-04-20*
