# Phase 53: W4 — Frontend Drawer + Progress Dialog + API Wrappers - Context

**Gathered:** 2026-07-07
**Status:** Ready for planning
**Source:** v1.19 STATE.md 锁定决策 + Phase 52 CONTEXT D-01..D-16（HTTP 契约 + audit + 权限已落地）+ Phase 51 CONTEXT D-10..D-18（PortWriteService/BatchResult 契约）+ REQUIREMENTS.md UI-01..UI-06 + ROADMAP Phase 53 段（7 条 Success Criteria）+ 前端代码 scout（ports/index.tsx, devices/index.tsx, MACHistoryPage.tsx, networkApi.ts, opsApi.ts, menuStore）

<domain>
## Phase Boundary

Phase 52 W3 已落地 6 个 HTTP 写端点 + `network:port:write` 权限 + audit 表。本 phase 是这 6 个端点的**唯一前端消费者**：在现有端口列表页 (`/network/ports`) 加挂写操作 UI（行内单端口操作 + 批量操作 Drawer），并提供 6 个权限隔离的 API wrapper 函数。

**In scope**:
- `xingran-react-frontend/src/lib/api/networkApi.ts` 扩展：导出 6 个 wrapper 函数 `writeShutdown` / `writeUndoShutdown` / `writeDescription` / `writeDot1xEnable` / `writeDot1xDisable` / `batchWritePorts`（ROADMAP Success Criteria #3）
- `xingran-react-frontend/src/components/network/port-write/PortWriteModal.tsx` 新建：**1 个统一 Modal**，props 带 `action` 切换 5 种单端口操作（D-01）
- `xingran-react-frontend/src/components/network/port-write/BulkWriteDrawer.tsx` 新建：批量操作 Drawer（已选端口只读列表 + 操作类型选择 + reason + indeterminate spinner + 结果面板 + 失败重试，D-02/D-03）
- `xingran-react-frontend/src/pages/network/ports/index.tsx` 改：新增"操作"列（Dropdown 触发 PortWriteModal）+ 顶部"批量配置(N)"按钮（触发 BulkWriteDrawer，D-06）+ 批量进行时禁用"刷新"按钮（UI-05）
- 操作原因（reason）字段 UX：预置常用原因 Select + "其他"展开 TextArea，5-200 字符校验（D-07）；description action 时 reason 可空（D-08）
- 权限 gating：`useMenuStore((s) => s.permissions)` + `hasPermission("network:port:write")` 控制操作入口可见性（D-05）
- 审计跳转：Toast "查看审计日志" 链接 `navigate('/monitor/logs?module=端口管理')`（D-04）

**Out of scope**:
- BATCH-05 批量实时进度反馈（WebSocket/SSE）— 52 D-12 推到 v1.19.x；本 phase 用 indeterminate spinner
- `sys_port_write_audit` 详情查看 UI — v1.19.x+（audit 表后端就绪，前端查看页后续补）
- 真机 SSH e2e / 真机 UAT — Phase 54
- 跨设备批量 — 51 D-17 已锁 `ErrMixedDevices` 拒绝
- 写命令前设备可达性预检（FUTURE-07）— v1.19.x+
- description 厂商字符上限精细化校验 — 后端设备侧已校验，前端只做保守上限（D-08 Claude discretion）
- API wrapper 的 sentinel error → 中文翻译 — 后端 handler 已翻译为 4xx + 中文 message，前端透传（D-11）

</domain>

<decisions>
## Implementation Decisions

### 单端口操作 Modal 形态（UI-01 / UI-02）

- **D-01: 1 个统一 PortWriteModal.tsx + action 参数（非 5 个独立 Modal）**
  - 新建 `src/components/network/port-write/PortWriteModal.tsx`
  - Props: `{ open, action, portRecord, onClose, onSuccess }`，`action ∈ {"shutdown","undo_shutdown","description","dot1x_enable","dot1x_disable"}`
  - 内部按 `action` 切换：Modal 标题（如"关闭端口 - GE0/0/1"）、图标、是否显示"新描述"输入框（仅 description action 显示）
  - 5 个 action 共享：reason 字段（D-07）+ 确认按钮 + 调对应 wrapper（D-09 wrapper 列表）
  - 端口列表页"操作"列 Dropdown 5 个 menu item 触发同一个 Modal 并传入 action（参考 devices/index.tsx 行内按钮组模式）
  - **不**建 5 个独立组件（避免 reason 字段×5 + 确认逻辑×5 重复）；**不**用 antd Modal.confirm（reason 字段 + description 输入需 Form，confirm 不够）

