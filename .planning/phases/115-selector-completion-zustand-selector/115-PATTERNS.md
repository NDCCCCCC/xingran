# Phase 115: selector-completion（Zustand selector 收尾） - Pattern Map

**Mapped:** 2026-09-14
**Files analyzed:** 15 个待改源文件 + 4 个待改测试文件（无新建文件，全部就地修改）
**Analogs found:** 19 / 19（全部有 exact 范本；本 phase 是机械模式替换，无 no-analog 项）

## 行号核验结论（planner 可直接锚定）

映射时对 RESEARCH 全部行号锚点做了实读复核，**全部吻合，无需再修正**：

| 锚点 | RESEARCH 值 | 实读值 | 状态 |
|------|------------|--------|------|
| tabsStore useTabs hook | 310-345 | 310-345 | ✓ |
| layoutStore useLayout hook | 287-346 | 287-346 | ✓ |
| RouteGuard.tsx | 33 | 33 | ✓ |
| DynamicRoutes.tsx | 105-106 | 105-106 | ✓ |
| 3D index.tsx | 36 | 36 | ✓ |
| BuildingView3D.tsx | 59 | 59 | ✓ |
| FloorView3D.tsx | 58-59 | 58-59 | ✓ |
| HubeiMap.tsx | 78（已修正） | 78 | ✓ |
| HubeiMapGL.tsx | 77（已修正） | 77 | ✓ |
| login/index.tsx | 47-48 | 47-48 | ✓ |
| profile/index.tsx | 47 | 47 | ✓ |
| my-duty/index.tsx | 64 | 64 | ✓ |
| my-notices/detail.tsx | 19 | 19 | ✓ |
| DashboardScopeSelector.tsx | 25 | 25 | ✓ |
| NotificationBell.tsx（OQ-1 挂账） | 75 | 67-75（解构块） | ✓ |
| 4 个 mock 测试文件行号 | 21-30 / 20-24 / 16-18 / 15-17 | 一致 | ✓ |

## File Classification

| 待改文件 | Role | Data Flow | Closest Analog | Match Quality |
|----------|------|-----------|----------------|---------------|
| `src/store/tabsStore.ts`（useTabs hook, 310-345） | hook（store 文件内导出） | event-driven（state 订阅） | `src/components/layout/shared/useRouteTabs.ts:45-48` | exact（D-05 唯一范本） |
| `src/store/layoutStore.ts`（useLayout hook, 287-346） | hook（store 文件内导出） | event-driven | `useRouteTabs.ts:45-48` | exact（D-05） |
| `src/router/RouteGuard.tsx:33` | guard 组件 | request-response（渲染门控） | `useRouteTabs.ts:45-48` | exact |
| `src/router/DynamicRoutes.tsx:105-106` | router 组件 | event-driven | `useRouteTabs.ts:45-48` | exact |
| `src/pages/operations/building-spaces-3d/index.tsx:36` | page 组件 | event-driven | `useRouteTabs.ts:45-48` | exact |
| `.../components/BuildingView3D.tsx:59` | component | event-driven | 同上 | exact |
| `.../components/FloorView3D.tsx:58-59` | component | event-driven | 同上 | exact |
| `.../components/HubeiMap.tsx:78` | component | event-driven | 同上 | exact |
| `.../components/HubeiMapGL.tsx:77` | component | event-driven | 同上 | exact |
| `src/pages/login/index.tsx:47-48` | page 组件（action-only ×3） | event-driven | `useRouteTabs.ts:45-46`（action selector 行） | exact |
| `src/pages/profile/index.tsx:47` | page 组件（action-only） | event-driven | 同上 | exact |
| `src/pages/my-notices/detail.tsx:19` | page 组件（action-only） | event-driven | 同上 | exact |
| `src/pages/duty/my-duty/index.tsx:64` | page 组件（state 字段 user） | event-driven | `useRouteTabs.ts:48`（state 字段 selector 行） | exact |
| `src/components/dashboard/settings/DashboardScopeSelector.tsx:25` | component（state 字段 user） | event-driven | 同上 | exact |
| `src/components/NotificationBell.tsx:67-75`（OQ-1，planner 定口径） | component（3 state + 4 action） | event-driven | 同上 | exact |
| `src/pages/login/index.test.tsx:21-30` | test（vi.mock 工厂） | n/a | `useRouteTabs.test.tsx:24-36`（dual-form mock） | exact |
| `src/pages/login/__tests__/index.test.tsx:20-24` | test | n/a | 同上 | exact |
| `src/pages/my-notices/__tests__/detail.test.tsx:15-18` | test | n/a | 同上 | exact |
| `src/components/dashboard/settings/__tests__/DashboardScopeSelector.test.tsx:14-17` | test | n/a | 同上 | exact |

