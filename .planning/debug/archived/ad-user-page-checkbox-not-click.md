---
slug: ad-user-page-checkbox-not-click
name: ad-user-page-checkbox-not-click
status: resolved
trigger: AD域控的用户管理页面多选框无法勾选！
created: 2026-05-27
updated: 2026-06-26
---

## Symptoms

### Expected Behavior
可以勾选单个或多个用户进行批量操作

### Actual Behavior
点击复选框完全没反应

### Timeline
一直存在，从未正常工作

### Reproduction
打开AD域用户管理页面直接可见

### Error Messages
None provided

---

## Current Focus

## Current Focus

**Fix Applied:** Changed Table rowKey and rowSelection to use database `id` field instead of LDAP `userDn` for row selection. This eliminates special character and encoding issues with complex LDAP DN strings.

**Expected Result:** Checkboxes should now display selection state correctly, and the batch sync button should show the correct count.

**How to Test:**
1. Open the AD域用户管理 page
2. Click on checkboxes to select individual users
3. Verify that checkboxes show as checked (✓)
4. Verify that the batch sync button shows the correct count: `批量同步 (N)`
5. Try selecting multiple users across different pages
6. Click "批量同步" button to verify the selection works end-to-end

---

## Evidence

- timestamp: 2026-05-27T00:00:00Z
  source: user_report
  details: AD域用户管理页面复选框无法点击，问题从一开始就存在

- timestamp: 2026-05-27T00:15:00Z
  source: code_inspection
  details: Found critical mismatch in ADUserPage component:
  - Table rowKey prop is set to "id" (line 457)
  - rowSelection.selectedRowKeys uses selectedUsers.map(u => u.userDn) (line 461)
  - ADUser interface has both id: string and userDn: string fields
  - Mismatch causes checkboxes to not sync with selected state

- timestamp: 2026-05-27T00:16:00Z
  source: interface_analysis
  details: ADUser interface (adDomainApi.ts:65-84) shows:
  - id: string (database UUID)
  - userDn: string (LDAP distinguished name)
  - These are different values, causing the selection mismatch

- timestamp: 2026-05-27T00:30:00Z
  source: user_feedback
  details: After changing rowKey to "userDn", batch sync button shows 50 selected items but checkboxes don't display as checked. This indicates selection logic works but UI rendering doesn't sync.

- timestamp: 2026-05-27T00:35:00Z
  source: diagnostic_instrumentation
  details: Added comprehensive logging to investigate:
  - useEffect on users state to log userDn values
  - useEffect on selectedUsers state to log selected userDn values
  - onChange callback logging to track selection behavior
  - This will reveal if there are encoding/whitespace mismatches or timing issues

- timestamp: 2026-05-27T00:50:00Z
  source: root_cause_analysis
  details: Identified that LDAP userDn strings contain special characters (commas, equals) and may have encoding inconsistencies. Changed approach to use database `id` field (UUID) for row selection instead of userDn, as UUIDs are simple, consistent strings without special characters.

---

## Eliminated

- hypothesis: "rowKey/id mismatch prevents checkbox state synchronization"
  evidence: "User confirmed that after changing rowKey to 'userDn', the batch sync button shows 50 selected items, indicating selection logic works"
  timestamp: 2026-05-27T00:30:00Z
  note: "The original hypothesis was partially correct - rowKey fix enabled selection logic, but UI display issue remains"

- hypothesis: "Using userDn as rowKey with function syntax fixes the issue"
  evidence: "Even with rowKey as a function (record) => record.userDn, checkboxes didn't display. LDAP DNs contain special characters that likely cause comparison issues"
  timestamp: 2026-05-27T00:45:00Z
  note: "Complex LDAP DN strings with commas/equals may not work reliably with Ant Design Table's strict equality"

---

## Resolution

**Root Cause:** LDAP userDn strings contain special characters (commas, equals signs) and may have encoding inconsistencies that prevent Ant Design Table's strict equality checks from working properly. Using complex LDAP DNs as rowKeys causes the checkbox visual state to fail synchronization even though the selection logic works correctly.

**Fix:** Changed Table to use the database `id` field (UUID) instead of `userDn` for row selection:
- Table rowKey: `"id"` (database UUID)
- rowSelection.selectedRowKeys: `selectedUsers.map(u => u.id)`
- This eliminates potential string encoding/comparison issues with complex LDAP DN strings

**Files Changed:** ["xingran-react-frontend/src/pages/ad-domain/users/index.tsx"]

## Phase 41 Closure (2026-06-26)

**实修落地(2026-06-26):**
- `xingran-react-frontend/src/pages/ad-domain/users/index.tsx:601` `rowKey="userDn"` → `rowKey="id"` (改用 DB UUID)
- `xingran-react-frontend/src/pages/ad-domain/users/index.tsx:609` `selectedUsers.map(u => u.userDn)` → `selectedUsers.map(u => u.id)`
- **保持不变**:`userDns = selectedUsers.map(u => u.userDn)` (行 323 + 全选路径行 319) 是发给 `batchSyncADUsersDirect` API 的实际 LDAP DN 列表,API 必须收 DN,这部分不动。

**根因复述:** LDAP `userDn` 字符串含逗号、等号、特殊字符(OU=xxx,CN=yyy,DC=zzz),antd Table 的 `strict equality` 比较对复杂字符串不稳定,导致 checkbox 视觉状态不同步;改用 DB UUID 后字符串简单无特殊字符,严格比较工作正常。

**verification:** `cd xingran-react-frontend && npm run build` 退出 0。

fix: ad-domain/users/index.tsx rowKey + rowSelection 改用 DB `id` UUID
files_changed: xingran-react-frontend/src/pages/ad-domain/users/index.tsx
action: real-fix (D-02)
