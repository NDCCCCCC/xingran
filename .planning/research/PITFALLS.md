# Pitfalls Research — v1.24「视觉回归修复 + 遗留定责与清偿」

**Domain:** 前端样式层回归修复（antd 6.6.4）+ 权限定责（#16）+ 诊断展示（3f130）+ 路由行为（F5）+ lint/react-refresh 清偿（RuoYi-Next v1.24）
**Researched:** 2026-10-28
**Confidence:** HIGH（样式层与 lint 层结论全部基于实际读取的 `node_modules/antd@6.6.4` 源码、`node_modules/@rc-component/input`、`node_modules/@ant-design/cssinjs`、`node_modules/eslint-plugin-react-refresh` 实现，以及 `npm run lint` 的实测输出（17 errors / 34 warnings）；功能层基于 `pkg/permission/config.go`、`internal/services/system/menu_service.go`、`internal/api/v1/operations/workstation_device_handler.go`、`src/router/DynamicRoutes.tsx` 等源码原文。个别断言标注 MEDIUM）

**Note:** 本文件此前为 v1.23 版本（研究对象为「前端契约全量收尾与遗留清偿」），已按 v1.24 范围**整体覆盖重写**；v1.23 原文可从 git 历史找回。

**v1.24 范围与 phase 命名（本文用逻辑 phase 名，ROADMAP 定稿时对号入座）：**

| Phase | 内容 |
|---|---|
| **P1 样式修复** | 输入框双重外框（v1.23 引入的回归） |
| **P2 #16 定责** | 行操作权限 PG 复测定责 + 修复 |
| **P3 3f130 展示** | 空因诊断前端展示 |
| **P4 F5 落地页** | F5 刷新落 `/dashboard` |
| **P5 豁免注释** | `ColumnConfigModal.tsx:136` |
| **P6 lint 清偿** | 17 errors + exhaustive-deps 12 处 |
| **P7 结构整理** | react-refresh 33 处清零 + devices 页拆分 |

---

## Critical Pitfalls

### 领域 A：样式层（输入框双重外框）

### Pitfall A1：`:not()` 内含组合子 —— CSS3 下整条规则被**丢弃**，不是"不匹配"

**What goes wrong:**
拟用 `.ant-input:not(.ant-input-affix-wrapper > .ant-input)` 收窄主题规则、排除 affix-wrapper 内层 input。CSS Selectors Level 3 规定 `:not()` 只接受**简单选择器**（元素名/通配/属性/类/ID/伪类），**组合子（`>`、空格、`+`、`~`）不允许出现在 `:not()` 参数中**。在 CSS3 解析器下，这条规则不是"匹配不到任何元素"，而是**整条声明被丢弃**（等价于该规则不存在）→ 收窄完全失效，双框原样保留，而且**开发/生产浏览器都不会报错**（无控制台告警）。

更危险的是"半生效"：如果只把 `border` 挪进 `:not()` 块而 `background`/`border-radius` 留在块外，规则被丢弃后会出现"边框规则没生效、其它规则生效"的错位认知，排查方向被带偏。

**Why it happens:**
CSS3 的 `:not()` 限制写进规范时还没有 `:is()`/`:where()`，开发者习惯用"排除型选择器"表达"排除某结构内的元素"，而 CSS 直到 Level 4 才放宽（`:not()` 接受 complex-real-selector-list）。

**How to avoid:**
**不要用 `:not()` 收窄**。改用下面三种之一（详见文末《附录 A：`:not()` 替代方案矩阵》）：
1. **wrapper 上下文 + 直接子元素组合器**（推荐，CSS3 合法）：
   ```css
   .ant-input-affix-wrapper > input.ant-input,
   .ant-input-affix-wrapper > textarea.ant-input { border: none !important; ... }
   ```
   特异性 (0,3,1)，组合子在 `:not()` **之外**，CSS3 完全合法。
2. **`:where()` 包裹 + 组合子在 `:where()` 外**（CSS3 合法，`:where()` 零特异性）：
   ```css
   :where(.ant-input-affix-wrapper) > :where(input.ant-input) { ... }
   ```
   用于「不要压制 antd 其它状态样式」的场合。
3. **antd 6 的 variant 类**（`.ant-input-outlined` 等）—— 语义最准，但对 `ConfigProvider` 全局 variant 变更敏感（见 A3）。

若确实必须用 `:not()` 收窄，**必须**同时：
- 确认 `browserslist` 全部目标 ≥ Chrome 88 / Firefox 84 / Safari 9；
- 加 `@supports selector(:not(* > *))` 包裹做能力探测；
- 在 CI 里加一条「`:not()` 参数内不得出现组合子」的 lint/脚本检查（`scripts/` 下已有 `check-*.mjs` 家族可扩展）。

**Warning signs:**
- 规则"看起来写了但一点效果都没有"，且浏览器控制台无任何报错；
- DevTools 里该规则被标为灰色（无效）；
- 同一元素上，块内第一条声明生效、后续不生效。

**Phase to address:** P1 样式修复（方案选型阶段就要定，不要留到实现期）。

---

### Pitfall A2：`!important` 无条件压过非 important —— 这才是双框的确切机制

**What goes wrong:**
误以为 antd 的 `border: none`（写在 `.ant-input-affix-wrapper > input.ant-input` 上，特异性 (0,2,1) 起步）会压过项目的 `.ant-input { border: 1px solid ... !important }`（特异性只有 (0,1,0)）。**事实相反**：CSS 的层叠顺序中，**带 `!important` 的作者声明优先于不带 `!important` 的作者声明，与特异性无关**。项目的 `!important` 无条件胜出 → affix-wrapper 内层 input 被画上 1px 边框 → 双框。

**项目内确证（`src/index.css`）：**
- `:871-880` — `.ant-input, .ant-input-number, .ant-select-selector { background / border: 1px solid var(--input-border) / color / border-radius / font-size / transition }` **全部带 `!important`**
- `:882-886` hover、`:888-893` focus、`:895-901` 暗色 focus、`:903-905` placeholder、`:908-914` affix-wrapper、`:916-918` affix hover、`:920-923` affix-focused、`:925-929` 暗色 affix-focused —— **同样全部 `!important`**

**antd 侧确证（`node_modules/antd/es/input/style/index.js`）：**
- `:379-395` —
  ```js
  [`> input${componentCls}, > textarea${componentCls}`]: {
    fontSize: 'inherit', border: 'none', borderRadius: 0, outline: 'none',
    background: 'transparent', color: 'inherit',
    '&:focus': { boxShadow: 'none !important' }
  }
  ```
  → `border: 'none'` **没有 `!important`** → 被项目 `!important` 击穿。
- 同理 `node_modules/antd/es/input/style/textarea.js:50-60`（TextArea 的 affix-wrapper 内层 textarea 也是 `border: 'none'`，无 `!important`）。

**Why it happens:**
`src/index.css` 共 5590 行，其中 `!important` 出现 1362 处。这个密度不是风格偏好，而是被 antd 6 的 cssinjs **hash 前缀**逼出来的（见 A4）。当一个项目所有覆盖规则都带 `!important` 时，任何一条漏掉 `!important` 的上游规则都会被击穿 —— 而 antd 自己的 `border: none` 恰恰就是这种"上游不带 important"的规则。

**How to avoid:**
- **修复必须同样带 `!important`**，且**特异性要高于 `.ant-input`**，否则会输给 antd 的 `> input.ant-input`（0,2,1）或项目的其它规则。
- 推荐写法（**新增到 `index.css` 末尾**，或在 `:871-880` 附近成组改写）：
  ```css
  /* affix-wrapper 内层 input/textarea 复位：antd 原生要求内层无边框 */
  .ant-input-affix-wrapper > input.ant-input,
  .ant-input-affix-wrapper > textarea.ant-input {
    border: none !important;
    border-radius: 0 !important;
    background: transparent !important;
    color: inherit !important;
    font-size: inherit !important;
    padding: 0 !important;
  }
  ```
  特异性 (0,3,1)，全部 `!important` → 同时压过项目 `.ant-input` (0,1,0)、项目暗色块 `[data-color-mode="dark"] input` (0,1,1)、antd 自身 (0,2,1)。
- **不要**只把 `border` 拆出去、留下 `background`/`border-radius`/`font-size` 在 `.ant-input` 上 —— 这些属性同样会让内层 input 出现**圆角裂边**（内层圆角 + wrapper 圆角不同心）、**字号不一致**（内层被强制 14px 而 antd 期望 `inherit`）、**背景不透明**（filled/hover 变体下会与 wrapper 露出色差）。必须**成组复位**。

**Warning signs:**
- 修完仍有"细边框"残留 → 只改了 `border` 没改 `border-radius`；
- 修复后输入框内部出现**两个圆角**或**四角接缝** → 内层 `border-radius` 未复位；
- 修复后输入框内文字与右侧 prefix 图标基线不齐 → `font-size` 未复位为 `inherit`。

**Phase to address:** P1 样式修复。

---

### Pitfall A3：antd 6 的 `variant` 体系 —— `.ant-input` 本身**没有边框**，边框只在变体类上

**What goes wrong:**
沿用 antd 5 的心智模型（`.ant-input` 自带边框），写出 `.ant-input-outlined` / `:not(.ant-input-outlined)` 之类的收窄规则，结果在 antd 6 上大面积失效。

**antd 6.6.4 实测事实：**
- `node_modules/antd/es/input/style/index.js:296-318` — `genInputStyle` 的结构是
  ```js
  [componentCls]: { ...resetComponent(token), ...genBasicInputStyle(token),
                    ...genOutlinedStyle(token), ...genFilledStyle(token),
                    ...genBorderlessStyle(token), ...genUnderlinedStyle(token) }
  ```
  而 `genBasicInputStyle`（同文件 `:46-71`）**不含任何 border 声明**。
  → **antd 6 的 `.ant-input` 基础类没有边框**；边框只存在于 `.ant-input-outlined` / `.ant-input-filled` / `.ant-input-underlined` 三个变体类里（`variants.js:25-40 / 177-193 / 280-295`）。
- `node_modules/antd/es/config-provider/context.js:4` — `Variants = ['outlined', 'borderless', 'filled', 'underlined']`。
- `node_modules/antd/es/form/hooks/useVariants.js:7-28` — 变体解析优先级：
  ```
  variant prop  >  bordered={false} → 'borderless'
                >  Form VariantContext
                >  config.components[component].variant
                >  config.variant（全局）
                >  'outlined'（默认）
  ```
  `enableVariantCls = Variants.includes(mergedVariant)`。

**本项目现状：** `src/design-system/components/AntdThemeBridge.tsx:178` 是**唯一**的 antd `<ConfigProvider>`，其 `theme.components.Input`（`:121-128`）只配了 `colorBgContainer / activeBorderColor / hoverBorderColor / activeShadow / borderRadius / controlHeight`，**没有 `variant`**；也没有设全局 `theme.variant`。
→ **当前所有 Input 都是 `outlined`**。但这是"恰好"，不是"契约"。

**为什么这是个陷阱：**
一旦有人给 `ConfigProvider` 加 `variant="borderless"`（或 `components.Input.variant`），则：
- 所有 Input 变 `ant-input-borderless` → **`.ant-input-outlined` 收窄规则全部失配** → 项目主题（背景/圆角/字号/边框色）**整体消失**；
- antd 的 borderless 变体是 `background: transparent; border: none`（`variants.js:141-150`），与项目 `--input-bg` 主题**语义冲突** → 输入框会"消失"在背景里。

**How to avoid:**
- 若采用「variant 类收窄」路线（`.ant-input-outlined`），**必须**同时：
  1. 在 `AntdThemeBridge.tsx` 显式锁死 `variant: "outlined"`（写进 `theme.components.Input`），并加注释说明"项目主题只适配 outlined"；
  2. 或者把主题样式写成 `:where(.ant-input-outlined, .ant-input-borderless, .ant-input-filled, .ant-input-underlined)` 的并集，再按变体分别覆写 border/background。
- 若采用「wrapper 上下文复位」路线（推荐，见 A2），**对 variant 完全免疫** —— 不依赖任何变体类名。
- 任何时候改 `AntdThemeBridge` 的 `theme` 结构，都要重新截图验证输入框（antd `hashed: true` 会重新生成 hash，`:173`）。

**Warning signs:**
- 某次改 `AntdThemeBridge` 后输入框"没有边框了"或"背景和页面一样"；
- 改动 `ConfigProvider` 后 `useVariants` 的 `mergedVariant` 变了但没人察觉；
- 变体类只出现在部分组件上（Input 有、TextArea 没有）→ 组件名 key 不同（`useVariant('input', ...)` vs `useVariant('textArea', ...)`，`TextArea.js:94`），TextArea **不 fallback** 到 `components.Input.variant`。

**Phase to address:** P1 样式修复（若走 variant 路线）；P6/P7 的任何 `AntdThemeBridge` 改动都要重跑视觉验证。

---

### Pitfall A4：antd 6 cssinjs 的 hash 前缀让 antd 规则特异性 **+1 类** —— 「顺序依赖」其实是「特异性依赖」

**What goes wrong:**
按 CSS 常规经验推断"新写的规则放在 `index.css` 后面就能覆盖 antd"。在 antd 6 下这个推断**不成立**。

**实测事实：** `node_modules/@ant-design/cssinjs/es/hooks/useStyleRegister.js:27-47` 的 `injectSelectorHash` 把 hashId 注入到 selector **第一个 compound 之后**：
```js
const firstPath = fullPath[0] || '';
const htmlElement = firstPath.match(/^\w+/)?.[0] || '';
firstPath = `${htmlElement}${hashSelector}${firstPath.slice(htmlElement.length)}`;
```
→ antd 生成的规则形如 `.css-dev-only-do-not-override-xxxx.ant-input-outlined`、`.css-dev-only-xxxx.ant-input-affix-wrapper > input.ant-input`，**每个都多一个类**。

**后果：**
| 规则 | 实际特异性 | 胜负 |
|---|---|---|
| 项目 `.ant-input { border: ... !important }` | (0,1,0) + important | **赢**（important 优先） |
| antd `.hash.ant-input-outlined { border-width/style/color }` | (0,2,0) 非 important | 输 |
| antd `.hash.ant-input-affix-wrapper > input.ant-input { border: none }` | (0,3,1) 非 important | 输（important 优先） |
| antd `.hash.ant-input-affix-wrapper > input.ant-input:focus { box-shadow: none !important }` | (0,4,1) **important** | **赢**（important 相等时比特异性） |
| 项目 `.ant-input:focus { box-shadow: 0 0 0 2px ... !important }` | (0,2,0) important | 输 → 内层 input 无焦点环 |

