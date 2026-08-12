---
phase: 14-frontend-ux
plan: 05b
subsystem: mac-history-frontend
tags: [frontend, three-state, empty-state, error-retry, mobile-responsive, mac-history, react-query]

# Dependency graph
requires:
  - phase: 14-05a
    provides: EmptyStateWithAction and ErrorAlertWithRetry shared components (D-18 / D-20)
  - phase: 14-04
    provides: exportMACHistory API + 导出当前查询 / 导出全量 button contract + network:mac:export permission
  - phase: 14-01
    provides: history.tsx structure with virtual Table + useTableQuery + 8-column config
  - phase: 14-02
    provides: trajectory.tsx 5 enhancements (time presets, URL prefill, dataZoom, dwell heatmap, Drawer)
provides:
  - history.tsx three-state (D-18 empty / D-19 skeleton / D-20 error) using 14-05a shared components
  - history.tsx re-introduces 14-04 export buttons with exportScope='current'|'all' preserved
  - history.tsx isFetching-driven table Spin (D-19) instead of full skeleton on every refetch
  - trajectory.tsx inline Alert replaced with ErrorAlertWithRetry (D-20)
  - trajectory.tsx empty state with EmptyStateWithAction (D-18)
  - devices/index.tsx action column 查看 MAC 历史 entry → /network/mac/history?deviceId=... (D-16)
  - devices/index.tsx detail Modal footer 查看 MAC 历史 entry (D-16)
  - network:mac:list permission gate on both device list and detail entry points
affects:
  - 14-UAT (新增三态交互 + 联动入口跳转 + 导出按钮回归测试条目)
  - 14-04 (已 close-out,但 exportScope 契约在 14-05b 重建后必须保持一致)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "三态抽象:list 页 first-load 用 Skeleton 占位,后续分页走 isFetching 触发表头 Spin(D-19)"
    - "错误码分级:1006 设备未找到 / 1007 token 失效触发 logout / 500 服务器错误 / 其他 generic"
    - "权限驱动 UI:useMenuStore.permissions 作为按钮可见性单一数据源(VDI 既有模式)"
    - "URL 参数驱动的页面间联动:list 跳 history 通过 deviceId query 串,history 页 useEffect 读取并预填表单"

key-files:
  created: []
  modified:
    - xingran-react-frontend/src/pages/network/mac/history.tsx
    - xingran-react-frontend/src/pages/network/mac/trajectory.tsx
    - xingran-react-frontend/src/pages/network/devices/index.tsx

key-decisions:
  - "useMenuStore.permissions 作为权限单一数据源(同 VDI VirtualMachineList 模式),network:mac:list 控制设备联动入口,network:mac:export 控制导出按钮"
  - "history.tsx 的 Skeleton 仅在 isLoading && list.length === 0 时渲染,避免每次切页都闪烁骨架(D-19)"
  - "isPlaceholderData 从 useTableQuery 解构中移除(原本用于 loading={isLoading || isPlaceholderData},改用 isFetching 单一信号)"
  - "history.tsx 重建 14-04 导出按钮时,exportScope 命名 ('current' | 'all') 与位置(工具栏查询/重置之后)严格保留"
  - "trajectory.tsx 错误态独立成块,不混入 Drawer '请先执行查询' Alert(后者属 info 级别,保留 inline Alert)"
  - "trajectory.tsx empty 触发条件:!error && queryParams && !isLoading && trajectoryData.length === 0"
  - "devices/index.tsx 详情 Modal footer 同样添加联动按钮,与列表行入口保持 UX 一致"

patterns-established:
  - "Pattern: 三态 (empty / skeleton / error) 必须使用 14-05a shared 组件,禁止在新页面写内联 Empty/Alert"
  - "Pattern: 跨模块入口按钮位置 — 列表行操作列 + 详情页头部操作区(Modal footer / page header)双入口"
  - "Pattern: 权限控制 — hasPermission helper 内部封装 menuPermissions.includes,UI 层不直接读 store"

requirements-completed: [UI-01, UI-02, UI-04]

# Metrics
duration: ~25min
completed: 2026-06-15
---

# Phase 14 Plan 05b: MAC 历史三态打磨 + 网络设备联动入口

**EmptyStateWithAction / ErrorAlertWithRetry / Skeleton 全面接入 history 与 trajectory 页 + 14-04 导出按钮在历史页工具栏完整保留 + 网络设备列表行与详情 Modal 新增"查看 MAC 历史"联动入口**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-06-15T01:21:50Z
- **Completed:** 2026-06-15T01:46:xxZ
- **Tasks:** 3 (1 logical task split into 3 file-specific commits)
- **Files modified:** 3

