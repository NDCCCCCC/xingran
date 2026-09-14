# Phase 115: selector-completion（Zustand selector 收尾） - Research

**Researched:** 2026-09-14
**Domain:** React 19 + Zustand v5 selector 订阅模式重构（纯前端性能重构，零行为变更）
**Confidence:** HIGH（全部结论基于实际代码 grep 与已安装 zustand 5.0.15 源码验证，无训练数据推断）

## Summary

本 phase 是机械性极强的模式替换：把全库剩余的 `useXxxStore()` 无参整店订阅改为 `useXxxStore((s) => s.field)` 逐字段 selector。实测 grep（2026-09-14，src 全量，排除测试文件）共 **29 处**无参调用，其中本 phase 范围（SELECTOR-01~05）**16 处**，其余 13 处归属 Phase 116（dashboard 11 处）、Phase 120（useTabSync.ts 死代码 1 处）与**无主**的 `NotificationBell.tsx:75`（1 处，见 Open Questions OQ-1）。

关键发现一：**4 个测试文件的 store mock 不是 selector 兼容形态**（`useAuthStore: () => ({ login })` 忽略 selector 参数），对应组件 selector 化后必然红：`pages/login/index.test.tsx`、`pages/login/__tests__/index.test.tsx`、`pages/my-notices/__tests__/detail.test.tsx`、`components/dashboard/settings/__tests__/DashboardScopeSelector.test.tsx`（后者断言 `isAdmin=true` 行为，会功能性失败）。修复范本已存在于 `useRouteTabs.test.tsx:24-36`（dual-form mock：`selector ? selector(state) : state`）。其余全部相关测试（TabBar/layoutStore/3D 页 4 个/routeGuard/notificationBell/profile/my-duty）都用**真实 store**（`setState` 注入），selector 化对它们完全透明，零改动。

关键发现二：**行号漂移已核实**。Phase 114 改动后 HubeiMap 整店订阅从审计的 73 行漂到 **78** 行、HubeiMapGL 从 86 漂到 **77** 行；tabsStore 的 useTabs hook 现为 **310-345** 行、layoutStore 的 useLayout hook 现为 **287-346** 行。useLayout 返回 10 个 store 字段 + `layoutConfig` + 7 个派生布尔（`isClassic` 等），layoutStore.test.ts:104-149 用真实 store `renderHook` 断言这些派生值——改造时派生值计算必须原样保留。

**Primary recommendation:** 逐字段 selector（每字段一次 `useXxxStore((s) => s.field)`，严格对齐 D-05 useRouteTabs:45-48 范本），不用 useShallow 对象 selector（项目 0 使用，v5 中返回新对象的 selector 不配 useShallow 会无限重渲）；4 个 mock 文件随组件同任务更新为 dual-form；验收 = grep 断言（本 phase 范围 16 处归零）+ 受影响测试文件全绿 + `npm run lint` + `npm run type-check`（D-04 零回归口径）。

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

（115-CONTEXT.md 无用户锁定决策——Implementation Decisions 全部归入 Claude's Discretion）

### Claude's Discretion

> All implementation choices are at Claude's discretion — pure infrastructure/performance phase. Use REQUIREMENTS.md SELECTOR-01~05 条目、success criteria 行号锚点与 D-05 复用范本（useRouteTabs:45 selector 风格）指导实现。

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope。

### 来自 REQUIREMENTS.md 的里程碑级锁定决策（约束本 phase）

- **D-01 范围**: 全量 33 findings + 8 死代码清理，不分批 defer
- **D-02 七 gate 不倒退**: go build / go test / 后端 coverage ≥78.33 基线 / 前端 45 dirs / lint / type-check / diff coverage 全程保持绿
- **D-03 bundle 基线不倒退**: size-limit 门禁（entry gzip 1MB / 全量 2.5MB）保持；改动不得推高 entry
- **D-04 回归纪律**: 纯性能重构（selector 化）以**现有测试零回归**为准，无新增行为测试要求
- **D-05 复用范本**: `useRouteTabs:45` selector 风格为唯一范本
- **D-06 Phase 编号**: 从 114 续编

### 跨 phase 文件冲突约束（ROADMAP）

