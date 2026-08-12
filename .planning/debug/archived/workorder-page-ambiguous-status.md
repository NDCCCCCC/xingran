---
slug: workorder-page-ambiguous-status
status: resolved
trigger: "Work order management page triggers 400 Bad Request on /api/v1/system/users/list due to ambiguous 'status' column"
created: 2026-06-17
updated: 2026-06-17
---

## Current Focus

hypothesis: CONFIRMED & FIXED - unqualified `status` in Where() collides with sys_dept.status after LEFT JOIN
test: New regression test TestUserList_StatusNotAmbiguous_AfterJoinSysDept passes with fix, fails without
expecting: Bug fixed, no other places with same pattern
next_action: Move to resolved/ and report

## Symptoms

expected: User list endpoint returns paginated users successfully
actual: 400 Bad Request - "column reference 'status' is ambiguous (SQLSTATE 42702)"
errors: column reference "status" is ambiguous (SQLSTATE 42702) — ambiguous column name: status (SQLite)
reproduction: Open 工单管理 page → triggers POST /api/v1/system/users/list with status=0
started: Pre-existing bug (introduced when List was refactored to LEFT JOIN sys_dept)

## Eliminated

- hypothesis: `dept_id = ?` at line 311 also ambiguous
  evidence: sys_dept has no `dept_id` column, only `id`. PostgreSQL accepts the unqualified reference.
  timestamp: 2026-06-17

## Evidence

- timestamp: 2026-06-17
  checked: internal/services/system/user_service.go line 305-307
  found: `if params.Status != nil { query = query.Where("status = ?", *params.Status) }` - unqualified status
  implication: After LEFT JOIN sys_dept is added (line 336/342), the query has both sys_user.status and sys_dept.status

- timestamp: 2026-06-17
  checked: Backend error log SQL
  found: `SELECT sys_user.*, sys_dept.dept_name, sys_dept.ancestors FROM "sys_user" LEFT JOIN sys_dept ON sys_dept.id = sys_user.dept_id WHERE status = 0`
  implication: SQLSTATE 42702 raised by PostgreSQL

- timestamp: 2026-06-17
  checked: Confirmed no other services with same bug pattern
  found: All other List queries that join tables with status (workstation/floor/serverroom/infopoint/room_device) use qualified column names like `sys_workstation.status`, `ops_floors.status`, etc.
  implication: Bug is isolated to user_service.go

- timestamp: 2026-06-17
  checked: SQLite test mimicking PostgreSQL behavior
  found: Same error "ambiguous column name: status" with unqualified status
  implication: SQLite serves as good proxy for the PostgreSQL bug

- timestamp: 2026-06-17
  checked: Reverting fix → test fails; re-applying → test passes
  found: Regression test TestUserList_StatusNotAmbiguous_AfterJoinSysDept reliably catches the bug
  implication: Test is sufficient as regression guard

## Resolution

root_cause: `internal/services/system/user_service.go:306` — `query.Where("status = ?", ...)` uses unqualified `status` column, which becomes ambiguous after the LEFT JOIN sys_dept at line 342 (since both sys_user and sys_dept have a `status` column). PostgreSQL raises SQLSTATE 42702.
fix: Change `query.Where("status = ?", *params.Status)` to `query.Where("sys_user.status = ?", *params.Status)` with explanatory comment.
verification:
  - `go build ./...` → success (no compile errors)
  - `go test ./internal/services/system/... -run TestUserList_StatusNotAmbiguous_AfterJoinSysDept` → PASS
  - Reverting the fix and re-running the test → FAIL with "ambiguous column name: status" (exact bug reproduction)
  - All other user-related tests still pass (pre-existing TestListAPIKeys failure is unrelated — uses LEFT() unsupported in SQLite)
files_changed:
  - internal/services/system/user_service.go (1 line of business logic + 4 lines of comment)
  - internal/services/system/user_list_status_test.go (new regression test file)
