---
phase: 89
title: 变量层与明暗色板重构 — 技术方案研究
milestone: v1.25 全站权限治理与样式收敛
researcher: gsd-phase-researcher
researched: 2026-10-28
status: complete
scope: 只读研究（未修改任何业务代码，未 git commit；仅新建本文件）
confidence: HIGH（antd 6.6.4 源码实证 + 本机 node 实跑复算）/ MEDIUM（暗色色值设计决策）
---

# Phase 89 技术方案研究：变量层与明暗色板重构

> 本文件是 Phase 89 的**只读研究**产出。全部数字来自下文 § Sources 列出的**本机实跑命令输出**，
> 行号以当前 HEAD（`ruoyi-react-frontend/src/index.css` = **5548 行**，`wc -l` 口径，与 88-QUANT-SNAPSHOT 一致）为准。
> 置信度标注：**HIGH** = 源码实证 / 实跑复核；**MEDIUM** = 由实证推导，需实机验证；**LOW** = 待定。

---

## ⚠️ 0. 先行结论：3 条会改变 Phase 89 计划形状的发现

> 这四条必须在写 PLAN 之前裁决，否则 Phase 89 会按错误前提施工。

| # | 发现 | 证据 | 影响 |
|---|---|---|---|
| **X-1** | **STACK.md 的 F3-c「Map Token 显式值无效」与 REQUIREMENTS 的「机制更正」都不成立。** 真实机制是 `alias.js:14-22` 的 **override 回填**：非 seed 名的显式 token 在算法之后**被重新盖回**，所以 `colorBgContainer: #FFFFFF` 在暗色下**确实生效**（实跑：暗色 `colorBgContainer=#FFFFFF`）。 | §1.1 全链源码 + 4 组实跑 | **STYLE-01 的「修法」必须改写**：不是「删掉显式值让它派生」，而是「**按模式分支显式给值**」。原修法会把暗色边框变成刺眼的 `#20a252`（见 §1.1 实跑 C 组）。 |
| **X-1b** | **REQUIREMENTS/PITFALLS 的「F3 成因①：组件级 token 在算法后 merge → 暗色算法改不动」在本仓当前配置下不成立。** `components.Input/Select/Card.colorBgContainer = cream.surface(#FFFFFF)` **与全局 `colorBgContainer` 同值** ⇒ 被 `getComponentToken.js:26-30`「与全局同值即删除」**剔除** ⇒ 实测 `componentToken.colorBgContainer === undefined` ⇒ **组件级这三项今天是纯 no-op，不是病灶**。真正的病灶是**全局** `theme.token.colorBgContainer`（`AntdThemeBridge.tsx:83`）。 | §1.1.5 实跑 | ⚠️ **产生一条实现硬约束**：89 改全局为分模式值时，**组件级三项若不同步删除**，它们就不再与全局同值 ⇒ 被保留 ⇒ `genCSSVarRegister` 会在 `.ant-input`/`.ant-select`/`.ant-card` 作用域注入 `--ant-color-bg-container: #FFFFFF` ⇒ **暗色白底从组件作用域重新长出来**。**必须同 commit 删除。** |
| **X-2** | **ROADMAP Phase 89 成功标准 #1 内部自相矛盾**：「暗色 `--input-bg` = `#0f2e1b`」与「输入框底与页面底对比度 ≥ 3:1」**不可同时成立**——暗色页面底就是 `#0f2e1b`（`index.css:390` `body{background:var(--bg-base)}` + `:200 --bg-base:#0f2e1b`），两者对比度 **1.00:1**；且 `#0f2e1b` 对**任何**背景的对比度上限只有 1.43:1（vs 纯黑），**数学上不可能达到 3:1**。 | §1.2.4 WCAG 实算 | 必须**改写判据**：3:1 只能落在**描边**（WCAG 1.4.11 管的是组件边界，不是填充），而现有暗色描边 `#2d4d33` 只有 1.56:1 → 需要提亮到 `#4a7d52`(3.05:1) 级别，而这属于 Phase 92 的暗色 mega-cluster 范围。**建议 89 登记为显式例外 + 移交 92**。 |
| **X-3** | **PITFALLS 第 2 批的 T1/T2/T3 是 Phase 90「STYLE-03 圆角统一」的交付物**，不是 Phase 89 的。若 89 执行 T1/T2/T3，矩阵 radius 会有 ~15 格差异，**直接击穿 89 自己的「除已登记 2 处外差异 = 0」硬判据**，并抢跑 Phase 90 的成功标准 #3。 | §1.5 逐条复核 | **89 的 T 批次只做「色值类」降级（T6/T7/T8/T9–T12/T13 提取变量/T14 迁移），圆角三条（T1/T2/T3）整体移交 Phase 90**。 |

---

## 1. 现状复核

### 1.1 F3 两层成因 —— 全链源码定位 + 本机实跑复算

#### 1.1.1 antd 6.6.4 主题 token 派生链（逐行）

```
<ConfigProvider theme={{ token, components, algorithm }}>        antd/es/config-provider/index.js
  │
  ├─① config-provider/index.js:213   mergedTheme = useTheme(theme, parentContext.theme, …)
  ├─② config-provider/index.js:392   themeObj = createTheme(algorithm)   ← darkAlgorithm 实例
  ├─③ config-provider/index.js:408   mergedToken = { ...defaultSeedToken, ...token }
  ├─④ config-provider/index.js:412-422
  │      return {
  │        theme: themeObj,
  │        token: mergedToken,                       // seed + 显式 token（全名）
  │        components: parsedComponents,
  │        override: { override: mergedToken, ...parsedComponents },   // ★★ 关键
  │        cssVar
  │      }
  │
  ├─⑤ useToken.js:102   useCacheToken(themeObj, [defaultSeedToken, rootDesignToken], { override, getComputedToken, … })
  │      └─ useCacheToken.js:86   mergedToken = Object.assign({}, defaultSeedToken, rootDesignToken)
  │                                // 显式 token 覆盖 seed（同名键）
  │
  ├─⑥ useToken.js:49-59  getComputedToken(originToken, overrideToken, theme)
  │      derivativeToken = theme.getDerivativeToken(originToken)     ← 算法跑
  │      mergedDerivativeToken = { ...derivativeToken, override }    ← override 挂进来
  │
  ├─⑦ themes/default/index.js:29-45  derivative(token)
  │      { ...token,                    ← 显式值先进来
  │        ...colorPalettes,
  │        ...genColorMapToken(token, {…}),   ← ★ Map Token 在此覆盖显式值
  │        ...genFontMapToken / genSizeMapToken / genControlHeight / genCommonMapToken }
  │      themes/dark/index.js:26-51   mergedMapToken = defaultAlgorithm(token)
  │                                    colorMapToken = genColorMapToken(token, {…})
  │                                    return { ...mergedMapToken, ...colorPalettes, ...colorMapToken, … }
  │                                    ← 暗色再次用 genColorMapToken 覆盖
  │
  ├─⑧ ★★ alias.js:9-23  formatToken(derivativeToken)         ←★ STACK.md 漏掉的这一层
  │      const { override, ...restToken } = derivativeToken;
  │      const overrideTokens = { ...override };                 // = mergedToken（seed+显式）
  │      Object.keys(seedToken).forEach(t => delete overrideTokens[t]);  // ★ 只删 **seed 名**
  │      const mergedToken = { ...restToken, ...overrideTokens }; // ★ 非 seed 的显式值被**盖回**
  │
  ├─⑨ useToken.js:62-78  components 逐个挂到 token 上（算法之后）
  │      mergedDerivativeToken[key] = componentTokens            ← ★ 组件 token 压过全局 token
  │
  └─⑩ cssinjs/useCacheToken.js:103  transformToken → --ant-* CSS 变量
```

**seed token 全集**（`antd/es/theme/themes/seed.js:19-70`，唯一权威）：
`colorPrimary / colorSuccess / colorWarning / colorError / colorInfo / colorLink / colorTextBase / colorBgBase /
fontFamily / fontFamilyCode / fontSize / lineWidth / lineType / motion* / borderRadius / sizeUnit / sizeStep /
sizePopupArrow / controlHeight / zIndexBase / zIndexPopupBase / opacityImage / wireframe / focusOutline / motion` + 15 个 preset 色。

> **seed 里没有 `colorBgContainer` / `colorBgElevated` / `colorBgLayout` / `colorBorder` / `colorBorderSecondary`。**
> ⇒ 它们是 **Map Token**（由 `genColorMapToken` → `generateNeutralColorPalettes` 派生），**不在** `alias.js` 的删除名单里
> ⇒ ⇒ **显式设置的 Map Token 在算法之后会被 `alias.js:22` 完整盖回 → 显式值生效。**

#### 1.1.2 本机实跑复算（4 组配置 × 明暗）

> 脚本：`antd/lib/theme/useToken.js#getComputedToken` + `cssinjs#createTheme`，**完整复刻** ⑥→⑨ 链路
> （含 `config-provider/index.js:417-420` 的 `override: { override: mergedToken, ...components }` 结构）。
> 输出（摘要）：

| 组 | 配置 | `colorBgBase` | `colorBgContainer` | `colorBgElevated` | `colorBgLayout` | `colorBorder` | `colorBorderSecondary` | `colorText` |
|---|---|---|---|---|---|---|---|---|
| **A** | **现状**（显式 map + 浅 seed） | `#F0ECE3` | **`#FFFFFF`** | `#FFFFFF` | `#F0ECE3` | **`#DBD7CE`** | `#C2BDB2` | `rgba(16,16,16,0.85)` |
| **B** | 只删显式 map，seed 仍浅奶油 | `#F0ECE3` | `#fefefe` | `#ffffff` | `#ebeae8` | **`#ffffff`** | `#ffffff` | `rgba(16,16,16,0.85)` |
| **C** | seed 切暗 `#0f2e1b`，map 全删 → 纯算法派生 | `#0f2e1b` | `#11552b` | `#146633` | `#0a331a` | **`#20a252`** | `#1a8443` | `rgba(240,236,227,0.85)` |
| **D** | seed 切暗 + **显式** map = 项目暗色板 | `#0f2e1b` | `#1a2e1f` | `#1f3524` | `#0f2e1b` | `#2d4d33` | `#1a2e1f` | `rgba(240,236,227,0.85)` |

**读数结论（4 条，全部 HIGH）**：

1. **A 组 = 现状病灶坐实**：暗色下 `colorBgContainer=#FFFFFF`（Select 白底）、`colorBorder=#DBD7CE`（浅灰边）、
   `colorText=rgba(16,16,16,0.85)`（**深色文字**！）。⇒ 显式值确实生效，**X-1 成立**。
2. **B 组证明「只删显式 map token」不够**：seed 仍浅奶油时，算法派生出来的是 `#fefefe` / `#ffffff` 边框 —— **一样是白**。
   ⇒ **seed 层必须同治**（与 ROADMAP「两层同治」一致）。
3. **C 组证明「删掉后让算法派生」不可行**：`colorBgBase=#0f2e1b`（饱和深绿）经 `getSolidColor(base, 26)` 派生出
   **`colorBorder=#20a252`（刺眼亮绿）**，且 `colorBgContainer=#11552b` **比页面底 `#0f2e1b` 更暗**（层级倒置）。
   ⇒ antd 的 dark 派生公式是为**中性灰**基底设计的，喂饱和品牌绿会产出错误色相与错误层级。
4. **D 组 = 推荐形态**：显式 map 值 = 项目现有暗色板 → **与 `index.css` 手写暗色 cluster 完全同值** → 视觉零漂移。

> **对 STACK.md 的更正（重要）**：STACK.md §0 表 F3-c 写「`darkAlgorithm` 不会改写显式 token」这条**机制不成立**，
> 其推理链（`genColorMapToken` 在 `...token` 之后 ⇒ 显式值无效）**漏掉了 `alias.js:14-22` 的 override 回填**。
> REQUIREMENTS.md 第 40–42 行的「机制更正」**沿用了同一个错误结论**，两者都需在本 phase 修正。
> 症状（暗色白底）不变，**修法必须改**。

#### 1.1.3 `AntdThemeBridge.tsx` 现有显式 token 分类表（逐条）

> 文件：`ruoyi-react-frontend/src/design-system/components/AntdThemeBridge.tsx`（192 行）
> 判定依据：`seed.js` 名单（是否被 `alias.js` 删除）+ `useToken.js:62-78`（组件 token 在算法之后 merge）。

**A. `theme.token`（`:74-92`，共 16 项）**

