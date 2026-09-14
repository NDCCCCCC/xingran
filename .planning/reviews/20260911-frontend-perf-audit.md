# XingRan 前端性能审计报告 (Vercel React Best Practices)

**审计日期:** 2026-09-11
**范围:** `xingran-react-frontend/src` 590 个源文件（pages 312 / components 132 / hooks 26 / store 9 等）
**方法:** 4 个并行审计代理按规则类别分工（数据获取 / Bundle / 重渲染 / 渲染+JS），Grep 全量扫描 + 逐文件 Read 确认，所有 finding 均有代码证据。Next.js 专属规则（server-*）不适用于 Vite SPA，已排除。
**规则来源:** Vercel React Best Practices 57 规则 / 8 类别

---

## 总体评价

代码库整体纪律性优于同类项目：

- ✅ 路由级 code splitting **73/73**（`import.meta.glob` + `lazy`，`router/componentLoader.tsx`）
- ✅ react-query v5 全局接入（staleTime 5min 去重，deptTree/dict/userOptions 收敛 canonical hooks）
- ✅ three/echarts/xlsx/markdown 零首屏泄漏；xlsx 4 处全动态 import；md-editor 用 nohighlight 子路径
- ✅ localStorage schema 全部 try-catch + 版本号 + partialize；事件监听清理规范
- ✅ `.size-limit.json` 门禁（entry gzip 1MB）+ manualChunks 按依赖族切分

问题集中在 3 个热点模块（dashboard / 3D 地图 / TabBar）和 selector 迁移未竟的尾部。

---

## HIGH（2 项）

### H-1 | rerender-defer-reads | dashboardStore 缓存写入 N² 重渲染级联

- **Location:** `src/hooks/useWidgetData.ts:70` + `src/store/dashboardStore.ts:405-412` + 7 个 widget 组件
- **Evidence:** `useWidgetData` 无 selector 全量订阅 store；每次数据到达无条件 `cacheWidgetData()` → 每次 `new Map(state.widgetDataCache)` 生成全新 store state → 全部全量订阅者重渲。widget 上的 `memo()` 完全失效（重渲来自自身 store 订阅，不走 props）。`timestamp: Date.now()` 使去重不可能。
- **Impact:** N widget 仪表盘每个轮询周期（默认 60s）≈ N² 次组件重渲（含 DashboardView / 路由包装层 / DashboardHome / WidgetEditor / DashboardSettings 及其余 N-1 个 widget）。
- **Fix:** ① `useWidgetData` 改字段 selector（`s => s.cacheWidgetData`，action 引用稳定）；② 照 `noticeStore` P1-M4 先例把 L1 缓存移出响应式 state（模块级 Map + getState 读写）；③ 写入前比较数据跳过无变化写入；④ dashboard 模块 9 处整店订阅改 selector。

### H-2 | js-set-map-lookups | HubeiMap 聚类 O(n²) 双循环 + 内层地图 API 调用

- **Location:** `src/pages/operations/building-spaces-3d/components/HubeiMap.tsx:281-318` + `HubeiMapGL.tsx:282-330`（同一算法复制两份），deps `[mapLoaded, buildings, currentZoom]`
- **Evidence:** 外层 forEach 内嵌套 forEach，内层每个 otherBuilding 都 `new BMapGL.Point()` + `map.pointToOverlayPixel()`（地图 API 调用）。`index.tsx:18` `PAGE_SIZE = 1000` 全量拉取。
- **Impact:** n=1000 时最多 100 万对比较 × ~200 万次 pointToOverlayPixel；每次缩放级别切换整段重跑 → 主线程长任务、地图卡死数秒。全库最重 JS 热点。
- **Fix:** 单遍循环预计算 `Map<id, pixel>`，内层只做像素距离比较；进一步按 40px 像素网格分桶（spatial hash）降为近 O(n)。两文件合并共享工具函数。

---

## MEDIUM（8 项）

### M-1 | bundle-conditional | ExcelImportLazy 只有 1/10 调用点在用

