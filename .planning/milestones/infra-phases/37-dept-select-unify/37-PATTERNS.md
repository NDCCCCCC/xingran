# Phase 37: 前端部门选择组件统一收敛 - Pattern Map

**Mapped:** 2026-06-22
**Files analyzed:** 15 (创建 0 + 修改 15; 本阶段为行为保持型重构,无新建文件)
**Analogs found:** 15 / 15(全部为内部 analog:本阶段是在既有标准件上推广收敛,analog = 各文件当前实现 + 已存在的 canonical 标准件)

> **重要前提说明**:本阶段是**纯前端、行为保持型重构**,所有 analog 均位于 `xingran-react-frontend/src/`。所谓"改造前 vs 改造后"的对比,planner 应基于本文件给出的"当前自 fetch 实现"对照"`useDeptTree`/`deptUtils` canonical 标准件"来组织。

---

## File Classification

| # | New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|-------------------|------|-----------|----------------|---------------|
| 1 | `hooks/useDeptTree.ts` | hook (canonical, 已存在) | request-response | 自身 + `lib/queryKeys.ts` | exact(已落地,推广) |
| 2 | `utils/deptUtils.ts` | utility (canonical, 已存在) | transform | 自身 | exact(已落地,新增导出) |
| 3 | `lib/dutyApi.ts` | api (canonical 锚点) | request-response | 自身(`getDeptTree`/`SimpleDept`) | exact |
| 4 | `lib/workorderApi.ts:253` | api | request-response | `lib/dutyApi.ts:281` | exact(重复 `SimpleDept`,改 re-export) |
| 5 | `lib/queryKeys.ts` | config (canonical, 已存在) | — | 自身 | exact |
| 6 | `components/shared/DepartmentTreeSelect.tsx` | component (受控下拉) | request-response(外部喂入) | 自身 + `lib/dutyApi.ts:SimpleDept` | role-match(删本地 `Department`/`convertDeptTreeData`) |
| 7 | `components/DeptTree/index.tsx` | component (筛选面板) | request-response | `hooks/useDeptTree.ts` + 自身 | exact(去内部 `post`+`transformToTreeData`) |
| 8 | `components/TargetSelector.tsx` | component | request-response | `hooks/useDeptTree.ts` + `notice/hooks/useTargetSelector.ts:51` | role-match(GET→hook) |
| 9 | `pages/system/user/hooks/useUserData.ts:73` | hook | request-response | `hooks/useDeptTree.ts` | exact |
| 10 | `pages/system/user/utils.tsx:24` | utility | transform | `utils/deptUtils.ts` + `DepartmentTreeSelect.convertDeptTreeData:49` | role-match(注意语义见下) |
| 11 | `pages/system/notice/hooks/useTargetSelector.ts:43` | hook | request-response | `hooks/useDeptTree.ts` | exact(GET→hook) |
| 12 | `pages/operations/workstations/hooks/useWorkstationData.ts:99` | hook | request-response + transform | `hooks/useDeptTree.ts` + `utils/deptUtils.trimTitleToLastSegment` | role-match(保留双向语义,高风险) |
| 13 | `pages/operations/buildings/useDepartmentTree.tsx` | hook | request-response | `hooks/useDeptTree.ts` | exact(合并,见批 2) |
| 14 | `pages/operations/buildings/useDepartmentData.ts` | hook | request-response | `hooks/useDeptTree.ts` | exact(与上一项合并) |
| 15 | `pages/network/devices/hooks/useDeviceData.ts:53` | hook | request-response | `hooks/useDeptTree.ts` | exact |
| 16 | `pages/duty/pools/index.tsx:172` | page (典型消费方 analog) | request-response | `hooks/useDeptTree.ts` | exact |
| 17 | `pages/workorder/orders/hooks/useWorkOrderData.ts:149` | hook | request-response | `hooks/useDeptTree.ts` + `lib/dutyApi.ts:getDeptTree` | exact(消除 `workorderApi.getDeptTree` 副本) |