| 行 | token | 分类 | 算法能否改写 | 暗色实际值（实跑 A 组） | 评价 |
|---|---|---|---|---|---|
| 75 | `colorPrimary` | **Seed** | 会被重算为暗色调色板 | `#15552d` | ✅ 入口正确，**保持不动** |
| 76 | `colorInfo` | Seed | 同上 | — | ✅ |
| 77 | `colorLink` | Seed | 同上 | — | ✅ |
| 78 | `colorSuccess` | Seed | 重算 | — | ✅ |
| 79 | `colorWarning` | Seed | 重算 | — | ✅ |
| 80 | `colorError` | Seed | 重算 | `#a1312c` | ✅ |
| 81 | `colorTextBase` | **Seed** | 是（作为派生输入） | `rgba(16,16,16,…)` | ❌ **浅色基底，暗色必炸** → 需按模式分支 |
| 82 | `colorBgBase` | **Seed** | 是（派生入口） | — | ❌ **浅奶油 → 暗色仍浅** → 需按模式分支 |
| 83 | `colorBgContainer` | **Map** | **显式值生效**（`alias.js:22`） | `#FFFFFF` | ❌ 需按模式分支 |
| 84 | `colorBgElevated` | Map | 显式生效 | `#FFFFFF` | ❌ 需按模式分支 |
| 85 | `colorBgLayout` | Map | 显式生效 | `#F0ECE3` | ❌ 需按模式分支 |
| 86 | `colorBorder` | Map | 显式生效 | `#DBD7CE` | ❌ 需按模式分支 |
| 87 | `colorBorderSecondary` | Map | 显式生效 | `#C2BDB2` | ❌ 需按模式分支 |
| 89 | `borderRadius` | Seed（尺寸） | 不改写 | 8 | ✅ 保持（与 Phase 90 的圆角统一同源） |
| 90 | `borderRadiusLG` | Map（尺寸） | 不改写 | 12 | ✅ |
| 91 | `fontFamily` | Seed | 不改写 | — | ✅ |

**B. `theme.components`（`:93-168`，8 组件）—— 算法之后 merge，算法改不动**

| 行 | 组件 / 键 | 分类 | 暗色实际值 | 评价 |
|---|---|---|---|---|
| 95-104 | `Button.*`（`colorPrimary`/`primaryColor`/`colorPrimaryHover`/`colorPrimaryActive`/`defaultBorderColor`/`borderRadius`/`controlHeight`/`fontWeight`） | 组件 token | `defaultBorderColor=#DBD7CE` | ⚠️ `defaultBorderColor` 在暗色下仍是浅灰 → 需按模式 |
| 107-118 | `Table.*`（`headerBg`/`headerColor`/`borderColor`/`rowHoverBg`/`headerSortActiveBg`/`headerSortHoverBg`/`headerFilterHoverBg`/`fixedHeaderSortActiveBg`/`rowSelectedBg`/`rowSelectedHoverBg`/`borderRadius`） | 组件 token | 全为 `cream.*` 浅值 | ⚠️ 暗色下表头=浅绿灰 → 需按模式（本次可不改，交 Phase 91/92，但须登记） |
| 122 | **`Input.colorBgContainer = cream.surface`** | 组件 token | **`#FFFFFF`** | ❌ **VISFIX-02 元凶之一**（层 ①） |
| 123-127 | `Input.activeBorderColor`/`hoverBorderColor`/`activeShadow`/`borderRadius`/`controlHeight` | 组件 token | 品牌绿（模式无关，合理） | ✅ 保持 |
| 130 | **`Select.colorBgContainer = cream.surface`** | 组件 token | **`#FFFFFF`** | ❌ **VISFIX-02 直接元凶**（层 ①） |
| 131-136 | `Select.activeBorderColor`/`hoverBorderColor`/`borderRadius`/`controlHeight`/`optionSelectedBg` | 组件 token | `optionSelectedBg=#E9EFEB` | ⚠️ `optionSelectedBg` 暗色下为浅绿灰 → 需按模式 |
| 139-148 | `Menu.itemBg`/`itemSelectedBg`/`itemSelectedColor`/`itemHoverBg`/`darkItem*` | 组件 token | 浅色值 | ⚠️ 暗色下选中底=浅绿灰 → 需按模式 |
| 150-155 | `Tabs.itemSelectedColor`/`itemActiveColor`/`itemHoverColor`/`inkBarColor` | 组件 token | 品牌绿 | ✅ 模式无关，合理 |
| 157-161 | `Tag.defaultBg`/`defaultColor`/`borderRadiusSM` | 组件 token | 淡黄 + 铜金 | ⚠️ 暗色下淡黄底未适配 → 登记，可留 Phase 92 |
| 164 | **`Card.colorBgContainer = cream.surface`** | 组件 token | **`#FFFFFF`** | ❌ 层 ① |
| 165 | `Card.headerBg` | 组件 token | `transparent` | ✅ |
| 166 | `Card.borderRadiusLG` | 组件 token | 12 | ✅ |

**C. 生效点确认（P-C2 复核）**

| 事实 | 证据 |
|---|---|
| 真正的 antd `<ConfigProvider theme={…}>` 在 | **`AntdThemeBridge.tsx:178`** |
| `src/components/ConfigProvider.tsx` **不渲染任何 Provider** | 该文件 `:69` `return children;`，只有 `useEffect`（`:37/:44/:56/:64`）做 store 同步 |
| 嵌套顺序 | `src/App.tsx:47-62`：`<AntdThemeBridge>` → `<ConfigProvider>`（项目） → `<ThemeProvider>` → children |
| `data-color-mode` 写到 `<html>` | `src/design-system/components/ThemeProvider.tsx:22/:30/:42` `root.setAttribute('data-color-mode', mode)` |
| 结论 | **改 `AntdThemeBridge.tsx` 即全站生效**；项目 `ConfigProvider` 保持冻结（P-C2） |

#### 1.1.4 双 ConfigProvider 之外的隐藏坑：`useMemo` 缓存粒度

`AntdThemeBridge.tsx:63-175` 的 `useMemo` deps = `[themeMode, density]`。
`themeMode` 已在 deps 内 ⇒ **按模式分支设 `colorBgBase` 不需要扩 deps**（`themeMode` 变化即重算）。
⚠️ 但若把色板选择改为读取**新的** store 字段 / 常量表，则**必须**同 commit 扩 deps —— 现有 C15 单测
（`src/design-system/components/__tests__/AntdThemeBridge.test.tsx:95-109`）**硬断言 `deps.length === 2`**，
新增依赖会让该测试**红**。⇒ **Phase 89 不得新增 useMemo 依赖项**（保持 `[themeMode, density]`），
否则必须同步改 C15 断言（不建议，属额外风险面）。

---

### 1.2 变量层现状复核（行号已按 88-03 后 HEAD 重新定位）

#### 1.2.1 `--input-bg` 三处定义 + 20 个消费点

| 位置 | 行 | 内容 | 作用域 | 生效顺序 |
|---|---|---|---|---|
| **定义 1** | `index.css:184` | `--input-bg: #ffffff;` | `:root` 块（起于 `:27`） | 最早 |
| **定义 2** | `index.css:254` | `--input-bg: #0f2e1b;` | `[data-color-mode="dark"]` 块（起于 `:198`） | 中 |
| **定义 3** | `index.css:4734` | `--input-bg: #fafaf9; /* 输入框底（原型实测） */` | **后置 `:root` 块（起于 `:4731`）** | **最后 → 胜出** |
| 浅色值来源 | `index.css:190` | `--input-placeholder: #707068;` | `[data-color-mode="light"]` 块（起于 `:150`） | 仅 light |
| 暗色值来源 | `index.css:267` | `--input-text-placeholder: #707068;` | `[data-color-mode="dark"]` 块 | 仅 dark |

**特异性判定**：`data-color-mode` 写在 `<html>` 上 ⇒ `:root`（0,1,0）、`[data-color-mode="dark"]`（0,1,0）、
`[data-color-mode="light"]`（0,1,0）**三者特异性完全相同** ⇒ **纯靠源码顺序决胜**。
`:root`@4731 在最后 ⇒ **暗色下 `--input-bg` 最终 = `#fafaf9`（浅色）** ⇒ **P-C3 坐实**。

**20 个消费点**（`index.css`，`var(--input-bg)` 16 处 + 状态变量 4 处）：

| 类别 | 行号 | 备注 |
|---|---|---|
| `var(--input-bg)`（16） | 873, 905, 1330, 2269, 2329, 2748, 2753, 2758, 3695, 3724, 3737, 3923, 3941, 4196, 4404, 4934 | 873 = `.ant-input/.ant-input-number` 规则组；905 = `.ant-input-affix-wrapper`；1330 = `.ant-pagination-item`（`var(--component-bg)`? 实为 `.ant-radio-inner/.ant-checkbox-inner` 暗色块）；2269 = 暗色 input/textarea 通用块；2329 = 暗色 Select 块；3695/3724/3737 = 暗色 `.ant-input` / wrapper / `.ant-input-number`；4934 = 某组件块 |
| `var(--input-bg-hover)` | 3701 | 暗色块 |
| `var(--input-bg-focus)` | 3706, 3729 | 暗色块 |
| `var(--input-bg-disabled)` | 3714 | 暗色块 |

> PITFALLS 记的是「22 处」；88-03 删死规则后**实为 20 处**（16 + 4）。

#### 1.2.2 三个候选修法对比

| 方案 | 做法 | 暗色结果 | 浅色结果 | 视觉 diff | 评价 |
|---|---|---|---|---|---|
| **① 删后置 `:root`@4734** | 直接删掉那一行 | `#0f2e1b` ✅ | **`#ffffff`**（回落到 `:root`@27 的定义）→ 输入框底由 `#fafaf9` 变 `#ffffff` | ❌ **浅色多一处未登记变更** | ❌ 不推荐 |
| **② 浅色值并入 `[data-color-mode="light"]`@150 + 删 4734** | light 块加 `--input-bg:#fafaf9`，删后置 `:root` 那行 | `#0f2e1b` ✅ | `#fafaf9`（不变）✅ | ✅ **浅色 0 差异** | ✅ **推荐** |
| **③ 提升暗色块特异性**（如 `[data-color-mode="dark"][data-color-mode]` 或 `:root[data-color-mode="dark"]`） | 不动 4734，把暗色块提到 (0,2,0) | `#0f2e1b` ✅ | `#fafaf9` ✅ | ✅ 0 差异 | ⚠️ 可行但**留下 `:root`@4734 这颗雷**（它继续覆盖一切后置同名变量），且与 88 之后「变量单一来源」方向背离 |

**⇒ 推荐 ②**，理由：
- 浅色模式**零差异**（方案 ① 会产生未登记的浅色变更）；
- `:root`@4731 这个「原型复刻残渣块」只剩 `--theme-font-display` / `--theme-zebra` / `--sidebar-gradient`
  三项（`--input-bg` 移走后），其中 `--theme-zebra` 也应一并迁走（见 §1.5 T14）；
- 与 Phase 92「暗色 mega-cluster 整块交还 token」方向一致：**消灭「同一变量多处就地重定义」**（C9 断言的目标口径）。

#### 1.2.3 `--input-bg-hover/-focus/-disabled` 只在暗色定义

| 变量 | 定义行 | 引用行 | 现状风险 |
|---|---|---|---|
| `--input-bg-hover` | 255（dark）`#1a2e1f` | 3701（dark 块） | 浅色下**无定义**（但引用也只在 dark 块 → 声明不生效，**当前无害**） |
| `--input-bg-focus` | 256（dark）`#0f2e1b` | 3706, 3729（dark 块） | 同上 |
| `--input-bg-disabled` | 257（dark）`#0a1f12` | 3714（dark 块） | 同上 |

**「底浅态深」的半明半暗混合态**：修 ② 之前，暗色 `--input-bg` 解析为 `#fafaf9`（浅），
而 hover/focus/disabled 分别为 `#1a2e1f`/`#0f2e1b`/`#0a1f12`（深）⇒ **同一控件的常态底与交互态底跨了明暗两极**。
修 ② 之后：常态 `#0f2e1b` / hover `#1a2e1f` / focus `#0f2e1b` / disabled `#0a1f12` ⇒ **同族深绿，混合态消失** ✅。

**统一方案（建议）**：四个变量**成组**落到 light/dark 双分区：
- `[data-color-mode="light"]`（@150）：`--input-bg: #fafaf9; --input-bg-hover: #f4f2ec; --input-bg-focus: #fafaf9; --input-bg-disabled: #eceae4;`
  （浅色态值项目当前**没有** → 属**新增定义但当前无引用**，不会产生任何渲染变化；仅为消除「只在暗色定义」的结构脆弱）
- `[data-color-mode="dark"]`（@198）：保持 254-257 现值不动。
> ⚠️ **注意**：浅色三态若被引入，**必须确认浅色模式下无规则引用它们**（当前 `var(--input-bg-hover)` 只在 dark 块），
> 否则会**新造出浅色态变化**（未登记）。建议：**89 只做「结构对称」的定义补齐，不新增任何引用点**。

#### 1.2.4 R11：placeholder 变量命名不统一 —— 含**一处 AA 回归预警**

