---
milestone: v1.33
status: defined
defined: 2026-09-11
---

# Requirements: XingRan-Next — Milestone v1.33 V133 前端性能治理 (Frontend Performance Remediation)

**Defined:** 2026-09-11
**Core Value:** 修复 2026-09-11 前端全量性能审计报告（`.planning/reviews/20260911-frontend-perf-audit.md`）全部 33 项 findings（2 HIGH + 8 MEDIUM + 3 次级 MEDIUM + 12 LOW）+ 8 项死代码/依赖清理，消除用户可感知卡顿（地图聚类 O(n²)、dashboard N² 重渲染级联、键击整表重渲），selector 迁移收尾，附带修复 3 个正确性 bug。

**输入来源:**
- `.planning/reviews/20260911-frontend-perf-audit.md`（2026-09-11 前端全量性能审计，590 文件，4 并行代理按 Vercel React Best Practices 57 规则扫描）
- `.planning/PROJECT.md` v1.33 段（D-01~D-06 锁定决策）
- 前序背景: quick task `260911-m76`（PR #19）修了相邻 6 批次；本审计为其 merge 后全面扫描，findings 经抽查确认全部残留

**锁定决策 (v1.33 init):**

- **D-01 范围**: 用户确认全量 33 findings + 8 死代码清理，不分批 defer
- **D-02 七 gate 不倒退**: go build / go test / 后端 coverage ≥78.33 基线 / 前端 45 dirs / lint / type-check / diff coverage 全程保持绿
- **D-03 bundle 基线不倒退**: size-limit 门禁（entry gzip 1MB / 全量 2.5MB）保持；改动不得推高 entry
- **D-04 回归纪律**: 行为变更（BUGFIX-01..03 / DATA-02 缓存语义 / BUNDLE-02 功能修复）附回归测试；纯性能重构（selector 化 / useMemo / Map 索引）以现有测试零回归为准
- **D-05 复用范本**: info-points:603 Map 索引 / executions:112 columns useMemo / useRouteTabs:45 selector 风格 / noticeStore P1-M4 缓存出 state 四个既有正确范本为准
- **D-06 Phase 编号**: 从 Phase 114 续编（v1.32 用 109-113）

---

## v1.33 Requirements

### MAP3D — 3D 地图性能（H-2 全库最重 JS 热点）

- [ ] **MAP3D-01**: 地图聚类消除 O(n²) 双循环——单遍预计算 `Map<id,pixel>`，内层只做像素距离比较，消除内层 `new BMapGL.Point()` + `pointToOverlayPixel` 地图 API 调用（HubeiMap.tsx:267-318 与 HubeiMapGL 同步修复）
- [ ] **MAP3D-02**: 聚类按 40px 像素网格分桶（spatial hash）降为近 O(n)；n=1000 楼宇、缩放切换时无主线程长任务（秒级卡死清零）
- [ ] **MAP3D-03**: HubeiMap/HubeiMapGL 两份复制聚类算法合并为共享工具函数（单一实现 + 单元测试）
- [ ] **MAP3D-04**: HubeiMap 渲染体 5 道全量 filter（664/667/672/676/705）收敛为 useMemo 一次计算 `{level1, level2, withCoords}`（deps `[buildings]`）
- [ ] **MAP3D-05**: map 级事件监听（zoomend/tiltend）补 effect cleanup（removeEventListener）
- [ ] **MAP3D-06**: 死组件 BuildingMarkers.tsx / CityMarkers.tsx 删除（连带 DEAD-02 的 @uiw/react-baidu-map 依赖移除）

### DASH — Dashboard 重渲染级联（H-1）

- [ ] **DASH-01**: `useWidgetData`（hooks/useWidgetData.ts:70,115）改字段 selector 订阅（`s => s.cacheWidgetData` 等，action 引用稳定），widget memo 恢复有效性
- [ ] **DASH-02**: widget 数据 L1 缓存移出响应式 state（dashboardStore.ts:405-412 模块级 Map + getState 读写，noticeStore P1-M4 先例）或写入前数据比较跳过无变化写入
- [ ] **DASH-03**: dashboard 模块 9 处整店订阅改 selector（dashboard-system index.tsx:30 / DashboardHome:19 / DashboardList:47 / DashboardView:36 / DashboardEdit:43 / edit.tsx:41 / view.tsx:31 / WidgetEditor:33 / DashboardSettings:30）
- [ ] **DASH-04**: DashboardGrid.tsx:42,49 移除 useWindowSize 订阅（改 `useState(() => clientWidth)` 初始化一次）+ gridProps（layouts 包装/containerPadding/handleLayoutChange）useMemo 化

### SELECTOR — Zustand selector 收尾（26 处整店订阅）

