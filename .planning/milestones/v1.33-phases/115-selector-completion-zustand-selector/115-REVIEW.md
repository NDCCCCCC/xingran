---
phase: 115-selector-completion-zustand-selector
reviewed: 2026-09-14T03:45:00Z
depth: standard
files_reviewed: 19
files_reviewed_list:
  - xingran-react-frontend/src/components/NotificationBell.tsx
  - xingran-react-frontend/src/components/dashboard/settings/DashboardScopeSelector.tsx
  - xingran-react-frontend/src/components/dashboard/settings/__tests__/DashboardScopeSelector.test.tsx
  - xingran-react-frontend/src/pages/duty/my-duty/index.tsx
  - xingran-react-frontend/src/pages/login/__tests__/index.test.tsx
  - xingran-react-frontend/src/pages/login/index.test.tsx
  - xingran-react-frontend/src/pages/login/index.tsx
  - xingran-react-frontend/src/pages/my-notices/__tests__/detail.test.tsx
  - xingran-react-frontend/src/pages/my-notices/detail.tsx
  - xingran-react-frontend/src/pages/operations/building-spaces-3d/components/BuildingView3D.tsx
  - xingran-react-frontend/src/pages/operations/building-spaces-3d/components/FloorView3D.tsx
  - xingran-react-frontend/src/pages/operations/building-spaces-3d/components/HubeiMap.tsx
  - xingran-react-frontend/src/pages/operations/building-spaces-3d/components/HubeiMapGL.tsx
  - xingran-react-frontend/src/pages/operations/building-spaces-3d/index.tsx
  - xingran-react-frontend/src/pages/profile/index.tsx
  - xingran-react-frontend/src/router/DynamicRoutes.tsx
  - xingran-react-frontend/src/router/RouteGuard.tsx
  - xingran-react-frontend/src/store/layoutStore.ts
  - xingran-react-frontend/src/store/tabsStore.ts
findings:
  critical: 0
  warning: 6
  info: 11
  total: 17
status: issues_found
---

# Phase 115: Code Review Report

**Reviewed:** 2026-09-14T03:45:00Z
**Depth:** standard
**Files Reviewed:** 19
**Status:** issues_found

## Summary

本次 review 覆盖 Phase 115 三个 commit（579da16..8a10865）的纯 selector 重构：`useXxxStore()` 整店订阅 → 逐字段 `useXxxStore(s => s.field)`，共 19 个文件（13 个源文件 + 4 个测试文件 + 2 个 store hook）。

**重构本体核验通过（零行为变更成立）：**
- 全部新 selector 的目标字段逐一比对过对应 store 定义（noticeStore / authStore / menuStore / visualizationStore / tabsStore / layoutStore），无拼错字段、无遗漏解构项；
- 所有 selector 均返回 store 中已有引用（原始值或 store 内对象/数组），无 `s => ({...})` 形式的现场新建对象——不存在 zustand v5 selector 身份失效导致的无限重渲隐患；
- 被 selector 订阅的 action 全部在 `create()` 内一次性定义，引用恒定，与整店订阅时代行为一致；
- 4 个测试 mock 改为双形态（selector / 无参）时保持了 `let` 变量的调用时求值语义（`beforeEach` 重赋值继续生效），实测 4 个测试文件 20/20 通过；
- 变更文件 eslint 0 error（11 个 warning 全部为既有 `_var`/`any`/react-refresh 类噪音）；
- 源码全量扫描确认 review 范围内已无 `useNoticeStore()` / `useAuthStore()` / `useMenuStore()` / `useVisualizationStore()` / `useTabsStore()` / `useLayoutStore()` 整店调用残留。

**但发现两类值得处理的问题：** (1) 一批与本次变更无关、但位于被审文件内的既有缺陷（detail 页空数据永久 Spin、my-duty 分页 stale-closure 双请求、3D 页 `level: 2` 硬编码导致一级楼宇层级永不渲染等）；(2) 一个 **范围缺口**——`useDashboardStore()` 仍有 12 处整店订阅未纳入本次 selector 化（详见 WR-05），phase 名为 "selector-completion" 但最大的残留源未覆盖。Phase 114 已挂账的 HubeiMap/HubeiMapGL XSS（CR-01）、zoom===10 语义（WR-01）、~100 行重复（WR-02）按要求不重复上报。