| 行 | 声明 | 所在块 | 有效性 |
|---|---|---|---|
| 189 | `--input-placeholder: #707068;` | `[data-color-mode="light"]`@150 | ✅ 仅 light |
| 267 | `--input-text-placeholder: #707068;` | `[data-color-mode="dark"]`@198 | ✅ 仅 dark |
| 900 | `color: var(--input-placeholder) !important;` | `.ant-input::placeholder`（无模式限定） | ⚠️ 暗色下**整条声明作废**（变量未定义）→ 回落到继承值 |
| 2276 | `color: var(--input-placeholder) !important;` | `[data-color-mode="dark"] .ant-input::placeholder, … textarea::placeholder` | ❌ **暗色下整条作废**（C6 豁免登记） |
| 2301 | `color: var(--input-placeholder) !important;` | `[data-color-mode="dark"] .ant-select-selection-placeholder` | ❌ **作废**；且该选择器在 antd 6 **不存在**（真实类名 `.ant-select-placeholder`，见 STACK.md A4.2）⇒ **双重死规则** |
| 3720 | `color: var(--input-text-placeholder) !important;` | `[data-color-mode="dark"] .ant-input::placeholder` | ✅ 生效（这是暗色占位符的**唯一有效来源**） |

**C6 豁免表 3 处的行号核实**（`scripts/css-contract-check.js:133-140`）：登记为 `2276 / 2301 / 4472`。
本 phase 实测：2276 ✅、2301 ✅、4472 ✅ —— **三处行号全部仍成立，无需修正**（88-03 后行号已随之更新过）。

**⚠️ 新发现（AA 回归预警，HIGH）**：修好 R10（暗色 `--input-bg` → `#0f2e1b`）后，
暗色占位符 `#707068`（由 3720 生效）的对比度将从 **4.81:1（现状 #707068 on #fafaf9）→ 2.95:1（#707068 on #0f2e1b）**
⇒ **从 AA 达标跌到 AA 不达标**。这不是 89 引入的缺陷，而是 89 的修复**暴露**出来的既有暗色板问题。
候选提亮值（vs `#0f2e1b`）：`#8f8f85` ≈ 3.6:1｜`#9a9a90` ≈ 4.2:1｜`#a3a399` ≈ 4.7:1（**AA 达标**）。

**统一方案（推荐）**：
1. **统一变量名为 `--input-text-placeholder`**（暗色已有、语义更准）；
2. 在 `[data-color-mode="light"]`@150 把 `--input-placeholder: #707068` 改名为 `--input-text-placeholder: #707068`
   （**值不变** ⇒ 浅色 0 差异）；
3. `:root`@27 补兜底 `--input-text-placeholder: #707068;`（保证任何作用域都有定义，C6 断言天然通过）；
4. 900 / 2276 / 2301 三处引用改名；**同时删除 2301 整条规则**（`.ant-select-selection-placeholder` 在 antd 6 不存在 = 死规则）；
5. **C6 豁免表 3 条全部删除**（`css-contract-check.js:133-140`），因为变量已全部有定义；
6. 暗色占位符提亮到 `#a3a399`（AA 4.7:1）⇒ **这是一处新增的暗色视觉变更，必须登记为第 3 处有意变更**
   （若不想在本 phase 扩大变更面，则**保留 `#707068` 并显式登记为 AA 欠账移交 Phase 92**）。

#### 1.2.5 `--theme-warning-text` / `--radius` 全局未定义

| 变量 | 引用行 | 引用形态 | C6 判定 | 建议 |
|---|---|---|---|---|
| `--theme-warning-text` | **4472** | `color: var(--theme-warning-text) !important;`（`.xr-tag-warning` 类，**无 fallback**） | ❌ 违规（C6 豁免登记中） | **定义**。`:root`@27 已有 `--theme-warning-bg: #fef3c7`；同族补 `--theme-warning-text: #905d00`（= `ruoyiBrand.functional.warningText`）⇒ 值与 5178 的 fallback `#905d00` **一致** ⇒ 渲染不变 ✅ |
| `--theme-warning-text` | 5178 | `color: var(--theme-warning-text, #905d00);`（**有 fallback**） | ✅ 通过 | 保留（定义后 fallback 不再生效，值相同） |
| `--radius` | 5515 | `border-radius: var(--radius, 8px);`（**有 fallback**） | ✅ 通过 | **保留不动**（fallback 生效 = 8px；定义一个全局 `--radius` 反而会与 `--theme-radius-*` 体系重复）。⇒ **不处理**，登记为「有意保留 fallback」 |

#### 1.2.6 其它变量层事实（供 89 顺带核对）

| 事实 | 行 | 影响 |
|---|---|---|
| `body { background: var(--bg-base) }` | `index.css:389-390` | 暗色页面底 = `--bg-base` = **`#0f2e1b`**（`:200`）⇒ **X-2 的实锤** |
| `--theme-text-inverse: #ffffff`（`:root`@27:78），**暗色块未重定义** | 78 | ⇒ T8（`color:#ffffff` → `var(--theme-text-inverse)`）**值不变**，渲染零差异 ✅ |
| `--theme-text-primary`：light `#101010`(:74) / dark `#f0ece3`(:231) | 74/231 | ⇒ T13 若并入 `--theme-text-primary`，**浅色 `#262626`→`#101010` 会变色** ⇒ T13 不可直接并入 |
| `--theme-bg-tertiary`：light `#e9efeb`(:68) / dark `#1f3524`(:210) | 68/210 | ⇒ T14「并入 `--theme-bg-tertiary`」会**改色**（`#faf8f3` → `#e9efeb`）⇒ **T14 建议值不成立** |
| `--theme-zebra: #faf8f3` 在后置 `:root`@4733（**两模式共用**） | 4733 | 引用点 1021（`.ant-table-tbody` 偶行，暗色被 1024-1029 覆盖 ✅）与 **5118（`.xr-table-zebra`，暗色无覆盖 ⇒ 暗色下近白斑马纹 = 既有缺陷）** |
| `--table-header-bg`：light `#e9efeb`(:190) / dark `#1f3524`(:258) | 190/258 | 已是双分区变量 ✅ |

---

### 1.3 hex 治理现状（STYLE-04）

#### 1.3.1 扫描器能力边界

`ruoyi-react-frontend/scripts/check-hardcoded-colors.mjs`（190 行）：

| 维度 | 实况 |
|---|---|
| 扫描范围 | `src/` 递归，**`.ts` / `.tsx` / `.css`**（`walk()`，`:86-97`）—— **不扫 `.less`** |
| 命中口径 | **仅 31 条固定 hex 映射**（`HEX_MAPS`，`:49-81`）+ **7 条 rgba 映射**（`RGBA_MAPS`，`:38-46`）；**任意新 hex 直接漏过** |
| 路径白名单 | `src/utils/three/colors.ts`、`src/design-system/tokens/`、`scripts/`（`:32`）—— **不含 `src/design-system/themes/`** |
| 值白名单 | 4 个：`#95de64` `#ff7875` `#ffc53d` `#69c0ff`（`:35`） |
| 退出码 | 有命中 → `exit(1)`（`:190`）；`--fix` 模式按 `HEX_MAPS` 批量替换（`:154-164`，**批量替换有误伤风险，不建议在生产用**） |
| 是否接 CI | ❌ **未接入**。`package.json` 的 `lint:ci` = `eslint . --max-warnings 0 && node scripts/assert-lint-budget.mjs && node scripts/check-exemption-reasons.mjs && node scripts/css-contract-check.js` —— **不含** `check-hardcoded-colors.mjs` |

#### 1.3.2 当前 exit=1 的文件清单（**实跑：17 个文件 / 150 处命中**）

> ⚠️ **与 PITFALLS / REQUIREMENTS 记的「18 个文件」不符 —— 实为 17 个**。
> 差额来源：PITFALLS 统计时存在 `src/design-system/themes/theme-styles.css`（D16），Phase 88-03 已删除它 ⇒ 18 − 1 = 17。
> **REQUIREMENTS.md 第 49 行的「18 个文件命中」需在本 phase 更正为 17。**

| # | 文件 | 命中数 | 行号 | 存活判定 | 处置建议 |
|---|---|---|---|---|---|
| 1 | `src/index.css` | **2** | 960, 966 | **活** | ✅ **T6/T7 直接消灭**（`var(--ant-color-error, #ff4d4f)` → `var(--ant-color-error)`；`--ant-color-warning` 同）。antd 6 `cssVar` 默认注入 `--ant-color-error:#ff4d4f` ⇒ fallback **永不生效** ⇒ **零渲染差异** |
| 2 | `src/components/dashboard/DashboardView.css` | **1** | 50 | **活** | `color: var(--theme-text-tertiary, #8c8c8c)` —— `--theme-text-tertiary` 在 `:root`:86 = `#598e5e`、dark:233 = `#c2bdb2` **两模式都有定义** ⇒ **fallback 是死值** ⇒ 去掉 fallback，**零渲染差异** ✅ |
| 3 | `src/components/dashboard/widgets/types/__tests__/statAndProgress.test.tsx` | **3** | 41, 54, 67 | **活（测试）** | `#1890ff` 是**断言里的期望值**（图表色板）⇒ 改值会改测试语义。两条路：① 断言改为读 `ruoyiBrand.functional.info`（`#337AB0`）—— **期望值变化 = 测试语义变化**；② 把该文件加进 `PATH_ALLOW` 并写明理由（图表色板是**数据可视化语义**，不是 UI 主题色）。**推荐 ②**（改动面最小、不动测试语义） |
| 4 | `src/design-system/themes/index.ts` | 3 | 23, 196, 337 | 💀 **死** | 见 §1.3.3 |
| 5 | `src/design-system/themes/flat2.0/dark.ts` | 27 | 4,18,18,19,21×9,23,24,25,26,26,38,39,40,41,46,47,48,107,107 | 💀 **死** | 删除 |
| 6 | `src/design-system/themes/flat2.0/light.ts` | 27 | 18,18,19,21×9,24,25,26,27,27,28,33,34,41,42,43,44,50,51,52 | 💀 **死** | 删除 |
| 7 | `src/design-system/themes/glassmorphism/dark.ts` | 8 | 18,18,23,24,25,26,26,41 | 💀 **死** | 删除 |
| 8 | `src/design-system/themes/glassmorphism/light.ts` | 10 | 18,18,21,24,25,26,27,27,28,44 | 💀 **死** | 删除 |
| 9 | `src/design-system/themes/luxury-quiet/dark.ts` | 5 | 23,24,25,26,27 | 💀 **死** | 删除 |
| 10 | `src/design-system/themes/luxury-quiet/light.ts` | 8 | 28,28,29,33,34,35,36,37 | 💀 **死** | 删除 |
| 11 | `src/design-system/themes/minimal/dark.ts` | 5 | 22,23,24,25,25 | 💀 **死** | 删除 |
| 12 | `src/design-system/themes/minimal/light.ts` | 13 | 18,18,20,20,20,23,24,25,27,42,43,50,51 | 💀 **死** | 删除 |
| 13 | `src/design-system/themes/neumorphism/dark.ts` | 8 | 21,23,23,26,27,28,29,29 | 💀 **死** | 删除 |
| 14 | `src/design-design-system/themes/neumorphism/light.ts` → `src/design-system/themes/neumorphism/light.ts` | 15 | 22,24,24,25,25,25,25,25,25,28,29,30,31,32,38 | 💀 **死** | 删除 |
| 15 | `src/design-system/components/ColorSwitcher.tsx` | 6 | 52,55,267,288,315,361 | 💀 **死组件** | 删除 |
| 16 | `src/design-system/constants/colorPresets.ts` | 5 | 20,21,41,43,50 | 💀 **死** | 删除 |
| 17 | `src/design-system/utils/color.ts` | 4 | 22,55,57,70 | 💀 **死** | 删除 |

> 汇总：**死链 14 文件 / 144 处** ｜ **活代码 3 文件 / 6 处**（`index.css` 2、`DashboardView.css` 1、测试 3）。

#### 1.3.3 死链实证：多主题体系已被 v1.22 Phase 65 废止但文件留存

| 事实 | 证据 |
|---|---|
| `colors.ts` 文件头注释自述 | `src/design-system/tokens/colors.ts:4-6`「v1.22 Phase 65 清理：旧多主题色阶 / 语义色 / 渐变 / 旧品牌色 四组导出已删除…`ruoyiBrand` 为本文件唯一导出常量」 |
| `AntdThemeBridge.tsx` 注释自述 | `:7-9`「v1.22 Phase 65：单一品牌主题，主色优先级链已移除 —— 不再订阅用户自定义颜色与主题 store（**多主题能力随 D-01 一并删除**）」 |
| `ColorSwitcher.tsx` / `ThemeSwitcher.tsx` **无任何活引用** | 全 `src/**/*.ts(x)` 搜索 `ColorSwitcher` / `ThemeSwitcher` / `themePresets` ⇒ **零活命中**（仅 `colorPresets.ts` 自身注释 + `store/themeStore.ts:56` 注释提到 `ThemeSwitcher.tsx`） |
| `src/design-system/themes/index.ts` 只被死组件引用 | `:10-14` 导入 `./minimal` `./glassmorphism` `./neumorphism` `./flat2.0` `./luxury-quiet`；反向引用者只有 `ColorSwitcher.tsx:19` 与 `ThemeSwitcher.tsx:11` |
| 结论 | **`src/design-system/themes/**`（13 文件）+ `ColorSwitcher.tsx` + `ThemeSwitcher.tsx` + `colorPresets.ts` + `utils/color.ts` 构成一条完整死链**，与 Phase 88 删掉的 8 个死 CSS 文件（`D16/D17`）**同族** |

