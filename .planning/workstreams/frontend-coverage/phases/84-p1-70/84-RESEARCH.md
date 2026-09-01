# Phase 84: P1 组件层 ≥70% - Research

**Researched:** 2026-08-27
**Domain:** 前端组件层覆盖率补齐(React 19 + Antd 6 + Zustand 5 + jsdom + Vitest 4)
**Confidence:** HIGH(基于实测 coverage 数据、83 plan0-5 已落库 harness、`BulkWriteDrawer.test.tsx` + `HealthCard.test.tsx` 实证样本、check-frontend-coverage.sh 三个 awk 锚点实测)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Phase 84 `plan 0` 落地 `renderWithProviders` + `createApiMock` 完整版,与 83 D-04/D-05/D-06 锁定决策对齐;不推迟到 84 中段或 P2。
- **D-02:** `renderWithProviders` 默认注入 `<MemoryRouter>` + `<App>`(antd App 提供 message/modal/notification context)。Zustand stores 按需参数注入并自动 reset(对齐 83 D-05,Zustand 官方 resetBetweenTests 模式)。
- **D-03:** `createApiMock(endpoint)` 端点工厂形态——生成 vi.fn() spy,支持 `.mockResolvedValue()` / `.mockRejectedValue()` / `.mockImplementationOnce()` 链式;**不引入 MSW**(零新依赖纪律,与 83 D-06 对齐)。提供可选 `mockApiBatch(handlers: Array<{endpoint, response}>)` 一次注册多端点。
- **D-04:** 扩展 `.coverage-fe-floors` 引入 components subdir 行(`components/shared` / `components/dashboard` / `components/layout` / `components/CronSelector` / `components/captcha` / `components/operations` / `components/network` / `components/reconciliation` / `design-system`)——与 ROADMAP SC 字面"subdir ≥70%"对齐,与 82 D-05 的 `pages/<subdir>` 二级粒度模式对称。
- **D-05:** `check-frontend-coverage.sh` 的 awk 路径聚合逻辑需在 3 处(`L219 / L316 / L381`)同步扩展 `components/<subdir>` 分支,与现有 `pages/<subdir>` 完全镜像(改后保持 82 D-07 "ratchet bump 是纯数据变更"——84 subdir 行一旦落地后续 bump 即纯数据)。
- **D-06:** `components` 聚合行保留并 bump 至 84 终点值(白名单外实测 ≥70%)——既满足 ROADMAP SC "全清" 又保留 82 D-05 一级目录粒度向后兼容。`design-system` 不挂在 components 下,而是与 hooks/store/services 同级顶层行(与现状 15.0 行对齐)。
- **D-07:** CronSelector(316) + captcha(154) + operations(149) 三个 subdir 各自独立 floor——ratchet 互不掩盖,任一 subdir 倒退会被 gate 抓到。
- **D-08:** 4 个 plan,plan 0 = harness;wave 1 = `shared`(892) || `dashboard`(1068) 并行;wave 2 = `layout`(507) || `CronSelector+captcha+operations`(619) 并行;wave 3 = `network + reconciliation + 零散 + design-system`(966 stmts)独立收口。
- **D-09:** wave 内并行 PR 互不阻塞,各自 bump 各自 subdir floor(纯数据变更);并行度选择依据 = 同 wave 内组件相互无依赖(都是叶子),且 stmts 量级匹配避免大目录独占 plan。
- **D-10:** 与 83 D-10 风格一致——wave 内可并行,wave 间串行(每 wave bump 后实测覆盖率单调上升再进下一 wave)。
- **D-11:** **模式 A 锁定**——每个组件测试至少包含一次 user event(`fireEvent` 或 `@testing-library/user-event`)+ 一次 props 渲染断言;子 hook/store/api mock 走 `vi.mock()` 路径;**对齐 BulkWriteDrawer / HealthCard 既有风格**(已实测覆盖率的两个组件样本)。
- **D-12:** 纯展示组件(如 `ModernTag`、`EmptyStateWithAction`)允许**单一渲染断言**(无 user event),但需有 props 变异(至少 2 个 props 组合)的快照——保证覆盖率含金量而非纯 0%→100% 暴力行覆盖。
- **D-13:** antd `Drawer`/`Modal`/`Select` 渲染需要的 polyfill(`ResizeObserver`、`getComputedStyle` 子集)在 `src/test/setup.ts` 集中沉淀,**不**在每个测试文件重写——延续 BulkWriteDrawer 经验,把 jsdom 补丁提到 setup 层。
- **D-14:** 每个 plan 完成即 bump 对应 subdir floor 至实测−0.5pp(沿用 82 D-06 / 83 D-11 噪声余量纪律),同 PR 追加 `.planning/frontend-coverage-baseline.md` ratchet 行。
- **D-15:** `components` 聚合行 floor 在 wave 3 完成后 bump 至白名单外实测≥70% 值(终点目标 = 70.0 - 0.5 = 69.5 一位小数);`design-system` 同步 bump 至 70.0 - 0.5 = 69.5(若白名单外实测 ≥70%)。
- **D-16:** 沿用 83 D-12——每个 plan 的 verify 含 `npm run test:coverage` + `bash .github/scripts/check-frontend-coverage.sh` + QUAL-01 159 存量测试不回归断言;phase 级 verify 仅汇总。

### Claude's Discretion

- 同一组件内多文件的拆分粒度(单文件 `.test.tsx` 还是按组件家族聚合 `__tests__/ComponentGroup.test.tsx`)——按现有 `__tests__/` 目录模式参考(`network/port-write/__tests__/BulkWriteDrawer.test.tsx`、`reconciliation/__tests__/HealthCard.test.tsx`)。
- D-13 polyfill 清单的具体边界(哪个 antd 组件需要哪个 polyfill)——执行阶段按实际渲染失败实证补齐,不前置。
- D-03 `mockApiBatch` 与单端点 mock 的使用偏好——以简洁优先,单端点不够时再批量。
- `renderWithProviders` 是否默认注入 `QueryClientProvider`(若组件用 `@tanstack/react-query`,参考 widgets 体系)——按需参数注入,不默认。

### Deferred Ideas (OUT OF SCOPE)

- antd `Table` / `Form` 全局测试模式 —— 84 不在 components/table 范围(ROADMAP 把 table 归 P2 页面层)。deferred 到 P2 plan 时再评估。
- Storybook / 视觉回归 —— REQUIREMENTS Out of Scope;deferred 到 v2 候选。
- 组件 E2E(Playwright) —— REQUIREMENTS Out of Scope;deferred 到 v2 候选。
- MSW 网络层 mock —— D-03 仍未采纳(零新依赖优先);若 P2 出现真实拦截需求再评估。
- `renderWithProviders` 的 store 注入 API 扩展(嵌套 store 组合 / 自定义 reset 时机)——84 阶段按实证需求定形,不前置设计。
- `components/table/` / `components/three/` / `components/DeptTree/` / `components/IconSelect/` / `components/NoticeDetail/` / `components/NotificationBell/` / `components/TargetSelector/` / `components/markdown/` / `components/modal/` / `components/charts/` / `components/asset/` —— 这些"零散组件"按 stmts 量级归入 wave 3(若超过 304 stmts 边界),具体清单在 wave 3 plan 内确认。
- 84 阶段不动 CI timeout / 分片优化 —— 沿用 82 D-04,先观察全量口径 transform 实际增量。

