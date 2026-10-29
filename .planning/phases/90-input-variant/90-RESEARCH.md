---
phase: "90"
slug: "input-variant"
status: researched
researched: "2026-10-29"
---

# Phase 90 — 研究（变量层与明暗色板 → Input 域交还）

> **本文件为两版合并稿**：研究员的续写哨兵（`<!-- gsd:write-continue -->`）曾覆盖前半段，导致「方案主体 A/B/C/D」与「验证/边界/Sources」分处两版。此稿为两版无损合并 + 口径同步（见文末 §7）。
> **口径以 `90-0*-PLAN.md` 为准**（经 planner → checker → fixer 三轮校验）；本文件为依据文档。

## 实现方案建议

### A. 批次 C：拆 `border` 简写（`index.css:880-892`）——本 phase 第一刀

#### A.1 拆解方案（推荐形态：只有 `border-color` 保 important）

```css
/* 拆后（当前 :880-892） */
.ant-input,
.ant-input-number {
  background: var(--input-bg) !important;            /* 保留（§B.2 论证） */
  border-width: 1px;                                 /* ★ 去 important：antd (0,1,0) 同特异性、项目后写胜 */
  border-style: solid;                               /* ★ 同上 */
  border-color: var(--input-border) !important;      /* ★ 唯一保 important 的长属性（压 antd status 色，见 A.3） */
  color: var(--input-text) !important;               /* 保留 */
  border-radius: var(--theme-radius-lg) !important;  /* ★ 6px→8px（STYLE-03，见 §C） */
  font-size: 14px !important;                        /* 保留（T4 判定见 §B.4） */
  transition: … !important;                          /* 保留（T5 已完成） */
}
```

| 声明 | 处置 | 目标 | 理由 |
|---|---|---|---|
| `border: 1px solid var(--input-border) !important`（`:883`） | **拆** | → `border-width` + `border-style`（去 important）+ `border-color`（保 important） | 简写会重置全部 longhand，压掉 antd 的 `border-color` 状态色（STACK A7.4 反模式）。width/style 无争夺者（antd outlined 的 width/style 是 (0,1,0) 普通声明，项目后写胜），**去掉 important 可使 M1/M2 净增为 0**，不击穿 C11 ratchet=1397 |
| `border-color` | **保留 + var 化（现状已是 var）** | 本 phase 保留；**交还 `colorBorder` 留给 91/92**（需先解决 disabled/hover 态的连锁，见 §B.3） | light `#dbd7ce` = `cream.border` = `colorBorder(light)`；dark `#2d4d33` = `dark.border` = `colorBorder(dark)` —— 常态值 89 后已 token 等价，但删除会连带改变 disabled 边框（§B.3），故 90 只拆简写不交还 |
| `background`（`:882`） | **保留** | 不动 | `--input-bg` light `#fafaf9` ≠ `colorBgContainer(light)` `#FFFFFF` —— **不等价**，删除即浅色输入框变纯白（未登记变更）。对齐二者属设计决策（把 `--input-bg` 并入 token 或设回 `components.Input.colorBgContainer`），后者与 89 X-1b 方向相背，**90 不做** |
| `color`（`:884`） | 保留 | 不动 | `--input-text` `#101010` ≠ `colorText` `rgba(16,16,16,0.88)`（α 不同），非严格等价 |
| `border-radius`（`:885`） | **改值 6→8** | `var(--theme-radius-lg)`（或删掉靠 `Input.borderRadius=8`，见 §C.2） | STYLE-03 |
| `font-size`（`:886`） | 保留 | T4 缓办（§B.4） | |
| `transition`（`:887-891`） | 保留 | — | T5 已完成 |

wrapper 组（`:917-923`）**同构处理**：`border` 简写 `:919` 同样拆三条，`border-radius: var(--theme-radius-md)`（`:921`）→ `var(--theme-radius-lg)`，`transition: all`（`:922`）→ 89 在主组做的「具体属性列表」此处未做，可顺手对齐（零渲染差异）。

#### A.2 为什么「只拆简写、不急于去 important 化 border-color」

ROADMAP 不可逆顺序 #2 的机理：`:883` 的 important 简写**胜过一切非 important 规则**。拆解后：
- `border-color … !important` 继续压住 antd 的 (0,3,0) status 色 → **错误态不会自动变红** →
  这正是 VISFIX-05 仍需「补 InputNumber status 规则」的原因（§D）。
- 若把 `border-color` 的 important 也去掉，antd 原生 status 机制立刻接管（乙方案，§D.2）——
  但会连锁改变 **hover/disabled/暗色 focus** 等一批态的渲染（§B.3 逐条列了不等价项），
  90 的变更预算会被撑爆。**推荐 90 走甲方案（拆简写保 color important），乙方案（彻底交还）登记为 91/92 的路径**——
  这与 ROADMAP 成功标准 #1 的表述「简写拆为长属性」完全一致，不欠需求债。

#### A.3 连带：批次 B（VIS-01 复位块 `:946-954`）的三选一处置评估

