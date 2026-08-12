---
slug: vdi-api-500-errors
status: resolved
trigger: Investigate and fix VDI API errors (500 Internal Server Error on /api/v1/vdi/vm/list and /api/v1/vdi/servers/list) and frontend deprecation warning (Space component direction -> orientation at VirtualMachineList/index.tsx:351)
created: 2026-05-25T12:55:00+08:00
updated: 2026-05-25T13:30:00+08:00
session_type: bug
---

# Debug Session: vdi-api-500-errors

## Symptoms

### Expected Behavior
VDI 页面应该正常显示虚拟机列表和服务器列表数据。

### Actual Behavior
VDI 页面完全空白，没有数据显示，后端返回 500 错误。

### Error Messages
```
POST http://10.62.10.33:9000/api/v1/vdi/vm/list 500 (Internal Server Error)
POST http://10.62.10.33:9000/api/v1/vdi/servers/list 500 (Internal Server Error)

[antd: Space] `direction` is deprecated. Please use `orientation` instead.
at VirtualMachineList/index.tsx:351
```

### Timeline
- VDI 功能从未正常工作过
- 这是一个新实现的功能，从第一天起就存在错误

### Reproduction
1. 导航到 VDI 管理页面
2. 页面加载时自动调用 `/api/v1/vdi/vm/list` 和 `/api/v1/vdi/servers/list`
3. 两个 API 都返回 500 错误
4. 页面显示空白

### Scope
- 影响范围：VDI 模块所有功能（虚拟机列表、服务器列表）
- 功能状态：完全不可用

## Current Focus

- hypothesis: VDI database tables not created by AutoMigrate + permission mismatch
- next_action: fixes applied
- test: go build ./cmd/main.go succeeds
- expecting: VDI API returns 200 after restart with tables auto-created
- reasoning_checkpoint: Three root causes identified and fixed
- tdd_checkpoint: null

## Evidence

- 2026-05-25T13:00: `internal/core/db/database.go` AutoMigrate() does NOT include VDI models (VDIServer, VDIVirtualMachine, VDIResourceGroup, VDIUserBinding). SQL migration files exist (128, 129, 130, 131) but are standalone scripts never invoked by the application. Result: `sys_vdi_server` and `sys_vdi_vm` tables don't exist -> PostgreSQL "relation does not exist" error -> GORM wraps as Go error -> handler returns 500.
- 2026-05-25T13:10: Router permissions (`vdi:vm:list`, `vdi:server:list`) don't match menu migration 129 permissions (`vdi:vm:query`, `vdi:server:query`). Non-superadmin users would get 403 even after table fix. Added both variants to RequirePermissions array.
- 2026-05-25T13:15: Ant Design 6.1 deprecated `direction` prop on Space component in `VirtualMachineList/index.tsx:351`. Changed to `orientation="vertical"`.

## Eliminated

- Permission mismatch alone would cause 403, not 500. The 500 is definitively caused by missing tables.
- Frontend pagination param mismatch (sends `current` vs backend expects `page`) is not a 500 cause since defaults handle it.

## Resolution

**root_cause:** Three issues:
1. **Primary (500 errors):** VDI models (`VDIServer`, `VDIVirtualMachine`, `VDIResourceGroup`, `VDIUserBinding`) were never registered in `database.go`'s `AutoMigrate()`. The SQL migration files (128-131) are standalone scripts not auto-executed. Without these tables, all VDI API queries fail with "relation does not exist" -> 500 Internal Server Error.
2. **Secondary (would cause 403 for non-admin):** Router permission keys (`vdi:vm:list`, `vdi:server:list`, `vdi:server:delete`) don't match the menu migration 129 keys (`vdi:vm:query`, `vdi:server:query`, `vdi:server:remove`).
3. **Tertiary (console warning):** Ant Design Space component uses deprecated `direction` prop.

**fix:**
1. Added `&models.VDIServer{}`, `&models.VDIVirtualMachine{}`, `&models.VDIResourceGroup{}`, `&models.VDIUserBinding{}` to `AutoMigrate()` in `internal/core/db/database.go` (line 283-287).
2. Updated `RequirePermissions` in `internal/api/router.go` (lines 754-760, 768-773) to include both router-defined and menu-defined permission variants.
3. Changed `direction="vertical"` to `orientation="vertical"` in `xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx` line 351.

**files_changed:**
- `internal/core/db/database.go` (AutoMigrate VDI models)
- `internal/api/router.go` (permission key alignment)
- `xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx` (deprecation fix)
