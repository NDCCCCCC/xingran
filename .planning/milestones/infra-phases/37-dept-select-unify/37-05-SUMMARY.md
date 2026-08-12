---
phase: 37-dept-select-unify
plan: 05
subsystem: frontend-dept-select
tags: [refactor, frontend, react, dept-tree, react-query, duty, workorder, type-dedup]
requires:
  - hooks/useDeptTree.ts (canonical hook, Phase 37-01 已落地)
  - lib/dutyApi.ts:SimpleDept (canonical 类型锚点, line 281)
  - lib/dutyApi.ts:getDeptTree (canonical fetch 函数, line 303)
  - components/shared/DepartmentTreeSelect.tsx (37-01 Task 2 改造后受控 props 为 SimpleDept[])
provides:
  - pages/duty/pools/index.tsx 消费 useDeptTree, 不再自 fetch
  - pages/workorder/orders/hooks/useWorkOrderData.ts 消费 useDeptTree, 不再自 fetch
  - lib/workorderApi.ts SimpleDept 改为 re-export from dutyApi (全项目唯一类型锚点)
affects:
  - 后续批 5 (notice/TargetSelector) — 同类迁移模板已就位
  - workorderApi.SimpleDept 全部下游 import 路径保持不变 (orders/index.tsx, useWorkOrderData.ts) 经 re-export 兼容
  - workorderApi.getDeptTree 副本删除后唯一消费方已迁移到 useDeptTree, 无悬空引用
tech-stack:
  added: []
  patterns:
    - "页面级自 fetch → canonical useDeptTree (同 37-02/37-04 模式推广到 duty/workorder)"
    - "类型 re-export 而非本地重定义 (T-37-09 mitigate: 保持外部 import 路径不变, 类型去重无 breaking change)"
    - "函数副本消除 (workorderApi.getDeptTree 删除, canonical 在 dutyApi.ts)"
key-files:
  created: []
  modified:
    - xingran-react-frontend/src/pages/duty/pools/index.tsx (+5/-13, 删 fetchDepts + depts useState + getDeptTree import, 加 useDeptTree)
    - xingran-react-frontend/src/pages/workorder/orders/hooks/useWorkOrderData.ts (+5/-15, 删 fetchDepts + depts useState + getDeptTree import + return 字段, 加 useDeptTree)
    - xingran-react-frontend/src/lib/workorderApi.ts (+6/-10, SimpleDept 改 re-export + getDeptTree 副本删除)
decisions:
  - "workorderApi.SimpleDept 改为 `export type { SimpleDept } from \"./dutyApi\"` 而非直接删除 — 保持外部 import 路径 (orders/index.tsx, useWorkOrderData.ts) 不破, 向后兼容 (T-37-09 mitigate)"
  - "workorderApi.getDeptTree 副本直接删除而非保留 — 消费方仅 useWorkOrderData.ts 一处, 已同步迁移到 useDeptTree hook, 无悬空引用 (grep 确认)"
  - "useWorkOrderData.ts 保留 `type SimpleDept` import — UseWorkOrderDataReturn.depts 字段类型仍用 SimpleDept[] (经 re-export 来自 dutyApi, 语义不变)"
  - "DepartmentTreeSelect 受控模式保持 — duty/pools 两处调用点 (form 筛选 line 418 + edit modal line 491) 数据仍由 depts 喂入, 37-01 Task 2 改造后 props 类型为 SimpleDept[] 与 depts 类型一致"
  - "不动 orders/index.tsx 的 unused `getDeptList` import (line 62, dead code 但与本次范围无关, CLAUDE.md Scope Constrainment)"
metrics:
  duration: ~7 分钟
  completed: 2026-06-22
  tasks_completed: 2
  files_touched: 3
  commits: 2
---

# Phase 37 Plan 05: 批 4 duty + workorder 模块迁移 + workorderApi 类型去重 Summary