| 选项 | 前提 | 后果 | 判定 |
|---|---|---|---|
| **删** | 仅当 `:883`/`:919` 的 important 边框**完全消失**且 antd 内层机制接管 | 甲方案下（color 仍 important）删除 → 双框**立刻回归**（309 输入框，顺序约束 #2 原文） | ❌ 甲下不可行 |
| **留** | 甲方案（拆简写保 color important） | 复位块继续必要；C1a-C1c 断言**零调整**；K1 继续成立 | ✅ **推荐** |
| **缩** | 乙方案（去 important 全交还） | antd `input/style/index.js:379-394` 已含全部 6 项复位 → 复位块可整块删（约 -9 行 / -6 处 important）；C1a 需改写为**反向断言**（复位块不得存在，防回潮），K1 从「保留」改「已交还」 | ⚠️ 可行但本 phase 不做：连锁 hover/disabled/焦点环渲染面太大（§B.3），且会抢跑 91/92 的交还验收 |

**结论：90 执行「留」，但复位块注释（`:940-945`）必须重写**——现注释写「项目 :871 的 .ant-input !important 曾覆盖它」，
拆简写后语义已变（现在是 `border-color … !important` 单属性在压 status 色），需同步更新，避免 91/92 的执行者被旧注释误导。
这正是 ROADMAP「复位块的删除 / 重写在同一 commit」中「重写」的落点；**C1a 断言无需调整**（正则锚定的是规则本体 `:49`，不是注释）。

#### A.4 hover / focus / disabled 色值来源核对（变量 vs antd token，逐条等价性）

| 态 | 项目变量（现状生效值） | antd token（Bridge 已设） | 等价？ | 处置 |
|---|---|---|---|---|
| 常态边框 | `--input-border` light `#dbd7ce` / dark `#2d4d33` | `colorBorder` light `#dbd7ce` / dark `#2d4d33`（89 分模式后） | ✅ 完全等价 | 90 保留手写；**91/92 可删**（唯一遗留顾虑是 disabled，下行） |
| hover 边框 | `--input-border-hover` `#598e5e`（明暗同值） | `Input.hoverBorderColor = greenPrimary #156031`（`:145`） | ❌ **不等价**（浅绿 #598e5e vs 深绿 #156031） | **90 保留手写 hover 规则**（`:894-897` / `:925-927` / 暗色 `:2293-2298`）。若要交还须先把 `hoverBorderColor` 改为 `#598e5e`（= 改 token 值 = 浅色 hover 视觉不变、暗色 hover 从 #598e5e 不变——明暗同值，可行但属 token 层变更，建议与 91 的 Select 域 hover 一起做、统一登记） |
| focus 边框 | `--input-border-focus` light `#156031` / dark `#1a6839` | `activeBorderColor = #156031`（模式无关） | ⚠️ light 等价 / **dark 不等价**（`#1a6839` vs `#156031`） | 保留手写（`:899-903` + 暗色 `:2300-2309`） |
| focus 焦点环 | light `0 0 0 2px var(--theme-primary-lighter)`；暗色双层 2px+12px glow（K4/K6/K7/C4a-C4c 守护） | `activeShadow = 0 0 0 2px rgba(21,96,49,0.15)`（`:146`） | ❌ 色不同 + 暗色双层 antd 无对应 | **焦点环整组保留**（K4-K7 一条都不动；C4a/C4c 断言继续锁） |
| disabled 边框 | light：无规则 → 走常态 `#dbd7ce`；dark `--input-border-disabled #1a2e1f`（`:273`→`:3724`） | antd disabled = `colorBorderSecondary` light `#C2BDB2` / dark `#1a2e1f`（= `dark.borderSubtle`） | dark ✅ 等价；**light ❌**（现状 `#dbd7ce`，antd `#C2BDB2`） | 90 保留；**将来交还 `colorBorder` 时 light disabled 会变色 `#dbd7ce`→`#C2BDB2`——必须作为登记变更写进 91/92 的 CHANGESET**（本条是 §A.2 不交还的实锤理由） |
| 底色 | `--input-bg` light `#fafaf9` / dark `#0f2e1b` | `colorBgContainer` light `#FFFFFF` / dark `#1a2e1f` | ❌ 两模式都不等价 | 保留手写（§A.1） |
| placeholder | `--input-text-placeholder`（89 已统一，`:912-914`/`:2287-2290`） | antd `colorTextPlaceholder`（派生值） | ❌ | 保留 |

### B. R1-R5 映射总表（逐条：手写规则 → token/variant，当前行号）

> 90 的实际交还量 = **表中标「90 做」的行**；标「保留 / 91-92 做」的行是本研究的显式不改清单，
> 每条都给了 token 名与等价性结论，91/92 可直接续表。