## Warnings

### WR-01: my-notices/detail.tsx — 成功返回空 data 时页面永久 Spin

**File:** `xingran-react-frontend/src/pages/my-notices/detail.tsx:29-41`
**Issue:** `setLoading(false)` 只在 `if (noticeData)` 分支内执行（行 32-41）。若 `getMyNoticeDetail` 正常 resolve 但 `response.data` 为 `null/undefined`（通知已被删除、后端返回空 data 的边界），loading 永远停留在 `true`，页面卡死在 Spin；下方 `if (!notice)` 的"通知不存在"分支从这条路径永远不可达。属未处理边界导致的挂起行为。
**Fix:**

```tsx
const response = await getMyNoticeDetail(id);
const noticeData = response.data;
if (noticeData) {
  setNotice(noticeData);
  setLoading(false);
  if (!noticeData.isRead) { /* ... */ }
} else {
  setLoading(false); // 走 !notice 分支显示 "通知不存在"
}
```

### WR-02: my-duty/index.tsx — 分页 handler 的 stale-closure 双请求 + pageSize 双源不一致

**File:** `xingran-react-frontend/src/pages/duty/my-duty/index.tsx:142-152`（关联 `113-124`、`418-423`）
**Issue:** `setCurrent(1)` / `setCurrent`+`setPageSize` 是异步 state 更新，紧随其后的 `loadSchedules()` 读取的是**当前 render 闭包里的旧** `paginationProps.current/pageSize`。结果：搜索时先用旧页码（如第 3 页）发一次错误请求，再由 effect（deps `[paginationProps.current, paginationProps.pageSize]`）用新页码补发一次；若旧请求后返回，表格数据与分页器错页。叠加因素：表格 `pagination={{ ...paginationProps, pageSize: 10 }}`（行 420）硬编码显示 10/页，而 `loadSchedules` 未传 size 时取持久化的 `paginationProps.pageSize`（用户配置可为 50）——翻页时 `handleTableChange` 的即时 `loadSchedules()` 用旧 size（50）+ 新 current 请求，返回行集与 UI 认知的 10/页不符。另外该 effect 每次 pagination 变化还连带重拉 `loadStats()` / `loadWorkOrders()`，二者与分页无关。
**Fix:**

```tsx
const handleSearch = () => {
  setCurrent(1);
  loadSchedules(1); // 显式传参，绕开 stale closure
};

const handleTableChange = (pagination: TablePaginationConfig) => {
  const page = pagination.current ?? 1;
  const size = pagination.pageSize ?? 10;
  setCurrent(page);
  setPageSize(size);
  loadSchedules(page, size); // 或直接删掉本次调用，仅靠 effect 重拉
};
```

并考虑把 `loadStats` / `loadWorkOrders` 拆到独立的 mount-only effect。

### WR-03: building-spaces-3d/index.tsx — `level: 2` 硬编码使"一级楼宇"层级永久为空

**File:** `xingran-react-frontend/src/pages/operations/building-spaces-3d/index.tsx:55`
**Issue:** 楼宇数据映射时无条件写 `level: 2 as const`，而 `HubeiMap.tsx:267` 的层级过滤是 `currentZoom === 10 ? buildings : level1`（`level1 = buildings.filter(b => b.level === 1)` 恒为 `[]`）。结果：**默认 zoom 8 下地图零标记**，用户必须缩放到 10 才能看到任何楼宇；左下角统计面板恒显"一级楼宇: 0"，与同面板"缩放级别 8-9: 显示一级楼宇（城市级汇总）"的图例（`HubeiMap.tsx:666-667`）直接矛盾。HubeiMapGL 的 `filterBuildingsByZoom`（`utils.ts:138-140`）同样受害。
**与 Phase 114 WR-01 的区别：** 114 挂账的是 `zoom===10` 这一过滤阈值本身的语义问题；本条是**数据源侧**缺陷——即使阈值语义修正，`level` 字段也从未被赋过 1，城市级汇总功能是死路径。两条需一起修才有意义。
**Fix:** 后端 Building 无 level 字段的前提下，至少先改为 `level: 1 as const` 让默认 zoom 可见（并同步修正图例/统计面板语义）；或按 cityCode 分组在低 zoom 下聚合为 level-1 标记，落实图例承诺的层级语义。

