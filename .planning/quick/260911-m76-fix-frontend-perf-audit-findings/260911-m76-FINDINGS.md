# 前端性能审计修复任务清单（quick 260911-m76）

来源：Vercel React Best Practices 全量审计（6 个并行审计 agent）+ 主会话逐条人工验证（2026-09-11）。
所有路径相对 `xingran-react-frontend/`。标注 ✅ 的条目已经主会话逐行核实（file:line 与证据确认无误），◐ 为抽查采信。

## 全局约束（所有执行 agent 必须遵守）

1. **验证三件套**：每批次完成后在 `xingran-react-frontend/` 下运行 `npm run type-check && npm run lint`；涉及行为的改动跑相关 `npx vitest run <文件>`；本 quick 最后统一跑 `npm run build` + `npm test`。
2. **提交规范**：conventional commits（项目有 commitlint），每批次原子提交，只提交本批次涉及的文件。**禁止提交** `internal/services/system/asset_columns_schema.json`（主仓根有一个无关的未提交改动，不要卷入）。
3. **遵循项目 CLAUDE.md**：前端 API 走 `@/lib/apiFactory`/既有封装；不引入新依赖（除零代码的删除）；不留兼容壳/死代码。
4. **不扩大范围**：只修清单内问题；发现清单外的疑似问题记录到 SUMMARY 的 deferred 节，不顺手修。
5. 改动风格与周围代码一致（中文注释惯例、既有 selector 写法参照 `src/components/ConfigProvider.tsx:31-32`、`src/design-system/components/ThemeProvider.tsx:26`）。

---

## 批次 1：build-bundle（首屏体积，CRITICAL）

目标：首屏 JS 从 ~1.43MB gzip 降至 ~690KB。纯配置 + 图表入口改动，零业务逻辑变化。

### 1.1 vite.config.ts 删除 echarts-for-react 特殊分支 ✅
- 位置：`vite.config.ts:189-191`（`if (id.includes("echarts-for-react")) return "vendor-react";`）
- 修法：删除该分支，让 echarts-for-react 落入后面的 `pkgName === "echarts" || "zrender"` 规则所在 chunk（vendor-echarts）。vendor-echarts→vendor-react 单向边，不会成环（echarts-for-react 依赖 react，被归入 vendor-echarts 后产生 vendor-echarts→vendor-react 边，仍为 DAG）。
- 注意：同步更新 187-191 行附近的注释（原注释解释已失效）。

### 1.2 vite.config.ts 把 Vite preload helper 独立成小 chunk ✅
- 背景：构建产物中 `__vitePreload` helper（被 entry 以 `import{_ as f}from"./vendor-three-*.js"` 引用）被 Rollup 分配进了 vendor-three，导致 entry 静态依赖 vendor-three（911KB 首屏加载）。
- 修法：manualChunks 函数开头（`if (!id.includes("node_modules")) return undefined;` 之前或之后均可，但要在所有其他分支前）加：
  ```ts
  // Vite preload helper（\0vite/preload-helper.js）被所有动态 import 共享，
  // 单独成 chunk 阻止它被分配进 vendor-three 造成 entry→vendor-three 静态边
  if (id.includes("preload-helper")) return "runtime-helper";
  ```
- 验证：改后 `npm run build`，检查 `dist/index.html` 的 modulepreload 不再包含 vendor-three。

### 1.3 vite.config.ts 修 @uiw/ 兜底钉死 markdown 生态 ✅
- 位置：`vite.config.ts:169-171`（`if (pkgName.startsWith("@uiw/")) return "vendor-react";`）
- 修法：在该分支**之前**加：
  ```ts
  if (pkgName === "@uiw/react-markdown-preview" || pkgName === "react-markdown") {
    return "vendor-md-editor";
  }
  ```
- 验证：build 后 vendor-react 头部不再 `import "./vendor-markdown-*.js"`。

