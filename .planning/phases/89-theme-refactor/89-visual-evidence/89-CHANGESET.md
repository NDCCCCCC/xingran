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

## 终判结论（Plan 04 · W4 收口，2026-10-29）

- **矩阵版本**：`matrix-diff-89.cjs`（白名单归类模式，登记表与本文档 A/B 两节同步）+ `matrix-fixture.js` / `matrix-sample.js`；采样方法 = `p89-final-run.cjs`（dev :4000 + `$TEMP/p75-uat` 后端 + Node WebSocket 直连 CDP；模式切换经 `/user/settings` 分段卡走 **settingsStore 服务器偏好**，非仅 data-color-mode 属性——89-02 经验沿用）。
- **终跑结果**：360 格（21 形态 × 明暗 × carrier/inner × 5 属性）；**light = 0 / dark = 20（全部 A 类）/ 未归类 = 0**，exit 0，`RESULT: PASS`。
- **A 类格数** = 20（16 形态 carrier.bg + f05/f06/f11/f19 共 4 格 inner.bg），`rgb(250,250,249) → rgb(15,46,27)`；**B 类格数 = 0（矩阵口径空集**，token 变更不可见，见上文 B 类实机校准；cssVar 侧生效已由 `probe-{light,dark}.txt` 实证：dark `--ant-color-bg-container=#1a2e1f` / `--ant-color-border=#2d4d33` / `--ant-color-bg-elevated=#1f3524`，light 三值零漂移 `#FFFFFF/#DBD7CE/#FFFFFF`）。
- **与 W3 末态对照**：差异集合**完全一致**（同为 20 格 A 类、形态集合不多不少）——W3 hex 批次确认为零渲染差异，89 全程无登记面之外的视觉改动。
- **执行人**：89-04 executor（GSD）；证据：`after-{light,dark}.json`、`matrix-diff-result-89.txt`、`probe-{light,dark}.txt`、`r2-*-{light,dark}.png` ×12。

## 未清偿债归口表（D-1~D-7，Plan 04 终段定稿）

> 编号以 89-RESEARCH.md §5.2 / 89-04-PLAN 为准（本文档上文 D-3/D-4/D-5 为 Plan 02 期临时编号，D-4/D-5 内容与本表 D-5/D-4 互换，以本表为权威）。

| # | 债 | 量化 | 归属 |
|---|---|---|---|
| **D-1** | 暗色输入框描边对比度 1.56:1 < 3:1（X-2 后 3:1 判据的承接面） | `--input-border #2d4d33` vs `#0f2e1b` = 1.56:1；提亮至 `#4a7d52` 3.05:1 级别 | **Phase 92**（暗色 mega-cluster 域） |
| **D-2** | X-2 例外：暗色输入框底 vs 页面底 3:1 判据数学不可满足 | 两者同为 `#0f2e1b` → 1:1；该色对任意背景上限 1.43:1 | **已登记例外**（89-QUANT-SNAPSHOT.md「X-2 显式例外登记」节 + REQUIREMENTS VISFIX-01 文字更正，本 phase 完成） |
| **D-3** | 暗色占位符 AA 对比度欠账 | `#707068`（`--theme-neutral-400` 解析值）on `#0f2e1b` = 2.95:1 < AA 4.5:1（串色修复**暴露**的既有问题，89 不提亮以守住 diff 预算） | **Phase 92** |
| **D-4** | `Table` / `Menu` / `Select.optionSelectedBg` 暗色 token 仍为浅值 | `cream.headerBg #E9EFEB` 等；暗色下由 index.css dark cluster 暂承 | **Phase 91/92** |
| **D-5** | `Button.defaultBorderColor` 暗色仍为浅值 `#DBD7CE`（保守不改） | 暗色按钮描边由 index.css dark cluster（`[data-color-mode="dark"] .ant-btn-default` → `#345b3a`）承接，token 侧欠账 | **Phase 92** |
| **D-6** | `.xr-table-zebra` 暗色近白斑马纹 | `index.css:5118` 无暗色覆盖 | **Phase 92** |
| **D-7** | REQUIREMENTS / 文档更正 | ① VISFIX-01「对比度达标」→ X-2 例外指向；② STYLE-04「18 个文件命中」→ 17；③ F3 前置障碍段机制表述对齐 X-1（alias 回填 + 按模式分支显式给值，核实已是对齐状态） | **本任务内完成**（.planning/REQUIREMENTS.md 已更正） |

另有两条既有登记沿用：`--theme-zebra` 后置 `:root` 未迁（Plan 03 T14 域）；hex ratchet 净增 2 处（W1 结构对称新定义，92 终态 ≤300/≤430 之外的 W3 已消化部分见终值快照——终值 M7=304 / M8=440 已回落至 W0 之下）。