### WR-04: DashboardScopeSelector — "系统仪表盘必须全局可见"约束只在单方向生效

**File:** `xingran-react-frontend/src/components/dashboard/settings/DashboardScopeSelector.tsx:57-88`
**Issue:** `handleIsSystemChange` 在开启 system 时强制 `scope = "global"`（行 83-85），但 `handleScopeChange` 没有反向守护：`isSystem=true` 时用户仍可把 scope 从 global 切到 private/dept，得到一个"非全局可见的系统仪表盘"，而 Switch 仍显示"是"，破坏组件自己注释声明的 invariant（行 82）。反向亦然：关闭 system 时 scope 停留在 global，用户无法一步回到原 scope。锁定逻辑（行 102）只覆盖 `isSystem && scope === "global"` 的组合，正好放过了这个不一致组合。
**Fix:**

```tsx
const handleScopeChange = (scope: DashboardScope) => {
  if (value?.isSystem && scope !== "global") {
    onChange({ scope: "global", deptId: undefined, isSystem: true }); // 或阻止并提示
    return;
  }
  // ...原逻辑
};

const handleIsSystemChange = (isSystem: boolean) => {
  const newValue = { scope: value?.scope || "private", isSystem, deptId: value?.deptId };
  if (isSystem) newValue.scope = "global";
  onChange(newValue);
};
```

（若后端已校验该约束，前端至少应同步 UI 状态避免"Switch=是 + scope=私有"的矛盾展示。）

### WR-05: selector 化范围缺口 — `useDashboardStore()` 仍有 12 处整店订阅

**File:** `xingran-react-frontend/src/hooks/useWidgetData.ts:70,115`、`xingran-react-frontend/src/hooks/useTabSync.ts:39`、`xingran-react-frontend/src/pages/dashboard-system/index.tsx:30`、`view.tsx:31`、`edit.tsx:41`、`components/DashboardView.tsx:36`、`DashboardList.tsx:47`、`DashboardHome.tsx:19`、`DashboardEdit.tsx:43`、`components/dashboard/widgets/WidgetEditor.tsx:33`、`components/dashboard/settings/DashboardSettings.tsx:30`
**Issue:** 全源码扫描发现 `useDashboardStore()`（无参整店订阅）在 review 范围之外仍有 12 处调用。其中 `useWidgetData` 是**每个 widget 实例各跑一次**的热 hook，而 `dashboardStore.cacheWidgetData`（`dashboardStore.ts:405-406`）在每次任一 widget 数据拉取完成时都会 `set()` —— 即任何一个 widget 刷新都会重渲所有 dashboard 页面/widget 消费者，正是本次 phase 在 layoutStore/tabsStore 注释里宣称要消除的重渲扇出（"任意未被订阅字段的 set() 不再触发本 hook 消费组件重渲"）。`useTabSync.ts:39` 还存在双重订阅：`useTabs()`（已 selector 化）内部再整店订阅 dashboardStore，部分抵消了 115-01 的收益。phase 名为 selector-completion，dashboardStore 是剩余的最大整 store，若为有意留待后续 phase，建议在 STATE/PLAN 里显式挂账，避免"完成"语义误导后续读者。
**Fix:** 对 12 处调用逐一改为 `useDashboardStore((s) => s.currentDashboard)` 等逐字段形式（字段引用语义与本次已迁移的 store 完全同构，迁移模式可直接复用 `useRouteTabs.ts:45-48` 范本）；`useTabSync` 中只需 `const currentDashboard = useDashboardStore((s) => s.currentDashboard)`。

### WR-06: 测试断言弱于用例名承诺 — 核心路径可被删空而测试保持绿

