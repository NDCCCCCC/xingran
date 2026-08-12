# Phase 37 设计文档 — 前端部门选择组件统一收敛

> 本文档由 brainstorming 产出 + 代码调查证据固化，作为 `/gsd-plan-phase 37` 的输入。
> 与 ROADMAP.md 的 Phase 37 条目一致；架构决策已 locked。

---

## 1. 背景与问题

前端"部门列表/选择器/树"实现**严重碎片化**，三类重复：

### 1.1 重复 fetch（约 6+ 处自行获取部门树，绕过缓存）
| 文件 | 行 | 方式 |
|------|----|------|
| `components/DeptTree/index.tsx` | 51 | `post('/system/departments/tree')`（**公共组件自己都没用 hook**） |
| `pages/system/user/hooks/useUserData.ts` | 75 | `post('/system/departments/tree')` |
| `pages/system/notice/hooks/useTargetSelector.ts` | 51 | `get('/system/departments/tree')`（GET 变体） |
| `pages/duty/pools/index.tsx` | 174 | `getDeptTree()`（dutyApi） |
| `pages/workorder/orders/hooks/useWorkOrderData.ts` | 149 | `getDeptTree()`（workorderApi 副本） |
| `pages/operations/workstations/hooks/useWorkstationData.ts` | 101 | `post('/system/departments/tree')` + 自定义 `buildTreeData` 路径拼接 |
| `pages/operations/buildings/useDepartmentTree.tsx` | 30 | `post('/system/departments/tree')` |
| `pages/operations/buildings/useDepartmentData.ts` | 30 | `post('/system/departments/tree')` |
| `pages/network/devices/hooks/useDeviceData.ts` | 55 | `post('/system/departments/tree')` |
| `components/TargetSelector.tsx` | 44 | `get('/system/departments/tree')`（GET 变体） |

### 1.2 重复类型定义（3 份）
- `lib/dutyApi.ts:281` — `SimpleDept { id, deptName, parentId?, children? }`
- `lib/workorderApi.ts:253` — `SimpleDept`（**与 dutyApi 完全相同的重复定义**）
- `components/shared/DepartmentTreeSelect.tsx:16` — `Department { id, deptName, children? }`（第 3 份，最精简）

### 1.3 重复树转换函数（4+ 份，语义有差异）
| 文件 | 函数 | 语义 |
|------|------|------|
| `DepartmentTreeSelect.tsx:49` | `convertDeptTreeData` | 全路径显示（从二级部门拼 `A / B / C`） |
| `pages/system/user/utils.tsx` | `convertDeptTreeData` | 同上语义的重复 |
| `DeptTree/index.tsx:74` | `transformToTreeData` | 转 antd `DataNode`（短名） |
| `pages/system/notice/hooks/useTargetSelector.ts` | `convertTree` | 重复转换 |
| `pages/operations/workstations/hooks/useWorkstationData.ts` | `buildTreeData` | 路径拼接（后接 `trimTitleToLastSegment` 反向裁剪） |

> **注意**：转换函数语义不完全相同（全路径 vs 短名 vs DataNode）。收敛时须保留语义维度，不能粗暴合成一个函数——应抽到 `deptUtils` 提供 2~3 个明确命名的转换（如 `toFullPathTree` / `toShortNameDataNode`），调用方按需选用。

---

## 2. 已有标准件（收敛基础，已存在但未推广）

### 2.1 数据层 ✅
- `hooks/useDeptTree.ts` — React Query 封装，queryKey `queryKeys.dept.tree()`，5min stale / 30min gc，`refetchOnWindowFocus: false`
- `hooks/useDeptTree.ts:47` — `useInvalidateDept()` 写后失效
- 基于 `lib/dutyApi.ts:303` `getDeptTree()` → `POST /system/departments/tree`

### 2.2 类型/工具层 ✅（已部分收敛）
- `utils/deptUtils.ts` — `DeptLikeNode`（兼容 `id`/`value`/`key` 三种节点形状）+ `filterExternalOrgDepts` / `findDeptNode` / `collectDescendantIds` / `trimTitleToLastSegment`

### 2.3 组件层
- `DeptTree`（面板，筛选用） — **自 fetch，需改造为消费 hook**
- `DepartmentTreeSelect`（表单下拉，**纯受控**） — 保持受控，数据由调用方从 hook 喂入
- `DeptSidebar`（运维侧边栏） — 封装 `DeptTree`，无需改

---

## 3. 目标架构（单向依赖，四层）

```
页面层  pages/*  →  只消费 hook + 渲染公共组件, 不再自 fetch/自转换
  │
数据层  useDeptTree() ← 全项目唯一部门树数据源
        (React Query 单缓存; useInvalidateDept 写后失效)
  │
API 层  getDeptTree() (dutyApi) → POST /system/departments/tree
  │
类型层  Dept = SimpleDept (canonical, 全项目唯一);
        DeptLikeNode (deptUtils) 跨形状兼容层

组件层 (并列, 职责正交, 不合并):
  DeptTree              筛选面板      → 改为消费 useDeptTree, 不再自 fetch
  DepartmentTreeSelect  表单下拉(受控) → 保持受控, 调用方喂 hook 数据
  DeptSidebar           运维侧边栏    → 继续封装 DeptTree
工具层  deptUtils (filter/find/collect/trim + 新增 toFullPathTree/toShortNameDataNode)
```

---

## 4. 收敛边界

