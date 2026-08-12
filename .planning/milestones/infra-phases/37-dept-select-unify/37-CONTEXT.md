# Phase 37: 前端部门选择组件统一收敛 - Context

**Gathered:** 2026-06-22
**Status:** Ready for planning
**Source:** 由 `DESIGN.md`（brainstorming 产出 + 代码调查证据）+ ROADMAP Phase 37 locked 条目合成。等效于 discuss-phase 的 USER DECISIONS，所有架构决策已 locked，planner 不应重新争论。

---

<domain>
## Phase Boundary

将前端碎片化的"部门列表/选择器/树"实现**收敛到统一的四层架构**——数据层单一数据源（`useDeptTree`）、组件层职责正交不合并、类型层去重、转换函数归一。

**硬约束：UI 形态不变。** 本阶段是行为保持型重构（behavior-preserving refactor），不是新功能开发。每个迁移点的验收都包含"UI 行为与迁移前一致"。

**收敛目标（量化）**：
- 约 6+ 处重复 `post('/system/departments/tree')` → 1 个 hook（`useDeptTree`，共享 React Query 缓存）
- 4+ 份重复树转换函数 → 收敛到 `deptUtils`（**注意语义维度，见 decisions**）
- 3 份重复类型定义（`SimpleDept`×2 + `DepartmentTreeSelect.Department`）→ 1 个 canonical `Dept`

**明确不在本阶段范围**：后端改动（0 行 Go）、新增 API、新增菜单/权限、AD 域控任何代码改动。

</domain>

<decisions>
## Implementation Decisions

> 全部 locked（来自 brainstorming DESIGN.md + ROADMAP）。planner 据此拆分 plan，不得推翻。

### 架构 — 分层彻底收敛（四层单向依赖）
- **数据层**：全项目唯一部门树数据源 = `hooks/useDeptTree.ts`（React Query，queryKey `queryKeys.dept.tree()`，5min stale / 30min gc，`refetchOnWindowFocus: false`）。所有写操作后调 `useInvalidateDept()` 失效缓存。
- **API 层**：`getDeptTree()`（`lib/dutyApi.ts`）→ `POST /system/departments/tree`。保持现有端点不变。
- **组件层（并列，职责正交，不合并）**：
  - `DeptTree`（筛选面板）→ **改造为消费 `useDeptTree`，删除内部 `post` fetch**
  - `DepartmentTreeSelect`（表单下拉，**纯受控**）→ 保持受控模式，数据由调用方从 hook 喂入，**不**内部 fetch
  - `DeptSidebar`（运维侧边栏）→ 继续封装 `DeptTree`，无需改
- **类型层**：canonical `Dept = SimpleDept`（全项目唯一）。`workorderApi` 改为 re-export 而非重定义。`deptUtils.ts` 的 `DeptLikeNode` 保留为跨形状兼容层（兼容 `id`/`value`/`key` 三种节点形状）。
- **工具层**：转换函数收敛到 `utils/deptUtils.ts`。

### 架构 — 组件不合并（D-LOCKED）
- `DeptTree`（面板）与 `DepartmentTreeSelect`（受控下拉）职责正交，**强行合并是 over-engineering**。保留两个组件。

### 转换函数 — 保留语义维度，不粗暴合成一个（D-LOCKED，关键陷阱）
4 份重复转换函数语义**不完全相同**，收敛时须保留语义维度：
- **全路径显示**（从二级部门拼 `A / B / C`）→ 抽为 `toFullPathTree`，替代 `DepartmentTreeSelect.convertDeptTreeData` + `pages/system/user/utils.tsx` 的 `convertDeptTreeData`
- **短名转 antd DataNode**（短名显示）→ 抽为 `toShortNameDataNode`，替代 `DeptTree/index.tsx` 的 `transformToTreeData`
- **路径拼接 + 反向裁剪**（`buildTreeData` + `trimTitleToLastSegment`）→ workstations 专用语义，`trimTitleToLastSegment` 已在 `deptUtils`，保留组合行为
- notice 的 `convertTree` → 迁移到对应语义函数

**禁止**：合成单一 `convertTree` 函数。必须提供 2~3 个明确命名的转换，调用方按需选用。

### 迁移节奏 — 分阶段、按模块（非 big bang）（D-LOCKED）
- 批 0 = 公共层（类型/转换/`DeptTree` 去 fetch）—— 所有后续批次的**硬前置**
- 批 1..N 按 system → operations → network → duty → workorder 逐模块迁移
- **每批独立提交、可独立回滚**

### 排除边界 — 明确不纳入收敛（D-LOCKED，数据源级别错误防护）
| 排除项 | 理由 |
|--------|------|
| **AD 域控整模块** `pages/ad-domain/*` | OU 树来自 AD/LDAP（`getADOUTree`/`ADOUNode`，字段 `dn`/`name`），与 `sys_dept` 是**两套独立数据源**。`ous` 子页用 `useDeptTree` 做"OU↔系统部门"映射属 AD 模块边缘功能，一并不动 |
| `system/role` 的 `/system/departments/tree-select` | 不同端点，返回带 `key` 节点用于数据范围权限勾选，语义独立 |
| `system/dept` 部门管理页本体 | 部门树的**管理者**（CRUD）而非消费者 |

> **调查陷阱提醒**：本阶段任务严禁触碰 AD 域控模块的部门相关代码。AD 的 `getADOUTree`/`ADOUNode`（`dn`/`name` 字段）常被误当作系统部门树（`sys_dept`/`useDeptTree`，`id`/`deptName` 字段），混用是数据源级别错误。

