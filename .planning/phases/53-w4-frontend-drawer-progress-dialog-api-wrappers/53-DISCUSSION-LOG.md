# Phase 53: W4 — Frontend Drawer + Progress Dialog + API Wrappers - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-07
**Phase:** 53-w4-frontend-drawer-progress-dialog-api-wrappers
**Areas discussed:** Modal 形态, 批量进度 UX, 失败重试, 审计跳转, 权限源, 批量入口, reason 字段, description 特例, Plan 拆分, Bundle 策略

---

## Modal 形态

| Option | Description | Selected |
|--------|-------------|----------|
| 1 个统一 Modal + action 参数 | PortWriteModal.tsx，props 带 action，5 action 共享 reason + 确认逻辑，description 多一个输入框。~250 行 | ✓ |
| 5 个独立 Modal 组件 | ShutdownModal / UndoShutdownModal / ... 各一文件，ROADMAP 字面。重复代码多 | |
| 1 个 Modal + 行内 Dropdown 触发 | 端口表加"操作"列 Dropdown，点选开同一个 Modal | |

**User's choice:** 1 个统一 Modal + action 参数
**Notes:** 用户选了最左侧选项（统一 Modal + action 参数）。第三个选项（Dropdown 触发）实际上与第一项不冲突——D-01 落地为"统一 Modal + 端口列表页操作列 Dropdown 触发"，两者结合。

---

## 批量进度 UX

| Option | Description | Selected |
|--------|-------------|----------|
| Indeterminate spinner + 最终结果面板 | Spin + "正在批量配置...(预计 ~1s/端口)"，返回后三卡片结果分区 + 失败明细 Table。不伪造 X/Y | ✓ |
| 耗时估算的假进度条 + 结果面板 | Progress animate 到 90%，响应后跳 100%。有进度感但是模拟值 | |
| 两阶段：预检 → 执行 | Drawer 第一步选端口+操作，第二步执行视图+结果。多一次交互 | |

**User's choice:** Indeterminate spinner + 最终结果面板
**Notes:** 用户明确选诚实方案。ROADMAP 字面 "X/Y ports" 因后端 batch 同步阻塞（52 D-11）不可行，降级为 indeterminate + 耗时提示。结果面板用 Statistic 三卡片（成功/跳过/失败）补足进度感。

---

## 失败重试

| Option | Description | Selected |
|--------|-------------|----------|
| 支持一键重试失败端口 | 结果面板失败分区"重试失败端口(N)"按钮，Failed[].portID → 新 batch | ✓ |
| 只读展示 + 引导单端口重试 | 失败明细只读，提示去端口列表页单端口重试 | |
| 失败 = 可展开行，单端口内联重试 | 失败列表可展开，单端口"重试"按钮调单端口 wrapper | |

**User's choice:** 支持一键重试失败端口
**Notes:** 用户选最友好方案。重试只收集 Failed（不重试 skipped，skipped = 设备已是目标态）。Drawer 状态机 select → executing → result → retry → result'，支持多次重试直到失败清零。

---

## 审计跳转

| Option | Description | Selected |
|--------|-------------|----------|
| 跳操作日志页 + module=端口管理 filter | navigate('/monitor/logs?module=端口管理')，operlog 页读 URL query 预填 | ✓ |
| 打开 Drawer 展示该模块最近 N 条 | 当前页 antd Drawer，调 operlog list 查 module=端口管理 最近 20 条 | |
| Toast 仅提示成功/失败，不提供链接 | 降级为纯文字提示，用户自行去操作日志页。偏离 UI-04 | |

**User's choice:** 跳操作日志页 + module=端口管理 filter
**Notes:** route 路径是 `/monitor/logs`（非 `/monitor/operlog`），reconciliation/exceptions:343-344 注释已确认此偏差。Path C（52 D-13）使 oper_log_id=NULL，不能精准跳单条 audit，只能跳模块级列表。planner 需确认 operlog 页是否已支持 URL query filter，若无则本 phase 补 module query param 读取。

---

## 权限源

| Option | Description | Selected |
|--------|-------------|----------|
| useMenuStore（代码现状） | MACHistoryPage / devices 页都用，menuPermissions.includes(perm) | ✓ |
| useAuthStore（ROADMAP 字面） | 需先改 store 加 permissions 字段，工作量外溢 | |
| 两者兼容 | menuStore 优先 fallback authStore，代码库无此先例 | |

**User's choice:** useMenuStore（代码现状）
**Notes:** ROADMAP Success Criteria #4 写 useAuthStore 是笔误——authStore 只管 token，不含 permissions。D-09 纠正。

