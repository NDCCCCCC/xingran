---
phase: 39-workstation-dept-location-alias
plan: 07
subsystem: frontend-operations-workstations
tags: [frontend, react, antd, drawer, alias, permissions]
requires:
  - 39-05 (locationAliasApi + LocationAlias type)
  - 39-06 (backend alias CRUD endpoints)
provides:
  - LocationAliasDrawer 组件 (列表 + 新增 + 删除 + 权限 gating)
  - 工位列表页 [⚙ 映射] 工具栏按钮 + Drawer state
affects:
  - xingran-react-frontend/src/pages/operations/workstations/index.tsx (工具栏注入)
tech-stack:
  added: []
  patterns:
    - antd Drawer + TreeSelect + Table + Popconfirm CRUD 组合
    - useMenuStore permissions 本地 hasPermission pattern (复用 MACHistoryPage 已有约定)
    - tanstack useQuery + invalidateQueries 双失效 (dept + locationAlias)
key-files:
  created:
    - xingran-react-frontend/src/pages/operations/workstations/LocationAliasDrawer.tsx
  modified:
    - xingran-react-frontend/src/pages/operations/workstations/index.tsx
decisions:
  - D-02: 不新增菜单项,Drawer 嵌入工位列表页工具栏
  - D-08: 权限 gating — canListAlias 控按钮可见,canAdd disable 新增,canDelete hide 删除
  - 复用 useMenuStore.permissions 本地 hasPermission (authHelpers 无此导出)
metrics:
  duration: ~15min
  completed: 2026-06-25
  tasks: 2
  files: 2
---

# Phase 39 Plan 07: LocationAliasDrawer 组件 + 工具栏按钮注入 Summary

为工位列表页工具栏添加 `[⚙ 映射]` 按钮 + Drawer 管理 UI, 承载 alias 完整 CRUD (D-02/D-08 决策落地) — 列表分页 + 新增表单(2 个 TreeSelect) + 删除(Popconfirm), 权限严格三态 gating, 写操作触发 useInvalidateDept + invalidate locationAlias 双失效。

## What Was Built

### Task 1: LocationAliasDrawer 组件 (`LocationAliasDrawer.tsx`, commit 313683e)

新建 `xingran-react-frontend/src/pages/operations/workstations/LocationAliasDrawer.tsx`, 导出 `LocationAliasDrawer({ open, onClose })`:

- **Drawer** width=600, title `"工位部门物理位置映射管理"`, `destroyOnHidden` + `maskClosable`
- **新增表单(折叠式)**:
  - `deptId` TreeSelect — 全量部门树 (`trimTitleToLastSegment(deptTreeData)`), required
  - `locationId` TreeSelect — 仅外部机构子树 (`filterExternalOrgDepts(fullDeptTreeData)`), required, `disabled={locationTreeData.length === 0}`
  - `scope` hidden Input — `initialValue="workstation"`, 后端兜底
  - `remark` TextArea — 选填
- **alias 列表 Table**:
  - 列: deptId / locationId / scope / remark / createdAt / 操作
  - 分页 current/pageSize/total, showSizeChanger
  - 操作列: Popconfirm + 删除按钮(canDelete=false 时返回 null 不渲染)
- **数据获取**:
  - `useDeptTree()` (DeptTreeNode 形状,含 isExternalOrg)
  - `useQuery(queryKeys.locationAlias.list({ pageNum, pageSize }))` → `locationAliasApi.list`
- **权限 gating**:
  - `canAdd = hasPermission("ops:location:alias:add")` → false 时新增按钮 disabled + Tooltip "无新增权限"
  - `canDelete = hasPermission("ops:location:alias:delete")` → false 时删除按钮 hidden
  - 权限源:`useMenuStore((s) => s.permissions)`, 本地 `hasPermission = (perm) => permissions.includes(perm)` (复用 MACHistoryPage 已有 pattern)
- **双失效**:
  - `useInvalidateDept()` (失效 `['dept', ...]` 全部 query)
  - `queryClient.invalidateQueries({ queryKey: queryKeys.locationAlias.all })` (失效 `['location-alias', ...]`)
  - `refreshAfterMutation()` 串联: refetch → invalidateDept → invalidateAliasAll

### Task 2: index.tsx 注入按钮 + Drawer state (commit c4e8398)

修改 `xingran-react-frontend/src/pages/operations/workstations/index.tsx`:

- **imports 追加**: `SettingOutlined`, `LocationAliasDrawer`, `useMenuStore`
- **state 块追加**:
  ```ts
  const [aliasDrawerOpen, setAliasDrawerOpen] = useState(false);
  const menuPermissions = useMenuStore((s) => s.permissions);
  const canListAlias = menuPermissions.includes("ops:location:alias:list");
  ```
