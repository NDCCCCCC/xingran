---
slug: dept-tree-black-background
name: dept-tree-black-background
status: resolved
trigger: 所有关于部门列表的页面都出现了同样的黑底问题（部门树面板下方出现大片纯黑区域）
created: 2026-10-15
updated: 2026-10-16
resolved: 2026-10-16
cycles: 3
---

## 症状 (Symptoms)

**期望行为:**
- 所有包含部门列表（部门树）的页面中，部门树面板下方区域应显示页面正常背景色（浅色）
- 截图页面：楼栋列表页（operations 模块），左侧为"部门列表"树形面板

**实际行为:**
- 部门树面板本身渲染正常（白色背景、树节点、搜索框、滚动条均正常）
- 但树面板容器下方出现大片**纯黑色区域**，占据面板容器的剩余高度
- 用户报告：**所有关于部门列表的页面都出现了同样的黑底问题**（共享组件问题，非单页面）

**错误信息:**
- 浏览器控制台（F12）**无报错** → 纯样式/渲染问题，非 JS 异常

**时间线:**
- **最近才出现**（之前正常）→ 指向近期前端改动引入

**主题状态:**
- 当前为**亮色主题，从未切换过暗色主题** → 排除 themeStore 状态残留

**重现步骤:**
1. 打开任何包含部门列表/部门树组件的页面（如楼栋列表页）
2. 观察左侧部门树面板下方区域 → 黑色

**关键线索:**
- 多页面复现 → 问题在共享组件（部门树组件及其容器，可能为 DeptTree / useDeptTree 相关，见 memory: rc-tree-virtual-motion-infinite-loop）
- 黑色区域在树面板容器内部（面板有白色部分+黑色部分，说明容器本身高度正常，是内部某层背景变黑）
- 无 JS 报错 → CSS 层面问题（背景色、透明度、CSS 变量、dark mode class 泄漏等）

---

## 当前关注点 (Current Focus)

**最终状态:** RESOLVED (2026-10-16)。3 个修复 cycle 全部完成，用户浏览器实机确认（cycle 1 黑底→白；cycle 2 圆角回归黑楔，失败；cycle 3 根除 Layout #000，圆角成立）。Resolution 见下。

---

## 证据 (Evidence)

- timestamp: 2026-10-15
  checked: git diff ruoyi-react-frontend/src/components/DeptTree/index.tsx + DeptSidebar.tsx
  found: 仅 prettier 格式化 + filterFn 单遍遍历重构，无样式改动
  implication: 共享组件本体不是元凶，转向 CSS/主题层
- timestamp: 2026-10-15
  checked: git diff ruoyi-react-frontend/src/index.css
  found: 未提交新增 `.dept-list-sider { background: transparent !important; }`（index.css:5250-5252），注释"业务页内嵌 Sider 不是侧边导航 — 排除全局深绿底，透出奶油画布"
  implication: !important 透明压过所有部门树 Sider 的内联 background:"#fff"
- timestamp: 2026-10-15
  checked: grep `background: "#000"` across pages
  found: 11 个业务页根 Layout 均为 `<Layout style={{ background: "#000", minHeight: "calc(100vh - 64px)" }}>`（buildings/floors/workstations/server-rooms/room-devices/info-points/dedicated-lines/network-devices/rpa×3），为已提交代码（buildings 页 git diff 为空）
  implication: 透明 Sider 透出的就是这层纯黑
- timestamp: 2026-10-15
  checked: index.css:428 全局规则 + 亮色主题变量
  found: `.ant-layout-sider { background: var(--sidebar-bg) !important }`；亮色主题 `--sidebar-bg: #FFFFFF`（index.css:127）
  implication: 收窄 transparent 范围后业务页自然回退为白色（与 HEAD 一致）
- timestamp: 2026-10-15
  checked: 用户页 system/user/index.tsx:575-583
  found: Layout/Sider 均 `background: "transparent"`，Sider 带 `dept-list-sider xr-dept-panel` 双类
  implication: transparent 规则真正服务的是用户页 xr 重设计；收窄到 .xr-dept-panel 即可两全
- timestamp: 2026-10-15
  checked: 业务页部门 Card 圆角真实来源（DeptTree/index.tsx Card 无显式 radius）
  found: AntdThemeBridge.tsx:90/166 设 borderRadiusLG:12，但 index.css:602 `.ant-card { border-radius: var(--theme-radius-lg) !important }` 以 CSS !important 胜出 → 实际渲染 8px（index.css:126 --theme-radius-lg: 8px，全文件唯一定义）
  implication: 背景圆角必须用 var(--theme-radius-lg)（8px）而非 antd token 12px，才能与用户看到的 Card 圆角一致
