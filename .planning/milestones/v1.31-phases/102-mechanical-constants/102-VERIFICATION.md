---
status: passed
phase: "102"
phase_name: mechanical-constants
verified_at: "2026-09-07T00:00:00Z"
requirements:
  CACHE-01: passed
  CACHE-02: passed
  STATUS-01: passed
  PAGI-01: passed
must_haves_verified: 4/4
---

## Goal Achievement Summary

Phase 102 mechanical-constants successfully delivers all 4 requirements. Captcha cache keys (CACHE-01) are fully constantized with 6 format constants and 13+ call-site replacements verified. CACHE-02 registers 68 key constants + 38 helper functions across 10 modules with AST scan guard active (12-file narrow scan, 0 violations). STATUS-01 replaces 21+ status literals with models constants and extends AST usage-point guard with whitelist. PAGI-01 eliminates utils.ParsePagination, migrating to NormalizePaginationWithMax and deleting the entire pagination.go file.

## Per-Requirement Evidence

### CACHE-01: passed

- **Constants registered**: `pkg/constants/cache.go` contains 6 captcha key format constants (CaptchaRateLimitKeyFormat, CaptchaDataKeyFormat, CaptchaAttemptsKeyFormat, LoginFailKeyFormat, CaptchaBgListKeyFormat, CaptchaCachePoolPrefixFormat) — verified by Read
- **Call-site replacement**: `internal/core/captcha.go` uses constants on lines 249, 297, 304, 326, 333, 356, 361, 367, 415, 419, 426, 503, 529 — 13 direct Sprintf sites verified with grep
- **captcha_background.go**: bg:list and cache:pool formats use constants (grep confirmed no inline literals)
- **Test**: `go test ./pkg/constants/ -run TestCaptchaCacheKeyEquivalence -count=1` — **ok** (6 case snapshot test)
- **Inline residue check**: `grep "captcha:rate|captcha:data|captcha:attempts|login:fail"` in production code (internal/core/*.go excluding tests) — **0 matches**
- **Deviation from plan**: None. All 16 direct sites replaced.

### CACHE-02: passed

- **Constants registered**: `internal/services/system/cache_keys.go` contains **68 prefix constants** and **38 GetXxxKey/GetXxxPattern helper functions** (counted via grep)
- **Root package formats**: `pkg/constants/cache.go` contains UserEndpointsKeyFormat + MacVendorKeyFormat (lines 41, 44)
- **Call-site replacement**: 10 modules verified (notice, settings, duty, workorder, knowledge, network, widget, rpa selector, api_endpoint, mac_vendor)
- **TestCacheKeyEquivalence**: `go test ./internal/services/system/ -run TestCacheKeyEquivalence -count=1` — **ok** (20 constant snapshots + 27 helper output assertions)
- **TestCacheKeyInlineResidue**: `go test ./internal/services/system/ -run TestCacheKeyInlineResidue -count=1` — **ok** (12-file narrow scan, 0 violations)
- **Root service test**: `go test ./pkg/constants/ -count=1` — **ok** (includes TestRootServiceCacheKeyEquivalence 2 cases)
- **D-102-1 revision compliance**: api_endpoint_service.go and mac_history_query_service.go (root package) correctly use `pkg/constants` — import cycle constraint respected
- **No independent pattern constants**: `grep 'CacheKey\w+Pattern\s*=' cache_keys.go` — 0 matches (D-102-4 compliance)
- **Deviation from plan**: None. 47 call sites replaced, GetNoticeMyNoticesPattern helper added per Rule 2.

### STATUS-01: passed

- **Replacement evidence** (grep verified constants used in cron.go):
  - Line 43: `int(models.JobLogStatusSuccess)`
  - Line 62: `int(models.JobLogStatusFailure)`
  - Line 235: `models.JobStatusNormal`
  - Line 407: `models.JobStatusNormal`
  - Line 435: `models.JobStatusPause`
- **WorkOrderStatus value lock**: `status_constants_test.go` line 207-211 registers all 5 values (Pending=0, Processing=1, Completed=2, Closed=3, Rejected=4); "WorkOrderStatus" added to watchedStatusPrefixes (line 75)
- **TestNoStatusLiteralUsage**: `go test ./internal/models/ -run TestNoStatusLiteralUsage -count=1` — **ok** (7 AST patterns, full backend scan)
- **TestStatusConstants**: `go test ./internal/models/ -run TestStatusConstants -count=1` — **ok**
- **Whitelist active**: `statusLiteralWhitelist` map at line 385 with geocoding_service.go whitelisted (line 386)
- **Inline status literals check**: `grep "Status:\s+0|Status:\s+1"` in internal/scheduler/*.go — **0 matches** in production code
- **Deviation from plan**: None. 21 sites + 2 newly exposed (base.go:191, reconciliation_tasks MisfirePolicy:1) all replaced.

### PAGI-01: passed

- **file_handler.go:163**: Uses `query.NormalizePaginationWithMax(req.Page, req.PageSize, constants.MaxListPageSize)` — verified by grep
- **Pagination.go deleted**: `ls internal/utils/pagination.go` — **FILE_DELETED** (confirmed)
- **TestParsePaginationAndOffset**: `grep -c "TestParsePaginationAndOffset" internal/utils/utils_74_12_test.go` — **0** (function deleted, remaining 9 tests preserved)
- **utils import preserved**: file_handler.go still uses `utils.BuildListResponse` / `utils.GetUserID` etc. — import not removed
- **Cap=100 comment**: present in file_handler.go (D-102-9 compliance)
- **Regression tests**: `go test ./internal/api/v1/system/ -run TestFile -count=1` — **ok** (0.199s)
- **Full suite**: `go build ./...` — **BUILD OK**
- **Deviation from plan**: None.

## Invariant Compliance

- **D-102-1**: api_endpoint_service.go + mac_history_query_service.go (root package) use pkg/constants — import cycle constraint respected. No system/cache_keys.go references from root package files.
- **D-102-5② (cache-key scan)**: TestCacheKeyInlineResidue active with 12-file narrow scope, allowedKeyResidues starts empty (IN-003 confirmed), scan logic 3-pass correct.
- **D-102-6 (status scan)**: TestNoStatusLiteralUsage active with 7 AST patterns, statusLiteralWhitelist populated (geocoding + 10 sqlite entries), migrations excluded.
- **D-03-3 (Phase 99 inheritance)**: utils/pagination.go entirely deleted. BuildPaginationResponse not referenced from deleted file. TestParsePaginationAndOffset removed.
- **Zero new models.XxxStatus constants**: STATUS-01 replaces with existing constants only (JobStatusNormal/Pause, JobLogStatusSuccess/Failure, DutyStatusNormal, VDIServerStatusNormal, WorkOrderStatus*). No new constants added.
- **Behavior equivalence (key string + TTL)**: TestCaptchaCacheKeyEquivalence and TestCacheKeyEquivalence use raw string literals as expected values (not constants — anti-Pitfall-5 guard). TTL lines unchanged (grep confirmed).
- **No backward-compat shim for utils.ParsePagination**: Entire file deleted, no stub remains.
- **No TBD/FIXME/XXX markers**: `grep -r "TBD|FIXME|XXX" .planning/phases/102-mechanical-constants/` — 0 matches.

## Gaps

[None]

## Human Verification Items

[None — this is a mechanical refactor with automated test guards. All verification is programmatic: AST scans, equivalence snapshots, regression tests, and build verification.]

## Full Test Run

```
go test ./pkg/constants/                           ok  0.595s
go test ./internal/services/system/                ok  4.120s
go test ./internal/models/                        ok  1.827s
go test ./internal/core/                          ok  149.843s
go test ./internal/scheduler/                     ok  8.045s
go test ./internal/api/v1/system/                 ok  0.199s
go build ./...                                    ok
```

## Review Findings

Code review (102-REVIEW.md) confirms: **0 Critical, 0 Warning, 3 Info** (IN-001 redundant int() cast in vdi_sync_tasks.go:85 — style only, no behavior impact; IN-002 TestParsePaginationAndOffset confirmed absent; IN-003 allowedKeyResidues empty as designed).

---

_Verified: 2026-09-07_
_Verifier: Claude (gsd-verifier)_
