# Phase 90 变更登记表（90-CHANGESET）

> 判定规则：矩阵 diff（`90-visual-evidence/matrix-diff-90.cjs`）中每条差异必须能归入下表类别之一；**未归类 = FAIL**。
> **与 89 的关键差别**：89 的红线是「light 差异必须 = 0 / dark 差异 > 0」（浅色零漂移）；**90 的变更本来就是明暗双向**（圆角 6→8 在 light/dark 都变）→ 90 的红线改为
> **「light 与 dark 各自差异 > 0（防空跑/恒等）+ 全部差异必须归类 + 归类后形态集合与登记**不多不少**」**。
> **登记表与 `90-visual-evidence/matrix-diff-90.cjs` 的代码内常量双写，两处必须一致**（代码内常量：`A_CELLS` / `B_CELLS` / `B_AFTER` / `C_CELLS` / `FORMS_90`）。
> 形态总数 **22**（89 的 21 形态 + `f22_input_number_status_error`）。

---

## 形态全集（22）

```
f01_input_plain  f02_input_prefix  f03_input_suffix  f04_input_allowclear
f05_input_addonbefore  f06_input_addonafter  f07_input_password  f08_input_search
f09_textarea  f10_textarea_showcount  f11_input_number
f12_select_single  f13_select_multiple  f14_datepicker  f15_range_picker
f16_cascader  f17_treeselect  f18_autocomplete  f19_mentions
f20_appinput  f21_apptextarea  f22_input_number_status_error
```

**采样属性（6）**：`bg` / `border` / `radius` / `shadow` / `btw` / `fontSize`（`fontSize` 为本 phase 新增）。
**格数口径**：**(22 形态 × 2 part − 6 个 inner=null 形态) × 6 属性 = 228 格 / 模式 → 明暗合计 456 格**（理论上限 528）。

---

## a 类 · 输入类圆角 6 → 8（STYLE-03）

| # | 类别 | 形态 × part × 属性 | before → after | 明暗 | 触发源 | 依据 |
|---|------|--------------------|----------------|------|--------|------|
| **a** | 输入类圆角 6→8 | **18 格 / 模式**（见下表） | `6px` → `8px` | **明暗双向**（light + dark 各 18 格） | `index.css:885`（`.ant-input, .ant-input-number` 组的 `border-radius: var(--theme-radius-md)`）→ 改 `var(--theme-radius-lg)`；`index.css:921`（`.ant-input-affix-wrapper` 组）→ 改 `var(--theme-radius-lg)` | `--theme-radius-lg` = `8px`（`index.css:130` 的 `:root` 静态定义，**无 JS 运行时覆盖** → computed 恒为 8px；W0 探针实测 `radiusLgVar=8px`，明暗同值） |

**a 类逐格清单（18 格 / 模式 · 明暗两模式完全相同）**

| # | 形态 | part | 属性 | before | after | 说明 |
|---|------|------|------|--------|-------|------|
| 1 | f01_input_plain | carrier | radius | `6px` | `8px` | `.ant-input` |
| 2 | f02_input_prefix | carrier | radius | `6px` | `8px` | `.ant-input-affix-wrapper` |
| 3 | f03_input_suffix | carrier | radius | `6px` | `8px` | wrapper |
| 4 | f04_input_allowclear | carrier | radius | `6px` | `8px` | wrapper |
| 5 | f05_input_addonbefore | carrier | radius | `6px` | `8px` | `.ant-input`（addon 组内层 input） |
| 6 | f05_input_addonbefore | **inner** | radius | `6px` | `8px` | **carrier 与 inner 是同一个 `input.ant-input` 元素** |
| 7 | f06_input_addonafter | carrier | radius | `6px` | `8px` | `.ant-input` |
| 8 | f06_input_addonafter | **inner** | radius | `6px` | `8px` | **carrier 与 inner 是同一个 `input.ant-input` 元素** |
| 9 | f07_input_password | carrier | radius | `6px` | `8px` | wrapper |
| 10 | f08_input_search | carrier | radius | `6px` | `8px` | wrapper |
| 11 | f09_textarea | carrier | radius | `6px` | `8px` | wrapper |
| 12 | f10_textarea_showcount | carrier | radius | `6px` | `8px` | wrapper |
| 13 | f11_input_number | carrier | radius | `6px` | `8px` | `.ant-input-number` 根 |
| 14 | **f14_datepicker** | **inner** | radius | `6px` | `8px` | **picker 内层 `input.ant-input`** |
| 15 | **f15_range_picker** | **inner** | radius | `6px` | `8px` | **picker 内层 `input.ant-input`** |
| 16 | f20_appinput | carrier | radius | `6px` | `8px` | wrapper |
| 17 | f21_apptextarea | carrier | radius | `6px` | `8px` | wrapper |
| 18 | f22_input_number_status_error | carrier | radius | `6px` | `8px` | `.ant-input-number` 根（f22 是 a+b 双落点形态） |

**a 类实测派生说明（防「凭空多写」）**：

- 本清单**不是先验推定**，而是由 W0 的 `before-{light,dark}.json` **逐格派生**：凡 `before.radius === '6px'` 的 part 即落入 a 类 → **light 18 格 / dark 18 格，两者集合完全相同**。
- **内层 `border-radius: 0` 保持 0 不变** → 不产生 0→8 的差异：受 VIS-01 复位块 `.ant-input-affix-wrapper > input.ant-input { border-radius: 0 !important }`（`index.css:946-954`）保护的内层 = f02 / f03 / f04 / f07 / f08 / f09 / f10 / f20 / f21（9 个）。
- **f11 / f22 内层 0px** 由 antd 自身的 InputNumber 内层复位保证（`.ant-input-number-input` 不在项目复位块作用域内，但 antd 6 内建复位生效）→ 不产生差异。
- **f12 / f13 / f16 / f17 / f18（Select 族）carrier 0px** → 不在 a 类。
- **f19_mentions（carrier + inner 均 0px）** → 不在 a 类（`.ant-mentions` textarea 不吃 `.ant-input` 选择器）。
- **f14 / f15 的 carrier 是 `.ant-picker` 根（0px）** → 只落 **inner**（picker 内层 `input.ant-input` 吃 `:885` 的 6px，且**不在** VIS-01 复位块作用域内）。