**范式已确立的旁证：** selector 风格 `useXxxStore((s) => s.field)` 在 src 下已有 **60 处 / 25 个文件**（App.tsx、sidebar.tsx、header.tsx、settings/index.tsx 等）；`useShallow` 在 src 下 **0 使用**（grep 证实，与 RESEARCH 一致）。本次改造只是把剩余 16 处旧形态收敛到既有主流形态。

## Pattern Assignments

### `src/store/tabsStore.ts` — useTabs hook（SELECTOR-01，hook / event-driven）

**Analog:** `src/components/layout/shared/useRouteTabs.ts`（D-05 唯一范本）

**现状——无参整店订阅 + 解构（310-345 行）**：
```typescript
export function useTabs() {
  const {
    tabs,
    activeTab,
    history,
    addTab,
    removeTab,
    setActiveTab,
    closeOtherTabs,
    closeAllTabs,
    closeLeftTabs,
    closeRightTabs,
    updateTab,
    pinTab,
    unpinTab,
    reset,
  } = useTabsStore();          // ← 无参整店订阅，任何 set() 都触发重渲

  return {
    tabs, activeTab, history,
    addTab, removeTab, setActiveTab, closeOtherTabs, closeAllTabs,
    closeLeftTabs, closeRightTabs, updateTab, pinTab, unpinTab, reset,
    hasTabs: tabs.length > 0,   // ← 派生值，必须保留
  };
}
```

**目标形态——照抄范本的逐字段 selector（useRouteTabs.ts:44-48）**：
```typescript
// Source: src/components/layout/shared/useRouteTabs.ts:44-48（线上运行中的 D-05 范本）
export function useRouteTabs() {
  const addTab = useTabsStore((s) => s.addTab);          // action 引用订阅
  const updateTab = useTabsStore((s) => s.updateTab);    // action 引用订阅
  const location = useLocation();
  const currentDashboard = useDashboardStore((s) => s.currentDashboard);  // state 字段订阅
```

**改造规则（SELECTOR-01）：**
1. 14 个字段每个一次 `useTabsStore((s) => s.xxx)`，hooks 数量固定 → 合法（React 19 无 rules-of-hooks 问题）
2. **返回对象形态一字不改**：TabBar.tsx:55（消费 10 字段）等所有消费组件零改动
3. `hasTabs: tabs.length > 0` 派生值原样保留（由局部 `tabs` 计算，非 store 订阅）
4. RESEARCH A3：`history` 在消费方中 0 人解构——默认保留返回形态（最小 diff，D-04 零回归优先）；去掉与否由 executor 自行判断，**不作为验收项**

---

### `src/store/layoutStore.ts` — useLayout hook（SELECTOR-02，hook / event-driven）

**Analog:** `src/components/layout/shared/useRouteTabs.ts:45-48`（同 D-05）

