---
status: completed
phase: 64-brand-token-layer-and-contrast-verification
source: 64-01-SUMMARY.md
started: 2026-08-18
updated: 2026-08-19
---

## Current Test

number: 5
name: QA-01 / SC#5 对比度自动验证跑绿
expected: |
  ```
  cd D:/code/ClaudeCode/guoguo/xingran-react-frontend && npm test -- --run src/design-system/tokens/colors.test.ts
  ```
  输出 PASS，所有断言通过；CI 等同门。
  关键断言：D-03 反向断言锁住 `#FFFFFF on #C09058 < 3.5:1`（铜金不可做实心主按钮）。
awaiting: done

## Tests

### 1. TOKEN-01 / SC#1 浏览器 DevTools CSS 变量验证
expected: 见 Current Test
result: pass
notes: |
  Light mode 变量全部正确：`--theme-primary: #156031`, `--theme-primary-hover: #2e7444`, `--theme-primary-active: #14542e`, `--theme-primary-light: #1a6839`, `--theme-primary-lighter: #e9efeb`, `--theme-neutral-100: #e9efeb`, `--theme-neutral-200: #dbd7ce`, `--theme-neutral-400: #707068`, `--theme-neutral-800: #14532d`, `--theme-bg-primary: #f0ece3`, `--theme-bg-surface: #ffffff`, `--theme-text-primary: #101010`.
  Dark mode 同样正确：`--sidebar-bg: #0a2418`, `--sidebar-text-active: #e0e0b0`, `--theme-bg-primary: #0f2e1b`, `--theme-bg-surface: #1a2e1f`, `--theme-text-primary: #f0ece3`.
  视觉（截图验证）：侧边栏白底 + 铜金激活态、页面奶油底 + 白卡双层纸感、主按钮绿底淡黄字、登录页深绿面板 + 铜金 SM2/SM3/SM4 标签 + 铜金登录按钮。
  对比度实测：`#FFFFFF on #156031 = 7.64:1 ✓`、`#FEF3C7 on #156031 = 6.86:1 ✓ (通过 AA)`、`#E0E0B0 on #156031 = 5.62:1 ✓`、`#707068 on #FFFFFF = 4.99:1 ✓`、`#FFFFFF on #B88850 = 3.15:1 ✓`、`#FFFFFF on #C09058 = 2.85:1 ✓ (反向断言锁住)`。
  偏差 1：表头底实测 `#FFFFFF` 白底而非 `#E9EFEB` 绿灰淡彩（待 SC#3 检查 AntdThemeBridge 是否覆盖 Table.headerBg）。
  偏差 2：主按钮文字色实测 `#FEF3C7` 而非 `#FFFFFF`（D-03 推荐值；但 AA 6.86:1 通过，属于"达标但非最优"）。记入 Gaps 建议 Phase 66 对齐。
  截图：test1-system-user-after-login.png（登录页）+ test1-system-user-page.png（系统用户页）

### 2. TOKEN-02 / SC#2 xingranBrand TS 常量可被 import
expected: 见上方 Test 2 description
result: pass
notes: |
  DevTools Console `import('/src/design-system/tokens/colors.ts')` 成功解析；`xingranBrand` 对象包含完整 11 个顶层键：
  - `green` 6 阶（50/100/200/300/400/900）+ `greenPrimary`/`greenPrimaryHover`/`greenPrimaryActive`/`greenPrimaryLight` 4 个扁平别名
  - `copper` 4 阶（400/500/600/700）+ `copperAccent`
  - `cream` 中性 9 项（canvas/surface/fg/muted/mutedStrong/border/borderStrong/headerBg/zebraBg）
  - `gradient` 2 项（brandPanel / canvas）
  - `functional` 6 项（success/successSolid/warning/warningText/danger/info）
  - `onDark` 3 项（white/lightYellow/paleYellow）
  所有值与 brand-spec.md 逐字核验一致：`#156031` / `#2E7444` / `#C09058` / `#F0ECE3` / `#DBD7CE` / `#707068` / `#E9EFEB` / `#FEF3C7` 等全部匹配。
  TS 类型 + JSDoc 注释包含 OKLch 值与 WCAG 对比度（T5 测试已通过验证）。

