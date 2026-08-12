---
status: partial
phase: 52-w3-router-handler-operlog-permission-migration
source: [52-VERIFICATION.md]
started: 2026-07-07T03:53:20Z
updated: 2026-07-07T03:53:20Z
---

## Current Test

[awaiting human testing on real PG dev DB + dev frontend]

## Tests

### 1. Apply migration_202 against real PG dev DB
**Description:** Bring up the docker-compose dev PG, start the backend, confirm `Migrate202PortWriteAudit` is invoked on startup.
**Expected:**
- `sys_port_write_audit` table exists with 12 columns + composite index `(device_id, port_id, created_at)` + single index on `created_at`
- `sys_menu` contains row `menu_name='端口配置'`, `menu_type='F'`, `perms='network:port:write'`, `visible=0`, `parent_id = (id of menu_name='端口状态' row)`
- `sys_role_menu` contains new rows for every role that previously held '端口状态' parent menu (precise, idempotent — `ON CONFLICT DO NOTHING`)

**result:** [pending]

### 2. Run auditConstraintNaming in dev DB
**Description:** Call `database.go:auditConstraintNaming(d.DB)` startup hook to confirm no constraint naming drift.
**Expected:** No `DROP` of any port_write_audit-related index/constraint. Audit passes silently (no FATA exit).

**result:** [pending]

### 3. End-to-end POST /network/ports/write/shutdown with real network:port:write permission
**Description:** Login as a user with the new permission; POST to `/network/ports/write/shutdown` with valid `portId`.
**Expected:**
- HTTP 200 + JSON `{code:0, data:{...}}` (response.Success)
- 1 new row in `sys_port_write_audit` (status='succeeded' or 'failed' or 'skipped' per PortResult)
- 1 new row in `sys_oper_log` with `module='端口管理'`, `oper_type=10 (OperTypeStatus)`, `oper_param` JSON containing `audit_ids:[<audit_id>]`
- Sentinel error path (e.g. unknown portId): HTTP 404, NO audit row, NO operlog row

**result:** [pending]

### 4. Frontend BulkWriteDrawer consumes sys_menu.perms='network:port:write'
**Description:** Phase 53 frontend reads menu permissions at build time; the new '端口配置' F-type row should make the 6 write buttons visible to roles already holding '端口状态' parent.
**Expected:** Buttons render in the port list page; clicking any triggers the corresponding POST /network/ports/write/<action> endpoint.

**result:** [pending]

## Summary

total: 4
passed: 0
issues: 0
pending: 4
skipped: 0
blocked: 0

## Gaps

None — all 4 items are out-of-scope for headless verification (require live PG/SSH/frontend). The single in-scope warning (WR-05: NetworkPortWrite not in GetRoutePermissions()) is classified as ACCEPTED_AS_KNOWN_LIMITATION with discoverability-only impact (router-level RequirePermissions still enforces the constant).
