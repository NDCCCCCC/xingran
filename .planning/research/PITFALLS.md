# PITFALLS: 项目 CSS 冲突面盘点与 antd 6 variant 收敛路径

**Project:** RuoYi-Next 运维管理系统
**Milestone:** v1.25 全站权限治理与样式收敛（② 样式层收敛定向研究 · 项目 CSS 侧）
**Researched:** 2026-10-28
**Confidence:** 总体 HIGH（盘点数字实跑得出；落地效果仍需实机验证）

> 本文件是 **只读研究**产出：未修改任何业务代码、未 git commit。所有统计数字来自下文
> [Sources § 实跑命令](#sources) 中列出的真实命令输出。行号以当前 HEAD
> (`ruoyi-react-frontend/src/index.css` = **5632 行**) 为准。
> 标注约定：**HIGH** = 实跑统计 / 源码实证；**MEDIUM** = 由实证推导；**LOW** = 需人工实机验证。

---

## 0. 执行摘要（先读这页）

### 0.1 与 v1.24 既有事实的复核结果

| v1.24 给出的数字 | 本次实跑复核 | 结论 |
|---|---|---|
| `index.css` 约 5000+ 行 | **5632 行** | ✅ 成立（比 v1.24 记录的 5622 又多 10 行） |
| `index.css` 中 `!important` **1359 处** | **1359 处** | ✅ **完全一致** |
| 命中 antd 选择器约 **219 行** | **798 行**含 `.ant-` / **748 条**顶层规则含 `.ant-` | ⚠️ **口径不同**（219 疑为 v1.24 当时的抽样或另一口径），本文件改用「规则条数」为准 |
| 全文件 **96 个分区** | **97 个** `/* =====` 分区块 | ⚠️ 差 1（文件自 v1.24 后有增补） |
| 暗色重复分区约 **60 个** | **44**（间隙 >12 行）/ **64**（间隙 >5 行）/ **100**（间隙 >3 行） | ✅ 「约 60」成立（对应间隙 >5 口径 = 64） |
| `--input-bg` 第 3 处就地重定义 `index.css:4807` | **实际在 `index.css:4815`**（`:root` 块起于 4812） | ⚠️ 行号有 8 行偏差，**结论不变** |
| 暗色重复分区用 `[data-color-mode="dark"]` | ✅ 574 处正确 + **8 处拼写错误 `[data-color_mode="dark"]`**（`index.css:3523-3545`, `3602-3608`） | ➕ **新发现：8 条必然死规则** |

### 0.2 五条关键发现（新增，v1.24 登记册未覆盖）

1. **F-01 · `--input-bg` 串色坐实且已定位到根因行**
   `index.css:4815` 的 `:root { --input-bg: #fafaf9 }` 与 `index.css:198` 的
   `[data-color-mode="dark"] { --input-bg: #0f2e1b }`（第 254 行）**特异性相同 (0,1,0)**，
   后者在前、前者在后 → **暗色下 `--input-bg` 最终解析为浅色 `#fafaf9`**。
   连锁效应：`--input-bg-hover` / `--input-bg-focus` / `--input-bg-disabled` **只在暗色块定义**
   （255/256/257），未被覆写 → 暗色下形成「底色浅、悬停/聚焦深」的**半明半暗混合态**。
   影响面：`--input-bg` 被 **22 处** `var()` 引用（`index.css` 内 22 行）。

2. **F-02 · 7 个 CSS 文件是「从未被 import 的死文件」**（共 **470 行** + **9 处 `!important`**）
   `theme-styles.css`（97 行 / 7 important）、`dashboard-system/index.css`（129 行）、
   `ChartWidget.css`（23 行 / 2 important）、`ListWidget.css`（77 行）、
   `MetricWidget.css`（24 行）、`ProgressWidget.css`（53 行）、`StatCardWidget.css`（67 行）。
   另有 `CADFloorPlanEditor.less`（85 行）**未被 import**，`FloorPlanEditor.less` 为 **0 字节空文件**。
   → 这是本里程碑「可整块删除」批次中**风险最低、收益最高**的一批。

3. **F-03 · `.ant-select-selector` 在 antd 6.6.4 已完全不存在 —— 死规则族扩大到 9 行 / 7 条规则**
   实证：`grep -r 'select-selector' node_modules/antd` 唯一命中在
   `node_modules/antd/dist/antd.css`，且形态是
   `.ant-color-picker .ant-color-picker-inner … .ant-color-picker-format-select.ant-select-single .ant-select-selector{…}`
   —— 是 **ColorPicker 内部的后代选择器**，普通 Select **不渲染** `.ant-select-selector` 元素。
   `node_modules/@rc-component/select/es/` 全目录 **零命中** `-selector`。
   → v1.24 登记册 S1-S4 全部坐实，可直接进「第 1 批｜可删」。

4. **F-04 · 暗色下有 2 处 `var()` 引用了「只在浅色块定义」的变量 → 声明直接失效**
   `index.css:2304`（`.ant-input::placeholder`）与 `index.css:2335`
   （`.ant-select-selection-placeholder`）使用 `var(--input-placeholder)`，
   而该变量**只在 `[data-color-mode="light"]`（184-190 区间）定义**，暗色块定义的是
   `--input-text-placeholder`。CSS 变量未定义时 `var()` 整条声明在计算值阶段作废 →
   **暗色 placeholder 颜色回落到继承值**，与设计意图（`#707068`）不符。
   另有 `--theme-warning-text`（2 次）与 `--radius`（1 次，有 fallback）**全局从未定义**。

5. **F-05 · hex 治理对 `.css` 近乎完全失明，且现有 CSS 扫描脚本本身已在失败态**
   `eslint.config.js:132` 的 hex 禁令块 `files: ["src/**/*.{ts,tsx}"]` —— **不含 `.css`**；
   而真正扫 `.css` 的 `scripts/check-hardcoded-colors.mjs` 虽有 31 条 hex 映射，
   但 `index.css` 里 311 处 6 位 hex 中**只有 2 处命中 denylist**（`#ff4d4f`@964、`#faad14`@970）
   → **309 / 311 = 99.4% 的硬编码色不受任何门禁约束**。
   且该脚本**当前 exit code = 1**（18 个文件命中），**未被接入 `lint:ci`** → 现状是「有扫描器、无门禁」。

### 0.3 四批规则清单规模（详见 [§ 四批规则清单](#四批规则清单)）

| 批次 | 条数 | 说明 |
|---|---|---|
| 第 1 批｜可整块删除 | **17** | 8 条死选择器规则 + 8 个死文件 + 1 条死变量 |
| 第 2 批｜可降级为 token 引用 | **14** | 硬编码 hex / 圆角 / 字号 / 阴影 |
| 第 3 批｜必须保留并登记 | **11** | 契约守护、焦点环、错误态还原 |
| 第 4 批｜需重写而非删除 | **11** | 必须迁到 variant / token 机制 |

### 0.4 与并行 `STACK.md`（同批 v1.25 CSS 定向研究）的对齐与**分歧**

对齐（双方实跑结论一致，可直接合并）：
`index.css:871-880` 的 `border` 简写压掉 status 色 / `index.css:958-965` 的 VIS-03 还原
只覆盖 Input 域 / `index.css:937-945` 的 VIS-01 复位块 / `index.css:2297-2301` 暗色 InputNumber
再压一次 / `AntdThemeBridge.tsx:82,122,130,164` 的浅色常量 / 1359 `!important` /
574 处 `[data-color-mode="dark"]` / 9 处 `.ant-select-selector` —— **行号与结论全部一致**。

**分歧 1（重要）· `STACK.md` §7 批次 B 建议「立刻删除 `index.css:937-945`」——本报告反对。**
`STACK.md` 的理由是「antd `input/style/index.js:382-395` 已声明同样的复位」。
但那个复位规则的**特异性只有 (0,1,1)**，而项目 `index.css:875` 的
`.ant-input { border: … !important }` 是 **(0,1,0) + `!important`** ——
`!important` 直接压过一切非 `!important` 规则，与特异性无关。
→ **只要 `:875` 还在，删掉 `:937-945` 就会立刻恢复双框**（309 个输入框）。
正确顺序是：先删/改 `:875`（第 4 批 R1），**同一 commit 内**再重估 `:937-945`。
（`STACK.md` 批次 C 与 B 顺序相反：B 在 C 之前。）**本报告把 `:937-945` 列入第 3 批（K1），
并在 Phase 4 明确「删覆盖与保留复位块必须同 commit」。**

**分歧 2 · `STACK.md` 未提及的三项本报告新增发现**：
① `index.css:4815` 的 `:root { --input-bg: #fafaf9 }` 串色（`STACK.md` 引用了
`index.css:1-5632` 的整体统计但未定位到这一行）；② 7 个从未 import 的死 CSS/`.less` 文件
（555 行）；③ `css-contract-assertions.js:48` 的 fixture 用了 antd 6 不存在的 class
→ 该动态断言**恒为 false**，VIS-03 实际处于失守状态。

---

## Critical Pitfalls

## Critical Pitfalls

### P-C1 · 收敛中途态「删一半破相」：边框/焦点环/错误态三源并存

- **What goes wrong:** 项目手写 CSS 与 antd 6 同时为同一属性提供边框与焦点环。若先删项目层
  而 antd 侧（`variant` / `colorBgContainer` / `colorBorder`）尚未就位，输入框会瞬间回到
  antd 默认 6px 圆角 + 默认蓝灰边框 + antd 默认焦点环，与 v1.24 已验收的「单圈 2px 绿焦点环 /
  错误态红框」不符；若先动 antd 侧而项目层还在，则出现**双框**（v1.24 P0 原症状复发）。
- **Why it happens:** `index.css:871-880`（`.ant-input` 边框/底/圆角，`!important`）、
  `index.css:908-914`（`.ant-input-affix-wrapper` 边框，`!important`）、
  `index.css:937-945`（VIS-01 内层复位）、`index.css:958-971`（VIS-03 错误态还原）
  四条规则构成一个**互相咬合的最小闭环**。任何一条被单独移除，闭环即破。
- **Consequences:** v1.24 已交付的 VIS-01/02/03 回归（双框数 3→0、焦点环单圈、错误态红框
  `rgb(186,54,48)`）全部失效；`scripts/css-contract-check.js` 的 C1/C4/C5 三项断言转为 FAIL。
- **Prevention:** 严格按 [§ 收敛顺序](#收敛顺序建议) 的 **Phase 3 → Phase 4** 顺序：先由
  `AntdThemeBridge` 把 antd 侧 token 调到与项目层**逐像素等价**（灰度等价期，两层同时生效且值相同），
  再在同一个 commit 内删除项目层对应规则。**禁止**出现「antd 侧已改、项目层未删」跨 commit 的窗口。
- **Detection:** 每个 commit 必跑 `node scripts/css-contract-check.js`（静态结构，退出码 0）
  + `scripts/css-contract-assertions.js`（浏览器 computed-style，`doubleCount` 必须 = 0）。

### P-C2 · 双 ConfigProvider 嵌套：主题 token 的真实生效点不在你以为的地方

- **What goes wrong:** `src/App.tsx:47-48` 是 `<AntdThemeBridge>` 包裹 `<ConfigProvider>`；
  `AntdThemeBridge.tsx:178` 内部才是真正的 antd `<ConfigProvider theme={antdThemeConfig}>`。
  项目自有的 `src/components/ConfigProvider.tsx` **不渲染任何 Provider**，它只是
  `useEffect` 同步 `settingsStore → themeStore/layoutStore` 的**副作用组件**（返回 `children`）。
  → 任何「我改了 ConfigProvider 的 theme 为什么没生效」的排查都会走错方向；
  任何在 `AntdThemeBridge` 之外的层级写 antd token 都无效。
- **Why it happens:** 两处都叫 `ConfigProvider`（`@/components/ConfigProvider` 与 antd 的），
  命名相同、语义完全不同；且 `AntdThemeBridge` 用 `useMemo([themeMode, density])` 缓存
  themeConfig —— **token 只在这两个值变化时重算**。若把「明暗色板」改成依赖其它 store 字段，
  必须同步扩 `useMemo` 依赖数组，否则 token 不更新。
- **Consequences:** 收敛期改 token 全部「看起来没反应」，浪费 phase；
  更糟的是有人会去改 `src/components/ConfigProvider.tsx`（改 themeStore 同步），
  造成**明暗切换延迟一帧**或**首次进入暗色闪烁**。
- **Prevention:** 收敛期只允许在 `AntdThemeBridge.tsx` 的 `antdThemeConfig` 内改 token；
  `src/components/ConfigProvider.tsx` **冻结**（只读）。
  若新增 token 输入源，必须在同一个 commit 内扩 `useMemo` 依赖并加一条断言。
- **Detection:** 在 `AntdThemeBridge.tsx` 的 `useMemo` deps 上加单测断言（见
  [§ 量化基线 § 3.3](#33-契约脚本扩展点)）；浏览器侧断言
  `getComputedStyle(document.querySelector('.ant-btn-primary')).backgroundColor`。

### P-C3 · 暗色 `--input-bg` 串色：不修这一条，后面所有「暗色对比度验收」都无意义

- **What goes wrong:** `index.css:4815` 的 `:root { --input-bg: #fafaf9 }` 与
  `index.css:254` 的暗色 `--input-bg: #0f2e1b` 特异性相同、后者在前 → 暗色下解析为 `#fafaf9`。
  实测（v1.24 记录）输入框底 `rgb(250,250,249)` 压在深绿页面底 `rgb(15,46,27)` 上，
  对比度约 **1.15:1**（远低于 WCAG AA 4.5:1 与 UI 组件 3:1）。
- **Why it happens:** `:root` 与 `[data-color-mode="dark"]` **特异性相同**（均为 0,1,0），
  纯靠**源码顺序**决胜；而项目把「原型实测」的浅色输入框底写进了后置的 `:root` 块，
  暗色块无法覆盖它。同时 `--input-bg-hover/-focus/-disabled` **只在暗色块定义**，
  于是形成「底浅、态深」的割裂。
- **Consequences:** 暗色模式下**全部 22 处** `var(--input-bg)` 引用点（`index.css` 内 22 行）
  拿到浅色底 —— 输入框/InputNumber/ColorPicker/Checkbox/Radio/Pagination 等一起失真；
  任何「暗色对比度达标」的验收都会被这一条单独判负。
- **Prevention:** 收敛 Phase 2 单独修这一条，且**不与任何 antd 覆盖改动同 commit**：
  1) 把 `--input-bg` 的浅色实测值移出后置 `:root`，改放 `:root`（第 27 行那个）并保留
     `[data-color-mode="light"]`（184 行）与 `[data-color-mode="dark"]`（254 行）两处；
  2) 或把 `:root`@4812 里的 `--input-bg` 改为 `--input-bg-light` 并只在 light 块引用；
  3) 补齐暗色块的 `--input-placeholder` → `--input-text-placeholder`（见 P-C4）。