### 2. TOKEN-02 / SC#2 xingranBrand TS 常量可被 import
expected: |
  在 DevTools Sources 面板搜 "xingranBrand"，应在 colors.ts 末尾看到导出对象，含 brand 属性（深绿/铜金/中性/渐变/onDark/functional 六个分组），每项带 OKLch 与 WCAG 注释。
  或者在 Console 跑：
  ```
  const m = await import('/src/design-system/tokens/colors.ts');
  console.log(Object.keys(m.xingranBrand));
  // 期望输出含 'green' / 'copper' / 'cream' / 'gradient' 等
  ```
result: pass (verified 2026-08-19 via chrome-devtools console import: Object.keys(xingranBrand) 含 green/copper/cream/gradient/onDark/functional；色值逐字核验 greenPrimary=#156031 hover=#2E7444 active=#14542E copperAccent=#C09058 cream=#F0ECE3 gradient.brandPanel=linear-gradient(135deg,#14532D 0%,#156031 60%,#1E6B3F 100%))

### 3. TOKEN-03 / SC#3 Antd 6 内置组件品牌化
expected: |
  - Antd Button `type="primary"` 默认渲染：绿底白字，hover 略浅绿
  - Antd Table 表头底色：#e9efeb（不是 #fafafa）
  - Antd Input focus 时边框：深绿 #156031
  - Antd Tabs 选中下划线：深绿
  - Antd Tag (SM2/SM3/SM4)：浅黄底 + 铜金字
  - Antd Menu 选中项：品牌主色高亮
result: pass_with_issues
issues_found:
  - Antd Button primary：✓ 绿底 `#156031`，但文字色 `#FEF3C7`（淡黄，6.86:1 AA）非 D-03 推荐 `#FFFFFF`（7.64:1）。视觉达标但非最优。
  - Antd Table 表头底：⚠ 实测 `#FFFFFF` 白底，非 `#E9EFEB` 绿灰淡彩。AntdThemeBridge 未覆盖 `Table.headerBg`。影响 COMP-02 Phase 66 施工。
  - Antd Input focus 边框：⚠ 实测 `#101010` 黑色（text-primary），非 `#156031` 品牌主色。AntdThemeBridge 未覆盖 `Input.activeBorderColor` 或 `Input.activeShadow`。
  - Antd Tag（⌘K / 超级管理员）：⚠ 文字色 `#156031` 深绿 + 底 `#1A6839` 绿浅阶，非 brand-spec 推荐的铜金字 `#B88850` + 浅黄底 `#FEF3C7`（淡黄标签是登录页品牌锚点）。
  - Antd Tag（启用状态）：✓ 底 `#E9EFEB` 绿灰淡彩（Antd 默认 success 色）。
  - Antd Menu 选中项：✓ 铜金 `#B8854C` 激活态（admin-design-plan 同款，现状已落地，Phase 65 决定是否统一为绿）。
  - Antd Card：✓ 白卡 `#FFFFFF` + 圆角 12px + 轻阴影 `rgba(0,0,0,0.05)`（但 T3 阴影色应为深绿暖色调，可能未在 Card 上生效 —— Phase 66 检查）。
  - Antd 二级按钮：✓ 白底 `#FFFFFF` + 边 `#DBD7CE` + 字 `#101010`。
  - Tabs：本页无 Antd Tabs 组件（侧边栏菜单非 Tabs），跳过。
notes: |
  AntdThemeBridge 的 `theme.token` 全局令牌已生效（主色 `#156031` 绿底 + 圆角 12px + 边框 `#DBD7CE`），但 `theme.components` 级覆盖不完整：Table / Input / Tag 三组组件的 headerBg / activeBorderColor / defaultBg/defaultColor 未接品牌令牌。
  D-03 反向断言 QA-01 通过（`#FFFFFF on #C09058 = 2.85:1 < 3.5:1` 锁住铜金误用），但正向断言未锁 `#FEF3C7` on `#156031 = 6.86:1`（AA 通过但非 D-03 最优）。
  Phase 66 应补齐 Table.headerBg / Input.activeBorderColor / Tag SM2-SM3-SM4 浅黄标签 / 主按钮文字色 → `#FFFFFF` 四处。

### 4. TOKEN-04 / SC#4 shadows / spacing / typography 调性对齐
expected: |
  - 卡片 / Modal 阴影：暖色调偏深绿（不再是中性黑或冷蓝）
  - 按钮 / 输入框圆角：8px（统一，不再 4/6/8 混用）
  - 主标题字重收敛（不再 700+ 死重），整体读感更纸面
  - 数字 / 代码块：JetBrains Mono（如能装上，否则退化到 monospace）
result: pass
notes: |
  **阴影：** `shadows` 全阶（xs/2xl/inner）色相 `rgba(15,46,27,*)` 深绿低饱和 —— 纸感暖色调 ✓
  `coloredShadows.primary = "0 4px 14px rgba(21, 96, 49, 0.4)"` —— 深绿品牌阴影 ✓
  `coloredShadows.copper = "0 4px 14px rgba(192, 144, 88, 0.4)"` —— 铜金品牌阴影 ✓
  `radius.control = "8px"` —— 统一控件圆角 ✓
  **间距：** `spacing.md = "16px"` / `xs = "4px"` / `lg = "24px"` —— 与 brand-spec 节奏一致 ✓
  **字体：**
  - `fontFamily.sans` 含 `"PingFang SC"`, `"Microsoft YaHei"`（中文无衬线栈）
  - `fontFamily.mono` 首项 `"JetBrains Mono"`（代码块品牌字体）✓
  - `fontFamily.serif` 含 `"Songti SC"`, `"Source Han Serif SC"`（衬线中文字体）
  - `fontWeight` 9 阶从 thin(100) 到 black(900)
  - `fontSize` 11 阶 xs(12) → 6xl(60)
  **遗留警告（Phase 65/66 处理）：**
  - `shadows.ts` 保留 `neumorphicShadows` / `glassShadows` / `directionalShadows` 三组多主题阴影（Phase 65 删 6 套主题时清）
  - Card 实测阴影 `rgba(0,0,0,0.05)` 仍走 Antd 默认而非 shadows 令牌 —— Phase 66 COMP-02 检查 AntdThemeBridge 是否覆盖 Card.boxShadow

### 5. QA-01 / SC#5 对比度自动验证跑绿
expected: |
  ```
  cd D:/code/ClaudeCode/guoguo/xingran-react-frontend && npm test -- --run src/design-system/tokens/colors.test.ts
  ```
  输出 PASS，所有断言通过；CI 等同门。
  关键断言：D-03 反向断言锁住 `#FFFFFF on #C09058 < 3.5:1`（铜金不可做实心主按钮）。
result: pass
notes: |
  **QA-01 对比度测试：** 20/20 断言通过（vitest 1.71s）。
  **全量测试：** 14/14 测试文件通过、100/100 断言通过（75.65s）。
  **无回归：** 现有测试零失败，包括 zustand / component / hook / util 测试。
  **D-03 反向断言验证：** `#FFFFFF on #C09058 = 2.85:1 < 3.5:1`（铜金不可做实心主按钮）—— locked ✓
  **实测对比度（浏览器验证）：**
  - `#FFFFFF on #156031` = 7.64:1（D-03 目标值，完全达标）
  - `#FEF3C7 on #156031` = 6.86:1（AA 通过，但非 D-03 最优；记入 Gaps 建议 Phase 66 收紧）
  - `#E0E0B0 on #156031` = 5.62:1（brand-spec 实测匹配）
  - `#707068 on #FFFFFF` = 4.99:1（≥4.9:1 达标）
  - `#FFFFFF on #B88850` = 3.15:1（大字达标）
  - `#FFFFFF on #C09058` = 2.85:1（D-03 反向断言通过）

## Summary

total: 5
passed: 4
pass_with_issues: 1
issues: 4
pending: 0
skipped: 0
blocked: 0

**Overall:** Phase 64 目标达成 —— 品牌令牌真相源落地、对比度自动验证 20/20 通过、构建回归全绿、应用保持 buildable。视觉偏差 4 项均为非阻塞，全部路由至 Phase 66 (COMP-01..04) 或 Phase 65 (主题移除) 处理。

## Gaps

<!-- YAML format for plan-phase --gaps consumption -->
- truth: "Antd Table 表头底色为 `#E9EFEB` 绿灰淡彩（brand-spec 实测）"
  status: failed
  reason: "实测白底 `#FFFFFF`。AntdThemeBridge 未覆盖 `Table.headerBg` 组件级 token。Phase 64 T4 只覆盖了 theme.token 全局主色，未覆盖 Table 组件覆盖。"
  severity: minor
  test: 3
  root_cause: "AntdThemeBridge.tsx 的 theme.components 不完整 —— 缺 Table 覆盖"
  artifacts: ["xingran-react-frontend/src/design-system/components/AntdThemeBridge.tsx"]
  missing: ["theme.components.Table = { headerBg: xingranBrand.cream.headerBg, headerColor: xingranBrand.cream.fg, ... }"]
  debug_session: "Phase 66 COMP-02 补齐"
- truth: "Antd Input focus 边框为品牌主色 `#156031`（brand-spec 推荐）"
  status: failed
  reason: "实测 `#101010` 黑色（text-primary）。AntdThemeBridge 未覆盖 `Input.activeBorderColor` / `Input.hoverBorderColor` / `Input.activeShadow`。"
  severity: minor
  test: 3
  root_cause: "AntdThemeBridge.tsx 的 theme.components.Input 不完整"
  artifacts: ["xingran-react-frontend/src/design-system/components/AntdThemeBridge.tsx"]
  missing: ["theme.components.Input = { activeBorderColor: xingranBrand.greenPrimary, hoverBorderColor: xingranBrand.greenPrimaryLight, activeShadow: ... }"]
  debug_session: "Phase 66 COMP-04 补齐"
- truth: "SM2/SM3/SM4 标签为浅黄底 `#FEF3C7` + 铜金字 `#B88850`（登录页品牌锚点）"
  status: failed
  reason: "实测 ⌘K / 超级管理员 Tag 底 `#1A6839` 绿浅阶 + 字 `#156031` 深绿。AntdThemeBridge 未覆盖 Tag 组件或默认 Tag 走 Antd 默认色。"
  severity: minor
  test: 3
  root_cause: "AntdThemeBridge.tsx 的 theme.components.Tag 未覆盖品牌浅黄配方；业务组件（⌘K 搜索 Tag / 用户管理 Tag）未指定品牌色"
  artifacts: ["xingran-react-frontend/src/design-system/components/AntdThemeBridge.tsx", "src/components/layout/header.tsx", "src/pages/system/user/index.tsx"]
  missing: ["theme.components.Tag = { defaultBg: xingranBrand.onDark.paleYellow, defaultColor: xingranBrand.copper.copperSolid, ... }"]
  debug_session: "Phase 66 COMP-04 补齐（登录页 SM2/SM3/SM4 已是正确配方，后台组件需对齐）"
- truth: "主按钮文字色为 `#FFFFFF`（7.64:1 最优，D-03 推荐值）"
  status: failed
  reason: "实测 `#FEF3C7`（淡黄，6.86:1）。Antd 6 默认 primary 按钮文字色未覆盖。AA 4.5:1 通过，但 D-03 推荐 7.64:1。"
  severity: cosmetic
  test: 3
  root_cause: "AntdThemeBridge.tsx 的 theme.components.Button.primaryColor / textColor 未覆盖"
  artifacts: ["xingran-react-frontend/src/design-system/components/AntdThemeBridge.tsx"]
  missing: ["theme.components.Button = { primaryColor: xingranBrand.greenPrimary, primaryTextColor: '#FFFFFF', primaryHover: xingranBrand.greenPrimaryHover, ... }"]
  debug_session: "Phase 66 COMP-03 补齐（D-03 按钮纪律）"