> ⚠️ **连带核查**：`src/design-system/utils/color.ts` 与 `src/types/theme` 是否只被这条死链引用？**必须逐个 grep 确认后再删**，
> 删除前不得遗漏 `utils/color.ts` 的其它消费者（`themes/index.ts:20` 明确引用它）。
> ⚠️ **`utils/color.ts` 的 `getLuminance` 被 `ColorSwitcher.tsx:20` 使用** —— 同链，一并可删。
> ⚠️ 删除死链属 **STYLE-04 的「让门禁能 exit 0」的前置**，与 Phase 88 的 D16/D17  precedent 一致（低风险、命中数 0 的死代码）。
> ⚠️ **C8 死文件断言只扫 `.css`/`.less`** ⇒ 删 TS 死链**不会**触发 C8；但 `knip`（`npm run deadcode`）可能报出，可作为佐证。

#### 1.3.4 `index.css` 与全 `.css` 的 hex 实测（88-03 后基线）

| 指标 | 口径 | PITFALLS（88-03 前） | **本 phase 实测（88-03 后）** | Phase 89 建议 ratchet |
|---|---|---|---|---|
| **M7** | `index.css` 6 位 hex 出现次数 | 311（去重 66） | **304**（**去重 64**） | **≤ 300**（T6/T7/T8/T14 各 -1 ⇒ 300） |
| **M8** | 全 `.css`/`.less` 6 位 hex | 465 | **446** | **≤ 430**（T6/T7/T8/T14 -4、T13 -6、T9–T12 -6） |
| M9 | `index.css` 行数 | 5632 | **5548** | 不增（±10 内） |

> **去重口径**：`(matches | Sort-Object -Unique).Count` = **64**（M7 的「去重 ≤20」终态目标仍是 Phase 92 的事）。

#### 1.3.5 `eslint.config.js` hex 禁令块是否扩到 `.css`

| 事实 | 位置 |
|---|---|
| hex 禁令 selector | `eslint.config.js:161` `selector: "Literal[value=/#[0-9a-fA-F]{3,8}\\b/]"` |
| 生效范围 | `eslint.config.js:132` `files: ["src/**/*.{ts,tsx}"]` —— **不含 `.css`** |
| 豁免层 | `:201+` 的「Phase 68 hex 豁免 override」块，`files` 含 `src/design-system/**` 等结构性目录；**豁免清单在 `scripts/exemption-reasons.json`（`layer` 字段目前只有 `P65`/`P66`）** |

**结论与建议**：

| 方案 | 评价 |
|---|---|
| **扩 `files` 到 `.css`** | ❌ **不推荐**。① ESLint 的 `no-restricted-syntax` 走 **JS/TS AST**，**不解析 `.css`** ⇒ 扩了也**不会生效**（静默无效，反而制造「已覆盖」的错觉）；② 即使生效，`Literal` selector 也匹配不到 CSS 文本；③ 真正的 CSS 门禁应是**独立脚本**（C13）。 |
| **独立 C13 静态断言（推荐）** | ✅ 在 `scripts/css-contract-check.js` 或新脚本 `scripts/css-hex-check.mjs` 里做**全量 `#rrggbb` 文本检测** + `exemption-reasons.json` 的 `layer:"CSS"` 豁免机制。 |
| **职责边界** | `check-hardcoded-colors.mjs` = **lint 门禁**（31 条 denylist，防「已知坏色回潮」，可 `--fix`）｜**C13** = **静态断言**（全量 hex 计数 + **ratchet 预算**，防「新增硬编码色」）。两者**不重复**：C13 管**总量与新增**，扫描器管**已知坏色的具体替换**。 |

---

### 1.4 T1–T14 逐条复核（88-03 后行号 + 建议值是否仍成立）

| # | PITFALLS 原行号 | **当前行号** | 现状 | 建议替换 | **建议是否仍成立** | 89 处置 |
|---|---|---|---|---|---|---|
| **T1** | `index.css:877` | **876** | `border-radius: var(--theme-radius-md) !important`（6px） | `var(--theme-radius-lg)`（8px） | ✅ 成立，但**属 Phase 90 / STYLE-03** | ❌ **移交 90**（X-3） |
| **T2** | `index.css:912` | **908** | 同 T1（wrapper） | 同 T1 | ✅ 同上 | ❌ **移交 90** |
| **T3** | `index.css:5563` | **5482** | `border-radius: 6px !important`（分页器） | `var(--theme-radius-lg)` | ✅ 成立，同属 STYLE-03 | ❌ **移交 90**（另注：`4449/4461/4473/4485/4497/5227` 另有 6 处 `6px`，需一并规划） |
| **T4** | `index.css:878` | **877** | `font-size: 14px !important` | `inherit` | ✅ 成立（antd 内层已 `inherit`） | ⚠️ **可做**，但会**改变字号渲染**（`size="large"` 放大）⇒ 属 Phase 90 的 variant 交还范围；**89 建议不做** |
| **T5** | `index.css:879` | **878** | `transition: all var(--theme-transition-fast) !important` | antd motion token | ✅ 成立 | ⚠️ **可做**（`transition: all` → 具体属性列表，视觉无差异）⇒ 可进 89 的「零差异 T 批次」 |
| **T6** | `index.css:964` | **960** | `border-color: var(--ant-color-error, #ff4d4f) !important` | `var(--ant-color-error)` | ✅ 成立，**fallback 是死值**（antd cssVar 注入 `--ant-color-error:#ff4d4f`） | ✅ **89 做**（零渲染差异 + 消灭 hex 门禁 1 处） |
| **T7** | `index.css:970` | **966** | `border-color: var(--ant-color-warning, #faad14) !important` | `var(--ant-color-warning)` | ✅ 同 T6 | ✅ **89 做** |
| **T8** | `index.css:5577` | **5496** | `color: #ffffff !important`（分页激活页码） | `var(--theme-text-inverse)` | ✅ **成立且值相同**（`:root`@78 = `#ffffff`，暗色块**未重定义**）⇒ 零渲染差异 | ✅ **89 做** |
| **T9** | `login.css:345` | **345** | `border: 1px solid #e7e5e4 !important` | `var(--login-input-border)` | ✅ 成立（**值不变**） | ✅ **89 做**（需在 `login.css` 顶部定义 4 个局部变量） |
| **T10** | `login.css:346` | **346** | `background: #fafaf9 !important` | `var(--login-input-bg)` | ✅ 成立 | ✅ **89 做** |
| **T11** | `login.css:361, 367` | **361, 367** | `border-color: #d4a574 !important` | `var(--login-accent)` | ✅ 成立 | ✅ **89 做** |
| **T12** | `login.css:362, 368` | **362, 368** | `background-color:#ffffff` / `rgba(212,165,116,0.15)` | `var(--login-input-bg)` / `var(--login-focus-ring)` | ✅ 成立 | ✅ **89 做** |
| **T13** | `ActionButtons.css:14,18,23,28,32,40` | **14, 18, 23, 27, 32, 40**（+ 62） | `#262626` / `#000000` / `#ba3630` / `#cf1322` | `var(--theme-text-primary)` / `var(--theme-text-inverse)` / `var(--theme-error)` / `var(--theme-error-hover)` | ❌ **建议值不成立**：`--theme-text-primary` 浅色 = `#101010` ≠ `#262626` ⇒ **浅色模式会变色**；且 `--theme-error` / `--theme-error-hover` **项目未定义** | ⚠️ **89 只做「提取为局部变量保值」**（定义 `--action-text:#262626` / `--action-text-hover:#000000` / `--action-danger:#ba3630` / `--action-danger-hover:#cf1322`）⇒ 值不变；或 **整体移交 92** |
| **T14** | `index.css:4814` | **4733** | `--theme-zebra: #faf8f3`（后置 `:root`） | 并入 `--table-header-bg` / `--theme-bg-tertiary` | ❌ **建议值不成立**：light `--theme-bg-tertiary=#e9efeb`、dark `#1f3524`，**都与 `#faf8f3` 不同** ⇒ 并入会**改色**（并让暗色斑马纹变绿灰） | ⚠️ **89 只做「迁出后置 `:root`」**：把 `--theme-zebra` 移入 `[data-color-mode="light"]` 块并补暗色值（或直接删掉该变量、让 1021/5118 改用 `--bg-surface` / `--theme-bg-tertiary`）—— **但 5118 `.xr-table-zebra` 暗色下无覆盖，现状是近白斑马纹（既有缺陷）**，改它属**新增变更** ⇒ **建议 89 只迁移变量位置、值不变**（暗色新增同名变量 = 登记变更） |

**⇒ 89 的 T 批次收敛结论**：
- **做（零渲染差异）**：T5、T6、T7、T8、T9–T12、T13（提取局部变量保值版）、T14（仅迁位置）
- **不做（移交 Phase 90 / STYLE-03）**：T1、T2、T3、T4
- **T13/T14 需按本表修正后的口径执行**（PITFALLS 原建议值会改色）

---

### 1.5 视觉验收基线复核（88 的矩阵基建）

| 项 | 实况 |
|---|---|
| fixture 脚本 | `.planning/phases/88-style-foundation/88-visual-evidence/matrix-fixture.js`（74 行，21 形态，注入 `.ant-app` / `[class*="css-var-"]` 作用域） |
| 采样脚本 | `matrix-sample.js`（38 行，逐格读 `bg`/`border`/`radius`/`shadow`/`btw` 5 项） |
| diff 脚本 | `matrix-diff.cjs`（24 行，`before-{light,dark}.json` × `after-{light,dark}.json`，逐格比对，输出 `matrix-diff-result.txt`） |
| 规模 | 21 形态 × 明暗 × carrier/inner × 5 属性 = **360 格**（88-03/88-04 实测均为「差异 0」） |
| 快照 | `before-light.json` / `before-dark.json` / `after-light.json` / `after-dark.json`（**88 末态基线，89 直接复用为 before**） |
| 已知陷阱 | 88-03 教训：fixture 若手写旧版 antd DOM（`.ant-select-selector`）会**误中已删死规则**产生 40 格伪差 ⇒ 89 的 fixture 必须用 antd 6 真实 DOM（`.ant-select-content` / `.ant-select-input` / `.ant-select-placeholder`） |

#### 1.5.1 89 的「2 处有意变更」在矩阵中的落点（**按 21 形态逐格推定**）

> **登记方式建议**：**不按「格子数」登记，而按「变更类别 + (形态, part, 属性) 枚举」登记**，
> 判定规则 = *「diff 结果中每一行差异都必须能归入 A/B 之一；无法归类 = FAIL」*。
> 理由：89 的两处变更都会**跨形态扩散**（Input 族 15 个形态、Select 族 5 个形态），按格子数登记会失真。

| 类别 | 触发源 | 推定影响的形态 | 推定影响的属性 | 暗色新值 |
|---|---|---|---|---|
| **A · 暗色输入类底色** | `--input-bg` 串色修正（`#fafaf9` → `#0f2e1b`） | f01–f11（Input/TextArea/InputNumber/affix/group/search）、f14/f15（picker 内层 input）、f20/f21（AppInput/AppTextArea） | `carrier.bg`（f01/f05/f06/f11/f14/f15 另有 `inner.bg`） | `rgb(15,46,27)` |
| **B · 暗色容器底色** | `colorBgContainer` 分色板（`#FFFFFF` → 暗 surface） | f12/f13/f16/f17/f18（Select 族，`.ant-select` 根 = `--ant-select-background-color`）；**f19（Mentions textarea）** —— antd `Mentions` 底色取自 `colorBgContainer`；**f14/f15 的 `.ant-picker` 根底色取自 `token.colorBgContainer`**（antd `date-picker/style/index.js:121/129/181/413` 的 `background:'transparent'` 落在内层，根仍由 `colorBgContainer` 决定 —— **需实机复核**） | `carrier.bg` | `rgb(26,46,31)`（= `#1a2e1f`） |
| **不变（须实测确认）** | — | f09/f10（TextArea 内层被 VIS-01 复位为 `transparent`）、`inner.*`（affix 内层复位） | — | — |

> ⚠️ **须实机复核的两点**：
> ① `.ant-picker` 根底色在 antd 6.6.4 到底取自 `colorBgContainer` 还是 `transparent`（源码 `date-picker/style/index.js` 有 4 处 `background:'transparent'`，但它们落在**内层**）；
> ② `.ant-mentions` textarea 底色来源。
> ⇒ **89 的 diff 脚本必须输出「未归类差异」清单**，由执行者把 B 类补全到实际形态清单后再签字。

---

## 2. 实现方案建议

### 2.1 新增 `ruoyiBrand.dark` 常量组 —— 建议色值表（可直接落码）

**推导依据（四条，全部可复核）**：

