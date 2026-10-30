---
phase: 88
title: 验证底座与零风险清理 — 技术方案研究
requirements: VISFIX-03, VISFIX-04, VISFIX-06 + STYLE-02 第 1 批（D1-D17）
researched: 2026-10-28
status: research-complete
sources: ROADMAP.md / REQUIREMENTS.md / research/PITFALLS.md / research/STACK.md / 实跑核查（本文件 §现状复核）
---

# Phase 88 研究：验证底座与零风险清理

> **只读研究**：未修改任何业务代码，未 git commit。所有数字来自本机实跑（Windows，
> `ruoyi-react-frontend/` 为前端根；行号以当前 HEAD `src/index.css` = **5632 行**为准）。
> PITFALLS 的行号清单经本次逐条 sed 复核，**全部仍然准确**（仅死文件行数有 ±17 的统计口径差，见 §1.3）。

---

## 现状复核

### 1.1 D1-D17 行号逐条 sed 复核结果

| 条目 | PITFALLS 行号 | 当前 HEAD 实测 | 判定 |
|---|---|---|---|
| D1 | `2093-2096` | `2093-2096`（注释「部门树搜索框样式」在 `:2092`） | ✅ 准确 |
| D2 | `2098-2102` | `2098-2102` | ✅ |
| D3 | `2104-2109` | `2104-2109` | ✅ |
| D4 | `2111-2116` | `2111-2116` | ✅ |
| **K8（活，勿删）** | `2119-2123` | `2119-2123`（注释「用户表单输入框样式」在 `:2118`） | ✅ |
| **K9（活，勿删）** | `2125-2130` | `2125-2130` | ✅ |
| D5 | `3523/3527/3531/3535/3539/3543` | 同（6 条 Anchor 规则，3523-3545 连续区段） | ✅ |
| D6 | `3602-3604, 3606-3608` | 同（Popconfirm 2 条） | ✅ |
| D7 | `873`（组内行） | `871-880` 组的第三选择器行 `873: .ant-select-selector {` | ✅ |
| D8 | `884` | `882-886` 组第三行 `884: .ant-select:hover .ant-select-selector {` | ✅ |
| D9 | `890` | `888-893` 组第三行 | ✅ |
| D10 | `897` | `895-901` 组第三行 | ✅ |
| D11 | `2329-2333` | 同 | ✅ |
| D12 | `3962-3966` | 同 | ✅ |
| D13 | `3968-3970` | 同 | ✅ |
| D14 | `3972-3977` | 同 | ✅ |
| D15 | `5579-5581` | 同（无 `!important`，纯装饰） | ✅ |
| `.ant-select-selector` 总出现 | 9 处 | **9 处**（`grep -c` 全 matches 口径） | ✅ |

**⚠️ D7-D10 的删除形态（关键）**：这四条的 `.ant-select-selector` 是**多行选择器组的第三行**，
同组还有 `.ant-input` / `.ant-input-number`（**活规则**，Phase 90 才交还 token）。删除的是**一个分支行**，
不是整组：

```css
/* 871-880 删 D7 后的形态（872 行尾逗号必须去掉）： */
.ant-input,
.ant-input-number {          /* ← 原 873 行 .ant-select-selector 删除 */
  background: var(--input-bg) !important;
  ...
}
```

### 1.2 K8/K9 与 D1-D4 的精确删除边界

D1-D4 家族 = `index.css:2092-2117`（`:2092` 注释 + 4 条规则 + `:2117` 空行）；K8 家族起于 `:2118` 注释。

```bash
# 精确删除（删除后 K8 注释上移到原 2092 位置，活规则零接触）
sed -i '2092,2117d' src/index.css
# 验证：删除后立即核对 K8/K9 仍在
grep -n 'user-form-input' src/index.css   # 应仍有 :2118 注释 + 2119/2125 两条规则（行号前移 26）
grep -c 'dept-tree-search' src/index.css  # 应为 0（TSX 的 1 处不受影响，该 grep 只扫 css）
```

> 多条删除操作需**从大行号往小行号依次执行**（5579 → 3962 → 3602 → 3523 → 2329 → 873 → 2092-2117），
> 否则前面的删除会使后面 PITFALLS 行号漂移；每删一批后重跑 `grep -n` 确认下一批行号。