### 1.4 echarts 按需入口（核心瘦身，~650KB raw）✅
- 现状：`node_modules/echarts-for-react/esm/index.js:2` 是 `import * as echarts from 'echarts'`（全量）；`src/components/charts/EChartsWrapper.tsx:24-28` lazy 加载该默认导出，`@/lib/echarts` 的按需注册实例从未传入组件（运行时靠全量兜底）。
- 修法：
  1. `src/lib/echarts.ts` 注册面补齐实际用到的图表：先 grep 所有 EChartsWrapper 使用方确认 series 类型（已知 `ChartWidget.tsx` 需要 line/bar/pie/area），在现有 CustomChart 基础上 `echarts.use([...])` 增加 `LineChart, BarChart, PieChart`（从 `echarts/charts` 导入）；如使用方有用 `LegendComponent`/`DatasetComponent` 等再按需补。
  2. `EChartsWrapper.tsx` 改为 `lazy(() => import("echarts-for-react/esm/core").then(m => ({ default: m.default })))`，渲染时传 `echarts={echartsCore}` prop（从 `@/lib/echarts` 导入实例，两个 import 并入同一 lazy chunk 的现有 Promise.all 结构）。
  3. `import type { EChartsOption } from "echarts"`（`MACHeatmapChart.tsx:13`、`ChartWidget.tsx:9`）是纯类型导入会被擦除，无需改。
- 验证：type-check + vitest 相关图表测试 + build 后 vendor-echarts 体积应从 ~1.14MB 显著下降；手动确认 dashboard 图表与 MAC 热力图渲染正常（测试覆盖即可）。

### 1.5 md-editor CSS 移到懒加载侧 ✅
- 位置：`src/pages/system/notice/components/NoticeForm.tsx:17`（`import "@uiw/react-md-editor/markdown-editor.css";`）
- 修法：删除该静态导入；在 `src/components/markdown/MarkdownEditor.tsx`（已有的 lazy 包装，约 :24 行 `lazy(() => import("@uiw/react-md-editor/nohighlight"))`）内部模块顶层导入该 CSS，使其随 JS 懒 chunk 一起加载。

### 1.6 构建验证与 size-limit 收紧 ✅
- 完成后 `npm run build`，记录 dist chunk 体积表（ls -la dist/assets）。
- `.size-limit.json` 阈值按新实测收紧（main 首屏 gzip 目标 ≤ 850KB；保留 total 上限），防止 chunk 图谱再次劣化。
- 把实测数字写进 SUMMARY（修复前后对比：修复前 vendor-react 2031KB / vendor-echarts 1136KB / vendor-three 911KB / vendor-markdown 372KB 全部首屏 modulepreload）。

---

## 批次 2：bugfix-ws-polling（功能 bug + 运行时行为）

### 2.1 通知 WebSocket 链路断裂修复（功能 bug）✅
- 现状：`src/components/layout/header.tsx:22-25` 调用无参 `useWebSocket()`，注释声称「模块级单例」但 `src/hooks/useWebSocket.ts` 无任何模块级单例（wsRef 是每实例 ref）、`connect()` 只能显式调用 → 通知 WS 连接数恒为 0，未读数实时推送失效。`NotificationBell.tsx:72` 附近注释「连接已在 Header 初始化」基于同一错误前提。
- 修法（最小侵入）：删除 header.tsx 的无参调用与错误注释；在 `NotificationBell.tsx` 中实际建立连接：`useWebSocket({ url: <通知WS地址>, onMessage: <现有未读数处理逻辑> })` + `useEffect(() => { connect(); }, [connect])`。WS URL 参照项目既有通知 WS 端点（grep `websocket|ws://|/ws` 找服务端通知端点与现有 DashboardView.tsx:128 的用法对齐）。若 NotificationBell 内已有相关 onMessage 逻辑就复用。
- 验证：type-check + 相关测试；dev server 下登录后 Network 面板出现 WS 连接（写进 SUMMARY 让用户复核）。

### 2.2 SyncMonitor 孤儿页删除 ✅
- 现状：`src/pages/ad/SyncMonitor/index.tsx` 无任何路由/import 引用（已验证），且 3 处 `post("/api/v1/ad/...")` 与 baseURL `/api/v1`（`src/lib/api.ts:55`）双前缀叠加，每 10s 打无效端点。
- 修法：删除整个 `src/pages/ad/SyncMonitor/` 目录。删除前 executor 再 grep 确认零引用（含 router 配置、菜单 mock）。

### 2.3 RPA/监控轮询收敛 ◐
- `src/pages/operations/rpa/executions/index.tsx:109-116`、`rpa/workers/index.tsx:142-149`：改为仅当存在 running 状态记录时启用 5s 轮询（参照 `useDiscoveryPolling` 的 runningTasks 条件判断），无任务时停轮询或降到 30s。
- `src/pages/monitor/cache/index.tsx:400-411`：interval 建立收敛为 mount-only effect + ref 读最新 fetcher，翻页不再拆建定时器（当前 deps 含 paginationProps.current/pageSize）。

