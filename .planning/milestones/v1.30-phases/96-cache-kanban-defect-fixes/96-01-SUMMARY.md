---
phase: "96"
plan: "01"
type: execute
subsystem: "cache"
tags: ["cache", "defect-fix", "CACHEDEF-01", "CACHEDEF-02", "CACHEDEF-03", "CACHEDEF-04"]
dependency_graph:
  requires: []
  provides: ["CACHEDEF-01", "CACHEDEF-02", "CACHEDEF-03", "CACHEDEF-04"]
  affects: ["internal/services/system/department_cache_impl.go", "internal/services/system/config_cache_impl.go", "internal/services/duty/duty_cache_impl.go", "internal/services/workorder/workorder_cache_impl.go"]
tech_stack:
  added: []
  patterns: ["base.GetOrSetJSON", "base.Invalidate", "base.InvalidatePattern", "CacheProvider"]
key_files:
  created:
    - "internal/services/system/department_cache_impl_96_01_test.go"
    - "internal/services/system/config_cache_impl_96_02_test.go"
    - "internal/services/duty/duty_cache_impl_96_03_test.go"
    - "internal/services/workorder/workorder_cache_impl_96_04_test.go"
  modified:
    - "internal/services/system/department_cache_impl.go"
    - "internal/services/system/config_cache_impl.go"
    - "internal/services/duty/duty_cache_impl.go"
    - "internal/services/workorder/workorder_cache_impl.go"
    - "internal/services/system/config_cache_impl_test.go"
    - "internal/services/duty/duty_cache_impl_test.go"
decisions:
  - "CACHEDEF-01: GetSelectDataWithCache writes BuildDeptCacheKey(\"tree:select\") (\"cache:dept:tree:select\") instead of bare CacheKeyDeptTree (\"dept:tree\"). Matches invalidation pattern at line 99."
  - "CACHEDEF-02: InvalidateConfigCache signature changed to (ctx, id, configKey). keys list includes fmt.Sprintf(\"config:id:%s\", id) in addition to config:all and config:key:<key>. Delete caller updated."
  - "CACHEDEF-03: parseInt function rewritten using strconv.Atoi with fallback to 0. Removes len(s)>=4 guard that caused 2-digit months (\"07\") to return 0 instead of 7."
  - "CACHEDEF-04: GetMyPending cache key changed from workorder:my_pending:<userID> to workorder:my_pending:<userID>:limit:<N>. nil req handled (limit defaults to 0)."
metrics:
  duration_minutes: 3
  completed_date: "2026-09-06"
  tasks_completed: 4
  files_created: 4
  files_modified: 6
  commits: 6
---

# Phase 96 Plan 01: CACHEDEF-01..04 缓存缺陷修复 Summary

## One-liner

Fix 4 cache determinism defects: dept select cache write key prefix, config delete missing id-based key, duty month parseInt length guard, workorder pending cache key limit dimension.

## Completed Tasks

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | CACHEDEF-01: department GetSelectDataWithCache write key | d723570 | department_cache_impl.go, department_cache_impl_96_01_test.go |
| 2 | CACHEDEF-02: config InvalidateConfigCache id-based key | 3684924 | config_cache_impl.go, config_cache_impl_96_02_test.go, config_cache_impl_test.go |
| 3 | CACHEDEF-03: duty parseInt len>=4 guard removal | da1fcc3 | duty_cache_impl.go, duty_cache_impl_96_03_test.go, duty_cache_impl_test.go (update) |
| 4 | CACHEDEF-04: workorder pending cache key limit dimension | e679650 | workorder_cache_impl.go, workorder_cache_impl_96_04_test.go |

## Fixes Applied

### CACHEDEF-01 — Department Select Cache Write Key
**File:** `internal/services/system/department_cache_impl.go:76`

- **Before:** `CacheKeyDeptTree` (bare constant `"dept:tree"`)
- **After:** `BuildDeptCacheKey("tree:select")` (`"cache:dept:tree:select"`)
- **Effect:** Write key now matches invalidation pattern `BuildDeptCacheKey("tree:select")+"*"` at line 99. Invalidation hits the written key correctly.
- **Regression tests:** 3 (write→invalidate→read path, pattern covers write key, GetTreeWithFilter unaffected)