- **Location:** `src/components/shared/ExcelImportLazy.tsx`（包装器已建好）vs 9 个静态 import 点：`pages/operations/assets/index.tsx:24`、`buildings:46`、`server-rooms:47`、`room-devices:51`、`info-points:53`、`floors:17`、`dedicated-lines:50`、`system/dept:20`、`system/user:38`。唯一用 Lazy 的是 workstations。
- **Fix:** 9 个调用点统一替换 `ExcelImportLazy`（props 完全兼容）；或反向删除 Lazy 包装。二选一，不留两套。

### M-2 | bundle-dynamic-imports | iconUtils 假动态导入（运行时必失败，兼功能缺陷）

- **Location:** `src/utils/iconUtils.tsx:546-550`
- **Evidence:** `import(\`@ant-design/icons/${iconName}\`)` bare specifier Vite 无法静态分析 → 不生成 chunk，运行时浏览器无法解析 bare specifier → `.catch` 静默返回 null。静态注册表（~90+ 图标）之外的菜单图标永远渲染为空且无人察觉。
- **Fix:** 删除动态分支（静态注册表已覆盖实际需求）；确需扩展用 `import.meta.glob` 或约束后端图标名进静态表。

### M-3 | rerender-defer-reads | useTabs/useLayout 便捷 hook 全店订阅

- **Location:** `src/store/tabsStore.ts:310-326`（解构 14 字段）+ `layoutStore.ts:287-299`（解构 10 字段）
- **Consumers:** `TabBar.tsx:55`（双布局常驻）、`useTabSync.ts:38`、`LayoutSwitcher.tsx:13`、`DensitySwitcher.tsx:41`
- **Impact:** tabsStore 任何写入（含 history 持久化）重渲常驻 TabBar；layoutStore 任何写入重渲 Header 开关。`useShallow` 全项目 0 使用。
- **Fix:** 内部改逐字段 selector（参照 `useRouteTabs.ts:45-48` 正确写法）。

### M-4 | rerender-defer-reads | 路由层整店订阅（潜伏级联）

- **Location:** `router/RouteGuard.tsx:33`（只取 permissions 但包裹受保护子树）、`router/DynamicRoutes.tsx:105-106`（menuStore+authStore 整店，包裹全部已认证路由）
- **Impact:** menuStore `loading`/`lastFetchTime`/`error` 变化即重渲整页。另 3D 页 5 处 visualizationStore 整店订阅（cameraPosition/cameraTarget 现无调用方，接相机跟踪前必须先改 selector）。
- **Fix:** 改 selector 订阅。

### M-5 | rendering-hoist-jsx | 7 处 columns 工厂每渲染重建

- **Location:** `monitor/job/index.tsx:95,103`（103 行零参数纯静态 + 受控搜索每键击触发整表重渲，最高优先）、`monitor/logs:203-210`、`system/dict:463-479`、`network/executions/modals/DetailDrawer.tsx:25`、`executions/index.tsx:122`（`_detailColumns` 死代码）、`templates/modals/VariablesModal.tsx:31-53`（**同一 3 列定义重复两次 = 6 列渲染 bug**）、`workstations/LocationAliasDrawer.tsx:156`
- **Impact:** antd Table 对 columns 引用比较，新 identity → 每次父渲染整表 body 重渲（pageSize 可达 200）。
- **Fix:** 零参数工厂提为模块级常量；其余 useMemo 包裹（logs 页 handleViewDetail 已 useCallback）。对照 `executions/index.tsx:112-118` 已正确 useMemo。

### M-6 | client-swr-dedup | VDI 服务器列表 4 处重复请求

- **Location:** `pages/vdi/VirtualMachineList/index.tsx:136, 468, 484, 653`
- **Evidence:** `vdiServerApi.list({current:1, pageSize:100})` 4 处独立调用；136 行 preload 结果被丢弃只用 find；468 行注释自认"还是需要调用API"——缓存命中仍重发。
- **Fix:** react-query（`queryKey: ['vdi','servers']`）替换 4 处裸调用。

### M-7 | rendering-content-visibility | 虚拟滚动覆盖缺口（5+ 大数据页）