- timestamp: 2026-10-15
  checked: Sider/Card 几何衔接（DeptSidebar.tsx:41 padding "0 16px 16px 0"）
  found: Sider 上/左 padding 为 0 → Card 左上角与白底左上角位置重合；底/右有 16px 空隙
  implication: 背景圆角=Card 圆角（8px）时左上角两弧线完全嵌套；背景半径更大会在左上角露黑楔——用户"对齐或略大"中选"对齐"
- timestamp: 2026-10-15
  checked: border-radius 级联竞争面
  found: index.css:428/2132 全局 .ant-layout-sider 只有 background/width/border-right，无 radius；antd Sider 默认无 radius；4781 .layout-sidebar 是另一组合不相关
  implication: `.dept-list-sider` 加 radius 无任何竞争，!important 稳赢
- timestamp: 2026-10-15
  checked: 修复后验证（圆角追加）
  found: postcss 全文件 PARSE OK 且目标规则含 border-radius: var(--theme-radius-lg) !important；vitest 3 文件 25 用例全过（deptSidebar + DeptTree/index.test + components.render，与基线一致）；grep/Read 显示的 `\*` 注释开头为工具转义伪影，od 原始字节确认 `/*` 正常；"Could not parse CSS stylesheet" 为 jsdom 对现代 CSS 的既有告警，与本改动无关
  implication: 修复语法有效、无回归，待用户浏览器确认视觉效果
- timestamp: 2026-10-15
  checked: cycle 2 人工验证结果（FAILED）+ index.css:5253-5262 当前源码
  found: 用户报告无圆角观感且下方/左侧出现黑色条带；源码中 background:#fff 与 border-radius 规则均完好存在（语法正确）
  implication: background 生效（面板白色正常）→ 变黑只能来自 border-radius 对背景的裁剪——楔形区透出下层 #000；排除"编辑丢失/语法损坏"
- timestamp: 2026-10-15
  checked: --theme-radius-lg 运行时解析（design-system/themes/index.ts:252 + minimal/light.ts:102）
  found: applyRadiusVariables 用 JS setProperty 覆盖 CSS :root 值；minimal 主题（getTheme 默认回退）radius.lg="6px"（dark 相同）
  implication: 变量必然解析（6px 或当前主题值），radius 确实渲染——用户提示的"变量未定义/值不对"假设排除；Card 与 Sider 同用此 var，半径恒一致
- timestamp: 2026-10-15
  checked: 级联竞争复检（index.css:428/2132 .ant-layout-sider 规则全文）
  found: 两处均无 border-radius（仅 background/border-right/width/transition/flex）
  implication: "radius 被 antd Sider 规则压掉"假设排除；.dept-list-sider 的 radius 唯一来源必胜出
- timestamp: 2026-10-15
  checked: #000 在 pages 全域 grep + 11 页 Content 背景
  found: background:"#000" 仅存在于 11 个业务页根 Layout（唯一出现点）；每页 Content 均 `<Content style={{ background: "#fff" }}>` 不透明全覆盖；git log 显示 #000 来自 lint/格式化提交（d2fa1be1），非刻意暗色画布设计
  implication: #000 是 vestigial——提交设计中完全不可见，只在透明/裁剪泄露时显形；改为浅色视觉零变化，且根除 cycle 0 与 cycle 2 的共同病根（透明层透出纯黑）
- timestamp: 2026-10-15
  checked: 测试对 #000 的断言（**/*.test.{ts,tsx} grep）
  found: 无任何测试断言 Layout background:"#000"（仅 contrast/iconColor 无关用例）
  implication: 改 Layout 背景色不会破坏现有测试
- timestamp: 2026-10-15
  checked: 修复后自验（Layout #000→#f5f5f5 ×11 + index.css 注释更新）
  found: postcss PARSE OK；vitest deptSidebar + components.render×2 + DeptTree 4 文件 47 用例全过；tsc --noEmit exit 0；grep src/pages 无 `background:"#000"` 残留；CSS 块复读确认（过程中一次注释未闭合笔误致规则被整段注释，已当场修复并复读验证）
  implication: 修复语法/类型/回归全绿，待用户浏览器实机确认圆角观感与无黑色区域
- timestamp: 2026-10-16
  checked: cycle 3 人工验证（human-verify）
  found: 用户浏览器实机确认"已确认修复"——黑底消除、面板四角圆角观感成立、用户管理页画布无变化、其余视觉无回归
  implication: 修复验证通过，会话 RESOLVED

