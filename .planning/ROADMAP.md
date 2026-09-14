# Roadmap: XingRan-Next — v1.33 V133 前端性能治理 (Frontend Performance Remediation)

## Milestones

- ✅ v1.0 – v1.32 — 已交付里程碑见 `.planning/MILESTONES.md` 与 `.planning/milestones/`（v1.32: Phases 109-113，SHIPPED 2026-09-09）
- 🚧 **v1.33 前端性能治理 (Frontend Performance Remediation)** — Phases 114-120（进行中）

**Milestone Goal:** 修复 2026-09-11 前端全量性能审计（`.planning/reviews/20260911-frontend-perf-audit.md`）全部 33 项 findings（2 HIGH + 8 MEDIUM + 3 次级 MEDIUM + 12 LOW）+ 8 项死代码/依赖清理 + 3 个正确性 bug；消除用户可感知卡顿（地图聚类 O(n²)、dashboard N² 重渲染级联、键击整表重渲），selector 迁移收尾。范围仅前端 `xingran-react-frontend/`，不改后端，不引入新业务功能。

**锁定决策（v1.33 init，详见 PROJECT.md / REQUIREMENTS.md）:**

- **D-01** 全量 33 findings + 8 死代码清理，不分批 defer
- **D-02** 七 gate 不倒退（go build / go test / 后端 coverage ≥78.33 / 前端 45 dirs / lint / type-check / diff coverage）
- **D-03** bundle 基线不倒退（size-limit 门禁 entry gzip 1MB / 全量 2.5MB；改动不得推高 entry）
- **D-04** 行为变更（BUGFIX-01..03 / BUNDLE-02 / DATA-02）附回归测试；纯性能重构以现有测试零回归为准
- **D-05** 复用范本：info-points:603 Map 索引 / executions:112 columns useMemo / useRouteTabs:45 selector 风格 / noticeStore P1-M4 缓存出 state
- **D-06** Phase 编号从 114 续编（v1.32 用 109-113，v1.31 用 102-108）

## Phases

**Phase Numbering:**

- Integer phases (114-120): Planned milestone work（D-06 续编，不 reset）
- Decimal phases: 本里程碑暂无（如需紧急插入用 `/gsd:phase insert`）

- [x] **Phase 114: map3d-clustering** - 地图聚类 O(n²) 消除（H-2）：Map 预计算 + 40px 像素网格分桶 + 双实现合并共享 + filter useMemo + 事件 cleanup + 死组件删除 (completed 2026-09-11)
- [x] **Phase 115: selector-completion** - Zustand selector 收尾：useTabs/useLayout 内部 selector 化 + 路由层 + 3D 页 5 处 + action-only 整店订阅清零 (completed 2026-09-14)
- [ ] **Phase 116: dashboard-cascade** - Dashboard N² 重渲染级联消除（H-1）：useWidgetData selector 化 + L1 缓存出 state + 9 处订阅收敛 + DashboardGrid 稳定化
- [ ] **Phase 117: render-columns-and-bugfix** - 渲染热点治理 + 正确性修复：7 处 columns 工厂记忆化 + Table virtual 补齐 + "0" 渲染 / VariablesModal 6 列 / CAD stale closure（附回归测试）
- [ ] **Phase 118: data-fetch** - 数据获取治理：VDI react-query 去重 + 菜单 hydrate-then-revalidate 消除整页门控 + 列配置缓存短路 + Promise.all 合并
- [ ] **Phase 119: misc-js-perf** - JS 微性能杂项：Map 索引（拖拽/搜索）+ 惰性 sessionStorage + scroll 状态短路 + expandedRowRender useCallback
- [ ] **Phase 120: bundle-dead-cleanup** - Bundle 优化 + 死代码/依赖卫生：ExcelImport 口径归一 + iconUtils 假动态导入删除 + phantom chunk 清零 + 4 僵尸依赖移除 + 5 处死代码删除

## Phase Details

### Phase 114: map3d-clustering（地图聚类 O(n²) 消除）

