---
phase: 83-p0-70
plan: 05
type: execute
wave: 4
depends_on:
  - 83-04
files_modified:
  - xingran-react-frontend/src/services/encryptionConfig.test.ts
  - xingran-react-frontend/src/services/captcha.test.ts
  - xingran-react-frontend/src/services/dashboardService.test.ts
  - xingran-react-frontend/src/services/configService.test.ts
  - xingran-react-frontend/src/services/cache/MenuCache.test.ts
  - xingran-react-frontend/src/services/cache/TTLMenuCache.test.ts
  - xingran-react-frontend/src/services/operations/buildings.test.ts
  - xingran-react-frontend/src/services/operations/floors.test.ts
  - xingran-react-frontend/src/services/operations/info-points.test.ts
  - xingran-react-frontend/src/services/operations/room-devices.test.ts
  - xingran-react-frontend/src/services/operations/server-rooms.test.ts
  - xingran-react-frontend/src/services/operations/workstations.test.ts
  - xingran-react-frontend/src/services/operations/dedicated-lines.test.ts
  - xingran-react-frontend/src/constants/storage.test.ts
  - xingran-react-frontend/src/constants/pageTitles.test.ts
  - xingran-react-frontend/src/constants/routes.test.ts
  - xingran-react-frontend/src/constants/upload.test.ts
  - xingran-react-frontend/src/constants/buttonStyles.test.tsx
  - xingran-react-frontend/src/types/config.test.ts
  - xingran-react-frontend/src/types/dashboard.test.ts
  - xingran-react-frontend/src/types/notice.test.ts
  - xingran-react-frontend/src/types/common.test.ts
  - xingran-react-frontend/src/types/widgets/helpers.test.ts
  - xingran-react-frontend/src/types/operations.test.ts
  - xingran-react-frontend/src/router/routeConfigManager.test.ts
  - xingran-react-frontend/src/router/routeGenerator.test.ts
  - xingran-react-frontend/src/router/componentLoader.test.tsx
  - xingran-react-frontend/src/router/DynamicRoutes.test.tsx
  - xingran-react-frontend/src/router/RouteGuard.test.tsx
  - xingran-react-frontend/src/test/utils/renderWithProviders.tsx
  - xingran-react-frontend/src/test/utils/createApiMock.ts
  - xingran-react-frontend/src/test/utils/mockAntdMessage.ts
  - xingran-react-frontend/src/test/utils/harness.example.test.ts
  - .coverage-fe-floors
  - .planning/frontend-coverage-baseline.md
autonomous: true
requirements:
  - INFRA-05
  - QUAL-01
  - QUAL-03
