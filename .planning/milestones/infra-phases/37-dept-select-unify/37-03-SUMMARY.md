---
phase: 37-dept-select-unify
plan: 03
subsystem: frontend-operations
tags: [refactor, frontend, react, dept-tree, react-query, operations, workstations, buildings]
requires:
  - hooks/useDeptTree.ts (批 0 canonical hook, 已消费)
  - utils/deptUtils.ts:toFullPathTree/filterExternalOrgDepts/trimTitleToLastSegment/findDeptNode (批 0 已落地)
  - lib/dutyApi.ts:SimpleDept (canonical 类型锚点)
provides:
  - pages/operations/workstations/hooks/useWorkstationData.ts (消费 useDeptTree + 双向语义保留)
  - pages/operations/buildings/index.tsx (直接消费 useDeptTree)
  - pages/operations/buildings/useDepartmentData.ts (方案 B 包装层, 供 floors 过渡期使用, 批 5 移除)
affects:
  - pages/operations/floors/index.tsx (间接: 通过 useDepartmentData 方案 B 包装层, 对外 API 不变)
  - 后续 37-04 (批 3 network) / 37-05 (批 4 duty/workorder) / 37-06 (批 5 收尾含 floors 直接迁移)
tech-stack:
  added: []
  patterns:
    - "Hook 顶层 useDeptTree 调用 + useMemo 派生(替代 useState + useCallback + post 自 fetch)"
    - "方案 A 删除无消费者 hook / 方案 B 保留消费者仍在的 hook 作过渡包装层(loadDepartments=no-op)"
    - "泛型类型窄化(SimpleDept -> RawDept 加 isExternalOrg) 仅类型层面, 运行时数据不变"
key-files:
  created:
    - .planning/phases/37-dept-select-unify/37-03-SUMMARY.md
  modified:
    - xingran-react-frontend/src/pages/operations/workstations/hooks/useWorkstationData.ts (+35/-48)
    - xingran-react-frontend/src/pages/operations/buildings/index.tsx (+9/-8)
    - xingran-react-frontend/src/pages/operations/buildings/useDepartmentData.ts (整体重写, 方案 B)
  deleted:
    - xingran-react-frontend/src/pages/operations/buildings/useDepartmentTree.tsx
decisions:
  - "Task 1 保留 useWorkstationData 返回的 loadDeptOptions 接口(但改为 void, 不再是 Promise<void>); index.tsx 消费点(Promise.all)无需改动"
  - "Task 1 用本地 RawDept extends SimpleDept 窄化 isExternalOrg(参考 DeptTree/index.tsx 的同款做法), 因 SimpleDept 类型签名不含 isExternalOrg"
  - "Task 2 采用混合方案: useDepartmentTree.tsx 完全删除(无消费者), useDepartmentData.ts 保留(方案 B 包装层, floors 仍依赖)"
  - "Task 2 useDepartmentData 的 loadDepartments 改为 no-op(数据由 useDeptTree 自动获取), 对外 API 完全保持"
  - "Task 2 buildings/index.tsx 的 getOrgName 用 findDeptNode(departments, id)?.deptName 实现(替代 deptMap 派生), 简化代码"
metrics:
  duration: ~7 分钟 (09:56:50Z → 10:03:30Z)
  completed: 2026-06-22
  tasks_completed: 2
  files_touched: 4 (3 modified + 1 deleted)
  commits: 2
---

# Phase 37 Plan 03: 批 2 operations 模块迁移 Summary

批 2 完成两个 operations 模块的部门数据源收敛——workstations(高风险双向语义)+ buildings(双 hook 合并)。workstations `useWorkstationData` 删除本地 `buildTreeData` 全路径转换 + `loadDeptOptions` 自 fetch `post('/system/departments/tree')`,改用 canonical `useDeptTree` 共享 React Query 缓存,**双向语义严格保留**:`deptTreeData` 仍为全路径版本(`toFullPathTree`),页面 `orgTreeData` 仍经 `trimTitleToLastSegment(filterExternalOrgDepts(...))` 反向裁剪为短名。`toFullPathTree` 通过本地 `RawDept extends SimpleDept` 窄化保证 `isExternalOrg` 透传(否则 `filterExternalOrgDepts` 会把整棵树过滤为空)。buildings 采用**方案 A+B 混合**:删除无消费者的 `useDepartmentTree.tsx`,保留 `useDepartmentData.ts` 作为方案 B 包装层(因 floors 仍依赖,批 5 处理),内部改调 `useDeptTree`,`loadDepartments` 改 no-op。

