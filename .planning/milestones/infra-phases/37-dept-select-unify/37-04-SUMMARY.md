---
phase: 37-dept-select-unify
plan: 04
subsystem: frontend-dept-select
tags: [refactor, frontend, react, dept-tree, react-query, network]
requires:
  - hooks/useDeptTree.ts (canonical hook, Phase 37-01 已落地)
  - lib/dutyApi.ts:SimpleDept (canonical 类型锚点)
provides:
  - pages/network/devices/hooks/useDeviceData.ts 消费 useDeptTree, 不再自 fetch
affects:
  - 后续批 4/5 (duty/workorder 模块) — 同类迁移模板
  - 0 个下游消费方被破坏 (返回对象 departments 字段保留, loadDepartments 字段被删除但全目录 grep 已清零)
tech-stack:
  added: []
  patterns:
    - "页面级 hook 自 fetch → canonical useDeptTree (同 37-02 模式推广到 network 模块)"
    - "ReturnType<typeof useDeptTree>['data'] 类型透传 (避免重复 import SimpleDept, 与 hook 返回类型严格一致)"
key-files:
  created: []
  modified:
    - xingran-react-frontend/src/pages/network/devices/hooks/useDeviceData.ts (+5/-19, 删 departments useState + loadDepartments useCallback + Department 本地类型引用)
    - xingran-react-frontend/src/pages/network/devices/index.tsx (-3, 删 loadDepartments 解构 + 2 处调用点)
decisions:
  - "UseDeviceDataReturn.departments 类型用 ReturnType<typeof useDeptTree>['data'] 而非直接 import SimpleDept[] — 与 hook 返回类型严格一致, 避免类型漂移"
  - "不删除 useDeviceData.ts 的 post/handleApiError import — loadStatistics/loadCredentials 仍依赖二者 (CLAUDE.md 范围限定原则)"
  - "不删除 index.tsx 的 handleApiError import — handleProbe/handleCreate/handleDelete 等多处仍在使用 (范围外不动)"
metrics:
  duration: ~4 分钟
  completed: 2026-06-22
  tasks_completed: 1
  files_touched: 2
  commits: 1
---

# Phase 37 Plan 04: 批 3 network 模块迁移 devices Summary

批 3 network 模块的最简单迁移——`useDeviceData.ts` 删除自 fetch 的 `loadDepartments` useCallback（含 `post('/system/departments/tree')` + `handleApiError` 包裹 + `setDepartments`），改消费 37-01 落地的 canonical hook `useDeptTree`；返回对象保留 `departments` 字段（消费方依赖此名）、删除 `loadDepartments` 字段；本地 `Department` 类型改为 `ReturnType<typeof useDeptTree>['data']` 类型透传，与 hook 返回类型严格一致。`index.tsx` 删除 `loadDepartments` 解构与两处调用点（openModal / openQuickCreateModal）。无转换函数（纯 list 数据替换），无高风险双向语义。

## What Was Built

### Task 1 — devices 迁移 useDeviceData 去 fetch 改 useDeptTree (commit 669b25b)

**文件**: `xingran-react-frontend/src/pages/network/devices/hooks/useDeviceData.ts` (+5/-19 行) 与 `pages/network/devices/index.tsx` (-3 行)

**useDeviceData.ts 改造**:

1. **顶部 import 调整**：
   - 删除 `import type { Department, AuthCredential, BaseResponse, PageResponse } from "@/types"` 中的 `Department`
   - 新增 `import { useDeptTree } from "@/hooks/useDeptTree"`
   - `post` / `handleApiError` 保留（loadStatistics/loadCredentials 仍依赖）

2. **hook 函数体**：
   - 删除 `const [departments, setDepartments] = useState<Department[]>([])`
   - 新增 `const { data: departments = [] } = useDeptTree();`（与 37-02 useUserData 模式一致）
   - 删除 `loadDepartments` useCallback（line 53-61，含 `post<Department[]>("/system/departments/tree", {})` + `setDepartments` + `handleApiError` 包裹 + `setDepartments([])` 兜底）

3. **UseDeviceDataReturn interface**：
   - `departments: Department[]` 改为 `departments: ReturnType<typeof useDeptTree>["data"]`（**决策**：不直接用 `SimpleDept[]`，与 hook 返回类型严格一致避免漂移）
   - 删除 `loadDepartments: () => Promise<void>` 字段

4. **返回对象**：保留 `departments`，删除 `loadDepartments`

**index.tsx 改造**:

1. **解构**（line 283-290）：删除 `loadDepartments`
2. **openModal**（line 554）：删除 `loadDepartments()` 调用，保留 `loadCredentials()` 调用
3. **openQuickCreateModal**（line 563）：删除 `loadDepartments()` 调用，保留 `loadCredentials()` 调用

**未改动（范围外）**：
- index.tsx 左侧 Sider 的 `<DeptTree />` 组件——37-01 Task 3 已让该组件内部消费 `useDeptTree`，无需重复处理
- index.tsx 两处 `<DepartmentTreeSelect departments={departments} />`（快速创建 modal line 867 / 编辑 modal line 932-937）——受控模式，数据仍由父组件通过 props 喂入，行为不变
- `handleApiError` 在 index.tsx 的 import——仍被 handleProbe / handleCreate / handleDelete / handleBatchDelete / handleCollectPorts 等 8 处使用，**不删除**（CLAUDE.md "Scope Constrainment" 原则）