> AD 域控整模块(`pages/ad-domain/*` 的 `getADOUTree`/`ADOUNode`,字段 `dn`/`name`)是**独立数据源,排除在外**。证据:`pages/ad-domain/{users,groups,ous,computers}/index.tsx` 均使用 `getADOUTree`,与 `sys_dept`/`useDeptTree`(`id`/`deptName`)字段级别不同。
> `system/role/hooks/useRoleData.ts:119` 使用 `/system/departments/tree-select`(不同端点,带 `key` 节点,数据范围权限专用)—— 排除。

---

## Pattern Assignments

### `hooks/useDeptTree.ts` (hook, canonical 锚点 — 已落地)

**Analog:** 自身(批 0 不改,后续所有批次引用此标准件)

**当前完整签名**(line 1-50):
```typescript
import { useQuery, useQueryClient, type UseQueryResult } from "@tanstack/react-query";
import { getDeptTree, type SimpleDept } from "@/lib/dutyApi";
import { queryKeys } from "@/lib/queryKeys";

export type DeptTreeNode = SimpleDept;  // line 20 — canonical 类型,全项目唯一

interface DeptTreeResponse {
  code: number;
  data?: DeptTreeNode[];
}

// 标准数据层 hook — 全项目唯一部门树数据源
export function useDeptTree(): UseQueryResult<DeptTreeNode[]> {
  return useQuery({
    queryKey: queryKeys.dept.tree(),             // ['dept', 'tree']
    queryFn: async () => {
      const res = (await getDeptTree()) as DeptTreeResponse;
      return (res.data ?? []) as DeptTreeNode[];
    },
    staleTime: 5 * 60 * 1000,                    // 5min stale
    gcTime: 30 * 60 * 1000,                      // 30min gc
    refetchOnWindowFocus: false,
  });
}

// 写后失效:部门 CRUD 后调用,使所有 useDeptTree 消费者下次访问时重新拉取
export function useInvalidateDept() {
  const qc = useQueryClient();
  return () => qc.invalidateQueries({ queryKey: queryKeys.dept.all });  // ['dept']
}
```

**Planner 提示**:本文件批 0 不需改动,但所有批次迁移消费方时**必须替换为**:
```typescript
const { data: departments, isLoading } = useDeptTree();
```
替代 `useState + useCallback + post('/system/departments/tree')` 自 fetch 模式。

---

### `utils/deptUtils.ts` (utility, canonical 锥点 — 批 0 新增导出)

**Analog:** 自身(批 0 在此新增 `toFullPathTree` / `toShortNameDataNode`)

**当前已有导出**(line 20-161):
```typescript
export interface DeptLikeNode {            // line 20 — 跨形状兼容层
  id?: string; value?: string; key?: string;
  isExternalOrg?: number; children?: DeptLikeNode[];
}
export function getDeptNodeId(node: DeptLikeNode): string  // line 29
export function filterExternalOrgDepts<T extends DeptLikeNode>(nodes: T[]): T[]   // line 52
export function findDeptNode<T extends DeptLikeNode>(nodes: T[], id: string): T | null  // line 77
export function collectDescendantIds<T extends DeptLikeNode>(nodes: T[], id: string): string[]  // line 95
export function trimTitleToLastSegment<T extends { title?: string; children?: T[] }>(nodes: T[]): T[]  // line 145
```

**批 0 新增导出签名建议**(planner 据下列语义实现):

```typescript
// 语义 1 — 全路径显示(从二级部门拼 "A / B / C")
// 替代:DepartmentTreeSelect.convertDeptTreeData:49 + workstations buildTreeData:78
export function toFullPathTree<T extends DeptLikeNode & { deptName?: string }>(
  nodes: T[],
  opts?: { startFromLevel?: 1 | 2 }   // 默认从二级开始(跳顶级)
): Array<T & { title: string; value: string; key: string }>;

// 语义 2 — 短名 antd DataNode(只显示 deptName)
// 替代:DeptTree/index.tsx transformToTreeData:74 + notice/useTargetSelector convertTree:52 + user/utils.tsx convertDeptTreeData:24
export function toShortNameDataNode<T extends DeptLikeNode & { deptName?: string }>(
  nodes: T[]
): Array<{ title: string; key: string; value?: string; children?: unknown[]; isLeaf?: boolean }>;
```