### 1.3 D16/D17 死文件 import 复核（9 个文件逐一实跑）

精确口径：`grep <basename>.css / <basename>.less`（带后缀）在 `src/**/*.{ts,tsx,css}` 中查 import 引用。

| 文件 | 实测行数¹ | import 引用 | 判定 |
|---|---|---|---|
| `src/design-system/themes/theme-styles.css` | 96 | 0（仅 `index.css:39/4721-4722` **注释**提到「merged from theme-styles.css」） | ✅ 死，可删 |
| `src/pages/dashboard-system/index.css` | 129 | 0 —— **重点核查过**：`routeGenerator.ts:129` 返回 `"pages/dashboard-system/index"` 字符串，但 `componentLoader.tsx:34` 的 `import.meta.glob("/src/pages/**/{index,detail}.tsx")` 只加载 **tsx**；`dashboard-system/index.tsx` 的 import 清单里**无** `./index.css` | ✅ 死，可删 |
| `.../widgets/types/ChartWidget.css` | 23 | 0（`widgetRegistry.ts` import 的是 `../types/ChartWidget` **tsx**；tsx 内亦无 css import） | ✅ 死 |
| `.../types/ListWidget.css` | 77 | 0 | ✅ 死 |
| `.../types/MetricWidget.css` | 24 | 0 | ✅ 死 |
| `.../types/ProgressWidget.css` | 53 | 0 | ✅ 死 |
| `.../types/StatCardWidget.css` | 67 | 0 | ✅ 死 |
| `src/components/cad-editor/CADFloorPlanEditor.less` | 84 | 0（`CADFloorPlanEditor.tsx` 诚，全仓无 `.less` 引用） | ✅ 死 |
| `src/components/shared/FloorPlanEditor.less` | **0 字节** | **1 处**：`FloorPlanEditor.tsx:18` `import "./FloorPlanEditor.less";` | 删文件 + **同 commit 删该 import 行** |

¹ `Get-Content` 行数口径（PITFALLS 用 `wc -l`，`theme-styles` 97→96 等差异为末行换行符口径，不影响结论）。合计 ≈ **553 行** ≈ 记载的 555 行。
**⚠️ 甄别警示**：`dashboard-system/edit.css` 与 `view.css` 是**活的**（`edit.tsx:21` / `view.tsx:20` import），**不在删除清单**。

### 1.4 VISFIX-06 契约脚本现状（已读全文）

- `scripts/css-contract-check.js`（89 行）：纯文本正则，C1a-C1c/C2/C3a-C3c/C4a-C4c/C5a-C5c。
  `:57` 的 **C3b = 正向断言** `.dept-tree-search .ant-input-affix-wrapper .ant-input` 存在 → 正在锁死死规则（P-C5 坐实）。
  `:77` 已有 `indexCssNoComment`（剥注释）先例，可复用于反转断言。
- `scripts/css-contract-assertions.js`（74 行）：浏览器 Console 粘贴片段。
  `:48` fixture 用 `ant-input-affix-wrapper-status-error`（antd 6.6.4 不存在）；`:69` `passed` 判定
  `/255,\s*77,\s*79/`（antd 默认红）→ 与项目品牌红 `rgb(186,54,48)`（`--ant-color-error` = `#BA3630`）不符 →
  即使 class 修对，`passed` 仍会 false。**两处都要改**。
- `css-contract-check.js` **未接入任何 npm script / CI**（`package.json` 的 `lint:ci` =
  `eslint + assert-lint-budget + check-exemption-reasons`，无 css-contract）→ 新断言必须显式接入门禁才有效。

### 1.5 弃用 prop 逐类 grep 实测（对照 STACK 46/13/17/4/6）

