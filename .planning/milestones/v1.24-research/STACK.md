# Stack Research — v1.24「样式层（CSS）技术方案：antd 6 结构冲突与输入框双框修复」

**Domain:** 前端样式层（`src/index.css` × antd 6.6.4 DOM/样式结构冲突）
**Project:** RuoYi-Next（ruoyi-go-backend）
**Milestone:** v1.24
**Researched:** 2026-10-28
**Confidence:** HIGH（全部结论基于 `node_modules/antd@6.6.4` 与 `@rc-component/input` 实读 + 项目 CSS 实读 + 级联规范推理；**未跑 dev server，未做实机截图**——凡标注「需实机确认」的条目均属此列）

---

## Executive Summary

**根因成立，且比用户描述的更严重。** 上报的双框不是渲染偶发，是 **CSS 级联的确定性结果**：

1. antd 6.6.4 明确要求 affix-wrapper 内层 input **无边框**（`node_modules/antd/es/input/style/index.js:382-395`：`> input.ant-input { border: 'none'; borderRadius: 0; background: 'transparent'; }`）。
2. 项目 `src/index.css:875` 用 `.ant-input { border: 1px solid var(--input-border) !important; }` 覆盖了它。**CSS 规范中 `!important` 声明优先于同源任意非 `!important` 声明，与特异性无关** → 内层 input 被画出边框 → 双框。
3. v1.23 让 `AppInput` 默认 `allowClear=true`（`src/components/ui/AppInput.tsx:12`），而 antd 的 affix 分支触发条件是 `hasPrefixSuffix(props) = !!(prefix || suffix || allowClear || showCount)`（`antd/es/input/utils.js:1-3`）→ **allowClear 单独一个 prop 就足以把 Input 推进 affix-wrapper 结构** → 双框从"少数"变"几乎全部"。

**同时发现 3 个此前未被登记的问题（同源、同一批 `!important` 造成）：**

| # | 发现 | 严重度 |
|---|------|--------|
| F1 | Form 校验错误态的输入框**边框不会变红**——项目 `border: ... !important` 压掉了 antd 的 status 边框色（`antd/es/input/style/variants.js:41-51,63-78`） | P1 |
| F2 | **antd 6 已不存在 `.ant-select-selector` / `.ant-select-selection-item` / `.ant-select-selection-placeholder` 这三个 class**（`antd/es/select/` 全目录 grep 无命中；`@rc-component/select/es/` 仅注释命中）→ 项目 CSS 里 **12 条 Select 规则全是死规则**，暗色模式 Select 的背景/文字/边框根本没被项目接管 | P1 |
| F3 | 暗色模式下 antd 容器底色被 `AntdThemeBridge` 写死为 `ruoyiBrand.cream.surface = #FFFFFF`（`src/design-system/components/AntdThemeBridge.tsx:83` + `src/design-system/tokens/colors.ts:76`）→ 任何"改走 antd token"的方案都会在暗色模式炸出白底 | P1（决定方案 C 的成本） |

**推荐：方案 A（最小覆盖）落地 P0 修复，方案 B 的"职责拆分"作为同里程碑第二步，方案 C 推迟到独立里程碑。** 理由与影响面见 §(d)。

---

## (a) 根因验证

### (a.1) antd 6.6.4 的 DOM 与样式事实

| 事实 | 证据（文件:行号） |
|------|------------------|
| antd 实装版本 6.6.4 | `node_modules/antd/package.json:3` |
| affix-wrapper 内层 input 原生 `border: none` | `antd/es/input/style/index.js:382-395`：`[> input${componentCls}, > textarea${componentCls}] { fontSize:'inherit'; border:'none'; borderRadius:0; outline:'none'; background:'transparent'; color:'inherit'; '&:focus': { boxShadow:'none !important' } }` |
| TextArea 内层 textarea 原生 `border: none` | `antd/es/input/style/textarea.js:50-61`：`&-affix-wrapper.ant-input-affix-wrapper > textarea.ant-input { border:'none'; background:'transparent'; '&:focus':{boxShadow:'none !important'} }` |
| affix 分支触发条件含 `allowClear` | `antd/es/input/utils.js:1-3` `hasPrefixSuffix = !!(props.prefix \|\| props.suffix \|\| props.allowClear \|\| props.showCount)` |
| variant class 的落点分叉 | `@rc-component/input/es/BaseInput.js:42-46`（**无** affix 时 variant class 加到 input 自身）、`:79-88`（**有** affix 时 variant class 加到 wrapper，`classNames?.variant` 混入 `affixWrapperCls`） |
| wrapper 的 class 集合 | `@rc-component/input/es/BaseInput.js:79-88`：`ant-input-affix-wrapper` + `ant-input-{variant}` + `ant-input-{size}` + `ant-input-affix-wrapper-focused` |
| TextArea 的 wrapper **也带** `ant-input-affix-wrapper` | `@rc-component/input/es/TextArea.js:147-151` 传入 `classNames.affixWrapper = 'ant-input-textarea-affix-wrapper'`，再由 `BaseInput.js:80` 拼上 `ant-input-affix-wrapper` |
| 内层 input 的 class 恒含 `ant-input` | `@rc-component/input/es/Input.js:159-161`：`clsx(prefixCls /* = 'ant-input' */, {...}, classNames?.input)` |

### (a.2) 项目规则与 antd 规则的冲突（级联推理）

```
index.css:871-880
  .ant-input { background/border/color/border-radius/font-size/transition !important }
                              └─ border 简写 → 展开为 border-width/style/color 三个 important 长属性
antd/es/input/style/index.js:384
  .ant-input-affix-wrapper > input.ant-input { border: none }   ← 非 important
```

**判定**：author `!important` > author 非 `!important`（CSS Cascade 4 §6.4.4）→ **项目规则胜出**，内层 input 获得 `1px solid var(--input-border)`。

**视觉证据链**（与用户截图逐项对上）：

| 截图现象 | 来源 |
|---------|------|
| 外层浅灰圆角框 | `.ant-input-affix-wrapper` 的 `border: 1px solid var(--input-border)`，`--input-border: #dbd7ce`（`index.css:185`） |
| 内层白色圆角输入框 | 内层 input 的 `background: var(--input-bg)` = `#ffffff`（`index.css:184`）+ `border-radius: var(--theme-radius-md)` = 8px（`index.css:877`，antd 原生应为 `borderRadius: 0`） |
| 内层边框随焦点变绿 | `index.css:891` `.ant-input:focus { border-color: var(--input-border-focus) !important }` → 命中**内层** input |

