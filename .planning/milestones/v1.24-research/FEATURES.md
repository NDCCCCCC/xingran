# Feature Research — v1.24「视觉回归修复 · 权限定责 · 诊断接线 · 结构清偿」

**Domain:** 前端视觉回归（P0）+ RBAC 越权定责 + 工位物理链路空因诊断接线 + lint/结构债清偿（企业运维管理系统，Go 1.24 + React 19.2 + Ant Design 6.6.4）
**Researched:** 2026-10-28
**Confidence:** HIGH（全部结论基于本地代码实测：lint 全量实跑、antd 6.6.4 / `@rc-component/input` 源码逐行核读、Go 中间件与 service 实读、v1.23 UAT 报告与 68.1 UI-SPEC 原文对照；**未做任何源码修改**）

> **口径校准（规划时以本节数字为准，与 seeds/用户描述的差异已逐条标出）**
>
> | 项 | 用户口径 | 实测 | 说明 |
> |---|---|---|---|
> | lint errors | 17 | **17** ✅ | 实跑 `npx eslint .`（与 `scripts/assert-lint-budget.mjs` 口径一致，非 `src` 子集） |
> | exhaustive-deps | 12 处 | **12 处，是 17 errors 的子集** | 不是额外的 12 个错误 |
> | react-refresh | 33 处 | **33 处 warning，分布在 17 个文件** | 与 `package.json` 的 `--max-warnings 34` 预算吻合 |
> | devices 页行数 | 692 | **714** | v1.22 Phase 67 T3 拆分后为 692（`67-SUMMARY.md:12`），此后又增长 22 行 |
> | warnings 总数 | — | **34** = react-refresh 33 + 1 条 unused-disable 指令（`src/components/shared/PageTable.tsx:180`） | |
>
> **17 errors 的精确构成**（`npx eslint .` 实跑）：`react-hooks/exhaustive-deps` 12 + `react-hooks/set-state-in-effect` 1 + `react-hooks/preserve-manual-memoization` 1 + `react-hooks/immutability` 1 + `react-hooks/globals` 1 + `local/no-large-dropdown-list` 1。
> 注意后 4 条 react-hooks 新规则来自 **`eslint-plugin-react-hooks@7.0.1` 的 React Compiler 规则集**（`eslint.config.js:58` 展开 `reactHooks.configs.recommended.rules`），不是 v1.23 遗留——清零它们需要读懂 Compiler 语义，不能机械加 `eslint-disable`。

---

## 0. 一句话结论（Lead 先读这 7 行）

1. **双框不是「样式不好看」，是 v1.23 的 `allowClear` 默认化把 309 个输入框（235 AppInput + 74 AppTextArea，89 个文件）全部切进了 antd 的 `affix-wrapper` 渲染路径**，而 `src/index.css:875` 的 `.ant-input { border: 1px solid ... !important }` 恰好命中 wrapper **内层**那个本该无边框的 `<input class="ant-input">`。这是全站视觉回归，影响面最大、成本最低、必须第一个做。
2. **#16 不是「前端按钮没藏」的 UI 小问题，而是后端越权**：全仓 39 处路由权限中间件**全部是 ANY-of 语义**（`RequirePermissions`/`RequirePermissionsWithQuery`），`internal/api/router.go:541-546` 把 `[list, add, edit, delete]` 放在同一个数组里 —— 只持有 `ops:building:list` 的用户**在 API 层就能删楼宇**。v1.23 UAT 给的根因候选②（"手工插库角色数据不完整"）**可以排除**。
3. **3f130 后端已交付（v1.23 commit 8b85fe6b），前端一行没接**：`src/lib/opsApi.ts:617-620` 仍调裸 `/physical` 拿数组。接上后，「有数据但 emptyReason=no_user_bound」是**正常业务态**（后端明确「未绑用户压过有数据」），必须用 Alert 而不是 Empty 呈现。
4. **F5 跳 /dashboard 的根因已定位到行**：`src/router/DynamicRoutes.tsx:315` 的 catch-all `<Navigate to="/dashboard">` 在**首帧菜单尚未水合**时抢先命中，随后 `:243-247` 的 `saveLastPath` 把 `ruoyi_last_visited_path` 覆写成 `/dashboard` —— 与 UAT 观测完全一致。
5. **ColumnConfigModal:136 不是「随手补个注释」**：68.1 UI-SPEC §1.1 判定表点名了 6 处「非删除非危险确认」必须带 `eslint-disable-next-line` 理由注释，**5 处已有、仅此 1 处缺**；且 §4 要求同步登记进豁免理由表。
6. **react-refresh 33 处是三种不同性质**，其中 **13 处是规则误判**（`export const X = <已导入标识符>` 被当成组件导出），**2 处是 `lazy()` 不被规则识别**，只有 18 处是真问题。
7. **额外发现 3 项未登记的债**（见 §8）：豁免理由表已失同步（`check-exemption-reasons.mjs` **exit 1 / 123 项**）、lint 预算双写死在两处、`AppSelect` 契约与项目自有规则 `local/no-large-dropdown-list` 直接冲突。

---

## 1. Feature Landscape

### 1.1 Table Stakes（本里程碑「不做完就不算收尾」的部分）

| # | Feature | Why Expected | Complexity | Notes |
|---|---------|--------------|------------|-------|
| F1 | **输入框单层外框**（P0） | v1.23 allowClear 默认化后**全站 309 个输入框**双框，用户一眼可见 | **LOW**（1 处 CSS + 1 处 JSX 契约） | 见 §2.1。根因已 100% 定位到 `src/index.css:875` 与 `src/components/ui/AppInput.tsx:12` / `AppTextArea.tsx:12` |
| F2 | **#16 行操作权限 + 后端越权修复** | 受限用户看到「删除」按钮＝系统看起来没有权限模型 | **HIGH**（前端补判定 + 后端改 39 处中间件语义） | 见 §2.2。**这是 v1.24 唯一的安全项**，必须单独排期并单独 UAT |
| F3 | **3f130 空因诊断前端接线** | 后端已交付的诊断结论没人看＝白做 | **LOW**（1 个 API wrapper + 1 个组件分支） | 见 §2.3。端点契约已完整（`PhysicalDevicesResult`） |
| F4 | **F5 刷新留在当前页** | 刷新跳仪表盘＝用户丢失上下文，分页恢复链路同时被废 | **MEDIUM**（首帧路由时序） | 见 §2.4。根因定位到 `DynamicRoutes.tsx:315` + `:243-247` |
| F5 | **ColumnConfigModal:136 豁免注释 + 理由表登记** | 契约登记完整性；68.1 §8 #12 明确列了它 | **LOW**（1 行注释 + 1 条 JSON） | 见 §2.5。连带发现豁免理由表整体失同步（§8-A） |
| F6 | **lint 17 errors + 33 react-refresh 清零** | 契约要有人守；`--max-warnings 34` 是 v1.24 拍的预算 | **MEDIUM**（机械为主，4 条 Compiler 规则要判断） | 见 §2.6。清零后必须同步降预算到 0，否则门禁形同虚设 |
| F7 | **devices 页 714 行拆分** | 可维护性 | **MEDIUM**（约 2h，历史两次延期） | 见 §2.7。**建议降级为 v1.24 可选项**，与 F1-F6 无依赖 |

### 1.2 Differentiators（做了明显提升，但不做不阻塞收尾）

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| D1 | **行操作权限判定下沉到契约层**（`createDeleteAction` / `ActionButtons` 接 `perm` 参数） | 一次实现、全站生效，杜绝「新页面又忘了藏」 | MEDIUM | v1.22 已锁定决策同构：「默认值必须下沉封装层，禁止逐页补 prop」。**强烈建议 F2 采用此路径**而非逐页加 `canEdit &&` |
| D2 | **CSS 契约测试**（jsdom 测不出 border，必须用 computed-style 断言） | 防止双框类回归再次发生 | LOW | 建议加一个 Playwright/agent-browser 冒烟脚本，断言 `.ant-input-affix-wrapper > .ant-input` 的 `border-width === '0px'` |
| D3 | **诊断端点统一契约**（`/physical` 与 `/physical-diagnosis` 二选一） | 避免两个端点返回不同形状、前后端各记一半 | LOW | 后端已保证 `result.Devices` 恒等于 `/physical` 返回值（`workstation_physical_diagnosis.go:56-58`），前端可直接替换 wrapper，**旧端点保留兼容** |
| D4 | **lint 预算归零 + CI 硬门禁** | 清零后立刻把 `--max-warnings 0` 写进 `package.json` 与 `assert-lint-budget.mjs`，防止回潮 | LOW | 见 §2.6「判据」 |

### 1.3 Anti-Features（看似合理、应明确排除）

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| A1 | **给 `.ant-input` 规则加 `:not(.ant-input-affix-wrapper *)` 之类的复杂选择器** | CSS `:not()` 复杂选择器在旧 Safari 不支持；且本项目 4 处暗色块（`:2252/:3717`）各自重复定义 border，改一处漏三处 | 显式覆写 `.ant-input-affix-wrapper > .ant-input { border: none !important; }`，4 个暗色/浅色块都要同步（见 §2.1 修法矩阵） |
| A2 | **把 AppInput/AppTextArea 的 `allowClear` 默认改回 false** | 「双框只是视觉问题，把默认值退回去就完了」——但这会**撤销 v1.23 的核心交付**（TextArea 一键清空，UAT SC1/SC2 已通过） | 修 CSS。根因在 CSS 覆盖，不在契约默认值 |
| A3 | **#16 只做前端隐藏按钮** | 后端 ANY-of 语义不变 ⇒ 受限用户仍可直接 curl 删除楼宇。**只藏按钮 = 假修复** | 必须同时修后端：按动作分权（§2.2「后端修法」） |
| A4 | **逐页给 39 个路由组把 `RequirePermissions` 换成 `RequireAllPermissions`** | `RequireAllPermissions` 会把 `list` 也变成必需 ⇒ 只读用户连列表都打不开 | 按 **动作** 拆中间件组（读一组 / 写一组），保持查询接口的 `spaces:list` 旁路能力 |
| A5 | **devices 页拆分放进 v1.24 必交** | 与 F1-F6 无依赖、纯整洁收益、历史已延期两次（68.1-PLAN:508/836 明确记「拆分风险 > 收益」） | 放 v1.24 可选项或 v1.25，**不占用 F1/F2 的时间预算** |
| A6 | **为 react-refresh 33 处统一加 `eslint-disable` 注释** | 33 处里 13 处是误判、2 处是 `lazy()` 合法用法；一刀切豁免＝把规则废掉 | 按 §2.6 的三类分流处理（拆分 18 / 规则修正 13 / 豁免 2） |

---

## 2. 逐项详析：done 判据 · 机械验证 · 依赖 · 风险

### 2.1 F1｜输入框双重外框（P0）

#### 根因（已 100% 定位，源码级证据）

**antd 6.6.4 的渲染契约**（`node_modules/@rc-component/input/es/BaseInput.js:42` + `:47-50`）：

```js
const hasAffix = hasPrefixSuffix(props);   // = !!(prefix || suffix || allowClear)
let element = cloneElement(inputElement, {
  value,
  className: clsx(inputElement.props?.className, !hasAffix && classNames?.variant) || null
});
```

- **关键点**：`ant-input-outlined`（antd 6 的边框承载类）**只在 `!hasAffix` 时挂到内层 input 上**；一旦有 affix，边框就转移到外层 `.ant-input-affix-wrapper`（`:79-83`、`:92` 的 `classes?.variant`），内层 `<input class="ant-input">` **本来就没有边框**。
- `hasPrefixSuffix` = `prefix || suffix || allowClear`（`utils/commonUtils.js:4-6`）——**`allowClear` 单独就足以切换渲染路径**。
- v1.23 把 `allowClear` 默认化（`src/components/ui/AppInput.tsx:12`、`AppTextArea.tsx:12`：`{ allowClear = true, ...rest }`）⇒ **全站所有 AppInput/AppTextArea 都走了 affix-wrapper 路径**。

**项目 CSS 的越权覆盖**（`src/index.css`）：

| 行号 | 规则 | 后果 |
|---|---|---|
| **875** | `.ant-input { border: 1px solid var(--input-border) !important; }`（块起于 `:871`） | **命中内层 `<input class="ant-input">`** ⇒ 内层被强制画出 1px 边框 + 圆角 + 背景 ⇒ 与外层 wrapper 边框叠加 = **双框** |
| 909-913 | `.ant-input-affix-wrapper { background/border/color/border-radius !important }` | 外层也被项目接管（这是**期望**的：外框由项目主题控制） |
| 888-892 / 920-923 | `.ant-input:focus` / `.ant-input-affix-wrapper-focused` 各自 `box-shadow: 0 0 0 2px` | 聚焦时**双层光晕**（内层 input 与外层 wrapper 各一圈） |
| 2252-2259 | `[data-color-mode="dark"] input, textarea, .ant-input, .ant-input-number, .ant-input-affix-wrapper { border-color: ... !important }` | 暗色下**全局元素选择器** `input`/`textarea` 再次命中内层；只改 `border-color` 不改 `border-width`，但 `:875` 的 `border` 简写已把宽度设成 1px ⇒ **暗色同样双框** |
| 3717-3721 | `[data-color-mode="dark"] .ant-input { border-color: ... !important }` | 同上，第三处重复 |
| 2268-2273 / 2275-2284 / 3723-3734 | 暗色 hover/focus 变体 | 各自再加一次 `border-color`/`box-shadow` |

**结论**：双框 = `:875` 的一条 `!important` 简写，落在 antd 6 刻意做成「内层无边框」的元素上。antd 自己的 CSS（`node_modules/antd/dist/antd.css`）对 TextArea 甚至显式写了 `.ant-input-affix-wrapper > textarea.ant-input { border: none; outline: none; background: transparent; }` —— 我们的 `!important` 精确地覆盖掉了它。

**影响面实测**：
- `<AppInput` 235 处 / 89 个文件；`<AppTextArea` 74 处 ⇒ **合计 309 个输入框全部双框**。
- `prefix=` / `suffix=` 全仓 139 处 ⇒ 加了 prefix/suffix 的裸 `<Input>`（若未走契约）同样双框。
- `Input.Search` / `addonBefore` / `addonAfter` 场景：antd 6 下内层同样是 `class="ant-input"` ⇒ 双框。

#### 用户可观察的「done」

> **任意一个输入框（含空值、有值、聚焦、清空按钮 hover、禁用、只读）在浅色与暗色两种模式下，屏幕上只出现一圈 1px 边框；聚焦时只出现一圈光晕；不存在「外框内还有一圈内框」。**

#### 完成判据如何机械验证

| # | 判据 | 验证方式 | 通过标准 |
|---|---|---|---|
| F1-A | **内层零边框** | 浏览器 console / agent-browser：`getComputedStyle(document.querySelector('.ant-input-affix-wrapper > .ant-input')).borderWidth` | `=== '0px'`（对 TextArea 同样成立） |
| F1-B | **外层单边框** | `getComputedStyle(document.querySelector('.ant-input-affix-wrapper')).borderWidth` | `=== '1px'`，且 `border-color` 等于 `var(--input-border)` 解析值 |
| F1-C | **无内层背景/圆角** | `getComputedStyle(内层).backgroundColor` / `.borderTopLeftRadius` | 透明 / `0px`（当前是 `--input-bg` + `--theme-radius-md`） |
| F1-D | **聚焦单圈光晕** | 聚焦后 `boxShadow` 计算值 | 只含 **1 组** `0 0 0 2px` 光环（当前内层与外层各一组 ⇒ 2 组） |
| F1-E | **DOM 层级不变量** | `document.querySelectorAll('.ant-input-affix-wrapper > .ant-input').length > 0` 且内层 `border-width = 0` | 结构性断言，与视觉解耦 |
| F1-F | **截图矩阵**（见 §3） | 11 类输入框 × 明暗 2 态 × 至少 3 个代表页面 | 全部无双框 |
| F1-G | **回归护栏** | `npm run lint` / `npm run build` / `npm test` | errors ≤ 17（不新增）、build exit 0、现有 309 处用例不回归 |
| F1-H | **契约不破** | grep `src/components/ui/AppInput.tsx` + `AppTextArea.tsx` | `allowClear = true` 默认值**保持不变**（v1.23 核心交付不被撤销） |

#### 修法矩阵（4 个块必须同步改 —— 本项目已踩过「flat config / 分散定义静默失效」的坑）

| 块 | 位置 | 需要做什么 |
|---|---|---|
| 浅色主块 | `src/index.css:871-905` | `.ant-input` 的 `border/background/border-radius` 需**排除 affix-wrapper 内层** |
| 浅色 wrapper 块 | `src/index.css:908-929` | 保留（外框由主题接管，正确） |
| 暗色块 A | `src/index.css:2252-2260` | 全局 `input, textarea` 元素选择器命中内层 ⇒ 需收敛到 `.ant-input-affix-wrapper > .ant-input` 时把 `border-*` 归零 |
| 暗色块 B | `src/index.css:3717-3757` | 同上（`.ant-input` / `.ant-input:hover` / `.ant-input:focus` / `.ant-input[disabled]` / `.ant-input-affix-wrapper*` 五处） |
| 局部已有覆写 | `src/index.css:2051-2084`（`.dept-tree-search .ant-input-affix-wrapper .ant-input`、`.user-form-input.ant-input-affix-wrapper .ant-input`） | 这两处**已经在做正确的事**（内层去边框）⇒ **可作为修法模板**：`src/index.css:2056` 的写法是本仓已有的、已验证的「内层清零」范式 |