**Planner 提示(DESIGN.md §1.3 警告的具体化)**:
- `DepartmentTreeSelect.convertDeptTreeData`(line 49-75)语义 = **全路径,从二级开始**(`currentPath.slice(1)`)
- `user/utils.tsx convertDeptTreeData`(line 24-39)语义 = **短名**(仅 `node.deptName`,**不是全路径** — 实测代码,与 DESIGN.md §1.3 描述略有出入,planner 须按"实测短名"归类到 `toShortNameDataNode`)
- `DeptTree/index.tsx transformToTreeData`(line 74-85)语义 = **短名**
- `notice/useTargetSelector.ts convertTree`(line 52-60)语义 = **短名**(同时透传 `...node`)
- `workstations/useWorkstationData.ts buildTreeData`(line 78-97)语义 = **全路径(顶级直接显示,二级起拼)**,且**必须透传 `isExternalOrg`**(否则 `filterExternalOrgDepts` 会把整棵树过滤为空)

**禁止**(D-LOCKED):合成单一 `convertTree`。必须保留 2~3 个语义维度。

---

### `lib/dutyApi.ts` (api, canonical 锚点 — `SimpleDept` 全项目唯一)

**Analog:** 自身(line 281, 303)

**当前 canonical 定义**(line 281-305):
```typescript
export interface SimpleDept {              // line 281 — canonical,全项目唯一
  id: string;
  deptName: string;
  parentId?: string;
  children?: SimpleDept[];
}

// 获取部门树(用于下拉选择)— 唯一 fetch 函数
export function getDeptTree(): Promise<BaseResponse<SimpleDept[]>> {
  return post("/system/departments/tree");  // line 303-305
}
```

**Planner 提示**:批 0 在 CONTEXT `<decisions>` 中 Claude's Discretion 允许将 `SimpleDept` 迁到 `types/`,但**强烈建议保留在 `dutyApi.ts`**(已落地、`useDeptTree` 已 import、消费者最多)。批 4 `workorderApi.ts` 改为 `export type { SimpleDept } from "./dutyApi"`。

---

### `lib/workorderApi.ts` (api — 批 4 改 re-export)

**Analog:** `lib/dutyApi.ts:281`

**当前重复定义**(line 253-258,**与 dutyApi 完全相同**):
```typescript
export interface SimpleDept {   // 重复定义
  id: string;
  deptName: string;
  parentId?: string;
  children?: SimpleDept[];
}
```

**改造后**:
```typescript
// 批 4:删本地 SimpleDept,改为从 canonical 锚点 re-export
export type { SimpleDept } from "./dutyApi";
// 同时检查 workorderApi 是否有 getDeptTree 副本,若有同样删除并让消费方直接 import dutyApi
```

---

### `components/shared/DepartmentTreeSelect.tsx` (component — 批 0)

**Analog:** 自身 + `lib/dutyApi.ts:SimpleDept`

**当前内部 `Department` 接口**(line 16-20,需删除):
```typescript
export interface Department {           // line 16 — 重复类型,批 0 删除
  id: string;
  deptName: string;
  children?: Department[];
}
```

**当前内部 `convertDeptTreeData`**(line 49-75,语义 = **全路径,从二级开始**):
```typescript
function convertDeptTreeData(departments: Department[], parentPath: string[] = []): TreeNode[] {
  // ...
  const convertNodes = (nodes: Department[], currentPath: string[]): TreeNode[] => {
    return nodes.map(node => {
      let displayPath: string;
      if (currentPath.length === 0) {
        displayPath = node.deptName;            // 顶级直接显示
      } else {
        const pathFromSecondLevel = [...currentPath.slice(1), node.deptName];
        displayPath = pathFromSecondLevel.join(" / ");   // 从二级开始拼 "A / B / C"
      }
      return { title: displayPath, value: node.id, key: node.id,
               children: node.children?.length ? convertNodes(node.children, [...currentPath, node.deptName]) : undefined };
    });
  };
  return convertNodes(departments, parentPath);
}
```

