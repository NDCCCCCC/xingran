---
phase: 84-p1-70
plan: 02a
type: execute
wave: 2
depends_on:
  - 84-01a
  - 84-01b
files_modified:
  - xingran-react-frontend/src/components/layout/__tests__/*.test.tsx
  - .coverage-fe-floors
  - .planning/frontend-coverage-baseline.md
autonomous: true
requirements:
  - COMP-03
  - QUAL-01
user_setup: []
must_haves:
  truths:
    - "[COMP-03] components/layout 16 文件 507 stmts(HybridLayout/ClassicLayout/InnovativeLayout + Sidebar/Header/Breadcrumb + shared/TabBar 等)语句覆盖率 ≥70%"
    - "[D-11] HybridLayout 测试覆盖路由跳转 / 菜单折叠 / 主题切换等核心交互场景"
    - "[D-02] layout 测试 useLayoutStore.setState({ currentLayout: \"hybrid\" }) beforeEach reset(对齐 83 D-05 Zustand resetBetweenTests)"
    - "[QUAL-01] 159 存量测试不回归 + 新增测试通过"
  artifacts:
    - path: xingran-react-frontend/src/components/layout/__tests__/HybridLayout.test.tsx
      provides: 主布局三件套路由跳转 / 菜单折叠 / 主题切换测试
    - path: xingran-react-frontend/src/components/layout/__tests__/ClassicLayout.test.tsx
      provides: 经典布局渲染 + 导航交互
    - path: xingran-react-frontend/src/components/layout/__tests__/InnovativeLayout.test.tsx
      provides: 新布局渲染 + 创新交互
    - path: xingran-react-frontend/src/components/layout/__tests__/Sidebar.test.tsx
      provides: 侧边栏菜单折叠 / 路由跳转断言
    - path: xingran-react-frontend/src/components/layout/__tests__/Header.test.tsx
      provides: 顶栏用户菜单 / 通知 / 主题切换断言
    - path: xingran-react-frontend/src/components/layout/__tests__/Breadcrumb.test.tsx
      provides: 面包屑根据路由渲染
    - path: xingran-react-frontend/src/components/layout/shared/__tests__/TabBar.test.tsx
      provides: TabBar 切换 / 关闭断言
    - path: xingran-react-frontend/src/components/layout/__tests__/layoutSwitcher.test.tsx
      provides: 布局切换器
    - path: .coverage-fe-floors
      provides: components/layout 行 bump 至实测 −0.5pp
    - path: .planning/frontend-coverage-baseline.md
      provides: 84-02a ratchet 行追加
  key_links:
    - from: xingran-react-frontend/src/components/layout/__tests__/HybridLayout.test.tsx
      to: xingran-react-frontend/src/store/layoutStore.ts
      via: beforeEach useLayoutStore.setState({ currentLayout: "hybrid" }) reset
    - from: xingran-react-frontend/src/components/layout/__tests__/Sidebar.test.tsx
      to: xingran-react-frontend/src/store/menuStore.ts
      via: vi.mock("@/store/menuStore") 提供 fixture 菜单数据
    - from: xingran-react-frontend/src/components/layout/__tests__/Header.test.tsx
      to: xingran-react-frontend/src/store/authStore.ts
      via: useAuthStore.setState({ user: { username: "..." } }) 注入 fixture
---

<objective>
将 `components/layout/` 16 文件 507 stmts(HybridLayout.tsx/ClassicLayout.tsx/InnovativeLayout.tsx + index.tsx + shared/TabBar.tsx + sidebar/Sidebar.tsx/sidebar-helper.ts/sidebar.utils.ts/sidebar.constants.ts + header/Header.tsx + breadcrumb.tsx + layout-switcher 等)语句覆盖率拉升至 ≥70%。HybridLayout 测试覆盖三种核心交互:(1)路由跳转 —— fireEvent.click sidebar 菜单 → useNavigate 被调;(2)菜单折叠 —— fireEvent.click 折叠按钮 → sidebar collapsed className 变化;(3)主题切换 —— fireEvent.click 主题按钮 → useThemeStore 状态变化。Sidebar 测试 mock menuStore 提供 fixture 菜单;Header 测试 useAuthStore.setState 注入 fixture user + fireEvent.user menu 展开。layout 测试必须 beforeEach useLayoutStore.setState({ currentLayout: "hybrid" }) reset(对齐 83 D-05 Zustand resetBetweenTests)。复用 wave 0 harness;同 PR bump components/layout floor 并追加基线文档 ratchet。

Purpose: COMP-03 是 P1 中等量级目录(507 stmts)——layout 是 routing + 菜单核心,模式 A 覆盖路由跳转 / 菜单折叠 / 主题切换是核心交互场景;useLayoutStore 不 reset 会导致单测过、一起跑部分失败(Pitfall #2)。

Output: 多个 layout 测试文件(主布局 + Sidebar + Header + TabBar 等)、components/layout floor bump、基线文档 ratchet 行。
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/workstreams/frontend-coverage/REQUIREMENTS.md
@.planning/workstreams/frontend-coverage/ROADMAP.md
@.planning/workstreams/frontend-coverage/phases/84-p1-70/84-CONTEXT.md
@.planning/workstreams/frontend-coverage/phases/84-p1-70/84-RESEARCH.md
@.planning/workstreams/frontend-coverage/phases/84-p1-70/84-VALIDATION.md
@.planning/workstreams/frontend-coverage/phases/84-p1-70/84-00-harness-and-gate-PLAN.md
@xingran-react-frontend/src/components/layout/
@xingran-react-frontend/src/store/layoutStore.ts
@xingran-react-frontend/src/store/menuStore.ts
@xingran-react-frontend/src/store/authStore.ts
@xingran-react-frontend/src/test/utils/renderWithProviders.tsx
@.coverage-fe-floors
@.planning/frontend-coverage-baseline.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: HybridLayout / ClassicLayout / InnovativeLayout 主布局三件套测试(路由跳转 + 菜单折叠 + 主题切换)</name>
  <files>
    xingran-react-frontend/src/components/layout/__tests__/HybridLayout.test.tsx
    xingran-react-frontend/src/components/layout/__tests__/ClassicLayout.test.tsx
    xingran-react-frontend/src/components/layout/__tests__/InnovativeLayout.test.tsx
    xingran-react-frontend/src/components/layout/__tests__/layoutSwitcher.test.tsx
  </files>
  <read_first>
    - xingran-react-frontend/src/components/layout/HybridLayout.tsx
    - xingran-react-frontend/src/components/layout/ClassicLayout.tsx
    - xingran-react-frontend/src/components/layout/InnovativeLayout.tsx
    - xingran-react-frontend/src/components/layout/index.tsx（layoutSwitcher 逻辑）
    - xingran-react-frontend/src/store/layoutStore.ts
    - xingran-react-frontend/src/store/themeStore.ts
    - xingran-react-frontend/src/test/utils/renderWithProviders.tsx
  </read_first>
  <action>
    1. 创建 `__tests__/HybridLayout.test.tsx` —— 主布局核心交互:
       - beforeEach: useLayoutStore.setState({ currentLayout: "hybrid" })(D-05 reset)
       - renderWithProviders(<HybridLayout>{content}</HybridLayout>) + content 渲染断言
       - fireEvent.click sidebar 菜单项 → useNavigate 被调断言(Pitfall #2 reset 防泄漏)
       - fireEvent.click 折叠按钮 → sidebar collapsed className 变化断言
       - fireEvent.click 主题切换按钮 → useThemeStore.setState 调用断言(可 mock useThemeStore)
    2. 创建 `__tests__/ClassicLayout.test.tsx` —— 经典布局:
       - beforeEach: useLayoutStore.setState({ currentLayout: "classic" })
       - 渲染经典布局样式断言
       - fireEvent.click 顶部菜单 + 路由跳转断言
    3. 创建 `__tests__/InnovativeLayout.test.tsx` —— 新布局:
       - beforeEach: useLayoutStore.setState({ currentLayout: "innovative" })
       - 渲染创新布局(可能含 3D / canvas 元素,简单断言容器存在即可)
       - fireEvent.click 创新交互按钮
    4. 创建 `__tests__/layoutSwitcher.test.tsx` —— 布局切换器:
       - renderWithProviders 渲染 + 三种布局按钮渲染断言
       - fireEvent.click 三种布局按钮 → useLayoutStore.setState({ currentLayout: ... }) 触发
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npx vitest run src/components/layout/__tests__/HybridLayout.test.tsx src/components/layout/__tests__/ClassicLayout.test.tsx src/components/layout/__tests__/InnovativeLayout.test.tsx src/components/layout/__tests__/layoutSwitcher.test.tsx 2>&1 | tail -20
    </automated>
  </verify>
  <done>
    - 4 个 layout 主件套测试通过,HybridLayout 三大核心交互全覆盖
  </done>
  <acceptance_criteria>
    - HybridLayout 测试覆盖路由跳转 / 菜单折叠 / 主题切换 3 类核心交互
    - 每个 layout 测试 beforeEach useLayoutStore.setState reset(Pitfall #2)
    - ClassicLayout / InnovativeLayout 各自核心交互断言
    - layoutSwitcher 测试覆盖三种布局切换
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 2: Sidebar / Header / Breadcrumb / shared/TabBar 配件测试(mock menuStore + authStore fixture)</name>
  <files>
    xingran-react-frontend/src/components/layout/__tests__/Sidebar.test.tsx
    xingran-react-frontend/src/components/layout/__tests__/Header.test.tsx
    xingran-react-frontend/src/components/layout/__tests__/Breadcrumb.test.tsx
    xingran-react-frontend/src/components/layout/shared/__tests__/TabBar.test.tsx
  </files>
  <read_first>
    - xingran-react-frontend/src/components/layout/sidebar/Sidebar.tsx
    - xingran-react-frontend/src/components/layout/sidebar/sidebar-helper.ts / sidebar.utils.ts / sidebar.constants.ts
    - xingran-react-frontend/src/components/layout/header/Header.tsx
    - xingran-react-frontend/src/components/layout/breadcrumb.tsx
    - xingran-react-frontend/src/components/layout/shared/TabBar.tsx
    - xingran-react-frontend/src/store/menuStore.ts
    - xingran-react-frontend/src/store/authStore.ts
  </read_first>
  <action>
    1. 创建 `__tests__/Sidebar.test.tsx` —— 侧边栏菜单折叠 + 路由跳转:
       - vi.mock("@/store/menuStore", ...) 提供 fixture 菜单数据(2-3 项 + 子菜单)
       - beforeEach: useLayoutStore.setState({ sidebarCollapsed: false })
       - renderWithProviders(<Sidebar />) + 菜单项渲染断言
       - fireEvent.click 子菜单项 → useNavigate 调用断言
       - fireEvent.click 折叠按钮 → sidebarCollapsed state 变化断言
    2. 创建 `__tests__/Header.test.tsx` —— 顶栏用户菜单 + 通知 + 主题切换:
       - beforeEach: useAuthStore.setState({ user: { username: "测试用户" } })
       - renderWithProviders(<Header />) + 用户名渲染断言
       - fireEvent.click 用户头像 → 下拉菜单展开 + fireEvent.click 登出 → 回调断言
       - fireEvent.click 通知图标 → 通知列表断言
       - fireEvent.click 主题切换按钮 → useThemeStore 状态变化
    3. 创建 `__tests__/Breadcrumb.test.tsx` —— 面包屑:
       - renderWithProviders 渲染不同 route("/system/user/list" 等)+ 面包屑路径断言(D-12 props 组合)
       - fireEvent.click 面包屑链接 → useNavigate 调用断言
    4. 创建 `shared/__tests__/TabBar.test.tsx` —— TabBar 切换 / 关闭:
       - beforeEach: useTabsStore.setState({ tabs: [{ key: "/a", title: "A" }, { key: "/b", title: "B" }] })
       - renderWithProviders(<TabBar />) + tab 标题渲染断言
       - fireEvent.click tab → 当前 tab 切换断言
       - fireEvent.click tab 关闭按钮 → tabs 数组变化断言
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npx vitest run src/components/layout/__tests__ src/components/layout/shared/__tests__ 2>&1 | tail -20
    </automated>
  </verify>
  <done>
    - 4 个 layout 配件测试通过,菜单 / 用户菜单 / 面包屑 / TabBar 全覆盖
  </done>
  <acceptance_criteria>
    - Sidebar 测试 mock menuStore + 折叠 / 路由跳转交互
    - Header 测试 authStore fixture + 用户菜单 / 通知 / 主题切换
    - Breadcrumb 测试不同 route 渲染断言
    - TabBar 测试 tabs 切换 / 关闭交互
    - 各 store 测试 beforeEach setState reset
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 3: 全量 vitest 验证 + components/layout floor bump + 基线文档 ratchet</name>
  <files>
    .coverage-fe-floors
    .planning/frontend-coverage-baseline.md
  </files>
  <read_first>
    - .coverage-fe-floors（当前 components/layout 0.0）
    - .planning/frontend-coverage-baseline.md
    - 82-CONTEXT.md D-06/D-07
  </read_first>
  <action>
    1. 跑 `cd xingran-react-frontend && npm run test:coverage` 全量测试,确认 159 存量 + 新增测试全 PASS(QUAL-01 不回归)
    2. 跑 `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors` 验证 gate 输出含 `PASS: components/layout` 行(实测 pct ≥70%)
    3. 读 components/layout 实测 pct,按 D-14 bump .coverage-fe-floors components/layout 行至 max(70.0, pct − 0.5) 并保留一位小数(向下截断)
    4. 在 .planning/frontend-coverage-baseline.md 追加 84-02a ratchet 行(同 PR,D-14)
    5. 再跑 gate 确认 components/layout PASS 且无 FAIL
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npm run test:coverage 2>&1 | tail -5
      bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep -E '^PASS: components/layout'
      bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep '^FAIL:' | wc -l
    </automated>
  </verify>
  <done>
    - components/layout floor bump 至实测 −0.5pp;基线文档 ratchet 行追加;gate PASS
  </done>
  <acceptance_criteria>
    - npm run test:coverage exit 0,Tests ≥ 159 + 新增测试数
    - gate 输出含 PASS: components/layout X.XX% >= 70.0%
    - .coverage-fe-floors components/layout 行更新
    - .planning/frontend-coverage-baseline.md 新增 84-02a ratchet 行
    - gate 总 FAIL 行数 = 0
  </acceptance_criteria>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| layout 测试 → Zustand stores | beforeEach setState reset,避免状态泄漏 |
| layout 测试 → menuStore | vi.mock menuStore fixture 数据 |
| layout 测试 → authStore | setState 注入 fixture user |
| layout 测试 → MemoryRouter | 通过 renderWithProviders 包裹 |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-84-2a-01 | Information Disclosure | useLayoutStore 状态泄漏 | mitigate | Pitfall #2: beforeEach setState({ currentLayout: "hybrid" }) |
| T-84-2a-02 | Tampering | menuStore fixture 暴露真实菜单 | mitigate | 测试用 fixture 菜单,不含真实业务数据 |
| T-84-2a-03 | Information Disclosure | useAuthStore fixture user | mitigate | 测试用假用户名,不写真实凭证 |
| T-84-2a-04 | Tampering | 路由跳转测试污染 MemoryRouter | mitigate | MemoryRouter 隔离,不影响其他测试 |
| T-84-2a-SC | Tampering | npm/pip/cargo installs | accept | 本 plan 不引入新包 |
</threat_model>

<verification>
1. `cd xingran-react-frontend && npm run test:coverage` 全量通过,Tests ≥ 159 + 新增
2. `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep '^PASS: components/layout'` 输出 ≥1 行 PASS,pct ≥70%
3. `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep '^FAIL:' | wc -l` = 0
4. `git diff .coverage-fe-floors` 显示 components/layout 行单向上调
5. `git diff .planning/frontend-coverage-baseline.md` 追加 84-02a ratchet 行
6. `grep -r 'renderWithProviders\|createApiMock' src/components/layout/ | wc -l` ≥ 3(SC-6 Reuse)
</verification>

<success_criteria>
- components/layout 16 文件 507 stmts 语句覆盖率 ≥70%(COMP-03 满足)
- 全量 vitest 0 失败,159 存量 + 新增测试全 PASS(QUAL-01 不回归)
- HybridLayout 测试覆盖路由跳转 / 菜单折叠 / 主题切换 3 类核心交互
- 各 store 测试 beforeEach setState reset(Pitfall #2 防范)
- Sidebar / Header / Breadcrumb / TabBar 配件测试覆盖
- .coverage-fe-floors components/layout 行 bump + 基线文档 ratchet 行追加(同 PR)
</success_criteria>

<output>
Create `.planning/workstreams/frontend-coverage/phases/84-p1-70/84-02a-layout-SUMMARY.md` when done
</output>