> **推荐修法（结论）**：新增一条集中覆写 —— `.ant-input-affix-wrapper > .ant-input { border: none !important; background: transparent !important; box-shadow: none !important; }` + `.ant-input-affix-wrapper > .ant-input:focus { box-shadow: none !important; }`（focus 的光晕交给 `.ant-input-affix-wrapper-focused`，`:920-923` 已有）。**不要**用复杂 `:not()` 选择器（浏览器兼容 + 4 个暗色块的漏改风险）。
>
> ⚠️ **`.ant-input-number` 不受影响**：antd 6 的 InputNumber 内层是 `.ant-input-number-input`（非 `.ant-input`），项目 CSS 未命中 ⇒ 单框。**`.ant-select-selector`（`:873`）同样单框**（antd 6 的 Select 边框也在 `.ant-select-selector` 上，同一元素）。**`.ant-picker` 项目 CSS 未设 border** ⇒ 单框。**结论：双框只影响 antd `Input` 与 `Input.TextArea`。**

#### 依赖与前置

- **无外部依赖**，可与 F2-F7 完全并行。
- 前置：无（不需要后端、不需要 PG 环境）。
- **强烈建议排在最前**：F1 是唯一的 P0，且改完立刻可截图验收；F2 的 PG 复测依赖外部环境，最容易卡。

#### 风险点

| 风险 | 概率 | 影响 | 缓解 |
|---|---|---|---|
| 漏改 4 个暗色/浅色块中的任一个 → 只在暗色或只在某状态双框 | **高** | 中 | 判据 F1-F 强制矩阵覆盖明暗 × 8 态；修法集中在**一条**新规则而非分散改 4 块 |
| 内层 `border:none` 后，`input:focus` 的浏览器默认 outline 冒出 | 中 | 中 | 显式 `outline: none !important`（antd 自身 CSS 对 textarea 就是这么做的） |
| 修好双框后**清空按钮（✕）与内层文字重叠**（antd 靠 `padding-inline-end: var(--ant-padding-lg)` 预留，`:875` 的 `padding` 未被项目覆盖，但 `border` 简写不影响 padding） | 低 | 低 | 截图矩阵必须包含「有值 + hover 显示 ✕」态 |
| `src/index.css` 有 5590 行、4 处 input 边框定义 —— 后续再加 input 样式会再次踩坑 | 中 | 中 | 建议在 §2.1 的结论里登记「input 边框唯一定义点」并写进 `docs/开发规范.md` |
| 顺手把 `allowClear` 默认改回 false「省事」 | 中 | **高**（撤销 v1.23） | F1-H 判据 + A2 反-feature 已登记 |

---

### 2.2 F2｜#16 行操作权限（**v1.24 唯一安全项**）

#### v1.23 UAT §8 #16 原文与观测

> `75-UAT.md:51`：
> **#16 行操作权限不变 ❌** — 受限用户 uatview（角色仅含 `ops:building:list/query`，无 delete）登录后楼宇页 9 行全部仍显示「编辑/删除」按钮（delCount=9）。
> **环境限制**：角色创建 API 在 sqlite 后端报 500（`ON CONFLICT` 不匹配，PG 不受影响），角色-菜单关联为手工插库，未走正常缓存加载链路，后端实际下发权限集未能独立核实（`/system/menus/permissions` 对该用户 403）。
> **根因候选**：①前端 hasPerm 在权限获取失败时未隐藏危险按钮；②手工插库角色数据不完整导致后端权限异常。**建议在 PG 生产库复测定责**。

`75-UAT.md:87` 的 FAIL-2 根因候选与上表相同。

#### 定责结论（本次研究的关键产出 —— 与 UAT 候选不同）

**候选②（"手工插库角色数据不完整"）可以排除，候选①（"权限获取失败时 fail-open"）也不成立。真实根因是两层叠加：**

**第一层：前端根本没有权限判定代码。**

```
src/pages/operations/buildings/index.tsx:374-378
  // 行操作：编辑 + 删除（数量/顺序与迁移前一致；删除走 createDeleteAction 唯一入口）
  const rowActions = (record: Building) => [
    { key: "edit", label: "编辑", icon: <EditOutlined />, onClick: () => openModal(record) },
    createDeleteAction(record.id, handleDelete, { target: `楼宇 ${record.name}` }),
  ];
```

- 该文件**全文无 `permissions` / `hasPerm` / `useMenuStore` 引用**（实测 grep 无命中）。
- 全仓只有 **12 个文件**用了 `useMenuStore.permissions`（`src/pages/` 下 10 个 + 2 个测试）；行操作权限判定是**逐页 opt-in 模式**，`network/ports`（`:131-132`）、`operations/workstations/LocationAliasDrawer`（`:63-65`）、`vdi/VirtualMachineList`（`:46`）有，**楼宇页没有**。
- ⇒ 不是「权限取不到时 fail-open」，而是**这条路径上从来没有 fail-close 的代码**。
- 补充事实：前端取权限的端点是 `POST /system/my-menus/permissions`（`src/lib/menuApi.ts:23-26`），后端 `internal/api/v1/system/menu_router.go:63-65` **没有挂任何权限中间件**（所有登录用户可访问）⇒ UAT 里那个 403 是打在了**管理员**端点 `/system/menus/permissions`（`internal/api/router.go:199-203` 需 `system:menu:*`）上，**不是**前端真正用的那个。⇒ 「前端拿不到权限集」这条路径不成立。

**第二层（更严重）：后端写接口的权限门是 ANY-of 语义 ⇒ 受限用户在 API 层就能删楼宇。**

```
internal/api/router.go:541-546
  buildings.Use(middleware.RequirePermissionsWithQuery([]string{
      "ops:building:list",
      "ops:building:add",
      "ops:building:edit",
      "ops:building:delete",
  }, []string{"ops:building:spaces:list"}, core))
```

- `pkg/middleware/permission.go:277-303` `RequirePermissions`：**「需要多个权限中的任意一个」**（`for _, permission := range permissions { if checkUserPermission(...) { c.Next(); return } }`）。
- `RequirePermissionsWithQuery`（`:316-352`）在非查询路径上 `checkPerms = permissions`（仍是那 4 个）⇒ **`POST /ops/building/:id/delete` 只要求用户命中 `ops:building:list` 之一**。
- `internal/api/v1/operations/building_handler.go:175-183` `Delete` **无任何 handler 级权限复核**。
- 全仓 `internal/api/router.go` 中 `middleware.Permission(`（单权限严格语义）**出现 0 次**；39 处全部是 ANY-of 的 `RequirePermissions` / `RequirePermissionsWithQuery`。
- ⇒ **只持有 `ops:building:list/query` 的 uatview，理论上 `POST /ops/building/:id/delete` 会返回 200 并真的删掉楼宇。**

**附带发现（放大风险）**：`pkg/middleware/permission.go:136-180` 的 `checkUserPermissionUncached` 有**两级继承**：
1. **F 型按钮权限 → C 型父菜单权限**（`:159-196`）：持有 `ops:building:query`（F 型）⇒ 命中 `ops:building:list`（C 型父）。
2. **模块级权限**（`:198-216`）：`extractModulePermission("ops:building:list")` ⇒ `"ops:building"`，若存在 perms 恰为 `ops:building` 的菜单行则再放行。

⇒ 即使把 ANY-of 数组改成 `[list]`，只读用户仍可能通过继承拿到写权限。**修法必须同时处理继承语义。**

#### 用户可观察的「done」

> **受限用户（仅持 `ops:building:list/query`）登录后：① 楼宇页 9 行**看不到**「编辑/删除」按钮；② 直接 `POST /ops/building/:id/delete` 返回 403「没有访问权限」且数据未删除；③ 超级管理员与持完整权限的用户**照常看到并能执行**按钮。**

#### 完成判据如何机械验证（分三层，缺一不可）

| 层 | 判据 | 验证方式 | 通过标准 |
|---|---|---|---|
| **L1 UI** | 受限用户楼宇页行操作列 | 浏览器（agent-browser）数 `.action-btn-link-danger` / 「编辑」链接 | delCount **= 0**、editCount **= 0**（对照 UAT 截图 `16-no-perm-no-delete-btn.png` 的 delCount=9） |
| **L2 API 兜底** | 受限用户直接调用写接口 | 用受限用户的 token `curl -X POST /api/v1/ops/building/{id}/delete` | HTTP **403** + body `code=403`；**且该楼宇仍存在**（`GET /ops/building/list` 能查到） |
| **L3 API 正向** | 管理员/全权限用户 | 同上 | HTTP 200 + 楼宇被删（证明不是「一律拒绝」） |
| **L4 权限集可核** | 受限用户的权限集 | `POST /system/my-menus/permissions`（**该端点无权限门**，`:63-65`） | 返回 `["ops:building:list","ops:building:query", …]`，**不含** `add/edit/delete` |
| **L5 全域** | 其他模块同款 | 对 `network/ports`（已有 canWrite）、`system/user`、`operations/workstation` 各抽 1 个写接口 | 受限用户一律 403 |
| **L6 不回归** | 现有契约 | `npm test` + v1.19 port-write 62 测试 + `npm run build` | 全绿 |

**「判为缺陷」的标准（PG 环境复测）** —— 满足**任一条**即判缺陷：

| # | 判据 | 观测值 | 判定 |
|---|---|---|---|
| **D1** | **UI 层**：受限用户楼宇页仍见「编辑」或「删除」 | delCount > 0 或 editCount > 0 | **前端缺陷**（缺权限判定） |
| **D2** | **API 层**：受限用户 `POST /ops/building/:id/delete` | 返回 **200** 且楼宇消失 | **后端越权缺陷（严重）** —— 优先级高于 D1 |
| **D3** | **API 层**：受限用户 `POST /ops/building/:id/update` | 返回 **200** 且内容被改 | 后端越权缺陷（严重） |
| **D4** | **API 层**：受限用户 `POST /ops/building`（新增） | 返回 200 | 后端越权缺陷（严重） |
| **D5** | **权限集**：`/system/my-menus/permissions` 返回含 `ops:building:delete` | 权限集含写权限 | **数据/角色配置问题**（非代码缺陷）⇒ 判为「环境构造错误」，须重建角色后重测 |
| **D6** | **权限集取不到**（该端点 404/500） | — | 环境问题，重测 |
| **D7** | **对照组**：全权限用户看不到「删除」按钮 | delCount = 0 | 修过头（fail-close 全禁），**误伤缺陷** |

> **判定顺序（D5/D6 优先）**：先核权限集（无权限门的端点，最可靠），再判 UI，最后判 API。这样能把「环境构造错误」与「代码缺陷」分开 —— v1.23 UAT 之所以无法定责，正是因为它把「角色-菜单关联为手工插库、未走正常缓存加载链路」当成了唯一疑点，而没去查那个**根本不需要权限**的权限集端点。

#### PG 环境复测的具体步骤（可直接下发给执行者）

**前置**
- PG 环境：`configs/config.yaml` → `database.type: postgres`；**不要**用 sqlite（`75-UAT.md:91` 记录角色批量分配菜单在 sqlite 上 500：`ON CONFLICT clause does not match any PRIMARY KEY or UNIQUE constraint`）。
- 浏览器 UAT 走 prod build + `npm run preview`（沿用 Phase 75 的 `VITE_API_BASE_URL=http://localhost:9000/api/v1 npm run build` → 4173）。
- 清 sessionStorage/localStorage 重新登录（Phase 75 约定）。

**步骤**
1. **建角色（UI 正常路径，不用手工插库）**：系统管理 → 角色管理 → 新增角色，角色键 `uat_readonly`，**只勾选**「楼宇管理」菜单 + 其下的「楼宇查询」按钮权限（**不勾** 新增/修改/删除）。⇒ 这一步直接绕过 sqlite 500 的坑，也是 v1.23 失败的根因。
2. **建用户**：系统管理 → 用户管理 → 新增用户 `uatview2`，分配角色 `uat_readonly`（**不要**给 admin 角色）。
3. **核权限集（决定性一步）**：用 `uatview2` 登录（或取其 access token）`POST /api/v1/system/my-menus/permissions` ⇒ 断言返回**不含** `ops:building:add/edit/delete`。
   - 若**含** ⇒ 角色配置/菜单继承有问题（`sys_menu` 的 `perms` 或父子结构），判 **D5**，先修数据再重测。
   - 若**不含** ⇒ 继续第 4 步。
4. **UI 复现**：登录 `uatview2` → 运维管理 → 楼宇管理 ⇒ 数行内「编辑」「删除」按钮数量。
5. **API 复现（决定性）**：同一 token 直接打写接口：
   - `POST /api/v1/ops/building/{真实楼宇ID}/delete` ⇒ 期望 **403**
   - `POST /api/v1/ops/building/{真实楼宇ID}/update`（改个备注）⇒ 期望 **403**
   - `POST /api/v1/ops/building`（新增一栋测试楼宇）⇒ 期望 **403**
   - `POST /api/v1/ops/building/list` ⇒ 期望 **200**（读必须仍然通）
6. **回归对照**：admin 账号重复第 4/5 步 ⇒ 期望按钮可见、接口 200。
7. **证据留档**：每步截图 + 每条 curl 的完整请求/响应，落 `screenshots/` + `api-transcript.md`。

**⚠️ 权限缓存 30s**：`pkg/middleware/permission.go:36` `mwPermCacheTTL = 30 * time.Second`；`InvalidateMiddlewarePermCache` **当前未接线**（`:64-66` 注释明说）⇒ **改角色后必须等 ≥30s 再测**，否则会拿到旧权限。UAT 脚本必须显式 `sleep 31`。

**⚠️ 生产环境红线**：生产 `10.62.10.39:9000` 开启 slider 验证码（`75-UAT.md:111`）⇒ 复测用**准生产 PG** 或本地 PG，不要硬闯生产。

#### 后端修法（结论 —— 这是本项的真正工作量）

| 方案 | 做法 | 评价 |
|---|---|---|
| **B1 按动作拆中间件（推荐）** | 读接口组挂 `RequirePermissions(["ops:building:list"], …)`；写接口**逐个**挂 `RequirePermissions(["ops:building:add"])` / `["ops:building:edit"]` / `["ops:building:delete"]` | 语义正确、可审计；但 `internal/api/router.go` 有 39 处组、约 150+ 端点 ⇒ 工作量大，**建议按域分波**（operations → system → network → 其他） |
| B2 用 `RequireAllPermissions` | 把数组换成 ALL | ❌ 会把 `list` 变必需 ⇒ 只读用户打不开列表 |
| B3 路由分组内按动作再分组 | 在 `buildings` 组内再开 `buildings.POST("/:id/delete", middleware.RequirePermissions([...]), handler.Delete)` | ✅ 与 B1 等价但改动局部、可逐模块推进（**推荐落地形态**） |
| B4 修继承语义 | 禁止「F 型按钮 → C 型父菜单」的继承用于**写**权限判定 | ⚠️ 必须做，但需评估 `checkUserPermissionUncached:159-196` 的既有依赖（注释说这是 P1 越权修复，不能直接删） |

**⚠️ 回归面（必须评估）**：`RequirePermissionsWithQuery` 的 `spaces:list` 旁路（`:539-546` 注释）是「楼宇空间」可视化页复用 building list 的机制 —— 改写权限分组时**不能破坏**这条旁路（`68.1 §8` 未覆盖，但 `pkg/middleware/permission.go:305-315` 的注释是明确契约）。

#### 前端修法（结论）

**推荐 D1（契约层下沉）**，与 v1.22 已锁定决策同构（「默认值必须下沉封装层，禁止逐页补 prop」，`PROJECT.md:30`）：

- 给 `createDeleteAction`（`src/components/shared/ActionButtons.tsx:114`）与 `ActionButton`（`:19-28`）增加**可选** `perm?: string`；
- `ActionButtons` 渲染时，若 `perm` 存在且 `useMenuStore.permissions` 不含之 ⇒ **不渲染该按钮**（fail-close）；
- 楼宇页 `rowActions` 传 `perm: "ops:building:delete"` / `perm: "ops:building:edit"`。

**取舍**：
- ✅ 一处实现、全站生效、杜绝新页面再忘。
- ⚠️ `ActionButtons.tsx` 是 `src/components/shared/**`（eslint 封装层豁免块 `eslint.config.js:265-287` 覆盖范围内）⇒ 在其中引入 `useMenuStore` 会让这个**纯展示组件**变成 store 依赖组件 —— 需评估 `__tests__` 里对 `ActionButtons` 的渲染测试是否会因无 store Provider 而炸。
- ⚠️ `perm` 缺失时**默认显示**（不 fail-close）还是**默认隐藏**（fail-close）？**建议默认显示 + 显式传 perm**（避免一次性禁用全站按钮造成大范围误伤），但**必须**在 SPEC 里写清这条，并登记「新增破坏性操作必须传 perm」的 lint/评审约束。
- ⚠️ 备选：逐页加 `canEdit &&`（最直白，但 v1.22 决策明确反对）。