> ⚠️ **口径补正（相对 90-01-PLAN 任务 1.5 的文字描述）**：PLAN 正文写 a 类落点为「f01…f11 / f20 / f21 的 **carrier.radius**，其中 **f05/f06 另含 inner.radius**」，
> **遗漏了 f14_datepicker.inner.radius 与 f15_range_picker.inner.radius 两格**。W0 实采证实这两格的 before 就是 `6px`（picker 内层 `input.ant-input` 吃 `index.css:885` 的 `!important` 6px），
> W1 改 `:885` 后必然变 8px → **必须登记**，否则 90-04 终跑会命中「未归类差异 = FAIL」。本表按**实采**为准（PLAN 的「a 类形态清单与实采 before 快照一致（不得凭空多写）」验收条款优先）。

---

## b 类 · InputNumber 错误态边框（VISFIX-05）

| # | 类别 | 形态 × part × 属性 | light before → after | dark before → after | 触发源 | 依据 |
|---|------|--------------------|---------------------|---------------------|--------|------|
| **b** | InputNumber 错误态红框 | **f22_input_number_status_error · carrier · border**（1 格 / 模式，明暗各 1 格） | `rgb(219, 215, 206)` → **`rgb(186, 54, 48)`** | `rgb(45, 77, 51)` → **`rgb(161, 49, 44)`** | 90-02 追加的 InputNumber status 还原规则（error 组 **6 选择器** = 常态 / `:hover` / `:focus` × 明暗，色值 `var(--ant-color-error)`） | antd 6.6.4 的 `getStatusClassNames` 把 status class 打在 `.ant-input-number` **根**（`ant-input-number-status-error`），且 antd 的 `genOutlinedStyle` 把边框画在**根**；现状被 `index.css:883` 的 `border` 简写 `!important` 压掉（(0,1,0)+imp > antd (0,3,0) 非 important） |

**b 类实测依据（W0 只读探针，证据 `90-visual-evidence/probe-status-before.json`）**：

| 探针项 | light | dark |
|---|---|---|
| `--ant-color-error`（cssVar scope 内解析值） | `#ba3630` | **`#a1312c`** |
| `.ant-input.ant-input-status-error` → `borderTopColor`（项目既有 VIS-03 规则 `:967-974` 命中） | `rgb(186, 54, 48)` | **`rgb(161, 49, 44)`** |
| `.ant-input-number.ant-input-number-status-error` → `borderTopColor`（**现状，被 `:883` 压掉**） | `rgb(219, 215, 206)` | `rgb(45, 77, 51)` |

> ⚠️ **W0 实测裁定（推翻 90-RESEARCH §7.3 的推定）**：研究 §7.3 写「`--ant-color-error` 明暗同值 `#BA3630` → 明暗双测 `rgb(186,54,48)` 成立」，并明文要求「执行期必须**实测**后按实测改判据，**不得伪造**」。
> **实测结果：暗色 cssVar scope 内 `--ant-color-error` = `#a1312c`**（antd `darkAlgorithm` 对 `colorError` 做了暗色派生），**不是** `#BA3630`。
> ⇒ b 类 after 值**明暗不同**：light = `rgb(186, 54, 48)`、dark = `rgb(161, 49, 44)`。
> 该裁定已同步落进 `matrix-diff-90.cjs` 的 `B_AFTER`（mode-aware 匹配），并与项目既有 VIS-03 规则的实测渲染**完全同值**（同源 `var(--ant-color-error)`）。
> **注**：项目 `index.css:272` 的 `--input-border-error: #ba3630` 仅在 **dark** 块定义 → **light 侧无对应错误态变量**，b 类**不可**改用该变量（研究 §7.2 BLOCKER 3 的同源约束）。

---

## c 类 · font-size（可选 · T4 裁定后回填）

| # | 类别 | 形态 × part × 属性 | before → after | 明暗 | 触发源 | 依据 |
|---|------|--------------------|----------------|------|--------|------|
| **c** | font-size | **（空集占位）** | — | — | `index.css:886` 的 `font-size: 14px !important`（T4 候选删除项） | **W0 登记为空集**：T4 的采纳与否由 **90-03** 裁定；W0 实采显示 **22 形态 × 38 个 part 的 `fontSize` 恒为 `14px`**（明暗一致）→ 若 T4 采纳并删除 `:886`，antd 内层 `font-size: inherit` → 载体仍解析为 `14px` → **预期零差异** |

**空集理由（显式登记）**：

1. `matrix-diff-90.cjs` 的 `C_CELLS = new Set()`，且 `REGISTRY` 的 c 类 `match = null` → **任何 `fontSize` 差异都会被判「未归类」→ FAIL**（= 收紧而非放松）。
2. **若 90-03 的 T4 裁定为「采纳且实测零差异」** → c 类恒为空集，本表维持现状即可。
3. **若裁定为「采纳且实测有差异」** → 90-03 必须回填本表的 before/after 与形态集合，**并同步修改 `matrix-diff-90.cjs` 的 `C_CELLS` 与 c 类 `match`**（两处必须一致）。
4. **T4 不在 W1**（研究 §6 W1 闸门已注明「T4 删 `:886` font-size 不在 W1，移出 W2 单独裁定」）。

---

## 未清偿债（移交 91 / 92）

> 内容在 90-03 回填；本波只落骨架。