- **Phase 116 (DASH)**: 依赖本 phase 先确立 selector 范式；dashboard 模块 11 处整店订阅**不在本 phase 范围**
- **Phase 118 (DATA-02)**: 与 SELECTOR-03 同触 `DynamicRoutes.tsx`——本 phase 对该文件**只改订阅形态**，不得动菜单加载/缓存数据流（ROADMAP 明示依赖顺序）
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SELECTOR-01 | `useTabs`（tabsStore.ts:310-326，解构 14 字段）内部改逐字段 selector（useRouteTabs:45-48 范本） | 实际锚点 **310-345**：14 store 字段 + `hasTabs` 派生；消费方 TabBar.tsx:55（10 字段）/ useTabSync.ts:38（死代码，Phase 120 删）；layoutStore.test.ts 同型测试已验证真实 store + renderHook 模式可行 |
| SELECTOR-02 | `useLayout`（layoutStore.ts:287-299，解构 10 字段）内部改逐字段 selector | 实际锚点 **287-346**：10 store 字段 + `layoutConfig` + 7 派生布尔；layoutStore.test.ts:104-149 断言派生值（isClassic/isCompact/layoutConfig/isHybrid）+ settings-changed 监听卸载——派生计算必须保留 |
| SELECTOR-03 | 路由层 RouteGuard.tsx:33 / DynamicRoutes.tsx:105-106 改 selector 订阅 | RouteGuard 仅消费 `permissions`；DynamicRoutes 消费 menuStore `allMenus/fetchAll/permissions` + authStore `isAuthenticated/initialized`（共 5 个 selector）；无行号漂移；DynamicRoutes 无直接渲染测试，靠 type-check + Phase 118 前向兼容 |
| SELECTOR-04 | 3D 页 5 处 visualizationStore 整店订阅改 selector | 行号已重新定位：index:36 / BuildingView3D:59 / FloorView3D:**58-59** / HubeiMap:**78**（原 73）/ HubeiMapGL:**77**（原 86）；各处消费字段已逐一列出（见清单）；4 个相关测试全部用真实 store，零 mock 改动 |
| SELECTOR-05 | 其余 action-only 整店订阅清理（my-notices/detail:19 / profile:47 / login:47-48 / my-duty:64 / DashboardScopeSelector:25） | 6 处中 4 处确为 action-only（markAsRead/updateUser/login/fetchMenus+fetchPermissions）；**2 处实为 state 字段**（my-duty:64 与 DashboardScopeSelector:25 均取 `user`，非 action）——同样用 `s => s.user` 单字段订阅；action 引用在 Zustand 中创建后稳定（已从 5.0.15 源码验证） |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| store 订阅粒度（useTabs/useLayout/RouteGuard/DynamicRoutes/3D 页/action-only 页） | Browser / Client | — | Zustand 是纯客户端状态库，订阅形态只影响 React 渲染层 |
| 4 个测试文件 mock 形态更新 | Browser / Client（测试层） | — | vitest + jsdom 单测，与 store 同任务落位 |
| 菜单加载/缓存数据流 | — | — | **明确不碰**：DynamicRoutes 的 fetchAll 时序与 TTLMenuCache 语义归 Phase 118 (DATA-02) |
| dashboard 订阅收敛 | — | — | **明确不碰**：dashboardStore 11 处归 Phase 116 (DASH-01/03) |

## Standard Stack

### Core

**本 phase 零新增依赖**——纯既有代码模式替换。

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| zustand | 5.0.15（已安装） | 状态订阅 | `useStore(api, selector)` 默认经 `useSyncExternalStore` + `Object.is` 比较（已从 node_modules 源码验证） |
| react | 19.2.8（已安装） | 渲染 | hooks 数量固定即合法，逐字段 selector 14/10 连发无问题 |
| typescript | ~5.9.3 | 类型 | `moduleResolution: "bundler"` 支持 `zustand/react/shallow` 子路径（如需） |
| vitest | ^4.0.18（runner 实测打印 v4.1.10） | 测试 | 真实 store `setState` 注入模式已在 12+ 相关测试文件中使用 |

### 不引入

