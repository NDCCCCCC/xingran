# Phase 76 Plan 03: Zustand Selector 拆分 Summary

## 执行摘要

完成 5 个子批次共 7 个 commit，所有子批次 type-check + lint 通过（lint 仅 1 个 pre-existing error），相关测试通过。

---

## 子批次 3.1: dashboard 族 selector 拆分

**Commit:** `7de0692` — `refactor(frontend): split dashboard store selectors for render isolation`

| 文件 | 改动 |
|------|------|
| `BaseWidget.tsx` | `const { viewMode, selectWidget, selectedWidgetId }` → 三个独立窄 selector |
| `useWidgetPolling.ts` | `const { cacheWidgetData }` → `useDashboardStore(s => s.cacheWidgetData)` |
| `GridItem.tsx` | `viewMode/selectedWidgetId/removeWidget/updateWidget` 各独立 selector |
| `DashboardGrid.tsx` | `viewMode` 独立 selector |
| `LayoutToolbar.tsx` | 6 个 dashboardStore 字段各独立 selector |
| `DashboardView.tsx` | `viewMode/setWsStatus/setIsRefreshing/updateWidgetData` 各独立 selector |

**Grep 自证:**
```
grep "useDashboardStore()" src/components/dashboard/ → 0 命中（target files）
grep "useDashboardStore()" src/hooks/useWidgetPolling.ts → 0 命中
```

**偏差:** WidgetEditor.tsx 和 DashboardSettings.tsx 仍有无参调用——这两文件不在本次 scope 内。

---

## 子批次 3.2: 布局壳与树根 selector 拆分

**Commit:** `6975998` — `refactor(frontend): split layout shell selectors for render isolation`

| 文件 | 改动 |
|------|------|
| `useRouteTabs.ts` | `useTabs()` → `useTabsStore(s => s.addTab/updateTab)`；`useDashboardStore()` → `s => s.currentDashboard` |
| `App.tsx` | `useMenuStore()` → `s => s.allMenus` |
| `ConfigProvider.tsx` | `useAuthStore/useSettingsStore` → 各自段独立 selector |
| `header.tsx` | `user/logout` 各独立 selector |
| `sidebar.tsx` | `menus/loading/fetchMenus/sidebarCollapsed/toggleSidebar` 各独立 selector |
| `LayoutProvider.tsx` | `currentLayout/sidebarCollapsed/density` 各独立 selector；context value `{}` → `useMemo` |
| `InnovativeLayout.tsx` | `user/logout` 各独立 selector |

**Grep 自证:**
```
App.tsx useMenuStore()/useAuthStore()/useSettingsStore()/useLayoutStore() → 0 命中
ConfigProvider.tsx 同上 → 0 命中
header.tsx/sidebar.tsx 同上 → 0 命中
LayoutProvider.tsx useLayoutStore() → 0 命中
InnovativeLayout.tsx useAuthStore() → 0 命中
useRouteTabs.ts useTabs()/useDashboardStore() → 0 命中
```

---

## 子批次 3.3: 其他订阅点 + 死 hook 删除

**Commit:** `7f39137` — `refactor(frontend): narrow selector in usePagination and my-notices; delete dead useRPAProgress hook`

| 改动 | 说明 |
|------|------|
| `usePagination.ts` | `useSettingsStore()` → `s => s.preferences` |
| `my-notices/index.tsx` | `useNoticeStore()` → 窄 selector |
| `VirtualMachineList/index.tsx` | 删除 `const { user: _user } = useAuthStore()` 死代码 |
| **删除** `useRPAProgress.ts` | 全项目零消费者，直接删除 |
| **删除** `__tests__/useRPAProgress.test.tsx` | 随 hook 删除 |
| `useNetworkHooks.test.tsx` | 移除 RPAProgress describe block（useRPAProgress import 已删）|

**Grep 自证:**
```
grep -r "useRPAProgress" src/ → 0 命中
```

---

## 子批次 3.4: useUserOptions 共享缓存 hook

**Commit:** `fee1b9b` — `feat(frontend): add useUserOptions shared hook replacing 5 hand-written user list calls`

### 新建文件
- `src/hooks/useUserOptions.ts` — React Query hook，5min stale，共享缓存
- `src/lib/queryKeys.ts` — 新增 `user.options: ["user", "options"]` key

### 替换的 5 处手拉调用

| 文件 | 改动 |
|------|------|
| `duty/management/index.tsx` | 移除 `getUserList` import；移除 `users` state + `fetchUsers`；改用 `useUserOptions()` |
| `duty/pools/index.tsx` | 移除全局 `getUserList` import（保留 poolMembers 专用查询） |
| `useScheduleData.ts` | 移除 `users` state + `fetchUsers`；改用 `useUserOptions()` |
| `useWorkOrderData.ts` | 同上 |
| `useTemplateData.ts` | 同上 |

### 类型兼容性修复
- `dutyApi.SimpleUser` 字段 `nickname` vs `workorderApi.SimpleUser` 字段 `nickName`
- 统一后 `useUserOptions` 使用 `workorderApi.getUserList`（返回 `nickName` 格式）
- 消费者统一使用 `nickName` 字段访问

**Grep 自证:**
```
grep "getUserList" duty/management/index.tsx → 0 命中（已移除）
grep "fetchUsers" duty/schedules/index.tsx → 0 命中
grep "fetchUsers" workorder/periodic/templates/index.tsx → 0 命中
```

---

## 子批次 3.5: assets columns memo 生效

**Commit:** `5f3cb33` — `perf(frontend): wrap assets columns array in useMemo for render stability`

- `assets/index.tsx:265` — `const columns: ColumnsType<Asset> = [...]` → `const columns = useMemo(() => [...], [getColumnSortOrder])`
- 删除原 eslint-disable 注释（依赖现已正确）

**注:** 该文件存在另一个 pre-existing lint error（`tableColumns` useMemo 缺 `handleDelete/handleEdit` dep），与本次改动无关，已 bypass。

---

## 测试修复

**Commit:** `2e5865b` — `fix(test): add QueryClientProvider to useScheduleData test wrapper`

- `useScheduleData.test.tsx` — 新增 `QueryClientProvider` wrapper（`useUserOptions` 内部调用 `useQuery` 需此 Provider）

---

## 最终验证

| 验证项 | 结果 |
|--------|------|
| `npm run type-check` | 通过 |
| `npm run lint` | 通过（1 个 pre-existing lint error，非本批次引入） |
| `npx vitest run` (相关测试) | 通过 |

### 全部 commit

```
5f3cb33 perf(frontend): wrap assets columns array in useMemo for render stability
2e5865b fix(test): add QueryClientProvider to useScheduleData test wrapper
fee1b9b feat(frontend): add useUserOptions shared hook replacing 5 hand-written user list calls
7f39137 refactor(frontend): narrow selector in usePagination and my-notices; delete dead useRPAProgress hook
6975998 refactor(frontend): split layout shell selectors for render isolation
7de0692 refactor(frontend): split dashboard store selectors for render isolation
```

---

## 偏差记录

1. **pre-existing lint error** — `assets/index.tsx:560` 的 `tableColumns` useMemo 缺 `handleDelete/handleEdit` dep，已 pre-existing，本批次未触碰。

2. **WidgetEditor/DashboardSettings** — `useDashboardStore()` 无参调用在这两个文件仍然存在，但它们不在本次 scope 范围内（不在 `files_modified` 列表中），未做修改。
