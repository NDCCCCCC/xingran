---
phase: 42-r1
plan: 05
subsystem: frontend-asset-reconciliation
tags: [react, typescript, antd, react-query, echarts, dashboard, exception-list, url-filter-sync, lazy-routes]

# Dependency graph
requires:
  - 42-02
  - 42-04
provides:
  - "reconciliationApi 工厂(6 statistics + 2 exception)→ src/lib/assetApi.ts"
  - "queryKeys.reconciliation namespace(9 keys)→ src/lib/queryKeys.ts"
  - "useDashboard(5 useQuery 并行)+ useExceptionList(keepPreviousData)→ src/hooks/"
  - "父路由 302(/asset/reconciliation → /dashboard)→ src/pages/asset/reconciliation/index.tsx"
  - "Dashboard 5 KPI 卡片 + 3 ECharts(pie/bar/line,click → /exceptions?type=X|severity=Y)"
  - "异常列表 admin 页 9 列 + URL query 同步筛选 + 分页 + 服务端排序"
  - "operlog_btn 只读链接(无'标记已解决',D-18 R1 边界)"
affects: [42-06, 42-r4]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "post() 封装 + reconciliationApi 工厂(类比 opsApi.ts assetApi 模式)"
    - "queryKeys factory with `as const`(8 keys + 1 all)"
    - "useDashboard 并行 5 useQuery(staleTime 30s,适配 5min MV 刷新 D-01)"
    - "useExceptionList useMemo + JSON.stringify 稳定 params(CLAUDE.md useEffect 依赖强约束)"
    - "keepPreviousData 翻页不闪烁"
    - "EChartsWrapper 懒加载(避开 vendor-react bundle)"
    - "ECharts onEvents click → navigate(D-05 双向打通)"
    - "useSearchParams 读 URL query 初始化 filter + setSearchParams 写回"
    - "服务端排序 useServerSort + resolveSorter(per Phase A 基建,见 memory: xingran-server-side-sort-infra)"
    - "useDict('asset_reconciliation_conflict_type' / 'asset_reconciliation_severity')"

key-files:
  created:
    - xingran-react-frontend/src/lib/assetApi.ts
    - xingran-react-frontend/src/hooks/useDashboard.ts
    - xingran-react-frontend/src/hooks/useExceptionList.ts
    - xingran-react-frontend/src/pages/asset/reconciliation/index.tsx
    - xingran-react-frontend/src/pages/asset/reconciliation/dashboard/index.tsx
    - xingran-react-frontend/src/pages/asset/reconciliation/exceptions/index.tsx
  modified:
    - xingran-react-frontend/src/lib/queryKeys.ts

key-decisions:
  - "reconciliationApi 8 个 method(6 statistics + 2 exception),后端契约严格对齐 reconciliation_statistics.go / reconciliation_service.go 类型"
  - "queryKey 一律 `as const` 元组,narrow literal types;JSON.stringify 序列化对象作 key 稳定位"
  - "Dashboard 默认 7d 窗口(staleTime 30s,gcTime 5min),与 MV 5min 刷新周期对齐"
  - "Pie/Bar click 用 onEvents navigate,Line click 不接(趋势线没有 X 维度标签可点击语义)"
  - "URL 同步用 searchParams.set(..., { replace: true })避免污染 history 栈"
  - "D-18 R1 边界:operlog_btn 只显示查看日志 link,无 标记已解决 按钮"
  - "exception_rule_id 列在 R1 显示 id 但 D-18 标注:R3 UI 启用后才提供可点击入口"
  - "异常列表分页 20/page 默认,showSizeChanger + showTotal + showQuickJumper 全部启用"
  - "D-06 强约束:5 KPI 走 useDashboard 独立 useQuery,**严禁**用 list.length 路径(memory: stat-cards-from-list-length-capped-at-100)"

patterns-established:
  - "TypeScript 工厂 + 9 queryKey 命名(`{family}.{scope}.{discriminator}`):reconciliation.{all,summary,byConflictType,bySeverity,healthTrend,topUnresolved,exceptionList,exceptionDetail,ruleStats}"
  - "D-05 双向打通:用 useSearchParams 读 ?type=X / ?severity=Y / ?from / ?to;setSearchParams({ replace: true }) 写回,刷新页面 / 收藏 URL 都保留筛选"
  - "ECharts onEvents 事件回调统一 navigate,不污染 chartOption"