- **D-02: reason 字段输入形态 = 预置常用原因 Select + "其他"展开 TextArea**
  - Select 选项（硬编码 const 数组）：`故障排查` / `安全合规` / `业务变更` / `临时测试` / `其他...`
  - 选"其他..."时展开 `Input.TextArea`（`rows={2}`），placeholder `"请输入操作原因（5-200 字符）"`
  - 选预设项时 reason 值 = 预设字符串（如 `"故障排查"`），5 字符满足下限
  - 实时字数计数（`showCount` + `maxLength={200}`）+ `validator` 校验 `5 <= trim(value).length <= 200`
  - 预设原因列表放 `src/components/network/port-write/constants.ts`（D-09 单 plan 内可调）

- **D-03: description action 特例 = reason 可空，"新描述"必填**
  - action=description 时：Modal 多一个"新描述"`Input`（必填，`maxLength={80}` 保守上限，placeholder `"请输入新端口描述"`）
  - action=description 时 reason 字段**不**必填（"新描述"本身已说明意图）；其他 4 个 action reason 必填
  - 校验规则按 action 条件分支（antd Form `rules` 动态判断）
  - 提交体：description action `{portId, description, reason?}`；其他 action `{portId, reason}`（对齐后端 52 D-16 PortWriteRequest struct）

### 批量操作 Drawer（UI-03 / UI-05）

- **D-04: BulkWriteDrawer 入口 = 端口列表页 rowSelection + 顶部"批量配置(N)"按钮**
  - 端口列表页顶部按钮区（现有 NetworkExport / 采集所有设备 / 批量删除 旁）新增"批量配置(N)"按钮
  - `disabled={selectedRowKeys.length === 0}`，与现有"批量删除(N)"按钮 UX 完全一致
  - 点击打开 BulkWriteDrawer，Drawer 内**不**重新选择端口（直接读 `selectedRowKeys`）
  - BulkWriteDrawer Props: `{ open, selectedPorts, onClose, onSuccess }`，`selectedPorts: DevicePortStatus[]`（含 deviceName/interfaceName/adminStatus 供只读展示）

- **D-05: 批量进度反馈 = Indeterminate spinner + 最终结果面板（不伪造 X/Y 进度）**
  - 提交后 Drawer 内容区切到执行视图：`<Spin tip="正在批量配置...（预计 ~1s/端口）" />`
  - **不**用 Progress 组件伪造 X/Y（后端 batch 同步阻塞 52 D-11，无实时进度；ROADMAP 字面 "X/Y ports" 降级为 indeterminate + 耗时提示）
  - 收到 BatchResult 响应后切到结果面板，三分区 Statistic 卡片：
    - ✓ 成功 `result.succeeded.length`（绿色）
    - ⚠ 跳过 `result.skipped.length`（灰色，Tag "无需操作"）
    - ✗ 失败 `result.failed.length`（红色）
  - 失败明细用 antd `Table`：列 = 接口名 / 错误原因（`port.error`），可展开行显示完整 `port.commandSent`
  - 跳过明细折叠展示（`Collapse`，默认收起）

- **D-06: 失败端口支持"一键重试"**
  - 结果面板失败分区底部"重试失败端口(N)"按钮（`type="primary"`，`disabled={result.failed.length === 0}`）
  - 点击：收集 `result.failed.map(p => p.portID)` → 新 `BatchWriteRequest{deviceID, action, portIDs: failedIds, description?}` → 调 `batchWritePorts` wrapper
  - Drawer 状态机：`select → executing → result → (retry) → executing → result'`
  - 重试结果**替换**当前结果面板（不累加历史）；成功+跳过的端口不进入重试范围
  - 若重试后仍有失败，按钮继续可用（支持多次重试，直到失败清零或用户放弃）

- **D-07: 批量操作期间禁用"刷新"按钮（UI-05）**
  - 端口列表页 `handleRefresh` 按钮的 `loading` / `disabled` 状态绑定 `batchInProgress` 全局信号
  - `batchInProgress` 通过 BulkWriteDrawer 的 `onExecutingChange` callback 上抛（父组件 useState）
  - 批量进行时刷新会与 Enqueue 触发的采集竞态（PROJECT.md Pitfall #6 / 53 CONTEXT D-11），禁用是硬约束
  - 同时禁用"采集所有设备"按钮（同类竞态）

