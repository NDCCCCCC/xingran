---
phase: 59-ad-login-guard-reconciliation
plan: 04
subsystem: frontend
tags: [react, antd, ad-sync, reconciliation, vitest, tabs, rbac]

requires:
  - phase: 59-ad-login-guard-reconciliation (59-03)
    provides: 只读端点 POST /api/v1/ad-domain/reconciliation/mismatch（DTO camelCase、分页归一、独立鉴权 ops:ad:config:reconciliation:list）+ migration_218 权限载体
  - phase: 36-ad-service-account-pool
    provides: AD 配置页 AccountPoolTab 分页模式与 configs/index.tsx 页面结构

provides:
  - 前端 API getDeptOUMismatchList + DeptOUMismatchRow / DeptOUMismatchType 类型（/ad-domain/reconciliation/mismatch）
  - 只读对账组件 ReconciliationTab（权限门 403 零请求 / 筛选显式提交 / 7 列表格 / 空态错误态 / 部分行渲染）
  - ADConfigPage 外层 Tabs（配置列表 / 对账）+ activeTab 会话持久化
  - ReconciliationTab.css 空态标题字重 600 !important 覆盖机制（共享组件零改动）
  - 38 个组件/API 契约自动化用例（权限门/空态/错误态/Tag 语义色/rowKey 唯一/筛选参数/持久化）

affects: [60-soft-delete-recovery, ad-domain-uat]

actuals:
  tokens: 9700
  tasks: 3
  commits: 3
  plan_head_before: 612a58c9dec955a3b033403d73c936be7408f098
  plan_head_after: e9df2e0b0fb1c3314124e8a95e14d72b99685005

tech-stack:
  added: []
  patterns:
    - "显式提交筛选形状：控件 onChange 只写 Form 草稿值，已提交筛选独立 state + queryTick 进 loadList 的 useCallback 原始值依赖；useEffect 依赖仅 [canView, loadList]"
    - "共享组件 inline 样式覆盖：本 Tab 专属包裹类 + 作者样式表 !important（先例 ActionButtons.css），共享组件零改动零新 props"
    - "测试注入 FormInstance seam：组件可选 form prop，测试用 Host 组件在内部 useForm 后捕获实例（antd v6 Form.useForm 是真 React hook 不能组件外调用）"
    - "空树常量喂树选组件：useDeptTree().data ?? EMPTY_DEPTS 模块级常量，避免内联 [] 新引用触发 rc-tree motion bail 陷阱（DeptTree #185 同款）"

key-files:
  created:
    - xingran-react-frontend/src/pages/ad-domain/configs/ReconciliationTab.tsx
    - xingran-react-frontend/src/pages/ad-domain/configs/ReconciliationTab.css
    - xingran-react-frontend/src/pages/ad-domain/configs/__tests__/ReconciliationTab.test.tsx
  modified:
    - xingran-react-frontend/src/lib/adDomainApi.ts
    - xingran-react-frontend/src/lib/adDomainApi.test.ts
    - xingran-react-frontend/src/pages/ad-domain/configs/index.tsx
    - xingran-react-frontend/src/pages/ad-domain/configs/__tests__/index.render.test.tsx

key-decisions:
  - "空态标题字重唯一可行机制 = ReconciliationTab.css 的 .reconciliation-tab-empty 包裹类 + font-weight:600 !important：EmptyStateWithAction 标题是 inline style(16px/500)，inline 只能被作者 !important 压过；共享组件不改、不加 props；无任何 hex 色值；jsdom 不算层叠 → 不做 vitest 断言，归视觉/UAT（UI-SPEC §规划期注意 2 的『修订契约』路线被显式否决）"
  - "筛选显式提交不引入 React Query：沿用模块内 AccountPoolTab 的 useState+useCallback 模式；已提交筛选（mismatchType/deptId/keyword）与 Form 草稿分离，加 queryTick 让相同筛选重复点「查询」也重拉"
  - "路由路径沿用 /ad-domain/reconciliation/mismatch：与 59-03 的 D-14 对齐决定一致，前端不加 /system/ad-config 别名"
  - "权限门用 useMenuStore().permissions.includes(RECONCILIATION_PERM)：权限字符串导出为组件常量单一事实源，与后端 RequirePermissions / migration_218 同一字面"
  - "ReconciliationTab 可选 form prop 作为测试 seam（缺省内部 useForm）：antd v6 Form.useForm 不能在组件外调用，测试用 Host 组件捕获实例后走 setFieldsValue 驱动提交路径"

