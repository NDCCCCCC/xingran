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