**改造方向(批 0)**:
1. 删除 `Department` 接口(line 16-20)
2. props 中 `departments?: Department[]` 改为 `departments?: SimpleDept[]`(import from `@/lib/dutyApi`)
3. 删除 `convertDeptTreeData`(line 49-75),改调用 `deptUtils.toFullPathTree`
4. **保持受控模式不变**(props `value`/`onChange`/外部 `departments`),不内部 fetch
5. `DepartmentTreeSelectWithTop` 同步处理(line 146-217)

**Planner 提示**:受控模式是 D-LOCKED,调用方从 `useDeptTree()` 喂入 `departments={departments ?? []}`,**禁止**在此组件内调 hook。

---

### `components/DeptTree/index.tsx` (component — 批 0 核心)

**Analog:** `hooks/useDeptTree.ts`

**当前自 fetch 实现**(line 48-72,**批 0 删除**):
```typescript
const loadDeptTree = useCallback(async () => {
  setLoading(true);
  try {
    const result = await post<Department[]>("/system/departments/tree");  // line 51 — 删
    let deptData = result.data || [];
    if (externalOnly) {
      deptData = filterExternalOrgDepts(deptData as Department[]);        // 保留(已用 deptUtils)
    }
    const treeData = transformToTreeData(deptData);                       // line 60 — 改 toShortNameDataNode
    setTreeData(treeData);
    const parentKeys = getParentKeys(treeData);
    setExpandedKeys(parentKeys.length > 0 ? [parentKeys[0]] : []);
  } catch (error) { /* ... */ }
  finally { setLoading(false); }
}, [externalOnly]);

useEffect(() => { loadDeptTree(); }, [loadDeptTree]);  // line 154-156 — 删
```

**当前 `transformToTreeData`**(line 74-85,语义 = **短名**):
```typescript
const transformToTreeData = (data: Department[]): DataNode[] => {
  if (!Array.isArray(data)) return [];
  return data.map(item => ({
    title: item.deptName,
    key: item.id,
    children: item.children?.length ? transformToTreeData(item.children) : undefined,
    isLeaf: !item.children || item.children.length === 0,
  }));
};
```

**改造方向(批 0)**:
1. import `useDeptTree`,删 `post` import
2. 删 `useState treeData/loading`、`loadDeptTree`、`transformToTreeData`、`useEffect`
3. 替换为:
   ```typescript
   const { data: rawDept = [], isLoading } = useDeptTree();
   const treeData = useMemo(() => {
     const filtered = externalOnly ? filterExternalOrgDepts(rawDept) : rawDept;
     return toShortNameDataNode(filtered);
   }, [rawDept, externalOnly]);
   ```
4. **保留**搜索逻辑(`onSearch`/`getExpandedKeys`/`filterTreeData` line 87-152)、**保留**展开逻辑(`onExpand`/`expandedKeys`/`autoExpandParent`)
5. props `externalOnly` 保留;`Department` 类型 import 从 `@/types` 改为 canonical `SimpleDept`

**Planner 提示**:批 0 验收要手动核对 3 个使用页(user / buildings / network devices)的筛选行为不变。搜索/展开逻辑不能动。

---

### `components/TargetSelector.tsx` (component — 批 5)

**Analog:** `hooks/useDeptTree.ts` + `notice/hooks/useTargetSelector.ts:51`

**当前 GET 变体自 fetch**(line 41-52):
```typescript
const loadDeptTree = async () => {
  setLoading(true);
  try {
    const response = await get<DeptTreeNode[]>("/system/departments/tree");  // line 44 — GET 变体
    setDeptTree(response.data || []);
  } catch (error) { /* ... */ }
  finally { setLoading(false); }
};
```

**改造方向(批 5)**:
1. 删 `loadDeptTree`/`deptTree useState`/相关 `useEffect`
2. 用 `useDeptTree()` 替代(GET 变体语义等同 POST,后端同端点)
3. 注意 `TargetSelector` 还同时加载 roles/users,这部分**不动**

---

### `pages/system/user/hooks/useUserData.ts` (hook — 批 1)

**Analog:** `hooks/useDeptTree.ts`