**现状（287-299 行解构 + 302-345 行派生/effect）**：
```typescript
export function useLayout() {
  const {
    currentLayout,
    configuration,
    sidebarCollapsed,
    density,
    syncFromSettings,
    toggleSidebar,
    setSidebarCollapsed,
    setDensity,
    setLayout,
    saveState,
  } = useLayoutStore();        // ← 10 字段整店订阅

  // effect 1 (302-306)：data-layout / data-density 属性同步，依赖 [currentLayout, density]
  // effect 2 (311-323)：settings-changed 事件监听，依赖 [syncFromSettings]

  const layoutConfig = layoutConfigs[currentLayout];   // ← 派生值

  return {
    layout: currentLayout, layoutConfig, configuration, sidebarCollapsed, density,
    syncFromSettings, toggleSidebar, setSidebarCollapsed, setDensity, setLayout, saveState,
    isClassic: currentLayout === "classic",      // ← 7 个派生布尔
    isHybrid: currentLayout === "hybrid",
    isInnovative: currentLayout === "innovative",
    isCompact: density === "compact",
    isComfortable: density === "comfortable",
    isSpacious: density === "spacious",
  };
}
```

**改造规则（SELECTOR-02）：**
1. 10 个字段逐字段 `useLayoutStore((s) => s.xxx)`
2. **两个 effect 原样保留**——`settings-changed` 监听与卸载清理被 `layoutStore.test.ts:104-149` 用真实 store `renderHook` 直接断言
3. `layoutConfig` + 7 个派生布尔计算原样保留（同测试断言 `isClassic/isCompact/layoutConfig/isHybrid`）
4. effect 依赖数组不变（`[currentLayout, density]` / `[syncFromSettings]`）——selector 化后这些名字仍在作用域内，只是来源从解构变为独立 const

---

### `src/router/RouteGuard.tsx:33`（SELECTOR-03，guard 组件 / request-response）

**Analog:** `useRouteTabs.ts:48`（单 state 字段订阅形态）

**现状（33 行）**：
```typescript
const { permissions: userPermissions } = useMenuStore();
```

**目标形态**：
```typescript
const userPermissions = useMenuStore((s) => s.permissions);
```

**安全守护项（V4，RESEARCH Security Domain 明确要求）**：41 行权限判断逻辑**逐字保留**，文件头"这不是安全边界——后端 API 必须独立校验"注释语义不变：
```typescript
// 41 行 — 不得改动
const hasPermission = permissions.some((p) => userPermissions.includes(p));
```
仅消费 `permissions` 一个字段，别名 `userPermissions` 保留。

---

### `src/router/DynamicRoutes.tsx:105-106`（SELECTOR-03，router 组件 / event-driven）

**Analog:** `useRouteTabs.ts:45-48`（跨两个 store 的逐字段订阅）

**现状**：
```typescript
const { allMenus, fetchAll, permissions } = useMenuStore();
const { isAuthenticated, initialized } = useAuthStore();
```

**目标形态（5 个独立 selector）**：
```typescript
const allMenus = useMenuStore((s) => s.allMenus);
const fetchAll = useMenuStore((s) => s.fetchAll);
const permissions = useMenuStore((s) => s.permissions);
const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
const initialized = useAuthStore((s) => s.initialized);
```

**本文件纪律（ROADMAP 跨 phase 冲突约束）：**
1. **只改订阅形态**——Phase 118 (DATA-02) 依赖本文件的菜单加载时序与 TTLMenuCache 语义，`fetchAll` 调用处（131-137 行 effect）与 `routeConfigManager.initialize`（151 行）一行不动
2. effect 依赖 `[isAuthenticated, initialized, allMenus.length, fetchAll]`（137 行）不变——订阅 `s.allMenus`（数组引用），**不要**改成 `s => s.allMenus.length`（RESEARCH Anti-Pattern 明示）
3. 117-125 行的 `useAuthStore.getState()` / `useAuthStore.setState(...)` 是 store 静态方法（非 hook 订阅），**不受本次改造影响，原样保留**

---

### SELECTOR-04 — 3D 页 5 处 visualizationStore（component / event-driven）

**Analog:** `useRouteTabs.ts:45-48`；5 处形态完全同构，逐处对照如下（现状行号已实读核验）：