- **Location:** 已启用（4 页参照）：`operations/assets:716`、`workstations:621`、`vdi/VirtualMachineList:952`、`network/mac/history:419` + `MACEventsTimeline.tsx:178` contentVisibility。未启用：`monitor/logs:370-412`（双 Table，最典型审计长列表）、`asset/reconciliation/exceptions:509`、`network/devices`、`network/ports`、`operations/info-points`
- **Impact:** pageSize 上限 200 兜底，非 P0；但拉满 + 宽表（scroll.x 1500）时 DOM 数千节点掉帧。
- **Fix:** 补 `virtual + scroll.y`，复制现成模式零新依赖。

### M-8 | async-suspense-boundaries | 菜单加载整页门控

- **Location:** `router/DynamicRoutes.tsx:190-193` + `store/menuStore.ts:60-76` + `services/cache/TTLMenuCache.ts:17`
- **Evidence:** `if (allMenus.length === 0) return <InitializingFallback />` 整屏 Spin 门控所有已认证 UI；TTLMenuCache 纯内存 Map，硬刷新必重新请求。
- **Fix:** 菜单+权限带版本号持久化 sessionStorage 做 hydrate-then-revalidate。

### 次级 MEDIUM（3 项）

- **async-defer-await:** `hooks/useColumnConfig.ts:122-135` — 缓存命中仍无条件 `await getByPageKey()`；改 early-return 或 react-query initialData
- **rerender-use-ref-transient-values:** `components/dashboard/layout/DashboardGrid.tsx:42,49` — `useWindowSize()` 仅用于 useState 初值，resize 每 tick 双重渲染；gridProps（layouts 包装对象/containerPadding/handleLayoutChange）每渲染重建未 memo
- **js-combine-iterations:** `HubeiMap.tsx:664-705` 渲染体 5 遍全量 filter（"有坐标"谓词重复 3 次），hover 即重算；组件裸导出无 memo

---

## LOW（12 项）

| # | 规则 | 位置 | 问题 |
|---|------|------|------|
| L-1 ⚠️正确性 | rendering-conditional-render | `operations/dedicated-lines/index.tsx:494`（monthlyFee?: number）+ `floors/components/FloorCardView.tsx:82`（area?: number） | 值为 0 时渲染字面 `"0"`；改 `!= null &&`。全库其余 189 处 `&&` 左值均为 boolean/object/string 无此风险 |
| L-2 ⚠️正确性 | （顺带） | `network/templates/modals/VariablesModal.tsx:31-53` | 同一 3 列定义重复出现两次 = **6 列渲染 bug**（见 M-5） |
| L-3 | js-set-map-lookups | `operations/workstations/hooks/useWorkstationView.ts:63-76` | 楼层全集（无分页端点，可上千）× 批量更新项 map+find；拖拽 10 万次比较。改 `new Map(items.map(...))` |
| L-4 | js-set-map-lookups | `system/notice/components/TargetSelector.tsx:101-108` | filterOption 内 users.find（pageSize 100）每键击 O(n²)；`info-points/index.tsx:1097` 同型（pageSize 50）。改 Map 索引 |
| L-5 | rerender-lazy-state-init | `hooks/useTableManager.ts:172` | useRef 急切初始化每次渲染读 sessionStorage（`readInitialFilters`，24 个列表页在用）。改 useState(() => ...) |
| L-6 | rerender-functional-setstate | `cad-editor/CADFloorPlanEditor.tsx:870-1028` | updater 内读闭包 `floorPlanData.snapToGrid/gridSize`（986-995）+ deps 含 floorPlanData（1022）→ 拖拽每帧重建回调 + stale-closure 隐患。改读 prev.* |
| L-7 | js-tosorted-immutable | `network/MACHeatmapChart.tsx:118` | isMobile 分支 render 体内 `[...cells].sort().slice(0,20)` 未 memo（desktop 分支已正确 useMemo）。提 useMemo |
| L-8 | rendering-svg-precision | `cad-elements/DoorElement.tsx:162,170,185,188` | 圆弧路径 hingePoint/openEndPoint 裸浮点未过 snapCoord（94-95 行既有机制残留遗漏） |
| L-9 | async-parallel | `duty/management/hooks/useHolidayData.ts:126-127` | `await fetchList(); await fetchYears();` 串行独立 GET。改 Promise.all |
| L-10 | client-event-listeners | `HubeiMap.tsx:150-151` / `HubeiMapGL.tsx` 同型 | map.addEventListener("zoomend"/"tiltend") 无 cleanup（实例随卸载丢弃，泄漏有界）。effect 返回 removeEventListener |
| L-11 | rerender-transitions | `layout/shared/TabBar.tsx:118-136,339` | scroll/resize 每 tick setState 新对象（已用 startTransition，缺值比较）。checkScrollState 结果浅比较跳过 |
| L-12 | rerender-memo | `workstations/index.tsx:657-666` | Table expandedRowRender 内 memo 的 HealthCard 收内联 onApplyException / WorkstationDeviceTable 收内联 onBadgeClick。useCallback 化 |