### API wrapper（UI-06 / Success Criteria #3）

- **D-08: networkApi.ts 导出 6 个 wrapper，跟随 `post()` 包装风格（非 opsApi factory）**
  - 扩展 `xingran-react-frontend/src/lib/api/networkApi.ts`，新增 6 个 async 函数：
    ```ts
    export const writeShutdown     = (portId: string, reason: string) => post("/network/ports/write/shutdown", { portId, reason });
    export const writeUndoShutdown = (portId: string, reason: string) => post("/network/ports/write/undo-shutdown", { portId, reason });
    export const writeDescription  = (portId: string, description: string, reason?: string) => post("/network/ports/write/description", { portId, description, reason });
    export const writeDot1xEnable  = (portId: string, reason: string) => post("/network/ports/write/dot1x-enable", { portId, reason });
    export const writeDot1xDisable = (portId: string, reason: string) => post("/network/ports/write/dot1x-disable", { portId, reason });
    export const batchWritePorts   = (req: BatchWriteRequest) => post("/network/ports/write/batch", req);
    ```
  - `BatchWriteRequest` 类型 = `{ deviceId: string; action: "shutdown"|"undo_shutdown"|"description"|"dot1x_enable"|"dot1x_disable"; portIds: string[]; description?: string }`（对齐后端 51 CONTEXT service 签名）
  - 返回值：单端口 wrapper 返回 `Promise<PortResult>`，batch wrapper 返回 `Promise<BatchResult>`（类型加到 `src/types/` 或 networkApi.ts inline）
  - **不**用 opsApi factory（opsApi 是 operations 模块专用，network 模块惯例是页面/wrapper 直接 `post()`，见 devices/index.tsx:411/450/491）
  - **不**在 wrapper 做 sentinel error → 中文翻译（后端 52 handler 已翻译，前端透传 response.Error message）

### 权限 gating（UI-01 / Success Criteria #4）

- **D-09: 权限源 = `useMenuStore`，非 ROADMAP 写的 `useAuthStore`（笔误纠正）**
  - 代码现状（MACHistoryPage.tsx + devices/index.tsx:260）：`const menuPermissions = useMenuStore((s) => s.permissions); const hasPermission = (p: string) => menuPermissions.includes(p);`
  - `authStore` 只管 token/refresh，**不含** permissions 数组 — ROADMAP Success Criteria #4 写 useAuthStore 是笔误
  - 端口列表页顶部加：`const canWrite = hasPermission("network:port:write");`
  - 控制：行内"操作"Dropdown 列（`canWrite ? <Dropdown/> : null`）+ "批量配置"按钮 disabled 状态
  - superadmin/admin 走后端旁路自动放行，前端 menuStore.permissions 已含 network:port:write（52 migration_202 GrantNewMenuToRolesHavingParent 已 seed）

### 审计跳转（UI-04）

- **D-10: Toast "查看审计日志" 链接 = `navigate('/monitor/logs?module=端口管理')`**
  - 单端口操作成功 / 批量操作完成后，`message.success(...)` 或 antd `App.useApp()` Toast
  - Toast 含"查看审计日志"链接（antd `message` 不直接支持链接，用 `App.useApp()` 的 `message.open({ content: <><span>操作成功</span><a onClick>查看审计日志</a></> })`）
  - 链接 `navigate('/monitor/logs?module=' + encodeURIComponent('端口管理'))`，操作日志页读 URL query 预填 module 筛选
  - **route 路径是 `/monitor/logs`（非 `/monitor/operlog`）**：React Router 注册的实际 path 是 `monitor/logs`（源码 src/pages/monitor/logs/index.tsx），API 端点才是 `/monitor/oper-logs/list`（reconciliation/exceptions/index.tsx:343-344 注释已确认此偏差）
  - **path C 约束**：52 D-13 锁定 `audit.oper_log_id` 列保留 NULL，audit_ids 嵌在 operlog `oper_param` JSON 里 → 前端**不能**精准跳单条 audit，只能跳模块级列表（module=端口管理 filter）
  - 先例：`src/pages/asset/reconciliation/exceptions/index.tsx:15` 已有 `operlog_btn (查看日志 /monitor/operlog?bizId=...)` 模式（planner 确认该页是否已支持 module filter query param，若无则本 phase 顺带补 operlog 页的 URL query filter 读取）