---

## 批量入口

| Option | Description | Selected |
|--------|-------------|----------|
| 列表页勾选 + 顶部按钮触发 | rowSelection 勾选后顶部"批量配置(N)"按钮（与"批量删除"同位置），Drawer 内只读展示已选 | ✓ |
| Drawer 内重新选择端口 | Drawer 自包含 Table 重新勾选，数据源重复 + 分页复杂 | |
| 两者结合 | 列表勾选为主 + Drawer 内可编辑，状态同步复杂 | |

**User's choice:** 列表页勾选 + 顶部按钮触发
**Notes:** 与现有"批量删除(N)"UX 完全一致，最小认知负担。

---

## reason 字段

| Option | Description | Selected |
|--------|-------------|----------|
| 纯文本输入 + 字数计数 | TextArea + 5-200 字符校验，强制手写 | |
| 预置常用原因下拉 + 其他 | Select（故障排查/安全合规/业务变更/临时测试/其他）+ 其他展开 TextArea | ✓ |
| AutoComplete 建议 | 常用原因作 suggestions，多行文本支持一般 | |

**User's choice:** 预置常用原因下拉 + 其他
**Notes:** 减少重复输入，预设项满足 5 字符下限；"其他"覆盖长尾。预设列表放 constants.ts。

---

## description 特例

| Option | Description | Selected |
|--------|-------------|----------|
| description + reason 都必填（独立） | description 多一个必填"新描述"，reason 仍必填 5-200 | |
| description 时 reason 可空 | "新描述"必填，reason 可空（新描述已说明意图） | ✓ |
| 合并为单字段 | 描述+原因混在一个输入框，后端拆分复杂 | |

**User's choice:** description 时 reason 可空
**Notes:** description action 的"新描述"本身说明意图，强制 reason 冗余。其他 4 action reason 必填。后端 52 D-16 PortWriteRequest struct reason 字段本就可空（`json:"reason,omitempty"`）。

---

## Plan 拆分

| Option | Description | Selected |
|--------|-------------|----------|
| 单 plan（53-01 涵盖全部） | networkApi 6 wrapper + PortWriteModal + BulkWriteDrawer + 列表页接入，避免跨 plan 契约漂移 | ✓ |
| 拆 2 plan（API+Modal / Drawer+接入） | API 契约先落地，UI 后接 | |
| 拆 3 plan（API / 组件 / 接入） | 粒度最细，3 plan 对 5-7 任务过重 | |

**User's choice:** 单 plan（53-01 涵盖全部）
**Notes:** 组件间依赖紧密（Modal/Drawer 都调 wrapper），单 plan 避免跨 plan 契约漂移。任务数估计 6-8。

---

## Bundle 策略

| Option | Description | Selected |
|--------|-------------|----------|
| 随端口页打进主 bundle | 组件不重（Modal ~250 + Drawer ~350 行），增量 < 50KB gzip 预算内 | ✓ |
| Drawer lazy load，Modal 随主 bundle | Drawer 仅在点"批量配置"时下载 | |
| 两者都 lazy load | 最大化初始 bundle 节省，但 Modal 高频点击延迟 | |

**User's choice:** 随端口页打进主 bundle
**Notes:** 简单直接，无 React.lazy 边界。组件增量在 ROADMAP 成功标准 #7 的 50KB gzip 预算内。

---

## Claude's Discretion

- PortWriteModal / BulkWriteDrawer 内部样式（Drawer/Modal 宽度、配色、Statistic 卡片布局）
- 预设原因列表具体项（D-02 给 4 项起点，planner 可调）
- description 长度上限（统一 80 字符保守值，不做厂商分支）
- wrapper 返回 TypeScript 类型位置（types/index.ts 或 networkApi.ts inline）
- BulkWriteDrawer 的 action 选择器（Select vs Radio.Group）
- 失败明细 Table 的 commandSent 长文本展示（Typography.Text ellipsis + Tooltip 或可展开行）

## Deferred Ideas

- BATCH-05 批量实时进度反馈（SSE/WebSocket）— v1.19.x（52 D-12 已锁）
- `sys_port_write_audit` 详情查看 UI — v1.19.x+
- operlog → audit 精准反查 — v1.19.x+（依赖 Path C 的 oper_param.audit_ids JSON 反查）
- description 厂商字符上限精细化 — v1.19.x+
- 预设原因可配置化（下沉 sys_dict）— v1.19.x+
- 批量操作历史记录侧栏 — v1.19.x+
