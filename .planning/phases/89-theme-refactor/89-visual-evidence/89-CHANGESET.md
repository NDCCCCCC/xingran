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

## B 类 · 暗色容器底色（Plan 02 触发 · token 层变更）

| # | 类别 | 形态 | part | 属性 | 暗色 before → after | 触发源 | 依据 |
|---|------|------|------|------|-------------------|--------|------|
| B | 暗色容器底色（token 层） | （矩阵 21 形态均无可见落点，见下方实机校准） | — | — | `--ant-color-bg-container`: `#FFFFFF` → `#1a2e1f`；`--ant-color-border`: `#DBD7CE` → `#2d4d33`；`--ant-color-bg-elevated`: `#FFFFFF` → `#1f3524` | `colorBgContainer`/`colorBgElevated`/`colorBgLayout`/`colorBorder`/`colorBorderSecondary` 分色板 + 组件级 Input/Select/Card.colorBgContainer 三项删除（X-1b） | AntdThemeBridge seed/map 层按模式显式给值 |

**B 类实机校准说明**（Plan 02 执行期实跑，替代 PLAN §2.4 的预判判据）：
- **PLAN 预判「f12/f13/f16/f17/f18 carrier.bg 暗色 after = rgb(26,46,31)」不成立**：矩阵 fixture 的 Select 族 carrier 采样点 = `.ant-select` **根元素**，其 computed `backgroundColor` 在明暗两模式下均为 `rgba(0, 0, 0, 0)`（透明，底色由内层 `.ant-select-content` / index.css dark mega-cluster 承载）⇒ token 变化在矩阵 360 格上**不产生任何可采样差异**。
- **token 层生效已由 cssVar 证据实机确认**（dark 组合 = store 偏好 dark + `data-color-mode="dark"`，dev :4000 + CDP 实测）：
  - dark：`--ant-color-bg-container` = `#1a2e1f` ✓ / `--ant-color-border` = `#2d4d33` ✓ / `--ant-color-bg-elevated` = `#1f3524` ✓（均 ≠ 白/浅值，X-1b 白底未重新长出）
  - light：`--ant-color-bg-container` = `#FFFFFF` ✓ / `--ant-color-border` = `#DBD7CE` ✓（零漂移）
  - 真实页面（/system/user，dark）：`.ant-card` = `rgb(31,53,36)`、`.ant-card-body` = `rgb(36,58,41)`，与 index.css 暗色板一致，无白底回归。
- 矩阵终跑：360 格，light=0 / dark=20（全部 A 类），exit 0。B 类形态集合按实机校准为**空集**（token 变更不可见于矩阵口径，Phase 92 删除暗色 mega-cluster 后才会暴露到组件可见层）。

## 未清偿债（移交）

| # | 债 | 量化 | 归属 |
|---|---|---|---|
| D-3 | 暗色占位符对比度不达标 | `#707068` on `#0f2e1b` = 2.95:1 < AA 4.5:1（串色修复**暴露**的既有问题；本 phase 不提亮以守住「除 A 类登记外 diff=0」预算） | Phase 92 |
| D-4 | `Button.defaultBorderColor` 仍为浅值 `#DBD7CE`（保守不改） | Plan 02 保守裁定：改了会在 A/B 登记面之外新增暗色按钮描边差异；暗色下按钮描边目前由 index.css dark cluster（`[data-color-mode="dark"] .ant-btn-default` → `#345b3a`）承接，token 侧欠账 | Phase 92 |
| D-5 | `Select.optionSelectedBg` 仍为浅值 `#E9EFEB`（保守不改） | 同上：改了会新增暗色选中行底差异；暗色下由 index.css dark cluster（`.ant-select-item-option-selected`）承接 | Phase 92 |
| — | `--theme-zebra` 仍在后置 `:root`（未迁） | index.css（原 4733 行） | Plan 03 T14 |
| — | `--radius` fallback（`var(--radius, 8px)`） | 有 fallback，C6 通过，**有意保留**（定义全局 --radius 会与 --theme-radius-* 体系重复） | 不处理 |
| — | hex ratchet 净增 2 处 | M7 310 → 312（light `--input-bg-hover` #f4f2ec / `--input-bg-disabled` #eceae4 两处**当前无引用**的结构对称新定义；focus 已 var 化、:root 兜底已 var 化、5178 fallback 已删除抵消） | W3/后续 plan 消化（M7 终态目标 ≤300） |
