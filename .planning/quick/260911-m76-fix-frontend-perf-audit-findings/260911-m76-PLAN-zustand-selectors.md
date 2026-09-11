---
quick_id: 260911-m76
slug: zustand-selectors
phase: 76-zustand-selectors
plan: 03
type: execute
wave: 1
depends_on: ["260911-m76-01-PLAN-build-bundle"]
files_modified:
  - xingran-react-frontend/src/components/dashboard/widgets/base/BaseWidget.tsx
  - xingran-react-frontend/src/hooks/useWidgetPolling.ts
  - xingran-react-frontend/src/components/dashboard/GridItem.tsx
  - xingran-react-frontend/src/components/dashboard/DashboardGrid.tsx
  - xingran-react-frontend/src/components/dashboard/LayoutToolbar.tsx
  - xingran-react-frontend/src/components/dashboard/DashboardView.tsx
  - xingran-react-frontend/src/components/layout/shared/useRouteTabs.ts
  - xingran-react-frontend/src/store/tabsStore.ts
  - xingran-react-frontend/src/App.tsx
  - xingran-react-frontend/src/components/ConfigProvider.tsx
  - xingran-react-frontend/src/components/layout/sidebar.tsx
  - xingran-react-frontend/src/components/layout/header.tsx
  - xingran-react-frontend/src/design-system/components/LayoutProvider.tsx
  - xingran-react-frontend/src/components/layout/InnovativeLayout.tsx
  - xingran-react-frontend/src/hooks/usePagination.ts
  - xingran-react-frontend/src/pages/my-notices/index.tsx
  - xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx
  - xingran-react-frontend/src/hooks/useRPAProgress.ts
  - xingran-react-frontend/src/hooks/useUserOptions.ts
  - xingran-react-frontend/src/lib/queryKeys.ts
  - xingran-react-frontend/src/pages/operations/assets/index.tsx
autonomous: false
requirements: []
must_haves:
  truths:
    - dashboard 族组件全部改用窄 selector，poll 期间不再集体重渲染
    - 布局壳与树根组件改用窄 selector，App/Dashboard/Sidebar 不因无关 state 重渲染
    - usePagination 影响的所有列表页不因无关 settings 字段重渲染
    - useRPAProgress 死 hook 已删除
    - 新 useUserOptions 共享缓存 hook 替换 5 处手拉
  artifacts:
    - path: xingran-react-frontend/src/hooks/useUserOptions.ts
      contains: "useQuery.*queryKey.*user.*options"
    - path: xingran-react-frontend/src/pages/operations/assets/index.tsx
      contains: "useMemo.*tableColumns"
  key_links:
    - from: dashboardStore
      to: BaseWidget/GridItem/DashboardGrid/LayoutToolbar/DashboardView
      via: narrow selectors instead of full store
    - from: useUserOptions.ts
      to: duty/management, duty/pools, duty/schedules, workorder/orders, workorder/periodic
      via: useQuery with 5min staleTime
---

<objective>
修复 zustand selector 导致的过度重渲染（dashboard poll 期间全量订阅者集体重渲染）。
</objective>

<context>
@xingran-react-frontend/src/store/dashboardStore.ts（405-414 cacheWidgetData 每次生成新 Map）
@xingran-react-frontend/src/components/dashboard/widgets/base/BaseWidget.tsx
@xingran-react-frontend/src/components/ConfigProvider.tsx:27-28（参照正面写法 `useThemeStore((state) => state.syncFromSettings)`）
@xingran-react-frontend/src/design-system/components/ThemeProvider.tsx:26（正面 selector 写法参照）
</context>

<tasks>

<task type="auto">
  <name>Task 1: dashboard 族 selector 拆分（3.1）</name>
  <files>
    xingran-react-frontend/src/components/dashboard/widgets/base/BaseWidget.tsx
    xingran-react-frontend/src/hooks/useWidgetPolling.ts
    xingran-react-frontend/src/components/dashboard/GridItem.tsx
    xingran-react-frontend/src/components/dashboard/DashboardGrid.tsx
    xingran-react-frontend/src/components/dashboard/LayoutToolbar.tsx
    xingran-react-frontend/src/components/dashboard/DashboardView.tsx
  </files>
  <action>
**每个文件**将 `useDashboardStore()` 无参调用替换为按实际消费字段拆开的独立 selector：

- BaseWidget.tsx:83：`const { viewMode, selectWidget, selectedWidgetId } = useDashboardStore()` → 三个独立 `useDashboardStore(s => s.viewMode)`、`s => s.selectedWidgetId`、`s => s.selectWidget`
- useWidgetPolling.ts:53：`const { cacheWidgetData } = useDashboardStore()` → `useDashboardStore(s => s.cacheWidgetData)`
- GridItem.tsx:37、DashboardGrid.tsx:43、LayoutToolbar.tsx:56、DashboardView.tsx:61：同样按实际消费字段拆
</action>
  <verify>
    <automated>grep -n "useDashboardStore()" xingran-react-frontend/src/components/dashboard/ xingran-react-frontend/src/hooks/useWidgetPolling.ts 2>/dev/null</automated>
  </verify>
  <done>BaseWidget/GridItem/DashboardGrid/LayoutToolbar/DashboardView/useWidgetPolling 全部改用窄 selector；无参调用清零</done>
</task>