#### 依赖与前置

| 依赖 | 类型 | 状态 |
|---|---|---|
| PG 环境（准生产或本地 PG） | **硬阻塞** | ❌ 需用户提供 |
| 能用 UI 正常路径建角色 | 硬阻塞 | ✅ PG 下正常（sqlite 下 500） |
| 权限缓存 30s 等待 | 流程 | ⚠️ 需写进 UAT 脚本 |
| 后端 ANY-of 语义改造 | 工作量 | ❌ 未开始（39 处路由组） |
| 前端契约组件改造 | 工作量 | ❌ 未开始 |

**排期建议**：F2 拆成 **F2a（前端 UI 判定）** 与 **F2b（后端按动作分权）** 两个可独立验收的子项；F2b 是安全项，优先级高于 F2a。**若 PG 环境短期拿不到，F2a 可先做（UI 判据可在 admin 账号下用「无 perm 的模拟 store」验证），但 F2b 与 D2/D3/D4 判据无法闭环 ⇒ 里程碑不可收档。**

#### 风险点

| 风险 | 概率 | 影响 | 缓解 |
|---|---|---|---|
| **只修前端不修后端** ⇒ 假修复 | **高** | **严重** | D2/D3/D4 判据强制 API 层验证；A3 已登记为反-feature |
| 后端改 ANY-of ⇒ 大面积 403 误伤 | 中 | 高 | 分域分波 + 每波用 admin + 一个受限账号双向回归；`spaces:list` 旁路必须保 |
| 改继承语义（`:159-196`）破坏「只读角色能开页面」 | 中 | 高 | 继承只保留在**读**判定路径；写判定不走继承 |
| 契约层加 `perm` ⇒ `ActionButtons` 单测全炸 | 中 | 中 | 先跑 `src/components/shared/__tests__/` 现有用例摸底；必要时把 store 读取上移到调用方 |
| PG 环境拿不到 ⇒ #16 再次无法定责 | **高** | 高 | 提前向用户申请准生产 PG 或本地 PG 实例（这是 v1.23 的 BLOCKERS #2，未解决） |
| 权限缓存 30s 导致复测假阴性 | 中 | 中 | UAT 脚本固定 `sleep 31` |

---

### 2.3 F3｜3f130 空因诊断前端展示

#### 后端已交付（v1.23 commit 8b85fe6b）—— 契约已完整，前端一行没接

**端点**：`POST /ops/workstation-device/:id/physical-diagnosis`（`internal/api/router.go:702`）
**Handler**：`internal/api/v1/operations/workstation_device_handler.go:174-206` `GetPhysicalDevicesWithDiagnosis`
**Service**：`internal/services/operations/workstation_device_service.go:620-671` `GetPhysicalDevicesWithDiagnosis`

**返回结构**（`internal/services/operations/workstation_physical_diagnosis.go:56-100`）：

```ts
{
  devices: WorkstationDevice[],        // 恒等于旧 /physical 端点的返回值
  total: number,                       // = len(devices)
  emptyReason?: string,                // omitempty；见下方枚举
  emptyMessage?: string,               // omitempty；面向运维的中文结论
  warnings?: string[],                 // omitempty；**有数据时也会有**
  diagnosis: {                         // 逐段链路计数
    workstationExists: boolean,
    userBound: boolean,                // ← 3f130 的断点
    infoPointTotal: number,
    infoPointActive: number,
    infoPointWithPort: number,
    portResolved: number,
    deviceResolved: number,
    macMatched: number,
    assetMatched: number,
    adMatched: number,
  },
}
```

**`emptyReason` 枚举**（`workstation_physical_diagnosis.go:27-52`）：

| 值 | 含义 | `emptyMessage` 要点 |
|---|---|---|
| `""`（`PhysicalEmptyReasonNone`） | 正常有数据（且已绑用户） | 省略 |
| `"no_user_bound"` | **工位未绑定 user_id —— 3f130 的实际命中项** | 「该工位未绑定用户（sys_workstation.user_id 为空）。物理链路面板按「用户→信息点→端口→MAC」维度呈现，未绑定用户时链路无法归属，故为空。请先在工位管理中为该工位绑定使用人。」 |
| `"info_point_misbound"` | 无 active 信息点 / 信息点未绑端口 | 分 3 种子情形，各有针对性指引 |
| `"device_or_mac_missing"` | 端口 orphan / 设备 orphan / 无 MAC 采集 | 分 3 种子情形，各有针对性指引 |

**⚠️ 关键语义（必须写进 SPEC，否则前端会做错）**：

`workstation_device_service.go:800-866` `classifyPhysicalEmptyReason` 的注释明确写着：

> **关键决策：未绑用户压过「有数据」。** 3f130 实测 mac_match_count=1（链路是通的）但前端面板仍空白——因为该面板按「用户→物理链路」维度呈现，未绑用户时召回结果无从归属。因此「未绑用户」就是这一现象的真实空因，**必须报 no_user_bound**；设备本身的数据不丢，仍在 Devices 里，同时 Warnings 也会提示该告警，两个通道互补。

即：**`emptyReason === "no_user_bound"` 且 `devices.length > 0` 是完全正常的组合，不是异常。**
> ⚠️ **文档不一致（需在 v1.24 顺手修正）**：`workstation_physical_diagnosis.go:63-65` 的结构体注释写「EmptyReason 空因代码。**有数据时为 ""（JSON 中省略）**」，与实现（`:813-827`）**矛盾**。后端测试 `workstation_physical_diagnosis_test.go:388` 明确断言「未绑定用户优先级最高 —— 即使能召回设备也报 no_user_bound」⇒ **以实现与测试为准**，顺手把结构体注释改掉。

**⚠️ 前端当前状态（未接）**：
- `src/lib/opsApi.ts:617-620` 的 `workstationDeviceApi.getPhysical` 仍调**裸端点** `/ops/workstation-device/${id}/physical` ⇒ 拿不到任何诊断字段。
- 全仓 grep `emptyReason|emptyMessage|physical-diagnosis` → **前端 0 命中**（只有 `BaseWidget.tsx` 的同名 prop 与 `managerApi.ts` 的无关 `warnings`）。
- 当前空态是**写死的静态文案**：`src/components/operations/WorkstationDeviceTable/index.tsx:604-607`
  ```
  <div style={{ fontSize: 11, color: "var(--text-secondary)", padding: "4px 0" }}>
    该工位暂无通过 MAC→port→infoPoint 反推到的设备。 请确认工位下是否已配置
    ops_info_points 关联到网络设备端口。
  </div>
  ```
  ⇒ 这段静态文案**只覆盖 `info_point_misbound` 的一种子情形**，对 `no_user_bound` / `device_or_mac_missing` 是**误导性文案**（会把运维引向错误方向）。

#### 承载位置

- 组件：`src/components/operations/WorkstationDeviceTable/index.tsx`（实测 **685** 行）；物理链路 Collapse 块在 `:580-612`，其中空态静态文案在 `:604-609`。
- 使用位置：**工位管理页 `/operations/workstations`** 的**行展开区**（`src/pages/operations/workstations/index.tsx:681-686` 的 `expandable.expandedRowRender`）⇒ 物理链路面板在**展开行**里的第 3 个 Collapse。
- 兄弟面板：AD 域控设备、资产设备（同一组件内 3 个 Collapse，样式一致）。

#### 用户可观察的「done」

> **展开工位管理页任意工位行 →「物理链路设备（N台）」折叠面板：无设备时显示后端给出的具体可诊断原因（而不是通用空白）；有设备但工位未绑用户时，表格照常显示并**额外**给出「该工位未绑定用户」的告警；后端返回的 warnings 全部可见。**

#### 展示方案取舍（结论）

**项目既有惯例（实测）**：`Empty` 51 个文件在用；`Alert` 在 `src/pages/operations/workstations/index.tsx:872/893` 已在用（本页就有 2 处 Alert）；`Result` 在 ad-domain / asset-reconciliation 等页在用。**本页内已有 Alert 先例 ⇒ 优先用 Alert，不用 Result**（Result 语义过重，适合整页状态）。

**推荐三段式渲染**（`WorkstationDeviceTable/index.tsx:580-612` 的 Collapse children 替换，现状空态 `:604-609`）：

| 后端状态 | 前端呈现 | 组件 | 理由 |
|---|---|---|---|
| `devices.length > 0` **且** `emptyReason === ""` | 现有 `<Table>`（不变） | Table | 正常路径，零改动风险 |
| `devices.length > 0` **且** `emptyReason === "no_user_bound"` | **`<Table>` + Table 上方一条 `Alert type="warning"`**，文案 = `emptyMessage` | **Table + Alert** | ⭐ **这就是 3f130 本体**：链路是通的（能召回设备），只是没绑用户。**必须显示表格**（数据没丢）+ **必须显示告警**（否则用户看到 1 台设备却以为「已绑定也没用」）。**绝不能渲染成 Empty** —— 渲染成 Empty 会「把有数据说成没数据」，是倒退 |
| `devices.length === 0` **且** `emptyReason !== ""` | `<Empty description={emptyMessage} image={Empty.PRESENTED_IMAGE_SIMPLE} />`；若 `emptyMessage` 很长（后端文案 60-100 字），改用 **`<Alert type="info" showIcon description={...} />`** 或 `Empty` 的 `description` + `styles` 自定义 | **Empty 或 Alert** | 见下方取舍 |
| `devices.length === 0` **且** `emptyReason === ""` | 兜底：保留现有静态文案 + 「重新加载」按钮 | Alert info | 后端契约保证 0 台必有 emptyReason（测试 `workstation_physical_diagnosis_test.go:272`），此分支是防御性兜底 |
| `warnings.length > 0`（任何组合下） | Collapse 标题下方或 Table 上方，`<Alert type="warning" showIcon>` 逐条列出 | Alert | 后端注释明说 warnings「**有数据时也会有**」，且是「非致命但必须可见」 |

**「有数据但 emptyReason=no_user_bound」的取舍 —— 明确结论**：

> **渲染「表格 + 警告 Alert」，不渲染 Empty，不渲染纯告警替代表格。**
> 理由链：(a) 后端已把「未绑用户」定为**真实空因**（服务层注释 + 测试 :388 双证），前端不得自行降级为「正常」；(b) 但 `devices` 里确实有可展示的设备（链路通），隐藏它会丢失信息；(c) 因此这是**两个并列事实**：「有 1 台设备（链路通）」+「未绑用户（归属缺失）」。单一组件无法表达 ⇒ 必须 **Table + Alert 双区**。
> ⚠️ 备选（不推荐）：把 `no_user_bound` 时也降级成 Empty ⇒ 与后端契约正面冲突，且后端测试会挂。
> ⚠️ 备选（不推荐）：只显示 Alert 不显示 Table ⇒ 丢失已召回的设备信息。

**「Empty vs Alert」的取舍（0 台场景）**：
- 后端 `emptyMessage` 是 **60-100 字的中文指引**（含操作建议，如「请在信息点管理中关联对应的设备端口」）⇒ antd `Empty` 的 `description` 适合短句，长文本在 `Empty` 里排版拥挤且无图标语义。
- **推荐**：0 台时用 **`<Alert type="info" showIcon description={emptyMessage} />`**（信息型、容纳长文本、有引导语义）；`Empty` 保留给「确实没有概念」的纯空态。
- 若 UI 评审坚持要 Empty 视觉，则用 `<Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={<span style={{fontSize:12}}>{emptyMessage}</span>} />`。
- **本项目已在用 `Empty.PRESENTED_IMAGE_SIMPLE`**：`src/components/dashboard/widgets/base/BaseWidget.tsx:231` ⇒ 视觉基线一致。

**是否展示 `diagnosis` 逐段计数**：
- **默认不展示**。它是排查用中间量（`infoPointActive` / `portResolved` / `macMatched`…），对普通运维是噪音。
- **建议**：折叠进一个「诊断详情」Popover/Tooltip（用 `diagnosis` 渲染成「信息点 0/2 有效 · 端口 0/1 解析 · 设备 0/1 · MAC 命中 0」这样的链路漏斗），仅在 `emptyReason !== ""` 时出现。
- **理由**：`diagnosis` 的设计意图（`workstation_physical_diagnosis.go:82-84`）是「区分『数据缺失』与『绑定错位』」，而 `emptyMessage` 已经把这个结论翻译成了人话 ⇒ 计数只用于**开发/运维二线**。

**是否保留旧端点 `/physical`**：
- **保留**（后端明确「向后兼容…旧端点 /physical 行为未变」，`workstation_device_handler.go:146-149`）。
- 前端**新增** `workstationDeviceApi.getPhysicalWithDiagnosis`（或直接把 `getPhysical` 切到诊断端点并取 `data.devices`）。
  - ⚠️ 注意：`post<T>()` 的返回是 `response.data`（`AGENTS.md` 约定：`result.data.list`）。诊断端点的 `data` 是 `{devices,total,...}` 对象 ⇒ `post<PhysicalDevicesResult>(...)` 后取 `result.data.devices`。
  - ⭐ **兼容建议**：把 `getPhysical` 的实现改成调诊断端点、但**对外仍返回 `WorkstationDevice[]`**（内部 `return result.data.devices`），再另加一个 `getPhysicalDiagnosis` 返回完整对象 ⇒ **调用方零改动**，同时拿到诊断信息。风险最低。

#### 完成判据如何机械验证

| # | 判据 | 验证方式 | 通过标准 |
|---|---|---|---|
| F3-A | 端点接线 | `grep -rn "physical-diagnosis" src/` | ≥1 命中；`opsApi.ts` 有对应 wrapper |
| F3-B | **3f130 本体** | 工位管理页 → 展开 3f130 工位行 → 「物理链路设备」 | 面板显示后端 `emptyMessage`（含「未绑定用户」字样）；**不是**空白、**不是**通用静态文案 |
| F3-C | 有数据 + no_user_bound | 同一工位（链路通、能召回设备） | **表格照常渲染 + 上方出现 warning Alert**；不渲染 Empty |
| F3-D | 0 台 + info_point_misbound | 造一个无信息点的工位 | 显示针对性指引（含「信息点」字样） |
| F3-E | 0 台 + device_or_mac_missing | 造一个端口 orphan 的工位 | 显示针对性指引（含「端口」/「MAC」字样） |
| F3-F | warnings 可见 | 后端返回 warnings 的工位 | 每条 warning 都在 UI 上可见（逐条核对） |
| F3-G | 正常路径不回归 | 任意有设备且已绑用户的工位 | 面板与改动前**逐像素一致**（表格、列、置信度徽标 `confidence` Tag） |
| F3-H | 旧端点兼容 | `POST /ops/workstation-device/:id/physical` | 仍返回裸数组（后端未改，前端不得擅自改后端） |
| F3-I | 网络失败兜底 | 停后端 / 断网 | 面板显示「加载失败」提示，**不空白**（现状 `index.tsx:127-129` 只 `console.warn` ⇒ 建议一并补 `message.warning`） |
| F3-J | 现有单测 | `npm test`（含 `src/components/operations/WorkstationDeviceTable/__tests__/`） | 全绿 |

#### 依赖与前置

- **后端已就绪**，无阻塞。
- 前置：`opsApi.ts` 新增 wrapper + `src/types/operations.ts` 新增 `PhysicalDevicesResult` / `PhysicalChainDiagnosis` 类型（建议放 `src/types/operations.ts` 邻近 `WorkstationDevice` 定义处 `:341-354`）。
- 依赖：需要 3f130 工位在 PG 环境可访问（否则只能用造数工位验证）。

#### 风险点

| 风险 | 概率 | 影响 | 缓解 |
|---|---|---|---|
| **把 `no_user_bound` 当成「空」渲染成 Empty** | **高**（直觉误导） | **高**（把有数据说成没数据） | SPEC 必须写死「Table + Alert」双区；F3-C 判据专测此组合 |
| 改 `getPhysical` 返回形状 ⇒ 其他调用方炸 | 中 | 高 | 保持 `getPhysical` 对外返回数组（内部转调诊断端点）；`grep -rn "getPhysical" src/` 核对调用方 |
| 后端 `emptyMessage` 文案变更 ⇒ 前端硬编码失效 | 中 | 中 | 前端**只显示后端文案**，不硬编码任何原因文案（唯一例外是 0 台 + 无 emptyReason 的防御兜底） |
| 后端结构体注释与实现矛盾（`:63-65` vs `:813-827`）被当作真相 | 中 | 中 | 以实现 + 测试为准；顺手修正注释 |
| 长文案在 `Empty` 里排版拥挤 | 中 | 低 | 优先 Alert |
| 诊断端点比 `/physical` 慢（多一次诊断查询）⇒ 面板加载变慢 | 低 | 低 | 后端已实现（`GetPhysicalDevicesWithDiagnosis` 复用同一 CTE）；实测耗时对比后再定是否并行 |
| `warnings` 与 `emptyMessage` 语义重叠 ⇒ 同一条信息显示两遍 | 中 | 低 | UI 上 `emptyMessage` 走 Alert 主区，`warnings` 只显示 `no_user_bound` 之外的条目（或折叠为「诊断详情」） |

---

### 2.4 F4｜F5 刷新落 /dashboard

#### 根因（已定位到行，与 UAT 观测逐条对应）