**这解释了为什么 `src/index.css` 里 `!important` 密度高达 1362 处** —— 不带 `!important` 的项目规则在 antd 6 下**几乎必然失效**。

**Why it happens:**
antd 5 时代 `hashed` 也存在，但很多项目只覆盖"根类"；antd 6 引入了 `cssVar` 模式 + 变体类 + 更细的子选择器，antd 侧特异性整体抬高，项目侧被迫全面升级为 `!important`。

**How to avoid:**
- 新增/修改 antd 覆盖规则时，**一律带 `!important`**，并**核算特异性**（不要只靠"写在后面"）。
- 需要**降低**特异性以避免压制 antd 其它状态样式时，用 `:where()`（零特异性），而不是靠选择器缩短。
- 反过来，需要**提高**特异性压住 antd 时，用「上下文 + 直接子元素」而不是靠堆类名。
- CI 可加一条：新增的 `.ant-*` 覆盖规则若不带 `!important` 且特异性 < (0,2,0) → 告警。

**Warning signs:**
- 规则"写了没效果"，但把 `!important` 加上就立刻生效 → 命中 antd hashed 规则；
- 同一条规则在 dev server 有效、在 `npm run build` 产物失效 → hash 变化导致特异性/顺序变化；
- 改 antd 主题 token 后部分覆盖失效 → `hashed: true`（`AntdThemeBridge.tsx:173`）重新生成 hash。

**Phase to address:** P1 样式修复 + P6 lint 清偿（可加自定义 lint 规则）。

---

### Pitfall A5：affix wrapper 的结构差异 —— 修 Input 时会漏掉 TextArea / InputNumber / Select / DatePicker / Cascader / TreeSelect

**What goes wrong:**
只盯着 `Input` 修，修完发现「带字数统计的多行输入框」还是双框；再修 TextArea，发现 `InputNumber` 加了前后缀按钮时也双框。**因为 antd 6 里"边框承载元素"因组件而异，且 affix 场景下 variant 类会"搬家"。**

**antd 6.6.4 实测的边框承载表：**

| 组件 / 场景 | 边框承载元素 | 该元素 class | 内层元素 class |
|---|---|---|---|
| Input 无 affix | `input` 自身 | `ant-input` + **`ant-input-outlined`** | — |
| Input 有 affix（allowClear/prefix/suffix） | wrapper `span` | `ant-input-affix-wrapper` + `ant-input-outlined` | `input.ant-input`（**无 variant 类**） |
| TextArea 无 affix | `textarea` 自身 | `ant-input` + `ant-input-outlined` | — |
| TextArea 有 showCount/allowClear | wrapper `span` | `ant-input-textarea-affix-wrapper` + `ant-input-affix-wrapper` + `ant-input-outlined` | `textarea.ant-input`（**无 variant 类**） |
| InputNumber | 根 `.ant-input-number` | `ant-input-number` | `.ant-input-number-input`（**不带 `ant-input`**，不命中项目 `.ant-input`） |
| Select / Cascader / TreeSelect | `.ant-select-selector` | — | — |
| DatePicker | `.ant-picker` 根 | — | `.ant-picker-input > input` |

**关键机制（variant 类"搬家"）** —— `node_modules/@rc-component/input/es/BaseInput.js`：
- `:43-45` — `className: clsx(inputElement.props?.className, !hasAffix && classNames?.variant)`
  → **有 affix 时，内层 input/textarea 拿不到 variant 类**
- `:79-88` — wrapper 的 className 合并 `classNames?.variant`
  → **variant 类落在 wrapper 上**
- 佐证：`node_modules/@rc-component/input/es/TextArea.js:137` — `isPureTextArea = !autoSize && !showCount && !allowClear`；`:179` textarea 元素 class 只取 `classNames?.textarea`；`node_modules/@rc-component/input/es/ResizableTextArea.js:114` — `className: clsx(prefixCls, className, ...)` → textarea 元素 = `ant-input` + size/hash 类。

**项目侧已有的不一致：** `src/index.css:871-873` 把 `.ant-input` / `.ant-input-number` / `.ant-select-selector` 并列当边框承载 —— **漏了 `.ant-picker`（DatePicker）**。DatePicker 走 antd 默认边框（`index.css:2321-2323` 只在暗色块里给 `.ant-picker` 设了 `background-color`/`border-color`，亮色下完全不管）→ **DatePicker 的边框色与 Input 不一致**（暗色用 `--input-border` #2d4d33，亮色用 antd token `colorBorder` = `ruoyiBrand.cream.border`）。这是既存不一致，v1.24 修双框时很容易被"顺手统一"或"顺手再引入回归"。

**How to avoid:**
- 修复规则必须**成对覆盖 Input 与 TextArea** 的 affix 内层：
  ```css
  .ant-input-affix-wrapper > input.ant-input,
  .ant-input-affix-wrapper > textarea.ant-input { ... }
  ```
  （`textarea.ant-input` 这一条覆盖 TextArea 的 showCount/allowClear 场景）
- 修复前先跑一遍**全站 affix 使用面统计**：`allowClear`（AppInput 默认开！见 A6）、`prefix`、`suffix`、`showCount`、`InputNumber` 的 `addonBefore/addonAfter`、`Select` 的 `showSearch`。
- **不要**顺手把 `.ant-picker` 加进边框规则 —— 那是独立的一致性议题，应单开条目，否则 DatePicker 的 hover/focus 态会与 Input 不匹配（项目没有 `.ant-picker:hover` / `:focus` 规则）。
- Select 侧确认无需改：项目已正确地把 `.ant-select-selector` 当承载元素（`index.css:873`），且 `.ant-select-selector` 内不含 `ant-input` 类 → 不会被 `.ant-input` 误伤。

**Warning signs:**
- 修复后「新建楼宇」弹窗里带字数统计的描述框仍双框 → 漏了 `textarea.ant-input`；
- InputNumber 加了 formatter/前后缀后双框；
- DatePicker 与相邻 Input 的边框粗细/颜色不一致（既存问题，别在本次扩大）。

**Phase to address:** P1 样式修复（覆盖矩阵见文末《附录 B：双框修复最小充分视觉验证矩阵》）。

---

### Pitfall A6：`AppInput` 默认 `allowClear=true` → 全站 Input 都走 affix-wrapper → 双框影响面 = 100%

**What goes wrong:**
低估影响面。`src/components/ui/AppInput.tsx:12`：
```tsx
export function AppInput({ allowClear = true, ...rest }: AppInputProps) {
  return <Input allowClear={allowClear} {...rest} />;
}
```
→ **所有契约输入框都带 allowClear → antd 一律渲染成 affix-wrapper → 内层 input 一定拿到项目 `.ant-input` 的边框 → 100% 双框**。

这解释了为什么 v1.23 上线 AppInput 契约后双框立刻成为"全站可见"的问题，而不是零星现象。也意味着：**修复的验收面 = 全站每一个输入框**，而不是"某个页面的某个输入框"。

**Why it happens:**
v1.22 契约把 `allowClear` 默认值下沉（见 v1.23 PITFALLS 的 Pitfall 1.1/2.3），而 antd 的 `allowClear` 必然触发 affix-wrapper 结构 —— 契约层的一个"体验增强"，在样式层触发了一次结构切换。**契约默认值与组件库 DOM 结构之间没有隔离层**。

**How to avoid:**
- 视觉验证必须按"**所有输入框都是 affix**"的前提设计，不能只测裸 `<Input>`。
- 反向检查：是否存在**没有** allowClear 的 Input（例如显式 `allowClear={false}`）→ 那些是"顶层 input"，走 `.ant-input-outlined` 分支 → **两条渲染路径都要验证**。
- 排查既有裸 Input 存量：`eslint.config.js:126` 的 P65 selector 已拦截裸 `<Input>`，v1.23 已把 P65 豁免清零 → 存量裸 Input 理论上为 0；仍应 grep 一次 `from "antd"` 里的 `Input` 直接使用作为交叉验证。

**Warning signs:**
- 修复只测了「显式 allowClear={false}」的输入框 → 恰好测的是没问题的分支；
- 某个页面输入框没有 clear 图标却仍是双框 → 它的 DOM 结构与其它不同（可能是 Input.Search / Input.Password / Input.OTP 等包装组件）。

**Phase to address:** P1 样式修复（验收矩阵的输入源选择）。

---

### Pitfall A7：暗色模式 —— 裸标签选择器 `[data-color-mode="dark"] input` 会绕过所有类选择器收窄

**What goes wrong:**
**暗色块里有一批「元素标签选择器」规则**（`src/index.css`）：

| 行号 | 选择器 | 特异性 | 作用 |
|---|---|---|---|
| `:2252-2259` | `[data-color-mode="dark"] input, textarea, .ant-input, .ant-input-number, .ant-input-affix-wrapper` | (0,1,1) / (0,1,0) | `background-color` / `color` / `border-color` `!important` |
| `:2268-2272` | `[data-color-mode="dark"] input:hover, textarea:hover, .ant-input:hover, .ant-input-affix-wrapper:hover` | (0,2,1) | `border-color` `!important` |
| `:2275-2281` | `[data-color-mode="dark"] input:focus, textarea:focus, .ant-input:focus, .ant-input-affix-wrapper:focus, .ant-input-affix-wrapper-focused` | (0,2,1) | `border-color` `!important` |

**关键点：`** `[data-color-mode="dark"] input` 是**标签选择器**，特异性 (0,1,1) **高于**任何单个类选择器（如 `.ant-input-outlined` (0,1,0)）。它会命中**所有** input 元素，**包括 affix-wrapper 内层 input**。

**因此：**
- 如果采用「把 `.ant-input` 改成 `.ant-input-outlined`」的收窄路线 → 亮色下内层 input 确实没有边框了（因为 `.ant-input-outlined` 不命中内层），**但暗色下 `[data-color-mode="dark"] input { border-color: var(--input-border) !important }` 仍然命中内层**。虽然只有 `border-color` 没有 `border-width/style`，**视觉上不会出边框**（border-color 单独不生成边框）→ 侥幸不炸。
- 但 `background-color: var(--input-bg) !important` 会**让内层 input 背景不透明**。在 antd 的 `filled` 变体或未来任何 wrapper 背景 ≠ input 背景的场景下 → **wrapper 与内层出现色差**（"边框内一圈灰"）。
- 采用「wrapper 上下文复位」路线（`.ant-input-affix-wrapper > input.ant-input`，特异性 (0,3,1)）→ **稳压** (0,1,1) 的标签规则 ✓ 暗色同样正确。

**暗色 token 事实**（`src/index.css`）：
- light 块 `:150-196`：`--input-bg: #ffffff`（`:184`）、`--input-border: #dbd7ce`（`:185`）、`--input-border-hover: #598e5e`（`:186`）、`--input-border-focus: #156031`（`:187`，注释标 COMP-04）、`--input-text: #101010`（`:188`）、`--input-placeholder: #707068`（`:189`）
- dark 块 `:198-268`：`--input-bg: #0f2e1b`（`:254`）、`--input-bg-hover`（`:255`）、`--input-bg-focus`（`:256`）、`--input-bg-disabled`（`:257`）、`--input-border: #2d4d33`（`:260`）、`--input-border-hover: #598e5e`（`:261`）、`--input-border-focus: #1a6839`（`:262`）、`--input-text: #f0ece3`（`:266`）
- **注意**：dark 块定义了 `--input-bg-hover` / `--input-bg-focus` / `--input-bg-disabled` / `--input-border-error` / `--input-border-disabled` / `--input-text-placeholder` / `--input-text-disabled`，但**项目 CSS 里没有任何规则消费它们**（`.ant-input` 规则用的是 `--input-bg` / `--input-border` 单一值）→ 一批死 token。

**How to avoid:**
- 选「wrapper 上下文复位」路线，天然规避标签选择器问题。
- 修复后**必须**在暗色下截图验证：① 输入框内层与 wrapper 背景是否同色；② 内层是否还有任何 1px 线；③ hover/focus 时内层是否变色（`[data-color-mode="dark"] input:hover` 会给内层改 `border-color` —— 无 border-width 则无视觉，但若未来内层有了 border-width 就会立刻暴露）。
- 顺手把 `--input-bg-hover` / `--input-bg-focus` / `--input-border-error` 等**死 token**登记为已知欠账（不要在本次顺手接线，会扩大回归面）。

**Warning signs:**
- 暗色下输入框内部有一圈比外部略浅/略深的矩形；
- 暗色下 hover 时输入框"内层"颜色变深；
- 只在暗色下出现的双框（多半是标签选择器与新规则的特异性竞争）。

**Phase to address:** P1 样式修复。

---

### Pitfall A8：焦点环 —— 修复后焦点环会**外扩 1px**，且必须确认只出现一层

**What goes wrong:**
对 `Input` 焦点态做判断时，误以为会看到"两圈焦点环"。实际机制：

**焦点环画在哪个元素上：**
- antd 的 `genBaseOutlinedStyle`（`node_modules/antd/es/input/style/variants.js:34-39`）：
  ```js
  '&:focus, &:focus-within': { borderColor: options.activeBorderColor, boxShadow: options.activeShadow, outline: 0, backgroundColor: token.activeBg }
  ```
  → **wrapper 上的 `:focus-within` 会在焦点落在内部 input 时命中** → 焦点环由 **wrapper** 承担。
- antd 显式关掉内层焦点环：`input/style/index.js:392-394` `&:focus { boxShadow: 'none !important' }` → 实际特异性 (0,4,1)（含 hash）。
- 项目 `.ant-input:focus { box-shadow: 0 0 0 2px var(--theme-primary-lighter) !important }`（`index.css:888-893`）(0,2,0) → **打不过 antd 的 (0,4,1)** → 内层无焦点环。
- 项目 `.ant-input-affix-wrapper-focused { box-shadow: 0 0 0 2px var(--theme-primary-lighter) !important }`（`index.css:920-923`）→ wrapper 上的焦点环。
- 暗色块 `:895-901` / `:925-929` 给两处各加了一层 `0 0 12px rgba(21,96,49,0.1)` 的外发光。