**Goal**: 湖北地图（HubeiMap/HubeiMapGL）在千级楼宇点位与缩放/倾斜切换时不再出现秒级主线程卡死——聚类算法单遍化 + 像素网格分桶，两份复制实现合并为单一共享函数并有单元测试守护（H-2 全库最重 JS 热点）
**Depends on**: Nothing (first phase)
**Requirements**: MAP3D-01, MAP3D-02, MAP3D-03, MAP3D-04, MAP3D-05, MAP3D-06
**Success Criteria** (what must be TRUE):

  1. 聚类为单遍 Map 预计算 + 40px 像素网格分桶：内层循环无 `new BMapGL.Point()` / `pointToOverlayPixel` 地图 API 调用；n=1000 楼宇缩放切换无秒级长任务（人工性能验证 + 聚类结果一致性单元测试通过，D-04 口径）
  2. HubeiMap 与 HubeiMapGL 共享同一聚类工具函数（单一实现 + 单元测试），两页聚类视觉结果一致
  3. HubeiMap 渲染体 5 道全量 filter（664/667/672/676/705）收敛为 useMemo 一次计算（deps `[buildings]`），buildings 未变时不重复过滤
  4. map 级 zoomend/tiltend 事件监听在组件卸载后不再触发（effect cleanup removeEventListener 生效）
  5. BuildingMarkers.tsx / CityMarkers.tsx 死组件已删除、全库无引用（为 Phase 120 的 @uiw/react-baidu-map 依赖移除解锁）

**Plans:** 3/3 plans complete

Plans:
**Wave 1**

- [x] 114-01-PLAN.md — 共享聚类纯函数 cluster.ts（40px 网格分桶）+ 参考实现对照一致性测试（RED→GREEN）

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 114-02-PLAN.md — HubeiMap/HubeiMapGL 改造：单遍预计算 + 共享函数消费 + 5 filter useMemo + 事件 cleanup

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 114-03-PLAN.md — 死组件删除（解锁 @uiw 依赖移除）+ cleanup spy 测试 + MAP3D-02 人工性能验证 checkpoint

### Phase 115: selector-completion（Zustand selector 收尾）

**Goal**: 全库剩余整店订阅清零（PROJECT 口径 26 处）——任意 store 切片变化只重渲真正消费该字段的组件，键击/弹窗等高频路径不再被无关 store 变化牵连重渲
**Depends on**: Phase 114（SELECTOR-04 与 MAP3D 同模块——3D 页文件紧随改完，减少文件冲突；无硬数据依赖）
**Requirements**: SELECTOR-01, SELECTOR-02, SELECTOR-03, SELECTOR-04, SELECTOR-05
**Success Criteria** (what must be TRUE):

  1. useTabs（tabsStore:310-326，14 字段解构）/ useLayout（layoutStore:287-299，10 字段解构）内部改为逐字段 selector（useRouteTabs:45 范本）：tabs/layout 无关字段变化不再触发消费组件重渲
  2. 路由层 RouteGuard:33 / DynamicRoutes:105-106 仅订阅所需字段：menuStore loading/lastFetchTime 变化不引发整页重渲
  3. 3D 页 5 处 visualizationStore 整店订阅全部为字段级 selector（building-spaces-3d index:36 / HubeiMap:73 / HubeiMapGL:86 / BuildingView3D:59 / FloorView3D:59）
  4. 5 处 action-only 整店订阅清零（my-notices/detail:19 / profile:47 / login:47-48 / my-duty:64 / DashboardScopeSelector:25）
  5. `useXxxStore()` 无参整店订阅 grep 归零；现有测试 + lint/type-check 零回归（纯性能重构，D-04 零回归口径）

**Plans:** 3/3 plans complete

**Wave 1**

- [x] 115-01-PLAN.md — useTabs（14 字段）/useLayout（10 字段 + 派生/effect 保留）hook 内部逐字段 selector 化（SELECTOR-01/02）
- [x] 115-02-PLAN.md — 路由层 RouteGuard:33 + DynamicRoutes:105-106（5 selector，权限逻辑/数据流零改动）+ 3D 页 5 处 visualizationStore（SELECTOR-03/04）

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 115-03-PLAN.md — 页面级 6 处单字段清理 + NotificationBell 纳入（OQ-1 定案 a）+ 4 mock dual-form 化 + phase gate grep/全量测试/lint/type-check（SELECTOR-05）

### Phase 116: dashboard-cascade（Dashboard 重渲染级联消除）

**Goal**: 可配置仪表盘（/dashboard-system）打开与数据刷新不再出现 N² 重渲染级联——单个 widget 数据到达只重渲该 widget，dashboard 模块整店订阅收敛，拖拽布局期间不重渲全部 widget（H-1）
**Depends on**: Phase 115（selector 范式先行，dashboard 订阅改造复用同一风格）
**Requirements**: DASH-01, DASH-02, DASH-03, DASH-04
**Success Criteria** (what must be TRUE):

  1. useWidgetData（hooks/useWidgetData.ts:70,115）改字段 selector 订阅（action 引用稳定）：widget memo 恢复有效，单个 widget 数据更新只重渲该 widget
  2. widget 数据 L1 缓存移出响应式 state（dashboardStore:405-412 模块级 Map + getState 读写，noticeStore P1-M4 先例）或写入前数据比较跳过无变化写入：轮询刷新不级联重渲无关 widget
  3. dashboard 模块 9 处整店订阅全部改 selector（index:30 / DashboardHome:19 / DashboardList:47 / DashboardView:36 / DashboardEdit:43 / edit:41 / view:31 / WidgetEditor:33 / DashboardSettings:30）
  4. DashboardGrid（:42,49）不再订阅窗口尺寸（clientWidth 初始化一次），gridProps（layouts 包装/containerPadding/handleLayoutChange）useMemo 化：拖拽调整期间 layouts 稳定时回调引用稳定