</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| COMP-01 | components/shared 21 文件 892 stmts ≥70% | BulkWriteDrawer / HealthCard 实证 + BulkWriteDrawer 已有 ResizeObserver 局部 polyfill → D-13 全局沉淀后此类组件可直接复用 setup.ts |
| COMP-02 | components/dashboard 29 文件 1068 stmts ≥70%(widgets / templates / utils / settings / layout 五子目录) | Widget.tsx → WidgetRenderer → widgetRegistry → types/*Widget 链条 + utils/dataFetcher 走 @/lib/api;测试 mock 边界清晰(子目录可按 family 聚合 `__tests__/`) |
| COMP-03 | components/layout 16 文件 507 stmts ≥70%(HybridLayout / Sidebar / Header + shared/TabBar) | useLayoutStore 持当前 layout → dynamic import 三个 Layout;需 mock useLayoutStore + MemoryRouter + 各 Layout 内 useRouteTabs hook |
| COMP-04 | components/CronSelector 316 + captcha 154 + operations 149 各目录 ≥70% | CronSelector = 纯算法 + antd(@breejs/later + cron-validate + cron-parser)真实向量直测;captcha 含 canvas / PointerEvent drag / App.useApp;operations 三件套为 opsApi 包装型 |
| COMP-05 | components/network 324(50.6%) + reconciliation 144(18.1%) + 零散 + design-system 194(15.0%)各目录 ≥70% | BulkWriteDrawer / HealthCard 已有测试可参考;`design-system` 与 hooks/store/services 同级顶层行(D-06),不走 components 二级拆分 |

</phase_requirements>

## Summary

本 phase 的目标是把 components 五个组件组(白名单外约 4,150 stmts)+ 顶层 design-system(194 stmts)从当前低位拉到每个目录 statements 覆盖率 ≥70%,并把 83 D-04 承诺但 P0 各 plan 因纯逻辑测试为主而未沉淀的 `renderWithProviders` + `createApiMock` harness 在 P1 开工前补齐(实测 `src/test/utils/` 目录在 commit HEAD 仍不存在,只有 setup.ts——说明 83-05 harness 落地时只标了"待 P1 实证定稿")。

**关键发现:**

1. **gate 扩展锚点已实测定位**——`check-frontend-coverage.sh` 在 L219(L_init 段)、L316(全局聚合段)、L381(各目录聚合段)三处重复出现完全相同的 awk key 派生逻辑:
   ```awk
   if (n == 1) key = "(src root)"
   else if (seg[1] == "pages") key = "pages/" seg[2]
   else key = seg[1]
   ```
   扩展为镜像 components 二级拆分只需在 `seg[1] == "pages"` 旁加一个 `else if (seg[1] == "components") key = "components/" seg[2]`,保持现有 `else key = seg[1]` 兜底(D-06 设计 system 不入 components 二级、design-system 走顶层 seg[1])。

2. **harness 实证样本已就位**——`BulkWriteDrawer.test.tsx` 提供 `Wrapper = MemoryRouter + App` + 局部 `ResizeObserverStub` + 子 hook 子 API mock;`HealthCard.test.tsx` 提供 `vi.mock("../hooks/*")` + `vi.mock("echarts-for-react")` 模式。84 plan 0 的 harness 不是从零设计,而是**抽提这两处实证 + 83 PATTERNS.md 已规划的形态**,接口签名和契约可一手定形。

3. **setup.ts 当前 polyfill 仅 matchMedia**——实测 `src/test/setup.ts` 只有 matchMedia(antd 部分组件需要);BulkWriteDrawer 文件内 inline 的 ResizeObserver Stub 需要提升到 setup 层(D-13 沉淀);其他 antd v6 polyfill(`getComputedStyle` 子集用于 Drawer body style 计算 / `IntersectionObserver` 用于 Affix/Tabs lazy mount / `canvas` getContext stub 用于 SliderCaptcha / `scrollTo` 用于 Modal)需实证补齐——但属于 D-13 边界,执行阶段按渲染失败按需加。

4. **vitest.config.ts 84 阶段无需调整**——`coverage.include` 全 src 已就位,白名单(cad-editor / cad-elements)锁死(D-12),`coverage.exclude` 不需变更。新增测试文件按现有模式自动进入全量口径分母。

5. **组件依赖图清晰**——五个组件组都是叶子节点(无相互依赖),与 83 D-10 "底层先清" 不同,84 wave 可按 stmts 量级切并行。各组件子依赖(子 hook / 子 store / 子 api / canvas)实证归纳见 `Architecture Patterns` 章节,每个 plan 可直接套用对应 mock 边界。

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| 共享原子组件(ModernTag / EmptyStateWithAction / ActionButtons / ErrorAlertWithRetry / BatchDeleteButton / BatchExportModal / FileUpload / GlobalSearch / ImageGallery / NetworkExport / ColumnConfigModal) | Browser client(纯展示) | — | 无路由 / store 依赖,jsdom 渲染 + props 变异即可 |
| Excel 导入导出(ExcelImport / ExcelImportLazy / ExcelExport) | Browser client + Web Worker(`xlsx` 包) | — | 需 mock `xlsx` 模块避免真实解析大文件;FileUpload 子组件已读 |
| FloorPlanEditor 系列(`.tsx` + `.constants` + `.hooks` + `.panZoom` + `.types` + `.less`) | Browser client(canvas 渲染) | — | 平面图编辑器依赖 panZoom 逻辑 + canvas;可走"渲染 + 缩放/平移 API 调用"断言 |
| 部门树选择(DepartmentTreeSelect) | Browser client | `@/lib/api` dept endpoint mock | 需 mock dept 树接口 |
| 仪表盘 Widget 体系(`Widget` → `WidgetRenderer` → `widgetRegistry` → `types/*Widget`) | Browser client | `widgets/base/BaseWidget` + `configs/widgetRegistry` | 子组件家族聚合 `__tests__/widgets.test.tsx` + mock widgetRegistry |
| 仪表盘布局(DashboardGrid / GridItem / LayoutToolbar / TemplatePreview / TemplateSelector) | Browser client | react-grid-layout 第三方 + `useDashboardStore` | react-grid-layout 提供测试 Provider;store 按需 mock |
| 仪表盘设置(DashboardScopeSelector / DashboardSettings / DataSourceForm / DisplayConfigForm / EndpointSelector / ParamsEditor / RefreshIntervalSelector / WidgetSelector) | Browser client | 表单 + DataSourceForm 调 utils/dataFetcher | 表单交互断言;dataFetcher mock get/post |
| 仪表盘 Widget 编辑(WidgetDataFilter / WidgetEditor) | Browser client | @/lib/api + ConfigProvider | antd Form 渲染 + 端点 mock |
| 仪表盘模板(presets.ts) + 数据获取(dataFetcher.ts) | Browser client | @/lib/api + WebSocket | 纯函数 + API mock;dataFetcher 可独立测试 CRUD + cache + error path |
| 布局三件套(HybridLayout / ClassicLayout / InnovativeLayout + sidebar + header + breadcrumb) | Browser client | `useLayoutStore` + `useRouteTabs` (layout/shared/) | store 注入 + dynamic Layout 切换 |
| 布局侧边栏(sidebar / sidebar-helper / sidebar.utils / sidebar.constants) | Browser client | `useMenuStore` + useNavigate | 菜单折叠 / 路由跳转断言 |
| Cron 表达式选择器(CronSelector + fields/* + utils.ts + constants.ts) | Browser client | `@breejs/later` + `cron-validate` + `cron-parser`(真实算法) | D-08 模式:真实向量直测,expression ↔ config 往返 + 边界字符串 |
| 验证码(CaptchaModal + SliderCaptcha + TextCaptcha) | Browser client | `@/services/captcha` + App.useApp + PointerEvent drag | SliderCaptcha 需 mock `getContext` canvas;PointerEvent fireEvent |
| Operations 三件套(DeptSidebar / StatisticsCards / WorkstationDeviceTable) | Browser client | `@/lib/api` ops endpoints | opsApi 包装型契约测试 |
| 网络端口写(BulkWriteDrawer / PortBindingModal / PortWriteModal / SetAccessVlanModal / constants) | Browser client | `@/lib/api/networkApi` + Drawer + Select | BulkWriteDrawer 已有 90% 基线,补充其他三 Modal |
| 网络拓扑 / MAC(MACEventsTimeline / MACHeatmapChart / macEventMeta) | Browser client | echarts mock | 与 HealthCard 同 echarts mock 模式 |
| 工位健康卡片(HealthCard / HealthBadge / ExceptionMatchList / ReconciliationDrawer / ReconciliationTimeline) | Browser client | `hooks/useReconciliationVisibility` / `hooks/useWorkstationHealth` | HealthCard 已有 5 测试用例;其他组件可走同 vi.mock 子 hook 模式 |
| 设计令牌(tokens/colors / typography / echartsTheme) | Browser client(纯静态) | — | 静态常量断言 + 导出完整性 |
| 设计桥接(AntdThemeBridge / ThemeProvider / LayoutProvider / DensitySwitcher / LayoutSwitcher / PageTitle / SettingsShell) | Browser client | ConfigProvider + Theme context | ConfigProvider 注入 + 主题切换断言 |
| 动画(animations/keyframes / transitions) | Browser client(纯常量) | — | 静态断言;jsdom 不需要 polyfill canvas/CSS |

## Standard Stack

### Core(已存在,无需新增)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| vitest | 4.1.10(lock) | 测试运行器 | Phase 82 已锁 4.1.10,v8 全量口径通过 `coverage.include` 控制 [VERIFIED: package-lock + 本地 `npm run test:coverage`] |
| @vitest/coverage-v8 | 4.1.10(lock) | v8 coverage provider | 与 vitest 同版本,生成 `coverage-final.json` [VERIFIED: package-lock] |
| @testing-library/react | 16.3.2 | 组件 / hook 渲染 | BulkWriteDrawer / HealthCard 实证使用 [VERIFIED: package.json] |
| @testing-library/dom | 10.4.1 | DOM 断言 | BulkWriteDrawer 实证使用 `screen, fireEvent, waitFor` [VERIFIED: package.json + BulkWriteDrawer L21] |
| @testing-library/jest-dom | 6.9.1 | DOM matchers | `src/test/setup.ts` L4 已引入 [VERIFIED: setup.ts] |
| jsdom | 27.4.0 | DOM 环境 | `vitest.config.ts` 的 `environment: "jsdom"` [VERIFIED: vitest.config.ts L9] |
| antd | 6.6.0 | UI 组件库 | 所有目标组件的运行时依赖 [VERIFIED: package.json] |
| react | 19.2.8 | 渲染框架 | 19.x [VERIFIED: package.json] |
| react-router-dom | 7.18.2 | 路由 | MemoryRouter 测试 wrapper [VERIFIED: package.json + BulkWriteDrawer L23] |
| zustand | 5.0.15 | 状态管理 | layout / dashboard store 按需注入 [VERIFIED: package.json] |
| echarts / echarts-for-react | 6.1.0 / 3.0.6 | 图表 | HealthCard 实证 mock echarts-for-react [VERIFIED: package.json + HealthCard L30-32] |

### 注意:版本声明失同步隐患(沿用 83 RESEARCH 警告)

`package.json` 中 `vitest: ^4.0.18` 与 `@vitest/coverage-v8: ^4.1.10`、`@vitest/ui: ^4.0.18` 声明基线不同。当前 lockfile 把三者锁到 `4.1.10`,`npm ci` 下安全。84 阶段不动此声明(83 IN-06 deferred)。

### 84 阶段新增外部包

**无**。D-03 已锁死"零新依赖纪律",MSW 不引入;`renderWithProviders` + `createApiMock` 完全用 vitest 原生 `vi.mock` + `MemoryRouter` + `App` 实现。

### 84 阶段配置不变项

- `vitest.config.ts`:`coverage.include` 全 src 不动,`coverage.exclude` 不动(白名单锁死 D-12),`environment: "jsdom"` 不动,`setupFiles: ["./src/test/setup.ts"]` 不动。
- `.coverage-fe-floors`:在现有 28 目录下新增 9 个 components subdir 行 + design-system 行(D-04),GLOBAL 与 `(src root)`/`api`/`hooks`/`store`/`lib`/`utils`/`services`/`router`/`constants`/`types`/`pages/*` 全部不动。
- `.github/workflows/ci.yml`:不动(D-05 gate 扩展在 check-frontend-coverage.sh 内,不在 yml 内)。

## Package Legitimacy Audit

本 phase **不引入新的外部包**,全部复用现有 devDependencies 和 antd / zustand / echarts / react 运行时依赖。无需执行 slopcheck 注册表审计。唯一治理项是 83 遗留的 vitest 版本声明统一(IN-06 deferred,不在本 phase)。

## Architecture Patterns

### 测试数据流(System Architecture Diagram)

```
Vitest runner
    ↓
src/test/setup.ts (jsdom + matchMedia + [plan 0 沉淀] ResizeObserver + getComputedStyle 子集 + IntersectionObserver + canvas getContext stub + scrollTo)
    ↓
测试文件 (*.test.ts / *.test.tsx)
    ├── 业务组件 ──→ renderWithProviders(ui) → MemoryRouter + App + 可选 stores reset
    │                  ├── 子组件 mock → vi.mock('@/components/...')
    │                  ├── 子 hook mock → vi.mock('../hooks/...')
    │                  ├── 子 store mock → vi.mock('@/store/...')
    │                  └── 子 api mock → vi.mock('@/lib/api') 或 createApiMock
    ├── 纯展示组件 ──→ renderWithProviders(ui) + props 变异断言(D-12)
    ├── CronSelector utils ──→ 真实 @breejs/later / cron-validate / cron-parser(D-08 向量直测)
    └── 端口写 modal 等 antd 组件 ──→ renderWithProviders + fireEvent + waitFor
    ↓
coverage-final.json (v8 provider, all src caliber)
    ↓
check-frontend-coverage.sh (L219/L316/L381 三处扩展后聚合 components/<subdir>)
    ↓
CI gate (GLOBAL + per-dir floors + diff coverage ≥80%)
```

### Recommended Project Structure(84 阶段新增)

```
xingran-react-frontend/src/
├── test/
│   ├── setup.ts                  # [扩展] 集中沉淀 antd polyfill (D-13)
│   └── utils/                    # [新增,被 coverage.exclude 排除]
│       ├── renderWithProviders.tsx   # [新增,plan 0]
│       ├── createApiMock.ts          # [新增,plan 0]
│       └── *.test.ts                 # [可选] harness 自身使用示例(83 模式)
├── components/
│   ├── shared/
│   │   ├── __tests__/                # [新增] ModernTag / EmptyStateWithAction / ActionButtons / ErrorAlertWithRetry 等家族聚合
│   │   ├── ExcelImport.test.tsx     # 或单文件,Claude's Discretion
│   │   ├── FloorPlanEditor.test.tsx # 含 panZoom + hooks 子模块断言
│   │   └── ...
│   ├── dashboard/
│   │   ├── __tests__/
│   │   │   ├── widgets.test.tsx     # BaseWidget + 各 *Widget + widgetRegistry
│   │   │   ├── layout.test.tsx      # DashboardGrid / GridItem / TemplateSelector
│   │   │   ├── settings.test.tsx    # Settings 系列 + DataSourceForm + EndpointSelector
│   │   │   └── dataFetcher.test.ts  # 纯类测试(独立于 react)
│   │   └── templates/presets.test.ts # 纯数据 + 默认值断言
│   ├── layout/
│   │   └── __tests__/
│   │       ├── HybridLayout.test.tsx
│   │       ├── Sidebar.test.tsx
│   │       └── Header.test.tsx
│   ├── CronSelector/
│   │   ├── __tests__/
│   │   │   ├── utils.test.ts        # 真实向量直测(D-08)
│   │   │   └── CronSelector.test.tsx
│   ├── captcha/
│   │   ├── __tests__/
│   │   │   ├── SliderCaptcha.test.tsx  # mock canvas getContext + PointerEvent drag
│   │   │   └── CaptchaModal.test.tsx
│   ├── operations/
│   │   ├── __tests__/
│   │   │   └── WorkstationDeviceTable.test.tsx
│   ├── network/
│   │   ├── port-write/
│   │   │   ├── __tests__/
│   │   │   │   ├── BulkWriteDrawer.test.tsx  # [已存在] 5 用例
│   │   │   │   ├── PortBindingModal.test.tsx
│   │   │   │   ├── PortWriteModal.test.tsx
│   │   │   │   ├── SetAccessVlanModal.test.tsx
│   │   │   │   └── constants.test.ts        # [已存在]
│   │   ├── MACEventsTimeline.test.tsx
│   │   └── MACHeatmapChart.test.tsx
│   ├── reconciliation/
│   │   ├── __tests__/                # [HealthCard.test.tsx 已存在]
│   │   │   ├── HealthCard.test.tsx
│   │   │   ├── HealthBadge.test.tsx
│   │   │   ├── ExceptionMatchList.test.tsx
│   │   │   └── ReconciliationDrawer.test.tsx
│   └── ...
└── design-system/
    ├── tokens/__tests__/             # colors / typography / echartsTheme 静态断言
    └── components/__tests__/         # AntdThemeBridge / ThemeProvider / LayoutProvider / DensitySwitcher 等
```

### Pattern 1: renderWithProviders + 按需 store reset

**What:** 默认包裹 `MemoryRouter` + antd `App`(提供 message/modal/notification context);Zustand stores 走参数按需注入并自动 reset(对齐 83 D-05,Zustand 官方 resetBetweenTests 模式)。

**When to use:** 任何需要挂载组件或 `renderHook` 且依赖路由 / antd context / 任意 store 的测试。

**接口签名(基于 BulkWriteDrawer L48-54 + 83 D-05 锁定形态):**

```typescript
// src/test/utils/renderWithProviders.tsx
import { render, type RenderOptions, type RenderResult } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

/**
 * 84 D-02/D-05 锁定形态:默认 <MemoryRouter><App>{ui}</App></MemoryRouter>。
 * stores 走参数按需注入 — 调用方传入 store reset 函数,
 * 每个测试 beforeEach 调用一次避免状态泄漏(对齐 Zustand resetBetweenTests)。
 */