| 可选 | 决定 | 理由 |
|------|------|------|
| `useShallow`（zustand/react/shallow） | **不用**（除非个别多字段对象场景 executor 判断必要） | D-05 范本为逐字段；项目当前 0 使用（grep 证实）；逐字段订阅粒度更细、无对象分配 |
| react-query 等数据层改造 | 不碰 | Phase 118 范围 |

## Package Legitimacy Audit

**本 phase 不安装任何外部包**（纯既有依赖的代码模式重构）——无需运行 legitimacy gate。zustand 5.0.15 已在 node_modules 实际验证（非 registry 推断）：`useShallow` 导出于 `zustand/react/shallow`（`esm/react/shallow.mjs` 实读），默认比较为 `Object.is`（`esm/react.mjs` 实读）。

## Architecture Patterns

### System Architecture Diagram

```
[zustand store]                  [订阅方改造点]                     [重渲效果]
tabsStore (persist) ──┬─ useTabs() hook (tabsStore.ts:310-345)◄── TabBar.tsx:55
  set() 任意字段写入    │    14 × useTabsStore(s=>s.x)              (常驻双布局)
  ── 整店订阅: 每次set ─┤                                           useTabSync.ts:38 (死代码/120)
  ── 逐字段: 仅被选    └─ useLayout() hook (layoutStore.ts:287-346) ◄── LayoutSwitcher:13 / DensitySwitcher:41
     slice 身份变化才    10 × useLayoutStore(s=>s.x) + 派生布尔
     触发该订阅
menuStore ──┬─ RouteGuard.tsx:33   → 仅 s.permissions
  loading/lastFetchTime/  DynamicRoutes.tsx:105 → s.allMenus / s.fetchAll / s.permissions
  error 高频写入            DynamicRoutes.tsx:106 → s.isAuthenticated / s.initialized (authStore)
  ── 整店订阅时: 菜单拉取过程 loading 翻转 → 重渲整个受保护路由树
visualizationStore ── 3D 页 5 处 (index:36 / BuildingView3D:59 / FloorView3D:58-59 /
  camera*/filters/select*   HubeiMap:78 / HubeiMapGL:77) → 各取 1-4 个字段
authStore/noticeStore ── action-only 4 处 + state 字段 2 处 (user) → 单字段订阅
```

主用例追踪：菜单刷新（fetchAll 置 loading=true → 写 lastFetchTime → 置 error）→ 整店订阅的 DynamicRoutes/RouteGuard 每次都重渲整棵已认证路由树 → selector 化后只有 `allMenus`/`permissions` 身份变化才触发，loading/lastFetchTime/error 写入不再牵连。

### Recommended Project Structure

零新文件。改动全部落在既有文件内：

```
xingran-react-frontend/src/
├── store/
│   ├── tabsStore.ts              # SELECTOR-01: useTabs hook 内部 (310-345)
│   └── layoutStore.ts            # SELECTOR-02: useLayout hook 内部 (287-346)
├── router/
│   ├── RouteGuard.tsx            # SELECTOR-03: :33
│   └── DynamicRoutes.tsx         # SELECTOR-03: :105-106（只改订阅形态，勿动数据流）
├── pages/operations/building-spaces-3d/
│   ├── index.tsx                 # SELECTOR-04: :36
│   └── components/{BuildingView3D,FloorView3D,HubeiMap,HubeiMapGL}.tsx  # :59/:58-59/:78/:77
├── pages/{login/index.tsx, profile/index.tsx, duty/my-duty/index.tsx, my-notices/detail.tsx}
│                                 # SELECTOR-05: :47-48 / :47 / :64 / :19
├── components/dashboard/settings/DashboardScopeSelector.tsx  # SELECTOR-05: :25
└── （测试同任务更新 4 文件，见 Pitfall 2）
```

### Pattern 1: 逐字段 selector（D-05 唯一范本）

**What:** 每个消费字段一次独立 `useXxxStore((s) => s.field)` 调用
**When to use:** 本 phase 全部 16 处
**Example:**
```typescript
// Source: src/components/layout/shared/useRouteTabs.ts:45-48（D-05 锁定范本，已在线上运行）
const addTab = useTabsStore((s) => s.addTab);
const updateTab = useTabsStore((s) => s.updateTab);
const currentDashboard = useDashboardStore((s) => s.currentDashboard);

// SELECTOR-05 action-only 改造后形态（login/index.tsx:47-48）：
const login = useAuthStore((s) => s.login);
const fetchMenus = useMenuStore((s) => s.fetchMenus);
const fetchPermissions = useMenuStore((s) => s.fetchPermissions);

// SELECTOR-05 state 字段改造后形态（DashboardScopeSelector.tsx:25）：
const user = useAuthStore((s) => s.user);
```