| # | 债 | 量化 | 归属 |
|---|---|---|---|
| **D-a** | `Input.hoverBorderColor` 需对齐 `#598e5e` | 现值 `#156031`（antd token）≠ 项目 `--input-border-hover` `#598e5e` → 删除手写 hover 规则前必须先对齐 token（对齐后明暗同值，视觉不变） | Phase 91 / 92 |
| **D-b** | `colorBorder` 交还时 light disabled 边框变更 | 交还 `colorBorder` 后 light disabled 边框 `#dbd7ce` → `#C2BDB2`（= antd `colorBorderSecondary` light）—— **必然的登记变更**，须写进 91/92 的 CHANGESET | Phase 91 / 92 |
| **D-c** | 乙方案路径图（去 important 全交还） | 去 `border-color` 的 `!important` + 删 VIS-01 复位块（`:946-954`）+ C1a 反转为「复位块不得存在」的路径图 | Phase 91 / 92 |
| **D-d** | `--theme-radius-md`（6px）非输入域消费残留 | **15 处**（13 CSS + 2 TSX：`GlobalSearch.tsx:281` / `:298`）+ 变量源头 `index.css:129` → 不属 STYLE-03 判据面，90 不动；92 终态盘点废存 | Phase 92 |
| **D-e** | 六处字面 6px 的归属组件清单 | `:4458` / `:4470` / `:4482` / `:4494` / `:4506` / `:5234`（`:5489` = 分页器，属 STYLE-03 口径内，不在债内）→ **W1 执行时逐处判归属并补全进本表** | 90-02（W1）回填 |

**W0 追加登记（本 phase 新产生，供 91/92 承接）**：

| # | 债 | 量化 | 归属 |
|---|---|---|---|
| **D-f** | wrapper 组的 `border` 简写（`index.css:919`）本波不拆 | `css-contract-check.js:76`（C4b）以**字面正则**守护 `border: 1px solid var(--input-border)` → 拆了就红 → 批次 C **只拆 `:883`**；wrapper 处置留债 | Phase 91 / 92 |
| **D-g** | C13 的 `stripComments` 正则不跨行（`/\/\*.*?\*\//g`） | `.` 不匹配换行 → **多行注释块内的 hex 被计入**（`index.css` raw 311 vs strip 后 304，差 7）→ 90 的注释块**不得含任何 6 位 hex**；修法（正则改 `[\s\S]*?`）本 phase 不动 C13 | Phase 92 |
| **D-h** | 暗色 `--ant-color-error` ≠ 品牌 `danger` 色 | 暗色 scope 内 `--ant-color-error` = `#a1312c`（antd `darkAlgorithm` 派生）≠ `ruoyiBrand.functional.danger` = `#BA3630` → 暗色错误态红框与品牌色不一致（暗色对比度更佳，但与品牌色板脱钩）→ 属 token 层归口 | Phase 91 / 92 |

**W1 追加登记（90-02 实跑新增）**：

| # | 债 | 量化 | 归属 |
|---|---|---|---|
| **D-i** | **M1 / M2 的 `!important` 计数口径把注释内的 `!important` 字面也计入**（与 D-g 的 C13 hex 同型缺陷） | 实测证据：W1 重写复位块 / VIS-03 / VISFIX-05 三处注释块时，把注释里的 `!important` 字面改成「important」措辞 → **M1 / M2 各 -7**（净），而 CSS 语义零变化（注释不参与层叠）；影响面：M1 / M2 预算的「真实声明覆盖率」被注释字面稀释；**修法 = 与 C13 一致，把计数改为先剥注释再计**（`stripCommentsKeepLines` 已有同款实现，见 `css-contract-check.js:123`） | Phase 91 / 92（**本 phase 只登记不动 C11**：`IMPORTANT_BUDGET` 保持 1397 不动） |
| **D-j** | **两处输入类圆角已去 important → 圆角改由 antd `components.Input.borderRadius` 决定**，若某页面**完全没有 antd Input 组件**则 antd cssinjs 不注入 Input 样式 → 裸 `.ant-input` 圆角退回 initial(0) | W1 实测：22 形态矩阵（明暗 456 格）+ 14 页巡检全部处于 antd 组件上下文，**零命中**；风险窗口 = 「纯裸 `<input class="ant-input">` 且页面无任何 antd Input」的极端场景；**修法 = 92 终态盘点时确认 antd cssinjs 注入时机，或回滚两处 important** | Phase 92（终态复核） |

---

## W1 落地记录（90-02 · 批次 C + 批次 B + VISFIX-05 + STYLE-03 + M9 压缩 · **单一 commit**）

> **行号基准声明（防 91/92 被漂移行号误导）**：本节全部 `:NNN` 行号 = **W1 前 HEAD 行号（`git show HEAD:./src/index.css`，5555 行）**，**仅作参考**。
> **W1 的行号漂移公式（自 W1 前 HEAD 起算）**：
> ① 批次 C 拆 `:883` 一行 → 三行 → `:884` 及其之后 **+2**；
> ② 复位块注释重写（6 行 → 12 行）→ **+6**；
> ③ VIS-03 注释块重写（11 行 → 15 行）→ **+4**；
> ④ VISFIX-05 新块（注释 9 + error 7 + warning 7 = 23 行）→ **+23**；
> ⑤ M9 空行压缩（删 39 处块内空行，落点行号 **6…298**，全在该区间内）→ 被删点之后 **−39**。
> ⇒ **净漂移 = +2 +6 +4 +23 −39 = −4**（`:883` 之前的行为 0；`:883` 之后且 > 298 的行号 **−4**；`:298` 之前的行号按各自删点位置浮动）。
> ⇒ **权威标识一律是「选择器锚点」**，行号仅供 91/92 复核时快速定位。

### ① 批次 C：拆 `border` 简写（唯一拆 `:883` 一处）

| 项 | 内容 |
|---|---|
| 位置 | `.ant-input, .ant-input-number` 块（W1 前 HEAD `:880-892`） |
| 落地 | `border: 1px solid var(--input-border) !important` → 三条 longhand：`border-width: 1px` / `border-style: solid` / **`border-color: var(--input-border) !important`**（**仅 color 保 important**，width/style 交还 antd variant 机制） |
| M1 / M2 影响 | **±0**（1 个 important → 1 个 important） |
| 视觉等价实测 | `borderTopWidth = 1px` / `borderTopStyle = solid` / `borderTopColor = rgb(219, 215, 206)`（light）—— 与 W0 基线**逐字相等**（证据 `probe90-light.json` / `after-light.json`） |
| 未拆的 wrapper 简写 | `.ant-input-affix-wrapper` 的 `border` 简写**保持原样**（`css-contract-check.js:76` 的 **C4b** 以字面正则守护）→ 登记为债 **D-f**（91/92 与 D-c 同批处置） |