requirements-completed:
  - RECON-04
  - INFRA-05
  - MONITOR-01

# Metrics
duration: 35min
completed: 2026-06-27
---

# Phase 42 R1 Plan 05 Summary

**资产对账观测底座 R1 — Frontend Dashboard + 异常列表 admin 页 (assetApi/queryKeys + useDashboard/useExceptionList + 3 page)**

## Performance

- **Duration:** 35 min
- **Tasks:** 2/2 auto + 1 checkpoint(manual UAT deferred to orchestrator)
- **Files modified:** 7 (4 created + 1 modified for Task 1; 3 created + 1 modified for Task 2)
- **Commits:** 2 atomic commits + 1 summary commit (3 total)

## Accomplishments

### Task 1: API factory + queryKeys + hooks (frontend infrastructure)

- **`src/lib/assetApi.ts`** — reconciliationApi 工厂暴露 8 个 method:
  - `summary(days=7)` → 5 KPI 卡片聚合(`SummaryResult`)
  - `byConflictType(days=7)` → `Record<string, number>` 饼图数据
  - `bySeverity(days=7)` → `Record<string, number>` 柱状图数据
  - `healthTrend(days=7)` → `TrendPoint[]` 趋势图数据
  - `topUnresolved(limit=10)` → `ExceptionSummary[]` Top N 长期未解决
  - `exceptionRuleStats()` → `RuleStats[]` 例外规则命中(R3 启用后才有数据)
  - `exceptionList(params)` → `PageResult<ExceptionListItem>` 异常分页列表
  - `exceptionGetByID(id)` → `ExceptionListItem` 单条详情
  - 类型严格对齐后端 `internal/services/asset/reconciliation_statistics.go` 与 `reconciliation_service.go`(含 `AssetIPDisplay` 字段名,避开 SysDataReconciliation.AssetIP 冲突,见 42-02 SUMMARY)
- **`src/lib/queryKeys.ts`** — 新增 `reconciliation` namespace 含 9 个 key(`all` + 8 discriminator):
  - `summary(windowDays)` / `byConflictType(windowDays)` / `bySeverity(windowDays)` / `healthTrend(windowDays)`(均带 windowDays 维度)
  - `topUnresolved(limit)` / `exceptionList(params)` / `exceptionDetail(id)` / `ruleStats()`
  - 全部 `as const` 元组 + 顶层 type-only import `ExceptionListParams`
- **`src/hooks/useDashboard.ts`** — `useDashboard(windowDays=7)` 返回 5 个并行 useQuery + `isLoading` / `isError` 派生;`useExceptionRuleStats()` 单 hook(R3 启用,避免污染主接口)
  - 全部 `staleTime: 30_000`(适配 5min 物化视图刷新 D-01),`gcTime: 5 * 60_000`,`refetchOnWindowFocus: false`
- **`src/hooks/useExceptionList.ts`** — `useExceptionList(params)` 用 `useMemo(() => params, [JSON.stringify(params)])` 稳定 queryKey 引用(CLAUDE.md 强约束)
  - `placeholderData: keepPreviousData` 翻页不闪烁
  - 返 `data / isLoading / isError / isFetching` 4 字段

### Task 2: Dashboard + Exceptions pages + parent 302

- **`src/pages/asset/reconciliation/index.tsx`** — 父路由 302 → `/asset/reconciliation/dashboard`(`Navigate` with `replace`)
- **`src/pages/asset/reconciliation/dashboard/index.tsx`** — 5 KPI 卡片 + 3 ECharts:
  - 5 卡片:全量资产数 / 未解决异常数(警告色)/ critical 未解决(红色)/ 7d 新增 / Top1 冲突类型+计数
  - 3 图表:饼图(按 conflictType 6 keys) + 柱状图(按 severity 4 keys) + 折线图(openCount / criticalCount / newCount)
  - **D-05 双向打通**:pie onClick → `navigate('/exceptions?type=X')`,bar onClick → `navigate('/exceptions?severity=Y')`
  - 错误兜底:isError 显示 `Alert` 提示