### Claude's Discretion

- **PortWriteModal / BulkWriteDrawer 内部样式**：Drawer 宽度（建议 720px）、Modal 宽度（建议 520px）、配色（跟随现有 Tag success/warning/error 风格）由 planner 按 antd 惯例
- **结果面板 Statistic 卡片布局**：三列 Row+Col vs 紧凑 Space，planner 选
- **预设原因列表具体项**：D-02 给了 4 项起点（故障排查/安全合规/业务变更/临时测试），planner 可按运维场景增减
- **description 长度上限**：D-03 给 80 字符保守值；华为/H3C/锐捷实际厂商上限不同（华为 24/80 视版本），前端不做厂商分支，统一 80 字符保守上限（后端设备侧会拒超长，前端保守值减少往返）
- **wrapper 返回类型定义位置**：`PortResult` / `BatchResult` / `BatchWriteRequest` TypeScript 类型加到 `src/types/index.ts`（对齐 DevicePortStatus 现有位置）或 networkApi.ts inline — planner 按 types/ 目录惯例
- **BulkWriteDrawer 的 action 选择器**：Select 5 个 action（与单端口 Dropdown 同 5 项）vs Radio.Group；批量必须同 device（51 D-17），action 是单选 — planner 选
- **失败明细 Table 的 commandSent 展示**：完整命令可能很长，用 `Typography.Text ellipsis` + Tooltip 或可展开行 — planner 选

</decisions>

<canonical_refs>
## Canonical References

**下游 agent (planner / researcher) 必须先读这些。**

### Phase 52 落地契约（本 phase 直接消费）
- `.planning/phases/52-w3-router-handler-operlog-permission-migration/52-CONTEXT.md` — D-01..D-16，HTTP 契约 + audit + 权限 + Path C（audit_ids 嵌 oper_param，oper_log_id NULL）
- `.planning/phases/52-w3-router-handler-operlog-permission-migration/52-01-SUMMARY.md` — 6 端点最终形状 + PortWriteRequest struct（handler 入参）
- `internal/api/v1/network/port_write_handler.go` — 6 handler，确认 request body shape（`{portId, description?, reason?}`）+ sentinel→HTTP 翻译（前端透传 message）
- `internal/api/v1/network/port_write_router.go` — 6 kebab 路径（`/network/ports/write/{shutdown,undo-shutdown,description,dot1x-enable,dot1x-disable,batch}`），前端 wrapper URL 对齐
- `pkg/permission/config.go:189` — `NetworkPortWrite = "network:port:write"` 常量（前端 menuStore.permissions 比对此串）

### Phase 51 service 契约（PortResult / BatchResult 形状）
- `.planning/phases/51-w2-portwriteservice-batch-orchestrator-mock-tests/51-CONTEXT.md` — D-10..D-18，service 签名 + BatchResult 三数组 + fail-fast 语义
- `internal/services/portwrite/port_write_service.go` — `PortResult{PortID,Action,Status,NoOp,CurrentState,Error,CommandSent}` + `BatchWriteRequest{DeviceID,Action,PortIDs,Description}` + `BatchResult{Succeeded,Failed,Skipped []PortResult}`（前端 TypeScript 类型镜像此形状）
- `Status ∈ {"succeeded","failed","skipped"}` + `NoOp bool`（skipped 与 NoOp 同义，前端结果面板按 Status 分区）

