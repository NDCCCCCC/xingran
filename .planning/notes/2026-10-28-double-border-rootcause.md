# v1.24 规划输入 — 双框（P0）根因实证

> 实测时间：2026-10-28｜证据来源：用户实机截图 + `node_modules/antd/dist/antd.css`（antd 6.6.4）原文静态核验
> 结论级别：**已实证（非推测）**

## 1. 用户可见现象

部分输入框出现「双重外框」：外层浅灰圆角矩形内又套一层白色圆角输入框。用户 2026-10-28 实机截图确认。

## 2. 根因链（完整实证）

### 2.1 antd 6.6.4 的原生契约

`node_modules/antd/dist/antd.css` 原文：

```css
.ant-input-affix-wrapper >input.ant-input,
.ant-input-affix-wrapper >textarea.ant-input{font-size:inherit;border:none;border-radius:0;outline:none;background:transparent;color:inherit;}
```

**antd 的设计意图**：当 Input 渲染为 affix wrapper 结构（带 `allowClear` / `prefix` / `suffix`）时——
- **边框只画在 `.ant-input-affix-wrapper` 上**
- 内层原生 `<input>` 元素必须 `border:none; border-radius:0; background:transparent; color:inherit`

DOM 结构：

```html
<span class="ant-input-affix-wrapper ant-input-outlined">
  <input class="ant-input ant-input-outlined" placeholder="...">   <!-- 内层：应无边框 -->
  <span class="ant-input-suffix"><span class="ant-input-clear-icon">×</span></span>
</span>
```

### 2.2 项目 CSS 的覆盖

`ruoyi-react-frontend/src/index.css:871-880`：

```css
.ant-input,
.ant-input-number,
.ant-select-selector {
  background: var(--input-bg) !important;
  border: 1px solid var(--input-border) !important;   /* ← 命中内层 input */
  color: var(--input-text) !important;
  border-radius: var(--theme-radius-md) !important;
  font-size: 14px !important;
  transition: all var(--theme-transition-fast) !important;
}
```

`index.css:908-914`（affix wrapper 自身）：

```css
.ant-input-affix-wrapper {
  background: var(--input-bg) !important;
  border: 1px solid var(--input-border) !important;
  ...
}
```

### 2.3 为什么 antd 的规则会输

| 规则 | 特异性 | important |
|---|---|---|
| antd `.ant-input-affix-wrapper >input.ant-input { border:none }` | 0,3,1 | **否** |
| 项目 `.ant-input { border: 1px solid ... }` | 0,1,0 | **是** |

CSS 规则优先级中，**`!important` 优先于非 `!important`**（与特异性无关）。antd 的 `border:none` 没有 `!important` → 项目规则胜出 → **内层 input 也被画出边框 → 双框**。

### 2.4 连带的 4 项视觉污染（同一机制）

| # | antd 原生（内层） | 被项目规则覆盖成 | 用户可见后果 |
|---|---|---|---|
| 1 | `border: none` | `1px solid var(--input-border)` | **双框（主症状）** |
| 2 | `border-radius: 0` | `var(--theme-radius-md)` | wrapper 圆角处内层轻微露角 |
| 3 | `background: transparent` | `var(--input-bg)` | 同色时不可见；variant 差异时露底 |
| 4 | 继承 wrapper 的 focus 环 | `index.css:888` `.ant-input:focus { box-shadow: 0 0 0 2px }` | **聚焦双环**：内层 input 画焦点环 + wrapper 通过 `.ant-input-affix-wrapper-focused`（`index.css:920`）也画焦点环 |

## 3. 为什么「有的输入框」才有，且 v1.23 后变普遍

- 只有渲染为 **affix-wrapper** 的 Input（带 `allowClear` / `prefix` / `suffix`）才有内层 input → 才双框
- **v1.23 之前**：绝大多数 Input 显式/隐式无 `allowClear` → 纯 `.ant-input` 单层结构 → 正常
- **v1.23 之后**：`AppInput` 契约把 `allowClear` 默认化为 `true` → **几乎所有 Input 变成 affix-wrapper** → 双框从少数变普遍