**观测**（`75-UAT.md:76`）：
> 「F5 刷新后 SPA 落到 `/dashboard` 且 `ruoyi_last_visited_path` 被覆盖为 `/dashboard`，无法经刷新路径回归原页。」

**根因链**（`src/router/DynamicRoutes.tsx`）：

| 行号 | 代码 | 作用 |
|---|---|---|
| 85-92 | `getLastPath()` | 从 `sessionStorage["ruoyi_last_visited_path"]` 读上次路径 |
| 173 | `const cachedMenuData = useMemo(() => readMenuCache(), [])` | **同步**从 sessionStorage 读菜单+权限缓存（DATA-02 hydrate-then-revalidate） |
| 179-186 | `useEffect(() => { … useMenuStore.setState({menus, allMenus, permissions}) }, [cachedMenuData, isAuthenticated])` | **在 effect 里**水合 store ⇒ 首次渲染时 store 仍为空 |
| 227-240 | `routeElements = useMemo(… allMenus.length === 0 ? [] : …)` | **首次渲染 ⇒ `routeElements = []`（空路由树）** |
| 275-277 | `if (allMenus.length === 0 && !cachedMenuData) return <InitializingFallback />` | 有缓存时**不阻塞** ⇒ 直接渲染空的 `<Routes>` |
| **315** | `<Route path="*" element={<Navigate to="/dashboard" replace />} />` | **catch-all**：首帧没有任何动态路由匹配 `/operations/buildings` ⇒ 命中 `*` ⇒ 跳 `/dashboard` |
| 243-247 | `useEffect(() => { if (isAuthenticated && location.pathname) saveLastPath(location.pathname) }, [location.pathname, isAuthenticated])` | location 变成 `/dashboard` ⇒ **把 `ruoyi_last_visited_path` 覆写成 `/dashboard`** ⇒ 观测中的第二个现象 |

**时序（为什么 catch-all 抢先）**：
1. 首帧渲染：`allMenus = []`、`cachedMenuData ≠ null` ⇒ 进入 `<Routes>`，`routeElements = []`。
2. `<Navigate to="/dashboard">` 是 react-router 的组件，其内部用 **passive `useEffect`** 触发导航。
3. `DynamicRoutes` 的水合 `useEffect`（:179）**也在 passive 阶段**。React 的 passive effect 执行顺序是 **子先父后** ⇒ `<Navigate>`（子）的 effect **先于** `DynamicRoutes`（父）的水合 effect 执行 ⇒ **导航先发生**。
4. 导航把 location 改成 `/dashboard` ⇒ 重渲染时 `allMenus` 已水合 ⇒ 动态路由齐了，但 **URL 已经是 `/dashboard`**。
5. `:243-247` 的 effect 把 `/dashboard` 写进 `ruoyi_last_visited_path` ⇒ **污染了「上次路径」**，导致连「登录后回到上次页」也一并失效。

**支撑事实**：
- `src/store/menuStore.ts:52` `create<MenuStore>()` **无 zustand persist** ⇒ 刷新后 `allMenus` 必为空（首帧）。
- `src/App.tsx:54-58` 只有一个 `<BrowserRouter>` + `<DynamicRoutes />`；没有别的重定向源。
- `src/hooks/useTabSync.ts` / `src/components/layout/shared/useRouteTabs.ts` 里的 `/dashboard` 逻辑只管**标签页**，**不做导航** ⇒ 排除。
- 未登录分支（`:263-270`）把一切重定向到 `/login` ⇒ 与本问题无关。

#### 用户可观察的「done」

> **在任意已授权页面上按 F5，页面停留在原地址原内容（不跳仪表盘）；`ruoyi_last_visited_path` 仍等于刷新前的路径。**

#### 完成判据如何机械验证

| # | 判据 | 验证方式 | 通过标准 |
|---|---|---|---|
| F4-A | **主路径** | 停在 `/operations/buildings`（或任意二级菜单页）→ F5 | URL 仍是 `/operations/buildings`；页面渲染楼宇列表；**不出现仪表盘** |
| F4-B | **存储不被污染** | F5 后读 `sessionStorage["ruoyi_last_visited_path"]` | 值 === 刷新前的 pathname（**不是** `/dashboard`） |
| F4-C | **深层路径** | 在 `/operations/buildstations?deptId=xxx`（带 query）F5 | pathname + search 均保留 |
| F4-D | **无权限路径仍跳仪表盘**（**不回归**） | 用受限账号直接访问一个无权限路径 → F5 | 仍跳 `/dashboard`（这是**期望行为**，catch-all 的正当用途） |
| F4-E | **未登录跳转不回归** | 登出后 F5 | 仍跳 `/login`（`:263-270` 分支） |
| F4-F | **根路径行为不回归** | 访问 `/` | 跳 `lastPath`（若有效）或 `/dashboard`（`:285` 逻辑不变） |
| F4-G | **首次登录无缓存** | 清 sessionStorage 后登录 | 正常进入，菜单加载后落到仪表盘（`allMenus.length === 0 && !cachedMenuData` 分支） |
| F4-H | **会话恢复链路连带验证** | 在第 2 页 F5 | 页码仍为第 2 页（与 v1.23 FAIL-1 的修复叠合，`:76` 明确说 F5 路径曾让分页恢复无法回归） |
| F4-I | 现有单测 | `npm test`（含 `src/router/__tests__/dynamic-routes-hydrate.test.tsx`） | 全绿 |
| F4-J | 回归面扫描 | grep `navigate("/dashboard")` / `<Navigate to="/dashboard"` | 只有 `DynamicRoutes.tsx:285`（根路径兜底）与 `:315`（catch-all）两处，均被本次改动覆盖 |

#### 修法方向（3 个候选，推荐 B）

| 方案 | 做法 | 优点 | 风险 |
|---|---|---|---|
| A | **把水合从 `useEffect` 提到首帧同步**（`useState(() => readMenuCache())` + 渲染期 `useMenuStore.setState` 或 `useLayoutEffect`） | 语义最正 | ⚠️ **单独做不够**：`useLayoutEffect` 早于 passive effect，但 `<Navigate>` 已经在首帧被挂载 ⇒ 仍会导航。必须配合 B |
| **B（推荐）** | **让 catch-all 在「菜单尚未就绪」时不跳转**：`<Route path="*" element={menusReady ? <Navigate to="/dashboard" replace /> : <InitializingFallback />} />`（`menusReady = allMenus.length > 0`） | 直击根因；**「菜单未加载」与「用户无权限」被明确区分** | 需保证水合最终一定发生（已有 `fetchAll` + 30s TTL + 兜底）；无缓存分支已走 `InitializingFallback`（`:275`） |
| C | **恢复路径后再判权限**：catch-all 里先尝试 `lastPath`，失败再 `/dashboard` | 兼容旧行为 | ❌ 把「无权限」和「未加载」又混在一起，易回归 D4 判据 |

**推荐组合：B（必须）+ A（推荐）**。B 保证「菜单未就绪时不做任何跳转决策」，A 保证首帧就有路由可匹配（消除首帧白屏/闪烁）。

**配套小修（强烈建议）**：`:243-247` 的 `saveLastPath` 应**跳过「自动兜底跳转」产生的 pathname**（否则任何一次 catch-all 跳转都会污染「上次路径」，即使修了 B 也会在「无权限路径」场景留下脏值）。可用一个 `redirectingRef` 或对 `/dashboard` 做「仅当用户主动点击仪表盘 tab 时才记录」的判定。

#### 依赖与前置

- 无外部依赖。
- ⚠️ **与 v1.23 FAIL-1（分页持久化恢复）强耦合**：`75-UAT.md:76` 明确「F5 刷新后 SPA 落到 /dashboard 且 `ruoyi_last_visited_path` 被覆盖 ⇒ 无法经刷新路径回归原页」⇒ **F4 是分页恢复链路的最后一环**，两者必须一起验收（F4-H）。
- ⚠️ 与 F1（双框）无耦合，可并行。

#### 风险点

| 风险 | 概率 | 影响 | 缓解 |
|---|---|---|---|
| 用 `useLayoutEffect` 水合 + 未加 catch-all 门 ⇒ 无效 | **高** | 中 | 必须同时做 B；判据 F4-A 覆盖 |
| catch-all 门写错 ⇒ 无权限用户**白屏卡死**（永远不跳仪表盘） | 中 | 高 | `menusReady` 用 `allMenus.length > 0`（有明确终态：菜单加载完成即 >0）；保留 3s 初始化超时兜底（`:190-204`） |
| 破坏「登录后回到上次页」（`:285`） | 中 | 中 | F4-F 判据专测 |
| 破坏未登录跳登录（`:263-270`） | 低 | 高 | F4-E 判据；改动限定在 `:315` 那一行 |
| `saveLastPath` 污染「上次路径」的副作用未清 | **高**（不改必然还在） | 中 | 配套小修 + F4-B 判据 |

---

### 2.5 F5｜ColumnConfigModal:136 豁免注释

#### 契约原文（68.1 UI-SPEC §1.1 判定表）

`.planning/milestones/v1.22-phases/68.1-closeout-debt-clearance/68.1-UI-SPEC.md:44`：

> | 非删除非危险确认（普通「是否」询问） | 6 | **明确排除**，保留原调用，但**必须加 `// eslint-disable-next-line no-restricted-syntax` 理由注释 + 登记进 §4 豁免理由表**（清单见 FACTS A.1④：`68.1-FACTS.md:119-124`——`src/components/dashboard/layout/LayoutToolbar.tsx:109`、`:142`、`src/components/operations/WorkstationDeviceTable/index.tsx:236`、`src/components/shared/ColumnConfigModal.tsx:136`、`src/pages/ad-domain/users/index.tsx:300`、`src/pages/operations/rpa/tasks/index.tsx:110`） |

**实测 6 处现状**：

| # | 位置 | 有无理由注释 |
|---|---|---|
| 1 | `src/components/dashboard/layout/LayoutToolbar.tsx:109` | ✅ `// eslint-disable-next-line no-restricted-syntax -- 非删除语义：重置仪表盘为可逆的展示态复位，UI-SPEC §1.1 判定规则排除` |
| 2 | `src/components/dashboard/layout/LayoutToolbar.tsx:143` | ✅ 同款 |
| 3 | `src/components/operations/WorkstationDeviceTable/index.tsx:234` | ✅ `// eslint-disable-next-line no-restricted-syntax -- 非删除语义：设置主设备是正向指派（可再次更改），UI-SPEC §1.1 判定规则排除` |
| 4 | **`src/components/shared/ColumnConfigModal.tsx:136`** | ❌ **无任何注释**（`:134-136` 只有 `// 处理重置` + 函数头 + `Modal.confirm({`） |
| 5 | `src/pages/ad-domain/users/index.tsx:300` | ✅ `// eslint-disable-next-line no-restricted-syntax -- 非删除语义：同步用户到系统用户表为幂等补写，不删除任何数据，UI-SPEC §1.1 判定规则排除` |
| 6 | `src/pages/operations/rpa/tasks/index.tsx:124` | ✅ `// eslint-disable-next-line no-restricted-syntax -- 非删除语义：执行 RPA 任务为发起动作，UI-SPEC §1.1 判定规则排除` |

⇒ **5/6 已完成，唯独第 4 处缺**。这正是 `75-UAT.md:47` §8 #12 备注的「1 处轻微偏离」。

**注意**：这一处 **eslint 不会报错** —— `src/components/shared/**` 在封装层豁免 override（`eslint.config.js:265-287`）内，`no-restricted-syntax` 被重声明为只剩「内网 IP」selector，**Modal.confirm selector 在该目录不生效**。所以这**不是 lint 债，是契约登记债**（`eslint-disable` 注释在此处是**显式登记**，不是「压错误」）。

#### 用户可观察的「done」

> **打开「列配置」弹窗 → 点「重置」→ 确认弹窗文案与行为不变（`确认重置` / `确定要重置列配置吗？此操作将清除您的个人设置。`）；同时源码该处带有与 §1.1 判定表同款的行内理由注释。**

#### 完成判据如何机械验证

| # | 判据 | 验证方式 | 通过标准 |
|---|---|---|---|
| F5-A | 注释存在 | `sed -n '130,140p' src/components/shared/ColumnConfigModal.tsx` | `Modal.confirm(` 上一行含 `// eslint-disable-next-line no-restricted-syntax -- <理由>` |
| F5-B | **理由格式一致** | 与其余 5 处 diff | 文案结构同款：`-- 非删除语义：<为什么不是删除>，UI-SPEC §1.1 判定规则排除` |
| F5-C | **6/6 全覆盖** | 逐处 grep §1.1 清单 | 6 处全部带注释 |
| F5-D | 理由表登记 | `scripts/exemption-reasons.json` | 该条目已登记（⚠️ 见下方「连带债」） |
| F5-E | 行为不变 | 浏览器：列配置 → 改列 → 重置 → 确认 | 列配置被重置；弹窗文案逐字不变 |
| F5-F | lint 不回归 | `npx eslint .` | errors ≤ 17、warnings ≤ 34（**不得因新增注释而产生 unused-disable**） |

⚠️ **F5-F 的坑**：`ColumnConfigModal.tsx` 在 `src/components/shared/**` 豁免块内 ⇒ `no-restricted-syntax` 的 Modal.confirm selector **不在该文件生效** ⇒ 新加的 `eslint-disable-next-line no-restricted-syntax` 会被 ESLint 报为 **「Unused eslint-disable directive」**（正是现在 `src/components/shared/PageTable.tsx:180` 那 1 条 warning 的同类）⇒ **warnings 反而 +1**。
⇒ **必须同时**：把 `src/components/shared/**` 豁免块改回「保留 Modal.confirm selector 但对已登记文件用 `allowExportNames`/白名单豁免」的方式，**或**在 `eslint.config.js` 给该文件加 `linterOptions.reportUnusedDisableDirectives: false`，**或**改用**非 eslint 指令**的行内注释（但那样就违反 §1.1「必须加 eslint-disable-next-line」的原文）。
👉 **推荐**：**修 eslint 配置**，让 `src/components/shared/ColumnConfigModal.tsx` 的豁免是「显式登记」而非「目录级全禁」。这同时能修掉 §8-A 暴露的「豁免理由表失同步」。

#### 依赖与前置

- 无外部依赖，**5 分钟量级**。
- ⚠️ 与 §8-A（豁免理由表失同步）**强耦合**，建议**合并成一个 phase** 一次做完。

#### 风险点

| 风险 | 概率 | 影响 | 缓解 |
|---|---|---|---|
| 只加注释 ⇒ warnings +1（unused-disable） | **高** | 中 | F5-F 判据 + 上文「必须同时修 eslint 配置」 |
| 改 `src/components/shared/**` 豁免块 ⇒ 波及该目录所有文件（`AppInput`/`AppSelect`/`confirmDelete`…） | 中 | 高 | 用 `files: ["src/components/shared/**"]` + `excludedFiles` 精确豁免，而不是整目录放开；改完必跑 `npx eslint .` 对比 errors/warnings |
| 「重置列配置」被后续误判为删除类 ⇒ 收口到 ConfirmDelete | 低 | 低 | §1.1 已明确它属「非删除非危险」，只需登记 |

---

### 2.6 F6｜lint 存量清零（17 errors + 33 react-refresh）

#### 基线（实跑 `npx eslint .`，与 `scripts/assert-lint-budget.mjs` 同口径）

**errors = 17**

| 规则 | 数量 | 位置 | 清零难度 | 备注 |
|---|---|---|---|---|
| `react-hooks/exhaustive-deps` | **12** | 见下表 | **MEDIUM** | 用户口径的「12 处」 |
| `react-hooks/set-state-in-effect` | 1 | `src/components/dashboard/DashboardView.tsx:161` | MEDIUM | React Compiler 规则：effect 内同步 setState ⇒ 级联渲染 |
| `react-hooks/preserve-manual-memoization` | 1 | `src/hooks/useRPAProgress.ts:78` | MEDIUM | Compiler 跳过优化：手动 memo 依赖不稳定 |
| `react-hooks/immutability` | 1 | `src/hooks/useRealtimeUpdates.ts:119` | MEDIUM | 「变量在声明前被访问」 |
| `react-hooks/globals` | 1 | `src/pages/ad-domain/configs/__tests__/ReconciliationTab.test.tsx:51` | **LOW** | 测试文件里重声明外部变量 |
| `local/no-large-dropdown-list` | 1 | `src/components/ui/AppSelect.tsx:16` | **见 §8-C（契约冲突，不是机械修）** | 自有规则要求 Select 必须有 `onSearch` |

**exhaustive-deps 12 处明细**