批 4 收尾两个模块的部门树消费收敛——`duty/pools` 删除页面级 `fetchDepts`（含 `getDeptTree()` 调用 + `depts` useState + useEffect 触发），改消费 canonical `useDeptTree`；`workorder/orders/hooks/useWorkOrderData` 同步删除 `fetchDepts`（调的是 workorderApi 的 `getDeptTree` 副本）+ `depts` useState + return 字段 `fetchDepts`，改用 hook；`lib/workorderApi.ts` 类型去重——本地 `SimpleDept` interface 改为 `export type { SimpleDept } from "./dutyApi"`（保持外部 import 路径不变，T-37-09 mitigate），`getDeptTree` 副本删除（消费方已全部迁移）。DepartmentTreeSelect 在 duty/pools 两处调用点保持受控模式（数据由 hook 喂入）。全项目 `SimpleDept` 定义数从 2（dutyApi + workorderApi）收敛为 1（dutyApi 唯一锚点）。

## What Was Built

### Task 1 — duty/pools 迁移 useDeptTree (commit ccbd504)

**文件**: `xingran-react-frontend/src/pages/duty/pools/index.tsx` (+5/-13 行)

1. **顶部 import 调整**：
   - 删除 dutyApi import 块中的 `getDeptTree` (line 17)
   - 新增 `import { useDeptTree } from '@/hooks/useDeptTree'`
   - `type SimpleDept` 保留（`getDeptAndChildrenIds` 工具函数 line 72/93/104 仍引用此类型）

2. **hook 函数体**：
   - 删除 `const [depts, setDepts] = useState<SimpleDept[]>([])` (原 line 67)
   - 新增 `const { data: depts = [] } = useDeptTree()` (新 line 68)
   - 删除 `fetchDepts` 函数 (原 line 171-179, 含 `getDeptTree()` 调用 + `setDepts` + console.error 兜底)
   - 删除 useEffect 内 `fetchDepts()` 触发 (原 line 185)；保留 `fetchList/fetchStats/fetchUsers` 三处调用

3. **DepartmentTreeSelect 受控模式保持** (D-LOCKED)：
   - 两处调用点 (form 筛选 line 418 + edit modal line 491) 的 `departments={depts}` prop 不变
   - `depts` 现在是 `SimpleDept[]` (DeptTreeNode 别名), 与 37-01 Task 2 改造后的 DepartmentTreeSelectProps.departments 类型 (`SimpleDept[]`) 严格一致
   - 数据由 hook 喂入,组件内部不调 hook (职责正交保留)

4. **`getDeptAndChildrenIds` 工具函数保留** (line 72-121)：
   - 依赖 `depts` 与 `SimpleDept` 类型,迁移后 `depts` 由 hook 提供,函数行为不变
   - 用于编辑值班池时根据 deptId 筛选 `filteredUsers`

### Task 2 — workorder 迁移 + workorderApi 类型/函数去重 (commit 0cde4f3)

**文件**: `xingran-react-frontend/src/pages/workorder/orders/hooks/useWorkOrderData.ts` (+5/-15 行) 与 `lib/workorderApi.ts` (+6/-10 行)

**useWorkOrderData.ts 改造**:

1. **顶部 import 调整**：
   - 删除 workorderApi import 块中的 `getDeptTree` (原 line 12)
   - 新增 `import { useDeptTree } from "@/hooks/useDeptTree"`
   - `type SimpleDept` 保留（`UseWorkOrderDataReturn.depts` 字段类型 line 53 仍引用）

2. **hook 函数体**：
   - 删除 `const [depts, setDepts] = useState<SimpleDept[]>([])` (原 line 83)
   - 新增 `const { data: depts = [] } = useDeptTree()`
   - 删除 `fetchDepts` useCallback (原 line 149-156, 调用 workorderApi 的 getDeptTree 副本)
   - useEffect 依赖数组移除 `fetchDepts` (原 line 174)
   - return 对象移除 `fetchDepts` 字段 (原 line 189)

3. **UseWorkOrderDataReturn interface**：
   - 删除 `fetchDepts: () => Promise<void>` 字段 (原 line 58)
   - `depts: SimpleDept[]` 字段保留 (消费方 `orders/index.tsx` 解构 `depts` 用于编辑 modal 的 `<Select>` 渲染 line 597)

**workorderApi.ts 类型去重 (T-37-09 mitigate)**:

1. **SimpleDept 类型 re-export** (原 line 253-258 本地 interface → re-export)：
   ```typescript
   // 改造前
   export interface SimpleDept {
     id: string;
     deptName: string;
     parentId?: string;
     children?: SimpleDept[];
   }

   // 改造后
   export type { SimpleDept } from "./dutyApi";
   ```
   - **外部 import 路径不变**：`pages/workorder/orders/index.tsx:64` 与 `useWorkOrderData.ts:15` 的 `import { type SimpleDept } from "@/lib/workorderApi"` 仍有效
   - **内部引用不变**：`WorkOrder.department?: SimpleDept` (line 84) 通过 re-export 间接引用 dutyApi 的 canonical 定义

2. **getDeptTree 副本删除** (原 line 575-577)：
   - 全项目 grep 确认 workorderApi.getDeptTree 的唯一消费方是 useWorkOrderData.ts (已迁移到 hook)
   - canonical getDeptTree 仍在 dutyApi.ts:303 (由 useDeptTree 内部调用)
   - 消费方若需直接调用 (不推荐),应改 `import { getDeptTree } from "@/lib/dutyApi"`

**orders/index.tsx 未改动**：
- `import { getDeptList, type SimpleDept } from "@/lib/workorderApi"` (line 62-65) 保持不变
- `getDeptList` 是另一个函数 (`/system/departments/list` 端点),与本 plan 无关,保留
- `type SimpleDept` 经 re-export 仍从 workorderApi 解析到 dutyApi 的 canonical 定义
- `depts` 解构自 `useWorkOrderData()` 返回值 (line 233),用于编辑 modal 的部门 `<Select>` line 597

## Verification

### Acceptance Criteria (全部通过)

| Criterion | Expected | Actual | Status |
|-----------|----------|--------|--------|
| `grep -c "fetchDepts" duty/pools/index.tsx` | = 0 | 0 | ✅ |
| `grep -cE "getDeptTree\|/system/departments/tree" duty/pools/index.tsx` | = 0 | 0 | ✅ |
| `grep -c "useDeptTree" duty/pools/index.tsx` | = 1 (调用点) | 1 调用点 + 3 注释提及 = 4 | ✅ |
| `grep -rn "interface SimpleDept" src/` 全项目总和 | = 1 (仅 dutyApi) | 1 (dutyApi.ts:281) | ✅ |
| `grep -c "export type { SimpleDept } from \"./dutyApi\"" workorderApi.ts` | = 1 | 1 | ✅ |
| `grep -c "fetchDepts" useWorkOrderData.ts` | = 0 | 0 | ✅ |
| `grep -c "useDeptTree" useWorkOrderData.ts` | = 1 (调用点) | 1 调用点 + 3 注释提及 = 4 | ✅ |
| `grep -c "getDeptTree" workorderApi.ts` | = 0 (副本已删) | 0 | ✅ |
| `grep -rn "/system/departments/tree" pages/duty/ pages/workorder/` | = 0 | 0 | ✅ |
| `grep -rc "interface SimpleDept" src/lib/` 总和 | = 1 (仅 dutyApi) | 1 (dutyApi), workorderApi=0 | ✅ |
| `npm run type-check` | exit 0 | exit 0 (无错误输出) | ✅ |
| `DepartmentTreeSelect departments={depts}` duty/pools 两处保留 | = 2 | 2 (line 418 + line 491) | ✅ |

### 行为保持验证

**静态行为等价性分析**：

| 维度 | 迁移前 | 迁移后 | 等价 |
|------|--------|--------|------|
| duty/pools 部门数据来源 | `getDeptTree()` (dutyApi fetch) | `useDeptTree()` (内部调 dutyApi.getDeptTree 同端点) | ✅ 同端点同数据 |
| duty/pools 部门缓存 | 每页独立 fetch,无共享 | React Query `['dept','tree']` 5min stale,跨页共享 | ✅ 等价且更优 |
| duty/pools DepartmentTreeSelect 显示 | `depts` (SimpleDept[]) 经 toFullPathTree({ startFromLevel: 2 }) 转换 | 同上 (depts 仍是 SimpleDept[], 37-01 Task 2 已统一) | ✅ 完全相同 |
| duty/pools `getDeptAndChildrenIds` 行为 | 递归 `depts` children 收集 ID | 同上 (depts 结构不变, hook 返回的 SimpleDept[] 含 children) | ✅ 完全相同 |
| workorder 部门数据来源 | `getDeptTree()` (workorderApi 副本 fetch) | `useDeptTree()` (canonical dutyApi.getDeptTree) | ✅ 同端点 `/system/departments/tree` |
| workorder 编辑 modal 部门下拉 | `depts.filter(dept => dept.id).map(...)` (orders/index.tsx line 597) | 同上 (depts 仍为 SimpleDept[], filter/map 不变) | ✅ 完全相同 |
| workorderApi.SimpleDept 类型 | 本地重定义 (与 dutyApi 完全相同字段) | re-export from dutyApi (canonical) | ✅ 运行时零变化 (TS 类型层面) |
| workorderApi.WorkOrder.department 字段 | 引用本地 SimpleDept | 引用 re-export 的 SimpleDept (同一 canonical 定义) | ✅ 运行时零变化 |