- **Detection:** 浏览器控制台执行
  `getComputedStyle(document.documentElement).getPropertyValue('--input-bg')`
  在 `data-color-mode="dark"` 下必须返回 `#0f2e1b`（或等价暗色值），**不是** `#fafaf9`。

### P-C4 · 暗色下 `var()` 引用未定义变量 → 声明整条作废（颜色回落到继承值）

- **What goes wrong:** `index.css:2304`（`[data-color-mode="dark"] .ant-input::placeholder`）
  与 `index.css:2335`（`.ant-select-selection-placeholder`）使用 `var(--input-placeholder)`，
  该变量**只在浅色块 `[data-color-mode="light"]` 定义**（`index.css:184-190` 区间），
  暗色块定义的是 `--input-text-placeholder`。CSS 变量未定义 → `var()` 所在声明
  在**计算值阶段**作废 → 该属性回落到继承值 / 初始值。
- **Why it happens:** 变量命名不统一（`--input-placeholder` vs `--input-text-placeholder`），
  而 `:root` 主块里也**没有**兜底定义 → 浅色能用（light 块定义了）、暗色失效（dark 块没定义）。
  静态审查容易漏，因为「在 CSS 里能找到这个变量名」。
- **Consequences:** 暗色下 placeholder 文字颜色不可控（可能与背景同色而**完全不可见**）；
  暗色 Select 的 placeholder 同样受影响 —— 而 Select 恰好是 v1.24 已知白底高风险区。
- **Prevention:** 收敛 Phase 2 与 P-C3 一起修：统一变量名，并在 `:root`（27 行）补全
  `--input-placeholder` 的默认值，使任何作用域都有兜底。
  顺带清理全局未定义变量：`--theme-warning-text`（2 处）、`--radius`（1 处，已有 fallback）。
- **Detection:** 新增静态检查「`var(--x)` 引用的变量必须在 `:root` 中出现」
  （见 [§ 量化基线 § 3.3](#33-契约脚本扩展点) C6）；浏览器侧断言
  `getComputedStyle(input).getPropertyValue('--input-placeholder')` 非空。

### P-C5 · `.dept-tree-search` 是死规则，而 `css-contract-check.js` C3b 正在**锁死这条死规则**

- **What goes wrong:** `index.css:2093-2116` 的 4 条规则形如
  `.dept-tree-search .ant-input-affix-wrapper { … }` —— **后代选择器**，
  要求存在一个「既是 `.dept-tree-search` 的后代、又带 `.ant-input-affix-wrapper`」的元素。
  但 `src/components/DeptTree/index.tsx:205` 把 `className="dept-tree-search user-form-input"`
  直接传给 `<AppInput>`。antd 6 的 `@rc-component/input@1.3.1`
  `es/BaseInput.js:129-131` 明确注释「`className` and `style` are always on the root element」，
  且 `es/utils/commonUtils.js` 的 `hasPrefixSuffix = !!(prefix || suffix || allowClear)`
  —— **AppInput 默认 `allowClear=true`** → 根元素就是 `.ant-input-affix-wrapper` 本身。
  ⇒ `class="ant-input-affix-wrapper ant-input-outlined dept-tree-search user-form-input"`
  ⇒ `.dept-tree-search .ant-input-affix-wrapper` **永不命中**（命中数 0，v1.24 已实测）。
- **Why it happens:** antd 5 → 6 把 `className` 从内层 `<input>` 移到了根元素（affix wrapper），
  而项目补丁是按 antd 5 的 DOM 假设写的；`!important` 让它「看起来在生效」（因为同位置的
  `.user-form-input.ant-input-affix-wrapper .ant-input` 规则**恰好命中**），掩盖了死规则。
- **Consequences:** 双向风险：
  1) 若按「登记册 #26 陷阱：勿删」处理 → 死规则长期占用契约预算，且**误导后续收敛者**；
  2) 若直接删 → `css-contract-check.js:57` 的 **C3b 断言会 FAIL**，
     使 CI 红灯，团队可能被迫「把死规则加回来」。
- **Prevention:** 收敛 Phase 1 同时做两件事：① 删除 `index.css:2093-2116`；
  ② **同步修改 `scripts/css-contract-check.js` 的 C3b 断言**，改为断言「`.dept-tree-search`
  死规则**不得存在**」（反向断言），并把 C3c（`.user-form-input.ant-input-affix-wrapper .ant-input`，
  **该规则是活的**）保留为正向断言。
- **Detection:** `node scripts/css-contract-check.js`（改后仍须 exit 0）；
  浏览器侧 `document.querySelectorAll('.dept-tree-search').length` 与
  `document.querySelector('.dept-tree-search').className` 取证，确认 class 落在 wrapper 上。

### P-C6 · `.ant-input-affix-wrapper` 内层复位（VIS-01）是「高特异性 + 契约双守护」，不可当普通覆盖处理

- **What goes wrong:** `index.css:937-945` 的复位规则
  （`border:none / border-radius:0 / background:transparent / color:inherit /
  font-size:inherit / padding:0`，全 `!important`，特异性 (0,3,1)）
  是 v1.24 修掉「双框（309 输入框）」的核心。收敛时若按「antd 6 已有 `border:none`」的直觉删掉它，
  或把它降级为 token 引用 → **双框立刻回归**。
- **Why it happens:** antd 自带的内层 `border:none` 特异性不足（antd 6 的内层规则形如
  `.ant-input-affix-wrapper > input.ant-input`，特异性 (0,1,1)），
  压不过项目 `:875` 的 `.ant-input { border: … !important }`（(0,1,0)+important）。
  换言之：**在项目层覆盖被删之前，复位规则必须存在**；删掉项目层覆盖之后，复位规则才变成冗余。
- **Consequences:** 双框回归（309 个输入框）、TextArea 100% 双框；
  `css-contract-check.js` C1a-C1c + C5 全 FAIL；
  `css-contract-assertions.js` 的 `doubleCount` 从 0 回到 3。
- **Prevention:** 把复位规则**移到最后**：Phase 4 把 `:875` / `:910` 的边框交还给 antd
  `variant` + `colorBorder` token 后，**同一 commit 内**才允许重新评估复位块。
  推荐的终局形态是：antd 6 的 `variant="outlined"` + `Input.activeShadow` token 接管，
  复位块**只保留 `padding:0` 与 `font-size:inherit`**（这两条 antd 不提供）。
  任何情况下 **C1 断言不得删除**。
- **Detection:** `css-contract-assertions.js`（浏览器）—— `doubleCount===0`、
  `inner.boxShadow==='none'`、`wrapperFocused.shadow!=='none'`。

## Moderate Pitfalls

### P-M1 · 暗色 mega-cluster（`index.css:2254-4517`，431 条规则 / 641 处 `!important`）是「双真相源」的物理载体

- **What goes wrong:** 暗色样式在项目里有**两套**完整实现：`index.css:198-384`（暗色 CSS 变量根）
  之外，`index.css:2254-4517` 又用 **431 条顶层规则 / 641 处 `!important`** 把每个 antd 组件
  的暗色外观手写了一遍。同一组件在两套里被定义（如 `.ant-tag` 出现在 `1230` / `2517` / `3823`，
  `.ant-card` 出现在 `608` / `2432` / `2853` / `3942`）。删任一套，另一套会「补位」，
  表现为**改了没反应 / 删了又变**。
- **Why it happens:** 逐组件手写暗色，而不是让 `darkAlgorithm` + token 接管。
  实测统计：**250 个不同的暗色目标组件，其中 80 个出现在 >1 个 cluster**
  （gap>12 口径共 44 个 cluster；gap>5 口径 64 个 ≈「约 60」）。
  两个巨型 cluster：C39 = `2254-2742`（90 规则 / 138 important）、
  C40 = `2780-4517`（341 规则 / 503 important）。
- **Consequences:** ① 收敛期任何「删除暗色覆盖」都会被后一个 cluster 复活 → 出现「看似没改」；
  ② 暗色下 antd 6 的新特性（`variant` 状态色、`status` 语义色、`cssVar` token）全部被压掉；
  ③ 收敛工时估算被严重低估（431 条规则 × 明暗两态 × 14 页面巡检）。
- **Prevention:** 收敛 Phase 6 把它当**一个原子单元**处理：先把 `AntdThemeBridge` 的
  `darkAlgorithm` token 调成与现有手写暗色**等效**（逐组件比对），再整块删除手写 cluster。
  **不允许**按组件分批删除 —— 因为每个组件的浅色规则也在别处，拆开必然中间态破相。
- **Detection:** 新增静态检查「同一 antd 目标 class 在 ≥2 个暗色 cluster 中出现」计数
  （见 §3.3 C7）；浏览器侧用 `css-contract-assertions.js` 的暗色 fixture 断言
  `innerDark.background` 必须等于暗色底。

### P-M2 · `!important` 总量 1435 处，但**分布极不均衡**——先删长尾，收益/风险比最优

- **What goes wrong:** 「从 1359 降到多少」的直觉做法是从头到尾逐条审 `!important`，
  会把工期拖成不可完成的审计；实际上 **94.7% 集中在 `index.css` 单文件**，
  且其中 620 处是 `box-shadow`、37 处是 `transition`（v1.24 登记册 §5 口径），
  属**跨分区集中现象**、低语义密度。
- **Why it happens:** 项目早期用 `!important` 兜住「antd 与手写层抢同一个属性」，
  于是**每个**声明都加了，包括纯装饰性的 `box-shadow` / `transition`。
- **Consequences:** 若不先做长尾清理，Phase 4-6 每动一个组件都要面对
  「这个 `!important` 是装饰性的还是语义性的」判断，吞吐极低。
- **Prevention:** 按 §3.1 的「**优先清理顺序**」执行：
  1) 死文件（470 行 / 9 important）→ 2) 死选择器（8 条规则）→ 3) 装饰性 `box-shadow`/`transition`
  → 4) 组件语义性 `!important`。前 3 步可在**不触碰任何活规则**的前提下砍掉可观数量。
