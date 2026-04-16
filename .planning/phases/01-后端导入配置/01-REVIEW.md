---
phase: 01-后端导入配置
reviewed: 2026-04-16T12:00:00Z
depth: standard
files_reviewed: 1
files_reviewed_list:
  - internal/services/operations/excel_config.go
findings:
  critical: 1
  warning: 1
  info: 2
  total: 4
status: issues_found
---

# Phase 01: Code Review Report

**Reviewed:** 2026-04-16T12:00:00Z
**Depth:** standard
**Files Reviewed:** 1
**Status:** issues_found

## Summary

Reviewed the workstation Excel import configuration changes that add `deptName` (department) and `userName` (user) reference columns to the workstation import config. One critical bug found: the user reference uses a wrong database column name (`user_name` instead of `username`), which will cause a runtime SQL error during import. One warning about non-unique department name lookups potentially returning incorrect IDs.

## Critical Issues

### CR-01: Wrong column name in sys_user reference -- will cause SQL error at runtime

**File:** `internal/services/operations/excel_config.go:123`
**Issue:** The `userName` column config uses `Reference: "sys_user.user_name"`, but the actual database column for the User model's `Username` field is `username` (not `user_name`). GORM's default snake_case convention converts the Go field `Username` to `username` as a single word, confirmed by existing queries in `internal/core/db/database.go:601` and `database.go:684` which use `WHERE username = ?`.

When the reference resolver executes the batch query, it will produce SQL like:
```sql
SELECT id, user_name as value FROM sys_user WHERE user_name IN (...) AND deleted_at IS NULL
```
This will fail with a PostgreSQL error: `column "user_name" does not exist`.

**Fix:**
```go
// Line 123: Change "sys_user.user_name" to "sys_user.username"
{Field: "userName", Header: "所属用户", Reference: "sys_user.username", DBField: "user_id"},
```

## Warnings

### WR-01: Non-unique dept_name lookup may silently assign wrong department

**File:** `internal/services/operations/excel_config.go:122`
**Issue:** The `deptName` column references `sys_dept.dept_name` for name-to-ID resolution. However, `dept_name` is not a unique column in `sys_dept` (only `dept_code` has a `uniqueIndex`). If two departments share the same `dept_name`, the `batchQueryIDs` method in `reference_resolver.go` will return an arbitrary one (the last row returned by the database), silently assigning the wrong department to the workstation.

The `idMap[r.Value] = r.ID` assignment at `reference_resolver.go:272` overwrites previous entries for the same value, so only one ID survives per department name.

**Fix:** Consider using `dept_code` as the reference field instead, since it has a unique index. If department name matching is required by business logic, add uniqueness validation or at least a warning when duplicates are detected:

Option A (preferred, uses existing unique field):
```go
{Field: "deptCode", Header: "所属部门编码", Reference: "sys_dept.dept_code", DBField: "dept_id"},
```

Option B (keep name-based but document the risk):
```go
{Field: "deptName", Header: "所属部门", Reference: "sys_dept.dept_name", DBField: "dept_id"},
// Note: If multiple departments share the same name, the import will pick one arbitrarily.
// Consider using dept_code for guaranteed uniqueness.
```

## Info

### IN-01: DeptName and UserName display fields not populated during import

**File:** `internal/services/operations/excel_config.go:122-123`
**Issue:** The Workstation model has both ID and name fields (`DeptID` + `DeptName`, `UserID` + `UserName`). The import config maps references to `DBField: "dept_id"` and `DBField: "user_id"` respectively, but does not populate the corresponding name fields (`DeptName`, `UserName`). After import, `DeptName` and `UserName` will be null/empty in the database, which may cause display issues in the UI that reads these denormalized fields.

This is consistent with how `buildingName` and `floorName` references work in the same config (they also only set `building_id` and `floor_id`), so it follows the existing pattern. If other code paths (e.g., manual workstation creation) populate the name fields, there could be inconsistency.

**Fix:** This matches the existing pattern for `buildingName`/`floorName`, so no immediate change needed. However, consider a post-import enrichment step that backfills `DeptName` and `UserName` from the resolved IDs, or update the list/detail APIs to resolve these names dynamically.

### IN-02: New fields are optional but header does not indicate this to users

**File:** `internal/services/operations/excel_config.go:122-123`
**Issue:** The `deptName` and `userName` columns are both non-required (`Required: false` by default), which is correct for backward compatibility. However, the generated Excel template headers will just show "所属部门" and "所属用户" without indicating these are optional fields. Users may fill them in expecting required behavior, or leave them blank without understanding their purpose.

**Fix:** Consider updating the Header text to clarify optionality, e.g., `"所属部门（选填）"` and `"所属用户（选填）"`, or add Instructions entries to the workstation config explaining the new optional columns.

---

_Reviewed: 2026-04-16T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