patterns-established:
  - "antd v6 Tabs 断言口径：激活面板 = [role=tabpanel]:not(.ant-tabs-content-hidden)（v6 已无 .ant-tabs-tabpane-active，测试注释已写明）"
  - "antd Button 可达名断言：两汉字文案自动插空格（查 询/重 置）且 loading 期带图标前缀，按钮定位用 unanchored regex /查\\s*询/"

requirements-completed: [GUARD-02]

coverage:
  - id: D1
    description: "前端 API：getDeptOUMismatchList POST /ad-domain/reconciliation/mismatch，未传分页注入 current=1/pageSize=10，筛选参数透传"
    requirement: GUARD-02
    verification:
      - kind: unit
        ref: "npx vitest run src/lib/adDomainApi.test.ts（describe 部门↔OU 对账）"
        status: pass
    human_judgment: false
  - id: D2
    description: "权限门：无 ops:ad:config:reconciliation:list 渲染 Result 403（文案逐字）且 getDeptOUMismatchList 调用次数为 0"
    requirement: GUARD-02
    verification:
      - kind: unit
        ref: "ReconciliationTab.test.tsx『无权限渲染 403 文案且不发起任何请求』"
        status: pass
    human_judgment: false
  - id: D3
    description: "状态面：0 行渲染 EmptyStateWithAction（不渲染空表头）、失败渲染 ErrorAlertWithRetry + 重新加载并可重试、加载中 Table loading、按钮 loading 防重复提交"
    requirement: GUARD-02
    verification:
      - kind: unit
        ref: "ReconciliationTab.test.tsx『有权限且返回空列表…』『请求失败渲染错误文案…』『点击「重新加载」重新发起请求』"
        status: pass
    human_judgment: false
  - id: D4
    description: "7 列只读表格：表头文案与顺序、列宽/scroll.x=1400、部门级 NULL 渲染 '-' 不隐藏行、四类 mismatchType 中文标签 + orange/red/gold/blue 预设色、已删除部门红 Tag、rowKey 组合键唯一"
    requirement: GUARD-02
    verification:
      - kind: unit
        ref: "ReconciliationTab.test.tsx『渲染 7 列表头且顺序…』『部门级行…渲染 -』『四类 mismatchType…』『deptDeletedAt 非空…』『rowKey 组合键…唯一』"
        status: pass
    human_judgment: false
  - id: D5
    description: "筛选显式提交：控件 onChange 零请求（源断言 grep==0），查询/重置/Enter 提交并回 current=1，翻页保持已提交筛选"
    requirement: GUARD-02
    verification:
      - kind: unit
        ref: "ReconciliationTab.test.tsx describe 筛选显式提交（4 用例）"
        status: pass
      - kind: other
        ref: "grep -c \"onChange=.*loadList\\|onChange=.*setCurrent(1)\" ReconciliationTab.tsx == 0"
        status: pass
    human_judgment: false
  - id: D6
    description: "外层 Tabs（配置列表/对账）+ activeTab usePersistedStateController 持久化；对账 Tab 内容区无修复/删除/导出按钮"
    requirement: GUARD-02
    verification:
      - kind: unit
        ref: "index.render.test.tsx『存在…两个 Tab 标签…』『切到对账 Tab 后内容区无写操作按钮』『activeTab 持久化…』"
        status: pass
    human_judgment: false
  - id: D7
    description: "空态标题字重 600（CSS !important 覆盖共享组件 inline 16px/500）在真实浏览器的层叠生效"
    requirement: GUARD-02
    verification: []
    human_judgment: true
    rationale: "jsdom 不计算层叠样式，无法在 vitest 做计算样式断言（plan 已登记 Manual-Only）；规则本体存在性由文件+import+包裹类锁定，视觉生效需 UAT 复核。"
  - id: D8
    description: "Tab 栏不换行/不截断（UI-SPEC backstop statement）与窄屏 Form inline 换行形态"
    requirement: GUARD-02
    verification: []
    human_judgment: true
    rationale: "纯视觉回归断言 jsdom 不可得；当前仅 2 个 Tab 且沿用 antd 默认折叠行为，归 59-UAT 视觉复核。"