1. **与项目现有暗色板同源**：`index.css:198-268` 的 `[data-color-mode="dark"]` 块已定义完整的暗色层级
   （`--bg-base`/`--bg-surface`/`--bg-elevated`/`--bg-overlay`/`--bg-overlay-hover`/`--bg-overlay-active`、
   `--border-base`/`--border-subtle`/`--border-strong`/`--border-focus`、
   `--text-primary`/`--text-secondary`/`--text-tertiary`/`--text-disabled`）。
   ⇒ **`ruoyiBrand.dark` 的作用是把「已经在 CSS 里、但 TS 侧看不见」的这套暗色板**提升为**双真相源的单一来源**，
   供 `AntdThemeBridge` 消费。**值一字不改** ⇒ 视觉零漂移。
2. **品牌绿家族**：`ruoyiBrand.green[900..50]` + `onDark.*` 已在 `colors.ts` 中，用于侧边栏与深底前景。
3. **深绿页面底实测值**：`rgb(15,46,27)` = `#0f2e1b`（v1.24 实测，与 `--bg-base` 一致）。
4. **WCAG 校验**：主/次/三级文字对 surface 均 ≥ 4.5:1（§2.1.2 实算）。

#### 2.1.1 建议色值表（**落码版**）

```ts
// ruoyi-react-frontend/src/design-system/tokens/colors.ts —— 在 ruoyiBrand 内新增：
  /**
   * 暗色中性色板（v1.25 Phase 89 · STYLE-01）
   * 来源：index.css `[data-color-mode="dark"]` 块（:198-268）逐值对齐 —— **值不变**，仅提升为 TS 侧真相源。
   * 推导：深绿低饱和纸感；层级 = canvas < surface < elevated < overlay < hover < active。
   * 对比度（WCAG）：fg 12.25:1 / fgMuted 9.94:1 / fgSubtle 7.71:1 / fgDisabled 2.95:1（占位符，见 §2.3.3 预警）。
   */
  dark: {
    /** 页面画布底 — 等价 --bg-base（实测 rgb(15,46,27)）→ antd colorBgBase / colorBgLayout */
    canvas: "#0f2e1b",
    /** 容器面（卡片/输入框/Select 底）— 等价 --bg-surface → antd colorBgContainer */
    surface: "#1a2e1f",
    /** 悬浮层（弹层/抽屉）— 等价 --bg-elevated → antd colorBgElevated */
    elevated: "#1f3524",
    /** 覆盖层（下拉面板）— 等价 --bg-overlay */
    overlay: "#243a29",
    /** 悬停面 — 等价 --bg-overlay-hover */
    hover: "#2d4d33",
    /** 激活面 — 等价 --bg-overlay-active */
    active: "#345b3a",
    /** 主描边 — 等价 --border-base → antd colorBorder */
    border: "#2d4d33",
    /** 次级描边 / 分割线 — 等价 --border-subtle → antd colorBorderSecondary */
    borderSubtle: "#1a2e1f",
    /** 强调描边 — 等价 --border-strong */
    borderStrong: "#345b3a",
    /** 聚焦描边（品牌绿）— 等价 --border-focus */
    borderFocus: "#1a6839",
    /** 主文字 — 等价 --text-primary（奶油白）→ antd colorTextBase */
    fg: "#f0ece3",
    /** 次级文字 — 等价 --text-secondary */
    fgMuted: "#dbd7ce",
    /** 三级文字 — 等价 --text-tertiary */
    fgSubtle: "#c2bdb2",
    /** 禁用文字 — 等价 --text-disabled */
    fgDisabled: "#707068",
  },
```

#### 2.1.2 对比度实算（WCAG 2.1，脚本复核）

| 前景 / 背景 | 对比度 | 判定 |
|---|---|---|
| `#f0ece3` / `#1a2e1f` | **12.25:1** | ✅ AAA |
| `#f0ece3` / `#0f2e1b` | 12.48:1 | ✅ AAA |
| `#dbd7ce` / `#1a2e1f` | 9.94:1 | ✅ AAA |
| `#c2bdb2` / `#1a2e1f` | 7.71:1 | ✅ AAA |
| `#707068` / `#1a2e1f`（暗色占位符） | **2.95:1** | ❌ **AA 不达标**（现状在 `#fafaf9` 上是 4.81:1 ⇒ **修复后回退**，见 §2.3.3） |
| `#2d4d33` / `#0f2e1b`（暗色描边） | **1.56:1** | ❌ WCAG 1.4.11 3:1 不达标（见 §0 X-2） |
| `#598e5e` / `#0f2e1b` | 3.82:1 | ✅ 3:1（hover/聚焦描边已达标） |
| `#4a7d52` / `#0f2e1b` | 3.05:1 | ✅ 3:1（**若要把常态描边提到 3:1 的最小改动值**） |
| `#fafaf9` / `#0f2e1b`（**现状**暗色输入底） | **14.09:1** | ⚠️ 高对比但「浅底压深底」 |

#### 2.1.3 `ruoyiBrand.dark` → antd token 映射表（**落码即用**）

| antd token | 层 | 浅色值（现状，**不变**） | 暗色值（**新增**） | 备注 |
|---|---|---|---|---|
| `colorBgBase` | **Seed** | `cream.canvas` `#F0ECE3` | `dark.canvas` `#0f2e1b` | ★ 算法入口，必须分模式 |
| `colorTextBase` | **Seed** | `cream.fg` `#101010` | `dark.fg` `#f0ece3` | ★ 同上 |
| `colorBgContainer` | Map | `cream.surface` `#FFFFFF` | `dark.surface` `#1a2e1f` | 显式值生效（§1.1.2 D 组） |
| `colorBgElevated` | Map | `cream.surface` `#FFFFFF` | `dark.elevated` `#1f3524` | |
| `colorBgLayout` | Map | `cream.canvas` `#F0ECE3` | `dark.canvas` `#0f2e1b` | |
| `colorBorder` | Map | `cream.border` `#DBD7CE` | `dark.border` `#2d4d33` | |
| `colorBorderSecondary` | Map | `cream.borderStrong` `#C2BDB2` | `dark.borderSubtle` `#1a2e1f` | |

---

### 2.2 `AntdThemeBridge.tsx` 改造（STYLE-01 主体）

#### 2.2.1 推荐形态：**按模式选板 + 显式给值 + 组件级回落到全局**

```tsx
// ruoyi-react-frontend/src/design-system/components/AntdThemeBridge.tsx
import { ruoyiBrand } from "@/design-system/tokens/colors";

const antdThemeConfig = useMemo<ThemeConfig>(() => {
  const isDark = themeMode === "dark";
  const algorithm = isDark ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm;
  const algorithms = density === "compact" ? [algorithm, antdTheme.compactAlgorithm] : algorithm;

  // ★ Phase 89 · STYLE-01：按模式取板（cream ↔ dark）
  //   注意：本组常量是**模式无关**的 —— 混用即触发 P-M8（暗色白底）。
  const board = isDark ? ruoyiBrand.dark : ruoyiBrand.cream;

  return {
    token: {
      colorPrimary: ruoyiBrand.greenPrimary,
      colorInfo: ruoyiBrand.greenPrimary,
      colorLink: ruoyiBrand.greenPrimary,
      colorSuccess: ruoyiBrand.functional.success,
      colorWarning: ruoyiBrand.functional.warning,
      colorError: ruoyiBrand.functional.danger,

      // —— Seed 层：算法的入口，必须随模式切换 ——
      colorTextBase: board.fg,
      colorBgBase: board.canvas,

      // —— Map 层：antd 的 dark 派生对**饱和品牌绿基底**会产出错误色相
      //    （实跑：colorBorder=#20a252 / colorBgContainer 比页面底更暗），
      //    故此处**显式给值**而非交给算法派生（详见 89-RESEARCH §1.1.2 C/D 组）。
      colorBgContainer: board.surface,
      colorBgElevated: board.elevated,
      colorBgLayout: board.canvas,
      colorBorder: board.border,
      colorBorderSecondary: board.borderSubtle,

      borderRadius: 8,
      borderRadiusLG: 12,
      fontFamily: fontFamily.sans,
    },
    components: {
      Button: {
        colorPrimary: ruoyiBrand.greenPrimary,
        primaryColor: ruoyiBrand.onDark.white,
        colorPrimaryHover: ruoyiBrand.greenPrimaryHover,
        colorPrimaryActive: ruoyiBrand.greenPrimaryActive,
        defaultBorderColor: board.border,          // ★ 暗色下不再是 #DBD7CE
        borderRadius: 8,
        controlHeight: 36,
        fontWeight: 500,
      },
      Table: {
        headerBg: board.headerBg,                  // ⚠️ 见 §2.2.3「本次不改」
        …
      },
      Input: {
        // ★★ 删除 colorBgContainer —— 回落全局 token（见 §2.2.2 方案对比）
        activeBorderColor: ruoyiBrand.greenPrimary,
        hoverBorderColor: ruoyiBrand.greenPrimary,
        activeShadow: "0 0 0 2px rgba(21, 96, 49, 0.15)",
        borderRadius: 8,
        controlHeight: 36,
      },
      Select: {
        // ★★ 删除 colorBgContainer（同上）
        activeBorderColor: ruoyiBrand.greenPrimary,
        hoverBorderColor: ruoyiBrand.greenPrimary,
        borderRadius: 8,
        controlHeight: 36,
        optionSelectedBg: board.headerBg,
      },
      Card: {
        // ★★ 删除 colorBgContainer（同上）
        headerBg: "transparent",
        borderRadiusLG: 12,
      },
      …
    },
    algorithm: algorithms,
    hashed: true,
  };
}, [themeMode, density]);   // ★ deps 保持 [themeMode, density]（C15 单测硬断言 length===2）
```

> ⚠️ **`cream` 组需补 `headerBg` 之外的同义键**：`cream` 已有 `headerBg`（`#E9EFEB`）、`zebraBg`（`#F7F5EE`）、
> `surface`/`canvas`/`fg`/`muted`/`mutedStrong`/`border`/`borderStrong`。
> `dark` 组**必须与 `cream` 组同构**（键名一一对应），否则 `board.headerBg` 在暗色下 `undefined`。
> ⇒ **`dark` 组需补 `headerBg: "#1f3524"`（= `--table-header-bg` 暗色值）、`zebraBg: "#1a2e1f"`**。

#### 2.2.2 A4 专项：`components.Input/Select/Card.colorBgContainer` 的两个方案对比

| 维度 | **方案甲：删掉组件级 `colorBgContainer`（回落全局）** | **方案乙：按模式显式设 `components.*.colorBgContainer`** |
|---|---|---|
| antd 机制 | 组件 token merge 时该键不存在 ⇒ 组件默认 token 取全局 `colorBgContainer`（`getComponentToken.js:23` `{...defaultToken, ...customToken}`） | 组件 token 在算法之后 merge，**模式无关** ⇒ 必须显式分支 |
| 暗色结果 | `#1a2e1f`（= 全局 dark.surface）✅ | `#1a2e1f` ✅（需写两处分支） |
| 浅色结果 | `#FFFFFF` ✅ 与现状一致 | `#FFFFFF` ✅ |
| **与 Phase 90/91/92 的相容性** | ✅ **最佳**：Phase 91 交还 Select 域 token 时，Select 底色**自动跟随全局色板**，不需再写一遍模式分支；Phase 92 删 431 条暗色规则后，暗色外观由**全局 token** 统一提供 | ⚠️ **较差**：Phase 91/92 仍要维护「全局 + 组件」**两处**模式分支，是「半吊子」——正是任务书点名要避免的 |
| 单一真相源 | ✅ 全局 token 是唯一来源 | ❌ 两处来源（易漂移） |
| 改动量 | 删 3 行 | 改 3 行 + 加分支 |
| **推荐** | ✅ **推荐方案甲** | ❌ |

> **「删掉会不会让 Input 变白？」**——不会：`.ant-input` 的**可见底色**由项目 CSS `background: var(--input-bg) !important`
> （`index.css:873/905/2269/…`）决定，**项目 CSS 赢过 antd token**。组件 token 只影响 **affix 内层 / Select / Card** 等
> 没有项目覆盖的地方。
> ⚠️ **唯一例外**：`Card.colorBgContainer` 删掉后回落到全局 `#FFFFFF`（浅） / `#1a2e1f`（暗）——
> 但项目 CSS 有 `.ant-card` 覆盖吗？**需 grep 确认**（`index.css` 有 `.ant-card` 规则，见 §1.1.4 引用），
> 若项目已用 `!important` 覆盖 Card 底色，则 Card token 改动**不可见**（无害）。

#### 2.2.3 本次**不**改的组件 token（登记，避免半吊子）

| 组件 token | 暗色下的浅值 | 本 phase 处置 | 理由 |
|---|---|---|---|
| `Table.headerBg` / `rowHoverBg` / `rowSelectedBg` / `borderColor` / `headerColor` | `#E9EFEB` / `#F7F5EE` / `#DBD7CE` | ❌ **不改** | 项目 CSS 有 `.ant-table` 覆盖（`index.css:972-…`）；改 token 会与手写暗色 cluster 冲突 ⇒ 归 Phase 91/92 |
| `Menu.itemSelectedBg` / `itemHoverBg` / `itemSelectedColor` | `#E9EFEB` / `#F7F5EE` | ❌ **不改** | 同上 |
| `Tag.defaultBg` / `defaultColor` | `#FEF3C7` / `#C09058` | ❌ **不改** | 暗色下淡黄标签是**设计意图**（onDark 家族） |
| `Select.optionSelectedBg` | `#E9EFEB` | ⚠️ **建议改**（→ `board.headerBg`，暗色 `#1f3524`） | 属 Select 域 → Phase 91；若 89 改，需登记为暗色视觉变更 |
| `Button.defaultBorderColor` | `#DBD7CE` | ⚠️ **建议改**（→ `board.border`，暗色 `#2d4d33`） | 与 `colorBorder` 同源；**边界清晰、影响面小**（按钮描边） |

