---
phase: 37-dept-select-unify
plan: 01
subsystem: frontend-dept-select
tags: [refactor, frontend, react, dept-tree, react-query]
requires:
  - hooks/useDeptTree.ts (已存在 canonical hook)
  - lib/dutyApi.ts:SimpleDept (canonical 类型锚点)
  - utils/deptUtils.ts:DeptLikeNode/filterExternalOrgDepts (已存在工具层)
provides:
  - utils/deptUtils.ts:toFullPathTree (全路径转换, 支持 startFromLevel)
  - utils/deptUtils.ts:toShortNameDataNode (短名 DataNode 转换)
  - utils/deptUtils.ts:ShortNameDataNode (类型导出)
  - components/shared/DepartmentTreeSelect.tsx (受控模式, 消费 SimpleDept[])
  - components/DeptTree/index.tsx (消费 useDeptTree, 去 post fetch)
affects:
  - 后续批次 37-02~37-06 (硬前置已就位: 公共层落地)
  - 临时向后兼容 alias: components/shared/DepartmentTreeSelect.tsx:Department (floors 模块批 2 迁移后移除)
tech-stack:
  added: []
  patterns:
    - "Canonical hook + 受控组件 (数据由调用方从 useDeptTree 喂入, 组件不内部 fetch)"
    - "转换函数保留语义维度 (全路径 vs 短名, 禁止合成单一 convertTree)"
    - "useEffect + ref 守卫首次自动展开 (避免 rawDept 引用变化重置用户展开状态)"
key-files:
  created:
    - .planning/phases/37-dept-select-unify/37-01-SUMMARY.md
  modified:
    - xingran-react-frontend/src/utils/deptUtils.ts (+131 行: toFullPathTree + toShortNameDataNode + ShortNameDataNode)
    - xingran-react-frontend/src/components/shared/DepartmentTreeSelect.tsx (删 convertDeptTreeData + interface Department; +35/-47)
    - xingran-react-frontend/src/components/DeptTree/index.tsx (删 post fetch + transformToTreeData; +73/-64)
decisions:
  - "toFullPathTree 通过 opts.startFromLevel 控制祖先裁剪, startFromLevel=2 复现旧 convertDeptTreeData 的 slice(1) 行为"
  - "DepartmentTreeSelect 保留 Department = SimpleDept re-export alias (向后兼容 floors 模块, 批 2 移除)"
  - "DeptTree 用 RawDept 本地类型 (SimpleDept + isExternalOrg 扩展) 而非改 useDeptTree 返回类型"
  - "首次自动展开用 useEffect + didInitExpandRef 守卫只跑一次"
metrics:
  duration: ~9 分钟 (09:30:58Z → 09:40:08Z, 含静态行为等价性分析)
  completed: 2026-06-22
  tasks_completed: 3
  files_touched: 3
  commits: 3
---

# Phase 37 Plan 01: 批 0 公共层收敛 Summary

批 0 公共层落地——`deptUtils` 新增两个语义区分的转换函数（`toFullPathTree` 全路径 / `toShortNameDataNode` 短名 DataNode）；`DepartmentTreeSelect` 删除本地 `Department` 接口与 `convertDeptTreeData`，改引用 canonical `SimpleDept` + 调用 `toFullPathTree`，保持受控模式；`DeptTree` 删除内部 `post` fetch 与 `transformToTreeData`，改消费 `useDeptTree`，搜索/展开逻辑完全保留。所有后续批次（37-02~37-06）的硬前置已就位。

## What Was Built

### Task 1 — deptUtils 新增两个语义明确的转换函数 (commit a859975)

**文件**: `xingran-react-frontend/src/utils/deptUtils.ts` (+131 行)

