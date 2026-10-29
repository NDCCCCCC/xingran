# Phase 90 量化快照

> 本文件为 **Phase 90 W0 起点（零业务代码改动）** 的量化基线。口径与 `89-QUANT-SNAPSHOT.md` 完全一致（Node 正则计数），**不得改口径**。
> 校验基线：`node scripts/css-contract-check.js` → **ALL PASS / exit 0**（C1a/C1b/C1c/C2/C3a-C3c/C4a-C4c/C5a-C5c/C6/C8/C11/C12/C13/C14/C14b-g 全 PASS）。
> 实测日期：2026-10-29（Phase 90-01 executor，W0）。

---

## 一、W0 起点（实测，2026-10-29）

| 指标 | 口径 | 89 终值 | **90 W0 实跑** | 与 89 终值差异 | 说明 |
|---|---|---|---|---|---|
| **M1** | 全 `.css`/`.less` `!important` 出现次数 | 1397 | **1397** | **0** | C11 ratchet = 1397（`IMPORTANT_BUDGET`），实测恰好贴顶 |
| **M2** | `index.css` `!important` 出现次数 | 1330 | **1330** | **0** | |
| **M3** | 含 `.ant-` 顶层规则 | 725（index 702） | 沿用 89 终值 | 0 | 89 未动规则级增删；90 W0 亦未动任何规则 |
| **M4** | 暗色顶层规则 | 569 / 502 | 沿用 89 终值 | 0 | 同上 |
| **M5** | 暗色分区 | ≈64 | 沿用 89 终值 | 0 | 同上 |
| **M6** | 重复暗色 class | ≈80 | 沿用 89 终值 | 0 | 同上 |
| **M7** | `index.css` 6 位 hex（C13 口径：逐行剥离 `/* */` 与 `//` 注释后计数） | 304（去重 64） | **304**（去重 64） | **0** | C13 预算 310；raw（未剥注释）= 311，差 7 来自多行注释块内的 hex |
| **M8** | 全 `.css`/`.less` 6 位 hex（C13 口径） | 440 | **440** | **0** | C13 预算 452 |
| **M9** | `index.css` 行数 | 5556 | **5556** | **0** | **口径 = `split('\n').length`；等价 `wc -l` = 5555；换算式 `M9 = wc -l + 1`**（详见第二节） |
| **M10** | `index.css` 分区块 | ≈97 | 沿用 89 终值 | 0 | 同 M3-M6 理由 |
| **M11** | 暗色 `--input-bg` | `#0f2e1b` | **`#0f2e1b`** | **0** | W0 实测（探针 `probe-dark.txt` → `inputBgVar`）；VISFIX-01 交付物保持 |
| **M12** | 暗色 placeholder 变量 | 非空 | **非空**（`--input-text-placeholder` = `#707068`） | **0** | W0 实测；R11 交付物保持 |
| **M13** | InputNumber 错误态边框 | `rgb(219,215,206)` | **`rgb(219, 215, 206)`**（light）<br>**`rgb(45, 77, 51)`**（dark，W0 补充实测） | **0** | `--input-border` light `#dbd7ce`（`index.css:192`）/ dark `#2d4d33`（`index.css:269`）；**这是本 phase 唯一的红→红换色目标**（VISFIX-05） |
| **M14** | 死样式文件 | 0 | **0** | **0** | C8 全绿 |
| **M15** | 圆角 | 6px 系 7 处 | **6px 系 7 处**（`index.css` 逐处实测行号齐全） | **0** | 见下方 M15 明细 |
| **M16** | antd 弃用警告 | 0 | 0（W0 未巡检；沿用 89 终值） | 0 | 14 页 × 明暗巡检收口在 90-04（`inspection-result-90.txt`） |

**结论：90 起点 ≡ 89 终态（M1/M2/M7/M8/M9/M11/M12/M13/M14/M15 全部零差异）** → 90 的终值可完全归因于本 phase 自身的变更。

### M15 明细（`border-radius: 6px` 字面 7 处，`index.css` 逐处实测）

