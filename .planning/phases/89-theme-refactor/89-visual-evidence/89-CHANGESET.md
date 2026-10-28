# Phase 89 变更登记表（89-CHANGESET）

> 判定规则：矩阵 diff（`matrix-diff-89.cjs`）中每条差异必须能归入下表类别之一；未归类 = FAIL。
> light 模式差异必须 = 0；dark 模式差异必须 > 0；归类后形态集合与登记**完全一致**（不多不少）。

## A 类 · 暗色输入类底色（本 plan Plan 01 触发）

| # | 类别 | 形态 | part | 属性 | 暗色 before → after | 触发源 | 依据 |
|---|------|------|------|------|-------------------|--------|------|
| A | 暗色输入类底色 | f01_input_plain, f02_input_prefix, f03_input_suffix, f04_input_allowclear, f05_input_addonbefore†, f06_input_addonafter†, f07_input_password, f08_input_search, f09_textarea, f10_textarea_showcount, f11_input_number†, f14_datepicker, f15_range_picker, f19_mentions†, f20_appinput, f21_apptextarea | carrier（† 另含 inner） | bg | rgb(250, 250, 249) → rgb(15, 46, 27) | `--input-bg` 串色修正（后置 `:root` #fafaf9 迁入 `[data-color-mode="light"]`，暗色回落 dark 块 `#0f2e1b`） | index.css light 块 188-191 / 原 4734 行删除 |

**实测校准说明**（研究 §1.5.1 推定 → 实跑补全）：
- `inner.bg` 实际落点 = f05 / f06 / f11 / f19（f01、f14、f15 内层被 VIS-01 复位为 `transparent`，不随变量变化 → 与推定一致部分修正）。
- **f19_mentions 归 A 不归 B**：`.ant-mentions` textarea 底色实测随 `--input-bg` 走（非 `colorBgContainer`），carrier+inner 两格同步变化。
- f12/f13/f16/f17/f18（Select 族）暗色无差异（B 类由 Plan 02 触发后再登记）。

## 未清偿债（移交）

| # | 债 | 量化 | 归属 |
|---|---|---|---|
| D-3 | 暗色占位符对比度不达标 | `#707068` on `#0f2e1b` = 2.95:1 < AA 4.5:1（串色修复**暴露**的既有问题；本 phase 不提亮以守住「除 A 类登记外 diff=0」预算） | Phase 92 |
| — | `--theme-zebra` 仍在后置 `:root`（未迁） | index.css（原 4733 行） | Plan 03 T14 |
| — | `--radius` fallback（`var(--radius, 8px)`） | 有 fallback，C6 通过，**有意保留**（定义全局 --radius 会与 --theme-radius-* 体系重复） | 不处理 |
| — | hex ratchet 净增 2 处 | M7 310 → 312（light `--input-bg-hover` #f4f2ec / `--input-bg-disabled` #eceae4 两处**当前无引用**的结构对称新定义；focus 已 var 化、:root 兜底已 var 化、5178 fallback 已删除抵消） | W3/后续 plan 消化（M7 终态目标 ≤300） |
