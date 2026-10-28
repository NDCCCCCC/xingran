# STACK — antd 6.6.4 样式层收敛定向研究

**Project:** RuoYi-Next 运维管理系统
**Milestone:** v1.25 全站权限治理与样式收敛（仅 ② 样式层收敛到 antd 6 variant 定向研究）
**Researched:** 2026-10-28
**Scope:** ② 定向研究（用户裁定，其余四条候选不做域外研究）
**Confidence:** 总体 HIGH（antd 6.6.4 源码实证）

---

> **方法论**：本机 `ruoyi-react-frontend/node_modules/antd` 版本 **6.6.4**（`node -e "require('./node_modules/antd/package.json').version"` 实测确认）。
> 所有 antd 内部事实均以 **本机源码 + 行号** 为最高权威，官方文档（ant.design）仅作辅证。
> 置信度标注：**HIGH** = 源码实证（附 `文件:行号`）／**MEDIUM** = 官方文档或源码间接推导／**LOW** = 社区信息，需验证。
> 未实际读到源码的行号一律写「未验证」，不做推测。

---

## 0. 关键前提修正（先于一切结论）

任务书给出的前置障碍 **F3** 描述为：

> `AntdThemeBridge.tsx` 第 83-84 行显式设了 `colorBgContainer: '#FFFFFF'` 和 `colorBorder: '#dbd7ce'`。
> 因为 antd 的 `darkAlgorithm` **不会改写显式传入的 token**，暗色模式下这些显式值原样保留。

逐条核对源码后，**F3 的「症状」成立，但「机制」不成立**。修正如下：

| # | F3 原文 | 源码核对结果 | 置信度 |
|---|---------|-------------|--------|
| F3-a | 文件路径 `src/components/theme/AntdThemeBridge.tsx` | **路径错误**。实际为 `ruoyi-react-frontend/src/design-system/components/AntdThemeBridge.tsx`（192 行）。`src/components/theme/` 目录不存在 | HIGH |
| F3-b | 「第 83-84 行」 | **行号偏差**。`colorBgContainer` 在 **:83**，`colorBorder` 在 **:86**（`:84` 是 `colorBgElevated`）。值不是字面量而是 `ruoyiBrand.cream.surface` / `ruoyiBrand.cream.border` = `#FFFFFF` / `#DBD7CE`（`src/design-system/tokens/colors.ts:76,84`） | HIGH |
| F3-c | 「`darkAlgorithm` **不会**改写显式传入的 token，显式值原样保留」 | **不成立**。`colorBgContainer` / `colorBorder` 在 antd 6.6.4 中是**由 `colorBgBase` 派生**的 Map Token，算法结果**覆盖** seed。见下文 A6 | HIGH |
| F3-d | 暗色下 Select 容器仍是白底 + 浅灰边 | **成立，但归因不同**。真正的元凶是 `components.Input/Select/Card` 的**组件级** `colorBgContainer: cream.surface`（组件 token 在算法之后 merge，算法改不动）+ `colorBgBase: cream.canvas` 浅奶油导致算法派生值在暗色下仍属浅色系 | HIGH |

> **对里程碑的直接含义**：修 F3 不是「把硬编码换成算法派生」这一件事，而是**两层同时改**：
> ① 全局 `theme.token` 里 `colorBgBase` / `colorTextBase` 才是算法入口（`colorBgContainer` / `colorBorder` 写了也白写）；
> ② `theme.components.*` 里的容器底色**必须按 mode 分支**，否则暗色永远回不到深色。

---

## A. antd 6.6.4 的 Input / Select / InputNumber 样式结构与 variant 机制

### A1. `variant` prop 的取值、默认值与类名生成

#### A1.1 取值集合（权威定义）

```js
// ruoyi-react-frontend/node_modules/antd/es/config-provider/context.js:4
export const Variants = ['outlined', 'borderless', 'filled', 'underlined'];
```

配套类型（`config-provider/context.d.ts:274-275`）：

```ts
export declare const Variants: readonly ["outlined", "borderless", "filled", "underlined"];
export type Variant = (typeof Variants)[number];
```

**结论：恰好 4 个取值** —— `outlined` / `borderless` / `filled` / `underlined`。任务书中列举的 `filled` 出现两次，实际没有第 5 个取值。