- **Detection:** 每 phase 结束跑一次全量 `!important` 计数（§3.1 口径命令），
  目标值见 [§ 量化基线](#量化基线)。

### P-M3 · 圆角漂移 6px vs 8px：项目 token 与 antd token 打架，且**视觉上只差 2px 极难肉眼验收**

- **What goes wrong:** `index.css:125` 定义 `--theme-radius-md: 6px`，
  而 `AntdThemeBridge.tsx:89` 设 `borderRadius: 8`。
  项目在 `index.css:877`（`.ant-input`）与 `index.css:912`（`.ant-input-affix-wrapper`）
  用 `var(--theme-radius-md)` = **6px** 覆盖 antd 的 **8px**；
  `index.css:5563` 更是硬写 `border-radius: 6px !important`（分页器）。
  → 同一个输入框，**外层 wrapper 6px、内层 antd 期望 8px**；
  收敛到 variant 后若 antd 侧用 8px、项目侧仍 6px，就是「同组件双圆角」。
- **Why it happens:** v1.22 品牌化时 antd 侧按设计 token 走 8px，
  项目 CSS 侧沿用早期更紧凑的 6px 体系，两套体系在同一属性上并存。
- **Consequences:** ① 收敛期切换瞬间 Input/Select/Button 圆角从 6 跳到 8，
  被误判为「破相」而回滚；② `css-contract-assertions.js` 记录的内层 `radius` 期望值
  也会随之内移，需要同步更新断言基线。
- **Prevention:** 收敛 Phase 3（`AntdThemeBridge` 重构）里**先统一圆角**：
  把 `--theme-radius-md` 改为 8px，或把 antd `borderRadius` 改为 6px；
  **二选一，同一 commit 内完成并全站同步**，避免半程两种圆角并存。
  建议统一为 **8px**（与 antd 默认 + 设计 token 一致，且 `index.css:126` 的
  `--theme-radius-lg` 已经是 8px，体系更自洽）。
- **Detection:** 浏览器断言 `getComputedStyle(wrapper).borderTopLeftRadius`
  与 `getComputedStyle(input).borderTopLeftRadius` **必须相等**（现在应为 0，
  因为内层被 VIS-01 复位为 0 —— 收敛后应改为「wrapper 8px / 内层 0px 且视觉连续」）。

### P-M4 · `InputNumber` 边框锁死：`border` 简写 + `!important` 压掉 antd 6 的 status 色

- **What goes wrong:** `index.css:871-880` 的选择器组里含 `.ant-input-number`，
  其中 `border: 1px solid var(--input-border) !important` 是**简写**，
  一次性重置 `border-width/style/color`。antd 6.6.4 的状态边框规则是
  `.ant-input-number-outlined.ant-input-number-status-error:not(.ant-input-number-disabled)`
  （特异性 0,3,0，**无** `!important`）→ 项目的 `!important` 简写**直接压掉**，
  InputNumber 的错误态边框**永远不会变红**。
- **Why it happens:** antd 6 把 InputNumber 的边框画在**根元素**
  （`.ant-input-number-outlined`，实测 `node_modules/antd/dist/antd.css`：
  `.ant-input-number-outlined{background:…;border-width:…;border-style:…;border-color:…}`），
  与普通 Input 的「边框在 wrapper、内层透明」结构**不同**；
  v1.24 的 VIS-03 修复只覆盖了 `.ant-input*` 域（登记册 #16 已登记为 P1 欠账）。
- **Consequences:** InputNumber 错误态持续是灰边框（v1.24 实测 `rgb(219,215,206)`），
  用户无从判断校验失败；收敛期若把 VIS-03「顺手扩面」到 InputNumber 而不加
  status 专用选择器，会把 antd 的 `colorError` 一并压掉。
- **Prevention:** 收敛 Phase 5 单独处理 InputNumber：
  - **不要**把 `.ant-input-number` 塞进 VIS-03 的 `.ant-input` 选择器组；
  - 改用 antd 6 的 InputNumber 组件 token（`components.InputNumber.colorError` / `activeBorderColor`），
    或写一条同特异性的 status 还原：
    `.ant-input-number.ant-input-number-status-error { border-color: var(--ant-color-error) !important }`
    （特异性 0,2,0 + important，可压过 `:875` 的 0,1,0+important）。
- **Detection:** 浏览器断言 InputNumber（`status="error"`）的
  `borderTopColor` 必须等于 `rgb(186,54,48)`；当前是 `rgb(219,215,206)`（**FAIL 现状**）。

### P-M5 · antd 6 弃用警告 2 类会污染「无新增 console error」的自动化判定

- **What goes wrong:** 收敛期大量新增/改动 antd 组件用法，若引入或保留弃用 prop，
  浏览器控制台会持续打印 warning，导致「本次收敛零新增 console error」的验收条件**永远 FAIL**。
  已实证的两类：
  1. **`maskClosable` → `mask.closable`**：`node_modules/antd/es/modal/Modal.js:126`
     明确列出 `[['maskClosable','mask.closable']]` 的 deprecated 映射；
     `node_modules/antd/es/modal/interface.d.ts:71-74` 标注 `@deprecated Please use \`mask.closable\` instead`。
     项目命中：`src/components/shared/FormModal.tsx:143`、`src/components/modal/BaseEditModal.tsx:61`、
     `src/components/shared/confirmDelete.ts:66`、`src/components/captcha/CaptchaModal.tsx:116`、
     `src/components/network/port-write/BulkWriteDrawer.tsx:241`。
  2. **`onSearch` → `showSearch.onSearch`**：`node_modules/@rc-component/select/es/Select.d.ts:85`
     标注 `@deprecated please use showSearch.onSearch`；antd 6 的 Select
     `showSearch?: boolean | (SearchConfig<OptionType> & {…})`（`antd/es/select/index.d.ts:78`）
     已是**对象形态**。同时 `node_modules/@rc-component/select/es/utils/warningPropsUtil.js:54-55`
     会打印运行时 warning：「`onSearch` should work with `showSearch` instead of use alone.」
     项目命中：`src/components/CronSelector/index.tsx:206`、
     `src/components/dashboard/settings/EndpointSelector.tsx:211`
     （两者都传了 `onSearch`；而 `AppSelect.tsx:18` 的
     `showSearch={showSearch ?? count >= 8}` 在选项 <8 时会传 `false` → 触发该 warning）。
     ⚠️ **口径更正**：任务书写的「`onSearch→showSearch`」在 antd 6.6.4 的 Select
     deprecated 列表（`antd/es/select/index.js:217-225`）中**并不存在**；
     真正的形态是 `showSearch.onSearch`（rc-select 的对象化 API）+ 上述运行时 warning。
- **Why it happens:** 弃用警告由 antd 在 dev 构建的 `devUseWarning` 中打印，
  与业务错误同级出现在 console；收敛期若同时改了 `FormModal` 等共享组件的 props，
  警告会**在每个页面复现**。
- **Consequences:** 自动化判定「无新增 console error」被污染 → 误判为回归；
  或为消除警告而**乱改共享组件 props**，破坏 v1.22 冻结契约（见 P-M7）。
- **Prevention:** 收敛 Phase 0 就把这两类弃用**单独清理**（改 `mask={{ closable: false }}`、
  改 `showSearch={{ onSearch }}`），**与样式收敛分离成独立 commit**；
  这样后续 phase 的 console 判定基线是干净的。
  注意 `FormModal` / `BaseEditModal` 的 `maskClosable` **默认值本身就是 `false`**
  （`FormModal.tsx:78`、`BaseEditModal.tsx:46`）→ 迁移是**纯等价**改动，风险低。
- **Detection:** 浏览器 Console 过滤 `Warning:` 前缀，断言 0 条；
  另加 `grep -rn 'maskClosable' src` 与 `grep -rn 'onSearch=' src` 的静态零命中断言。

### P-M6 · `.dept-tree-search` 之外的另外两条「局部补丁」是**活规则**，不能跟死规则一起删

- **What goes wrong:** v1.24 登记册 #26 把 `index.css:2051-2088` 的 2 条局部补丁
  一律标为「保留（陷阱）」。实测其中 2 条**是活的**、2 条**是死的**：
  - **活**：`index.css:2119-2123` `.user-form-input.ant-input-affix-wrapper .ant-input`
    —— 复合选择器要求同一元素同时具备 `user-form-input` + `ant-input-affix-wrapper`，
    而 antd 6 把 `className` 放在**根元素**（affix wrapper）上 → **命中**；
  - **活**：`index.css:2125-2130` 同族的 focus 变体；
  - **死**：`index.css:2093-2116` `.dept-tree-search …`（见 P-C5）。
  若把 4 条一起删 → 部门树搜索框**丢失透明底/无边框**样式（回到 antd 描边输入框），
  登录/部门树两处外观回归。
- **Why it happens:** 同一个 `className="dept-tree-search user-form-input"`
  同时挂了两组补丁，其中一组用了**后代选择器**（antd 5 时代 class 在内层 `<input>` 时成立），
  另一组用了**复合选择器**（antd 6 时代 class 在根元素时成立）——
  迁移期两套写法并存，只有其中一套在 antd 6 下活着。
- **Consequences:** 误删活规则 → 部门树搜索框外观回归（用户在登录后第一屏可见）。
- **Prevention:** Phase 1 删除前**逐条**用
  `document.querySelectorAll('<选择器>').length` 在浏览器取证；
  只有命中数为 0 的才进删除清单，并同步更新 `css-contract-check.js` 的 C3b/C3c 断言方向。
- **Detection:** `css-contract-check.js` C3c（`.user-form-input.ant-input-affix-wrapper .ant-input`）
  保持为正向断言；新增 C3b 反向断言（死规则不得存在）。

### P-M7 · v1.22 冻结契约在样式收敛中的间接破坏面

- **What goes wrong:** 收敛会触碰 `AppInput` / `AppSelect` / `AppTextArea` 的默认 props
  （`allowClear` / `showSearch`）以适配 antd 6 variant；若在收敛中「顺手」改了默认 props，
  就会破坏 v1.22 冻结契约（`eslint.config.js:132-196` 的 `no-restricted-syntax` 强制
  裸 antd Input/Select/Modal.confirm/Popconfirm 一律禁止，全站必须走封装层）。
- **Why it happens:** 收敛 Phase 4/5 要判断「输入框到底要不要 `variant` 显式传」，
  而 `AppInput` 是**全站唯一入口** —— 改一个默认值 = 改全站 309 个输入框。
  且 `AppInput.tsx:12` 的 `allowClear = true` 默认值**直接决定 DOM 是否生成 affix wrapper**
  （`hasPrefixSuffix` 含 `allowClear`）→ **改默认值会让 VIS-01 复位规则的匹配面整体消失**
  （`.ant-input-affix-wrapper > input.ant-input` 无 wrapper 可选）。
- **Consequences:** ① 契约失效（eslint 豁免链 `scripts/exemption-reasons.json` 失配）；
  ② 双框回归（affix wrapper 消失后，内层 input 成为 carrier，项目 `:875` 的
  `.ant-input { border !important }` 立刻画出边框）；
  ③ 焦点环从 wrapper 迁移到 input，与 v1.24「焦点环归 wrapper」的 VIS-02 契约冲突。
- **Prevention:** 收敛期间 **冻结 `AppInput`/`AppSelect`/`AppTextArea` 的默认值**；
  确需变更时（例如 antd 6 variant 需要显式 `variant` prop），
  应通过 **`<ConfigProvider>` 的 `variant` 全局配置**或 `components.Input.variant` token 实现，
  **不要**在封装层逐个传 prop。
- **Detection:** `node scripts/check-exemption-reasons.mjs`（退出码 0）；
  `node scripts/css-contract-check.js`；`npm run lint`；
  `src/components/ui/__tests__/` 下契约测试全绿。

### P-M8 · 双 ConfigProvider + `darkAlgorithm` 不改写显式 token → 暗色 Select 仍是白底

- **What goes wrong:** `AntdThemeBridge.tsx:83-87` 显式设置
  `colorBgContainer: ruoyiBrand.cream.surface`（`#FFFFFF`）、
  `colorBgElevated: ruoyiBrand.cream.surface`、`colorBorder: ruoyiBrand.cream.border`（`#DBD7CE`）、
  `colorBorderSecondary: ruoyiBrand.cream.borderStrong`。
  antd 的 `darkAlgorithm` **只改写它自己派生的 token**，对**显式提供**的 token 值原样保留
  （`ruoyiBrand.cream.*` 是常量对象，与主题模式无关）。
  → 暗色下 `colorBgContainer` 仍是 `#FFFFFF`、`colorBorder` 仍是 `#DBD7CE`。
  加上 `AntdThemeBridge.tsx:130` 的 `Select.colorBgContainer` 同样写死 `#FFFFFF`，
  以及项目侧 `.ant-select-selector` 死规则导致**没有任何项目规则接管 Select 边框**
  → 暗色 Select 白底浅灰边。
- **Why it happens:** token 覆盖发生在 `algorithm` 之后应用（`ThemeConfig.token` 是最终值），
  而 `ruoyiBrand` 是**模式无关**的常量 —— 把「品牌常量」当「主题 token」用是根因。
- **Consequences:** 暗色 Select / InputNumber / 部分 Card 在深绿页面上呈白块，
  对比度不达标；这是 v1.25 F3（明暗色板重构）必须**先于**样式收敛解决的根因。
- **Prevention:** 收敛 Phase 3 必须把 `AntdThemeBridge` 的 `token` 改成
  **按 `themeMode` 分支的函数**（`themeMode === 'dark' ? darkTokens : lightTokens`），
  即「暗色不再复用 `ruoyiBrand.cream.*`」。这是 P-M1 暗色 cluster 收敛的**硬前置**。
- **Detection:** 浏览器断言暗色下
  `getComputedStyle(document.querySelector('.ant-select')).backgroundColor`
  ≠ `rgb(255,255,255)`；并对 `AntdThemeBridge` 增加「dark 模式不返回 cream.* 常量」的单元测试。

### P-M9 · `!important` 治理与「hex 治理」是两件事，收敛期会引入新的硬编码色

- **What goes wrong:** 收敛期把 CSS 规则改写成 token 引用时，开发者容易**临时**写死一个
  hex「先把效果调对」。而 hex 门禁对 `.css` 几乎失明（P-F05 / §0.2 第 5 条），
  这些临时值会**静默沉淀**为新的技术债，且比原来的规则更难发现。
- **Why it happens:** `eslint.config.js:132` 的 `no-restricted-syntax` 块
  `files: ["src/**/*.{ts,tsx}"]` —— **不含 `.css`**；
  `scripts/check-hardcoded-colors.mjs` 虽扫 `.css`，但只有 **31 条**固定 hex 映射，
  `index.css` 中 311 处 6 位 hex 只有 2 处命中（`#ff4d4f`@964、`#faad14`@970），
  即 **99.4% 不在 denylist**；且该脚本**当前 exit=1**、**未接入 `lint:ci`**。
- **Consequences:** 收敛完成后 `!important` 降了一半，但硬编码 hex 反而增加，
  「样式层收敛」的实际成果被稀释；且这些新 hex 没有 antd token 对应物，
  **暗色模式无法自动跟随** → 制造新的串色。
- **Prevention:** 收敛 Phase 0 扩展门禁：
  1) 把 `scripts/check-hardcoded-colors.mjs` 接入 `lint:ci`（先修它当前的 18 个失败文件）；
  2) 为 `.css` 增加**通用** hex 检测（`#rrggbb` 全量，不只是 31 条映射），
     允许清单用 `scripts/exemption-reasons.json` 同款「文件+理由」机制豁免；
  3) 约定收敛期新增的 CSS 声明**一律用 `var(--theme-*)` 或 `var(--ant-*)`**。
- **Detection:** `node scripts/check-hardcoded-colors.mjs`（exit 0）；
  新增 `.css` 通用 hex 计数基线（§3.1）。

### P-M10 · 静态契约脚本的能力边界：`css-contract-check.js` 只查「文本存在性」，不查「DOM 命中数」

- **What goes wrong:** `scripts/css-contract-check.js` 的 C1/C3/C4/C5 全部是
  **对 CSS 文本做正则匹配**（`indexCss.match(...)` / `indexCss.includes(...)`），
  **不解析 DOM、不计算特异性、不验证选择器是否命中**。
  → 一个规则**文本在**就 PASS，哪怕它是死规则（P-C5 的 C3b 就是活例子：
  它正在守护一条命中数 0 的死规则）；反过来，一个规则被改成无效语法也可能 PASS。
  `scripts/css-contract-assertions.js` 是**浏览器 Console 粘贴片段**，
  依赖 `document.getComputedStyle`，且**无法在 CI 自动跑**（jsdom 无法解析 antd 6 的 `@layer` 嵌套）。
- **Why it happens:** 契约脚本是在 v1.24「双框修复」这一具体场景下写的，
  目标是**防止已修好的东西被改坏**，不是**校验 CSS 语义正确性**。
- **Consequences:** 收敛期若依赖它判断「这条规则能不能删」，会得到假阳性；
  且新增的 variant 收敛规则**完全没有契约守护** —— 收敛成果不可回归验证。
- **Prevention:** 收敛 Phase 0 **扩展契约脚本**（见 §3.3）：
  增加「死选择器不得存在」「CSS 变量必须有定义」「同一 antd 目标不得有 ≥2 处暗色定义」
  三类静态断言；并在 `css-contract-assertions.js` 中**修正已有的错误 fixture**
  （见 P-M11）。