| # | 手写规则（当前行号） | 等价 token / 机制 | 90 处置 | 渲染等价性（computed-style 预期） |
|---|---|---|---|---|
| **R1** | `:880-892` 主组 | 见 §A.1 逐属性 | **拆简写**（width/style 去 important、color 保 important、radius 6→8）；background/color/font-size/transition 保留 | 常态 computed 值全部不变；仅 `borderRadius` 6→8（已登记变更） |
| **R1b** | `:917-923` wrapper 组 | 同 R1 | 同构拆简写 + radius 6→8；`transition: all` → 属性列表 | 同上；wrapper/inner 均 8px（§C.4 验证 wrapper=8/inner=0） |
| **R2** | hover `:894-897`（主）/ `:925-927`（wrapper）/ 暗色 `:2293-2298` | `Input.hoverBorderColor`（现值 #156031，与现状 #598e5e 不等价） | **保留**；token 对齐移交 91（需把 `hoverBorderColor` 改 `#598e5e` 并登记） | 删则 hover 边框 `rgb(89,142,94)` → `rgb(21,96,49)`（浅色可见变色，未登记）|
| **R3** | focus `:899-903` + 暗色双层 `:905-910` | `activeBorderColor` + `activeShadow` | **保留**（焦点环 K4/K6 是 v1.24 契约，C4a 锁定） | 删则焦点环色/层数变（违规） |
| **R4** | wrapper hover/focused `:925-932` + 暗色 `:934-938` | 同 R2/R3（wrapper 侧） | **保留**（C4b/C4c 锁定） | 同上 |
| **R5** | status 还原 `:967-974` / `:976-980` | antd 原生 status 规则 (0,3,0) | **扩面补 InputNumber（VISFIX-05，§D）**；Input 域规则本身保留 | 见 §D |

**结论**：90 真正「交还」的是 R1/R1b 的**结构与圆角**，以及 R5 的 **InputNumber 扩面**；
hover/focus/底色/文字色四维的手写规则全部**有意保留**（逐条不等价，已列 91/92 的续办清单）。
这与 ROADMAP R1-R5 的表述（「拆成三份」+「focus 环需保留手写」）一致。

#### B.4 附：T4（`font-size: 14px !important`，`:886`）判定

antd 内层本就 `font-size: inherit`（`input/style/index.js:379-381`），carrier 默认 `fontSize: 14`。
**删除等价的充要条件 = 全站无 `size="large"` 输入控件**（large 时 antd 用 16px，手写 14px 会压住它）。
执行者先跑 `grep -rn 'size="large"' src --include=*.tsx`（含 `Input`/`InputNumber`/`Select` 场景）：
零命中 → 可删（M1 -1，computed 不变）；有命中 → 保留并登记 91。**研究判定：倾向删（零风险收益），但以 grep 结果为准。**

### C. 圆角统一（STYLE-03 / T1-T3 from X-3）

#### C.1 6px 出现点全量盘点（当前 HEAD）

| 类别 | 位置 | 说明 |
|---|---|---|
| **变量源头** | `:129` `--theme-radius-md: 6px;` | 全部「var 化的 6px」的根 |
| 主组 / wrapper（T1/T2） | `:885` / `:921` | 输入类圆角本体（STYLE-03 主目标） |
| `--theme-radius-md` 其它消费 | `:692, :851-857, :1180, :1193, :1366, :1430, :1504, :1521, :1773`（共 12 处引用，除 :885/:921 外 10 处） | 非 Input 域（按钮组/卡片/表格部件等），**不属于 STYLE-03 判据面**，90 不动（改源头 `--theme-radius-md: 8px` 一行就能全站统一，但会波及 10 处非输入组件 → 属未登记变更面扩大，**90 不改源头**） |
| `border-radius: 6px` 字面（M15 口径 7 处） | `:4458, :4470, :4482, :4494, :4506, :5234, :5489` | 前 5 处在暗色/日期区段（执行者需逐处读上下文判归属组件）；`:5234` 无 important；`:5489` = 分页器（K11 组） |
| 89-QUANT M15 口径 | 「`border-radius: 6px` 系仍 7 处」 | 与本盘点一致 ✅ |

#### C.2 删掉手写 6px 后谁自动变 8、谁要显式补

- **自动变 8**（token 已就位）：`.ant-input`、`.ant-input-number`、`.ant-input-affix-wrapper`
  —— 全局 `token.borderRadius: 8`（`:110`）+ `components.Input.borderRadius: 8`（`:147`）+ `Select.borderRadius: 8`（`:153`）。
  做法：把 `:885`/`:921` 的 `var(--theme-radius-md)` 改为 `var(--theme-radius-lg)`（8px，**保 important、最小 diff**）
  或直接删声明靠 token。**推荐改 var 而非删**：diff 一词、不动选择器结构、C11/C13 零波动、且暗色块无 radius 覆盖不需顾虑。
- **需要显式补**：
  - **分页器 `:5489`**：`6px → var(--theme-radius-lg)`（K11 规则组 `:5483-5491` 其余项原样保留）。
  - `:4458-4506` 五处 + `:5234`：逐处判归属——antd 组件的改 `var(--theme-radius-lg)`；项目自有 UI 的同样改 var（统一 8px 是 STYLE-03 的「圆角统一」语义，且 6→8 在这 6 处属**已登记变更**，须全部写入 CHANGESET 并在矩阵/巡检中核可见性）。
  - `borderRadiusSM` **不需要补**：输入类不消费 SM 档（SM 档仅 Tag=`:180` 已设 4、Checkbox/Radio 等小件走 antd 派生）；Modal/Drawer 走 `borderRadiusLG=12` 不动。