### (a.3) 为什么是"部分"输入框 —— 三处项目内已有的局部补丁（关键旁证）

项目里已经有人踩过同一个坑并**就地打了补丁**，但从未推广到全局：

| 位置 | 内容 | 效果 |
|------|------|------|
| `src/pages/login/login.css:351-358` | `.login-form .ant-input-affix-wrapper > input.ant-input { background:transparent; border:none; box-shadow:none; padding:0; height:auto } !important` | 特异性 (0,4,0) 压过 `index.css:871` 的 (0,1,0) → **登录页无双框** |
| `src/index.css:2056-2060` | `.dept-tree-search .ant-input-affix-wrapper .ant-input { border:none; background:transparent; box-shadow:none } !important` | 部门树搜索框无双框 |
| `src/index.css:2077-2081` | `.user-form-input.ant-input-affix-wrapper .ant-input { border:none; background:transparent; box-shadow:none } !important` | 用户表单输入框无双框 |

**这三处就是"方案 A 的现成先例"，写法可直接复用。**

### (a.4) TextArea 是否受影响 —— 受影响

`AppTextArea` 默认 `allowClear=true`（`src/components/ui/AppTextArea.tsx:13`）→ rc TextArea 走 affix 分支 → wrapper 同时带 `ant-input-affix-wrapper`（有边框）+ 内层 `textarea.ant-input`（被 `index.css:875` 画边框）→ **TextArea 同样双框**，且项目没有针对 textarea 的任何局部补丁 → TextArea 100% 双框。

### (a.5) Select / Cascader / TreeSelect 是否受影响 —— **无双框，但规则整体失效**

- antd 6 Select 的边框画在 **`.ant-select`** 上（`antd/es/select/style/select-input.js:106-112`：`borderWidth/borderStyle/borderColor` 挂在 `[componentCls]` = `.ant-select`），项目从未给 `.ant-select` 加边框 → **无双框**。
- 但 `.ant-select-selector` 这个 class 在 antd 6 **已不存在**：
  - `antd/es/select/` 全目录 grep `selector` → 仅命中 token 命名 `selectorBg`（`style/token.js:47`）与注释；
  - `@rc-component/select/es/` grep `selector` → 仅 2 处注释（`BaseSelect/index.js:361`、`SelectInput/index.js:120`）；
  - antd 6 Select 样式改用 `.ant-select-content` / `.ant-select-value` / `.ant-select-input` / `.ant-select-prefix` / `.ant-select-suffix` / `.ant-select-clear`（`select-input.js:123,136,210,129,206`）。
- **结论**：项目 CSS 中下列 12 条规则是**死规则**（选择器永不命中）：
  `index.css:873`（`.ant-select-selector` 边框/背景/圆角）、`:884`（hover 边框）、`:890`（focus 边框 + 焦点环）、`:897`（暗色 focus）、`:2287`（暗色背景/文字/边框）、`:2293`（暗色 placeholder）、`:2297`（暗色选中项文字）、`:3920`（暗色 selector）、`:3926`（暗色 hover）、`:3930`（暗色 focus）、`:5537`（分页 options 字体）。
  → **暗色模式 Select 外观完全由 antd token 决定** = `AntdThemeBridge` 写死的 `cream.surface = #FFFFFF` → 暗色 Select 很可能白底（**需实机确认**，见 §(f)）。
- `.ant-select-item` / `.ant-select-item-option-selected` 在 antd 6 **仍存在**（`@rc-component/select/es/OptionList.js:54,338`、`antd/es/select/style/dropdown.js:27`）→ 下拉列表的暗色覆盖有效。

### (a.6) InputNumber / DatePicker / Checkbox / Radio 是否双框 —— **否**

| 组件 | 内层元素 | 内层边框 | 判定 |
|------|---------|---------|------|
| InputNumber | `input.ant-input-number-input` | `border: 0`（`antd/es/input-number/style/index.js:114-121`）且 class 不匹配 `.ant-input` | 无双框；边框在 `.ant-input-number-outlined`（`input-number/style/index.js:46-57` 复用 `genOutlinedStyle`） |
| DatePicker | `input`（`.ant-picker-input > input`，无 `ant-input` class） | 项目 `[data-color-mode="dark"] input`（`index.css:2252`）只设 `background-color/color/border-color`，**不设 border-width/style** → 无边框生成 | 无双框；项目 `:2326` 已显式把 picker 内 input 背景改回 transparent |
| Checkbox / Radio | `.ant-checkbox-inner` / `.ant-radio-inner` 本身即边框元素 | 项目只改 `background`/`border-color`（`index.css:1275-1299`） | 无双框；破坏 variant/状态色（F1 同类，P2） |

---

## (b) antd 6 `variant` 体系与项目手写规则的冲突面

antd 6 引入 **Selectors 之外的第二套变化维度 `variant`**（`outlined` / `borderless` / `filled` / `underlined`），默认 `outlined`：

| 事实 | 证据 |
|------|------|
| 默认 variant = `outlined` | `antd/es/form/hooks/useVariants.js:24` `mergedVariant = ctxVariant ?? configComponentVariant ?? configVariant ?? 'outlined'` |
| 全局开关存在 | `ConfigProvider` 支持 `variant` prop（`antd/es/config-provider/index.d.ts:25`）、`Form` 支持 `variant` 上下文（`VariantContext`） |
| 组件覆盖点 | Input（`antd/es/input/Input.js:129`）、TextArea（`TextArea.js:94`）、InputNumber（`input-number/index.js:83`）、Select（`select/index.js:104`）、Cascader（`cascader/index.js:169`）、TreeSelect（`tree-select/index.js:118`）、DatePicker/RangePicker（`generateSinglePicker.js:106`/`generateRangePicker.js:92`）、TimePicker（`time-picker/index.js:36`） |
| variant 样式实现 | `antd/es/input/style/variants.js`（`genOutlinedStyle:52-81`、`genBorderlessStyle:137-175`、`genFilledStyle:202-231`、`genUnderlinedStyle:307-345`），Select 另有变量驱动实现（`select/style/select-input.js:42-58,290-370`） |
| `bordered` 已 deprecated | `antd/es/input/Input.d.ts:64-70`（`bordered` → 用 `variant="borderless"`） |

**冲突面量化**：项目 `index.css` 用 `border: 1px solid … !important` **简写**统一覆盖边框，等于把四种 variant 全部压成 outlined 的形状与颜色 —— 但**压的方向不一致**：