| 类 | STACK/PITFALLS 记载 | 本次实测（排除 `__tests__`/`*.test.*`） | 全口径（含测试） | 甄别结论 |
|---|---|---|---|---|
| `maskClosable` | 46 | **34 处 / 22 文件** | **46**（测试 12 处） | ✅ 全部需迁；**含 Drawer**：`LocationAliasDrawer.tsx:224` 裸 `maskClosable`（**Drawer.d.ts:27 同样标 `@deprecated → mask.closable`**） |
| `Card bordered` | ≈13 | **6 处**（`MACEventsTimeline.tsx`×4 + `MACHistoryPage.tsx`×2，全部 `bordered={false}`） | — | ⚠️ **实测 6 ≠ 记载 ≈13**：其余 bordered 命中是 Descriptions ≈16（**未弃用，不动**）、Table 7 处（`WorkstationDeviceTable/index.tsx`，**未弃用，不动**）、业务配置字段 2 处（`DisplayConfigForm.tsx:60` / `WidgetSelector.tsx:124` / `TableWidget.tsx:110` 为 `TableDisplayConfig.bordered` 对象字段，**非 antd prop，不动**） |
| `Descriptions bordered` / `Table bordered` | — | ≈16 + 7 | — | **未弃用**（STACK §B.4/B.6 + `antd/es/card` 源码仅 Card 弃用）→ 计划显式写防误伤 |
| `size="middle"` | 17 | **17 处 / 16 文件** | — | ✅ 全迁 `medium`（Table/Divider/Descriptions 枚举统一） |
| `size="default"` | 1 | **1 处** | — | `Card.js:76` 源码证实 Card 的 `default→medium`；按组件语义迁 |
| `bodyStyle` | 4 | **4 处**（`ad-domain/ous/index_with_dept.tsx`×2 + `workstations/views/FloorPlanView.tsx`×2） | — | ⚠️ **全部是 Card 的 bodyStyle**（STACK 记「Modal prop」口径需更正；`Card.js:93` 源码证实 Card bodyStyle→`styles.body` 弃用） |
| `destroyOnClose` | 6 | **6 处**（Modal×5 + `FixSuggestionDetailDrawer.tsx:85` **Drawer×1**） | — | ✅ 全迁 `destroyOnHidden`（Modal.js:126 + Drawer.d.ts:21 双弃用） |
| Select 六类（dropdownRender/dropdownClassName/popupClassName/onDropdownVisibleChange/dropdownMatchSelectWidth/showArrow） | 0 | **0** | — | ✅ 无需处理（91 的验收项天然满足） |
| `onSearch` | 271（不迁） | 271 | — | **明确不迁**（antd 6.6.4 未弃用，关键决策表锁定） |

**`maskClosable` 形态分布（34 处生产代码）**：`maskClosable={false}` ×20｜解构默认值/类型定义 ×11｜
`{expr}` ×3（`BulkWriteDrawer.tsx:241` `{!isExecuting}`、`FormModal.tsx:143` / `BaseEditModal.tsx:61` 透传）。
测试文件 12 处与被测组件同 commit 更新（FormModal/BaseEditModal/confirmDelete 的测试断言了旧 prop 形态，见 §3.3）。

### 1.6 其余基线复跑

| 指标 | 实测 | 与记载对照 |
|---|---|---|
| M1 全 `.css` `!important` | **1435** | ✅ 一致 |
| M2 `index.css` `!important` | **1359** | ✅ 一致 |
| `data-color_mode` 拼写错误 | **8 处**（全在 index.css） | ✅ 一致 |
| `.module.css` 含 `.ant-` | **6 条**（`building-spaces/components/styles.module.css` 唯一文件） | ✅ 一致 |
| `var()` 引用未定义变量 | `--input-placeholder` 定义仅 `:189`（light 块），引用 3 处（暗色 2 处失效 + light 1 处有效）；`--theme-warning-text` 引用 2 处 0 定义；`--radius` 引用 1 处（有 fallback） | ✅ 一致 |
| `AntdThemeBridge.tsx` useMemo | `:63`，deps `[themeMode, density]`（`:175`） | ✅ 一致 |
| 暗色重复目标 class | 80（PITFALLS §3.2 聚类口径，复算需 cssparse 脚本，未重复实跑） | 引用原值 |

---

## 实现方案建议

### 2.1 VISFIX-06：契约脚本自身修复（最小 diff）

**① `css-contract-assertions.js`（3 处）**：