**关键不变量**：
- DepartmentTreeSelect 受控模式保持 (D-LOCKED)：组件内部 `useDeptTree` 调用 = 0，数据由调用方从 hook 喂入
- `depts` 在两页都是 `SimpleDept[]` (DeptTreeNode 别名)，与 37-01 Task 2 改造后的 props 类型严格一致
- `getDeptAndChildrenIds` (duty/pools) 与 `depts.filter(dept => dept.id).map(...)` (workorder) 的运行时行为完全不变

## Deviations from Plan

无。plan 执行完全按写定的 action 步骤完成：
- Task 1 删除 fetchDepts + depts useState + getDeptTree import + useEffect 触发，改 useDeptTree；DepartmentTreeSelect 两处受控数据由 hook 喂入
- Task 2 useWorkOrderData 同步迁移；workorderApi.SimpleDept 改 re-export；getDeptTree 副本删除（消费方 grep 确认仅 useWorkOrderData 一处）

## Known Stubs

无。本批 4 为纯重构，无 placeholder/TODO/mock 数据。

## Threat Flags

无新增 threat surface。threat_model 既有条目覆盖：
- T-37-08 (Information Disclosure, accept): duty/pools + workorder 部门数据展示数据来源不变
- T-37-09 (Tampering, mitigate): workorderApi.SimpleDept re-export 引用断裂风险已消除——删除前 grep 确认所有 `from "@/lib/workorderApi"` 的 SimpleDept 引用（orders/index.tsx + useWorkOrderData.ts）经 re-export 后仍有效；getDeptTree 副本消费方仅 useWorkOrderData 一处，已同步迁移
- T-37-SC (n/a, accept): 0 新依赖

## TDD Gate Compliance

本 plan frontmatter `type: execute`（非 `tdd`），无 plan-level TDD gate 强制要求。
本批 4 为行为保持型重构（纯数据源替换 + 类型去重），静态行为等价性分析已逐维度证明等价。

## Self-Check: PASSED

### Created/Modified files exist

- ✅ FOUND: `xingran-react-frontend/src/pages/duty/pools/index.tsx` (modified)
- ✅ FOUND: `xingran-react-frontend/src/pages/workorder/orders/hooks/useWorkOrderData.ts` (modified)
- ✅ FOUND: `xingran-react-frontend/src/lib/workorderApi.ts` (modified)
- ✅ FOUND: `.planning/phases/37-dept-select-unify/37-05-SUMMARY.md` (this file)

### Commits exist

- ✅ FOUND: ccbd504 (refactor(37-05): duty/pools 迁移到 useDeptTree)
- ✅ FOUND: 0cde4f3 (refactor(37-05): workorder 迁移 + workorderApi 类型/函数去重)

### D-LOCKED decisions honored

- ✅ DepartmentTreeSelect 在 duty/pools 受控模式保持（两处 `departments={depts}` 不变，组件内部未调 hook）
- ✅ workorderApi.SimpleDept 改 re-export 而非本地删除（保持外部 import 路径不变，向后兼容）
- ✅ workorderApi.getDeptTree 副本删除，canonical 在 dutyApi.ts 唯一
- ✅ AD 域控整模块未触碰（数据源级别错误防护边界保留）
- ✅ role 的 tree-select 端点 `/system/departments/tree-select` 未触碰
- ✅ system/dept 部门管理页本体未触碰（管理者而非消费者）