| 行号 | 原文 | 归属 |
|---|---|---|
| `:4458` | `border-radius: 6px !important;` | 暗色/日期区段（待 90-02 逐处判归属组件，债 D-e） |
| `:4470` | `border-radius: 6px !important;` | 同上 |
| `:4482` | `border-radius: 6px !important;` | 同上 |
| `:4494` | `border-radius: 6px !important;` | 同上 |
| `:4506` | `border-radius: 6px !important;` | 同上 |
| `:5234` | `border-radius: 6px;`（无 important） | 待 90-02 判归属（债 D-e） |
| `:5489` | `border-radius: 6px !important;` | **分页器**（K11 组）→ 属 STYLE-03 判据口径内，W1 改 6→8 |

> `--theme-radius-md: 6px`（`index.css:129`）与 `--theme-radius-lg: 8px`（`index.css:130`）为 `:root` 静态定义，**无 JS 运行时覆盖** → computed 恒为 8px（W0 探针实测 `radiusMdVar=6px` / `radiusLgVar=8px`，明暗同值）。
> `--theme-radius-md` 的**非输入域消费 15 处**（13 CSS + 2 TSX）**不在 STYLE-03 判据面**，90 不动（债 D-d）。

---

## 二、M9 口径与预算前置约束（W0 只登记，不动手）

> **`M9 = wc -l + 1`** —— checker 口径是 `readFileSync('src/index.css').split('\n').length`（`css-contract-check.js:266`），而 `wc -l` 只计换行符数，两者**恒差 1**（文件以换行符结尾时）。**两口径不得混用。**

| 口径 | W0 实跑值 |
|---|---|
| `split('\n').length`（**checker 口径 / 本文件唯一权威口径**） | **5556** |
| `wc -l` | **5555** |
| 换算式 | **`M9 = wc -l + 1`** |

**预算状态（`C13_M9_BUDGET = 5558`，无容差）**：

- 起点 M9 = 5556 → **只剩 +2 行余量**。
- W1 的批次 C 拆行（+2）、InputNumber status 新块（+26）、复位块 + VIS-03 注释重写（+2~8）净增 **≥ +28** → **不压缩必然超预算 → C13 FAIL → W1 闸门必红**。
- ⇒ **W1 必须在同一 commit 内压缩 `index.css` 的纯冗余空行**。

### W1 后 M9 的唯一判据

> **最终 `M9 = (wc -l) + 1 ≤ 5558`**（等价 `wc -l` ≤ 5557）
> —— **不设任何绝对删除量口径**（绝对删除量在净增 G > 0 时推不出可行解，属错误口径，已作废）。
> 删除量 D 由 M9 判据反推：**最终 M9 = 5556 + G − D ≤ 5558 ⟺ D ≥ G − 2**（建议 D = G − 2 + 4 余量）。

**三条可执行约束**（W0 实测已确认可行空间）：

1. **D ≤ 58** —— 被删行**全部为块内空行**（`index.css` 实测块内空行恰 **58 处**）。
2. 被删行**前后两个非空行的缩进均 > 0**（即位于规则块 / 多行声明内部），**禁改任何 CSS 声明与注释内容**。
3. **块间单空行计数 === 801 不变** —— `index.css` 实测空行共 **859** 处 = 块内空行 **58** + 块间空行 **801**。

**暗色只读区约束（实测已确认）**：

- 暗色只读区 = `:2254-2742` / `:2780-4517`。
- 58 处块内空行中**落在暗色只读区内的 = 0 处** → **全部可删**（不与只读区冲突）。
- 唯一的「连续 ≥2 空行」折叠组位于 **`:3552`（×2）**，落在暗色只读区 **`:2780-4517`** 内 → **不动**（这也保证了「块间单空行 801 不变」在折叠语义上自洽）。

**`C13_M9_BUDGET` 保持 5558 一行不动** —— 压缩是唯一腾挪手段，不改预算。

### 复现命令（`ruoyi-react-frontend/`）

```js
// M1（全 .css/.less）：逐文件 (text.match(/!important/g)||[]).length 求和
// M2（index.css）    ：同上，单文件
// M7（index.css 6 位 hex，C13 口径）：
//   const strip = (line) => line.replace(/\/\*.*?\*\//g,'').replace(/(^|[^:])\/\/.*$/,'$1');
//   (strip(text).match(/#[0-9a-fA-F]{6}\b/g)||[]).length
// M8（全 .css/.less） ：同上，逐文件求和
// M9：text.split('\n').length   ← checker 口径（= wc -l + 1）
// 空行分类（块内 / 块间）：见 §四 的判定口径
```

