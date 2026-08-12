---
status: partial
phase: 23-ad-group-sync
source: 23-06-SUMMARY.md, 23-08-SUMMARY.md, 23-09-SUMMARY.md, 23-10-SUMMARY.md, 23-11-SUMMARY.md, 23-12-SUMMARY.md, 23-13-SUMMARY.md, 23-14-SUMMARY.md, 23-16-SUMMARY.md
started: 2026-05-26T12:30:00Z
updated: 2026-07-08T00:00:00Z
closed_by_audit: .planning/reports/uat-audit-2026-07-08.md
---

## Current Test

[testing complete]

## Tests

### 1. Cold Start Smoke Test
expected: Start the application from scratch. Server boots without errors, database migrations complete, and the API health check returns success.
result: pass

### 2. View Department-Group Mappings
expected: Navigate to AD group mapping page. Table displays all department-to-group mappings with department name, AD group DN, mapping type, and status indicators (active/inactive, sync enabled/disabled).
result: issue
reported: "API调用返回404错误：POST /api/v1/ad-domain/configs/:id/sync-groups。前端路径与后端路由不匹配。已修复：更新adDomainApi.ts中的路径为 /ad-domain/groups/sync-by-config/:id"
severity: major

### 3. Create Department-Group Mapping
expected: Click "Create Mapping" button. Dialog opens with form fields for department selection, AD group DN, AD config, mapping type (auto/manual), and sync toggle. Submit creates new mapping and refreshes table.
result: deferred
reason: UAT 审计 2026-07-08 判定 build_needed,根因双重: (a) 前端 UI 已创建(23-16)但未集成到 App.tsx/Sidebar.tsx;(b) 依赖 T5 的 SM4 解密修复。v1.19 归档前主动 defer 到后续 AD 重构 phase,验收门槛 = 集成路由 + 菜单 seed + SM4 修复。

### 4. Auto-Map Departments
expected: Click "Auto-Map" button. System scans departments and AD groups, automatically creates mappings for groups matching "cxhub-{dept}" pattern (with "部" suffix removed), and shows count of mappings created.
result: deferred
reason: UAT 审计 2026-07-08 判定 build_needed,依赖 T3 映射管理页面,随之 defer。

### 5. Sync Department Members to AD Group
expected: Select a mapping and click "Sync Members". System displays sync status (in-progress), then shows result: total members, added count, removed count, and execution time.
result: deferred
reason: UAT 审计 2026-07-08 判定 third_party。SM4 解密失败 `cipher: message authentication failed`,根因详见 Gaps 段(encryptPassword() 在 cipher 为 nil 时返回明文,decryptPassword() 后续按 SM4-GCM 解密明文导致 auth 失败)。修复项列入 Gaps 但 v1.19 归档前不实施,active defer 到 AD 重构 phase。

### 6. Sync All Members for AD Configuration
expected: Navigate to sync monitoring page. Click "Sync All" button for an AD configuration. System syncs all active mappings sequentially and displays batch results with per-department details.
result: deferred
reason: UAT 审计 2026-07-08 判定 third_party,根因同 T5 SM4,active defer。

### 7. View Sync Logs
expected: Navigate to sync monitoring page. Table displays sync log entries with timestamp, department name, group name, members added/removed, status, and duration. Logs are filterable by department and date range.
result: deferred
reason: UAT 审计 2026-07-08 判定 build_needed。同步监控页面组件未创建,active defer。

### 8. Scheduled Sync Execution
expected: Wait for 15-minute scheduled sync (or trigger manually). Cron job executes sync for all active mappings, updates last_sync_at timestamps, and creates log entries. No errors in application logs.
result: deferred
reason: UAT 审计 2026-07-08 判定 third_party。Cron 受 SM4 失败影响,active defer。

### 9. MemberOUDN Configuration
expected: Navigate to AD configuration page. Edit a configuration and set "Member OU DN" field (e.g., "OU=本部部门分组,DC=example,DC=com"). Save succeeds and field persists to database.
result: deferred
reason: UAT 审计 2026-07-08 判定 third_party。AD 配置页面编辑受 SM4 影响,active defer。

### 10. Department Change Handling
expected: Change a user's department assignment in system. Sync detects change, removes user from old department's AD group, and adds to new department's AD group. Audit log records the change.
result: deferred
reason: UAT 审计 2026-07-08 判定 third_party。依赖同步功能,active defer。

## Summary

total: 10
passed: 1
issues: 1
pending: 0
skipped: 0
blocked: 0
deferred: 8
deferred_at: 2026-07-08
deferred_reason: 8 项 blocked 全部 defer 到后续 AD 重构 phase(原因为 SM4 解密失败 + 前端 UI 集成缺失);T2 issue 已修复但保留 issue 标记直至回归;详见 .planning/reports/uat-audit-2026-07-08.md