- `borderless` 本应 `border: none` → 项目强加 1px 边框 → **borderless 变体失效**（凭空多一条框）；
- `underlined` 本应只有下边框（`variants.js:282` `borderWidth: '${unit(lineWidth)} 0'`）→ 项目简写把四边都设成 1px → **形状全错**；
- `filled` 本应透明边框 + `colorFillTertiary` 底 → 项目强加实色边框 → 失效；
- `outlined` 的 **status 态**（error/warning）边框色 → 被 important 压掉 → **失效**（F1）。

**当前业务侧影响面**：项目未对 Input/Select/DatePicker 传 `variant`（全仓 grep `variant=` 仅命中 Card/Tag 的 12 处），所以**现状只有"永远 outlined"这一条路径生效**；但 antd 内部组件（Table 筛选器、Transfer 搜索、Modal 内组件）与未来新代码会用到 variant → 属于**技术债型 P2**，不是当前 P0。

---

## (c) 全局冲突面盘点

严重度定义：**P0** = 用户实机可见的错误视觉，本次必须修；**P1** = 用户可见但非本次上报范围（应在 v1.24 内一并登记/修复）；**P2** = 技术债/潜在风险（记录 + 排期）。

### C-1 输入类（本次 P0 主战场）

| # | 项目规则（文件:行号） | 冲突对象 | 用户可见影响 | 严重度 |
|---|----------------------|---------|------------|--------|
| 1 | `index.css:875` `.ant-input { border: 1px solid var(--input-border) !important }` | antd `> input.ant-input { border:none }`（`input/style/index.js:384`） | **双框**：外层浅灰框内嵌白底圆角输入框 | **P0** |
| 2 | `index.css:874` `.ant-input { background: var(--input-bg) !important }` | antd `background:'transparent'`（同 :387） | 内层 input 底色不透明 → 与 wrapper 底色割裂；暗色下更明显 | P0 |
| 3 | `index.css:877` `.ant-input { border-radius: var(--theme-radius-md) !important }` | antd `borderRadius: 0`（同 :385） | 内层出现 8px 圆角矩形（antd 设计为直角） | P0 |
| 4 | `index.css:871` 命中 `textarea.ant-input` | antd `textarea.js:54` `border:'none'` | **TextArea 同样双框**（AppTextArea 默认 allowClear） | **P0** |
| 5 | `index.css:879` `transition: all … !important` | antd 内层无 transition | 内层 input 跟随全部属性动画（含宽度），轻微抖动 | P2 |
| 6 | `index.css:878` `font-size: 14px !important` | antd `fontSize:'inherit'`（`input/style/index.js:383`） | `size="large"` 时内层文字不放大 | P2 |
| 7 | `index.css:891` `.ant-input:focus { border-color: var(--input-border-focus) !important }` | 命中内层 input | 聚焦时**内层边框也变绿**（双框的第二重症状） | P1 |
| 8 | `index.css:892` `.ant-input:focus { box-shadow: 0 0 0 2px … !important }` | antd `> input.ant-input:focus { box-shadow:none !important }`（`input/style/index.js:392-394`，特异性 (0,3,0) > 项目 (0,2,0)） | **当前未双焦点环**（antd 特异性更高胜出）→ 修复时**不可删掉这条**，否则内层焦点环会冒出来 | P1（陷阱项） |
| 9 | `index.css:2252-2260` 暗色 `[data-color-mode="dark"] input, textarea, .ant-input … { background-color/color/border-color !important }` | 命中 affix-wrapper 内层 input/textarea | 暗色下内层被刷成 `--input-bg` 不透明底 + 边框色 | P0 |
| 10 | `index.css:2275-2284` 暗色 `input:focus { box-shadow: 0 0 0 2px … !important }` | 同 #8（antd 特异性更高） | 暗色下内层焦点环目前被压制；**修复时不可放开** | P1（陷阱项） |
| 11 | `index.css:3717-3734` 暗色第二套 `.ant-input` / `:hover` / `:focus` | 同 #1/#7 | 同上，且与 2252 段**重复定义**（双份真相源） | P1 |
| 12 | `index.css:875` + antd `genOutlinedStatusStyle`（`variants.js:41-51,63-78`） | `.ant-input-outlined.ant-input-status-error` 的 `borderColor`（非 important） | **Form 校验错误时输入框边框不变红**（F1） | P1 |
| 13 | `index.css:910` `.ant-input-affix-wrapper { border: 1px solid … !important }` | antd `.ant-input-outlined` 边框体系（`variants.js:52-81`） | 边框色/形状被项目锁死，variant/status 失效；**这是 wrapper 的唯一边框来源，修复时必须保留** | P1 |
| 14 | `index.css:916-923` wrapper 的 `:hover` / `-focused` 边框与焦点环 | antd `&:hover` / `&:focus-within`（`variants.js:30-39`） | 焦点环被项目接管（视觉与 Input 一致）→ **必须保留** | P1（不可误删） |
| 15 | `index.css:905` `.ant-input::placeholder` | 内层 input 也命中 | 占位符色在内层生效（视觉一致，无害） | P2 |
| 16 | `index.css:874-876` 命中 `.ant-input-number`（同一规则块第 2 个选择器） | antd `.ant-input-number-outlined` | InputNumber 边框被项目锁死 → status 失效；**无双框** | P1 |
| 17 | `index.css:873` `.ant-select-selector` | antd 6 **无此 class** | **死规则**（F2）：Select 的边框/底色/圆角/hover/focus 全部不由项目接管 | P1 |
| 18 | `index.css:2287-2299`、`:3920-3936` 暗色 `.ant-select-selector` / `-selection-item` / `-selection-placeholder` | 同上 | **死规则** → 暗色 Select 走 antd token = `cream.surface #FFFFFF` → 暗色白底风险 | P1 |
| 19 | `index.css:2253` `[data-color-mode="dark"] textarea` | TextArea 内层 textarea | 暗色 TextArea 内层不透明底 | P0 |
| 20 | `index.css:2326-2329` `.ant-picker-input > input { background-color: transparent !important }` | — | **正面案例**：这是"给 antd 6 正确收口"的写法样板 | P2（可作模板） |
| 21 | `index.css:4234-4248` 暗色 `.ant-picker` / `:hover` / `-focused` | antd picker 变量驱动的边框色 | DatePicker 边框色被项目锁死 → borderless 变体失效；status 失效；无双框 | P2 |
| 22 | `index.css:2036/2046` `.cron-selector-tabs .ant-space/.ant-select` | — | 仅布局约束，无冲突 | — |
| 23 | `index.css:5515-5523` `.ant-pagination .ant-pagination-item { border:none !important }` | antd 6 分页项边框 | 分页项边框被抹掉（项目刻意设计），非缺陷但属强制覆盖 | P2 |
| 24 | `index.css:5537` `.ant-pagination-options .ant-select-selector` | 死 class | 字体声明失效 | P2 |
| 25 | `src/pages/login/login.css:340-358` | 同 #1/#2/#3 | 登录页因局部补丁**当前无双框**；若全局修复时误删 login.css 的补丁 → 登录页回归双框 | P1（回归陷阱） |
| 26 | `src/index.css:2051-2088`（`.dept-tree-search` / `.user-form-input`） | 同 #1 | 局部补丁在全局修复后变冗余（无害），但**不能删**，否则那两处回归双框 | P1（回归陷阱） |

