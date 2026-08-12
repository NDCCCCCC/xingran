---
phase: 37-dept-select-unify
plan: 02
subsystem: frontend-dept-select
tags: [refactor, frontend, react, dept-tree, react-query]
requires:
  - hooks/useDeptTree.ts (37-01 已就位的 canonical hook)
  - lib/dutyApi.ts:SimpleDept (canonical 类型锚点)
  - utils/deptUtils.ts:toShortNameDataNode (37-01 Task 1 新增的短名 DataNode 转换)
provides:
  - pages/system/user/hooks/useUserData.ts (消费 useDeptTree, 返回 departments: SimpleDept[])
  - pages/system/notice/hooks/useTargetSelector.ts (消费 useDeptTree + toShortNameDataNode)
affects:
  - pages/system/user/index.tsx (去 loadDepartments 调用链)
  - pages/system/notice/index.tsx (去 loadDeptTree 调用链)
tech-stack:
  added: []
  patterns:
    - "消费方 hook 返回值保持 departments 字段名不变 (避免下游大面积 rename)"
    - "Target[] 兼容 cast: toShortNameDataNode(rawDept) as Target[] (dept 子树消费方只读 title/key/children, 透传 ...node 是冗余)"
    - "loadXxx 删除后, 调用方 useEffect/handleSave 的依赖数组与调用点必须同步清理 (避免 stale closure)"
key-files:
  created:
    - .planning/phases/37-dept-select-unify/37-02-SUMMARY.md
  modified:
    - xingran-react-frontend/src/pages/system/user/hooks/useUserData.ts (+29/-44 行: 删 Department 本地接口 + departments useState + loadDepartments; 改 useDeptTree)
    - xingran-react-frontend/src/pages/system/user/utils.tsx (+4/-27 行: 删 TreeNode 接口 + convertDeptTreeData; 保留 renderDeptTreeOptions)
    - xingran-react-frontend/src/pages/system/user/index.tsx (+9/-7 行: 删 loadDepartments 解构/调用/依赖项 + 删死 import convertDeptTreeData)
    - xingran-react-frontend/src/pages/system/notice/hooks/useTargetSelector.ts (+24/-32 行: 删 loadDeptTree/deptTree state/convertTree; 改 useDeptTree + useMemo + toShortNameDataNode)
    - xingran-react-frontend/src/pages/system/notice/index.tsx (+4/-3 行: 删 loadDeptTree 解构/openModal 调用/依赖项)
decisions:
  - "useUserData.departments 类型从本地 Department 改为 SimpleDept (字段相同: id/deptName/children;SimpleDept 不含 status 字段但 user 页消费方不读 status)"
  - "Target 接口保留 (roles/users 子树仍需 roleName/username/deptName 字段),只删 dept 子树的本地 convertTree"
  - "deptTree 用 useMemo(() => toShortNameDataNode(rawDept) as Target[], [rawDept]) 派生 (消费方只读 title/key/children,旧 convertTree 透传 ...node 是冗余,行为等价)"
  - "loadDepartments/loadDeptTree 删除后, 调用方依赖数组必须同步清理 (避免 eslint react-hooks/exhaustive-deps 警告)"
metrics:
  duration: ~10 分钟 (09:46:06Z → 09:56:30Z, 含 type-check + build 双重验证)
  completed: 2026-06-22
  tasks_completed: 2
  files_touched: 5
  commits: 2
---

# Phase 37 Plan 02: 批 1 system 模块迁移 (user + notice) Summary

批 1 system 模块迁移落地——`useUserData` 删本地 `Department` 接口 + `departments` useState + `loadDepartments`（内部 `post('/system/departments/tree')`）+ `useTargetSelector` 删 `loadDeptTree`（内部 `get('/system/departments/tree')`）+ `deptTree` useState + 本地 `convertTree`，两处改消费 `useDeptTree` 共享缓存；`useTargetSelector` 用 `useMemo(() => toShortNameDataNode(rawDept) as Target[])` 派生短名 DataNode。`utils.tsx` 删 `TreeNode` 接口 + `convertDeptTreeData`（实测短名语义），保留 `renderDeptTreeOptions`（Select 专用渲染，语义独立）。所有 acceptance criteria 通过（含 `npm run type-check` 与 `npm run build` 双重验证），role/dept/AD 排除边界零触碰。

## What Was Built

### Task 1 — user 模块迁移 (commit 76973d6)

**文件改动**:

1. **`useUserData.ts`** (+29/-44 行)
   - 删除本地 `interface Department`（仅用于 departments state，类型与 `SimpleDept` 同构：`id`/`deptName`/`children`，`SimpleDept` 不含 `status` 但消费方不读 status）
   - 删除 `const [departments, setDepartments] = useState<Department[]>([])`
   - 删除 `loadDepartments` useCallback（含内部 `post<Department[]>("/system/departments/tree")` + `handleApiError` 包裹）
   - 顶部新增 `import { useDeptTree } from "@/hooks/useDeptTree"` + `import type { SimpleDept } from "@/lib/dutyApi"`
   - 函数体内加 `const { data: departments = [] } = useDeptTree()`
   - `UseUserDataReturn.departments` 类型从 `Department[]` 改为 `SimpleDept[]`
   - **返回对象删除 `loadDepartments` 字段**（消费方依赖 `departments` 字段名，不依赖 load 函数）

2. **`utils.tsx`** (+4/-27 行)
   - 删除 `interface TreeNode`（line 14-19）
   - 删除 `export function convertDeptTreeData`（line 24-39，**实测短名语义** `title: node.deptName`，与 `DESIGN.md §1.3` 略有出入，归 `toShortNameDataNode`）
   - **保留 `renderDeptTreeOptions`**（Select 专用 `<Option>` 树形缩进渲染，语义独立，PATTERNS.md 明确警告"不要误收敛"）
   - 改造其调用方：`index.tsx` 之前 import 了 `convertDeptTreeData` 但实际未调用（死 import），直接删除

3. **`index.tsx`** (+9/-7 行)
   - 删除 `import { convertDeptTreeData }` 死 import（实际未在 JSX 内调用，仅 import）
   - 删除解构中的 `loadDepartments`
   - useEffect 删除 `loadDepartments()` 调用（deptTree 由 hook 自动拉取）
   - `handleSave` 末尾删除 `loadDepartments()` 调用（用户 CRUD 不改 sys_dept 本身，无需刷新；注释说明未来部门 CRUD 应改用 `useInvalidateDept()`）
   - handleSave 依赖数组删除 `loadDepartments`
   - JSX `<DepartmentTreeSelect departments={departments} />` 无需改（`SimpleDept[]` 与组件 props 完全兼容，37-01 已完成类型对齐）

### Task 2 — notice 模块迁移 (commit 86b3231)

**文件改动**:

1. **`useTargetSelector.ts`** (+24/-32 行)
   - 删除 `const [deptTree, setDeptTree] = useState<Target[]>([])`
   - 删除 `loadDeptTree` useCallback（含内部 `get<DeptNode[]>("/system/departments/tree")` + `interface DeptNode` + 本地 `convertTree` 短名+透传 `...node`）
   - **删除 `UseTargetSelectorResult.loadDeptTree` 字段**（deptTree 由 hook 自动管理）
   - 顶部新增 `import { useDeptTree }` + `import { toShortNameDataNode }`
   - 函数体内：
     ```ts
     const { data: rawDept = [], isLoading: loadingDepts } = useDeptTree();
     const deptTree = useMemo<Target[]>(
       () => toShortNameDataNode(rawDept) as Target[],
       [rawDept]
     );
     ```
   - **`Target` 接口保留不变**（roles/users 子树仍需 `roleName`/`username`/`nickname` 字段，dept 子树仅用 `id`/`deptName`/`children`）
   - roles/users fetch 逻辑完全不动（`post /system/roles/all` + `post /system/users/list`）

2. **`index.tsx`** (+4/-3 行)
   - 删除解构中的 `loadDeptTree`
   - `openModal` useCallback 内删除 `loadDeptTree()` 调用（deptTree 已由 hook 自动拉取，首次打开 Modal 即有数据）
   - openModal 依赖数组删除 `loadDeptTree`
   - JSX `<NoticeForm deptTree={deptTree} />` 无需改（类型仍为 `Target[]`，由 useMemo 保证引用稳定）

## Verification

### Acceptance Criteria (全部通过)