**当前自 fetch**(line 73-81):
```typescript
const loadDepartments = useCallback(async () => {
  try {
    const result = await post<Department[]>("/system/departments/tree");  // line 75
    setDepartments(result.data || []);
  } catch (error) {
    handleApiError(error, "加载部门列表", false);
    setDepartments([]);
  }
}, []);
```

**改造方向(批 1)**:
- 删 `departments useState`/`loadDepartments`/返回值中的 `loadDepartments`
- 替换为 `const { data: departments = [] } = useDeptTree();`
- 调用方(`pages/system/user/index.tsx`)把 `departments` 从 hook 返回值取,`loadDepartments` 调用点删除
- 配合 `pages/system/user/utils.tsx convertDeptTreeData` 删除(见下)

---

### `pages/system/user/utils.tsx` (utility — 批 1)

**Analog:** `utils/deptUtils.ts`(批 0 新增 `toShortNameDataNode`)

**当前 `convertDeptTreeData`**(line 24-39,**实测语义 = 短名**,与 DESIGN.md §1.3 描述略有出入):
```typescript
export function convertDeptTreeData(departments: { id: string; deptName: string; children?: unknown[] }[]): TreeNode[] {
  if (!departments || departments.length === 0) return [];
  const convertNodes = (nodes) => nodes.map(node => ({
    title: node.deptName,                   // 短名,不是全路径
    value: node.id,
    key: node.id,
    children: node.children?.length ? convertNodes(node.children) : undefined,
  }));
  return convertNodes(departments);
}
```

**改造方向(批 1)**:
- 删本地 `convertDeptTreeData`(line 24-39)、`TreeNode` 接口(line 14-19)
- 调用方改 import `toShortNameDataNode` from `@/utils/deptUtils`
- `renderDeptTreeOptions`(line 44-70)是 Select 专用渲染(不是 TreeSelect),**保留**(语义独立,仅渲染 `<Option>` 树形缩进)

**Planner 提示**:`renderDeptTreeOptions` 是另一种 UI 形态(`Select` 而非 `TreeSelect`),**不要**误收敛到 `toShortNameDataNode`。

---

### `pages/system/notice/hooks/useTargetSelector.ts` (hook — 批 1)

**Analog:** `hooks/useDeptTree.ts`

**当前 GET 变体自 fetch + convertTree**(line 43-67,语义 = **短名** + 透传 `...node`):
```typescript
const loadDeptTree = useCallback(async () => {
  setLoadingDepts(true);
  try {
    interface DeptNode { id: string; deptName: string; children?: DeptNode[]; }
    const response = await get<DeptNode[]>("/system/departments/tree");   // line 51 — GET 变体
    const convertTree = (nodes: DeptNode[]): Target[] => nodes.map(node => ({
      ...node, title: node.deptName, key: node.id, value: node.id,
      children: node.children ? convertTree(node.children) : undefined,
    }));
    setDeptTree(convertTree(response.data || []));
  } catch (error) { /* ... */ }
  finally { setLoadingDepts(false); }
}, []);
```

**改造方向(批 1)**:
- 删 `loadDeptTree`/`deptTree useState`,改 `const { data: rawDept = [] } = useDeptTree();`
- `convertTree` 改调 `toShortNameDataNode`(注意 `Target` 接口有额外字段 `roleName`/`username` 等,dept 子树只用到 `id/deptName/children`,可安全替换)
- 保持 `Target` 接口不变(roles/users 还需要)

---

### `pages/operations/workstations/hooks/useWorkstationData.ts` (hook — 批 2, 高风险)

**Analog:** `hooks/useDeptTree.ts` + `utils/deptUtils.trimTitleToLastSegment`

**当前 `buildTreeData` + `loadDeptOptions`**(line 78-109,**特殊双向语义**):
```typescript
const buildTreeData = useCallback((depts: DepartmentNode[]): DeptTreeNode[] => {
  const build = (nodes, ancestorNames) => nodes.map((node) => {
    const name = node.deptName || node.name || "";
    const title = ancestorNames.length === 0 ? name : [...ancestorNames, name].join(" / ");
    return {
      title,                                   // 全路径(顶级直接,二级起拼)
      value: node.id,
      key: node.id,
      isExternalOrg: node.isExternalOrg,       // 必须透传(line 75-77 注释警告)
      children: node.children?.length ? build(node.children, [...ancestorNames, name]) : undefined,
    };
  });
  return build(depts, []);
}, []);

const loadDeptOptions = useCallback(async () => {
  try {
    const result = await post("/system/departments/tree", {}) as { data: DepartmentNode[] };  // line 101
    setDeptTreeData(buildTreeData(deptList));    // 设的是全路径版本
  } catch (error) { handleApiError(error, "加载部门选项", false); }
}, [setDeptTreeData, buildTreeData]);
```