### C-2 非输入类结构冲突（记录 + 排期，不属本次 P0）

| # | 项目规则 | 冲突对象 | 用户可见影响 | 严重度 |
|---|---------|---------|------------|--------|
| 27 | `index.css:942-951` `.ant-table-thead > tr > th { border-bottom:2px; border-top:1px; padding:14px 16px; text-align:left } !important` | antd 6 Table 的 `bordered` / `variant` / sticky header 列宽对齐 | Table 的 bordered 变体、排序列对齐被强制 | P2 |
| 28 | `index.css:1087-1105` / `:2409-2431` `.ant-modal-content { background/border/border-radius/box-shadow !important }` | antd 6 Modal 的 variant（无边框形态） | Modal 无边框变体失效，统一被加 1px 边 | P2 |
| 29 | `index.css:1140-1151` / `:2446-2460` `.ant-dropdown-menu { background/border !important }` | antd 6 Dropdown variant | Dropdown 无边框变体失效 | P2 |
| 30 | `index.css:1275-1284` `.ant-radio-checked .ant-radio-inner` / `.ant-checkbox-checked .ant-checkbox-inner` | antd 6 Checkbox/Radio 的 variant 与状态色 | 选中态颜色被锁死，antd 状态色（如 error 态勾选）失效 | P2 |
| 31 | `index.css:852-855` `.ant-btn[disabled] { opacity:0.5 !important }` | antd 6 Button 的 disabled token（`colorTextDisabled`/`colorBgContainerDisabled`） | 禁用按钮用 opacity 压暗而非 token 配色，暗色模式下对比度可能不足 | P2 |
| 32 | `index.css:1287-1293` 暗色 `.ant-radio-inner/.ant-checkbox-inner` | 同 #30 | 同上 | P2 |
| 33 | `index.css:1249-1264` `.ant-badge-count` / `index.css:1522-1525` `.ant-btn` 等 | antd 6 组件 token | 颜色锁死，token 体系对 Badge/Button 部分失效 | P2 |
| 34 | 全文件 **1346 处 `!important`**（grep `!important` 计数） | antd 6 全部 CSS-in-JS token 体系 | 项目手写层事实上**接管了 antd 6 的组件样式层**，`AntdThemeBridge` 的 `theme.components` token 对大部分组件形同虚设 | P2（架构性） |

---

## (d) 三方案对比与推荐

### (d.1) 方案 A —— 最小覆盖（**推荐**）

```css
/* index.css：紧邻 908 行的 affix-wrapper 规则块追加 */
.ant-input-affix-wrapper > .ant-input,
.ant-input-textarea-affix-wrapper > .ant-input {
  border: none !important;
  background: transparent !important;
  box-shadow: none !important;
}
```

| 维度 | 评估 |
|------|------|
| 改动文件/行数 | `xingran-react-frontend/src/index.css`，**+1 个规则块（≈6 行）**，不动任何 TS/TSX |
| 是否修复双框 | ✅ 是（同时覆盖 Input 与 TextArea；`ant-input-textarea-affix-wrapper` 同时带 `ant-input-affix-wrapper`，第一条已命中，第二条为冗余保险） |
| 视觉回归风险 | **低**。新增规则只作用于"affix-wrapper 内层 input"，这些元素当前唯一被覆盖的样式就是 `border/background/border-radius`，恢复为 antd 原生（无边框、transparent、直角）后，视觉 = antd 设计意图。**不动 wrapper 边框、不动 focus 规则 → 焦点环不丢** |
| 需回归页面 | 登录页、系统用户/角色/字典、运维楼宇/楼层/工位、工单、值班、知识库、监控日志、AD 域控账号、设置页 —— 各截 **明/暗** 两态 |
| 暗色模式影响 | 正面：内层不再被 `index.css:2252-2260`/`:3717-3744` 的 `background-color: var(--input-bg) !important` 染成不透明底；**但注意 `#9/#11` 两条暗色规则仍会给内层 input 上 `border-color`** → 因 `border-style` 已为 none 而无可见边框（`border-color` 单独不生成边框），实测需确认 |
| 与 v1.23 冻结契约 | **零冲突**：不动 `AppInput`/`AppTextArea`/`AppSelect` 契约组件、不动 JSX、不动 `ruoyiBrand` token、不动 hex-exemptions 登记制（CSS 不在 eslint 面内，见 `68-UI-SPEC.md:137`） |
| 残留问题 | 不解决 F1（错误态不变红）、F2（Select 死规则）、F3（暗色白底底色）→ 需另立条目 |

### (d.2) 方案 B —— 把 `.ant-input` 规则收窄为「非 affix-wrapper 内层」

> **前提修正（重要）**：任务描述中的「CSS `:not()` 不支持组合子选择器」在现代标准下**不成立**。Selectors Level 4 的 `:not()` 接受 *complex selector list*（含后代/子代组合子），浏览器支持：Chrome/Edge 88+、Firefox 84+、Safari 16.4+。项目无 `browserslist` 约束（`package.json` 无该字段，Vite 默认现代 target），生产为企业内网 Chrome/Edge → **可用**；若需兼容 Safari 16.4 以下则不可用（须 Phase 1 确认内网浏览器基线）。

**实现方式（三选一）：**