```bash
cd ruoyi-react-frontend && node scripts/css-contract-check.js; echo "exit=$?"
```

---

## 三、矩阵基线事实（22 形态 before 快照）

> 证据：`90-visual-evidence/before-light.json` / `before-dark.json`（dev :4000 + `$TEMP/p75-uat` 后端 + Node WebSocket 直连 Chrome CDP；模式切换经 `/user/settings` 分段卡走 **settingsStore 服务器偏好**，非仅 `data-color-mode` 属性 —— 89-02 经验沿用）。

| 项 | 值 |
|---|---|
| 形态数 | **22**（89 的 21 形态 + `f22_input_number_status_error`） |
| 采样属性 | **6**（`bg` / `border` / `radius` / `shadow` / `btw` / **`fontSize`** ← 本 phase 新增） |
| inner=null 的形态 | **6**（f01 / f12 / f13 / f16 / f17 / f18 —— f18 的 `input.ant-input` 命中不到，实际只有 `ant-select-input`） |
| **每模式格数** | **(22 形态 × 2 part − 6 个 inner=null 形态) × 6 属性 = 38 × 6 = 228 格** |
| **明暗合计格数** | **456 格**（理论上限 22×2×2×6 = 528，实测 456 —— 与 90-RESEARCH §7.3 一致） |
| `mode` 字段 | `before-light.json` = `light`；`before-dark.json` = `dark`（采样器自带断言，漂移即 exit 1） |
| `fontSize` 实测取值 | **恒为 `14px`**（明暗全部 38 个 part 一致）→ c 类（font-size）的「零差异」预期在基线上成立（最终裁定归 90-03 的 T4） |

### 防串基线核对：与 89 `after-{light,dark}.json` 的 21 形态交集

> 90 的 before 快照**一律在 90 的 fixture 下重采**（**不复制** 89 的 `after-*.json` —— 89 的 21 形态快照缺 f22，复制后 f22 会被判「结构缺失」而误 FAIL）。
> 「与 89 after 的交集逐格全等」= **防串基线核对**，证明 90 起点确实等于 89 终态。

| 项 | 结果 |
|---|---|
| 比对口径 | 交集 **21 形态** × 明暗 × carrier/inner × **5 属性**（`bg`/`border`/`radius`/`shadow`/`btw`；89 采样器未采 `fontSize`，故不含） |
| 格数口径 | **(21 形态 × 2 part − 6 个 inner=null 形态) × 5 属性 = 180 格 / 模式**；**明暗合计 360 格** |
| **核对结论** | **每模式 180 格、明暗合计 360 格逐格全等（0 处差异）** ✅ |
| 形态数校验 | before 两份均为 **22** 形态，`f22_input_number_status_error` 均存在 ✅ |
| f22 落点 | light：`carrier.border = rgb(219, 215, 206)`、`carrier.radius = 6px`；dark：`carrier.border = rgb(45, 77, 51)`、`carrier.radius = 6px` ✅ |

### 状态层基线事实（W0 只读探针，证据 `90-visual-evidence/probe-status-before.json`）

| 探针项 | light | dark |
|---|---|---|
| `--ant-color-error`（cssVar scope 内解析值） | `#ba3630` | **`#a1312c`** |
| `.ant-input.ant-input-status-error` → `borderTopColor`（项目 VIS-03 规则 `:967-974` 命中） | `rgb(186, 54, 48)` | **`rgb(161, 49, 44)`** |
| `.ant-input-number.ant-input-number-status-error` → `borderTopColor`（现状被 `:883` 的 `border` 简写 `!important` 压掉） | `rgb(219, 215, 206)` | `rgb(45, 77, 51)` |
| `.ant-input-number`（普通）→ `borderTopColor` | `rgb(219, 215, 206)` | `rgb(45, 77, 51)` |
| `.ant-input` → `borderTopLeftRadius` / `fontSize` | `6px` / `14px` | `6px` / `14px` |

