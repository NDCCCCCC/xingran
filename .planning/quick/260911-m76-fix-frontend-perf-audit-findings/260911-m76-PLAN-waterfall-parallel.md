---
quick_id: 260911-m76
slug: waterfall-parallel
phase: 76-waterfall-parallel
plan: 04
type: execute
wave: 1
depends_on: []
files_modified:
  - xingran-react-frontend/src/pages/operations/floors/useFloorPlanEditor.ts
  - xingran-react-frontend/src/pages/duty/management/hooks/useScheduleData.ts
  - xingran-react-frontend/src/pages/duty/management/hooks/useHolidayData.ts
  - xingran-react-frontend/src/pages/operations/floors/index.tsx
  - xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx
  - xingran-react-frontend/src/pages/operations/info-points/index.tsx
  - xingran-react-frontend/src/pages/network/discoveries/index.tsx
  - xingran-react-frontend/src/pages/ad-domain/ous/index.tsx
  - xingran-react-frontend/src/pages/duty/holidays/utils.tsx
  - xingran-react-frontend/src/pages/my-notices/detail.tsx
  - xingran-react-frontend/src/pages/workorder/orders/hooks/useWorkOrderData.ts
  - xingran-react-frontend/src/hooks/useTableManager.ts
autonomous: false
requirements: []
must_haves:
  truths:
    - 平面图保存并行化（saveWalls/saveDoors/saveTexts Promise.allSettled，每批 10）
    - duty mutation 后双刷新改为 Promise.all 并行
    - 楼层平面图编辑器打开并行化（loadFloorPlanData 与 building/选项并行）
    - VDI 快速创建 7 次串行改为分层并行
    - info-points 编辑回显 5 级串行改为并行
    - 弹窗选项请求不阻塞弹窗打开
    - AD OU 映射穿梭框循环并行化
    - 工单 fetchList current/pageSize 收进 ref，翻页不重跑初始化 effect
  artifacts:
    - path: xingran-react-frontend/src/pages/operations/floors/useFloorPlanEditor.ts
      contains: "Promise.allSettled"
    - path: xingran-react-frontend/src/pages/workorder/orders/hooks/useWorkOrderData.ts
      contains: "currentRef"
---

<objective>
将串行 waterfall 改为并行执行，提升交互响应速度。
</objective>

<context>
@xingran-react-frontend/src/pages/operations/floors/useFloorPlanEditor.ts:140-199（saveWalls/saveDoors/saveTexts 逐元素 await）
@xingran-react-frontend/src/pages/duty/management/hooks/useScheduleData.ts:131-216（5 处 await fetchList + await fetchWeeklyDuty）
@xingran-react-frontend/src/hooks/useTableManager.ts:226-239（currentRef 模式参照）
</context>

<tasks>

<task type="auto">
  <name>Task 1: 平面图保存并行化（4.1）</name>
  <files>xingran-react-frontend/src/pages/operations/floors/useFloorPlanEditor.ts</files>
  <action>
将 `useFloorPlanEditor.ts` 第 140-199 行附近的 `saveWalls/saveDoors/saveTexts` 三个 for-of 循环（逐元素 await）改为：

```ts
// 分批并行（每批 10 个元素），避免并发洪峰
const BATCH_SIZE = 10;
async function saveBatch(items: T[], saveFn: (item: T) => Promise<void>) {
  const results = await Promise.allSettled(
    items.map(item => saveFn(item))
  );
  const failed = results.filter(r => r.status === "rejected");
  if (failed.length > 0) {
    message.error(`保存失败 ${failed.length} 项`);
  }
}
```

三个 save 函数各自改为 `saveBatch(walls, saveWall)` 等，失败时统一收集并显示。
</action>
  <verify>
    <automated>cd xingran-react-frontend && npm run type-check 2>&1 | tail -5</automated>
  </verify>
  <done>saveWalls/saveDoors/saveTexts 改为 Promise.allSettled 并行分批；type-check 通过</done>
</task>

<task type="auto">
  <name>Task 2: duty mutation 后双刷新并行化（4.2）</name>
  <files>
    xingran-react-frontend/src/pages/duty/management/hooks/useScheduleData.ts
    xingran-react-frontend/src/pages/duty/management/hooks/useHolidayData.ts
  </files>
  <action>
**useScheduleData.ts（约 :131,:149,:170,:190,:216）：**
抽 `refreshAll(page?)` 函数：
```ts
const refreshAll = async (page?: number) => {
  await Promise.all([fetchList(page), fetchWeeklyDuty(currentWeekStart)]);
};
```
5 个调用点替换为 `refreshAll()` 或 `refreshAll(page)`。

**useHolidayData.ts:126-127：** 确认已是 `Promise.all([fetchList(), fetchYears()])`，若不是则改为并行。
</action>
  <verify>
    <automated>grep -n "await fetchList\|await fetchWeekly" xingran-react-frontend/src/pages/duty/management/hooks/useScheduleData.ts</automated>
  </verify>
  <done>useScheduleData 5 个调用点改为 refreshAll 内部 Promise.all；type-check 通过</done>
</task>

<task type="auto">
  <name>Task 3: 楼层平面图编辑器打开并行化（4.3）</name>
  <files>xingran-react-frontend/src/pages/operations/floors/index.tsx</files>
  <action>