- [ ] **SELECTOR-01**: `useTabs`（tabsStore.ts:310-326，解构 14 字段）内部改逐字段 selector（useRouteTabs:45-48 范本）
- [ ] **SELECTOR-02**: `useLayout`（layoutStore.ts:287-299，解构 10 字段）内部改逐字段 selector
- [ ] **SELECTOR-03**: 路由层 RouteGuard.tsx:33 / DynamicRoutes.tsx:105-106 改 selector 订阅（仅取所需字段，防 menuStore loading/lastFetchTime 变化重渲整页）
- [ ] **SELECTOR-04**: 3D 页 5 处 visualizationStore 整店订阅改 selector（building-spaces-3d index:36 / HubeiMap:73 / HubeiMapGL:86 / BuildingView3D:59 / FloorView3D:59）
- [ ] **SELECTOR-05**: 其余 action-only 整店订阅清理（my-notices/detail:19 / profile:47 / login:47-48 / my-duty:64 / DashboardScopeSelector:25）

### BUNDLE — Bundle 优化

- [ ] **BUNDLE-01**: ExcelImport 懒加载口径统一——9 个静态调用点（assets/buildings/server-rooms/room-devices/info-points/floors/dedicated-lines/dept/user）迁 ExcelImportLazy，或反向删除 Lazy 包装；二选一不留两套（phase 内定方向）
- [ ] **BUNDLE-02**: iconUtils.tsx:546-550 假动态导入删除（bare specifier Vite 不可分析、运行时必失败被 catch 静默吞掉）；静态注册表为准，兼菜单图标功能修复，附回归测试
- [ ] **BUNDLE-03**: 路由 glob（componentLoader.tsx:34-47）排除 `**/modals/**`、`**/components/**`、`**/hooks/**`，phantom chunk 清零（knowledge/articles/modals 等）
- [ ] **BUNDLE-04**: EChartsWrapper.tsx:22-28 误导性注释修正，或升级真懒加载（`@/lib/echarts` 挪进 wrapper 内同一 Promise.all）

### RENDER — 渲染性能

- [ ] **RENDER-01**: 7 处 columns 工厂静态化/useMemo 化（monitor/job:95,103 受控搜索键击整表重渲最高优先 / monitor/logs:203 / system/dict:463 / DetailDrawer:25 / executions:122 / VariablesModal:31 / LocationAliasDrawer:156；executions:112-118 范本）
- [ ] **RENDER-02**: 5+ 大数据页补 Table `virtual` + `scroll.y`（monitor/logs:370 / asset/reconciliation/exceptions:509 / network/devices / network/ports / operations/info-points；对照 assets:716 等 4 个已启用页）
- [ ] **RENDER-03**: MACHeatmapChart.tsx:118 移动端分支 top-k spread+sort 提 useMemo（desktop 分支已正确）
- [ ] **RENDER-04**: DoorElement.tsx:162,170,185,188 hingePoint/openEndPoint 补 snapCoord 精度处理（对齐 94-95 既有机制）

### DATA — 数据获取

- [ ] **DATA-01**: VDI 服务器列表 4 处裸调用（VirtualMachineList:136,468,484,653）归一 react-query `queryKey: ['vdi','servers']`（共享缓存去重）
- [ ] **DATA-02**: 菜单+权限带版本号持久化 sessionStorage，hydrate-then-revalidate 先渲染外壳再补拉，消除硬刷新整页门控（DynamicRoutes:190 InitializingFallback）；缓存语义变更附回归测试
- [ ] **DATA-03**: useColumnConfig.ts:122-135 缓存新鲜时 early-return（消除无条件网络请求）
- [ ] **DATA-04**: useHolidayData.ts:126-127 双独立 GET 改 Promise.all

### BUGFIX — 正确性修复（行为变更，附回归测试）

- [ ] **BUGFIX-01**: dedicated-lines/index.tsx:494（monthlyFee）+ FloorCardView.tsx:82（area）值为 0 时渲染字面 "0" 修复（改 `!= null &&`）
- [ ] **BUGFIX-02**: VariablesModal.tsx:31-53 同一 3 列定义重复渲染 6 列 bug 修复（去重列定义）
- [ ] **BUGFIX-03**: CADFloorPlanEditor.tsx:870-1028 updater 内 stale closure 修复（读 `prev.snapToGrid`/`prev.gridSize`，从 deps 删除 floorPlanData）

### MISC — JS 性能杂项

- [ ] **MISC-01**: useWorkstationView.ts:63-76 批量更新建 `Map<id,item>` 索引（拖拽 O(全集×批量) → O(全集)）
- [ ] **MISC-02**: TargetSelector.tsx:101-108 filterOption 建 Map 索引替代 find（每键击 O(n²) → O(n)）
- [ ] **MISC-03**: useTableManager.ts:172 useRef 急切 sessionStorage 读改惰性初始化（24 个列表页共享）
- [ ] **MISC-04**: TabBar.tsx:118-136,339 scroll 状态值比较，无变化跳过 setState
- [ ] **MISC-05**: workstations/index.tsx:657-666 expandedRowRender 内联回调 useCallback 化（HealthCard/WorkstationDeviceTable memo 生效）