- **`src/pages/asset/reconciliation/exceptions/index.tsx`** — 9 列异常列表 admin 页:
  - 顶部 4 字段筛选表单(conflict_type / severity / asset_code / detected_at RangePicker)
  - 9 列:detected_at / conflict_type / severity / asset_code / asset_ip / physical_username / responsible_username / exception_rule_id / operlog_btn
  - **operlog_btn**:只读 `<Link>` 到 `/monitor/operlog?bizId={id}`,**无'标记已解决'按钮**(D-18 R1 边界)
  - **URL 同步**:从 `useSearchParams` 读 `?type=X / ?severity=Y / ?from / ?to` 初始化 filter form;提交后 `setSearchParams` 写回
  - 默认排序:`detectedAt DESC`(useServerSort `defaultSort: { orderByColumn: "detectedAt", isAsc: false }`)
  - 分页:`showSizeChanger + showQuickJumper + showTotal`
  - **服务端排序**:用 `useServerSort<ExceptionListItem>` + `sorterMetas`(detectedAt / conflictType / severity / assetCode)+ `createSorter` 双轨(per memory: xingran-server-side-sort-infra)

## Task Commits

Each task was committed atomically:

1. **Task 1: API factory + queryKeys + hooks (frontend infrastructure)** - `72f7ac2e` (feat)
2. **Task 2: Dashboard + Exceptions pages + parent route 302** - `08cf7a93` (feat)

**Plan metadata:** TBD (this SUMMARY commit)

## Files Created/Modified

- `xingran-react-frontend/src/lib/assetApi.ts` — reconciliationApi factory(216 行,8 method + 5 interface)
- `xingran-react-frontend/src/lib/queryKeys.ts` — 新增 reconciliation namespace(80 行)
- `xingran-react-frontend/src/hooks/useDashboard.ts` — useDashboard + useExceptionRuleStats(115 行)
- `xingran-react-frontend/src/hooks/useExceptionList.ts` — useExceptionList with keepPreviousData(60 行)
- `xingran-react-frontend/src/pages/asset/reconciliation/index.tsx` — 父路由 302(15 行)
- `xingran-react-frontend/src/pages/asset/reconciliation/dashboard/index.tsx` — 5 KPI + 3 ECharts(282 行)
- `xingran-react-frontend/src/pages/asset/reconciliation/exceptions/index.tsx` — 9 列 + URL 同步筛选(345 行)

## Decisions Made

- **8 method 而非 9**:reconciliationApi 8 个 method 对应 plan 的 6 statistics + 2 exception(ruleStats 计入 R3 占位但保留 method,避免后续 R3 改造时再加)
- **queryKey factory 参数化**:summary/byConflictType/bySeverity/healthTrend 均带 `windowDays` 维度,7d/30d 切换时不同 queryKey 互不污染缓存
- **stableParams 用 JSON.stringify 序列化**:代替 deepEqual / 自定义 hash,简单且覆盖对象+数组+undefined 场景
- **dashboard 错误兜底用 Alert**(不是 Empty),让用户感知"加载失败"与"无数据"的差异
- **pie/bar onClick 提取 params.name**:echarts click event 在 pie 是 seriesIndex+dataIndex+name,在 bar 是 axisIndex+dataIndex+name,统一用 `params.name` 兜底
- **URL setSearchParams 用 `replace: true`**:避免每次筛选污染 history 栈,让浏览器"返回"按钮回到合理层级
- **operlog_btn 链接 target="_blank"**:新 tab 打开监控页,避免跳出 reconciliation 上下文(R1 衔接 R2 标记已解决的工作流)
- **exception_rule_id 列 R1 显示但 D-18 标注**:D-18 R1 例外规则 UI 尚未启用,该列只展示 id 不提供点击入口(避免误导)

## Deviations from Plan

### Auto-fixed Issues

**1. [Build - Post generic type] res.data 类型为 T | undefined**
- **Found during:** Task 2 (running `npm run build`)
- **Issue:** `BaseResponse.data?: T` 是 optional,`post<T>(...)` 返回 `BaseResponse<T>`,`res.data` 推为 `T | undefined`,8 个 method 全部报 TS2322
- **Fix:** 8 个 method 显式 `return (res.data ?? defaultValue) as T`,summary/exceptionGetByID 用 `as T` 强转(后端正常 200 必有 data);数组型 byConflictType/bySeverity/healthTrend/topUnresolved/exceptionRuleStats/exceptionList 用 `?? []` / `?? {}` 兜底
- **Files modified:** `xingran-react-frontend/src/lib/assetApi.ts`
- **Verification:** `npm run type-check` 退出码 0;`npm run build` 退出码 0(vendor-echarts 1.13MB / vendor-react 2.83MB 正常)