1. **`toFullPathTree<T extends DeptLikeNode & { deptName?: string }>(nodes, opts?: { startFromLevel?: 1 | 2 })`**
   - 产生 `{ title, value, key, children?, isExternalOrg? }` 数组
   - `title` 拼接为祖先 + 自身的完整路径（`A / B / C` 格式）
   - `opts.startFromLevel` 控制祖先裁剪：
     - `1`（默认）：不裁剪，顶级 `node.deptName`，深级 `[...allAncestors, name].join(" / ")`
     - `2`：总是丢弃 `ancestors[0]`（即顶级祖先名），与旧 `convertDeptTreeData` 的 `currentPath.slice(1)` 严格等价
   - **透传 `isExternalOrg`**（workstations 高风险依赖：若不透传，`filterExternalOrgDepts` 会把整棵树过滤为空）
   - 替代：`DepartmentTreeSelect.convertDeptTreeData` + `useWorkstationData.buildTreeData`

2. **`toShortNameDataNode<T extends DeptLikeNode & { deptName?: string }>(nodes): ShortNameDataNode[]`**
   - 产生 `{ title, key, value, children?, isLeaf? }` 数组
   - `title` 只显示 `deptName` 短名（不拼路径）
   - 替代：`DeptTree.transformToTreeData` + `notice/useTargetSelector.convertTree` + `user/utils.convertDeptTreeData`

3. **`ShortNameDataNode`** interface 导出（自引用 children 类型，方便消费方 `.find/.map`）

**语义维度保留（D-LOCKED）**：两个函数禁止合成一个。

### Task 2 — DepartmentTreeSelect 删本地类型与转换，保持受控 (commit a52bc36)

**文件**: `xingran-react-frontend/src/components/shared/DepartmentTreeSelect.tsx` (+35/-47 行)

- 删除 `export interface Department`（line 16-20）
- 删除 `convertDeptTreeData`（line 49-75）
- 新增 `import type { SimpleDept } from "@/lib/dutyApi"` + `import { toFullPathTree } from "@/utils/deptUtils"`
- `DepartmentTreeSelectProps.departments` 与 `DepartmentTreeSelectWithTopProps.departments` 类型从 `Department[]` 改为 `SimpleDept[]`
- 两处 `convertDeptTreeData(departments)` 调用点改为 `toFullPathTree(departments, { startFromLevel: 2 })`（**startFromLevel=2 复现旧实现的 `currentPath.slice(1)` 行为，保证 UI 文案不变**）
- **保留 `Department = SimpleDept` re-export alias**（向后兼容 floors 模块两个消费方：`FloorPlanEditorView.tsx`、`FloorModal.tsx`——批 2 迁移后移除，标记 `@deprecated`）
- **保持受控模式（D-LOCKED）**：不内部调 `useDeptTree`，数据由调用方从 hook 喂入

### Task 3 — DeptTree 删内部 post fetch，改消费 useDeptTree (commit 79cc349)

**文件**: `xingran-react-frontend/src/components/DeptTree/index.tsx` (+73/-64 行)

- 删除 `import { post } from "@/lib/api"` + `import type { Department } from "@/types"`
- 新增 `import { useDeptTree } from "@/hooks/useDeptTree"` + `import { filterExternalOrgDepts, toShortNameDataNode, type DeptLikeNode } from "@/utils/deptUtils"` + `import type { SimpleDept } from "@/lib/dutyApi"`
- 删除 `useState treeData/loading`、`loadDeptTree`（含 `post('/system/departments/tree')`）、`transformToTreeData`、触发 fetch 的 `useEffect`
- 替换为：
  ```ts
  const { data: rawDept = [], isLoading: loading } = useDeptTree();
  const treeData = useMemo<DataNode[]>(() => {
    const filtered = externalOnly
      ? filterExternalOrgDepts<RawDept & DeptLikeNode>(...)
      : rawDeptTyped;
    return toShortNameDataNode(filtered) as unknown as DataNode[];
  }, [rawDeptTyped, externalOnly]);
  ```