| 实现 | 写法 | 评价 |
|------|------|------|
| B-1 直接子代 + `:not()` | `.ant-input:not(.ant-input-affix-wrapper > .ant-input) { …边框职责… }` | 语义最准，但 **`:not()` 内的选择器参与特异性计算** → 该规则特异性从 (0,1,0) 升到 (0,2,0)，会盖住 antd 的 status 规则（(0,3,0) 仍胜，但会盖住别的 (0,1,0)/(0,2,0) 规则）→ 需连带复核 |
| B-2 `:where()` 降特异性 | `:where(.ant-input):not(.ant-input-affix-wrapper > .ant-input) { … }` | 特异性 (0,0,0)（`:not()` 内仍是 :where → 0）→ 语义准且不抢戏，但**项目原规则的 `!important` 全部要摘掉**，否则 `:where()` 降特异性对 `!important` 无效（important 声明不看 `:where()` 的降级？不 —— `:where()` 确实把特异性降到 0，但 `!important` 内部比较时同样看特异性；摘掉 important 后 antd 的 token 体系才会回来）→ **改动量骤增** |
| B-3 **职责拆分（推荐实现）** | 把 `index.css:871-880` 拆成两组：<br>①「边框职责」→ 只挂真正的边框元素：`.ant-input`（顶层）+ `.ant-input-affix-wrapper` + `.ant-input-number`；<br>②「内容职责」（color / font-size / placeholder / transition）→ 用 `:where()` 低特异性保留给所有 input（含内层） | 语义最清晰：**边框永远由外层画，内容永远由内层继承**，从根上杜绝"再有人给内层加边框" |

| 维度 | 评估 |
|------|------|
| 改动文件/行数 | `src/index.css`：871-880 拆分重写（**≈ +18 / −10 行**）；若顺带清理 2051-2088 两处局部补丁（**−12 行**，全局修好后冗余）→ 合计约 ±40 行，**仍只动 1 个文件** |
| 是否修复双框 | ✅ 是（B-3 的「边框职责」不再落到内层；等价于 A 但更根本） |
| 视觉回归风险 | **中**。摘 `!important` / 换 `:where()` 会让 antd token 重新接管 → 边框色从 `--input-border` 变回 `token.colorBorder`（AntdThemeBridge:86 = `cream.border #DBD7CE` —— 亮色恰好同值 ✅）；但**暗色模式**会回到 `cream.border`（浅米色）→ 与暗色背景冲突 → **必须同步处理 F3**（否则暗色回归） |
| 需回归页面 | 同方案 A，且**暗色必查**（暗色是 B 的主要风险面） |
| 暗色模式影响 | ⚠️ **有风险**：一旦摘掉 `!important`，暗色下 antd token 的浅色 `cream.*` 会重新生效。B 必须与「AntdThemeBridge 按 mode 切色板」或「保留暗色段的 `!important`」二选一 |
| 与 v1.23 冻结契约 | 不碰 TS 侧契约组件 ✅；但若顺手动 `AntdThemeBridge` 的 token 来源 → 触碰「TS 侧唯一色值真相源 = ruoyiBrand」契约 → 需按 v1.22 冻结契约流程解冻 |

**裁决**：B **不能单独作为 P0 修复**（它必须先解决 F3 暗色底色问题，改动面反而大于 A）。建议：**A 先止血，B(B-3 职责拆分) 作为 v1.24 的第二个 Phase**，且第二 Phase 需连带修 F3。

### (d.3) 方案 C —— 弃用手写边框，统一走 antd 6 `variant` + `ConfigProvider` theme token

| 维度 | 评估 |
|------|------|
| 基础设施 | **已存在**：`src/design-system/components/AntdThemeBridge.tsx` 已通过 `ConfigProvider theme` 配了 `Input`（`:121-128`：`colorBgContainer/activeBorderColor/hoverBorderColor/activeShadow/borderRadius:8/controlHeight:36`）与 `Select`（`:129-136`）；`ConfigProvider variant` 全局开关在 `antd/es/config-provider/index.d.ts:25` 可用 |
| 改动文件/行数 | ①`AntdThemeBridge.tsx`：**必须改为按 `themeMode` 选色板**（否则暗色白底，见 F3，≈ +40/−20 行）；②`index.css`：删除/降级 **871-929、2252-2299、3717-3769、3920-3936** 四批（**≈ −120 行**）；③可能新增 `design-system/tokens/` 明暗双套色板文件 |
| 是否修复双框 | ✅ 是（手写边框规则消失后无冲突源） |
| 视觉回归风险 | **高**。回归面 = 全站所有表单控件 + 全部弹层 + 明暗两态；且 antd 6 的 `Input` 边框色由静态 `borderColor`（`variants.js:29`）驱动、`Select` 由 CSS 变量 `--ant-select-border-color` 驱动（`select-input.js:21`），两条链路的 token 名不同 → 需要分别验证 `token.colorBorder` / `components.Input.colorBorder` 的落点 |
| 暗色模式影响 | ⚠️ **这是方案 C 的成败点**：当前 `AntdThemeBridge:83` 写死 `colorBgContainer: ruoyiBrand.cream.surface (#FFFFFF)`，darkAlgorithm **不会**改写显式 token → 暗色下 Input/Select/Modal 全白底。方案 C 必须先修 AntdThemeBridge 的明暗色板，否则暗色全面回归 |
| 与 v1.23 冻结契约 | ❌ **触碰冻结契约**：`ruoyiBrand`（`design-system/tokens/colors.ts`）是 v1.22 起的「TS 侧唯一色值真相源」；重写 AntdThemeBridge 的 token 映射 = 改这个契约的落地方式。需要走「解冻 → 新里程碑」流程，不适合 P0 止血 |
| 落地形态建议 | 独立里程碑「样式层收敛」：把 index.css 中 antd 相关 `!important` 从 1346 处降到「只保留业务语义色」，antd 组件外观 100% 由 token 驱动 |

### (d.4) 推荐

> **v1.24 采用方案 A 修 P0（双框），并在同一里程碑的第二个 Phase 追加「方案 B-3（职责拆分，仅限 Input 域，不动 AntdThemeBridge）」；方案 C 推迟为 v1.25 独立里程碑「样式层收敛」。**

理由：
1. **P0 的本质是一处 CSS 覆盖**，A 用 6 行、0 TS 改动、0 契约影响止血；B/C 都把 P0 修复捆绑到架构改造上，与「先止血、再治理」相悖。
2. **A 不可能引入新的双焦点环**：antd 自身用 `> input.ant-input:focus { box-shadow: none !important }`（`input/style/index.js:392-394`，特异性 (0,3,0)）压制内层焦点环，A 的新规则特异性 (0,2,0)+important 也不改 box-shadow 语义 → 焦点环仍只在 wrapper 上（`index.css:920-923`）。
3. **B-3 值得做**，因为它把「边框由谁画」这一职责固化进 CSS 结构，杜绝复发；但它必须与 F3（暗色色板）解耦——**在不动 AntdThemeBridge 的前提下只做 Input 域的职责拆分**，保留暗色段的 `!important`。
4. **C 是正解但不是此刻的解**。它要求先解冻 `ruoyiBrand` 契约、修正 AntdThemeBridge 的明暗色板、并对全站 1346 处 `!important` 做系统清理——这是一个里程碑量级的工程，与 P0 止血混在一起会让回归无法归因。