> **原则**：89 只改**与 `colorBgContainer` / `colorBorder` 同源的容器与描边类 token**，
> 其余（Table/Menu/Tag 域）**整体移交 Phase 91/92**，避免在 89 造成无法登记的暗色视觉变更。

---

### 2.3 变量层改造（B 块：VISFIX-01 + R11）

#### 2.3.1 R10：`--input-bg` 串色（**推荐方案 ②**）

```diff
  /* index.css :150  [data-color-mode="light"] 块内新增 */
+ --input-bg: #fafaf9;            /* 输入框底（原型实测，迁自后置 :root） */
+ --input-bg-hover: #f4f2ec;      /* 结构对称补齐，当前无引用点 → 零渲染影响 */
+ --input-bg-focus: #fafaf9;
+ --input-bg-disabled: #eceae4;

  /* index.css :4731  后置 :root 块 */
  :root {
    --theme-font-display: Georgia, "Songti SC", "Source Han Serif SC", serif;
-   --theme-zebra: #faf8f3;        /* 迁至 light/dark 双分区（见 §1.4 T14） */
-   --input-bg: #fafaf9;           /* ★ 迁至 [data-color-mode="light"] @150 */
    --sidebar-gradient: linear-gradient(168deg, #14532d 0%, #156031 58%, #1a6839 100%);
  }
```

#### 2.3.2 R11：placeholder 变量统一

```diff
  /* index.css :27  :root 兜底 */
  :root {
+   --input-text-placeholder: #707068;   /* 兜底：任何作用域都有定义（C6 天然通过） */
    …
  }
  /* index.css :189  light 块 —— 仅改名，值不变 */
-   --input-placeholder: #707068;
+   --input-text-placeholder: #707068;
  /* index.css :267  dark 块 —— 不变 */
    --input-text-placeholder: #707068;
  /* index.css :900  通用 placeholder */
-   color: var(--input-placeholder) !important;
+   color: var(--input-text-placeholder) !important;
  /* index.css :2276  暗色 placeholder（原先整条作废） */
-   color: var(--input-placeholder) !important;
+   color: var(--input-text-placeholder) !important;
  /* index.css :2300-2302  ★ 整条删除 */
- [data-color-mode="dark"] .ant-select-selection-placeholder {
-   color: var(--input-placeholder) !important;
- }
    /* 理由：① 变量改名后已无未定义引用；② `.ant-select-selection-placeholder` 在 antd 6.6.4
       不存在（真实类名 `.ant-select-placeholder`，STACK.md A4.2）⇒ 双重死规则 */
```

**同步**：`scripts/css-contract-check.js:133-140` 的 `EXEMPTED_UNREF_VARS` **3 条全部删除**
（`--input-placeholder`@2276 / @2301 / `--theme-warning-text`@4472）⇒ C6 豁免表清空。

#### 2.3.3 `--theme-warning-text` 定义 + **暗色占位符 AA 预警处置**

```diff
  /* index.css :91 附近（:root，与 --theme-warning-bg 同族） */
+   --theme-warning-text: #905d00;   /* 警告文字（= ruoyiBrand.functional.warningText） */
```
⇒ `4472` 的 `var(--theme-warning-text)` 生效，值 = `#905d00` = `5178` 的原 fallback ⇒ **渲染零差异** ✅

**暗色占位符 AA 预警（决策点）**：

| 选项 | 动作 | 后果 |
|---|---|---|
| **①（推荐）** | 暗色 `--input-text-placeholder` 由 `#707068` → **`#a3a399`**（对 `#0f2e1b` **4.7:1**，AA 达标） | 暗色占位符**变亮** ⇒ **新增第 3 处有意变更**（须登记 + 截图） |
| ② | 保持 `#707068` | 暗色占位符 **2.95:1**，低于 AA 4.5:1 ⇒ **须在 phase 报告显式登记为 AA 欠账并移交 Phase 92**（暗色 mega-cluster 归口） |

> **建议 ①**：它把「修复串色引入的对比度回退」在**同一个 phase 内闭合**，
> 避免把一个**新造出来的** AA 违规留给后续 phase（这正是 P-M9「hex/色值治理引入新债」的同类风险）。
> 代价是「2 处有意变更」变 3 处 —— 需在 PLAN 阶段向用户确认，或由规划者决定改判据。

#### 2.3.4 `--radius`（`:5515`）—— **不动**

`border-radius: var(--radius, 8px)` 的 fallback 生效（8px），定义全局 `--radius` 反而与 `--theme-radius-*` 体系重复。
⇒ **保留 fallback，不处理**（C6 因有 fallback 已通过）。

---

### 2.4 hex 门禁落地（STYLE-04 + C13）

#### 2.4.1 三件套（职责严格分离）

| 件 | 位置 | 职责 | 本 phase 动作 |
|---|---|---|---|
| **① 既有扫描器** | `scripts/check-hardcoded-colors.mjs` | **lint 门禁**：拦「31 条已知坏 hex / 7 条 rgba」回潮，可 `--fix` | **修到 exit 0** + **接入 `lint:ci`** |
| **② C13 静态断言** | `scripts/css-contract-check.js`（新增）**或**新脚本 `scripts/css-hex-check.mjs` | **全量 `#rrggbb` 计数 + ratchet 预算**（含去重数），阻止「新增硬编码色」 | 新增 |
| **③ 豁免机制** | `scripts/exemption-reasons.json` | 新增 `layer: "CSS"` 维度 | 新增 + 扩展 `check-exemption-reasons.mjs` 的校验维度 |

#### 2.4.2 让扫描器 exit 0 的**完整动作清单**（按 §1.3.2）

| 动作 | 文件 | 命中 | 手法 | 渲染影响 |
|---|---|---|---|---|
| **A1** | `src/index.css:960, 966` | 2 | T6 / T7：去掉死 fallback | 无 |
| **A2** | `src/components/dashboard/DashboardView.css:50` | 1 | 去掉死 fallback `var(--theme-text-tertiary, #8c8c8c)` → `var(--theme-text-tertiary)`（该变量两模式都有定义） | 无 |
| **A3** | `src/components/dashboard/widgets/types/__tests__/statAndProgress.test.tsx:41,54,67` | 3 | **加进 `PATH_ALLOW`** + 在 `exemption-reasons.json` 写明理由（图表数据色板，非 UI 主题色；改值即改测试语义） | 无（测试文件） |
| **A4** | **死链 14 文件**（`themes/**` 13 + `ColorSwitcher.tsx` + `colorPresets.ts` + `utils/color.ts`，见 §1.3.3） | 144 | **删除整条死链**（Phase 88 D16/D17 同 precedent）—— 删除前逐个 grep 确认无活引用 | 无（死代码） |
| — | 合计 | **150** | | |

> ⚠️ **A4 的前置核查清单**（删除前必做）：
> ① `grep -rn "design-system/themes" src` ⇒ 只剩 `ColorSwitcher.tsx` / `ThemeSwitcher.tsx`；
> ② `grep -rn "ColorSwitcher\|ThemeSwitcher\|themePresets" src` ⇒ 无活引用；
> ③ `grep -rn "utils/color" src` ⇒ 确认 `utils/color.ts` 无链外消费者；
> ④ `grep -rn "@/types/theme" src` ⇒ 确认 `ThemeType`/`ColorMode` 是否只服务死链（**若被活代码用，只删 themes 目录，保留 types**）；
> ⑤ `npm run type-check` 全绿（删除后无残留 import）；
> ⑥ `npm run test` 全绿。

#### 2.4.3 C13 断言设计（**与扫描器不重复**）

**建议放在 `scripts/css-contract-check.js`（与 C6/C8/C11/C12 同族）**，形态：

```js
// ---------- C13 .css hex 全量计数 + ratchet（阻断） ----------
// 口径：
//   1) 扫描 src 下全部 .css / .less，剥离注释后统计 6 位 hex（#rrggbb）与 3 位 hex（#rgb）
//   2) 计数分「总次数」与「去重值数」两个口径，与 ROADMAP M7 / M8 对齐
//   3) ratchet：只许不增（与 C11 !important 同纪律）
//   4) 豁免：exemption-reasons.json 的 layer:"CSS" 条目按「文件 + 行号」逐条放行
// 注意（职责边界）：本断言只管「总量与新增」，不管「某个已知 hex 是不是坏色」——
//       那是 scripts/check-hardcoded-colors.mjs 的 lint 职责。两者互补，不重复。
```

**建议 ratchet 基线（Phase 89 落地值）**：

| 指标 | 89 起点（实跑） | 89 建议 ratchet | 终态目标（Phase 92） |
|---|---|---|---|
| `index.css` 6 位 hex 总数 | **304** | **≤ 300** | ≤ 60（去重 ≤ 20） |
| `index.css` 去重 hex | **64** | **≤ 60** | ≤ 20 |
| 全 `.css`/`.less` 6 位 hex 总数 | **446** | **≤ 430** | ≤ 120 |

#### 2.4.4 `exemption-reasons.json` 的 `layer:"CSS"` 扩展位

现有结构（`scripts/exemption-reasons.json`）：`{ version, updated, description, entries: [{ file, layer, dimension, reason, userVisible, difference }] }`，
`layer` 目前只有 `P65` / `P66`。

**扩展设计**（建议）：

```json
{
  "file": "src/components/dashboard/widgets/types/__tests__/statAndProgress.test.tsx",
  "layer": "CSS",
  "dimension": "hex:chart-palette",
  "reason": "ECharts 数据可视化色板（#1890ff = 图表系列色），非 UI 主题色；改值即改测试断言语义",
  "userVisible": false,
  "difference": "无 —— 仅测试夹具的期望色板值"
}
```

⚠️ **配套**：`scripts/check-exemption-reasons.mjs`（122 行）当前只校验 P65/P66 两个锚点数组 ——
新增 `layer:"CSS"` 后必须**同步扩展该脚本**（校验 CSS 豁免条目与 C13 实际放行集合一一对应），
否则会出现「登记表有、C13 没放行」或反之的漂移。

---

## 3. Validation Architecture

### 3.1 分层验证矩阵

| 层 | 工具 | 断言内容 | 判据 | 落地位置 |
|---|---|---|---|---|
| **L1 静态契约** | `node scripts/css-contract-check.js` | C1–C14 全 PASS（现有）+ **C13**（hex ratchet）新增 | `exit 0` | `lint:ci` 链尾 |
| **L2 静态契约** | `node scripts/check-exemption-reasons.mjs` | 豁免表 ↔ 锚点一一对应（含新增 `layer:"CSS"`） | `exit 0` | `lint:ci` |
| **L3 lint 门禁** | `node scripts/check-hardcoded-colors.mjs` | 31 条 denylist + 7 条 rgba 零命中 | `exit 0` **且接入 `lint:ci`** | `lint:ci` 新增一项 |
| **L4 lint 预算** | `node scripts/assert-lint-budget.mjs` | eslint 预算 | `exit 0` | 已在 `lint:ci` |
| **L5 单测（TS）** | `npx vitest run src/design-system/components/__tests__/AntdThemeBridge.test.tsx` | 见 §3.2 | 全绿 | 新增用例 |
| **L6 单测（全量）** | `npm run test` | 既有 3960 用例零回归 | 全绿 | — |
| **L7 类型** | `npm run type-check` | 删死链后无残留 import | 0 error | — |
| **L8 浏览器 computed-style** | 浏览器 Console 粘贴 `css-contract-assertions.js` + **新增 89 探针** | 见 §3.3 | 见 §3.3 | — |
| **L9 视觉矩阵** | `matrix-fixture.js` + `matrix-sample.js` + `matrix-diff.cjs` | 见 §3.4 | **白名单外差异 = 0** | 88-visual-evidence 复用 |
| **L10 页面巡检** | 14 页截图（88 的 `p01`–`p14` + `r2-*`） | 逐页比对 | 无未登记差异 | 89-visual-evidence |

### 3.2 新增单测（`AntdThemeBridge.test.tsx` 已在，88-02 建）

**沿用现有 mock 结构**（`prefsState` + `vi.mock("@/store/settingsStore")` + `antdTheme.useToken()` 探针）。

