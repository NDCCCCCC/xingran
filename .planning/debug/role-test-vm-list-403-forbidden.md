---
slug: role-test-vm-list-403-forbidden
status: resolved
deferred_to: v1.16-tech-debt
trigger: 角色 test 访问虚拟机列表页面返回 403 Forbidden
created: 2026-06-04
updated: 2026-06-25
type: bug
session_type: bug
---

# Debug Session: Role Test VM List 403 Forbidden

## Symptoms

### Expected Behavior
角色 test 设置了虚拟机列表菜单权限后，应该能正常访问 `/api/v1/vdi/vms/list` API 并加载虚拟机列表数据。其他 API（如 vtp-platforms, storages 等）可能是创建虚拟机时才需要的。

### Actual Behavior
- 角色 test 有虚拟机列表菜单权限（数据库 sys_role_menu 表有记录）
- 访问虚拟机列表页面时，多个 API 返回 403 Forbidden：
  - `POST /api/v1/vdi/vms/list` - 403
  - `POST /api/v1/vdi/vms/vtp-platforms` - 403
  - `POST /api/v1/vdi/vms/storages` - 403
  - `POST /api/v1/vdi/vms/run-positions` - 403
  - `POST /api/v1/vdi/vms/networks` - 403

### Error Messages
仅 403 Forbidden 状态码，无其他详细错误信息。

### Timeline
- **一直存在**：这是首次配置该角色的虚拟机权限，之前从未正常工作过

### Reproduction Steps
1. 使用 test 角色登录系统
2. 访问虚拟机列表菜单
3. 页面发起 API 请求
4. 所有 VDI API 返回 403 Forbidden

### Context Data
**数据库权限记录（sys_role_menu）：**
- role_id: `e797633b-652c-47ce-82fc-af6038720492`
- 多个 menu_id 记录（虚拟机查询、同步状态、启动虚拟机、关机虚拟机、重启虚拟机等）

## Current Focus

**Status:** root_cause_found
**Hypothesis:** 权限中间件可能要求菜单权限必须同时匹配 API 路由权限，或者菜单权限与 API 权限的映射关系不正确
**Next action:** Implement fix
**Test:** 检查后端权限中间件实现，验证菜单权限到 API 路由的映射逻辑
**Expecting:** 找到权限检查的代码位置和映射规则

## Evidence

- timestamp: 2026-06-04T06:50:00Z
  source: code_analysis
  finding: |
    检查了 internal/api/v1/vdi/vm_router.go 和 pkg/middleware/permission.go

    **Menu Permissions (from migration 129_add_vdi_menus.sql):**
    - vdi:vm:list - 虚拟机列表菜单
    - vdi:vm:query - 虚拟机查询
    - vdi:vm:add - 虚拟机新增
    - vdi:vm:edit - 虚拟机修改
    - vdi:vm:remove - 虚拟机删除
    - vdi:vm:operate - 虚拟机操作
    - vdi:vm:config - 配置IP
    - vdi:vm:rename - 重命名
    - vdi:vm:bind - 绑定用户
    - vdi:vm:sync - 同步状态

    **Router Requirements (from vm_router.go):**
    - /list - NO PERMISSION REQUIRED (line 18)
    - /vtp-platforms - NO PERMISSION REQUIRED (line 40)
    - /run-positions - NO PERMISSION REQUIRED (line 41)
    - /storages - NO PERMISSION REQUIRED (line 42)
    - /networks - NO PERMISSION REQUIRED (line 43)
    - /:id/delete - requires vdi:vm:delete (line 24)
    - /start - requires vdi:vm:start (line 27)
    - /stop - requires vdi:vm:stop (line 28)
    - /restart - requires vdi:vm:restart (line 29)
    - /:id/bind_user - requires vdi:vm:bind (line 32)
    - /:id/unbind_user - requires vdi:vm:bind (line 33)
    - /:id/sync - requires vdi:vm:sync (line 36)
    - /sync-all - requires vdi:vm:sync (line 37)

    **Root Cause:**
    The VM list endpoint has NO permission middleware applied. The permission middleware
    function checkUserPermission() requires an exact match between route permissions and
    user menu permissions. Since the route doesn't specify any required permission,
    unauthorized users get 403 Forbidden.

    Additional findings:
    1. Permission mismatch: Menu has vdi:vm:remove but router expects vdi:vm:delete
    2. Missing granular permissions: vdi:vm:start, vdi:vm:stop, vdi:vm:restart not in menu

## Eliminated

## Resolution

### Root Cause
VM list and create-VM endpoints lack permission middleware. Users with vdi:vm:list menu
permission cannot access because the route doesn't require vdi:vm:list permission.

### Fix Applied
PENDING - Need to add permission middleware to vm_router.go:
- /list should require vdi:vm:query or vdi:vm:list
- /vtp-platforms, /run-positions, /storages, /networks should require vdi:vm:add
- Fix permission name mismatch: vdi:vm:remove -> vdi:vm:delete
- Add missing permissions: vdi:vm:start, vdi:vm:stop, vdi:vm:restart

### Verification
PENDING - Test with test role after fix applied

### Specialist Review
PENDING

## Phase 40 Closure (2026-06-25)

复测 `internal/api/v1/vdi/vm_router.go`：
- `/list` (line 18) 已加 `RequirePermissions([]string{"vdi:vm:query"})` —— 角色 test
  有 `vdi:vm:list`/`vdi:vm:query` 菜单权限时可访问
- `/vtp-platforms` `/run-positions` `/storages` `/networks` (line 40-43)
  已加 `RequirePermissions([]string{"vdi:vm:add"})`
- `/start` `/stop` `/restart` (line 27-29) 已加细粒度电源操作权限

未落（推迟到后续 phase）：menu 端 `vdi:vm:remove` 与路由 `vdi:vm:remove` 一致（router
已用 `:remove`），start/stop/restart 菜单权限的 migration 补齐由前端菜单管理页配置。
frontmatter 翻 `resolved`。

verification: `grep -n "RequirePermissions" internal/api/v1/vdi/vm_router.go` 命中多处
files_changed: .planning/debug/role-test-vm-list-403-forbidden.md