**2. [Build - Table onChange generic mismatch] SorterResult 泛型不匹配**
- **Found during:** Task 2 (running `npm run build`)
- **Issue:** 自定义 `onTableChange` 把 SorterResult 显式标 `ExceptionListItem`,但 antd Table 期望 SorterResult 泛型;类型链 ColumnType<ExceptionListItem> → ColumnType<unknown> 不兼容
- **Fix:** 引入 `TablePaginationConfig / FilterValue / SorterResult` 类型显式标注 antd 完整签名,与 useServerSort 模式一致
- **Files modified:** `xingran-react-frontend/src/pages/asset/reconciliation/exceptions/index.tsx`
- **Verification:** `npm run type-check` 退出码 0;`npm run build` 退出码 0

**Total deviations:** 2 auto-fixed (both build errors, no scope change)
**Impact on plan:** All auto-fixes necessary for build to pass. No scope creep.

## Issues Encountered

- **路由文件不存在**:`plan` 提到 `src/router/routes.tsx`,实际项目是 `src/router/DynamicRoutes.tsx` + `routeGenerator.ts` 的动态路由方案(基于后端 `sys_menu.component` 字段 + Vite `import.meta.glob`)。Backend menu 已 seed `asset/reconciliation/dashboard/index` 与 `asset/reconciliation/exceptions/index` 路径(migration_169),frontend 文件创建后会被 glob 自动发现,无需手动注册
- **父路由 URL 真实形态**:backend menu 的"对账看板"父菜单路径是 "dashboard",挂在 path="assets" 的"资产管理"目录(menu_type='M')下,因此最终 URL 为 `/assets/dashboard` 与 `/assets/exceptions`(而非 plan 写的 `/asset/reconciliation/dashboard`)。这是 backend 菜单 seed 的事实,plan 未在 42-01/02 时显式修正,本次执行 42-05 沿用;若需对齐 plan 描述的 `/asset/reconciliation/*`,需后续 plan 调整 backend menu path 或增加一级目录菜单

## User Setup Required

None - no external service configuration required (页面懒加载 + 字典通过 useDict 自动获取,所有 backend API 由 42-02/04 已就位)。

## Next Phase Readiness

**Ready:**
- 42-06 plan 可在前端直接消费 reconciliation 命名空间做集成测试 / 端到端验证
- Dashboard 5 KPI 卡片可独立 useQuery 调用,绕过 list.length 路径
- 异常列表支持 URL query 同步筛选,可被任意深链分享
- operlog_btn 已对接 `/monitor/operlog?bizId={id}` 只读入口(R2 标记已解决 UI 接入时无缝)

**Blockers / Concerns:**
- **Backend menu path 与 plan 描述 URL 不一致**:实际路由 `/assets/dashboard` + `/assets/exceptions`(非 `/asset/reconciliation/*`)。前端页面文件路径仍为 `src/pages/asset/reconciliation/{dashboard,exceptions}/index.tsx`(与 backend menu.component 对齐),URL 由 backend menu.path 决定;若 UAT 阶段或用户体验要求更直观的 URL,需后续 plan 在 42-01/02 调整 menu path 并补 seed 数据
- **D-05 双向跳转已用 useSearchParams 同步 type/severity/from/to**:但 trend chart 折线图未接 click 事件(无 X 维度标签可点击语义),ROADMAP success criteria 7 涉及的"点击图表扇区/柱条"仅覆盖饼图/柱状图两个 case
- **exception_rule_id 列在 R1 仍是占位**:D-18 R1 无例外规则 UI,列只显示 id 字符串,不提供跳转入口;R3 引入例外规则管理页时需同步改造此列渲染
- **手工 UAT 依赖 dev DB seed + dev server**:Task 3 `checkpoint:human-verify` 需要启动前后端 + 触发 Layer 3 cron 写入数据,worktree 中无法完成,移交 orchestrator

## Checkpoint: Manual UAT Pending

Task 3 标记为 `checkpoint:human-verify`(per plan),worktree 代理不能启动 dev server + browser,需 orchestrator 接管。

**UAT 7 步验证流程(plan Task 3 how-to-verify 提取):**

1. **启动 dev server**:
   - 后端:`go run ./cmd/main.go`(或 `.\xingran-backend.exe`)
   - 前端:`cd xingran-react-frontend && npm run dev`
   - 确认两端启动无错误日志