| # | 文件:行 | 症状 |
|---|---|---|
| 1 | `src/components/dashboard/DashboardView.tsx:151` | 缺 `message` |
| 2 | `src/design-system/components/ColorSwitcher.tsx:109` | 缺 `preferences.theme.customColors.primary` / `.sidebar` |
| 3 | `src/pages/ad-domain/configs/ReconciliationTab.tsx:115` | **不必要依赖** `queryTick` |
| 4 | `src/pages/ad/SyncMonitor/index.tsx:91` | 缺 `fetchLogs` / `fetchStatus` |
| 5 | `src/pages/system/captcha-background/hooks/useCaptchaData.ts:47` | 缺 `setTotal` |
| 6 | `src/pages/system/captcha-background/hooks/useCaptchaModals.ts:143` | 缺 `message` |
| 7 | `src/pages/system/captcha-background/hooks/useCaptchaModals.ts:172` | 缺 `message` |
| 8 | `src/pages/system/captcha-background/hooks/useCaptchaModals.ts:183` | 缺 `message` |
| 9 | `src/pages/system/captcha-background/hooks/useCaptchaModals.ts:193` | 缺 `message` |
| 10 | `src/pages/system/captcha-background/hooks/useCaptchaModals.ts:202` | 缺 `message` |
| 11 | `src/pages/system/captcha-background/index.tsx:64` | 缺 `loadBackgrounds` / `loadStatistics` / `paginationProps` |
| 12 | `src/pages/system/settings/default-theme.tsx:70` | 缺 `message` |

> **可观测规律**：8/12 是「缺 `message`」——`message` 来自 `App.useApp()`。**antd 的 `App.useApp()` 返回值在 antd 6 中是稳定引用**，但**类型上是 `MessageInstance` 对象字面量**；React Compiler 规则与 exhaustive-deps 都要求把它列进依赖数组。**最省事的正确修法**：把 `const { message } = App.useApp()` 换成模块级或 `useMemo` 稳定的引用，或直接加 `// eslint-disable-next-line react-hooks/exhaustive-deps -- App.useApp() 返回值稳定` （**注意**：仓库已有此类先例，`src/components/operations/WorkstationDeviceTable/index.tsx:141` 就是 `// eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable`）。
> ⇒ **⚠️ 但清零 errors ≠ 加 disable 注释**：`eslint-disable` 注释本身会产生 **unused-disable warning**（若该行本就不报），或掩盖真实问题。**建议**：对「message 稳定」这类**已论证**的情况用带理由的 disable（与既有先例同款）；对其余 4 处（ColorSwitcher 的深层依赖、SyncMonitor 的 fetch 函数、captcha 的 `loadBackgrounds`/`paginationProps`、default-theme）**真修**（用 `useCallback` 稳定函数 + 正确依赖）。

**warnings = 34** = react-refresh 33 + 1 条 unused-disable（`src/components/shared/PageTable.tsx:180`）

#### react-refresh 33 处的性质分类（★ 本项的核心产出）

**规则机制**（`node_modules/eslint-plugin-react-refresh/index.js`）：
- 配置：`eslint.config.js:59` `"react-refresh/only-export-components": ["warn", { allowConstantExport: true }]`。
- `allowConstantExport` 只放行初始化式为 `Literal | UnaryExpression | TemplateLiteral | BinaryExpression` 的 `export const`（`index.js:247-255`）⇒ **`export const X = [...]`（ArrayExpression）、`= {...}`（ObjectExpression）、`= 已导入的标识符` 都不被放行**。
- 报告条件（`index.js:220-231`）：**只有当文件里 `hasReactExport === true` 时**才报 `namedExport`。
- `hasReactExport` 的两个来源：① 导出名匹配 `/^[A-Z][a-zA-Z0-9_]*$/` 的**函数**（`:99`）；② 导出名匹配大写驼峰且初始化式**不是** `notReactComponentExpression` 集合里的类型（`index.js:119-125`）—— **`Identifier` 不在该集合里**（`:257-269`）⇒ `export const STATUS_OPTIONS = SOME_IMPORTED_CONST`（Identifier）**会被判定为「组件导出」**。

**分类结果（17 个文件）**：

| 类别 | 数量 | 文件:行 | 性质 | 处置 |
|---|---|---|---|---|
| **P1 组件 + 非组件工具同文件**（真问题） | **18** | `shared/ActionButtons.tsx:114`(`createDeleteAction`)、`:167`(`createCustomDeleteAction`)、`shared/FormModal.tsx:61`(`useFormModalForm`)、`shared/ModernTag.tsx:75`(`renderStatusTag`)、`shared/statusHelpers.tsx:63`(`statusLabel`)、`:68`(`statusColor`)、`network/port-write/PortWriteModal.tsx:78`(`showAuditLinkToast`)、`design-system/ColorSwitcher.tsx:36`(`primaryColorPresets`)、`:57`(`sidebarColorPresets`)、`design-system/LayoutProvider.tsx:20`(`useLayoutContext`)、`design-system/ThemeProvider.tsx:23`(`useThemeContext`)、`ad-domain/configs/ReconciliationTab.tsx:42`(`MISMATCH_TYPE_META`)、`:66`(`rowKeyOf`)、`settings/index.tsx:226`(`userSettingsCategories`)、`system/settings/index.tsx:30`(`systemSettingsCategories`)、`router/DynamicRoutes.tsx:85`(`getLastPath`)、`:95`(`saveLastPath`)、`:108`(`clearLastPath`) | **真·Fast Refresh 边界**：HMR 时这些导出被替换，组件状态/引用可能不刷新 | **必须拆**（工具/常量搬到 `utils/` `constants/` `hooks/`），**或**逐处 `eslint-disable` + 理由登记 |
| **P2 误判**（文件里根本没有组件导出） | **13** | `pages/operations/workstations/constants.tsx:17,31,36,41,46,51,56`（7）、`pages/system/dept/constants.tsx:13,19,24`（3）、`pages/system/dict/constants.tsx:18,22,29`（3） | **规则误判**：这三个文件**零组件导出**；触发点是 `export const STATUS_OPTIONS = WORKSTATION_STATUS_OPTIONS`（Identifier 初始化 + 大写名 ⇒ 被判为组件导出）⇒ 同文件其余导出全被连坐 | **改代码而非豁免**（✅ 已实测，见下方「P2 修法的实测结论」）：把 Identifier 初始化改成**字面量拷贝**（`[...X]` / `{...X}`）或把导出名改小写 ⇒ **13 处一次性消零，零 import 破坏** |
| **P3 `lazy()` 包装器** | **2** | `building-spaces-3d/components/BuildingModel3D.tsx:316`、`FloorPlan3D.tsx:693`（`export const XXLazy = lazy(() => import(...))`） | 规则不认 `lazy` 为 HOC（`reactHOCs = ["memo","forwardRef"]`，`index.js:70`）⇒ 判为非组件导出 | **合理例外**：① 加 `// eslint-disable-next-line react-refresh/only-export-components -- lazy() 包装组件，Fast Refresh 对 lazy 边界不适用` + 理由登记；**或** ② 把 lazy 包装器挪到独立的 `lazyComponents.ts` 文件（**推荐**，与 P1 同款拆法，零豁免） |
| **合计** | **33** | 17 个文件 | | |

> ⭐ **结论**：**只有 18 处是「必须拆」的真问题**；**13 处是 3 个文件里的规则误判，改 5 行即可消零**；**2 处是 `lazy()` 的规则盲区**（推荐用拆分而非豁免）。⇒ **「清零 33」的真实工作量 ≈ 18 处拆分 + 3 个文件的 5 行改写 + 2 处挪文件**，远比「拆 33 处」轻。

#### P2 修法的实测结论（✅ 用仓库同版本插件 + 同配置在仓库外沙箱复现）

在同一份 `eslint-plugin-react-refresh` + 同一 `{ allowConstantExport: true }` 配置下，用 4 个最小样例实测：

| 变体 | 写法 | 警告数 | 结论 |
|---|---|---|---|
| **a 现状** | `export const STATUS_OPTIONS = WORKSTATION_STATUS_OPTIONS;` | **3** | 复现真实文件的连坐现象 ✅ |
| **b 字面量拷贝** | `export const STATUS_OPTIONS = [...WORKSTATION_STATUS_OPTIONS];` | **0** | ✅ **可用**（init 变成 ArrayExpression ⇒ 落入 `notReactComponentExpression` ⇒ 不再置 `hasReactExport`） |
| **c 重导出** | `export { WORKSTATION_STATUS_OPTIONS as STATUS_OPTIONS };` | **3** | ❌ **无效**（导出名仍是大写 ⇒ 仍置 `hasReactExport`）—— ⚠️ **网上常见的「改成 re-export」建议在这里是错的** |
| **d 改小写名** | `export const statusOptions = WORKSTATION_STATUS_OPTIONS;` | **0** | ✅ 可用，但要改 89 处 import 的名字 ⇒ 成本高 |

**⇒ P2 的推荐修法（三选一，按优先级）**：
1. **b 字面量拷贝**（改动最小、零 import 破坏）：把 5 处 Identifier 初始化改成字面量 —— `workstations/constants.tsx:13`、`dept/constants.tsx:10`、`dict/constants.tsx:11` 与 `:14`。
   ⚠️ 前置检查：这 5 个共享常量（`WORKSTATION_STATUS_OPTIONS` / `NORMAL_STOP_OPTIONS` / `ENABLE_DISABLE_OPTIONS` / `ENABLE_DISABLE_TAG_CONFIG`，均定义于 `src/constants/status.ts:28/34/42/57`）**必须确认不存在就地 mutate**（`push`/`splice`/下标赋值）—— 全仓 grep 未发现 mutate 点，但拷贝会失去引用相等性，若有 `===` 比较需改。
2. **拆文件**（更彻底、零拷贝语义风险）：把三个 `constants.tsx` 按「纯数据 / JSX 渲染」拆开 —— 纯数据部分改名为 `.ts`（**规则只扫描 `.jsx`/`.tsx`**，`eslint-plugin-react-refresh/index.js:67` ⇒ `.ts` 文件天然免疫），JSX 渲染函数另置 `renderTags.tsx`（只导出小写函数 ⇒ 零警告）。
   ⚠️ 拆分后**不得**让 `constants.ts` 与 `constants.tsx` 同名共存（import 路径会歧义）；建议命名为 `constants.ts` + `renderTags.tsx`。
3. **d 改小写名** —— 成本最高，仅在前两项都受阻时用。

> ⚠️ **实测另外确认的 4 条边界**（同样在沙箱复现）：
> - `.ts` 文件**不扫描** ⇒ 迁到 `.ts` 即零警告（`e-dotts.ts` = 0）。
> - `export const X = lazy(() => import(...))` **被报**（`f-lazy.tsx` = 1）⇒ P3 归类成立。
> - 组件 + 小写函数 ⇒ 报 1（`g-comp-func.tsx` = 1）⇒ P1 归类成立。
> - 组件 + `export const X = {…}` / `= [{…}]` ⇒ 报 1（`h`/`j` = 1）；但组件 + `export const X = "lit"` / `= 1+2` ⇒ **不报**（`i` = 0，`allowConstantExport` 对 Literal/Unary/Binary 生效）⇒ P1 里 `ColorSwitcher.tsx:36/57`（数组字面量）属真问题，而 `ReconciliationTab.tsx:39`（`RECONCILIATION_PERM`，Literal）与 `PortWriteModal.tsx:50`（`AUDIT_LOG_PATH`，Binary）本就合规。

#### 「必须拆 / 可豁免 / 合理例外」三类分流（★ 用户要求的分类思路）

| 档 | 判据 | 名单 | 推荐做法 |
|---|---|---|---|
| **① 必须拆** | 文件**导出组件**且**导出非组件工具/常量**；这些导出**被其它文件 import** ⇒ HMR 时确实会失效 | P1 的 18 处 | 拆出：<br>• `ActionButtons.tsx` → `ActionButtons.tsx`（组件）+ `actionCreators.ts`（`createDeleteAction` / `createCustomDeleteAction`）<br>• `FormModal.tsx` → `useFormModalForm` → `hooks/useFormModalForm.ts`<br>• `ModernTag.tsx` → `renderStatusTag` → `renderStatusTag.tsx`（或并入已有的 `shared/statusHelpers.tsx`）<br>• `statusHelpers.tsx` → `statusLabel`/`statusColor` → `statusHelpers.ts`（`.tsx`→`.ts`，无 JSX 后不再是 tsx ⇒ 规则不扫）<br>• `PortWriteModal.tsx` → `showAuditLinkToast` → `port-write/utils.ts`（⚠️ v1.19 port-write 冻结面，见风险）<br>• `ColorSwitcher.tsx` → 两个 preset 数组 → `colorPresets.ts`<br>• `LayoutProvider.tsx` / `ThemeProvider.tsx` → `useLayoutContext` / `useThemeContext` → `contexts.ts`（⚠️ `createContext` 导出本身被规则单独处理，需 `allowExportNames` 或独立文件）<br>• `ReconciliationTab.tsx` → `MISMATCH_TYPE_META` → `constants.ts`；`rowKeyOf` → `utils.ts`<br>• `settings/index.tsx` / `system/settings/index.tsx` → 两个 categories 数组 → `constants.ts`<br>• `DynamicRoutes.tsx` → `getLastPath`/`saveLastPath`/`clearLastPath` → `lastPath.ts` |
| **② 可用 eslint 豁免登记** | 导出**不被其它文件 import**（纯本地）**或** 是**类型/常量别名**，拆分收益低于 import 链改动风险 | 优先候选：`ReconciliationTab.tsx:66`(`rowKeyOf` 仅本页用)、`ModernTag.tsx:75`(`renderStatusTag` 若仅本页用)、`PortWriteModal.tsx:78`（v1.19 冻结面，见下） | `// eslint-disable-next-line react-refresh/only-export-components -- <理由>` + **登记进 `scripts/exemption-reasons.json`**（与 §8-A 一并修） |
| **③ 合理例外（不该改）** | ① `lazy()` 包装器（规则盲区，Fast Refresh 对 lazy 边界本就不适用）；② `index.ts` barrel 文件（`export * from` 会触发规则的 `exportAll` 消息，但本仓 barrel 需自带规则豁免）；③ Context 文件（`createContext` 导出被规则专门识别，规则的 `reactContext` 消息即为此设计） | P3 的 2 处 + 未来 barrel | 用**文件级** `/* eslint-disable react-refresh/only-export-components -- <理由> */` + 登记，**不要**逐行 disable |

> ⭐ **强烈建议**：**P2 的 13 处绝不要用 eslint 豁免**。它们是**规则误判**，用豁免会把「规则没覆盖到真实问题」永久固化。正确解法是 `export { X as Y }` 重导出（`:13` `export const STATUS_OPTIONS = WORKSTATION_STATUS_OPTIONS;` ⇒ `export { WORKSTATION_STATUS_OPTIONS as STATUS_OPTIONS };`）——**改动 1 行、零 import 破坏、13 处 warning 一次性消零**。

#### 完成判据如何机械验证

| # | 判据 | 验证方式 | 通过标准 |
|---|---|---|---|
| F6-A | **errors 归零** | `npx eslint .` | `errorCount === 0`（当前 17） |
| F6-B | **warnings 归零** | 同上 | `warningCount === 0`（当前 34 = 33 + 1） |
| F6-C | **预算同步归零** | `package.json` 的 `lint`/`lint:ci` + `scripts/assert-lint-budget.mjs` | `--max-warnings 0`；`BUDGET.maxErrors = 0`、`maxWarnings = 0`（当前 17/34）——**不改预算 = 门禁形同虚设** |
| F6-D | **门禁自证** | `npm run lint:ci` | exit 0；且**故意注入 1 条 error 后必须 exit 1**（证明门禁真的生效） |
| F6-E | **单测不回归** | `npm test` | 全绿（拆文件可能打断 mock 路径 ⇒ 重点看 `src/components/shared/__tests__/`、`src/router/__tests__/`、`src/design-system/components/__tests__/`） |
| F6-F | **类型不回归** | `npm run type-check` / `npm run build` | exit 0（拆分 + `export { X as Y }` 改写最易在此暴露问题） |
| F6-G | **禁用注释无残留** | `npx eslint . --report-unused-disable-directives` | 无 unused-disable（当前 `PageTable.tsx:180` 那 1 条要么修要么删） |
| F6-H | **豁免理由表同步** | `node scripts/check-exemption-reasons.mjs` | **exit 0**（当前 **exit 1 / 123 项**，见 §8-A） |
| F6-I | **豁免登记 100% 带理由** | 同上 | 每条 `reason` 非空、`userVisible=true` 的 `difference` 非空 |

#### 依赖与前置

- 无外部依赖。
- ⚠️ **F6 与 F5 强耦合**：§1.1 的 6 处豁免要登记进理由表 ⇒ 理由表必须先修好（§8-A）。
- ⚠️ **F6 与 F4 耦合**：`DynamicRoutes.tsx` 的 3 个函数拆出去后，F4 的修复也在同一文件 ⇒ **F4 + F6 的 DynamicRoutes 部分合并为一个 commit**。
- ⚠️ **F6 与 F2 耦合**：`ActionButtons.tsx` 的 `createDeleteAction` 正是 F2 推荐的 `perm` 参数落点 ⇒ **F6 拆 `ActionButtons` 与 F2 改 `ActionButtons` 必须同一波**，否则两次改同一文件。

#### 风险点

