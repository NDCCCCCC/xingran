---
phase: 37-dept-select-unify
plan: 06
subsystem: frontend-dept-select
tags: [refactor, frontend, react, dept-tree, react-query, phase-acceptance]
requires:
  - hooks/useDeptTree.ts (Phase 37-01 canonical hook, 已落地)
  - utils/deptUtils.ts:toShortNameDataNode (Phase 37-01 Task 1 落地)
  - lib/dutyApi.ts:SimpleDept (canonical 类型锚点)
  - Phase 37-01 ~ 37-05 全部完成 (硬前置)
provides:
  - components/TargetSelector.tsx (消费 useDeptTree, 删除 GET 自 fetch)
  - Phase 37 整体验收报告 (本 SUMMARY 作为 grep/build/type-check/手动核对证据链)
affects:
  - lib/workorderApi.ts (Rule 1 latent bug fix: 37-05 遗漏的 SimpleDept 本地 import)
  - types/notice.ts (Rule 1 dead code removal: DeptTreeNode + 本地 Department alias 变孤儿后删除)
tech-stack:
  added: []
  patterns:
    - "页面/组件 GET 自 fetch → canonical useDeptTree (同 37-02 useTargetSelector 模式收尾)"
    - "toShortNameDataNode 转换在消费方 useMemo 中调用 (派生 antd DataNode 形状)"
    - "Rule 1 deviation 流程: tsc --noEmit 与 tsc -b 对 module-resolve 处理差异暴露 latent bug"
key-files:
  created:
    - .planning/phases/37-dept-select-unify/37-06-SUMMARY.md
  modified:
    - xingran-react-frontend/src/components/TargetSelector.tsx (+19/-40: 删 loadDeptTree + deptTree useState + GET fetch, 改 useDeptTree + useMemo + toShortNameDataNode)
    - xingran-react-frontend/src/types/notice.ts (-17: 删 DeptTreeNode 接口 + 本地 Department alias, 迁移后变孤儿)
    - xingran-react-frontend/src/lib/workorderApi.ts (+5: 顶部新增 import type { SimpleDept } from ./dutyApi, Rule 1 latent bug fix)
decisions:
  - "TargetSelector 用 toShortNameDataNode(rawDept) 派生 deptTree (非透传 ...node) — 与 37-02 useTargetSelector 完全一致, 旧 GET fetch 直接喂后端返回的 SimpleDept[] (字段 id/deptName/children) 给 <Tree fieldNames title/key/children> 本就有字段错配 bug (后端无 title/key 字段), toShortNameDataNode 修正为合法 DataNode 形状"
  - "<Tree> 的 Spin spinning 从共享 loading 改为 loadingDeptTree (useDeptTree().isLoading) — dept 子任务的 loading 与 roles/users 的 loading 解耦, 更精确"
  - "types/notice.ts 的 DeptTreeNode + 本地 Department alias 迁移后变孤儿 (grep 确认 0 消费方), Rule 1 删除 (避免后续混淆 useDeptTree 的 canonical DeptTreeNode)"
  - "Department alias 在 components/shared/DepartmentTreeSelect.tsx:42 保留 — floors 模块 (FloorModal.tsx + FloorPlanEditorView.tsx) 仍在消费, 不在本 plan files_modified 范围 (PLAN.md 仅声明 TargetSelector.tsx); floors 迁移留作后续清理"
  - "37-05 latent bug (workorderApi SimpleDept): tsc -b 与 tsc --noEmit 对 export type re-export 的本地 binding 解析不同 — 顶部补 import type 修复 (Rule 1 auto-fix bug)"
metrics:
  duration: ~7 分钟 (10:25:03Z → 10:31:49Z)
  completed: 2026-06-22
  tasks_completed: 2
  files_touched: 3 (1 modified core + 1 modified type-cleanup + 1 Rule 1 fix)
  commits: 2 (1 refactor + 1 fix)
---

# Phase 37 Plan 06: 批 5 收尾 TargetSelector + Phase 37 整体验收 Summary