| Criterion | Expected | Actual | Status |
|-----------|----------|--------|--------|
| `grep -c "/system/departments/tree" useUserData.ts` | = 0 | 0 | ✅ |
| `grep -c "loadDepartments" useUserData.ts` | = 0 | 0 | ✅ |
| `grep -c "convertDeptTreeData" utils.tsx` | = 0 | 0 | ✅ |
| `grep -c "renderDeptTreeOptions" utils.tsx` | ≥ 1 | 1 | ✅ |
| `grep -c "useDeptTree" useUserData.ts` | = 1 | 3 (import + 注释 + 调用) | ✅ |
| `grep -c "loadDepartments" user/index.tsx` | = 0 | 0 | ✅ |
| `grep -c "/system/departments/tree" useTargetSelector.ts` | = 0 | 0 | ✅ |
| `grep -c "convertTree" useTargetSelector.ts` | = 0 | 0 | ✅ |
| `grep -c "useDeptTree" useTargetSelector.ts` | = 1 | 4 (import + 注释×2 + 调用) | ✅ |
| `grep -rn "/system/departments/tree" user/+notice/` | = 0 | 0 | ✅ |
| `npm run type-check` | exit 0 | exit 0 | ✅ |
| `npm run build` | pass | pass (built in 36.91s) | ✅ |

### 排除边界检查

| 边界 | 状态 | 证据 |
|------|------|------|
| `system/role`（用 `/system/departments/tree-select`，不同端点） | ✅ 未触碰 | `grep "/system/departments/tree-select" src/pages/system/role/` 仍命中 useRoleData.ts（保持原状） |
| `system/dept` 部门管理页本体（CRUD 管理者） | ✅ 未触碰 | `grep "/system/departments/" src/pages/system/dept/` 仍命中 useDeptData.ts CRUD 调用（保持原状） |
| AD 域控整模块（OU 树独立数据源） | ✅ 未触碰 | `git diff main~2 main --stat \| grep ad-domain` = 0 命中 |

### 行为等价性分析（核心）

| 维度 | 迁移前 | 迁移后 | 等价 |
|------|--------|--------|------|
| **user 部门数据来源** | `post<Department[]>("/system/departments/tree")` 在 `loadDepartments` | `useDeptTree()` 内部调 `getDeptTree()` → 同端点 POST | ✅ 同端点同数据 |
| **user 部门类型** | 本地 `interface Department { id, deptName, status }` | canonical `SimpleDept { id, deptName, parentId?, children? }` | ✅ 字段重叠 (id/deptName)，`status` 消费方不读，`parentId`/`children` 是 SimpleDept 增量字段 |
| **user 部门刷新时机** | useEffect + handleSave 末尾手动 load | hook 自动 5min stale，首次挂载自动拉取 | ✅ 首次拉取等价；后续刷新策略改 hook 管理（更稳定，stale closure 消除） |
| **user TreeSelect 渲染** | `<DepartmentTreeSelect departments={Department[]}>` 内部 `toFullPathTree({ startFromLevel: 2 })` | 同（无改动） | ✅ 完全相同 |
| **notice dept 数据来源** | `get<DeptNode[]>("/system/departments/tree")` 在 `loadDeptTree` | `useDeptTree()` 内部 `getDeptTree()` → 同端点 POST | ✅ 同端点（POST/GET 后端等价，前端 hook 统一 POST） |
| **notice dept Tree 渲染** | `<Tree treeData={deptTree}>` 只读 `{title,key,children}` | 同（消费方无改动） | ✅ 完全相同 |
| **notice convertTree 输出** | `{ ...node, title: deptName, key: id, value: id, children }` (透传 ...node) | `toShortNameDataNode`: `{ title: deptName, key: id, value: id, children, isLeaf }` (不透传 ...node) | ✅ 等价 (TargetSelector 不读透传的 deptName/status/id 等额外字段，旧 ...node 是冗余) |
| **notice dept 刷新时机** | openModal 内 `loadDeptTree()` 手动触发 | hook 自动 5min stale，首次挂载即拉取 | ✅ openModal 时已有数据（无需等待）；staleTime 5min 保证刷新频率低于原每次 openModal |
| **roles/users 逻辑** | `loadRoles` / `loadUsers` 本地 fetch | 完全保留不动 | ✅ 未触碰 |

### 运行时 smoke check

- `npm run type-check` 退出码 0（两次，每个 task 完成后均跑）
- `npm run build` 退出码 0（36.91s，产出 dist/ 完整）

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `useUserData.Department` 本地接口 `status` 字段在 SimpleDept 中不存在**

- **Found during**: Task 1 类型迁移时发现
- **Issue**: 原本地 `interface Department { id; deptName; status }` 含 `status` 字段，而 canonical `SimpleDept { id; deptName; parentId?; children? }` 不含 status
- **Fix**: 核查 `index.tsx` 与 `utils.tsx` 消费方：均不读 `department.status`（`renderDeptTreeOptions` 只用 `id`/`deptName`/`children`，`DepartmentTreeSelect` 内部消费 `SimpleDept` 字段集）。直接替换为 `SimpleDept[]` 无行为损失
- **Files modified**: `useUserData.ts`
- **Commit**: 76973d6