### CACHEDEF-02 — Config Delete Missing ID-Based Key
**File:** `internal/services/system/config_cache_impl.go:71-78,117`

- **Before:** `InvalidateConfigCache(ctx, configKey)` — only invalidated `config:all` and `config:key:<key>`
- **After:** `InvalidateConfigCache(ctx, id, configKey)` — adds `config:id:<id>` to keys list
- **Effect:** Single-item Delete now invalidates the `GetByID` cache key, preventing stale reads
- **Regression tests:** 4 (id key inclusion, Delete passes id, GetByID cache key format, signature verification)

### CACHEDEF-03 — Duty parseInt Length Guard
**File:** `internal/services/duty/duty_cache_impl.go:333-344`

- **Before:** `if len(s) >= 4` guard + 4-digit-only loop — `parseInt("07")` returned 0
- **After:** `strconv.Atoi` with fallback to 0 — parses all digit strings
- **Effect:** `parseInt("07")=7`, `parseInt("1")=1`, `parseInt("123")=123`. Cache write/invalidation keys match (`duty:monthly:2026:7`).
- **Existing test updates:** `TestDutyService_GenerateSchedule` (month key `2026:0`→`2026:8`), `TestDutyService_ManualDuty` (same), `TestDutyService_ParseInt` table (updated expected values to strconv.Atoi semantics)
- **Regression tests:** 6 (2-digit month, 4-digit year, single digit, empty string, 3-digit, cache key consistency)

### CACHEDEF-04 — Workorder Pending Cache Key Limit Dimension
**File:** `internal/services/workorder/workorder_cache_impl.go:210-214`

- **Before:** `cacheKey := fmt.Sprintf("workorder:my_pending:%s", userID)` — no limit in key
- **After:** `cacheKey := fmt.Sprintf("workorder:my_pending:%s:limit:%d", userID, limit)` — nil req handled (limit=0)
- **Effect:** Different limit values produce distinct keys (`limit:5` vs `limit:10`), eliminating cache pollution across different page sizes
- **Regression tests:** 4 (limit=10 key format, limit=20 key format, nil req key format, keys are distinct)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Existing duty test suite locked buggy parseInt behavior**
- **Found during:** Task 3 (CACHEDEF-03) verification
- **Issue:** `TestDutyService_ParseInt` table test expected the buggy behavior (`"08"`→0, `"123"`→0). `TestDutyService_GenerateSchedule` and `TestDutyService_ManualDuty` mock expectations used month key `2026:0` (the bug) instead of `2026:8` (correct).
- **Fix:** Updated all three test functions to use correct expected values matching `strconv.Atoi` semantics.
- **Files modified:** `duty_cache_impl_test.go`
- **Commit:** dab2ee5

**2. [Rule 1 - Bug] Existing config test called InvalidateConfigCache with old 1-arg signature**
- **Found during:** Task 2 (CACHEDEF-02) GREEN verification
- **Issue:** `config_cache_impl_test.go:150` called `InvalidateConfigCache(ctx, "sys.k1")` — 1 arg, but fix changed signature to 2 args `(ctx, id, configKey)`.
- **Fix:** Updated call to `InvalidateConfigCache(ctx, "test-id-123", "sys.k1")`.
- **Files modified:** `config_cache_impl_test.go`
- **Commit:** 3684924

## Threat Flags

None — all changes are internal cache key corrections within the application cache layer.

## Verification

| Check | Result |
|-------|--------|
| `go build ./...` | PASS (0 errors) |
| `go test ./internal/services/system/` | PASS |
| `go test ./internal/services/duty/` | PASS |
| `go test ./internal/services/workorder/` | PASS |
| CACHEDEF-01 test | 3/3 PASS |
| CACHEDEF-02 test | 4/4 PASS |
| CACHEDEF-03 test | 6/6 PASS |
| CACHEDEF-04 test | 4/4 PASS |

## Self-Check

- `internal/services/system/department_cache_impl.go` exists at d723570
- `internal/services/system/config_cache_impl.go` exists at 3684924
- `internal/services/duty/duty_cache_impl.go` exists at da1fcc3
- `internal/services/workorder/workorder_cache_impl.go` exists at e679650
- All 4 regression test files created
- All commits verified in git log

## Self-Check: PASSED
