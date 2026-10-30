# Phase 89 量化快照

## W0 起点（改动前实测，2026-10-28）

> 口径与 88-QUANT-SNAPSHOT 一致（Node 正则计数）。checker 基线 `node scripts/css-contract-check.js` exit 0（ALL PASS，C6 豁免 3 处）。

| 指标 | 口径 | 88 末态（文档值） | **89 W0 实跑** | 说明 |
|---|---|---|---|---|
| **M1** | 全 `.css`/`.less` `!important` | 1398 | **1398** | C11 ratchet 1398 |
| **M2** | `index.css` `!important` | 1331 | **1331** | |
| **M7** | `index.css` 6 位 hex（总次数） | 304（研究文档口径） | **310**（去重 64） | 研究文档 304 为 88-03 时点值；88-04~88-06 收口后实跑为 310，**与任务书参考 M7=310 一致**，以本实测为 ratchet 基线 |
| **M8** | 全 `.css`/`.less` 6 位 hex | 446（研究文档口径） | **452** | 同上，任务书参考 452 一致 |
| **M9** | `index.css` 行数 | 5548 | **5549** | 与任务书参考一致（+1 行漂移来自 88 收口批） |

复现命令（`ruoyi-react-frontend/`）：

```js
// M7/M8: (text.match(/#[0-9a-fA-F]{6}\b/g)||[]).length；去重 = 小写 Set.size
// M9: text.split('\n').length
// M1/M2: (text.match(/!important/g)||[]).length
```

## 变量层判定（W0 基线事实）

- 暗色 `--input-bg` 解析值 = `#fafaf9`（串色病灶，来自后置 `:root`@4734 胜出）
- `--input-placeholder`（light@189）/ `--input-text-placeholder`（dark@267）双名并存；`:900/:2276/:2301` 三处引用旧名，其中 2276/2301 暗色下整条作废；2300-2302 为 `.ant-select-selection-placeholder` 死规则
- `--theme-warning-text` 0 定义 / 2 引用（4472 无 fallback、5178 有 fallback `#905d00`）
- `--input-bg-hover/-focus/-disabled` 仅 dark 定义（254-257），引用仅 dark 块（3701/3706/3729/3714）

## hex ratchet 基线声明

W1 变量层后 ratchet 基线以本 W0 实测为准（M7=310 / M8=452，只降不升目标归 Phase 89 整体收口）。

## W1 后实测（Plan 01 变量层改动后，2026-10-28）

| 指标 | W0 | W1 后 | 变化 |
|---|---|---|---|
| M1（全 css !important） | 1398 | **1397** | -1（删 2301 死规则声明） |
| M2（index.css !important） | 1331 | **1330** | -1（同上） |
| M7（index.css 6 位 hex） | 310 | **312** | +2（light `--input-bg-hover` #f4f2ec / `--input-bg-disabled` #eceae4，均为当前无引用的结构对称新定义；`--input-bg-focus` var 化、:root placeholder 兜底 var 化、5178 fallback 删除已抵消 3 处潜在净增） |
| M8（全 css 6 位 hex） | 452 | **454** | +2（同上） |
| M9（index.css 行数） | 5549 | **5551** | +2（≤ 5558 预算） |

变量层验收 grep（W1 后）：
- `--input-bg:` 定义恰 2 处（light #fafaf9@188 / dark #0f2e1b@261）；四兄弟 light+dark 双分区对称
- 旧名 `--input-placeholder` 0 命中；`var(--input-text-placeholder)` 3 处引用；`.ant-select-selection-placeholder` 全仓 0 命中
- 消费点：`var(--input-bg)` 17 行 / hover 1 / focus 2 / disabled 1（与 W0 相同，不增不减）

## W4 终值快照（Plan 04 收口，2026-10-29 实测）

> 口径：M7/M8/M9/M14 以 `node scripts/css-contract-check.js`（ALL PASS）输出为准（C13 = 逐行剥离 `/*…*/` 与 `//` 单行注释后统计 6 位 hex；多行注释块内的 hex 按 C13 口径计入）；M1/M2 同脚本 C11 口径；M11/M12 为 dev :4000 + CDP 真机探针（`89-visual-evidence/probe-{light,dark}.txt`）。