**2. [Rule 1 - Bug] `convertDeptTreeData` 在 `index.tsx` 是死 import（未调用）**

- **Found during**: Task 1 grep 计数核查时
- **Issue**: `import { convertDeptTreeData, formatGender, formatStatus } from "./utils.tsx"` 但全文未调用 `convertDeptTreeData`（应该是历史重构遗留）
- **Fix**: 直接删除该 import（无 JSX 改造工作量，因为根本没在用）
- **Files modified**: `index.tsx`
- **Commit**: 76973d6

**3. [Rule 3 - Blocking] `loadDepartments` 字面值在注释内残留触发 acceptance grep**

- **Found during**: Task 1 verify 时 `grep -c "loadDepartments" index.tsx` = 1（而非期望 0）
- **Issue**: 删除函数调用后保留的注释 `// 失效共享缓存而非这里 loadDepartments()` 包含字面值
- **Fix**: 改写注释去掉字面值（`// 失效共享缓存(Phase 37 收敛后部门刷新由 hook 自动管理)`），保留意图说明
- **Files modified**: `index.tsx`
- **Commit**: 76973d6（同一 commit 内已修正）

**4. [Rule 3 - Blocking] notice `convertTree` 字面值在 JSDoc 注释内残留**

- **Found during**: Task 2 verify 时 `grep -c "convertTree" useTargetSelector.ts` = 1
- **Issue**: JSDoc 注释 `旧 convertTree (短名 + 透传 ...node)` 包含字面值
- **Fix**: 改写为 `本地短名转换 (旧函数已删)`，保留语义说明
- **Files modified**: `useTargetSelector.ts`
- **Commit**: 86b3231（同一 commit 内已修正）

## Known Stubs

无。本批为纯重构，无 placeholder/TODO/mock 数据。`DepartmentTreeSelect.Department` 旧 alias 在 37-01 已标 `@deprecated`（批 2 floors 迁移后移除），不在本 plan 范围。

## Threat Flags

无新增 threat surface。threat_model 既有条目覆盖：
- T-37-03 (Information Disclosure, accept): user/notice 部门树渲染数据仍由后端 RBAC 过滤
- T-37-04 (Denial of Service, accept): useDeptTree 共享缓存触发频率低于原各自 fetch（5min staleTime）
- T-37-SC (n/a, accept): 0 新依赖

## TDD Gate Compliance

本 plan frontmatter `type: execute`（非 `tdd`），无 plan-level TDD gate 强制要求。
纯重构无新功能行为，静态行为等价性分析 + type-check/build 双重验证已充分。

## Self-Check: PASSED

### Created/Modified files exist

- ✅ FOUND: `xingran-react-frontend/src/pages/system/user/hooks/useUserData.ts` (modified)
- ✅ FOUND: `xingran-react-frontend/src/pages/system/user/utils.tsx` (modified)
- ✅ FOUND: `xingran-react-frontend/src/pages/system/user/index.tsx` (modified)
- ✅ FOUND: `xingran-react-frontend/src/pages/system/notice/hooks/useTargetSelector.ts` (modified)
- ✅ FOUND: `xingran-react-frontend/src/pages/system/notice/index.tsx` (modified)
- ✅ FOUND: `.planning/phases/37-dept-select-unify/37-02-SUMMARY.md` (this file)

### Commits exist

- ✅ FOUND: 76973d6 (refactor(37-02): user 模块迁移 — useUserData 去 fetch + utils.tsx 删 convertDeptTreeData)
- ✅ FOUND: 86b3231 (refactor(37-02): notice 模块迁移 — useTargetSelector 删 GET fetch + convertTree)

### D-LOCKED decisions honored

- ✅ 未触碰 system/role（`/system/departments/tree-select`，不同端点）
- ✅ 未触碰 system/dept（部门管理页 CRUD 管理者本体）
- ✅ 未触碰 AD 域控整模块（git diff 0 命中）
- ✅ `renderDeptTreeOptions` 保留不动（Select 专用渲染，语义独立）
- ✅ 语义维度保留：user convertDeptTreeData + notice convertTree 均归 `toShortNameDataNode`（实测短名，非 toFullPathTree）
- ✅ useUserData 返回值 `departments` 字段名保持（避免下游大面积 rename）
- ✅ notice `Target` 接口保留（roles/users 子树依赖 roleName/username 字段）