`index.tsx` 第 417-442 行 `handleEditFloorPlan` 中：
1. `loadFloorPlanData(floor.id)` 与 `buildingApi.get` + 双选项加载（`loadBuildingOptionsByDept` / `loadFloorOptionsByBuilding`）并行
2. 两个选项加载在拿到 building 后再 `Promise.all`
3. 编辑器 loading 态只绑 `loadFloorPlanData`（不绑选项加载）
</action>
  <verify>
    <automated>cd xingran-react-frontend && npm run type-check 2>&1 | tail -5</automated>
  </verify>
  <done>handleEditFloorPlan 改为并行加载；type-check 通过</done>
</task>

<task type="auto">
  <name>Task 4: VDI 快速创建分层并行（4.4）</name>
  <files>xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx</files>
  <action>
`index.tsx` 第 652-742 行 `loadQuickCreateDefaults` 中：
1. server 确定后：`Promise.all([listResourceGroups, listResources, listVTPPlatforms])`
2. vmpPlatform 确定后：`Promise.all([listRunPositions, listStorages, listNetworks])`
3. 参照同文件 `preloadVDIData`（:147 附近）的既有 Promise.all 写法
</action>
  <verify>
    <automated>cd xingran-react-frontend && npm run type-check 2>&1 | tail -5</automated>
  </verify>
  <done>loadQuickCreateDefaults 改为分层并行；type-check 通过</done>
</task>

<task type="auto">
  <name>Task 5: info-points 编辑回显 + 弹窗选项请求并行（4.5/4.6）</name>
  <files>
    xingran-react-frontend/src/pages/operations/info-points/index.tsx
    xingran-react-frontend/src/pages/network/discoveries/index.tsx
  </files>
  <action>
**info-points/index.tsx:453-547（openModal）+ :562-571（preloadCascaderPath）：**
`initCascaderOptions()` 与 ports/workstation 查询并行（无依赖）；workstation→floor 真依赖保留；preloadCascaderPath 内部 floors 与 workstations 请求并行。

**discoveries/index.tsx:88-93：**
openModal 先 `setModalState(...modalVisible: true)` 再后台 `loadDepartments()`（Select 加 loading 态）。
</action>
  <verify>
    <automated>cd xingran-react-frontend && npm run type-check 2>&1 | tail -5</automated>
  </verify>
  <done>info-points 并行化；discoveries 弹窗打开与选项加载解耦；type-check 通过</done>
</task>

<task type="auto">
  <name>Task 6: AD OU + 节假日 + 通知详情并行化（4.7/4.8/4.9）</name>
  <files>
    xingran-react-frontend/src/pages/ad-domain/ous/index.tsx
    xingran-react-frontend/src/pages/duty/holidays/utils.tsx
    xingran-react-frontend/src/pages/my-notices/detail.tsx
  </files>
  <action>
**ous/index.tsx:243-262：** create/delete 两循环 → `Promise.allSettled` 合并并行，按 fulfilled/rejected 分别提示。

**holidays/utils.tsx:44-45：** `Promise.all([import("xlsx"), file.arrayBuffer()])` → 改为先 import("xlsx")（动态 import 返回 module，arrayBuffer 同步）保持并行结构但确保顺序正确：`Promise.all([import("xlsx"), Promise.resolve(file.arrayBuffer())])` 或确认当前是否已正确并行。

**my-notices/detail.tsx:30-38：** `setNotice` 后立即 `setLoading(false)`；`markNoticeAsRead(id).then(() => markAsRead(id)).catch(...)` 后台执行不计入 spinner。
</action>
  <verify>
    <automated>cd xingran-react-frontend && npm run type-check 2>&1 | tail -5</automated>
  </verify>
  <done>AD OU 并行；节假日导入并行；通知详情不阻塞 loading；type-check 通过</done>
</task>

<task type="auto">
  <name>Task 7: 工单 fetchList currentRef 模式（4.10）</name>
  <files>
    xingran-react-frontend/src/pages/workorder/orders/hooks/useWorkOrderData.ts
    xingran-react-frontend/src/hooks/useTableManager.ts
  </files>
  <action>
参照 `src/hooks/useTableManager.ts:226-239` 的 `currentRef` 模式：
1. `useWorkOrderData.ts` 中 `current`/`pageSize` 收进 ref（`currentRef`）
2. `fetchList` 的 useCallback deps 去掉 current/pageSize
3. 初始化 effect（:175-179）不再因翻页重跑
4. 保留 `setCurrent(result.data?.current ?? 1)`（:137），只改 deps 不改行为
</action>
  <verify>
    <automated>cd xingran-react-frontend && npm run type-check 2>&1 | tail -5</automated>
  </verify>
  <done>fetchList useCallback deps 去掉 current/pageSize；翻页不重跑初始化 effect；type-check 通过</done>
</task>

</tasks>

<verification>
cd xingran-react-frontend && npm run type-check && npm run lint && npm test
</verification>

<success_criteria>
- 平面图保存并行化（Promise.allSettled + 分批 10）
- duty 双刷新并行化
- 楼层编辑器打开并行化
- VDI 快速创建分层并行
- info-points 编辑回显并行
- 弹窗打开与选项加载解耦
- AD OU 循环并行化
- 工单翻页不重跑初始化 effect
</success_criteria>

<output>
创建 `.planning/quick/260911-m76-fix-frontend-perf-audit-findings/260911-m76-04-PLAN-SUMMARY.md`
</output>