export interface RenderWithProvidersOptions extends Omit<RenderOptions, "wrapper"> {
  /** MemoryRouter initialEntries,默认 ["/"] */
  route?: string;
  /** 调用方提供的 store reset 列表 — 每个测试 beforeEach 调用一次 */
  resetStores?: Array<() => void>;
  /** QueryClientProvider 包装(若组件用 @tanstack/react-query) — 按需注入(D-13 boundary) */
  queryClient?: unknown;
}

export function renderWithProviders(
  ui: ReactElement,
  options: RenderWithProvidersOptions = {}
): RenderResult {
  const { route = "/", ...renderOptions } = options;
  return render(
    <MemoryRouter initialEntries={[route]}>
      <AntdApp>{ui}</AntdApp>
    </MemoryRouter>,
    renderOptions
  );
}
```

**使用范式(D-02 + D-11):**

```typescript
import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, fireEvent } from "@testing-library/dom";
import { useLayoutStore } from "@/store/layoutStore";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { HybridLayout } from "../HybridLayout";

describe("HybridLayout", () => {
  beforeEach(() => {
    useLayoutStore.setState({ currentLayout: "hybrid" }); // D-05 按需 store reset
  });

  it("renders sidebar + header + content area", () => {
    renderWithProviders(<HybridLayout><div>content</div></HybridLayout>);
    expect(screen.getByRole("banner")).toBeInTheDocument(); // header
    // Sidebar / Content 断言...
  });

  it("collapses sidebar on menu toggle click", () => {
    renderWithProviders(<HybridLayout><div /></HybridLayout>);
    fireEvent.click(screen.getByRole("button", { name: /collapse/i }));
    // 侧边栏 collapsed className 断言
  });
});
```

**实证对照 — BulkWriteDrawer.test.tsx L48-54:**

```typescript
function Wrapper({ children }: { children: ReactNode }) {
  return (
    <MemoryRouter>
      <App>{children}</App>
    </MemoryRouter>
  );
}
```

D-02 抽提的 `renderWithProviders` 等价于这个 Wrapper,但额外提供:1) `route` 参数切换 initialEntries;2) `resetStores` 自动调用;3) 标准化 testID / container 查询返回。

### Pattern 2: createApiMock 端点工厂

**What:** 端点工厂形态的 `@/lib/api` mock;支持单端点 `createApiMock(endpoint, response)` 与批量 `mockApiBatch(handlers)`。

**When to use:** 组件 / hook / store 测试中需 mock 多个 API 端点时。

**接口签名(基于 83 D-06 锁定形态 + BulkWriteDrawer L39-42 实证):**

```typescript
// src/test/utils/createApiMock.ts
import { vi, type Mock } from "vitest";

/** mockApi 工厂返回的端点 spy 集合 */
export interface ApiMockHandle {
  post: Mock;
  get: Mock;
  put: Mock;
  del: Mock;
  /** 注册 endpoint 命中时的返回 — 支持链式 mockResolvedValueOnce */
  endpoint: Mock;
}

/** 单端点 mock — BulkWriteDrawer 实证 L39-42 抽象 */
export function createApiMock(endpoint: string): ApiMockHandle {
  const post = vi.fn();
  const get = vi.fn();
  const put = vi.fn();
  const del = vi.fn();
  const endpointSpy = vi.fn();

  vi.mock("@/lib/api", () => ({
    post: (...args: unknown[]) => {
      const [url] = args;
      if (url === endpoint) return endpointSpy(...args);
      return post(...args);
    },
    get: (...args: unknown[]) => get(...args),
    put: (...args: unknown[]) => put(...args),
    del: (...args: unknown[]) => del(...args),
    upload: (...args: unknown[]) => post(...args),
    postFormData: (...args: unknown[]) => post(...args),
  }));

  return { post, get, put, del, endpoint: endpointSpy };
}