**Plans**: 3/3 plans created

### Phase 117: render-columns-and-bugfix（渲染热点治理 + 正确性修复）

**Goal**: 受控搜索/弹窗等高频交互不再键击整表重渲，大数据页表格启用虚拟滚动；顺带修复 3 个正确性 bug（"0" 渲染 ×2、VariablesModal 6 列、CAD stale closure），全部行为变更附回归测试（D-04 红→绿）
**Depends on**: Nothing（与 114-116 无文件交集，可并行执行；BUGFIX-02 与 RENDER-01 同文件 VariablesModal.tsx，收敛本 phase 内聚）
**Requirements**: RENDER-01, RENDER-02, RENDER-03, RENDER-04, BUGFIX-01, BUGFIX-02, BUGFIX-03
**Success Criteria** (what must be TRUE):

  1. 7 处 columns 工厂静态化/useMemo 化（executions:112-118 范本；monitor/job:95,103 受控搜索最高优先）：每键击不再整表重渲；VariablesModal 列定义去重后恰好渲染 3 列而非 6 列（回归测试锁定 BUGFIX-02）
  2. 5+ 大数据页（monitor/logs:370 / asset/reconciliation/exceptions:509 / network/devices / network/ports / operations/info-points）启用 Table `virtual` + `scroll.y`：大数据量滚动流畅（对齐 assets:716 既有 4 个已启用页）
  3. MACHeatmapChart:118 移动端分支 top-k spread+sort 提 useMemo（desktop 分支已正确）；DoorElement（:162,170,185,188）hingePoint/openEndPoint 补 snapCoord 精度处理（对齐 94-95 既有机制）
  4. dedicated-lines:494 monthlyFee=0 与 FloorCardView:82 area=0 正确渲染数值 0 而非空白（改 `!= null` 判断，回归测试锁定 BUGFIX-01）
  5. CADFloorPlanEditor（:870-1028）updater 读 `prev.snapToGrid`/`prev.gridSize` 并从 deps 删除 floorPlanData：snap 操作始终使用最新设置、无 stale closure（回归测试锁定 BUGFIX-03）

**Plans**: TBD（预估 4）

### Phase 118: data-fetch（数据获取治理）

**Goal**: 重复请求归一、硬刷新不再整页门控、缓存新鲜时不击穿——数据获取层去掉可感知的等待与冗余流量
**Depends on**: Phase 115（DATA-02 与 SELECTOR-03 同触 DynamicRoutes.tsx，selector 改动先落避免同文件冲突）
**Requirements**: DATA-01, DATA-02, DATA-03, DATA-04
**Success Criteria** (what must be TRUE):

  1. VDI 服务器列表 4 处裸调用（VirtualMachineList:136,468,484,653）归一 react-query `queryKey: ['vdi','servers']`：同页多组件共享缓存只发 1 次网络请求（网络面板可观察）
  2. 硬刷新时菜单+权限带版本号从 sessionStorage hydrate 立即渲染外壳、后台 revalidate 补拉：InitializingFallback（DynamicRoutes:190）整页门控消除（缓存语义变更，回归测试锁定 DATA-02）
  3. useColumnConfig（:122-135）缓存新鲜时 early-return：不再每次进页无条件发网络请求
  4. useHolidayData（:126-127）双独立 GET 合并 Promise.all：请求数 2→1，首屏数据等待缩短

**Plans**: 3/3 plans created

**Wave 1**

- [x] 118-01-PLAN.md — DATA-01 VDI react-query 去重 + DATA-04 Promise.all 合并（纯性能，零回归）
- [x] 118-02-PLAN.md — DATA-02 sessionStorage hydrate-then-revalidate + TDD 回归测试（行为变更）
- [x] 118-03-PLAN.md — DATA-03 useColumnConfig 缓存短路 early-return（纯性能，零回归）

### Phase 119: misc-js-perf（JS 微性能杂项）