**关键**:`buildTreeData` 输出**全路径**版本;**页面层另有 `trimTitleToLastSegment`** 在 `orgTreeData` useMemo 处反向裁剪为短名(用于"所属部门"下拉)。这是**双向逻辑**,迁移必须保持:
- `deptTreeData` = 全路径(`useMemo` + `toFullPathTree`)
- `orgTreeData` = 派生自 `deptTreeData` + `filterExternalOrgDepts`(外部机构子集)
- "所属部门"下拉数据 = `trimTitleToLastSegment(orgTreeData)` 反向裁剪

**改造方向(批 2)**:
1. 删 `DepartmentNode` 本地接口(line 14-20)、`buildTreeData`、`loadDeptOptions` 中的 `post`
2. 替换为 `const { data: rawDept = [] } = useDeptTree();`
3. `deptTreeData = useMemo(() => toFullPathTree(rawDept), [rawDept])`(注意 `toFullPathTree` 要透传 `isExternalOrg`)
4. `DepartmentNode` 用 `DeptTreeNode`(本文件局部类型 line 33-45)替代,或 import `SimpleDept`
5. **批 2 验收重点**:核对"所属机构"下拉(全路径)与"所属部门"下拉(短名,经 `trimTitleToLastSegment`)显示与迁移前一致

**Planner 提示**:这是 CONTEXT `<specifics>` 标注的高风险点,建议批 2 单独提交并单独验证。

---

### `pages/operations/buildings/useDepartmentTree.tsx` (hook — 批 2, 与下一项合并)

**Analog:** `hooks/useDeptTree.ts`

**当前自 fetch + `convertToTreeData`**(line 27-44):
```typescript
const loadDepartments = useCallback(async () => {
  setLoading(true);
  try {
    const result = await post<DepartmentOption[]>("/system/departments/tree", {});   // line 30
    const convertToTreeData = (nodes: DepartmentNode[]): DepartmentOption[] =>
      nodes.map(node => ({
        id: node.id, deptName: node.deptName,
        children: node.children?.length ? convertToTreeData(node.children!) : undefined,
      }));
    setDepartments(convertToTreeData(result.data || []));
  } catch (error) { handleApiError(error, "加载部门列表", false); }
  finally { setLoading(false); }
}, []);
```

**改造方向(批 2)**:删 `DepartmentOption`/`DepartmentNode` 本地接口、`loadDepartments`/`convertToTreeData`,改用 `useDeptTree()`。保留 `deptMap`/`getOrgName` 派生逻辑(line 47-64)。

---

### `pages/operations/buildings/useDepartmentData.ts` (hook — 批 2, 与上一项合并)

**Analog:** `hooks/useDeptTree.ts`(与上一项语义几乎完全相同)

**当前实现**(line 26-48):与上一项 `useDepartmentTree` 几乎完全重复(`post` + `convertToTreeData` + `getDeptMap` + `getOrgName`)。

**改造方向(批 2)**:CONTEXT `<decisions>` Claude's Discretion 第 4 条允许"顺带合并到 `useDeptTree`"。建议:
- **方案 A(推荐)**:两个 hook 都删除,buildings 页直接用 `useDeptTree()` + 本地 `useMemo` 计算 `deptMap`/`getOrgName`
- 方案 B:保留其中一个 hook 文件作为 buildings 模块包装层,内部调 `useDeptTree()`

**Planner 决定**。

---

### `pages/network/devices/hooks/useDeviceData.ts` (hook — 批 3)

**Analog:** `hooks/useDeptTree.ts`