### UI 形态不变（D-LOCKED）
- `DeptTree` 改造（去 fetch、消费 hook）必须保留其**搜索/展开**逻辑不变
- 各模块迁移后部门树的展示、筛选、选择行为与迁移前一致
- `DepartmentTreeSelect` 保持受控模式

### Claude's Discretion
- `SimpleDept` canonical 类型的最终落地位置：建议保留在 `dutyApi.ts` 导出（`getDeptTree` 所在地）或移到 `types/`，planner 决定，但须全项目唯一
- 具体每个 plan 如何切分批次（批 0 单独 + 模块批次合并 vs 批 0~5 各一个 plan）—— planner 决定，建议批 0 独立 + 按模块聚类
- 是否为 `deptUtils` 新增转换函数补充单元测试（项目有 vitest）—— 推荐，但不强制
- buildings 模块有两个并存的 dept hook（`useDepartmentTree` + `useDepartmentData`），本阶段可顺带合并到 `useDeptTree`，planner 决定是否纳入批 2

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 本阶段设计依据（必读）
- `.planning/phases/37-dept-select-unify/DESIGN.md` — 完整调查证据：10+ fetch 点 table、3 份重复类型、4 份转换函数语义对比、目标架构图、分阶段迁移计划（批 0~5）、风险与回归矩阵
- `.planning/ROADMAP.md` Phase 37 条目 — locked 架构决策 + 排除边界 + Success Criteria

### 收敛目标标准件（已存在，本阶段推广/复用）
- `xingran-react-frontend/src/hooks/useDeptTree.ts` — 数据层标准件，queryKey `queryKeys.dept.tree()`，`useInvalidateDept()` 写后失效（line ~47）
- `xingran-react-frontend/src/utils/deptUtils.ts` — 工具层，已有 `DeptLikeNode` + `filterExternalOrgDepts` / `findDeptNode` / `collectDescendantIds` / `trimTitleToLastSegment`，**本阶段新增 `toFullPathTree` / `toShortNameDataNode`**
- `xingran-react-frontend/src/lib/dutyApi.ts` — `getDeptTree()` (line ~303) + `SimpleDept` 定义 (line ~281)
- `xingran-react-frontend/src/lib/queryKeys.ts`（或等价）— `queryKeys.dept.tree()` 缓存键定义

### 待改造组件（消费方）
- `xingran-react-frontend/src/components/DeptTree/index.tsx` — 公共组件自 fetch（line 51 `post`，line 74 `transformToTreeData`），**批 0 核心**
- `xingran-react-frontend/src/components/shared/DepartmentTreeSelect.tsx` — 受控下拉，内部 `Department` 接口（line 16）+ `convertDeptTreeData`（line 49），删类型 + 数据外部喂入
- `xingran-react-frontend/src/components/TargetSelector.tsx` — GET 变体 fetch（line 44）

### 受影响模块（file:line 见 DESIGN.md §1.1）
- system: `pages/system/user/hooks/useUserData.ts:75`、`pages/system/user/utils.tsx`、`pages/system/notice/hooks/useTargetSelector.ts:51`
- operations: `pages/operations/workstations/hooks/useWorkstationData.ts:101`、`pages/operations/buildings/useDepartmentTree.tsx:30`、`pages/operations/buildings/useDepartmentData.ts:30`
- network: `pages/network/devices/hooks/useDeviceData.ts:55`
- duty: `pages/duty/pools/index.tsx:174`
- workorder: `pages/workorder/orders/hooks/useWorkOrderData.ts:149` + `lib/workorderApi.ts:253`（重复 `SimpleDept`）

</canonical_refs>

<specifics>
## Specific Ideas

### 验收命令（必须全部通过）
- `grep -rn "/system/departments/tree" xingran-react-frontend/src/` 命中点（排除 role tree-select / dept 管理页 / AD 模块后）= 0，全部经 `useDeptTree`
- `SimpleDept` 全项目唯一定义；`workorderApi` re-export 而非重定义
- `DepartmentTreeSelect` 内部 `Department` 接口删除
- `cd xingran-react-frontend && npm run build` 通过
- `cd xingran-react-frontend && npm run type-check` 通过

### 行为保持验证（手动）
迁移前后逐页核对部门树展示/筛选/选择一致：user / notice / workstations / buildings / network devices / duty pools / workorder orders。

### workstations 特殊语义（高风险点）
`useWorkstationData.ts:101` 的 `buildTreeData` 做路径拼接，后接 `trimTitleToLastSegment` 反向裁剪——双向逻辑特殊。批 2 须单独验证"所属机构"下拉显示与迁移前一致。

</specifics>

<deferred>
## Deferred Ideas

- duty/pools 混用 `DepartmentTreeSelect` + 自 fetch 的进一步 UI 优化（本阶段只统一数据源，不改 UI）
- React Query 缓存刷新策略调优（本阶段沿用 `useInvalidateDept` 现有语义）
- AD 域控 `ous` 子页的"OU↔系统部门"映射是否可解耦（独立数据源，本阶段不动）
- `system/dept` 管理页是否改用 hook（管理者而非消费者，本阶段不强制）

</deferred>

---

*Phase: 37-dept-select-unify*
*Context gathered: 2026-06-22，由 DESIGN.md（brainstorming locked 决策）合成，等效 discuss-phase 输出*
