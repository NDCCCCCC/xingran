# Phase 76 Plan 04: waterfall-parallel 批次 4 执行摘要

## 一行总结
将 10 个串行 waterfall 改为并行执行，涵盖平面图保存、值班调度、楼层编辑器、VDI、信息点、网络发现、AD 域、节假日、通知、工单模块。

## 改动的文件

| 文件 | 改动 |
|------|------|
| `src/pages/operations/floors/useFloorPlanEditor.ts` | saveWalls/Doors/Texts 逐元素 await → Promise.allSettled 并行分批 |
| `src/pages/duty/management/hooks/useScheduleData.ts` | 5 处 fetchList + fetchWeeklyDuty → Promise.all 并行 |
| `src/pages/operations/floors/index.tsx` | handleEditFloorPlan: loadFloorPlanData 与 building/选项并行 |
| `src/pages/vdi/VirtualMachineList/index.tsx` | loadQuickCreateDefaults 分层并行（server→3路→3路）|
| `src/pages/operations/info-points/index.tsx` | openModal: initCascader 与 workstation/ports 并行；preloadCascaderPath 内 floors + workstations 并行 |
| `src/pages/network/discoveries/index.tsx` | openModal 先 setModalState，再后台 loadDepartments |
| `src/pages/ad-domain/ous/index.tsx` | create/delete 循环 → Promise.allSettled 并行 |
| `src/pages/duty/holidays/utils.tsx` | import("xlsx") + file.arrayBuffer() 真并行 |
| `src/pages/my-notices/detail.tsx` | 已读标记 fire-and-forget，setLoading(false) 提前到数据返回后 |
| `src/pages/workorder/orders/hooks/useWorkOrderData.ts` | current/pageSize 收进 ref，fetchList useCallback deps 去掉 current/pageSize |

## Grep 自证

| 验证项 | 文件 | 结果 |
|--------|------|------|
| 无 `for (const wall/door/text) of` | `useFloorPlanEditor.ts` | 0 matches |
| 无 `await fetchList(); await fetchWeekly` | `useScheduleData.ts` | 0 matches |
| 无 `await vmApi.list` 单个串行 | `VirtualMachineList/index.tsx` | 0 matches (全在 Promise.all 中) |
| `workstationApi.get` + `initCascaderPromise` 并行 | `info-points/index.tsx` | 第 483 行 |
| `setModalState` 先于 `loadDepartments()` | `discoveries/index.tsx` | 第 91 行 |
| `Promise.allSettled` 处理 toAdd/toRemove | `ous/index.tsx` | 存在 |
| `Promise.all([import("xlsx"), file.arrayBuffer()])` | `holidays/utils.tsx` | 第 44 行 |
| `setLoading(false)` 在 `markNoticeAsRead` 之前 | `detail.tsx` | 第 35 行 |
| `currentRef` / `pageSizeRef` 模式 | `useWorkOrderData.ts` | 存在 |

## 验证结果

- **type-check**: PASSED (`tsc --noEmit`)
- **lint**: PASSED（1355 warnings，1 error pre-existing，0 new）
- **tests**: 28 test files, 169 tests PASSED
  - `src/pages/operations/floors/` — PASSED
  - `src/pages/duty/management/` — PASSED
  - `src/pages/vdi/` — PASSED
  - `src/pages/operations/info-points/` — PASSED
  - `src/pages/workorder/orders/` — PASSED

## 偏差

无偏离 plan。10 项全部按规格实施。

## Commit

```
b97e69b perf(waterfall): parallelize 10 waterfall串行瓶颈 (batch 4)
```

10 files changed, 193 insertions(+), 143 deletions(-)