- **wrapper=8 / 内层=0 且视觉连续**：复位块 `:949` 的 `border-radius: 0 !important` 继续保证内层 0（甲方案下复位块保留，§A.3）——无需新增任何规则。

#### C.3 分页器 `:5489` 与 wrapper 的关系

**无关系**。`.ant-pagination .ant-pagination-item` 是分页页码的独立元素，不在 Input/wrapper 的嵌套链上；
它出现在 STYLE-03 判据里只是因为 ROADMAP 把 `index.css:877/912/5563` 三点并列为「含分页器」。
它是 K11（刻意设计：透明边框 + active 绿底白字）的成员，**只改 radius 一个词，其余逐字保留**。

### D. InputNumber 错误态（VISFIX-05）

#### D.1 现状成因链（当前 HEAD 全部重核完毕）

1. `:883` `border: 1px solid var(--input-border) !important` 简写重置 longhand → antd 的
   `.ant-input-number-outlined.ant-input-number-status-error:not(…)`（(0,3,0)，无 important）被压；
2. `:967-974` 的 status 还原组（K2）选择器**只写了 `.ant-input` / `.ant-input-affix-wrapper` 变体，漏 `.ant-input-number`**
   （`:958` 注释自述「只覆盖 Input 域，不动 Select/DatePicker」——InputNumber 是被漏的第 3 个）；
3. 暗色 `:2276-2285`（(0,2,0)+important）把 `border-color` 再压一次。
   ⇒ 现状 `borderTopColor = rgb(219,215,206)`（= `--input-border` light `#dbd7ce`），M13 FAIL。

#### D.2 修复方案（与批次 C 同 commit；对照 88-03 修 VIS-03 的手法）

**88-03 的先例手法 = 「同级/更高特异性 + important 补 status 规则」，90 直接复用**：

```css
/* 在 :967-974 的选择器组内追加 3 个变体（同 commit）： */
.ant-input-number.ant-input-number-status-error,                        /* (0,2,0)+imp > :883 (0,1,0)+imp ✅ */
.ant-input-number.ant-input-number-status-error:hover,                  /* (0,2,1)+imp > :896 hover (0,1,1)+imp ✅ */
[data-color-mode="dark"] .ant-input-number.ant-input-number-status-error /* (0,3,0)+imp > 暗色 :2284 (0,2,0)+imp ✅ */
```

- **特异性链已逐条算过**（上表注释）：三态（常态/hover/暗色）全部压得过现存 important 源；
  `:976-980` 的 warning 组同理追加 3 个变体（保持 K3 对称，防止修完 error 漏 warning）。
- 色值走 `var(--ant-color-error)`（与 K2 同源，`rgb(186,54,48)`；antd 6 cssVar 注入已由 88/89 实证）。
  注意 light 下 `--input-border-error`（`:272` 仅 dark 定义）**不可用**——沿用 K2 的 `var(--ant-color-error)` 才是明暗一致的唯一来源。
- **「拆完 C 后 antd status 规则自然生效吗？」——甲方案下不会**（`border-color` 仍带 important，A.2），
  必须补上面 3 个变体；**乙方案下去掉 important 后自然生效、无需补**（antd 机制完整，STACK :1018 实证）。
  90 按甲执行 ⇒ **补**。
- 期望值：`getComputedStyle(inputNumber).borderTopColor === 'rgb(186,54,48)'`（明暗双测；暗色下同样命中，
  因暗色变体 (0,3,0) 胜 (0,2,0)）。M13 ✅。

#### D.3 不动项

- **不把 `.ant-input-number` 塞进 `.ant-input` 的其它规则组**（P-M4 Prevention 原文）：DOM 结构不同
  （InputNumber 边框在根、无 affix wrapper），合并会引入双框新形态。
- 不加 `components.InputNumber` 组件 token 块（91 的 Select/InputNumber 域交还归口；90 用 CSS 补丁达成 M13 即可，避免 Bridge 与 CSS 两处同时改引入无法归因的中间态）。

<!-- gsd:write-continue -->
## Validation Architecture

### 1. 矩阵判据（90 有意变更登记，复用 89 的白名单归类模式）

复用基建：`.planning/phases/89-theme-refactor/89-visual-evidence/{matrix-fixture.js, matrix-sample.js, matrix-diff-89.cjs}`
（89 已把 `matrix-diff.cjs` 改造为「白名单 + 归类、未归类 = FAIL」模式；90 直接复制为
`.planning/phases/90-input-variant/90-visual-evidence/matrix-diff-90.cjs`，before 取 89 的 `after-{light,dark}.json`）。

⚠️ **与 89 的关键差别：90 的变更是明暗双向的**（圆角 6→8 两种模式都变），不像 89 是「light=0 / dark⊆白名单」。