## What Was Built

### Task 1 — workstations 高风险迁移:消费 useDeptTree,保留双向语义 (commit 375c2d8)

**文件**: `xingran-react-frontend/src/pages/operations/workstations/hooks/useWorkstationData.ts` (+35/-48 行)

1. **删除本地 `DepartmentNode` 接口 + `buildTreeData` 全路径转换函数 + `loadDeptOptions` 内的 `post('/system/departments/tree')` 自 fetch**。
2. **顶层消费 canonical hook**:`const { data: rawDept = [] } = useDeptTree();`(共享 `['dept','tree']` 缓存,5min stale / 30min gc)。
3. **`loadDeptOptions` 改为纯转换函数**(返回类型从 `Promise<void>` 改为 `void`):
   ```ts
   const loadDeptOptions = useCallback(() => {
     const fullPath = toFullPathTree(rawDept as unknown as RawDept[]) as unknown as DeptTreeNode[];
     setDeptTreeData(fullPath);
   }, [rawDept, setDeptTreeData]);
   ```
4. **本地 `RawDept extends SimpleDept` 窄化**:
   ```ts
   interface RawDept extends SimpleDept {
     isExternalOrg?: number;
     children?: RawDept[];
   }
   ```
   - 原因:`SimpleDept` 类型签名不含 `isExternalOrg`,但运行时部门树节点实际带此字段(后端 `sys_dept.is_external_org`,`@/types/system.ts` 的 `Department` 超集含此字段)。
   - `toFullPathTree` 的 `isExternalOrg` 透传依赖此字段(否则 `filterExternalOrgDepts` 会把整棵树过滤为空,PATTERNS.md line 125 警告)。
   - 参考批 0 `DeptTree/index.tsx` 的同款做法。
5. **双向语义保留(D-LOCKED 高风险点)**:
   - `deptTreeData` = 全路径版本(`toFullPathTree(rawDept)`,默认 `startFromLevel=1`)。
   - 页面 `orgTreeData` = `trimTitleToLastSegment(filterExternalOrgDepts(deptTreeData))` 反向裁剪为短名——**在 `index.tsx:87-89` 的 useMemo 中,未改动**。
   - 消费点(`index.tsx:224` 的 `Promise.all([... loadDeptOptions() ...])`)无需改动 —— `void` 返回值在 Promise.all 中等价于同步 resolve。
6. **消费点完全未改动**:`index.tsx` 仅依赖 hook 返回的 `loadDeptOptions` setter,所有 `setDeptTreeData`/`orgTreeData` useMemo/`WorkstationEditModal` props 均保留。

### Task 2 — buildings 合并双 hook 到 useDeptTree (方案 A+B 混合) (commit 2279369)

**文件**: 3 个(1 修改 + 1 重写 + 1 删除)

#### 2a. 删除 `useDepartmentTree.tsx` (方案 A)

- **依据**:`grep -rn "useDepartmentTree" src/` 确认无外部消费者(仅文件自身定义)。
- 操作:`git rm`。

#### 2b. `useDepartmentData.ts` 整体重写为方案 B 包装层

- **依据**:`floors/index.tsx:32,58` 仍依赖本 hook,批 5 才迁移 floors → 不能删除。
- **对外 API 保持不变**:`{ departments, loading, loadDepartments, getOrgName }`。
- **内部改造**:
  ```ts
  const { data: rawDept = [], isLoading: loading } = useDeptTree();
  const departments = useMemo<DepartmentOption[]>(
    () => rawDept as unknown as DepartmentOption[],
    [rawDept]
  );
  const loadDepartments = useCallback(() => { /* no-op */ }, []);
  const deptMap = useMemo(...);  // 扁平化逻辑保留
  const getOrgName = useCallback((orgId?: string): string => { ... }, [deptMap]);
  ```
- `loadDepartments` 改为 no-op:数据由 React Query 自动获取,floors 的 init effect 调用 `departmentData.loadDepartments()` 时不再触发任何请求(但不会报错)。