官方文档佐证（MEDIUM）：[Input 组件文档](https://ant.design/components/input/)「Variants of Input, there are four variants: outlined filled borderless and underlined」；[InputNumber 文档](https://ant-design.antgroup.com/components/input-number) 同款四变体。

#### A1.2 默认值与解析优先级

```js
// antd/es/form/hooks/useVariants.js:7-28
const useVariant = (component, variant, legacyBordered, fallbackComponent) => {
  const config = React.useContext(ConfigContext);
  const { variant: configVariant, [component]: componentConfig } = config;
  const ctxVariant = React.useContext(VariantContext);
  const fallbackComponentConfig = fallbackComponent ? config[fallbackComponent] : undefined;
  const configComponentVariant = componentConfig?.variant ?? fallbackComponentConfig?.variant;
  const isVariantConfigured = typeof variant !== 'undefined' || legacyBordered === false
    || typeof ctxVariant !== 'undefined' || typeof configComponentVariant !== 'undefined'
    || typeof configVariant !== 'undefined';
  let mergedVariant;
  if (typeof variant !== 'undefined') {
    mergedVariant = variant;                    // ① prop 最高
  } else if (legacyBordered === false) {
    mergedVariant = 'borderless';              // ② 旧 bordered={false} → borderless
  } else {
    // form variant > component global variant > fallback component global variant > global variant
    mergedVariant = ctxVariant ?? configComponentVariant ?? configVariant ?? 'outlined';  // ③④⑤ + 默认
  }
  const enableVariantCls = Variants.includes(mergedVariant);
  return [mergedVariant, enableVariantCls, isVariantConfigured];
};
```

| 优先级 | 来源 | 代码位置 | 说明 |
|---|---|---|---|
| ① | 组件 `variant` prop | `useVariants.js:18-19` | 单组件最高优先级 |
| ② | 旧 `bordered={false}` | `useVariants.js:20-21` | 兼容层，等价 `borderless` |
| ③ | `Form` 的 `variant`（`VariantContext`） | `useVariants.js:24` 前段 `ctxVariant` | **表单内全部输入控件统一** |
| ④ | `ConfigProvider` 组件级 `<Component>.variant` | `useVariants.js:15` | 如 `input={{ variant }}` / `select={{ variant }}` |
| ⑤ | `ConfigProvider` 全局 `variant` | `useVariants.js:24` 末段 `configVariant` | 全站统一 |
| — | **默认** | `useVariants.js:24` 末位 `?? 'outlined'` | **`outlined`** |

**默认 variant = `outlined`**（HIGH，`useVariants.js:24`）。

`ConfigProvider` 组件级 `variant` 的类型契约（`config-provider/context.d.ts`）：

| 行 | 配置键 | 可含 `variant` |
|---|---|---|
| 206 | `InputConfig` | ✅ |
| 207 | `InputPasswordConfig` | ✅ |
| 208 | `InputSearchConfig` | ✅ |
| 209 | `TextAreaConfig` | ✅ |
| 233 | `SelectConfig` | ✅ |
| 256 | `InputNumberConfig` | ✅ |
| 226 | `FormConfig` | ✅（经 `VariantContext` 广播给表单内控件） |
| 217 | `CardConfig` | ✅ |
| 263/264/265 | `DatePickerConfig` / `RangePickerConfig` / `TimePickerConfig` | ✅ |
| 257/261 | `CascaderConfig` / `TreeSelectConfig` | ✅ |
| 267 | `MentionsConfig` | ✅ |

#### A1.3 类名生成

variant 与 status 类**同槽位、同元素**，见 `antd/es/input/Input.js:159-161`：

```js
classNames: {
  ...
  variant: clsx({
    [`${prefixCls}-${variant}`]: enableVariantCls      // 例：ant-input-outlined
  }, getStatusClassNames(prefixCls, mergedStatus)),      // 例：ant-input-status-error
  ...
}
```

`getStatusClassNames`（`antd/es/_util/statusUtils.js:3-11`）：

```js
export const getStatusClassNames = (prefixCls, status, hasFeedback) => clsx({
  [`${prefixCls}-status-success`]:    status === 'success',
  [`${prefixCls}-status-warning`]:    status === 'warning',
  [`${prefixCls}-status-error`]:      status === 'error',
  [`${prefixCls}-status-validating`]: status === 'validating',
  [`${prefixCls}-has-feedback`]:      hasFeedback,
});
```

| 组件 | variant 类 | status 类 | 源码位置 |
|---|---|---|---|
| Input | `ant-input-outlined` 等 | `ant-input-status-error` | `input/Input.js:159-161` |
| TextArea | `ant-input-outlined` 等（**前缀仍是 `ant-input`**，不是 `ant-input-textarea`） | `ant-input-status-error` | `input/TextArea.js:143-145` |
| Select | `ant-select-outlined` 等 | `ant-select-status-error` | `select/index.js:200-206` |
| InputNumber | `ant-input-number-outlined` 等 | `ant-input-number-status-error` | `input-number/index.js:101-102` |

#### A1.4 variant 是否出现在 TextArea / Search / Password 上

| 组件 | 有 `variant` prop | 是否落到 DOM | 源码 |
|---|---|---|---|
| `Input.TextArea` | ✅（`TextArea.js:35` 解构 `variant: customVariant`；`:94` `useVariant('textArea', ...)`） | ✅ 走 `classNames.variant` 槽位（`TextArea.js:143-145`） | HIGH |
| `Input.Search` | ✅（`Search.js:34` `variant: customizeVariant`；`:52-54` 分别取 `mergedVariant` 与 `inputVariant`） | ✅ 内部透传给 `Input` | HIGH |
| `Input.Password` | ✅（`Password.js:35` `variant: customizeVariant`；`:46` `useVariant('inputPassword', ..., 'input')`） | ✅ 内部透传给 `Input` | HIGH |

**Search 的细节**（`Search.js:52-54, 107-109`）：搜索框主体与「搜索按钮」可以不同 variant —— 按钮侧类名为 `ant-input-search-btn-${variant}`。

---

### A2. 边框 / 背景 / 圆角 / 内边距 / 字号 / 颜色 的规则归属

#### A2.1 关键结论（先说结论）

> **antd 6 的边框声明在「持有 variant 类的那一个元素」上**，而不是固定在 `.ant-input` 或 `.ant-input-affix-wrapper` 上。
> 有 affix（prefix/suffix/allowClear/showCount）时，那一个元素是 **`.ant-input-affix-wrapper`**；没有 affix 时是 **`<input class="ant-input ant-input-outlined">`** 本身。
> `focus-within` 焦点环画在**同一个元素**上。

这一结论直接来自 slot 落点（`@rc-component/input/es/BaseInput.js`）：

```js
// @rc-component/input/es/BaseInput.js:43-46   —— 无 affix：variant 槽位 → 内层 input
let element = cloneElement(inputElement, {
  value,
  className: clsx(inputElement.props?.className, !hasAffix && classNames?.variant) || null
});

// @rc-component/input/es/BaseInput.js:79-88   —— 有 affix：variant 槽位 → affix-wrapper
const affixWrapperCls = clsx(affixWrapperPrefixCls, {
  [`${prefixCls}-disabled`]: disabled,
  [`${affixWrapperPrefixCls}-disabled`]: disabled,
  [`${affixWrapperPrefixCls}-focused`]: focused,     // ← 见 A2.5
  [`${affixWrapperPrefixCls}-readonly`]: readOnly,
  [`${affixWrapperPrefixCls}-input-with-clear-btn`]: suffix && allowClear && value
}, classes?.affixWrapper, classNames?.affixWrapper, classNames?.variant);   // ← variant 落点
```

#### A2.2 逐属性规则归属

| 属性 | 声明规则 | 源码位置 | 置信度 |
|---|---|---|---|
| **边框**（width/style/color） | `.ant-input-outlined { background: colorBgContainer; borderWidth: lineWidth; borderStyle: lineType; borderColor: colorBorder }` | `antd/es/input/style/variants.js:25-40`（`genBaseOutlinedStyle`） | HIGH |
| **背景** | 同上，`background: token.colorBgContainer` | `variants.js:26` | HIGH |
| **圆角** | 基础 `.ant-input { borderRadius: token.borderRadius }`；`underlined` 强制 `borderRadius: 0` | `input/style/index.js:55`（`genBasicInputStyle`）；`variants.js:285`；affix-wrapper 覆盖 `variants.js` 无关，见 `input/style/index.js:438-440`（`.ant-input-underlined { borderRadius: 0 }`） | HIGH |
| **内边距** | `padding: ${paddingBlock} ${paddingInline}`，来自 **Input 组件级 token** `paddingBlock/paddingInline`（由 `controlHeight` 与字号反算） | `input/style/index.js:51`；计算式 `input/style/token.js:36-46` | HIGH |
| **字号** | `fontSize: token.inputFontSize`；`-sm` / `-lg` 变体 | `input/style/index.js:53`；`:41-45`（`genInputSmallStyle`）；`:27-40`（`genInputLargeStyle`） | HIGH |
| **文字色** | `color: token.colorText` | `input/style/index.js:52` | HIGH |
| **占位符色** | `&::placeholder { color }` | `input/style/index.js:8-20`（`genPlaceholderStyle`），调用点 `:57` | HIGH |
| **过渡** | `transition: all ${motionDurationMid}` | `input/style/index.js:56` | HIGH |

#### A2.3 affix-wrapper 结构下的两层职责（v1.24 教训的源码确认）

```js
// antd/es/input/style/index.js:373-395  —— wrapper 层
[affixCls]: {                                   // .ant-input-affix-wrapper
  ...genBasicInputStyle(token),                // padding / color / fontSize / borderRadius
  display: 'inline-flex',
  [`> input${componentCls}`]: { padding: 0 },
  [`> input${componentCls}, > textarea${componentCls}`]: {
    fontSize: 'inherit',
    border: 'none',          // ← 内层 input 无边框
    borderRadius: 0,
    outline: 'none',
    background: 'transparent',
    color: 'inherit',
    '&:focus': { boxShadow: 'none !important' }   // ← 内层不画焦点环
  },
  ...
}
```

| 事实 | 源码 | 置信度 |
|---|---|---|
| affix-wrapper 的**内层** input/textarea 被强制 `border: none` | `input/style/index.js:384` | HIGH |
| 内层 input 被强制 `borderRadius: 0`、`background: transparent`、`font-size: inherit`、`padding: 0` | `input/style/index.js:383-388, 379-381` | HIGH |
| 内层 `:focus` 的 `boxShadow` 被 `none !important` 压掉 | `input/style/index.js:392-394` | HIGH |
| 边框实际画在 **wrapper**（wrapper 持 `ant-input-outlined` 类） | `variants.js:25-40` + `BaseInput.js:88` | HIGH |

#### A2.4 焦点环（focus-within）画在哪一层

```js
// antd/es/input/style/variants.js:34-39
'&:focus, &:focus-within': {
  borderColor: options.activeBorderColor,
  boxShadow: options.activeShadow,      // 0 0 0 {controlOutlineWidth}px {controlOutline}
  outline: 0,
  backgroundColor: token.activeBg
},
```

- 无 affix：`:focus` 命中 `<input.ant-input.ant-input-outlined>` → 环画在 input 上。
- 有 affix：`:focus-within` 命中 `wrapper.ant-input-affix-wrapper.ant-input-outlined`（内层 input 获焦时 wrapper 匹配 `:focus-within`）→ **环画在 wrapper 上**，且内层 `boxShadow:none !important` 保证不出现双环。
- 焦点环颜色 token：`activeShadow = \`0 0 0 ${controlOutlineWidth}px ${controlOutline}\``（`input/style/token.js:50`），`controlOutline` 来自 alias 层 `getAlphaColor(colorPrimaryBg, colorBgContainer)`（`theme/util/alias.js:82`）。

**项目现状对照**：`src/index.css:920-923` 用 `.ant-input-affix-wrapper-focused` 画焦点环 —— 该类由 `@rc-component/input/es/BaseInput.js:84` 添加，源码注释明写 `// Not used, but keep it`（即 antd 自身样式不消费它）。**能用，但依赖内部实现细节，属脆弱写法。**

#### A2.5 InputNumber 的边框归属

```js
// antd/es/input-number/style/index.js:57-91
[componentCls]: {                       // .ant-input-number
  ...resetComponent(token),
  ...genBasicInputStyle(token),
  ...genOutlinedStyle(token, {...}),    // ← 直接复用 Input 的 variants.js
  ...genFilledStyle(token, {...}),
  ...genUnderlinedStyle(token, {...}),
  ...genBorderlessStyle(token),
  [`${componentCls}-input`]: {          // ← 内层 input
    ...resetComponent(token),
    width: '100%',
    paddingBlock: varRef('input-padding-block'),
    backgroundColor: 'transparent',
    border: 0,                          // ← 内层无边框
    borderRadius: 0,
    outline: 0,
    fontSize: 'inherit', lineHeight: 'inherit',
  },
}
```

→ **InputNumber 边框画在 `.ant-input-number` 根元素上**（与 Input 的 `.ant-input-outlined` 同构），内层 `.ant-input-number-input` 无边框。

### A3. `status="error" / "warning"` 的 class 落点与颜色 token

#### A3.1 class 落点

| 组件 | status 类 | 落在哪个元素 | 源码 |
|---|---|---|---|
| Input | `ant-input-status-error` | 无 affix → `<input>` 本身；有 affix → `.ant-input-affix-wrapper` | `input/Input.js:159-161` + `BaseInput.js:45, 88` |
| TextArea | `ant-input-status-error` | `<textarea>` 本身 / `.ant-input-affix-wrapper` | `input/TextArea.js:143-145` |
| Select | `ant-select-status-error` | `.ant-select` **根元素** | `select/index.js:200-206` |
| InputNumber | `ant-input-number-status-error` | `.ant-input-number` **根元素** | `input-number/index.js:101-102` |

**否定 v1.24 的一条历史教训**（该教训本身正确，此处再确证）：

> ❌ **不存在** `ant-input-affix-wrapper-status-error` 这类别名。
> status 类与 variant 类**同级同元素**：`class="ant-input-affix-wrapper ant-input-outlined ant-input-status-error"`。

对应 CSS 规则形如（`variants.js:41-51`）：

```js
const genOutlinedStatusStyle = (token, options) => ({
  [`&${token.componentCls}-status-${options.status}:not(${token.componentCls}-disabled)`]: {
    ...genBaseOutlinedStyle(token, options),
    [`${token.componentCls}-prefix, ${token.componentCls}-suffix`]: { color: options.affixColor }
  },
  [`&${token.componentCls}-status-${options.status}${token.componentCls}-disabled`]: {
    borderColor: options.borderColor
  }
});
```

展开后（以 Input 为例）：

- `.ant-input-outlined.ant-input-status-error:not(.ant-input-disabled)` → **特异性 (0,3,0)**（`:where(hash)` 贡献 0）
- `.ant-input-outlined.ant-input-status-warning:not(.ant-input-disabled)` → 同上
- 附加规则：`.ant-input-outlined.ant-input-status-error.ant-input-disabled` → 禁用态保持灰边

#### A3.2 status 颜色 token 清单

**Input（outlined 变体）**，`antd/es/input/style/variants.js:63-78`：

| 语义 | token 名 | 源码行 |
|---|---|---|
| error 常态边框 | `colorError` | `variants.js:65` |
| error hover 边框 | `colorErrorBorderHover` | `variants.js:66` |
| error 聚焦边框 | `colorError` | `variants.js:67` |
| error 聚焦环 | `errorActiveShadow` → `0 0 0 {controlOutlineWidth}px {colorErrorOutline}` | `variants.js:68` / `input/style/token.js:51` |
| error 前后缀图标色 | `colorErrorAffix`（alias = `colorError`） | `variants.js:69` / `theme/util/alias.js:66` |
| warning 常态边框 | `colorWarning` | `variants.js:72` |
| warning hover 边框 | `colorWarningBorderHover` | `variants.js:73` |
| warning 聚焦边框 | `colorWarning` | `variants.js:74` |
| warning 聚焦环 | `warningActiveShadow` → `... {colorWarningOutline}` | `variants.js:75` / `input/style/token.js:52` |
| warning 前后缀图标色 | `colorWarningAffix`（alias = `colorWarning`） | `variants.js:77` / `theme/util/alias.js:67` |

> ⚠️ **不存在 `colorErrorBorder` 用于 Input 边框**：`colorErrorBorder` 是 Map Token（`theme/themes/shared/genColorMapToken.js:61`），但 Input/Select 的 outlined 边框用的是 `colorError`，hover 才用 `colorErrorBorderHover`。写 CSS 时若用 `var(--ant-color-error-border)` 会得到比边框更浅的色。

**Input（filled 变体）**，`variants.js:213-228`：背景 `colorErrorBg` / `colorErrorBgHover` / `colorWarningBg` / `colorWarningBgHover`，输入文字 `colorErrorText` / 边框 `colorError`。

**Input（borderless 变体）**，`variants.js:164-171`：无边框，仅改文字色 `colorError` / `colorWarning` + `:focus-visible` 的 `outline`（`variants.js:122-127, 132`）。

**Select**，`antd/es/select/style/select-input.js:297-312`：

| 语义 | token 名 |
|---|---|
| error 边框 / hover / 聚焦 | `colorError` / `colorErrorBorderHover` / `colorError` |
| error 聚焦环 | `colorErrorOutline` |
| error 前后缀 | `colorErrorAffix` |
| warning 边框 / hover / 聚焦 | `colorWarning` / `colorWarningHover` / `colorWarning` |
| warning 聚焦环 | `colorWarningOutline` |
| warning 前后缀 | `colorWarningAffix` |

**InputNumber**：完全复用 Input 的 `variants.js`（`input-number/style/index.js:57, 65, 78, 86` 的 import），token 清单同 Input。

---

### A4. Select 在 antd 6.6.4 的 DOM 结构、类名与 variant 关系

#### A4.1 `.ant-select-selector` 不存在（源码实证）

对 `node_modules/antd/es/**` 全量 grep `select-selector`，**唯一命中**是：

```js
// antd/es/color-picker/style/input.js:41-45
'&-single': {
  [`${antCls}-select-selector`]: {     // ← antCls = 'ant' ⇒ .ant-select-selector
    padding: 0,
    border: 0
  },
```

而 antd 6 的 Select DOM 中**根本没有**这个元素（见 A4.2）→ **这条 antd 自身的规则也是死规则**。

同时对 `node_modules/@rc-component/select/es/**` grep `selector` 只命中注释里的英文单词 "selector"，**无任何类名**。

> **HIGH 置信度结论：antd 6.6.4 不存在 `.ant-select-selector` 类。**
> 这与 v1.24 的历史教训一致，且已由源码从「项目实测」升级为「源码实证」。

#### A4.2 Select 的真实 DOM 结构与类名

```
<div class="ant-select ant-select-single ant-select-outlined ant-select-show-arrow
            ant-select-show-search ant-select-focused ant-select-status-error css-var-root css-dev-only-...">
  <span class="ant-select-prefix">…</span>
  <div class="ant-select-content">          ← SelectContent（rc-select）
    <span class="ant-select-selection-item">…</span>   （select-input.js:31）
    <span class="ant-select-placeholder">…</span>      （select-input.js:129）
    <input class="ant-select-input" />       （select-input.js:210-224）
  </div>
  <span class="ant-select-suffix">…</span>    （select-input.js:159）
  <button class="ant-select-clear">…</button>（select-input.js:220）
</div>
```

源码依据：

| 类名 | 源码 |
|---|---|
| `.ant-select`（根） | `antd/es/select/index.js:200-206` |
| `.ant-select-single` / `-multiple` | `@rc-component/select/es/BaseSelect/index.js:449, 448` |
| `.ant-select-focused` | `BaseSelect/index.js:447` |
| `.ant-select-allow-clear` / `-show-arrow` / `-disabled` / `-loading` / `-open` / `-customize-input` / `-show-search` | `BaseSelect/index.js:450-456` |
| `.ant-select-prefix` / `-suffix` / `-clear` | `@rc-component/select/es/SelectInput/index.js:208, 213, 220` |
| `.ant-select-content` | `select-input.js:136` |
| `.ant-select-input` | `select-input.js:210` |
| `.ant-select-placeholder` | `select-input.js:129` |
| `.ant-select-selection-item` | `select-input.js:31` |
| `.ant-select-dropdown` | `select/style/dropdown.js:34` |
| `.ant-select-item` | `select/style/dropdown.js:27` |
| `.ant-select-item-option-selected` / `-active` / `-disabled` | `select/style/dropdown.js:32, 107, 115, 118` |

> ⚠️ **`.ant-select-selection-placeholder` 在 antd 6 已不存在**（改为 `.ant-select-placeholder`）。项目 `src/index.css:2335` 引用的正是前者 → 死规则。

#### A4.3 边框声明在哪一层：**在 `.ant-select` 根元素上，且由 CSS 变量驱动**

antd 6 的 Select 是**变量驱动**的（与 Input 的直接 token 写法不同）：

```js
// antd/es/select/style/select-input.js:76-121
const [varName, varRef] = genCssVar(antCls, 'select');   // → --ant-select-*
return {
  [componentCls]: [{
    [varName('border-radius')]: token.borderRadius,
    [varName('border-color')]: '#000',
    [varName('border-size')]: token.lineWidth,
    [varName('background-color')]: token.colorBgContainer,
    ...
    // Border
    borderRadius: varRef('border-radius'),
    borderWidth:  varRef('border-size'),
    borderStyle:  token.lineType,
    borderColor:  varRef('border-color'),      // ← 边框在 .ant-select 根
    // Background
    background:   varRef('background-color'),
    ...
  }],
```

variant 通过**作用域**切换变量值：

```js
// antd/es/select/style/select-input.js:42-58
const genSelectInputVariantStyle = (token, variant, colors, errorColors, warningColors, patchStyle) => ({
  [`&${componentCls}-${variant}`]: [genSelectInputVariableStyle(token, colors), {
    [`&${componentCls}-status-error`]:   genSelectInputVariableStyle(token, { ...colors, ...errorColors }),
    [`&${componentCls}-status-warning`]: genSelectInputVariableStyle(token, { ...colors, ...warningColors })
  }, patchStyle]
});
```

四个 variant 的作用域（`select-input.js:290-371`）：`outlined` / `filled` / `borderless` / `underlined`，与 `Variants` 常量一一对应。

| 结论 | 置信度 |
|---|---|
| Select 边框在 `.ant-select` 根元素（**不在** `.ant-select-selector`，因为该元素不存在） | HIGH |
| Select 的边框色 = `var(--ant-select-border-color)`，由 `.ant-select-outlined` 等 variant 作用域赋值 | HIGH |
| Select 的 status 作用域 = `.ant-select.ant-select-status-error`（**双类同元素**） | HIGH |
| Select 焦点环在 `.ant-select` 根（`&${componentCls}-focused` → `boxShadow`） | HIGH（`select-input.js:29-33`） |
| Select 与 variant 的关系：**与 Input 完全同构**（prop → 类名 → CSS 变量作用域），只是实现载体是 CSS 变量而非直接 token | HIGH |

#### A4.4 Select 的 variant 解析

```js
// antd/es/select/index.js:104
const [variant, enableVariantCls] = useVariants('select', customizeVariant, bordered);
```

→ 默认 `'outlined'`，类名 `ant-select-outlined` 落在 `.ant-select` 根（`select/index.js:204`）。

---

### A5. InputNumber 的错误态结构与类名 —— 「InputNumber 错误态仍灰」的归因

#### A5.1 antd 6.6.4 的 InputNumber 错误态是**完整可用**的

| 事实 | 源码 | 置信度 |
|---|---|---|
| status 类 = `ant-input-number-status-error`，落在 `.ant-input-number` 根 | `input-number/index.js:101`（`getStatusClassNames(prefixCls, status, hasFeedback)`） | HIGH |
| variant 类 = `ant-input-number-outlined`，同一元素 | `input-number/index.js:102` | HIGH |
| 错误态边框规则存在且优先级足够 | `input/style/variants.js:41-51, 63-70`，被 InputNumber 复用（`input-number/style/index.js:57`）→ 展开为 `.ant-input-number-outlined.ant-input-number-status-error:not(.ant-input-number-disabled)`，**特异性 (0,3,0)** | HIGH |
| 禁用态类在**根**元素 | `@rc-component/input-number/es/InputNumber.js:502`（`[`${prefixCls}-disabled`]: disabled`）→ `:not(.ant-input-number-disabled)` 正确排除禁用态 | HIGH |
| 内层 input 无边框 | `input-number/style/index.js:120`（`border: 0`） | HIGH |

#### A5.2 真正让「错误态仍灰」的是**项目 CSS**（不是 antd）

| # | 项目规则 | 位置 | 覆盖关系 |
|---|---|---|---|
| 1 | `.ant-input, .ant-input-number, .ant-select-selector { background: … !important; border: 1px solid var(--input-border) !important; … }` | `src/index.css:871-880` | `border` **简写 + !important** → 压掉 antd 的 `border-color`（长属性、无 important）。antd 的 status 规则 (0,3,0) 也压不过 `!important` |
| 2 | 暗色块 `[data-color-mode="dark"] .ant-input-number { border-color: var(--input-border) !important; }` | `src/index.css:2297-2301` | (0,2,0) + `!important` → 同样压过 status 边框色 |
| 3 | **Input 域**补了 status 还原规则 `.ant-input.ant-input-status-error, .ant-input-affix-wrapper.ant-input-status-error { border-color: var(--ant-color-error, #ff4d4f) !important; }` | `src/index.css:958-965` | ✅ 但**只覆盖 Input 域** |
| 4 | **InputNumber 域没有对应补规则** | 缺口 | `ant-input-number.ant-input-number-status-error` 在全项目 CSS 中**不存在** |

> **结论（HIGH）**：「InputNumber 错误态仍灰」= 项目 CSS 的 `border` 简写 `!important`（`index.css:875`）压掉 antd 的 status `border-color`，**加上**项目只给 Input 域补了 status 规则、漏补 InputNumber 域。**antd 6.6.4 本身没有这个 bug。**
>
> 附带证据：项目 `index.css:951-957` 的注释已正确记录了「antd 6 把 status class 打在 wrapper 元素自身、名为 `ant-input-status-error`、不存在 `ant-input-affix-wrapper-status-error`」——**这条 v1.24 教训完全正确**，只是 InputNumber 域没有按同一模式补齐。

---

### A6. `ConfigProvider` theme 的 token 派生链（`token` vs `algorithm` vs `components.*`）

#### A6.1 完整派生链（源码逐步）

```
ConfigProvider theme={{ token, components, algorithm }}
   │
   ├─① theme/useToken.js:102
   │     useCacheToken(mergedTheme, [defaultSeedToken, rootDesignToken], {...})
   │     → useCacheToken.js:86  mergedToken = Object.assign({}, defaultSeedToken, rootDesignToken)
   │        ⇒ **显式 theme.token 与 seed 合并，seed 被显式值覆盖**
   │
   ├─② useCacheToken.js:91
   │     compute(mergedToken, override, theme)
   │     → theme/useToken.js:49-81  antd 自定义 getComputedToken
   │        ├─ :50  derivativeToken = theme.getDerivativeToken(originToken)
   │        │        → @ant-design/cssinjs/es/theme/Theme.js:19-21
   │        │          derivatives.reduce((result, derivative) => derivative(token, result), undefined)
   │        │          ⇒ 多算法链式：后一个算法的 map 入参 = 前一个算法的输出
   │        │             defaultAlgorithm → darkAlgorithm → compactAlgorithm
   │        ├─ :56-58 mergedDerivativeToken = { ...derivativeToken, override }
   │        └─ :61  formatToken(...)  → theme/util/alias.js:9-23（生成 alias token）
   │           └─ :62-79 components.* 逐个挂到 mergedDerivativeToken[key]
   │
   ├─③ 算法内部（themes/default/index.js:29-45）
   │     return {
   │       ...token,                                  // ← seed + 显式 token 在此进入
   │       ...colorPalettes,
   │       ...genColorMapToken(token, {...}),        // ← **在 ...token 之后 ⇒ 覆盖显式值**
   │       ...genFontMapToken(token.fontSize),
   │       ...genSizeMapToken(token),
   │       ...genControlHeight(token),
   │       ...genCommonMapToken(token),
   │     }
   │
   ├─④ genColorMapToken（themes/shared/genColorMapToken.js:35-36）
   │     return { ...neutralColors, colorPrimary: …, colorError: …, … }
   │     neutralColors = generateNeutralColorPalettes(colorBgBase, colorTextBase)   ← :21
   │
   ├─⑤ 明色派生（themes/default/colors.js:18-45）
   │     colorBgContainer: getSolidColor(colorBgBase, 0)      ← :38   （= colorBgBase 原色）
   │     colorBgLayout:    getSolidColor(colorBgBase, 4)      ← :37
   │     colorBgElevated:  getSolidColor(colorBgBase, 0)      ← :39
   │     colorBorder:       getSolidColor(colorBgBase, 15)     ← :42
   │     colorBorderSecondary: getSolidColor(colorBgBase, 6)   ← :44
   │
   ├─⑥ 暗色派生（themes/dark/index.js:40-51 → dark/colors.js:20-47）
   │     return { ...mergedMapToken, ...colorPalettes, ...colorMapToken, ... }
   │     ⇒ colorMapToken（含 colorBgContainer/colorBorder）**覆盖** mergedMapToken
   │     colorBgContainer: getSolidColor(colorBgBase, 8)       ← dark/colors.js:40
   │     colorBgElevated:  getSolidColor(colorBgBase, 12)      ← :39
   │     colorBgLayout:    getSolidColor(colorBgBase, 0)       ← :41
   │     colorBorder:       getSolidColor(colorBgBase, 26)     ← :44
   │
   ├─⑦ 组件级 token 合并（@ant-design/cssinjs-utils/es/util/getComponentToken.js:5,23）
   │     customToken = { ...token[component] }        // = theme.components.Input
   │     mergedToken = { ...defaultToken, ...customToken }
   │     ⇒ **组件级 token 在算法之后 merge ⇒ 组件 token 压过全局 token**
   │
   └─⑧ cssVar 转换（@ant-design/cssinjs/es/util/css-variables.js:22-47）
         token2CSSVar(key, 'ant') → --ant-color-bg-container / --ant-color-primary / …
         变量注入选择器： :where(.hash).css-var-root { --ant-*: … }
```

#### A6.2 三个层级的交互规则（可直接当规范用）

| 层级 | 写法 | 与算法关系 | 能否被算法改写 | 置信度 |
|---|---|---|---|---|
| **Seed** | `theme.token.colorBgBase` / `colorTextBase` / `colorPrimary` / `fontSize` / `borderRadius` / `controlHeight` | **算法的输入** | —— | HIGH |
| **Map** | `theme.token.colorBgContainer` / `colorBorder` / `colorBgLayout` / `colorBgElevated` / `colorErrorBg` … | **算法的输出** | **会被算法覆盖**（因 `genColorMapToken` 在 `...token` 之后展开） | HIGH |
| **Alias** | `theme.token.colorTextPlaceholder` / `colorSplit` / `controlOutline` … | 由 Map 计算 | 会被算法覆盖 | HIGH |
| **组件 token** | `theme.components.Input.colorBgContainer` | **算法之后 merge** | **算法改不动** | HIGH |
| **组件 `algorithm`** | `theme.components.Input.algorithm = true`（≥5.8.0） | 让组件 token 参与算法派生 | —— | MEDIUM（官方文档） |

#### A6.3 「`darkAlgorithm` 会不会改写显式传入的 `token`」—— 精确回答

> **对 `colorBgContainer` / `colorBorder` / `colorBgLayout` / `colorBgElevated` 等「由 `colorBgBase` 派生」的 Map Token：会。显式值无效。**
>
> 源码证据链：`themes/default/index.js:33`（`genColorMapToken` 在 `...token` 之后）→ `shared/genColorMapToken.js:36`（`...neutralColors`）→ `themes/default/colors.js:38,42` / `themes/dark/colors.js:40,44`（由 `colorBgBase` 派生）。暗色算法路径额外在 `themes/dark/index.js:45` 用 `...colorMapToken` 覆盖 `...mergedMapToken`。

> **对 `colorPrimary` / `colorError` / `colorTextBase` 等「算法不重新派生」的 token：不会被改写**（`genColorMapToken.js:41-92` 是**重新生成**品牌色阶，`colorPrimary: primaryColors[6]` 等；`colorError` 同理 → 显式传 `colorError` 也**不生效**，实际生效的是色阶第 6 档）。
>
> **对 `borderRadius` / `fontSize` / `controlHeight` 等纯尺寸 token：不会被改写**（算法只重算色阶与派生色）。

#### A6.4 本项目的具体病根（对照 `AntdThemeBridge.tsx`）

```tsx
// ruoyi-react-frontend/src/design-system/components/AntdThemeBridge.tsx:74-92
token: {
  colorPrimary: ruoyiBrand.greenPrimary,     // ✅ 品牌 token（算法会重算色阶，但入口正确）
  colorInfo:   ruoyiBrand.greenPrimary,
  colorLink:   ruoyiBrand.greenPrimary,
  colorSuccess: ruoyiBrand.functional.success,
  colorWarning: ruoyiBrand.functional.warning,
  colorError:   ruoyiBrand.functional.danger,
  colorTextBase: ruoyiBrand.cream.fg,          // ⚠️ 暗色下仍是浅色基底
  colorBgBase:   ruoyiBrand.cream.canvas,      // ⚠️ #F0ECE3 浅奶油 —— 暗色算法入口
  colorBgContainer: ruoyiBrand.cream.surface,  // ❌ 无效（Map Token，被算法覆盖）
  colorBgElevated:  ruoyiBrand.cream.surface,  // ❌ 无效（同上）
  colorBgLayout:    ruoyiBrand.cream.canvas,   // ❌ 无效（同上）
  colorBorder:      ruoyiBrand.cream.border,   // ❌ 无效（同上）
  colorBorderSecondary: ruoyiBrand.cream.borderStrong, // ❌ 无效（同上）
  borderRadius: 8, borderRadiusLG: 12, fontFamily: fontFamily.sans,   // ✅ 有效
},
```

```tsx
// AntdThemeBridge.tsx:121-136
components: {
  Input:  { colorBgContainer: ruoyiBrand.cream.surface, activeBorderColor: greenPrimary, hoverBorderColor: greenPrimary, activeShadow: "0 0 0 2px rgba(21,96,49,0.15)", borderRadius: 8, controlHeight: 36 },
  Select: { colorBgContainer: ruoyiBrand.cream.surface, activeBorderColor: greenPrimary, hoverBorderColor: greenPrimary, borderRadius: 8, controlHeight: 36, optionSelectedBg: … },
  Card:   { colorBgContainer: ruoyiBrand.cream.surface, headerBg: "transparent", borderRadiusLG: 12 },
}
```

**暗色下的实际结果（源码推演，HIGH）**：

| 项 | 明色实际值 | 暗色实际值 | 推导 |
|---|---|---|---|
| 全局 `colorBgContainer` | `getSolidColor(#F0ECE3, 0)` ≈ `#F0ECE3` | `getSolidColor(#F0ECE3, 8)` ≈ 更亮的浅奶油 | `default/colors.js:38` / `dark/colors.js:40` |
| 全局 `colorBorder` | `getSolidColor(#F0ECE3, 15)` | `getSolidColor(#F0ECE3, 26)` —— **浅色系** | `default/colors.js:42` / `dark/colors.js:44` |
| `components.Input.colorBgContainer` | `#FFFFFF` | **`#FFFFFF`（不变！）** | `getComponentToken.js:23` —— 算法之后 merge |
| `components.Select.colorBgContainer` | `#FFFFFF` | **`#FFFFFF`（不变！）** | 同上 |

→ **「暗色下 Select 容器仍是白底 + 浅灰边」的两条成因**：
① `components.Select.colorBgContainer = #FFFFFF` 写死，算法改不动；
② `colorBgBase = #F0ECE3` 让算法派生的 `colorBorder` 也是浅色系。
这与 F3 描述的症状**完全一致**，但机制是「**组件 token 写死 + seed 基底是浅色**」，不是「显式 token 被算法原样保留」。

### A7. antd 6 官方推荐的定制方式优先级 + 全局 CSS `!important` 覆盖的正确姿势与反模式

#### A7.1 官方优先级（依据 customize-theme 文档 + 源码）

| 优先级 | 机制 | 官方/源码依据 | 适用 | 不适用 |
|---|---|---|---|---|
| **P0** | **Seed Token**（`theme.token.colorPrimary` / `colorBgBase` / `colorTextBase` / `fontSize` / `borderRadius` / `controlHeight` …） | 文档「Seed Token means the origin of all design intent」；源码 `themes/seed.js` | 改**源头**，让算法自然派生明暗两套 | 想给单个组件换底色（会污染全站） |
| **P1** | **`variant` prop / 组件级 `variant` 配置 / `Form variant`** | `useVariants.js:7-28`；`config-provider/context.d.ts:206,226,233,256` | 形态切换（outlined/filled/borderless/underlined） | 颜色微调 |
| **P2** | **Component Token**（`theme.components.Input.xxx`） | `getComponentToken.js:23`；文档「Customize Component Token」 | 单组件精确控制，**算法之后生效**（算法改不动） | 需要随明暗自动切换的值（**必须按 mode 分支**） |
| **P3** | **`algorithm` 自定义派生**（`theme.algorithm` / `components.X.algorithm`） | 文档「Map Token … recommended to implement custom Map Token through `theme.algorithm`」 | 定制 Map Token 的**派生关系** | 逐值硬改 |
| **P4** | **cssVar 消费**（`var(--ant-*)`） | `css-variables.js:2-3, 22-47`；antd 6 默认开启 cssVar（`theme/useToken.js:96-99`） | 项目自有 CSS **读取** antd 的值，不再覆盖 antd | 改 antd 组件本身 |
| **P5** | **`@layer` / `hashPriority` / `postcss-increase-specificity`** | compatible-style 文档「@layer」「:where in selector」 | 需要「项目 CSS 赢」时的**结构性**手段 | 逐条补丁 |
| **P6** | 全局 CSS 覆盖 antd 组件（**含 `!important`**） | 官方文档**未列为推荐方式** | 极少数无法用 token 表达的特例 | —— |

#### A7.2 `:where` 降特异性 —— 对 v1.24 教训的**修正**

> **v1.24 教训「antd 6 的 cssinjs hash 前缀使 antd 规则特异性 +1 类 → 写在后面就能覆盖在 antd 6 下不成立」——按 antd 6.6.4 源码，该结论不成立。**

源码证据：

```js
// @ant-design/cssinjs/es/StyleContext.js:46-51
const StyleContext = React.createContext({
  hashPriority: 'low',        // ← antd 6 默认值
  ...
});

// @ant-design/cssinjs/es/util/index.js:136-145
export function where(options) {
  const { hashCls, hashPriority = 'low' } = options;
  ...
  return hashPriority === 'low' ? `:where(${hashSelector})` : hashSelector;
}
```

→ antd 6 生成的规则形如 `:where(.css-bAMboO).ant-input-outlined`，**hash 部分特异性为 0**，整条规则特异性 = `.ant-input-outlined` 的 **(0,1,0)**。

配合样式注入顺序（`useCacheToken.js:121-126`：`prepend: 'queue'` → antd 样式被**前置**到 `<head>` 顶部）：

> **同特异性时，项目静态 CSS（后）胜出 antd cssinjs 样式（前）。「写在后面就能覆盖」在 antd 6 下成立。**

那项目 1359 处 `!important` 的真正来源是（按影响排序）：
1. **简写 vs 长属性**：`border: 1px solid X`（简写）vs antd 的 `border-color`（长属性）—— 简写会重置全部 longhand，与特异性无关；
2. **antd 的复合选择器特异性更高**：`:where(.hash).ant-input-outlined.ant-input-status-error:not(.ant-input-disabled)` = **(0,3,0)**，项目的 `.ant-input` = (0,1,0) 压不过；
3. **伪类叠加**：`:hover` / `:focus` / `:focus-within` 各 +1 类；
4. **状态类复合**：`status` + `:not()` + `disabled` 三重叠加。

> **结论**：收敛的抓手不是「继续堆 `!important` 提特异性」，而是**把要改的值搬进 token / component token**，让 antd 自己生成正确规则。

#### A7.3 覆盖 antd 组件的「正确姿势」

| # | 做法 | 具体写法 | 依据 |
|---|---|---|---|
| 1 | **改 token / component token** | `theme.components.Input.colorBgContainer = …`；`theme.token.colorBgBase = …` | `getComponentToken.js:23` |
| 2 | **消费 antd 的 CSS 变量**（项目 CSS 读取，而非覆盖） | `background: var(--ant-color-bg-container);` `border-color: var(--ant-color-border);` `color: var(--ant-color-error);` | `css-variables.js:2-3`；变量挂在 `:where(.hash).css-var-root`（`css-variables.js:14-20`） |
| 3 | **按 mode 分支的组件 token** | `components.Input.colorBgContainer = mode === 'dark' ? undefined : cream.surface` | 算法管不到组件 token，只能自己分支 |
| 4 | **用 `variant` prop 而不是 CSS** | `<Select variant="filled" />`、`<ConfigProvider input={{ variant: 'outlined' }}>` | `useVariants.js:7-28` |
| 5 | **确需覆盖时，用 `:where` 对齐特异性**（而非 `!important`） | `.ant-input { border-color: var(--x) }` 天然 (0,1,0) = antd 同级 | `util/index.js:145` |
| 6 | **确需更高权重时用 `@layer` 或 `hashPriority="high"` 统一处理**（而不是逐条 `!important`） | `<StyleProvider layer>` + `@layer antd { … }` | compatible-style 文档「@layer」 |

#### A7.4 「反模式」清单（本项目命中项已标注）

| 反模式 | 为什么错 | 本项目命中 |
|---|---|---|
| ❌ 用 `!important` 提特异性去压 antd 的**普通规则** | antd 规则 = `:where(hash).<class>` (0,1,0)；同特异性后写即胜，`!important` 纯属浪费 | 1359 处 `!important` |
| ❌ 覆盖 antd 的**内部 DOM 结构类**（`.ant-select-selector`、`.ant-input-number-handler*`） | antd 6 已重构 DOM，此类**永不命中**（死规则） | `.ant-select-selector` ×9、`.ant-select-selection-placeholder` ×1、`.ant-input-number-handler-wrap` ×1、`.ant-input-number-handler` ×1、`.ant-picker-input` ×1 |
| ❌ 用 `border` 简写去改**边框色** | 简写重置所有 longhand，会连带干掉 antd 的 `border-color` 状态色 | `index.css:875`（直接导致 InputNumber 错误态仍灰） |
| ❌ 覆盖 `.ant-input-affix-wrapper > input.ant-input` 的 `border` | antd 已声明 `border:none`（`input/style/index.js:384`），再写一遍是冗余 | `index.css:937-945`（v1.24 修复，双框） |
| ❌ 依赖 `ant-input-affix-wrapper-focused` | rc 源码注释 `// Not used, but keep it`（`BaseInput.js:84`），属内部实现 | `index.css:920-923` 等 7 处 |
| ❌ 在 `:not()` 内放组合子（`.a:not(.b > .c)`） | CSS3 下整条规则被**静默丢弃** | ✅ 项目已规避（grep `:not\([^)]*>` 无命中） |
| ❌ 指望 `theme.token.colorBgContainer` 生效 | Map Token，会被算法覆盖 | `AntdThemeBridge.tsx:83,86` |

---

## B. antd 6 的弃用 / 迁移 API 清单（与本项目相关）

> 依据：① 本机 antd 6.6.4 源码中的 `devUseWarning(...).deprecated(...)` 调用；② 官方 [From v5 to v6](https://ant.design/docs/react/migration-v6/) 文档。
> 两者一致处标 HIGH。

#### B.1 输入框域（Input / Select / InputNumber / TextArea / Search / Password）

| 旧 API | 新 API | 源码依据 | 官方文档 | 置信度 |
|---|---|---|---|---|
| `Input bordered` | `Input variant` | `input/Input.js:49`（`[['bordered','variant'], …]`） | ✔ Input | HIGH |
| `Input addonBefore` / `addonAfter` | `Space.Compact` | `input/Input.js:49` | ✔ Input.Number 段（InputNumber 同） | HIGH |
| `Input.Group` | `Space.Compact` | —（组件级） | ✔ Input 段 | HIGH |
| `InputNumber bordered` | `InputNumber variant` | `input-number/index.js:149` | ✔ InputNumber | HIGH |
| `InputNumber addonBefore` / `addonAfter` | `Space.Compact` | `input-number/index.js:149` | ✔ InputNumber | HIGH |
| `TextArea bordered` | `TextArea variant` | `input/TextArea.js:45` | （文档未单列 TextArea，但同源） | HIGH（源码）/ MEDIUM（文档） |
| `Select bordered` | `Select variant` | `select/index.js:224` | ✔ Select | HIGH |
| `Select dropdownMatchSelectWidth` | `popupMatchSelectWidth` | `select/index.js:218` | ✔ Select | HIGH |
| `Select dropdownStyle` | `styles.popup.root` | `select/index.js:219` | ✔ Select | HIGH |
| `Select dropdownClassName` / `popupClassName` | `classNames.popup.root` | `select/index.js:220-221` | ✔ Select | HIGH |
| `Select dropdownRender` | `popupRender` | `select/index.js:222` | ✔ Select | HIGH |
| `Select onDropdownVisibleChange` | `onOpenChange` | `select/index.js:223` | ✔ Select | HIGH |
| `Select showArrow` | 默认行为；隐藏用 `suffixIcon={null}` | `select/index.js:229` | ✔ Select | HIGH |

#### B.2 关于 `onSearch` / `showSearch` 的**精确结论**

> ⚠️ **任务书中「antd 6 建议 `showSearch`（替代 `onSearch`）」这一条，在 antd 6.6.4 中查无实据。**

| 组件 | `onSearch` 状态 | `showSearch` 状态 | 证据 |
|---|---|---|---|
| `Select` | **未弃用**，仍在 `Select` API 中；仅 `combobox` 模式触发 | **未弃用**，是"开启可搜索"的开关 | `select/index.js:215-231` 的 `deprecatedProps` 列表中**不含** `onSearch` / `showSearch`；官方 migration 文档 Select 段也不含二者 |
| `Input.Search` | **未弃用**，`onSearch(value, e, { source })` 仍是主 API（`Search.js:30, 79-97`） | `Input.Search` 无 `showSearch` prop（`Search.js:18-40` 解构中无此项） | 同上 |

**正确的表述是**：`showSearch` 是 antd 5.13 起引入的**开关**（与 `variant` 同批），用来打开 Select 的可搜索态；它**不是** `onSearch` 的替代品。`onSearch` 在 antd 6 仍是官方 API。置信度 **HIGH**（源码 + 官方迁移文档双向确认）。

> 若任务目标是把「输入即搜」的交互统一，antd 6 的正解是 `variant` + `showSearch` + `filterOption`，而**不是**替换 `onSearch`。

#### B.3 Modal

```js
// antd/es/modal/Modal.js:126-127
[['bodyStyle', 'styles.body'], ['maskStyle', 'styles.mask'], ['destroyOnClose', 'destroyOnHidden'],
 ['autoFocusButton', 'focusable.autoFocusButton'], ['focusTriggerAfterClose', 'focusable.focusTriggerAfterClose'],
 ['maskClosable', 'mask.closable']].forEach(([deprecatedName, newName]) => {
  warning.deprecated(!(deprecatedName in props), deprecatedName, newName);
});
```

| 旧 API | 新 API | 版本 | 置信度 |
|---|---|---|---|
| `Modal maskClosable` | `Modal mask.closable` | 6.3.0 起弃用 | HIGH（`Modal.js:126` + 官方文档） |
| `Modal bodyStyle` | `styles.body` | 6.0.0 | HIGH |
| `Modal maskStyle` | `styles.mask` | 6.0.0 | HIGH |
| `Modal destroyOnClose` | `destroyOnHidden` | 6.0.0 | HIGH |
| `Modal focusTriggerAfterClose` | `focusable.focusTriggerAfterClose` | 6.2.0 | HIGH |
| `Modal autoFocusButton` | `focusable.autoFocusButton` | 6.0.0（源码） | HIGH（源码） |

补充（**非弃用但行为变更**，易踩坑）：antd 6 引入 `mask` 遮罩选项与模糊效果；**v6.0.0–v6.2.x 默认开启 blur，v6.3.0 起默认关闭**。若需开启：`ConfigProvider modal={{ mask: { blur: true } }}`（官方 migration 文档「Overlay components」）。

#### B.4 Table

| 旧 API | 新 API | 源码依据 | 置信度 |
|---|---|---|---|
| `pagination.position` | `pagination.placement` | `table/InternalTable.js:403` | HIGH |
| `onSelectInvert` | `onChange` | `table/hooks/useSelection.js:229` | HIGH |
| `onSelectMultiple` / `onSelectAll` / `onSelectNone` | `onChange` | 官方文档 | MEDIUM |
| 列 `filterDropdownOpen` | `filterDropdownProps.open` | `table/hooks/useFilter/FilterDropdown.js:132-135` | HIGH |
| 列 `onFilterDropdownOpenChange` | `filterDropdownProps.onOpenChange` | 同上 | HIGH |
| `locale.filterCheckall` | `locale.filterCheckAll` | `FilterDropdown.js:136` | HIGH |
| `Table size="middle"` | `size="medium"` | 6.3.0–6.3.2 枚举统一 | MEDIUM（官方文档） |
| `Table bordered` | **未弃用**（仍在 API 中） | 官方 Table 段无 `bordered` | MEDIUM |

#### B.5 Form

| 事项 | 内容 | 依据 | 置信度 |
|---|---|---|---|
| `Form.List` 的 `onFinish` 行为变更 | v6 中 `onFinish` **不再包含未注册的 Form.List 子项数据**，不再需要 `getFieldsValue({ strict: true })` 过滤 | 官方 migration 文档「Form onFinish no longer includes all data from Form.List」 | MEDIUM |
| `Form` 的 `variant` | **新增**（非弃用）：`FormConfig.variant`（`context.d.ts:226`），经 `VariantContext` 广播给表单内所有输入控件（`useVariants.js:13, 24`） | 源码 | HIGH |
| `Form` 的其他弃用 | 源码中 `antd/es/form/**` **无任何 `deprecated(` 调用**（grep 无命中） | 源码 | HIGH |

#### B.6 其它高频组件（与本项目相关，节选）

| 组件 | 旧 API → 新 API | 置信度 |
|---|---|---|
| `Card` | `bordered` → `variant`；`headStyle` → `styles.header`；`bodyStyle` → `styles.body` | HIGH（官方文档） |
| `Descriptions` | `bordered` **未弃用**；但 `size="default"` → `"large"`、`size="middle"` → `"medium"` | MEDIUM |
| `Tag` | `bordered={false}` → `variant="filled"`；`color="xxx-inverse"` → `variant="solid"` | MEDIUM |
| `Dropdown` | `dropdownRender` → `popupRender`；`overlayClassName` → `classNames.root`；`destroyPopupOnHide` → `destroyOnHidden` | MEDIUM |
| `Tooltip` | `overlayClassName` → `classNames.root`；`overlayStyle` → `styles.root` | MEDIUM |
| `Space` / `Space.Compact` | `direction` → `orientation`；`split` → `separator` | MEDIUM |
| `DatePicker` | `dropdownClassName` / `popupClassName` / `popupStyle` → `classNames.popup.root` / `styles.popup.root`；`bordered` → `variant`；`onSelect` → `onCalendarChange` | MEDIUM |
| `ConfigProvider` | `dropdownMatchSelectWidth` → `popupMatchSelectWidth` | MEDIUM |
| 全局 | `size="default"` → `"medium"`（Avatar/Badge/Card/Progress/Steps/Switch/Spin）；`size="middle"` → `"medium"`（Table/Divider/Descriptions） | MEDIUM |

#### B.7 本项目**实际**命中量（`src/**/*.tsx` 静态统计，2026-10-28）

| 旧 API 写法 | 命中次数 | 是否 antd 组件 prop | 需甄别 |
|---|---|---|---|
| `maskClosable` | **46** | ✅ Modal/Drawer prop | 全部需迁移 → `mask={{ closable: … }}` |
| `bordered`（合计 14 处 JSX + 2 处对象字面量） | 14 | 混合 | `Card bordered`（**已弃用**）≈13 处 `bordered={false}`；`Descriptions bordered`（**未弃用**）≈14 处；`Table bordered` 0 处 |
| `destroyOnClose` | 6 | ✅ Modal/Drawer prop | 需迁移 → `destroyOnHidden` |
| `bodyStyle` | 4 | ✅ Modal prop | 需迁移 → `styles.body` |
| `size="middle"` | 17 | ✅ Table/Divider | 需迁移 → `size="medium"` |
| `size="default"` | 1 | ✅ | 需迁移 → 按组件语义（`medium` 或 `large`） |
| `dropdownRender` / `dropdownClassName` / `onDropdownVisibleChange` / `showArrow` / `popupClassName` / `addonAfter` / `addonBefore` / `Input.Group` / `filterDropdownOpen` | **0** | —— | ✅ 无需处理 |
| `onSearch`（271）/ `showSearch`（340） | 大量 | 多为业务搜索回调 / antd Select | **非弃用**，无需迁移（见 B.2） |

## C. 迁移与收敛的官方建议

### C.1 有无「从手写覆盖迁移到 variant/token 机制」的官方指引？

**没有专门针对"CSS 覆盖收敛"的官方指南页**（截至 antd 6.6.8 文档站，检索 `customize-theme` / `compatible-style` / `migration-v6` 三页均未发现 "reduce CSS override" 专题）。置信度 **MEDIUM**（基于官方文档站三页穷举 + web 检索）。

但官方提供了**三组可直接使用的官方抓手**，覆盖了收敛所需的能力：

| 抓手 | 用途 | 官方位置 | 适用边界 |
|---|---|---|---|
| **① 三层 Token 规范**（Seed → Map → Alias） | 回答"该改哪一层" | [customize-theme](https://ant.design/docs/react/customize-theme)「Design Token」 | 只管 token 层的收敛，**不管**遗留 CSS 的删除 |
| **② 组件语义 classNames / styles**（`antd semantic <Component>`） | 回答"该用哪个语义槽位代替内部类名" | [CLI](https://ant.design/docs/react/cli) / `antd semantic Table` | **不覆盖** antd 内部 DOM 类名（那是"未公开"，不算 API） |
| **③ CSS 兼容/降权重机制**（`:where` / `@layer` / `hashPriority` / PostCSS 提权重插件） | 回答"确需覆盖时怎么写得稳" | [compatible-style](https://ant.design/docs/react/compatible-style) | 只解决**权重**，不解决"不该覆盖" |

### C.2 官方迁移脚本 / 工具

#### C.2.1 `@ant-design/cli`（**当前官方推荐**，v6 迁移的官方工具）

来源：[ant.design/docs/react/cli](https://ant.design/docs/react/cli) · [GitHub: ant-design/ant-design-cli](https://github.com/ant-design/ant-design-cli)

| 命令 | 用途 | 对本项目的价值 |
|---|---|---|
| `antd lint ./src` | **检查弃用 API、可访问性缺口、性能问题、最佳实践** | ⭐ 直接对应 B 节的 46 处 `maskClosable` / 14 处 `bordered` / 6 处 `destroyOnClose` 等 |
| `antd migrate <from> <to> [--apply ./src]` | 迁移清单（可自动修复 / 需人工），`--apply` 输出 **agent-ready migration prompt** | ⭐ 可直接把 v5→v6 的 API 迁移交给 agent 执行 |
| `antd info <Component>` | 单组件 props 表（含默认值、since、**deprecated 状态**） | 核对本文 B 节表格 |
| `antd token [Component]` | 全局 / 组件级 Design Token 取值 | 为「token 收敛」提供权威 token 清单 |
| `antd semantic <Component>` | `classNames` / `styles` 语义结构 | 把 CSS 覆盖改写成语义槽位 |
| `antd usage ./src` | 项目 antd 导入统计 | 评估收敛影响面 |
| `antd doctor` | 10 项环境诊断（React 版本、peer deps、SSR 等） | 前置检查 |
| `antd changelog [v1] [v2] [component]` | 跨版本 API diff | 核对弃用引入版本 |
| `antd setup --client claude` | 安装 antd skill 到 agent 环境 | 让后续 agent 直接用 `antd` 命令查知识 |

安装：`npm install -g @ant-design/cli`（要求 Node ≥ 20）。官方文档标注「55+ per-minor snapshots across v3/v4/v5/v6」「18 commands」「完全离线」。

> ⚠️ **适用边界**：`antd lint` / `antd migrate` 处理的是 **API 弃用迁移**，**不做** CSS 覆盖收敛，也不生成 variant/token 改造方案。`antd migrate 5 6` 是否含自动修复项**未验证**（文档只给了 `migrate 3 4`、`migrate 4 5 --apply` 的示例）。

#### C.2.2 `@ant-design/codemod-v5`（**注意：这是 v4→v5，不是 v5→v6**）

- 仓库：[ant-design/codemod-v5](https://github.com/ant-design/codemod-v5) —— 「codemod cli for antd v5 upgrade … help upgrade antd v5 using jscodeshift and postcss」。
- 官方 v4→v5 迁移页（[5x.ant.design/docs/react/migration-v5](https://5x.ant.design/docs/react/migration-v5/)）曾推荐它。
- **v5→v6 官方迁移页（[migration-v6](https://ant.design/docs/react/migration-v6/)）推荐的是 `@ant-design/cli`，未提 codemod 包。**
- 置信度 **MEDIUM**（网页检索 + 官方文档表述）。

#### C.2.3 其它官方工具

| 工具 | 位置 | 用途 |
|---|---|---|
| **Theme Editor** | [ant.design/theme-editor](https://ant.design/theme-editor) | 交互式调 token，导出 `theme` 配置 |
| **`theme.getDesignToken(config)`** | 源码 `antd/es/theme/getDesignToken.js`；官方文档「Static consume」 | 静态取 token（无 React 环境），可用于生成项目 CSS 变量表 |
| **`antd/design.md`** | [ant.design/design.md](https://ant.design/design.md) | 设计语言描述文件（google-labs/design.md 规范），给 AI agent 描述品牌体系用 |
| **`@ant-design/static-style-extract`** | 官方文档「Zero Runtime」 | 抽取静态 CSS（本项目暂不涉及 zeroRuntime） |

### C.3 官方「变体」示例

- [Input](https://ant.design/components/input/) 「Variants of Input」演示区（outlined / filled / borderless / underlined）
- [InputNumber](https://ant-design.antgroup.com/components/input-number) 同款四变体
- 官方文档确认变体数量为 **4**（与源码 `Variants` 常量一致）

### C.4 官方「减少覆盖」的推荐技术组合（可作为本项目 Phase 的技术路线图）

```tsx
// 1) 用 ConfigProvider 组件级 variant 统一形态（不用逐组件传 prop）
<ConfigProvider
  input={{ variant: 'outlined' }}
  inputNumber={{ variant: 'outlined' }}
  select={{ variant: 'outlined' }}
  theme={{ token: { /* Seed token：只改源头 */ }, algorithm: [darkAlgorithm, compactAlgorithm] }}
>
```

```tsx
// 2) 表单内统一形态
<Form variant="outlined">…</Form>   // 经 VariantContext 广播给表单内所有输入控件
```

```tsx
// 3) 需要"项目 CSS 稳赢 antd"时，用结构性手段而不是逐条 !important
<StyleProvider layer>…</StyleProvider>   // + CSS: @layer antd { … }
```

```css
/* 4) 项目自有 CSS 改为「消费」antd 的 CSS 变量，而不是「覆盖」antd */
.my-card { background: var(--ant-color-bg-container); border-color: var(--ant-color-border); }
.my-error { color: var(--ant-color-error); }
```

---

## Sources

### 本机源码（`ruoyi-react-frontend/node_modules/`，antd 6.6.4 实测）

**antd — Input / variant**
- `antd/package.json`（版本 6.6.4）
- `antd/es/config-provider/context.js:4`（`Variants` 常量）
- `antd/es/config-provider/context.d.ts:195,206-209,217,226,233,256,257,261,263-267,389`（组件级 `variant` 契约）
- `antd/es/form/hooks/useVariants.js:1-29`（variant 解析优先级与默认值）
- `antd/es/_util/statusUtils.js:1-12`（status 类名生成）
- `antd/es/input/Input.js:23-182`（`bordered` 弃用、classNames.variant 槽位）
- `antd/es/input/TextArea.js:22-163`（TextArea variant / status 落点）
- `antd/es/input/Search.js:18-198`（Search variant → 按钮/输入双 variant）
- `antd/es/input/Password.js:22-146`（Password variant → Input）
- `antd/es/input/style/index.js:1-567`（genBasicInputStyle / genInputStyle / genAffixStyle / 内层 input 复位）
- `antd/es/input/style/variants.js:1-345`（outlined / filled / underlined / borderless 的全部边框、状态、焦点规则）
- `antd/es/input/style/token.js:1-60`（Input 组件 token：paddingBlock/paddingInline/activeShadow 等）
- `antd/es/input/style/textarea.js:1-111`
- `antd/es/input-number/index.js:25-203`（status/variant 类落点、bordered 弃用）
- `antd/es/input-number/style/index.js:1-316`（两层结构、内层 border:0、复用 Input variants）

**antd — Select**
- `antd/es/select/index.js:31-287`（variant 解析、className 组装、deprecated 列表）
- `antd/es/select/style/index.js:1-149`
- `antd/es/select/style/select-input.js:1-378`（CSS 变量驱动的边框、variant 作用域、status 作用域）
- `antd/es/select/style/dropdown.js:1-149`（`.ant-select-item*` 类名）
- `antd/es/select/style/token.js:47`（`selectorBg` 组件 token）

**antd — theme / token**
- `antd/es/theme/useToken.js:1-115`（派生链入口、`getComputedToken`、components 合并）
- `antd/es/theme/context.js:1-13`（`defaultConfig.override`）
- `antd/es/theme/themes/default/index.js:1-46`（defaultAlgorithm：`...token` 后接 `genColorMapToken`）
- `antd/es/theme/themes/default/colors.js:1-46`（明色 `colorBgContainer/colorBorder` 由 `colorBgBase` 派生）
- `antd/es/theme/themes/default/colorAlgorithm.js`
- `antd/es/theme/themes/default/theme.js:1-4`
- `antd/es/theme/themes/dark/index.js:1-53`（darkAlgorithm：`...colorMapToken` 覆盖 `...mergedMapToken`）
- `antd/es/theme/themes/dark/colors.js:1-48`（暗色 `colorBgContainer/colorBorder` 由 `colorBgBase` 派生）
- `antd/es/theme/themes/dark/colorAlgorithm.js:1-6`（`getSolidColor` = lighten）
- `antd/es/theme/themes/shared/genColorMapToken.js:1-96`（`...neutralColors` 进入 Map Token）
- `antd/es/theme/util/alias.js:1-186`（formatToken、alias token、`override` 合并规则）
- `antd/es/theme/util/genStyleUtils.js:1-57`（`genCssVar` → `--ant-<component>-<name>`）
- `antd/es/theme/getDesignToken.js`

**antd — 弃用 API**
- `antd/es/modal/Modal.js:71,106,126-127,181,235`（maskClosable → mask.closable 等）
- `antd/es/table/InternalTable.js:403`（pagination.position → pagination.placement）
- `antd/es/table/hooks/useFilter/FilterDropdown.js:120-140`（filterDropdownOpen / onFilterDropdownOpenChange / filterCheckall）
- `antd/es/table/hooks/useSelection.js:229`（onSelectInvert → onChange）
- `antd/es/color-picker/style/input.js:41-45`（antd 自身残留的 `.ant-select-selector` 死规则）

**@ant-design/cssinjs**
- `@ant-design/cssinjs/package.json`
- `@ant-design/cssinjs/es/hooks/useCacheToken.js:1-155`（`mergedToken` 合并、`compute` 调用、`prepend: 'queue'` 注入顺序）
- `@ant-design/cssinjs/es/hooks/useStyleRegister.js:20-89`（`injectSelectorHash`）
- `@ant-design/cssinjs/es/StyleContext.js:1-85`（**`hashPriority: 'low'` 默认值**）
- `@ant-design/cssinjs/es/util/index.js:1-165`（`where()` → `:where(hash)`）
- `@ant-design/cssinjs/es/util/css-variables.js:1-47`（`token2CSSVar`、`transformToken`）
- `@ant-design/cssinjs/es/theme/Theme.js:1-22`（多算法 reduce）
- `@ant-design/cssinjs/es/theme/createTheme.js:1-17`

**@ant-design/cssinjs-utils**
- `@ant-design/cssinjs-utils/es/util/getComponentToken.js:1-33`（**组件 token 在算法之后 merge**）

**@rc-component**
- `@rc-component/input/es/BaseInput.js:1-139`（**variant 槽位落点：内层 input vs affix-wrapper**）
- `@rc-component/input/es/Input.js:11-204`（focused 状态 → `ant-input-affix-wrapper-focused`）
- `@rc-component/input/es/TextArea.js:115-191`
- `@rc-component/select/es/BaseSelect/index.js:25-540`（`.ant-select-*` 根类名）
- `@rc-component/select/es/SelectInput/index.js:190-236`（Select DOM 结构）
- `@rc-component/input-number/es/InputNumber.js:485-548`（`.ant-input-number-actions` / `-action` / `-disabled`）

### 本项目源码（只读，未修改）
- `ruoyi-react-frontend/src/design-system/components/AntdThemeBridge.tsx:1-192`（真实路径，非任务书所写路径）
- `ruoyi-react-frontend/src/design-system/tokens/colors.ts:72-89`（`cream.canvas/surface/border/borderStrong`）
- `ruoyi-react-frontend/src/index.css:1-5632`（1359 处 `!important`；574 处 `[data-color-mode="dark"]`；9 处 `.ant-select-selector`）
- `ruoyi-react-frontend/src/App.tsx:8,41-62`（AntdThemeBridge 挂载点）

### 外部 URL
- [Ant Design — Customize Theme](https://ant.design/docs/react/customize-theme)
- [Ant Design — CSS Compatible](https://ant.design/docs/react/compatible-style)
- [Ant Design — From v5 to v6（迁移指南）](https://ant.design/docs/react/migration-v6)
- [Ant Design — CLI（@ant-design/cli）](https://ant.design/docs/react/cli)
- [Ant Design — Input 组件文档](https://ant.design/components/input/)
- [Ant Design — InputNumber 组件文档](https://ant-design.antgroup.com/components/input-number)
- [GitHub — ant-design/ant-design-cli](https://github.com/ant-design/ant-design-cli)
- [GitHub — ant-design/codemod-v5（v4→v5，非 v5→v6）](https://github.com/ant-design/codemod-v5)
- [ant.design — Theme Editor](https://ant.design/theme-editor)
- [ant.design — design.md](https://ant.design/design.md)
- [5x.ant.design — V4 to V5（含 codemod-v5 指引）](https://5x.ant.design/docs/react/migration-v5/)
- [GitHub issue #58232 — Borderless inputs: no focus styles](https://github.com/ant-design/ant-design/issues/58232)（borderless 变体的 focus 行为，MEDIUM）

---

## 对本项目的可执行结论

> 下列 12 条按"可直接开工"排序。括号内为置信度。

### 1️⃣ 收敛的正确入口是 **token / component token / variant 三层**，不是继续加 CSS 覆盖规则（HIGH）

官方优先级（customize-theme 文档 + `getComponentToken.js:23` 源码）：
**Seed Token（`colorBgBase`/`colorTextBase`/`colorPrimary`/`fontSize`/`borderRadius`/`controlHeight`）→ `variant` prop/配置 → Component Token → 自定义 algorithm → 消费 `--ant-*` CSS 变量**。
全局 CSS `!important` 覆盖**不在**官方推荐清单内，应视为最后的例外手段。

### 2️⃣ **必须先纠正任务书的 F3 机制描述**：显式 `theme.token.colorBgContainer` 在 antd 6.6.4 中**无效**（会被算法覆盖）（HIGH）

`colorBgContainer` / `colorBorder` / `colorBgLayout` / `colorBgElevated` 是 **Map Token**，由 `colorBgBase` 派生（`themes/default/colors.js:38,42`、`themes/dark/colors.js:40,44`），且 `genColorMapToken` 在 `...token` 之后展开（`themes/default/index.js:33`）→ 显式值被覆盖。
⇒ **`AntdThemeBridge.tsx:83,84,85,86,87` 这 5 行 `colorBgContainer/colorBgElevated/colorBgLayout/colorBorder/colorBorderSecondary` 全部是无效代码，可直接删除**（行为不变，因为它们本来就从未生效）。
⇒ 反过来，**要改容器底色只能改 `colorBgBase`**。

### 3️⃣ 暗色下"白底 + 浅灰边"的**真正**成因是两条，修复要同时做（HIGH）

| 成因 | 位置 | 修法 |
|---|---|---|
| ① `components.Input/Select/Card.colorBgContainer = cream.surface(#FFFFFF)` —— **组件 token 在算法之后 merge，darkAlgorithm 改不动** | `AntdThemeBridge.tsx:122, 130, 164` | 暗色模式下**不传** `colorBgContainer`（或按 mode 分支给暗色值），让算法派生值生效 |
| ② `colorBgBase = cream.canvas(#F0ECE3)` 浅奶油 —— 暗色算法 `getSolidColor(#F0ECE3, 8)` 仍是浅色系 | `AntdThemeBridge.tsx:82` | 补一组**暗色品牌中性色**（dark canvas / dark fg），暗色时切换 `colorBgBase` / `colorTextBase` |

⇒ 改造后应达到：**暗色下 `colorBgContainer` / `colorBorder` 完全由 `colorBgBase` 派生，项目 CSS 一行不写也不会出现白底**。

### 4️⃣ 项目 CSS 有 **13 处 antd 6 已不存在的类名 = 纯死规则**，可直接删（HIGH）

| 类名 | 引用次数 | 项目位置 | antd 6.6.4 实况 |
|---|---|---|---|
| `.ant-select-selector` | **9** | `index.css:873, 884, 890, 897, 2329, 3962, 3968, 3972, 5579` | ❌ 类不存在（`select/style/**` 与 `@rc-component/select/**` 全量 grep 无命中；唯一残留是 antd 自身的 `color-picker/style/input.js:42` 死规则） |
| `.ant-select-selection-placeholder` | 1 | `index.css:2335` | ❌ antd 6 改为 `.ant-select-placeholder`（`select/style/select-input.js:129`） |
| `.ant-input-number-handler-wrap` | 1 | `index.css:3813` | ❌ antd 6 改为 `.ant-input-number-actions`（`input-number/style/index.js:159`） |
| `.ant-input-number-handler` | 1 | `index.css:3818` | ❌ antd 6 改为 `.ant-input-number-action`（`input-number/style/index.js:150`） |
| `.ant-picker-input` | 1 | `index.css`（picker 域） | ❌ `antd/es/date-picker/**` 全量 grep 无此类名 |

⇒ **这 13 处必须删或改名**（不是"样式不准"，而是**根本不生效**）。特别注意：`.ant-select-selector` 出现在 `index.css:873` 的**三选一简写**里，与 `.ant-input`、`.ant-input-number` 共享 `border`/`background` 简写 —— 删它时**不能**连带删掉 `.ant-input` / `.ant-input-number` 的部分。

### 5️⃣ 「InputNumber 错误态仍灰」= **项目 CSS 压掉了 antd 的状态边框色**，不是 antd 的 bug（HIGH）

antd 6.6.4 的 InputNumber 错误态**完全可用**：status 类 `ant-input-number-status-error` 落在 `.ant-input-number` 根，特异性 (0,3,0) 的状态规则存在（`input/style/variants.js:41-51` + `input-number/style/index.js:57`）。
被压掉的原因：
- `index.css:875` 的 `border: 1px solid var(--input-border) !important`（**简写**）压掉 antd 的 `border-color`（长属性、无 important）；
- `index.css:2297-2301` 暗色块的 `[data-color-mode="dark"] .ant-input-number { border-color: … !important }` 再压一次；
- 项目只给 Input 域补了 status 规则（`index.css:958-965`），**InputNumber 域漏补**。

⇒ 修法（两条同时）：① 把 `border` 简写拆成 `border-width` / `border-style` / `border-color` 三个长属性；② 补 `.ant-input-number.ant-input-number-status-error`（+ warning）规则，或直接依赖 token（更优）。

### 6️⃣ **v1.24 的"写在后面就能覆盖不成立"这条教训需要修正**：antd 6 默认用 `:where` 降特异性（HIGH）

`@ant-design/cssinjs/es/StyleContext.js:47` 默认 `hashPriority: 'low'` → `util/index.js:145` 生成 `:where(.css-xxx)` → **hash 部分特异性为 0**，antd 规则实际特异性 = (0,1,0)；再叠加 `useCacheToken.js:123` 的 `prepend: 'queue'`（antd 样式前置到 `<head>` 顶部）。
⇒ **同特异性下，项目静态 CSS（后）赢 antd。** 1359 处 `!important` 的真正来源是「`border` 简写 vs 长属性」+「antd 复合选择器（status + `:not()` + 伪类）达到 (0,3,0)」。
⇒ **收敛的正确手段是把这批值搬进 token，不是继续提特异性。**

### 7️⃣ 项目可以**立刻删除**的 CSS 批次（可估算行数与 `!important` 收益）

| 批次 | 内容 | 估算规模 | 风险 |
|---|---|---|---|
| **A** | 13 处死规则（见结论 4） | ~13 处引用 / 约 25 行 | 极低（本就无效） |
| **B** | `index.css:937-945` `.ant-input-affix-wrapper > input.ant-input { border:none … }` 整块 | ~9 行 / 5 处 `!important` | 极低（antd `input/style/index.js:382-395` 已声明同样的复位） |
| **C** | `index.css:871-880` 的 `border` 简写 → 拆 `border-width`/`border-style`/`border-color`，并**去掉 `!important`** | ~10 行 / 约 10 处 `!important` | 低（同特异性后写胜；需同步验证 status/focus 态） |
| **D** | `index.css:920-923` 等 7 处 `.ant-input-affix-wrapper-focused` → 改为 `:focus-within` | ~7 处 | 低（rc 源码注释 `// Not used, but keep it` 表明该类非 antd 契约） |
| **E** | 暗色重复分区：`index.css:2294-2333` 与 `3955-3980` 两块对 Input/Select 的暗色覆盖 —— 改为消费 `var(--ant-color-bg-container)` / `var(--ant-color-border)` 后可整块删 | ~40 行 / 约 40 处 `!important` | 中（需先完成结论 3 的 token 改造） |
| **F** | `index.css:2297`（暗色 `.ant-input-number` border-color） | 1 处 | 低（随批次 C/E 一并处理） |

⇒ **保守估计：批次 A+B+C+D 可先落地，约 -90 行、-25 处 `!important`、0 行为变更**；批次 E+F 需在 AntdThemeBridge 改造（结论 3）完成后落地，约 -40 行、-40 处 `!important`。
⇒ **总体目标（估算，需实测）：1359 → 600 左右的 `!important`，index.css 5632 → 约 4800 行**。⚠️ 此为**估算**，实际以 Phase 实测为准。

### 8️⃣ variant 收敛的**最短路径**是 ConfigProvider / Form 两处配置，不必逐组件传 prop（HIGH）

- `ConfigProvider` 组件级：`input={{ variant: 'outlined' }}`、`inputNumber={{ variant: 'outlined' }}`、`select={{ variant: 'outlined' }}`（类型见 `config-provider/context.d.ts:206,233,256`）；
- 表单内统一：`<Form variant="outlined">` —— 经 `VariantContext` 广播给表单内所有输入控件（`useVariants.js:13, 24`；`FormConfig.variant` 见 `context.d.ts:226`）；
- 默认已是 `outlined`（`useVariants.js:24`），**所以第一步其实不需要改任何代码**；需要做的是**把 CSS 里那些"想给某类输入框换形态"的手写规则，改写成 variant 配置**。

### 9️⃣ status 类落点已由源码确证，可直接改写项目注释与规则（HIGH）

- **不存在** `ant-input-affix-wrapper-status-error`（项目 `index.css:952` 的注释已正确记录此点 ✅）；
- 真实 DOM：`class="ant-input-affix-wrapper ant-input-outlined ant-input-status-error"` —— status 与 variant **同级同元素**；
- 规则形如 `.ant-input-outlined.ant-input-status-error:not(.ant-input-disabled)`（特异性 (0,3,0)）；
- Select 侧：`.ant-select.ant-select-status-error`（`select/index.js:206`）；
- InputNumber 侧：`.ant-input-number.ant-input-number-status-error`（`input-number/index.js:101`）。

### 10️⃣ B 节的 API 迁移应**单独排期**，与 CSS 收敛解耦（HIGH）

| 项 | 数量 | 迁移目标 |
|---|---|---|
| `maskClosable` | **46** | `mask={{ closable: … }}` |
| `bordered`（`Card` 上的，**已弃用**） | ≈13 | `variant`（`Card` 的 `bordered={false}` → `variant="borderless"`） |
| `bordered`（`Descriptions` 上的，**未弃用**） | ≈14 | **不动** |
| `destroyOnClose` | 6 | `destroyOnHidden` |
| `bodyStyle` | 4 | `styles.body` |
| `size="middle"` | 17 | `size="medium"` |
| `size="default"` | 1 | 按组件语义 |

⇒ ⚠️ **`Descriptions bordered` 与 `Table bordered` 在 antd 6 **未**弃用，不要一起改。**
⇒ 官方工具：`npm i -g @ant-design/cli` → `antd lint ./src` 直接产出清单。

### 11️⃣ **不要**把 `onSearch` 迁移成 `showSearch`（HIGH，纠正任务书）

antd 6.6.4 中 `Select.onSearch` / `Input.Search.onSearch` **均未弃用**（源码 `select/index.js:215-231` 的弃用清单与官方 migration 文档 Select 段都不含二者）。`showSearch` 是 antd 5.13 引入的**"开启可搜索"开关**，不是 `onSearch` 的替代品。项目 271 处 `onSearch` / 340 处 `showSearch` **不需要为 antd 6 做迁移**。

### 12️⃣ 建议的 Phase 拆分与验收口径（MEDIUM —— 基于以上 HIGH 事实的排期建议）

| Phase | 内容 | 验收口径 |
|---|---|---|
| **P0** | AntdThemeBridge 重构：删 5 行无效 Map Token；`colorBgBase`/`colorTextBase` 按 mode 切换；`components.Input/Select/Card` 去掉写死的 `colorBgContainer` | 暗色下 `getComputedToken` 输出的 `colorBgContainer` ≠ `#FFFFFF`；页面无白底容器 |
| **P1** | 删死规则批次 A/B/D（13 处死类 + affix 内层复位 + `affix-wrapper-focused`） | `index.css` 中不再出现 `select-selector` / `selection-placeholder` / `handler-wrap` / `handler` / `picker-input` |
| **P2** | `border` 简写拆长属性（批次 C）+ 补 InputNumber status 规则（结论 5） | InputNumber 错误态边框变 `var(--ant-color-error)`；`.ant-input` 无 `border` 简写 `!important` |
| **P3** | 暗色重复分区收敛（批次 E/F）：改为消费 `var(--ant-*)` | `index.css` 中 `[data-color-mode="dark"]` 对 Input/Select 的覆盖归零 |
| **P4** | variant 配置化（结论 8）+ API 弃用迁移（结论 10，可用 `antd lint` 驱动） | `antd lint ./src` 无 deprecated 告警 |

---

### 仍待验证的点（LOW / 未验证）

| # | 待验证项 | 当前状态 | 验证方式 |
|---|---|---|---|
| 1 | `antd migrate 5 6 --apply ./src` 是否含自动修复项 | **未验证**（官方文档只给了 `3→4`、`4→5` 示例） | 装 `@ant-design/cli` 后实测 |
| 2 | 本项目实际运行时 antd 6.6.4 生成的 CSS 是否真为 `:where(hash)` 形态 | 源码推导 HIGH，但**未在浏览器实测** | DevTools 查 `<style data-css-hash>` 内容 |
| 3 | 批次 A/B/C/D/E/F 的**精确行数与 `!important` 收益** | **估算**（结论 7） | 实施时逐条 diff 实测 |
| 4 | 项目暗色品牌中性色（dark canvas / dark fg）的取值 | **不存在**，需设计决策 | 新建 `ruoyiBrand.dark` 常量组 |
| 5 | antd 6 `borderless` 变体的 focus 行为（社区 issue #58232 提到「borderless inputs: no focus styles」） | MEDIUM | 实测 `:focus-visible` outline 是否出现 |
| 6 | `Descriptions bordered` / `Table bordered` 在 6.6.4 具体版本的弃用状态 | MEDIUM（源码未在 form/table grep 到 `bordered` 弃用告警） | `antd info Descriptions` / `antd info Table` |