### 2.4 useDiscoveryPolling stale closure + 定时器漂移 ◐
- 位置：`src/pages/network/discoveries/hooks/useDiscoveryPolling.ts:17-44`
- 修法：`discoveries` 快照改 `useRef` + effect 内同步（或 setState 函数式读取），effect deps 收敛为 `[onPoll]`，使 interval 不随每次轮询结果拆建。

### 2.5 dataFetcher WS 池幽灵连接 ✅
- 现状：`src/components/dashboard/utils/dataFetcher.ts:36` 的 `wsConnections` Map 永不清理，`closeWebSocket()`（:218-227 附近）全项目 0 调用；同一 dashboard 端点还有 DashboardView.tsx:128 的 useWebSocket 实例并存。
- 修法：在 DashboardView 挂载/卸载生命周期里对 dataFetcher 的 WS 通道做清理（卸载时调用 `dataFetcher.closeWebSocket(channel)` 或等价批量关闭）；`ws.onclose` 中从 Map 移除条目。同时删除生产死代码 `src/hooks/useRealtimeUpdates.ts`（仅测试引用——连同其测试文件一起删，先 grep 确认）。

### 2.6 zustand persist 补 version ✅
- 4 个 persist store（`authStore.ts`、`tabsStore.ts`、`dashboardStore.ts`、`settingsStore.ts`）的 persist **配置层**全部无 `version`/`migrate`（注意：settingsStore.ts:58 的 `version: 2` 是 state 字段不是 persist 配置，不要动它）。
- 修法：各 persist 第二参数补 `version: 1`（首次引入即 1），暂无需 migrate 逻辑（当前持久化字段向后兼容），加注释说明字段结构变更时需递增版本并补 migrate。

### 2.7 api.ts encryptionKeyStore 滞留清理 ✅
- 现状：`src/lib/api.ts:48` 的 Map 在 :266 每次加密请求写入，仅 :346/:382（响应解密分支）与 :500（400 重放分支）删除；响应加密关闭（默认）时正常成功响应的条目永久滞留。
- 修法：在响应处理完成路径上统一删除（响应拦截器 finally 语义处 `encryptionKeyStore.delete(requestId)`），保证无论响应是否加密都清理。注意 400 重放路径依赖条目存在，删除时机须在重放判定之后。

### 2.8 dualLevelCache/geocodingCache 定时清理阻塞 ◐
- `src/utils/dualLevelCache.ts:269`（5min setInterval）与 `src/utils/geocodingCache.ts:214`：cleanup 全量 `Object.keys(localStorage)` 逐条 getItem+JSON.parse。
- 修法：cleanup 改 `requestIdleCallback`（fallback setTimeout）分片执行，每片处理少量条目；保持 TTL 语义不变。

### 2.9 TabBar 双通道 resize + 路由 chunk 预取 ◐
- `src/components/layout/shared/TabBar.tsx:140-146`：删除 window resize 监听（ResizeObserver 已覆盖容器尺寸变化），同步删 cleanup 中对应 removeEventListener。
- 利用现成的 `src/router/componentLoader.tsx:231-233` `preloadComponents`（当前 0 调用）：TabBar 标签 hover（onMouseEnter）时预取目标路由 chunk。

### 2.10 删除 jsonata 死依赖 ◐
- `package.json:48` `"jsonata": "^2.2.2"` 全 src 零引用（仅注释提及）。`npm uninstall jsonata`。

---

## 批次 3：zustand-selectors（重渲染）

参照正面写法：`useThemeStore((state) => state.syncFromSettings)`。actions 可用独立 selector 取（zustand action 引用稳定）。全批次完成后 grep 验证 `useDashboardStore()` / `useTabs()` / `useAuthStore()` 等无参调用清零（除确实消费全部 state 的）。