<task type="auto">
  <name>Task 2: 布局壳与树根 selector 拆分（3.2）</name>
  <files>
    xingran-react-frontend/src/components/layout/shared/useRouteTabs.ts
    xingran-react-frontend/src/store/tabsStore.ts
    xingran-react-frontend/src/App.tsx
    xingran-react-frontend/src/components/ConfigProvider.tsx
    xingran-react-frontend/src/components/layout/sidebar.tsx
    xingran-react-frontend/src/components/layout/header.tsx
    xingran-react-frontend/src/design-system/components/LayoutProvider.tsx
    xingran-react-frontend/src/components/layout/InnovativeLayout.tsx
  </files>
  <action>
**每个文件**按实际消费字段拆 selector：
- useRouteTabs.ts:45-47：`useTabs()` → `s => s.addTab` / `s => s.updateTab`；`useDashboardStore()` → `s => s.currentDashboard`
- tabsStore.ts:308 的 `useTabs()`：TabBar 只需 tabs/activeTab，改为 `useShallow` 或窄 selector；保留 useTabs 导出兼容其他调用方
- App.tsx:27：`useMenuStore()` → `s => s.allMenus`
- ConfigProvider.tsx:27-28：`useAuthStore()` → `s => s.isAuthenticated`；`useSettingsStore()` 三个独立 selector
- sidebar.tsx:95-96、header.tsx:18：按消费字段拆
- LayoutProvider.tsx:23：按消费字段拆；:53 的 context value 内联对象 → `useMemo`
- InnovativeLayout.tsx:132：同 header 模式
</action>
  <verify>
    <automated>grep -n "useMenuStore()\|useAuthStore()\|useSettingsStore()" xingran-react-frontend/src/App.tsx xingran-react-frontend/src/components/ConfigProvider.tsx 2>/dev/null</automated>
  </verify>
  <done>布局壳与树根组件全部改用窄 selector；无参调用清零</done>
</task>

<task type="auto">
  <name>Task 3: 其他订阅点 selector 拆分（3.3）</name>
  <files>
    xingran-react-frontend/src/hooks/usePagination.ts
    xingran-react-frontend/src/pages/my-notices/index.tsx
    xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx
    xingran-react-frontend/src/hooks/useRPAProgress.ts
  </files>
  <action>
- usePagination.ts:57：`useSettingsStore()` → `s => s.preferences`
- my-notices/index.tsx:33：只取 `s => s.unreadCount` + actions
- VirtualMachineList/index.tsx:48：删除死代码 `const { user: _user } = useAuthStore();`
- useRPAProgress.ts:61：删除该 hook 文件（grep 确认无消费者后）；同时删除其测试文件
</action>
  <verify>
    <automated>grep -rn "useRPAProgress" xingran-react-frontend/src/ --include="*.ts" --include="*.tsx" 2>/dev/null | grep -v "node_modules"</automated>
  </verify>
  <done>usePagination/my-notices 改窄 selector；useRPAProgress 已删除，零引用</done>
</task>

<task type="auto">
  <name>Task 4: useUserOptions 共享缓存 hook（3.4）</name>
  <files>
    xingran-react-frontend/src/hooks/useUserOptions.ts
    xingran-react-frontend/src/lib/queryKeys.ts
    xingran-react-frontend/src/pages/duty/management/index.tsx
    xingran-react-frontend/src/pages/duty/pools/index.tsx
    xingran-react-frontend/src/pages/duty/schedules/hooks/useScheduleData.ts
    xingran-react-frontend/src/pages/workorder/orders/hooks/useWorkOrderData.ts
    xingran-react-frontend/src/pages/workorder/periodic/templates/hooks/useTemplateData.ts
  </files>
  <action>
1. 在 `src/lib/queryKeys.ts` 补 `user.options: ["user", "options"]` key
2. 新建 `src/hooks/useUserOptions.ts`：
   ```ts
   import { useQuery } from "@tanstack/react-query";
   import { getUserList } from "@/lib/api"; // 找实际 user list API
   import { queryKeys } from "@/lib/queryKeys";

   export function useUserOptions() {
     return useQuery({
       queryKey: queryKeys.user.options,
       queryFn: () => getUserList({ status: 0 }),
       staleTime: 5 * 60 * 1000,
     });
   }
   ```
   保持各消费方现有数据形状（SimpleUser[] 之类）与 loading 行为
3. 替换 5 处手拉 user options 调用
</action>
  <verify>
    <automated>cd xingran-react-frontend && npm run type-check 2>&1 | tail -5</automated>
  </verify>
  <done>useUserOptions hook 创建；5 处手拉调用已替换；type-check 通过</done>
</task>

<task type="auto">
  <name>Task 5: assets columns memo 生效（3.5）</name>
  <files>xingran-react-frontend/src/pages/operations/assets/index.tsx</files>
  <action>
找到 `src/pages/operations/assets/index.tsx` 的 `tableColumns` useMemo（:564-579 附近），确认依赖数组（:264 的 columns 数组）包含 handler（useCallback）与 sorterMetas 等，确保 useMemo 依赖真正稳定。删除 :264 的自认失效注释。
</action>
  <verify>
    <automated>cd xingran-react-frontend && npm run type-check 2>&1 | tail -5</automated>
  </verify>
  <done>assets/index.tsx columns useMemo 依赖稳定；失效注释已删除</done>
</task>

</tasks>

<verification>
cd xingran-react-frontend && npm run type-check && npm run lint
</verification>

<success_criteria>
- dashboard 族无参 useDashboardStore() 调用清零
- 布局壳无参 useAuthStore/useMenuStore/useSettingsStore 调用清零
- useRPAProgress 零引用
- useUserOptions hook 替换 5 处手拉
- 43 个列表页不因无关 settings.preferences 重渲染
</success_criteria>

<output>
创建 `.planning/quick/260911-m76-fix-frontend-perf-audit-findings/260911-m76-03-PLAN-SUMMARY.md`
</output>