### 4.1 明确排除（不纳入，方案单列理由）
| 排除项 | 理由 | 证据 |
|--------|------|------|
| **AD 域控整模块** `ad-domain/*` | OU 树来自 AD/LDAP，与 `sys_dept` 是两套独立数据源 | `users/computers/groups` 用 `getADOUTree` + `ADOUNode`（字段 `dn`/`name`）；`ous` 用 `useDeptTree` 做"OU↔系统部门"映射，属 AD 模块边缘功能，一并不动 |
| `system/role` 的 `/system/departments/tree-select` | 不同端点，带 `key` 节点用于数据范围权限勾选，语义独立 | `pages/system/role/hooks/useRoleData.ts:119` |
| `system/dept` 部门管理页本体 | 部门树的管理者（CRUD）而非消费者 | `pages/system/dept/index.tsx` + `hooks/useDeptData.ts:44` |

### 4.2 纳入收敛（实际受影响）
- 公共组件：`DeptTree`（去自 fetch）
- system：`user`、`notice`
- operations：`workstations`、`buildings`（含 `useDepartmentTree`/`useDepartmentData`）、`floors`
- network：`devices`
- duty：`pools`
- workorder：`orders`
- `TargetSelector`（GET→统一 POST 经 hook）

---

## 5. 分阶段迁移计划（批 0 公共层 → 批 1..N 模块）

### 批 0 — 公共层（所有后续批次的前置依赖）
1. **类型层去重**：`SimpleDept` 收敛为单一 canonical（建议放 `types/` 或保留 `dutyApi` 导出，`workorderApi` 改 re-export）；删 `DepartmentTreeSelect` 内部 `Department`，改引用 canonical
2. **转换函数归一**：`deptUtils` 新增 `toFullPathTree`（全路径，替代 DepartmentTreeSelect/user 的版本）+ `toShortNameDataNode`（短名，替代 DeptTree 的 transformToTreeData）；迁移调用方
3. **组件层**：`DeptTree` 改为消费 `useDeptTree`，删除内部 `post` + `transformToTreeData`，搜索/展开逻辑保留
4. 验收：`DeptTree` 三处使用页（user/buildings/network devices）行为不变

### 批 1 — system 模块
- `user/hooks/useUserData.ts:75` → `useDeptTree`
- `notice/hooks/useTargetSelector.ts:51` → `useDeptTree`（GET 改经 hook）
- `system/user/utils.tsx` convertDeptTreeData → 删除，用 deptUtils

### 批 2 — operations 模块
- `workstations/hooks/useWorkstationData.ts:101` → `useDeptTree`（注意其 `buildTreeData` 路径拼接语义 + `trimTitleToLastSegment` 反向裁剪，须保留行为）
- `buildings/useDepartmentTree.tsx` + `useDepartmentData.ts` → 合并到 `useDeptTree`（buildings 有两个并存的 dept hook，本批可顺带合并）

### 批 3 — network 模块
- `devices/hooks/useDeviceData.ts:55` → `useDeptTree`

### 批 4 — duty + workorder
- `duty/pools/index.tsx:174` → `useDeptTree`
- `workorder/orders/hooks/useWorkOrderData.ts:149` → `useDeptTree`（改用 dutyApi 的 `getDeptTree` 经 hook，消除 workorderApi 副本）

### 批 5 — TargetSelector + 收尾
- `components/TargetSelector.tsx:44` → `useDeptTree`
- 全量 grep 验证：`grep -rn "/system/departments/tree" src/` 剩余命中只剩排除项（role tree-select / dept 管理页 / AD 模块）

> 每批独立提交、可独立回滚。批 0 是硬前置（后续批次都依赖 canonical 类型 + hook 化的 DeptTree）。

---

## 6. 验收标准（Success Criteria）

- [ ] `grep -rn "/system/departments/tree" xingran-react-frontend/src/` 命中点（排除 role/dept管理页/AD 后）= 0，全部经 `useDeptTree`
- [ ] `SimpleDept` 全项目唯一定义；`workorderApi` re-export 而非重定义
- [ ] `DepartmentTreeSelect` 内部 `Department` 接口删除
- [ ] 4+ 份转换函数收敛到 `deptUtils`（`toFullPathTree` + `toShortNameDataNode`）
- [ ] `DeptTree` 不再内部 `post`，消费 `useDeptTree`
- [ ] 各模块迁移后 UI 行为不变（手动验证：user/notice/workstations/buildings/devices/duty/pools/workorder 的部门树展示与筛选）
- [ ] `cd xingran-react-frontend && npm run build` 通过
- [ ] `cd xingran-react-frontend && npm run type-check` 通过

---

## 7. 风险与回归

| 风险 | 缓解 |
|------|------|
| 转换函数语义差异（全路径 vs 短名）导致 UI 文案变化 | 批 0 在 `deptUtils` 提供 2 个语义明确的函数，逐调用方核对显示行为 |
| workstations 的 `buildTreeData` 路径拼接 + `trimTitleToLastSegment` 双向逻辑特殊 | 批 2 单独验证"所属机构"下拉显示与迁移前一致 |
| `DeptTree` 改造影响 3 个使用页（user/buildings/devices）的筛选 | 批 0 改造后立即手动验证这 3 页 |
| duty/pools 混用 `DepartmentTreeSelect` + 自 fetch | 批 4 统一为 hook 喂数据给 `DepartmentTreeSelect`，保持受控模式 |
| React Query 缓存与原各自 fetch 的刷新时机不同 | `useInvalidateDept` 在部门写操作后调用；验证部门 CRUD 后列表刷新正常 |

---

## 8. 下一步

运行 `/gsd-plan-phase 37`，由 planner 基于本 DESIGN 拆分为可执行的 plan 文件（建议按批 0~5 各自一个 plan，或批 0 单独 + 模块批次合并）。