| 文件:行 | 现状 | 改造后（字段清单） |
|---------|------|-------------------|
| `building-spaces-3d/index.tsx:36` | `const { viewLevel } = useVisualizationStore();` | `const viewLevel = useVisualizationStore((s) => s.viewLevel);`（1 字段） |
| `BuildingView3D.tsx:59` | `const { selectedBuilding, navigateToMap } = useVisualizationStore();` | `selectedBuilding` / `navigateToMap` 两个 selector（1 state + 1 action） |
| `FloorView3D.tsx:58-59` | `const { selectedFloor, selectedBuilding, navigateToBuilding, navigateToMap } =\n  useVisualizationStore();` | **拆 4 个独立调用**（Pitfall 5）：`selectedFloor` / `selectedBuilding` / `navigateToBuilding` / `navigateToMap`；`selectedFloor` 在 `loadWorkstations` 的 `useCallback` 依赖（79 行 `[selectedFloor]`）中直接消费，引用语义不变 |
| `HubeiMap.tsx:78` | `const { clearSelection, navigateToBuilding } = useVisualizationStore();` | `clearSelection` / `navigateToBuilding` 两个 selector（RESEARCH Code Examples 已给对照） |
| `HubeiMapGL.tsx:77` | `const { clearSelection, navigateToBuilding } = useVisualizationStore();` | 同 HubeiMap |

统一目标形态示例（HubeiMap.tsx:78）：
```typescript
const clearSelection = useVisualizationStore((s) => s.clearSelection);
const navigateToBuilding = useVisualizationStore((s) => s.navigateToBuilding);
```

4 个相关测试文件全部用真实 store（`setState` 注入），**零 mock 改动**。

---

### SELECTOR-05 — action-only / user 单字段清理（5 文件 + OQ-1 挂账 1 文件）

**Analog:** `useRouteTabs.ts:45-46`（action 引用订阅）/ `:48`（state 字段订阅）

**action-only 4 处（action 引用在 zustand 5.0.15 中创建后稳定，已从 node_modules 源码验证，无需 useMemo/useCallback 包裹）：**

| 文件:行 | 现状 | 改造后 |
|---------|------|--------|
| `login/index.tsx:47-48` | `const { login } = useAuthStore();`<br>`const { fetchMenus, fetchPermissions } = useMenuStore();` | `const login = useAuthStore((s) => s.login);`<br>`const fetchMenus = useMenuStore((s) => s.fetchMenus);`<br>`const fetchPermissions = useMenuStore((s) => s.fetchPermissions);`（3 个 selector） |
| `profile/index.tsx:47` | `const { updateUser } = useAuthStore();` | `const updateUser = useAuthStore((s) => s.updateUser);` |
| `my-notices/detail.tsx:19` | `const { markAsRead } = useNoticeStore();` | `const markAsRead = useNoticeStore((s) => s.markAsRead);`（49 行 useCallback 依赖 `[id, navigate, markAsRead]` 不变） |

**state 字段 2 处（RESEARCH 已纠正：非 action，取 `user`）：**

| 文件:行 | 现状 | 改造后 |
|---------|------|--------|
| `my-duty/index.tsx:64` | `const { user } = useAuthStore();` | `const user = useAuthStore((s) => s.user);` |
| `DashboardScopeSelector.tsx:25` | `const { user } = useAuthStore();` | `const user = useAuthStore((s) => s.user);`（26 行 `isAdmin` 派生、31 行 `dataScope` 派生原样保留） |

**OQ-1 挂账点（planner 必须在 PLAN 中定口径，二选一）：**
`src/components/NotificationBell.tsx:67-75` 现状解构 7 项（`unreadCount/notifications/loading` 3 state + `markAsRead/markAllAsRead/removeNotification/setNotifications` 4 action）。若纳入 SELECTOR-05：拆 7 个 selector，其两个测试（`components/__tests__/notificationBell.interact.test.tsx`、`components.render.test.tsx`）均用真实 store，零 mock 风险。若不纳入：grep 断言必须显式排除该文件（否则 criterion 5 验收红）。

---

### 4 个 mock 测试文件（test，随组件同任务更新）