**结论：现状（双框）下焦点环本来就只有一圈（wrapper 上的），但它紧贴内层那条多余边框的外沿。** 修复后内层边框消失 → **焦点环整体向外扩 1px**。这在视觉上是一个"细微但可察觉"的位移，**必须截图对比确认没有"跳动感"**。

**另一个必须防的坑：** 若修复时顺手给内层 input 加了 `box-shadow: none`（不带 `!important`），会被 antd 的 `:focus-within` 逻辑绕过；若加的 `box-shadow: none !important` 落在**过宽的选择器**上（例如 `.ant-input:focus`），会把 **wrapper 上的焦点环也一起干掉**（因为 `:focus-within` 匹配的元素是 wrapper，但 `.ant-input:focus` 匹配的是内层 —— 两者不同元素，但如果规则写成 `.ant-input-affix-wrapper:focus-within` 就会误伤）。

**How to avoid:**
- 修复规则里**不要出现 `box-shadow`**（内层焦点环交给 antd 的 `none !important`）。
- 验收时逐个状态截图：默认 / hover / focus（wrapper 上应恰好一圈 2px 品牌绿光晕 + 一圈 12px 外发光，暗色下叠加）。
- 同时验证 **TextArea 的 focus**（antd 走 `textarea.js:58-60` 的 `boxShadow: 'none !important'`）与 **InputNumber 的 focus**（项目 `.ant-input-number:focus` 规则 (0,2,0) 与 antd InputNumber 根的 focus 态竞争）。

**Warning signs:**
- 聚焦后焦点环变成"双层"（内层一圈 + 外层一圈）；
- 聚焦后焦点环消失；
- 暗色下焦点环的颜色与亮色不一致（两套 box-shadow 定义分散在 `:888-901` 和 `:920-929` 两处，容易只改一处）。

**Phase to address:** P1 样式修复。

---

### Pitfall A9：视觉层没有自动化守卫 —— v1.23 的四道闸门全部对双框失明

**What goes wrong:**
双框是 v1.23 引入的回归，而 v1.23 的所有闸门（`npm run type-check` / `npm run test` / `npm run lint` / `npm run build`）**全部通过**。原因不是闸门坏了，而是**闸门覆盖的是"代码正确性"，双框是"渲染正确性"**：
- type-check：CSS 语法合法 → 绿；
- vitest：jsdom **不加载 `src/index.css`**，也不做布局计算 → 无边框概念 → 绿；
- eslint：本项目 eslint 规则集里**没有 CSS 规则**（`eslint.config.js` 只有 JS/TS 规则 + 一个自定义 `local/no-large-dropdown-list`）→ 绿；
- build：CSS 会被打包，但 Vite 不校验视觉效果 → 绿。

**Why it happens:**
"闸门全绿 = 没问题"是流程惯性。实际上**没有任何一道闸门能看见"多了一条 1px 边框"**。

**How to avoid:**
本里程碑**建议**补一条**轻量**守卫（二选一，按成本排序）：
1. **CSS 契约单测（推荐，成本最低）**：用 vitest 读取 `src/index.css` 文本，断言「不存在对 `.ant-input` 无条件设置 `border` 的规则」/「存在 affix-wrapper 内层复位规则」。这类"样式契约测试"不需要浏览器，秒级，能挡住**同形态复发**（有人再写一条 `.ant-input { border: ... !important }`）。
2. **结构断言测试**：用 jsdom 渲染 `<AppInput />`，断言 affix-wrapper 的内层 input **不存在**（antd 的 `allowClear` 在 jsdom 下会渲染出 `.ant-input-affix-wrapper` 与内层 `input.ant-input`）→ 至少锁住"结构"这一半。
3. （不推荐本次做）截图对比基线：需要引入 Playwright/Puppeteer，与本项目"零新依赖"约定冲突（见 v1.23 UAT 约定用 `agent-browser` skill）。

**Warning signs:**
- 修完之后任何人往 `src/index.css` 加一条 `.ant-input { ... }` 都能再次引入双框，而 CI 全绿；
- 下一个里程碑再次出现"视觉回归"类问题。

**Phase to address:** P1 样式修复（守卫与修复同 phase 落地，不要留到收尾）；P7 结构整理（拆分文件时一并加断言）。

---

### 领域 B：功能层

### Pitfall B1：#16 的真正根因不是 fail-open，而是**行操作层根本没有权限门控**

**What goes wrong:**
v1.23 UAT（`.planning/milestones/v1.23-phases/75-uat/75-UAT.md:51, 84-87`）给出两个根因候选：
> ① 前端 `hasPerm` 类判定在权限列表获取失败/为空时默认放行危险按钮（fail-open）；② 本地 sqlite 手工插库角色未触发权限缓存加载，后端下发的权限集异常。

**两个候选经源码核对都不成立**：

**候选①不成立 —— 楼宇页没有 hasPerm，也没有 fail-open：**
- `src/pages/operations/buildings/index.tsx:374-378`：
  ```tsx
  const rowActions = (record: Building) => [
    { key: "edit", label: "编辑", icon: <EditOutlined />, onClick: () => openModal(record) },
    createDeleteAction(record.id, handleDelete, { target: `楼宇 ${record.name}` }),
  ];
  ```
  → **无条件渲染**，没有任何权限判断。整个 `src/pages/operations/buildings/` 目录 grep `permission` **零命中**。
- `src/components/shared/ActionButtons.tsx:15-24` —— `ActionButton` 接口**根本没有 `permission` 字段**；`:114-159` `createDeleteAction` 的 options 里也没有权限参数 → **共享组件层也不具备门控能力**。
- 对照组证明"部分页面有门控"：`src/pages/network/ports/index.tsx:131-132`
  ```ts
  const hasPermission = (perm: string) => menuPermissions.includes(perm);
  const canWrite = hasPermission("network:port:write");
  ```
  → 端口页做了，楼宇页没做。所以**#16 是"契约缺失"而不是"实现 bug"**。

**候选②不成立 —— 权限链路本身是自洽的：**
- 前端权限来源：`src/lib/menuApi.ts:23-26` → `POST /system/my-menus/permissions`
- 后端 handler：`internal/api/v1/system/menu_handler.go:335-348`
- 后端聚合：`internal/services/system/menu_service.go:509-569`
  `sys_user_role` → `sys_role_menu` → **`:543` `appendAncestorMenuIDs`（自动补所有祖先菜单）** → `sys_menu.perms` 去重
  → **不区分菜单类型**（菜单/按钮都收）、**不过滤 `status`**、**`perms` 直接 Pluck 整串（不拆逗号）**
- 后端权限码口径（`internal/core/db/init_data.go:693, 752-755`）：
  ```
  楼宇管理菜单   ops:building:list
  楼宇查询       ops:building:query
  楼宇新增       ops:building:add
  楼宇修改       ops:building:edit
  楼宇删除       ops:building:delete
  ```
  → **编辑 = `:edit`、删除 = `:delete`**，天然就是行操作门控所需的映射。
- `pkg/permission/config.go:223-290` 的 `GetRoutePermissions()` 是**静态的「路由 → 权限码」表**（路由级授权用），**不是**用户权限集，且**每次调用返回新切片、无任何缓存** → 用户猜测的"后端 GetRoutePermissions 缓存"**不存在**。真正有缓存的是 `internal/services/system/menu_cache_impl.go:115-120`：
  ```go
  GetMenuUserPermissionsKey(userID)  // menu:user:perms:{userID}
  s.GetExpiration(services.CacheConfigMenuTree, 30*time.Minute)  // 30 分钟 TTL
  ```

**Why it happens:**
UAT 的排查在 sqlite 环境 + 手工插库的环境下进行，两个变量叠加导致"看起来像权限问题"，实际是"根本没做权限控制"。**行操作权限从来不是契约的一部分** —— v1.22 契约体系（AppInput/AppSelect/PageTable/SearchForm/FormModal/ConfirmDelete）里**没有"行操作权限"这一维**。

**How to avoid（v1.24 定责的判别方法，按顺序执行，每步都能单独终止）：**
1. **先查库，判定"后端是否发了写权限"**（这是区分"后端发了写权限给只读角色" vs "前端没门控"的唯一硬判据）：
   ```sql
   SELECT m.id, m.name, m.menu_type, m.perms
   FROM sys_user_role ur
   JOIN sys_role_menu rm ON rm.role_id = ur.role_id
   JOIN sys_menu m ON m.id = rm.menu_id
   WHERE ur.user_id = '<受限用户UUID>';
   ```
   - 结果里出现 `ops:building:edit` / `ops:building:delete` → **后端把写权限发给了只读角色**（数据问题：角色勾选了按钮菜单，或 `appendAncestorMenuIDs` 补祖先时带上了带 perms 的父菜单）→ 修数据，不改前端。
   - 结果里**只有** `ops:building:list` / `ops:building:query` → **后端口径正确，前端缺门控** → 走第 2 步。
2. **再查后端实际下发**（用正确端点！）：`POST /system/my-menus/permissions`。
   > ⚠️ UAT 记录（`75-UAT.md:51`）写的是「`/system/menus/permissions` 对该用户 403」——**这个端点前端根本不调用**（`src/lib/menuApi.ts:24` 用的是 `/system/my-menus/permissions`）。用错端点会得出"后端不下发权限"的错误结论。
3. **再查缓存**：`menu:user:perms:{userID}` 有 30 分钟 TTL（`menu_cache_impl.go:118`）。角色↔菜单变更走 `InvalidateUserMenuCache`（`menu_cache_impl.go:157-162`），菜单本身的写操作走 `InvalidateMenuCache`（`:122-136`）。**手工插库绕过全部失效路径** → PG 复测时必须走 UI 正常路径，或手动清 key。

**Warning signs:**
- 任何"看起来像 fail-open"的表述都要先被"这里到底有没有 hasPerm"证伪/证实；
- UAT 记录里引用了前端不调用的端点；
- 排查时把"路由→权限码静态表"（`pkg/permission/config.go`）与"用户权限集"混为一谈。

**Phase to address:** P2 #16 定责（第 1~3 步是"定责"，不写代码）。

---

### Pitfall B2：#16 的两难 —— fail-closed 会造成"本该能用的按钮也看不见"

**What goes wrong:**
若 #16 定责为"前端缺门控"，最直接的修法是"默认隐藏危险按钮，权限到了才显示"（fail-closed）。但这会引入**可用性回归**：
- 权限列表来自 `POST /system/my-menus/permissions`，**是一次异步网络请求**。在它返回之前 `menuStore.permissions` 是 `[]`（`src/store/menuStore.ts:55` 初始值）→ **首屏所有危险按钮全部消失**；
- 权限请求失败（网络抖动、token 刷新失败、后端 5xx）→ `permissions` 永远为空 → **管理员也看不到删除按钮**，且**没有任何提示**（用户只会以为"系统没有这个功能"）；
- 菜单缓存（`MENU_CACHE` sessionStorage）能缓解首屏空窗，但 `DynamicRoutes.tsx:179-186` 的 hydrate 写的是 `cachedMenuData.permissions`，**缓存缺失或版本不匹配（`MENU_CACHE_VERSION`）时仍然为空窗**。

反过来，若采用 fail-open（默认显示），就等于没修 #16。

**业界常见折中（按本项目适配度排序）：**

| 方案 | 机制 | 适配度 |
|---|---|---|
| **A. 三态门控（推荐）** | `permissions` 未加载完成 → 按钮**禁用**（保留可见性 + `aria-disabled`），加载完成且有权限 → 可用；加载完成且无权限 → **不渲染** | 高。既不误显示，也不误隐藏；用户看得见"这里有个删除"但暂时点不动 |
| B. 后端推送式能力 | 由后端在列表数据里回传 `canEdit/canDelete` 标志 | 低。需要改 N 个列表接口，范围失控 |
| C. 页面级 `AuthGate` | 在列表页外层包一次权限判定，缺权限则整块降级 | 中。粒度粗（整个操作列一起没），但实现简单 |
| D. 纯 fail-closed + 显式错误态 | 权限加载失败时显示"权限信息加载失败，点击重试" | 中。诚实但需要额外 UI |

**本项目推荐做法（A 的具体落地）：**
1. `ActionButton` 接口新增**可选** `permission?: string`（**向后兼容**，存量调用点不传即保持原行为）；
2. `createDeleteAction` 的 options 新增**可选** `permission?: string`；
3. 门控判定集中到**一个** helper（不要在每个页面各写一遍）：
   ```
   权限未就绪 → 渲染按钮但 disabled（title/aria-label 说明"权限加载中"）
   权限就绪 & 含该码 → 正常
   权限就绪 & 不含该码 → 不渲染
   ```
4. 「权限未就绪」的判定必须基于**明确的加载态**（`menuStore` 的 `loading` / `error` 字段 + `permissions` 长度），**不能**用 `permissions.length === 0` 判空 —— 空权限集与未加载完在数据上不可区分。
5. 危险操作（删除）**必须**在**服务端**也校验（后端 `pkg/middleware/permission.go` + `internal/api/router.go` 已有 `ops:building:delete` 授权）→ **前端门控是 UX，不是安全边界**（与 `src/router/RouteGuard.tsx:5` 的既有注释口径一致）。

**Why it happens:**
"隐藏按钮"看起来是零成本的 UX 修补，但它把一个**同步可判定**的问题（这个用户有没有这个权限）变成了**依赖异步状态**的问题。权限数据未就绪这个中间态被忽略，就会把"偶尔按钮消失"变成"用户投诉功能丢失"。

**Warning signs:**
- 修复后用户报"刷新页面后删除按钮闪一下才出现"；
- 修复后弱网/接口失败时管理员也看不到删除按钮，且无提示；
- 修复只改了 1~2 个页面，其它页面仍然无门控（**#16 复测必须覆盖多个列表页**，否则只是"点过"而不是"修好"）。

**Phase to address:** P2 #16（门控契约 + helper 落地）。

---

### Pitfall B3：#16 的范围膨胀 —— 「行操作权限」在 v1.24 是**全站契约化**，不是单点 bug