```tsx
// 追加到 src/design-system/components/__tests__/AntdThemeBridge.test.tsx
describe("Phase 89 · STYLE-01 明暗色板", () => {
  const tokenOf = () => seenTokens[seenTokens.length - 1] as Record<string, any>;

  it("dark 模式：全局 colorBgContainer 不再是 #FFFFFF", () => {
    prefsState.mode = "dark";
    const { rerender } = mountBridge();
    rerender(<AntdThemeBridge><TokenProbe /></AntdThemeBridge>);
    expect(tokenOf().colorBgContainer).not.toBe("#FFFFFF");
    expect(tokenOf().colorBgContainer.toLowerCase()).toBe("#1a2e1f");
  });

  it("dark 模式：colorBorder 不再是浅色 #DBD7CE", () => {
    prefsState.mode = "dark";
    const { rerender } = mountBridge();
    rerender(<AntdThemeBridge><TokenProbe /></AntdThemeBridge>);
    expect(tokenOf().colorBorder.toLowerCase()).not.toBe("#dbd7ce");
  });

  it("dark 模式：components.Select.colorBgContainer 不再为 #FFFFFF（VISFIX-02）", () => {
    prefsState.mode = "dark";
    const { rerender } = mountBridge();
    rerender(<AntdThemeBridge><TokenProbe /></AntdThemeBridge>);
    expect(tokenOf().Select.colorBgContainer?.toLowerCase()).not.toBe("#ffffff");
  });

  it("dark 模式：components.Input / Card.colorBgContainer 不再为 #FFFFFF（STYLE-01 层①）", () => {
    prefsState.mode = "dark";
    const { rerender } = mountBridge();
    rerender(<AntdThemeBridge><TokenProbe /></AntdThemeBridge>);
    const t = tokenOf();
    expect(t.Input.colorBgContainer?.toLowerCase()).not.toBe("#ffffff");
    expect(t.Card.colorBgContainer?.toLowerCase()).not.toBe("#ffffff");
  });

  it("light 模式：色板不变（cream.surface = #FFFFFF），回归守护", () => {
    prefsState.mode = "light";
    const { rerender } = mountBridge();
    rerender(<AntdThemeBridge><TokenProbe /></AntdThemeBridge>);
    expect(tokenOf().colorBgContainer.toLowerCase()).toBe("#ffffff");
    expect(tokenOf().colorBorder.toLowerCase()).toBe("#dbd7ce");
  });

  it("seed 层 colorBgBase 随模式切换（算法入口）", () => {
    // light → #f0ece3 ; dark → #0f2e1b
  });
});
```

> ⚠️ **实测提示**：组件级 token 删掉 `colorBgContainer` 后，`token.Input.colorBgContainer` 的实际值由 antd 的
> `prepareComponentToken` 回落决定，**可能为 `undefined`**（若组件默认 token 未显式带该键）。
> ⇒ 断言应写成 **`not.toBe("#ffffff")`**（容忍 `undefined`），**不要**写成 `toBe("#1a2e1f")`。
> 这一点**必须先跑一次探针确认**（L5 首轮）。

### 3.3 浏览器 computed-style 断言（新增 89 探针）

```js
// 89 探针（浏览器 Console 粘贴；明暗各跑一次）
(function () {
  const mode = document.documentElement.getAttribute('data-color-mode');
  const root = getComputedStyle(document.documentElement);
  const pick = (sel) => {
    const el = document.querySelector(sel);
    return el ? getComputedStyle(el) : null;
  };
  const inp  = pick('.ant-input');
  const sel_ = pick('.ant-select');
  const card = pick('.ant-card');
  return JSON.stringify({
    mode,
    // VISFIX-01：暗色 --input-bg 必须 = #0f2e1b
    inputBgVar: root.getPropertyValue('--input-bg').trim(),
    // R11：placeholder 变量必须非空
    placeholderVar: root.getPropertyValue('--input-text-placeholder').trim(),
    // VISFIX-01：输入框底 vs 页面底对比度
    inputBg: inp && inp.backgroundColor,
    pageBg: getComputedStyle(document.body).backgroundColor,
    // VISFIX-02：暗色 Select 底色 ≠ 白
    selectBg: sel_ && sel_.backgroundColor,
    cardBg: card && card.backgroundColor,
    // antd cssVar 侧（真实生效点）
    antBgContainer: root.getPropertyValue('--ant-color-bg-container').trim(),
    antBorder: root.getPropertyValue('--ant-color-border').trim(),
  }, null, 2);
})();
```

**判据**：

| 模式 | `--input-bg` | `--input-text-placeholder` | `.ant-select` 底 | `--ant-color-bg-container` |
|---|---|---|---|---|
| light | `#fafaf9` | `#707068` | `rgb(255,255,255)` | `#FFFFFF` |
| **dark** | **`#0f2e1b`** | **非空** | **`≠ rgb(255,255,255)`** | **`≠ #FFFFFF`** |

### 3.4 视觉矩阵的「2 处有意变更」登记与判定

**登记物**：`89-visual-evidence/89-CHANGESET.md`（新增），逐条登记：

```
| # | 类别 | 形态 | part | 属性 | 暗色 before → after | 触发源 | 依据 |
|---|------|------|------|------|-------------------|--------|------|
| A | 暗色输入类底色 | f01..f11,f14,f15,f20,f21 | carrier(+inner) | bg | rgb(250,250,249) → rgb(15,46,27) | --input-bg 串色修正 | index.css:4734 迁移 |
| B | 暗色容器底色 | f12,f13,f16,f17,f18(+f19/f14f15 待实测) | carrier | bg | rgb(255,255,255) → rgb(26,46,31) | colorBgContainer 分色板 | AntdThemeBridge token |
```

**判定规则（建议写进 `matrix-diff.cjs` 的 89 分支）**：

```
1. 逐格 diff（沿用 88 的 5 属性 × carrier/inner 口径）；
2. 把每条差异映射到「变更类别」（A/B/C…）；
3. ★ 断言：
   - light 模式差异数 = 0（浅色必须零漂移）；
   - dark 模式差异数 > 0（证明变更真实生效，防恒等空跑）；
   - ★ 每条差异都必须能归类，**未归类差异 = FAIL**；
   - 归类后，差异的「形态集合」必须与 89-CHANGESET.md 登记的形态集合**完全一致**（不多不少）；
4. 自检：暗色内部 light↔dark 的差异格数 > 0（沿用 88 的「防恒等空跑」自检）。
```

> ⚠️ **与 88 的关键差别**：88 的判据是「差异 = 0」；89 的判据是「**差异 ⊆ 已登记集合**」。
> ⇒ **`matrix-diff.cjs` 必须改造为支持「白名单 + 归类」模式**（这是 89 的第一个可交付物，建议单列为 Task）。

### 3.5 量化快照（M1–M16）在 89 的预期值

| 指标 | 口径 | 88 末态（88-04 实跑） | **89 起点（本 phase 实测）** | **89 预期** | 说明 |
|---|---|---|---|---|---|
| **M1** | 全 `.css`/`.less` `!important` | 1398 | **1398** | **≤ 1398**（ratchet 不上调；若 T5 落地则 1396，可下调 ratchet） | C11 ratchet 现状 1398 |
| **M2** | `index.css` `!important` | 1331 | **1331** | ≤ 1331（T5 落地后 1330） | |
| **M3** | 含 `.ant-` 顶层规则（全 `.css`） | 未清算 | **725**（`index.css` 702） | ≤ 725 | 89 不删 `.ant-` 规则 |
| **M4** | 暗色顶层规则 | 569（grep 口径）/ 502（解析口径） | **569 / 502** | 不增 | |
| **M5** | 暗色分区 | 64 | ≈64（口径依脚本） | 不增 | |
| **M6** | 重复暗色 class | 80 | ≈80 | 不增 | |
| **M7** | `index.css` 6 位 hex | — | **304（去重 64）** | **≤ 300（去重 ≤ 60）** | T6/T7/T8/T14 各 -1 |
| **M8** | 全 `.css` 6 位 hex | — | **446** | **≤ 430** | 再减 T13(6) + T9–T12(6) |
| **M9** | `index.css` 行数 | 5548 | **5548** | ≤ 5558 | 迁变量会小幅增减 |
| **M10** | `index.css` 分区块 | 97 | ≈97 | 不增 | |
| **M11** | 暗色 `--input-bg` | `#fafaf9`（错） | `#fafaf9` | **`#0f2e1b`** ✅ | VISFIX-01 核心 |
| **M12** | 暗色 placeholder 变量 | 未定义 | 未定义 | **非空** ✅ | R11 |
| **M13** | InputNumber 错误态边框 | `rgb(219,215,206)` | 同 | **不变**（属 Phase 90） | 89 不动 |
| **M14** | 死样式文件 | 0 | **0** | **0** ✅ | 删死链是 `.ts`，不触发 C8 |
| **M15** | 圆角 | 6px/8px 混合 | 同 | **不变**（属 Phase 90） | X-3 |
| **M16** | antd 弃用警告 | 0 | 0 | **0** ✅ | 89 不动 antd 用法 |

---

## 4. 波次建议

> 总原则：**「变量层先、色板后」**（ROADMAP 成功标准 #4 明确要求），且 **STYLE-01 必须先于 Phase 90/91/92 的变体交还**。

| 波 | 内容 | 依赖 | 闸门（必须全绿才进下一波） | 可并行 |
|---|---|---|---|---|
| **W0 · 口径与基线冻结**（零改动） | ① 落 `89-QUANT-SNAPSHOT.md`（起点基线）；② 补 `scripts/css-analyze.mjs`（PITFALLS §3.2 的聚类脚本正式化）；③ 复制 88 的 `after-{light,dark}.json` 为 89 的 `before-*` | — | `node scripts/css-contract-check.js` exit 0（现状绿） | 与 W1 无依赖，可与 W1 并行 |
| **W1 · 变量层（B 块）** | R10（`--input-bg` 迁 light 块）+ R11（placeholder 改名 + `:root` 兜底 + 删 2301 死规则）+ `--theme-warning-text` 定义 + `--input-bg-*` 四变量双分区对称 + **删除 C6 豁免 3 条** | W0 | ① `css-contract-check.js` exit 0（C6 豁免清空仍绿）② 浏览器 `--input-bg` = `#0f2e1b`、placeholder 变量非空 ③ **矩阵：dark 差异全部归类为 A；light 差异 = 0** ④ 14 页巡检无破相 | W1 可与 W2 并行（文件面：`index.css` 变量区 vs `colors.ts`/`AntdThemeBridge.tsx`；**但两者都会改 `index.css` 的不同区段**，建议串行以避免冲突） |
| **W2 · 色板重构（A 块 · STYLE-01 主体）** | ① `colors.ts` 新增 `ruoyiBrand.dark`（含 `headerBg`/`zebraBg` 同构键）② `AntdThemeBridge.tsx` 按模式选板 + 删 3 处组件级 `colorBgContainer` + `Button.defaultBorderColor` 分模式 ③ 同步单测 | W0（**可与 W1 并行**，但**必须先于 W3**） | ① 新增单测全绿（§3.2）② 浏览器 `.ant-select` 底 ≠ 白、`--ant-color-bg-container` ≠ `#FFFFFF`（dark）③ **矩阵：dark 差异归类为 A+B，light = 0** ④ C15 仍绿（deps.length === 2） | 可与 W1 并行（不同文件） |
| **W3 · hex 门禁（STYLE-04 + C13）** | ① 删死链 14 文件 ② A1/A2/A3 三处处置 ③ **新增 C13 断言 + `exemption-reasons.json` 的 `layer:"CSS"` + `check-exemption-reasons.mjs` 扩展** ④ `check-hardcoded-colors.mjs` **接入 `lint:ci`** ⑤ T 批次（**仅 T5/T6/T7/T8/T9–T12/T13-提取变量/T14-迁位置**，**不含 T1/T2/T3/T4**） | W0 | ① `node scripts/check-hardcoded-colors.mjs` **exit 0** ② `npm run lint:ci` exit 0 ③ C13 ratchet 达标（M7 ≤ 300 / M8 ≤ 430）④ `npm run type-check` + `npm run test` 全绿 ⑤ **矩阵：T 批次必须是零渲染差异**（T13/T14 若改值 → 必须登记为变更） | 可与 W1/W2 并行（`login.css` / `ActionButtons.css` / `index.css` 尾段） |
| **W4 · 收口** | ① 落 `89-QUANT-SNAPSHOT.md` 终值 ② 落 `89-CHANGESET.md` 最终归类表 ③ 14 页巡检截图入库 ④ 登记未清偿债（AA 欠账 / 描边 3:1 / Table·Menu·Tag 域暗色 token / `--theme-zebra` 暗色缺陷） | W1+W2+W3 | ① `lint:ci` + `css-contract-check.js` exit 0 ② 单测全绿 ③ 矩阵「未归类差异 = 0」 ④ M1–M16 快照落盘 | — |

### 4.1 严格顺序约束（不可交换）

