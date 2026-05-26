---
status: partial
phase: 23-ad-group-sync
source: 23-06-SUMMARY.md, 23-08-SUMMARY.md, 23-09-SUMMARY.md, 23-10-SUMMARY.md, 23-11-SUMMARY.md, 23-12-SUMMARY.md, 23-13-SUMMARY.md, 23-14-SUMMARY.md, 23-16-SUMMARY.md
started: 2026-05-26T12:30:00Z
updated: 2026-05-26T12:45:00Z
---

## Current Test

[testing paused — 8 items outstanding]

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
result: blocked
blocked_by: release-build
reason: "前端UI组件已创建（23-16-SUMMARY.md）但未集成到应用路由中。用户报告：'没有创建映射按钮'。需要：1) 添加路由到App.tsx 2) 添加菜单项到Sidebar.tsx"

### 4. Auto-Map Departments
expected: Click "Auto-Map" button. System scans departments and AD groups, automatically creates mappings for groups matching "cxhub-{dept}" pattern (with "部" suffix removed), and shows count of mappings created.
result: blocked
blocked_by: release-build
reason: "前端UI未集成，依赖测试3的映射管理页面"

### 5. Sync Department Members to AD Group
expected: Select a mapping and click "Sync Members". System displays sync status (in-progress), then shows result: total members, added count, removed count, and execution time.
result: blocked
blocked_by: third-party
reason: "SM4解密失败：'cipher: message authentication failed'。AD配置中的密码无法解密，导致同步API返回500错误。需要：1) 重新加密AD密码 2) 或修复SM4密钥配置"

### 6. Sync All Members for AD Configuration
expected: Navigate to sync monitoring page. Click "Sync All" button for an AD configuration. System syncs all active mappings sequentially and displays batch results with per-department details.
result: blocked
blocked_by: third-party
reason: "与测试5相同的SM4解密问题"

### 7. View Sync Logs
expected: Navigate to sync monitoring page. Table displays sync log entries with timestamp, department name, group name, members added/removed, status, and duration. Logs are filterable by department and date range.
result: blocked
blocked_by: release-build
reason: "前端UI未集成，同步监控页面组件未创建"

### 8. Scheduled Sync Execution
expected: Wait for 15-minute scheduled sync (or trigger manually). Cron job executes sync for all active mappings, updates last_sync_at timestamps, and creates log entries. No errors in application logs.
result: blocked
blocked_by: third-party
reason: "与测试5相同的SM4解密问题，定时同步也会失败"

### 9. MemberOUDN Configuration
expected: Navigate to AD configuration page. Edit a configuration and set "Member OU DN" field (e.g., "OU=本部部门分组,DC=example,DC=com"). Save succeeds and field persists to database.
result: blocked
blocked_by: third-party
reason: "AD配置页面受SM4解密问题影响，编辑功能可能无法正常工作"

### 10. Department Change Handling
expected: Change a user's department assignment in system. Sync detects change, removes user from old department's AD group, and adds to new department's AD group. Audit log records the change.
result: blocked
blocked_by: third-party
reason: "依赖同步功能，受SM4解密问题影响"

## Summary

total: 10
passed: 1
issues: 2
pending: 0
skipped: 0
blocked: 7

## Gaps

- truth: "Frontend API path matches backend route for group sync operations"
  status: fixed
  reason: "User reported: POST /api/v1/ad-domain/configs/:id/sync-groups returned 404. Frontend path did not match backend route after WR-06 fix."
  severity: major
  test: 2
  root_cause: "Frontend adDomainApi.ts was not updated when backend route was moved from configs to groups group in WR-06 fix"
  artifacts:
    - path: "ruoyi-react-frontend/src/lib/adDomainApi.ts"
      issue: "syncADGroups function uses wrong API path"
  missing:
    - "Update API path from /ad-domain/configs/:id/sync-groups to /ad-domain/groups/sync-by-config/:id"
  debug_session: ""

- truth: "AD configuration passwords can be decrypted with SM4"
  status: failed
  reason: "User reported: SM4 解密失败: cipher: message authentication failed. API returns 500 when attempting group sync."
  severity: blocker
  test: 5
  root_cause: ""
  artifacts: []
  missing: []
  debug_session: ""

- truth: "Department-group mapping UI is accessible in the application"
  status: failed
  reason: "User reported: '没有创建映射按钮'. Frontend component exists (23-16) but is not integrated into App.tsx routes or Sidebar.tsx menu."
  severity: major
  test: 3
  root_cause: "Frontend integration incomplete - component created but not wired into application"
  artifacts:
    - path: "ruoyi-react-frontend/src/pages/ad/GroupMapping/index.tsx"
      issue: "Component exists but no route registered"
    - path: "ruoyi-react-frontend/src/App.tsx"
      issue: "Missing route for /ad/group-mapping"
    - path: "ruoyi-react-frontend/src/components/layout/Sidebar.tsx"
      issue: "Missing menu item for AD group mapping"
  missing:
    - "Add route to App.tsx for GroupMapping page"
    - "Add menu item to Sidebar.tsx with permission check"
    - "Import GroupMapping component in App.tsx"
  debug_session: ""