### ② 批次 B（= 留 · 甲方案）：复位块本体保留 + 注释重写

- **本体零调整**：`.ant-input-affix-wrapper > input.ant-input, .ant-input-affix-wrapper > textarea.ant-input` 的六项复位（`border: none` / `border-radius: 0` / `background: transparent` / `color: inherit` / `font-size: inherit` / `padding: 0`）**逐字未动** → **C1a / C1b / C1c 零调整**（`git diff` 中 C1 块无 +/- 行）。
- **注释重写**（两处，均只改注释不动规则）：
  - 复位块注释：去掉全部绝对行号（`:871` / `:916-929` / `:2252-2284` / `:3717-3757`），改用「章节名 / 选择器」锚点；新增四点内容（为何仍需保留 / 批次 C 已落地与因果链 / 故意不写 box-shadow —— 焦点环归 wrapper / 本块在 91/92 之前不得删除 + D-c 指针）。
  - **VIS-03 注释块一并重写**：原 `:871` / `:910` 的过期行号与「border 简写」过期语义全部清除；改为「border 已拆为 `border-width` / `border-style` / `border-color` 三属性，仅 color 保留 important」；**Phase 77 事实说明**（status class 打在 wrapper 自身 / 真实 DOM / antd 规则形态）与「只覆盖 Input 域，不动 Select/DatePicker」边界句**保留**；`/* VIS-03` 起始标识保留。
- **两条注释块均零 6 位 hex**（C13 多行注释误计入防线）且**零 `!important` 字面**（D-i 口径规避，见 ⑥）。

### ③ VISFIX-05：InputNumber status 边框色还原（error / warning 各 6 选择器）

- 落地形态（紧贴 VIS-03 warning 规则块插入，**块内零空行** → 块间单空行计数恒 801）：
  - error 组 6 选择器 = 常态 / `:hover` / `:focus` × 明暗（light 3 + `[data-color-mode="dark"]` 3）→ `border-color: var(--ant-color-error) !important`
  - warning 组 6 选择器 = 同构 → `border-color: var(--ant-color-warning) !important`
- **色值口径（mode-aware，明暗双测）**：`--ant-color-error` 在 cssVar scope 内 light = `#ba3630` → **`rgb(186, 54, 48)`**；dark = `#a1312c` → **`rgb(161, 49, 44)`**（`darkAlgorithm` 对 seed 名 `colorError` 的派生值，**不是** `#BA3630`）。warning：light `#b07a20` → `rgb(176, 122, 32)`。
- **实测（探针 `probe90-light.json` / `after-{light,dark}.json`）**：error 态 `borderTopColor` = light `rgb(186, 54, 48)` / dark `rgb(161, 49, 44)`；warning 态 = light `rgb(176, 122, 32)`；**与登记的 b 类 after 值逐字相等**。
- **契约**：新增 **C16a / C16b / C16c** 三条静态断言防回潮（C16c 用非贪婪不限长 `[\s\S]*?\{[\s\S]*?var\(…\)` → 对多行选择器排版免疫）；既有断言（C1a/C1b/C1c/C3a-c/C4a-c/C5a-c/C11/C13）**零改动**。
- **边界**：只覆盖 `.ant-input-number` 域，**不碰** Input 域的 VIS-03 规则本体、不碰 Select / DatePicker。

### ④ STYLE-03：圆角 6 → 8（三处）

| 位置（W1 前 HEAD） | 选择器锚点（权威） | 变更 |
|---|---|---|
| `:885` | `.ant-input, .ant-input-number` | `var(--theme-radius-md) !important` → **`var(--theme-radius-lg)`**（**去 important**，见下方裁定） |
| `:921` | `.ant-input-affix-wrapper` | `var(--theme-radius-md) !important` → **`var(--theme-radius-lg)`**（**去 important**，见下方裁定） |
| `:5489` | `.ant-pagination .ant-pagination-item` | `6px !important` → **`var(--theme-radius-lg) !important`**（**保留 important**：K11 刻意设计的透明无边框分页项，`border: none !important` 同块存在，去 important 会让 antd 分页边框长回来） |

> **⚠️ 裁定（PLAN 假设被实跑证据推翻 · 执行期自行裁定）**
> **PLAN 原文**：90-02 任务 2.4 要求三处圆角「**保留 `!important`**」（理由：最小 diff）。
> **推翻依据（实跑证据）**：
> ① **antd token 已是真相源** —— `AntdThemeBridge.tsx:147` 的 `components.Input.borderRadius = 8`，与 `--theme-radius-lg = 8px` **完全一致**；项目手写 `!important` 属**对抗 antd token**，与 Phase 90 主责（STYLE-02「把手写 `!important` 收敛到 antd 6 官方 variant/token 机制」）方向相反；
> ② **视觉零漂移实测** —— 去掉两处 `!important` 前后**各采一次完整 after 快照**（明暗各 22 形态 × 2 part × 6 属性 = 456 格），两次差异集合**逐格完全相同**（38 格 / light 19 / dark 19 / 归类 38 / 未归类 0）→ 圆角交还 antd token 后 computed 值零变化；
> ③ **内层 radius 不受影响** —— wrapper 内层由 VIS-01 复位块的 `border-radius: 0 !important` 独占；f11 / f22 的 `.ant-input-number-input` 内层 0px 由 antd 内建复位保证（矩阵 f11/f22 inner 无差异）；
> ④ **M1 账** —— 去 important 贡献 M1 / M2 各 **−2**，是「M1 不升」得以成立的**真实声明级**抵消（区别于 D-i 的注释字面规避）。
> **裁定**：**移除两处 `border-radius` 的 `!important`（分页器保留）**，圆角值改由 antd `components.Input.borderRadius` 决定 —— 这正是 STYLE-03「输入类组件圆角统一由 6px 收敛到设计 token 的 8px」的**机制语义**（不是硬写一个 8px 字面）。
> **残留风险（登记为债 D-j）**：若某页面**完全没有 antd Input 组件**，antd cssinjs 不会注入 Input 样式 → 裸 `.ant-input` 的圆角会退回 initial(0)。当前 22 形态矩阵 + 14 页巡检全部处于 antd 组件上下文，**零命中**。