/** 批量端点 mock — D-03 可选 helper */
export function mockApiBatch(
  handlers: Array<{ endpoint: string; response?: unknown }>
): Record<string, ApiMockHandle> {
  const handles: Record<string, ApiMockHandle> = {};
  handlers.forEach(({ endpoint, response }) => {
    const handle = createApiMock(endpoint);
    if (response !== undefined) handle.endpoint.mockResolvedValue(response);
    handles[endpoint] = handle;
  });
  return handles;
}
```

**使用范式 — 端口写 modal:**

```typescript
import { createApiMock } from "@/test/utils/createApiMock";

const apiMock = createApiMock("/network/port/binding");
apiMock.endpoint.mockResolvedValueOnce({ code: 0, data: { bound: true } });

renderWithProviders(<PortBindingModal portId="p-1" />);
fireEvent.click(screen.getByRole("button", { name: "绑定" }));

await waitFor(() => {
  expect(apiMock.endpoint).toHaveBeenCalledWith("/network/port/binding", expect.objectContaining({ portId: "p-1" }));
});
```

### Pattern 3: 真实算法向量直测(CronSelector utils 适用)

**What:** CronSelector 的 utils.ts / constants.ts 走"真实 @breejs/later + cron-validate + cron-parser + 确定性字符串"直测,与 83 D-08 国密向量同模式。

**When to use:** 无 DOM / 无 antd 依赖的纯逻辑工具。

**接口签名:**

```typescript
import { describe, it, expect } from "vitest";
import {
  expressionToCronConfig,
  cronConfigToExpression,
  validateCronExpression,
  getNextRunTimes,
} from "../utils";

describe("CronSelector utils", () => {
  it("expression ↔ config 往返", () => {
    const expr = "0 0 12 * * *";
    expect(cronConfigToExpression(expressionToCronConfig(expr))).toBe(expr);
  });

  it("validateCronExpression 接受合法 6 段表达式", () => {
    expect(validateCronExpression("*/5 * * * *").valid).toBe(true);
  });

  it("getNextRunTimes 真实 later 解析", () => {
    const next = getNextRunTimes("0 0 12 * * *", 3);
    expect(next).toHaveLength(3);
    expect(next[0]).toBeInstanceOf(Date);
  });
});
```

### Pattern 4: 子 hook / 子 store vi.mock 链(BulkWriteDrawer + HealthCard 通用)

**What:** 测试中用 `vi.mock("../hooks/...")` 替换子 hook,返回受控值。

**When to use:** 组件依赖同目录 hooks 目录或子 store 的派生值。

**HealthCard L20-32 实证:**

```typescript
const mockUseReconciliationVisibility = vi.fn();
const mockUseWorkstationHealth = vi.fn();

vi.mock("../hooks/useReconciliationVisibility", () => ({
  useReconciliationVisibility: () => mockUseReconciliationVisibility() as boolean,
}));
vi.mock("../hooks/useWorkstationHealth", () => ({
  useWorkstationHealth: (_id: string) => mockUseWorkstationHealth() as ReturnType<typeof vi.fn>,
}));
vi.mock("echarts-for-react", () => ({
  default: () => <div data-testid="echarts-mock" />,
}));
```

### Pattern 5: 第三方 canvas / echarts mock

**What:** canvas / echarts 在 jsdom 下渲染复杂,用 vi.mock 替换为轻量 stub。

**When to use:** SliderCaptcha(canvas drag)、MACHeatmapChart / MACEventsTimeline(echarts)、WidgetRenderer 派生(echarts)。

**范式:**

```typescript
// canvas getContext stub — SliderCaptcha 适用
vi.mock("@/utils/canvas", () => ({
  getCanvasContext: () => ({ /* stub methods */ }),
}));