| 变更类别 | 触发源 | 预期落点（形态 × part × 属性） | light | dark |
|---|---|---|---|---|
| **R-90a · 圆角 6→8** | `:885`/`:921` 改 `--theme-radius-lg` | Input 族全部形态（f01-f11, f14-f15, f20-f21）的 `carrier.radius`（wrapper/根），`inner.radius` 保持 0 不变 | radius 差异**必须 >0**（防空跑）且**全部归类为 a** | 同 light |
| **R-90b · InputNumber 错误态红** | §D 补的 3 变体 | 若 fixture 含 status=error 形态：error 形态 `carrier.border`；否则矩阵不变，靠 computed 探针（§2）覆盖 | border 差异归类 b | 同 |
| **R-90c ·（可选，若执行者采纳 §B.4 删 font-size）** | `:886` 删除 | 仅 size=large 形态的 `fontSize`（21 形态矩阵若全为默认 size 则 diff=0） | 0 或归类 c | 同 |
| hover/focus/底色/文字色 | §A.4/§B 有意保留 | **不得出现任何差异** | 0 | 0 |

判定规则：每条差异归类（a/b/c…）；未归类 = FAIL；归类的形态集合与 `90-CHANGESET.md` 登记集**不多不少**。
`:4458-4506/:5234/:5489` 的 6 处圆角不在 21 形态矩阵内 → 用 14 页巡检截图 + computed 探针单独核（§2 第 5 条）。

### 2. computed-style 断言清单（浏览器 Console 粘贴；明暗各跑一次）

| # | 断言 | 判据 | 对应契约 |
|---|---|---|---|
| 1 | 输入类圆角 | `.ant-input` / `.ant-input-affix-wrapper` / `.ant-input-number` / `textarea.ant-input` 的 `borderTopLeftRadius === '8px'`；分页器 `.ant-pagination .ant-pagination-item` 同为 8px；**wrapper=8px 且内层 input=0px 且视觉连续** | STYLE-03 |
| 2 | InputNumber 错误态 | `status="error"` 的 `.ant-input-number` `borderTopColor === 'rgb(186,54,48)'`（**常态 + hover + focus（明暗各一次）= 6 态**） | VISFIX-05 / M13 |
| 3 | 双框 | `css-contract-assertions.js` 粘贴：`doubleCount === 0`（明暗双测）；fixture 已在 88 修为真实 class | VIS-01 |
| 4 | 焦点环单圈 | wrapper（`-focused` 或 `:focus-within`）`box-shadow !== 'none'` 且**内层** `boxShadow === 'none'`；`box-shadow === '0 0 0 2px var(--theme-primary-lighter)'` 解析值不变（light） | VIS-02 / C4 |
| 5 | 错误/警告还原不回退 | Input/TextArea 错误态边框仍 `rgb(186,54,48)`（C5a/C5c 断言 PASS）；InputNumber 不得影响 Input 域既有断言 | VIS-03 |
| 6 | hover/focus 底色回归守护 | `.ant-input:hover` borderColor 解析 `rgb(89,142,94)`（#598e5e）、focus `rgb(21,96,49)`（light）——**90 不应改变它们，此探针证明「保留」成立** | §B 显式不改清单 |

### 3. 静态契约（CI 侧）

- `node scripts/css-contract-check.js` **exit 0**（C1a-C1c/C3/C4/C5 零调整——甲方案下复位块保留）；
  **C11 ratchet=1397 保持不动**（拆简写按 §A.1「只有 color 带 important」的手法，M1 净变化 ≈ 0，见 §5）。
- C13 per-file 基线（`css-hex-baseline.json`）：90 无任何新增 hex（`#598e5e` 等全部走既有 var）→ **零新基线条目**；
  M7/M8 不增。
- 新增静态断言（可并入本 phase 或 91）：**「`.ant-input-number` 的 status 还原规则必须存在」**（正向，
  仿 C5a 正则，防 VISFIX-05 回潮）；若未来走乙方案删复位块，C1a 需反转为「复位块不得存在」。

### 4. 单测 / lint

- `npm run lint` 0 problems；`npm run test`（3960 用例）全绿——90 不动任何 tsx，预期零回归；
- 不扩 `AntdThemeBridge` deps（C15 deps.length===2 保持）；90 若不动 Bridge（推荐形态下 Bridge 零改动），相关单测自然绿。

### 5. 量化快照预期（M1-M16，落盘 `90-QUANT-SNAPSHOT.md`）

| 指标 | 90 起点（实跑） | **90 预期** | 说明 |
|---|---|---|---|
| M1 / M2 | 1397 / 1330 | **≈ 1396-1399 / 1329-1332**（ROADMAP 终线 M1≤1100 / M2≤1050 不在本 phase 达成——那是 R1-R5 全面交还+暗色块删除后的 91/92 结果；90 的成功标准实为「不升」） | 拆简写 `1→3 长属性但仅 color 保 important` ⇒ ±0；补 InputNumber status 3 变体共用 1 声明 ⇒ +0；圆角改 var ⇒ ±0；§B.4 删 font-size ⇒ **-1**；批次 B 甲方案 ⇒ ±0。**结论：C11=1397 不动，实测值应 ≤1397** |
| M3 | 未清偿（89 沿用 725/702） | 不增（90 不删规则条目，只改声明） | |
| M7 / M8 | 304 / 440（C13 口径，89 W4） | **不增**（零新 hex；`--theme-radius-lg` 是 var） | C13 per-file 基线零新条目 |
| M9 | 5556（`split('\n').length` 口径；`wc -l`=5555） | **≤5558 但必须净减恰好 30 行**：C13 预算无容差、只剩 +2 行，而 W1 净增 ≥+30（拆简写 +2 / status 新块 +26 / 注释重写 +2~6）→ 同 commit 压缩 `index.css` 纯冗余空行（实测空行 859、块内空行 58 处），**净减恰好 30 行**（`wc -l`=5525、M9=5526 ≤5558 ✓）；全删 58 行会跌到 5497 < 5525 反违反下限。`C13_M9_BUDGET` 保持 5558 一行不动 |
| **M13** | `rgb(219,215,206)` | **`rgb(186,54,48)`** ✅ | VISFIX-05 |
| **M15** | 6px 系 7 处 + md 系混合 | **输入类 + 分页器 = 8px**；`--theme-radius-md` 非输入域 10 处消费**不变**（登记为 92 前的口径残留） | STYLE-03 |
| M11 / M12 / M14 / M16 | 已达标 | 保持 | 89 已交付，90 不回退 |