```js
// :48 fixture class 修正（status class 在 antd 6 落在 wrapper 自身）
- const w5 = el('span', 'ant-input-affix-wrapper ant-input-outlined ant-input-affix-wrapper-status-error', light);
+ const w5 = el('span', 'ant-input-affix-wrapper ant-input-outlined ant-input-status-error', light);

// :69 passed 期望色：antd 默认红 → 项目品牌红（--ant-color-error #BA3630）
- /255,\s*77,\s*79/.test(cs(w5).borderTopColor),
+ /186,\s*54,\s*48/.test(cs(w5).borderTopColor),

// :14/:22 注释同步更新（wrapperError 期望 rgb(186,54,48)；历史记录注明暗色实测 rgb(161,49,44)）
```

**验证「真的能 PASS」的可执行流程**（两选一）：
- **agent-browser（推荐，环境已具备该 skill）**：`npm run dev`（:4000）→ 登录进入任一含输入框页面 →
  对该页执行 eval 注入 `css-contract-assertions.js` 全文 → 断言返回 JSON `passed === true`。
- **人工粘贴**：同环境打开 DevTools Console → 整段粘贴（脚本自带 fixture 注入与清理，不触碰真实表单）→ 看 `[css-contract-assertions]` JSON。

**② `css-contract-check.js` C3b 反转（1 行 + 注释）**：

```js
// :57 反转：正向锁死 → 反向断言（死规则不得存在）。复用 :77 的剥注释文本避免注释误伤。
- check('C3b dept-tree-search 补丁保留', /\.dept-tree-search \.ant-input-affix-wrapper \.ant-input/.test(indexCss));
+ const indexCssNoComment = indexCss.replace(/\/\*[\s\S]*?\*\//g, '');   // 上移到 C3 之前
+ check('C3b dept-tree-search 死规则不得存在（VISFIX-03，反向断言）', !/\.dept-tree-search[^\n]*\.ant-input-affix-wrapper/.test(indexCssNoComment));
// C3c（:58 user-form-input 活规则正向断言）保持不变
```

**顺序硬约束**：C3b 反转与 D1-D4 删除必须**同一 commit**——先反转后不删 = 断言红（死规则还在）；
先删后反转 = 中间窗口 CI 红。

### 2.2 C6-C15 逐个设计

| 编号 | 检测逻辑 | 当前 HEAD 命中 | 误报风险与职责边界 | 落地形态建议 |
|---|---|---|---|---|
| **C6** 变量必须定义 | 扫 `index.css`：收集所有 `--x:` 定义（按块作用域）+ 所有 `var(--x)` 引用；引用处作用域无定义且无 fallback → 报 | 3 处（`--input-placeholder` 暗色 2 + `--theme-warning-text` 2 去重后 3 引用） | 低；与 eslint P65/P66/P68（JS 侧）零重叠 | **带豁免清单落地**（初始登记 3 处），Phase 89 R11 清偿后清空豁免 |
| **C7** 暗色目标不重复 | 复用 PITFALLS §3.2 聚类算法（gap>5 切 cluster → 同一 `.ant-*` 目标出现在 ≥2 cluster 即报） | **80** | 中（算法口径敏感） | **延后 Phase 92**（本 phase 不修暗色 → 断言常红，同 C13 逻辑，见 §5） |
| **C8** 死文件检测 | 枚举 `src/**/*.css|less`，对每个 basename 查 `src/**/*.{ts,tsx}` 的 import 语句（`import "…<name>"` / `from "…<name>"`），入口文件白名单（`index.css` 被 `main.tsx` 引） | 删除 D16/D17 后 **0** | 中——需注意动态 import（`import.meta.glob` 当前只 glob tsx）；留 `allowlist` 字段兜底 | **立即落地，阻断** |
| **C9** 变量不重复赋值 | 同一 `--x` 在 >1 个选择器块赋值（白名单：明暗成对变量如 `--input-bg` light/dark 双写） | 数处（`--input-bg`×3、`--sidebar-width`×4 等） | 中（明暗双写是合法模式，白名单是关键） | **延后 Phase 89**（R9/R10 是 89 的活，先落地必红） |
| **C10** module.css 不含 `.ant-` | `*.module.css` 文件文本含 `.ant-` 即报 | **6**（styles.module.css Modal 域） | 极低 | **落地但先豁免该 1 文件**（迁出归 Phase 91），或延后 91 落地 |
| **C11** `!important` 预算 | 全 `.css` 计数（M1 口径）> 预算值即 fail；**ratchet 模式**（预算只能降不能升） | 删除第 1 批后 ≈**1395** | 极低 | **立即落地，预算 1395**，随各 phase 递减（92 终值 ≤480） |
| **C12** `data-color_mode` 拼写 | `.css` 文本含 `data-color_mode` 即报 | 删 D5/D6 后 **0** | 极低 | **立即落地，阻断** |
| **C13** `.css` hex 全量 | 已裁定**归 Phase 89**（STYLE-04 门禁 + `exemption-reasons.json` 扩 `layer:"CSS"`；先落会被自己的门禁卡死——P-m7 决议） | 311（index.css） | — | 不落 |
| **C14** 弃用 prop 零命中 | `src/**/*.{ts,tsx}`（排除 `*.test.*` 可先排除后收紧）文本匹配弃用清单：`maskClosable`/`destroyOnClose`/`bodyStyle`/`headStyle`/`maskStyle`/`footerStyle`/`contentWrapperStyle`/`size="middle"`/`size="default"`/`dropdownRender`/`dropdownClassName`/`popupClassName`/`onDropdownVisibleChange`/`dropdownMatchSelectWidth`/`showArrow`；**`bordered` 不进正则**（Descriptions/Table 未弃用，会大量误报——Card bordered 靠 code review 或后续按组件甄别） | 清理后 **0** | `bordered` 必须排除；`onSearch` 绝不进清单（关键决策） | **立即落地，阻断**，放 `css-contract-check.js` 或独立 `check-deprecated-antd.mjs` |
| **C15** AntdThemeBridge useMemo 依赖覆盖 | vitest 单测 `src/design-system/components/__tests__/AntdThemeBridge.test.tsx`：断言 `antdThemeConfig` 的 useMemo deps 包含其读取的全部外部状态（当前 `themeMode`/`density`；89 扩色板输入源时此测试逼着同步扩 deps——P-C2 的 detection） | 0（测试新写） | 极低 | **立即落地**（纯新增测试） |