### ⑤ M9 预算腾挪：纯冗余空行压缩（同 commit 内）

| 项 | 值 |
|---|---|
| **唯一硬判据** | **M9 = (`wc -l src/index.css`) + 1 ≤ 5558**（checker 口径 `split('\n').length`；**M9 = `wc -l` + 1，两口径不得混用**） |
| 压缩前 | `wc -l` = 5552 → **M9 = 5593** |
| **G**（本轮非压缩改动净增） | **+37** = 批次 C 拆行 (+2) + 复位块注释重写 (+6) + VIS-03 注释重写 (+4) + VISFIX-05 新块 (+23) + 圆角值改写 (±0) |
| **D**（删除的块内空行数） | **39**（由 M9 判据反推：最终 M9 = 5556 + G − D ≤ 5558 ⟺ **D ≥ G − 2 = 35**；取 **D = 39**（余量 4）） |
| 压缩后 | `wc -l` = 5553 → **M9 = 5554 ≤ 5558** ✅ |
| 约束① D ≤ 58 | **39 ≤ 58** ✅（压缩前实测块内空行 59 处，含 VIS-03 注释块内新增的 1 处） |
| 约束② 被删行全部为块内空行 | ✅ 逐行校验（判定口径 = 该空行**前后两个非空行缩进均 > 0**） |
| 约束③ 块间单空行计数 | **801 → 801 不变** ✅（压缩前后实测） |
| 禁区 | 暗色只读区（`:2254-2742` / `:2780-4517`）内 **零处**被删（压缩前实测 59 处块内空行落在只读区内的 = 0）；六处非输入域 6px 未碰；**一行 CSS 声明与注释内容都未改**（只删空行） |
| **`C13_M9_BUDGET`** | **仍为 5558**（`git diff` 中该常量行无 +/-）✅ |
| **被删行号清单（删除前，1-based）** | `6, 12, 17, 35, 47, 60, 70, 74, 82, 90, 93, 99, 109, 118, 126, 134, 139, 144, 165, 172, 178, 182, 186, 214, 221, 229, 237, 243, 249, 254, 260, 268, 274, 278, 284, 287, 290, 295, 298`（**共 39 行，全部落在文件头 `:6`–`:298` 区间的块内空行**；复核脚本 `90-visual-evidence/p90-compress.cjs` 逐条自检「非空行 / 非块内 / 落在暗色只读区」三类禁区） |

### ⑥ M1 / M2 净变化分解（W0 起点 → W1 后）

| 指标 | W0 起点 | W1 后 | 净变化 | 分解 |
|---|---|---|---|---|
| **M1**（全 `.css/.less` `!important`） | 1397 | **1392** | **−5** | 批次 C 拆简写 **±0**；两处圆角去 important **−2**；VISFIX-05 两条新规则体 **+2**；**注释字面规避（D-i 口径）** −5 |
| **M2**（`index.css` `!important`） | 1330 | **1325** | **−5** | 同上（本 phase 只改 `index.css`，故 M1 与 M2 净变化相等） |
| **M7**（`index.css` 6 位 hex） | 304 | **304** | **0** | 零新 hex（色值全部走 `var(--ant-color-error)` / `var(--ant-color-warning)`；新增注释块零 6 位 hex） |
| **M8**（全 `.css/.less` 6 位 hex） | 440 | **440** | **0** | 同上 |
| **M9**（`index.css` 行数） | 5556 | **5554** | **−2** | G(+37) − D(39) |

> **C11 ratchet 判定**：M1 = 1392 ≤ **1397** ✅（只降不升，未上调预算）。
> **口径澄清（重要，防 91/92 误读）**：M1 的 −5 里，**只有 −2 是真实声明级收敛**（两处圆角交还 antd token）；另 **−5 是 D-i 的注释字面规避**（CSS 语义零变化），**+2 是 VISFIX-05 的功能性新增**。净 −5。91/92 继续按 M1 ≤ 1397 的 ratchet 推进即可。

### ⑦ W1 顺带修正：`matrix-diff-90.cjs` 的结果文件写入 bug

- **现象**：脚本的判定逻辑（`classified` / `unclassified` 两个数组）**一直正确**，但写 `matrix-diff-result-90.txt` 时用了 `diffs.map(d => d.cls ? … : 'UNCLASSIFIED ')` —— `diffs` 数组里的对象**没有 `cls` 属性**（`cls` 只加在 `classified` 数组的副本上）→ **结果文件把全部差异都写成 `UNCLASSIFIED`**，与真实判定（`exit 0` / 未归类 0）矛盾 → **证据文件误导读者**。
- **修正**：改写为「`classified.map(d => '['+d.cls+'] '+d.text)` ++ `unclassified.map(d => 'UNCLASSIFIED '+d.text)`」，结果文件现在逐条带正确分类标签。
- **未放宽任何归类逻辑**：`A_CELLS`（18 格）/ `B_CELLS`（1 格）/ `C_CELLS`（空集）/ `B_AFTER`（mode-aware）**一字未改**，归类仍是**逐格精确**（`prop === 'radius' && before === '6px' && after === '8px' && A_CELLS.has(...)` → 非白名单格一律判未归类）。

---

## 终判结论（由 90-04 回填）