- **Detection:** `node scripts/css-contract-check.js` 退出码 0 即全部静态断言通过。

### P-M11 · `css-contract-assertions.js` 的错误态 fixture 用了 antd 6 **不存在**的 class → 该断言永远 FAIL

- **What goes wrong:** `scripts/css-contract-assertions.js:48` 构造
  `ant-input-affix-wrapper ant-input-outlined ant-input-affix-wrapper-status-error`，
  而 `css-contract-check.js:79-81`（C5b）**明令禁止**出现
  `ant-input-affix-wrapper-status-error`（该 class 在 antd 6.6.4 中不存在）。
  → 动态断言的 `wrapperError.borderTopColor` 永远拿不到错误态红框，
  `passed` 条件（`:69` 要求 `/255,\s*77,\s*79/`）**恒为 false**。
- **Why it happens:** C5b 是在 v1.24 Phase 77 才修正的（从 `affix-wrapper-status-error`
  改成 `ant-input-status-error`），但 `css-contract-assertions.js` 的 fixture **没有同步更新**。
  实测 antd 6 把 status class 打在 wrapper **自身**，名为 `ant-input-status-error`
  （`node_modules/antd/es/input/Input.js:161` → `getStatusClassNames(prefixCls, mergedStatus)`）。
- **Consequences:** 动态断言脚本**不可用**（永远报 `passed: false`），
  团队可能因此误以为「错误态断言失败 = 有回归」，
  或干脆不再运行它 —— **VIS-03 的回归守护实际处于失守状态**。
- **Prevention:** 收敛 Phase 0 修正 fixture：
  `el('span', 'ant-input-affix-wrapper ant-input-outlined ant-input-status-error', light)`
  并把 `passed` 判定的期望色从 `rgb(255,77,79)` 改为**品牌红 `rgb(186,54,48)`**
  （实测 antd `colorError` = `ruoyiBrand.functional.danger` = `#BA3630`；
  v1.24 实测普通 Input 错误态边框 = `rgb(186,54,48)`）。
- **Detection:** 浏览器粘贴运行 `css-contract-assertions.js`，断言 `passed === true`。

### P-M12 · 阶段拆分风险：把「删规则」与「改 token」拆到不同 phase 会制造**双份真相**

- **What goes wrong:** 若 Phase A 先删一批手写规则、Phase B 再改 `AntdThemeBridge`，
  中间态里「antd 默认样式」和「项目剩余覆盖」会短暂共存，
  用户看到的是**既不是 v1.24 的样子、也不是收敛后的样子**。
  典型表现：删掉 `.ant-input` 边框覆盖后、antd token 还没调 → 输入框边框从
  `#DBD7CE` 变成 antd 默认 `#D9D9D9`（`--ant-color-border`），圆角从 6 变 8，颜色略偏 —— **极难肉眼判定为回归**。
- **Why it happens:** 收敛天然是「双侧同时移动」的动作，但项目管理习惯是「一个 phase 一个可验收产物」。
- **Consequences:** 中间态截图对比会得到「轻微差异」的模糊结论，
  验收者倾向于「再改一点」，最终回到原点。
- **Prevention:** 每个 phase 必须**自带完整可验收态**：
  「改 token + 删对应覆盖」放在**同一个 commit**；
  phase 边界只在**组件域**上切（Input 域一个 phase、Select 域一个 phase），
  绝不在**同一组件域内部**切两次。
- **Detection:** 每个 phase 结束时跑 §3.1 的全部度量，
  并把 phase 末截图与 v1.24 视觉验证记录（`.planning/notes/v1.24-visual-verification.md` 的
  21 形态 × 明暗矩阵）逐格比对。

---

## Minor Pitfalls

### P-m1 · 8 条 `[data-color_mode="dark"]` 拼写错误的死规则（`index.css:3523-3545`, `3602-3608`）

- **What goes wrong:** 选择器写成了 `[data-color_mode="dark"]`（下划线而非连字符），
  与项目实际使用的 `[data-color-mode="dark"]` 不匹配 → **永不命中**。
  影响的是 Anchor 组件（6 条，3523/3527/3531/3535/3539/3543）与 Popconfirm 按钮（2 条，3602/3606）。
- **Why it happens:** 复制粘贴时把连字符写成下划线，且因为这些规则**只影响暗色**
  （浅色下无对应规则），明色下看不出异常，暗色下 Anchor/Popconfirm 缺样式也**不显眼**。
- **Consequences:** 暗色 Anchor 链接色 / Popconfirm 按钮色退化为 antd 默认。
  本身无害，但**污染暗色规则计数**，让「暗色重复分区」的真实数量被高估。
- **Prevention:** 收敛 Phase 1 直接**删除**这 8 条（它们从无效果）；
  若将来要恢复 Anchor/Popconfirm 暗色样式，应走 token 而非补拼写。
- **Detection:** 静态断言「`.css` 中不得出现 `data-color_mode`」。

### P-m2 · 7 个未 import 的 CSS 死文件（470 行 / 9 `!important`）+ 1 个未 import 的 `.less`（85 行）+ 1 个 0 字节 `.less`

- **What goes wrong:** `src/design-system/themes/theme-styles.css`（97 行 / 7 important）
  内含 `.ant-menu-item-selected` / `.ant-tabs-ink-bar` / `.layout-sidebar` / `.layout-header`
  等**与 `index.css` 平行重复**的手写主题规则，并带有**旧 Tailwind 灰蓝 fallback**
  （`#e5e7eb` / `#1f2937` / `#6b7280` / `#3b82f6`）。它**从未被 import**，
  → 死代码 + 误导性「第二套主题层」假象。
  其余 6 个：`src/pages/dashboard-system/index.css`（129 行）、
  `src/components/dashboard/widgets/types/{ChartWidget,ListWidget,MetricWidget,ProgressWidget,StatCardWidget}.css`
  （23+77+24+53+67 = 244 行），同样从未被 import。
  `src/components/cad-editor/CADFloorPlanEditor.less`（85 行）也未 import；
  `src/components/shared/FloorPlanEditor.less` 虽被 import（`FloorPlanEditor.tsx:18`）但**0 字节**。
- **Why it happens:** 组件重构后样式文件被遗留；`theme-styles.css` 的注释（`index.css:4721-4722`）
  说明其规则**已被合并进 `index.css`**（「Theme Component Overrides (merged from theme-styles.css)」）
  → 文件本身是历史残留，但**未被删除**。
- **Consequences:** ① 收敛者可能误读 `theme-styles.css` 当作「要收敛的第二套主题层」，
  浪费工时或做出错误改动；② `check-hardcoded-colors.mjs` 扫到这些死文件里的旧色值并报错
  （它是当前 18 个失败文件之一）；③ 死文件里的 `!important` 计入基线，虚高。
- **Prevention:** 收敛 Phase 1 直接删除 7 个 CSS 死文件 + 1 个死 `.less`；
  `FloorPlanEditor.less`（0 字节）删除并同时移除 `FloorPlanEditor.tsx:18` 的 import。
  **删除前先确认无引用**（已用 `grep -rn "<basename>" src` 全量确认）。
- **Detection:** 新增静态检查「`.css`/`.less` 文件必须被至少一个 `.tsx/.ts` import」
  （见 §3.3 C8）。

### P-m3 · `index.css:4806` 与 `1879/1885/1891` 的 `--sidebar-width` 双定义（布局漂移，非色串）

- **What goes wrong:** `index.css:4806-4808` 的
  `[data-layout="classic"], [data-layout="hybrid"] { --sidebar-width: 240px; }`
  与 `index.css:1879`（classic → 200px）、`1885`（hybrid → 200px）
  特异性相同 (0,1,0)，后者在**前** → **240px 胜出**，
  而 `index.css:1891` 的 `[data-layout="innovative"]`（80px）不受影响。
  这是**布局维度**的重复真相源（不是颜色串色），但同属「同一变量多处就地重定义」的模式。
- **Why it happens:** 侧栏宽度在「原型实测 240px」时于文件尾部补写，忘了收敛前面那组。
- **Consequences:** 侧栏宽度比设计值宽 40px；在窄屏下挤压主区。
- **Prevention:** 收敛 Phase 2 把 `--sidebar-width` 的三处定义**合并到一处**
  （保留 `[data-layout=...]` 家族，删掉尾部那处），或把尾部那处改为
  `--sidebar-width: var(--sidebar-width-prototype)` 之类明确的单一来源。
- **Detection:** 静态断言「同一 CSS 变量在多个选择器块中重复赋值」计数归零
  （见 §3.3 C9，注意要给 `--input-bg` 这类**有意的明暗双写**留白名单）。

### P-m4 · `index.css:5563` 硬写 `border-radius: 6px !important`（分页器）与 `:5577` 的 `#ffffff`

- **What goes wrong:** 同一区块内既有 token 引用（`var(--theme-bg-primary)`、`var(--theme-text-primary)`）
  又有硬编码（`6px`、`#ffffff`）→ 圆角漂移（P-M3）与 hex 治理盲区（P-M9）在此叠加。
- **Why it happens:** 分页器区块（`index.css:5556-5588`）是「原型 demo 复刻」，
  按像素抄了一份，抄的时候没走 token。
- **Consequences:** 收敛到 token 后分页器圆角会从 6px 跳到 8px，需在验收矩阵里单列一格。
- **Prevention:** Phase 2 改为 `var(--theme-radius-lg)`（8px）与 `var(--theme-text-inverse)`；
  圆角变化需在验收记录里显式声明为「有意变更」。
- **Detection:** §3.1 的「`.css` 内 hex 字面量计数」应从 311 降至个位数量级。

### P-m5 · `login.css` 的登录页补丁是**硬编码琥珀金体系**，与品牌绿体系并行

- **What goes wrong:** `src/pages/login/login.css:340-370` 整块使用
  `#e7e5e4`（边框）、`#fafaf9`（底）、`#d4a574`（悬停/聚焦边框 + 3px 焦点环）
  —— 这是一套**登录页专属的琥珀金体系**，与全站品牌绿（`#156031`）不同源，
  且**完全不受任何 hex 门禁约束**。收敛时若「统一到品牌绿」，登录页视觉会明显改变；
  若不统一，它就继续是 token 化收容之外的孤岛。
- **Why it happens:** 登录页是「原型复刻」页，色值来自原型截图而非 `brand-spec.md`。
  另注：`#fafaf9` 与 `index.css:4815` 的 `--input-bg: #fafaf9` **同值** —— 说明
  `4815` 那处 `:root` 重定义是**从登录页复制过来的**，这解释了它为什么被误放进全局 `:root`。
- **Consequences:** 收敛范围蔓延到登录页 → 工期与风险显著上升。
- **Prevention:** 明确**边界**：v1.25 ② 收敛**不改登录页视觉**；
  但允许把其硬编码色**提取为 CSS 变量**（如 `--login-input-border` / `--login-accent`），
  值保持不变 —— 治理收益（可追溯）拿到，视觉风险为零。
  同时修正 `index.css:4815` 的来源（见 P-C3）。
- **Detection:** 登录页 3 个形态（明/暗 × 聚焦/错误）× 焦点环颜色截图比对，
  要求与 v1.24 视觉记录逐像素一致。

### P-m6 · `styles.module.css` 的 CSS Module 与全局 `index.css` 的**层级竞争**

- **What goes wrong:** `src/pages/operations/building-spaces/components/styles.module.css`（370 行）
  使用 CSS Module（局部类名），但它**同时**含 6 条 antd 选择器规则
  （Modal 域）—— CSS Module 只对**本地类名**做作用域化，**antd 的全局类名不会被改写**，
  → 这 6 条规则实际是**全局规则**，会泄漏到全站 Modal。
- **Why it happens:** 开发者以为放进 `.module.css` 就自动局部化。
- **Consequences:** 楼宇/工位页面的 Modal 样式会覆盖全站 Modal 样式（收敛后表现会变化）。
- **Prevention:** 收敛 Phase 5（Modal 域）**必须**把 `styles.module.css` 中这 6 条
  antd 选择器**迁出**到 `index.css` 的 Modal 域统一治理，或改用 `antd` 的 `classNames`/`styles` 语义化 API。
- **Detection:** 静态断言「`.module.css` 内不得出现 `.ant-` 选择器」（见 §3.3 C10）。

### P-m7 · `!important` 计数的 6 处口径差（1435 vs 1429 vs 1359）

- **What goes wrong:** 报告/验收用不同的 grep 口径会得到不同数字（1359 / 1429 / 1435），
  导致「到底降了多少」无法比对。
- **Why it happens:** ① `grep -o '!important' | wc -l` = **1435**（全 `.css`，按出现次数）；
  ② 按规则块统计 = **1429**（含死文件里的 9 处）；③ 仅 `index.css` = **1359**；
  ④ 解析器按规则体统计 = 1353（差 6 处是落在注释/字符串里的 `!important` 文本）。
- **Consequences:** 验收口径不统一 → 争议。
- **Prevention:** **在 Phase 0 固定一个口径**并在所有报告中沿用：
  **「`find src -name '*.css' -exec cat {} + | grep -o '!important' | wc -l` = 全量出现次数」**，
  同时**并列报告**「`index.css` 单文件数」。本文件 §3.1 采用该口径。
- **Detection:** 把 §3.1 的命令写进 CI，每次 phase 记录快照。

### P-m8 · antd 6 `variant` 的可用取值与作用域（收敛的技术前提）

- **What goes wrong:** 误以为 `variant` 只能逐组件传 prop，导致要给 309 个输入框逐个加 `variant`。
- **Why it happens:** 不了解 antd 6 的 `useVariants` 解析顺序。
- **Consequences:** 大量样板代码；或为了少改而继续手写覆盖。
- **Prevention:** 掌握 antd 6 的 `useVariants`（`node_modules/antd/es/form/hooks/useVariants.js`）解析顺序：
  **组件 `variant` prop → `legacyBordered === false` → `Form` 上下文 `VariantContext`
  → `components.<Comp>.variant` → 全局 `config.variant` → 默认 `'outlined'`**。
  → **最优解**：在 `AntdThemeBridge` 的 `ConfigProvider theme` 里设置
  `components.Input.variant` / `components.Select.variant` / 全局 `variant`，
  **零样板**。可用取值为 `Variants` 数组（`'outlined' | 'filled' | 'borderless'`）。