> ⚠️ **W0 实测裁定（推翻 90-RESEARCH §7.3 的推定）**：研究 §7.3 写「`--ant-color-error` 明暗同值 `#BA3630` → 明暗双测 `rgb(186,54,48)` 成立」，并要求「执行期必须实测后按实测改判据，**不得伪造**」。
> **实测结果：暗色 scope 内 `--ant-color-error` = `#a1312c`（antd `darkAlgorithm` 对 `colorError` 做了暗色派生），不是 `#BA3630`。**
> ⇒ **b 类（InputNumber 错误态红框）的 after 值明暗不同**：light = `rgb(186, 54, 48)`、dark = `rgb(161, 49, 44)`。
> 该裁定已同步落进 `matrix-diff-90.cjs` 的 `B_AFTER` 常量（mode-aware 匹配）。

### 矩阵工具 90 化的自检证据（W0）

以 `node matrix-diff-90.cjs before`（before 与 before 自比）做**防空跑红线自检**：

- 比对单元格总数 = **456**（与 §三 的格数口径一致）；
- 差异数 = 0（light=0 / dark=0）→ **light 与 dark 双侧防空跑红线同时触发** → `exit 1`；
- 同时触发 a / b 两类的「登记落点未在实测中出现」校验 → 证明「形态集合不多不少」校验生效。

> 该自检属**故意 FAIL**（恒等空跑必须红），因此 `matrix-diff-result-90.txt` 未作为 W0 产物保留（由终跑重新生成）。

---

## 四、空行压缩空间实测（W1 腾挪手段的可执行性证据）

| 项 | 实测值 | 判定 |
|---|---|---|
| `index.css` 总行数（`split('\n').length`） | 5556 | — |
| 空行总数 | **859** | — |
| **块内空行**（前一个非空行与后一个非空行缩进均 > 0） | **58** | = **D 的上限** |
| **块间空行**（顶层块之间） | **801** | **必须 === 801 不变** |
| 块内空行落在暗色只读区 `:2254-2742` / `:2780-4517` 内 | **0** | 58 处全部可删 |
| 连续 ≥2 空行的折叠组 | 1 组：**`:3552`（×2）** | 落在暗色只读区 `:2780-4517` 内 → **不动** |

**判定口径（块内 / 块间）**：某空行行号 `i`，向前/向后各找最近的非空行 `p` / `n`；若 `indent(p) > 0 且 indent(n) > 0` → 块内空行，否则 → 块间空行。

---

## 五、W1 后 / W2 后 / W3 终值（三张空表 · 供后续 plan 回填）

### W1 后（批次 C + 批次 B + VISFIX-05 + STYLE-03 + M9 空行压缩，单一 commit）

| 指标 | W0 起点 | 终值 | 变化 | 判定 |
|---|---|---|---|---|
| M1 | 1397 | **1392** | **−5** | ✅ ≤ 1397（C11 ratchet 未上调）；分解 = 批次 C ±0 / 两处圆角去 important −2 / VISFIX-05 两条新规则体 +2 / **注释字面规避（D-i）−5** |
| M2 | 1330 | **1325** | **−5** | ✅ ≤ 1330（本 phase 只改 `index.css`，故与 M1 净变化相等） |
| M7 | 304 | **304** | **0** | ✅ 不增（零新 hex；色值全部走 `var(--ant-color-error)` / `var(--ant-color-warning)`；新增注释块零 6 位 hex） |
| M8 | 440 | **440** | **0** | ✅ 不增 |
| **M9** | 5556（`wc -l` = 5555） | **M9 = 5554**（`wc -l` = **5553**） | **−2** | ✅ **唯一硬判据：最终 M9 = (wc -l) + 1 ≤ 5558**（实测 5554 ≤ 5558）；**G = +37 / D = 39**（D ≥ G − 2 = 35；D ≤ 58 ✅；被删 39 行全为块内空行；**块间单空行计数 801 → 801 不变** ✅；`C13_M9_BUDGET` 仍为 5558 未动 ✅） |
| M13 | `rgb(219, 215, 206)`（light）/ `rgb(45, 77, 51)`（dark） | **light `rgb(186, 54, 48)` / dark `rgb(161, 49, 44)`** | 换色 | ✅ **VISFIX-05 达成**（探针 `probe90-light.json` + 矩阵 b 类实测，明暗双测、期望值按模式不同） |
| M15 | 6px 系 7 处 | **输入类 + 分页器 = 8px**；`6px` 字面剩余 **6 处**（Tag 5 + Tree 1，均非输入域、已登记）；`--theme-radius-md` 非输入域 **15 处**消费未动 | 6→8（3 处） | ✅ **STYLE-03 达成**（`.ant-input` / `.ant-input-number` / `.ant-input-affix-wrapper` / `.ant-pagination .ant-pagination-item` = 8px；wrapper 内层 input 仍 0px） |
| 矩阵终跑格数 | 456 | **456**（before 22 形态 / after 22 形态） | — | ✅ 形态集合 22 = 22，无结构缺失 |
| 矩阵差异分类 | — | **light 19 / dark 19（合计 38）= a 类 36（18/模式）+ b 类 2（1/模式）+ c 类 0** | — | ✅ **未归类 = 0**；exit 0；差异集合与 `90-CHANGESET.md` 的 a/b 登记**不多不少** |