### DEAD — 死代码与依赖卫生

- [ ] **DEAD-01**: 死代码删除：hooks/useTabSync.ts（含其测试）/ components/dashboard/DashboardView.tsx（含 useWidgetPolling 消费链核对）/ building-spaces-3d/utils.ts getWorkstationStats / executions/index.tsx:122 `_detailColumns` / components/operations/index.ts barrel
- [ ] **DEAD-02**: 僵尸依赖移除：cron-parser / @react-spring/three / maath / @uiw/react-baidu-map（依赖 MAP3D-06 先行）
- [ ] **DEAD-03**: vite.config.ts:23,169-171 react-markdown 过时注释修正

## 范围外（锁定 D-01/D-06）

| 项 | 理由 |
|----|------|
| 后端任何修改 | v1.33 仅前端；后端问题已在 v1.32 收尾 |
| 前端覆盖率推新目标 | v1.28 已收口 45.13% |
| 新业务功能 | 性能治理里程碑，不引入功能 |
| captcha barrel / layout 三布局 lazy 化 | 审计 LOW 可选项，收益小（entry 几十 KB），留观察 |
| lib/api.ts SM2/SM4 双 await Promise.all 化 | 本地同步计算，收益可忽略 |
| `server-*` Next.js 规则 | Vite SPA 不适用 |

## 回归纪律（锁定 D-04）

- **行为变更（红→绿）**: BUGFIX-01..03 / BUNDLE-02（iconUtils 功能修复）/ DATA-02（菜单缓存语义）——每个修复先有失败的测试再修复
- **纯性能重构（零回归）**: MAP3D/DASH/SELECTOR/BUNDLE-01/03/04/RENDER/DATA-01/03/04/MISC/DEAD——现有测试全绿 + 新增单元测试覆盖共享工具（MAP3D-03 聚类函数 / MISC-01/02 Map 索引）
- **gate 全程**: D-02 七 gate + D-03 size-limit；MAP3D-02 性断言以"无秒级长任务"人工验证 + 聚类结果一致性单元测试守护

## 进度追踪

Phase 映射由 roadmapper 填充（2026-09-12，`.planning/ROADMAP.md` Phases 114-120）。

| Requirement | Phase | Status |
|-------------|-------|--------|
| MAP3D-01 | Phase 114 | Pending |
| MAP3D-02 | Phase 114 | Pending |
| MAP3D-03 | Phase 114 | Pending |
| MAP3D-04 | Phase 114 | Pending |
| MAP3D-05 | Phase 114 | Pending |
| MAP3D-06 | Phase 114 | Pending |
| SELECTOR-01 | Phase 115 | Pending |
| SELECTOR-02 | Phase 115 | Pending |
| SELECTOR-03 | Phase 115 | Pending |
| SELECTOR-04 | Phase 115 | Pending |
| SELECTOR-05 | Phase 115 | Pending |
| DASH-01 | Phase 116 | Pending |
| DASH-02 | Phase 116 | Pending |
| DASH-03 | Phase 116 | Pending |
| DASH-04 | Phase 116 | Pending |
| RENDER-01 | Phase 117 | Pending |
| RENDER-02 | Phase 117 | Pending |
| RENDER-03 | Phase 117 | Pending |
| RENDER-04 | Phase 117 | Pending |
| BUGFIX-01 | Phase 117 | Pending |
| BUGFIX-02 | Phase 117 | Pending |
| BUGFIX-03 | Phase 117 | Pending |
| DATA-01 | Phase 118 | Pending |
| DATA-02 | Phase 118 | Pending |
| DATA-03 | Phase 118 | Pending |
| DATA-04 | Phase 118 | Pending |
| MISC-01 | Phase 119 | Pending |
| MISC-02 | Phase 119 | Pending |
| MISC-03 | Phase 119 | Pending |
| MISC-04 | Phase 119 | Pending |
| MISC-05 | Phase 119 | Pending |
| BUNDLE-01 | Phase 120 | Pending |
| BUNDLE-02 | Phase 120 | Pending |
| BUNDLE-03 | Phase 120 | Pending |
| BUNDLE-04 | Phase 120 | Pending |
| DEAD-01 | Phase 120 | Pending |
| DEAD-02 | Phase 120 | Pending |
| DEAD-03 | Phase 120 | Pending |

**Coverage:**
- v1.33 requirements: 38 total
- Mapped to phases: 38（Phase 114-120，每项恰好映射一个 phase）
- Unmapped: 0 ✓

---
*Requirements defined: 2026-09-11*
*Last updated: 2026-09-12 — roadmapper 填充进度追踪表（Phases 114-120），Coverage 38/38*
