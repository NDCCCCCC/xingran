---
phase: 83-p0-70
plan: 04
type: execute
wave: 3
depends_on:
  - 83-02
  - 83-03
files_modified:
  - xingran-react-frontend/src/hooks/useTableManager.test.tsx
  - xingran-react-frontend/src/hooks/useColumnConfig.test.tsx
  - xingran-react-frontend/src/hooks/useWidgetData.test.tsx
  - xingran-react-frontend/src/hooks/useDataHooks.test.tsx
  - xingran-react-frontend/src/hooks/useTableHooks.test.tsx
  - xingran-react-frontend/src/hooks/useNetworkHooks.test.tsx
  - xingran-react-frontend/src/hooks/useUtilityHooks.test.tsx
  - xingran-react-frontend/src/store/authStore.test.ts
  - xingran-react-frontend/src/store/tabsStore.test.ts
  - xingran-react-frontend/src/store/menuStore.test.ts
  - xingran-react-frontend/src/store/dashboardStore.test.ts
  - xingran-react-frontend/src/store/settingsStore.test.ts
  - xingran-react-frontend/src/store/layoutStore.test.ts
  - xingran-react-frontend/src/store/noticeStore.test.ts
  - xingran-react-frontend/src/store/themeStore.test.ts
  - xingran-react-frontend/src/store/visualizationStore.test.ts
  - .coverage-fe-floors
  - .planning/frontend-coverage-baseline.md
autonomous: true
requirements:
  - INFRA-03
  - INFRA-04
  - QUAL-01