**Analog:** `src/components/layout/shared/__tests__/useRouteTabs.test.tsx:24-36`（dual-form mock 既有范本，已在线上运行）

**范本原文（24-36 行）**：
```typescript
vi.mock("@/store/tabsStore", () => {
  // 兼容 selector 调用（useTabsStore(s => s.addTab)）与无参调用（useTabs()）两种形态
  const useTabsStoreFn: any = (selector?: (s: unknown) => unknown) =>
    selector ? selector(mockUseTabsReturn) : mockUseTabsReturn;
  useTabsStoreFn.getState = vi.fn(() => ({
    ...mockUseTabsReturn,
    tabs: mockUseTabsReturn.tabs,
  }));
  return {
    useTabs: vi.fn(() => mockUseTabsReturn),
    useTabsStore: useTabsStoreFn,
  };
});
```

**逐文件改造对照（4 个必改，缺一即红）：**

1. **`src/pages/login/index.test.tsx:21-30`**（authStore + menuStore 两个 mock）：
```typescript
// 现状（忽略 selector 参数）：
vi.mock("@/store/authStore", () => ({
  useAuthStore: () => ({ login: mockLogin }),
}));
// 改为 dual-form（mockLogin 来自 vi.hoisted，工厂内可直接引用）：
vi.mock("@/store/authStore", () => {
  const useAuthStoreFn: any = (selector?: (s: unknown) => unknown) =>
    selector ? selector({ login: mockLogin }) : { login: mockLogin };
  return { useAuthStore: useAuthStoreFn };
});
// menuStore mock（25-30 行）同型改造，state = { fetchMenus: mockFetchMenus, fetchPermissions: mockFetchPermissions }
```

2. **`src/pages/login/__tests__/index.test.tsx:20-24`**（authStore）：
```typescript
// 现状：
vi.mock("@/store/authStore", () => ({
  useAuthStore: vi.fn(() => ({ login: vi.fn() })),
}));
// 改为 dual-form（state 提到工厂作用域，login 的 vi.fn() 保持）：
vi.mock("@/store/authStore", () => {
  const state = { login: vi.fn() };
  const useAuthStoreFn: any = (selector?: (s: unknown) => unknown) =>
    selector ? selector(state) : state;
  return { useAuthStore: useAuthStoreFn };
});
```

3. **`src/pages/my-notices/__tests__/detail.test.tsx:15-18`**（noticeStore）——**闭包读 `let` 变量，调用时求值**：
```typescript
// 现状（noticeStoreState 是 let，beforeEach 里重赋值）：
let noticeStoreState: Record<string, any> = {};
vi.mock("@/store/noticeStore", () => ({
  useNoticeStore: () => noticeStoreState,
}));
// 改为 dual-form——保持对 let 变量的惰性读取（不要在工厂执行时快照）：
vi.mock("@/store/noticeStore", () => {
  const useNoticeStoreFn: any = (selector?: (s: unknown) => unknown) =>
    selector ? selector(noticeStoreState) : noticeStoreState;
  return { useNoticeStore: useNoticeStoreFn };
});
```

4. **`src/components/dashboard/settings/__tests__/DashboardScopeSelector.test.tsx:14-17`**（authStore——**功能性破坏案例**：测试 47/56/76 行改 `mockUser = { isAdmin: true, ... }` 后组件拿不到 isAdmin，admin 分支断言全红）：
```typescript
// 现状：
let mockUser: any = { isAdmin: false, dataScope: "dept", deptId: "d1" };
vi.mock("@/store/authStore", () => ({
  useAuthStore: () => ({ user: mockUser }),
}));
// 改为 dual-form——同样保持对 let mockUser 的调用时求值：
vi.mock("@/store/authStore", () => {
  const useAuthStoreFn: any = (selector?: (s: unknown) => unknown) =>
    selector ? selector({ user: mockUser }) : { user: mockUser };
  return { useAuthStore: useAuthStoreFn };
});
```
**语义守护（RESEARCH Security Domain）**：此处用 dual-form 而非简化返回——`isAdmin/dataScope` 断言语义（46-53 行"管理员 → 显示系统仪表盘 Switch"）必须原样保住。