- **Detection:** 浏览器断言 `document.querySelector('.ant-input').className`
  含 `ant-input-outlined`。

## 四批规则清单

> 口径：文件:行号 全部来自当前 HEAD 实跑（`index.css` = 5632 行）。
> 每条都给出**删除/降级前必须验证的无害性证据**。
> 置信度：第 1 批 **HIGH**（死类名/死文件有源码或 grep 实证）；第 2 批 **HIGH**（取值实跑）；
> 第 3 批 **HIGH**（契约脚本 + 注释实证）；第 4 批 **MEDIUM**（重写方向为推导，需实机验证）。

### 第 1 批｜可整块删除（17 条）

| # | 文件:行号 | 规则原文摘要 | 删除理由 | 删除前必须验证的无害性证据 | 置信度 |
|---|---|---|---|---|---|
| D1 | `index.css:2093-2096` | `.dept-tree-search .ant-input-affix-wrapper { border:none!important; background:transparent!important }` | **后代选择器**，class 落在 wrapper 自身 → 命中数 0 | ① `node_modules/@rc-component/input/es/BaseInput.js:129-131` 明示 className 在 root；② `hasPrefixSuffix` 含 `allowClear` → root = wrapper；③ 浏览器 `document.querySelectorAll('.dept-tree-search .ant-input-affix-wrapper').length === 0`；④ 同元素的 `.user-form-input.ant-input-affix-wrapper .ant-input` 命中数 > 0（活规则顶上） | HIGH |
| D2 | `index.css:2098-2102` | `.dept-tree-search .ant-input-affix-wrapper .ant-input { border/background/box-shadow:none!important }` | 同 D1，命中数 0 | 同 D1 ③（`.dept-tree-search .ant-input-affix-wrapper .ant-input` 命中数 === 0） | HIGH |
| D3 | `index.css:2104-2109` | `.dept-tree-search .ant-input-affix-wrapper:focus, ...-focused { border/background/box-shadow:none!important }` | 同 D1，命中数 0 | 同 D1 ③ | HIGH |
| D4 | `index.css:2111-2116` | `.dept-tree-search ...:focus .ant-input, ...-focused .ant-input { … }` | 同 D1，命中数 0 | 同 D1 ③ | HIGH |
| D5 | `index.css:3523,3527,3531,3535,3539,3543`（6 条规则） | `[data-color_mode="dark"] .ant-anchor` / `-wrapper` / `-ink::before` / `-link` / `-link:hover` / `-link-active` | **拼写错误**（下划线），与实际 `[data-color-mode="dark"]` 不匹配 → 命中数 0 | `grep -c 'data-color_mode' src/index.css` = 8；项目全站仅用 `data-color-mode`（`themeStore` / `AntdThemeBridge` / `settingsStore` 统一） | HIGH |
| D6 | `index.css:3602-3604, 3606-3608`（2 条规则） | `[data-color_mode="dark"] .ant-popconfirm .ant-popconfirm-buttons .ant-btn` / `.ant-btn-primary` | 同 D5，命中数 0 | 同 D5 | HIGH |
| D7 | `index.css:873`（在 871-880 规则组内） | `.ant-input, .ant-input-number, .ant-select-selector { background/border/color/border-radius/font-size/transition !important }` 中的 `.ant-select-selector` 分支 | antd 6.6.4 的 Select **不渲染** `.ant-select-selector` 元素 | `grep -r 'select-selector' node_modules/@rc-component/select/es/` **零命中**；`node_modules/antd/dist/antd.css` 中唯一命中是 ColorPicker 内部后代选择器 `.ant-color-picker … .ant-select-selector`；浏览器 `document.querySelectorAll('.ant-select-selector').length === 0` | HIGH |
| D8 | `index.css:884` | `.ant-select:hover .ant-select-selector { border-color !important }` | 同 D7（死后代） | 同 D7 | HIGH |
| D9 | `index.css:890` | `.ant-select-focused .ant-select-selector { border-color/box-shadow !important }` | 同 D7 | 同 D7 | HIGH |
| D10 | `index.css:897`（在 895-901 规则内） | `[data-color-mode="dark"] .ant-select-focused .ant-select-selector { box-shadow !important }` | 同 D7 | 同 D7 | HIGH |
| D11 | `index.css:2329-2333` | `[data-color-mode="dark"] .ant-select-selector { background-color/color/border-color !important }` | 同 D7 | 同 D7 | HIGH |
| D12 | `index.css:3962-3966` | `[data-color-mode="dark"] .ant-select-selector { background/border/color !important }`（暗色二块） | 同 D7 | 同 D7 | HIGH |
| D13 | `index.css:3968-3970` | `[data-color-mode="dark"] .ant-select:hover .ant-select-selector { … }` | 同 D7 | 同 D7 | HIGH |
| D14 | `index.css:3972-3977` | `[data-color-mode="dark"] .ant-select-focused .ant-select-selector { border-color/box-shadow !important }` | 同 D7 | 同 D7 | HIGH |
| D15 | `index.css:5579-5581` | `.ant-pagination .ant-pagination-options .ant-select-selector { font-family: var(--theme-font-mono) }` | 同 D7（且无 `!important`，纯装饰） | 同 D7 | HIGH |
| D16 | `src/design-system/themes/theme-styles.css`（97 行 / 7 `!important`） | `.ant-menu-item-selected` / `.ant-tabs-ink-bar` / `.layout-sidebar` / `.layout-header` / `.layout-tab-bar` / `.tab-scroll-button` 手写主题层 + 旧灰蓝 fallback | **从未被 import**（`grep -rn 'theme-styles' src` 仅命中 `index.css` 的注释） | ① `grep -rn "theme-styles" src --include=*.tsx --include=*.ts` = 0 命中；② `index.css:4721-4722` 注释明写其规则已 merged 进 `index.css` | HIGH |
| D17 | `src/pages/dashboard-system/index.css`（129 行）<br>`src/components/dashboard/widgets/types/{ChartWidget,ListWidget,MetricWidget,ProgressWidget,StatCardWidget}.css`（244 行 / 2 `!important`）<br>`src/components/cad-editor/CADFloorPlanEditor.less`（85 行）<br>`src/components/shared/FloorPlanEditor.less`（0 字节） | 组件样式残留 | **全部从未被 import**；`FloorPlanEditor.less` 虽被 import 但为空文件 | ① 逐文件 `grep -rn "<basename>" src --include=*.tsx --include=*.ts` = 0 命中；② `wc -c FloorPlanEditor.less` = 0；③ `index.css` / `BaseWidget.css` 已覆盖其全部视觉 | HIGH |

> **D1-D4 的连带动作**：删除后必须同步修改 `scripts/css-contract-check.js:57` 的
> **C3b** 断言（当前它在守护这条死规则）→ 改为**反向断言**「`.dept-tree-search .ant-input-affix-wrapper` 不得存在」，
> 同时保留 **C3c**（`.user-form-input.ant-input-affix-wrapper .ant-input`，**活规则**）为正向断言。

### 第 2 批｜可降级为 token 引用（14 条）

| # | 文件:行号 | 现状（硬编码） | 建议替换为 | 降级理由 | 验证方式 | 置信度 |
|---|---|---|---|---|---|---|
| T1 | `index.css:877` | `border-radius: var(--theme-radius-md) !important` → **6px** | `var(--theme-radius-lg)`（**8px**）或改为 antd `borderRadius` token | 项目 token 6px 与 antd `AntdThemeBridge.tsx:89` 的 `borderRadius: 8` 冲突 | `getComputedStyle(input).borderTopLeftRadius` 单值断言（当前 6px） | HIGH |
| T2 | `index.css:912` | 同 T1（wrapper） | 同 T1 | 同 T1 | wrapper radius 断言 | HIGH |
| T3 | `index.css:5563` | `border-radius: 6px !important`（分页器） | `var(--theme-radius-lg)` | 裸 px，绕过 token 体系 | 分页按钮 radius 断言 | HIGH |
| T4 | `index.css:878` | `font-size: 14px !important` | `inherit`（antd 内层已是 `inherit`）/ antd `fontSize` token | 锁死字号 → `size="large"` 不放大（登记册 #5/#6） | `<Input size="large">` 的 `fontSize` 断言 | HIGH |
| T5 | `index.css:879` | `transition: all var(--theme-transition-fast) !important` | antd motion token / `transition: border-color .2s` | `transition: all` 触发全属性过渡（登记册 #5） | DevTools 观察过渡属性数 | HIGH |
| T6 | `index.css:964` | `var(--ant-color-error, #ff4d4f)` | `var(--ant-color-error)`（去掉 fallback） | antd 6 默认 `cssVar` 已注入 `--ant-color-error:#ff4d4f`（`node_modules/antd/dist/antd.css` 实测）→ fallback 永不生效，是**死色** | 浏览器 `getComputedStyle(el).getPropertyValue('--ant-color-error')` 非空 | HIGH |
| T7 | `index.css:970` | `var(--ant-color-warning, #faad14)` | `var(--ant-color-warning)` | 同 T6 | 同 T6 | HIGH |
| T8 | `index.css:5577` | `color: #ffffff !important`（分页激活页码文字） | `var(--theme-text-inverse)` | `#ffffff` 是 `--theme-text-inverse` 的字面值 | 暗色下断言文字色 | HIGH |
| T9 | `login.css:345` | `border: 1px solid #e7e5e4 !important` | `var(--login-input-border)`（新增变量，**值不变**） | 登录页专用色板需可追溯，但**不并入品牌绿**（避免扩大范围，见 P-m5） | 登录页截图逐像素比对 | HIGH |
| T10 | `login.css:346` | `background: #fafaf9 !important` | `var(--login-input-bg)` | 同 T9；**注意该值与 `index.css:4815` 同源** → 修 P-C3 时不要把登录页色误引入全局 `:root` | 同 T9 | HIGH |
| T11 | `login.css:361, 367` | `border-color: #d4a574 !important`（hover / focus） | `var(--login-accent)` | 同 T9 | 同 T9 | HIGH |
| T12 | `login.css:362, 368` | `background-color:#ffffff` / `rgba(212,165,116,0.15)` | `var(--login-input-bg)` / `var(--login-focus-ring)` | 同 T9；`rgba` 也是 hex 治理盲区 | 同 T9 | HIGH |
| T13 | `ActionButtons.css:14,18,23,28,32,40`（6 条） | `#262626` / `#000000` / `#ba3630` / `#cf1322` | `var(--theme-text-primary)` / `var(--theme-text-inverse)` / `var(--theme-error)` / `var(--theme-error-hover)` | 全部是 `ruoyiBrand` 已有值的字面复制 | 表格操作列明暗两态截图 | HIGH |
| T14 | `index.css:4814` | `--theme-zebra: #faf8f3`（在**后置 `:root`**） | 并入 `--table-header-bg` / `--theme-bg-tertiary`（`:root` 已有同义变量） | 第 3 处同名变量体系的雏形；与 D16 同属「原型复刻值渗入全局 `:root`」 | 静态断言「`:root` 不得新增颜色变量（白名单外）」 | MEDIUM |

### 第 3 批｜必须保留并登记（11 条）

| # | 文件:行号 | 规则摘要 | 必须保留的理由 | 登记位置 |
|---|---|---|---|---|
| K1 | `index.css:937-945` | VIS-01 内层复位（6 项 `!important`，特异性 (0,3,1)） | antd 6 内层 `border:none` 特异性 (0,1,1) 压不过项目 `:875` 的 `!important`；在项目层覆盖被删之前它**必须存在**，否则双框回归 | `css-contract-check.js` **C1a-C1c**（已守护）；建议升级为「Phase 4 后重新评估」条目 |
| K2 | `index.css:958-965` | VIS-03 `status-error` 边框色还原（`var(--ant-color-error) !important`） | 错误态红框是 v1.24 已交付验收项；antd 的 status 色是普通声明，会被 `border` 简写 `!important` 压掉 | `css-contract-check.js` **C5a** |
| K3 | `index.css:967-971` | `status-warning` 边框色还原 | 同 K2 | `css-contract-check.js` **C5c** |
| K4 | `index.css:888-893` | `.ant-input:focus` 焦点环 `0 0 0 2px var(--theme-primary-lighter) !important` | 焦点环唯一来源之一 | `css-contract-check.js` **C4a** |
| K5 | `index.css:920-923` | `.ant-input-affix-wrapper-focused` 焦点环 | 焦点环**归 wrapper**（VIS-02 契约） | `css-contract-check.js` **C4c** |
| K6 | `index.css:895-901` | 暗色 `.ant-input:focus` 双层焦点环（2px + 12px glow） | 暗色焦点环需更强可见度 | v1.24 视觉记录 |
| K7 | `index.css:925-929` | 暗色 `.ant-input-affix-wrapper-focused` 焦点环 | 同 K6 | v1.24 视觉记录 |
| K8 | `index.css:2119-2123` | `.user-form-input.ant-input-affix-wrapper .ant-input { border:none!important; background:transparent!important; box-shadow:none!important }` | **活规则**（复合选择器命中 antd 6 的 root=wrapper），删则部门树搜索框外观回归 | `css-contract-check.js` **C3c** |
| K9 | `index.css:2125-2130` | `.user-form-input…:focus .ant-input, …-focused .ant-input` | 同 K8 | 需**新增**正向断言（当前 C3 只覆盖非 focus 变体） |
| K10 | `login.css:351-358` | `.login-form .ant-input-affix-wrapper > input.ant-input { font-size/background/border/box-shadow/padding/height }` | 登录页内层复位；误删 → 登录页双框回归（登记册 #25「陷阱」） | `css-contract-check.js` **C3a** |
| K11 | `index.css:5557-5578` | 分页器 `.ant-pagination .ant-pagination-item` 透明边框 + active 绿底白字 | 登记册 #23：**刻意设计**（原型 demo 复刻），非缺陷；`border:none` 会移除 antd 分页边框，是产品决策 | v1.24 登记册 #23（保留项）；建议在 Phase 5 用 `variant="borderless"` 等价表达 |