**门禁接入**：`css-contract-check.js` 当前不在任何 script 里。建议：
`package.json` 的 `lint:ci` 追加 `&& node scripts/css-contract-check.js`（零新依赖，与既有链同风格）。
`css-contract-assertions.js` 是浏览器片段，**不进 CI**（jsdom 解析不了 antd 6 `@layer`，P-M10 已实证），保持人工/agent-browser 执行。

### 2.3 D1-D17 删除协议（每条删前取证 → 删 → 删后复核）

**删前取证（浏览器命中数）**：环境有 **agent-browser skill**，可自动化；无浏览器时手贴 Console 亦可。
对每条死规则取**完整选择器**的命中数：

```js
// 必须全部 === 0 才进删除清单（D1-D4 家族）
document.querySelectorAll('.dept-tree-search .ant-input-affix-wrapper').length                     // 0
document.querySelectorAll('.dept-tree-search .ant-input-affix-wrapper .ant-input').length          // 0
// 佐证活规则（必须 > 0，证明 K8/K9 顶上）：
document.querySelectorAll('.user-form-input.ant-input-affix-wrapper .ant-input').length            // >0
// D7-D15 家族
document.querySelectorAll('.ant-select-selector').length                                           // 0（全站）
// D5/D6 家族：拼写错误无对应 DOM 属性可查，用 getAttribute 佐证
document.documentElement.getAttribute('data-color-mode')                                           // 'light'|'dark'（不存在 color_mode）
```

**删除顺序（从大行号到小行号，避免行号漂移）**：
1. D15 `5579-5581` → 2. D12-D14 `3962-3977` → 3. D6 `3602-3608` → 4. D5 `3523-3545` →
5. D11 `2329-2333` → 6. D1-D4 `2092-2117`（§1.2 边界）→ 7. D10 `897` 行+去逗号 → 8. D9 `890` 行+去逗号 →
9. D8 `884` 行+去逗号 → 10. D7 `873` 行+去逗号 → 11. 死文件 8 个 `git rm` + `FloorPlanEditor.tsx:18` import 行删除。

**每步验证**：`node scripts/css-contract-check.js` exit 0（C3b 已反转）→ 最终 `git diff` 复核无活规则误删。