（`lib/api.ts:262` SM2/SM4 双 await 为本地同步计算可忽略；`components/captcha` barrel 三布局进 entry 为 LOW 可选项，归入死代码清理批次顺带。）

---

## 死代码与依赖卫生（7 项，零运行时成本）

1. `hooks/useTabSync.ts` — 已被 `useRouteTabs` 取代，仅测试引用，内部整店订阅反模式
2. `components/dashboard/DashboardView.tsx` — 无任何 importer（注意与 `pages/dashboard-system` 下同名区分），是 `useWidgetPolling` 唯一消费者
3. `building-spaces-3d/components/BuildingMarkers.tsx` + `CityMarkers.tsx` — 零消费方，连带 `@uiw/react-baidu-map` 僵尸依赖
4. `components/operations/index.ts` barrel — 0 消费方
5. `building-spaces-3d/utils.ts:355-368` `getWorkstationStats` — 5 遍 filter 死代码
6. `package.json`：`cron-parser`、`@react-spring/three`、`maath` 零引用；`vite.config.ts:23,169-171` 注释按不存在的 `react-markdown` 描述
7. 路由 glob `**/modals/index.tsx` 误匹配 `knowledge/articles/modals/index.tsx`（modal barrel 非路由页）产出 phantom chunk — glob 加排除 `!**/modals/**` / `!**/components/**` / `!**/hooks/**`
8. `network/executions/index.tsx:122` `_detailColumns` 死变量（见 M-5）

---

## 合规确认（无需整改）

| 规则 | 结论 |
|------|------|
| client-passive-event-listeners | PASS — 仅 2 个 wheel 用 passive:false 且均 preventDefault（画布缩放必需）；scroll 均 passive:true |
| client-localstorage-schema | PASS — JSON.parse 全 try-catch；4 个 persist store 全 partialize + version |
| rerender-dependencies | PASS — 88 种 dep 数组逐一核对无内联对象/数组 |
| rerender-derived-state | PASS — 唯一 effect→ref 镜像是正确模式 |
| js-hoist-regexp | PASS — 0 命中 |
| js-min-max-loop | PASS — 0 命中 |
| rendering-animate-svg-wrapper | PASS — 0 命中 |
| async-parallel（主体） | PASS — 登录页已 Promise.all；info-points 级联为层级依赖正确串行 |

**可复用的正确范本:** `info-points/index.tsx:603-609`（useMemo+Map 索引）、`executions/index.tsx:112-118`（columns useMemo）、`useRouteTabs.ts:45-48`（selector 风格）、`useBackupDiff.ts:70-95`（ref 滚动同步）、`noticeStore` P1-M4（缓存移出响应式 state）。

**验证手段:** `ANALYZE=true npm run build`（stats.html）/ `npm run size`（size-limit 门禁）/ `npm run deadcode`（knip）/ 七 gate（go build / go test / 后端 coverage / 前端 45 dirs / lint / type-check / diff coverage）。

---

## 建议修复顺序

1. H-2 + M-5（用户可感知卡顿：地图聚类 + monitor/job 键击整表重渲）
2. H-1 + M-3 + M-4（dashboard 级联 + selector 收尾 26 处整店订阅）
3. M-1 + M-2（ExcelImportLazy 推广 + iconUtils 假动态导入删除——兼功能修复）
4. LOW + 死代码清理（npm run deadcode 复核）