---

## (e) 视觉回归验证方法（最小手段）

### (e.1) 主手段：**Computed-Style 断言脚本**（比截图更精确，可脚本化）

在任意页面 DevTools Console 粘贴（或用 agent-browser skill 自动化执行）：

```js
// ① 双框断言：affix-wrapper 内层 input 必须无边框
const inner = [...document.querySelectorAll('.ant-input-affix-wrapper > .ant-input, .ant-input-textarea-affix-wrapper > .ant-input')];
const bad = inner.filter(el => getComputedStyle(el).borderTopWidth !== '0px');
console.log('双框元素数 =', bad.length, bad.map(e => e.outerHTML.slice(0, 80)));

// ② 焦点环断言：焦点环只在外层
const w = document.querySelector('.ant-input-affix-wrapper');
w.focus?.();
const ws = getComputedStyle(w);
console.log('wrapper box-shadow =', ws.boxShadow);            // 应非 none
console.log('inner box-shadow  =', getComputedStyle(w.querySelector('input.ant-input')).boxShadow); // 应为 none

// ③ 状态色断言：错误态边框应变红（AntD token colorError）
//    在 Form 提交校验失败后执行：getComputedStyle(el).borderTopColor
```

**断言口径**：
- 双框数必须 **= 0**（当前应 > 0，可先跑一次取基线）；
- `inner box-shadow === 'none'`（内层不得有焦点环）；
- `wrapper box-shadow !== 'none'`（外层焦点环不得丢）；
- 错误态 `borderTopColor` 应等于 antd `colorError`（`#BA3630` 量级），**不是** `--input-border`（F1 的验收口径）。

**全站扫描**：上述脚本在任何页面都能一次性覆盖"该页所有输入框"，比逐页截图省 10 倍工作量。把它固化成 dev-only 脚本（可选，Phase 1 决定是否入库）。

### (e.2) 辅手段：截图对照（只截必要的）

**必须截图的输入框类型矩阵**（每种 × {默认, focus, 错误态} × {亮色, 暗色}）：

| # | 类型 | 关键构造 | 期望（修复后） | 风险点 |
|---|------|---------|--------------|--------|
| 1 | 普通 Input | `<Input />`（allowClear=false） | 单框，focus 绿环 | 最基线 |
| 2 | **allowClear Input** | `<AppInput />`（默认 allowClear=true） | **单框**（当前双框） | P0 主场景 |
| 3 | prefix Input | `<Input prefix={<UserOutlined/>} />` | 单框，prefix 与边框同高 | prefix 挤压内层高度 |
| 4 | suffix Input | `<Input suffix="元" />` | 单框 | 同上 |
| 5 | prefix+suffix | 同时 | 单框 | 内层 `padding:0` 与 suffix 间距 |
| 6 | password | `<Input.Password />` | 单框 + 眼睛图标不叠框 | 密码框通常是双框高发点 |
| 7 | search | `<Input.Search />` / antd `Search` | 单框 + 搜索按钮 | antd 6 Search 单独一套 variant（`antd/es/input/Search.js:52-54`） |
| 8 | readonly | `readOnly` | 单框，底色不变 | antd readonly 无专门边框态 |
| 9 | disabled | `disabled` | 单框，底/边色变灰 | `index.css:3736` 暗色 disabled 段 |
| 10 | status=error | `status="error"` | **边框变红**（F1 验收） | 项目 important 压制 |
| 11 | status=warning | `status="warning"` | 边框变黄 | 同上 |
| 12 | TextArea（无 clear） | `<Input.TextArea />` | 单框 | `textarea.ant-input` 命中同规则 |
| 13 | TextArea（allowClear） | `<AppTextArea />` | **单框** | P0 |
| 14 | TextArea + showCount | `showCount` | 单框 + 计数不叠框 | `ant-input-textarea-affix-wrapper` |
| 15 | Select（单选） | `<AppSelect />` | 单框，focus 绿环 | `.ant-select` 边框（非 selector） |
| 16 | Select（多选） | `mode="multiple"` | 单框，tag 不溢出 | multiple 变体样式不同 |
| 17 | Select（带搜索） | `showSearch` | 单框 | |
| 18 | Select status=error | | 边框变红 | antd 变量驱动，项目未干预 |
| 19 | DatePicker | `<DatePicker />` | 单框 | `.ant-picker` 边框 |
| 20 | InputNumber | `<InputNumber />` | 单框 + 步进器不叠框 | 内层 `ant-input-number-input` 不匹配 `.ant-input` |
| 21 | Cascader | `<Cascader />` | 单框 | 复用 Select 边框 |
| 22 | TreeSelect | `<TreeSelect />` | 单框 | 同上 |
| 23 | Modal 内表单 | `<FormModal>` | 弹层内单框 | 弹层内 antd 组件脱离页面 CSS 作用域的情况 |
| 24 | 部门树搜索框 | `.dept-tree-search` | 单框（回归陷阱 #26） | 局部补丁不得被误删 |
| 25 | 用户表单输入框 | `.user-form-input` | 单框（回归陷阱 #26） | 同上 |
| 26 | 登录页 | `/login` | 单框（回归陷阱 #25） | `login.css:351` 局部补丁不得被误删 |

**回归页面清单（明+暗两态截图）**：
`/login` → `/system/user` → `/system/role` → `/system/dict` → `/system/dept` → `/operations/building` → `/operations/floor` → `/operations/workstation` → `/workorder/orders` → `/duty/management` → `/knowledge/articles` → `/monitor/job` → `/ad-domain/accounts` → `/settings`（Phase 70 双层纸感）。
每页至少覆盖：列表搜索表单 + 新建/编辑弹窗 + 表格内 inline 编辑。

### (e.3) 暗色模式验证

- 切换入口：设置页主题切换（或 `document.documentElement.setAttribute('data-color-mode','dark')`，项目暗色选择器为 `[data-color-mode="dark"]`，见 `index.css:608` 等 200+ 处）；
- 暗色必查三项：**内层 input 底色**（不得出现比 wrapper 更亮的块）、**焦点环**（`box-shadow` 仍存在）、**disabled 态**（`index.css:3736-3740`）。
- 暗色额外必查 **Select**：`.ant-select-selector` 是死规则 → 暗色 Select 是否白底（F2 验证点），若确认为白底 → **另开条目**，不要混进本次 P0。

### (e.4) 不引入的东西