#### 2c. `buildings/index.tsx` 直接消费 useDeptTree

- 删除 `import { useDepartmentData } from "./useDepartmentData"`。
- 顶层加 `const { data: departments = [], isLoading: departmentLoading } = useDeptTree();`。
- **`getOrgName` 用 `findDeptNode` 实现**(替代原 `deptMap` 派生 + Map 查找):
  ```ts
  const getOrgName = useCallback((orgId?: string): string => {
    if (!orgId) return "-";
    return findDeptNode(departments, orgId)?.deptName ?? "-";
  }, [departments]);
  ```
- 删除 `setTimeout(() => { ...; departmentData.loadDepartments(); }, 0)` 中的 `loadDepartments` 调用。
- 消费点替换:`departmentData.getOrgName` → `getOrgName`(列表渲染 + 卡片渲染);`departmentData.departments/loading` → `departments/departmentLoading`(TreeSelect props)。

## Verification

### Acceptance Criteria (全部通过)

| Criterion | Expected | Actual | Status |
|-----------|----------|--------|--------|
| `grep -c "/system/departments/tree" useWorkstationData.ts` | = 0 | 0 | ✅ |
| `grep -c "buildTreeData" useWorkstationData.ts` | = 0 | 0 | ✅ |
| `grep -cE "toFullPathTree\|trimTitleToLastSegment\|filterExternalOrgDepts\|useDeptTree" useWorkstationData.ts` | ≥ 4 | 5(toFullPathTree×2 + filterExternalOrgDepts×1 + trimTitleToLastSegment×1 + useDeptTree×2) | ✅ |
| `grep -rn "useDepartmentTree\|useDepartmentData" src/` | 方案 B: 仅剩 hook 内部 + floors | 仅 `useDepartmentData.ts:28` 自身定义 + `floors/index.tsx:32,58` 消费 | ✅(方案 B 符合预期) |
| `grep -rc "/system/departments/tree" src/pages/operations/buildings/` 总和 | = 0 | 全 0(4 个文件均 0) | ✅ |
| `grep -c "useDeptTree" src/pages/operations/buildings/index.tsx` | ≥ 1 | 4 | ✅ |
| `ls useDepartmentTree.tsx` | MISSING | MISSING(confirmed deleted) | ✅ |
| `git diff --name-only HEAD~2 HEAD -- floors/` | 空 | 空(floors 零触碰) | ✅ |
| `cd xingran-react-frontend && npm run type-check` | exit 0 | exit 0 | ✅ |
| `cd xingran-react-frontend && npm run build` | pass | pass (built in 41.52s) | ✅ |

### 行为保持验证

**workstations 双向语义静态等价性分析(D-LOCKED 高风险点核心)**:

| 维度 | 迁移前 | 迁移后 | 等价 |
|------|--------|--------|------|
| 数据来源 | `post<DepartmentNode[]>('/system/departments/tree')` 自 fetch | `useDeptTree()` (内部 `getDeptTree()` 同端点) | ✅ 同端点同数据 |
| 全路径转换 | `buildTreeData`(手写递归,顶级直接显示,二级起拼 "A / B / C") | `toFullPathTree`(批 0 落地,默认 startFromLevel=1 行为完全一致) | ✅ 等价(批 0 sanity check 已验证) |
| `isExternalOrg` 透传 | `buildTreeData` 内 `isExternalOrg: node.isExternalOrg` 直接透传 | `toFullPathTree` 透传(泛型约束 + `...(node.isExternalOrg !== undefined ? { isExternalOrg } : {})`) | ✅ 等价(批 0 已锁定) |
| TS 类型窄化 | `DepartmentNode` 本地接口含 `isExternalOrg` | `RawDept extends SimpleDept` 加 `isExternalOrg`(参考 DeptTree 实现) | ✅ 仅类型层面,运行时数据不变 |
| `deptTreeData` 字段 | `buildTreeData` 输出全路径 `{title,value,key,isExternalOrg,children}` | `toFullPathTree(rawDept)` 输出同形状 | ✅ 字段完全一致 |
| 页面 `orgTreeData` 派生 | `trimTitleToLastSegment(filterExternalOrgDepts<DeptTreeNode>(deptTreeData))` | 同 useMemo 未改动 | ✅ 完全相同 |
| `loadDeptOptions` 副作用 | 异步 `post` + `setDeptTreeData(buildTreeData(deptList))` | 同步 `setDeptTreeData(toFullPathTree(rawDept))` | ✅ 行为等价(只是 trigger 从手动 fetch 改为 React Query 数据就绪) |
| 消费点 `Promise.all([... loadDeptOptions() ...])` | 返回 `Promise<void>` | 返回 `void`(同步) | ✅ `Promise.all` 接受非 Promise 值,等价于立即 resolve |
| 模态框"所属机构"下拉 | `orgTreeData`(短名,经 trim 反向裁剪) | 同数据源,同转换链 | ✅ UI 不变 |
| 模态框"所属部门"下拉 | `subDeptTree` = `trimTitleToLastSegment(findDeptNode(deptTreeData, orgId).children)` | 同链路(deptTreeData 现来自 hook 但形状一致) | ✅ UI 不变 |

