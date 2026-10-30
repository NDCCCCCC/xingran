# Phase 91: Select / InputNumber / Form / Table / Modal 域交还 - Technical Research

> 记录 Phase 91 的技术调研、代码现状、样式定位、Token 映射矩阵与执行风险分析。
> 日期：2026-10-30

---

## 一、调研目标与核心问题

1. **手写覆盖现状与定位**：查清 `index.css` 中各组件（Select、Table、Modal、Form、InputNumber、Dropdown、Menu、Tabs、Tag、Alert、Pagination）的手写规则精确行号。
2. **Token 系统现状与缺口**：查清 `AntdThemeBridge.tsx` 当前对这些组件的配置缺口（例如 Table 硬编码浅色 `cream.*`，Modal / InputNumber / Form 尚未配置）。
3. **CSS Module 违规选择器**：准确定位 `styles.module.css` 中 6 条违规 `.ant-*` 全局选择器，并确定迁入 `index.css` 的具体章节。
4. **弃用属性现状（VISFIX-04）**：全仓扫描确认是否存在未清偿的 Select 弃用 prop。
5. **量化指标起点**：确认当前全仓 `!important`（M1）、`index.css`（M2）、顶层 `.ant-` 规则数（M3）与 hex 数（M7/M8）。

---

## 二、调研发现与代码实证

### 1. 手写样式在 `index.css` 的章节分布实证

| 组件域 | `index.css` 起始行号 | 主要手写覆盖内容 | 交还 antd 6 机制 |
|---|---|---|---|
| **Table 域**（承接 R8） | `:980-1065`（浅色）<br>`:2407-2429`（暗色重复） | thead 背景、tbody 斑马纹、行 hover/active 背景（含硬编码 `#f3eee3`）、单元格 padding/font-size | 交还 `AntdThemeBridge` 中的 `components.Table`：`headerBg` / `rowHoverBg` / `rowSelectedBg` / `borderColor`。按 `board` 明暗分色。 |
| **Select 域** | `:1816-1850`<br>`:2310-2335`<br>`:3416-3435`<br>`:3904-3925` | 下拉菜单背景（`.ant-select-dropdown`）、选项选中态（`.ant-select-item-option-selected`）、选项 hover/active 态 | 依靠 Phase 89 定型的 `--theme-bg-overlay` 与 antd CSS 变量，删除死选择器与重复暗色覆盖。 |
| **Modal 域** | `:1136-1185`（浅色章节）<br>`:2420-2435`<br>`:3974-3990` | 弹窗容器（`.ant-modal-content`）、头部（`.ant-modal-header`）、标题、关闭按钮 | 在 `AntdThemeBridge` 增加 `components.Modal`：`contentBg` / `headerBg` / `titleColor` / `borderRadiusLG: 16`。 |
| **InputNumber 域** | `:948-978` | status-error 与 status-warning 边框覆盖 | Phase 90 已将其统一为 8px 圆角与 CSS 变量，与 Input 域保持一致。 |
| **Form 域** | `:1070-1130` | 表单 label 字体、margin、explain 错误提示 | 交还 antd 原生 Form token（`labelColor`、`itemMarginBottom: 16`）。 |
| **Dropdown / Menu 域** | `:1816-1855`, `:463-489` | 下拉菜单背景、边框、菜单项选中 | 交还 `Dropdown` / `Menu` token。 |
| **Tabs / Tag / Alert** | `:850-870`, `:1190-1230` | 选项卡指示条、标签圆角、警告条背景 | 保持品牌绿主色（`colorPrimary`），清除手写 `!important`。 |

### 2. `AntdThemeBridge.tsx` 的配置现状与重构方案

- **现状分析**：
  在 `src/design-system/components/AntdThemeBridge.tsx:107-118` 中，目前 `Table` 组件被写死了浅色常量：
  ```tsx
  Table: {
    headerBg: ruoyiBrand.cream.headerBg,
    headerColor: ruoyiBrand.cream.fg,
    borderColor: ruoyiBrand.cream.border,
    rowHoverBg: ruoyiBrand.cream.zebraBg,
    ...
  }
  ```
  在暗色模式下（`isDark === true`），Table 的表头与悬浮态仍读取 `cream`，导致暗色下无法正常适配。
  同时，`Modal`、`Dropdown`、`InputNumber` 等在 `components` 字典中尚未配置。