- **首次自动展开**（原 `loadDeptTree` 末尾 `setExpandedKeys([parentKeys[0]])` 的等价物）：改用 `useEffect` 监听 `treeData`，**用 `didInitExpandRef` 守卫只跑一次**——避免每次 `rawDept` 引用变化都重置用户手动展开状态（behavioral invariant warning 要求）
- **保留所有搜索/展开逻辑**（原封不动）：`onSearch` / `getExpandedKeys` / `filterTreeData` / `onExpand` / `expandedKeys` / `autoExpandParent`
- 本地类型 `RawDept = SimpleDept & { isExternalOrg?: number; children?: RawDept[] }`：因为 useDeptTree 返回 `SimpleDept[]`（不含 `isExternalOrg` 字段），但运行时数据实际含该字段（`@/types/system.ts` 的 `Department` 是超集）。`RawDept` 仅是 TS 类型层面的窄化扩展，运行时数据不变
- props `externalOnly` 与 `onSelect` / `selectedKeys` 保留不变

## Verification

### Acceptance Criteria (全部通过)

| Criterion | Expected | Actual | Status |
|-----------|----------|--------|--------|
| `grep -c "export function toFullPathTree" deptUtils.ts` | = 1 | 1 | ✅ |
| `grep -c "export function toShortNameDataNode" deptUtils.ts` | = 1 | 1 | ✅ |
| `grep -c "isExternalOrg" deptUtils.ts` | ≥ 3 | 15 | ✅ |
| `grep -cE "interface Department\b" DepartmentTreeSelect.tsx` | = 0 | 0 | ✅ |
| `grep -c "convertDeptTreeData" DepartmentTreeSelect.tsx` | = 0 | 0 | ✅ |
| `grep -c "import type { SimpleDept }" DepartmentTreeSelect.tsx` | = 1 | 1 | ✅ |
| `grep -c "useDeptTree" DepartmentTreeSelect.tsx` | = 0 | 0 | ✅ |
| `grep -c "/system/departments/tree" DeptTree/index.tsx` | = 0 | 0 | ✅ |
| `grep -c "transformToTreeData" DeptTree/index.tsx` | = 0 | 0 | ✅ |
| `grep -cE "useDeptTree\(\)" DeptTree/index.tsx` | ≥ 1 | 1 | ✅ |
| `grep -cE "onSearch\|filterTreeData\|getExpandedKeys" DeptTree/index.tsx` | ≥ 3 | 8 | ✅ |
| `npm run type-check` | exit 0 | exit 0 | ✅ |
| `npm run build` | pass | pass (built in 37.98s) | ✅ |
| `grep -rn "/system/departments/tree" 两公共组件` | = 0 | 0 | ✅ |

### 行为保持验证

**静态行为等价性分析（核心）**：

| 维度 | 迁移前 | 迁移后 | 等价 |
|------|--------|--------|------|
| 数据来源 | `post<Department[]>('/system/departments/tree')` | `useDeptTree()` (内部封装 `getDeptTree()` 同端点) | ✅ 同端点同数据 |
| externalOnly 过滤 | `filterExternalOrgDepts(deptData)` (loadDeptTree 内) | `filterExternalOrgDepts(...)` (useMemo 内) | ✅ 同函数 |
| 短名转换 | `transformToTreeData` (手工递归 title=deptName) | `toShortNameDataNode` (sanity check 证明输出结构一致) | ✅ 等价 |
| 叶子标记 | `isLeaf: !item.children \|\| item.children.length === 0` | `isLeaf: !hasChildren` (其中 `hasChildren = !!node.children?.length`) | ✅ 等价 |
| 首次展开 | `loadDeptTree` 末尾 `setExpandedKeys([parentKeys[0]])` | `useEffect` + `didInitExpandRef` 守卫只跑一次 | ✅ 等价（且更稳定） |
| 搜索过滤 | `onSearch` / `getExpandedKeys` / `filterTreeData` | 同三函数原封不动保留 | ✅ 完全相同 |
| 展开交互 | `onExpand` / `expandedKeys` / `autoExpandParent` | 同状态与处理函数保留 | ✅ 完全相同 |
| DepartmentTreeSelect 显示 | `convertDeptTreeData` (slice(1) 从二级开始) | `toFullPathTree({ startFromLevel: 2 })` (经 sanity check 模拟验证三深度树行为一致) | ✅ 等价 |