不引入 Playwright / Cypress / 视觉快照库（v1.23 已确立：agent-browser skill 覆盖实机 UAT）。断言脚本走 DevTools Console / agent-browser 即可。

---

## Installation

```bash
# 前端：零安装
# 本次为纯 CSS 研究与修复，不新增任何依赖
```

---

## Alternatives Considered

| Recommended | Alternative | 何时用 Alternative |
|-------------|-------------|-------------------|
| 方案 A（最小覆盖） | 方案 B-3（职责拆分） | 当需要**根治**「谁画边框」并愿意接受暗色回归风险时；建议作为 A 之后的第二 Phase，且不动 AntdThemeBridge |
| 方案 A | 方案 C（ConfigProvider token 统一） | 当里程碑可以承载全站样式层收敛 + 解冻 `ruoyiBrand` 契约 + 重建明暗色板时（建议 v1.25） |
| 方案 A | 改 `AppInput` 默认 `allowClear=false` | **不可接受**：allowClear 是 v1.23 已冻结的交互契约（`AppInput.tsx:11-14`），回退等于撤销契约 |
| 方案 A | 给 antd 提 issue / 升级 antd | 无效：冲突源是项目 CSS 的 `!important`，antd 侧行为符合设计 |
| 方案 A | 删掉 `index.css:875` 的 `!important`（保留声明） | **不可行**：antd 的 `border:none`（非 important）会赢 → 内层边框消失，但项目对**裸** Input 的边框也会一起消失 → 出现「无框 Input」，破坏现有视觉 |
| 方案 A | 只改 `index.css:2252-2260` 暗色段 | 不修亮色模式（用户截图是亮色），且暗色段不是双框主因 |

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| 给内层 input 加 `border: none` 之外的重置（如 `padding:0; height:auto`）到全局 | antd 的 `> input.ant-input` 已经用 `padding:0`（`input/style/index.js:380`）管理内层几何；全局重置会破坏 `size="small"/"large"` 的高度计算 | 只覆盖 `border/background/box-shadow` 三项（对齐 `login.css:351-358` 的最小集合） |
| 用 `!important` 修 antd 的 `!important` | 会陷入特异性军备竞赛；antd 已在 `input/style/index.js:392-394` 用 `box-shadow:none !important` 做过压制 | 认清「边框画在哪一层」，让内层根本不画边框 |
| 在修复里删掉 `index.css:891-893` / `:920-923` 的 focus 规则 | 删掉后 wrapper 的焦点环会丢失（`:920-923` 是 wrapper 焦点环唯一来源）；若同时删 `:892` 的内层 box-shadow 而没加内层压制，会冒出**双焦点环** | 保留全部 focus 规则，只新增内层 `border:none` |
| 顺手删除 `login.css:351-358`、`index.css:2051-2088` 局部补丁 | 它们是当前唯一让登录页/部门树/用户表单无双框的原因；全局修复验证通过前删除 = 制造回归 | 全局修复验证通过后再作为「清理项」单独提交 |
| 用 `:not()` 收窄时同时保留 `!important` | `:not()` 内选择器参与特异性计算，会把规则抬到 (0,2,0)，盖住 antd 的 status 规则（更糟） | 拆职责 + `:where()` 降特异性 + 摘 `!important`（方案 B-3），或直接用子代组合器（方案 A） |
| 在 CSS 里继续加裸 hex（如 `login.css` 的 `#e7e5e4`/`#d4a574`） | hex-exemptions 登记制只覆盖 JS/TS（`68-UI-SPEC.md:137` 明确 CSS 不在 eslint 面内），CSS 裸 hex 无机器守卫 | 用 `var(--input-*)` / `ruoyiBrand` 单一真相源；新增 TS 色值必须进 `ruoyiBrand` |
| 在本次 P0 里顺手做方案 C | 全站 1346 处 `!important` + 明暗色板重建，回归无法归因 | 拆成 v1.25 独立里程碑 |

---

## Stack Patterns by Variant

**如果 Phase 1 采用方案 A**：新规则必须放在 `index.css` 的 Input 段落（紧邻 `:908` 的 `.ant-input-affix-wrapper` 块），且**必须**同时写 `border: none !important` + `background: transparent !important` + `box-shadow: none !important` 三项——只写 `border` 会留下暗色下内层不透明底（`index.css:2257` 的 `background-color: var(--input-bg) !important` 特异性 (0,1,1) vs 新规则 (0,2,0)+important → 新规则胜出，但若漏写 background 就会露底）。

**如果采用方案 B-3（Input 域职责拆分）**：拆分后 CSS 结构固定为三层——
① 边框层：`.ant-input`（顶层）/ `.ant-input-affix-wrapper` / `.ant-input-number`；
② 内容层：`:where(.ant-input)` 携带 color/font-size/placeholder/transition；
③ 状态层：`:where(.ant-input:focus)` / `:where(.ant-input:hover)` 只改 border-color/box-shadow。
任何新组件接入时按这三层填，**禁止**再出现「给 `.ant-input` 加边框」的写法。

**如果采用方案 C（未来里程碑）**：`AntdThemeBridge` 必须先拆成 `lightColorBoard` / `darkColorBoard` 两套映射，再让 `ConfigProvider` 按 `themeMode` 选板；`ConfigProvider variant="outlined"` 可作为全局默认（与 antd 默认一致，仅作显式声明，避免未来 ConfigProvider variant 被误改）。

---

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| antd 6.6.4 | `@rc-component/input`（BaseInput/TextArea 的 affix 分支） | affix 分支条件 `prefix||suffix||allowClear||showCount`（`antd/es/input/utils.js:1-3`）是 v1.23 契约化的技术前提 |
| antd 6.6.4 | antd 6.6.4 的 `variant` 体系 | `outlined/borderless/filled/underlined`（`antd/es/input/style/variants.js`），默认 `outlined`（`form/hooks/useVariants.js:24`） |
| CSS `:not()` 复杂选择器 | Chrome/Edge 88+、Firefox 84+、Safari 16.4+ | 仅方案 B 需要；项目无 browserslist → Vite 默认现代 target ✅；若内网有旧 Safari 则方案 B 不可用（方案 A 不受影响） |
| `index.css` `:871-929` | antd 6.6.4 | 该批规则**按 antd 5 的 class 命名书写**（`.ant-select-selector`/`.ant-select-selection-item`），antd 6 已改名为 `.ant-select-content`/`.ant-select-value` → 死规则 |
| `AppInput`/`AppTextArea` 契约 | antd 6.6.4 | `allowClear` 默认值来自 v1.23 冻结契约，修复方案不得改动契约组件本身 |

