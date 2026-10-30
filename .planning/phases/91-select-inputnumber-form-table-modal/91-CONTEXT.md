# Phase 91: Select / InputNumber / Form / Table / Modal 域交还 - Context

**Gathered:** 2026-10-30
**Status:** Ready for planning

<domain>
## Phase Boundary

本 Phase 承接样式收敛线 B 中 STYLE-02 规则收敛第 4 批中的其余组件域（R8 以及 Select、InputNumber、Form、Table、Modal、Dropdown、Menu、Tabs、Tag、Alert、Pagination）的手写覆盖清理，并将全站样式交还 antd 官方 `variant` 与 `components.*` theme token。同时彻底清零 VISFIX-04 遗留的 antd 6 弃用属性（Select 域六类弃用 prop），并将 `styles.module.css` 中的全局选择器违规迁出，达成全站无新增弃用警告且量化规则指标大幅收敛（M1 ≤ 900, M3 ≤ 560）。

**范围约束**：
- 本 Phase 仅处理除 Input 域之外的表单、数据展示与浮层导航组件的手写覆盖规则交还与弃用 prop 迁移。
- 暗色 mega-cluster（R6/R7 的 431 条规则）整体删除归属于后续的 **Phase 92**，本 Phase 不做块级暗色删除。
- 巨型组件文件拆分（`room-devices/index.tsx`）归属于 **Phase 93**，本 Phase 不越界进行业务重构。
</domain>

<decisions>
## Implementation Decisions

### 1. 组件域交还的批次切分与执行节奏
- **D-01:** 采用**三波递进推进**策略，将 10 余个组件域按功能族高内聚划分，分步实施与逐波验证：
  - **Wave 1（表单输入域）**：`Select` + `InputNumber` + `Form`（清理 `.ant-select-selector` 死选择器、消除手写边框，交还 antd `--ant-select-*` 变量与 Form token）。
  - **Wave 2（数据展示与容器域）**：`Table`（核心承接 R8）+ `Pagination` + `Tabs` + `Tag` + `Alert`（Table thead/tbody/container 样式交还 `Table.headerBg` / `rowHoverBg` 等 token）。
  - **Wave 3（浮层与导航域 + 弃用清理收口）**：`Modal` + `Dropdown` + `Menu` + `styles.module.css` 违规全局选择器迁出 + VISFIX-04 剩余弃用 prop（`dropdownStyle`、`dropdownMatchSelectWidth` 等）全仓清零。
  — **Reversibility:** costly — 涉及多个核心组件和全局样式的渐进交还，波次切分保证每波独立可测试与可回滚。

### 2. Table / Modal 等组件 token 的视觉等价性策略
- **D-02:** 以 **`ruoyiBrand` 语义化主题色板为唯一真相源**（`src/design-system/tokens/colors.ts` 中已于 Phase 89 定型的明暗色常量），在 `src/design-system/components/AntdThemeBridge.tsx` 中按 `themeMode` 分支完整配置对应组件的 `components.*` token。
- **D-03:** 彻底淘汰历史手写 CSS 中遗留的孤立脏 hex，允许符合设计系统的微调优化，确保组件背景、边框、悬浮与激活态与全局品牌中性色/语义色自然咬合。
  — **Reversibility:** costly — 影响全站表格与弹窗的明暗视觉呈现，统一通过 AntdThemeBridge 维护保证风格一致。

### 3. Select 弃用属性的批量治理方式
- **D-04:** 采取**双管齐下**策略：
  1. **组件层兼容透传**：在公用组件 `src/components/ui/AppSelect.tsx` 封装层提供向后兼容映射（自动将旧的 `dropdownStyle` / `dropdownMatchSelectWidth` / `onDropdownVisibleChange` 等转换为 antd 6 新标准属性），防止个别遗漏引发运行时警告。
  2. **业务页面精准替换**：对全站直接引用原生 antd `<Select>` / `<TreeSelect>` 的业务页面代码，通过精准查找批量更新为 antd 6 官方标准写法（如 `popupRender` / `styles.popup.root` / `classNames.popup.root` / `onOpenChange` / `popupMatchSelectWidth` 等），并清理无意义的 `showArrow` 默认值。
  3. **静态门禁验证**：由 `scripts/css-contract-check.js` 中的 C14 / C14f 静态断言进行阻断校验，确保浏览器控制台 `Warning:` 条数降至 0。
  — **Reversibility:** costly — 触碰数十个业务页面，但有静态断言保护，杜绝控制台噪音。