## Accomplishments

- history.tsx 三态打磨:D-18 EmptyStateWithAction 引导去设备管理(描述"该范围内未采集到 MAC 记录,请检查设备是否启用了 MAC 采集/端口采集周期" + 按钮"前往设备管理" → /network/devices);D-19 首次加载 3 行 Skeleton 占位,后续分页 isFetching 触发表头 Spin;D-20 ErrorAlertWithRetry 替换内联 Alert(错误码 1006/1007/500 分级)
- history.tsx 14-04 导出按钮重建:导出当前查询 (primary + DownloadOutlined + exportScope='current') + 导出全量 (default + DownloadOutlined + exportScope='all'),位置在工具栏查询/重置按钮之后不变,可见性由 hasPermission('network:mac:export') 控制,useState<exporting> 管理两个按钮 loading 互斥
- trajectory.tsx 错误态:ErrorAlertWithRetry 替换内联 Alert,refetch 作为 onRetry
- trajectory.tsx 空态:queryParams 已就绪但 trajectoryData 空时,EmptyStateWithAction 提示"该 MAC 在此时间范围内无轨迹数据" + "查看事件时间线" 按钮(链接到 #timeline 锚点)
- devices/index.tsx 行操作列新增"查看 MAC 历史" 按钮(icon: HistoryOutlined),点击 navigate(/network/mac/history?deviceId={record.id}),权限由 network:mac:list 控制
- devices/index.tsx 详情 Modal footer 同步添加"查看 MAC 历史"按钮,关闭按钮左侧,保持 UX 双入口一致

## Task Commits

Each task was committed atomically:

1. **history.tsx three-state + export buttons** - `6eafd3e` (feat(14-05b): polish history page)
2. **devices/index.tsx link entry** - `2e3956d` (feat(14-05b): add 查看 MAC 历史 entry on network device list and detail modal)
3. **trajectory.tsx three-state** - `24bff39` (feat(14-05b): polish trajectory page — replace inline Alert with ErrorAlertWithRetry + add empty state)

## Files Created/Modified

- `xingran-react-frontend/src/pages/network/mac/history.tsx` — Three-state (EmptyStateWithAction/ErrorAlertWithRetry/Skeleton) + 14-04 导出按钮重建 + useMenuStore 权限
- `xingran-react-frontend/src/pages/network/mac/trajectory.tsx` — ErrorAlertWithRetry 替换内联 Alert + EmptyStateWithAction 空态提示 + useQuery 增加 refetch 解构
- `xingran-react-frontend/src/pages/network/devices/index.tsx` — HistoryOutlined import + 查看 MAC 历史 行内按钮(操作列)+ 详情 Modal footer 按钮(关闭按钮前) + useMenuStore 权限

## Decisions Made

- **useMenuStore.permissions 作为权限单一数据源:** 沿用 VDI VirtualMachineList 既有模式(menuPermissions + hasPermission helper),保持代码一致性,避免引入新的权限读取方式
- **14-04 导出按钮重建原因:** 当前 main 分支 history.tsx 缺少 14-04 注入的按钮(因 safe_resume_gate close-out 后回退),14-05b 必须重建以满足 D-13 + 14-04 SUMMARY 自检要求
- **Skeleton 触发条件:** `isLoading && list.length === 0` — 仅在"首次加载 + 无任何缓存数据"时显示骨架,避免每次切页都闪烁(D-19 关键点)
- **isFetching 替代 isPlaceholderData:** useTableQuery 原始 `loading={isLoading || isPlaceholderData}` 用 || 混合了两个状态,改为 `loading={isFetching}` 单一信号(更准确:只要正在拉取就 Spin,首次加载由外层 Skeleton 接管)
- **trajectory.tsx 14-02 Drawer 内 Alert 保留:** Drawer 内的 "请先执行查询" Alert 是 info 级别引导,非错误态,保留 inline Alert 而非替换为 ErrorAlertWithRetry(避免误用 1006/1007/500 文案)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Re-built 14-04 export buttons lost in main branch**
- **Found during:** Task 1 pre-check (current main history.tsx inspection)
- **Issue:** 14-04 SUMMARY 中 `safe_resume_gate` close-out 后,`导出当前查询` / `导出全量` 按钮未出现在 main 分支的 history.tsx;但 plan must-have #6 明确要求"14-04 注入的导出按钮 prop 名称/位置不变"
- **Fix:** 在 6eafd3e commit 中重建按钮 — 保持 prop 名称 `exportScope='current'|'all'`、按钮文字、图标、权限点 `network:mac:export`、工具栏位置(查询/重置之后)与 14-04 SUMMARY 描述完全一致;同时补齐 `handleExport` 函数与 `exporting` state
- **Files modified:** xingran-react-frontend/src/pages/network/mac/history.tsx
- **Verification:** grep 命中 `导出当前查询` / `导出全量` / `exportScope` / `hasPermission('network:mac:export')` 全部通过
- **Committed in:** 6eafd3e (part of task 1 commit)

**2. [Rule 2 - Missing Critical] Added Skeleton for first load (D-19)**
- **Found during:** Task 1 history.tsx 改造
- **Issue:** 14-01 history.tsx 没有 D-19 要求的 Skeleton 占位,使用 `loading={isLoading || isPlaceholderData}` 让 Table 自带 Spin,但首次加载全空时仅显示一个空表格骨架行,UX 弱
- **Fix:** `renderTable()` 与 `renderCardList()` 顶部加 `if (isLoading && list.length === 0)` 早返,渲染 `<Skeleton active paragraph={{ rows: 3 }} />`;Table `loading` 改为单一 `isFetching` 信号
- **Files modified:** xingran-react-frontend/src/pages/network/mac/history.tsx
- **Verification:** npx tsc --noEmit exit 0;Skeleton 在桌面/移动分支均存在
- **Committed in:** 6eafd3e (part of task 1 commit)

**3. [Rule 2 - Missing Critical] Added device detail Modal entry for D-16 consistency**
- **Found during:** Task 2 devices/index.tsx 改造
- **Issue:** PLAN.md Task 2 action 写"若设备详情页 detail.tsx 存在,同样在头部操作区添加该按钮" — 当前 main 是 Modal-based 详情(无独立路由),操作区在 footer
- **Fix:** 详情 Modal footer 数组中,在 关闭按钮 之前插入"查看 MAC 历史"按钮,icon HistoryOutlined,onClick 调 navigate,与列表行入口同源
- **Files modified:** xingran-react-frontend/src/pages/network/devices/index.tsx
- **Verification:** npx tsc --noEmit exit 0;Modal footer 含 "查看 MAC 历史" + navigate(/network/mac/history?deviceId=...)
- **Committed in:** 2e3956d (part of task 2 commit)

**4. [Rule 1 - Bug] Trajectory empty state误判**
- **Found during:** Task 1 trajectory.tsx 改造
- **Issue:** 初版考虑用 trajectoryData 长度判断空态,但用户在 Drawer 输入但未点查询时 trajectoryData 为 undefined,误显示"该 MAC 在此时间范围内无轨迹数据"
- **Fix:** 空态触发条件改为 `!error && queryParams && !isLoading && (!trajectoryData || trajectoryData.length === 0)`,要求 queryParams 已就绪(已点击查询),避免未查询时误提示
- **Files modified:** xingran-react-frontend/src/pages/network/mac/trajectory.tsx
- **Verification:** npx tsc --noEmit exit 0;边界条件正确
- **Committed in:** 24bff39 (part of trajectory commit)

---

**Total deviations:** 4 auto-fixed (1 missing critical in main, 1 missing critical per D-19, 1 missing critical per D-16, 1 bug fix)
**Impact on plan:** All auto-fixes align plan must-haves with current main state. No scope creep.

## Issues Encountered

- 14-04 export buttons 缺失问题发现早,在 Task 1 启动前 grep 已知 — 在 commit 6eafd3e 同步重建,未阻塞
- trajectory empty state 边界条件有微调(初版过于宽泛,fix 后更精准) — 未影响其他任务,合并到 24bff39 commit
- npx tsc --noEmit 全程 exit 0,无 TypeScript 编译错误

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 14 frontend UX 闭环 — 5 plans (14-01 / 14-02 / 14-03 / 14-04 / 14-05a / 14-05b) 全部完成
- 14-UAT.md 已存在 (status: testing),需 orchestrator 跑 UAT 验证:
  1. history.tsx 桌面表格 + 移动卡片 + 三态 (empty/skeleton/error) + 14-04 导出按钮
  2. trajectory.tsx 错误码分级文案 + 空态提示
  3. devices 行操作列 + 详情 Modal 双入口跳转 /network/mac/history?deviceId=...
- 移动端响应式 (Grid.useBreakpoint xs) 已在 history.tsx 落地,Phase 15 性能优化阶段可继续
- Phase 15 (性能优化) 与 Phase 22C-22D (VDI 扩展) 仍为 Planned 状态,Phase 14 完成解除前置依赖

---

*Phase: 14-frontend-ux*
*Completed: 2026-06-15*