### 3.1 dashboard 族（最重要）✅
- `src/components/dashboard/widgets/base/BaseWidget.tsx:83`：`const { viewMode, selectWidget, selectedWidgetId } = useDashboardStore()` → 三个独立 selector（`s => s.viewMode`、`s => s.selectedWidgetId`、`s => s.selectWidget`）。
- `src/hooks/useWidgetPolling.ts:53`：`const { cacheWidgetData } = useDashboardStore()` → `useDashboardStore(s => s.cacheWidgetData)`。
- 同样处理：`GridItem.tsx:37`、`DashboardGrid.tsx:43`、`LayoutToolbar.tsx:56`、`DashboardView.tsx:61`（各文件按实际消费字段拆 selector）。
- 背景：`dashboardStore.ts:405-414` cacheWidgetData 每次生成新 Map，poll 期间全量订阅者集体重渲染。

### 3.2 布局壳与树根 ✅
- `src/components/layout/shared/useRouteTabs.ts:45-47`：`useTabs()` → `useTabsStore(s => s.addTab)` / `s => s.updateTab`；`useDashboardStore()` → `s => s.currentDashboard`。
- `src/store/tabsStore.ts:308` 的 `useTabs()`：TabBar 只需要 tabs/activeTab（`TabBar.tsx:54`），改造 useTabs 用 `useShallow`（`import { useShallow } from 'zustand/react/shallow'`）或让 TabBar 直接窄 selector；保留 useTabs 导出兼容其他调用方（grep 消费方决定）。
- `src/App.tsx:27`：`useMenuStore()` → `s => s.allMenus`。
- `src/components/ConfigProvider.tsx:27-28`：`useAuthStore()` → `s => s.isAuthenticated`；`useSettingsStore()` → 三独立 selector。
- `src/components/layout/sidebar.tsx:95-96`、`header.tsx:18`、`src/design-system/components/LayoutProvider.tsx:23`：按消费字段拆。
- `LayoutProvider.tsx:53`：context value 内联对象 → `useMemo`。
- `InnovativeLayout.tsx:132` 同 header 模式。

### 3.3 其他订阅点 ✅/◐
- `src/hooks/usePagination.ts:57`：`useSettingsStore()` → `s => s.preferences`（43 个列表页连带）。
- `src/pages/my-notices/index.tsx:33`：只取 `s => s.unreadCount` + actions。
- `src/pages/vdi/VirtualMachineList/index.tsx:48`：删除死代码 `const { user: _user } = useAuthStore();`。
- `src/hooks/useRPAProgress.ts:61`：全项目无消费者——直接删除该 hook 文件（先 grep 确认含测试）。

### 3.4 useUserOptions 共享缓存 ◐
- 新建 `src/hooks/useUserOptions.ts`：`useQuery({ queryKey: queryKeys.user.options（在 src/lib/queryKeys.ts 补 key）, queryFn: getUserList({status:0}), staleTime: 5min })`，返回 options 列表。
- 替换 5 处手拉：`pages/duty/management/index.tsx`、`pages/duty/pools/index.tsx`、`pages/duty/schedules/hooks/useScheduleData.ts:132-139`、`pages/workorder/orders/hooks/useWorkOrderData.ts:153-160`、`pages/workorder/periodic/templates/hooks/useTemplateData.ts`。
- 注意保持各消费方现有数据形状（SimpleUser[] 之类）与 loading 行为。

### 3.5 assets columns memo 生效 ✅
- `src/pages/operations/assets/index.tsx:264-265`：columns 数组（~280 行）本身包 `useMemo`，依赖其中的 handler（已是 useCallback）与 sorterMetas 等；使 `:564-579` 的 tableColumns useMemo 依赖真正稳定。删除 :264 的自认失效注释。

---

## 批次 4：waterfall-parallel（瀑布流并行化）

### 4.1 平面图保存逐元素串行（最重）✅
- `src/pages/operations/floors/useFloorPlanEditor.ts:140-199`：saveWalls/saveDoors/saveTexts 三个 for-of 内逐元素 await。
- 修法：各改为 `Promise.allSettled` 并行，分批（每批 10）避免并发洪峰；收集 rejected 项，全部完成后若有失败统一 `message.error`（列出失败数量），全成功才走现有成功提示。saveFloorPlan 调用链的成功判定相应调整。

### 4.2 duty mutation 后串行双刷新 ✅
- `src/pages/duty/management/hooks/useScheduleData.ts` 5 处（约 :131,:149,:170,:190,:216）：`await fetchList(); await fetchWeeklyDuty(...)` → 抽 `refreshAll(page?)` 内部 `await Promise.all([fetchList(page), fetchWeeklyDuty(currentWeekStart)])`，5 个调用点复用。
- `src/pages/duty/management/hooks/useHolidayData.ts:126-127`：`Promise.all([fetchList(), fetchYears()])`。