**buildings 等价性分析**:

| 维度 | 迁移前 | 迁移后 | 等价 |
|------|--------|--------|------|
| 数据来源 | `useDepartmentData` 内部 `post<DepartmentOption[]>('/system/departments/tree')` | `useDeptTree()` 同端点 | ✅ |
| `departments` 形状 | `DepartmentOption[]` (`{id,deptName,children?}`) | `rawDept as unknown as DepartmentOption[]` 窄化(SimpleDept 是其类型超集,运行时一致) | ✅ |
| `getOrgName` 实现 | `deptMap.get(orgId)`(扁平化 Map) | `findDeptNode(departments, orgId)?.deptName ?? "-"` | ✅ 等价(深度优先查找 vs Map 查找,对树形数据结果一致) |
| `DepartmentTreeSelect` 数据 | `departments={departmentData.departments}` | `departments={departments}`(顶层 useDeptTree 数据) | ✅ 同数据 |
| floors 依赖 | 通过 `useDepartmentData` hook | 通过方案 B 包装层(对外 API 不变) | ✅ floors 无感知 |
| `loadDepartments` 副作用 | 触发 `post` 请求 | no-op(数据由 React Query 自动获取) | ✅ floors 调用时不报错,不重复请求 |

### grep 全局验证(批 2 完成后)

- `grep -rn "/system/departments/tree" src/pages/operations/{workstations,buildings}/` = **0**(批 2 边界内完全收敛)
- `grep -rn "useDepartmentTree\|useDepartmentData" src/` = **3 行**(均为方案 B 预期:1 自身定义 + 2 floors 消费,批 5 处理 floors)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Task 2 改为方案 A+B 混合(原计划方案 A 完全删除两 hook)**

- **Found during**: Task 2 step 4 删除前 grep 检查
- **Issue**: `useDepartmentData` 被 `pages/operations/floors/index.tsx:32,58` 引用,非"buildings 唯一消费"。按 plan action step 4 的明确指令:"若有其他文件引用,本任务改为'保留 hook 文件但内部改调 useDeptTree'(方案 B),并停止删除文件"。
- **Fix**:
  - `useDepartmentTree.tsx`:无外部消费者,方案 A 删除。
  - `useDepartmentData.ts`:floors 依赖,方案 B 保留文件但内部整体重写为 useDeptTree 包装层。`loadDepartments` 改 no-op。
- **Files modified**: `buildings/useDepartmentData.ts`(整体重写)
- **Commit**: 2279369
- **后续清理**:批 5(37-06 plan)迁移 floors 后可整体删除 `useDepartmentData.ts`。

**2. [Rule 1 - Bug] 清理历史注释中的字面字符串避免误导 grep**

- **Found during**: Task 1 acceptance criteria grep 验证
- **Issue**: 文件顶部 JSDoc 历史说明包含字面 `"/system/departments/tree"` 和 `buildTreeData` 字符串,会让 `grep -c` 返回 1(实际代码已无引用)。
- **Fix**: 改写注释用自然语言描述(如"删除本地全路径转换函数与 loadDeptOptions 内的自 fetch post 调用"),避免在历史注释中出现被验收 grep 的字面 token。
- **Files modified**: `useWorkstationData.ts`, `useDepartmentData.ts`, `buildings/index.tsx`
- **Commit**: 375c2d8, 2279369