### 第 4 批｜需重写而非删除（11 条）

| # | 文件:行号 | 现状 | 重写方向 | 依赖前置 | 置信度 |
|---|---|---|---|---|---|
| R1 | `index.css:871-880` | `.ant-input, .ant-input-number, .ant-select-selector { background/border/color/border-radius/font-size/transition !important }` | 拆成三份：① 边框/底色 → 交还 antd `variant` + `Input.colorBgContainer`/`colorBorder`/`hoverBorderColor`/`activeBorderColor`；② 圆角 → 统一 8px（T1）；③ font-size/transition → 移除（antd 已 `inherit`） | Phase 3（F3）+ Phase 4 | MEDIUM |
| R2 | `index.css:882-886` | `.ant-input:hover … { border-color !important }` | 交还 antd `Input.hoverBorderColor`（`AntdThemeBridge.tsx:124` 已设 `#156031`） | Phase 3 | MEDIUM |
| R3 | `index.css:888-893` | `.ant-input:focus … { border-color + box-shadow !important }` | 交还 antd `Input.activeBorderColor` + `activeShadow`（`AntdThemeBridge.tsx:123,125` 已设） | Phase 3 + Phase 4 | MEDIUM |
| R4 | `index.css:908-914` | `.ant-input-affix-wrapper { background/border/color/border-radius/transition !important }` | 同 R1（wrapper 侧） | Phase 3 + Phase 4 | MEDIUM |
| R5 | `index.css:916-918, 920-923` | wrapper hover / focused 边框与焦点环 | hover → token；focused 环 → antd `activeShadow`（**但 VIS-02「焦点环归 wrapper」契约需保留** → 需 antd 侧 wrapper 焦点环或保留手写） | Phase 3 + Phase 4 | MEDIUM |
| R6 | `index.css:2254-2742` | 暗色 cluster **C39**：90 规则 / 138 `!important` | 整块迁到 `darkAlgorithm` + 明暗分色板 token（F3） | **F3 硬前置** | MEDIUM |
| R7 | `index.css:2780-4517` | 暗色 cluster **C40**：341 规则 / 503 `!important` | 同 R6 | **F3 硬前置** | MEDIUM |
| R8 | `index.css:976-979, 2407-2429` | `.ant-table` 背景/圆角 + 暗色 thead/tbody/container | 交还 antd `Table.headerBg` / `rowHoverBg` / `rowSelectedBg`（`AntdThemeBridge.tsx:107-118` 已设 `cream.*`） | Phase 3 + Phase 6 | MEDIUM |
| R9 | `index.css:4806-4808` + `1879/1885/1891` | `--sidebar-width` 两处真相（200px vs 240px） | 合并为单一来源（保留 `[data-layout=…]` 家族，删除尾部重复定义） | 无（Phase 2 即可） | HIGH |
| R10 | `index.css:4815` | `:root { --input-bg: #fafaf9 }`（第 3 处，暗色被覆写） | 把浅色实测值并入 `[data-color-mode="light"]`（184 行）或 `:root`（27 行）+ 暗色 254 行；**绝不能留在 4812 这个后置 `:root`** | 无（Phase 2 即可，**必须最先做**） | HIGH |
| R11 | `index.css:2304, 2335` | 暗色规则使用 `var(--input-placeholder)`（只在 light 定义） | 统一变量名为 `--input-text-placeholder`（暗色块已有），并在 `:root` 补兜底 | 无（Phase 2 即可） | HIGH |

## 量化基线

### 3.1 当前值 → 收敛目标值

> 口径固定（见 P-m7）：**M1** = `find src -name "*.css" -exec cat {} + | grep -o '!important' | wc -l`
> （全 `.css` 出现次数）；**M2** = 仅 `src/index.css`；**M3** = 顶层规则条数（含 `.ant-` 选择器）；
> **M4** = 暗色顶层规则条数；**M5** = `index.css` 内 6 位 hex 字面量出现次数。
> **每次 phase 结束跑一遍并把数字记入 phase 报告。**

