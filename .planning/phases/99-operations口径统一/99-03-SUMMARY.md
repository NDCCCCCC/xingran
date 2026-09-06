---
phase: 99
phase_name: operations口径统一
plan: 99-03
subsystem: operations
tags: [refactor, orgId-filter, ancestor-matching, V130R-08]
dependency_graph:
  requires: []
  provides: []
  affects: [workstation_service, infopoint_service, building_service]
tech_stack:
  added: [dept_filter.go helper]
  patterns: [shared GORM scope helper for recursive dept filtering]
key_files:
  created:
    - internal/services/operations/dept_filter.go
    - internal/services/operations/dept_filter_test.go
  modified:
    - internal/services/operations/workstation_service.go
    - internal/services/operations/infopoint_service.go
decisions:
  - id: V130R-08-1
    decision: Extract shared BuildDeptRecursiveFilter helper to eliminate duplicated orgId filtering logic across building/workstation/infopoint services
  - id: V130R-08-2
    decision: Fix 3-condition to 4-condition ancestor matching pattern to correctly match deptId when it appears in the MIDDLE of the hierarchy path
metrics:
  duration_minutes: 15
  completed_date: 2026-09-06
  tasks_completed: 4
  files_changed: 4
  lines_added: ~300
---

# Phase 99 Plan 99-03: V130R-08 orgId 子部门筛选共享 helper

## One-liner
Extract shared `BuildDeptRecursiveFilter` GORM scope helper and fix 3-condition ancestor漏匹配 bug where deptId in middle of hierarchy path was missed.

## Bug Fixed

**Root Cause:** workstation_service and infopoint_service used a 3-condition SQL pattern:
```sql
(b.org_id = ? OR d.ancestors LIKE ? OR d.ancestors = ?)
```

When querying for a department that appears in the **middle** of an ancestor path (e.g., querying for `d2` where `d3.ancestors = "/d1/d2/d3/"`), the condition `d.ancestors LIKE '%,d2'` cannot match because `d2` is not at the **end** of the path.

**Fix:** 4-condition pattern:
```sql
(b.org_id = ? OR d.ancestors LIKE ? OR d.ancestors LIKE ? OR d.ancestors = ?)
-- Added: d.ancestors LIKE '%,<deptId>,%' to match middle positions
```

## Changes

### Created: `internal/services/operations/dept_filter.go`
Shared helper `BuildDeptRecursiveFilter(deptID, idColumn string) base.Scope` that returns a GORM scope for "department + all sub-departments" filtering with the correct 4-condition pattern.

### Modified: `internal/services/operations/workstation_service.go`
Fixed 4 occurrences of the orgId filter in:
- `Statistics()` method (line 80)
- `GetWorkstationDeptOptions()` subquery (line 118)
- `filterScope()` method (line 294)
- `SearchWorkstationOptions()` method (line 462)

### Modified: `internal/services/operations/infopoint_service.go`
Fixed orgId filter in `filterScope()` method (line 198-203).

## Tests Added

`internal/services/operations/dept_filter_test.go` with 6 passing tests:
- `TestBuildDeptRecursiveFilter_MiddleOfHierarchy` - Core bug: deptId at middle of path is matched
- `TestBuildDeptRecursiveFilter_RootDept` - Root dept (empty ancestors) is matched
- `TestBuildDeptRecursiveFilter_LeafDept` - Leaf dept (ancestors end with deptId) is matched
- `TestBuildDeptRecursiveFilter_EmptyDeptID` - Empty deptID returns full set (no filter)
- `TestWorkstationOrgFilter_MiddleAncestor` - Full integration: workstation under d3 (d3.ancestors contains d2) is found when querying for d2
- `TestWorkstationOrgFilter_BrokenThreeCondition` - Explicitly shows broken 3-condition returns 0, fixed 4-condition returns 1

## Deviations from Plan

None - plan executed exactly as written.

## Verification

```bash
go build ./internal/services/operations/...  # PASS
go test -run "TestBuildDeptRecursiveFilter|TestWorkstationOrgFilter" ./internal/services/operations/  # PASS (6 tests)
```

## Commit

- `f57f886` refactor(99-03): extract shared BuildDeptRecursiveFilter, fix ancestor middle漏匹配 (V130R-08)