### Pattern 2: useTabs/useLayout hook 内部改造（消费组件零改动）

**What:** hook 体内 14/10 连发逐字段订阅，返回对象形态不变
**When to use:** SELECTOR-01/02
**Example:**
```typescript
// Source: 现有 useTabs (tabsStore.ts:310-345) 改造示意
export function useTabs() {
  const tabs = useTabsStore((s) => s.tabs);
  const activeTab = useTabsStore((s) => s.activeTab);
  const history = useTabsStore((s) => s.history);
  const addTab = useTabsStore((s) => s.addTab);
  // …共 14 个字段，hooks 数量固定 → 合法
  return { tabs, activeTab, history, addTab, /* … */, hasTabs: tabs.length > 0 };
}
```
注意：`useLayout` 的派生值（`layoutConfig = layoutConfigs[currentLayout]`、`isClassic` 等 7 个布尔）与 `settings-changed` 监听 effect **原样保留**——layoutStore.test.ts:114-149 直接断言它们。

### Pattern 3: 测试 dual-form mock（selector 兼容）

**What:** mock 同时支持 selector 调用与无参调用
**When to use:** 4 个需更新的 mock 文件
**Example:**
```typescript
// Source: src/components/layout/shared/__tests__/useRouteTabs.test.tsx:24-36（既有范本）
vi.mock("@/store/authStore", () => {
  const useAuthStoreFn: any = (selector?: (s: unknown) => unknown) =>
    selector ? selector(mockState) : mockState;
  useAuthStoreFn.getState = vi.fn(() => mockState);
  return { useAuthStore: useAuthStoreFn };
});
```

### Anti-Patterns to Avoid

- **对象 selector 不配 useShallow**：`useXxxStore(s => ({a: s.a, b: s.b}))` 每次返回新对象 → `Object.is` 永不相等 → useSyncExternalStore "getSnapshot should be cached" 警告 + 无限重渲。本 phase 逐字段方案天然规避；若 executor 选择对象形态，必须 `import { useShallow } from "zustand/react/shallow"`。[VERIFIED: node_modules/zustand@5.0.15 esm/react.mjs + esm/react/shallow.mjs 源码]
- **顺手"修复"范围外组件**：`components/layout/__tests__/InnovativeLayout.test.tsx:23-25`、`components/__tests__/ConfigProvider.test.tsx` 等也有非 selector 兼容 mock，但对应组件不在本 phase 16 处清单内——今天测试是绿的，不要动。
- **在 DynamicRoutes 里顺手改菜单加载时序**：Phase 118 DATA-02 依赖本文件只做订阅形态变更。
- **给 DynamicRoutes 订阅派生值**：订阅 `s.allMenus`（数组引用），不要 `s => s.allMenus.length`（数字虽稳定但 effect 依赖用的是 `allMenus.length === 0` 判断，保持订阅原字段最简）。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 浅比较多字段订阅 | 手写 useMemo + 自定义比较 | 逐字段 selector（D-05）或 `useShallow` | 手写比较易漏字段、闭包陈旧 |
| 测试中支持两种 store 调用形态 | 每个测试新发明 mock 结构 | `useRouteTabs.test.tsx:24-36` dual-form 范本 | 已验证、已注释、可复制 |
| action 引用稳定性保证 | 自己缓存 action 到 ref/module 变量 | 直接 `s => s.action` | Zustand action 在 `create()` 时一次性生成，`set()` 只合并 partial 不替换 action 身份 [VERIFIED: 5.0.15 esm/react.mjs createImpl] |
| 无参订阅的"安全"写法 | `useXxxStore()` + useMemo 挑字段 | selector 订阅 | 整店订阅在每次 `set()` 都产生新 state 对象，useMemo 救不了重渲 |

## Runtime State Inventory

