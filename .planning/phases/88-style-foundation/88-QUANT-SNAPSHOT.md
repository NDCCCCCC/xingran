# Phase 88-03 量化快照（删除第 1 批 D1-D17 后实测）

> 采样时间：2026-10-28 17:20（本地 sqlite UAT 环境 + :4000 dev server）
> 口径来源：ROADMAP.md §量化口径（M1-M16）

| 指标 | 口径 | 88-03 前 | 88-03 后（实跑） | PLAN 预估 | 备注 |
|---|---|---|---|---|---|
| **M1** | 全 `.css`/`.less` `!important` 计数 | 1435 | **1398** | ≈1395 | C11 ratchet 定值 1398（只许低不许高，预估偏低 3，取实测） |
| **M2** | `index.css` `!important` 计数 | 1359 | **1331** | ≈1328 | 同上偏差 3 |
| **M3** | `index.css` 含 `.ant-` 顶层规则 | 748 | 未清算 | — | Phase 90 范围 |
| **M4** | 暗色顶层规则（行首 `[data-color-mode="dark"]` grep 口径） | 506（含 8 拼写错） | **569**（拼写错 0） | ≈498 | 本 grep 口径含多选择器组的续行；正式清算归 Phase 92 |
| **M9** | `index.css` 行数 | 5632 | **5548** | ≈5580 | 实删 84 行（D1-D15 + 分支行） |
| **M14** | 死样式文件数 | 8 + 1（带 import） | **0**（C8 全绿） | 0 | 9 个文件全删，含 FloorPlanEditor.less + 其 import |

## 死规则 grep 复核（删后）

```
grep -c 'dept-tree-search'   src/index.css → 0
grep -c 'data-color_mode'    src/index.css → 0
grep -c 'ant-select-selector' src/index.css → 0（9 → 0）
grep -c 'user-form-input'    src/index.css → 3（K8/K9 活规则 + 命中保留）
```

## 视觉 diff 判定（唯一硬判据）

- 21 形态 × 明暗 × carrier/inner × 5 属性（bg/border/radius/shadow/btw）= **360 格**
- 删除前后逐格 diff：**0 差异**（`matrix-diff-result.txt`）
- fixture 挂载点：`.ant-app.css-var-*`（antd cssVar scope，88-01 启示）
- 注意：初版 fixture 手写了旧版 antd `.ant-select-selector` DOM，会误中已删死规则产生 40 格伪差；
  已按 antd 6 真实 DOM（`.ant-select-content`/`.ant-select-input`，真实页该 class 命中为 0）修正后重取基线复测。
