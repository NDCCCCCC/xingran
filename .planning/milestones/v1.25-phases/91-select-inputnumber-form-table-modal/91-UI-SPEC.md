---
phase: "91"
slug: "select-inputnumber-form-table-modal"
status: approved
shadcn_initialized: false
preset: not applicable
created: "2026-10-30"
---

# Phase 91 — UI Design Contract

> 视觉与交互契约：Select / InputNumber / Form / Table / Modal 等组件域的手写覆盖规则交还 antd 6 官方 `variant` / `components.*` token，全站清除 Select 弃用属性，规范化 CSS Module 全局选择器。

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

> 项目设计系统核心组件提供清单。
> Provenance: Enumerated by package.json & AntdThemeBridge.tsx — 11 components — antd@6.6.4 — 2026-10-30.

| Component | Import path | Notes |
|-----------|-------------|-------|
| `Select` / `AppSelect` | `@/components/ui/AppSelect` 或 `antd` | 下拉选择器；需清零 `dropdownStyle` / `dropdownMatchSelectWidth` 等弃用 prop |
| `InputNumber` | `antd` | 数字输入框；需与 Input 域统一 8px 圆角及错误态边框色 |
| `Form` | `antd` | 表单容器；需交还 label、item margin 等原生 token |
| `Table` | `antd` | 数据表格（承接 R8 核心）；表头/斑马纹/选中态需按 `board` 语义化映射 |
| `Modal` | `antd` | 对话框；浮层背景、header、标题交还 `Modal` token |
| `Dropdown` | `antd` | 下拉菜单浮层；交还 `Dropdown` token 与 `popup` 结构 |
| `Menu` | `antd` | 导航与操作菜单；保持深色侧边栏与浅色内容区分离 |
| `Tabs` | `antd` | 标签页；保持品牌绿选中态（`itemSelectedColor`） |
| `Tag` | `antd` | 状态标签；4px 圆角与状态语义色 |
| `Alert` | `antd` | 提示警告横条；消除弃用 `message` prop 并交还 token |
| `Pagination` | `antd` | 分页器；统一 8px 圆角与品牌绿激活高亮 |

---

## Spacing Scale

依据 `src/design-system/tokens/spacing.ts`（基于 8px 基准网格）：

| Token | Value | Usage |
|-------|-------|-------|
| xs | 4px | 图标微距、inline 细微填充、Tag 内边距 |
| sm | 8px | 基础网格步长、紧凑元素间距、按钮小边距 |
| md | 16px | 默认元素间距、表单项 marginBottom、卡片内边距 |
| lg | 24px | 弹窗内边距、页面区域分块间距 |
| xl | 32px | 大布局分区网格留白 |
| 2xl | 48px | 结构性断点间距 |
| 3xl | 64px | 顶层页面级留白 |

Exceptions: none

---

## Typography

依据 `src/design-system/tokens/typography.ts`：

| Role | Size | Weight | Line Height |
|------|------|--------|-------------|
| Body | 16px (base) / 14px (sm) | 400 (normal) | 1.5 (normal) |
| Label | 14px (sm) | 500 (medium) | 1.5 (normal) |
| Heading | 20px (xl) ~ 24px (2xl) | 600 (semibold) | 1.25 (tight) ~ 1.375 (snug) |
| Display | 30px (3xl) ~ 36px (4xl) | 700 (bold) | 1.25 (tight) |

---

## Color

依据 `src/design-system/tokens/colors.ts`（`ruoyiBrand` 单一真相源，Phase 89 明暗双板）：

| Role | Value (Light / Dark) | Usage |
|------|-----------------------|-------|
| Dominant (60%) | Light: `#F0ECE3` (canvas) / `#FFFFFF` (surface)<br>Dark: `#0f2e1b` (canvas) / `#1a2e1f` (surface) | 画布底、卡片底、表格容器底 |
| Secondary (30%) | Light: `#E9EFEB` (headerBg) / `#DBD7CE` (border)<br>Dark: `#1f3524` (headerBg) / `#2d4d33` (border) | 表头背景、斑马纹交互底、边框线、分割线 |
| Accent (10%) | `#156031` (greenPrimary) / hover `#2E7444` / active `#14542E` | 品牌主按钮、激活选中项高亮、Tabs 游标、焦点环边框 |
| Destructive | `#BA3630` (functional.danger) | 删除/危险操作按钮、错误态边框（`ant-input-status-error`） |

Accent reserved for: 主操作按钮、选中的 Tab 下划线、高亮行激活指示条、聚焦状态边框（严格杜绝全界面泛滥）。

---

## Copywriting Contract

| Element | Copy |
|---------|------|
| Primary CTA | 确 定 / 查 询 / 保 存 |
| Empty state heading | 暂无数据 |
| Empty state body | 当前筛选条件下未检索到匹配记录，请调整搜索条件或重新加载 |
| Error state | 操作失败：网络连接异常或服务内部错误，请稍后重试 |
| Destructive confirmation | 确认删除：确定要删除选中的数据项吗？删除后将无法恢复 |

---

## UI Considerations

> Populated for Phase 91 component domain migration and deprecation clearance.

Applicable state considerations resolved: 5 covered, 2 backstop, 0 unresolved

| Category | Element(s) | Status | Resolution / Reason |
|----------|------------|--------|---------------------|
| {border-and-bg} | Select 容器与下拉面板 | ✅ covered | 手写边框交还 antd CSS 变量，彻底移除 `.ant-select-selector` 死选择器，暗色下与全局 surface/overlay 严格同源 |
| {table-striping} | Table 表头与斑马纹 | ✅ covered | Table `headerBg` / `rowHoverBg` / `rowSelectedBg` 绑定 `board` 明暗色板，暗色下彻底消除浅色白块 |
| {modal-surfaces} | Modal 弹窗及标题栏 | ✅ covered | Modal 的 `contentBg` 与 `headerBg` 完全交还 `Modal` token，关闭图标采用继承主文本色 |
| {deprecations} | Select / TreeSelect 弃用属性 | ✅ covered | `dropdownStyle` 改为 `styles.popup.root`，`dropdownMatchSelectWidth` 改为 `popupMatchSelectWidth`，`onDropdownVisibleChange` 改为 `onOpenChange`，`showArrow` 默认值清除 |
| {css-module-clean} | `styles.module.css` 全局选择器 | ✅ covered | 6 条违规 `.ant-*` 选择器就近迁入 `src/index.css` 对应章节，C10 契约零破坏 |
| {input-number-status} | InputNumber 状态与圆角 | 🧪 backstop | 保证 InputNumber 错误态边框色（`rgb(186, 54, 48)`）与统一的 8px 圆角不回归 |
| {visual-zero-diff} | 21 形态 × 明暗双向矩阵 | 🧪 backstop | 实施前后运行 `matrix-diff-90.cjs` 验证样式回归，差异严格控制在已登记的优化项内 |

---

## Registry Safety

| Registry | Blocks Used | Safety Gate |
|----------|-------------|-------------|
| antd 6 official | Select, Table, Modal, Form, InputNumber, Tabs, Tag, Pagination | Verified via `@ant-design/icons` and official antd 6.6.4 definitions |
| third-party | none | 零新增第三方 UI 依赖 |

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
