---
phase: 01-后端导入配置
verified: 2026-04-16T17:30:00Z
status: human_needed
score: 4/4 must-haves verified
overrides_applied: 0
human_verification:
  - test: "Import a workstation Excel file containing department names and usernames, then query sys_workstation table to verify dept_id and user_id are populated with correct UUIDs"
    expected: "dept_id and user_id fields contain valid UUIDs matching the department and user records"
    why_human: "Requires running the application and database — cannot test data insertion programmatically without a live PostgreSQL instance"
  - test: "Import a workstation Excel file with a non-existent department name or username"
    expected: "Import succeeds, dept_id or user_id is empty/NULL for that row, no error raised"
    why_human: "Requires running the application and triggering the import flow — cannot verify runtime error handling statically"
---

# Phase 1: 后端导入配置 Verification Report

**Phase Goal:** 在 Excel 导入配置中添加部门和用户列，实现名称到 ID 自动匹配
**Verified:** 2026-04-16T17:30:00Z
**Status:** human_needed
**Re-verification:** No -- initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | workstation Excel import config contains "所属部门" column (IMPORT-01) | VERIFIED | Line 122: `{Field: "deptName", Header: "所属部门", Reference: "sys_dept.dept_name", DBField: "dept_id"}` |
| 2 | workstation Excel import config contains "所属用户" column (IMPORT-02) | VERIFIED | Line 123: `{Field: "userName", Header: "所属用户", Reference: "sys_user.username", DBField: "user_id"}` |
| 3 | Both fields optional, match failure does not block import (IMPORT-03) | VERIFIED | Neither column has `Required: true`; `validateReferenceFields` (line 712) skips non-required columns |
| 4 | Build passes, no new errors | VERIFIED | `go build ./...` exits with code 0 |

**Score:** 4/4 truths verified

### ROADMAP Success Criteria Assessment