duration: 44min
completed: 2026-10-19
status: complete
---

# Phase 59 Plan 04: 对账只读 Tab 前端接入 Summary

AD 配置页新增只读「对账」Tab：`getDeptOUMismatchList` 接通 59-03 端点，权限门 403 零请求、筛选显式提交、7 列语义色表格、空态/错误态/部分行渲染全部按 59-UI-SPEC 落地并由 38 个组件/API 用例锁定。

## Performance

- **Duration:** 44min
- **Started:** 2026-10-19T14:52:28Z
- **Completed:** 2026-10-19T15:36:29Z
- **Tasks:** 3
- **Files modified:** 7 (3 created, 4 modified; +899/-2)

## Accomplishments

- 前端 API 层：`DeptOUMismatchType` 四值联合 + `DeptOUMismatchRow`（与后端 DTO json tag 逐字对齐）+ `getDeptOUMismatchList`（`/ad-domain/reconciliation/mismatch`，复用 `withDefaultPagination`）。
- `ReconciliationTab`：`useMenuStore` 权限门（无权限 403 且零请求）→ `<Form layout="inline">` 筛选行（不一致类型/部门树/关键字，onChange 零请求）→ 7 列只读表格（语义色 Tag、已删除红 Tag、ellipsis+Tooltip、`rowKey` 组合键、`scroll.x=1400`、pageSize 20）→ `ErrorAlertWithRetry` / `EmptyStateWithAction` 状态面。
- `ADConfigPage` 外层 Tabs（配置列表/对账）+ `activeTab` 会话持久化；两个 Tab 内容均保持挂载，切回不重置筛选。
- 反模式契约全绿：0 hex 色值、0 原生 axios、0 写操作入口、0 UUID 列、0 排序/导出/批量选择、`useEffect` 依赖仅原始值。
- 38 个自动化用例（API 20 + 组件 13 + 页面渲染 5）全绿，`npm run type-check` 通过。

## Task Commits

Each task was committed atomically:

1. **Task 1: 端到端：API 函数 + 权限门 + 空态对账 Tab + 页面 Tabs 接入（tracer）** - `a5abca44` (feat)
2. **Task 2: 筛选、分页、加载/错误态与部分行渲染** - `08484062` (feat)
3. **Task 3: 文案、颜色与反模式契约收口** - `e9df2e0b` (feat)

**Plan metadata:** committed with this SUMMARY (docs commit)。

## Files Created/Modified