### 4.3 楼层平面图编辑器打开串行链 ✅
- `src/pages/operations/floors/index.tsx:417-442`（handleEditFloorPlan）：`loadFloorPlanData(floor.id)`（只依赖入参）与「buildingApi.get → 双选项加载」并行；两个选项加载（loadBuildingOptionsByDept / loadFloorOptionsByBuilding）在拿到 building 后再 `Promise.all`。编辑器 loading 态只绑 loadFloorPlanData。

### 4.4 VDI 快速创建 7 次串行 ✅
- `src/pages/vdi/VirtualMachineList/index.tsx:652-742`（loadQuickCreateDefaults）：按依赖分层并行——server 确定后 `Promise.all([listResourceGroups, listResources, listVTPPlatforms])`，vmpPlatform 确定后 `Promise.all([listRunPositions, listStorages, listNetworks])`。参照同文件 `preloadVDIData`（:147 附近）的既有 Promise.all 写法。

### 4.5 info-points 编辑回显 5 级串行 ✅
- `src/pages/operations/info-points/index.tsx:453-547`（openModal）+ `:562-571`（preloadCascaderPath）：`initCascaderOptions()` 与 ports/workstation 查询并行（无依赖）；workstation→floor 真依赖保留；preloadCascaderPath 内部 floors 与 workstations 请求并行。

### 4.6 弹窗被选项请求阻塞 ◐
- `src/pages/network/discoveries/index.tsx:88-93`：openModal 先 `setModalState(...modalVisible: true)` 再后台 `loadDepartments()`（Select 加 loading 态）。

### 4.7 AD OU 组映射穿梭框串行循环 ◐
- `src/pages/ad-domain/ous/index.tsx:243-262`：create/delete 两循环 → `Promise.allSettled` 合并并行，按 fulfilled/rejected 分别提示。

### 4.8 节假日 xlsx 导入串行 ◐
- `src/pages/duty/holidays/utils.tsx:44-45`：`Promise.all([import("xlsx"), file.arrayBuffer()])`。

### 4.9 通知详情已读标记阻塞 loading ◐
- `src/pages/my-notices/detail.tsx:30-38`：`setNotice` 后立即 `setLoading(false)`；`markNoticeAsRead(id).then(() => markAsRead(id)).catch(...)` 后台执行，不计入 spinner。

### 4.10 工单 fetchList 引用稳定（与 3.4 同文件不同函数）✅
- `src/pages/workorder/orders/hooks/useWorkOrderData.ts`：current/pageSize 收进 ref（参照 `src/hooks/useTableManager.ts:226-239` 的 currentRef 模式），fetchList 的 useCallback deps 去掉 current/pageSize → 初始化 effect（:175-179）不再因翻页重跑。注意 `setCurrent(result.data?.current ?? 1)`（:137）保留，只改 deps 不改行为。

---

## 批次 5：cad-3d-render（CAD/3D 渲染优化）

### 5.1 SVG 坐标精度 ✅
- `src/components/cad-elements/WorkstationElement.tsx:76-91,308-324`、`DoorElement.tsx:90-104,159-185`：rotatePoint 等三角函数出口坐标 `Math.round(v * 100) / 100`（可抽 `snapCoord` helper）。全仓无现成取整 helper，在 cad-elements 内新建小工具即可。

### 5.2 CAD 编辑器高频态走 ref ✅
- `src/components/cad-editor/CADFloorPlanEditor.tsx:127`：`lastMousePos` state → ref（周围 :129-133 已全是 ref，模式现成）；:630 非 drag 态的 setState 同步改；确认 boxSelectEnd 保留 state（需视觉反馈）。

### 5.3 cad-elements 组件 memo 化 ◐
- `WallElement/DoorElement/WorkstationElement/TextElement` 包 `React.memo`；`CADFloorPlanEditor.tsx:1399-1449` 的内联 `onSelect={() => ...}` / `onHover={...}` 改为 useCallback 稳定引用或 (id, type) 数据式回调（改动子组件签名时保持调用语义）。