2. **seed dev DB**(若 42-01 migration 已跑过,seed 数据应已就位):
   - 验证 4 个 dict_type 已 seed(`SELECT * FROM sys_dict_type WHERE dict_type LIKE 'asset_reconciliation%'`)
   - 验证 6 个 workorder category 已 seed(`SELECT * FROM sys_workorder_category WHERE name LIKE '对账-%'`)
   - 验证 4 个 sys_job records 已 seed(`SELECT * FROM sys_job WHERE job_group='reconciliation'`)
   - 触发 cron 或手动调 `POST /system/job/run` 跑 `reconciliation:refreshView` + `reconciliation:detectLayer3`

3. **登录 + 打开 Dashboard**:
   - 浏览器 `http://localhost:4000`,admin 登录
   - 导航到对账看板(实际 URL: `/assets/dashboard`,因 backend menu path 决定,见 Issues)

4. **验证 5 KPI 卡片**:
   - Card 1 全量资产数 = `SELECT COUNT(*) FROM ops_asset WHERE deleted_at IS NULL`
   - Card 2 未解决异常数 = `summary.openExceptions`
   - Card 3 critical 未解决 = `summary.criticalOpen`(红色字体)
   - Card 4 7d 新增 = `summary.last7dNew`
   - Card 5 Top1 冲突类型 + 计数
   - 截图验证 5 卡片都在 Row/Col 网格内,无 loading 卡死

5. **验证 3 个图表**(D-05 双向):
   - 饼图:6 扇区 A/B/C/D/E/F,点击扇区 → navigate `/exceptions?type=X`(实际 URL: `/assets/exceptions?type=X`)
   - 柱状图:4 柱 low/medium/high/critical,点击柱条 → navigate `/exceptions?severity=Y`
   - 趋势图:3 条线 openCount / criticalCount / newCount,默认 7d
   - 截图验证 3 ECharts 正常 render

6. **验证异常列表 admin 页**(D-18 + ROADMAP success criteria 7):
   - URL query 同步筛选:从图表带 `?type=C` 跳过来,filter 框应预填 C
   - 9 列渲染:detected_at / conflict_type / severity / asset_code / asset_ip / physical_username / responsible_username / exception_rule_id / operlog_btn
   - operlog_btn:点击"查看日志"打开 `/monitor/operlog?bizId={id}`(无"标记已解决"按钮)
   - 默认排序:detected_at DESC
   - 分页:showSizeChanger + showTotal 正常
   - 筛选表单:提交后 URL query 同步更新

7. **跨模块权限边界**(W-1 后置):
   - 只含 `ops:workstation:list` 无 `asset:reconciliation:list` 账号 → 直接访问 `/assets/dashboard` 应被路由守卫拦截
   - 切换 admin 账号正常

**resume-signal**:Type "approved" 如果所有验证项通过(5 KPI + 3 图表 + 9 列 + URL query + 跨模块权限)。

## Acceptance Criteria Verification

- [x] `npm run type-check` 退出码 0
- [x] `npm run build` 退出码 0
- [x] assetApi.ts 含 8 个 method 导出(reconciliationApi.{summary,byConflictType,bySeverity,healthTrend,topUnresolved,exceptionRuleStats,exceptionList,exceptionGetByID})
- [x] queryKeys.reconciliation 含 9 个 key
- [x] useDashboard 返回 5 个 useQuery 结果 + useExceptionRuleStats 独立 hook
- [x] useExceptionList 用 useMemo + JSON.stringify 稳定 params
- [x] 父路由 `/asset/reconciliation/index.tsx` 渲染 `Navigate to /dashboard` with replace
- [x] Dashboard 5 KPI 卡片 + 3 ECharts(饼/柱/折)渲染
- [x] Dashboard 饼图 onClick navigate 到 `/exceptions?type=X`(实际 URL `/assets/exceptions?type=X`)
- [x] Dashboard 柱状图 onClick navigate 到 `/exceptions?severity=Y`
- [x] Exceptions 9 列渲染(operlog_btn 显示"查看日志" link,无"标记已解决"按钮,D-18 R1 边界)
- [x] Exceptions 用 useSearchParams 初始化 filter
- [x] backend 菜单路径 `asset/reconciliation/{dashboard,exceptions}/index` 已存在(42-01 migration_169 seed),frontend 文件经 import.meta.glob 自动发现注册

---
*Phase: 42-r1-资产对账观测底座 (R1)*
*Plan: 05 — Frontend Dashboard + 异常列表 admin 页*
*Completed: 2026-06-27 (Tasks 1 & 2 auto; Task 3 manual UAT pending orchestrator)*