> 本 phase 是纯前端订阅形态重构（refactor 触发）。逐类排查结论：

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | **None** — tabsStore persist（localStorage `tabs-storage`，partialize 仅 tabs/activeTab/history）不因订阅形态改变；selector 化不触持久化结构 | none |
| Live service config | **None** — 纯浏览器端，无外部服务配置 | none |
| OS-registered state | **None** — 无 OS 级注册 | none |
| Secrets/env vars | **None** — 不涉密钥/环境变量 | none |
| Build artifacts | **None** — 无编译产物/安装包受影响；size-limit 门禁（D-03）预期无感（改动不增依赖不增代码量级） | none |

## Common Pitfalls

### Pitfall 1: v5 对象 selector 无限重渲
**What goes wrong:** selector 返回字面量对象/数组且未配 useShallow → 每次快照新身份 → 死循环重渲或 React 警告
**Why it happens:** zustand v5 移除了默认浅比较与 equalityFn 参数，`useSyncExternalStore` 用 `Object.is` 比较快照
**How to avoid:** 一律逐字段（本 phase 方案）；确需对象时 `useShallow` 包裹 selector
**Warning signs:** 控制台 "The result of getSnapshot should be cached" / 页面卡死循环

### Pitfall 2: 4 个测试 mock 非 selector 兼容（必改清单）
**What goes wrong:** mock 形如 `useAuthStore: () => ({ login: mockLogin })`，忽略 selector 参数 → 组件拿到的 `login` 是整个对象而非函数 → "login is not a function" 或断言失败
**Why it happens:** mock 按旧的无参调用形态编写
**How to avoid:** 同任务改为 dual-form（Pattern 3）。必改文件：
1. `src/pages/login/index.test.tsx:21-30`（authStore + menuStore 两个 mock）
2. `src/pages/login/__tests__/index.test.tsx:20-24`（authStore）
3. `src/pages/my-notices/__tests__/detail.test.tsx:16-18`（noticeStore）
4. `src/components/dashboard/settings/__tests__/DashboardScopeSelector.test.tsx:15-17`（authStore——**功能性破坏**：测试改 `mockUser = { isAdmin: true, ... }` 后组件拿不到 isAdmin，admin 分支断言全红）
**Warning signs:** 改完组件跑受影响测试文件即现

### Pitfall 3: 首跑冷缓存假失败（已实测复现一次）
**What goes wrong:** 5 个测试文件合并首跑时 `pages/login/__tests__/index.test.tsx` "导出为函数组件"（`await import("../index")` 全链路冷加载）超时红；单独重跑与热缓存批量重跑均绿（28/28）
**Why it happens:** 冷 transform/import 缓存 + 默认 5s testTimeout 叠加
**How to avoid:** 判定回归前先重跑一次；不要把首跑红直接当 selector 化引入的回归
**Warning signs:** 失败点是纯 import 断言、单独跑绿

### Pitfall 4: 期望落差不实——tabs/layout 的重渲收益有限（如实设定验收口径）
**What goes wrong:** 若把 success criterion 1 验证为"TabBar 渲染次数实测下降"会失望
**Why it happens:** tabsStore 每个 action 都同时写 `tabs` + `history`（新数组），useTabs 又订阅全部 3 个 state 字段（其中 `history` 在 useTabs 消费方中 **0 人使用**）；layoutStore 同理（4 个 state 字段全被 useLayout 订阅）。因此 selector 化只消除"被选 slice 身份未变的 set()"（persist rehydration、未来新增字段、action 身份之外的无效写入）引起的重渲
**How to avoid:** 验收口径 = 订阅形态机械正确（grep/代码审查）+ 测试零回归；实质重渲收益锚定在 SELECTOR-03（menuStore loading/error/lastFetchTime 高频写不再牵连整棵路由树）与 action-only 清理。可选微优化（executor 自行判断）：useTabs 的 `history` 字段 0 消费，可从 hook 返回中去掉（store 本体保留），但非必需
**Warning signs:** 验收任务里出现"用 Profiler 测量渲染次数下降"类断言