- **矩阵版本**：`matrix-diff-90.cjs`（白名单归类模式，登记表 = 本文件 a/b/c 三节）+ `matrix-fixture.js`（22 形态）/ `matrix-sample.js`（6 属性）；采样方法 = `p90-run.cjs`（dev :4000 + `$TEMP/p75-uat` 后端 + Node WebSocket 直连 CDP；模式切换经 `/user/settings` 分段卡走 **settingsStore 服务器偏好**）。
- **终跑结果（待填）**：____ 格；light = ____ / dark = ____；未归类 = ____；exit ____。
- **a 类格数（待填）**：____ 格（预期 18 × 2 = 36）。
- **b 类格数（待填）**：____ 格（预期 1 × 2 = 2）。
- **c 类格数（待填）**：____ 格（预期 0 —— 除非 90-03 的 T4 裁定改写本表）。
- **证据文件名（待填）**：`after-{light,dark}.json` / `matrix-diff-result-90.txt` / `inspection-result-90.txt` / `probe-{light,dark}.txt` / `probe-status-before.json`。
- **执行人（待填）**：90-04 executor（GSD）。

---

## W2 补记（Plan 03 / 2026-10-29）—— R1-R5 保留论证 + T4 裁定 + 圆角归属 + 债表定稿

### ① R1-R5 保留论证表（色值级论证 —— 每条附具体数字，不接受「不等价」三字而无数字）

> **判据**：项目手写变量与 antd token **逐值相等**才可交还；不等价的必须**先对齐 token 再删规则**。
> **R1（批次 C）已在本 phase 前波（W1，commit `89c2f62d`）完成** → 不再列为待办。

| 项 | 规则位置（选择器锚点为权威） | 现状值 | 交还目标与取值 | 等价性（含数字） | 裁定 | 债 |
|---|---|---|---|---|---|---|
| **R2 hover** | `.ant-input:hover` / `.ant-input-number:hover` | `border-color: var(--input-border-hover)` = light `#598e5e` = `rgb(89,142,94)` / dark `#598e5e` = **同值** | `components.Input.hoverBorderColor` = `ruoyiBrand.greenPrimary` = **`#156031`** = `rgb(21,96,49)` | **不等价** —— `#598e5e`(89,142,94) vs `#156031`(21,96,49)，绿通道差 46、蓝通道差 45 | **保留** | **D-a** |
| **R3 focus 边框** | `.ant-input:focus` / `.ant-input-number:focus` | `var(--input-border-focus)` = light `#156031` = `rgb(21,96,49)` / dark `#1a6839` = `rgb(26,104,57)` | `components.Input.activeBorderColor` = `greenPrimary` = `#156031` = `rgb(21,96,49)` | light **等价**（21,96,49 = 21,96,49）；dark **不等价**（26,104,57 vs 21,96,49，绿通道差 5、蓝通道差 8） | **保留**（暗色不等价） | **D-a** 同批 |
| **R3/K4 焦点环** | `.ant-input:focus` 的 `box-shadow` | `0 0 0 2px var(--theme-primary-lighter)` = `#e9efeb` = **不透明** `rgb(233,239,235)` | `components.Input.activeShadow` = `0 0 0 2px rgba(21,96,49,0.15)` = **15% 透明** | **不等价** —— 不透明 vs 15% alpha，**透底色不同 = 可见差异** | **保留**（K4 契约，C4a 守护） | — |
| **R5 wrapper hover** | `.ant-input-affix-wrapper:hover` | `var(--input-border-hover)` = `#598e5e`（同 R2） | 同 R2 | **不等价**（同 R2） | **保留** | **D-a** 同源 |
| **K5 wrapper 焦点环** | `.ant-input-affix-wrapper-focused` | `0 0 0 2px var(--theme-primary-lighter)` = `rgb(233,239,235)` 不透明 | 同 R3 焦点环 | **不等价** + VIS-02「焦点环归 wrapper」契约 | **保留**（C4c 守护） | — |
| **K6/K7 暗色焦点环** | `[data-color-mode="dark"] .ant-input:focus` / `.ant-input-affix-wrapper-focused` | 双层环 `0 0 0 2px rgba(21,96,49,.15)` + `0 0 12px rgba(21,96,49,.1)`（实测 `rgba(21,96,49,0.15) 0px 0px 0px 2px, rgba(21,96,49,0.08) 0px 0px 20px 0px`） | antd **无等价物**（无 12px 扩散环） | **不等价** —— 视觉扩散环无 token 对应物 | **保留** | — |
| **底色** | `.ant-input` / `.ant-input-number` 的 `background` | `var(--input-bg)` = light `#fafaf9` = `rgb(250,250,249)` / dark `#0f2e1b` = `rgb(15,46,27)` | `colorBgContainer` = light `#FFFFFF` / dark `#1a2e1f` = `rgb(26,46,31)` | **不等价** —— light 差 `#fafaf9` vs `#FFFFFF`（各通道差 5）；dark 差 `#0f2e1b` vs `#1a2e1f`（绿通道差 11） | **保留**（VISFIX-01 已把暗色 `--input-bg` 对齐页面底，再交还 = **白色底回归**） | — |
| **文字色** | `.ant-input` 的 `color` | `var(--input-text)` = light `#101010` = `rgb(16,16,16)` / dark `#f0ece3` = `rgb(240,236,227)` | `colorTextBase` = `board.fg` = light `#f0ece3` / dark `#f0ece3` | **不等价** —— light `#101010`(16,16,16) vs `#f0ece3`(240,236,227)，**亮度差 224** | **保留** | — |
| **disabled** | 暗色块 `[data-color-mode="dark"]` 的 `--input-bg-disabled` / `--input-border-disabled` / `--input-text-disabled` | 暗色 disabled 三项由项目变量另行承接 | antd `colorBgDisabled` / `colorBorder` / `colorTextDisabled` | **不等价** —— light disabled 边框 `#dbd7ce` → 交还后为 `cream.borderStrong` = `#C2BDB2` = `rgb(194,189,178)`，**各通道差 25/26/28** | **保留** | **D-b**（此为**必然登记变更**） |

**K1-K11 本 phase 现状标注**：

