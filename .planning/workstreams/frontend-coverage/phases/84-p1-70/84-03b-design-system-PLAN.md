---
phase: 84-p1-70
plan: 03b
type: execute
wave: 3
depends_on:
  - 84-03a
files_modified:
  - xingran-react-frontend/src/design-system/tokens/__tests__/*.test.ts
  - xingran-react-frontend/src/design-system/components/__tests__/*.test.tsx
  - xingran-react-frontend/src/design-system/utils/__tests__/*.test.ts
  - xingran-react-frontend/src/design-system/animations/__tests__/*.test.ts
  - .coverage-fe-floors
  - .planning/frontend-coverage-baseline.md
autonomous: true
requirements:
  - COMP-05
  - QUAL-01
user_setup: []
must_haves:
  truths:
    - "[COMP-05] design-system 194 stmts(现 15.0%)语句覆盖率 ≥70%(tokens / components / utils / animations 四子目录各自 family 聚合)"
    - "[D-15] design-system 顶层 floor bump 至 ≥69.5% + components 聚合行 floor bump 至白名单外实测 ≥70% 值(终点目标 69.5 一位小数)"
    - "[D-06] design-system 与 hooks/store/services 同级顶层行,不走 components 二级拆分"
    - "[QUAL-01] 159 存量测试不回归 + 新增测试通过"
  artifacts:
    - path: xingran-react-frontend/src/design-system/tokens/__tests__/colors.test.ts
      provides: colors token 静态断言(D-12 纯数据)
    - path: xingran-react-frontend/src/design-system/tokens/__tests__/typography.test.ts
      provides: typography token 静态断言
    - path: xingran-react-frontend/src/design-system/tokens/__tests__/echartsTheme.test.ts
      provides: echarts theme token 静态断言
    - path: xingran-react-frontend/src/design-system/components/__tests__/AntdThemeBridge.test.tsx
      provides: 主题桥接(ConfigProvider 注入 + 主题切换断言)
    - path: xingran-react-frontend/src/design-system/components/__tests__/ThemeProvider.test.tsx
      provides: ThemeProvider 渲染 + token 注入
    - path: xingran-react-frontend/src/design-system/components/__tests__/LayoutProvider.test.tsx
      provides: LayoutProvider
    - path: xingran-react-frontend/src/design-system/components/__tests__/DensitySwitcher.test.tsx
      provides: 密度切换器 fireEvent + useSettingsStore state 变化
    - path: xingran-react-frontend/src/design-system/components/__tests__/LayoutSwitcher.test.tsx
      provides: 布局切换器
    - path: xingran-react-frontend/src/design-system/components/__tests__/PageTitle.test.tsx
      provides: 页面标题渲染
    - path: xingran-react-frontend/src/design-system/components/__tests__/SettingsShell.test.tsx
      provides: 设置面板壳层
    - path: xingran-react-frontend/src/design-system/utils/__tests__/*.test.ts
      provides: design-system utils 工具函数测试
    - path: xingran-react-frontend/src/design-system/animations/__tests__/*.test.ts
      provides: animations 静态常量断言
    - path: .coverage-fe-floors
      provides: design-system 行 bump 至 ≥69.5% + components 聚合行 bump 至 ≥69.5%(D-15 wave 3 末尾终点)
    - path: .planning/frontend-coverage-baseline.md
      provides: 84-03b ratchet 行追加 + 84 phase 收口行
  key_links:
    - from: xingran-react-frontend/src/design-system/components/__tests__/AntdThemeBridge.test.tsx
      to: xingran-react-frontend/src/design-system/tokens
      via: 显式包 <ConfigProvider theme={{...}}>{ui}</ConfigProvider> 而非依赖默认(避免 Pitfall #8 配置漂移)
    - from: xingran-react-frontend/src/design-system/components/__tests__/DensitySwitcher.test.tsx
      to: xingran-react-frontend/src/store/settingsStore.ts
      via: useSettingsStore.setState({ density: "compact" }) beforeEach reset
    - from: xingran-react-frontend/src/design-system/tokens/__tests__/echartsTheme.test.ts
      to: xingran-react-frontend/src/design-system/tokens/echartsTheme.ts
      via: 静态常量值断言
---

<objective>
落地 wave 3 末尾两步:(1)`design-system` 194 stmts(现 15.0%)语句覆盖率拉升至 ≥70% —— tokens/(colors/typography/echartsTheme 静态常量断言,D-12 纯数据)+ components/(AntdThemeBridge / ThemeProvider / LayoutProvider / DensitySwitcher / LayoutSwitcher / PageTitle / SettingsShell 7 组件,ConfigProvider 显式注入避免 Pitfall #8 配置漂移)+ utils/(工具函数)+ animations/(keyframes / transitions 静态常量);(2)按 D-15 收口 bump `components` 聚合行 floor 至白名单外实测 ≥70% 值(终点目标 70.0 - 0.5 = 69.5 一位小数)+ `design-system` 顶层行同步 bump 至 69.5。design-system 顶层 floor 行确认 gate 走顶层 seg[1] 兜底(D-06)。同 PR 两个 ratchet 行(design-system bump + components 聚合行 bump)+ 基线文档追加 84-03b 行 + 84 phase 收口行(P0/基线 3.67% → P1 收口 70% 全链路 ratchet 历史)。

Purpose: COMP-05 顶层 design-system 收口 + D-15 wave 3 末尾 components 聚合行 bump 至 69.5 是 P1 phase 的终点验收标志——此时白名单外 components 五个组件组 + design-system 全清,全局 ≥70% 由"白名单外所有目录 per-dir ≥70%"数学保证(82 D-05 / ROADMAP 推导)。

Output: design-system 多个 family 测试 + 两个 floor bump(components 聚合行 + design-system)+ 基线文档 84-03b 行 + 84 phase 收口行。
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
@xingran-react-frontend/src/design-system/
@xingran-react-frontend/src/store/settingsStore.ts
@xingran-react-frontend/src/test/utils/renderWithProviders.tsx
@xingran-react-frontend/src/test/utils/createApiMock.ts
@.coverage-fe-floors
@.planning/frontend-coverage-baseline.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: design-system tokens + animations 静态常量测试(D-12 纯数据)</name>
  <files>
    xingran-react-frontend/src/design-system/tokens/__tests__/colors.test.ts
    xingran-react-frontend/src/design-system/tokens/__tests__/typography.test.ts
    xingran-react-frontend/src/design-system/tokens/__tests__/echartsTheme.test.ts
    xingran-react-frontend/src/design-system/animations/__tests__/keyframes.test.ts
    xingran-react-frontend/src/design-system/animations/__tests__/transitions.test.ts
  </files>
  <read_first>
    - xingran-react-frontend/src/design-system/tokens/colors.ts
    - xingran-react-frontend/src/design-system/tokens/typography.ts
    - xingran-react-frontend/src/design-system/tokens/echartsTheme.ts
    - xingran-react-frontend/src/design-system/animations/keyframes.ts
    - xingran-react-frontend/src/design-system/animations/transitions.ts
  </read_first>
  <action>
    1. 创建 `tokens/__tests__/colors.test.ts` —— 静态常量断言(D-12):
       - PRIMARY_COLOR / SUCCESS_COLOR / WARNING_COLOR / ERROR_COLOR / NEUTRAL_PALETTE 等常量值断言
       - assert 导出键数量 + 各色值符合 hex/rgb 格式
    2. 创建 `tokens/__tests__/typography.test.ts` —— 字体令牌:
       - FONT_FAMILY / FONT_SIZE_SCALE / FONT_WEIGHT / LINE_HEIGHT 常量值断言
       - assert 各 scale 步长一致(如 12/14/16/20/24)
    3. 创建 `tokens/__tests__/echartsTheme.test.ts` —— ECharts 主题:
       - DEFAULT_THEME / DARK_THEME / COMPACT_THEME 常量值断言
       - assert theme 对象含 color/textStyle/title/legend 等 ECharts 字段
    4. 创建 `animations/__tests__/keyframes.test.ts` —— 关键帧:
       - FADE_IN / FADE_OUT / SLIDE_IN / SCALE_IN 等 keyframes 字符串断言
       - assert 包含 @keyframes 关键字 + 0%/100% 标记
    5. 创建 `animations/__tests__/transitions.test.ts` —— 过渡:
       - FAST_TRANSITION / NORMAL_TRANSITION / SLOW_TRANSITION 常量值断言
       - assert 包含 transition property + duration + easing
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npx vitest run src/design-system/tokens/__tests__ src/design-system/animations/__tests__ 2>&1 | tail -20
    </automated>
  </verify>
  <done>
    - tokens + animations 共 5 个测试通过,纯常量断言覆盖率 100%
  </done>
  <acceptance_criteria>
    - 5 个测试文件覆盖 colors / typography / echartsTheme / keyframes / transitions
    - 各文件 assert 导出完整性 + 关键值
    - D-12 纯数据组件单一静态断言模式
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 2: design-system components 测试(AntdThemeBridge + ThemeProvider + LayoutProvider + DensitySwitcher + LayoutSwitcher + PageTitle + SettingsShell)</name>
  <files>
    xingran-react-frontend/src/design-system/components/__tests__/AntdThemeBridge.test.tsx
    - xingran-react-frontend/src/design-system/components/__tests__/ThemeProvider.test.tsx
    - xingran-react-frontend/src/design-system/components/__tests__/LayoutProvider.test.tsx
    - xingran-react-frontend/src/design-system/components/__tests__/DensitySwitcher.test.tsx
    - xingran-react-frontend/src/design-system/components/__tests__/LayoutSwitcher.test.tsx
    - xingran-react-frontend/src/design-system/components/__tests__/PageTitle.test.tsx
    - xingran-react-frontend/src/design-system/components/__tests__/SettingsShell.test.tsx
    - xingran-react-frontend/src/design-system/utils/__tests__/index.test.ts
  </files>
  <read_first>
    - xingran-react-frontend/src/design-system/components/AntdThemeBridge.tsx
    - xingran-react-frontend/src/design-system/components/ThemeProvider.tsx
    - xingran-react-frontend/src/design-system/components/LayoutProvider.tsx
    - xingran-react-frontend/src/design-system/components/DensitySwitcher.tsx
    - xingran-react-frontend/src/design-system/components/LayoutSwitcher.tsx
    - xingran-react-frontend/src/design-system/components/PageTitle.tsx
    - xingran-react-frontend/src/design-system/components/SettingsShell.tsx
    - xingran-react-frontend/src/design-system/utils/index.ts（如含工具）
    - xingran-react-frontend/src/store/settingsStore.ts
  </read_first>
  <action>
    1. 创建 `components/__tests__/AntdThemeBridge.test.tsx` —— 主题桥接(Pitfall #8):
       - 显式包 `<ConfigProvider theme={{...}}>{ui}</ConfigProvider>` 而非依赖默认(避免 token undefined)
       - renderWithProviders(<AntdThemeBridge><div>content</div></AntdThemeBridge>) + 渲染断言
       - fireEvent.click 切换主题 → theme token 变化断言
    2. 创建 `components/__tests__/ThemeProvider.test.tsx` —— ThemeProvider:
       - renderWithProviders(<ThemeProvider><Child /></ThemeProvider>) + token 注入断言
       - 不同 theme props("light" / "dark")渲染差异断言
    3. 创建 `components/__tests__/LayoutProvider.test.tsx` —— LayoutProvider:
       - renderWithProviders(<LayoutProvider density="compact">{content}</LayoutProvider>) + content 渲染 + 紧凑间距断言
       - fireEvent.change density → 变化断言
    4. 创建 `components/__tests__/DensitySwitcher.test.tsx` —— 密度切换器:
       - beforeEach: useSettingsStore.setState({ density: "default" }) reset(Pitfall #2 防范)
       - renderWithProviders(<DensitySwitcher />) + 渲染 3 种密度按钮
       - fireEvent.click "compact" → useSettingsStore state 变化断言
    5. 创建 `components/__tests__/LayoutSwitcher.test.tsx` —— 布局切换器:
       - renderWithProviders(<LayoutSwitcher />) + 渲染多种布局选项
       - fireEvent.click 布局按钮 → onSelect 回调断言
    6. 创建 `components/__tests__/PageTitle.test.tsx` —— 页面标题:
       - renderWithProviders(<PageTitle title="用户管理" />) + 标题文本断言(D-12 props 组合: 不同 title)
    7. 创建 `components/__tests__/SettingsShell.test.tsx` —— 设置面板壳层:
       - renderWithProviders(<SettingsShell>{tabs}</SettingsShell>) + 渲染 tab 列表
       - fireEvent.click tab → tab 内容切换断言
    8. 创建 `utils/__tests__/index.test.ts` —— design-system utils:
       - 工具函数如 pxToRem / getThemeVar / mergeTokens 单元测试
       - 边界值 + 不同 input 输出断言
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npx vitest run src/design-system/components/__tests__ src/design-system/utils/__tests__ 2>&1 | tail -30
    </automated>
  </verify>
  <done>
    - design-system components + utils 共 8 个测试通过,每个组件 ≥70%
  </done>
  <acceptance_criteria>
    - AntdThemeBridge 显式包 ConfigProvider theme 注入(Pitfall #8 防范)
    - ThemeProvider / LayoutProvider 不同 props 渲染断言
    - DensitySwitcher / LayoutSwitcher fireEvent + state 变化
    - PageTitle 多 props 组合(D-12)
    - SettingsShell tab 切换
    - design-system utils 工具函数测试
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 3: 全量 vitest 验证 + design-system 行 bump + components 聚合行 bump(D-15 wave 3 末尾终点)</name>
  <files>
    .coverage-fe-floors
    .planning/frontend-coverage-baseline.md
  </files>
  <read_first>
    - .coverage-fe-floors（当前 design-system 0.0 + components 聚合行 4.9）
    - .planning/frontend-coverage-baseline.md（ratchet 行格式 + 84 全部 plan 行已加）
    - 82-CONTEXT.md D-06 / D-07 / D-13
    - 84-CONTEXT.md D-15（终点目标 69.5）
  </read_first>
  <action>
    1. 跑 `cd xingran-react-frontend && npm run test:coverage` 全量测试,确认 159 存量 + 所有 wave 0/1/2/3 新增测试全 PASS(QUAL-01 不回归)
    2. 跑 `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors` 验证 gate 输出:
       - PASS: design-system X.XX% >= 70.0%(实测)
       - PASS: components X.XX% >= 4.9(当前聚合行 floor,准备升至 69.5)
       - 各 subdir PASS(shared / dashboard / layout / CronSelector / captcha / operations / network / reconciliation)
    3. 按 D-15 wave 3 末尾 bump 两个 floor:
       - .coverage-fe-floors components 行(聚合行,非 shared/dashboard 等 subdir 行)从 4.9 → 69.5(一位小数,白名单外实测 ≥70% 时即可 bump 至 70.0 - 0.5)
       - .coverage-fe-floors design-system 行从 0.0 → 69.5(白名单外实测 ≥70% 时同样 bump 至 70.0 - 0.5)
       - 注:若实测 < 70%,则按 D-14 纪律 bump 至实测 − 0.5pp(向下截断一位小数);实测 ≥ 70% 才升级至 69.5
    4. 在 .planning/frontend-coverage-baseline.md 追加两行 84-03b ratchet:
       - 84-03b-components-aggregate: components 聚合行 4.9 → 69.5(终点)
       - 84-03b-design-system: design-system 顶层行 0.0 → 69.5(终点)
    5. 追加 84 phase 收口行(若前 plan 已加可合并):
       - 84 phase 收口:ratchet 终态(components 五个 subdir + design-system + components 聚合行)
       - per-dir 终值表 + commit 短 SHA
    6. 再跑 gate 确认所有 PASS 且无 FAIL
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npm run test:coverage 2>&1 | tail -5
      bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep -E '^PASS: (design-system|components )'
      bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep '^FAIL:' | wc -l
    </automated>
  </verify>
  <done>
    - design-system floor bump 至 ≥69.5%;components 聚合行 bump 至 69.5%;基线文档追加 84-03b 行 + 84 phase 收口行
  </done>
  <acceptance_criteria>
    - npm run test:coverage exit 0,Tests ≥ 159 + 所有 wave 0/1/2/3 新增测试数
    - gate 输出 PASS: design-system X.XX% >= 69.5%
    - gate 输出 PASS: components X.XX% >= 69.5%(聚合行)
    - .coverage-fe-floors 两个 row 更新(components 聚合行 + design-system)
    - .planning/frontend-coverage-baseline.md 新增 84-03b ratchet + 84 phase 收口行
    - gate 总 FAIL 行数 = 0
    - Phase 84 全部 SC 满足(SC-1..SC-9 from 84-VALIDATION.md)
  </acceptance_criteria>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| design-system tokens → 静态常量 | 无运行时副作用,纯断言 |
| design-system components → ConfigProvider | 显式注入 theme token(避免 Pitfall #8 token undefined) |
| design-system components → settingsStore | beforeEach useSettingsStore.setState({ density: "default" }) reset |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-84-3b-01 | Information Disclosure | ConfigProvider token 注入 undefined | mitigate | Pitfall #8: 显式包 <ConfigProvider theme={...}>{ui}</ConfigProvider> |
| T-84-3b-02 | Tampering | useSettingsStore 状态泄漏 | mitigate | Pitfall #2: beforeEach useSettingsStore.setState({ density: "default" }) |
| T-84-3b-03 | Tampering | design-system 聚合行 bump 时机不当(实测 < 70%) | mitigate | D-15 锁"白名单外实测 ≥70% 时才 bump 至 69.5";否则按 D-14 实测 − 0.5pp |
| T-84-3b-04 | Information Disclosure | echartsTheme token fixture 暴露真实主题配置 | mitigate | 测试用 fixture token,不写真实业务 |
| T-84-3b-SC | Tampering | npm/pip/cargo installs | accept | 本 plan 不引入新包 |
</threat_model>

<verification>
1. `cd xingran-react-frontend && npm run test:coverage` 全量通过,Tests ≥ 159 + 所有 wave 0/1/2/3 新增
2. `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep '^PASS: design-system'` 输出 PASS,pct ≥69.5%
3. `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep '^PASS: components '` 输出 PASS(components 聚合行,pct ≥69.5%)
4. `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors | grep '^FAIL:' | wc -l` = 0
5. `git diff .coverage-fe-floors` 显示两个 row 单向上调(components 聚合行 + design-system)
6. `git diff .planning/frontend-coverage-baseline.md` 追加 84-03b ratchet + 84 phase 收口行
7. `grep -r 'renderWithProviders\|createApiMock' src/design-system/ | wc -l` ≥ 3(SC-6 Reuse)
8. 全量 vitest 退出 0,QUAL-01 159 存量测试 + 所有 P1 新增测试不回归
</verification>

<success_criteria>
- design-system 194 stmts 顶层行 ≥70%(COMP-05 顶层行满足)
- components 聚合行 ≥69.5%(D-15 wave 3 末尾终点,白名单外实测 ≥70%)
- 全量 vitest 0 失败,159 存量 + 所有 P1 新增测试全 PASS(QUAL-01 不回归)
- design-system tokens + animations 纯静态常量 100% 覆盖(D-12 纯数据)
- design-system components 显式 ConfigProvider 注入(Pitfall #8 防范)
- design-system components 测试 useSettingsStore reset(Pitfall #2 防范)
- Phase 84 全部 SC 满足(SC-1..SC-9 from 84-VALIDATION.md):components 五个组件组 + design-system 顶层 + components 聚合行 + harness 沉淀 + QUAL-01 不回归 + ratchet 单调
- .coverage-fe-floors 两个 row bump + 基线文档两个 ratchet 行 + 84 phase 收口行追加(同 PR)
</success_criteria>

<output>
Create `.planning/workstreams/frontend-coverage/phases/84-p1-70/84-03b-design-system-SUMMARY.md` when done
</output>