| # | Criterion | Status | Evidence |
|---|-----------|--------|----------|
| 1 | Import Excel with dept/user names -> `sys_workstation.dept_id`/`user_id` store correct UUIDs | ? NEEDS HUMAN | Config is correct; data flow traced through `applyReferenceResults` -> `prepareRecordsForUpsert` -> `BatchUpsert`, but end-to-end requires live DB |
| 2 | Import Excel without dept/user info -> fields empty, no error | VERIFIED | No `Required: true` on either column; `extractReferenceRequests` skips empty values (line 265 `continue`); `validateReferenceFields` skips non-required refs (line 712) |
| 3 | Non-existent dept/user name -> field left empty, import continues | VERIFIED | `applyReferenceResults` line 398: on resolve failure, only logs debug message; `prepareRecordsForUpsert` line 656-657: if resolved field not in record, `exists=false`, field skipped |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/services/operations/excel_config.go` | workstation config with dept/user columns | VERIFIED | Lines 122-123 contain both columns; build passes |
| `internal/models/workstation.go` | DeptID and UserID fields on model | VERIFIED | `DeptID *string` (line 47), `UserID *string` (line 58) |
| `internal/services/operations/reference_resolver.go` | Generic table.field resolution | VERIFIED | `parseReference` (line 217) splits on "."; `batchQueryIDs` queries arbitrary table+field |
| `internal/services/operations/excel_service.go` | Reference application and validation logic | VERIFIED | `applyReferenceResults` handles missing refs gracefully; `validateReferenceFields` skips non-required |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| excel_config.go deptName | sys_dept table | Reference: "sys_dept.dept_name" | WIRED | ReferenceResolver queries `SELECT id FROM sys_dept WHERE dept_name = ?` |
| excel_config.go userName | sys_user table | Reference: "sys_user.username" | WIRED | ReferenceResolver queries `SELECT id FROM sys_user WHERE username = ?` |
| ReferenceResolver | sys_workstation table | DBField: "dept_id" / "user_id" | WIRED | `prepareRecordsForUpsert` maps resolved UUIDs to DBField names |
| excel_config.go | excel_service.go | Config lookup via `ExcelConfigs["workstation"]` | WIRED | `GetExcelConfig` returns config; service iterates `config.Columns` |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| excel_config.go (deptName column) | `Reference: "sys_dept.dept_name"` | ReferenceResolver.batchQueryIDs | Yes -- queries sys_dept table for UUID by dept_name | FLOWING |
| excel_config.go (userName column) | `Reference: "sys_user.username"` | ReferenceResolver.batchQueryIDs | Yes -- queries sys_user table for UUID by username | FLOWING |
| excel_service.go (applyReferenceResults) | `data["dept_id"]`, `data["user_id"]` | refResults map from resolver | Yes -- writes resolved UUID to data map, deletes original name field | FLOWING |
| excel_service.go (prepareRecordsForUpsert) | `preparedRecord["dept_id"]`, `preparedRecord["user_id"]` | record map with resolved fields | Yes -- reads resolved field, writes to preparedRecord via DBField | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Go build succeeds | `go build ./...` | Exit code 0, no errors | PASS |
| deptName column config present | `grep -c "deptName.*所属部门.*sys_dept" excel_config.go` | Match found (line 122) | PASS |
| userName column config present | `grep -c "userName.*所属用户.*sys_user" excel_config.go` | Match found (line 123) | PASS |
| Neither field marked Required | Visual inspection of lines 122-123 | No `Required: true` on either | PASS |
| sys_user.username reference correct | Cross-check with models/user.go | `Username` field -> GORM maps to `username` column in `sys_user` table | PASS |
| sys_dept.dept_name reference correct | Cross-check with models/dept.go | `DeptName` field -> GORM maps to `dept_name` column in `sys_dept` table | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| IMPORT-01 | Plan 01 | Excel import config add "所属部门" column via sys_dept.dept_name -> dept_id | SATISFIED | Line 122 in excel_config.go |
| IMPORT-02 | Plan 01 | Excel import config add "所属用户" column via sys_user.username -> user_id | SATISFIED | Line 123 in excel_config.go |
| IMPORT-03 | Plan 01 | Both fields optional, match failure leaves empty, no blocking | SATISFIED | No Required flag; validateReferenceFields skips non-required; applyReferenceResults only logs on failure |

No orphaned requirements -- REQUIREMENTS.md maps IMPORT-01/02/03 to Phase 1, all three covered by Plan 01.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (none) | - | - | - | - |

No TODO/FIXME/placeholder/stub patterns found in modified file.

### Deviation from Plan

The PLAN specified `Reference: "sys_user.user_name"` but the actual code uses `Reference: "sys_user.username"`. This is a **correct deviation** -- the Go model field `Username` maps to database column `username` (single word), not `user_name`. The original plan had an incorrect column name. Commit `30e4795` explicitly fixed this from `sys_user.user_name` to `sys_user.username`. This deviation improves correctness.

### Human Verification Required

### 1. End-to-end import with valid dept/user names

**Test:** Create a workstation Excel file with known department names and usernames from `sys_dept` and `sys_user`. Upload via the import endpoint.
**Expected:** After import, query `sys_workstation` and verify `dept_id` contains the correct UUID from `sys_dept` and `user_id` contains the correct UUID from `sys_user`.
**Why human:** Requires running application with live PostgreSQL and Redis. Cannot verify database write results statically.

### 2. Import with non-existent dept/user names

**Test:** Create a workstation Excel file with department names that do not exist in `sys_dept` and usernames not in `sys_user`.
**Expected:** Import succeeds without error. Records are created with `dept_id` and `user_id` as NULL. No import failure recorded for these rows.
**Why human:** Requires running application to trigger the full import flow and verify runtime error handling behavior.

### Gaps Summary

No code gaps found. All four must-haves are verified through static analysis:

1. **IMPORT-01 SATISFIED:** The "所属部门" column is configured at line 122 with correct `Reference: "sys_dept.dept_name"` and `DBField: "dept_id"`.
2. **IMPORT-02 SATISFIED:** The "所属用户" column is configured at line 123 with correct `Reference: "sys_user.username"` and `DBField: "user_id"`.
3. **IMPORT-03 SATISFIED:** Both fields lack `Required: true`. The `validateReferenceFields` function explicitly skips non-required reference fields (line 712). The `applyReferenceResults` function only logs a debug message on resolve failure (line 398).
4. **Build passes:** `go build ./...` succeeds with exit code 0.

The data flow is complete from Excel column definition -> ReferenceResolver query -> resolved UUID written to data map -> preparedRecord via DBField -> BatchUpsert to database.

The status is `human_needed` because ROADMAP success criterion #1 (actual UUID storage in database after import) can only be verified with a running application and database instance. The static analysis confirms all wiring is correct and the code should work as intended.

---

_Verified: 2026-04-16T17:30:00Z_
_Verifier: Claude (gsd-verifier)_
