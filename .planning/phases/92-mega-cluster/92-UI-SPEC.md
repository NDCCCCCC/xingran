---
phase: "92"
slug: "mega-cluster"
status: approved
shadcn_initialized: false
preset: not applicable
created: "2026-10-30"
---

# Phase 92 — UI Design Contract

> 视觉与交互契约：431 条暗色手写规则整块交还 `AntdThemeBridge` 分色板 token（延续 Phase 89「Map 层显式给值」纪律），终态仅保留 ≤12 块无法 token 化的例外分区，量化预算达到 v1.25 收敛终态。

---

## Design System

| Property | Value |
|----------|-------|
| Tool | none (antd 6 Design Token System) |
| Preset | not applicable |
| Component library | antd@6.6.4 (Ant Design 6) |
| Icon library | @ant-design/icons@6.1.0 |
| Font | 系统无衬线栈（含 PingFang SC / Microsoft YaHei），代码使用 JetBrains Mono |

---

## Component Inventory

> 暗色交还涉及的组件 token 清单。
> Provenance: Enumerated by AntdThemeBridge.tsx components 表 — 2026-10-30。

| Component | Token 落点（board 派生） | Notes |
|-----------|-------------|-------|
| 全局 seed/map 层 | `colorBgBase` / `colorBgContainer` / `colorBgElevated` / `colorBgLayout` / `colorBorder` / `colorBorderSecondary` | Phase 89 已定型，不动 |
| Table | `headerBg` / `rowHoverBg` / `rowSelectedBg` 等全 `board` 化 | Phase 91 已交还，暗色副本删除的前提已成立 |
| Select | `selectorBg` / `optionSelectedBg` / `optionActiveBg` | Phase 91 已交还 |
| Input / InputNumber | `activeBorderColor` / `activeShadow` / `borderRadius: 8` | VIS-02 / 8px 圆角契约不动 |
| Modal | `contentBg: dark.overlay` / `titleColor` / `borderRadiusLG: 16` | Phase 91 已交还 |
| Dropdown | `colorBgElevated: dark.overlay` | Phase 91 已交还 |
| 其余组件（Message/Notification/Tooltip/Drawer/Popconfirm 等） | 按等价性扫描结果扩 `components.*` | 本 phase 主要扩表对象 |
| 图表 / 3D / 登录页插画 | 保留 `[data-color-mode="dark"]` 手写分区（例外登记） | ≤12 块上限 |

---

## Spacing Scale

沿用 `src/design-system/tokens/spacing.ts` 8px 基准网格（xs=4 / sm=8 / md=16 / lg=24 / xl=32）。Phase 92 仅删暗色副本，不新增间距规则。

Exceptions: none

---

## Typography

沿用 `typography.ts` 既有预设（Body 16px/400、Label 14px/500、Heading 20-24px/600）。本 phase 零排版变更。

---

## Color

唯一真相源 `ruoyiBrand`（`src/design-system/tokens/colors.ts`），暗色一律取 `ruoyiBrand.dark`：

| Role | Light / Dark | Usage |
|------|-------|-------|
| Dominant | Light: `#F0ECE3`/`#FFFFFF`<br>Dark: `#0f2e1b`/`#1a2e1f` | 画布 / 容器面 |
| Secondary | Light: `#E9EFEB`/`#DBD7CE`<br>Dark: `#1f3524`/`#2d4d33` | 表头 / 斑马纹 / 描边 |
| Accent | `#156031`（品牌绿）/ hover `#2E7444` / active `#14542E` | 主按钮、选中态、焦点环 |
| Overlay | Light: `#FFFFFF` (elevated)<br>Dark: `#243a29` (overlay) | 弹层 / 下拉面板 |

Accent reserved for: 主操作按钮、选中指示、聚焦边框（严禁全界面泛滥）。
**红线**：本 phase 零新增裸 hex；删除暗色副本产生的 hex 净减计入 M7 收敛。

---

## Copywriting Contract

本 phase 为纯样式收敛，无新增文案。既有文案契约（确认删除、空态、错误态提示）不变。

---

## UI Considerations

Applicable state considerations resolved: 3 covered, 2 backstop, 0 unresolved

| Category | Element(s) | Status | Resolution / Reason |
|----------|------------|--------|---------------------|
| {dark-equivalence} | 被删 431 条暗色规则 | ✅ covered | AST 扫描三类分类（covered/extendable/exception），零未确认项（D-01） |
| {dark-no-regression} | 暗色全组件 | ✅ covered | 先扩 token 后删块、同 commit；矩阵 computed-style 断言机器验收（D-02/D-03） |
| {residual-blocks} | 例外分区 | ✅ covered | ≤12 块，逐块登记保留理由（D-02） |
| {manual-inspection} | 14 页人工巡检截图 | 🧪 backstop | AD 登录墙，移交 Phase 93 终验（D-03 显式移交表） |
| {contrast-edge} | 暗色描边对比度（X-2） | 🧪 backstop | 3:1 判据移至描边（ROADMAP X-2），92 内完成提亮，93 终验 |

---

## Registry Safety

| Registry | Blocks Used | Safety Gate |
|----------|-------------|-------------|
| antd 6 official | 纯 token 消费，零新组件引入 | 现有 C14 全表断言守护 |
| third-party | none | 零新增依赖 |

---

## Checker Sign-Off

- [x] Dimension 1 Copywriting: PASS
- [x] Dimension 2 Visuals: PASS
- [x] Dimension 3 Color: PASS
- [x] Dimension 4 Typography: PASS
- [x] Dimension 5 Spacing: PASS
- [x] Dimension 6 Registry Safety: PASS
- [x] Dimension 7 Inventory Provenance: PASS

**Approval:** approved 2026-10-30