**What goes wrong:**
把 #16 当成"修一个楼宇页"来做。实际盘点后：
- `src/components/shared/ActionButtons.tsx` 的 `ActionButton` / `createDeleteAction` 是**全站行操作唯一入口**（契约组件）；
- 行操作分布：楼宇、工位、机房、楼层、网络设备、用户/角色/部门/岗位/字典/菜单、通知、值班、工单、对账……**数十个页面**；
- 每个页面的权限码**不统一**：ops 走 `ops:building:edit` / `ops:building:delete`（`init_data.go:752-755`），system 走 `system:user:edit` / `system:user:remove`（`pkg/permission/config.go` 里 `UserEdit` / `UserRemove`），network 走 `network:port:write`（**单一码覆盖全部写操作**，`config.go:280-288`）→ **不存在统一的「行操作 → 权限码」约定**。

**Why it happens:**
UAT 把 #16 记成一条"行操作权限不变"的回归条目，读者会以为曾经有过这个契约。实际上 v1.22 的契约体系里没有这一维。

**How to avoid:**
- v1.24 **明确切范围**：P2 只做「**契约 + 样板 + N 个代表页面**」，并在 phase 验收里写清"本里程碑不覆盖全部页面"，剩余页面登记为 Known Gaps。
- 建立**权限码映射表**（模块 × 操作 → 权限码），作为契约的单一真相源；映射缺失的模块**默认 fail-closed + 显示禁用态**，而不是"默认显示"。
- 特别处理 `network:port:write` 这类**一码多操作**的情况：要么拆分权限码（后端改动大），要么在契约层允许"一个 permission 覆盖一组操作"。

**Warning signs:**
- PR diff 出现 30+ 文件的权限改动；
- 映射表靠"猜"（前端自己写字符串字面量，与后端 `init_data.go` 不同步）→ 必须有一致性校验脚本；
- 只测了 1 个页面就宣布 #16 关闭。

**Phase to address:** P2 #16（先切范围，再定契约）。

---

### Pitfall B4：3f130 空因诊断 —— 后端注释自相矛盾，且"有数据 + 空因"会同时出现

**What goes wrong:**
v1.24 要做「3f130 空因诊断前端展示」。后端已就绪（commit `8b85fe6b`）：
- 路由：`internal/api/router.go:702` → `POST /ops/workstation-device/:id/physical-diagnosis`
- handler：`internal/api/v1/operations/workstation_device_handler.go:147-191`
- 空因常量：`internal/services/operations/workstation_physical_diagnosis.go:27-43`
  ```
  PhysicalEmptyReasonNone                = ""                 // 有数据
  PhysicalEmptyReasonNoUserBound         = "no_user_bound"
  PhysicalEmptyReasonInfoPointMisbound   = "info_point_misbound"
  PhysicalEmptyReasonDeviceOrMACMissing  = "device_or_mac_missing"
  ```
  字段 `EmptyReason string \`json:"emptyReason,omitempty"\``（`:57`）→ **空串即 JSON 省略**。
- 判定：`internal/services/operations/workstation_device_service.go:800-867` `classifyPhysicalEmptyReason`
- 告警：同文件 `:869-887` `collectPhysicalWarnings`

**⚠️ 后端注释自相矛盾（必须先对齐再写前端）：**
- `:804` 「未绑定用户 → no_user_bound ← **最高优先级，压过「有数据」**」
- `:809-812` 「**未绑用户压过「有数据」**……设备本身的数据不丢，仍在 Devices 里，同时 Warnings 也会提示该告警，两个通道互补」
- `:820` 「若此处对 deviceCount>0 返回 ""，需求「可区分空因状态」对 3f130 就没有解决」
- **但** `:871-874`（`collectPhysicalWarnings` 的注释）却写：「此时 devices 非空 → **emptyReason 必须为空串**（否则就是误报空因）……故用 Warnings 承载」

**以代码为准的实际行为：**
`classifyPhysicalEmptyReason` 的 `:822` `if !diag.UserBound { return no_user_bound, ... }` 是**无条件先判**（在 `deviceCount > 0` 判断之前）→ 3f130 的实际响应是：
> `devices` **非空**（mac_match_count=1）+ `emptyReason = "no_user_bound"` + `warnings` 里**也**有「未绑定用户」告警（`:879` 无条件追加）。

**这正是用户描述的 UI 矛盾：「有数据但 emptyReason=no_user_bound」——用户看到设备列表，却同时看到"导致为空"的空因与警告。**

**前端展示的正确做法：**
- **`devices.length > 0` 时**：把 `emptyReason` 当成**根因告警**而不是"空的原因"；UI 上用 **warning 样式（Alert warning）** 呈现，标题措辞用「该工位未绑定用户，物理链路按用户维度归属，可能显示不全」，**不要**用"为空/无数据"的空态文案。
- **`devices.length === 0` 时**：用 `emptyReason` + `emptyMessage` 渲染**空态**（这是它被设计出来的场景）。
- **不要**在有数据时同时渲染「空态组件」+「有数据的列表」—— 那是把两套互斥的语义叠在同一块 UI 上。

**前端现状（尚未接入）：**
- `src/components/operations/WorkstationDeviceTable/index.tsx:109-128` —— 物理链路仍走**旧端点**，失败时只 `console.warn(physicalResult.reason)`；`:580-591` 折叠面板标题写死「物理链路设备（{physicalDevices.length}台）」，**0 行也显示**。
- 全仓 `grep emptyReason|no_user_bound` 在 `ruoyi-react-frontend/src` **零命中** → 前端完全没接。

**How to avoid:**
- **第一步不是写 UI，是对齐契约**：把 `workstation_device_service.go:871-874` 的矛盾注释修正（或明确"emptyReason 是根因，允许与 devices 共存"），并加一条**不变量测试**锁住"有数据 + 未绑用户 → emptyReason 仍为 no_user_bound 且 devices 非空"（现有测试 `workstation_physical_diagnosis_test.go:271-303` 只锁了"0 台设备必带非空 emptyReason"，**没锁"有数据时空因仍可非空"**）。
- 前端新增**明确的类型**（不要用 `any` 接 JSON），例如把 `emptyReason` 建模为判别联合：`{ code: "" } | { code: "no_user_bound" | ... }`。
- 3f130 本身**还没绑 user_id**（数据侧动作，见 `.planning/debug/workstation-3f130-physical-link-empty.md:106`）→ **验收前必须先做数据绑定，否则复验必然还是"空白 + 空因"**，会被误判为"前端没修好"。

**Warning signs:**
- UI 上出现"共 1 台设备" + "无数据" 空态卡片并存；
- 空因文案说"因此为空"但列表里明明有设备；
- 修复后截图仍显示空白 → 是数据没绑（不是代码问题）。

**Phase to address:** P3 3f130 展示（先契约对齐 → 再前端接入 → 最后绑数据复验）。

---

### Pitfall B5：F5 落 `/dashboard` 的根因不是"缺默认落地页"，而是「深层 URL 落进 `path="*"` 兜底」+「兜底路径反写 LAST_PATH」

**What goes wrong:**
`src/router/DynamicRoutes.tsx`：
- `:283-286` — `<Route path="/" element={<Navigate to={lastPath && lastPath !== "/" ? lastPath : "/dashboard"} replace />} />`
  → 只有 **URL 恰好是 `/`** 时才用 `lastPath`。
- `:315` — `<Route path="*" element={<Navigate to="/dashboard" replace />} />`（Layout 内兜底）
- `:227-240` — `routeElements` = `RouteGenerator.generate(allMenus)` 再 `.filter(routeConfigManager.hasPermission(...))`
  → **无权限的路由根本不进路由树**。
- `:258-260` — `!initialized` → `InitializingFallback`；`:275-277` — `allMenus.length === 0 && !cachedMenuData` → `InitializingFallback`。
- `:243-247` — `useEffect` **无条件** `saveLastPath(location.pathname)`（只排除 `/login`，见 `:95-105`）。
- `:85-92` `getLastPath` / `:108-115` `clearLastPath` / `src/constants/storage.ts:37` `LAST_PATH = "ruoyi_last_visited_path"`。

**F5 掉到 `/dashboard` 的完整链路：**
1. 用户在 `/operations/buildings` → LAST_PATH = `/operations/buildings`。
2. F5 → **URL 仍是 `/operations/buildings`**（不是 `/`）→ `:283` 的 `/` 路由**不匹配**。
3. 若此刻动态路由尚未就绪（`allMenus` 仍空、或该路径因无权限被 filter 掉）→ 命中 `:315` 的 `*` → **强制跳 `/dashboard`**。
4. 跳转完成后 `:243-247` 的 effect 把 `/dashboard` **写回 LAST_PATH** → **原路径被自毁式覆盖**，再也回不去。
5. UAT 已实证第 4 步：「F5 刷新后 SPA 落到 `/dashboard` 且 `ruoyi_last_visited_path` 被覆盖为 `/dashboard`」（`75-UAT.md:76`）。

**所以「改路由 fallback 让 F5 落 `/dashboard`」这个命题本身就是错的** —— F5 **已经**在落 `/dashboard`，问题是**不该落**。

**改 `path="*"` 的 fallback 会连带破坏的既有行为：**
| 既有行为 | 位置 | 改 fallback 的影响 |
|---|---|---|
| 未登录 → 跳登录页 | `:263-270`（`*` → `/login`） | 若把兜底统一改成"落 dashboard"，未登录用户会被送到需要鉴权的页面 → 无限重定向 |
| 无权限 deep link → 落到某处 | `:238` 把无权限路由 filter 掉 → 命中 `*` | 无权限用户应看到 403/提示，而不是被静默送到 dashboard（"看起来有权限"） |
| deep link 直达（`/system/notice/:id`） | `:299-306` `RouteGuard` + `fallback="/system/notice"` | `RouteGuard` 的 fallback 依赖"匹配不到"时的行为；改兜底会与之抢路由 |
| 静态资源路径 | `:249-255` `isUploadsPath` → `UploadsFallback` | 不受影响，但需保留该早退分支 |

**How to avoid:**
- **正确的修法是三件事一起做**：
  1. **菜单/路由未就绪时不要渲染 `*` 兜底**：`:275-277` 已经用 `InitializingFallback` 挡住了一部分情况，但**只挡住了 `allMenus` 为空**；一旦 `allMenus` 有内容而**目标路径不在其中**（无权限/路径写错/菜单还在加载），仍会命中 `*` → 应当在"目标路径尚未在 `routeElements` 中出现"时保持 loading 或渲染 403，而不是立刻跳 dashboard。
  2. **不要把兜底跳转产生的路径写回 LAST_PATH**：`:243-247` 需要排除"由 `*` 兜底触发的跳转"（例如在跳转前记录一个"正在兜底"的 ref，或改为只在 `routeElements` 命中当前路径时保存）。
  3. **LAST_PATH 要有版本号**：`MENU_CACHE` 有 `MENU_CACHE_VERSION`（`DynamicRoutes.tsx:74-78`），而 `LAST_PATH` **没有** → 跨版本/跨发布残留的旧路径会被直接 `Navigate` 过去（可能 404 → 又命中 `*` → 又跳 dashboard，形成"弹跳"）。
- **回归验证矩阵**（每条都要实际点一遍）：
  | 场景 | 期望 |
  |---|---|
  | 在 `/operations/buildings` 按 F5 | 仍停 `/operations/buildings`，LAST_PATH 不变 |
  | 在 `/dashboard` 按 F5 | 停 `/dashboard` |
  | 直接粘贴 URL 打开一个**有权限**的深层路径 | 直达该页 |
  | 直接粘贴 URL 打开一个**无权限**的深层路径 | 403 / 明确提示，**不是**静默跳 dashboard |
  | 未登录时打开任意深层 URL | 跳 `/login`，登录后回到原路径（`LAST_PATH` 语义） |
  | 登出 | `authStore.ts:130` 清 LAST_PATH → 再登录不回旧页 |
  | 直接访问 `/` | 跳 `lastPath` 或 `/dashboard`（现有语义不变） |

**Warning signs:**
- 修完 F5 后，用户刷新会看到"闪一下 dashboard 再跳回原页"（说明是 `*` 兜底 + `Navigate` 的竞态）；
- LAST_PATH 里出现 `/dashboard` 且用户抱怨"回不去刚才的页面"；
- 未登录用户被送到 dashboard 后白屏/无限重定向。

**Phase to address:** P4 F5 落地页（先做回归矩阵，再改代码）。

---

### Pitfall B6：react-refresh 清零 —— `export type` 会被 `verbatimModuleSyntax` 强制要求，且**运行时完全无差别**

**What goes wrong:**
react-refresh 规则要求"一个文件只导出组件"。清零 33 处 warning 的自然做法是"把非组件导出改成 `export type`"（对类型导出）或"把常量/函数挪到单独文件"（对值导出）。这里有一个**编译期与运行时的关键分界**：

**结论（基于 `node_modules/eslint-plugin-react-refresh/index.js` 实测）：**
- `:193-194` —
  ```js
  } else if (node.type === "ExportNamedDeclaration") {
    if (node.exportKind === "type") continue;   // ← 类型导出直接跳过
  ```
- `:177-178` — `ExportAllDeclaration` 同样 `if (node.exportKind === "type") continue;`

→ **`exportKind === "type"` 的导出既不计入"非组件导出"，也不触发 warning。**

**因此：**
| 写法 | `exportKind` | react-refresh | 运行时 | `verbatimModuleSyntax: true` |
|---|---|---|---|---|
| `export { SomeType }` | `"value"` | **触发 warning** | 保留 import（TS 会保留，因为无法确认是类型） | **TS 报错**（须用 type-only） |
| `export type { SomeType }` | `"type"` | **跳过，不触发** | **完全擦除** | 正确写法 ✓ |
| `export { type SomeType }` | `"type"` | 同上 | 完全擦除 | 正确写法 ✓ |

**编译期交互结论：**
- 项目 `tsconfig.app.json:14` — `"verbatimModuleSyntax": true`（**且未单独设 `isolatedModules`**，TS 5.9 下 `verbatimModuleSyntax` 已隐含其语义）。
- `verbatimModuleSyntax: true` 的含义：**import/export 语句不做类型擦除**——凡是没有 `type` 修饰符的绑定，一律按运行时绑定处理并保留。
- 因此 `export { SomeType }`（SomeType 是纯接口/类型别名）在 `verbatimModuleSyntax` 下**会被 TS 报错**（TS1205 "Re-exporting a type when 'verbatimModuleSyntax' is enabled requires using 'export type'" / TS1484 同类）。
- **运行时无差别**：`export type` 在转译阶段（Vite/esbuild）被完全擦除，产物里不存在该导出 → 任何运行时行为（import 该符号并使用）都会变成 `undefined`/构建错误 —— 所以**只有当该符号确实是纯类型时**改写才安全。