## Verification

### Acceptance Criteria (全部通过)

| Criterion | Expected | Actual | Status |
|-----------|----------|--------|--------|
| `grep -c "/system/departments/tree" useDeviceData.ts` | = 0 | 0 | ✅ |
| `grep -rc "loadDepartments" devices/` 总和 | = 0 | 0 | ✅ |
| `grep -c "useDeptTree" useDeviceData.ts` | ≥ 1 | 4 (import + type + call + comment) | ✅ |
| `grep -rc "/system/departments/tree" devices/` 递归 | = 0 | 0 | ✅ |
| `npm run type-check` exit code | = 0 | 0 (tsc --noEmit 无输出) | ✅ |

### 行为保持验证（静态分析）

| 维度 | 迁移前 | 迁移后 | 等价 |
|------|--------|--------|------|
| 数据来源 | `post<Department[]>('/system/departments/tree')` | `useDeptTree()` (内部封装 `getDeptTree()` 同端点) | ✅ 同端点同数据 |
| 缓存策略 | 无（每次 openModal/openQuickCreateModal 都重新拉取） | React Query 共享 `['dept', 'tree']` 缓存（5min stale, 30min gc） | ✅ 性能提升, 数据一致 |
| 数据形状 | `Department[]` (`@/types` 定义) | `SimpleDept[]` (`@/lib/dutyApi` canonical, useDeptTree 返回类型) | ✅ 字段 id/deptName/parentId/children 兼容 |
| openModal 行为 | 显示 modal + loadDepartments + loadCredentials | 显示 modal + loadCredentials（departments 由 hook 顶层提供） | ✅ 等价且无延迟（数据已在缓存） |
| openQuickCreateModal 行为 | resetFields + setProbeResult + setVisible + loadDepartments + loadCredentials | resetFields + setProbeResult + setVisible + loadCredentials | ✅ 等价 |
| DepartmentTreeSelect 受控模式 | `departments={departments}` 外部喂入 | 同（departments 来自 useDeptTree 仍外部喂入） | ✅ 完全相同 |
| 左侧 DeptTree 组件 | 内部已消费 useDeptTree (37-01 Task 3) | 未改动 | ✅ 不受影响 |
| 错误处理 | `try/catch + handleApiError("加载部门列表")` | React Query 内部管理 error 状态 | ✅ 迁移后不再弹 toast (与 37-01/37-02 决策一致) |

### UAT 说明

本批自动化验收已全部通过。手动 UAT 推荐用户在批 4 (duty/pools) 或批 5 完成后统一核对（届时 useDeptTree 消费链覆盖更全面）：
1. devices 页打开 → 左侧 DeptTree 显示与勾选筛选
2. 点击 "手动新增" → "所属部门" DepartmentTreeSelect 显示完整路径（从二级起拼）
3. 点击 "快速创建" → "所属部门" DepartmentTreeSelect 同上

## Deviations from Plan

None - plan executed exactly as written.

PLAN Task 1 action 6 提到 "如果 devices 页还使用了 DeptTree 组件做左侧筛选(由 37-01 Task 3 完成后已经消费 useDeptTree),无需额外处理"——核对 index.tsx line 626 `<DeptTree ...>` 确认属此情形，**未做额外处理**。

PLAN Task 1 action 4 提到 "若本地 Department interface 仅用于 departments state, 改 import SimpleDept"——本批采用 `ReturnType<typeof useDeptTree>['data']` 类型透传（决策见 frontmatter），语义与 `SimpleDept[]` 等价，但更严格（hook 返回类型变化时自动跟随）。

## Known Stubs

无。本批为纯重构，无 placeholder/TODO/mock 数据。

## Threat Flags

无新增 threat surface。threat_model 既有条目覆盖：
- T-37-07 (Information Disclosure, accept): devices 部门筛选数据来源不变（同端点 `/system/departments/tree`）
- T-37-SC (n/a, accept): 0 新依赖

## TDD Gate Compliance

本 plan frontmatter `type: execute`（非 `tdd`），无 plan-level TDD gate 强制要求。改造为纯数据源替换（删除 fetch 调用、改消费 hook），无新业务逻辑需要单元测试覆盖。`npm run type-check` 通过 + grep 自动化验收覆盖全部 acceptance criteria。

## Self-Check: PASSED

### Created/Modified files exist

- ✅ FOUND: `xingran-react-frontend/src/pages/network/devices/hooks/useDeviceData.ts` (modified)
- ✅ FOUND: `xingran-react-frontend/src/pages/network/devices/index.tsx` (modified)
- ✅ FOUND: `.planning/phases/37-dept-select-unify/37-04-SUMMARY.md` (this file)

### Commits exist

- ✅ FOUND: 669b25b (refactor(37-04): devices 迁移到 useDeptTree)

### Plan acceptance criteria honored

- ✅ `grep -c "/system/departments/tree" useDeviceData.ts` = 0
- ✅ `grep -rc "loadDepartments" devices/` 总和 = 0
- ✅ `grep -c "useDeptTree" useDeviceData.ts` ≥ 1 (实际 4)
- ✅ `npm run type-check` 退出码 0
- ✅ 返回对象保留 `departments` 字段（消费方依赖）
- ✅ 删除返回对象 `loadDepartments` 字段
- ✅ AD 模块、role tree-select、dept 管理页排除边界零触碰