**当前自 fetch**(line 53-61):
```typescript
const loadDepartments = useCallback(async () => {
  try {
    const result = await post<Department[]>("/system/departments/tree", {});  // line 55
    setDepartments(result.data || []);
  } catch (error) {
    handleApiError(error, "加载部门列表", false);
    setDepartments([]);
  }
}, []);
```

**改造方向(批 3)**:删 `departments useState`/`loadDepartments`,改 `const { data: departments = [] } = useDeptTree()`。

---

### `pages/duty/pools/index.tsx` (page — 批 4, 典型消费方迁移 analog)

**Analog:** `hooks/useDeptTree.ts`(planner 可以此页为"迁移前/后"模板推广到其他消费方)

**当前自 fetch**(line 171-179):
```typescript
// 获取部门列表
const fetchDepts = async () => {
  try {
    const result = await getDeptTree();            // line 174 — 来自 lib/dutyApi
    setDepts(result.data || []);
  } catch (error) {
    console.error('获取部门列表失败:', error);
  }
};

useEffect(() => {
  fetchList(1, paginationProps.pageSize);
  fetchStats();
  fetchUsers();
  fetchDepts();                                    // line 185 — 删
  // eslint-disable-next-line react-hooks/exhaustive-deps
}, []);
```

**改造方向(批 4)**:
1. 删 `fetchDepts`/`depts useState`/`useEffect` 中的 `fetchDepts()` 调用
2. 顶层加 `const { data: depts = [] } = useDeptTree();`
3. duty/pools 还混用 `DepartmentTreeSelect`(受控)—— 数据由 `depts` 喂入,保持受控模式

**Planner 提示**:这是迁移模板 analog。所有"页面级 useEffect + fetchXxx + useState"消费方都按此模式替换为 `useDeptTree()` 顶层调用。

---

### `pages/workorder/orders/hooks/useWorkOrderData.ts` (hook — 批 4)

**Analog:** `hooks/useDeptTree.ts` + `lib/dutyApi.ts:getDeptTree`

**当前自 fetch**(line 149-156,通过 `workorderApi.getDeptTree` 副本):
```typescript
const fetchDepts = useCallback(async () => {
  try {
    const result = await getDeptTree();     // line 151 — import 自 workorderApi(副本)
    setDepts(result.data || []);
  } catch (error) { console.error("获取部门列表失败:", error); }
}, []);
```

**改造方向(批 4)**:
1. 删 `fetchDepts`/`depts useState`,改 `const { data: depts = [] } = useDeptTree()`
2. 配合 `workorderApi.ts:253` 删 `SimpleDept`(改为 re-export from dutyApi)
3. 若 `workorderApi` 还有 `getDeptTree` 副本,一并删除(让所有消费方直接用 `useDeptTree`/`dutyApi.getDeptTree`)

---

## Shared Patterns

### Canonical 数据层 hook 调用(替换所有自 fetch)

**Source:** `hooks/useDeptTree.ts`
**Apply to:** 所有批 1~5 的消费方 hook/page(共 9 处自 fetch)

**统一替换模式**:
```typescript
// ❌ 迁移前 — 每处都自 fetch
const [depts, setDepts] = useState<XxxDeptType[]>([]);
const fetchDepts = useCallback(async () => {
  try {
    const result = await post("/system/departments/tree", {});  // 或 get, 或 getDeptTree()
    setDepts(result.data || []);
  } catch (error) { handleApiError(error, "...", false); }
}, []);
useEffect(() => { fetchDepts(); }, [fetchDepts]);

// ✅ 迁移后 — 共享 React Query 缓存
import { useDeptTree } from "@/hooks/useDeptTree";
const { data: depts = [], isLoading } = useDeptTree();
```

**收益**:所有消费者共享同一 `['dept', 'tree']` 缓存条目(5min stale, 30min gc),写操作后调 `useInvalidateDept()` 统一失效。

---

### Canonical 类型引用(替换所有重复 `Department`/`SimpleDept`)

**Source:** `lib/dutyApi.ts:281`
**Apply to:** `workorderApi.ts:253`、`DepartmentTreeSelect.tsx:16`、各 hook 本地 `DepartmentNode`/`DepartmentOption` 接口