**Goal**: 拖拽、搜索过滤、列表初始化、Tab 滚动等高频微交互的 O(n²) 与急切开销清零——共享工具带单元测试守护
**Depends on**: Nothing（与其他 phase 无文件交集，可并行执行）
**Requirements**: MISC-01, MISC-02, MISC-03, MISC-04, MISC-05
**Success Criteria** (what must be TRUE):

  1. useWorkstationView（:63-76）批量拖拽更新建 `Map<id,item>` 索引：O(全集×批量) → O(全集)（Map 索引单元测试覆盖，D-05 info-points:603 范本）
  2. TargetSelector（:101-108）filterOption 建 Map 索引替代 find：每键击 O(n²) → O(n)（单元测试覆盖）
  3. useTableManager（:172）useRef 急切 sessionStorage 读改惰性初始化：24 个共享列表页首帧不再同步读 storage
  4. TabBar（:118-136,339）scroll 状态值比较：无变化跳过 setState，无冗余重渲
  5. workstations/index.tsx（:657-666）expandedRowRender 内联回调 useCallback 化：HealthCard / WorkstationDeviceTable memo 生效，展开行操作不重渲已渲染行

**Plans**: 3/3 plans created

### Phase 120: bundle-dead-cleanup（Bundle 优化 + 死代码/依赖卫生）

**Goal**: bundle 口径归一与构建产物卫生（ExcelImport 懒加载统一、phantom chunk 清零）+ 全部死代码与僵尸依赖移除；entry gzip 基线不推高（D-03 size-limit 门禁保持绿）
**Depends on**: Phase 114（DEAD-02 依赖 MAP3D-06 先删 BuildingMarkers/CityMarkers 死组件）；建议在 Phase 116/117 之后执行（DEAD-01 的 DashboardView.tsx / executions `_detailColumns` 在对应 selector/columns 改造完成后删除，避免同文件二次翻动）
**Requirements**: BUNDLE-01, BUNDLE-02, BUNDLE-03, BUNDLE-04, DEAD-01, DEAD-02, DEAD-03
**Success Criteria** (what must be TRUE):

  1. ExcelImport 懒加载口径唯一：9 个静态调用点（assets/buildings/server-rooms/room-devices/info-points/floors/dedicated-lines/dept/user）归一（迁 ExcelImportLazy 或反向删 Lazy，方向 phase 内二选一定案不留两套）；size-limit 门禁保持绿、entry gzip 不推高
  2. iconUtils（:546-550）假动态导入删除（bare specifier Vite 不可分析、运行时必失败被静默吞掉）：菜单图标经静态注册表正常渲染——兼功能修复，回归测试锁定（D-04）
  3. 路由 glob（componentLoader:34-47）排除 `**/modals/**`、`**/components/**`、`**/hooks/**`：phantom chunk 清零（knowledge/articles/modals 等不再产出独立 chunk）；EChartsWrapper（:22-28）误导性注释修正或升级真懒加载（phase 内定）
  4. 僵尸依赖 cron-parser / @react-spring/three / maath / @uiw/react-baidu-map 从 package.json 移除：npm install + build + type-check 全绿
  5. 死代码清零：hooks/useTabSync（含其测试）/ components/dashboard/DashboardView.tsx（含 useWidgetPolling 消费链核对）/ building-spaces-3d getWorkstationStats / executions `_detailColumns` / components/operations barrel 删除、全库无引用；vite.config（:23,169-171）react-markdown 过时注释修正；现有测试零回归

**Plans**: TBD（预估 4）

## Progress

**Execution Order:**
Phases execute in numeric order: 114 → 115 → 116 → 117 → 118 → 119 → 120
（Phase 117 / 119 与前序无文件交集可并行；120 收尾必须最后——DEAD-02 依赖 114 的 MAP3D-06，DEAD-01 建议在 116/117 改造完成后执行）

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 114. map3d-clustering | 3/3 | Complete    | 2026-09-14 |
| 115. selector-completion | 3/3 | Complete    | 2026-09-14 |
| 116. dashboard-cascade | 0/3 | Not started | - |
| 117. render-columns-and-bugfix | 0/TBD | Not started | - |
| 118. data-fetch | 0/TBD | Not started | - |
| 119. misc-js-perf | 0/TBD | Not started | - |
| 120. bundle-dead-cleanup | 0/TBD | Not started | - |

**Coverage:** 38/38 v1.33 requirements mapped ✓（明细见 `.planning/REQUIREMENTS.md` 进度追踪表）

**回归纪律（D-04）:** 行为变更（BUGFIX-01..03 / BUNDLE-02 / DATA-02）红→绿附回归测试；纯性能重构（其余 33 项）现有测试零回归 + 新增单元测试覆盖共享工具（MAP3D-03 / MISC-01 / MISC-02）