### 4. CSS Module 违规选择器的迁出归宿
- **D-05:** `styles.module.css` 中的 6 条违规 `.ant-*` 全局选择器**就近迁入 `src/index.css` 对应组件章节**（如 Form 或 Modal 节）。
- **D-06:** 保持全局样式的单一入口与章节集中管理，严守「不新增额外 CSS 文件」的架构规范，同时满足 C10（`module.css 不含 .ant-`）契约断言。
  — **Reversibility:** reversible — 选择器行迁移，属于纯 CSS 组织归集。

### Claude's Discretion
- 具体各波次中的文件修改批次划分由规划器（planner）精细编排。
- 在消除 `.ant-select-selector` 等死选择器时，若发现相邻存在完全无效的历史废弃属性，可在确保无害性前提下顺手清理。
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 需求与路线图基线
- `.planning/REQUIREMENTS.md` — STYLE-02、STYLE-04、VISFIX-04 详细条款与量化口径
- `.planning/ROADMAP.md` §Phase 91 — 成功标准五条（Select交还、各域token交还、CSS Module迁出、弃用清零、量化达标）
- `.planning/research/PITFALLS.md` §四批规则清单 & §3.1 量化基线 — R8 规则详情、M1-M16 指标与收敛顺序

### 先验 Phase 关键成果与架构资产
- `.planning/phases/89-theme-refactor/89-QUANT-SNAPSHOT.md` — Phase 89 定型的 `ruoyiBrand.dark` 色板与 `AntdThemeBridge` 派生机制
- `.planning/phases/90-input-variant/90-CHANGESET.md` — Phase 90 Input 域长属性拆分与 VIS-01 复位块重构先例
- `src/design-system/components/AntdThemeBridge.tsx` — antd 6 主题桥接配置（`components.*` token 定义核心入口）
- `scripts/css-contract-check.js` — 静态契约断言（C7、C10、C12、C14、C14f 等硬门禁）
</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `src/design-system/tokens/colors.ts`: `ruoyiBrand.light` 与 `ruoyiBrand.dark` 语义色常量集，作为 `components.*` token 派生源。
- `src/components/ui/AppSelect.tsx`: 项目已封装的公共 Select 契约组件，可作为属性兼容映射和变体统一控制的核心切入点。
- `src/design-system/components/AntdThemeBridge.tsx`: 全局 antd ConfigProvider 主题桥接层，支持按 `themeMode`（light / dark）分支配置各组件 token。

### Established Patterns
- **低特异性优先**：antd 6 生成的 `:where(.css-xxx)` 特异性为 0，交还官方 token 即可自然生效，无需手写 `!important`。
- **静态断言双保险**：任何手写 CSS 清理和属性弃用均通过 `scripts/css-contract-check.js` 自动化回归阻断。

### Integration Points
- `src/index.css`: 清理手写 Table、Select、Form、Modal 等覆盖样式，吸收 `styles.module.css` 迁出的全局规则。
- `src/design-system/components/AntdThemeBridge.tsx`: 集中扩充并纠偏 `components.Table`、`components.Select`、`components.Modal` 等组件的明暗 token。
- 业务页面 TSX：逐个清理 Select 弃用 prop，使浏览器 Console 保持 0 Warning。
</code_context>

<specifics>
## Specific Ideas
- 严格遵循 ruoyiBrand 语义化主题色板，不复刻手写脏 hex，实现视觉与代码的双重整洁。
- 保证每个波次完成后立即通过 `npm run lint:ci` 与静态契约断言，确保全流程处于绿灯保护下。
</specifics>

<deferred>
## Deferred Ideas
- **Phase 92**: 暗色 mega-cluster（R6/R7 共 431 条规则）的大块清理与 darkAlgorithm 终态收口。
- **Phase 93**: `src/pages/operations/room-devices/index.tsx` 815 行巨型文件的解耦拆分与明暗双主题截图矩阵终验。
</deferred>

---

*Phase: 91-select-inputnumber-form-table-modal*
*Context gathered: 2026-10-30*