**结论：这是 v1.23「契约默认值下沉」的正向改造，撞上项目旧 CSS 覆盖链造成的视觉回归。** 契约本身没错，错在 CSS 层没有随结构变化同步收窄。

## 4. 精确修复候选（待研究定稿）

### 候选 A：显式还原内层（最小、最对症）

```css
/* affix-wrapper 内层 input 还原 antd 原生：边框/圆角/背景只在 wrapper */
.ant-input-affix-wrapper > .ant-input,
.ant-input-affix-wrapper > textarea.ant-input {
  border: none !important;
  border-radius: 0 !important;
  background: transparent !important;
}
```

+ **焦点环去重**（内层 `:focus` 的 box-shadow 置 `none`，由 wrapper 的 `.ant-input-affix-wrapper-focused` 独家承担）。

- 特异性 0,3,1 且同为 `!important` → 稳定压过项目 `.ant-input`（0,1,0）
- 只动 affix-wrapper 结构，**不影响**纯 `.ant-input`（无 wrapper 的 Input，如 `Input.Password` 不带 clear、readonly 回显区等）

### 候选 B：收窄主规则作用域

让 `.ant-input` 的 border 规则不命中 affix-wrapper 内层。

- **陷阱**：CSS `:not()` 在 Level 3 只支持简单选择器，`.ant-input:not(.ant-input-affix-wrapper > .ant-input)` **无效**；`:not()` 支持复杂选择器是 Selectors Level 4（Chrome 88+ / Safari 9+ 部分支持 / Firefox 84+ 支持）——现代浏览器可行，但需确认本项目目标浏览器基线
- 替代写法：`.ant-input-affix-wrapper .ant-input` 上下文选择器（无法直接「排除」，仍需显式还原，等价于候选 A）

### 候选 C：弃用手写边框规则，收敛到 antd 6 variant + ConfigProvider token

- 架构最干净，但改动面覆盖全站输入类组件，视觉回归风险最高
- 与项目既有 `design-system/`（`AntdThemeBridge.tsx` / `ThemeProvider.tsx`）的关系需研究确认——若已有 token 桥接，手写 `!important` 可能是历史遗留的双轨

## 5. 连带审计范围（研究任务已下发）

需核查是否还有其它 antd 6 结构被项目的 `!important` 规则误伤：

| 组件 | 边框承载元素 | 风险 |
|---|---|---|
| Input（affix wrapper） | `.ant-input-affix-wrapper` | **已确认双框** |
| TextArea | `.ant-input`（自身，无 wrapper） | 需确认项目 `.ant-input` 规则是否双重命中 |
| InputNumber | `.ant-input-number` + 内部 affix wrapper | 需核查 |
| Select | `.ant-select-selector`（内嵌于 `.ant-select`） | 需核查是否双层边框 |
| DatePicker / Cascader / TreeSelect | 各自的 selector 元素 | 需核查 |
| Checkbox / Radio / Switch | 非 border 承载 | 低风险 |

## 6. 视觉验证矩阵（修复后必测）

| 维度 | 取值 |
|---|---|
| 聚焦态 | 未聚焦 / **聚焦（双环去重验证）** |
| 配色模式 | 浅色 / 暗色 |
| 组件类型 | 普通 Input、allowClear Input、prefix Input、suffix Input、Password、Search、readonly、disabled、TextArea、Select、DatePicker、InputNumber |
| 页面场景 | 列表页搜索区（inline Form）、弹窗表单、表格内联编辑、暗色模式页 |

## 7. 流程教训（须写进 v1.24 Pitfalls）