### 6. 波次建议（C → B → R5 扩面 → 圆角 → R1-R5 收尾评估）

| 波 | 内容 | 闸门（全绿才进下一波） |
|---|---|---|
| **W0** | 落 `90-QUANT-SNAPSHOT.md` 起点；复制 89 `after-*.json` → 90 `before-*.json`；`css-contract-check.js` exit 0 基线 | checker ALL PASS |
| **W1 · 批次 C + 批次 B + R5 扩面 + 圆角（单一 commit，不可拆）** | ① `:883`/`:919` 拆简写（仅 color 保 important）；② 复位块 `:946-954` 注释重写（保留本体，§A.3）；③ `:967-980` 追加 InputNumber error/warning 各 3 变体（§D.2）；④ `:885`/`:921`/`:5489` 6→8（§C.2）；⑤ 若 §B.4 grep 判定成立，删 `:886` font-size | checker exit 0 + §2 断言 1-6 全过 + 矩阵差异 ⊆ {a,b,c} + M13/M15 探针过 + lint/test 绿。**顺序敏感点：③ 的暗色变体选择器必须写在 `:2276-2285` 之前的位置即可（特异性已分胜负，与源码顺序无关，但按 §D.2 的 (0,3,0) 写法放置于 :967 组最稳）** |
| **W2 · R1-R5 剩余评估收尾** | §B 显式不改清单复核（hover/focus/disabled/底色）+ §B.4 T4 裁定 + 登记移交 91/92 的续办清单（hoverBorderColor 对齐 / colorBorder 交还的 light-disabled 变更 / 乙方案路径） | `90-CHANGESET.md` + 未清偿债落盘 |
| **W3 · 收口** | 14 页巡检截图入库 `90-visual-evidence/`；M1-M16 终值快照；`matrix-diff-result-90.txt` | 巡检无未登记差异；快照与预期偏差逐条说明 |

### 7. 与 86 线的文件边界确认

- **86 动 `src/pages/**/index.tsx`（76 页 JSX）+ `useActionPermission`；90 动 `src/index.css` +
  `scripts/css-contract-check.js` + `90-visual-evidence/` + （仅 W2 可能）`AntdThemeBridge.tsx` ——
  **文件面零交集**，两线可全程并行。
- **90 在推荐形态下完全不碰任何页面/组件文件**：InputNumber 错误态用 CSS 补丁解决（§D.2），
  不需要动页面组件的 `status` prop 传法；不动 `AppInput`/`AppSelect`/`AppTextArea` 默认 props
  （P-M7 冻结：改默认值会令 affix wrapper 匹配面整体消失、双框回归）。
- 唯一注意：86 若并行改到含 `status="error"` 的表单页，90 的探针取样页（建议 `/system/users` 新增弹窗）
  需避开 86 正在改写的页面，或巡检前与 86 线确认页面处于可渲染态。

## 边界与不做什么

| # | 不做项 | 理由 / 归属 |
|---|---|---|
| B-1 | 不做 Select 域（含 `hoverBorderColor` 对齐、`optionSelectedBg` 分模式） | Phase 91 |
| B-2 | 不删暗色 mega-cluster（`:2254-2742` / `:2780-4517` 的 InputNumber/Input 相关段**只读不动**） | Phase 92 |
| B-3 | 不做页面拆分 | Phase 93 |
| B-4 | **不去掉 `border-color` 的 important（乙方案）**、不删 VIS-01 复位块 | §A.2/§A.3：连锁 hover/disabled/焦点环渲染面大，登记为 91/92 路径 |
| B-5 | 不交还 `colorBorder` / `hoverBorderColor`（`--input-border*` 全组保留） | §A.4：disabled light 态不等价，交还须带登记变更，归 91/92 |
| B-6 | 不改 `--theme-radius-md: 6px` 源头值、不动其非输入域 10 处消费 | 波及面未登记；`--theme-radius-md` 的最终废存归 92 终态盘点 |
| B-7 | 不动 `components.InputNumber`（Bridge 不新增组件块） | §D.3 |
| B-8 | 不动 `AppInput`/`AppSelect`/`AppTextArea` 默认 props（P-M7 冻结）；不碰 `src/components/ConfigProvider.tsx`（P-C2 冻结） | 契约 |
| B-9 | 不扩 Bridge `useMemo` deps（C15 length===2） | 契约；推荐形态下 Bridge 本就零改动 |
| B-10 | 零新依赖（stylelint/playwright/postcss 等） | 沿用 v1.25 决策 |
| B-11 | 不迁 `onSearch` 等弃用 prop | antd 6.6.4 未弃用（88 实证）；M16=0 保持即可 |

