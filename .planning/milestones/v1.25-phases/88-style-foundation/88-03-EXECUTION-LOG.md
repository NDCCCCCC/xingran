# Phase 88-03 执行日志（VISFIX-03 + STYLE-02 第 1 批 D1-D17）

执行时间：2026-10-28 16:42–17:20｜环境：dev server :4000 + `$TEMP/p75-uat` sqlite 后端 :9000（admin/admin123）

## 任务 3.1 删前取证（浏览器 querySelectorAll，页面 /system/user）

| 取证项 | 选择器 | 命中数 | 预期 | 判定 |
|---|---|---|---|---|
| D1 | `.dept-tree-search .ant-input-affix-wrapper` | 0 | 0 | ✅ |
| D2 | `.dept-tree-search .ant-input-affix-wrapper .ant-input` | 0 | 0 | ✅ |
| D3 | `.dept-tree-search …:focus / -focused` | 0 | 0 | ✅ |
| D4 | `.dept-tree-search …:focus .ant-input / -focused .ant-input` | 0 | 0 | ✅ |
| D7 | `.ant-select-selector`（全站口径） | 0 | 0 | ✅ |
| D8-D11/D15 | `.ant-select* … .ant-select-selector` 变体 | 0 | 0 | ✅ |
| D12-D14 | `[data-color_mode="dark"] …`（拼写属性） | 0（属性不存在） | 0 | ✅ |
| D5/D6 | `data-color_mode` DOM 属性佐证：`getAttribute('data-color-mode')`='light'，`getAttribute('data-color_mode')`=null | — | — | ✅ |
| **K8（活）** | `.user-form-input.ant-input-affix-wrapper .ant-input` | **1** | >0 | ✅ |
| K9（活） | `.user-form-input…-focused .ant-input` | 0（静态无焦点，:focus/-focused 态规则） | — | 静态采样预期 |
| 属性 | `data-color-mode` 存在 / `data-color_mode` 不存在 | ✓/✓ | — | ✅ |

补充实证（删后发现）：antd 6.6.4 真实 DOM 的 Select 结构为 `.ant-select > .ant-select-content > input.ant-select-input`，
**不存在 `.ant-select-selector`**（/system/role 真实页 `.ant-select`×2、`.ant-select-selector`×0 实测）——坐实 D7-D15 全家族死规则判定。

## 任务 3.2 删除清单（同一原子变更集，未 commit）

### D1-D15（index.css，按大→小行号 sed，删后 grep 全 0）

| 条目 | 原行号 | 内容 |
|---|---|---|
| D15 | 5579-5581 | `.ant-pagination .ant-pagination-options .ant-select-selector` |
| D12-D14 | 3962-3977 | 暗色 `.ant-select-selector` / hover / focused 三条 |
| D6 | 3602-3608 | `[data-color_mode] .ant-popconfirm…buttons .ant-btn(-primary)` 两条 |
| D5 | 3523-3545 | `[data-color_mode] .ant-anchor*` 六条 |
| D11 | 2329-2333 | `[data-color-mode="dark"] .ant-select-selector` |
| D1-D4 | 2092-2117 | `.dept-tree-search` 家族四条 + 注释 + 空行（K8/K9 零接触，前移 26 行 intact） |
| D10 | 897 | dark focused 分支行 + 上行去逗号 |
| D9 | 890 | focused 分支行 + 上行去逗号 |
| D8 | 884 | hover 分支行 + 上行去逗号 |
| D7 | 873 | selector 分支行 + 上行去逗号（871-880 组内 `.ant-input` / `.ant-input-number` 活分支保留） |

执行插曲：分支行删除会连带 `{`，首次脚本产生 4 处缺 `{` 语法损坏，diff 复核时发现并立即修复（补回 ` {`），
修复后括号平衡校验 898/898、git diff 逐行复核无活规则误删。

### 9 个死文件（rm，未 git add/commit）