> ⚠️ 本节编号之后的 **第 8 节**为 STACK 研究完成后追加的增补（含 3 处既有局部补丁、TextArea 100% 双框、F1/F2/F3、`:not()` 修正与最终方案推荐），**以第 8 节为准**。
1. **契约默认值下沉会改变 DOM 结构** —— 任何让组件「多一层 wrapper」的默认值变更，都必须同步审计 CSS 选择器命中面。v1.23 的 `allowClear` 默认化把 121 个文件的 Input 从单层变成双层结构，却没有任何样式层的闸门能发现。
2. **CSS 层缺少自动化守卫** —— type-check / vitest / eslint / build 四道闸门全绿，双框照样发生。视觉层要么补截图对比，要么补「结构契约」测试（如断言「affix-wrapper 内层 input 计算样式 border 为 none」）。
3. **`!important` 是覆盖链的隐形放大器** —— antd 特异性更高的规则被低特异性 + `!important` 压过，静态阅读 CSS 极易漏判。
4. **`grep` 检测面**（v1.23 教训）—— 本次双框若靠 grep 找 `.ant-input` 命中文件，同样会被换行形态骗过。

---

## 8. 【2026-10-28 增补】STACK 研究的三项关键实证（修正本文第 4 节）

### 8.1 「有的输入框才有双框」的真正解释：**已有 3 处局部补丁**

项目里早已存在 3 处针对同一问题的**局部修复**，它们无双框，其余全双框：

| 位置 | 内容 | Phase 77 实测复核 |
|---|---|---|
| `src/pages/login/login.css:351-358` | `.login-form .ant-input-affix-wrapper > input.ant-input { border:none; ... }` | ✅ **生效**（真实 DOM 命中 2 处；内层 0 框 / 透明底 / shadow none；登录页双框 = 0） |
| `src/index.css:2056-2060`（基线行号；Phase 76 加 13 处 `!important` 后下移，现 `:2084-2108`） | `.dept-tree-search` 域的同款还原 | ❌ **死规则，从未命中**（见下） |
| `src/index.css:2077-2081`（现 `:2111-2122`） | `.user-form-input` 域的同款还原 | ✅ **生效**（buildings / system-user 各命中 1） |

→ **这些补丁的存在本身就是「同一 bug 已被零散修过」的证据**；v1.24 修复时**不得删除这 3 处**（否则该 3 处回退成双框）。

#### ⚠️ v1.24 Phase 77 实测更正：第 2 处补丁是**从未生效的死规则**

第 2 处用的是**后代选择器** `.dept-tree-search .ant-input-affix-wrapper`（要求 `.dept-tree-search` 是**祖先**），
但实际 `className="dept-tree-search user-form-input"` 落在 **AppInput 自身**上（`DeptTree/index.tsx:205`），
即该 class 与 `.ant-input-affix-wrapper` **是同一个元素** → 后代选择器永不命中（实测命中数 = 0）。

**含义修正**：本节原表述「这 3 处都是已修过同一 bug 的证据 / 该 3 处修复前无双框」**不准确**——
**部门树搜索框在 Phase 76 之前其实也是双框**，它是靠 VIS-01 复位块才变成单框的（属于**改善**而非**保持**）。
该死规则无害，但**会误导维护者**以为该处已修过。

⇒ 后续维护提示：第 2 处应改写为**同元素选择器**（`.dept-tree-search.ant-input-affix-wrapper > input.ant-input`）
或直接删除（VIS-01 复位块已全局覆盖）——**本 phase 未改动**（Phase 77 边界为「不改业务代码」），登记为后续项。

### 8.2 TextArea **100% 双框**（比 Input 更严重）

`AppTextArea` 默认 `allowClear=true` → TextArea 也渲染为 affix-wrapper 结构，内层是 `textarea.ant-input`：

- antd 原生（`antd/es/input/style/textarea.js:50-61`）：`border: none`
- 项目 `index.css:871` 的 `.ant-input` 规则命中 `textarea.ant-input` → 覆盖为有边框 → **双框**
- 且 TextArea **无任何局部补丁** → 全站 TextArea 一律双框

### 8.3 连带三项同源问题（STACK 研究发现）