**`.getState` 附加项说明**：范本里 `useTabsStoreFn.getState = vi.fn(...)` 是因为 useRouteTabs 内部调用了 `useTabsStore.getState()`；上述 4 个被测组件均只用 hook、不调 `.getState()`，所以 4 个 mock **可以不加** `.getState`（加上与范本一致，亦无害）。

## Shared Patterns

### 逐字段 selector（D-05 唯一范本）
**Source:** `src/components/layout/shared/useRouteTabs.ts:45-48`
**Apply to:** 全部 15 处源码改造点（SELECTOR-01~05），每字段一次独立调用，action 与 state 字段形态相同
```typescript
const addTab = useTabsStore((s) => s.addTab);                            // action
const currentDashboard = useDashboardStore((s) => s.currentDashboard);   // state
```

### dual-form mock（selector 兼容测试）
**Source:** `src/components/layout/shared/__tests__/useRouteTabs.test.tsx:24-36`
**Apply to:** 4 个必改 mock 文件；核心一行 `selector ? selector(state) : state`；对 `let` 变量保持调用时求值

### 派生值/副作用原样保留
**Source:** `src/store/tabsStore.ts:343`（hasTabs）、`src/store/layoutStore.ts:302-345`（2 effect + layoutConfig + 7 布尔）
**Apply to:** SELECTOR-01/02——hook 返回对象形态与 effect 链一字不改，消费组件零改动

### store 静态方法不改造
**Source:** `src/router/DynamicRoutes.tsx:117-125`（`useAuthStore.getState()` / `.setState`）、`useRouteTabs.ts:69,110,139`
**Apply to:** 所有文件——`useXxxStore.getState()` / `.setState()` 是非 hook 静态调用，与订阅形态无关，保持原样

### Action 引用稳定性（不包裹 memo）
**Source:** zustand 5.0.15 `esm/react.mjs` createImpl（RESEARCH [VERIFIED]）
**Apply to:** SELECTOR-05 action-only 4 处——直接 `s => s.action`，禁止 useMemo/useCallback 包裹 action

## Anti-Patterns（映射阶段确认为硬约束）

1. **禁用对象 selector**：`useXxxStore(s => ({a: s.a}))` 无 `useShallow` 会无限重渲（v5 `Object.is` 比较）。本项目 `useShallow` 0 使用——本 phase 维持 0 使用，一律逐字段。
2. **禁碰范围外 mock**：`components/layout/__tests__/InnovativeLayout.test.tsx:23-25`、`components/__tests__/ConfigProvider.test.tsx` 也有非 selector 兼容 mock，但对应组件不在 16 处清单内——今天测试是绿的，不要动。
3. **DynamicRoutes 只改订阅形态**：菜单加载时序（131-137 行）、`routeConfigManager.initialize`（151 行）归 Phase 118。
4. **RouteGuard 权限逻辑逐字保留**（41 行 + 文件头 UX-only 注释）。
5. **验收禁止 Profiler 渲染计数断言**（RESEARCH Pitfall 4：tabs/layout 两簇重渲收益有限，实质收益锚定 SELECTOR-03 路由树）。

## No Analog Found

无。本 phase 全部 19 个待改文件均有库内 exact 范本（selector 形态 60 处既有使用 + dual-form mock 范本 1 处既有使用），无需求助 RESEARCH 外部模式。

## Metadata

**Analog search scope:** `xingran-react-frontend/src/{store,router,components,pages,hooks}`
**Files read:** 19（2 范本 + 2 store + 2 路由 + 5 个 3D 页 + 5 个 SELECTOR-05 页面/组件 + NotificationBell + 4 mock 测试——范本与待改文件有重叠）
**Pattern extraction date:** 2026-09-14
**交叉验证:** `Store((s) => s.` 库内 60 处/25 文件（selector 形态为主流）；`useShallow` 0 处（与本 phase 方案一致）