| K | 含义 | 本 phase 现状 |
|---|---|---|
| K1 | VIS-01 内层复位块 | **保留**（批次 B = 留，甲方案；W1 只重写注释，本体六项逐字未动） |
| K2/K3 | VIS-03 status 还原 | **保留并扩展到 InputNumber 域**（W1 新增 error/warning 各 6 选择器，C16a/b/c 守护） |
| K4-K7 | 焦点环（4 处） | **保留**（色值级论证见上表） |
| K8/K9 | `.user-form-input` 活补丁 | **未触碰** |
| K10 | login 复位块 | **未触碰**（独立 `login.css`） |
| K11 | 分页器刻意设计 | **保留**且仅改圆角（6px → 8px） |

---

### ② T4 裁定：删 `font-size: 14px !important` —— **采纳（删除）**

**裁定输入一：grep 结论** —— 全仓 `size="large"` 命中逐处归类后，**没有一处是普通 `<Input size="large">`**（多数为 `<Spin>` / `<Space>` / `<Drawer>` / `<Button>`，与 Input 字号无关）；与 Input 字号真正相关的仅 3 处：`GlobalSearch.tsx:239`（`<Search size="large">`）、`knowledge/view/index.tsx:192`（`<Search size="large">`）、`login/index.tsx:264`（`<Form size="large">`）。

**裁定输入二：机制分析（逐层推演）** ——
① 复位块 `.ant-input-affix-wrapper > input.ant-input { font-size: inherit !important }`（特异性 (0,2,1)）**压过** `:886` 的 (0,1,0)+important → **wrapper 内层 input 的字号由 wrapper 决定**，不依赖 `:886`；
② 普通 `<Input>`（非 wrapper）：antd 自身 `fontSize` token = 14px → 删 `:886` 后仍是 14px；
③ InputNumber：`:886` 只命中**根** `.ant-input-number`，而内层 `.ant-input-number-input` 的字号由 antd 的 `resetComponent` 承担（SM 14 / LG 16）→ 不依赖 `:886`；
④ 登录页：`login.css` K10 复位块独立接管。
⇒ 预期视觉差异 = **0**（登记册 #5/#6 的病灶实际被复位块提前消化，只剩「源码层面仍锁死字号」这一债务）。

**裁定输入三：实测复核（已真跑浏览器探针，非只推演）** ——

| 实测项 | 方法 | 结果 |
|---|---|---|
| 矩阵 `fontSize` 维度 | `p90-matrix-run.cjs` 采 before/after 两版矩阵 → 逐格比对 | **fontSize 维度差异 = 0**（light / dark 两侧均为 0；其他维度差异 19 = a/b 两类登记变更） |
| 明暗双侧 fontSize | `probe-{light,dark}.txt` | light `14px` / dark `14px`，**前后逐字相等** |
| 权威断言片段 | `p90-contract-assert.cjs`（复用 `scripts/css-contract-assertions.js` 的判定式） | `passed = true`（双框 0 / 内层 shadow none / 焦点环单圈 / 错误态 `rgb(186,54,48)`） |

**裁定结论：采纳（删除 `:886`）**，`c` 类（font-size 维度）**登记为空集**。

**量化影响**：M1 **1392 → 1391**（≤ ratchet 1397 ✅）；M2 **1330 → 1329**；M9 **5554 → 5553**（≤ 5558 ✅）。（三项均经 C13 实跑复核）

---

### ③ 圆角归属清单

> **⚠️ 行号基准声明（必读，防 91/92 被漂移行号误导）**：
> 本清单的行号一律标注为「**W1 前 HEAD 基准，仅参考**」。漂移公式：**① W1 批次 C 拆 `:883` 一行 → 三行**，使其后 **+2**；**② W1 空行压缩**（删块内空行 D 行，D ≤ 58 且由 M9 判据反推，无绝对值口径）使被删空行之后 **−D**；**③ W2 T4 删 `:886` 一行**使其后 **−1**。三者叠加后的实际行号**不稳定**（取决于压缩落点）→ **清单的权威标识是「选择器锚点」，行号只作参考**。
> **本表所列行号为 W2 执行期的实测值**（= W1 后 + T4 后），已与下方锚点一一对应。

#### 3.1 六处字面 `6px`（非输入域，本 phase **不改**，登记）

| 行号（W2 实测） | 选择器锚点（权威） | 域 | 是否输入域 | 本 phase 处置 |
|---|---|---|---|---|
| `:4455` | `.modern-tag-success` | Tag | 否 | 不改（登记，债 **D-e**） |
| `:4467` | `.modern-tag-error` | Tag | 否 | 不改（登记，债 **D-e**） |
| `:4479` | `.modern-tag-warning` | Tag | 否 | 不改（登记，债 **D-e**） |
| `:4491` | `.modern-tag-default` | Tag | 否 | 不改（登记，债 **D-e**） |
| `:4503` | `.modern-tag-processing` | Tag | 否 | 不改（登记，债 **D-e**） |
| `:5231` | `.xr-dept-panel .ant-tree .ant-tree-node-content-wrapper` | Tree | 否 | 不改（登记，债 **D-e**） |

#### 3.2 `--theme-radius-md`（6px）的 15 处非输入域消费（13 CSS + 2 TSX，本 phase **全部不动**）

| 行号（W2 实测） | 选择器锚点（权威） | 域 | 处置 |
|---|---|---|---|
| `:653` | `.ant-btn` | Button | 不动 |
| `:812` | `.ant-btn-group > .ant-btn:first-child`（`border-top-left-radius`） | Button | 不动 |
| `:813` | `.ant-btn-group > .ant-btn:first-child`（`border-bottom-left-radius`） | Button | 不动 |
| `:817` | `.ant-btn-group > .ant-btn:last-child`（`border-top-right-radius`） | Button | 不动 |
| `:818` | `.ant-btn-group > .ant-btn:last-child`（`border-bottom-right-radius`） | Button | 不动 |
| `:1177` | `.ant-modal-close-x:hover` | Modal | 不动 |
| `:1190` | `.ant-dropdown-menu` | Dropdown | 不动 |
| `:1363` | `.ant-pagination-item` | Pagination | 不动（**实测不生效** —— 见 3.3） |
| `:1427` | `.ant-tooltip-inner` | Tooltip | 不动 |
| `:1501` | `.ant-message-notice-content` | Message | 不动 |
| `:1518` | `.ant-notification-notice` | Notification | 不动 |
| `:1770` | `.ant-popover-inner` | Popover | 不动 |
| `:5275` | `.xr-settings-nav-item` | Settings | 不动 |
| `GlobalSearch.tsx:281` | TSX 内联 `borderRadius: "var(--theme-radius-md)"` | GlobalSearch | 不动 |
| `GlobalSearch.tsx:298` | TSX 内联 `borderRadius: "var(--theme-radius-md)"` | GlobalSearch | 不动 |

