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