**关键前提（改写前必须逐个确认）：**
1. 该导出**确实是纯类型**（interface / type alias），**不是**「同名 interface + const」的合并导出；
2. **没有**任何 `.js`/`.mjs`/运行时消费方按值引用它；
3. `tsconfig.app.json:39` 的 `exclude` 排除了 `src/**/*.test.ts(x)` —— **测试文件不在 type-check 范围内** → 测试里的 import 写法不会被 `tsc` 拦，但会被 vitest/esbuild 拦 → **改写导出后必须跑测试**。

**How to avoid（清零路线，按类型分流）：**
| 导出种类 | 处理方式 | 影响面 |
|---|---|---|
| 纯类型（interface / type alias） | `export { X }` → `export type { X }` | 零运行时影响；warning 直接消失 |
| 字面量常量（`export const X = "abc"`） | **已豁免**（`eslint.config.js:59` `allowConstantExport: true`，规则实现 `index.js:95` `constantExportExpressions.has(init.type)`） | 无需动 |
| 对象/数组/函数常量 | 挪到 `constants.ts`（非 `.tsx`）或 `xxx.constants.ts` | 需要改 import 路径 |
| Hook / 工具函数 | 挪到 `hooks/` 或 `utils/` | 需要改 import 路径 |
| `export * from "./x"` | 规则 `index.js:180` 直接报 `exportAll` → 必须改成具名 re-export，或把 barrel 排除出 lint 范围 | 需逐个确认 |

**barrel 文件的豁免边界：** `src/components/shared/`（PageTable / FormModal / ActionButtons / statusHelpers / ModernTag）**本身就是 33 处里的重灾区**（5 个文件 7 处）。这些是**契约组件**，`src/components/shared/**` 在 `eslint.config.js:271-283` 已被整目录豁免（只保留内网 IP selector）→ 但**react-refresh 规则不在 no-restricted-syntax 里，不受该目录豁免影响**。→ 契约组件的"常量与组件同文件"是**有意设计**（收拢契约），把它们拆散会破坏"契约组件自包含"的原则。
→ **建议**：react-refresh 33 处清零应**先划出"契约层豁免名单"**（`src/components/shared/**` + `src/components/ui/**`），其余文件再逐一清零；否则会把契约组件拆成一堆碎片文件，违背 v1.22 契约化的初衷。

**Warning signs:**
- 改完后 `npm run type-check` 报 TS1205/TS1484 → 漏改；
- 改完后 vitest 报 "does not provide an export named X" → 某个测试或运行时消费方依赖了该值导出；
- 文件数激增（每个常量一个文件）→ 过度拆分，违背契约意图；
- 契约组件被拆散后，`src/components/shared/` 不再"自包含"。

**Phase to address:** P7 react-refresh 清零（先定豁免边界，再逐文件改）。

---

### Pitfall B7：devices 页拆分 —— `useMemo`/`useCallback` 依赖面 + 共享契约组件的冻结交互

**What goes wrong:**
`src/pages/network/devices/index.tsx` 实测 **714 行**（用户口径 692 行，接近但不等 —— 以实测为准）。结构：

| 行 | 内容 |
|---|---|
| `:81-169` | `getDeviceTableColumns(props)` —— 纯函数（89 行） |
| `:173` | 主组件 `DeviceManagement` |
| `:192-197` | 5 个 `useState` |
| `:205` | `getDeviceTypeText` `useCallback` |
| `:214` | `deviceTypeOptions` `useMemo` |
| `:246` | `sorterMetas` `useMemo` |
| `:262` | `request` `useCallback`（数据请求） |
| `:277` | `deptFilters` `useMemo`（**被 `:16` 注释标为"引用稳定"的联动关键**） |
| `:367-368` | 弹窗态 `useState` |
| `:451` | `buildRowActions` `useCallback`（行操作） |
| `:516` | `reloadRef`（可变 ref 桥接） |
| `:519` | `useEffect` |
| `:524-713` | `columns` `useMemo`（**~190 行**） |

**拆分的连带风险：**
1. **`columns` 的 `useMemo` 依赖面爆炸**：`:524` 的 `columns` 依赖 `getDeviceTableColumns` 的全部入参 + `buildRowActions` + `request` + `sorterMetas` … 拆分后若把 `buildRowActions` 挪到子文件，`useCallback` 的稳定性依赖链会变 → **`columns` 每次渲染都重建 → 表格全量 re-render**（本仓库 `AGENTS.md` 明确警告过 useEffect 依赖不稳定导致无限请求）。
2. **`deptFilters`（`:277`）的引用稳定性是左侧部门树联动的命脉**（`:16` 注释：「左侧部门树 → extraFilters={deptFilters}（useMemo 引用稳定）联动」）→ 拆开后若 `deptFilters` 变成新对象 → `PageTable` 的 `extraFilters` 引用变化 → **触发 `applyFilters` 回第 1 页**（正是 v1.23 FAIL-1 刚修的 `keepPage` 语义所在，`75-UAT.md:74-81`）→ **v1.24 拆分 devices 页会重新引入 v1.23 刚关掉的 bug**。
3. **共享契约组件的冻结交互**：`src/components/network/port-write/` 下 4 个文件承载 v1.19/v1.20.1 的**冻结交互**（`PortWriteModal.tsx:21`、`PortBindingModal.tsx:24`、`SetAccessVlanModal.tsx:21`、`BulkWriteDrawer.tsx:41` 注释明写「submitting 防重 / reason `__custom__` 状态机 / batchInProgress」）。它们与 devices 页**不直接共享文件**，但**共享 `src/components/shared/ActionButtons.tsx` / `ConfirmDelete.tsx`**；devices 页 `:462` 注释还写明「ActionButton 契约（T1 冻结）无 aria-busy/title 透传位」→ **拆分时若"顺手"给 `ActionButton` 加 aria-busy 透传位，会波及 port-write 的冻结交互**（它依赖 ActionButton 现有形状）。
4. **`src/pages/operations/room-devices/index.tsx` 实测 815 行，比 devices 还长** → 只拆 devices 会立刻产出一个新的"超长文件"债务。

**How to avoid:**
- 拆分按**纯函数优先**：`getDeviceTableColumns`（`:81-169`）是纯函数 → 挪到 `components/DeviceTableColumns.tsx`（或 `columns.tsx`）**零 hook 风险**，先做这一刀。
- `columns` 的 `useMemo` **留在主组件**，只把列定义数据挪出去；搬出去后 `useMemo` 依赖**必须逐个核对**（`getDeviceTableColumns` 的入参全部列出来）。
- `buildRowActions`（`:451`）**不要**跨文件搬运 `useCallback` 依赖 —— 若必须搬，把 `handleDelete` / `openModal` 一起搬，否则 `useCallback` 捕获到旧闭包。
- 拆分的**验收**必须包含：① 左侧部门树联动后仍停在当前页（`keepPage` 语义）；② 表格排序/筛选；③ 行操作；④ 弹窗提交后刷新；⑤ `useMemo` 依赖稳定（用 React DevTools Profiler 或加一个"渲染次数"断言）。
- **v1.19 冻结交互**：本里程碑**禁止**修改 `src/components/network/port-write/**` 与 `src/components/shared/ActionButtons.tsx` 的对外形状；如确需改动，单开一个 phase 并附冻结交互回归用例。

**Warning signs:**
- 拆分后部门树联动把页码打回第 1 页（v1.23 FAIL-1 复发）；
- 表格在每次父组件渲染时整体重绘（性能退化）；
- port-write 的 submitting 防重失效（重复提交）。

**Phase to address:** P7 结构整理。

---

### 领域 C：流程层

### Pitfall C1：v1.23 教训复用 —— 四条已付出代价的规则必须变成 v1.24 的执行纪律

**What goes wrong:**
v1.23 的四条教训在 v1.24 会被重新触犯，因为 v1.24 的任务形态高度相似（批量改代码 + 闸门 + 实机验证）：

**① 契约默认值下沉必须配 selector 拦截。**
v1.23 的 `AppInput` 默认 `allowClear=true`（`src/components/ui/AppInput.tsx:12`）落地时**没有**任何 CSS 层面的守卫 → 双框直到 UAT 才发现。
→ v1.24 推论：**任何"默认行为下沉"（CSS 默认值、props 默认值、hook 默认值）都必须同时落一条"可检测的守卫"**（lint selector / 契约单测 / 脚本），否则下一次样式重构会静默破坏它。

**② grep 检测面漏检换行形态。**
v1.23 的 `<Select\n` 漏检教训 → v1.24 的 CSS 检测尤其危险：CSS 选择器**换行等价**（`.ant-input,\n.ant-input-number { }` 是一个规则），任何按行 grep `.ant-input` 的脚本都会**漏掉多选择器规则**，或**把一个选择器行误当成一个完整规则**。
→ 推论：样式守卫脚本必须**按规则块解析**（先按 `{}` 切分再提取选择器），不能按行 grep。
同理，`index.css` 的多行规则（`:871-880` 就是三行选择器 + 一块声明）必须整体识别。

**③ 中文规划文件禁用 PowerShell 管道读写（GBK 控制台静默毁文件）。**
v1.24 要写 `.planning/research/PITFALLS.md`（**简体中文**）、以及 phase/plan 文档。
→ 纪律：**写文件用 `write` 工具或 UTF-8 明确编码的脚本**；**不要**用 `Get-Content | ... | Set-Content`（Windows PowerShell 5.1 默认读取按 ANSI/GBK 解释 → 中文变乱码且**不报错**）。
→ 读写后**必须**回读校验（`read` 一遍关键段落），不要相信"命令成功"。

**④ subagent 崩溃后要「验收遗留 → 接管或重派」。**
v1.24 范围大（7 个 phase），若用 subagent 并行：
→ 纪律：每个 subagent 交付后**先验收产出物是否落盘**（文件存在 + 内容对得上），再决定接管或重派；**不要**因为"收到完成通知"就认为任务完成。

**How to avoid:**
把上述四条**写进 P1/P6/P7 的 plan 头部**作为执行前置，而不是指望执行 agent 记得。

**Warning signs:**
- 规划文件出现 `锟斤拷` / `�` 字符；
- CSS 守卫脚本报告"0 命中"但 `index.css` 明显有对应规则；
- subagent 报告"完成"但工作区文件没变。

**Phase to address:** P1 / P6 / P7 全部（写进各自 plan 的前置清单）。

---

### Pitfall C2：子代理/并行执行的"完成"不可信 —— 交付物必须先验收再采信

**What goes wrong:**
并行推进 7 个 phase 时，每个 phase 都有"看起来完成"的产出（commit、报告、文件）。若不验收就采信，错误会**沉淀到收档**，且 v1.23 已有先例：UAT 记录里的「`/system/menus/permissions` 对该用户 403」**引用了一个前端不调用的端点**（`src/lib/menuApi.ts:24` 用的是 `/system/my-menus/permissions`）→ 一条**从未执行过**的验证被写成了失败证据，直接把 #16 的定责带偏。

**Why it happens:**
"报告里写了"被当成"验证过"；且失败项在总结文档里往往被压缩成一句话（"端点 403"），细节（哪个端点、什么参数、什么身份）丢失。

**How to avoid:**
- 任何 UAT / 复测记录**必须**包含：① 复用的**准确** URL/端点/接口路径；② 身份与权限集（受限于哪个角色）；③ 环境（prod build？sqlite/PG？加密开/关？）；④ 复现步骤；⑤ 截图。
- 定责类结论**必须**有可重跑的 SQL/命令，而不是结论句。
- 每条 v1.24 验收项在关闭前，对照**源码/配置**复核一次端点与常量（本 pitfall 就是这么发现的）。

**Warning signs:**
- 报告里出现「某接口 403」但没有写完整路径；
- 复测环境与生产配置不一致（sqlite vs PG、加密关 vs 开、dev server vs prod build）；
- 结论引用了代码里不存在的标识符。

**Phase to address:** P2 / P3 / P4（每个 phase 的验收清单都加"端点/常量复核"一步）。

---

### Pitfall C3：eslint-plugin-react-hooks v7 的新规则族 —— 清偿 exhaustive-deps 会**引出新 error**

**What goes wrong:**
`eslint.config.js:106` 把 `react-hooks/exhaustive-deps` 设为 **error**（不是 warn）→ 12 处存量 error 必须在 v1.24 清零。但实测 lint 的 **17 errors** 里，**只有 12 处是 exhaustive-deps**，另外 5 处是**别的规则**：

| 规则 | 位置 | 说明 |
|---|---|---|
| `react-hooks/set-state-in-effect` | `src/components/dashboard/DashboardView.tsx:161` | 新规则：effect 内同步 setState |
| `react-hooks/immutability` | `src/hooks/useRealtimeUpdates.ts:130` | 新规则：渲染期改捕获的变量 |
| `react-hooks/globals` | `src/pages/ad-domain/configs/__tests__/ReconciliationTab.test.tsx:51` | 新规则：改写组件外声明的变量 |
| `react-hooks/compiler` | `src/hooks/useRPAProgress.ts:78`（Compilation Skipped: Existing memoization could not be preserved） | **React Compiler 规则族** |
| `react-hooks/compiler` | `src/hooks/useRealtimeUpdates.ts:119`（Cannot access variable before it is declared） | React Compiler 规则族 |
| `local/no-large-dropdown-list` | `src/components/ui/AppSelect.tsx:16` | **项目自定义规则** |

安装版本：**`eslint-plugin-react-hooks@7.0.1`**（v6+ 引入 React Compiler 规则族，v7 继续扩展）。