## Gaps

- truth: "Frontend API path matches backend route for group sync operations"
  status: fixed
  reason: "User reported: POST /api/v1/ad-domain/configs/:id/sync-groups returned 404. Frontend path did not match backend route after WR-06 fix."
  severity: major
  test: 2
  root_cause: "Frontend adDomainApi.ts was not updated when backend route was moved from configs to groups group in WR-06 fix"
  artifacts:
    - path: "xingran-react-frontend/src/lib/adDomainApi.ts"
      issue: "syncADGroups function uses wrong API path"
  missing:
    - "Update API path from /ad-domain/configs/:id/sync-groups to /ad-domain/groups/sync-by-config/:id"
  debug_session: ""

- truth: "AD configuration passwords can be decrypted with SM4"
  status: failed
  reason: "User reported: SM4 解密失败: cipher: message authentication failed. API returns 500 when attempting group sync."
  severity: blocker
  test: 5
  root_cause: "AD config passwords stored as plaintext or encrypted with different key/algorithm. decryptPassword() in utils.go silently fails and returns ciphertext instead of propagating error. When SM4 cipher was not initialized during password creation, encryptPassword() returned plaintext. Now decryptPassword() tries to decrypt plaintext as SM4-GCM, causing authentication failure."
  artifacts:
    - path: "internal/services/addomain/utils.go"
      issue: "decryptPassword() silently returns ciphertext on decryption failure (line 48-51)"
      issue: "encryptPassword() returns plaintext when cipher is nil (line 22-27)"
    - path: "internal/services/addomain/config.go"
      issue: "TestConnection and Sync operations call decryptPassword without error checking"
    - path: "pkg/crypto/sm4.go"
      issue: "SM4-GCM authentication fails when decrypting non-SM4 data"
  missing:
    - "Modify decryptPassword() to return (string, error) for proper error propagation"
    - "Add backward compatibility: try SM4 decryption, fall back to plaintext if fails"
    - "Add looksLikeSM4Ciphertext() helper to distinguish encrypted vs plaintext passwords"
    - "Update all callers (config.go, group_sync_service.go) to handle decryption errors"
    - "Data migration: detect and re-encrypt plaintext passwords in sys_ad_config table"
    - "Ensure SM4 cipher is initialized before any AD operations (SetADSM4Cipher called early)"
  debug_session: "SM4密码解密失败的详细分析已记录"

- truth: "Department-group mapping UI is accessible in the application"
  status: failed
  reason: "User reported: '没有创建映射按钮'. Frontend component exists (23-16) but is not integrated into App.tsx routes or Sidebar.tsx menu."
  severity: major
  test: 3
  root_cause: "Missing database menu entry in sys_menu table. Component exists at xingran-react-frontend/src/pages/ad/GroupMapping/index.tsx and backend API exists at /api/v1/ad/groups/mappings, but there's no corresponding menu entry to make it accessible through navigation. Migration 133 created tables but didn't add menu entries."
  artifacts:
    - path: "xingran-react-frontend/src/pages/ad/GroupMapping/index.tsx"
      issue: "Complete functional component exists but not accessible"
    - path: "xingran-react-frontend/src/App.tsx"
      issue: "No route registered for group mapping page"
    - path: "xingran-react-frontend/src/components/layout/Sidebar.tsx"
      issue: "No menu item - renders from menuStore.allMenus which comes from sys_menu table"
    - path: "internal/api/v1/addomain/group_sync_handler.go"
      issue: "Complete handler implementation exists"
    - path: "internal/api/v1/addomain/group_sync_router.go"
      issue: "Router properly configured but may not be registered in main router"
    - path: "sys_menu table"
      issue: "Missing menu entry for '部门-组映射' under 'AD域管理' parent"
  missing:
    - "Create database migration (e.g., 135_add_group_mapping_menu.sql) following migration 017 pattern"
    - "Add menu item: '部门-组映射' with path='group-mapping', component='/ad-domain/group-mapping', menu_type='C'"
    - "Add button permissions: ops:ad:group:mapping:add, ops:ad:group:mapping:edit, ops:ad:group:mapping:delete, ops:ad:group:mapping:view"
    - "Assign permissions to enabled roles (similar to migration 017)"
    - "Verify SetupGroupSyncRouter is called in internal/api/router.go"
    - "Add icon (NodeIndexOutlined or PartitionOutlined) and order_num=4 (after '用户组管理')"
  debug_session: "前端UI集成缺失的详细分析已记录"
