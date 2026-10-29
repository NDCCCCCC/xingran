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

---

## 终判结论（由 90-04 回填）

- **矩阵版本**：`matrix-diff-90.cjs`（白名单归类模式，登记表 = 本文件 a/b/c 三节）+ `matrix-fixture.js`（22 形态）/ `matrix-sample.js`（6 属性）；采样方法 = `p90-run.cjs`（dev :4000 + `$TEMP/p75-uat` 后端 + Node WebSocket 直连 CDP；模式切换经 `/user/settings` 分段卡走 **settingsStore 服务器偏好**）。
- **终跑结果（待填）**：____ 格；light = ____ / dark = ____；未归类 = ____；exit ____。
- **a 类格数（待填）**：____ 格（预期 18 × 2 = 36）。
- **b 类格数（待填）**：____ 格（预期 1 × 2 = 2）。
- **c 类格数（待填）**：____ 格（预期 0 —— 除非 90-03 的 T4 裁定改写本表）。
- **证据文件名（待填）**：`after-{light,dark}.json` / `matrix-diff-result-90.txt` / `inspection-result-90.txt` / `probe-{light,dark}.txt` / `probe-status-before.json`。
- **执行人（待填）**：90-04 executor（GSD）。