- `xingran-react-frontend/src/lib/adDomainApi.ts` - 追加对账 API 与类型（路径 `/ad-domain/reconciliation/mismatch`，默认分页注入）。
- `xingran-react-frontend/src/lib/adDomainApi.test.ts` - 新 describe：端点路径 + 默认/覆盖分页断言。
- `xingran-react-frontend/src/pages/ad-domain/configs/ReconciliationTab.tsx` - 新组件：权限门/筛选/表格/分页/状态面；导出 `RECONCILIATION_PERM`、`MISMATCH_TYPE_META`、`rowKeyOf` 供测试复用。
- `xingran-react-frontend/src/pages/ad-domain/configs/ReconciliationTab.css` - 空态标题字重 600 `!important` 覆盖（无 hex）。
- `xingran-react-frontend/src/pages/ad-domain/configs/index.tsx` - 外层 Tabs 接入 + `usePersistedStateController` activeTab 持久化；既有配置逻辑零改动。
- `xingran-react-frontend/src/pages/ad-domain/configs/__tests__/ReconciliationTab.test.tsx` - 13 用例：权限门/空态/表头顺序/NULL 行/Tag 色/已删除 Tag/rowKey/错误重试/筛选提交/翻页保筛选。
- `xingran-react-frontend/src/pages/ad-domain/configs/__tests__/index.render.test.tsx` - mock 补 `getDeptOUMissingList`（resolve `{data:{list,total}}` 形状）+ 5 用例：Tab 标签/默认态/无写操作/持久化。

## Decisions Made

- **空态标题字重覆盖机制。** 共享 `EmptyStateWithAction` 的标题是 inline style（16px/500），唯一不改共享组件的覆盖方式是作者样式表 `!important`：新建同目录 `ReconciliationTab.css`，在组件内以 `reconciliation-tab-empty` 包裹类生效。UI-SPEC 给的另一条路（接受现状并修订契约）按契约优先原则否决。jsdom 不计算层叠 → 字重不做 vitest 断言，登记 Manual-Only 由视觉/UAT 复核。
- **筛选显式提交的依赖形状。** 控件 onChange 只写 Form 草稿值；已提交筛选为独立原始值 state，`loadList` useCallback 依赖它们 + `queryTick`（相同筛选重复点「查询」也重拉），`useEffect` 依赖仅 `[canView, loadList]`——同时满足 CLAUDE.md 稳定依赖规则与「onChange 不自动请求」禁令。
- **测试驱动提交路径用 form 注入 seam。** antd v6 `Form.useForm` 是真 React hook（组件外调用直接抛 Invalid hook call），组件加可选 `form` prop，测试用 Host 组件在函数体内创建并捕获实例后 `setFieldsValue` 驱动。
- **权限门在组件层短路。** `loadList` 首行 `if (!canView) return` + effect 内 `if (!canView) return`，无权限时连请求 effect 都不进入（测试断言调用次数为 0）。

## Deviations from Plan

### Harness note (not a code deviation)

- **Protected-branch guard vs `branching_strategy: none`**：仓库配置 `branching_strategy: "none"`，orchestrator 以 `<sequential_execution>` 指示 SEQUENTIAL=true 在配置分支 main 上提交（未改配置、未用 --no-verify），与 59-01/59-02/59-03 同款处置。

### Auto-fixed Issues

**1. [Rule 1 - Bug] 测试基线修正：空态断言与筛选标签文本冲突**
- **Found during:** Task 2
- **Issue:** Task 2 加入筛选行后，Form label「不一致类型」使旧的「空态不渲染空表头」断言（`queryByText("不一致类型")` 为 null）假失败。
- **Fix:** 改为以表格节点缺席断言（`container.querySelector(".ant-table")` 为 null）。
- **Files modified:** `ReconciliationTab.test.tsx`
- **Verification:** 全套 13 用例 3 连跑全绿。
- **Committed in:** `08484062`

**2. [Rule 3 - Blocking] antd v6 测试环境适配（4 项）**
- **Found during:** Task 2 / Task 3
- **Issue:** ① `Form.useForm` 组件外调用抛 Invalid hook call；② antd Button 两汉字文案自动插空格（"查 询"）且 loading 期带图标前缀，精确 name 匹配失败；③ 请求 resolve 后 `setLoading(false)` 未 flush 前按钮仍在 loading（点击被吞）；④ antd v6 Tabs 激活面板已无 `.ant-tabs-tabpane-active`，改用 `.ant-tabs-content-hidden` 反选。
- **Fix:** Host 组件捕获 form 实例；按钮定位改 unanchored regex；加 `waitButtonsIdle()` 等 `.ant-btn-loading` 消失；面板选择器改 `[role="tabpanel"]:not(.ant-tabs-content-hidden)`。
- **Files modified:** `ReconciliationTab.test.tsx`、`index.render.test.tsx`
- **Verification:** 18 用例全绿（含 3 连跑 flake check）。
- **Committed in:** `08484062` / `e9df2e0b`