**视觉 diff = 0 判定流程**（复用 `.planning/notes/v1.24-visual-verification.md`，**已确认可复用**）：
- 该 note 提供：21 形态 × 明暗矩阵完整口径（§2 表，11 输入类 + Select 单/多 + picker 族 + AppInput 等）、
  14 页巡检清单、以及三条关键环境事实（headless 需 CDP `Emulation.setFocusEmulationEnabled` 开焦点模拟；
  focus emulation 不跨 CDP session，须同连接「开模拟→立刻求值/截图」；焦点环 0.15s 过渡需等 ≥450ms 再取样）。
- 本 phase 执行：删除前后各跑一轮 21 形态矩阵（computed-style 四元组 `backgroundColor/borderColor/borderRadius/boxShadow`）
  + 明暗 14 页巡检截图；逐格 diff 必须为空。死规则命中数为 0 的前提下 diff=0 是**数学必然**，矩阵是兜底证据。
- 截图证据入库 phase 目录（沿用 v1.24 的 `v1.24-visual-evidence/` 惯例）。

### 2.4 弃用清理替换规则表（VISFIX-04，独立 commit）

| 类 | 机械替换规则 | 非机械例外（逐处人工） |
|---|---|---|
| `maskClosable={false}` ×20 | → `mask={{ closable: false }}` | 若同元素已有 `mask={…}` prop → 合并进既有对象 |
| `maskClosable={true}` / 裸 `maskClosable` ×2（含 Drawer:224） | → `mask={{ closable: true }}`（裸 prop = true） | Drawer 与 Modal 同规则（Drawer.d.ts:27 同弃用） |
| `maskClosable={expr}` ×3 | → `mask={{ closable: <expr> }}` | `FormModal.tsx:143` / `BaseEditModal.tsx:61` 是**透传位**：组件 props 的 `maskClosable?: boolean` 需同步改名为 `mask`（对外 API 变化 → **检查其全部调用方**；或保留组件 prop 名 `maskClosable`、内部转发 `mask={{closable}}`——**推荐后者**，零调用方破坏，但外层 prop 名仍含旧词 → 需 checker 裁定 C14 是否豁免这两个文件） |
| `confirmDelete.ts:66` `maskClosable: false`（对象字面量，Modal.confirm 静态方法） | → `mask: { closable: false }` | 确认 Modal.confirm 的 config 同样走 deprecated 检查（是，`Modal.js:126` 对 props 对象统一检查） |
| `Card bordered={false}` ×6 | → `variant="borderless"` | 无受控/表达式形态（6 处全是字面 `false`） |
| `size="middle"` ×17 | → `size="medium"` | 纯枚举替换；逐文件确认组件是 Table/Divider/Descriptions/Spin 等（全部支持 medium） |
| `size="default"` ×1 | → `size="medium"` | 确认组件（Card.js:76 证实 Card 语义） |
| `bodyStyle` ×4（全部 Card） | → `styles={{ body: {…} }}` | 若同元素已有 `styles` prop → 合并 |
| `destroyOnClose` ×6 | → `destroyOnHidden` | Drawer 的 1 处同规则；裸 prop 形态保持裸 |
| `onSearch` / `showSearch` | **不动** | 关键决策锁定（未弃用）；`AppSelect` 的 `showSearch ?? count>=8` 联动 warning 属 rc-select 提示，非 deprecated 警告，不属本 phase |

**验证**：
1. `grep -rn 'maskClosable\|destroyOnClose\|bodyStyle\|size="middle"' src --include=*.tsx --include=*.ts`（排除测试后）= 0；
   C14 断言接入门禁长期守护。
2. **浏览器警告归零验证**：`maskClosable`/`bodyStyle` 警告在打开任一使用 `FormModal`/`BaseEditModal` 的页面即触发
   （如 `/system/users` 新增弹窗、`/monitor/cache`、`/network/credentials`）——Console 过滤 `Warning:` 计数应为 0
   （清理前：每页开 Modal 即报）。**注意**：`onSearch should work with showSearch`（`AppSelect` 联动提示）**不在清理范围**，
   验收口径要写「无 **deprecated** 类警告」，不得写成「Console 零 Warning」。
