---
status: issues_found
phase: 37-dept-select-unify
depth: standard
files_reviewed: 18
critical: 1
warning: 1
info: 0
total: 2
reviewed_at: 2026-06-22
reviewer: feature-dev:code-reviewer (standard depth)
diff_base: fb4ea76
---

# Phase 37 Code Review — 部门选择统一收敛

**范围**：18 个源文件（`fb4ea76..HEAD`），standard depth。

## 发现

### CR-1 [Critical] 根目录 `src/components/TargetSelector.tsx` 是零引用 dead code

**文件**：`xingran-react-frontend/src/components/TargetSelector.tsx`

**证据**（全仓 `grep "import.*TargetSelector"`）：
- 唯一引用者在 `pages/system/notice/components/NoticeForm.tsx:7` → `import { TargetSelector } from "./TargetSelector"`
- 该相对路径解析到 `pages/system/notice/components/TargetSelector.tsx`（6月16日，纯展示组件，从 `useTargetSelector` hook 接收 `deptTree` prop），**不是**根目录的 `src/components/TargetSelector.tsx`
- 根目录文件（6月22日，37-06 改造，含 `useDeptTree` + `toShortNameDataNode`）在整个代码库**零引用者**

**影响**：
- 37-06 Task 1 迁移了一个无用文件（加了 `useDeptTree` 但没人用）
- **不影响 phase 37 核心收敛目标** —— notice 的部门数据实际已通过 37-02（`useTargetSelector` hook 迁移到 `useDeptTree`）收敛；活跃的 `notice/components/TargetSelector.tsx` 本就是受控模式（从 hook 喂 `deptTree` prop）
- type-check / build 双通过（改 dead code 不破坏任何东西）

**建议**：删除根目录 `src/components/TargetSelector.tsx`（dead code 清理）。保留 `Department = SimpleDept` alias 时注意：该 alias 在 `DepartmentTreeSelect.tsx:42` 是为 floors 保留，与此 dead 文件无关。

**处理**：标记为 follow-up（不阻塞 phase 完成）。删除文件需确认无动态 import，建议独立提交。

---

### WR-1 [Warning] workstations 初始化 effect 因 `loadDeptOptions` 依赖 rawDept 导致重复请求（37-03 引入的回归）

**文件**：`xingran-react-frontend/src/pages/operations/workstations/hooks/useWorkstationData.ts:93-96`
**消费端**：`xingran-react-frontend/src/pages/operations/workstations/index.tsx:221-230`

**问题链**：
1. `useWorkstationData` 顶层 `useDeptTree()` 首次返回 `rawDept = []`
2. `loadDeptOptions`（useCallback）依赖 `[rawDept, setDeptTreeData]`
3. `index.tsx` init effect 的 `Promise.all([loadStatistics, loadFloorOptions, loadDeptOptions, loadUserOptions])` 依赖数组含 `loadDeptOptions`
4. React Query 部门树数据到达 → `rawDept` 从 `[]` 变 `[...depts]` → `loadDeptOptions` 引用变化 → **init effect 重新运行** → `loadStatistics` / `loadFloorOptions` / `loadUserOptions` 被重复请求

**与迁移前的差异**：原 `loadDeptOptions` 是 `async` + 内部 `post()` fetch，依赖数组仅 `[setDeptTreeData]`（无 `rawDept`），init effect 只运行一次。37-03 改为消费 `useDeptTree` 后，`rawDept` 进入依赖链，引入此回归。

**影响**：页面挂载时多一轮 statistics/floors/users 网络请求（性能问题，非功能错误；最终数据正确）。

**修复方案（最小改动，已应用）**：
- `loadDeptOptions` 改为稳定引用（依赖 `[]`），保留为 init 占位（index.tsx 不动）
- 新增 `useEffect` 监听 `rawDept` 变化时派生 `deptTreeData`（`toFullPathTree` + `setDeptTreeData`）
- 效果：init effect 只在挂载时跑一次；`rawDept` 到达只更新 `deptTreeData`，不触发 init effect 重跑

---

## 通过项（无高置信度问题）

**workstations 双向语义（高风险）正确**：
- `toFullPathTree`（deptUtils）通过 `...node` 展开透传 `isExternalOrg`，外加显式条件添加（冗余但无害）
- `filterExternalOrgDepts` 递归保留 `isExternalOrg===1` 子树 + 祖先链，不会把树过滤为空
- `orgTreeData = trimTitleToLastSegment(filterExternalOrgDepts(deptTreeData))` 语义链正确
- `EditModal` 的 `subDeptTree` 用 `findDeptNode` + `trimTitleToLastSegment` 正确复用全量树

**`useDeptTree` hook 使用规范**：所有调用者在组件/hook 顶层无条件调用，依赖数组正确。

**`useMemo` 依赖完整**：`DeptTree/index.tsx` 的 `treeData`、`buildings/index.tsx` 的 `getOrgName`、`useDepartmentData` 的派生均依赖正确。`as unknown` 窄化在运行时是恒等操作，引用稳定性不受影响。

**受控 `DepartmentTreeSelect` 正确**：通过 `value/onChange/departments/treeData` props 受控，未内部 fetch（符合 D-LOCKED）。antd Form `<Form.Item>` 包裹的调用方通过克隆注入 `value/onChange`，受控模式正确。

**类型安全（re-export + import type）正确**：`workorderApi.ts` 顶部 `import type { SimpleDept }` + `export type { SimpleDept }` 模式正确（`export type` 仅类型层面，本地用需单独 `import type`，37-06 已修复 37-05 的 latent bug）。

**删除 `loadDepartments`/`fetchDepts` 后无悬空引用**：`getDeptTree` 副本已删（仅 dutyApi 定义），`useDepartmentData.loadDepartments` no-op 作为 floors 兼容层继续被调用（返回 undefined，`Promise.all` 接受）。

## 总结

Phase 37 整体收敛逻辑正确。核心工具函数（`toFullPathTree`/`toShortNameDataNode`/`filterExternalOrgDepts`/`trimTitleToLastSegment`）实现稳健，workstations 高风险双向语义正确工作。`useDeptTree` React Query 缓存使用规范。

两个发现：
- **CR-1**：根目录 TargetSelector.tsx 是 dead code（37-06 误迁移）→ follow-up 清理
- **WR-1**：workstations init effect 重复请求（37-03 回归）→ **已修复**（loadDeptOptions 稳定化 + effect 派生 deptTreeData）