### Pitfall 5: FloorView3D 一次解构 4 字段（58-59 两行）
**What goes wrong:** 漏改或改成对象 selector
**How to avoid:** 拆 4 个独立调用：`selectedFloor / selectedBuilding / navigateToBuilding / navigateToMap`；其中 `selectedFloor` 在 `loadWorkstations` 回调里直接消费，保持引用语义不变

## Code Examples

### SELECTOR-04 实际改造对照（HubeiMap.tsx:78）

```typescript
// Source: src/pages/operations/building-spaces-3d/components/HubeiMap.tsx:78（现状）
const { clearSelection, navigateToBuilding } = useVisualizationStore();

// 改造后（D-05 风格）：
const clearSelection = useVisualizationStore((s) => s.clearSelection);
const navigateToBuilding = useVisualizationStore((s) => s.navigateToBuilding);
```

### SELECTOR-03 实际改造对照（DynamicRoutes.tsx:105-106）

```typescript
// Source: src/router/DynamicRoutes.tsx:105-106（现状）
const { allMenus, fetchAll, permissions } = useMenuStore();
const { isAuthenticated, initialized } = useAuthStore();

// 改造后（5 个独立 selector；effect 依赖 [isAuthenticated, initialized, allMenus.length, fetchAll] 不变）：
const allMenus = useMenuStore((s) => s.allMenus);
const fetchAll = useMenuStore((s) => s.fetchAll);
const permissions = useMenuStore((s) => s.permissions);
const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
const initialized = useAuthStore((s) => s.initialized);
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `useStore(store)` + equalityFn 第二参 / 默认浅比较 | selector + `Object.is`，浅比较显式 `useShallow` | zustand v5（2024） | 返回新对象的 selector 必须配 useShallow |
| `create()` 直调（v4 默认导出） | `create<T>()(...)` curried | v4.1+ / v5 | 本项目 store 均已是 curried 形态 [VERIFIED: menuStore.ts:52 等] |
| `import { shallow } from "zustand/shallow"`（配合旧 API） | `import { useShallow } from "zustand/react/shallow"` | v5 | 若需要，用新路径（已在 node_modules 验证存在） |

**Deprecated/outdated:**
- `equalityFn` 作为 useStore 第二参数的写法在 v5 文档中已淡出——本项目不要引入

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | 审计口径"26 处"与实测 29 处的差值源于审计计数方式（SELECTOR-05 六处含 login 双行 + 各文件行号漂移）；以 REQUIREMENTS 行号锚点 + 本实测清单为准 | Summary / 清单 | 低——机械清单以 grep 为准，规划不受影响 |
| A2 | `NotificationBell.tsx:75` 不属于任何 requirement（审计 M-3/M-4 均未列名），默认留在 Phase 116/120 之外的挂账 | Open Questions OQ-1 | 若用户预期"grep 全库归零"是本 phase 验收项，则验收会红——必须由 planner 显式定口径（见 OQ-1） |
| A3 | useTabs 的 `history` 字段 0 消费方（TabBar/useTabSync 均不解构它）——可留可去，默认保留（不碰返回形态） | Pitfall 4 | 低——保留是零风险选项 |

## Open Questions (RESOLVED)

1. **Success criterion 5 的 grep 归零口径（需 planner 定案）**
   - What we know: 实测全库 29 处无参调用；本 phase 16 处；Phase 116 收 11 处（DASH-01 两处 + DASH-03 九处）；Phase 120 DEAD-01 删 useTabSync.ts（1 处）；`components/NotificationBell.tsx:75`（useNoticeStore：unreadCount/notifications/loading + 4 个 action）**不属于任何 requirement**
   - What's unclear: criterion 5 "`useXxxStore()` 无参整店订阅 grep 归零" 是否要求本 phase 完成时全库为 0？若是，NotificationBell 必须纳入本 phase（1 处小改，其测试 `components/__tests__/notificationBell.interact.test.tsx` 与 `components.render.test.tsx` 均用真实 store，零 mock 风险）
   - Recommendation: 二选一并写进 PLAN——(a) 把 NotificationBell.tsx:75 纳入 SELECTOR-05（推荐：1 行级改动即让 criterion 5 在本 phase 可验，剩余 12 处全部有明确归属 phase）；(b) criterion 5 的 grep 断言排除 dashboardStore 家族（Phase 116）+ NotificationBell（挂账），本 phase 断言 = 16 处归零
   - **定案 (2026-09-14, planner):** 采纳推荐 (a)——NotificationBell.tsx:67-75（3 state + 4 action 共 7 项）纳入 115-03；criterion 5 收口 = 全库无参调用恰好 12 处且全部命中白名单三组前缀 `hooks/(useTabSync|useWidgetData).ts` / `pages/dashboard-system/` / `components/dashboard/`（dashboardStore 家族 11 + useTabSync 1，Phase 116/120 清单；03-T3 定案口径，已同步 VALIDATION Criterion 5 行）

2. **useTabs 是否顺带去掉 0 消费的 `history` 字段**
   - What we know: `history` 在 useTabs 消费方中无人解构（A3）
   - Recommendation: 默认保留返回形态（最小 diff，D-04 零回归优先）；不作为验收项
   - **定案 (2026-09-14, planner):** 不做——`history` 字段保留在 useTabs 返回对象（115-01 Task 1 约束 2），D-04 最小 diff 优先，非验收项

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | npm scripts / vitest | ✓ | v24.19.0（满足 24+ 要求） | — |
| zustand | 全部订阅改造 | ✓ | 5.0.15（node_modules 实证） | — |
| vitest | 测试回归 | ✓ | ^4.0.18（runner v4.1.10） | — |
| eslint / tsc | lint / type-check gate | ✓ | 项目脚本 `npm run lint` / `npm run type-check` | — |

**Missing dependencies with no fallback:** 无
**Missing dependencies with fallback:** 无

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Vitest 4（^4.0.18，runner 实测 v4.1.10）+ @testing-library/react，jsdom，setup `src/test/setup.ts` |
| Config file | `xingran-react-frontend/vitest.config.ts`（coverage include `src/**/*.{ts,tsx}` 全量口径，GOV-01） |
| Quick run command | `cd xingran-react-frontend && npx vitest run <受影响测试文件...>`（实测 5 文件 11s 热缓存） |
| Full suite command | `cd xingran-react-frontend && npx vitest run`（CI 口径）；coverage gate：`npm run test:coverage` + `.github/scripts/check-frontend-coverage.sh`（45 dirs floor，D-02） |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SELECTOR-01 | useTabs 逐字段订阅，行为不变 | 既有回归（零新测试，D-04） | `npx vitest run src/store/tabsStore.test.ts src/components/layout/shared/__tests__/TabBar.render.test.tsx src/components/layout/shared/__tests__/useRouteTabs.test.tsx src/hooks/useUtilityHooks.test.tsx` | ✅ |
| SELECTOR-02 | useLayout 逐字段订阅 + 派生值保留 | 既有回归 | `npx vitest run src/store/layoutStore.test.ts`（:104-149 renderHook 断言派生值/事件监听） | ✅ |
| SELECTOR-03 | RouteGuard/DynamicRoutes 仅订阅所需字段 | 既有回归（RouteGuard）+ type-check（DynamicRoutes 无直接渲染测试） | `npx vitest run src/router/__tests__/routeGuard-lastpath.test.tsx && npm run type-check` | ✅ |
| SELECTOR-04 | 3D 页 5 处 selector 化 | 既有回归（4 文件全真实 store） | `npx vitest run src/pages/operations/building-spaces-3d/ src/store/visualizationStore.test.ts` | ✅ |
| SELECTOR-05 | action-only/user 单字段订阅 | 既有回归 + **4 个 mock 文件必改**（Pitfall 2） | `npx vitest run src/pages/login/ src/pages/my-notices/__tests__/detail.test.tsx src/components/dashboard/settings/__tests__/DashboardScopeSelector.test.tsx src/pages/profile/ src/pages/duty/my-duty/` | ✅（mock 更新后） |
|Criterion 5| 无参调用清零 | grep 断言（机械） | `grep -rnE "use[A-Z][A-Za-z0-9]*Store\(\)" xingran-react-frontend/src --include="*.ts" --include="*.tsx" | grep -vE "__tests__|\.test\."` → 期望计数见 OQ-1 口径 | — |
| 七 gate（D-02/D-03 相关项） | lint / type-check / 前端 coverage floor / size-limit | 全套 | `npm run lint && npm run type-check && npm run test:coverage && npm run size` | ✅ |

### Sampling Rate

- **Per task commit:** 该 task 触及的测试文件 quick run + `npm run type-check`（type-check 全量 ~快，防漏改类型）
- **Per wave merge:** 受影响 4 大簇测试（store 簇 / router 簇 / 3D 簇 / 页面簇）+ `npm run lint`
- **Phase gate:** `npx vitest run` 全绿 + lint + type-check + grep 断言达标 → `/gsd:verify-work`

### Wave 0 Gaps

None — 既有测试基础设施完整覆盖全部 phase 需求（唯一"新工作"是 4 个既有 mock 文件的就地修改，随组件同任务落位，不需新测试文件）。

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | 本 phase 不触认证逻辑（login 页仅改 store 订阅形态，登录流程/SM2+SM4 加密链路不动） |
| V3 Session Management | no | authStore 双 token 机制不触 |
| V4 Access Control | **yes（守护项）** | RouteGuard 权限判断逻辑（`permissions.some(p => userPermissions.includes(p))`）必须逐字保留；其文件头注释"这不是安全边界——后端 API 必须独立校验"语义不变 |
| V5 Input Validation | no | 无输入处理变更 |
| V6 Cryptography | no | 不涉国密/密钥 |

### Known Threat Patterns for 本 stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| 前端权限守卫被误当安全边界 | Elevation of Privilege | 保持 RouteGuard "UX-only" 注释与逻辑原样；selector 化不改变权限数据来源（同一 `permissions` 数组引用） |
| 测试 mock 弱化权限行为 | — | DashboardScopeSelector.test.tsx 更新 mock 时保持 `isAdmin/dataScope` 断言语义不变（dual-form mock 而非简化返回） |

（`security_enforcement` 未显式关闭 → 本节按规保留；结论：本 phase 无新增攻击面，唯一要求是**零行为变更纪律**。）

## Sources

### Primary (HIGH confidence)

- 实测 grep（2026-09-14）：`use[A-Z][A-Za-z0-9]*Store\(\)` 全 src = 29 处非测试调用，逐处行号与消费字段逐一实读核对（tabsStore.ts:310-345 / layoutStore.ts:287-346 / RouteGuard.tsx:33 / DynamicRoutes.tsx:105-106 / 3D 页 5 文件 / 6 处 SELECTOR-05 页面 / NotificationBell.tsx:67-75）
- `node_modules/zustand@5.0.15` 源码实读：`esm/react.mjs`（identity selector + useSyncExternalStore + createImpl action 稳定性）、`esm/react/shallow.mjs`（useShallow 实现）、package.json exports
- 受影响测试文件 mock 形态实读（4 个必改 + 12 个真实 store 确认）
- 项目内范本实读：`useRouteTabs.ts:45-48`（D-05）、`useRouteTabs.test.tsx:24-36`（dual-form mock）

### Secondary (MEDIUM confidence)

- 实测运行记录：5 个受影响测试文件两轮批量 + 单文件隔离跑（28/28 绿；首跑出现 1 例冷缓存超时假失败，Pitfall 3）
- `.planning/reviews/20260911-frontend-perf-audit.md`（M-3/M-4 findings 原文——行号与实测的差异已在正文标注）

### Tertiary (LOW confidence)

- 无——本 phase 所有结论均来自代码实证，无 WebSearch 依赖（WebFetch 在本环境被网络策略拦截，zustand 语义改以本地 node_modules 源码验证，置信度更高）

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH — 零新增依赖，zustand 语义从已安装源码验证
- Architecture: HIGH — 16 处改造点全部实读定位，消费字段逐一枚举
- Pitfalls: HIGH — 4 个必改 mock 文件经实读确认（含功能性破坏案例）；首跑假失败经两轮实测复现/排除
- 清单完整性: HIGH — 全库 grep 双模式（ripgrep + grep）交叉验证 29 处，无遗漏入口

**Research date:** 2026-09-14
**Valid until:** 2026-10-14（稳定——行号以执行时 grep 为准，Phase 116/118 若先行会漂移 DynamicRoutes/dashboard 行号，但当前顺序由 ROADMAP 锁定）