// 或更简单 — 直接 stub HTMLCanvasElement.prototype.getContext
HTMLCanvasElement.prototype.getContext = vi.fn().mockReturnValue({
  fillRect: vi.fn(),
  clearRect: vi.fn(),
  // ...
});
```

### Anti-Patterns to Avoid

- **每测试文件 inline 写 ResizeObserver / matchMedia / App wrapper:** 违反 D-13;统一在 setup.ts 与 renderWithProviders 沉淀,BulkWriteDrawer 已踩过此坑(实测 inline ResizeObserver Stub L27-36)。
- **mock 整个 antd 模块:** 失去组件实际渲染逻辑,覆盖率虚假提升;只 mock 子组件(echarts-for-react)或单 antd 组件子组件(Modal.confirm 等 message 调用)。
- **在 setup.ts 里全局 vi.mock 子模块:** 影响所有测试,优先在测试文件顶部 mock。
- **把 harness 放在被 coverage 统计的目录:** harness 必须严格在 `src/test/utils/`(被 `"src/test/"` exclude 排除),否则 helper 代码拉低覆盖率(83 PATTERNS Pitfall #6)。
- **CronSelector utils 测试中 mock @breejs/later / cron-validate:** 失去真实 cron 解析断言;沿用 83 D-08 国密模式真实调用。
- **renderWithProviders 全量注入所有 store:** 违反 D-05 resetBetweenTests 原则,导致测试间状态泄漏;按需 reset。
- **DashboardGrid 测试中引入真实 react-grid-layout:** 该库依赖 ResizeObserver 与 getBoundingClientRect,jsdom 渲染异常;mock 该库 + 断言子组件。
- **components/dashboard/widgets 整体聚合一个巨型测试:** 29 文件 1068 stmts 走 family 聚合(`__tests__/widgets.test.tsx` / `layout.test.tsx` / `settings.test.tsx`)—Claude's Discretion 边界,按组件家族聚合优于单文件大杂烩。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 子组件 + Router + App wrapper 模板 | 每个测试手写 `<MemoryRouter><App>` | `renderWithProviders` (plan 0 沉淀) | 减少样板;BulkWriteDrawer L48-54 + HealthCard 已实证需要 |
| API 端点 mock + 多端点注册 | 每个测试手写 `vi.mock('@/lib/api', ...)` + `mockPost.mockResolvedValueOnce` | `createApiMock` / `mockApiBatch` | D-03 锁定;零新依赖;支持链式 mockResolvedValueOnce |
| antd message / modal 抑制 | spy console + 禁用 message | 在 `renderWithProviders` 中 `<App>` 自动注入(无需 mock) | BulkWriteDrawer L51 用 App 包裹已实证;真实 context 渲染 |
| ResizeObserver / matchMedia polyfill | 测试文件 inline Stub | setup.ts 集中沉淀(D-13) | BulkWriteDrawer L27-36 inline 重复;基础设施前移避免污染 |
| Zustand store reset | 手动 localStorage.clear + 重新 mount | `useXxxStore.setState(initialState)` | Zustand 官方 resetBetweenTests 模式;无 DOM 副作用 |
| 复杂 cron 解析逻辑 | mock `@breejs/later` / `cron-validate` | 真实调用 + 确定性向量 | 83 D-08 模式;mock 后无法验证 cron 字符串边界 |
| dashboard Widget 渲染分发 | 真实 widgetRegistry dispatch | mock widgetRegistry 返回 stub 组件 | widgetRegistry dispatch 内部逻辑由 widgetRegistry 自身测试覆盖;WidgetRenderer 测试只需断言分发到正确组件 |
| Canvas / echarts 渲染 | 真实 canvas 渲染 + zrender | vi.mock("echarts-for-react") / stub getContext | jsdom 不支持;HealthCard L30-32 已实证 |

## Runtime State Inventory

本 phase 为 greenfield 测试补齐(非 rename/refactor),无数据迁移 / OS 注册表 / Secrets 变更。运行时关注点:

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| **Stored data** | 无 — jsdom localStorage/sessionStorage 每个测试进程隔离,通过 `useXxxStore.setState` + `localStorage.clear()` reset | 无 |
| **Live service config** | 无 — CI / CI 配置无变更 | 无 |
| **OS-registered state** | 无 | 无 |
| **Secrets / env vars** | 无 — 不引入新依赖,无新增环境变量 | 无 |
| **Build artifacts** | 无 — `src/test/utils/` 创建后会被 vitest transform,但被 `"src/test/"` exclude 排除,不进 coverage-final.json | 无 |

**Nothing found in category:** 全部明确无运行时状态变更。

## Common Pitfalls

### Pitfall 1: gate L219/L316/L381 三处漏改导致 subdir 行不聚合

**What goes wrong:** 84 plan 0 / wave 1 仅扩展其中一处 awk,导致 `--init` 模式生成正确但 gate 运行模式聚合不到 `components/shared` 行,新行在 floors 表里但 profile 中找不到 → b) 方向 violation "components/shared 未登记 floor" → exit 4。

**Why it happens:** 三处 awk 块功能相同(L219 `--init` / L316 GLOBAL_TABLE / L381 DIR_AGG)但上下文独立,复制粘贴漏改是高频 bug。

**How to avoid:**
1. 改前在 IDE 中对 `seg[1] == "pages"` 做列编辑,三处同步插入 `else if (seg[1] == "components") key = "components/" seg[2]`。
2. 改后本地跑 `bash .github/scripts/check-frontend-coverage.sh --init coverage/coverage-final.json | grep components` 确认生成正确。
3. 改后跑 `bash .github/scripts/check-frontend-coverage.sh coverage/coverage-final.json .coverage-fe-floors` 验证 dir-registered check 通过。

**Warning signs:** gate 输出 `FAIL: components/shared 未登记 floor` 而 floors 表中已存在。

### Pitfall 2: renderWithProviders 默认未注入 store,组件读 undefined 抛错

**What goes wrong:** components/layout HybridLayout 读 `useLayoutStore().currentLayout`,若测试未 reset store,store 持有初始状态 `"hybrid"` 不会抛错;但当组件依赖 `useDashboardStore().widgets` 等非空初始状态时,测试数据被污染。

**Why it happens:** D-02 + D-05 强调 "按需注入 + reset",但执行阶段容易为了"快速过测试"省略 beforeEach reset。

**How to avoid:**
- `hybridLayout.test.tsx` 必加 `beforeEach(() => useLayoutStore.setState({ currentLayout: "hybrid" }))`
- `dashboard/DashboardGrid.test.tsx` 必加 `beforeEach(() => useDashboardStore.setState({ widgets: [] }))`
- D-12 QUAL-01 159 存量测试不回归的前提下,新 store 测试都用 `setState` reset。

**Warning signs:** 单测过、一起跑部分失败;测试输出 `Cannot read property 'currentLayout' of undefined`。

### Pitfall 3: components/shared 21 文件共享组件被 FloorPlanEditor 拖低

**What goes wrong:** FloorPlanEditor(`.tsx` + `.constants` + `.hooks` + `.panZoom` + `.types` + `.less`)单个组件就可能 200+ stmts,加 `.less` 文件不计 stmts 但 `.tsx` 含全部业务;若只断言简单 render 不进入 panZoom 分支,该组件覆盖率可能仅 30%,拉低整个 `components/shared` 至 60%。

**Why it happens:** FloorPlanEditor 内有 zoom/pan/select 三套状态机与 panZoom 纯函数模块(.panZoom.ts),测试需调用 panZoom 工具与组件实例方法才能覆盖。

**How to avoid:**
- FloorPlanEditor.tsx 测试覆盖 mount + 缩放事件(zoom in/out)+ 拖动 + 选择事件。
- FloorPlanEditor.panZoom.ts 独立测试 pure function(screenToWorld / worldToScreen 等)。
- FloorPlanEditor.hooks.ts 测试自定义 hook(`useFloorPlan` 之类)。
- FloorPlanEditor.constants.ts 静态断言。

**Warning signs:** `npm run test:coverage` 后 `components/shared` 行 pct 在 50-65% 区间。

### Pitfall 4: components/dashboard widgets 测试聚合过粗,family 内子组件未实际渲染

**What goes wrong:** `__tests__/widgets.test.tsx` 单测试文件 mock 整个 widgetRegistry,断言"渲染了某个 widget"——但实际只跑通 mock 分发路径,真实 ChartWidget / ListWidget / MetricWidget / ProgressWidget / StatCardWidget / TableWidget 代码未被执行,覆盖率虚假。

**Why it happens:** 一次性 mock widgetRegistry 比按子组件单独测试省事,但失去对每个 Widget 实现的覆盖。

**How to avoid:**
- BaseWidget.test.tsx 独立断言 base 行为。
- 各 *Widget.test.tsx 单独渲染,断言 props → 渲染产物。
- widgetRegistry.test.tsx 静态断言:每个 widget type 都注册到 component map。
- 模板 presets.ts 测试:createWidget 工具函数 + DEFAULT_* 常量完整性。

**Warning signs:** `npm run test:coverage` 显示 widgets 子目录各 *Widget.tsx 单文件覆盖率 < 50%。

### Pitfall 5: captcha SliderCaptcha 拖动事件 PointerEvent 在 jsdom 下不触发

**What goes wrong:** SliderCaptcha 用 `PointerEvent`(`onPointerDown`/`onPointerMove`/`onPointerUp`)实现拖动;jsdom 24+ 才有 PointerEvent,且拖动逻辑涉及 rAF 节流与滑块状态机,fireEvent pointerdown 可能不进入 drag 分支。

**Why it happens:** rAF 在 jsdom 下不真实运行;PointerEvent 需要 polyfill 或 stub。

**How to avoid:**
- `src/test/setup.ts` 增补 `PointerEvent` stub(若 jsdom 缺):
  ```typescript
  if (typeof globalThis.PointerEvent === "undefined") {
    globalThis.PointerEvent = MouseEvent as unknown as typeof PointerEvent;
  }
  ```
- SliderCaptcha 测试不模拟完整拖动,而是断言:**(a)** 加载时调用 `getCaptcha` API + 渲染背景图 + 滑块;(b)**(c)** 点击刷新按钮调用 `getCaptcha` 再次;(d)**(c)** 验证失败时调用 `onError`;(e)**(c)** 验证成功时调用 `onVerified` ——通过 mock `verifySliderCaptcha` 返回不同结果验证 onVerified/onError 触发,不模拟真实 PointerEvent 拖动过程。
- 拖动逻辑的真实覆盖率由 unit test SliderCaptcha 内部 `handleDragStart/Move/End` 函数(若 extract)或由 `useRef` `currentX` setState 触发覆盖。

**Warning signs:** 测试运行时报 `PointerEvent is not defined`;或 fireEvent 后组件状态不变。

### Pitfall 6: CronSelector 测试中 mock later / cron-validate 导致边界用例失效

**What goes wrong:** CronSelector utils 含 expression ↔ config 双向转换 + getNextRunTimes,若 mock `@breejs/later` 与 `cron-validate`,将失去对真实 cron 字符串(`*/5`、`0 0 12 * * *`、`0 0 0 1 1 ?` 等)的解析测试。

**Why it happens:** 真实 later 库在 jsdom 下行为稳定,无需 mock;但开发者习惯性 mock 第三方以"避免副作用"。

**How to avoid:**
- D-08 模式对齐:真实调用 `@breejs/later` / `cron-validate` / `cron-parser`,固定字符串 + 确定性输出。
- 边界字符串:`"*/5 * * * * *"`(每 5 秒)、`"0 0 0 1 1 ?"`(每 1 月 1 日 0 时)、`"0 0 9-17 * * *"`(工作小时)、`"30 0 0 1,15 * ?"`(每月 1/15 日 0:30)——覆盖 6 段、范围、列表、问号。
- expect(getNextRunTimes("0 0 12 * * *", 1)[0].getHours()).toBe(12) 类型断言。

**Warning signs:** 边界字符串测试失败时,优先怀疑 `cron-validate` vs `@breejs/later` 语义差异,而非测试代码。

### Pitfall 7: bulkwriteDrawer 已有 5 用例但仍依赖 inline ResizeObserver,与 D-13 集中沉淀冲突

**What goes wrong:** BulkWriteDrawer.test.tsx L27-36 inline `ResizeObserverStub` 类,84 plan 0 把 ResizeObserver 沉淀到 setup.ts 后,inline Stub 仍存在或测试运行时空 Stub 不一致。

**How to avoid:**
- plan 0 在 setup.ts 中沉淀 `ResizeObserver` / `IntersectionObserver` / `PointerEvent` 等 polyfill,**同时移除** BulkWriteDrawer.test.tsx 的 inline Stub(改为只保留 `import "@/test/setup"` 的副作用)。BulkWriteDrawer 改动的边界是 CLAUDE.md "Scope Constrainment" 允许的(测试基础设施沉淀),不修业务逻辑。
- 改动 BulkWriteDrawer 后跑 `npx vitest run src/components/network/port-write/__tests__/BulkWriteDrawer.test.tsx` 验证 5 用例不回归。

**Warning signs:** setup.ts 加 polyfill 后 BulkWriteDrawer.test.tsx 重复声明 `ResizeObserverStub` 报 `Cannot redeclare`。

### Pitfall 8: design-system AntdThemeBridge / ThemeProvider 需 ConfigProvider 注入

**What goes wrong:** design-system/components/AntdThemeBridge.tsx 与 ThemeProvider.tsx 强依赖 antd ConfigProvider 注入 theme token,jsdom 下 `theme.darkAlgorithm` / `theme.compactAlgorithm` 返回值与真实浏览器一致但若未提供 ConfigProvider 父级,Provider 子组件读不到 token → 渲染异常。

**How to avoid:**
- renderWithProviders 默认已包 antd App(ConfigProvider 内置);AntdThemeBridge / ThemeProvider 测试用例中显式包 `<ConfigProvider theme={{...}}>{ui}</ConfigProvider>` 而非依赖默认 ConfigProvider(避免配置漂移导致断言失败)。
- 主题切换断言:点击 DensitySwitcher → `useSettingsStore` 或 `localStorage` 中 mode 变化断言。

**Warning signs:** 设计组件渲染时 token undefined / 主题色不变化。

## Code Examples

### 完整 renderWithProviders + createApiMock 形态

```typescript
// src/test/utils/renderWithProviders.tsx
import { render, type RenderOptions, type RenderResult } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntdApp } from "antd";
import type { ReactElement } from "react";

export interface RenderWithProvidersOptions extends Omit<RenderOptions, "wrapper"> {
  route?: string;
  resetStores?: Array<() => void>;
}

export function renderWithProviders(
  ui: ReactElement,
  { route = "/", ...rest }: RenderWithProvidersOptions = {}
): RenderResult {
  return render(
    <MemoryRouter initialEntries={[route]}>
      <AntdApp>{ui}</AntdApp>
    </MemoryRouter>,
    rest
  );
}
```

```typescript
// src/test/utils/createApiMock.ts
import { vi, type Mock } from "vitest";

export interface ApiMockHandle {
  post: Mock;
  get: Mock;
  put: Mock;
  del: Mock;
  endpoint: Mock;
}

export function createApiMock(endpoint: string): ApiMockHandle {
  const post = vi.fn();
  const get = vi.fn();
  const put = vi.fn();
  const del = vi.fn();
  const endpointSpy = vi.fn();
  vi.mock("@/lib/api", () => ({
    post: (...args: unknown[]) => (args[0] === endpoint ? endpointSpy(...args) : post(...args)),
    get: (...args: unknown[]) => get(...args),
    put: (...args: unknown[]) => put(...args),
    del: (...args: unknown[]) => del(...args),
    upload: (...args: unknown[]) => post(...args),
    postFormData: (...args: unknown[]) => post(...args),
  }));
  return { post, get, put, del, endpoint: endpointSpy };
}
```

### gate 三处扩展实证片段(L219 / L316 / L381)

```awk
# 改前(L219 / L316 / L381 完全一致):
if (n == 1) key = "(src root)"
else if (seg[1] == "pages") key = "pages/" seg[2]
else key = seg[1]