**关键陷阱：**
1. **改代码引出连锁 error**：`useRealtimeUpdates.ts:119` 的 compiler 错误是 "Cannot access variable before it is declared" —— 修 `useRealtimeUpdates.ts:130` 的 immutability 错误，很可能**触发或加剧** compiler 错误 → 需要在**同一个文件**上一起改并一起验。
2. **exhaustive-deps 的"正确修法"本身会撞新规则**：把 `message`（antd `App.useApp()` 返回值）加进 `useCallback` 依赖 → 依赖数组变化 → 但 antd 的 `message` 实例在 `<App>` 重新挂载时会变 → 可能引发 `useCaptchaModals.tsx` 的 5 处回调频繁重建 → **反而制造新的性能/闭包问题**。→ 这 5 处（`useCaptchaModals.tsx:143/172/183/193/202`）**必须逐处判断语义**，不能机械加依赖。
3. **契约组件自身违规**：`src/components/ui/AppSelect.tsx:16` 报 `local/no-large-dropdown-list`（"Select/AutoComplete/Cascader 缺少 onSearch prop;必须使用服务端搜索 + 防抖"）。而 `AppSelect.tsx:18` 的契约是 `showSearch = showSearch ?? count >= 8` —— **契约自己开了搜索却没有 onSearch**。清零这条 error 只有三条路：① 改契约（加 onSearch/去掉自动开搜索）；② 给契约组件加豁免；③ 放宽自定义规则。
   > 附带发现（文档漂移）：`75-UAT.md:66` 记「AppSelect showSearch + optionFilterProp="value"」，而 `AppSelect.tsx:19` 实际是 `optionFilterProp="label"` → **UAT 记录与代码不符**（v1.23 PITFALLS Pitfall 2.2 早已预警过 `optionFilterProp="label"` 的破坏性，这里正好是同一条）。

**How to avoid:**
- 12 处 exhaustive-deps **按语义分组**清偿，不要一把梭：
  | 组 | 文件 | 修法方向 |
  |---|---|---|
  | A. `message` 依赖（6 处） | `useCaptchaModals.tsx`×5、`default-theme.tsx:70` | 用 `useMemo` 稳定引用（`const stableMessage = useMemo(() => message, [message])`）或加 `eslint-disable-next-line` + 理由（**必须写明"message 实例稳定"**） |
  | B. 内部函数依赖（3 处） | `useCaptchaData.ts:47`、`ad/SyncMonitor/index.tsx:91`、`captcha-background/index.tsx:64` | 把内联函数提成 `useCallback` 或改用 ref |
  | C. 不必要依赖（1 处） | `ReconciliationTab.tsx:115`（`queryTick` 多余） | 直接删依赖 |
  | D. 其它（2 处） | `DashboardView.tsx:151`、`ColorSwitcher.tsx:109` | 逐处判断 |
- **清零的判据必须是「errors 17 → 0」且「warnings 34 → 不上升」**，并把计数写成脚本断言（`scripts/assert-lint-budget.mjs` 已存在，可扩展）。
- 5 处非 exhaustive-deps 的 error **同期清偿**（否则 lint 永远不是 0，且它们的修法会互相干扰）。
- `AppSelect` 的 `local/no-large-dropdown-list`：**先决定契约方向**再改。若保留"≥8 自动开搜索"，则必须补 `onSearch`（服务端搜索）或给契约组件加显式豁免（因为 AppSelect 是"契约"本身，不是业务页）。

**Warning signs:**
- 修完 12 处 exhaustive-deps 后 errors 反而变多（compiler / set-state-in-effect 被引出）；
- `npm run lint` 的 warning 数从 34 上升（改代码引出新 warning）；
- `useCaptchaModals` 的回调引用每次渲染都变 → 弹窗里的 effect 反复触发。

**Phase to address:** P6 lint 清偿。

---

### Pitfall C4：封装层「目录级」豁免是契约的盲区 —— `src/components/shared/**` 里发生什么都看不见

**What goes wrong:**
`eslint.config.js:271-283` 的「封装层豁免 override」把 `src/components/ui/**` 与 `src/components/shared/**` 的 `no-restricted-syntax` **整条重声明为只剩内网 IP selector** → 这两个目录里：
- 裸 `<Input>` / `<Select>` / `<TextArea>` / `<Modal.confirm>` / `<Popconfirm>` / 裸 hex —— **全部不被拦**。

后果与 v1.24 的两处任务直接相关：
1. **`ColumnConfigModal.tsx:136` 的 `Modal.confirm`**（重置列配置，非删除）**正是因为在 `src/components/shared/**` 才没被拦**。v1.22 契约要求「剩余 8 处 `Modal.confirm` 合法点全部带行内理由注释」—— 但目录级豁免让"哪几处是合法点"**无法被工具区分**。UAT 记的「[低] `ColumnConfigModal.tsx:136` 补豁免理由注释」实际是：**在这个目录级豁免的掩护下，这一处 `Modal.confirm` 需要行内理由注释来声明"我不是危险删除入口"**。
   > 补充事实：`scripts/exemption-reasons.json` **没有** `ColumnConfigModal` 条目，`scripts/check-exemption-reasons.mjs:101-104` 校验的是「eslint 锚点数组 ↔ json 条目」双向相等 —— 该文件根本不在锚点数组里，所以校验脚本不会报错。→ **登记制与目录级豁免之间存在一个"无人区"**。
2. **`AppSelect.tsx:16` 触发 `local/no-large-dropdown-list`** —— 该规则是**独立规则**（不在 `no-restricted-syntax` 里），所以**不受封装层豁免保护** → 契约组件自己被自家规则拦下。

**How to avoid:**
- P5（`ColumnConfigModal:136`）的最小正确做法：**加行内理由注释**，说明"此处是列配置重置（非数据删除），不走 ConfirmDelete 契约"，并**在 `scripts/exemption-reasons.json` 之外另设"封装层合法点清单"**（或把该文件登记进 json 并同步 eslint 锚点，**注意两个方向都要改** —— `check-exemption-reasons.mjs:103-104` 会双向报错）。
- **不要**为了"让 lint 过"而把 `src/components/shared/**` 从豁免里摘掉 —— 一摘就爆出该目录里所有裸用法（契约组件内部本来就该用裸 antd），**会瞬间引入几十条新 error**。这正是 `eslint.config.js:146-149` 注释警告的 flat config 覆盖事故的同款陷阱。
- 长期方案：把封装层豁免从"目录级"改成"文件级 + 具体 selector 白名单"，但这是独立工程，v1.24 不做。

**Warning signs:**
- 有人把 `src/components/shared/**` 从豁免列表里删掉 → lint error 数暴涨；
- `ColumnConfigModal` 加了注释但没登记 json → `check-exemption-reasons.mjs` 报「[多理由]」；
- json 加了条目但没同步 eslint 锚点 → 报「[缺理由]」。

**Phase to address:** P5 豁免注释（同时把"封装层合法点登记"的口径写进注释）。

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|---|---|---|---|
| 用 `.ant-input:not(.ant-input-affix-wrapper > .ant-input)` 收窄 | 看起来最"精准" | CSS3 下整条规则被**静默丢弃**，收窄完全失效且无任何报错 | **Never**（改用 wrapper 上下文直接子元素） |
| 用 `:not()` 内含组合子 + `@supports` 兜底 | 保留 `:not()` 语义 | 需维护双份规则；browserslist 变更时静默退化 | 仅当明确锁定 Chrome 88+ 且有 CI 断言 |
| 把 `.ant-input` 改成 `.ant-input-outlined` | 语义精确、代码最短 | `ConfigProvider` 全局 variant 一变，主题样式**整体消失**；暗色标签选择器 `[data-color-mode="dark"] input` 仍命中内层 | 仅当同时在 `AntdThemeBridge` 锁死 `variant: "outlined"` |
| 修双框时只改 `border`，不动 `border-radius`/`background`/`font-size` | 改动最小 | 输入框出现四角接缝 / 字号不齐 / 背景色差 | **Never**（成组复位） |
| 顺手把 `.ant-picker`（DatePicker）也纳入边框规则 | 主题统一 | 项目没有 `.ant-picker:hover/:focus` 规则 → DatePicker 交互态与 Input 不匹配 | Never（单独立项） |
| 给 12 处 exhaustive-deps 机械加依赖 | 立刻清零 | `message` 依赖不稳定 → 回调频繁重建 → 引出新 error（compiler / set-state-in-effect） | Never（按语义分组，见 C3） |
| 用 `eslint-disable-next-line` 压掉 5 处非 exhaustive-deps error | 数字归零 | 把 React Compiler 规则族的问题永久掩盖 | 仅限"确认语义安全 + 写明理由" |
| 把 `src/components/shared/**` 从封装层豁免里摘掉 | "契约纯净" | 契约组件内部本来就用裸 antd → 瞬间几十条新 error | **Never** |
| 为清 react-refresh 把契约组件拆成几十个碎片文件 | warning 归零 | 破坏"契约组件自包含"（v1.22 契约化的初衷） | Never（先划契约层豁免名单） |
| `#16` 只修楼宇页 | 一天收工 | 其它列表页仍无门控 → 复测再开一条债 | Never（要么切范围明说，要么全站契约化） |
| 权限门控用 `permissions.length === 0` 判断"未就绪" | 一行代码 | 空权限集与未加载完不可区分 → 接口失败时管理员也看不到按钮 | Never（用显式 loading/error 态） |
| #16 复测用手工插库的角色 | 绕开 sqlite 500 | 绕过 `InvalidateUserMenuCache` → 权限集 30 分钟缓存不失效 → 复测结论无效 | Never（走 UI 正常路径，或手动清 `menu:user:perms:{id}`） |
| 用 `/system/menus/permissions` 复测 | 看起来像在查权限 | **前端根本不调这个端点**（`src/lib/menuApi.ts:24` 用的是 `/system/my-menus/permissions`）→ 结论无效 | **Never** |
| 3f130 前端只按 `emptyReason` 渲染空态 | 契约直白 | 有数据时同时显示"共 1 台"+"因未绑用户而为空" → 语义打架 | Never（按 `devices.length` 分流空态/告警） |
| F5 问题改 `path="*"` 的 fallback | 三行改完 | 破坏未登录跳登录 / 无权限跳 403 / deep link 直达 / 未就绪 loading 四条既有行为 | **Never**（改就绪判定 + 停止反写 LAST_PATH） |
| 直接把 `export { X }` 改成 `export type { X }` | warning 消失 | 若 X 同时是值（同名 const）→ 运行时 undefined | 仅当逐个确认是纯类型 + 跑测试 |
| 在 `tsconfig` 里加 `isolatedModules` | 强化模块边界 | `verbatimModuleSyntax: true` 已隐含该语义，加了不改变任何行为 | Never（无收益） |
| 用 PowerShell `Get-Content | Set-Content` 改中文规划文件 | 顺手 | **GBK 静默毁文件**，不报错 | **Never**（用 `write` 工具 + 回读校验） |
| CSS 守卫脚本按行 grep `.ant-input` | 写得快 | 多行选择器规则（`index.css:871-880`）会被漏检或误判 | **Never**（按 `{}` 切规则块再解析选择器） |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| antd 6.6.4 Input 结构 | 以为 `.ant-input` 自带边框 | antd 6 的 `.ant-input` **无 border**；边框在 `.ant-input-outlined/-filled/-underlined`（`input/style/index.js:296-318`） |
| antd 6.6.4 affix 结构 | 以为 variant 类总在内层 input | 有 affix 时 variant 类**落在 wrapper**，内层只有 `ant-input`（`@rc-component/input/es/BaseInput.js:43-45, 79-88`） |
| antd 6.6.4 cssinjs | 以为"写在后面就能覆盖 antd" | hash 前缀让 antd 规则特异性 **+1 类**（`@ant-design/cssinjs/es/hooks/useStyleRegister.js:27-47`）→ 必须 `!important` + 算特异性 |
| antd 6.6.4 `ConfigProvider` | 以为项目那个 `src/components/ConfigProvider.tsx` 是 antd 的 | 项目那个是"配置同步 Provider"；真正的 antd ConfigProvider 在 `src/design-system/components/AntdThemeBridge.tsx:178` |
| antd 6 `useVariant` | 以为 `components.Input.variant` 会影响 TextArea | TextArea 走 `useVariant('textArea', ...)`（`TextArea.js:94`），**不 fallback** 到 Input 的 variant |
| 后端用户权限集 | 查 `pkg/permission/config.go` 的 `GetRoutePermissions()` | 那是**路由→权限码静态表**（无缓存、每次新切片），**不是**用户权限集；用户权限集在 `internal/services/system/menu_service.go:509-569` |
| 后端权限缓存 | 以为改了角色立即生效 | `menu:user:perms:{userID}` **30 分钟 TTL**（`menu_cache_impl.go:115-120`）；手工插库绕过全部失效路径 |
| 后端 `sys_menu.perms` | 以为可以用逗号多值 | `Pluck("perms", &perms)` **整串取出不拆逗号**（`menu_service.go:547-552`）→ 前端必须精确匹配 |
| 后端权限码历史 | 直接用当前命名 | migration_159 曾把 `ops:buildings:*` 迁到 `ops:building:*`（`migration_159_align_ops_perms.go:14-38`）→ 老库可能仍是旧命名 |
| 前端权限数据 | 用 `/system/menus/permissions` | 前端调的是 `POST /system/my-menus/permissions`（`src/lib/menuApi.ts:24`） |
| 物理链路诊断端点 | 只用旧的 `/physical` | 新端点 `POST /ops/workstation-device/:id/physical-diagnosis`（`router.go:702`）才带 emptyReason/warnings |
| 物理链路空因语义 | 认为 emptyReason 只在空时有值 | `classifyPhysicalEmptyReason` 无条件先判 `!UserBound`（`workstation_device_service.go:822`）→ **有数据也会带空因** |
| eslint-plugin-react-hooks | 以为只有 exhaustive-deps | v7.0.1 还有 `set-state-in-effect` / `immutability` / `globals` / `compiler` 规则族，均为 error |
| eslint-plugin-react-refresh | 以为 `export type` 仍会触发 | 规则实现 `if (node.exportKind === "type") continue;`（`index.js:178,194`）→ **类型导出不触发** |
| tsconfig 模块语义 | 以为没开 `isolatedModules` 就能随便 re-export 类型 | `verbatimModuleSyntax: true`（`tsconfig.app.json:14`）**强制** `export type` |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|-----------------|
| devices 拆分后 `deptFilters` 引用不稳 | 部门树联动把页码打回第 1 页（v1.23 FAIL-1 复发） | `deptFilters` 留在主组件 `useMemo`；拆分只搬纯函数 | 拆分后首次联动 |
| devices 拆分后 `columns` useMemo 失效 | 表格每次父渲染全量重绘 | `columns` 的 useMemo 留主组件，依赖逐个核对 | 拆分后任意状态更新 |
| `useCaptchaModals.tsx` 5 处回调加 `message` 依赖 | 回调引用每次渲染变 → effect 反复触发 | 用 `useMemo` 稳定引用或写明理由的 disable | 清偿 exhaustive-deps 时 |
| 权限门控改成"每次渲染读 permissions" | 无（React 会跳过同引用） | 保持 `useMenuStore((s) => s.permissions)` 订阅 | — |
| `menu:user:perms` 30 分钟 TTL | 改了角色权限但界面不变 | 走 UI 正常路径（触发失效）或手动清 key | 权限变更后 30 分钟内 |
| 全站 Input 都走 affix-wrapper | 每次渲染多一层 wrapper DOM | antd 结构决定，无法避免；注意不要在 wrapper 内再包测量逻辑 | 已发生（AppInput 默认 allowClear） |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| 只做前端行操作门控，后端不加校验 | 门控只是 UX，用户可直接调 API 删除 | 后端 `internal/api/router.go` 已按 `ops:building:delete` 等校验 → **必须保持**；前端门控永远标注"非安全边界"（同 `src/router/RouteGuard.tsx:5` 口径） |
| 为 #16 把 `network:port:write` 拆成多个权限码但只改前端 | 后端仍认旧码 → 新码无效或旧码仍放行 | 权限码是**前后端契约**，改码必须同步 `init_data.go` + `router.go` + 前端映射表 + 一致性校验脚本 |
| 引入「后端在列表里回传 canEdit/canDelete」方案 | N 个列表接口全改 → 范围失控、易漏 | v1.24 不选该方案（见 B2 折中表） |
| 在前端日志里打印完整权限集 | 权限结构泄露 | 用 dev-only 且脱敏；参考 v1.23 对 hardcoded color 的同类纪律 |
| 为"方便复测"在生产库手工插 `sys_role_menu` | 权限绕过审计，且 30 分钟缓存不一致 | 复测走 UI 正常路径；必须插库时记录到测试记录并事后清理 |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| 权限未就绪时直接隐藏删除按钮 | 用户以为系统没有删除功能，且无提示 | 三态门控：未就绪 → 禁用 + 说明；就绪无权限 → 不渲染 |
| 3f130 页面同时显示"共 1 台"和"因未绑用户而为空" | 用户看不懂，怀疑数据错乱 | 有数据 → warning Alert 措辞改为"未绑定用户，链路按用户维度归属，可能显示不全"；无数据 → 空态用 emptyMessage |
| 修复双框后焦点环外扩 1px | 聚焦瞬间"跳一下" | 视觉验证必须覆盖 focus 态截图对比 |
| DatePicker 与 Input 边框色不一致 | 同一表单里控件风格割裂（既存问题） | 登记为独立欠账，v1.24 不顺手改 |
| F5 后被静默送到 dashboard 且原路径被覆盖 | 用户"回不去刚才的页面" | 修就绪判定 + 不反写 LAST_PATH；否则至少在 403 时明确提示 |
| 清空搜索框触发 `onChange(undefined)` | 表单提交 400 | 封装层归一化 undefined（v1.23 契约精神延续） |
| 契约组件 `AppSelect` 无 `onSearch` 却自动开搜索 | 大列表卡顿、过滤行为不符业务 | 先定契约方向（见 C3），不要让规则与契约互相打脸 |