### 前端现有模式（scout 确认）
- `xingran-react-frontend/src/pages/network/ports/index.tsx` — **本 phase 改造目标页**：现有 rowSelection + 批量删除按钮 + 顶部按钮区布局 + useTableManager + usePagination；新增"操作"列 + "批量配置"按钮的挂载点
- `xingran-react-frontend/src/pages/network/devices/index.tsx:193-260` — "操作"列 + 行内按钮组先例（handleCollectPorts/openDetailModal 模式，本 phase 端口页"操作"列 Dropdown 镜像此）
- `xingran-react-frontend/src/pages/network/devices/index.tsx:260` — `useMenuStore((s) => s.permissions)` + `hasPermission` 权限 gating 先例（D-09 据此）
- `xingran-react-frontend/src/pages/network/mac/history/MACHistoryPage.tsx` — 同样 menuStore 权限 gating（`canExport = hasPermission("network:mac:export")`，D-09 第二先例）
- `xingran-react-frontend/src/lib/api/networkApi.ts` — **本 phase 扩展目标文件**：现有 4 个 export（queryMACHistory/getMACEvents/exportMACHistory/batchExport），加 6 个 write wrapper（D-08）
- `xingran-react-frontend/src/lib/api.ts` — `post()` 包装函数（处理 SM2+SM4 加密 + token refresh，wrapper 直接调）
- `xingran-react-frontend/src/lib/opsApi.ts` — operations 模块 API factory（**本 phase 不沿用**，network 模块用 post() 直接调，D-08 已决策）
- `xingran-react-frontend/src/pages/asset/reconciliation/exceptions/index.tsx:15,343-344` — "查看日志"跳转 operlog 先例 + route 路径偏差注释（`/monitor/logs` 而非 `/monitor/operlog`，D-10 据此）
- `xingran-react-frontend/src/pages/monitor/logs/index.tsx` — operlog 页本体（route `/monitor/logs`，API `/monitor/oper-logs/list`）；planner 确认是否已读 URL query filter，若无则本 phase 补 module query param 预填
- `xingran-react-frontend/src/utils/errorHandler.ts` — `withErrorHandling` wrapper（Toast + 错误处理，本 phase 单端口操作复用）
- `xingran-react-frontend/src/hooks/useTableManager.ts` — 表格 + 排序 + 分页 hook（端口列表页已用，批量按钮接入点）

### v1.19 锁定决策
- `.planning/PROJECT.md` §"Current Milestone: v1.19" — init 决策（device_id 直连 / 厂商 map / OperType / 权限隔离 / sys_port_write_audit 真相源）
- `.planning/REQUIREMENTS.md` UI-01..UI-06 + AUDIT/PERM/INFRA/CONV/PORT/BATCH 段
- `.planning/ROADMAP.md` Phase 53 段 — 7 条 Success Criteria（注意 #4 写 useAuthStore 是笔误，D-09 纠正为 useMenuStore；#2 写 "X/Y ports" 降级为 indeterminate spinner，D-05）
- `.planning/STATE.md` §"Critical Pitfalls → Mitigation Map" — Pitfall #6（batch 与 Enqueue 竞态，UI-05 禁刷新落地）

### antd 组件参考
- antd `Drawer` / `Modal` / `Form` / `Select` / `Input.TextArea` / `Statistic` / `Table` / `Spin` / `Dropdown` / `message`（App.useApp）— 均为现有依赖，无新包

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `useMenuStore((s) => s.permissions)` + `hasPermission(perm)` — 权限 gating，D-09 直接复用
- `post("/network/...", body)` from `@/lib/api` — wrapper 调用底座（SM2+SM4 加密 + token refresh 已内置）
- `useTableManager` + `usePagination` — 端口列表页已用，rowSelection + selectedRowKeys 已就绪，D-06 批量按钮直接读 selectedRowKeys
- `withErrorHandling(fn, { onSuccess, onError })` — 单端口操作 Toast + 错误处理
- `App.useApp()` 的 `message` — Toast 反馈（支持 ReactNode content，D-10 链接 Toast 用此）
- antd `Table` expandable 行 — 失败明细 commandSent 长文本展开（端口列表页 expandable 已有先例）
- `NetworkExport` / `BatchExportModal` — 顶部按钮区组件先例（D-06 "批量配置"按钮挂同位置）

### Established Patterns
- **行内"操作"列 + Dropdown/按钮组**：devices/index.tsx:193-260（本 phase 端口页"操作"列镜像）
- **顶部"批量X(N)"按钮 + rowSelection**：端口页现有"批量删除(N)"（D-06 "批量配置(N)"完全对齐此 UX）
- **权限 gating 隐藏入口**：`canX = hasPermission(...)` + 条件渲染（MACHistoryPage canExport、devices 页）
- **Drawer 结果展示**：ReconciliationDrawer / FixSuggestionDetailDrawer（antd Drawer + 内部分区，D-05 结果面板参考）
- **Modal + Form + reason 输入**：duty/management/modals/ManualScheduleModal（Form + 校验 + 提交模式，PortWriteModal 参考）
- **message.open 带 ReactNode**：D-10 Toast 链接（antd message 支持 content 为 ReactNode）