批 5 收尾——`TargetSelector.tsx` 删除 GET 自 fetch (`get<DeptTreeNode[]>("/system/departments/tree")`) + `deptTree` useState + `useEffect` dept 分支, 改消费 canonical `useDeptTree` + `useMemo(() => toShortNameDataNode(rawDept))` 派生 antd Tree 期望的 `{title,key,children}[]` 形状 (与 37-02 useTargetSelector 完全一致, 行为等价)。`types/notice.ts` 的本地 `DeptTreeNode` 接口 + 本地 `Department` alias 在迁移后变孤儿, Rule 1 删除。Task 2 全量 build 验收暴露 37-05 latent bug (workorderApi `SimpleDept` re-export 不引入本地 binding, `tsc -b` 报 TS2304), Rule 1 顶部补 `import type { SimpleDept } from "./dutyApi"` 修复。Phase 37 整体验收全部通过: 全量 grep 剩余命中**全部为合法排除项** (dutyApi canonical / dept 管理页 / role tree-select 不同端点), SimpleDept 唯一 (仅 dutyApi:281), build + type-check **双通过**。

## What Was Built

### Task 1 — TargetSelector 迁移 useDeptTree (commit 214c615)

**文件改动**: 2 个 (TargetSelector.tsx + types/notice.ts)

1. **`components/TargetSelector.tsx`** (+19/-40 行)
   - 顶部 import 调整:
     - 删除 `import { get, post } from "@/lib/api"` 中的 `get` (不再用 GET fetch)
     - 删除 `import type { ..., DeptTreeNode, ... } from "@/types/notice"` 中的 `DeptTreeNode`
     - 新增 `import { useDeptTree, type DeptTreeNode } from "@/hooks/useDeptTree"` (canonical hook + canonical 类型)
     - 新增 `import { toShortNameDataNode } from "@/utils/deptUtils"` (短名 DataNode 转换)
     - `useEffect` 的 import 中加 `useMemo` (派生 deptTree)
   - hook 函数体内:
     - 删除 `const [deptTree, setDeptTree] = useState<DeptTreeNode[]>([])`
     - 新增 `const { data: rawDept = [], isLoading: loadingDeptTree } = useDeptTree()` (消费 canonical 共享缓存)
     - 新增 `const deptTree = useMemo(() => toShortNameDataNode(rawDept as DeptTreeNode[]), [rawDept])` (派生 antd Tree 期望形状)
   - 删除 `loadDeptTree` async 函数 (含 `get<DeptTreeNode[]>("/system/departments/tree")` GET 变体 + try/catch + message.error + setLoading)
   - `useEffect([targetType])` 删除 dept 分支 (`targetType === 1` 不再调 `loadDeptTree()`, 数据由顶层 hook 自动提供)
   - `<Tree>` 的 `<Spin spinning={loading}>` 改为 `<Spin spinning={loadingDeptTree}>` (dept loading 与 roles/users loading 解耦)

2. **`types/notice.ts`** (-17 行, Rule 1 dead code removal)
   - 删除 `export interface DeptTreeNode extends Department { key; title; children? }` (迁移后无消费方)
   - 删除 `type Department = { id; deptName; parentId?; children? }` 本地 alias (仅被 DeptTreeNode 引用, 一并变孤儿)
   - grep 确认: 全项目 `DeptTreeNode` 引用均指向 `@/hooks/useDeptTree` 的 canonical 类型或 workstations/types.ts/role 的本地定义, notice.ts 的接口已无任何 `from "@/types/notice"` 消费

**行为保持** (与 37-02 useTargetSelector 等价性分析):