**必须登记的未清偿债（移交 91/92）**：
D-a `Input.hoverBorderColor` 需对齐 `#598e5e`（现为 #156031）后才能删手写 hover 规则；
D-b `colorBorder` 交还时 light disabled 边框 `#dbd7ce`→`#C2BDB2` 是必然登记变更；
D-c 乙方案（去 important 全交还 + 删复位块 + C1a 反转）路径图；
D-d `--theme-radius-md` 非 Input 域 10 处消费的 6px 残留（92 终态归口）；
D-e `:4458-4506/:5234` 六处圆角的归属组件清单（W1 执行时补全进 CHANGESET）。

## Sources

### 项目文件（当前 HEAD 全部实读复核）
- `.planning/ROADMAP.md`（:51-57 三处不可逆顺序约束；:122-131 Phase 90 成功标准；:242-251 详情节）
- `.planning/REQUIREMENTS.md`（STYLE-02/03/05、VISFIX-05、机制更正段 :38-48）
- `.planning/phases/89-theme-refactor/89-RESEARCH.md`（§0 X-1/X-1b/X-2/X-3、§1.1 AntdThemeBridge 逐行、§1.4 T1-T14、§3 验证架构）
- `.planning/phases/89-theme-refactor/89-QUANT-SNAPSHOT.md`（W0/W1/W4 基线：M1=1397/M2=1330/M7=304/M9=5556/M13/M15）
- `.planning/research/PITFALLS.md`（P-C1/P-C6/P-M4/P-M7、第 3 批 K1-K11 :687-701、第 4 批 R1-R11 :703-717、量化基线表）
- `.planning/research/STACK.md`（A2.3/A2.4/A2.5 Input/焦点环/InputNumber DOM、A3 status class 与色 token、A7.3/A7.4 覆盖正/反模式、:1018 InputNumber status 完整性）
- `ruoyi-react-frontend/src/index.css`（:129-130, :188-195, :261-275, :880-980, :2276-2309, :4458-4506, :5234, :5483-5491 — 逐区间实读）
- `ruoyi-react-frontend/src/design-system/components/AntdThemeBridge.tsx`（212 行全文：board 分模式 :80、token :86-113、Input :143-149、无 InputNumber 块）
- `ruoyi-react-frontend/scripts/css-contract-check.js`（C1a 正则 :48-59、C3b 反转 :64-72、C4/C5 :74-98、C11=1397 :221-235、C13 基线 :237-286）
- `.planning/phases/88-style-foundation/88-visual-evidence/`（矩阵脚本 + 21 形态快照 + 14 页截图清单）
- `.planning/phases/89-theme-refactor/89-visual-evidence/`（`matrix-diff-89.cjs` 白名单归类模式、probe-*.txt）

### 本机实跑（2026-10-29，`ruoyi-react-frontend/`）
- `node -e` 计数：M1=1397 / M2=1330 / M9=5556 / hex 粗口径 311
- `node scripts/css-contract-check.js` → **ALL PASS / exit 0**

### 置信度汇总
| 结论 | 置信度 |
|---|---|
| 当前 HEAD 行号（:880-980 / :2276-2309 / :5483-5491 / 圆角 12+7 处） | HIGH（实读 + grep 双核） |
| 拆简写「仅 color 保 important ⇒ M1 ±0」的推演 | HIGH（CSS 语义直推，W1 实跑复核） |
| InputNumber **6 选择器**特异性链（3 档 × 常态/hover/focus）：light 常态 (0,2,0) / light 交互 (0,3,0) / dark 常态 (0,3,0) / dark 交互 (0,4,0)——全部压过现存 important 源 | HIGH（逐条算过；checker 更正：原稿的 (0,2,1)/(0,3,1) 系伪类按 class 级计的误标，已校正；W1 浏览器探针终验） |
| hover/focus/底色「不等价、保留」清单 | HIGH（变量值 vs Bridge token 值逐对对照） |
| §B.4 T4 删除等价性 | MEDIUM（依赖 `size="large"` grep 结果，执行时裁定） |
| 量化预期区间 | HIGH（口径与 89 一致，实跑终值为准） |

---

*研究完成：2026-10-29｜只读研究，未修改任何业务代码，未 git commit。*

---

## 7. 口径更正（2026-10-29 checker / fixer 复核后）

> 本节是 plan-checker + plan-fixer 两轮复核的**结论固化**——执行者与后续 phase（91/92）读本文件时以此节为准。执行口径以 `90-0*-PLAN.md` 为准。

### 7.1 四条 BLOCKER 的最终解