| 指标 | 当前值 | 目标值 | 降幅 | 达成路径 | 验证命令 / 方法 |
|---|---|---|---|---|---|
| **M1** `!important`（全 `.css`） | **1435** | **≤ 480** | −67% | 第 1 批 −40（死规则 31 + 死文件 9）；第 2 批 −12；第 4 批 R1-R8（Input/Select/Table/Modal 域交还 token）≈ −900 | `find src -name "*.css" -exec cat {} + \| grep -o '!important' \| wc -l` |
| **M2** `!important`（`index.css`） | **1359** | **≤ 420** | −69% | 同上 | `grep -o '!important' src/index.css \| wc -l` |
| **M3** 含 `.ant-` 的顶层规则条数（全 `.css`） | **748** | **≤ 420** | −44% | 第 1 批删 D5-D15 共 15 条；第 4 批 R1-R8 覆盖 Input(38)/Select(28)/Table(79)/Modal(45)/Form(10) 共 200 条 | 本文件 [§ 3.3](#33-契约脚本扩展点) 的 C11 计数断言 |
| **M3'** 含 `.ant-` 的**行数**（`index.css`） | **798** | **≤ 460** | −42% | 同上 | `grep -c '\.ant-' src/index.css` |
| **M4** 暗色顶层规则条数 | **506**（正确拼写）+ **8**（拼写错误） | **≤ 150** | −72% | D5/D6 删 8 条死规则；R6/R7 整块删除 C39+C40（**431 条**），改由 `darkAlgorithm` token 接管 | `grep -c 'data-color-mode="dark"' src/index.css`（当前 574） |
| **M5** 暗色分区（cluster，间隙 >5 行） | **64**（≈「约 60」） | **≤ 12** | −81% | R6/R7 合并后，暗色样式只剩「:root 变量块」+「少量 token 例外」 | 附录 §3.2 的聚类脚本 |
| **M6** 重复暗色目标 class（出现在 >1 个 cluster） | **80** | **0** | −100% | 同 M4 | §3.2 聚类脚本 |
| **M7** `index.css` 6 位 hex 字面量 | **311**（去重 **66**） | **≤ 60**（去重 **≤ 20**） | −81% | 第 2 批 T1-T14 | `grep -oE '#[0-9a-fA-F]{6}\b' src/index.css \| wc -l` |
| **M8** 全 `.css` 6 位 hex 字面量 | **465** | **≤ 120** | −74% | 第 1 批删死文件（含 `theme-styles.css` 的旧灰蓝 fallback）+ 第 2 批 | `find src -name "*.css" -exec cat {} + \| grep -oE '#[0-9a-fA-F]{6}\b' \| wc -l` |
| **M9** `index.css` 行数 | **5632** | **≤ 3200** | −43% | 第 1 批（死规则 ~45 行）+ R6/R7（431 条规则 ≈ 1100 行）+ R1-R5 | `wc -l src/index.css` |
| **M10** `index.css` 分区块（`/* =====`） | **97** | **≤ 97**（只重组不增） | — | — | `grep -cE '^\/\* ={5,}' src/index.css` |
| **M11** 暗色下 `--input-bg` 解析值 | **`#fafaf9`（错）** | **`#0f2e1b`（对）** | — | R10 | 浏览器 `getComputedStyle(document.documentElement).getPropertyValue('--input-bg')` |
| **M12** 暗色 placeholder 变量 | **未定义（声明作废）** | **非空** | — | R11 | 浏览器 `getComputedStyle(input).getPropertyValue('--input-placeholder')` |
| **M13** InputNumber 错误态边框 | **`rgb(219,215,206)`（错）** | **`rgb(186,54,48)`（对）** | — | 第 4 批 InputNumber 专项 | 浏览器 computed-style |
| **M14** 死文件（从未 import 的 `.css`/`.less`） | **8 个 / 555 行** | **0** | −100% | D16/D17 | §3.3 C8 静态断言 |
| **M15** 圆角（Input/wrapper） | **6px（项目）/ 8px（antd）不一致** | **8px 统一** | — | T1/T2 + Phase 3 | 浏览器 `borderTopLeftRadius` |
| **M16** 浏览器 console 的 antd 弃用警告 | **2 类**（`maskClosable` / `onSearch`） | **0** | — | Phase 0 单独清理 | DevTools Console 过滤 `Warning:` |

### 3.2 暗色聚类脚本（用于复算 M5 / M6）

```bash
# 暗色 cluster（间隙 > N 行切分）与「重复暗色目标 class」计数
# 依赖：本文件同目录的 cssparse.py（见 Sources）
python - <<'PY'
import sys, re, collections
sys.path.insert(0, r'<TEMP>/cssparse.py')
from cssparse import parse_css
blocks, _ = parse_css('src/index.css')
rules = [b for b in blocks if b['type'] == 'rule']
dark = [b for b in rules if 'data-color-mode="dark"' in b['sel'] or 'data-color_mode="dark"' in b['sel']]
GAP = 5
clusters, cur, prev = [], [], None
for b in dark:
    if prev is not None and b['start'] - prev > GAP:
        clusters.append(cur); cur = []
    cur.append(b); prev = b['end']
if cur: clusters.append(cur)
print('M5 暗色分区数(gap>%d):' % GAP, len(clusters))
tmap = collections.defaultdict(set)
for i, c in enumerate(clusters, 1):
    for b in c:
        for m in re.finditer(r'\.ant-([a-z0-9-]+)', b['sel']):
            tmap[m.group(1)].add(i)
print('M6 重复暗色目标 class:', sum(1 for v in tmap.values() if len(v) > 1))
PY
```

### 3.3 契约脚本扩展点（收敛的「可度量」基础设施）

#### 现有脚本的**实际能力**（已实跑验证）

| 脚本 | 能力 | 盲区 | 实跑结果 |
|---|---|---|---|
| `scripts/css-contract-check.js`（89 行） | **纯文本正则**：C1a-C1c（VIS-01 复位块存在 + 6 项复位 + 不写 box-shadow）、C2（禁 `:not(组合子)`）、C3a/C3b/C3c（3 处局部补丁）、C4a-C4c（焦点环 / wrapper 边框 / wrapper 焦点环）、C5a-C5c（status 边框色还原） | **不解析 DOM、不算特异性、不验证选择器命中数** → 死规则照样 PASS（见 P-C5 / P-M10）；只覆盖 `index.css` + `login.css` 两个文件 | 需在改动后实跑确认（本 Phase 只读，未运行修改后的脚本） |
| `scripts/css-contract-assertions.js`（74 行） | 浏览器 computed-style 断言：`doubleCount`、`inner`、`innerTextarea`、`innerDark`、`wrapperFocused`、`wrapperError` | **不是 CI 脚本**（需人工粘贴到 Console）；jsdom 无法解析 antd 6 的 `@layer` 嵌套；**fixture 用了不存在的 class**（`:48` `ant-input-affix-wrapper-status-error`）→ `passed` 恒 false（见 P-M11） | 需人工实机验证 |
| `scripts/check-exemption-reasons.mjs`（122 行） | 校验 `eslint.config.js` 的 `__P65_EXEMPTION_FILES__` / `__P66_EXEMPTION_FILES__` 两个锚点数组与 `scripts/exemption-reasons.json` **一一对应**（缺/多/重复/理由空/维度空/userVisible 无 difference 全查） | **只管 JS/TS 的 eslint 豁免**，**完全不涉及 `.css`** → hex 治理对 CSS 失明的根因就在这里 | `node scripts/check-exemption-reasons.mjs` 退出码 0（未改动） |
| `scripts/check-hardcoded-colors.mjs`（190 行） | 递归扫 `src/` 下 `.ts/.tsx/.css`；31 条 hex 映射 + 7 条 rgba 映射；路径白名单（`src/utils/three/colors.ts` / `src/design-system/tokens/` / `scripts/`）；值白名单 4 个 | ① **只拦 31 个已知 hex**，任意新 hex 直接漏过（`index.css` 311 处只命中 2 处）；② **未接入 `lint:ci`** | **实跑 exit code = 1**，18 个文件命中（含 `src/index.css:964` `#ff4d4f`、`src/index.css:970` `#faad14`） |
| `eslint.config.js:132-196` | `no-restricted-syntax` 对 `src/**/*.{ts,tsx}` 禁裸 hex（`Literal[value=/#[0-9a-fA-F]{3,8}\b/]`）、裸 `<Input>`/`<Select>`/`Modal.confirm`/`<Popconfirm>`/`<Input.TextArea>` | **`files` 不含 `.css`** → CSS 完全不在射程 | `npm run lint` 现为绿（`lint:ci` 另含 `assert-lint-budget.mjs` + `check-exemption-reasons.mjs`） |

#### 建议的扩展点（收敛 Phase 0 落地，作为「可度量」的载体）

| 编号 | 新增断言 | 实现位置 | 覆盖的风险 |
|---|---|---|---|
| **C6** | `var(--x)` 引用的变量必须在 `:root` 中出现（否则声明作废） | `css-contract-check.js` 新增 C6 | P-C4 / M11 |
| **C7** | 同一 antd 目标 class 不得出现在 ≥2 个暗色 cluster | `css-contract-check.js` 新增 C7 | P-M1 / M5、M6 |
| **C8** | `.css`/`.less` 文件必须被至少一个 `.tsx/.ts` import | `css-contract-check.js` 新增 C8 | P-m2 / M14 |
| **C9** | 同一 CSS 变量在多个选择器块中重复赋值（白名单除外） | `css-contract-check.js` 新增 C9 | P-m3 / R9、R10 |
| **C10** | `.module.css` 内不得出现 `.ant-` 选择器 | `css-contract-check.js` 新增 C10 | P-m6 |
| **C11** | `!important` 总量上限（Phase 0 → Phase N 递减阶梯） | `css-contract-check.js` 新增 C11 或独立 `scripts/css-budget-check.mjs` | P-M2 / M1、M2 |
| **C12** | `.css` 内不得出现 `data-color_mode`（连字符拼写） | `css-contract-check.js` 新增 C12 | P-m1 / D5、D6 |
| **C13** | `.css` 内 hex 字面量**全量**检测（非 31 条映射），走 `exemption-reasons.json` 同款豁免机制 | 新增 `scripts/css-hex-check.mjs`（或扩展 `check-hardcoded-colors.mjs`） | P-M9 / M7、M8 |
| **C14** | `maskClosable` / `onSearch` 零命中 | `css-contract-check.js` 或 `check-exemption-reasons.mjs` | P-M5 / M16 |
| **C15** | `AntdThemeBridge` 的 `useMemo` 依赖数组必须覆盖其读取的全部外部状态 | 新增单测 `src/design-system/components/__tests__/AntdThemeBridge.test.tsx` | P-C2 |

> **注意**：C13 若直接落地会让 `check-hardcoded-colors.mjs` 的失败文件数从 18 暴涨，
> 必须**先**把 `scripts/exemption-reasons.json` 的机制扩展到 `.css`（新增 `layer: "CSS"`），
> 否则 Phase 0 就会因为「新增硬编码色」被自己的门禁卡死。

---

## 收敛顺序建议

> 总原则：**每个 phase 内部「改 token + 删对应覆盖」必须在同一个 commit**；
> phase 边界只切在**组件域**上；**绝不**把同一个组件域拆成两个 phase。

### Phase 0 · 基础设施与口径冻结（**无视觉变更**）

| 项 | 内容 |
|---|---|
| 做什么 | ① 固定 M1-M16 口径并把 §3.2 聚类脚本落到 `scripts/`；② 修正 `css-contract-assertions.js:48` 的错误 fixture（P-M11）；③ 新增 C6-C12 静态断言；④ **单独**清理 2 类 antd 弃用（`maskClosable` → `mask={{closable:false}}`；`onSearch` → `showSearch={{onSearch}}`），与样式改动**分开 commit**；⑤ 把 `check-hardcoded-colors.mjs` 接入 `lint:ci`（先修其 18 个失败文件） |
| 如何验证 | `node scripts/css-contract-check.js`（exit 0）→ `node scripts/check-exemption-reasons.mjs`（exit 0）→ `npm run lint:ci`（exit 0）→ 浏览器 Console 零 `Warning:` → 视觉零变化（截图 diff = 0） |
| 如何回滚 | 单 commit revert；无 CSS 语义变更，回滚零风险 |

### Phase 1 · 零风险删除（**第 1 批 17 条**，无语义变更）

| 项 | 内容 |
|---|---|
| 做什么 | 删除 D1-D15（死规则）+ D16/D17（死文件）；同步**反转** `css-contract-check.js` 的 C3b 断言；删除 `FloorPlanEditor.tsx:18` 的空 `.less` import |
| 为什么这个顺序安全 | 这些规则**命中数全为 0**（死类名 / 拼写错误 / 从未 import），删除前后 computed-style **完全相同**；不触碰任何活规则 → 不可能破相 |
| 如何验证 | ① `node scripts/css-contract-check.js`（exit 0，含反转后的 C3b）；② 浏览器 `document.querySelectorAll('.ant-select-selector').length === 0`、`.dept-tree-search .ant-input-affix-wrapper` 命中 0（删除前后均应为 0）；③ **21 形态 × 明暗矩阵**截图 diff = 0（复用 `.planning/notes/v1.24-visual-verification.md` 的形态清单）；④ M1: 1435 → **1395**；M2: 1359 → **1328**；M4: 506 → **498** |
| 如何回滚 | 单 commit revert；因为无视觉变更，回滚零风险 |

### Phase 2 · 变量层修正（**串色与未定义变量**，无语义变更）

| 项 | 内容 |
|---|---|
| 做什么 | R10（`--input-bg` 串色）、R11（`--input-placeholder` → `--input-text-placeholder`）、R9（`--sidebar-width` 双定义）、T6/T7（去掉死的 hex fallback）、T1/T2/T3（圆角统一 8px）、T8（`#ffffff` → token）、T14（`--theme-zebra` 合并） |
| 为什么这个顺序安全 | 只改 **CSS 变量层**，不碰任何 antd 覆盖规则；影响面由「变量的所有消费点」决定，可枚举、可推理。**必须在 Phase 1 之后、Phase 3 之前** —— 否则 Phase 3 重构色板时会把这个串色源一起搬走，问题被掩盖而不是解决 |
| 如何验证 | ① 浏览器：暗色 `--input-bg` === `#0f2e1b`；`--input-placeholder` 非空；② 明暗各 14 页巡检（v1.24 已有巡检清单）；③ M1 目标 ≤ 1383；M7 目标 ≤ 300 |
| 如何回滚 | 单 commit revert；变量层回滚即刻恢复 |

### Phase 3 · `AntdThemeBridge` 明暗色板重构（**F3 硬前置**）

| 项 | 内容 |
|---|---|
| 做什么 | ① `antdThemeConfig.token` 改为**按 `themeMode` 分支**（暗色不再复用 `ruoyiBrand.cream.*` —— 解决 P-M8 的白底 Select）；② `components.*` 的暗色 token 补齐；③ 圆角统一（`borderRadius: 8` 与 `--theme-radius-lg` 对齐）；④ 扩展 `useMemo` 依赖 + 加 C15 单测 |
| 为什么必须在此之前 | R1-R8（把项目覆盖交还 antd token）**只有在 antd 侧 token 已经与现有手写样式等效时**才安全。若先删覆盖再改 token，中间态就是「antd 默认蓝灰 + 8px 圆角」—— 与 v1.24 验收差距巨大，必然被误判为破相 |
| 如何验证 | ① **等价性比对**：在 Phase 3 前后，对 21 个形态 × 明暗矩阵逐格比对 computed-style（`backgroundColor` / `borderColor` / `borderRadius` / `boxShadow`），差异**必须为 0**（本 phase 不改视觉）；② 单测：dark 模式下 `theme.token.colorBgContainer !== '#FFFFFF'`；③ M1 不变（纯 TS 改动） |
| 如何回滚 | 单 commit revert；不涉及 CSS，零风险 |

### Phase 4 · Input 域收敛到 variant（**最大收益块**）

| 项 | 内容 |
|---|---|
| 做什么 | R1-R5：`index.css:871-880 / 882-886 / 888-893 / 908-914 / 916-923` 交还 antd `variant` + `Input` token；同步重估 K1（VIS-01 复位块）—— antd 6 内层已有 `border:none`，若 `.ant-input` 覆盖被删则复位块仅保留 `padding:0` + `font-size:inherit`；**K2/K3（VIS-03 错误态）按 M13 同样要重测**（因为 antd status 色会重新被看到） |
| 为什么这个顺序 | Input 是全站 309 个输入框的载体，且是 v1.24 验收最密集的域（21 形态中 11 个是 Input）；先做它能让 K1-K7 这批「必须保留」的规则一次性定型，后续域不再反复 |
| 如何验证 | ① `css-contract-check.js` C1/C4/C5 全 PASS（**若复位块被重写，需同步更新 C1 的期望字符串**）；② `css-contract-assertions.js`（修好 fixture 后）`passed === true`；③ 21 形态中 11 个 Input 形态 × 明暗逐格比对；④ M1 目标 ≤ 1100；M3 目标 ≤ 690 |
| 如何回滚 | 单 commit revert；若复位块被重写，需**同时** revert C1 断言改动 |

### Phase 5 · Select / InputNumber / Form 域收敛

| 项 | 内容 |
|---|---|
| 做什么 | Select：D7-D15 已删 → 现在把 `.ant-select` 相关的手写边框交还 `Select` token（`AntdThemeBridge.tsx:129-136` 已设 `colorBgContainer` 等，**但暗色仍是白底 —— 依赖 Phase 3 的分色板**）；InputNumber：R1 中拆出的 `.ant-input-number` 分支单独处理，加 status 还原（0,2,0+important）→ 修 M13；Form：`.ant-form-item-explain-error`（`index.css:1582`）交还 `Form` token；`styles.module.css` 的 6 条 antd Modal 选择器迁出（P-m6） |
| 为什么在 Phase 4 之后 | Select 与 Input 共享 `Input` 域的 token 语义与 variant 解析链（`useVariants`）；先做 Input 可把「variant 在 DOM 上落成什么 class」的实测结论复用到 Select/InputNumber |
| 如何验证 | ① 浏览器：暗色 Select 背景 ≠ `rgb(255,255,255)`；InputNumber `status="error"` 边框 = `rgb(186,54,48)`；② Select / InputNumber / Cascader / TreeSelect / AutoComplete / Mentions 6 个形态 × 明暗；③ M1 目标 ≤ 900 |
| 如何回滚 | 单 commit revert |

### Phase 6 · Table / Modal / Nav / Feedback 域收敛

| 项 | 内容 |
|---|---|
| 做什么 | R8（Table thead/tbody/container → `Table.headerBg`/`rowHoverBg`/`rowSelectedBg`）；Modal content/header/title/close-x → `Modal` token；Dropdown / Menu / Tabs / Tag / Alert / Pagination 等 → 对应 `components.*` token |
| 为什么在最后 | 这些域的**手写覆盖与 antd 默认差异最小**（v1.24 登记册 #27-#33 全部是 P2），风险最低、收益也最低 → 放在最后，用前面 phase 积累的 token 等价性经验兜底 |
| 如何验证 | 14 页巡检 + v1.24 视觉记录逐页比对；M1 目标 ≤ 600；M3 目标 ≤ 500 |
| 如何回滚 | 单 commit revert |

### Phase 7 · 暗色 mega-cluster 合并（**最后的原子操作**）

| 项 | 内容 |
|---|---|
| 做什么 | R6（`index.css:2254-2742`，90 规则）+ R7（`index.css:2780-4517`，341 规则）→ 整块删除，改为由 Phase 3 的分色板 token + 少量例外承担 |
| 为什么必须最后 | 这是**唯一一个「一次性删 431 条规则」**的操作。若在 Phase 4-6 之前做，暗色下所有组件会瞬间回到 antd `darkAlgorithm` 默认色 → 全站暗色破相；Phase 4-6 完成后，浅色侧已被 token 化，暗色侧的删除才只是「去掉冗余副本」 |
| 如何验证 | ① 暗色 21 形态 × 逐格比对；② 暗色 14 页巡检；③ M4 目标 ≤ 150；M5 目标 ≤ 12；M6 目标 = 0；M1 目标 ≤ 480；M9 ≤ 3200 |
| 如何回滚 | 单 commit revert（文件级回滚，`git revert` 即恢复到 Phase 6 末态） |

### 收敛顺序的「为什么不会中间态破相」——三条保证

1. **死代码先行（Phase 1）**：第一刀砍掉的是**命中数 0** 的规则 → computed-style 逐格 diff 必为 0，
   不存在「看起来变了但说不清哪里变」的情况。
2. **变量层先行（Phase 2）**：变量层的变更**可枚举**（`--input-bg` 有 22 个消费点），
   每个消费点都能单独断言 → 变更影响可预测。
3. **token 等价 → 再删覆盖（Phase 3 → 4-6）**：
   Phase 3 的**验收标准就是「视觉 diff = 0」**。只有当 antd 侧 token 已经被证明能
   **一模一样**地复现现有手写样式时，Phase 4-6 才有资格删除覆盖。
   这是整个顺序的**核心保险** —— 它把「删除」这个不可逆动作推迟到「等效」被证明之后。

### 哪些改动**必须**等 F3（Phase 3）之后才能做

| 改动 | 为什么必须等 F3 |
|---|---|
| R1-R5（Input 域交还 token） | antd 侧 `Input.colorBgContainer` / `colorBorder` 在暗色下仍是 `#FFFFFF`/`#DBD7CE`（`AntdThemeBridge.tsx:83,86,122,130`）→ 先删项目覆盖 = 暗色输入框变白块 |
| R6/R7（暗色 mega-cluster 删除） | 同上；且暗色手写 cluster 目前是**唯一**提供暗色底色/边框的来源 |
| Select 域收敛（Phase 5） | `Select.colorBgContainer` 同样写死 `#FFFFFF` |
| Table 域收敛（R8） | `Table.headerBg` 等全部指向 `cream.*` 常量，暗色下恒为浅色 |
| 圆角统一（T1/T2/T3） | antd `borderRadius: 8` 已在 `AntdThemeBridge.tsx:89`；但**同一 token 也被暗色 cluster 依赖**，需先完成分色板 |

**不必等 F3 的**：Phase 0（工具）、Phase 1（死代码）、Phase 2（变量层）、
`maskClosable`/`onSearch` 弃用清理、`.module.css` 的 antd 选择器迁出。

---

## Phase-Specific Warnings

> 对应 v1.25 ② 线可能拆出的 8 个 phase（Phase 0-7，与上节一一对应）。

| Phase | 最可能的坑 | 触发信号 | 缓解措施 |
|---|---|---|---|
| **P0 · 工具与口径** | ① `css-contract-assertions.js` 的 fixture 用了 antd 6 不存在的 class → `passed` 恒 false，团队误以为「错误态断言失败 = 有回归」而放弃该脚本；② 新增的 C13（CSS hex 全量检测）会让 `check-hardcoded-colors.mjs` 失败文件数暴涨，**Phase 0 把自己卡死** | 断言脚本输出 `passed: false` 但 computed-style 看起来正常 | ① 先修 fixture（`:48` → `ant-input-status-error`，期望色改 `rgb(186,54,48)`）；② C13 **延后**到 Phase 1/2 之后，或先扩展 `exemption-reasons.json` 支持 `layer: "CSS"` |
| **P1 · 零风险删除** | ① 删 D1-D4 后，部门树搜索框**同时**失去 `.user-form-input` 那条活规则的兜底 —— 若两条一起被误判为死规则，搜索框会变回带边框的 antd 输入框；② 删 `theme-styles.css` 时有人误以为 `.ant-menu-item-selected` 的样式会丢 —— 实际它在 `index.css:463-489` 等处已有等价实现 | 部门树搜索框出现描边/白底 | 删除前对每条死规则跑 `document.querySelectorAll('<完整选择器>').length === 0`；D1-D4 与 K8/K9 成对处理，只删前者 |
| **P2 · 变量层** | ① 修 `--input-bg` 时误把登录页的 `#fafaf9` 提升为全局 `--input-bg` 的浅色值（它本就是登录页色板，源自 `login.css:346`）；② 变量改名 `--input-placeholder` → `--input-text-placeholder` 时漏改浅色块 → 浅色 placeholder 反向失效 | 浅色/暗色出现「一个好一个坏」的镜像症状 | ① 只在 `[data-color-mode="light"]` 保留 `--input-placeholder` 别名指向 `--input-text-placeholder`；② 改名后**两个方向**都断言（浅色 + 暗色） |
| **P3 · F3 色板重构** | ① `useMemo` 依赖数组漏加新状态源 → token 不更新，表现为「切了暗色但颜色没变」；② 暗色 Select 变成**另一种**错误色（把 `#FFFFFF` 换成深绿但 `--border` 没同步）→ 比现在的白底更难判断 | 「主题切换后样式没反应」或「暗色下边框和底色不同源」 | ① C15 单测断言 `useMemo` deps 覆盖全部外部读取；② 分色板必须**成组**定义（bg / border / text 三件套），不允许只改其一 |
| **P4 · Input 域** | ① 删 `.ant-input` 覆盖后，antd 内层 `border:none`（特异性 0,1,1）**不足以**压过什么 → 若同时有别的规则给内层上边框，双框回归；② 重写 K1（VIS-01 复位块）时把 `box-shadow` 也写进去 → **焦点环变双圈**（VIS-02 回归）；③ `font-size: inherit` 被移除 → `size="large"` 不放大 | 内层 input 出现 1px 边框；焦点环变 2 圈 | ① 删覆盖与**保留**复位块必须同 commit；② 复位块**永远不写 box-shadow**（C1b 断言）；③ `css-contract-assertions.js` 的 `doubleCount` 必须 = 0 |
| **P5 · Select / InputNumber / Form** | ① Select 在 antd 6 的边框画在 `.ant-select` 根上，项目若继续按 `.ant-select-selector` 写规则 → **完全不生效**且没人发现（因为死规则不报错）；② InputNumber 的 status 还原选择器写错特异性（写成 0,1,0 + important）→ 压不过 `:875` | Select 边框在暗色下消失或恒白；InputNumber 错误态仍灰 | ① 收敛后**禁止**再出现 `.ant-select-selector`（C13/C12 扩展断言）；② InputNumber status 还原用 0,2,0 + important，并加浏览器断言 M13 |
| **P6 · Table / Modal / 其它** | ① Table thead 交还 token 后，`bordered` 变体的分隔线位置变化（登记册 #27 已登记「bordered/排序列对齐被强制」）；② Modal content 交还 token 后，`borderless` 变体不再被强制去边框 | 表格列宽/对齐错位、Modal 出现多余边框 | 逐个变体做「有/无 bordered」「有/无 borderless」两态验收，不能只测默认态 |
| **P7 · 暗色 mega-cluster** | ① 431 条规则一次性删除，若其中有**唯一**提供某组件暗色样式的规则 → 该组件暗色下彻底裸奔（antd 默认）；② 删除后 `index.css` 从 5632 行骤降到 ~3000 行，**极易误删相邻的有效规则** | 某组件暗色下变成 antd 默认外观；或整个暗色块消失 | ① 删除前用 §3.2 脚本导出被删规则清单，逐条确认「antd token 能提供等价样式」；② 删除**只做块级操作**（按 cluster 边界整段删），不逐条手改；③ 删除后立刻跑暗色 21 形态 + 14 页巡检 |

---

## Sources

### 读过的文件（前端）

| 文件 | 用途 |
|---|---|
| `ruoyi-react-frontend/src/index.css`（5632 行） | 主战场：全量盘点、死规则定位、变量作用域分析 |
| `ruoyi-react-frontend/src/pages/login/login.css`（504 行） | 登录页局部补丁（K10）、硬编码琥珀金色板（T9-T12） |
| `ruoyi-react-frontend/src/components/shared/ActionButtons.css`（80 行） | 表格操作按钮（8 条暗色规则 / 18 `!important`，T13） |
| `ruoyi-react-frontend/src/design-system/themes/theme-styles.css`（97 行） | 死文件 D16（含旧灰蓝 fallback） |
| `ruoyi-react-frontend/src/components/dashboard/widgets/base/BaseWidget.css`（311 行） | 4 条通用 antd 选择器 |
| `ruoyi-react-frontend/src/pages/operations/building-spaces/components/styles.module.css`（370 行） | CSS Module 内混入 6 条 antd 全局选择器（P-m6） |
| `ruoyi-react-frontend/src/design-system/components/AntdThemeBridge.tsx`（192 行） | **真 antd ConfigProvider**（`:178`）；token 映射（`:74-92`）；8 组件覆盖（`:93-168`）；`darkAlgorithm`（`:65`）；`hashed: true`（`:173`） |
| `ruoyi-react-frontend/src/design-system/tokens/colors.ts`（134 行） | `ruoyiBrand` 真相源：`cream.surface=#FFFFFF`、`cream.border=#DBD7CE`、`functional.danger=#BA3630` 等 |
| `ruoyi-react-frontend/src/design-system/components/ThemeProvider.tsx` | `data-color-mode` 属性写入方 |
| `ruoyi-react-frontend/src/components/ConfigProvider.tsx`（72 行） | 项目自有 Provider（**仅副作用同步**，不渲染 Provider）→ P-C2 |
| `ruoyi-react-frontend/src/App.tsx`（67 行） | **双 Provider 嵌套点**：`:47` AntdThemeBridge / `:48` 项目 ConfigProvider |
| `ruoyi-react-frontend/src/components/ui/AppInput.tsx`（16 行） | `allowClear = true` 默认 → 决定 DOM 是否有 affix wrapper（P-C5 的关键前提） |
| `ruoyi-react-frontend/src/components/ui/AppSelect.tsx`（26 行） | `showSearch ?? count >= 8`（P-M5 的 warning 触发条件） |
| `ruoyi-react-frontend/src/components/DeptTree/index.tsx:190-220` | `className="dept-tree-search user-form-input"` 落在 `AppInput` 上（P-C5 / P-M6） |
| `ruoyi-react-frontend/src/components/shared/FormModal.tsx` / `src/components/modal/BaseEditModal.tsx` / `src/components/shared/confirmDelete.ts` / `src/components/captcha/CaptchaModal.tsx` / `src/components/network/port-write/BulkWriteDrawer.tsx` | `maskClosable` 弃用命中点（P-M5） |
| `ruoyi-react-frontend/src/components/CronSelector/index.tsx:206` / `src/components/dashboard/settings/EndpointSelector.tsx:211` | `onSearch` 弃用命中点（P-M5） |
| `ruoyi-react-frontend/scripts/css-contract-check.js`（89 行） | 现有静态契约能力与盲区（§3.3） |
| `ruoyi-react-frontend/scripts/css-contract-assertions.js`（74 行） | computed-style 断言 + **错误 fixture**（P-M11） |
| `ruoyi-react-frontend/scripts/check-exemption-reasons.mjs`（122 行） | 豁免 ↔ 理由一一对应校验（**只管 JS/TS**） |
| `ruoyi-react-frontend/scripts/check-hardcoded-colors.mjs`（190 行） | CSS hex 扫描器（**当前 exit=1**） |
| `ruoyi-react-frontend/eslint.config.js:118-200` | hex 禁令块 `files: ["src/**/*.{ts,tsx}"]`（**不含 .css**） |
| `ruoyi-react-frontend/package.json` | `lint:ci` 组成（**不含** `check-hardcoded-colors.mjs`） |
| `.planning/notes/v1.24-css-audit-register.md`（99 行） | 36 条既有登记（34+2），本文件第 3/4 批与其对齐 |
| `.planning/notes/v1.24-visual-verification.md`（25.8 KB） | 21 形态 × 明暗矩阵 + 14 页巡检清单（验收基线） |

### 读过的 node_modules 源码（用于「antd 6 是否有该 class / 该 prop」的实证）

| 文件 | 结论 |
|---|---|
| `node_modules/antd/package.json` | antd **6.6.4** |
| `node_modules/antd/es/input/Input.js:129-176` | status class 由 `getStatusClassNames(prefixCls, mergedStatus)` 生成 → `ant-input-status-error`（**wrapper 自身**）；`className` 传给 `RcInput` 作为 root class |
| `node_modules/@rc-component/input@1.3.1/es/BaseInput.js:129-131` | 注释明示「`className` and `style` are always on the root element」 |
| `node_modules/@rc-component/input@1.3.1/es/utils/commonUtils.js` | `hasPrefixSuffix = !!(prefix \|\| suffix \|\| allowClear)` → **`allowClear=true` 即生成 affix wrapper** |
| `node_modules/antd/dist/antd.css` | ① `.ant-select-selector` 唯一命中在 ColorPicker 内部后代选择器；② `.ant-input-number-outlined` 承载 InputNumber 边框；③ `.ant-input-number-outlined.ant-input-number-status-error:not(...)` 承载状态边框（0,3,0，**无 important**）；④ `--ant-color-error:#ff4d4f` 存在（cssVar 运行时注入） |
| `node_modules/@rc-component/select/es/**` | `-selector` 零命中 → Select 不渲染 `.ant-select-selector` |
| `node_modules/antd/es/select/index.js:104,217-229` | `useVariants('select', …)`；deprecated 列表 = `dropdownMatchSelectWidth / dropdownStyle / dropdownClassName / popupClassName / dropdownRender / onDropdownVisibleChange / bordered` + `showArrow` |
| `node_modules/antd/es/select/index.d.ts:78` | `showSearch?: boolean \| (SearchConfig<OptionType> & {...})` → antd 6 的对象形态 |
| `node_modules/@rc-component/select/es/Select.d.ts:85-104` | `onSearch` / `searchValue` / `autoClearSearchValue` / `filterOption` / `filterSort` / `optionFilterProp` 均标注 `@deprecated please use showSearch.*` |
| `node_modules/@rc-component/select/es/utils/warningPropsUtil.js:53-55` | 运行时 warning：「`onSearch` should work with `showSearch` instead of use alone.」 |
| `node_modules/antd/es/modal/Modal.js:126` / `es/modal/interface.d.ts:71-74` | `maskClosable` → `mask.closable` 弃用映射与 `@deprecated` 标注 |
| `node_modules/antd/es/form/hooks/useVariants.js` | variant 解析顺序：prop → `bordered===false` → `VariantContext` → `components.<C>.variant` → 全局 `variant` → `'outlined'` |

### 实跑命令（全部在本仓库 `ruoyi-react-frontend/` 目录下执行，Windows Git Bash）

```bash
# —— A1 样式文件清单与行数 ——
find src -type f \( -name "*.css" -o -name "*.less" -o -name "*.scss" -o -name "*.sass" \) | sort
find src -type f -name "*.css" | sort | while read f; do printf "%6d  %s\n" "$(wc -l < "$f")" "$f"; done
find src -type f -name "*.css" -exec cat {} + | wc -l          # 8494

# —— A2 !important 分布 ——
find src -type f -name "*.css" | sort | while read f; do n=$(grep -o '!important' "$f" | wc -l); [ "$n" -gt 0 ] && printf "%6d  %s\n" "$n" "$f"; done
find src -type f -name "*.css" -exec cat {} + | grep -o '!important' | wc -l   # 1435
grep -o '!important' src/index.css | wc -l                                     # 1359

# —— A3 antd 选择器分布与域分类 ——
find src -type f -name "*.css" | sort | while read f; do n=$(grep -c '\.ant-' "$f"); [ "$n" -gt 0 ] && printf "%6d  %s\n" "$n" "$f"; done
grep -o '\.ant-[a-zA-Z0-9_-]*' src/index.css | sort -u | wc -l                  # 286 唯一类名
python <TEMP>/domain.py $(find src -name "*.css" | sort)                       # 748 条规则 × 10 域

# —— A4 暗色分区 ——
grep -nE '\[data-color-mode="dark"\]|\[data-color_mode="dark"\]' src/index.css
grep -c 'data-color-mode="dark"' src/index.css                                 # 574
grep -c 'data-color_mode="dark"' src/index.css                                 # 8
python <TEMP>/cssparse.py src/index.css summary                                # 506 + 8 条暗色规则
python <TEMP>/darkcluster.py src/index.css                                      # 44/64/100 cluster + 80 重复目标

# —— A5 变量作用域与未定义变量 ——
python <TEMP>/cssparse.py src/index.css vars                                    # :root 之外的就地重定义
python <TEMP>/varuse.py src/index.css                                           # 未定义 var + 暗色作用域缺失
grep -n -- '--input-bg' src/index.css                                           # 184 / 254 / 4815 三处
grep -n ':root' src/index.css                                                   # 27 / 4812

# —— B 死规则 / 死文件 ——
grep -rn 'dept-tree-search' src                                                # 1 处 TSX + 6 处 CSS
grep -rn 'import .*\.css' src --include=*.tsx --include=*.ts
grep -r 'select-selector' node_modules/@rc-component/select/es/                 # 零命中
grep -o 'ant-select-selector' node_modules/antd/dist/antd.css | wc -l           # 20（全部在 ColorPicker 上下文）
grep -rn 'hasPrefixSuffix\|className.*root element' node_modules/@rc-component/input/es/

# —— C 门禁能力 ——
node scripts/css-contract-check.js          # 静态契约（改动前）
node scripts/check-exemption-reasons.mjs    # 豁免 ↔ 理由
node scripts/check-hardcoded-colors.mjs; echo "exit=$?"   # 实跑 exit=1，18 个文件
grep -n 'no-restricted-syntax\|files:\|Literal\[value=/#' eslint.config.js
node -e "console.log(JSON.stringify(require('./package.json').scripts,null,1))"
```

> **说明**：本 Phase 为**只读研究**。除本文件 `.planning/research/PITFALLS.md` 外，
> 未修改任何源码，未执行 `git commit`。§3.2/§3.3 中引用的 `cssparse.py` / `domain.py` /
> `varuse.py` / `darkcluster.py` 为本次研究用的临时解析脚本（位于系统临时目录，
> 未写入仓库）；落地时建议正式化为 `scripts/css-analyze.mjs`（Node 版，与项目工具链一致）。