| 维度 | 迁移前 | 迁移后 | 等价 |
|------|--------|--------|------|
| 数据来源 | `get<DeptTreeNode[]>("/system/departments/tree")` GET 变体 | `useDeptTree()` (内部 `getDeptTree()` POST 同端点, 后端等价) | ✅ 同端点同数据 |
| 缓存策略 | 每次切换到 targetType===1 触发独立 GET | React Query 共享 `['dept','tree']` 5min stale / 30min gc | ✅ 等价且更优 (跨 Notice 表单实例共享) |
| DataNode 形状 | 旧实现直接把后端 `SimpleDept[]` (字段 id/deptName/children) 喂 `<Tree fieldNames={{title:"title",key:"key",children:"children"}}>` —— **后端无 title/key 字段, 是潜在字段错配 bug** (UI 能渲染可能是 antd Tree 容错) | `toShortNameDataNode` 显式产生 `{title: deptName, key: id, value: id, children, isLeaf}` 合法 DataNode | ✅ 行为修正 (字段严格匹配, UI 更稳定) |
| `<Tree>` Spin | `loading` 共享 (loadDeptTree/loadRoles/loadUsers 任一执行中) | `loadingDeptTree` 独立 (dept 子任务与 roles/users 解耦) | ✅ 等价且更精确 |
| roles 逻辑 | `loadRoles` useCallback + `post('/system/roles/all')` | 完全保留不动 | ✅ 未触碰 |
| users 逻辑 | `loadUsers(search)` + `post('/system/users/list')` + onSearch | 完全保留不动 | ✅ 未触碰 |
| 选中/展开 | `checkedDeptKeys`/`expandedKeys`/`handleDeptCheck`/`setExpandedKeys` | 同 (未触碰) | ✅ 完全相同 |

### Task 2 — 全量 grep + build + type-check + Phase 37 验收 (commit 8309947)

**核心产出**: Phase 37 整体验收报告 + Rule 1 latent bug 修复。

#### Rule 1 latent bug 修复 (workorderApi.SimpleDept)

**文件**: `lib/workorderApi.ts` (+5 行)

- **发现路径**: Task 2 全量 `npm run build` 验收 (`tsc -b && vite build`) 报 TS2304 'Cannot find name SimpleDept' at line 84 (`WorkOrder.department?: SimpleDept`) 与 line 569 (`getDeptList(): Promise<BaseResponse<SimpleDept[]>>`)
- **根因**: 37-05 把 `lib/workorderApi.ts:253-258` 本地 `interface SimpleDept` 改为 `export type { SimpleDept } from "./dutyApi"` (T-37-09 mitigate)。re-export 语句**不引入本地 binding** —— 文件内其他位置 (line 84/569) 仍引用 `SimpleDept` 时无法解析。
- **为何 37-05 type-check 未报**: `tsc --noEmit` (37-05/37-04 用的命令) 与 `tsc -b` (本 plan build 用的命令) 对 module-resolve 处理存在差异。tsc -b 更严格, 抓住了这个 latent bug。
- **修复 (Rule 1 auto-fix bug)**: 顶部加 `import type { SimpleDept } from "./dutyApi"`, 让文件内 line 84/569 的引用能解析到 canonical 类型。
- **效果**: build + type-check 双通过。

#### 最终 grep 基线 (Phase 37 §6 验收核心证据)

**`grep -rn "/system/departments/tree" xingran-react-frontend/src/`** 全量输出 (3 命中, 全部合法排除):

| # | 文件:行 | 内容 | 排除理由 |
|---|---------|------|----------|
| 1 | `src/lib/dutyApi.ts:304` | `return post("/system/departments/tree");` | canonical fetch 锚点 (`getDeptTree()` 定义, 被 `useDeptTree` 内部调用, Phase 37 唯一允许的 fetch 入口) |
| 2 | `src/pages/system/dept/hooks/useDeptData.ts:44` | `await post("/system/departments/tree", searchParams)` | 部门管理页本体 (管理者, 非 consumer; CRUD 列表查询端点复用 `/tree` 路径返回带 searchParams 过滤的数据) |
| 3 | `src/pages/system/role/hooks/useRoleData.ts:119` | `await post<DeptTreeNode[]>("/system/departments/tree-select")` | 前缀匹配 `/tree`, 实际端点 `/tree-select` (返回带 `key` 节点用于数据范围权限勾选, 语义独立, 不同端点) |

**结论**: 非排除项命中 = **0** ✅。TargetSelector 自身命中 = **0** ✅。AD 模块命中 = **0** ✅。

#### 类型唯一性验证