## "Looks Done But Isn't" Checklist

- [ ] **双框修复**：是否同时覆盖 `input.ant-input` 与 `textarea.ant-input` 两种内层（`> input` / `> textarea`）—— 只修 Input 会漏 TextArea 的 showCount/allowClear 场景
- [ ] **双框修复**：是否成组复位 `border` + `border-radius` + `background` + `color` + `font-size`（只改 border → 四角接缝 / 字号不齐 / 背景色差）
- [ ] **双框修复**：新规则是否带 `!important` 且特异性 ≥ (0,3,1)（否则输给 antd hashed 规则与暗色标签选择器）
- [ ] **双框修复**：是否在 `index.css` 中引入 `:not()` 内含组合子的写法（CSS3 静默丢弃）
- [ ] **双框修复**：暗色模式是否单独截图（`[data-color-mode="dark"] input` 标签选择器会绕过类收窄）
- [ ] **双框修复**：focus 态是否确认**只有一圈**焦点环（wrapper 承担），且修复后外扩 1px 可接受
- [ ] **双框修复**：是否有**样式契约单测**（防止有人再写一条 `.ant-input { border: ... !important }`）—— 没有守卫 = 下次还会回归
- [ ] **#16 定责**：是否先跑 `sys_menu.perms` 查询判定"后端是否发了写权限"，再谈前端（顺序反了会修错地方）
- [ ] **#16 定责**：复测端点是否为 `POST /system/my-menus/permissions`（**不是** `/system/menus/permissions`）
- [ ] **#16 定责**：是否走 UI 正常路径建角色（否则 30 分钟权限缓存不失效）
- [ ] **#16 修复**：权限"未就绪"与"权限为空"是否用**不同状态**区分（不能用 `length === 0`）
- [ ] **#16 修复**：是否复测了**多个**列表页（只测 1 页 = 点过不是修好）
- [ ] **3f130 展示**：后端 `workstation_device_service.go:871-874` 的矛盾注释是否已对齐（否则前端按哪套理解都可能错）
- [ ] **3f130 展示**：是否有"有数据 + 空因"的**不变量测试**（现有测试只锁了"0 台必带空因"）
- [ ] **3f130 展示**：3f130 的 `user_id` 是否**已绑定**（没绑 → 复验必然仍空白 → 被误判为前端没修好）
- [ ] **3f130 展示**：前端是否按 `devices.length` 分流「空态」与「告警」两套 UI
- [ ] **F5**：是否**没有**改 `path="*"` 的 fallback（改了会破坏未登录/403/deep link/loading 四条行为）
- [ ] **F5**：兜底跳转产生的路径是否**不再写回** `ruoyi_last_visited_path`
- [ ] **F5**：LAST_PATH 是否加版本号（跨版本残留会弹跳）
- [ ] **F5**：回归矩阵 7 条是否逐条实测（尤其"无权限 deep link 应 403 而非静默跳 dashboard"）
- [ ] **react-refresh**：是否先划出「契约层豁免名单」（`src/components/shared/**` + `src/components/ui/**`）再清零
- [ ] **react-refresh**：`export { X }` → `export type { X }` 是否逐个确认 X 是**纯类型**、无运行时消费方
- [ ] **react-refresh**：改完后是否跑了 `npm run type-check` **和** `npm run test`（测试文件不在 `tsconfig.app.json` 的 include 内，type-check 不会拦）
- [ ] **lint 清偿**：17 处 error 是否**按规则分组**（12 exhaustive-deps + set-state-in-effect + immutability + globals + compiler×2 + local/no-large-dropdown-list）而不是按文件凑
- [ ] **lint 清偿**：`useCaptchaModals.tsx` 的 5 处 `message` 依赖是否逐处判断语义（机械加依赖会引出新 error）
- [ ] **lint 清偿**：`AppSelect.tsx:16` 的 `local/no-large-dropdown-list` 是否**先定契约方向**再改
- [ ] **lint 清偿**：errors 是否 0、warnings 是否**未上升**（用 `scripts/assert-lint-budget.mjs` 断言，不靠人眼）
- [ ] **devices 拆分**：`deptFilters`（`:277`）引用是否仍稳定（否则 v1.23 FAIL-1 复发）
- [ ] **devices 拆分**：`columns`（`:524`）的 `useMemo` 依赖是否逐个核对
- [ ] **devices 拆分**：`src/components/network/port-write/**` 与 `ActionButtons.tsx` 的对外形状是否**未被改动**（v1.19 冻结交互）
- [ ] **ColumnConfigModal:136**：是否加了行内理由注释；是否需要同时登记 `scripts/exemption-reasons.json` 与 eslint 锚点（**双向**，否则 `check-exemption-reasons.mjs` 报「[多理由]」/「[缺理由]」）
- [ ] **全局**：中文规划文件写入后是否回读校验（GBK 静默毁文件）
- [ ] **全局**：subagent 交付是否先验收落盘再采信

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| `:not()` 规则被静默丢弃 | LOW | 换成 `.ant-input-affix-wrapper > input.ant-input` 形态 → DevTools 确认规则不再灰 → 重新截图 |
| 修复后出现双焦点环 | LOW | 移除复位规则里的 `box-shadow` 声明 → 确认内层焦点环仍由 antd 的 `box-shadow: none !important` 关闭 |
| 修复后输入框四角接缝 | LOW | 复位规则补 `border-radius: 0 !important` |
| 暗色下内层背景色差 | LOW | 复位规则补 `background: transparent !important`（特异性 (0,3,1) 稳压暗色标签选择器） |
| 收窄方案（variant 类）后主题整体消失 | MEDIUM | 在 `AntdThemeBridge.tsx` 显式锁 `variant: "outlined"` → 重新截图全站输入框 |
| 契约组件被拆成碎片 | MEDIUM | `git revert` 拆分 commit → 改为划「契约层豁免名单」→ 重新清零其余文件 |
| `export type` 改错导致运行时 undefined | MEDIUM | `git revert` 该文件 → 逐个确认是纯类型 → 跑 `npm run test` |
| #16 门控导致"按钮消失"投诉 | MEDIUM | 改为三态门控（未就绪 → 禁用 + 提示）→ 补"权限加载慢"的实机复验 |
| #16 修错地方（改了前端但后端发了写权限） | MEDIUM | 跑 `sys_menu.perms` 查询判定 → 若后端确实发了写权限 → 回滚前端改动，改数据（角色菜单勾选） |
| #16 复测结论无效（手工插库/用错端点） | MEDIUM | 重走 UI 正常路径 + 用 `POST /system/my-menus/permissions` 取权限集 + 截图 + 重写复测记录 |
| 3f130 页面出现"有数据 + 空态"并存 | LOW | 按 `devices.length` 分流渲染 → 空态/告警二选一 |
| 3f130 复验仍空白 | LOW | 先查 `sys_workstation.user_id` 是否为空（**数据侧动作未做**），不要改前端 |
| F5 修完变成"闪一下 dashboard 再跳回" | MEDIUM | 改为"目标路径未在 routeElements 中时保持 loading/403"，不靠 `*` 兜底 |
| F5 修完未登录用户被送进需鉴权页 | MEDIUM | 恢复 `:263-270` 的未登录分支（`*` → `/login`），只在已认证分支改兜底 |
| 拆 devices 后页码被打回第 1 页 | LOW | 把 `deptFilters` / `columns` 的 `useMemo` 移回主组件 → 复测部门树联动 |
| exhaustive-deps 清偿引出新 error | MEDIUM | `git revert` 单文件 → 按 C3 分组表重做 → 跑 `npm run lint` 对账 |
| 中文规划文件被 GBK 毁 | MEDIUM | 从 git 历史恢复 → 用 `write` 工具重写 → 回读校验 |
| 样式回归再次发生 | MEDIUM | 补 CSS 契约单测（A9 方案 1）→ 在 CI 中作为 error |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| A1 `:not()` 组合子 | P1 样式修复（方案选型） | 代码审查 + DevTools 确认规则非灰 + CI 断言 `:not()` 参数内无组合子 |
| A2 `!important` 压过非 important | P1 样式修复 | 复位规则带 `!important` 且特异性 ≥ (0,3,1) |
| A3 antd 6 variant 体系 | P1 样式修复（若走 variant 路线） | `AntdThemeBridge` 锁 `variant`；全站输入框截图无主题丢失 |
| A4 cssinjs hash 前缀 | P1 样式修复 + P6（可加 lint 规则） | 新增 `.ant-*` 覆盖规则均带 `!important`；dev 与 prod build 产物均验 |
| A5 affix 结构差异 | P1 样式修复 | `> input.ant-input` 与 `> textarea.ant-input` 双分支都验；InputNumber/Select 不回归 |
| A6 AppInput 默认 allowClear | P1 样式修复 | 视觉验证矩阵同时覆盖「有 clear icon」与「无 clear icon」两类输入框 |
| A7 暗色标签选择器 | P1 样式修复 | 暗色模式截图（默认/hover/focus） |
| A8 焦点环 | P1 样式修复 | focus 态截图对比，确认单圈 + 外扩 1px 可接受 |
| A9 视觉层无守卫 | P1 样式修复（与修复同 phase） | CSS 契约单测进 CI |
| B1 #16 根因是"无门控" | P2 #16 定责 | `sys_menu.perms` 查询 + `POST /system/my-menus/permissions` 双证据 |
| B2 fail-closed 可用性回归 | P2 #16 修复 | 权限慢/失败场景实机复验：按钮禁用但可见 + 有提示 |
| B3 #16 范围膨胀 | P2 #16（切范围） | 契约文档写明覆盖页面清单与 Known Gaps；一致性校验脚本通过 |
| B4 3f130 空因语义矛盾 | P3 3f130 展示 | 后端注释对齐 + 不变量测试 + 前端按 `devices.length` 分流截图 |
| B5 F5 落 dashboard | P4 F5 落地页 | 7 条回归矩阵逐条实测 + LAST_PATH 不被反写 |
| B6 `export type` 与 verbatimModuleSyntax | P7 react-refresh | `npm run type-check` + `npm run test` 双绿；warning 数下降且 errors 不上升 |
| B7 devices 拆分 | P7 结构整理 | 部门树联动保页码 + 表格不整体重绘 + port-write 冻结交互未动 |
| C1 v1.23 教训复用 | P1 / P6 / P7（写进 plan 前置） | 规划文件回读校验；CSS 守卫按规则块解析 |
| C2 交付物先验收 | P2 / P3 / P4 | 复测记录含准确端点/身份/环境/截图 |
| C3 react-hooks v7 新规则族 | P6 lint 清偿 | errors 17→0 且 warnings 34→不上升（脚本断言） |
| C4 封装层目录级豁免盲区 | P5 豁免注释 | 行内理由注释 + `check-exemption-reasons.mjs` 双向通过 |

---

## 附录 A：`:not()` 替代方案矩阵

> 背景：CSS Selectors Level 3 的 `:not()` 只接受**简单选择器**；含组合子（`>`、空格、`+`、`~`）的参数在 CSS3 解析器下导致**整条规则被丢弃**。Level 4 放宽为 complex-real-selector-list（Chrome 88+ / Firefox 84+ / Safari 9+），但构建链不会为老浏览器降级 `:not()` 内的复杂选择器 —— 老浏览器上同样**静默丢弃整条规则**。