3. **vitest 影响面**：全口径 46 - 34 = **12 处测试文件命中** `maskClosable`——含 `FormModal`/`BaseEditModal`/`confirmDelete`
   的契约测试可能断言了旧 prop 形态（如 `getByRole` 断言不受影响，但若有 `props` 快照/`render` 传参断言需同 commit 更新）。
   跑 `npm run test` 全量（基线 569 文件 3960 用例）确认零回归。
4. **防误伤（显式写入计划）**：`Descriptions bordered`（≈16）/ `Table bordered`（7）/ 业务字段 `bordered`（2）**一律不改**；
   `onSearch`（271）/ `showSearch`（340）**不改**。

---

## Validation Architecture

| 维度 | 自动化命令 / 方法 | 通过判据 |
|---|---|---|
| 静态契约 | `node scripts/css-contract-check.js`（已扩 C3b 反转 + C8/C12/C14/C11） | exit 0，全 PASS |
| 门禁接入 | `npm run lint:ci`（追加 css-contract 后） | exit 0 |
| lint / 类型 / 测试回归 | `npm run lint` / `npm run type-check` / `npm run test` | 0 problems / 全绿 / 3960 用例全绿 |
| 弃用 prop 零命中 | `grep -rn 'maskClosable' src …` 逐类 + C14 断言 | 0 命中（排除测试） |
| 浏览器动态断言 | agent-browser eval `css-contract-assertions.js` | `passed === true`，`doubleCount === 0` |
| 死规则命中数 | agent-browser `document.querySelectorAll(...)` 逐条 | 删前全 0 / 活规则 >0 |
| 视觉 diff | 21 形态 × 明暗 computed-style 矩阵（v1.24 note 方法：CDP 焦点模拟 + ≥450ms 取样）+ 明暗 14 页巡检截图 | 逐格 diff = 0（**唯一硬判据**） |
| console 基线 | 打开 FormModal 类页面（`/system/users` 新增）过滤 `Warning:` | 0 条 deprecated 警告 |
| 量化快照 | M1/M2/M9 等 §3.1 口径命令，phase 报告落盘 | M1 1435→≈1395；M9 5632→≈5580；M14 死文件 0 |

人工项：暗色矩阵截图肉眼复核一遍（防 computed-style 漏 interpolation）；`ruoyiBrand` 相关无改动确认。

---

## 波次建议

**先修底座 → 再落断言 → 再删 → 再清弃用**（每波一个 commit，独立可回滚）：

| 波次 | 内容 | 依赖 | 验证闸门 |
|---|---|---|---|
| **W1** | fixture 修复（assertions 3 处）+ C3b 反转（**与 W3 第一刀同 commit**，见 §2.1） | — | agent-browser `passed === true`（W1 即可验，fixture 修复独立于删除） |
| **W2** | 新断言落地：立即可阻断的 **C8 / C12 / C14 / C11(ratchet=1395) / C15**；接 `lint:ci`。C8/C12/C14 在 W3/W4 之前落地会红 → **先以 warn-only 接入，W3/W4 完成后翻转为阻断**（或直接排 W3/W4 之后落地） | — | `lint:ci` exit 0 |
| **W3** | D1-D17 删除（§2.3 协议，含 C3b 反转同 commit） | W1 | 删前命中数取证全 0 → check exit 0 → **21 形态视觉 diff = 0**（唯一硬判据，W3 收口时执行） |
| **W4** | VISFIX-04 弃用清理（独立 commit，与 W3 无文件冲突） | 无硬依赖（建议最后，让 console 基线干净） | grep 零命中 + deprecated 警告 0 + vitest 全绿 + 抽样形态复核 |

**「本 phase 落地 vs 延后」断言清单**（checker 曾裁定 C13 归 89 因「Phase 0 会被自己的门禁卡死」——同一逻辑适用）：