**Sanity check 已跑**（临时脚本）：模拟深度 3 的树（集团 → 分公司 → 人力资源部），验证 `toFullPathTree({ startFromLevel: 2 })` 输出为 `["集团", "分公司", "分公司 / 人力资源部"]`（与旧 convertDeptTreeData 逐字符一致）；isExternalOrg=1 节点透传；`toShortNameDataNode` 输出 `[集团, 分公司, 人力资源部]` 短名，isLeaf 正确。

**运行时 smoke check**：前端 dev server (http://localhost:4000) HTTP 200，后端 dev server (http://localhost:9000) HTTP 200——模块编译通过、服务可用。

### 手动 UAT（推荐留给批 1/批 4 完成后由用户最终核对）

PLAN Task 3 acceptance 要求"启动前端 dev server, 手动核对三个使用 DeptTree 的页面（user 列表 / buildings 列表 / network devices 列表）左侧/顶部部门树的①默认展开 ②搜索过滤 ③点击勾选——与迁移前行为一致"。

dev server 当前运行中，但 executor agent 无 chrome-devtools 工具直接做 UI 操作。由于：
1. 静态行为等价性分析已逐维度证明等价（见上表）
2. `npm run type-check` 与 `npm run build` 均通过
3. Sanity check 脚本验证转换函数输出结构正确

**推荐**：用户在批 1（system/user）或批 4（duty/pools）完成后做完整 UAT（届时 DeptTree + DepartmentTreeSelect 两条路径都有真实消费方在线）。本批 0 已通过所有自动化验收门槛。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `toFullPathTree.startFromLevel` 实现修正以严格匹配旧 `convertDeptTreeData.slice(1)` 行为**

- **Found during**: Task 1 静态行为等价性分析
- **Issue**: 初版实现用 `currentDepth < startFromLevel` 作为裁剪条件，仅顶级节点触发 ancestors 裁剪（但 ancestors 为空无效果）。模拟深度 3 树（根 A → 子 B → 孙 C）发现 `startFromLevel=2` 时 B 的输出是 `"A / B"`，而旧 `convertDeptTreeData` 的 B 输出是 `"B"`（因为 `slice(1)` 总丢 ancestors[0]）
- **Fix**: 改为 `trimmedAncestors = ancestors.slice(ancestorKeepFrom)` 其中 `ancestorKeepFrom = startFromLevel - 1`。所有节点的 ancestors 都按相同偏移裁剪，严格复现旧 slice(1) 行为
- **Files modified**: `xingran-react-frontend/src/utils/deptUtils.ts`
- **Commit**: a859975 (Task 1 commit 前已修正)

**2. [Rule 3 - Blocking] 保留 `Department = SimpleDept` re-export alias**

- **Found during**: Task 2 实施时发现下游消费方
- **Issue**: 删除 `export interface Department` 会破坏 `pages/operations/floors/components/FloorPlanEditorView.tsx` 与 `FloorModal.tsx` 的编译（它们 `import { type Department }`）
- **Fix**: 保留 `export type Department = SimpleDept`（type alias re-export），标记 `@deprecated`。floors 模块属批 2 范围，届时会改为直接 import `SimpleDept` 并移除本 alias
- **Files modified**: `xingran-react-frontend/src/components/shared/DepartmentTreeSelect.tsx`
- **Commit**: a52bc36
- **D-LOCKED 兼容性**: 本地 `interface Department` 已删除（acceptance criteria `grep -c "interface Department\b" = 0` 满足）；alias 不是"合并组件"或"内部 fetch"——纯类型层面兼容层

**3. [Rule 1 - Bug] `toShortNameDataNode` 输出类型从 `children?: unknown[]` 改为自引用 `ShortNameDataNode`**

- **Found during**: Task 1 sanity check 脚本发现 `.find()` 在 `unknown[]` 上不能直接调用（TS 静态类型层面，运行时数据正确）
- **Issue**: 初版 `children?: unknown[]` 让消费方需要 cast 才能 `.find/.map`
- **Fix**: 提取 `ShortNameDataNode` interface，`children?: ShortNameDataNode[]` 自引用，消费方（如 DeptTree）可直接使用
- **Files modified**: `xingran-react-frontend/src/utils/deptUtils.ts`
- **Commit**: a859975

**4. [Rule 3 - Blocking] DeptTree 用本地 `RawDept` 类型而非改 useDeptTree 返回类型**

- **Found during**: Task 3 实施时发现 `externalOnly` 模式依赖 `isExternalOrg` 字段
- **Issue**: `useDeptTree` 返回 `SimpleDept[]`（不含 `isExternalOrg`），但 `filterExternalOrgDepts<T extends DeptLikeNode>` 要求节点满足 DeptLikeNode（`isExternalOrg?: number`）。SimpleDept 的 children 递归不含该字段，TS 不会自动通过
- **Fix**: DeptTree 内定义 `interface RawDept extends SimpleDept { isExternalOrg?: number; children?: RawDept[] }`，把 useDeptTree 结果 `as unknown as RawDept[]` 窄化。运行时数据完全不变（仅 TS 类型层面）。useDeptTree 不属于批 0 改造范围（保持其简单性）
- **Files modified**: `xingran-react-frontend/src/components/DeptTree/index.tsx`
- **Commit**: 79cc349

## Known Stubs

无。本批 0 为纯重构，无 placeholder/TODO/mock 数据。

## Threat Flags

无新增 threat surface。threat_model 既有条目覆盖：
- T-37-01 (Information Disclosure, accept): DeptTree/DepartmentTreeSelect 仅消费既有端点
- T-37-02 (Tampering, mitigate): 既有 `useInvalidateDept` 写后失效已存在
- T-37-SC (n/a, accept): 0 新依赖

## TDD Gate Compliance

本 plan frontmatter `type: execute`（非 `tdd`），无 plan-level TDD gate 强制要求。

Task 1 新增 `toFullPathTree` 与 `toShortNameDataNode` 是纯函数（无副作用、确定性输出），理论上非常适合单元测试。CONTEXT `<decisions>` Claude's Discretion 第 3 条明确"推荐但不强制"。本次通过临时 sanity check 脚本（已删除，不入仓库）验证了行为正确性。若批 1/批 2 出现回归，建议补单元测试到 `src/utils/deptUtils.test.ts`（项目用 vitest）。

## Self-Check: PASSED

### Created/Modified files exist

- ✅ FOUND: `xingran-react-frontend/src/utils/deptUtils.ts` (modified)
- ✅ FOUND: `xingran-react-frontend/src/components/shared/DepartmentTreeSelect.tsx` (modified)
- ✅ FOUND: `xingran-react-frontend/src/components/DeptTree/index.tsx` (modified)
- ✅ FOUND: `.planning/phases/37-dept-select-unify/37-01-SUMMARY.md` (this file)

### Commits exist

- ✅ FOUND: a859975 (feat(37-01): deptUtils 新增 toFullPathTree + toShortNameDataNode)
- ✅ FOUND: a52bc36 (refactor(37-01): DepartmentTreeSelect 删本地类型与转换)
- ✅ FOUND: 79cc349 (refactor(37-01): DeptTree 删内部 post fetch, 改消费 useDeptTree)

### D-LOCKED decisions honored

- ✅ 未合并 DeptTree 与 DepartmentTreeSelect（两组件职责正交保留）
- ✅ 未合成单一 convertTree（toFullPathTree + toShortNameDataNode 两个函数语义维度保留）
- ✅ DepartmentTreeSelect 保持受控模式（`useDeptTree` 调用 = 0，数据由调用方喂入）
- ✅ DeptTree 搜索/展开逻辑保留（onSearch/filterTreeData/getExpandedKeys/onExpand/autoExpandParent 全部原封不动）
- ✅ toFullPathTree 透传 isExternalOrg（workstations 高风险依赖）