## 已排除 (Eliminated)

- hypothesis: themeStore 暗色主题状态残留 — 用户确认亮色主题且从未切换过，排除
- hypothesis: JS 运行时异常导致渲染异常 — 浏览器控制台无报错，排除
- hypothesis: DeptTree/DeptSidebar 组件近期改动引入 — git diff 仅 prettier 格式化 + filterFn 重构，无样式改动，排除
- hypothesis: 收窄 transparent 后回退 var(--sidebar-bg) 即可 — 工作区该变量已全部改为品牌深绿(#14532d/#0a2418)，回退会变深绿面板，排除（改为显式 #fff）
- hypothesis: (cycle 2 失败归因) var(--theme-radius-lg) 未定义或值不对 — 实测 JS 运行时 setProperty 覆盖（minimal lg=6px），变量必解析，排除
- hypothesis: (cycle 2 失败归因) radius 被更后位置/更高特异性规则压掉 — index.css:428/2132 全局 .ant-layout-sider 无 radius，.dept-list-sider 是唯一 radius 来源，排除
- hypothesis: (cycle 2 失败归因) 编辑未落盘/规则块语法损坏 — 源码 5253-5259 白底+radius 完好且面板白色正常，排除；真因 = radius 裁剪楔形透出 #000（结构必然，圆角+黑底不可共存）

## Resolution

root_cause: 三层递进——①(cycle 0 原始 bug) 未提交 xr 重设计在 index.css 新增 `.dept-list-sider { background: transparent !important }`，透出 11 个业务页根 Layout 的 vestigial `background:"#000"`。②(cycle 1 已验证修复) Sider 显式 #fff + 收窄 transparent 到 `.xr-dept-panel`。③(cycle 2 回归根因，本次确认) 给 Sider 白底加 border-radius 后，背景在四个圆角处被裁剪，裁掉的楔形区域必然透出**父层**背景——父层是 #000 → 面板三角出现黑色缺口/条带；"没出现圆角"是因为左上角与 Card 同半径（运行时主题 JS 注入 var(--theme-radius-lg)，minimal=6px）弧线完全重合不可见，其余角表现为黑色缺陷而非圆角观感。核心矛盾：Sider 背后是纯黑时，圆角与白底不可共存。
fix: 根除病根：11 个业务页根 Layout `background:"#000"` → `"#f5f5f5"`（antd 浅画布；#000 经查为 vestigial——每页 Content 均 #fff 不透明全覆盖、#000 在提交设计中完全不可见、git log 显示其来自 lint 格式化提交而非暗色设计意图）。视觉零变化 + 圆角楔形从透黑变为透浅灰：黑楔消失、四角圆角观感首次成立，且未来任何透明/裁剪处理都不再可能透出纯黑（cycle 0 的 bug 向量被永久关闭）。保留 cycle 1 的 `.dept-list-sider{background:#fff!important;border-radius:var(--theme-radius-lg)!important}` 与 `.xr-dept-panel` transparent 例外；更新 index.css 过时注释。
verification: postcss 全文件 PARSE OK；vitest 4 文件 47 用例全过（deptSidebar + components.render×2 + DeptTree）；npx tsc --noEmit -p tsconfig.app.json exit 0 零错误；grep 确认 src/pages 无 `background:"#000"` 残留；CSS 块人工复读确认注释闭合、规则完好（修复过程中曾出现一次注释未闭合导致整条规则被注释掉的笔误，已当场发现并修复）。**用户浏览器实机验证通过（2026-10-16，cycle 3 human-verify："已确认修复"）。**
files_changed:
  - ruoyi-react-frontend/src/index.css
  - ruoyi-react-frontend/src/pages/operations/buildings/index.tsx
  - ruoyi-react-frontend/src/pages/operations/floors/index.tsx
  - ruoyi-react-frontend/src/pages/operations/workstations/index.tsx
  - ruoyi-react-frontend/src/pages/operations/server-rooms/index.tsx
  - ruoyi-react-frontend/src/pages/operations/room-devices/index.tsx
  - ruoyi-react-frontend/src/pages/operations/info-points/index.tsx
  - ruoyi-react-frontend/src/pages/operations/dedicated-lines/index.tsx
  - ruoyi-react-frontend/src/pages/operations/rpa/tasks/index.tsx
  - ruoyi-react-frontend/src/pages/operations/rpa/workers/index.tsx
  - ruoyi-react-frontend/src/pages/operations/rpa/executions/index.tsx
  - ruoyi-react-frontend/src/pages/network/devices/index.tsx