| 风险 | 概率 | 影响 | 缓解 |
|---|---|---|---|
| **为清零而批量加 `eslint-disable`** ⇒ 掩盖真问题 + 制造 unused-disable warning（净 warnings 不降） | **高** | 中 | 分流表已明确「P2 必须改代码」；判据 F6-B 要求净归零 |
| `export { X as Y }` 改写影响 89 处 import | 低 | 中 | `export { X as Y }` 与 `export const Y = X` **对 import 方完全等价**；`npm run type-check` 兜底 |
| 拆 `PortWriteModal.tsx` ⇒ 触碰 **v1.19 port-write 冻结面** | **中** | **高** | `src/components/network/port-write/*` 是 v1.19 交付冻结区（`MILESTONES.md:121`「62 测试全绿」）；⇒ **建议 F6 对 port-write 只做豁免登记，不拆文件**（归入「② 可豁免」档） |
| 拆 `LayoutProvider`/`ThemeProvider` 的 context 导出 ⇒ `createContext` 触发规则的 `reactContext` 消息 | 中 | 低 | 把 Context 本体与 Provider 分离到 `context.ts`（规则对 `createContext` 有专门分支 `index.js:105-108`） |
| exhaustive-deps 真修 ⇒ 引入无限请求循环（`CLAUDE.md` 明确警告过这个坑） | 中 | 高 | 每次修改后跑 `npm test` + 浏览器目视该页；「缺 `message`」类统一走既有 disable 先例 |
| `local/no-large-dropdown-list` 的 AppSelect 冲突无解 ⇒ errors 降不到 0 | **中** | 中 | 见 §8-C：这是**契约冲突**，必须先做决策，不能靠 disable 糊过去（否则 F6-A 不可达成） |
| 清零后 CI 变慢/变脆 | 低 | 低 | 预算门禁只判数字，不判内容 |

---

### 2.7 F7｜devices 页 714 行拆分

#### 历史（三次延期的事实链）

| 时点 | 事实 | 出处 |
|---|---|---|
| v1.22 Phase 67 T3 | devices `index.tsx` **1282 → 692** 行，另拆出 `DeviceModals`(425) + `DeviceStatsCards`(71) + `DeptTreePanel`(24) | `67-SUMMARY.md:12,24,69` |
| 68.1 | 实测 692 行，PLAN 预估 350-420 ⇒ 登记为「规模事实，非缺陷」 | `68.1-FACTS.md:322`、`68.1-VERIFICATION.md:27` |
| 68.1 PLAN | **明确不拆**：「拆分风险高于收益」；若 Lead 要求需追加 ~2h | `68.1-PLAN.md:508, 836, 852` |
| v1.23 seeds | 结转：「若要拆需追加 ~2h」 | `v1.23-seeds.md` §B |
| **实测 2026-10-28** | **714 行**（较 692 又 +22） | `wc -l src/pages/network/devices/index.tsx` |

#### 当前文件结构（实测）

```
src/pages/network/devices/
├── index.tsx            714 行  ← 目标文件
├── constants.ts         （VENDOR_OPTIONS / DEVICE_TYPE_OPTIONS / STATUS_OPTIONS / *_TAG_COLOR）
├── utils.tsx            （formatDateTime / getOptionLabel / getStatusColor）
├── components/
│   ├── DeptTreePanel.tsx   24 行
│   ├── DeviceModals.tsx   404 行
│   └── DeviceStatsCards.tsx 71 行
├── hooks/
│   ├── useDeviceData.ts   110 行
│   └── useDeviceModals.ts  73 行
└── __tests__/（4 个）
```

**`index.tsx` 内部块分布（实测行号）**：

| 块 | 行范围 | 行数 | 依赖 | 拆分难度 |
|---|---|---|---|---|
| ① 文件头 + import | 1-67 | 67 | — | — |
| ② `getDeviceTableColumns()` 纯函数 | 69-171 | ~103 | 8 个 props（`navigate`/`handleCollectPorts`/`openDetailModal`/`openModal`/`collectingDeviceId`/`canViewMACHistory`/`getDeviceTypeText`） | **LOW**（纯函数，无副作用、无 hook） |
| ③ 主组件状态与 hooks | 173-283 | ~111 | `useMenuStore`/`useDict`/`usePersistedStateController`/`useDeviceData`/`useDeviceModals`/`useMemo`×3 | **LOW**（现状已是 hook 化） |
| ④ 数据操作 handlers | 285-420 | ~136 | 2 个 Form 实例 + 6 个 setState + `post`/`handleApiError`/`handleSuccess` | **MEDIUM** |
| ⑤ 行操作组装 `buildRowActions` | 451-514 | ~64 | `collectingDeviceId`/`canViewMACHistory`/`navigate`/handlers | MEDIUM |
| ⑥ 搜索字段 `searchFields` | 539-563 | ~25 | `deviceTypeOptions`/`VENDOR_OPTIONS`/`STATUS_OPTIONS` | **LOW** |
| ⑦ 页面 JSX（Layout + PageTable + toolbar + Alert + 3 个 Modal） | 565-711 | ~147 | 全部上述 | **MEDIUM**（JSX props 极多，见下） |

#### 拆分边界建议（★ 用户要求的结论）

**推荐 4 刀（按「低风险先行」排序，每刀独立可验收）**：

| 刀 | 抽出 | 目标文件 | 理由 | 风险 |
|---|---|---|---|---|
| **D1** | `getDeviceTableColumns()` | `components/DeviceTableColumns.tsx` | **纯函数**，无 hook、无副作用 ⇒ 拆出零行为风险；103 行直接减半 | **LOW**。⚠️ 它返回 `ColumnsType<NetworkDevice>`，需要 `ColumnsType` 类型 import；⚠️ 它目前接收 8 个 props，拆出后仍要传（可用一个 props 对象，与现状一致） |
| **D2** | `searchFields` 定义 | `constants.ts`（或 `searchFields.ts`） | 纯数据；且 `searchFields` 里 `deviceTypeOptions` 是动态的 ⇒ 抽成**函数** `buildDeviceSearchFields(deviceTypeOptions)` 更合适 | **LOW** |
| **D3** | ④⑤ handlers + `buildRowActions` | `hooks/useDeviceActions.ts` | 把 `handleProbe`/`handleQuickCreate`/`handleCreate`/`handleDelete`/`handleBatchExport`/`handleCollectPorts`/`openModal`/`openQuickCreateModal`/`buildRowActions` 收进一个 hook，**params 收窄为** `{ quickCreateForm, editForm, loadDataRef, message, … }` | **MEDIUM**。⚠️ **`reloadRef` 是 PageTable children 渲染期回填的 ref**（`index.tsx:516` 声明、`:627` 回填 `ctx.loadData`）⇒ 搬进 hook 后要保持「ref 由外部注入」的语义不变，不能改成 state（会引入渲染时序问题） |
| **D4** | ⑦ JSX 外壳 | `components/DeviceTablePanel.tsx` + `components/DevicePageShell.tsx` | 把 `PageTable` + toolbar（NetworkExport / 快速创建 / 手动新增）+ 已选择 Alert 拆出 | **MEDIUM～HIGH**。⚠️ PageTable 的 `children` 渲染函数（`:620-648`）依赖 `ctx.loadData` 回填 `reloadRef` ⇒ 拆分时要保持这条「ref 回填」链路 |

**推荐落地顺序**：**D1 → D2 →（可选）D3 → D4**。D1+D2 约 125 行，直接把 714 → ~590；D3+D4 再降到 ~380-420（与 68.1 PLAN 预估的 350-420 对齐）。

**不建议拆的**：
- `DeviceModals.tsx`（404 行）—— 已是纯弹窗集合，props 密集，拆分收益低、风险高；**保持现状**（它是 v1.22 刚拆出来的，再拆会打破 68.1 的验收基线）。
- `useDeviceData.ts` / `useDeviceModals.ts` —— 已是 hook 形态。

#### 与 React.memo / 性能的关系（结论）

- **`React.memo` 与本次拆分无关，也不是解决手段**。devices 页的渲染成本主要在 `PageTable` 的 `virtual` + `scroll={{x:1600,y:600}}`（`:586-587`）虚拟滚动，**行组件**才是渲染热点，而 `DeviceTableColumns` 只是**列定义**（不产生行 DOM）。
- ⚠️ `columns` 已经是 `useMemo`（`:524-536`），但其依赖数组是 `[collectingDeviceId, canViewMACHistory, deviceTypeDict]`（**故意不完整**，`:535` 有 `eslint-disable-next-line react-hooks/exhaustive-deps`）⇒ 每次 `collectingDeviceId` 变化都会重建 columns ⇒ **列定义重建会触发 Table 的列重新渲染**。这才是真正的性能点。
  - ⭐ **发现（值得写进 SPEC）**：`collectingDeviceId` 在「采集端口」进行中会变化 ⇒ 每次变化都重建 columns（8 个 sorter + 若干 cell renderer）⇒ 在 100+ 行的虚拟表上会造成可感知的抖动。**拆分时顺手把「采集中」状态从 columns 依赖里摘出去**（例如把 `collectingDeviceId` 放进一个 `useRef` + 事件回调，或用 `useCallback` 稳定 cell renderer），这才是有实际收益的「可维护性 + 性能」改进。
  - ⚠️ **不要**为了性能在拆分中给 `DeviceManagement` 整体套 `React.memo` —— 它接收无 props，memo 无意义；给 `DeviceTablePanel` 套 memo 才有意义（配合 D4）。
- `React.memo` 的**正确用法**（若 D4 落地）：`DeviceStatsCards`（已是独立组件）可 memo 化；`DeviceModals` **不要** memo（props 极多且含 Form 实例，浅比较必然失配）。

#### 与 v1.19 port-write 冻结交互的关联性（结论）

| 关联点 | 判定 |
|---|---|
| **代码重叠** | **无**。v1.19 port-write 的前端交付面是 `src/components/network/port-write/*`（`PortWriteModal` / `BulkWriteDrawer` / `PortBindingModal` / `SetAccessVlanModal` / `constants.ts`）与 `src/pages/network/ports/index.tsx`。**devices 页不在其中**。 |
| **功能耦合** | devices 页「采集端口」→ `POST /network/devices/:id/collect-ports` → 写 `sys_device_port_status` ⇒ **ports 页读它**。v1.19 冻结的是「**端口写命令**（shutdown / description / dot1x / vlan / binding）」，**不含端口采集**。⇒ 拆 devices 页不触碰冻结语义。 |
| **真正要小心的地方** | ① **F6 会拆 `src/components/network/port-write/PortWriteModal.tsx`**（react-refresh P1 的 `showAuditLinkToast`）⇒ **这才是 v1.19 冻结面的触碰点** ⇒ 建议 **F6 对 port-write 只做豁免登记、不拆文件**（见 §2.6 风险表）。② devices 页的 `canViewMACHistory` 权限门（`:179`）依赖 `network:mac:list`，与 v1.19 无关。 |
| **回归要求** | 拆完后必须跑：**v1.19 port-write 62 个测试全绿**（`MILESTONES.md:12` 记录的基线）+ `src/pages/network/devices/__tests__/` 4 个用例 + `src/components/network/port-write/__tests__/` 6 个用例。 |

#### 用户可观察的「done」

> **用户视角零变化**：devices 页的所有交互（列表 / 搜索 / 部门树联动 / 采集端口 / 快速创建 / 编辑 / 删除 / 批量导出 / MAC 历史入口 / 分页 / 排序）**行为与视觉完全不变**；改动只体现在「文件从 714 行拆成 ≤4 个文件、单文件 ≤ 400 行」。**

#### 完成判据如何机械验证

| # | 判据 | 验证方式 | 通过标准 |
|---|---|---|---|
| F7-A | 文件规模 | `wc -l src/pages/network/devices/index.tsx` | ≤ 400 行（目标）；且新增文件均 ≤ 250 行 |
| F7-B | **行为不变** | 浏览器逐项走查（见下） | 与改动前截图逐项一致 |
| F7-C | 既有测试 | `npm test`（含 devices 4 个 + port-write 6 个） | 全绿 |
| F7-D | **v1.19 冻结** | v1.19 port-write 62 测试 | 全绿；`src/components/network/port-write/*` **diff 为空**（若采纳 F6 的「不拆」建议） |
| F7-E | 类型/构建 | `npm run type-check` + `npm run build` | exit 0 |
| F7-F | lint 不回归 | `npx eslint .` | 不新增 errors/warnings |

**行为不变逐项清单（UAT 脚本）**：① 列表加载 + 分页 + 排序（9 个 sorter）；② 搜索（5 字段）+ 重置；③ 左侧部门树联动筛选；④ 「采集端口」（含进行中 disabled + `aria-busy`）；⑤ 「快速创建」（含「设备探测」）；⑥ 「手动新增 / 编辑」（含凭证回填 `ensureCredential`）；⑦ 「删除」（ConfirmDelete 文案 + 对象名）；⑧ 批量删除（INV-8 逐字文案）；⑨ 批量导出（NetworkExport）；⑩ 「查看 MAC 历史」入口（权限门 `canViewMACHistory`）；⑪ 统计卡片随刷新联动；⑫ 「已选择 N 个设备」Alert + 取消选择。

#### 依赖与前置

- 无外部依赖。
- ⚠️ **建议列为 v1.24 可选项**（见 A5）——与 F1-F6 无依赖，且历史已明确「拆分风险 > 收益」。
- ⚠️ 若 F7 期间同时做 F6 的 `ActionButtons` 拆分（D 组）⇒ `createDeleteAction` 的 import 路径会变 ⇒ 需与 F6 合并排期。

#### 风险点

| 风险 | 概率 | 影响 | 缓解 |
|---|---|---|---|
| `reloadRef` 回填链路在拆分中失效（PageTable children 里回填 `ctx.loadData`） | **中** | **高**（删除/批量操作后不刷新） | D3/D4 保留「外部注入 ref」语义；UAT 清单第 4/7/8 项专测「操作后列表刷新」 |
| `columns` 的 `useMemo` 依赖不完整（既有 eslint-disable）⇒ 拆分时顺手「修正」依赖 ⇒ **引入无限渲染循环**（`CLAUDE.md` 明确警告） | **中** | **高** | 拆分时**保持依赖数组原样**（含 disable 注释），性能优化另开一个独立改动并单独验证 |
| `DeviceModals` 被顺手动（404 行）⇒ props 契约漂移 | 中 | 中 | 明确「不拆 DeviceModals」 |
| 触碰 v1.19 冻结面 | 低 | **高** | F6 对 port-write 只登记豁免 |
| 与 F6/F2 同文件改动冲突 | 中 | 中 | `ActionButtons` 相关的三件事（F6 拆分、F2 加 `perm`、F7 引用）**合并为一个 wave** |

---

## 3. 双框完成判据矩阵（★ 用户指定专项）

### 3.1 矩阵维度

- **输入框类型（11 类）**：普通 / allowClear / prefix / suffix / password / search / readonly / disabled / TextArea / Select / DatePicker / InputNumber
- **颜色模式（2）**：浅色（`data-color-mode="light"`）/ 暗色（`data-color-mode="dark"`）
- **交互态（5）**：默认 / 有值 / 聚焦（focus）/ hover（含 ✕ 显形）/ 禁用

### 3.2 必测矩阵（⭐ = 必测；⚠️ = 受影响类型；— = 不受本缺陷影响，但仍需确认未回归）

| # | 输入框类型 | 现状判定 | 为什么 | 浅色 | 暗色 | 备注 |
|---|---|---|---|---|---|---|
| 1 | **普通 `<Input>`（无 affix，走 `.ant-input-outlined`）** | ✅ 单框（antd 与项目 CSS 落在**同一元素**） | `:875` 的 border 与 antd 的 `ant-input-outlined` border 在同一 `<input>` 上 ⇒ 只有一圈 | ⭐ | ⭐ | **回归基线**：这是「正确形态」的参照 |
| 2 | **`<AppInput>`（allowClear 默认 true）** | ❌ **双框** | `hasPrefixSuffix` 命中 `allowClear` ⇒ 走 affix-wrapper；内层 `<input class="ant-input">` 被 `:875` 强加边框 | ⭐ | ⭐ | **309 个受影响输入框的主体**；空值态也要截（此时 ✕ 隐藏但 wrapper 仍在） |
| 3 | **`<AppInput allowClear={false}>`** | ❌ 双框（**仍然**） | `hasPrefixSuffix` 还需 prefix/suffix；仅 allowClear=false 时 antd **仍可能**渲染 affix-wrapper？—— ⭐**必须实测确认**：若 antd 6 在 `allowClear=false` 且无 prefix/suffix 时**不**渲染 wrapper，则此行是**单框**（可作为「修复后行为的对照」） | ⭐ | ⭐ | **这是判定修复是否生效的最强对照**：修复后第 2 行与第 3 行应**完全同形** |
| 4 | **prefix（搜索图标等）** | ❌ 双框 | `prefix` 独立触发 `hasPrefixSuffix` | ⭐ | ⭐ | 全仓 139 处 prefix/suffix；典型：`src/components/shared/ColumnConfigModal.tsx:169`、`src/components/DeptTree/index.tsx:204`、`src/components/IconSelect/index.tsx:131` |
| 5 | **suffix（单位/按钮等）** | ❌ 双框 | 同上 | ⭐ | ⭐ | |
| 6 | **password（`type="password"`）** | ❌ 双框 | type 不影响渲染路径 | ⭐ | ⭐ | 额外看点：密码框的 ✕ 显形是否遮挡掩码 |
| 7 | **search（`Input.Search` / `<Search>`）** | ❌ 双框 | antd 6 的 `Input.Search` = `ant-input-group-wrapper` > `ant-input-affix-wrapper` > `input.ant-input` ⇒ 内层仍被命中 | ⭐ | ⭐ | 实例：`src/components/shared/GlobalSearch.tsx:231` |
| 8 | **readonly** | ❌ 双框 | readonly 不改渲染路径（`BaseInput.js` 只把 clear icon 置为 `needClear=false`） | ⭐ | ⭐ | 额外看点：readonly 时 antd 加 `.ant-input-affix-wrapper-readonly` |
| 9 | **disabled** | ❌ 双框 | 同上；额外看点：`:3736` 的暗色 `[data-color-mode="dark"] .ant-input[disabled]` **只改内层**，与外层 `:3746` 不同步 ⇒ **禁用态可能出现内外两层不同色** | ⭐ | ⭐ | ⚠️ **暗色禁用态是双重边框 + 双层底色的重灾区** |
| 10 | **TextArea（`<AppTextArea>`）** | ❌ 双框 | antd 自己的 CSS 写了 `.ant-input-affix-wrapper > textarea.ant-input { border: none; outline: none; background: transparent; }` ⇒ 被 `:875` 的 `!important` **精确覆盖** | ⭐ | ⭐ | ⭐ **证据最硬的类型**：antd 明确要无边框，我们覆盖了它 |
| 11 | **Select（`<AppSelect>`）** | ✅ 单框 | antd 6 的 Select 边框在 `.ant-select-selector` 上（同一元素），项目 `:873` 与 antd 同元素 ⇒ 一圈；`allowClear` 在 Select 里是 selector 内的图标，不产生嵌套 input | ⭐ | ⭐ | **回归项**：确认修复没有把 Select 的边框改没 |
| 12 | **DatePicker** | ✅ 单框 | 项目 CSS **未对 `.ant-picker` 设 border**（浅色只有 `:1497-1522` 的 dropdown/panel 样式）；antd 自己的 `.ant-picker` 承担边框 | ⭐ | ⭐ | 回归项 |
| 13 | **InputNumber** | ✅ 单框 | antd 6 的 InputNumber 内层是 `.ant-input-number-input`（**不是** `.ant-input`）⇒ `:875` 不命中；边框在 `.ant-input-number-outlined`（同一外层元素） | ⭐ | ⭐ | 回归项 |