---

## (f) 是否需要 Phase 1 之外的额外调查

**Phase 1 内必须实测确认（本次研究为静态代码分析，无法替代）：**

| # | 待确认项 | 方法 | 影响 |
|---|---------|------|------|
| V1 | 修复前/后全站"双框元素数"基线 | §(e.1) 断言脚本 | 验收主口径 |
| V2 | antd 6 的 `status` class 落在 root 还是 wrapper（决定 F1 的修复点：`:888` 的 `.ant-input:focus` / `:908` 的 `.ant-input-affix-wrapper`） | DevTools 查触发校验失败时的 class 列表 | F1 修复定位 |
| V3 | 暗色模式下 `.ant-select` 的实际背景色（验证 F2/F3 的"白底"推断） | 暗色下读 `getComputedStyle(document.querySelector('.ant-select')).backgroundColor` | 若确为白底 → **另开条目**，不混 P0 |
| V4 | `index.css:2257` 的暗色 `background-color: var(--input-bg) !important` 在内层 input 上是否仍生效（修复后需确认无残留底色差） | computed style | 决定方案 A 是否要写 `background: transparent !important` |

**建议在 Phase 1 之外单独排期的调查（不阻塞 P0）：**

| # | 调查项 | 理由 |
|---|--------|------|
| X1 | **全站 `!important` 债务盘点**（本文件 C-2 已列 34 条结构性冲突，完整清单需按 §(c) 展开到组件级） | 架构性 P2，属 v1.25「样式层收敛」范围 |
| X2 | **`AntdThemeBridge` 明暗色板重构**（F3） | 方案 C 的前置，且当前 `cream.*` 固定值是暗色下多处手写 `!important` 补偿的**根因** |
| X3 | **antd 6 Select class 重命名导致的死规则全量清点**（F2） | 项目 CSS 与 antd 5/6 的 class 命名断层不止 `.ant-select-selector`，需系统比对（`antd 5 → 6` changelog + 全量 class diff） |
| X4 | **`.ant-select-selection-item` → antd 6 新类名**的映射与项目用色迁移 | X3 的一部分，但独立可查 |

---

## Sources

**项目源码（本仓）**
- `xingran-react-frontend/src/index.css` — 1346 处 `!important`；关键段：`:184-189`（亮色 `--input-*` 变量）、`:254-268`（暗色 `--input-*` 变量）、`:871-929`（Input 段）、`:1275-1299`（Radio/Checkbox）、`:2051-2088`（两处局部补丁）、`:2252-2299`（暗色段一）、`:3717-3769`（暗色段二）、`:3920-3936`（暗色 Select）、`:4234-4248`（暗色 Picker）、`:5515-5539`（分页）
- `xingran-react-frontend/src/components/ui/AppInput.tsx:11-14` / `AppTextArea.tsx:11-15` / `AppSelect.tsx:13-24` — v1.23 冻结契约组件
- `xingran-react-frontend/src/pages/login/login.css:340-370` — 方案 A 的现成先例
- `xingran-react-frontend/src/design-system/components/AntdThemeBridge.tsx:74-174` — `theme.token` / `theme.components` 映射（Input `:121-128`、Select `:129-136`）
- `xingran-react-frontend/src/design-system/tokens/colors.ts:72-89` — `ruoyiBrand.cream.*`（`surface: #FFFFFF`、`border: #DBD7CE`）
- `xingran-react-frontend/src/main.tsx:2` — `import "./index.css"`（CSS 层入口顺序）
- `.planning/milestones/v1.22-phases/68-p2/68-UI-SPEC.md:137` — 「CSS 文件不在 eslint 面内，CSS 裸 hex 以 hex-exemptions.md + review 收口」
- `.planning/milestones/v1.23-ROADMAP.md:92` — v1.22 冻结契约清单（含 AppInput·AppSelect 契约、hex-exemptions 登记制）

**antd 6.6.4（node_modules 实读）**
- `node_modules/antd/package.json:3` — 版本 6.6.4
- `node_modules/antd/es/input/style/index.js:361-395` — affix-wrapper 内层 input 的 `border:none` / `borderRadius:0` / `background:transparent` / `boxShadow:none !important`
- `node_modules/antd/es/input/style/textarea.js:44-61` — TextArea 内层 textarea 的 `border:none`
- `node_modules/antd/es/input/style/variants.js:25-81,137-175,202-231,307-345` — outlined/borderless/filled/underlined 四变体 + status 边框色
- `node_modules/antd/es/input/utils.js:1-3` — `hasPrefixSuffix` 含 `allowClear`
- `node_modules/antd/es/input/Input.js:129,142,152-176` — variant class 与 classNames 分发
- `node_modules/antd/es/input/TextArea.js:94,132-152` — TextArea 的 variant 与 affixWrapper class
- `node_modules/antd/es/form/hooks/useVariants.js:16-27` — 默认 `outlined` + 全局/组件级 variant 解析
- `node_modules/antd/es/config-provider/index.d.ts:25` — `ConfigProvider variant`
- `node_modules/antd/es/select/style/select-input.js:7-58,79-138,290-370` — Select 边框在 `.ant-select`，由 CSS 变量驱动；无 `-selector` 类
- `node_modules/antd/es/select/style/index.js:8-97` / `dropdown.js:27` — `.ant-select-item` 仍存在
- `node_modules/antd/es/input-number/style/index.js:46-60,114-132` — InputNumber 基类无边框、内层 `border:0`
- `node_modules/antd/es/date-picker/style/index.js:93-152` — Picker 边框/内层 input
- `node_modules/@rc-component/input/es/BaseInput.js:42-46,55-103` — affix 分支与 variant class 落点
- `node_modules/@rc-component/input/es/Input.js:151-171` — `<input>` 的 class 恒含 `ant-input`
- `node_modules/@rc-component/input/es/TextArea.js:137-189` — TextArea wrapper class 组装
- `node_modules/@rc-component/select/es/OptionList.js:54,338` — `.ant-select-item` / `.ant-select-item-option` 仍存在

**规范**
- CSS Cascade Level 4 §6.4.4 — `!important` 声明的优先级高于同源非 `!important` 声明（与特异性无关）
- Selectors Level 4 — `:not()` 接受 complex selector list（Chrome/Edge 88+、Firefox 84+、Safari 16.4+）；`:where()` 特异性恒为 0

---
*Stack research for: v1.24 样式层技术方案（antd 6 结构冲突 / 输入框双框 P0）*
*Researched: 2026-10-28*