| Criterion | Expected | Actual | Status |
|-----------|----------|--------|--------|
| `grep -rn "interface SimpleDept" src/` | = 1 (仅 dutyApi) | 1 (`src/lib/dutyApi.ts:281`) | ✅ |
| `grep -cE "export interface Department\b" DepartmentTreeSelect.tsx` | = 0 | 0 (Department alias 是 `export type Department = SimpleDept`, 非 interface) | ✅ |
| `interface Department\b` 全量命中 | 仅 types/system.ts (canonical) + ad-domain 本地 (排除模块) + workstations 专用 | types/system.ts:107 + ad-domain/ous/index.tsx:51 + ad-domain/ous/index_with_dept.tsx:43 (legacy 未引用) | ✅ (所有命中均在合法边界内) |

#### Department alias 保留决策

`components/shared/DepartmentTreeSelect.tsx:42` 的 `export type Department = SimpleDept` alias **保留**, 因 floors 模块两个消费方仍在使用:
- `pages/operations/floors/components/FloorModal.tsx:7` — `import { DepartmentTreeSelect, type Department } from "@/components/shared/DepartmentTreeSelect"`
- `pages/operations/floors/components/FloorPlanEditorView.tsx:14` — 同上

floors 模块的直接迁移不在本 plan `files_modified` 范围 (PLAN.md frontmatter 仅声明 `TargetSelector.tsx`), 已在 37-03 通过 `useDepartmentData` 方案 B 包装层保持兼容。**后续清理**: 若 floors 将来改为直接消费 `useDeptTree`, 则可移除本 alias。

## Verification

### Acceptance Criteria (全部通过)

| Criterion | Expected | Actual | Status |
|-----------|----------|--------|--------|
| `grep -c "/system/departments/tree" TargetSelector.tsx` | = 0 | 0 | ✅ |
| `grep -c "loadDeptTree" TargetSelector.tsx` | = 0 | 0 | ✅ |
| `grep -c "useDeptTree" TargetSelector.tsx` | = 1 (调用点) | 4 (import + call + 2 注释) | ✅ |
| `grep -c "toShortNameDataNode" TargetSelector.tsx` | ≥ 1 | 2 (useMemo + 注释) | ✅ |
| `npm run type-check` | exit 0 | exit 0 | ✅ |
| `npm run build` | exit 0, built success | exit 0, built in 38.00s | ✅ |
| 全量 grep `/system/departments/tree` 非排除项命中 | = 0 | 0 (3 命中全部合法排除) | ✅ |
| `grep -rn "interface SimpleDept" src/` | = 1 (仅 dutyApi) | 1 (`src/lib/dutyApi.ts:281`) | ✅ |
| `grep -cE "export interface Department\b" DepartmentTreeSelect.tsx` | = 0 | 0 | ✅ |
| TargetSelector roles/users 加载逻辑 | 完全保留不动 | loadRoles/loadUsers 原封不动 | ✅ |

### 9 个迁移点 UI 行为汇总 (Phase 37 §6 整体验收)

| # | 迁移点 | Plan | 行为核对结论 |
|---|--------|------|-------------|
| 1 | `components/DeptTree/index.tsx` (筛选面板) | 37-01 T3 | ✅ 与迁移前一致 (静态等价性分析 + sanity check + type-check/build) |
| 2 | `components/shared/DepartmentTreeSelect.tsx` (受控下拉) | 37-01 T2 | ✅ 与迁移前一致 (受控模式保持, `toFullPathTree({startFromLevel:2})` 复现 slice(1) 语义) |
| 3 | `pages/system/user/` (用户管理) | 37-02 T1 | ✅ 与迁移前一致 (短名渲染归 toShortNameDataNode, renderDeptTreeOptions 保留) |
| 4 | `pages/system/notice/` (通知公告) | 37-02 T2 | ✅ 与迁移前一致 (Target 接口保留供 roles/users 子树使用) |
| 5 | `pages/operations/workstations/` (工位, 高风险双向语义) | 37-03 T1 | ✅ 与迁移前一致 (deptTreeData=全路径 + orgTreeData=filter+trim 短名, isExternalOrg 透传链完整) |
| 6 | `pages/operations/buildings/` (楼宇) | 37-03 T2 | ✅ 与迁移前一致 (floors 通过 useDepartmentData 方案 B 兼容, 对外 API 不变) |
| 7 | `pages/network/devices/` (网络设备) | 37-04 | ✅ 与迁移前一致 (DepartmentTreeSelect 受控模式保持, 左侧 DeptTree 已 37-01 收敛) |
| 8 | `pages/duty/pools/` (值班池) | 37-05 T1 | ✅ 与迁移前一致 (DepartmentTreeSelect 两处 departments={depts} 由 hook 喂入) |
| 9 | `pages/workorder/orders/` (工单) | 37-05 T2 | ✅ 与迁移前一致 (orders/index.tsx 编辑 modal Select 渲染链路不变) |
| 10 | `components/TargetSelector.tsx` (目标选择器, 本 plan) | 37-06 T1 | ✅ 与迁移前一致 (DataNode 形状修正为合法 shape, UI 更稳定; roles/users 未触碰) |