> **⚠️ 第 3 行的关键判定**：修复后**第 2 行（allowClear）与第 3 行（allowClear=false）必须视觉完全一致**。如果修复后第 2 行仍与第 3 行不同 ⇒ 说明 wrapper 的内层边框没有被清掉 ⇒ 修复不完整。这是**最省事的机械判据**：不需要逐类型截图，只要「allowClear 开/关」两态同形即可证明根因被切断。

### 3.3 截图矩阵（最小充分集）

**每格 = 1 张截图**，共 **11 类型 × 2 模式 = 22 张**（其中未受影响的 3 类可只截浅色 ⇒ 实际 19 张）；每张必须包含 **默认态 + 聚焦态**（聚焦态用 `Tab` 或脚本 `.focus()` 触发，验证无双圈光晕）。

| 页面（代表） | 覆盖类型 | 浅色 | 暗色 |
|---|---|---|---|
| `/operations/buildings`（新增/编辑楼宇弹窗） | 2 普通 allowClear、3 allowClear=false、10 TextArea | ⭐ | ⭐ |
| `/system/user`（用户表单） | 4 prefix、5 suffix、9 disabled、6 password | ⭐ | ⭐ |
| `/system/dict`（字典页） | 8 readonly | ⭐ | ⭐ |
| `/operations/workstations`（工位弹窗） | 11 Select、12 DatePicker、13 InputNumber | ⭐ | ⭐ |
| 全局搜索（顶栏） | 7 search | ⭐ | ⭐ |
| `/operations/buildings` 搜索卡 | 1 裸 Input（对照基线） | ⭐ | ⭐ |

**切暗色的操作路径**（截图脚本必须走真实路径，不能只改 DOM 属性）：
`系统设置 → 默认主题`（`src/pages/system/settings/default-theme.tsx`）或个人设置（`src/pages/settings/index.tsx:68` 读 `preferences.theme.mode`）切换「深色模式」→ `document.documentElement[data-color-mode]` 由 `src/store/themeStore.ts:50-51` 写入 ⇒ **截图前须断言 `document.documentElement.getAttribute('data-color-mode') === 'dark'`**。

### 3.4 附带必须核对的非双框项（修复的连带风险）

| # | 项 | 判据 |
|---|---|---|
| N1 | 内层 `outline` | `getComputedStyle(内层).outlineWidth === '0px'`（清 border 后浏览器默认 focus ring 可能冒出来） |
| N2 | 内层 `backgroundColor` | 透明（当前是 `var(--input-bg)` ⇒ 修复后内层应与外层同色或透明） |
| N3 | 内层 `borderRadius` | `0px`（当前是 `var(--theme-radius-md)`） |
| N4 | ✕ 清空按钮位置 | 有值 + hover 时 ✕ 出现，**不压住文字**（antd 用 `padding-inline-end: var(--ant-padding-lg)` 预留；项目 `:875` 的 `border` 简写不影响 padding ⇒ 预期无位移，但必须目视） |
| N5 | 聚焦光晕圈数 | 外层 `box-shadow` 含 **1 组** `0 0 0 2px` |
| N6 | 禁用态双层底色 | 暗色下内外层 `background-color` 一致 |
| N7 | Select/DatePicker/InputNumber 边框未消失 | 三者的 `borderWidth === '1px'` |

---

## 4. #16 定责方法（★ 用户指定专项 · 完整版）

### 4.1 前端权限判定链路在哪（实测）

**结论：没有统一链路，是逐页 opt-in。**

```
useMenuStore.permissions                       ← 唯一权限来源（zustand，非 persist）
   ├─ fetchAll() / fetchPermissions()          src/store/menuStore.ts:101-170
   │     └─ getUserPermissions()               src/lib/menuApi.ts:23-26
   │          └─ POST /system/my-menus/permissions   ← 后端无权限门（menu_router.go:63-65）
   ├─ 判定写法 A（页面内自建）：  const hasPerm = (p) => menuPermissions.includes(p)
   │     · src/pages/network/ports/index.tsx:131-132   （canWrite = hasPermission("network:port:write")）
   │     · src/pages/network/mac/history/MACHistoryPage.tsx:118-119
   │     · src/pages/operations/workstations/LocationAliasDrawer.tsx:63-65
   │     · src/pages/vdi/VirtualMachineList/index.tsx:46
   ├─ 判定写法 B（组件内自建）：  src/pages/operations/workstations/index.tsx:121
   └─ 路由级： routeConfigManager.hasPermission(path, permissions)  src/router/routeConfigManager.ts:90-105
             + RouteGuard  src/router/RouteGuard.tsx:41-43
```

**❌ 用户问题里提到的 `useMenuStore.canWrite` —— 不存在。** `useMenuStore` 只有 `permissions: string[]` 字段（`src/store/menuStore.ts:21`），**没有 `canWrite` 这类派生 getter**；`canWrite` 是 `network/ports` 页面自己定义的局部 const（`src/pages/network/ports/index.tsx:132`）。⇒ **v1.24 若要统一，必须先新建这个派生层**（见 D1）。

**❌ 「行操作按钮的权限判定」在 `ActionButtons` / `createDeleteAction` 里 —— 也不存在。** `src/components/shared/ActionButtons.tsx:19-28` 的 `ActionButton` 接口**没有 `perm` 字段**，`createDeleteAction`（`:114`）也不接受权限参数 ⇒ **按钮渲染层完全不知道权限的存在**。

**⇒ 页面前端权限判定的「唯一真源」= `useMenuStore.permissions`（即后端 `/system/my-menus/permissions` 返回的字符串数组）。**

### 4.2 全仓「有权限判定」vs「没权限判定」的分布（决定 F2 的工作量）

| 层次 | 现状 | F2 影响 |
|---|---|---|
| **路由级** | ✅ 有（`routeConfigManager.hasPermission` / `RouteGuard`） | 不动 |
| **页面级（菜单/按钮）** | ⚠️ 12 个文件用 `useMenuStore.permissions`（`src/pages/` 下 10 个 + 2 测试） | 大部分页面**无**行操作权限判定 |
| **组件级（行操作按钮）** | ❌ **无**（`ActionButton` 无 `perm`） | F2 的核心落点（D1 契约下沉） |
| **后端 API 级** | ❌ **ANY-of 语义**（39 处路由组，`Permission()` 用 0 次） | F2b 的核心 |

### 4.3 如何构造受限用户做 UI 复测

**推荐路径（绕开 sqlite 500）**：见 §2.2「PG 环境复测的具体步骤」7 步。要点：
1. **PG 环境**（sqlite 下角色批量分配菜单 500 —— `75-UAT.md:91`）。
2. **UI 正常路径建角色**（角色管理 → 新增 → 只勾「楼宇管理」菜单 + 「楼宇查询」按钮），**不要手工插库**（v1.23 就是栽在这里）。
3. **建用户并绑定该角色**（用户管理 → 新增 → 角色 `uat_readonly`）。
4. **先核权限集**：`POST /system/my-menus/permissions`（无权限门）⇒ 必须**不含** `ops:building:add/edit/delete`。含 ⇒ 判 D5（环境问题），先修数据。
5. **UI 复测**：受限账号登录 → 楼宇管理 → 数按钮。
6. **API 复测（决定性）**：同 token 直接打 delete/update/create ⇒ 期望 403。
7. **对照**：admin 账号重复 ⇒ 期望按钮可见 + API 200。
8. **⏱ 权限缓存 30s**：改角色后 `sleep 31` 再测（`pkg/middleware/permission.go:36,64-66`）。

**降级方案（拿不到 PG 时）**：
- 用 **sqlite + 手工插库**（v1.23 做法）只能验 UI 层，**验不了后端**（因为 v1.23 的手工插库角色绕过了菜单缓存加载链路，权限集不可信）。
- 或用 **admin 账号 + 前端 mock**：在浏览器里 `useMenuStore.setState({ permissions: ["ops:building:list","ops:building:query"] })` ⇒ 模拟受限用户的**前端**表现。⚠️ 这只能验 F2a（前端 UI 判定），**验不了 F2b（后端 403）**。
- ⇒ **降级方案只能让 F2a 收档，F2b 必须等 PG。**

### 4.4 「判为缺陷」的标准（终版）

在 PG 环境、权限集已核实为「只有 list/query」的前提下：

| # | 判据 | 观测 | 判定 | 归属 |
|---|---|---|---|---|
| **D1** | 受限用户楼宇页仍见「编辑」 | editCount > 0 | 前端缺陷 | 前端（F2a） |
| **D2** | 受限用户仍见「删除」 | delCount > 0 | 前端缺陷 | 前端（F2a） |
| **D3** | 受限用户 `POST /ops/building/:id/delete` 返回 200 且楼宇消失 | — | **后端越权（严重）** | 后端（F2b） |
| **D4** | 受限用户 `POST /ops/building/:id/update` 返回 200 且内容被改 | — | **后端越权（严重）** | 后端（F2b） |
| **D5** | 受限用户 `POST /ops/building` 返回 200 | — | **后端越权（严重）** | 后端（F2b） |
| **D6** | 权限集本身含 add/edit/delete | — | **环境/数据问题**（非代码缺陷） | 重建角色后重测 |
| **D7** | 权限集端点 404/500 | — | 环境问题 | 修复后重测 |
| **D8** | 全权限用户**也**看不到「删除」 | delCount = 0 | **误伤缺陷**（fail-close 全禁） | 前端（F2a） |
| **D9** | 受限用户 `POST /ops/building/list` 返回 403 | — | **读权限被误伤** | 后端（F2b） |

> **判定优先级**：**D6/D7（环境）→ D9（读被误伤）→ D3/D4/D5（后端越权）→ D1/D2（前端）→ D8（误伤）**。
> **关键**：D3/D4/D5 的优先级高于 D1/D2 —— 因为「按钮可见」只是表象，「API 可越权」才是实质危害。v1.23 UAT 只测了 D1/D2，所以无法定责；**v1.24 必须加上 API 层判据**。

### 4.5 与 v1.23 UAT §8 #16 描述的对照表

| 维度 | v1.23 UAT §8 #16 原文 | 本次研究结论 | 变化 |
|---|---|---|---|
| 现象 | 受限用户仍见「编辑/删除」按钮（delCount=9） | **确认复现** | — |
| 根因候选① | 前端 `hasPerm` 在权限获取失败时未隐藏危险按钮（fail-open） | ❌ **不成立**：楼宇页**根本没有任何权限判定代码**；且前端权限端点 `/system/my-menus/permissions` **无权限门**，不会被 403 | 候选被排除 |
| 根因候选② | 手工插库角色数据不完整导致后端权限异常 | ❌ **可排除**：即便权限完全正确，前端也不看权限；且**后端 ANY-of 语义让只读用户能真删** | 候选被排除 |
| 缺失的第三种可能 | — | ⚠️ **未被发现**：**后端写接口 ANY-of 语义 ⇒ API 层越权** | **新发现（严重）** |
| 环境限制 | sqlite 角色创建 API 500 | ✅ 已给出绕过路径（PG + UI 正常路径建角色） | 可解 |
| 复测步骤 | 「建议在 PG 生产库复测定责」 | ✅ 已给出 7 步可执行脚本（含权限集核验 + API 直打 + 30s 缓存等待） | 可执行 |
| 判缺陷标准 | 未定义 | ✅ 已定义 9 条（D1-D9） | 可判 |
| 建议 | — | ⭐ **建议把 F2 拆成 F2a（前端）+ F2b（后端），F2b 单独排期并优先** | 新增 |

---

## 5. Feature Dependencies

```
[F1 双框修复]
    └──requires──> [antd 6.6.4 affix-wrapper 渲染契约]（已读源码，无需改）
    └──requires──> [4 处 CSS 块同步修改]（index.css:871/908/2252/3717）
    └──conflicts──> [F6-A errors 归零]（同一 CSS/组件文件，但无技术冲突）
    ⭐ 与 F2/F3/F4/F5/F7 完全独立 —— 可第一天就开工

[F2 行操作权限]
    ├──F2a 前端 UI 判定
    │     └──requires──> [契约层下沉：ActionButton 增加 perm 字段]（src/components/shared/ActionButtons.tsx:19-28）
    │     └──requires──> [useMenuStore.permissions]（src/store/menuStore.ts:21）
    │     └──requires──> [PG 环境 + 受限用户]（硬阻塞）
    │     └──shares──> [F6 的 ActionButtons 拆分]（同一文件 ⇒ 同 wave）
    └──F2b 后端按动作分权
          └──requires──> [39 处路由组改造]（internal/api/router.go）
          └──requires──> [继承语义处置]（pkg/middleware/permission.go:159-196）
          └──requires──> [PG 环境]（硬阻塞）
          └──conflicts──> [spaces:list 旁路契约]（permission.go:305-315 注释）
          ⭐ 优先级高于 F2a（安全）

[F3 3f130 诊断前端]
    └──requires──> [后端诊断端点]（✅ 已交付 v1.23 commit 8b85fe6b）
    └──requires──> [opsApi.ts 新增 wrapper]（src/lib/opsApi.ts:617）
    └──enhances──> [WorkstationDeviceTable 的空态文案]（:604-607 静态文案）

[F4 F5 刷新留原页]
    └──requires──> [菜单缓存 hydrate 时序]（src/router/DynamicRoutes.tsx:173/179-186/275/315）
    └──requires──> [saveLastPath 副作用处置]（:243-247）
    └──shares──> [F6 的 DynamicRoutes 拆分]（同一文件 ⇒ 同 wave）
    └──enhances──> [v1.23 FAIL-1 分页持久化恢复]（75-UAT.md:76-81）

[F5 ColumnConfigModal:136]
    └──requires──> [§1.1 六处清单一致]（68.1-UI-SPEC.md:44）
    └──requires──> [eslint 封装层豁免块处置]（eslint.config.js:265-287）
    └──shares──> [§8-A 豁免理由表失同步]（同一机制 ⇒ 同一 phase）

[F6 lint 清零]
    └──requires──> [F5 的 eslint 配置调整]（否则新增注释 ⇒ unused-disable）
    └──requires──> [§8-A 理由表同步]（exemption-reasons.json）
    └──requires──> [§8-C AppSelect 契约决策]（否则 errors 降不到 0）
    └──shares──> [F2a 的 ActionButtons] / [F4 的 DynamicRoutes]（同 wave）
    └──depends──> [F6-D 预算归零]（package.json + assert-lint-budget.mjs 双写）

[F7 devices 拆分]
    └──independent──> F1/F2/F3/F4/F5/F6
    └──shares──> [F6 的 ActionButtons]（若 F6 拆了 createDeleteAction）
    └──guards──> [v1.19 port-write 冻结]（不碰 src/components/network/port-write/*）
```

### Dependency Notes

- **F1 是唯一「零依赖 + 零风险 + 最大用户感知」的项** ⇒ 必须第一天开工，且不应等任何环境。
- **F2 是唯一的安全项，且是唯一「硬阻塞在外部环境」的项** ⇒ 必须**第一时间向用户申请 PG 环境**（v1.23 的 BLOCKERS #2 未解决，已欠了一个里程碑）。
- **F6 拆出的三处共享文件（`ActionButtons` / `DynamicRoutes` / `port-write`）与 F2a、F4 撞车** ⇒ 建议 F6 分三刀，与 F2a、F4 合并 wave，避免同一文件两次改动。
- **F7 与所有项无依赖** ⇒ 放最后，或移出 v1.24。