> **W1 的 a/b 类逐格实测（与登记的三方一致核对）**：
> - **a 类 = 18 格/模式**（carrier 14：f01 / f02 / f03 / f04 / f05 / f06 / f07 / f08 / f09 / f10 / f11 / f20 / f21 / f22；inner 4：f05 / f06 / f14 / f15）→ 明暗各 18 格，两模式集合完全相同；
> - **b 类 = 1 格/模式**（`f22_input_number_status_error.carrier.border`；light `rgb(219,215,206)` → `rgb(186,54,48)`；dark `rgb(45,77,51)` → `rgb(161,49,44)`）；
> - **c 类 = 空集**（T4 裁定见 W2 表）。
> - **必须为 0 的四个维度全部为 0**：`bg` / `shadow` / `btw` / `fontSize` → 改动**未外溢**到登记面之外。
> - **证据**：`90-visual-evidence/matrix-diff-result-90.txt`（456 格 / 38 差异 / 归类 38 / 未归类 0）。

### W2 后（T4 裁定 + §B 显式不改清单复核 + 续办清单落盘）

| 指标 | 起点 | 终值 | 变化 | 判定 |
|---|---|---|---|---|
| M1 / M2 | 1397 / 1330 | | | T4 删 `:886` font-size ⇒ M1/M2 各 −1（若采纳） |
| M7 / M8 | 304 / 440 | | | 不增 |
| M9 | 5556 | | | ≤ 5558 |
| c 类（font-size）登记表 | 空集占位 | | | T4 裁定后回填（采纳且零差异 ⇒ 恒空集；采纳且有差异 ⇒ 登记 before/after 与形态集合） |

### W3 终值（90-04 收口）

| 指标 | 起点 | 终值 | 变化 | 判定 |
|---|---|---|---|---|
| M1 / M2 | 1397 / 1330 | | | |
| M7 / M8 | 304 / 440 | | | |
| M9 | 5556 | | | ≤ 5558 |
| M11 / M12 / M14 | 保持 | | | 89 已交付，90 不回退 |
| M13 | `rgb(219, 215, 206)` / `rgb(45, 77, 51)` | | | ✅ VISFIX-05 |
| M15 | 6px 系 7 处 | | | ✅ STYLE-03 |
| M16 | 0 | | | 14 页 × 明暗巡检 `deprecated` 计数 = 0（`inspection-result-90.txt`） |
| 矩阵终跑 | 456 格 | | | light/dark 各自 > 0；未归类 = 0；形态集合不多不少 |
| 证据文件名 | — | | | `after-{light,dark}.json` / `matrix-diff-result-90.txt` / `inspection-result-90.txt` / `probe-{light,dark}.txt` |

---

## 六、复现总览（逐波复跑清单）

```bash
cd ruoyi-react-frontend
node scripts/css-contract-check.js; echo "exit=$?"        # 静态契约闸门（W0/W1/W2/W3 各跑）
node ../.planning/phases/90-input-variant/90-visual-evidence/p90-run.cjs --baseline   # W0/W1 采 before-*.json
node ../.planning/phases/90-input-variant/90-visual-evidence/p90-run.cjs              # W3 采 after-*.json + 巡检截图
node ../.planning/phases/90-input-variant/90-visual-evidence/matrix-diff-90.cjs       # 矩阵终判（默认比 after）
```

*本 phase（90-01 / W0）产出：起点快照 + 矩阵工具三件套 90 化 + f22 形态 + before 基线；**零业务代码改动**（`git diff --stat -- ruoyi-react-frontend/src/` 与 W0 开始时逐字一致）。*
