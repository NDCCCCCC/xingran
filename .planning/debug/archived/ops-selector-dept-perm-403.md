---
slug: ops-selector-dept-perm-403
status: resolved
trigger: "空间管理/运维管理所有页面几乎都需要部门管理权限，部门树/用户列表/机房列表接口全部 403"
created: 2026-06-17
updated: 2026-06-17
related: space-mgmt-role-no-perm.md (已解决, 同类权限命名空间割裂)
root_cause: "跨模块选择器复用导致权限命名空间割裂: 运维页面内嵌的 <DeptTree>/用户选择器/机房列表复用 system 模块的 /departments/tree、/users/list 及 ops/serverRoom/list, 但运维/空间角色不持有 system:dept / system:user / ops:serverroom 权限"
---

## Resolution

fix: |
  用 RequirePermissionsWithQuery 改造三个路由组 (沿用上次楼宇空间修复的同一中间件):
  - router.go `/system/departments` 组: 只读路径(/tree,/list,/tree-select)额外接受 opsSelectorReadPerms
    (ops:{building,floor,workstation,serverroom}:list + ops:building:spaces:list); 写操作严格。
    选 opsSelectorReadPerms (全运维读权限) 而非仅 spaces, 是因为部门树在**每个**运维页面都出现 (用户原话"所有页面都需要部门权限")。
  - router.go `/system/users` 组: 只读路径额外接受 [ops:workstation:list, ops:building:spaces:list]
    (仅工位页用户选择器需要, 范围收窄避免不必要的用户数据暴露; 用户列表仍受 DataScopePermission 限制)。
  - router.go `/ops/serverRoom` 组: 只读路径额外接受 [ops:building:spaces:list]
    (空间角色访问机房管理页时需要机房列表)。
  - 新增包级 var `opsSelectorReadPerms` 复用。
  安全性: 部门树 GetTree 不应用数据权限故放宽门后正常返回全树(组织结构低敏感);
  所有写操作 (create/update/delete/batch) 末段非 list/tree, 仍走严格权限 → 无越权。
verification: |
  - go build ./... 通过 (exit 0)
  - go test ./pkg/middleware/ 通过
  - 新增 TestRequirePermissionsWithQuery_SelectorEndpoints (6 子测试): 空间角色读 部门树/用户列表/机房列表=200,
    部门/用户/机房写操作=403(防越权)。复用 newTestCoreWithSQLite + grantUserRoleMenu 夹具。
files_changed:
  - internal/api/router.go (新增 opsSelectorReadPerms var; users/departments/serverRoom 三组 RequirePermissions → RequirePermissionsWithQuery)
  - pkg/middleware/permission_query_test.go (新增跨模块选择器回归测试 + runQueryPermCasePerms 通用夹具)
deploy_note: 仅后端改动, 重启 xingran-backend.exe 生效, 无前端改动/重构建。


# Debug Session: 空间管理所有页面部门树 403

## Symptoms

浏览器控制台在 楼宇管理/楼层管理/工位管理/机房管理 多个页面上报:

- `POST /api/v1/system/departments/tree` → **403** (每个页面都报, 因都渲染 `<DeptTree>`/`<DeptSidebar>`)
  - 调用源: `index.tsx:78` (DeptTree) ← `useDepartmentData.ts:29`
- `POST /api/v1/system/users/list` → **403** (工位页用户选择器)
  - 调用源: `useWorkstationData.ts:133`
- `POST /api/v1/ops/serverRoom/list` → **403** (机房页表格+统计)
  - 调用源: `statisticsHelper.ts:22`, `ServerRoomManagement/index.tsx`
- 对照: `POST /api/v1/ops/building/list` → **200** (上次修复 RequirePermissionsWithQuery 已生效)

前端 `errorHandler.ts:184` 把 403 渲染为「没有权限访问」/「加载部门树失败」。

## Root Cause (确认)

**跨模块选择器复用导致权限命名空间割裂 — 上次同类 bug 的复发, 但这次是 ops 页面 → system 接口。**

运维/空间管理页面内嵌的部门树选择器 `<DeptTree>`(侧边栏 `DeptSidebar`)、工位用户选择器、机房列表,
都调用了 **system 模块** 的 list/tree 接口, 而这些接口由路由组级 `RequirePermissions` 严格保护:

| 接口 | 路由组 | 组级权限要求 (OR) | 用户角色实际持有 |
|------|--------|-------------------|------------------|
| `/system/departments/tree` | router.go:197 `/departments` | `system:dept:{list,add,edit,view}` | ❌ 无 (空间角色仅 ops:building:spaces:*) |
| `/system/users/list` | router.go:144 `/users` | `system:user:{list,add,edit,view}` | ❌ 无 |
| `/ops/serverRoom/list` | router.go:665 `/serverRoom` | `ops:serverroom:{list,add,edit,delete}` | ❌ 无; 且该组未迁移到 RequirePermissionsWithQuery |

「空间管理」角色勾全楼宇空间菜单 → 持有 `ops:building:spaces:list` 等 → 与 `system:dept:*` / `system:user:*` / `ops:serverroom:*` **零交集** → 三类选择器接口全部 403。

### 关键证据

- `pkg/permission/config.go`: `UserList="system:user:list"`, `DeptList="system:dept:list"` 等
- `internal/api/router.go:144-150` users 组 `RequirePermissions([UserList,UserAdd,UserEdit,UserView])`
- `internal/api/router.go:197-203` departments 组 `RequirePermissions([DeptList,DeptAdd,DeptEdit,DeptView])`
- `internal/api/router.go:665-671` serverRoom 组 `RequirePermissions([ops:serverroom:list/add/edit/delete])` (注意小写 `serverroom`)
- `internal/api/v1/system/department_handler.go:56-69` `GetTree` 调 `GetTreeWithFilter`, **不应用 DataScope** → 放宽权限门后部门树能正常返回全树 (不会因数据权限为空)
- 上次修复 `RequirePermissionsWithQuery` (提交 5e8bac1) 已让 `/ops/{building,floor,workstation}/list` 对空间角色放行 → building/list 200 证实有效

## Fix Options

**A (推荐, 与上次修复一致)**: 用 `RequirePermissionsWithQuery` 改造 `/system/departments`、`/system/users`、`/ops/serverRoom` 三个路由组。
对只读路径 (/tree, /list, /tree-select) 额外接受 operations 读权限
(`ops:building:list`/`ops:floor:list`/`ops:workstation:list`/`ops:serverroom:list`/`ops:building:spaces:list`);
写操作 (create/update/delete/batch) 保持严格。复用上次验证过的中间件, 无前端改动。
- 风险: 低; dept tree 不应用数据权限故能正常返回; user list 受 DataScope 限制(符合数据权限语义)

**B (最简, 较宽)**: 移除 dept `/tree`、user `/list` 读路径的 RequirePermissions 门, 改为所有登录用户可读, 依赖 DataScope 限制可见数据。serverRoom 迁移到 RequirePermissionsWithQuery。
- 风险: 中; 任何登录用户可读组织树/用户列表 (信息面较广)

**C (最干净, 工作量大)**: 新增聚合选择器接口 `/system/selector/depts/tree`、`/system/selector/users/list`, 前端 DeptTree 改调新接口。
- 工作量: 新 handler+service+router + 前端改动

**D (零代码, 配置)**: 给空间管理角色补 `system:dept:list`、`system:user:list`、`ops:serverroom:list` 权限。
- 优点: 立即可用; 缺点: 治标, 每个空间角色都要配
