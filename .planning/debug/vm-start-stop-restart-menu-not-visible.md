---
slug: vm-start-stop-restart-menu-not-visible
status: resolved
created: 2026-06-03
updated: 2026-06-03
trigger: 虚拟机相关的启动(vdi:vm:start)、关机(vdi:vm:stop)、重启(vdi:vm:restart)三个菜单项在数据库sys_menu表中存在且visible=1,但在菜单管理和角色管理界面中没有显示这些选项。请调查为什么这些菜单项没有在前端界面显示。
type: bug
tdd_checkpoint:
reasoning_checkpoint:
---

# Debug Session: VM Start/Stop/Restart Menu Items Not Visible

## Current Focus

**Hypothesis:** UUID collision in migration 144 prevented menu creation
**Next Action:** fix applied
**Test:** Verify menus appear after restart
**Expecting:** Menus visible in menu management and role permission assignment
**Reasoning Checkpoint:** Confirmed
**TDD Checkpoint:** N/A

## Symptoms

**Expected Behavior:**
在数据库中新增虚拟机启动、关机、重启菜单项后，应该能在「系统管理 > 菜单管理」和「系统管理 > 角色管理 > 权限配置」界面中看到并配置这些菜单项。

**Actual Behavior:**
这三个菜单项（启动: vdi:vm:start, 关机: vdi:vm:stop, 重启: vdi:vm:restart）在数据库 `sys_menu` 表中存在且 `visible=1`，但在菜单管理和角色管理界面中都看不到这些选项。

**Error Messages:**
无错误消息，界面正常显示其他菜单项，唯独这三个新增的菜单项不显示。

**Timeline:**
新增后从未显示 - 这三个菜单项是最近新增的（2026-06-03 04:52:34 创建），添加后在界面中一直没有显示。

**Reproduction:**
1. 在数据库中新增三个菜单项：
   - 启动虚拟机 (id: b9062e5e-ea98-434d-a05b-d51f4d80abe2, perms: vdi:vm:start, order_num: 20)
   - 关机虚拟机 (id: 2d67bf01-a731-4b2d-8894-2a561a1e531b, perms: vdi:vm:stop, order_num: 21)
   - 重启虚拟机 (id: 1631d86a-eb04-43b0-80ed-0453a3c74253, perms: vdi:vm:restart, order_num: 22)
2. 打开「系统管理 > 菜单管理」界面
3. 查看虚拟机列表菜单 (id: 770e8400-e29b-41d4-a716-446655440002) 下的子菜单
4. 观察到其他菜单项（查询、新增、修改、删除等）正常显示，但启动/关机/重启三个菜单项不显示
5. 打开「系统管理 > 角色管理」，选择角色配置权限
6. 在虚拟机相关权限中同样看不到这三个新增的权限项

**Additional Context:**
- 其他虚拟机菜单项（新增、修改、删除、绑定用户等）正常显示
- 受影响界面：菜单管理和角色权限配置界面都看不到
- 数据库状态：记录存在，visible=1, status=0（正常）

## Evidence

- 2026-06-03: Migration 129 (129_add_vdi_menus.sql) created VDI server config menus with IDs 770e8400-...-446655440020 through 023 (perms: vdi:server:add, vdi:server:edit, vdi:server:remove, vdi:server:test), parent: 770e8400-...-446655440018
- 2026-06-03: Migration 144 (migration_144_vdi_granular_permissions.go) attempted to create VM operation menus using the SAME IDs 020-023 for different menus (perms: vdi:vm:start, vdi:vm:stop, vdi:vm:restart, vdi:vm:delete)
- 2026-06-03: Migration 144's `SELECT 1 FROM sys_menu WHERE id = ?` check found existing records (the VDI server config menus) and skipped all inserts
- 2026-06-03: Confirmed via grep that migration 129 and 130 both use IDs 020-023 for VDI server config buttons
- 2026-06-03: Migration 144 also ignored the OrderNum field from its struct, hardcoding 0 in the INSERT VALUES clause
- 2026-06-03: Go build passes after fix

## Eliminated

- Frontend filtering: The menu management page calls `/system/menus/tree` which returns all menus without filtering by visible or menu_type
- Cache issue: Both cached and non-cached GetTree implementations return all menus with status=0
- Parent ID issue: The manually inserted menus have correct parent_id pointing to 770e8400-...-446655440002
- deleted_at issue: GORM soft delete filters are standard and the manually inserted records have deleted_at=NULL
- Role permissions issue: The menus would not appear in role management even without role_menu entries because the tree-select endpoint shows all menus

## Resolution

**Root Cause:** UUID collision between migration 129 and migration 144. Migration 129 (129_add_vdi_menus.sql) first created VDI server config button menus using IDs ending in 020-023 (770e8400-e29b-41d4-a716-446655440020 through 023). Migration 144 (migration_144_vdi_granular_permissions.go) later attempted to create VM operation menus using the exact same UUIDs. Since migration 144 checked for existing records by ID before inserting (`SELECT 1 FROM sys_menu WHERE id = ?`), it found the VDI server config menus already occupying those IDs and silently skipped all inserts. The vdi:vm:start, vdi:vm:stop, vdi:vm:restart, and vdi:vm:delete menus were never created.

**Fix:** Updated migration 144 to use non-colliding UUIDs (770e8400-e29b-41d4-a716-446655440024-027 instead of 020-023). Added idempotency checks: the migration now checks by perms column first (handles manually inserted records), falls back to gen_random_uuid() if the target ID is occupied, and also added a top-level check to skip entirely if all 4 menus already exist by perms. Also fixed the INSERT to use the OrderNum from the struct instead of hardcoded 0.

**Verification:** Restart the backend server. After migration runs, check that menus with perms vdi:vm:start, vdi:vm:stop, vdi:vm:restart, vdi:vm:delete exist in sys_menu. Verify they appear in menu management and role permission assignment UIs. If manually inserted records still exist, they may need to be cleaned up (delete the old manual records or they will coexist).

**Files Changed:**
- `internal/core/db/migrations/migration_144_vdi_granular_permissions.go` - Fixed UUID collision, added idempotency, added proper OrderNum