# 改后(三处同步):
if (n == 1) key = "(src root)"
else if (seg[1] == "pages") key = "pages/" seg[2]
else if (seg[1] == "components") key = "components/" seg[2]
else key = seg[1]
# design-system 走顶层 seg[1] 兜底("design-system"),无需特殊分支(D-06)
```

### BulkWriteDrawer Wrapper → renderWithProviders 迁移(plan 0 顺手迁移)

```typescript
// 改前 — BulkWriteDrawer.test.tsx L48-54
function Wrapper({ children }: { children: ReactNode }) {
  return (
    <MemoryRouter>
      <App>{children}</App>
    </MemoryRouter>
  );
}

// 改后 — 使用 renderWithProviders
import { renderWithProviders } from "@/test/utils/renderWithProviders";

// 替换 render(<X />, { wrapper: Wrapper }) 为 renderWithProviders(<X />)
```

### CronSelector utils 真实向量直测

```typescript
// 沿用 83 D-08 模式
import { describe, it, expect } from "vitest";
import { expressionToCronConfig, cronConfigToExpression, getNextRunTimes } from "../utils";

describe("CronSelector utils", () => {
  it("expression ↔ config 往返", () => {
    const cases = ["0 0 12 * * *", "*/5 * * * * *", "0 0 0 1 1 ?", "0 0 9-17 * * *"];
    for (const expr of cases) {
      expect(cronConfigToExpression(expressionToCronConfig(expr))).toBe(expr);
    }
  });

  it("getNextRunTimes 调用真实 later 解析", () => {
    const next = getNextRunTimes("0 0 12 * * *", 3);
    expect(next).toHaveLength(3);
    expect(next[0].getHours()).toBe(12);
    expect(next[0].getMinutes()).toBe(0);
  });
});
```

### 子 hook vi.mock 子组件范式(HealthCard L20-32 复用)

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render } from "@testing-library/react";
import { screen, fireEvent } from "@testing-library/dom";

const mockUseFoo = vi.fn();
vi.mock("../hooks/useFoo", () => ({
  useFoo: () => mockUseFoo(),
}));

vi.mock("echarts-for-react", () => ({
  default: () => <div data-testid="echarts-mock" />,
}));

import { MyCard } from "../MyCard";

describe("MyCard", () => {
  beforeEach(() => mockUseFoo.mockReset());

  it("renders null when hook returns false", () => {
    mockUseFoo.mockReturnValue({ visible: false });
    const { container } = render(<MyCard id="x" />);
    expect(container.firstChild).toBeNull();
  });

  it("calls onAction button on click", () => {
    mockUseFoo.mockReturnValue({ visible: true, data: { x: 1 } });
    const onAction = vi.fn();
    render(<MyCard id="x" onAction={onAction} />);
    fireEvent.click(screen.getByRole("button", { name: "动作" }));
    expect(onAction).toHaveBeenCalledTimes(1);
  });
});
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Vitest 4 `coverage.all`(被移除) | `coverage.include` 显式圈定 `src/**/*.{ts,tsx}` | Phase 82 | 584 文件全部计入报告,未测试文件以 0% |
| 全局 threshold 配置(vitest 内置) | 外部 bash gate + `.coverage-fe-floors` ratchet | Phase 82 | 避免阈值配置与实测漂移导致 `test:coverage` 自锁 |
| 业务层测试 inline `<MemoryRouter><App>` Wrapper | `renderWithProviders` 工厂(D-02 锁定) | Phase 84 | 减少样板;统一 antd context + reset 入口 |
| 单文件 mock `@/lib/api` | `createApiMock` 端点工厂 | Phase 84 | P1/P2 数千个组件测试复用;零新依赖 |
| inline polyfill(ResizeObserver / matchMedia) | `src/test/setup.ts` 集中沉淀 | Phase 84 | D-13 沉淀;避免每测试重复 |
| `pages/<subdir>` 二级聚合(82 D-05) | 扩展 `components/<subdir>` 二级聚合 + 顶层 `design-system` 行 | Phase 84 | D-04 锁定;与 pages 镜像;设计系统与 hooks/store/services 同级 |

**Deprecated/outdated:**
- BulkWriteDrawer.test.tsx L27-36 inline ResizeObserver Stub → plan 0 迁移至 setup.ts 后移除。
- 83 PATTERNS.md 描述的 `src/test/utils/` 目录中 harness 形态(`renderWithProviders.tsx` / `createApiMock.ts` / `mockAntdMessage.ts`)在 commit HEAD 仍未实际落地(实测只有 setup.ts)——83-05 仅在 plan summary 标记完成,实际文件未创建。84 plan 0 必须从零创建这两个文件。

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `src/test/utils/` 目录在 commit HEAD **不存在**(`setup.ts` 是唯一文件),83 PATTERNS.md 描述的 harness 形态仅在 pattern doc 中,实际文件未创建。83-05 plan summary 标注"完成"但 git tree 实证缺失。 | Summary / Architecture / Patterns | 84 plan 0 假设 harness 落库会重新审视 — 已实证必须从零创建 `renderWithProviders.tsx` + `createApiMock.ts` |
| A2 | BulkWriteDrawer.test.tsx L27-36 的 inline `ResizeObserverStub` 可安全移除(测试仍能通过),因为 setup.ts D-13 沉淀后全局可用 | Pitfall #7 | 若 BulkWriteDrawer 在 Drawer 打开瞬间依赖特定 ResizeObserver 行为,移除 inline 后覆盖率不变但行为可能漂移;需 plan 0 跑测试验证 5 用例不回归 |
| A3 | `check-frontend-coverage.sh` L219/L316/L381 三处 `else if (seg[1] == "pages")` 是 awk key 派生的唯一相关行,扩展 components 二级拆分不需要触碰其它段 | Pitfall #1 | 若脚本某段(如 L212-L247 `--init` 模式)未触及,但生成 floors 输出时已正确聚合(--init 模式与 L219 同一处),plan 0 改后用 `--init` 验证 |
| A4 | design-system 与 hooks/store/services 同级顶层行(seg[1] 兜底),不走 components 二级拆分 | D-06 / D-15 | 若 design-system 内子目录(`tokens/` / `components/` / `utils/` / `animations/`)中某个子目录被期待独立 floor,需 D-06 重新审视;当前 D-06 锁 design-system 顶层行 |
| A5 | CronSelector 内部 `useState` + `useImperativeHandle` + antd Tabs + `@breejs/later` 真实运行在 jsdom 下稳定,无 antd Modal/Popover 触发 jsdom 抛错 | CronSelector 测试 | 若 Tabs lazy mount 触发 jsdom 缺 IntersectionObserver,需 setup.ts 沉淀 IntersectionObserver polyfill(D-13 边界) |
| A6 | SliderCaptcha 拖动逻辑真实 PointerEvent 在 jsdom 下可通过 `fireEvent.pointerDown` 触发,无需 stub PointerEvent | captcha 测试 | 若 jsdom 版本低于 24,PointerEvent undefined,需 setup.ts polyfill;实测 jsdom 27.4.0(VERIFIED),有 PointerEvent,但拖动 rAF 节流可能跳过中间态 |
| A7 | components/dashboard/widgets 子组件家族(ChartWidget / ListWidget / MetricWidget / ProgressWidget / StatCardWidget / TableWidget)共 12 文件可在 family `__tests__/widgets.test.tsx` 下聚合,无需每 Widget 单文件 | Architecture Patterns | 若 family 聚合后单文件 < 70%,按 Claude's Discretion 拆分为单 Widget `*.test.tsx` |
| A8 | operations/WorkstationDeviceTable 仅 1 文件 index.tsx + types.ts(2 文件),149 stmts 全部来自 index.tsx | COMP-04 | 若 index.tsx 包含多组件导出,family 聚合到 `__tests__/WorkstationDeviceTable.test.tsx` 即可 |

## Open Questions

1. **BulkWriteDrawer.test.tsx Wrapper 改造是否在 plan 0 范围?**
   - What we know: D-02 锁定 renderWithProviders 形态;D-13 集中 polyfill;BulkWriteDrawer L27-36 inline ResizeObserver 是迁移候选。
   - What's unclear: 改 Wrapper + 移除 inline polyfill 是否算 plan 0 必修项,还是 wave 1 起各 plan 顺手迁移?
   - Recommendation: plan 0 内**只创建 harness 文件 + setup.ts 沉淀 polyfill**,**不改 BulkWriteDrawer.test.tsx 本身**——保留"基础设施就绪,各 plan 顺手迁移"的边界,避免 plan 0 跨 2 文件改动;wave 1 起的 components/shared / network plan 顺手把 BulkWriteDrawer L48-54 改为 renderWithProviders。

2. **是否在 plan 0 顺手引入 `mockAntdMessage.ts`?**
   - What we know: 83 D-06 / 83 PATTERNS.md 提到三件套,但 84 D-02/D-03 只锁 renderWithProviders + createApiMock。
   - What's unclear: message mock 是必备还是 nice-to-have?
   - Recommendation: **plan 0 不创建**,按 84 D-02/D-03 边界只做两件;若 wave 内测试需要 mock antd message,在测试文件顶部 vi.mock antd 子模块即可(83 PATTERNS 模式)。

3. **D-13 polyfill 清单边界 — 哪些 antd 组件触发哪些 polyfill?**
   - What we know: D-13 锁"实际渲染失败实证补齐,不前置"。
   - What's unclear: Wave 1 dashboard widgets 中 ECharts 渲染失败怎么办?Wave 2 captcha canvas drag 失败怎么办?
   - Recommendation: plan 0 setup.ts 沉淀**已知必要**的 polyfill(matchMedia 已有 + ResizeObserver 必须);**IntersectionObserver / canvas getContext / PointerEvent** 在 wave 内测试失败时按需补,留在对应 wave plan 落地。

4. **Subagent / parallel test runs 是否影响 coverage.json?**
   - What we know: vitest 4 默认单进程,可加 `--shard` / `--pool=forks`。
   - What's unclear: 多 fork 并发跑测试是否影响 coverage-final.json 一致性?
   - Recommendation: 不调整 vitest pool/shard 配置,沿用 82 D-04 / 83 deferred(先观察全量 transform 实际增量);84 阶段 CI timeout 沿用 15min 不变。

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | 前端 build + test | ✓ | 24.19.0(实测) | — |
| npm | 包管理 | ✓ | 11.17.0 | — |
| node_modules | vitest / testing-library / antd / sm-crypto / zustand / echarts | ✓ | 已存在 | `npm install` |
| vitest | 测试运行器 | ✓ | 4.1.10(lock) | — |
| jsdom | DOM 环境 | ✓ | 27.4.0 | — |
| bash / sh | gate 脚本(awk 零依赖) | ✓ | Git Bash | — |
| GitHub Actions CI | gate 真实验证 | ✓ | ci.yml 84 阶段不动 | 本地 gate 脚本 |
| @breejs/later | CronSelector utils 真实调用 | ✓ | 4.2.0 | — |
| cron-validate | CronSelector utils 真实调用 | ✓ | 1.5.3 | — |
| cron-parser | CronSelector utils 真实调用 | ✓ | 5.10.0 | — |
| echarts / echarts-for-react | dashboard widget / network timeline mock | ✓ | 6.1.0 / 3.0.6 | — |
| antd | 全部组件 | ✓ | 6.6.0 | — |

**Missing dependencies with no fallback:** 无。

**Missing dependencies with fallback:** 无(84 阶段零新依赖)。

## Validation Architecture

> nyquist_validation 在 frontend-coverage workstream 启用(`workflow.nyquist_validation` 未显式 false,按 83 RESEARCH 默认启用),本节必须存在。

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Vitest 4.1.10 + @vitest/coverage-v8 4.1.10 + jsdom 27.4.0 + @testing-library/react 16.3.2 |
| Config file | `xingran-react-frontend/vitest.config.ts` |
| Quick run command | `cd xingran-react-frontend && npx vitest run src/components/<subdir>/__tests__/<file>.test.tsx` |
| Full suite command | `cd xingran-react-frontend && npm run test:coverage` |
| Gate script command | `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors` |
| Diff gate command | `bash .github/scripts/check-frontend-diff-coverage.sh xingran-react-frontend/coverage/coverage-final.json <base-ref> 80` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| COMP-01 | components/shared 21 文件 892 stmts ≥70% | coverage gate per-dir + 各 family 子测试 | `bash .github/scripts/check-frontend-coverage.sh ... \| grep '^components/shared '` | ❌ Wave 1 plan 1a 创建 |
| COMP-02 | components/dashboard 29 文件 1068 stmts ≥70% | coverage gate per-dir + widgets/layout/settings/family 聚合测试 | `bash .github/scripts/check-frontend-coverage.sh ... \| grep '^components/dashboard '` | ❌ Wave 1 plan 1b 创建 |
| COMP-03 | components/layout 16 文件 507 stmts ≥70% | coverage gate per-dir + HybridLayout/Sidebar/Header 三件套 | `bash .github/scripts/check-frontend-coverage.sh ... \| grep '^components/layout '` | ❌ Wave 2 plan 2a 创建 |
| COMP-04a | components/CronSelector 316 stmts ≥70% | coverage gate per-dir + utils 真实向量直测 | `bash .github/scripts/check-frontend-coverage.sh ... \| grep '^components/CronSelector '` | ❌ Wave 2 plan 2b 创建 |
| COMP-04b | components/captcha 154 stmts ≥70% | coverage gate per-dir + SliderCaptcha/CaptchaModal | `bash .github/scripts/check-frontend-coverage.sh ... \| grep '^components/captcha '` | ❌ Wave 2 plan 2b 创建 |
| COMP-04c | components/operations 149 stmts ≥70% | coverage gate per-dir + WorkstationDeviceTable 等 | `bash .github/scripts/check-frontend-coverage.sh ... \| grep '^components/operations '` | ❌ Wave 2 plan 2b 创建 |
| COMP-05a | components/network 324 stmts ≥70%(现 50.6%) | coverage gate per-dir + port-write 其余 Modal + MAC | `bash .github/scripts/check-frontend-coverage.sh ... \| grep '^components/network '` | ❌ Wave 3 plan 3a 创建 |
| COMP-05b | components/reconciliation 144 stmts ≥70%(现 18.1%) | coverage gate per-dir + HealthCard 扩 + 其余组件 | `bash .github/scripts/check-frontend-coverage.sh ... \| grep '^components/reconciliation '` | ❌ Wave 3 plan 3a 创建 |
| COMP-05c | components 零散 304 stmts ≥70% | coverage gate per-dir + table/three/DeptTree/IconSelect/NoticeDetail/NotificationBell/TargetSelector/markdown/modal/charts/asset | `bash .github/scripts/check-frontend-coverage.sh ... \| grep '^components/<subdir> '` | ❌ Wave 3 plan 3a 创建 |
| COMP-05d | design-system 194 stmts ≥70%(现 15.0%) | coverage gate per-dir(顶层)+ tokens/components/utils/animations 各自 family 测试 | `bash .github/scripts/check-frontend-coverage.sh ... \| grep '^design-system '` | ❌ Wave 3 plan 3b 创建 |
| QUAL-01 | 159 存量测试不回归(83 终点基线) | vitest 全量 | `npm run test:coverage` + `Tests 159+ passed` 计数 | ✅ 现有 |

### Success Criteria Observable Signals

| Success Criterion | Signal | How to Observe |
|-------------------|--------|----------------|
| SC-1: components/shared ≥70% | `components/shared` 行 gate 输出 PASS | `bash .github/scripts/check-frontend-coverage.sh ... \| grep '^PASS: components/shared'` |
| SC-2: components/dashboard ≥70% | `components/dashboard` 行 PASS | 同上 grep |
| SC-3: components/layout + CronSelector + captcha + operations 各 ≥70% | 四个 subdir 行全 PASS | 同上 grep |
| SC-4: components/network + reconciliation + 零散 + design-system 各 ≥70% | 全部 subdir 行 + `design-system` 顶层行 PASS | 同上 grep |
| SC-5: components 聚合行 ≥ 69.5% | `components` 行 PASS | `bash .github/scripts/check-frontend-coverage.sh ... \| grep '^PASS: components '` |
| SC-6: harness 沉淀 | `src/test/utils/renderWithProviders.tsx` + `src/test/utils/createApiMock.ts` 文件存在,至少 wave 1/2/3 各有 1 个测试 import 使用 | `ls` + `grep -r 'renderWithProviders\|createApiMock' src/components/` |
| SC-7: setup.ts polyfill 集中 | `src/test/setup.ts` 含 matchMedia + ResizeObserver;BulkWriteDrawer.test.tsx 不再 inline ResizeObserver Stub | `grep` |
| SC-8: QUAL-01 159 存量测试不回归 | `npm run test:coverage` 退出 0,测试计数 ≥ 159 | `npm run test:coverage 2>&1 \| tail` |
| SC-9: CI gate 绿 + ratchet 单调上升 | `.coverage-fe-floors` 中各 subdir floor 被 bump 且 `.planning/frontend-coverage-baseline.md` 追加新行 | `git diff .coverage-fe-floors frontend-coverage-baseline.md` |

### Nyquist 维度映射 / 假信号 vs 真信号 / 可量化 rubric

**Nyquist 4 维度映射(参考 83 VALIDATION 思路):**

| Nyquist 维度 | 84 验证映射 | 真信号 | 假信号(需警惕) |
|--------------|------------|--------|----------------|
| **Correctness(行为正确)** | 组件测试中 `user event` + `props 渲染断言` 双向覆盖(D-11) | fireEvent.click 后状态变化断言 + DOM 文本/role 查询命中 | 只断言 `expect(container).toBeTruthy()` 这种无意义 wrapper |
| **Coverage(覆盖率)统计** | `npm run test:coverage` + `bash .github/scripts/check-frontend-coverage.sh` 二者均绿 | gate 输出 PASS 行数 = 28 + 9 subdir 行 + components 聚合行 = 38 行 PASS | gate 绿但 profile 中含未测试文件 0% 计入;全局 weighted avg 提升但各 subdir 仍 0% |
| **Robustness(回归守护)** | QUAL-01 159 存量测试 + BulkWriteDrawer 5 用例 + HealthCard 5 用例不回归 | 全量 vitest 退出 0;`Tests N passed (N >= 159)` | 单测过、一起跑部分失败(状态泄漏 / 未 reset store) |
| **Reuse(模式沉淀)** | harness `renderWithProviders` + `createApiMock` 在 wave 1/2/3 各有 ≥1 个 import | `grep -r 'renderWithProviders' src/components/ \| wc -l >= 3` | harness 存在但无任何测试 import(死代码) |

**假信号 vs 真信号(本 phase 重点):**

| 信号类型 | 假信号(看似覆盖到位) | 真信号(实际覆盖到位) |
|---------|----------------------|----------------------|
| 覆盖率数字 | gate 输出 `PASS: components/shared 75.0%`(看似 70%+)实际因为 family 内某大文件 0% 拉低 | `npm run test:coverage` 各 *Widget.tsx / *Drawer.test.tsx 单文件覆盖率均 ≥60% |
| 测试通过 | 全量 vitest 退出 0,159 测试通过 | 关闭全部 components/* 测试,只剩其他 19 文件,仍 159 通过(说明 P1 测试未真存在) |
| gate 绿 | `.coverage-fe-floors` 中 `components/shared` = 70.0 | subdir 行实测 ≥ 70.0%(json 复算) — 不依赖 floors 表数 |
| ratchet bump | `.coverage-fe-floors` 中 `components/shared` 从 0 → 70(同 commit) | 该 commit 实测覆盖率 ≥70.0%;本地 `npx vitest run src/components/shared --coverage` 复核 |

**可量化 rubric(plan-level verify):**

每个 plan 完成必须命中以下 4 项量化指标:

1. **覆盖率目标命中**:`bash .github/scripts/check-frontend-coverage.sh ... | grep '^PASS: <subdir> '` 输出 ≥1 行 PASS,且 `<subdir>` 行 pct ≥ 目标 floor。
2. **测试文件存在**:`ls src/components/<subdir>/__tests__/*.test.tsx | wc -l` ≥ family 数。
3. **测试通过**:全量 vitest 退出 0,`Tests` 计数 ≥ 上一阶段基线。
4. **ratchet 同步**:同 commit 改 `.coverage-fe-floors(<subdir>` 行)+ `.planning/frontend-coverage-baseline.md` 追加新行。

**Sampling Rate:**

- **Per task commit(各 test file 完成):** `npx vitest run <新建/修改测试文件>` 确保单文件通过;涉及大组件家族时 `npx vitest run src/components/<subdir>/__tests__ --coverage` 局部验证。
- **Per wave merge(每 wave 末尾 PR):** 全量 `npm run test:coverage` + `bash .github/scripts/check-frontend-coverage.sh ...` + `bash .github/scripts/check-frontend-diff-coverage.sh ... HEAD 80`;按 D-14 bump 各 subdir floor 至实测−0.5pp 并追加基线文档(同 commit)。
- **Phase gate(phase 末尾):** 全量测试 ≥ 159 passed、gate 全 38 行 PASS(28 既有 + 9 subdir + 1 components 聚合)、CI frontend job 与 frontend-coverage-diff job 双绿、`.planning/frontend-coverage-baseline.md` 含 84 全部 ratchet 行。

### Wave 0 Gaps(plan 0 必须先创建)

- [ ] `src/test/utils/renderWithProviders.tsx` — Router + antd App + 按需 stores reset
- [ ] `src/test/utils/createApiMock.ts` — 端点工厂 + 批量注册 helpers
- [ ] `src/test/setup.ts` 增补 `ResizeObserver` polyfill(已知必要);`IntersectionObserver` / `PointerEvent` / canvas getContext 按需
- [ ] `.coverage-fe-floors` 新增 9 个 components subdir 行 + design-system 行 + components 聚合行(初值 = 0,D-14 后逐 plan bump)
- [ ] `.github/scripts/check-frontend-coverage.sh` L219/L316/L381 三处扩展 `components/<subdir>` 二级聚合分支
- [ ] `src/components/network/port-write/__tests__/BulkWriteDrawer.test.tsx` plan 0 末尾验证 5 用例仍 PASS(不强制改 Wrapper,仅 setup.ts 沉淀后允许)

## Security Domain

本 phase 不新增业务功能,仅补测试。涉及的安全相关代码已在运行中。

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | 弱(captcha 测试覆盖 SliderCaptcha / CaptchaModal) | mock `verifySliderCaptcha` 返回 token 验证 onVerified 触发 |
| V5 Input Validation | 中(CronSelector / DepartmentTreeSelect 涉及字符串解析) | CronSelector utils 真实解析 + 边界字符串断言 |
| V7 Error Handling | 弱(组件错误边界) | ErrorAlertWithRetry / EmptyStateWithAction 测试覆盖异常分支 |
| V8 Data Protection | 弱(captcha token 验证) | CaptchaModal 验证成功后调用 onSuccess 传 token |

### Known Threat Patterns for Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| 验证码 drag bypass(滑块未真正拖动到位置) | Tampering | mock `verifySliderCaptcha` 返回失败 + 断言 onError 触发 |
| Cron 注入(恶意 cron 表达式执行任意命令) | Elevation of Privilege | utils.validateCronExpression 真实 cron-validate + 边界字符串 |
| DepartmentTreeSelect 越权 dept 查询 | Information Disclosure | mock dept endpoint + 断言只调一次 |

## Sources

### Primary (HIGH confidence)

- 实测 coverage 数据:`xingran-react-frontend/coverage/coverage-final.json` 起点快照(2026-08-23 baseline + 2026-08-24 83 各 plan ratchet 行)— `.planning/frontend-coverage-baseline.md` 完整保留
- `xingran-react-frontend/vitest.config.ts` — coverage.include / exclude 真相源、jsdom 环境、15s timeout、setupFiles(`./src/test/setup.ts`)
- `xingran-react-frontend/src/test/setup.ts` — 仅 matchMedia polyfill(84 plan 0 沉淀基线)
- `.github/scripts/check-frontend-coverage.sh` L1-L443 全文 — L212 `--init` 段 / L309 `GLOBAL_TABLE` 段 / L374 `DIR_AGG` 段三处 awk 锚点实测确认
- `.github/scripts/check-frontend-diff-coverage.sh` — pathspec 镜像 + WR-01 fail-closed(83-01 落库)
- `.coverage-fe-floors` — 28 目录现状(components 4.9 / design-system 15.0 / pages/login 61.6 等)
- `xingran-react-frontend/src/components/network/port-write/__tests__/BulkWriteDrawer.test.tsx` L1-L449 — Wrapper + ResizeObserverStub + 子 hook mock 实证
- `xingran-react-frontend/src/components/reconciliation/__tests__/HealthCard.test.tsx` L1-L145 — 子 hook + echarts-for-react mock 实证
- `xingran-react-frontend/src/components/CronSelector/index.tsx` L1-L30 + `constants.ts` L1-L30 + `utils.ts` L1-L20 — CronSelector 依赖 @breejs/later + cron-validate + cron-parser + antd 实证
- `xingran-react-frontend/src/components/dashboard/Widget.tsx` L1-L28 + `widgets/WidgetRenderer.tsx` L1-L30 — dashboard widgets 分发链实证
- `xingran-react-frontend/src/components/captcha/SliderCaptcha.tsx` L1-L40 + `CaptchaModal.tsx` L1-L40 — captcha 依赖 `@/services/captcha` + PointerEvent + App.useApp 实证
- `xingran-react-frontend/src/components/layout/HybridLayout.tsx` L1-L40 + `index.tsx` L1-L30 — layout 依赖 useLayoutStore + useRouteTabs 实证
- `xingran-react-frontend/src/components/dashboard/utils/dataFetcher.ts` L1-L40 — dashboard 依赖 @/lib/api 实证
- `xingran-react-frontend/package.json` — 全部 deps 版本已 lock(`vitest 4.1.10` / `@testing-library/react 16.3.2` / `jsdom 27.4.0` / `antd 6.6.0` / `@breejs/later 4.2.0` / `cron-validate 1.5.3` / `cron-parser 5.10.0` / `echarts 6.1.0` / `echarts-for-react 3.0.6` / `zustand 5.0.15`)

### Secondary (MEDIUM confidence)

- 83 D-04/D-05/D-06 harness 形态(locked)— `83-CONTEXT.md` D-04/D-05/D-06 段
- 83 D-08 国密向量直测模式 — `83-RESEARCH.md` Pattern 6
- 82 D-05 pages 二级拆分 awk 镜像 — `82-CONTEXT.md` D-05 段 + `82-REVIEW.md` 实证
- 82 D-06 ratchet 余量 −0.5pp 纪律 — `82-CONTEXT.md` D-06 段
- 82 D-07 floors 数据真相源 + ratchet 是纯数据变更 — `82-CONTEXT.md` D-07 段
- Zustand 官方 resetBetweenTests 模式 — Zustand 文档

### Tertiary (LOW confidence)

- SliderCaptcha PointerEvent 在 jsdom 27.4.0 下的稳定性 — 需 wave 2 plan 2b captcha 测试实证
- DashboardGrid 中 react-grid-layout 在 jsdom 下的渲染稳定性 — 需 wave 1 plan 1b dashboard 测试实证
- FloorPlanEditor.tsx 内 panZoom 状态机覆盖率(若不调用内部 ref 方法,该文件覆盖率可能 < 50%) — 需 wave 1 plan 1a shared 测试实证
- AntdThemeBridge / ThemeProvider 依赖 ConfigProvider token 注入的 jsdom 渲染稳定性 — 需 wave 3 plan 3b design-system 测试实证

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — 版本、配置、既有测试模式均已实测核对;vitest / jsdom / antd / zustand / echarts 版本全部 lock
- Architecture: HIGH — 五个组件组源码已读,依赖方向清晰,既有 BulkWriteDrawer / HealthCard 测试模式可复用
- Pitfalls: MEDIUM-HIGH — 基于既有 19 个测试文件 + 83 P0 实证 + 项目历史问题(CLAUDE.md useEffect 规则、Phase 82/83 gate 修复经验)
- Gate 扩展锚点: HIGH — L219/L316/L381 三处 awk 段实测定位确认,镜像 components 二级拆分无需触碰其它段
- Harness 现状: HIGH — `src/test/utils/` 不存在已实测(`find src/test -type f` 只返回 setup.ts),83 PATTERNS.md 描述的 harness 仅 pattern doc 形态,84 plan 0 必须从零创建

**Research date:** 2026-08-27
**Valid until:** 2026-09-10(Vitest 4 + jsdom 27 稳定;React 19 + antd 6 稳定;vitest 版本声明失同步(IN-06)非本 phase 治理项)