### Integration Points
- `xingran-react-frontend/src/pages/network/ports/index.tsx` — 主改造点：加"操作"列 + "批量配置"按钮 + canWrite gating + batchInProgress 禁刷新
- `xingran-react-frontend/src/lib/api/networkApi.ts` — 扩展 6 wrapper
- `xingran-react-frontend/src/components/network/port-write/` — 新目录：PortWriteModal.tsx + BulkWriteDrawer.tsx + constants.ts（预设原因）
- `xingran-react-frontend/src/types/index.ts`（或 networkApi.ts inline）— PortResult / BatchResult / BatchWriteRequest TypeScript 类型
- `xingran-react-frontend/src/pages/monitor/logs/index.tsx` — 可能需补 URL query filter 预填（planner 确认，D-10 依赖）

</code_context>

<specifics>
## Specific Ideas

- **D-05 诚实进度**：用户明确选 indeterminate spinner + 耗时提示，不伪造 X/Y 进度。ROADMAP 字面 "X/Y ports" 因后端 batch 同步阻塞（52 D-11）不可行；若未来 BATCH-05（v1.19.x）落地 SSE/WS，再升级为真实 X/Y。本 phase 在结果面板用 `Statistic` 三卡片（成功 N / 跳过 N / 失败 N）补足"进度感"。
- **D-06 重试只针对 Failed**：一键重试收集 `result.failed.map(p => p.portID)`，**不**重试 skipped（skipped = 设备已是目标态，重试无意义）。重试范围自然收敛。
- **D-07 预设原因起点 4 项**：故障排查 / 安全合规 / 业务变更 / 临时测试。这 4 项覆盖运维常见场景；"其他..."覆盖长尾。planner 可按实际运维调优。
- **D-08 wrapper 不翻译 sentinel**：后端 52 handler 已把 `ErrPortNotFound` → `response.Error(c, 404, "端口不存在")` 等中文 message，前端 `post()` 拦截器已统一弹错误 Toast，wrapper 只透传 Promise reject。避免前后端双重翻译。
- **D-10 route 路径偏差**：reconciliation/exceptions/index.tsx:343-344 注释明确"React Router 注册的实际 path 是 'monitor/logs'，直接 navigate '/monitor/oper-logs' 或 '/logs' 都会 fallback"——本 phase 跳转必须用 `/monitor/logs`，不是 `/monitor/operlog`（后者会触发 dashboard fallback，memory `xingran-dynamic-route-menu-not-seeded-fallback-dashboard` 相关坑）。
- **Toast 链接实现**：antd `message.success(text)` 不支持链接，改用 `App.useApp()` 的 `message.open({ type:'success', content: <><span>操作成功，</span><Link onClick={...}>查看审计日志</Link></>, duration: 5 })`。需要 `App` 上下文（端口页已用 `const { message } = App.useApp()`，D-10 复用）。

</specifics>

<deferred>
## Deferred Ideas

- **BATCH-05 批量实时进度反馈（SSE/WebSocket）** — v1.19.x：52 D-12 已锁，需重构 Phase 51 batch 同步契约为流式 + 动 28 测试。本 phase D-05 indeterminate spinner 是 MVP 过渡。
- **`sys_port_write_audit` 详情查看 UI** — v1.19.x+：audit 表后端就绪（52 落地），前端查看页（按 device_id/port_id/operator/时间范围 筛选 + before/after diff 展示）后续补。D-10 的 Toast 跳 operlog 是模块级过渡。
- **operlog → audit 精准反查** — v1.19.x+：52 D-13 Path C 使 oper_log_id 列为 NULL，operlog→audit 只能靠 oper_param.audit_ids JSON 反查。未来若加 audit 详情页，可读 oper_param.audit_ids 跳转。
- **description 厂商字符上限精细化** — v1.19.x+：华为/H3C/锐捷端口描述字符上限不同（华为 24/80 视版本），前端 D-03 统一 80 字符保守值；未来按 device.vendor 分支校验。
- **预设原因可配置化** — v1.19.x+：D-07 预设原因硬编码 const，未来可下沉到 sys_dict（字典管理）让运维自定义。
- **批量操作历史记录** — v1.19.x+：BulkWriteDrawer 当前不保留历史批次记录；未来可加"最近 10 次批量操作"侧栏（读 sys_port_write_audit GROUP BY oper_log_id）。

### Reviewed Todos (not folded)
None — cross_reference_todos 未发现匹配本 phase 的 pending todo。

</deferred>

---

*Phase: 53-w4-frontend-drawer-progress-dialog-api-wrappers*
*Context gathered: 2026-07-07*