| 约束 | 理由 |
|---|---|
| **W1（变量层）先于 W2（色板）** | ROADMAP 成功标准 #4 明确：「变量层与色板重构**分两步落地**，避免串色源被色板重构一起搬走、问题被掩盖」。若同 commit，VISFIX-01 的因果链将不可复现、回归守护失效。 |
| **W2（色板）先于 Phase 90/91/92** | ROADMAP「三处不可逆顺序约束」第 1 条：Phase 89 是 90/91/92 的**硬前置**。暗色白底未修就交还 token = 把白底固化进新规则。 |
| **W3 不含 T1/T2/T3/T4** | X-3：圆角是 Phase 90 / STYLE-03 的交付物。 |
| **W3 的死链删除先于 C13 断言上线** | C13 是**全量** hex 断言；若先上线 C13 再删死链，W3 一开始就红。 |
| **`exemption-reasons.json` 扩展先于 `check-hardcoded-colors.mjs` 接入 `lint:ci`** | 否则 `lint:ci` 首跑即红（扫描器 exit 1）。 |

---

## 5. 边界与不做什么

### 5.1 本 phase **不做**

| # | 不做项 | 理由 / 归属 |
|---|---|---|
| B-1 | **不迁 `onSearch`** | antd 6.6.4 未弃用（88-04 已实证） |
| B-2 | **不改 `colorPrimary` / `colorError` / `colorSuccess` 等品牌与功能色** | 它们是 **Seed**，算法会按模式重算色阶；改动会造成**大范围暗色视觉变更**，超出「2 处已登记变更」预算 |
| B-3 | **不碰 `Table` / `Menu` / `Tag` 组件 token 的暗色适配** | 属 Phase 91/92 域；89 改会造成未登记的暗色变更 |
| B-4 | **不做圆角统一（T1/T2/T3）、不做 `font-size` 迁 `inherit`（T4）** | Phase 90 / STYLE-03 交付物（X-3） |
| B-5 | **不改 VIS-01 复位块（`:933-941`）、不改 `border` 简写（`:874`）** | Phase 90 批次 C/B，且**必须同 commit** |
| B-6 | **不删 InputNumber 错误态欠账（M13）** | Phase 90 / VISFIX-05 |
| B-7 | **不动 `src/components/ConfigProvider.tsx`** | P-C2 冻结（只读） |
| B-8 | **不扩 `eslint.config.js` 的 hex 禁令到 `.css`** | ESLint `no-restricted-syntax` 走 JS/TS AST，**扩了也不生效**（§1.3.5）；CSS 门禁走 C13 独立脚本 |
| B-9 | **不引入新依赖**（stylelint / playwright / postcss 等） | 沿用 v1.23/v1.24「零新依赖」决策 |
| B-10 | **不改登录页视觉** | T9–T12 只做「提取为变量、值不变」 |
| B-11 | **不扩大 `useMemo` deps** | C15 单测硬断言 `deps.length === 2`；新增依赖会破该断言 |
| B-12 | **不修 `.xr-table-zebra` 暗色近白斑马纹**（`index.css:5118` 无暗色覆盖） | 既有缺陷，登记移交 Phase 92 |

### 5.2 本 phase **必须登记的未清偿债**

| # | 债 | 量化 | 归属 |
|---|---|---|---|
| D-1 | **暗色输入框描边对比度 1.56:1 < 3:1**（WCAG 1.4.11） | `--input-border #2d4d33` vs `#0f2e1b` | Phase 92（暗色 mega-cluster，描边提亮至 `#4a7d52` 级别） |
| D-2 | **ROADMAP 成功标准 #1 的 3:1 判据不可满足**（X-2） | `#0f2e1b` 对任意背景 ≤ 1.43:1 | **需规划者改判据或登记例外** |
| D-3 | **暗色占位符对比度**（若采纳 §2.3.3 选项 ②） | `#707068` on `#0f2e1b` = 2.95:1 | Phase 92 |
| D-4 | **`Table` / `Menu` / `Select.optionSelectedBg` 的暗色 token 仍为浅色** | `cream.headerBg #E9EFEB` | Phase 91/92 |
| D-5 | **`Button.defaultBorderColor` 暗色值**（若 89 采纳 §2.2.3 建议改则本债消失） | — | 89 内解决 / 否则 92 |
| D-6 | **`.xr-table-zebra` 暗色近白斑马纹** | `index.css:5118` | Phase 92 |
| D-7 | **REQUIREMENTS/PITFALLS 的「18 个文件命中」需更正为 17**；**STACK.md F3-c 与 REQUIREMENTS「机制更正」需更正**（X-1） | §1.3.2 / §1.1.2 | 文档更新（建议随本 phase 落盘） |

---

## 6. Sources

### 6.1 读过的项目文件

| 文件 | 用途 |
|---|---|
| `.planning/ROADMAP.md` | Phase 89 Goal / 成功标准 / 三处不可逆顺序约束 / M1–M16 口径 |
| `.planning/REQUIREMENTS.md`（STYLE-01/04、VISFIX-01/02、关键决策表） | 需求基线与 F3 前置障碍段 |
| `.planning/research/PITFALLS.md`（1026 行） | P-C3/P-C4/P-M8/P-M9/F-01/F-04/F-05、第 2 批 T1–T14、四批清单、量化基线 |
| `.planning/research/STACK.md`（801+ 行） | antd 6.6.4 token 派生链、variant 机制、Select DOM 结构 |
| `.planning/phases/88-style-foundation/88-QUANT-SNAPSHOT.md` | 88 末态量化基线（M1=1398 / M2=1331 / M9=5548 / M16=0） |
| `.planning/phases/88-style-foundation/88-visual-evidence/{matrix-fixture.js,matrix-sample.js,matrix-diff.cjs,before-*.json,after-*.json,matrix-diff-result.txt}` | 视觉矩阵基建与基线快照 |
| `ruoyi-react-frontend/src/design-system/components/AntdThemeBridge.tsx`（192 行） | **F3 改造目标**（全文读） |
| `ruoyi-react-frontend/src/design-system/tokens/colors.ts`（134 行） | `ruoyiBrand` 全量（**无 `dark` 组**） |
| `ruoyi-react-frontend/src/index.css`（5548 行，LF 行尾） | 变量层 / 暗色板 / hex / T1–T14 定位 |
| `ruoyi-react-frontend/src/pages/login/login.css`（503 行） | T9–T12 定位 |
| `ruoyi-react-frontend/src/components/shared/ActionButtons.css`（80 行） | T13 定位 + 暗色覆盖确认 |
| `ruoyi-react-frontend/src/components/dashboard/DashboardView.css`（51 行） | hex 命中 #1 |
| `ruoyi-react-frontend/scripts/css-contract-check.js`（1006+ 行） | C1–C14f/g 现状 + C6 豁免表 3 处 |
| `ruoyi-react-frontend/scripts/check-hardcoded-colors.mjs`（190 行） | 扫描器能力边界 |
| `ruoyi-react-frontend/scripts/exemption-reasons.json`（311 行） | 豁免结构（`layer` 维度） |
| `ruoyi-react-frontend/eslint.config.js`（468 行，§110-209） | hex 禁令块与豁免块 |
| `ruoyi-react-frontend/package.json` | `lint:ci` 组成（**不含** `check-hardcoded-colors.mjs`） |
| `ruoyi-react-frontend/src/components/ConfigProvider.tsx` | P-C2 复核（`:69 return children`） |
| `ruoyi-react-frontend/src/App.tsx`（:47-62） | Provider 嵌套顺序 |
| `ruoyi-react-frontend/src/design-system/components/ThemeProvider.tsx` | `data-color-mode` 写入 `<html>` |
| `ruoyi-react-frontend/src/design-system/components/__tests__/AntdThemeBridge.test.tsx`（110 行） | C15 断言结构（`deps.length === 2`） |
| `ruoyi-react-frontend/src/design-system/themes/**`（13 文件）、`components/{ColorSwitcher,ThemeSwitcher}.tsx`、`constants/colorPresets.ts`、`utils/color.ts` | 死链实证 |

### 6.2 读过的 node_modules 源码（antd 6.6.4）

| 文件:行 | 结论 |
|---|---|
| `antd/es/config-provider/index.js:213,392,408-422` | `override: { override: mergedToken, ...parsedComponents }` 的构造 |
| `antd/es/theme/useToken.js:49-81` | `getComputedToken`：`{...derivativeToken, override}` + `formatToken` + 组件 token 后 merge |
| `antd/es/theme/useToken.js:102` | `useCacheToken(theme, [defaultSeedToken, rootDesignToken], { override, getComputedToken })` |
| `antd/es/theme/util/alias.js:9-23` | ★★ **override 回填**：只删 seed 名，其余显式 token 盖回算法结果 |
| `antd/es/theme/themes/seed.js:19-70` | ★ **seed 全集**（**不含** `colorBgContainer` 等 Map Token） |
| `antd/es/theme/themes/default/index.js:29-45` | `genColorMapToken` 在 `...token` **之后**（算法内覆盖） |
| `antd/es/theme/themes/dark/index.js:26-51` | 暗色：`mergedMapToken` + `colorMapToken` 二次覆盖 |
| `antd/es/theme/themes/default/colors.js:18-45` | 浅色中性色派生公式（`getSolidColor(colorBgBase, N)`） |
| `antd/es/theme/themes/dark/colors.js:20-47` | 暗色中性色派生公式（`colorBgContainer=base@8` / `colorBorder=base@26`） |
| `antd/es/theme/themes/default/colorAlgorithm.js:3-6` | `getSolidColor` = `FastColor.darken(brightness)` |
| `antd/es/theme/themes/shared/genColorMapToken.js:35-95` | Map Token 输出清单 |
| `antd/es/theme/getDesignToken.js:5-14` | 静态 API 路径（`override: config.token`）—— 与运行时路径**不同**，勿混用 |
| `@ant-design/cssinjs/es/hooks/useCacheToken.js:45-59,68-135` | `mergedToken = Object.assign({}, ...tokens)` + `compute(mergedToken, override, theme)` |
| `@ant-design/cssinjs-utils/es/util/getComponentToken.js:4-31` | 组件 token merge：`{...defaultToken, ...customToken}`（算法之后） |
| `antd/es/date-picker/style/index.js:121,129,181,413` | picker 内层 `background:'transparent'`（根底色来源**待实机复核**） |

### 6.3 实跑命令（本机 `ruoyi-react-frontend/`，Windows PowerShell）

```powershell
# —— 变量层定位 ——
Select-String -Path src/index.css -Pattern '--input-bg'                 # 23 行（3 定义 + 20 消费）
Select-String -Path src/index.css -Pattern 'input-(text-)?placeholder'  # 6 行
Select-String -Path src/index.css -Pattern '--theme-warning-text|--radius' # 3 行
Select-String -Path src/index.css -Pattern '^\s*:root\s*\{'             # 27 / 4731
Select-String -Path src/index.css -Pattern '\[data-color-mode="light"\]' # 150

# —— hex 实测 ——
$idx = Get-Content src/index.css -Raw
([regex]::Matches($idx,'#[0-9a-fA-F]{6}\b')).Count                       # 304
(([regex]::Matches($idx,'#[0-9a-fA-F]{6}\b') | % {$_.Value.ToLower()} | Sort-Object -Unique).Count)  # 64
$all = Get-ChildItem src -Recurse -Include *.css,*.less | Get-Content -Raw
([regex]::Matches($all,'#[0-9a-fA-F]{6}\b')).Count                        # 446
([regex]::Matches($all,'!important')).Count                              # 1398
([regex]::Matches($idx,'!important')).Count                              # 1331

# —— 门禁实跑 ——
node scripts/check-hardcoded-colors.mjs ; echo "exit=$LASTEXITCODE"      # exit=1，17 文件 / 150 命中
node -e "console.log(JSON.stringify(require('./package.json').scripts,null,1))"  # lint:ci 组成

# —— antd token 实跑（复刻 useToken.js:49-81 链路）——
node <probe>.cjs   # A/B/C/D 四组 × 明暗，见 §1.1.2
node <cr>.cjs      # WCAG 对比度实算，见 §2.1.2

# —— 量化口径 ——
node <m>.cjs       # M3/M4/M5/M9 复算
```

### 6.4 置信度汇总

| 结论 | 置信度 |
|---|---|
| antd token 派生链（override 回填 → 显式 Map Token 生效） | **HIGH**（源码 + 实跑双证） |
| `ruoyiBrand.dark` 色值表 | **HIGH**（与 `index.css` 暗色板逐值对齐）｜**设计决策 MEDIUM**（`elevated/overlay/hover/active` 的层级关系建议实机确认） |
| `--input-bg` 三处定义与 20 个消费点行号 | **HIGH** |
| `--input-placeholder` 统一方案 | **HIGH** |
| hex 实测基线（304/64/446/1398/1331） | **HIGH** |
| 17 个失败文件与处置 | **HIGH**（文件与命中数）｜**死链可删性 MEDIUM**（需执行者按 §2.4.2 的 6 项核查确认） |
| T1–T14 行号复核与建议值成立性 | **HIGH**（行号）｜**T13/T14 建议值「不成立」的判定 HIGH**（有具体色值对照） |
| 3:1 判据不可满足 | **HIGH**（数学 + 源码双证） |
| 矩阵「2 类变更」的形态落点 | **MEDIUM**（`.ant-picker` 根底色 / `.ant-mentions` 底色来源**待实机复核**） |

---

*研究完成：2026-10-28｜只读研究，未修改业务代码，未 git commit。*