theme-styles.css｜dashboard-system/index.css｜ChartWidget/ListWidget/MetricWidget/ProgressWidget/StatCardWidget.css｜CADFloorPlanEditor.less｜FloorPlanEditor.less（0 字节）
+ `FloorPlanEditor.tsx:18` `import "./FloorPlanEditor.less";` 同变更集删除（grep 复核 0 命中）。
`dashboard-system/edit.css` / `view.css` 未触碰（活文件）。

### check.js（scripts/css-contract-check.js）

1. **C3b 反转**：`indexCssNoComment` 定义上移至 C3 之前；`:57` 改反向断言 `!…dept-tree-search…ant-input-affix-wrapper…`（含「不得存在」）；C3c 原样。
2. **C8/C12 翻转阻断**：warn-only 移除，命中才计 failure（并修正标签 0 命中时显示 PASS）；`--strict` 保留为冗余。
3. **C11 预算**：1435 → **1398**（PLAN 预估 1395，实跑 1398；ratchet 只许低不许高 → 取实测）。
4. **C6 豁免行号同步**（连带变更，PLAN 未列）：删除使行号前移，豁免表 2306/2336/4553 → 2276/2301/4472，否则 C6 误报 FAIL。

`node scripts/css-contract-check.js` exit 0（默认与 --strict 均 ALL PASS，22 项全 PASS）。

## 任务 3.3 视觉 diff = 0（唯一硬判据）

- 方法：21 形态 fixture 注入 `.ant-app.css-var-*`（cssVar scope），删前（临时还原 HEAD index.css）与删后各采样明暗两轮，
  carrier+inner × bg/border/radius/shadow/btw 逐格比对。
- 结果：**360 格，0 差异**（`matrix-diff-result.txt`）。
- 截图：14 页 × 明暗 = 28 张，`88-visual-evidence/p01…p14-*-light|dark.png`；
  肉眼复核 p03 user、p06 menu、p07 buildings、p08 workstations 暗色——部门树搜索框无描边/白底回潮、无双框；
  白底 Select、浅奶油输入底为登记在案的既有暗色欠账（89/92 范围），非本批引入。

## 闸门结果（实跑）

| 命令 | 结果 |
|---|---|
| `node scripts/css-contract-check.js` | exit 0，ALL PASS（22 项，C8/C12 阻断态、C11=1398） |
| `npm run lint:ci` | exit 0（errors 0 / ≤0） |
| `npm run lint` | exit 0，0 problems |
| `npm run type-check` | exit 0 |
| `npm run test -- --run` | exit 0，570 文件 / 3963 用例全过 |
| `npm run build` | exit 0，✓ built in 33.35s |

## 偏离 PLAN 之处

1. **C11 定值 1398（非 1395）**：PLAN「实跑确认 ≈1395 再定值，只许低不许高」——实测 1398，按 ratchet 语义取实测值。
2. **C6 豁免行号同步**（check.js 豁免表 3 处行号前移）：PLAN 未列，但属删除的必然连带，不改则 C6 误报。
3. **fixture 口径修正**：初版 fixture 手写 `.ant-select-selector` 旧 DOM 导致 40 格伪差；按 antd 6 真实 DOM 修正后 diff=0。
   该发现反向坐实了死规则判定（真实页全站 0 命中）。
4. **残留孤儿注释 2 处**（未删，严格按 PLAN 行号范围）：`index.css` 原 `:3520-3522`「Anchor 锚点暗色模式」段头注释与
   原 `:3961` `/* Select */` 注释，其下规则已删。纯注释、无样式影响，可归后续批次顺手清理。
5. K9 静态命中 0：:focus/-focused 态规则，静态无焦点采样为 0 属预期；K8=1 已证明活规则在顶。

## 遗留 / 观察

- 仓内 `internal/services/system/asset_columns_schema.json` 出现非本 plan 的工作区改动（疑似后端进程写），未触碰，请主会话甄别。
- 未执行 git add/commit（按约束）；同 commit 清单 = index.css + check.js + FloorPlanEditor.tsx + 9 个删除文件。