**File:** `xingran-react-frontend/src/pages/my-notices/__tests__/detail.test.tsx:73-83, 98-106`；`xingran-react-frontend/src/components/dashboard/settings/__tests__/DashboardScopeSelector.test.tsx:69-77`
**Issue:** 影响测试可靠性（断言缺失）：
- detail.test.tsx 行 73-83 用例名为"加载成功 + 未读 → 渲染 detail + **调用 markAsRead**"，但**没有任何断言**验证 `markNoticeAsRead`（API）或 `markAsRead`（store）被调用——detail 页的核心已读标记逻辑（detail.tsx:37-40）整段删除该用例依然通过；
- detail.test.tsx 行 98-106 用例名承诺"navigate 调用"，实际只断言了"加载失败"文案，未断言跳转到 `/user-notices`（mock 路由已备好 `user-notices-list` 节点却未使用）；
- DashboardScopeSelector.test.tsx 行 69-77 用例名为"handleScopeChange → 选择 dept → onChange 调用"，实际只断言 Select 渲染存在，`onChange` 从未被触发。
**Fix:**

```tsx
// detail.test.tsx 未读用例补充：
const { markNoticeAsRead } = await import("@/lib/noticeApi");
await waitFor(() => expect(markNoticeAsRead).toHaveBeenCalledWith("n1"));

// 加载失败用例补充：
await waitFor(() => expect(baseElement.textContent).toContain("user-notices-list"));

// DashboardScopeSelector：移除"onChange 调用"承诺，或改名为"渲染 Select"
```

## Info

### IN-01: DashboardScopeSelector — 完全相同的 if/else 分支 + 未完成意图注释

**File:** `xingran-react-frontend/src/components/dashboard/settings/DashboardScopeSelector.tsx:63-71`
**Issue:** `if (dataScope === "all")` 与 `else` 两个分支体逐字相同（均为 `newValue.deptId = user?.deptId`），且注释自认"这里暂时使用用户部门"——死条件分支 + 悬置 TODO。
**Fix:** 收敛为 `if (scope === "dept") { newValue.deptId = user?.deptId; }`，并把"管理员可选部门"的真实意图登记为待办而非注释。

### IN-02: FloorView3D — 未使用变量 `_typeColor` 及其唯一支撑 import

**File:** `xingran-react-frontend/src/pages/operations/building-spaces-3d/components/FloorView3D.tsx:271`（import 见行 17）
**Issue:** `const _typeColor = getWorkstationTypeColorCSS(workstation.type)` 的结果从未被使用；该函数也是 `getWorkstationTypeColorCSS` 这个 import 的唯一使用点。死代码。
**Fix:** 删除行 271 与 import 中的 `getWorkstationTypeColorCSS`（或在卡片上真正展示类型色）。

### IN-03: FloorView3D — 空实现的 onWorkstationClick

**File:** `xingran-react-frontend/src/pages/operations/building-spaces-3d/components/FloorView3D.tsx:126-131`
**Issue:** `onWorkstationClick={() => { /* 工位点击事件处理 */ }}` 传入了空处理器，死代码路径；若 FloorPlan3DLazy 内部据此渲染可点击态，还会产生误导性的交互暗示。
**Fix:** 实现点击逻辑，或把 prop 改为可选并在未传时不绑定点击。

### IN-04: profile 页 — 加载失败后永久停留在"加载中..."占位

**File:** `xingran-react-frontend/src/pages/profile/index.tsx:50-57, 169-171`
**Issue:** `loadProfile` 失败仅弹一次性 toast，`profile` 保持 `null`，行 169 的 `if (!profile) return <div>加载中...</div>` 使页面以假加载态永久占位，无错误状态、无重试入口。
**Fix:** 增加失败态分支（错误文案 + 重试按钮），或至少区分 `loading` 与 `error` 两个状态。

### IN-05: 值班类型映射函数跨文件三处重复

**File:** `xingran-react-frontend/src/pages/duty/my-duty/index.tsx:155-180`；`xingran-react-frontend/src/pages/profile/index.tsx:156-167, 304-310`
**Issue:** `getDutyTypeColor` / `getDutyTypeText` 在 my-duty 与 profile 各有一份，profile 行 304-310 还有第三份内联三元重复。新增值班类型需改三处。
**Fix:** 提取到共享常量模块（项目已有同类先例 `src/constants/status.ts`），如 `src/constants/duty.ts` 导出 `DUTY_TYPE_OPTIONS`。

### IN-06: HubeiMap — 死引用 polygonRef

**File:** `xingran-react-frontend/src/pages/operations/building-spaces-3d/components/HubeiMap.tsx:58, 164`
**Issue:** `polygonRef` 只在 fallback 边界路径赋值，全文件无任何读取点；与 HubeiMapGL 共享版（GL 侧同样只在 addFallbackMask 赋值）构成对称死代码。
**Fix:** 删除该 ref，或补上 unmount 时 `removeOverlay` 的清理用途（当前地图实例本身未 destroy，见下注）。