| # | 问题 | 机制 | 严重度 |
|---|---|---|---|
| **F1** | **Form 错误态边框不变红** | antd 的 status 边框色是普通声明（`antd/es/input/style/variants.js:41-51,63-78`），被项目 `border: ... !important` 简写压掉 | P1 |
| **F2** | **antd 6 已无 `.ant-select-selector` / `.ant-select-selection-item` / `.ant-select-selection-placeholder`** | antd 6 改用新类名；项目 **12 条 Select 规则全是死规则**（`index.css:873/884/890/897/2287/2293/2297/3920/3926/3930/5537`）→ 暗色 Select 完全走 antd token | P1 |
| **F3** | **暗色白底** | `design-system/AntdThemeBridge.tsx:83` `colorBgContainer: ruoyiBrand.cream.surface` = `#FFFFFF`（`design-system/tokens/colors.ts:76`），`darkAlgorithm` **不改写显式 token** → 任何「改走 antd token」的方案在暗色下会炸白底 | P1，**方案 C 的成败点** |

**其余组件的核查结论**：

- `InputNumber`：内层 `.ant-input-number-input` 的 `border:0` 且**不匹配** `.ant-input` → 无双框（但边框色被锁死）
- `DatePicker`：内层 input 无 `ant-input` class → 无双框
- `Checkbox` / `Radio`：无双框，但状态色被锁死
- `Select`：边框在 `.ant-select` 上（`antd/es/select/style/select-input.js:106-112`）→ **无双框**

### 8.4 修正：`:not()` 复杂选择器在现代浏览器**已支持**

本文第 4 节候选 B 与原任务书称「CSS `:not()` 不支持组合子选择器（CSS3 限制）」——**该说法在当前目标环境已过时**：

- `:not()` 支持复杂选择器属 **Selectors Level 4**，浏览器基线：Chrome/Edge 88+、Firefox 84+、Safari 16.4+
- 本项目无 `browserslist` 字段，Vite 7 默认 target 为现代浏览器 → **可用**
- 但仍需注意 `:not()` 会**抬高特异性**（如 `.ant-input:not(.ant-input-affix-wrapper > .ant-input)` → 0,2,0），可能盖住 antd 的 status 样式；而 `:where()` 可降特异性但**必须摘掉 `!important`**，否则暗色回落到 `cream.*` 浅色（F3）

### 8.5 STACK 研究的方案推荐（修正本文第 4 节）

| 方案 | 形态 | 评价 |
|---|---|---|
| **A（推荐）** | `index.css` 加 1 个规则块（≈6 行）：`.ant-input-affix-wrapper > .ant-input, .ant-input-textarea-affix-wrapper > .ant-input { border:none !important; background:transparent !important; box-shadow:none !important; }` | 1 文件、0 TS 改动、0 契约影响；焦点环不丢（antd 自身 `box-shadow:none !important` 仍压制内层）；须同时写 `background`（否则 `index.css:2257` 暗色 `background-color:var(--input-bg)!important` 残留） |
| B | 收窄主规则作用域（3 种实现，职责拆分 B-3 最优） | **不能单独做 P0**（必须连带解决 F3 暗色色板），且不能删 focus 规则（否则双焦点环） |
| C | 收敛到 antd 6 variant + ConfigProvider token | 需先重构 AntdThemeBridge 明暗色板（F3），并删/降级 4 批手写规则（≈-120 行）；触碰「ruoyiBrand 是 TS 侧唯一色值真相源」的 v1.22 冻结契约 → **推迟到 v1.25 独立里程碑** |

**推荐路线**：v1.24 = **方案 A 止血**（Phase 1）+ **B-3 仅限 Input 域的职责拆分**（Phase 2，不动 AntdThemeBridge）；方案 C 推迟。

### 8.6 回归陷阱（务必写进执行计划）

1. **不得删除** `login.css:351-358`、`index.css:2051-2088` 三处局部补丁
2. **不得删除** `index.css:892` 与 `index.css:2275-2284` 的 focus 规则——当前靠 antd 特异性 (0,3,0) 压住内层焦点环，误删会变双焦点环
3. **不得删除** `index.css:910`（wrapper 边框——它是唯一的边框来源）与 `index.css:916-923`（wrapper 焦点环——唯一来源）