| 断言 | 处置 | 理由 |
|---|---|---|
| C6 变量必须定义 | **落地（带豁免清单，3 处登记）** | 检测本身有价值；命中点 R11 归 89，豁免机制让 89 清偿后清空即阻断 |
| **C7 暗色目标不重复** | **延后 Phase 92** | 命中 80，本 phase 不修暗色分区（92 主责）→ 常红必被绕过或强改豁免，与 C13 同理 |
| C8 死文件检测 | **落地（阻断）** | W3 后命中 0，天然绿 |
| C9 变量不重复赋值 | **延后 Phase 89** | 命中含 `--input-bg`×3 / `--sidebar-width`×4，R9/R10 是 89 的活 |
| C10 module.css 不含 `.ant-` | **落地（1 文件豁免）** 或延后 91 | 6 条 Modal 迁出是 91 的活；豁免机制成本低于断言缺席 |
| C11 `!important` 预算 | **落地（ratchet 1395）** | 防回升是收敛期第一需求 |
| C12 拼写断言 | **落地（阻断）** | W3 后命中 0 |
| C13 hex 全量 | 延后 89（已裁定） | exemption 机制需先扩展 `layer:"CSS"` |
| C14 弃用零命中 | **落地（阻断）** | W4 后命中 0 |
| C15 useMemo 单测 | **落地** | 纯新增测试，且是 89 扩色板输入源时的安全网 |

---

## 边界与不做什么

1. **不动任何活规则**：R1-R5（Input 域）、暗色 cluster（R6/R7）、`--input-bg` 串色（R10）、K1-K11 全部不碰（89/90/91/92 的活）。
2. **不迁 `onSearch`**（271 处）——未弃用（关键决策锁定）；`Descriptions bordered` / `Table bordered` / 业务字段 `bordered` 不改。
3. **不接 CI 的新浏览器断言**：`css-contract-assertions.js` 保持人工/agent-browser 执行（jsdom 解析不了 antd 6 `@layer`）。
4. **不引新依赖**（stylelint / Playwright / codemod 均不用；`@ant-design/cli` 可选作 W4 的交叉核对工具，非必需）。
5. **不修暗色 / 不动色板 / 不动 hex**（C7/C9/C13 延后）。
6. **不 git commit** 由执行 phase 按仓库惯例处理；本 phase 每个 commit 仍需用户确认（CLAUDE.md 规则）。
7. `dashboard-system/edit.css` / `view.css` 是活文件，**不得**随 D17 一起删。
8. `src/components/ConfigProvider.tsx` 冻结（P-C2）；`AppInput`/`AppSelect`/`AppTextArea` 默认值冻结（P-M7）。

---

## Sources

### 实跑命令（Windows / `ruoyi-react-frontend/`）

```bash
wc -l src/index.css                                                          # 5632
sed -n '2090,2132p;3520,3546p;3598,3612p;871,901p;2329,2333p;3962,3977p;5579,5581p' src/index.css
grep -c 'data-color_mode' src/index.css                                      # 8
# 死文件：逐个 Get-Content 行数 + grep basename（带 .css/.less 后缀精确口径）
grep -o '!important'（全 .css 汇总）= 1435；index.css = 1359
grep -c 'ant-select-selector' src/index.css                                  # 9（all-matches）
# 弃用 prop：maskClosable 34 生产/46 全口径；Card bordered 6；Descriptions 16；Table 7；
#            size="middle" 17；bodyStyle 4（全 Card）；destroyOnClose 6；Select 六类 0
# node_modules 源码：antd/es/drawer/Drawer.d.ts:21,27（destroyOnClose/maskClosable 弃用）、
#                   antd/es/card/Card.js:76,93（size default/bordered/bodyStyle 弃用）
```

### 读过的文件

- `.planning/ROADMAP.md`（Phase 88 条目 + 三处不可逆顺序约束 + M1-M16 口径）
- `.planning/REQUIREMENTS.md`（VISFIX-03/04/06 + STYLE-02 第 1 批 + 关键决策表）
- `.planning/research/PITFALLS.md`（§四批清单 / P-C5 / P-M10 / P-M11 / §3.3）
- `.planning/research/STACK.md`（§B 弃用清单 / §对本项目的可执行结论）
- `scripts/css-contract-check.js`（89 行）/ `scripts/css-contract-assertions.js`（74 行）
- `package.json`（`lint:ci` 组成，无 css-contract）
- `.planning/notes/v1.24-visual-verification.md`（21 形态矩阵 + 14 页巡检 + CDP 方法，可复用）
- 源码定位：`routeGenerator.ts:129` / `componentLoader.tsx:34-36` / `FloorPlanEditor.tsx:18` /
  `AntdThemeBridge.tsx:63,175` / `LocationAliasDrawer.tsx:224` / `FormModal.tsx:143` / `BaseEditModal.tsx:61`