user_setup: []
must_haves:
  truths:
    - "[D-12] services、router、constants、types 四个目录 statements 覆盖率均 ≥70%"
    - "[D-04][D-05][D-06] src/test/utils/ 下沉淀 renderWithProviders / createApiMock / mockAntdMessage 三件套，且 stores 参数支持按需注入并自动 reset"
    - 至少有一个 P0 测试导入使用 harness 三件套中的至少一个，并验证 store 注入
    - "[D-11] services/router/constants/types 四个目录 floor 被 bump 到实测 −0.5pp 并追加基线文档"
  artifacts:
    - path: xingran-react-frontend/src/services/*.test.ts
      provides: services 层测试
    - path: xingran-react-frontend/src/router/*.test.ts(x)
      provides: router 层测试
    - path: xingran-react-frontend/src/constants/*.test.ts
      provides: constants 层测试
    - path: xingran-react-frontend/src/types/*.test.ts
      provides: types 层测试
    - path: xingran-react-frontend/src/test/utils/
      provides: 公共测试 harness
    - path: .coverage-fe-floors
      provides: services/router/constants/types floor 更新
  key_links:
    - from: src/test/utils/renderWithProviders.tsx
      to: src/router/*.test.tsx
      via: MemoryRouter + ConfigProvider wrapper，可选 stores reset
    - from: src/test/utils/createApiMock.ts
      to: src/services/*.test.ts
      via: vi.mock('@/lib/api') 端点工厂
---

<objective>
将 services（238 stmts）、router（272 stmts）、constants（84 stmts）、types（32 stmts）四个目录语句覆盖率均提升至 ≥70%；在 src/test/utils/ 沉淀公共测试 harness（renderWithProviders / createApiMock / mockAntdMessage）并落实 D-05 的 stores 按需注入 + 自动 reset；确保至少有一个 P0 测试使用 harness 示例并验证 store 注入；在同一 commit 中 bump 四个目录 floor 及基线文档，完成 Phase 83 P0 基建层全清。

Purpose: 补齐 P0 最后四个目录，同时为 P1/P2 组件与页面测试提供可复用 harness。
Output: 25+ 个测试文件、harness 三件套 + 示例、四个目录 floor 提升、基线文档追加行。
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
@xingran-react-frontend/src/services/encryptionConfig.ts
@xingran-react-frontend/src/services/captcha.ts
@xingran-react-frontend/src/services/dashboardService.ts
@xingran-react-frontend/src/services/configService.ts
@xingran-react-frontend/src/services/cache/MenuCache.ts
@xingran-react-frontend/src/services/cache/TTLMenuCache.ts
@xingran-react-frontend/src/services/operations/buildings.ts
@xingran-react-frontend/src/services/operations/floors.ts
@xingran-react-frontend/src/services/operations/info-points.ts
@xingran-react-frontend/src/services/operations/room-devices.ts
@xingran-react-frontend/src/services/operations/server-rooms.ts
@xingran-react-frontend/src/services/operations/workstations.ts
@xingran-react-frontend/src/services/operations/dedicated-lines.ts
@xingran-react-frontend/src/constants/status.test.ts
@xingran-react-frontend/src/constants/storage.ts
@xingran-react-frontend/src/constants/pageTitles.ts
@xingran-react-frontend/src/constants/routes.ts
@xingran-react-frontend/src/constants/upload.ts
@xingran-react-frontend/src/constants/buttonStyles.tsx
@xingran-react-frontend/src/types/config.ts
@xingran-react-frontend/src/types/dashboard.ts
@xingran-react-frontend/src/types/notice.ts
@xingran-react-frontend/src/types/common.ts
@xingran-react-frontend/src/types/widgets/helpers.ts
@xingran-react-frontend/src/types/operations.ts
@xingran-react-frontend/src/router/routeConfigManager.ts
@xingran-react-frontend/src/router/routeGenerator.ts
@xingran-react-frontend/src/router/componentLoader.tsx
@xingran-react-frontend/src/router/DynamicRoutes.tsx
@xingran-react-frontend/src/router/RouteGuard.tsx
@xingran-react-frontend/src/test/setup.ts
@.coverage-fe-floors
@.planning/frontend-coverage-baseline.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: services、constants、types 层测试补齐</name>
  <files>
    xingran-react-frontend/src/services/encryptionConfig.test.ts
    xingran-react-frontend/src/services/captcha.test.ts
    xingran-react-frontend/src/services/dashboardService.test.ts
    xingran-react-frontend/src/services/configService.test.ts
    xingran-react-frontend/src/services/cache/MenuCache.test.ts
    xingran-react-frontend/src/services/cache/TTLMenuCache.test.ts
    xingran-react-frontend/src/services/operations/buildings.test.ts
    xingran-react-frontend/src/services/operations/floors.test.ts
    xingran-react-frontend/src/services/operations/info-points.test.ts
    xingran-react-frontend/src/services/operations/room-devices.test.ts
    xingran-react-frontend/src/services/operations/server-rooms.test.ts
    xingran-react-frontend/src/services/operations/workstations.test.ts
    xingran-react-frontend/src/services/operations/dedicated-lines.test.ts
    xingran-react-frontend/src/constants/storage.test.ts
    xingran-react-frontend/src/constants/pageTitles.test.ts
    xingran-react-frontend/src/constants/routes.test.ts
    xingran-react-frontend/src/constants/upload.test.ts
    xingran-react-frontend/src/constants/buttonStyles.test.tsx
    xingran-react-frontend/src/types/config.test.ts
    xingran-react-frontend/src/types/dashboard.test.ts
    xingran-react-frontend/src/types/notice.test.ts
    xingran-react-frontend/src/types/common.test.ts
    xingran-react-frontend/src/types/widgets/helpers.test.ts
    xingran-react-frontend/src/types/operations.test.ts
  </files>
  <read_first>
    - xingran-react-frontend/src/services/encryptionConfig.ts
    - xingran-react-frontend/src/services/captcha.ts
    - xingran-react-frontend/src/services/dashboardService.ts
    - xingran-react-frontend/src/services/configService.ts
    - xingran-react-frontend/src/services/cache/MenuCache.ts
    - xingran-react-frontend/src/services/cache/TTLMenuCache.ts
    - xingran-react-frontend/src/services/operations/buildings.ts
    - xingran-react-frontend/src/services/operations/floors.ts
    - xingran-react-frontend/src/services/operations/info-points.ts
    - xingran-react-frontend/src/services/operations/room-devices.ts
    - xingran-react-frontend/src/services/operations/server-rooms.ts
    - xingran-react-frontend/src/services/operations/workstations.ts
    - xingran-react-frontend/src/services/operations/dedicated-lines.ts
    - xingran-react-frontend/src/constants/status.test.ts（静态断言模板）
    - xingran-react-frontend/src/constants/storage.ts
    - xingran-react-frontend/src/constants/pageTitles.ts
    - xingran-react-frontend/src/constants/routes.ts
    - xingran-react-frontend/src/constants/upload.ts
    - xingran-react-frontend/src/constants/buttonStyles.tsx
    - xingran-react-frontend/src/types/config.ts
    - xingran-react-frontend/src/types/dashboard.ts
    - xingran-react-frontend/src/types/notice.ts
    - xingran-react-frontend/src/types/common.ts
    - xingran-react-frontend/src/types/widgets/helpers.ts
    - xingran-react-frontend/src/types/operations.ts
  </read_first>
  <action>
    1. services 测试：
       - encryptionConfig.test.ts：mock @/lib/api，覆盖 getEncryptionConfig/getCachedEncryptionConfig/clearEncryptionConfigCache 的缓存命中、缓存过期、请求失败分支。
       - captcha.test.ts：覆盖验证码生成/校验辅助函数。
       - dashboardService.test.ts：mock @/lib/api，覆盖仪表盘数据获取与 Widget 数据转换。
       - configService.test.ts：覆盖配置读取与默认值。
       - cache/MenuCache.test.ts 与 TTLMenuCache.test.ts：覆盖 set/get/clear/ttl 过期。
       - operations/buildings.test.ts、floors.test.ts、info-points.test.ts、room-devices.test.ts、server-rooms.test.ts、workstations.test.ts、dedicated-lines.test.ts：对每个 operations service 模块做端点契约测试，mock @/lib/api。
    2. constants 测试：
       - storage.test.ts：覆盖 storage key 常量、santizePathForKey、clearTableStateByPath。
       - pageTitles.test.ts：覆盖页面标题映射表键值。
       - routes.test.ts：覆盖路由常量与权限映射。
       - upload.test.ts：覆盖上传常量与限制。
       - buttonStyles.test.tsx：覆盖 ButtonType/ButtonSize/BUTTON_STYLES/DELETE_CONFIRM/BATCH_DELETE_CONFIRM/STATUS_ICONS 静态结构与 JSX 图标渲染（16 stmts，补齐可提升 constants 目录 pct）。
    3. types 测试：
       - config.test.ts/dashboard.test.ts/notice.test.ts/common.test.ts：覆盖类型守卫（isValidXxx）、默认值对象、辅助函数；纯接口类型文件无需测试。
       - widgets/helpers.test.ts：覆盖 asWidgetComponent/createLazyWidget 的类型转换与懒加载封装（4 stmts）。
       - operations.test.ts：覆盖 DeviceSourceLabels 与 DEVICE_SOURCE_LABELS 别名（2 stmts，虽小但 types 目录基数低，补齐有助于稳过 70%）。
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npx vitest run src/services src/constants src/types --reporter=verbose
    </automated>
  </verify>
  <done>
    - services/constants/types 目录新增测试全部通过。
  </done>
  <acceptance_criteria>
    - src/services 下新增测试覆盖 encryptionConfig、captcha、dashboardService、configService、cache 与 operations 子目录 7 个模块。
    - src/constants 下新增 storage/pageTitles/routes/upload/buttonStyles 测试。
    - src/types 下新增 config/dashboard/notice/common/widgets/helpers/operations 测试，覆盖类型守卫与默认值。
    - npx vitest run src/services src/constants src/types 输出 Tests ... passed，exit 0。
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 2: router 测试与公共 harness 三件套创建</name>
  <files>
    xingran-react-frontend/src/router/routeConfigManager.test.ts
    xingran-react-frontend/src/router/routeGenerator.test.ts
    xingran-react-frontend/src/router/componentLoader.test.tsx
    xingran-react-frontend/src/router/DynamicRoutes.test.tsx
    xingran-react-frontend/src/router/RouteGuard.test.tsx
    xingran-react-frontend/src/test/utils/renderWithProviders.tsx
    xingran-react-frontend/src/test/utils/createApiMock.ts
    xingran-react-frontend/src/test/utils/mockAntdMessage.ts
    xingran-react-frontend/src/test/utils/harness.example.test.ts
  </files>
  <read_first>
    - xingran-react-frontend/src/router/routeConfigManager.ts
    - xingran-react-frontend/src/router/routeGenerator.ts
    - xingran-react-frontend/src/router/componentLoader.tsx
    - xingran-react-frontend/src/router/DynamicRoutes.tsx
    - xingran-react-frontend/src/router/RouteGuard.tsx
    - xingran-react-frontend/src/hooks/usePagination.test.tsx（MemoryRouter wrapper 模式）
    - xingran-react-frontend/src/lib/api/__tests__/networkApi.test.ts（vi.mock 模式）
    - xingran-react-frontend/src/utils/antdMessage.ts
    - xingran-react-frontend/src/store/menuStore.ts
    - xingran-react-frontend/src/test/setup.ts
  </read_first>
  <action>
    1. 创建 harness 三件套（落点 src/test/utils/，已被 coverage.exclude 排除）：
       - renderWithProviders.tsx：默认包裹 MemoryRouter + AntD ConfigProvider；通过 options 参数可选传入 route、initialEntries；stores 参数接收 Record<storeName, partialState>，在渲染前调用对应 useXxxStore.setState(partialState, true) 完成 reset（D-05）。支持 store 名：authStore、tabsStore、menuStore、dashboardStore、settingsStore、layoutStore、noticeStore、themeStore、visualizationStore。仅注入测试需要的 store，避免全量注入导致状态泄漏。
       - createApiMock.ts：导出 createApiMock() 返回 { post, get, put, del } 四个 vi.fn()，并在模块级执行 vi.mock('@/lib/api', () => ({ post, get, put, del, upload: post, postFormData: post, postLongRequest: post }))；提供 registerEndpoint(method, endpoint, response) 与 registerError(method, endpoint, error) 辅助。
       - mockAntdMessage.ts：导出 mockAntdMessage() 返回 message spy 对象，并执行 vi.mock('@/utils/antdMessage', () => ({ getAppMessage: () => messageSpy }))。
    2. router 测试：
       - routeConfigManager.test.ts：使用 fixture 菜单数据，覆盖 initialize、isInitialized、getRouteTitle、getRoutePath、getFlatMenuIds、menu 转换异常分支。
       - routeGenerator.test.ts：覆盖 generateRoutesFromMenus 输出结构、嵌套路由、外部链接、隐藏路由。
       - componentLoader.test.tsx：mock React.lazy 的 import()，覆盖 loadComponent 成功/失败/重试。
       - DynamicRoutes.test.tsx：使用 renderWithProviders 渲染 DynamicRoutes，mock routeConfigManager 与 componentLoader，验证路由匹配时渲染对应组件；若 jsdom 下懒加载不稳定，mock import() 返回同步组件。
       - RouteGuard.test.tsx：mock @/store/menuStore，覆盖无权限要求放行、有权限放行、无权限重定向 fallback、无权限显示 inline 403、fallbackElement 优先级（11 stmts，补齐 router 目录）。
    3. 创建 src/test/utils/harness.example.test.ts（不计入 coverage），演示 createApiMock 与 mockAntdMessage 的典型用法，并验证 stores 注入：使用 renderWithProviders 渲染一个读取 menuStore 的占位组件，传入 stores: { menuStore: { permissions: ['system:user:list'] } }，断言组件文本反映注入状态。
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npx vitest run src/router src/test/utils --reporter=verbose
    </automated>
  </verify>
  <done>
    - router 测试与 harness 三件套创建并通过；renderWithProviders 的 stores 参数在 harness.example.test.ts 与 RouteGuard.test.tsx 中得到验证。
  </done>
  <acceptance_criteria>
    - src/test/utils/ 存在 renderWithProviders.tsx、createApiMock.ts、mockAntdMessage.ts、harness.example.test.ts 四个文件。
    - renderWithProviders 实现 stores 参数：可接收至少一个 Zustand store 的初始状态并在渲染前调用 setState(partialState, true) 自动 reset。
    - src/router 下新增 routeConfigManager、routeGenerator、componentLoader、DynamicRoutes、RouteGuard 测试。
    - npx vitest run src/router src/test/utils 输出 Tests ... passed，exit 0。
    - renderWithProviders 至少被 src/router/DynamicRoutes.test.tsx 与 src/test/utils/harness.example.test.ts 导入使用。
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 3: harness 示例完善与 ratchet bump</name>
  <files>
    xingran-react-frontend/src/test/utils/harness.example.test.ts
    .coverage-fe-floors
    .planning/frontend-coverage-baseline.md
  </files>
  <read_first>
    - xingran-react-frontend/src/test/utils/renderWithProviders.tsx
    - xingran-react-frontend/src/test/utils/createApiMock.ts
    - xingran-react-frontend/src/test/utils/mockAntdMessage.ts
    - xingran-react-frontend/src/test/utils/harness.example.test.ts
    - .coverage-fe-floors（确认 services/router/constants/types 当前 floor）
    - .planning/frontend-coverage-baseline.md
    - .github/scripts/check-frontend-coverage.sh
  </read_first>
  <action>
    1. 确认 src/test/utils/harness.example.test.ts（不计入 coverage）演示 createApiMock 与 mockAntdMessage 的典型用法，并包含 store 注入断言。
    2. 基于 Plan 02~04 测试中的重复样板，评估是否将部分测试的 vi.mock('@/lib/api') 替换为 import { createApiMock } from '@/test/utils/createApiMock'。P0 阶段以"新增/修改测试文件使用 harness"为主，不强制回改已通过测试，避免引入回归。
    3. 运行 cd xingran-react-frontend && npm run test:coverage。
    4. 运行 bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors。
    5. 读取 services/router/constants/types 实测 pct，将 .coverage-fe-floors 中 services 行从 3.3、router 行从 0.0、constants 行从 38.8、types 行从 12.0 分别 bump 至 max(70.0, pct − 0.5) 并保留一位小数。
    6. 在 .planning/frontend-coverage-baseline.md 追加 ratchet 行：日期、phase 83-05、weighted_avg、total_stmts、total_covered、0pct_pkg_count、当前 commit 短 SHA、ratchet_from、ratchet_to。
    7. 重新运行 gate 脚本确认 services/router/constants/types PASS 且 global PASS。
  </action>
  <verify>
    <automated>
      cd xingran-react-frontend && npm run test:coverage 全量通过；
      bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors 输出 PASS: services ... >= 70.0%、PASS: router ... >= 70.0%、PASS: constants ... >= 70.0%、PASS: types ... >= 70.0%。
    </automated>
  </verify>
  <done>
    - harness 示例到位并验证 store 注入，四个目录 floor 更新，基线文档追加，Phase 83 P0 全清。
  </done>
  <acceptance_criteria>
    - src/test/utils/ 存在 harness.example.test.ts 并通过，且其中包含 renderWithProviders stores 注入的断言。
    - .coverage-fe-floors 中 services、router、constants、types 四行均 ≥70.0。
    - .planning/frontend-coverage-baseline.md 新增 ratchet 行包含 83-05、commit SHA、四个目录 ratchet 值。
    - gate 脚本输出四个目录均 PASS 且 global PASS。
    - npm run test:coverage 全量通过，159 存量测试不回归。
  </acceptance_criteria>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| harness → 测试代码 | harness 位于 src/test/utils/，被 coverage.exclude 排除，helper 代码不计入分母 |
| router 测试 → React Router | 使用 MemoryRouter 与 renderWithProviders，不操作真实浏览器 history |
| services 测试 → @/lib/api | 通过 createApiMock 或 vi.mock 拦截，无真实网络请求 |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-83-05-01 | Information Disclosure | harness 示例 | mitigate | 示例测试仅使用假 token/假端点，不暴露真实凭证 |
| T-83-05-02 | Denial of Service | DynamicRoutes 懒加载 | mitigate | jsdom 下不稳定时用 mock import() 返回同步组件 |
| T-83-05-03 | Tampering | routeConfigManager fixture | mitigate | 使用只读 fixture 数据，不修改运行时路由配置 |
| T-83-05-04 | Information Disclosure | constants 测试 | accept | 仅验证静态常量值，无敏感数据 |
| T-83-05-SC | Tampering | npm/pip/cargo installs | accept | 本 plan 不引入新包；无安装步骤 |
</threat_model>

<verification>
1. cd xingran-react-frontend && npm run test:coverage 全量通过，159 存量测试不回归。
2. bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors 通过，services/router/constants/types 均 PASS，global PASS。
3. git diff 检查：新增测试文件、harness 文件、.coverage-fe-floors 四行、.planning/frontend-coverage-baseline.md 追加行。
4. 抽样检查 gate 输出中 services pct ≥70.00%、router pct ≥70.00%、constants pct ≥70.00%、types pct ≥70.00%。
5. 确认 src/test/utils/ 下文件未出现在 coverage-final.json 的 per-file 列表中（被 exclude 排除）。
</verification>

<success_criteria>
- services、router、constants、types 四个目录 statements 覆盖率均 ≥70%（gate 输出 PASS）。
- src/test/utils/ 存在 renderWithProviders.tsx、createApiMock.ts、mockAntdMessage.ts 与 harness.example.test.ts。
- renderWithProviders 的 stores 参数支持按需注入并自动 reset，至少一个 P0 测试（如 harness.example.test.ts / RouteGuard.test.tsx）验证该行为。
- .coverage-fe-floors 中 services/router/constants/types 四行 bump 至 ≥70.0，基线文档追加同一 commit。
- 全量 vitest 0 失败，159 存量测试不回归。
- harness 文件不计入 coverage 分母。
</success_criteria>

<output>
Create `.planning/workstreams/frontend-coverage/phases/83-p0-70/83-05-services-router-harness-SUMMARY.md` when done
</output>