| # | 问题 | 最终解 |
|---|---|---|
| **1** | M9 预算不可达（C13 硬上限 5558 无容差，起点 M9=5556 只剩 +2 行，W1 净增 ≥+30） | 同 commit 压缩 `index.css` **纯冗余空行**，**净减恰好 30 行**（`wc -l`=5525、M9=5526 ≤5558 ✓）。压缩规则限定两类：连续 ≥2 空行折叠为 1（实测仅 1 组且在暗色只读区内 → 可减 0）；**规则块/注释块内部**的空行（实测 58 处，全在暗色只读区外）。**块间单空行一个不删**；禁改任何 CSS 声明与注释内容、禁碰暗色只读区 `:2254-2742`/`:2780-4517`、禁碰六处非输入域 6px。`C13_M9_BUDGET` 保持 5558 一行不动 |
| **2** | 注释内的 hex 触发 C13 per-file FAIL | C13 的 stripComments 用 `/\/\*.*?\*\//g`（`.` **不跨行**）→ **多行注释块内的 hex 被计入**（实测 raw 311 vs strip 后 304，差 7；per-file 基线 `index.css`=304 硬上限）。故 PLAN 给定的注释块**不得含任何 6 位 hex**（`#dbd7ce`→`--input-border`、`#BA3630`→`ruoyiBrand.functional.danger`）。**债 D-g**：修法为把正则改为 `[\s\S]*?`——本 phase **不动 C13** |
| **3** | verify 自撞（查 `--input-border-error` 不出现，而自己的注释恰好写着它） | verify 改在**剥注释文本**上判定（`t.replace(/\/\*[\s\S]*?\*\//g,'')`，与既有 C5b 同风格）；注释文本改为「light 侧无对应错误态变量」 |
| **4** | 行号漂移（拆 `:883` 一行→三行 ⇒其后全部 +2，8 处硬编码 verify 全误红） | 90-02 的 verify **全部改内容锚点定位**（按选择器文本/regex 找行）；90-03 的清单行号列标注「**W1 前 HEAD 基准，仅参考**」+ 另设「**选择器锚点（权威）**」列；PLAN 内所有 `:NNN` 均声明以 W1 前 HEAD 为基准 |

### 7.2 竞争源核对表的最终更正

原表把 `:2295`/`:2302` 列为 InputNumber 竞争源——**实读不成立**：这两块选择器只含 `input / textarea / .ant-input / .ant-input-affix-wrapper(-focused)`，**不含 `.ant-input-number`**。真实竞争源 5 条：

| 竞争源 | 选择器 | 特异性 | 胜负 |
|---|---|---|---|
| `:894-897` | `.ant-input:hover, .ant-input-number:hover` | (0,2,0) | light 常态 (0,2,0) **同级 → 靠源码顺序取胜**（本块在 `:980` 之后，晚于全部项目级源；空行压缩只删空行不改块相对顺序 → 依赖稳定） |
| `:899-903` | `.ant-input:focus, .ant-input-number:focus` | (0,2,0) | 同上 |
| 暗色 base | `:2276-2285` | (0,2,0) | dark 常态 (0,3,0) **严格胜** |
| 暗色 `:3745-3748` | — | (0,2,0) | 同上 |
| 暗色 `:3750-3755` | 暗色 InputNumber 专项 focus | (0,3,0) | dark 交互 (0,4,0) **严格胜** |

**档位设计维持不变**：3 档 × 常态/hover/focus = **6 选择器/组**（error/warning 各 6），一个不减，不新增类级提升档（`.ant-input-number` 已作类级出现两次，基础档已达 (0,2,0)）。表脚注：**伪类按 CLASS 级计**。

### 7.3 其他口径更正

- **格数**：89 终跑 **360 格**（每模式 180 = (21×2−6)×5，6 个 inner=null 形态被跳过）——不是 210。90 理论 528 格、**实测 456 格**（(22×2−6)×6×2）
- **矩阵判据 90 化**：89 的「light 差异必须 = 0」红线在 90 **必然误 FAIL**（圆角变更明暗双向）→ 改为「light/dark 各自差异 > 0 + 全归类 + 形态集合不多不少」
- **before 快照重采**：不复制 89 的 after-*.json（缺 f22 会被判「结构缺失」误 FAIL）；以「与 89 after-*.json 的 21 形态交集 210 格逐格全等」作防串基线核对
- **C4b 字面依赖**：`css-contract-check.js:76` 用字面正则守护 `border: 1px solid var(--input-border)` → **wrapper 的 `:919` 简写不得拆**，批次 C **只拆 `:883`**；wrapper 处置登记为债 **D-f**
- **antd 6.6.4 真实类名**：`ant-input-number-status-error`（`getStatusClassNames` 落在 `.ant-input-number` **根**，边框也画在根）——写成 `ant-input-status-error` 对 InputNumber 不命中（VISFIX-06 同型陷阱）
- **色值**：`--ant-color-error` 明暗同值 `#BA3630` → 「明暗双测 rgb(186,54,48)」成立；`css-contract-assertions.js:14` 旧注释的「暗色 rgb(161,49,44)」是**静态 harness（无 ConfigProvider cssVar scope）的记录、非真实 App 值** → 执行期必须**实测**后按实测改判据，**不得伪造**
- **态数**：InputNumber 错误态 = **常态 + hover + focus（明暗各一次）= 6 态**