---

## 6. MVP Definition（v1.24 必交 vs 可选）

### Launch With（v1.24 必交）

- [ ] **F1 输入框单层外框** — 309 个输入框的视觉回归，1 处 CSS 根因，最高性价比
- [ ] **F2a 前端行操作权限判定** — 用户可见的安全信号
- [ ] **F2b 后端按动作分权**（+ 继承语义处置）— ⭐ **唯一实质安全项**；不可延期
- [ ] **F3 3f130 诊断前端展示** — 后端已交付，前端接线成本低
- [ ] **F4 F5 刷新留原页** — 用户日常高频动作
- [ ] **F5 ColumnConfigModal:136 豁免注释 + 理由表同步** — 契约完整性
- [ ] **F6 lint 17 errors + 33 react-refresh 清零 + 预算归零** — 契约守护

### Add After Validation（v1.24 中后段 / v1.24.x）

- [ ] **§8-A 豁免理由表全量同步**（`check-exemption-reasons.mjs` 从 exit 1 → 0）— 与 F5/F6 合并做
- [ ] **F7 devices 页拆分**（D1 + D2 两刀，714 → ~590）— 无依赖、可延后
- [ ] **§8-C AppSelect 契约与 `local/no-large-dropdown-list` 的对齐决策** — 需产品/架构决策
- [ ] **F7 的 D3/D4 两刀**（→ ~400 行）— 视 F7 前两刀的实际手感再定
- [ ] **D2 CSS 契约测试脚本**（computed-style 断言）— 防双框类回归复发

### Future Consideration（v1.25+）

- [ ] **双框类回归的制度化防护**：把「antd 组件 DOM 层级 + computed style」纳入组件冒烟脚本
- [ ] **`useMenuStore` 派生权限层**（`canWrite(perm)` / `canRead(perm)`）—— F2a 若走 D1 路径会自然产生
- [ ] **行操作权限的全面铺开**（12 个已用 `useMenuStore` 的页面之外的所有列表页）
- [ ] **`requireSinglePermission` 风格的中间件**（比 ANY-of 更严格的默认口径），逐模块替换

---

## 7. Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority | 外部阻塞 |
|---------|------------|---------------------|----------|----------|
| F1 双框修复 | **HIGH**（309 个输入框，全站可见） | **LOW** | **P0** | 无 |
| F2b 后端按动作分权 | **HIGH**（越权 = 数据损坏） | **HIGH** | **P0** | ⭐ PG 环境 |
| F2a 前端权限判定 | HIGH | MEDIUM | **P1** | ⭐ PG 环境（可用 mock 降级） |
| F3 3f130 诊断展示 | MEDIUM（运维排障效率） | **LOW** | **P1** | 无（需 3f130 工位可达） |
| F4 F5 刷新留原页 | MEDIUM（日常高频） | MEDIUM | **P1** | 无 |
| F6 lint 清零 | LOW（用户无感，DX 收益） | MEDIUM | **P2** | 无（但依赖 F5 + §8-A + §8-C） |
| F5 ColumnConfigModal:136 | LOW | **LOW** | **P2** | 无 |
| F7 devices 拆分 | LOW | MEDIUM | **P3** | 无 |
| §8-A 理由表同步 | LOW（流程完整性） | LOW | **P2** | 无 |
| §8-C AppSelect 契约决策 | LOW | LOW（决策成本高） | **P3** | ⭐ 需产品决策 |

> **排期建议（3 波）**：
> - **Wave 1（第 1-2 天，零外部依赖）**：F1 + F5 + F3 + §8-A ⇒ 拿满「低风险高感知」收益。
> - **Wave 2（第 2-4 天，需 PG）**：F2b（后端，按域分波）+ F2a + F4 ⇒ 收掉安全项与高频项。
> - **Wave 3（第 4-5 天）**：F6（含三处共享文件拆分）+ F7（可选 D1/D2）。

---

## 8. 新发现的遗漏项与新增风险（★ 用户指定）

### A. 豁免理由表已与 eslint 配置失同步 —— `check-exemption-reasons.mjs` **exit 1 / 123 项**

```
$ node scripts/check-exemption-reasons.mjs
[FAIL] 豁免清单 ↔ 理由校验失败，共 123 项：
  - [缺理由] 仍被豁免但理由清单无对应条目：P65|src/__p65_no_exemptions__/**
  - [多理由] 理由清单有条目但已不在豁免清单（已收缩，应删除）：P65|src/components/CronSelector/fields/DayField.tsx
  …（共 122 条「多理由」）
当前 eslint 豁免：P65 1 / P66 37（合计 38）；理由清单 159 条
EXIT=1
```

- **性质**：v1.23 把 P65 豁免清零后，`scripts/exemption-reasons.json`（159 条）**没有同步收缩**（实际只剩 38 条）。
- **影响**：68.1 §8 #12/#18 的「豁免清单 100% 带理由 + 脚本幂等」验收口径**已不成立**（UAT #18 只验了生成脚本 diff 为空，**没跑这个校验脚本**）。
- **归属**：与 F5「契约登记完整性」**完全同族** ⇒ 建议合并成一个 phase。
- **注意**：这条校验脚本**不在 `npm run lint` / `lint:ci` 里** ⇒ CI 不会拦 ⇒ 需要显式接线。

### B. lint 预算双写死在两处，清零时必须同步

| 位置 | 当前值 | 应改为 |
|---|---|---|
| `package.json` → `scripts.lint` | `eslint . --max-warnings 34` | `--max-warnings 0` |
| `package.json` → `scripts.lint:ci` | `eslint . --max-warnings 34` | `--max-warnings 0` |
| `scripts/assert-lint-budget.mjs:20` | `maxErrors: 17` | `0` |
| `scripts/assert-lint-budget.mjs:21` | `maxWarnings: 34` | `0` |

⚠️ 只改 `package.json` 不改脚本 ⇒ `lint:ci` 仍按 17/34 放行；反之亦然。**两处必须同时改**，并在判据 F6-D 里加一条「故意注入 1 条 error ⇒ `npm run lint:ci` 必须 exit 1」。

### C. `AppSelect` 契约与项目自有规则 `local/no-large-dropdown-list` **正面冲突**（F6 的隐藏阻塞）

- 规则（`eslint-rules/no-large-dropdown-list.cjs:62-70`）：**任何 `<Select>` / `<AutoComplete>` JSX 缺少 `onSearch` 即报 error**（"必须使用服务端搜索 + 防抖"）。
- 但 Phase 65 的 `AppSelect` 契约（`src/components/ui/AppSelect.tsx:13-25`）是 **客户端过滤**：`showSearch ?? count >= 8` + `optionFilterProp="label"` ⇒ **故意不用服务端搜索**。
- ⇒ `src/components/ui/AppSelect.tsx:16` 报 error（`local/no-large-dropdown-list`）。
- **⇒ F6 的 errors 归零被这一条卡住**，且它**不是 lint 债，是两个契约打架**。
- **三个选项**（需决策）：
  1. **改规则**：给 `local/no-large-dropdown-list` 加 `excludedFiles: ["src/components/ui/**"]`（契约封装层豁免），并在规则文档里写明「AppSelect 走客户端过滤；服务端搜索由调用方按需透传 `onSearch`（`...rest` 已支持）」。⭐ **推荐**。
  2. **改 AppSelect**：把 `onSearch` 透传做成必填？⇒ 破坏「透传封装」契约，74 处调用方全要改。❌
  3. **加 eslint-disable**：在 `AppSelect.tsx:16` 加带理由的 disable。⚠️ 但这会让「规则要求服务端搜索」这条**项目级安全约束**在一个封装层被公然豁免，且 §2.6 的 P2 原则（不要用 disable 掩盖）会被质疑。**次选**。
- ⚠️ 顺带：`eslint.config.js:24-26` 的注释说「20 处 `eslint-disable local/no-large-dropdown-list` 注释随之报 'rule not found'」⇒ 说明这个规则**历史上就有 20 处 disable**，与本次矛盾同源。

### D. 后端权限模型的系统性风险（超出 #16 范围，但必须登记）

| 发现 | 位置 | 风险 |
|---|---|---|
| 全部 39 处路由组用 ANY-of 语义 | `internal/api/router.go`（`RequirePermissions` 39 次，`Permission()` **0 次**） | 任何持有 `*:list` 的用户都能调用同组的写接口 |
| F 型按钮权限 → C 型父菜单的继承 | `pkg/middleware/permission.go:159-196` | 只读按钮权限可放大为父菜单访问权；写判定若复用同一函数则同样被放大 |
| 模块级权限兜底 | `pkg/middleware/permission.go:198-216` | `extractModulePermission` 把 `x:y:z` 降级成 `x:y`；若存在 perms 恰为 `x:y` 的菜单行 ⇒ 再放行 |
| 中间件权限缓存 30s 且失效函数未接线 | `pkg/middleware/permission.go:36, 64-66`（注释明说「当前未接线」） | 角色/权限变更后最长 30s 提权窗口；UAT 必须 `sleep 31` |
| 业务逻辑层零权限复核 | 例：`internal/api/v1/operations/building_handler.go:175-183` | 路由组一旦配错，handler 层无兜底 |

⇒ **建议在 v1.24 至少做一件事**：把「全仓 39 处 ANY-of 路由组」列成**显式债务清单**（文件 + 组 + 涉及的 4 类权限 + 是否含写接口），哪怕 v1.24 只修 `ops:building` 一个组，也要把清单登记下来，否则 #16 修完「楼宇」而「工位/机房/资产/专线/信息点/网络设备…」**同样的洞还在**。

### E. 其它

| # | 发现 | 说明 |
|---|---|---|
| E1 | 后端 `PhysicalDevicesResult` 结构体注释与实现矛盾 | `workstation_physical_diagnosis.go:63-65` 写「有数据时 EmptyReason 为空」，实现 `:813-827` 与测试 `:388` 明确「未绑用户压过有数据」⇒ 顺手修正注释，否则前端会照注释写错逻辑 |
| E2 | `WorkstationDeviceTable` 的物理链路查询失败只 `console.warn` | `src/components/operations/WorkstationDeviceTable/index.tsx:127-129` ⇒ 失败时用户看到的是「0 台」的误导性空态，而非「加载失败」⇒ 建议 F3 顺手补 `message.warning` |
| E3 | `src/components/shared/PageTable.tsx:180` 有 1 条 unused-disable 指令 | 这是当前 34 warnings 里的第 34 条；F6 顺手清掉 |
| E4 | `devices` 页 `columns` 的 `useMemo` 依赖不完整 | `src/pages/network/devices/index.tsx:524-536`（`:535` 有 disable）⇒ `collectingDeviceId` 变化会重建整列定义 ⇒ 虚拟表抖动。**F7 拆分时不要顺手「修正」依赖**（会引入无限渲染），性能优化另开 |
| E5 | `useMenuStore` 无 `canWrite` 派生层 | 用户问题里假设的 `useMenuStore.canWrite` **不存在** ⇒ 若 v1.24 想统一权限判定，需先建派生层（D1） |
| E6 | `check-exemption-reasons.mjs` 未接入 CI | `package.json` 的 `lint`/`lint:ci` 都不跑它 ⇒ §8-A 类失同步不会被拦 |
| E7 | 端口采集与 v1.19 冻结的边界 | 「采集端口」写 `sys_device_port_status`，**不在** v1.19 冻结的「端口写命令」范围内 ⇒ F7 拆分不触碰冻结语义（但 F6 拆 `PortWriteModal.tsx` 会触碰 ⇒ 建议不拆） |

---

## 9. Sources

- **代码实测**（本次研究直接读取 / 执行，未修改任何源码）：
  - `xingran-react-frontend/src/index.css`（871-929 / 2252-2284 / 3717-3757 输入框边框四块；2051-2084 已有的「内层清零」正确范式）
  - `node_modules/antd/dist/antd.css`（`.ant-input-affix-wrapper > textarea.ant-input { border: none }` 等 antd 6.6.4 实际产物）
  - `node_modules/@rc-component/input/es/BaseInput.js:42,47-50,79-92` + `es/utils/commonUtils.js:4-6`（affix-wrapper 渲染契约）
  - `node_modules/eslint-plugin-react-refresh/index.js`（`:57-79` 选项、`:85-125` 判定、`:150-231` 报告条件、`:247-269` 常量表达式集合）
  - `src/components/ui/AppInput.tsx:12` / `AppTextArea.tsx:12`（allowClear 默认化）
  - `src/components/operations/WorkstationDeviceTable/index.tsx:77-140` 四路查询、`:580-612` 物理链路 Collapse、`:604-609` 现有静态空态文案（实测 685 行）
  - `src/pages/operations/buildings/index.tsx:374-378`（行操作无权限判定）
  - `src/pages/network/devices/index.tsx`（实测 714 行；块边界 69-171 / 285-420 / 451-514 / 539-563 / 565-711）
  - `src/router/DynamicRoutes.tsx`（85-115 / 173-186 / 227-240 / 243-247 / 275-277 / 280-317）
  - `src/store/menuStore.ts:21,52,101-170` / `src/lib/menuApi.ts:23-26` / `src/lib/opsApi.ts:600-620`
  - `src/pages/network/ports/index.tsx:131-132`（`canWrite` 的真实定义位置）
  - `src/components/shared/ActionButtons.tsx:19-28,114,167` / `FormModal.tsx:61` / `ModernTag.tsx:75` / `statusHelpers.tsx:63,68,73` / `ColumnConfigModal.tsx:134-146`
  - `src/pages/operations/workstations/constants.tsx` / `src/pages/system/dept/constants.tsx` / `src/pages/system/dict/constants.tsx`（P2 误判三文件）
  - `eslint.config.js`（`:58-59` 规则配置、`:265-287` 封装层豁免块、`:97-138` no-restricted-syntax 块）
  - `scripts/assert-lint-budget.mjs:20-21`（预算 17/34）/ `scripts/check-exemption-reasons.mjs`（**exit 1 / 123 项**）/ `scripts/exemption-reasons.json`（159 条）
  - `package.json`（`lint` / `lint:ci` = `--max-warnings 34`）
- **后端实测**：
  - `internal/api/router.go:195-203, 539-546, 686, 702`（路由组权限配置）
  - `pkg/middleware/permission.go:36, 64-66, 125-216, 277-303, 305-352`（ANY-of 语义、继承、模块兜底、缓存）
  - `internal/api/v1/operations/building_handler.go:175-183`（无 handler 级复核）
  - `internal/api/v1/operations/workstation_device_handler.go:146-149, 174-206`
  - `internal/services/operations/workstation_physical_diagnosis.go:27-52, 56-100`（枚举 + 返回结构）
  - `internal/services/operations/workstation_device_service.go:620-671, 795-900`（诊断 + 空因分类 + warnings）
  - `internal/services/operations/workstation_physical_diagnosis_test.go:201-272, 388`（契约断言）
  - `internal/core/db/init_data.go:693, 752-755`（楼宇菜单与按钮权限点）
- **规划文档**：
  - `.planning/milestones/v1.23-phases/75-uat/75-UAT.md`（§8 #12/#16/#19 原文、FAIL-1/FAIL-2、BLOCKERS）
  - `.planning/milestones/v1.22-phases/68.1-closeout-debt-clearance/68.1-UI-SPEC.md:44`（§1.1 判定表六处清单）、`:312`（§8 #12 判据原文）
  - `.planning/milestones/v1.22-phases/67-roi/67-SUMMARY.md:12,24,69`（devices 1282→692 拆分记录）、`67-VERIFICATION.md:27`
  - `.planning/milestones/v1.22-phases/68.1-closeout-debt-clearance/68.1-PLAN.md:508, 836, 852`（不拆分的决策记录）、`68.1-FACTS.md:322`
  - `.planning/v1.23-seeds.md` §B/C/D（devices 692 结转、lint 基线、v1.22 冻结契约）
  - `.planning/PROJECT.md:30`（默认值必须下沉封装层的锁定决策）、`:9,11`（Phase 53 port-write 权限 gating 记录）
  - `.planning/MILESTONES.md:12, 106-133`（v1.19 port-write 冻结面与 62 测试基线）
- **本次执行的只读命令**：`npx eslint .`（17 errors / 34 warnings）、`node scripts/check-exemption-reasons.mjs`（exit 1）、`grep`/`wc -l`/`sed -n` 系列源码定位。
- **仓库外沙箱实验**（`%TEMP%/rr-test/`，**未在仓库内创建任何文件**，实验后已清理）：用**仓库同版本**的 `eslint-plugin-react-refresh` + 同配置 `{ allowConstantExport: true }` + `@typescript-eslint/parser` 对 9 个最小样例实测，验证 §2.6 的分类与修法（a 现状=3 警告 / b 字面量拷贝=0 / c 重导出=3 / d 改小写名=0 / e `.ts` 后缀=0 / f `lazy()`=1 / g 组件+小写函数=1 / h 组件+对象字面量=1 / i 组件+字面量=0）。

---
*Feature research for: v1.24 视觉回归修复 · 权限定责 · 诊断接线 · 结构清偿*
*Researched: 2026-10-28*