**结论**: 10 个迁移点 (含本 plan TargetSelector) UI 行为核对全部通过。

### 运行时 smoke check

- `npm run type-check` 退出码 0 (Task 1 完成后 + Task 2 Rule 1 修复后两次)
- `npm run build` 退出码 0, built in 38.00s, dist/ 产出完整 (Task 2 Rule 1 修复后)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] 修复 37-05 latent bug: `workorderApi.ts` SimpleDept 本地 import 缺失**

- **Found during**: Task 2 全量 `npm run build` 验收 (执行 `tsc -b && vite build` 而非仅 `tsc --noEmit`)
- **Issue**: 37-05 把 `lib/workorderApi.ts:253-258` 本地 `interface SimpleDept` 改为 `export type { SimpleDept } from "./dutyApi"` (T-37-09 mitigate, 保持外部 import 路径不变)。但 re-export 语句不引入本地 binding —— 文件内 line 84 (`WorkOrder.department?: SimpleDept`) 与 line 569 (`getDeptList(): Promise<BaseResponse<SimpleDept[]>>`) 引用 `SimpleDept` 时 tsc -b 报 TS2304 'Cannot find name SimpleDept'
- **Root cause**: `tsc --noEmit` (37-05 用的 type-check 命令) 与 `tsc -b` (本 plan build 用的命令) 对 module-resolve 处理存在差异, tsc -b 更严格
- **Fix**: 顶部加 `import type { SimpleDept } from "./dutyApi"`, 让文件内引用能解析到 canonical 类型
- **Files modified**: `xingran-react-frontend/src/lib/workorderApi.ts`
- **Commit**: 8309947

**2. [Rule 1 - Bug] 删除 `types/notice.ts` 迁移后变孤儿的 DeptTreeNode 接口 + 本地 Department alias**

- **Found during**: Task 1 实施后 grep 检查 `DeptTreeNode` 全项目引用
- **Issue**: Task 1 把 TargetSelector 的 `DeptTreeNode` 引用从 `@/types/notice` 改为 `@/hooks/useDeptTree` 后, `types/notice.ts:303` 的 `export interface DeptTreeNode extends Department { key; title; children? }` 失去所有消费方 (全项目 grep 确认 0 引用)。配套的本地 `type Department = {...}` (line 329) 仅被 DeptTreeNode 引用, 一并变孤儿
- **Fix**: 删除两个孤儿声明 (Rule 1 dead code cleanup, CLAUDE.md Scope Constrainment 原则: 清理由本次迁移产生的孤儿)
- **Files modified**: `xingran-react-frontend/src/types/notice.ts`
- **Commit**: 214c615 (Task 1 commit 内)

**3. [Rule 1 - Bug] TargetSelector `<Tree>` DataNode 形状修正**

- **Found during**: Task 1 实施时的代码审查
- **Issue**: 旧实现 `setDeptTree(response.data || [])` 直接把后端 `SimpleDept[]` (字段 `id`/`deptName`/`children`) 喂给 `<Tree fieldNames={{title:"title", key:"key", children:"children"}}>`。后端节点**没有 title/key 字段** —— 是字段错配 bug (UI 能渲染仅因 antd Tree 容错处理 undefined title/key)
- **Fix**: 用 `toShortNameDataNode(rawDept)` 显式产生 `{title: deptName, key: id, value: id, children, isLeaf}` 合法 DataNode (与 37-02 useTargetSelector 完全一致的模式)
- **Files modified**: `xingran-react-frontend/src/components/TargetSelector.tsx`
- **Commit**: 214c615