### 5.4 3D useFrame 收敛即停 ◐
- `src/pages/operations/building-spaces-3d/components/FloorPlan3D.tsx:98`（:516 每工位实例化）与 `BuildingModel3D.tsx:41`：lerp 收敛判定 `Math.abs(target - cur) < 1e-3` 时直接赋值并 return，避免 N 实例 60fps 常驻回调。

### 5.5 SVG 几何属性 CSS transition ◐
- `src/components/cad-editor/CADFloorPlanEditor.less:27-34`：`.cad-wall-control-point` 的 `transition: r` + `:hover { r: 6 }` → 改为 transform scale 方案（circle 包 g 或直接 `transform: scale()` + transform-box: fill-box; transform-origin: center）保持视觉近似。
- `src/index.css:4887-4894`：`.sidebar-toggle svg` 的 transform transition 移到按钮包裹层。

### 5.6 MAC 时间线累积列表 ◐
- `src/components/network/MACEventsTimeline.tsx:188-201`：`TimelineItem` 包 `React.memo`；累积容器加 `content-visibility: auto; contain-intrinsic-size: 64px`（内联 style 或 less）。

### 5.7 CAD 按 id 查找 Map 化 ◐
- `CADFloorPlanEditor.tsx:407-426`（_selectedElements useMemo）：floorPlanData 变化时建 `Map<id, {el, type}>`（O(n)），多选查找走 O(1)；`:601-614` findNearbyWallNode 用平方距离比较（`dx*dx+dy*dy < threshold*threshold`）替代 sqrt。

### 5.8 平面图视图死代码 ◐
- `src/pages/operations/workstations/views/FloorPlanView.tsx:156`：删除 `_fullWorkstation` 的 O(n²) find（无任何引用）。

---

## 批次 6：js-misc（JS 微性能 + 静态提升）

### 6.1 DeptTree 双重遍历 ✅
- `src/components/DeptTree/index.tsx:159-178`：filter 内 `filterFn(node.children)`（:169）与 map 内再次 `filterFn(node.children)`（:174）→ 改单遍：先递归算 children，据 `titleMatch || children.length > 0` 决定保留，null 后统一 `filter(Boolean)`。

### 6.2 ports 批量写入选中集 ◐
- `src/pages/network/ports/index.tsx:815`：`selectedPorts={portStatus.filter(p => selectedRowKeys.includes(p.id))}` → useMemo + `new Set(selectedRowKeys)`。

### 6.3 静态映射提升模块级（52 处，量大力薄）◐
- 审计清单（组件体内/render 回调内重建的静态映射/选项数组）：最差样本 `src/pages/duty/my-duty/index.tsx:198-205,215-222`（表格 render 回调内，每行每格重建）；其余分布：`pages/workorder/statistics/index.tsx:153`、`pages/operations/info-points/index.tsx:622`、`pages/operations/dedicated-lines/index.tsx:264`、`pages/operations/room-devices/index.tsx:319`、`components/CronSelector/fields/WeekField.tsx:19-28`、`pages/network/credentials/index.tsx:260-271`、`pages/network/mac/index.tsx:252` 等（grep `const \w+Map: Record` / `const \w+Options.*=.*\[$` 在组件体内命中）。
- 修法：纯数据映射（statusMap/colorMap/options）提升到模块级 `const`；依赖组件内变量的不动。优先做 render 回调内与高频页的，其余按 grep 命中全做。每文件 type-check 守护。
- 参照既有模块级先例：`pages/network/devices/utils.tsx:24`。

### 6.4 湖北地图 tooltip 坐标 ◐
- `src/pages/operations/building-spaces-3d/components/HubeiMapGL.tsx:80,421` 与 `HubeiMap.tsx:68,430`：tooltip 跟随光标的 `setTooltipPosition` 每像素 setState → tooltip 抽独立小组件自监听 mousemove，或坐标走 ref + 直接改 tooltip DOM transform（二选一，以改动小者为准）。

---

## 已知不需要修的（审计确认合规，避免误伤）

- 路由懒加载/xlsx 动态导入/icons 具名导入/`new RegExp` 零命中/`.sort()` 全安全/useEffect deps 零内联对象/数字短路渲染零命中 —— 均已合规。
- `echarts-for-react` 若按 1.4 改 core 入口后，`import type` 残留无需处理。
- `useRealtimeUpdates.ts` 删除时连同其测试文件（仅测试引用它）。