**合计 15 处**（13 CSS + 2 TSX）。研究 §B-6 / §C-1 记「10 处」，**以实测 15 处为准**。

#### 3.3 顺带核实（防回归）

`.ant-pagination .ant-pagination-item` (0,2,0)+`!important` **压过** `.ant-pagination-item` (0,1,0)+`!important` → **分页器圆角实测恒为 8px**（两条规则并存不冲突）。W2 实测：分页器 item 的 `borderTopLeftRadius` = **8px**，而 `:1363` 的 `--theme-radius-md`（6px）**实际不生效**。

#### 3.4 M15 口径更新（90 后真实值）

> **M15 = 「输入类（`.ant-input` / `.ant-input-number` / `.ant-input-affix-wrapper`）+ 分页器 = 8px」**；`6px` **字面剩余 6 处**（Tag 5 + Tree 1，**均非输入域**，已逐处登记）；`--theme-radius-md` 的 **15 处**非输入域消费**全部未动**。

---

### ④ 未清偿债（移交 91/92）—— 定稿 **D-a ~ D-g 七条**

| 债 | 描述 | 量化 / 证据 | 归属 |
|---|---|---|---|
| **D-a** | `Input.hoverBorderColor` 需对齐 `#598e5e` 后才可删手写 hover 规则 | Bridge 现为 `greenPrimary #156031` = `rgb(21,96,49)` ≠ 项目 `--input-border-hover #598e5e` = `rgb(89,142,94)`（绿通道差 46） | Phase 91/92 |
| **D-b** | `colorBorder` 交还时 light disabled 边框 `#dbd7ce` → `#C2BDB2` 是**必然登记变更** | `cream.borderStrong = #C2BDB2` = `rgb(194,189,178)`；暗色 disabled 由 `--input-border-disabled` 另行承接 | Phase 91/92 |
| **D-c** | **乙方案路径图**：去 important 全交还（`border-color` 去 `!important`）+ 删 VIS-01 复位块 + **C1a 断言反转为「复位块不得存在」** | 前提：antd variant/token 全面接管且矩阵 diff 可控；顺序 = 先批次 C 拆简写（**已完成**，W1）→ 再删复位块 → **同 commit** 反转 C1a | Phase 91/92（**不得在 90 做**） |
| **D-d** | `--theme-radius-md`（6px）非输入域消费 **15 处** + 源头 `:129` 的最终废存 | 消费面见 3.2（以**选择器锚点**为权威）；波及 Button / Modal / Dropdown / Pagination / Tooltip / Message / Notification / Popover / Settings / GlobalSearch 十个域 | Phase 92 |
| **D-e** | 六处字面 `6px`（Tag 5 + Tree 1）的归属与是否统一到 8px | 见 3.1 归属清单（以**选择器锚点**为权威） | Phase 92（Tag/Tree 域交还时一并处置） |
| **D-f** | wrapper 的 `border` 简写（`.ant-input-affix-wrapper`）未拆（因 C4b 断言依赖字面 `border: 1px solid var(--input-border)`） | C4b 正则见 `scripts/css-contract-check.js` | Phase 91/92（与 D-c 同批：拆简写 → **同 commit** 改 C4b 断言口径） |
| **D-g** | **C13 的 `stripComments` 用 `/\/\*.*?\*\//g`，`.` 不跨行 → 多行注释块内的 6 位 hex 被误计入 per-file 基线** | 实测：`index.css` raw hex **311** vs strip 后 **304**（差 **7** = 7 处多行注释内 hex 被计入）；影响 `src/index.css` 的 per-file 基线 **304 硬上限**（多一个注释 hex 即 305 → C13 FAIL）；**修法 = 把 `stripComments` 改成跨行版 `/\/\*[\s\S]*?\*\//g`** | Phase 91/92（**本 phase 不动 C13**：`C13_M9_BUDGET` 保持 5558、`stripComments` 一行不改；90 侧以「新增注释块零 6 位 hex」规避） |

---

### ⑤ 本 phase 明确不做（对齐研究 §B-1 ~ B-11）

不做 Select 域｜不删暗色 mega-cluster（`:2254-2742` / `:2780-4517` 只读）｜不做页面拆分｜不动 `AppInput` / `AppSelect` / `AppTextArea` 默认 props（P-M7 冻结）｜不碰 `src/components/ConfigProvider.tsx`（P-C2 冻结）｜不扩 Bridge `useMemo` deps（C15 `length === 2`）｜**零新依赖**（不引 stylelint / Playwright / jscodeshift）｜**不迁 `onSearch`**（antd 6.6.4 未弃用）｜不删 VIS-01 复位块本体｜不改 `permissionCodes.ts`（属 Phase 86 线）｜不改任何 `.tsx` 组件逻辑（属 Phase 86 线）。

---

### ⑥ 与 86 线的文件边界确认

86 动 `src/pages/**/index.tsx`（76 页 JSX）+ `useActionPermission` + `scripts/check-gating-coverage.mjs`；90 动 `src/index.css` + `scripts/css-contract-check.js` + `90-visual-evidence/` + `.planning` 文档 → **文件面零交集**，两线可全程并行。
**唯一注意**：探针取样页（`/system/user`、`/knowledge/view`）若正被 86 改写，需先与 86 线确认该页处于可渲染态。