user_setup: []
must_haves:
  truths:
    - "[D-10][D-12] hooks 目录 27 文件 1050 stmts 的语句覆盖率从 8.10% 提升到 ≥70%"
    - "[D-10][D-12] store 目录 9 文件 589 stmts 的语句覆盖率从 4.75% 提升到 ≥70%"
    - 各 Zustand store 测试使用 setState(initialState) reset 模式，按需注入不包 Provider
    - "[D-11] hooks/store 目录 floor 被 bump 到实测 −0.5pp 并追加基线文档"
  artifacts:
    - path: xingran-react-frontend/src/hooks/*.test.tsx
      provides: hooks 层测试文件
    - path: xingran-react-frontend/src/store/*.test.ts
      provides: store 层测试文件
    - path: .coverage-fe-floors
      provides: hooks/store floor 更新
  key_links:
    - from: store tests
      to: src/store/*.ts
      via: beforeEach 调用 useXxxStore.setState(initialState)
    - from: hooks tests
      to: src/hooks/*.ts
      via: renderHook + MemoryRouter wrapper
---

<objective>
将 hooks 目录（27 文件 1050 stmts）语句覆盖率从 8.10% 提升到 ≥70%，将 store 目录（9 文件 589 stmts）从 4.75% 提升到 ≥70%；Zustand store 测试采用 setState reset 模式，hooks 测试使用 renderHook + MemoryRouter；并在同一 commit 中 bump hooks 与 store floor 及基线文档。

Purpose: hooks/store 是组件层状态逻辑的核心，补齐后 P1 组件测试可直接复用这些 mock 与 reset 模式。
Output: 15+ 个测试文件、hooks/store floor 提升、基线文档追加行。
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/workstreams/frontend-coverage/ROADMAP.md
@.planning/workstreams/frontend-coverage/REQUIREMENTS.md
@.planning/workstreams/frontend-coverage/phases/83-p0-70/83-CONTEXT.md
@.planning/workstreams/frontend-coverage/phases/83-p0-70/83-RESEARCH.md
@.planning/workstreams/frontend-coverage/phases/83-p0-70/83-PATTERNS.md
@xingran-react-frontend/src/hooks/usePagination.test.tsx
@xingran-react-frontend/src/hooks/useServerSort.test.tsx
@xingran-react-frontend/src/hooks/usePersistedState.test.ts
@xingran-react-frontend/src/hooks/useTableManager.ts
@xingran-react-frontend/src/hooks/useColumnConfig.ts
@xingran-react-frontend/src/hooks/useWidgetData.ts
@xingran-react-frontend/src/hooks/usePersistedState.ts
@xingran-react-frontend/src/store/authStore.ts
@xingran-react-frontend/src/store/tabsStore.ts
@xingran-react-frontend/src/store/menuStore.ts
@xingran-react-frontend/src/store/dashboardStore.ts
@xingran-react-frontend/src/store/settingsStore.ts
@xingran-react-frontend/src/store/layoutStore.ts
@xingran-react-frontend/src/store/noticeStore.ts
@xingran-react-frontend/src/store/themeStore.ts
@xingran-react-frontend/src/store/visualizationStore.ts
@xingran-react-frontend/src/test/setup.ts
@.coverage-fe-floors
@.planning/frontend-coverage-baseline.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: hooks 层测试补齐</name>
  <files>
    xingran-react-frontend/src/hooks/useTableManager.test.tsx
    xingran-react-frontend/src/hooks/useColumnConfig.test.tsx
    xingran-react-frontend/src/hooks/useWidgetData.test.tsx
    xingran-react-frontend/src/hooks/useDataHooks.test.tsx
    xingran-react-frontend/src/hooks/useTableHooks.test.tsx
    xingran-react-frontend/src/hooks/useNetworkHooks.test.tsx
    xingran-react-frontend/src/hooks/useUtilityHooks.test.tsx
  </files>
  <read_first>
    - xingran-react-frontend/src/hooks/usePagination.test.tsx（renderHook + MemoryRouter 模式）
    - xingran-react-frontend/src/hooks/useServerSort.test.tsx（sorter meta mock 模式）
    - xingran-react-frontend/src/hooks/useTableManager.ts
    - xingran-react-frontend/src/hooks/useColumnConfig.ts
    - xingran-react-frontend/src/hooks/useWidgetData.ts
    - xingran-react-frontend/src/hooks/usePersistedState.ts
    - xingran-react-frontend/src/lib/api.ts（了解 API wrapper 签名）
  </read_first>
  <action>
    1. 使用 Glob 列出 src/hooks 下所有未测试的 *.ts(x) 文件，按 coverage statement 数降序排列。实际未测 hooks 及语句数：useTableManager 123、useColumnConfig 122、useWidgetData 92、useRealtimeUpdates 82、useWallDrawing 79、useWebSocket 75、useWidgetPolling 60、useTabSync 59、useImageUpload 37、useRPAProgress 37、useADConfigs 28、useReconciliationWebSocket 26、useCaptcha 21、useNetworkStatus 19、useDashboard 17、useSidebarDeptFilter 10、useWindowSize 8、useExceptionList 7、useDeptTree/useDict/useRoleList 各 6、useTableSettings 5、useAliasByLocation 4、useTableQuery 2。
    2. 对高扇出 hooks 单独建测试：useTableManager.test.tsx、useColumnConfig.test.tsx、useWidgetData.test.tsx，使用 renderHook + MemoryRouter wrapper；mock @/lib/api 与相关 API 模块；覆盖正常加载、分页/排序/列配置持久化、错误分支、参数变化时重新请求。
    3. 对其余 hooks 按功能分组创建组合测试：
       - useDataHooks.test.tsx：useADConfigs、useAliasByLocation、useDashboard、useDeptTree、useDict、useExceptionList、useReconciliationWebSocket、useWidgetPolling。
       - useTableHooks.test.tsx：useTableQuery、useTableSettings（与 useTableManager/useColumnConfig 互补）。
       - useNetworkHooks.test.tsx：useNetworkStatus、useRealtimeUpdates、useRPAProgress、useWebSocket。
       - useUtilityHooks.test.tsx：useCaptcha、useImageUpload、useRoleList、useSidebarDeptFilter、useTabSync、useWallDrawing、useWindowSize。
    4. 注意 useEffect 依赖稳定性：测试中传给 hook 的对象/数组参数使用 useMemo 包裹或保持引用稳定，避免无限循环。
    5. 测试中清理 sessionStorage/localStorage。
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npx vitest run src/hooks --reporter=verbose
    </automated>
  </verify>
  <done>
    - hooks 目录新增测试全部通过。
  </done>
  <acceptance_criteria>
    - src/hooks 下新增 *.test.tsx 文件覆盖 useTableManager、useColumnConfig、useWidgetData 及剩余未测试 hooks（分组文件：useDataHooks、useTableHooks、useNetworkHooks、useUtilityHooks）。
    - npx vitest run src/hooks 输出 Tests ... passed，exit 0。
    - 159 存量测试不回归。
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 2: store 层测试补齐</name>
  <files>
    xingran-react-frontend/src/store/authStore.test.ts
    xingran-react-frontend/src/store/tabsStore.test.ts
    xingran-react-frontend/src/store/menuStore.test.ts
    xingran-react-frontend/src/store/dashboardStore.test.ts
    xingran-react-frontend/src/store/settingsStore.test.ts
    xingran-react-frontend/src/store/layoutStore.test.ts
    xingran-react-frontend/src/store/noticeStore.test.ts
    xingran-react-frontend/src/store/themeStore.test.ts
    xingran-react-frontend/src/store/visualizationStore.test.ts
  </files>
  <read_first>
    - xingran-react-frontend/src/store/authStore.ts
    - xingran-react-frontend/src/store/tabsStore.ts
    - xingran-react-frontend/src/store/menuStore.ts
    - xingran-react-frontend/src/store/dashboardStore.ts
    - xingran-react-frontend/src/store/settingsStore.ts
    - xingran-react-frontend/src/store/layoutStore.ts
    - xingran-react-frontend/src/store/noticeStore.ts
    - xingran-react-frontend/src/store/themeStore.ts
    - xingran-react-frontend/src/store/visualizationStore.ts
    - xingran-react-frontend/src/hooks/usePersistedState.test.ts（storage cleanup 模式）
  </read_first>
  <action>
    1. 为每个 store 创建同目录 *.test.ts。beforeEach 中调用 useXxxStore.setState(initialState) 重置状态，并视需要 localStorage.clear() / sessionStorage.clear()。
    2. authStore.test.ts：mock @/lib/api 的 post/get，覆盖 login/logout/initializeAuth/fetchUserMenus/refreshToken/getTokenManager；使用 vi.useFakeTimers() 测试 TokenManager 集成；覆盖 401/无 token/刷新失败分支。
    3. tabsStore.test.ts：覆盖 addTab/closeTab/closeOthers/closeAll/switchTab/restoreFromStorage/persist、不同 tab 类型、重复添加、历史栈。
    4. menuStore.test.ts：mock @/lib/api，覆盖 fetchMenus/setMenus/clearMenus/filterMenus/expandCollapse/menuMap。
    5. dashboardStore.test.ts：覆盖 addWidget/removeWidget/updateWidget/moveWidget/loadPreset/reset/persist。
    6. settingsStore.test.ts：覆盖 setTheme/setLayout/setDensity/resetSettings/persist，验证 localStorage 写入。
    7. layoutStore.test.ts：覆盖 toggleSidebar/setCollapsed/setTheme/persist。
    8. noticeStore.test.ts：mock @/lib/api，覆盖 fetchNotices/markRead/markAllRead/clear。
    9. themeStore.test.ts：覆盖 setThemeMode/setColorPrimary/applyTheme/persist。
    10. visualizationStore.test.ts：覆盖 setData/updateData/clear/selection。
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npx vitest run src/store --reporter=verbose
    </automated>
  </verify>
  <done>
    - store 目录新增测试全部通过。
  </done>
  <acceptance_criteria>
    - src/store 下 9 个 store 文件每个都有对应的 *.test.ts。
    - npx vitest run src/store 输出 Tests ... passed，exit 0。
    - authStore 与 tabsStore 测试使用 vi.useFakeTimers() 与 setState reset。
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 3: 覆盖率验证与 ratchet bump</name>
  <files>
    .coverage-fe-floors
    .planning/frontend-coverage-baseline.md
  </files>
  <read_first>
    - .coverage-fe-floors（确认 hooks/store 当前 floor）
    - .planning/frontend-coverage-baseline.md（确认追加行格式）
    - .github/scripts/check-frontend-coverage.sh
  </read_first>
  <action>
    1. 运行 cd xingran-react-frontend && npm run test:coverage。
    2. 运行 bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors。
    3. 读取 hooks 与 store 实测 pct，将 .coverage-fe-floors 中 hooks 行从 7.6、store 行从 4.3 分别 bump 至 max(70.0, pct − 0.5) 并保留一位小数。
    4. 在 .planning/frontend-coverage-baseline.md 追加 ratchet 行：日期、phase 83-04、weighted_avg、total_stmts、total_covered、0pct_pkg_count、当前 commit 短 SHA、ratchet_from、ratchet_to。
    5. 重新运行 gate 脚本确认 hooks/store PASS 且 global PASS。
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npm run test:coverage 全量通过；
      bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors 输出 PASS: hooks ... >= 70.0% 与 PASS: store ... >= 70.0%。
    </automated>
  </verify>
  <done>
    - hooks/store floor 更新，基线文档追加，gate 通过。
  </done>
  <acceptance_criteria>
    - .coverage-fe-floors 中 hooks 行 ≥70.0 且 store 行 ≥70.0。
    - .planning/frontend-coverage-baseline.md 新增 ratchet 行包含 83-04、commit SHA、hooks/store ratchet 值。
    - gate 脚本输出 hooks PASS 与 store PASS。
    - npm run test:coverage 全量通过。
  </acceptance_criteria>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| store 测试 → localStorage/sessionStorage | jsdom storage 隔离；测试使用 fake 数据 |
| authStore 测试 → @/lib/api | vi.mock 拦截登录/刷新/菜单端点 |
| hooks 测试 → 真实路由 | 使用 MemoryRouter，不触发浏览器导航 |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-83-04-01 | Information Disclosure | authStore 测试 | mitigate | 使用假用户名/密码/token，不引入真实凭证 |
| T-83-04-02 | Denial of Service | fake timers | mitigate | TokenManager 与自动刷新测试统一使用 vi.useFakeTimers() |
| T-83-04-03 | Tampering | store persist | mitigate | 验证 localStorage 写入值与读取回退行为 |
| T-83-04-04 | Elevation of Privilege | 401 处理 | mitigate | authStore 测试覆盖刷新失败清理 token 并跳转登录 |
| T-83-04-SC | Tampering | npm/pip/cargo installs | accept | 本 plan 不引入新包；无安装步骤 |
</threat_model>

<verification>
1. cd xingran-react-frontend && npm run test:coverage 全量通过，159 存量测试不回归。
2. bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors 通过，hooks 与 store 行均 PASS。
3. git diff 检查：仅新增测试文件、.coverage-fe-floors hooks/store 行、.planning/frontend-coverage-baseline.md 追加行。
4. 抽样检查 gate 输出中 hooks pct ≥70.00% 且 store pct ≥70.00%。
</verification>

<success_criteria>
- hooks 目录 statements 覆盖率 ≥70%（gate 输出 PASS）。
- store 目录 statements 覆盖率 ≥70%（gate 输出 PASS）。
- 所有 Zustand store 有对应测试文件并使用 setState reset 模式。
- .coverage-fe-floors 中 hooks 与 store floor bump 至 ≥70.0，基线文档追加同一 commit。
- 全量 vitest 0 失败，159 存量测试不回归。
</success_criteria>

<output>
Create `.planning/workstreams/frontend-coverage/phases/83-p0-70/83-04-hooks-store-coverage-SUMMARY.md` when done
</output>