- **工具栏按钮**(在 `导出` 按钮之后, `批量删除` 按钮之前):
  ```tsx
  {canListAlias && (
    <Button icon={<SettingOutlined />} onClick={() => setAliasDrawerOpen(true)}>
      映射
    </Button>
  )}
  ```
- **Drawer 渲染**(WorkstationEditModal 之后, ExcelImportLazy 之前):
  ```tsx
  <LocationAliasDrawer open={aliasDrawerOpen} onClose={() => setAliasDrawerOpen(false)} />
  ```

## 关键 API 适配 (vs plan 伪代码)

| Plan 描述 | 实际签名 | 处理 |
| --- | --- | --- |
| `import { hasPermission } from "@/utils/authHelpers"` | authHelpers **无此导出** (只有 getAccessToken/getAuthHeaders/refreshEncryptionConfig) | 复用 MACHistoryPage 已 established pattern: 从 `useMenuStore((s) => s.permissions)` 派生本地 `hasPermission` |
| `filterExternalOrgDepts(fullDeptTreeData as never)` | `filterExternalOrgDepts<T extends DeptLikeNode>(nodes: T[])` — `DeptTreeNode` (title/value/key + isExternalOrg?) 已满足 `DeptLikeNode` 约束 | 去掉 `as never`, 显式泛型 `filterExternalOrgDepts<DeptTreeNode>(...)` |
| `trimTitleToLastSegment(deptTreeData)` | `trimTitleToLastSegment<T extends { title?: string; children?: T[] }>` — DeptTreeNode 满足 | 中间加一层 cast (`as unknown as DeptTreeNode[]`),因 useDeptTree 返回 `DeptTreeNode[]` 来自 `SimpleDept` 形状,字段名兼容但 TS 严格模式下需显式断言 |
| `locationAliasApi.list/create/delete` | 与 Plan 39-05 产出完全对齐 (PageResponse<LocationAlias> / { deptId, locationId, scope?, remark? } / string) | 直接使用 |

## Verification

- `npm run type-check`: **EXIT=0** (TypeScript 严格模式 0 error)
- `npx eslint src/pages/operations/workstations/LocationAliasDrawer.tsx src/pages/operations/workstations/index.tsx`:
  - **0 errors**, 1 warning
  - 唯一 warning 在 index.tsx:540 `expandedRowRender` 是**预存在**(WorkstationDeviceTable 子表渲染),非本次改动引入, 依 SCOPE BOUNDARY 规则保留不动
- `npm run lint`(全仓): 3864 pre-existing problems,本次改动**未引入任何 NEW error**(对比 baseline 一致)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] hasPermission 不在 authHelpers 导出**
- **Found during:** Task 1 imports
- **Issue:** Plan 伪代码 `import { hasPermission } from "@/utils/authHelpers"` — 但 authHelpers 只导出 token/encryption 相关函数,无 hasPermission
- **Fix:** 复用 `pages/network/mac/history/MACHistoryPage.tsx:104-105` 已 established pattern: `useMenuStore((s) => s.permissions)` + 本地 `hasPermission = (perm) => permissions.includes(perm)`
- **Files modified:** LocationAliasDrawer.tsx, index.tsx (Task 2 复用同一 pattern)
- **Commit:** 313683e (Task 1), c4e8398 (Task 2)

**2. [Rule 1 - Bug] handleCreate values 隐式 any**
- **Found during:** Task 1 lint verification
- **Issue:** `const values = await form.validateFields()` 推导为 `any`,触发 5 个 `@typescript-eslint/no-unsafe-assignment` / `no-unsafe-member-access` warning
- **Fix:** 显式 cast `(await form.validateFields()) as { deptId: string; locationId: string; scope?: string; remark?: string }`,消除 5 个 warning
- **Files modified:** LocationAliasDrawer.tsx
- **Commit:** c4e8398 (随 Task 2 一并提交)

## Known Stubs

无。

## Threat Flags

无新增安全相关表面。复用现有 `ops:location:alias:*` 权限串(Plan 39-04 已注册)与 locationAliasApi(Plan 39-05 产出), 本 plan 仅消费已存在的 trust boundary, 不引入新的网络端点 / 鉴权路径 / 文件访问模式。

## Self-Check: PASSED

- FOUND: xingran-react-frontend/src/pages/operations/workstations/LocationAliasDrawer.tsx
- FOUND: xingran-react-frontend/src/pages/operations/workstations/index.tsx (modified)
- FOUND commit: 313683e (`feat(39-07): add LocationAliasDrawer`)
- FOUND commit: c4e8398 (`feat(39-07): inject [⚙ 映射] toolbar button`)
- npm run type-check EXIT=0 ✓
- npm run lint on modified files: 0 errors ✓
