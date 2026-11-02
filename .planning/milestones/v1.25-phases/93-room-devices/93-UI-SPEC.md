---
phase: "93"
slug: "room-devices"
status: approved
shadcn_initialized: false
preset: not applicable
created: "2026-10-30"
---

# Phase 93 — UI Design Contract

> 视觉与交互契约：`room-devices` 页面大单体按 Phase 83 范式解耦拆分（零渲染与零行为变化），全系统 STYLE-05 明暗双主题视觉终验（21 形态 computed-style 机器断言 + 14 页巡检报告）。

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

> `room-devices` 涉及的核心 UI 组件与展示项：
> Provenance: Enumerated by src/pages/operations/room-devices/index.tsx — 2026-10-30.

| Component | Usage in room-devices | Contract Constraint |
|-----------|------------------------|---------------------|
| `Table` | 设备列表核心数据展示 | 列定义抽离为 `columns.tsx`，保持排序/筛选/操作列三态门控零变化 |
| `Form` / `Input` / `Select` | 顶部搜索栏与筛选区 | 搜索字段抽离为 `searchFields.ts`，8px 圆角与透明底色契约 |
| `Button` / `ActionButton` | 工具栏操作（新增、批量删除、导出、导入）与行操作 | 严格绑定三态门控权限码，按钮间距与样式保持原样 |
| `Modal` / `Upload` | 设备新增/编辑对话框与 Excel 导入弹窗 | 抽离为受控弹窗状态，由 `useRoomDeviceActions` 管理 |
| `Tag` / `Badge` | 设备状态与端口绑定指示器 | 保持语义状态色与圆角 |

---

## Spacing Scale

依据 `src/design-system/tokens/spacing.ts`（基于 8px 基准网格）：

| Token | Value | Usage |
|-------|-------|-------|
| xs | 4px | 行内按钮微距、Tag 内边距 |
| sm | 8px | 搜索区表单间距、紧凑网格间隙 |
| md | 16px | 表格与筛选区分块间距、Modal 内边距 |
| lg | 24px | 页面级留白与大卡片内边距 |

Exceptions: none

---

## Typography

依据 `src/design-system/tokens/typography.ts`：

| Role | Size | Weight | Line Height |
|------|------|--------|-------------|
| Body | 14px (sm) / 16px (base) | 400 (normal) | 1.5 (normal) |
| Table Header | 13px ~ 14px | 600 (semibold) ~ 700 | 1.4 |
| Modal Title | 16px (base) ~ 20px (xl) | 600 (semibold) | 1.3 |

---

## Color

依据 `src/design-system/tokens/colors.ts`（`ruoyiBrand` 单一真相源）：

| Role | Value (Light / Dark) | Usage |
|------|-----------------------|-------|
| Dominant | Light: `#F0ECE3` / `#FFFFFF`<br>Dark: `#0f2e1b` / `#1a2e1f` | 画布底、卡片底、表格背景 |
| Secondary | Light: `#E9EFEB` / `#DBD7CE`<br>Dark: `#1f3524` / `#2d4d33` | 表头底色、分割线、边框线 |
| Accent | `#156031` (greenPrimary) / hover `#2E7444` | 主操作按钮、聚焦边框、激活状态 |
| Status Error | `#BA3630` (functional.danger) | 错误态输入框边框（`rgb(186, 54, 48)`）与危险删除操作 |

---

## Copywriting Contract

| Element | Copy |
|---------|------|
| Primary CTA | 确 定 / 搜 索 / 导 出 |
| Destructive confirmation | 确定要删除选中的机房设备吗？删除后关联的端口绑定将解除 |
| Empty state body | 暂无机房设备数据 |
| Batch actions | 批量删除 / 批量导出 |

---

## UI Considerations

> Populated for Phase 93 refactoring and final verification.

Applicable state considerations resolved: 4 covered, 1 backstop, 0 unresolved

| Category | Element(s) | Status | Resolution / Reason |
|----------|------------|--------|---------------------|
| {code-splitting} | `room-devices/index.tsx` | ✅ covered | 解耦为 `columns.tsx`、`searchFields.ts`、`hooks/useRoomDeviceActions.ts`、主文件 `index.tsx` ≤ 350 行，子文件 ≤ 250 行，零行为变化 |
| {reloadRef-injection} | 表格刷新协议 | ✅ covered | `reloadRef` 外部注入协议 4 段链路逐字保留，父组件调用零破坏 |
| {useMemo-deps} | 列定义缓存策略 | ✅ covered | `columns` 的 `useMemo` 依赖（故意不完整依赖）保持原状，绝不「顺手修正」引发无限渲染 |
| {style-05-hard-metrics} | 全系统明暗核心指标 | ✅ covered | computed-style 断言暗色 `--input-bg = #0f2e1b`、InputNumber 错误态边框 = `rgb(186, 54, 48)`、输入类 8px 圆角全达标 |
| {inspection-report} | 14 页业务界面终验 | 🧪 backstop | 承接 Phase 92 移交清单，生成 `93-VISUAL-INSPECTION-REPORT.md` 核对报告 |

---

## Registry Safety

| Registry | Blocks Used | Safety Gate |
|----------|-------------|-------------|
| antd 6 official | Table, Form, Input, Button, Modal, Upload, Tag | Verified under antd 6.6.4 |
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