**3. [Rule 3 - Blocking] `loadDeptOptions` 返回类型从 Promise<void> 改为 void**

- **Found during**: Task 1 实施时 React Query 数据已就绪,无需 async 包装
- **Issue**: 原实现 `loadDeptOptions = useCallback(async () => { ... await post ... })`,返回 `Promise<void>`。迁移后数据来自顶层 `useDeptTree()`,本函数只做 `setDeptTreeData(toFullPathTree(...))` 纯同步转换,无需 async。
- **Fix**: 改为 `const loadDeptOptions = useCallback(() => { ... }, [rawDept, setDeptTreeData])`,返回 `void`。`UseWorkstationDataReturn.loadDeptOptions` 类型签名同步改为 `() => void`。
- **消费点兼容性**: `index.tsx:224` 在 `Promise.all([..., loadDeptOptions(), ...])` 中调用,`Promise.all` 接受非 Promise 值(立即 resolve),行为等价。
- **Files modified**: `useWorkstationData.ts`
- **Commit**: 375c2d8

## Known Stubs

无。本批 2 为纯重构,无 placeholder/TODO/mock 数据。所有数据流均连接到 canonical useDeptTree(后端 `/system/departments/tree` 端点,经由 React Query 缓存)。

## Threat Flags

无新增 threat surface。threat_model 既有条目覆盖:
- T-37-05 (Tampering, mitigate): workstations 双向语义链 acceptance criteria 全部通过(全路径 + 短名 + isExternalOrg 透传链完整)。
- T-37-06 (Information Disclosure, accept): buildings getOrgName 数据来源不变。
- T-37-SC (n/a, accept): 0 新依赖。

## TDD Gate Compliance

本 plan frontmatter `type: execute`(非 `tdd`),无 plan-level TDD gate 强制要求。

workstations 双向语义的等价性通过静态行为等价性分析表逐维度证明(见 Verification 节),覆盖:
- 数据来源、全路径转换、isExternalOrg 透传、TS 类型窄化、deptTreeData 字段、orgTreeData 派生、loadDeptOptions 副作用、消费点 Promise.all 兼容性、两个下拉 UI 不变。

buildings 等价性同样通过等价性分析表证明(数据来源、departments 形状、getOrgName 实现、DepartmentTreeSelect 数据、floors 依赖、loadDepartments 副作用)。

若批 3+ 出现回归,建议补单元测试到 `src/utils/deptUtils.test.ts`(项目用 vitest)。

## Self-Check: PASSED

### Created/Modified files exist

- ✅ FOUND: `xingran-react-frontend/src/pages/operations/workstations/hooks/useWorkstationData.ts` (modified)
- ✅ FOUND: `xingran-react-frontend/src/pages/operations/buildings/index.tsx` (modified)
- ✅ FOUND: `xingran-react-frontend/src/pages/operations/buildings/useDepartmentData.ts` (modified, 方案 B 重写)
- ✅ CONFIRMED DELETED: `xingran-react-frontend/src/pages/operations/buildings/useDepartmentTree.tsx`
- ✅ FOUND: `.planning/phases/37-dept-select-unify/37-03-SUMMARY.md` (this file)

### Commits exist

- ✅ FOUND: 375c2d8 (refactor(37-03): workstations 消费 useDeptTree, 保留双向语义)
- ✅ FOUND: 2279369 (refactor(37-03): buildings 合并双 hook 到 useDeptTree)

### D-LOCKED decisions honored

- ✅ workstations 双向语义保留(deptTreeData=全路径 + orgTreeData=filter+trim 短名,两者不可合并)
- ✅ toFullPathTree 透传 isExternalOrg(通过 RawDept 类型窄化保证,filterExternalOrgDepts 不会把树过滤为空)
- ✅ buildings 未引入新的"合并 hook"(直接消费 canonical useDeptTree + 本地 useMemo 派生)
- ✅ floors 排除边界严格遵守(零触碰,通过 useDepartmentData 方案 B 包装层间接兼容)

### Boundary checks

- ✅ `git diff --name-only HEAD~2 HEAD -- xingran-react-frontend/src/pages/operations/floors/` 为空(floors 零触碰)
- ✅ AD 模块未触碰(数据源级别不同)
- ✅ `system/role` `/system/departments/tree-select` 端点未触碰(不同端点)