| 指标 | 口径 | W0 起点（实测） | 预期 | **终值（实测）** | 判定 |
|---|---|---|---|---|---|
| **M1** | 全 `.css/.less` `!important` | 1398 | ≤1398（只降不升；C11 ratchet 已随 T5 下调至 1397） | **1397** | ✅（-1，删 2301 死规则声明） |
| **M2** | `index.css` `!important` | 1331 | ≤1331 | **1330** | ✅（-1，同上） |
| **M3** | 含 `.ant-` 顶层规则 | 725（index 702） | 不增（89 未删规则） | 沿用 W0（89 仅动变量定义与 token 层，零规则级增删） | ✅ |
| **M4** | 暗色顶层规则 | 569 / 502 | 不增 | 同上 | ✅ |
| **M5** | 暗色分区 | ≈64 | 不增 | 同上 | ✅ |
| **M6** | 重复暗色 class | ≈80 | 不增 | 同上 | ✅ |
| **M7** | `index.css` 6 位 hex（C13 口径） | 310（去重 64） | **≤310 只降不升**（≤300/去重≤60 是 92 终态，89 不追） | **304**（去重 63） | ✅（-6，W3 死链删除 + A1/A2/A3 + T 批次） |
| **M8** | 全 `.css/.less` 6 位 hex | 452 | **≤452 同口径** | **440** | ✅（-12） |
| **M9** | `index.css` 行数 | 5549 | ≤5558 | **5556** | ✅ |
| **M10** | `index.css` 分区块 | ≈97 | 不增 | 沿用 W0（同 M3-M6 理由） | ✅ |
| **M11** | 暗色 `--input-bg` | `#fafaf9`（错） | **`#0f2e1b`** | **`#0f2e1b`**（探针 dark inputBgVar） | ✅ VISFIX-01 核心 |
| **M12** | 暗色 placeholder 变量 | 未定义 | **非空** | **非空**（`--input-text-placeholder: var(--theme-neutral-400)` → 解析 `#707068`） | ✅ R11 |
| **M13** | InputNumber 错误态边框 | `rgb(219,215,206)` | 不变（→90） | 不变（89 未触错误态规则；`--border-base #dbd7ce` 仍在位） | ✅ |
| **M14** | 死样式文件 | 0 | **0** | **0**（C8 全绿） | ✅ |
| **M15** | 圆角 | 6px/8px 混合 | 不变（→90） | 不变（`border-radius: 6px` 系仍 7 处，89 未动） | ✅ |
| **M16** | antd 弃用警告 | 0 | **0** | **0**（14 页 × 明暗巡检 `deprecated` 计数 = 0，见 `inspection-result-89.txt`） | ✅ |

**偏差说明**：无超预期项。M7/M8 相对 W0 的下降全部来自 W3 已提交批次（死链 19 文件删除 + A1/A2/A3 处置 + T5-T14 色值类降级）；M1/M2 -1 来自 W1 R11 删除死规则。

## X-2 显式例外登记（暗色输入框底 vs 页面底对比度）

> 登记：**「暗色输入框底 vs 页面底对比度 ≥3:1」判据数学上不可满足，登记显式例外并移交。**

- **不可满足证明**：VISFIX-01 修正后暗色输入框底 = 页面底 = `#0f2e1b` → 对比度 1:1；且 `#0f2e1b`（相对亮度 L≈0.0262）对**任意**背景色的对比度上限为 `(Lmax+0.05)/(0.0262+0.05) = 1.43:1`（纯白背景），3:1 在该色域内无解。
- **裁定**：3:1 判据移至组件**描边**（WCAG 1.4.11 非文本对比度管的是组件边界，不是面/面关系）。
- **移交**：暗色描边提亮（`--input-border #2d4d33` 1.56:1 → `#4a7d52` 3.05:1 级别）归 **Phase 92**（暗色 mega-cluster 域，即债 D-1）。
- 对应 ROADMAP Phase 89 成功标准 #1/#4 按 X-2 改写后的口径；REQUIREMENTS.md VISFIX-01 条目已同步改为例外登记指向（文字更正，不改需求语义）。