- **重构方案**：
  将 `Table` 及新增的 `Modal`、`Dropdown`、`Form` 组件配置统一改为从 `board`（`const board = isDark ? ruoyiBrand.dark : ruoyiBrand.cream;`）读取：
  ```tsx
  Table: {
    headerBg: board.headerBg,
    headerColor: board.fg,
    borderColor: board.border,
    rowHoverBg: isDark ? board.hover : board.zebraBg,
    rowSelectedBg: board.headerBg,
    rowSelectedHoverBg: isDark ? board.active : board.zebraBg,
    borderRadius: 8,
  },
  Modal: {
    contentBg: board.overlay,
    headerBg: "transparent",
    titleColor: board.fg,
    borderRadiusLG: 16,
  },
  Dropdown: {
    colorBgElevated: board.overlay,
  },
  Form: {
    labelColor: board.fg,
    itemMarginBottom: 16,
  },
  ```

### 3. CSS Module 全局选择器违规定位与迁出方案

- **精确位置**：
  `ruoyi-react-frontend/src/pages/operations/building-spaces/components/styles.module.css`：
  - 行 114: `.buildingModal :global(.ant-modal-wrap)`
  - 行 118: `.buildingModal :global(.ant-modal-content)`
  - 行 123: `.buildingModal :global(.ant-modal-header)`
  - 行 129: `.buildingModal :global(.ant-modal-title)`
  - 行 135: `.buildingModal :global(.ant-modal-close)`
  - 行 140: `.buildingModal :global(.ant-modal-close:hover)`
- **迁出方案**：
  这 6 条针对特定建筑空间弹窗定制的样式直接迁入 `src/index.css` 的 Modal 模态框章节（`:1136` 附近），使用标准全局选择器 `.building-space-modal .ant-modal-*`；而在 `styles.module.css` 中完全删除这 6 处违规选择器，使 `module.css` 不再包含裸露的 `.ant-`，满足 C10 契约。

### 4. Select 弃用属性扫描结果（VISFIX-04）

- 对全站 `src/pages/**` 进行 AST 与正则扫描：
  - `dropdownRender` / `dropdownStyle` / `dropdownClassName` / `dropdownMatchSelectWidth` / `onDropdownVisibleChange` 当前命中均为 **0**。
  - 在 Phase 88（C14/C14f 数据驱动全表推进）中，业务代码中的老弃用属性已经完成全面清理。
- 为巩固成果，在 `src/components/ui/AppSelect.tsx` 中保留兼容映射（作为契约守护），同时确保 `npm run lint:ci` 下的 C14/C14f 检查持续通过。

### 5. 当前量化基线（M1 ~ M16 起点）

- **M1（全仓 `!important`）**：当前为 **1391**（预算限制为 ≤ 1397）。
- **M2（`index.css` 的 `!important`）**：当前为 **1315**。
- **M3（顶层含 `.ant-` 规则数）**：当前为 **712**。
- **M7（`index.css` 的 6 位 hex）**：当前为 **304** ≤ 310。
- **M8（全 `.css` 的 6 位 hex）**：当前为 **440** ≤ 452。
- **M9（`index.css` 行数）**：当前为 **5552** ≤ 5558。

---

## 三、三波递进执行规划

按照 `91-CONTEXT.md` 的 D-01 决策，Phase 91 划分为 3 个波次执行：

```
Phase 91: Select / InputNumber / Form / Table / Modal 域交还
 ├── Plan 91-01 (Wave 1: 表单输入域): Select + InputNumber + Form 交还 token，清理手写边框与死选择器
 ├── Plan 91-02 (Wave 2: 数据展示与容器域): Table (R8) + Pagination + Tabs + Tag 样式交还 board token，消除硬编码 hex
 └── Plan 91-03 (Wave 3: 浮层与导航域 + 契约收口): Modal + Dropdown 样式交还，styles.module.css 全局选择器迁出，量化预算下调与回归验收
```

---

## 四、技术风险与防线

1. **表格斑马纹与悬浮色差风险**：
   - 风险：删除 `index.css:1026-1049` 手写斑马纹后，若 antd 默认表格背景与页面底色相同，可能导致斑马纹消失。
   - 防线：在 `AntdThemeBridge` 中将 `rowHoverBg` 与 `zebraBg` 精确绑定到 `board.zebraBg`（浅色 `#F7F5EE`，暗色 `#1a2e1f`），确保双层纸感与可读性完整保留。
2. **C11 / C13 契约预算不增反降防线**：
   - 任何改动只能减少或保持 `!important` 与 hex 数，绝不允许新增任何 `!important` 或硬编码十六进制色。
3. **C10 契约防线**：
   - `styles.module.css` 迁出后，立即运行 `scripts/css-contract-check.js` 验证全仓 `.module.css` 零全局污染。