| # | 写法 | CSS3 合法性 | 特异性 | 运行时效果 | 评价 |
|---|---|---|---|---|---|
| **1（推荐）** | `.ant-input-affix-wrapper > input.ant-input,`<br>`.ant-input-affix-wrapper > textarea.ant-input { border: none !important; ... }` | ✅ 合法（组合子在 `:not()` 外） | (0,3,1) | 精确命中「antd 要求无边框」的两个内层元素 | **首选**：不依赖 variant 类、对 `ConfigProvider` 变更免疫、对暗色标签选择器免疫 |
| 2 | `:where(.ant-input-affix-wrapper) > :where(input.ant-input) { ... }` | ✅ 合法 | (0,0,1) + `!important` | 同上 | 适合"要压住 antd 状态样式但不提高特异性"的场合 |
| 3 | `.ant-input-outlined { /* 主题边框 */ }` | ✅ 合法 | (0,1,0) | 命中顶层 input/textarea（**有 variant 类**），不命中 affix 内层 | 语义最准，但 `ConfigProvider` 全局 variant 一变就整体失效 → 必须配套锁 `variant: "outlined"` |
| 4 | `.ant-input:not(:where(.ant-input-affix-wrapper > .ant-input)) { ... }` | ⚠️ 需 Level 4 | (0,1,0) | 语义正确 | **不推荐**：`:not()` 内仍含组合子 → CSS3 丢弃；需 `@supports selector(:not(* > *))` 兜底 + browserslist 锁定 |
| 5 | `.ant-input:not(.ant-input-affix-wrapper > .ant-input) { ... }` | ❌ **CSS3 无效** | — | **整条规则被丢弃** | **禁用**（用户原始提法） |
| 6 | 语义化钩子类：`AppInput` 传 `className="app-input"` / `rootClassName="app-input-root"`，CSS 用 `.app-input` / `.app-input-root` | ✅ | 可控 | 完全由项目掌控 | 结构性最优，但需改代码 + 覆盖存量裸 Input（v1.23 后存量应为 0） |

**本项目推荐组合：方案 1（复位）+ 保持 `.ant-input` 主题规则不动。**
理由：改动最小、不依赖 antd 变体类、对 `ConfigProvider` 与暗色标签选择器都免疫，且能同时覆盖 Input 与 TextArea。

**如果将来要"收窄主题规则"而非"复位内层"**（即让 `.ant-input` 不再命中内层），则用**方案 3**，并强制在 `AntdThemeBridge.tsx` 写死：
```ts
Input: { /* ...现有 token... */, variant: "outlined" }   // 项目主题只适配 outlined
```

---

## 附录 B：双框修复的最小充分视觉验证矩阵

**前置**：prod build（`npm run build && npm run preview`）+ 真实数据 + 暗色模式可切（`settingsStore.preferences.theme.mode`）。

| # | 页面 / 状态 | 组件形态 | 必看点 |
|---|---|---|---|
| 1 | 楼宇管理列表页搜索区（`/operations/buildings`） | `AppInput`（**有 clear icon** → affix-wrapper） | 内层无 1px 边框；无四角接缝；字号 14px；placeholder 颜色正确 |
| 2 | 新增/编辑楼宇弹窗 —— 名称输入框 | `AppInput` 有 affix | 同上；焦点态只有一圈 2px 绿环 |
| 3 | 同一弹窗 —— **描述 TextArea（带字数统计）** | `Input.TextArea` + `showCount` → textarea affix-wrapper | **最易漏的一格**：内层 textarea 无边框；count 角标位置不偏移 |
| 4 | 同一弹窗 —— **描述 TextArea（无字数统计）** | 纯 textarea（顶层） | **必须单列**：这是"顶层 textarea"分支，与 #3 的 DOM 结构完全不同 |
| 5 | 任意含 `InputNumber` 的表单 | `InputNumber`（可能带 formatter / 前后缀） | `.ant-input-number` 根的边框正常；内层 `.ant-input-number-input` 不被误伤 |
| 6 | 任意含 `Select` 的表单 | `AppSelect`（`optionFilterProp="label"`） | `.ant-select-selector` 边框/背景与 Input 一致；`showSearch` 开启后下拉正常 |
| 7 | 任意含 `DatePicker` 的表单 | `.ant-picker` | **只做回归确认**（既存不一致，本次不改）：边框色与 Input 不同属已知欠账，不得在本次被"顺手统一" |
| 8 | 表格内联编辑单元格 | `AppInput` / `AppSelect` 嵌在 `Table` 内 | 单元格高度不塌陷（复位 `padding: 0` 不得影响顶层 input）；无错位 |
| 9 | **暗色模式**下重复 #1 / #2 / #3 | 同上 | 内层与 wrapper **背景同色**；hover/focus 时内层不变色；边框色 `--input-border` #2d4d33 |
| 10 | **readonly** 输入框 | `readOnly` | antd 的 readonly 样式（`input[readonly]`）不被复位规则破坏；文字仍可选中 |
| 11 | **disabled** 输入框 | `disabled` | 背景 `--input-bg-disabled`、文字 `--input-text-disabled`（暗色 token 已定义但项目未接线 → 记录为欠账）；无边框加深 |
| 12 | 错误态表单 | `status="error"` | antd 的 error 边框（`--input-border-error` #ba3630）未被复位规则压掉 |
| 13 | 焦点态特写（放大截图） | 任意 affix Input | **只有一圈** 2px 焦点环 + 一圈 12px 外发光（暗色下两层叠加）；修复后环外扩 1px 与修复前对比无"跳动" |
| 14 | 密码框 / 搜索框（`Input.Password` / `Input.Search`） | 特殊包装 | 图标不与边框重叠；无额外内框 |

**每个格子必须记录**：截图文件名、build commit、暗/亮、浏览器窗口宽度（≥1280，验证 fixed 列不受影响）。

---

## 附录 C：#16 两难的最优折中建议（可直接进 plan）

**结论：采用「三态门控 + 后端保持强校验」，不采用纯 fail-closed。**

1. **契约层**：在 `src/components/shared/ActionButtons.tsx` 的 `ActionButton` 与 `createDeleteAction` 的 options 上新增**可选** `permission?: string`（向后兼容，存量调用点不传即保持现状）。
2. **判定层**：新增**单一** helper（建议 `src/components/shared/useActionPermission.ts`）：
   ```
   权限未就绪（loading 或 error） → 渲染按钮但 disabled，title/aria-label 提示"权限信息加载中"
   权限就绪且含该码              → 正常可用
   权限就绪且不含该码            → 不渲染
   ```
   **"未就绪"必须用显式状态位**（`menuStore` 的 `loading` / `error`），**禁止**用 `permissions.length === 0` 判定。
3. **映射层**：建立「模块 × 操作 → 权限码」单一真相源（建议 `.planning/phases/<P2>/action-permission-map.json`），**由脚本从后端 `init_data.go` / `pkg/permission/config.go` 校验一致性**（防止前后端漂移 —— migration_159 的历史已经证明这条命名会漂）。
4. **范围**：P2 交付「契约 + helper + N 个代表页面（楼宇/工位/用户/网络设备）」，其余页面登记 Known Gaps；**不**在本里程碑做全站铺开。
5. **安全边界**：前端门控在 UI 与注释中明确标注"非安全边界"；后端 `internal/api/router.go` 的权限中间件**不改**（已是真正的边界）。
6. **验收**：① PG 环境走 UI 正常路径建受限角色；② 用 `POST /system/my-menus/permissions` 取权限集作为证据；③ 覆盖"有权限 / 无权限 / 权限接口失败"三种状态；④ 覆盖 ≥2 个列表页。

---

## 附录 D：react-refresh 33 处清零的 `export type` 编译期交互结论

**结论一句话**：
> 在 `verbatimModuleSyntax: true`（`tsconfig.app.json:14`）下，把 `export { X }` 改成 `export type { X }` **既是清除 react-refresh warning 的必要手段，也是 TS 强制要求的写法**，且**运行时完全无差别**（type-only 导出在转译期被完全擦除）；但**必须逐个确认 X 是纯类型**，否则运行时引用会变成 `undefined`。

**依据**：
- `node_modules/eslint-plugin-react-refresh/index.js:193-194`（以及 `:177-178` 对 `exportAll`）—— `if (node.exportKind === "type") continue;` → **类型导出被规则显式跳过**。
- `tsconfig.app.json:14` `"verbatimModuleSyntax": true` → import/export 语句不做类型擦除，没有 `type` 修饰符的绑定按运行时绑定处理 → `export { SomeType }` 会报 TS1205/TS1484 → **必须** `export type { SomeType }` 或 `export { type SomeType }`。
- `tsconfig.app.json:16` `"noEmit": true` + Vite/esbuild 转译 → `export type` 在产物中不存在 → 运行时行为不变。
- `tsconfig.app.json:39` `exclude` 排除了 `src/**/*.test.ts(x)` → **测试文件不在 type-check 范围内** → 改写导出后必须额外跑 `npm run test`。

**"纯类型"判定清单**（逐个执行）：
1. 该符号在源文件里的声明是 `interface` / `type X = ...`（不是 `const` / `function` / `class`）？
2. 源文件里**没有**同名的值导出（`export const X` + `export interface X` 合并）？
3. 全仓（含 `*.test.ts` / `*.test.tsx`）grep 该符号的**所有** import 点，逐个确认是**仅用于类型位置**（`import type` 或类型注解）？
4. 没有 `.js` / `.mjs` / 动态 `import()` 按值引用它？

**改造后的验证三件套**：`npm run type-check`（绿）+ `npm run test`（绿）+ `npm run lint`（errors 不升、warnings 按预期下降）。

---

## Sources

- `ruoyi-react-frontend/src/index.css`（5590 行；`:150-196` light token、`:198-268` dark token、`:871-929` Input 主题块、`:2212-2600+` 暗色覆盖块、`:2321-2323` `.ant-picker`）
- `ruoyi-react-frontend/src/design-system/components/AntdThemeBridge.tsx:63-190`（antd ConfigProvider 唯一入口、`components.Input` token、`hashed: true`）
- `ruoyi-react-frontend/src/components/ConfigProvider.tsx`（项目自有 ConfigProvider，**非** antd 的）
- `ruoyi-react-frontend/src/components/ui/AppInput.tsx`（默认 `allowClear=true`）、`AppSelect.tsx:16-19`（契约组件自身触发 `local/no-large-dropdown-list`）
- `ruoyi-react-frontend/src/components/shared/ActionButtons.tsx:15-24,114-159`（`ActionButton` 无 permission 字段）、`ColumnConfigModal.tsx:136`（封装层内的 `Modal.confirm`）
- `ruoyi-react-frontend/src/pages/operations/buildings/index.tsx:374-378`（行操作无门控）、`src/pages/network/ports/index.tsx:131-132`（对照：有门控）
- `ruoyi-react-frontend/src/lib/menuApi.ts:23-26`、`src/store/menuStore.ts`、`src/router/DynamicRoutes.tsx`（路由/LAST_PATH/菜单缓存）、`src/router/RouteGuard.tsx`、`src/constants/storage.ts:37`
- `ruoyi-react-frontend/src/components/operations/WorkstationDeviceTable/index.tsx:109-128,580-591`（物理链路面板，尚未接诊断端点）
- `ruoyi-react-frontend/src/pages/network/devices/index.tsx`（714 行；`:81-169` 纯函数、`:277` `deptFilters`、`:451` `buildRowActions`、`:524` `columns`）；`src/components/network/port-write/*.tsx`（v1.19/v1.20.1 冻结交互）
- `ruoyi-react-frontend/eslint.config.js`（`:59` react-refresh、`:106` exhaustive-deps、`:120-183` P65 契约块、`:184-263` hex 豁免块、`:271-283` 封装层豁免、`:291-317` P66 块）+ `scripts/exemption-reasons.json` / `scripts/check-exemption-reasons.mjs` / `scripts/assert-lint-budget.mjs`
- `ruoyi-react-frontend/tsconfig.app.json:14,16,39`（`verbatimModuleSyntax` / `noEmit` / `exclude`）
- `node_modules/antd@6.6.4`：`es/input/style/index.js:46-71,296-318,361-452`、`es/input/style/variants.js:25-40,122-175,177-231,280-345`、`es/input/style/textarea.js:4-60`、`es/input/Input.js:129,159-176`、`es/input/TextArea.js:82-161`、`es/form/hooks/useVariants.js:7-28`、`es/config-provider/context.js:4`
- `node_modules/@rc-component/input/es/BaseInput.js:43-45,79-88`、`es/TextArea.js:137,179`、`es/ResizableTextArea.js:114`
- `node_modules/@ant-design/cssinjs/es/hooks/useStyleRegister.js:27-47`（hash 前缀注入）
- `node_modules/eslint-plugin-react-refresh/index.js:95,178,194`；`node_modules/eslint-plugin-react-hooks@7.0.1`
- `pkg/permission/config.go:223-290`（静态路由权限表，无缓存）、`internal/services/system/menu_service.go:509-569`（用户权限聚合）、`internal/services/system/menu_cache_impl.go:115-136,157-162`（30 分钟权限缓存 + 失效路径）、`internal/services/system/cache_keys.go:414-420`
- `internal/core/db/init_data.go:693,752-755`（`ops:building:*` 权限码）、`internal/core/db/migrations/archive/applied/migration_159_align_ops_perms.go`（权限码命名漂移前科）、`internal/api/router.go:539-546,702`
- `internal/services/operations/workstation_device_service.go:800-887`（空因分类与告警，**注释自相矛盾**）、`internal/services/operations/workstation_physical_diagnosis.go:21-64`（空因常量与字段）、`internal/api/v1/operations/workstation_device_handler.go:147-191`
- `.planning/milestones/v1.23-phases/75-uat/75-UAT.md:51,74-87,105-106`（#16 FAIL-2 与环境限制）；`.planning/debug/workstation-3f130-physical-link-empty.md:89,106,113`；`.planning/milestones/v1.23-REQUIREMENTS.md:55,59,83,136-137`
- 本次实测：`npm run lint` → **17 errors / 34 warnings**（33 处 `react-refresh/only-export-components`，12 处 `react-hooks/exhaustive-deps`）

---
*Pitfalls research for: RuoYi-Next v1.24 视觉回归修复 + 遗留定责与清偿*
*Researched: 2026-10-28*