---

**Total deviations:** 2 auto-fixed（1 × Rule 1 测试断言基线，1 × Rule 3 测试环境适配，均不触及生产代码行为）。
**Impact on plan:** 生产代码零偏移；测试基建口径（v6 Tabs 面板类名、Button 可达名）已沉淀为本文件 patterns-established 供后续 plan 复用。

## Issues Encountered

None.（三个任务的 `<automated>` verify 与验收 grep 全部在 2 次修复预算内通过；全计划验证 `npm run type-check` + 3 测试文件 38 用例全绿。）

## Known Stubs

无。组件/API/样式全部实装，无 TODO/FIXME/占位分支；所有列均绑定真实数据源。

## User Setup Required

None - 无新增外部服务配置。（部署沿用 59-03 的操作项：管理员需为相关角色授予 `ops:ad:config:reconciliation:list`，否则本 Tab 对非超管角色恒渲染 403——这是 D-17 锁定的预期行为，不是缺陷。）

## Next Phase Readiness

- Phase 59 四个 plan 全部完成；GUARD-02 的可观测闭环（视图 → 端点 → 权限载体 → 前端 Tab）已打通。
- 前端契约已定型：`getDeptOUMismatchList(params)` + `DeptOUMismatchRow`，`RECONCILIATION_PERM` 从组件导出，后续软删恢复（Phase 60）可直接复用类型与权限常量。
- 未完成项：两条 Manual-Only 视觉断言（空态字重、Tab 栏不溢出）归 59-VALIDATION/UAT；59-03 遗留的部署前置（真实启动跑一次 PG 分支 migration_217/218）仍待人工。

## Self-Check

- FOUND: xingran-react-frontend/src/lib/adDomainApi.ts
- FOUND: xingran-react-frontend/src/lib/adDomainApi.test.ts
- FOUND: xingran-react-frontend/src/pages/ad-domain/configs/ReconciliationTab.tsx
- FOUND: xingran-react-frontend/src/pages/ad-domain/configs/ReconciliationTab.css
- FOUND: xingran-react-frontend/src/pages/ad-domain/configs/index.tsx
- FOUND: xingran-react-frontend/src/pages/ad-domain/configs/__tests__/ReconciliationTab.test.tsx
- FOUND: xingran-react-frontend/src/pages/ad-domain/configs/__tests__/index.render.test.tsx
- FOUND: a5abca44 (task 1)
- FOUND: 08484062 (task 2)
- FOUND: e9df2e0b (task 3)
- `git rev-list --count 612a58c9..HEAD` = 3（与 ledger `gsd-plan-head-before-59-04` 一致）
- `npm run type-check` exit 0
- `npx vitest run src/lib/adDomainApi.test.ts src/pages/ad-domain/configs/__tests__/ReconciliationTab.test.tsx src/pages/ad-domain/configs/__tests__/index.render.test.tsx` → 3 files / 38 tests 全绿（组件套件另做 3 连跑 flake check）
- 源断言：ReconciliationTab.tsx hex=0 / `from "axios"`=0；`onChange=.*loadList|setCurrent(1)`=0；index.tsx `= usePersistedStateController`=1
- 无 package.json / package-lock.json 变更（T-59-SC 零新依赖确认）

## Self-Check: PASSED

---
*Phase: 59-ad-login-guard-reconciliation*
*Completed: 2026-10-19*
