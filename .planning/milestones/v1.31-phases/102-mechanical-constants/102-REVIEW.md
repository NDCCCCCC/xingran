---
status: clean
files_reviewed: 27
files_reviewed_list:
  - internal/api/v1/system/file_handler.go
  - internal/core/captcha.go
  - internal/core/captcha_background.go
  - internal/models/status_constants_test.go
  - internal/scheduler/cron.go
  - internal/scheduler/mac_history_matview_tasks.go
  - internal/scheduler/mac_history_tasks.go
  - internal/scheduler/reconciliation_tasks.go
  - internal/scheduler/vdi_sync_tasks.go
  - internal/scheduler/workorder_tasks.go
  - internal/services/api_endpoint_service.go
  - internal/services/duty/duty_cache_impl.go
  - internal/services/knowledge/knowledge_cache_impl.go
  - internal/services/mac_history_query_service.go
  - internal/services/network/cache_impl.go
  - internal/services/rpa/selector_learner.go
  - internal/services/scheduler/job_service.go
  - internal/services/system/cache_keys.go
  - internal/services/system/cache_keys_102_test.go
  - internal/services/system/notice_cache_impl.go
  - internal/services/system/settings_cache_impl.go
  - internal/services/system/widget_data_fetcher.go
  - internal/services/workorder/base.go
  - internal/services/workorder/workorder_cache_impl.go
  - internal/utils/utils_74_12_test.go
  - pkg/constants/cache.go
  - pkg/constants/cache_102_test.go
critical: 0
warning: 0
info: 3
total: 3
---

## Findings

### IN-001: Redundant int() type cast in vdi_sync_tasks.go

**File:** `internal/scheduler/vdi_sync_tasks.go:85`
**Summary:** `server.Status != int(models.VDIServerStatusNormal)` uses an unnecessary `int()` conversion. `models.VDIServerStatusNormal` is an untyped int constant; the field `server.Status` is `int`; the cast is redundant.

**Recommendation:** Simplify to `server.Status != models.VDIServerStatusNormal` (the comparison is type-safe since both are int). However, this is purely a style cleanup; the current code is functionally correct and does not affect behavior. Since this is a refactor-only phase with an invariant of "behavior equivalence," this is classified as Info — no action required.

---

### IN-002: utils_74_12_test.go — TestParsePaginationAndOffset confirmed absent

**File:** `internal/utils/utils_74_12_test.go`
**Summary:** Grep across the entire repository for `TestParsePaginationAndOffset` and `ParsePaginationAndOffset` returns zero source-code references outside planning artifacts. The remaining 9 tests in `utils_74_12_test.go` (template_engine 5 + response_builder 1 + string_helper 1 + db_helper 2) are independent of pagination. No orphan symbol references detected.

**Verification:** `grep -r "ParsePaginationAndOffset" --include="*.go" .` in non-planning paths yields no matches. The executor's claim that only `TestParsePaginationAndOffset` was removed is confirmed.

---

### IN-003: cache_keys_102_test.go — allowedKeyResidues starts empty (zero whitelist)

**File:** `internal/services/system/cache_keys_102_test.go:132`
**Summary:** `var allowedKeyResidues = map[string]int{}` — the scan guard starts with zero exempt entries as designed. The 12-file narrow scan (`keyLiteralFiles`, line 135) covers exactly the files touched by Phase 102 CACHE-02. The scan logic (3-pass: colon presence → segment regex `[a-z0-9_]*` → non-empty count) is correctly implemented and will hard-fail on any new inline cache key literal in those files.

**Observation:** `cache_keys_102_test.go` itself is not in `keyLiteralFiles` (it is the guard, not a guarded file), so the test file can use `fmt.Sprintf` for its own snapshot strings without triggering false positives.

---

## Structural Findings (fallow)

The following were confirmed present and correct in their respective files, consistent with the Phase 102 mechanical-constants audit ledger:

- `pkg/constants/cache.go`: 10 cache key format constants (6 captcha + 2 root service + 2 existing) all match their literal values from captcha.go/captcha_background.go
- `pkg/constants/cache_102_test.go`: `TestCaptchaCacheKeyEquivalence` (6 cases) + `TestRootServiceCacheKeyEquivalence` (2 cases) — snapshot columns are raw string literals, not `constants.Xxx` references (Pitfall 5 guard)
- `internal/models/status_constants_test.go`: `TestNoStatusLiteralUsage` AST scan with 7 patterns + `TestStatusConstantsStability` + `TestStatusConstantsCriticalFamilies`; `WorkOrderStatus*` registered (line 207-211)
- `internal/services/system/cache_keys.go`: All 10-module key registry constants in place, helper functions use `CacheKeyXxx` constants (no inline string literals)
- All 4 cache_impl files (duty, knowledge, network, workorder) and 3 system cache files (notice, settings, widget_data_fetcher) use `systemServices.CacheKeyXxx` / `systemServices.GetXxxKey` helpers exclusively
- `selector_learner.go:363` uses `systemServices.GetRpaSelectorBestKey` (confirmed via comment on line 362)
- `mac_history_query_service.go:257` uses `constants.MacVendorKeyFormat`
- `api_endpoint_service.go:64` uses `constants.UserEndpointsKeyFormat`
- `file_handler.go:163` uses `query.NormalizePaginationWithMax` with `constants.MaxListPageSize` (PAGI-01 convergence confirmed)
- `job_service.go:271-275` uses `constants.DefaultCurrent` / `constants.DefaultPageSize` (PAGI-01 convergence confirmed)
- `workorder/base.go:119` uses `queryutil.NormalizePagination` (PAGI-01 convergence confirmed)

---

## Summary

Phase 102 mechanical-constants refactor is clean. All four requirements (CACHE-01, CACHE-02, STATUS-01, PAGI-01) are correctly implemented:

- **CACHE-01**: 6 captcha key format constants registered in `pkg/constants/cache.go`; all 13 direct Sprintf call sites in captcha.go and 3 in captcha_background.go replaced with constants
- **CACHE-02**: 10-module key registry in `cache_keys.go` + 47 call-site replacements across 10 modules; AST scan guard `TestCacheKeyInlineResidue` active with 12-file narrow scope
- **STATUS-01**: Status literal AST scan `TestNoStatusLiteralUsage` active; `WorkOrderStatus*` constants registered in `status_constants_test.go`; scheduler files use `models.JobStatusNormal` etc. exclusively
- **PAGI-01**: `utils.ParsePagination` eliminated; all callers converge to `pkg/query.NormalizePagination` or `NormalizePaginationWithMax`

The only finding is a style-level Info (redundant `int()` cast) that does not affect runtime behavior. No behavior changes, no security regressions, no broken invariants.

---

_Reviewed: 2026-09-07T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