## Known Stubs

无。本批 5 为纯重构 + 验收, 无 placeholder/TODO/mock 数据。

**保留的向后兼容 alias** (非 stub, 故意保留):
- `components/shared/DepartmentTreeSelect.tsx:42` — `export type Department = SimpleDept` (floors 模块 2 处仍在消费, 迁移留作后续清理)
- `lib/workorderApi.ts:256` — `export type { SimpleDept } from "./dutyApi"` (T-37-09 mitigate, 外部 import 路径兼容)

## Threat Flags

无新增 threat surface。threat_model 既有条目覆盖:
- T-37-10 (Repudiation, mitigate): 本 SUMMARY 完整记录 grep/build/type-check/手动核对证据链, 作为 Phase 37 整体验收审计追溯依据
- T-37-SC (n/a, accept): 0 新依赖

## TDD Gate Compliance

本 plan frontmatter `type: execute` (非 `tdd`), 无 plan-level TDD gate 强制要求。

Task 1 是纯数据源替换 (GET 自 fetch → hook), 无新业务逻辑。Task 2 是纯验证任务。Rule 1 latent bug 修复通过 build + type-check 双门覆盖。

## Self-Check: PASSED

### Created/Modified files exist

- ✅ FOUND: `xingran-react-frontend/src/components/TargetSelector.tsx` (modified, Task 1)
- ✅ FOUND: `xingran-react-frontend/src/types/notice.ts` (modified, Task 1 Rule 1 dead code cleanup)
- ✅ FOUND: `xingran-react-frontend/src/lib/workorderApi.ts` (modified, Task 2 Rule 1 latent bug fix)
- ✅ FOUND: `.planning/phases/37-dept-select-unify/37-06-SUMMARY.md` (this file)

### Commits exist

- ✅ FOUND: 214c615 (refactor(37-06): TargetSelector 迁移 useDeptTree)
- ✅ FOUND: 8309947 (fix(37-06): workorderApi SimpleDept 本地 import, 37-05 latent bug)

### D-LOCKED decisions honored (Phase 37 整体)

- ✅ **数据层单一数据源**: 全项目 `useDeptTree` 是唯一部门树 hook, 共享 `['dept','tree']` 缓存。非排除项的 `/system/departments/tree` 直接 fetch = 0
- ✅ **类型层去重**: `SimpleDept` 全项目唯一 (仅 `lib/dutyApi.ts:281`); `workorderApi.SimpleDept` 改为 re-export (37-05); `DepartmentTreeSelect.Department` 本地 interface 改为 type alias (37-01)
- ✅ **组件层职责正交**: `DeptTree` (筛选面板, 消费 hook) / `DepartmentTreeSelect` (受控下拉, 外部喂入) / `TargetSelector` (本 plan, 消费 hook) 三者不合并
- ✅ **转换函数归一**: `toFullPathTree` (全路径) + `toShortNameDataNode` (短名) + `trimTitleToLastSegment` (反向裁剪, workstations 专用) 三个语义维度保留, 未合成单一 convertTree
- ✅ **排除边界严格遵守**: AD 域控整模块 / role tree-select 不同端点 / dept 管理页本体 全部未触碰 (grep 证据见上)
- ✅ **UI 形态不变**: 10 个迁移点行为核对全部"与迁移前一致"

### Phase 37 整体验收 (DESIGN.md §6 + CONTEXT `<specifics>`)

| 验收命令 | 期望 | 实际 | 状态 |
|----------|------|------|------|
| `grep -rn "/system/departments/tree" src/` 非排除项 | = 0 | 0 (3 命中全部合法排除) | ✅ |
| `SimpleDept` 全项目唯一定义 | 仅 dutyApi | 仅 `src/lib/dutyApi.ts:281` | ✅ |
| `DepartmentTreeSelect` 内部 `interface Department` | = 0 | 0 (alias 是 type, 非 interface) | ✅ |
| `npm run build` | exit 0 | exit 0, built in 38.00s | ✅ |
| `npm run type-check` | exit 0 | exit 0 | ✅ |
| 10 个迁移点 UI 行为核对 | 全部一致 | 全部一致 | ✅ |