**统一引用**:
```typescript
import type { SimpleDept } from "@/lib/dutyApi";   // 全项目唯一
// 跨形状兼容(对消费方树转换代码)用:
import type { DeptLikeNode } from "@/utils/deptUtils";
```

---

### 错误处理

**Source:** `@/utils/errorHandler` 的 `handleApiError`(已在多个 hook 中使用)
**Apply to:** 所有 hook 改造点

**注意**:`useDeptTree` 内部不做错误处理(React Query 默认把错误吞进 `error` 状态),消费方迁移后**不再需要** `try/catch + handleApiError` 包裹 fetch 调用 —— 但消费方可以读 `useDeptTree()` 的 `error` 状态决定是否 toast。

**迁移前**:`try { ... } catch (e) { handleApiError(e, "加载部门列表", false); }`
**迁移后**:不需要(由 React Query 统一管理;若需 UI 反馈,读 `isError`/`error`)

---

### 缓存失效(写操作后)

**Source:** `hooks/useDeptTree.ts:47` `useInvalidateDept`
**Apply to:** 所有部门 CRUD 操作后(system/dept 管理页本体 —— 但 CONTEXT `<deferred>` 标注 dept 管理页不强制改)

**当前部门 CRUD 失效**:`system/dept` 管理页自己处理刷新(本阶段不动)。
**改造后预期**:若 dept 管理页将来改用 `useDeptTree`,写操作末尾调 `const invalidate = useInvalidateDept(); invalidate();`。本阶段不强制。

---

### useEffect 依赖稳定性(React Best Practice)

**Source:** `CLAUDE.md` "Frontend/React Best Practices > useEffect Dependencies"
**Apply to:** 所有批 1~5 改造点

**迁移后的天然收益**:删掉自 fetch 的 `useEffect` 后,**依赖不稳定问题自动消失**(不再有 `loadDeptTree` 是否 `useCallback` 依赖正确的纠结)。`useDeptTree()` 顶层调用,React Query 自管理生命周期。

**Planner 提示**:改造时若发现原代码有 `useEffect(depts)` 之类依赖,迁移后 depts 来自 hook 顶层,引用稳定(React Query 返回结构稳定),不会触发无限重渲染。

---

## No Analog Found

本阶段**全部为内部 analog**(在既有标准件上推广),无"代码库中找不到对应"的情况。唯一需要外部参考的是 React Query 的 `useQuery`/`useQueryClient` 标准用法,但 `useDeptTree.ts` 已经在用,planner 直接复用模式。

---

## Metadata

**Analog search scope:**
- `xingran-react-frontend/src/hooks/` — 全部 hook
- `xingran-react-frontend/src/components/{DeptTree,shared,TargetSelector}.tsx`
- `xingran-react-frontend/src/lib/{dutyApi,workorderApi,queryKeys,api}.ts`
- `xingran-react-frontend/src/utils/deptUtils.ts`
- `xingran-react-frontend/src/pages/{system/user,system/notice,operations/workstations,operations/buildings,network/devices,duty/pools,workorder/orders}/`
- `xingran-react-frontend/src/pages/ad-domain/*`(仅核对排除依据,不映射)
- `xingran-react-frontend/src/pages/system/role/hooks/useRoleData.ts`(仅核对排除依据,不映射)

**Files scanned:** 18
**Pattern extraction date:** 2026-06-22

**关键提醒给 planner**:
1. 批 0 是所有后续批次的硬前置 —— 必须先完成 `deptUtils` 新增导出 + `DepartmentTreeSelect` 删类型 + `DeptTree` 去 fetch,再做任何模块迁移
2. workstations 批 2 是高风险点(`buildTreeData` + `trimTitleToLastSegment` 双向语义),建议单独 plan/单独验证
3. `user/utils.tsx convertDeptTreeData` 实测是**短名**语义(非 DESIGN.md §1.3 暗示的"全路径"),归到 `toShortNameDataNode`
4. AD 模块、role tree-select、dept 管理页本体 —— 严禁触碰(数据源级别错误防护)
5. `DepartmentTreeSelect` 必须保持受控模式,数据外部喂入,**禁止**内部 `useDeptTree`