### IN-07: layoutStore — useLayout 的 DOM 同步 effect 与 applyToDOM 部分重复且不一致

**File:** `xingran-react-frontend/src/store/layoutStore.ts:302-306`（对照 `275-280`）
**Issue:** `useLayout` 的 effect 只同步 `data-layout` / `data-density`，而 `applyToDOM` 同步三项（多 `data-sidebar-collapsed`）。同一关注点两份实现已出现字段漂移：初始 mount 若无任何 action 触发，`data-sidebar-collapsed` 属性不会被设置，依赖该属性的 CSS 规则首帧不生效。
**Fix:** effect 内直接调 `useLayoutStore.getState().applyToDOM()`，删除手写的两行 setAttribute。

### IN-08: tabsStore — addTab 原地修改调用方的 tab 对象

**File:** `xingran-react-frontend/src/store/tabsStore.ts:58-61`
**Issue:** `tab.pinned = true; tab.closable = false;` 直接修改调用方传入的对象而非在 set 时规范化副本。当前无调用方复用该对象故无害，但属 mutation-of-argument 气味，后续复用（如"再次打开已关闭标签"场景）会踩坑。
**Fix:** `const normalized = isDashboardTab(tab.key) ? { ...tab, pinned: true, closable: false } : tab;` 后续一律使用 `normalized`。

### IN-09: selector 化约定缺少回归守护

**File:** 约定锚点 `xingran-react-frontend/src/store/tabsStore.ts:311-312`、`xingran-react-frontend/src/store/layoutStore.ts:288-289`（注释宣称的规范无守护）
**Issue:** 本次 phase 的核心交付是"整店订阅 → 逐字段 selector"约定，但没有任何机制阻止后续代码重新引入 `useXxxStore()`。项目对同类约定均有 AST 守护先例（`src/lib/apiFactory.invariants.test.ts`、Go 侧 `cache_invariants_92_test.go`），此处是缺口。
**Fix:** 增加一个轻量 AST 扫描测试：对 `src/**`（排除 `__tests__`）硬性失败于 `use(Auth|Menu|Notice|Visualization|Tabs|Layout)Store\(\)` 无参调用；dashboardStore 若暂不迁移可先进 warning 层白名单（对应 WR-05）。

### IN-10: 双形态 store mock 手写四处重复

**File:** `xingran-react-frontend/src/components/dashboard/settings/__tests__/DashboardScopeSelector.test.tsx:15-21`；`xingran-react-frontend/src/pages/login/index.test.tsx:21-34`；`xingran-react-frontend/src/pages/login/__tests__/index.test.tsx:20-26`；`xingran-react-frontend/src/pages/my-notices/__tests__/detail.test.tsx:16-22`
**Issue:** 同一段"兼容 selector 调用与无参调用两种形态"的 4 行 mock 工厂被复制了 4 份，且均为裸函数形态、缺少真实 zustand store 的 `getState` / `setState` / `subscribe` 静态成员——一旦被测组件开始使用 `useAuthStore.getState()`（profile 页已有此类用法），mock 会以晦涩方式炸掉。
**Fix:** 在 `src/test/utils/` 增加 `createDualFormStoreMock(initialState)` helper 统一产出，并可顺带补 `getState: () => state` 提升兼容面。

### IN-11: NotificationBell — WebSocket 推送后下拉列表不再加载服务端首页

**File:** `xingran-react-frontend/src/components/NotificationBell.tsx:105-120`
**Issue:** 下拉打开时仅在 `notifications.length === 0` 时拉取列表。会话期间 WebSocket 推送的 `addNotification`（noticeStore 行 76 前插 + 上限 50 条）使 length > 0，此后每次打开下拉只显示本轮 WS 推送项，服务端首页列表永远不加载——用户看到的通知列表是残缺的，且无任何过期提示。
**Fix:** 用"上次拉取时间戳"做陈旧判定（如超过 N 分钟或跨 WS 推送即重拉），替代单纯依赖 `length === 0`。

---

_Reviewed: 2026-09-14T03:45:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
