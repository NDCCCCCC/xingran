---
phase: "96"
verified: "2026-09-06T17:40:00Z"
status: passed
score: 6/6 must-haves verified
overrides_applied: 0
gaps: []
verification_fixed: "2026-09-06T17:43:00Z"
gap_fix:
  - file: "internal/services/monitor/cache_service_test.go"
    change: "TestCacheService_GetCacheInfo_PrefixNotStripped_IdentityQuirk → TestCacheService_GetCacheInfo_PrefixStripped; mock expects normalized key 'k'; commit acf962a"
    artifacts:
      - path: "internal/services/monitor/cache_service_test.go"
        issue: "TestCacheService_GetCacheInfo_PrefixNotStripped_IdentityQuirk (lines 559-570) locks in broken prefix-stripping as a 'Q1 quirk' — name and comment explicitly document buggy behavior (prefix never stripped, xingran:k passed unchanged to provider). After CACHEDEF-05 fixes normalizeCacheKeyForService to use strings.HasPrefix+TrimPrefix, the code normalizes xingran:k to k before calling the provider, but the mock expects the full xingran:k, causing panic."
    missing:
      - "Update mock expectation: provider.On(\"Get\", mock.Anything, \"k\") instead of provider.On(\"Get\", mock.Anything, \"xingran:k\")"
      - "Update assertion: info.Key will be \"k\" (normalized), not \"xingran:k\""
      - "Rename or update test comment — it no longer documents a 'quirk', it documents fixed behavior"
---

# Phase 96: 确定性缓存/看板缺陷修复 Verification Report

**Phase Goal:** 修复 5 项缓存确定性缺陷 + 删除 JOBSTAT-01 死代码，每项修复附回归测试
**Verified:** 2026-09-06T17:40:00Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|--------|--------|----------|
| 1 | Department cache invalidation truly hits the written key after CACHEDEF-01 fix | VERIFIED | department_cache_impl.go:76 uses BuildDeptCacheKey("tree:select"); invalidation pattern at line 99 uses BuildDeptCacheKey("tree:select")+"*" — keys match |
| 2 | Config single-delete invalidates both key:id and key:key caches after CACHEDEF-02 fix | VERIFIED | config_cache_impl.go:71 signature includes id parameter; keys list at line 75 includes fmt.Sprintf("config:id:%s", id) |
| 3 | Duty month parsing returns correct integers after CACHEDEF-03 fix | VERIFIED | duty_cache_impl.go:335-344 uses strconv.Atoi with fallback to 0; no len>=4 guard; 6 regression tests pass |
| 4 | Workorder pending cache keys include limit dimension after CACHEDEF-04 fix | VERIFIED | workorder_cache_impl.go:215 uses fmt.Sprintf with :limit:%d; nil req handled (limit=0); 4 regression tests pass |
| 5 | normalizeCacheKeyForService uses strings.HasPrefix + strings.TrimPrefix after CACHEDEF-05 fix | VERIFIED | cache_service.go:767-770 uses strings.HasPrefix + strings.TrimPrefix; key[:6] no longer present; TestNormalizeCacheKeyForService passes |
| 6 | job_utils.go deleted and preserved tests pass | VERIFIED | job_utils.go does not exist; no references to GetJobStatistics/FormatDuration in api_v1/; ws_notice and router test groups all pass (5 tests) |

**Score:** 5/6 truths verified; 1 truth FAILED (go test ./... not fully passing)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/services/system/department_cache_impl.go` | CACHEDEF-01 fixed | VERIFIED | Line 76: BuildDeptCacheKey("tree:select") |
| `internal/services/system/config_cache_impl.go` | CACHEDEF-02 fixed | VERIFIED | Lines 71-78: InvalidateConfigCache(ctx,id,configKey) with config:id: key |
| `internal/services/duty/duty_cache_impl.go` | CACHEDEF-03 fixed | VERIFIED | Lines 335-344: strconv.Atoi-based parseInt |
| `internal/services/workorder/workorder_cache_impl.go` | CACHEDEF-04 fixed | VERIFIED | Line 215: cache key includes :limit:%d |
| `internal/services/monitor/cache_service.go` | CACHEDEF-05 fixed | VERIFIED | Lines 767-770: strings.HasPrefix + strings.TrimPrefix |
| `internal/api/v1/job_utils.go` | Deleted | VERIFIED | File does not exist |
| `department_cache_impl_96_01_test.go` | Regression test | VERIFIED | 3 tests pass |
| `config_cache_impl_96_02_test.go` | Regression test | VERIFIED | 4 tests pass |
| `duty_cache_impl_96_03_test.go` | Regression test | VERIFIED | 6 tests pass |
| `workorder_cache_impl_96_04_test.go` | Regression test | VERIFIED | 4 tests pass |
| `cache_service_test.go` (existing) | Regression test | FAILED | TestCacheService_GetCacheInfo_PrefixNotStripped_IdentityQuirk breaks after CACHEDEF-05 |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|--------|
| department_cache_impl.go:76 | cache_keys.go:179 | BuildDeptCacheKey("tree:select") | VERIFIED | Write key matches invalidation pattern |
| config_cache_impl.go:71 | config_cache_impl.go:118 | InvalidateConfigCache signature + id param | VERIFIED | Signature updated and caller updated |
| duty_cache_impl.go:334 | duty_cache_impl.go:307 | parseInt + cache key format | VERIFIED | No len>=4 guard, cache keys match |
| workorder_cache_impl.go:215 | workorder_cache_impl.go:207 | :limit: dimension in key | VERIFIED | nil req handled, key includes limit |
| cache_service.go:766 | cache_service.go:295 | normalizeCacheKeyForService | VERIFIED | Correct prefix stripping |

### Regression Tests

| Test | Package | Result | Details |
|------|---------|--------|---------|
| TestDeptCacheImpl96_CACHEDEF01_* (3 tests) | services/system | PASS | |
| TestConfigCacheImpl96_CACHEDEF02_* (4 tests) | services/system | PASS | |
| TestDutyCacheImpl96_CACHEDEF03_* (6 tests) | services/duty | PASS | |
| TestWorkOrderCacheImpl96_CACHEDEF04_* (4 tests) | services/workorder | PASS | |
| TestNormalizeCacheKeyForService | services/monitor | PASS | |
| TestWs8003_* + TestRtr8003_* (5 tests) | api/v1 | PASS | Preserved ws_notice + router groups |
| TestCacheService_GetCacheInfo_PrefixNotStripped_IdentityQuirk | services/monitor | FAIL | Breaks after CACHEDEF-05 fix |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| go build ./... | `go build ./...` | 0 errors | PASS |
| CACHEDEF-01 test | `go test ./internal/services/system/ -run TestDeptCacheImpl96` | 3/3 pass | PASS |
| CACHEDEF-02 test | `go test ./internal/services/system/ -run TestConfigCacheImpl96` | 4/4 pass | PASS |
| CACHEDEF-03 test | `go test ./internal/services/duty/ -run TestDutyCacheImpl96` | 6/6 pass | PASS |
| CACHEDEF-04 test | `go test ./internal/services/workorder/ -run TestWorkOrderCacheImpl96` | 4/4 pass | PASS |
| CACHEDEF-05 test | `go test ./internal/services/monitor/ -run TestNormalizeCacheKeyForService` | PASS | PASS |
| JOBSTAT-01 preserved tests | `go test ./internal/api/v1/ -run TestWs8003\|TestRtr8003` | 5/5 pass | PASS |
| Pre-existing monitor test suite | `go test ./internal/services/monitor/ -run TestCacheService_` | 20 pass, 1 fail | FAIL |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| None in Phase 96 code | - | - | - | |

No debt markers (FIXME/TODO/TBD) or stub implementations found in Phase 96 modified files.

### Human Verification Required

None — all verifications are programmatic.

### Gaps Summary

**1 gap blocking goal achievement:**

**Pre-existing quirk lock test breaks after CACHEDEF-05 fix**

`TestCacheService_GetCacheInfo_PrefixNotStripped_IdentityQuirk` (cache_service_test.go:559-570) was written to document and lock in the broken `key[:6]` prefix-stripping behavior as a "Q1 quirk." The test's own comment explicitly states: "前缀永不被剥离 —— 'xingran:k' 原样传给 provider" (prefix is never stripped — "xingran:k" passed unchanged to provider).

After CACHEDEF-05 fixes `normalizeCacheKeyForService` to use `strings.HasPrefix`/`TrimPrefix`, the code now correctly normalizes `"xingran:k"` to `"k"` before calling the cache provider (cache_service.go:398). The test mock was set up expecting `"xingran:k"` to be passed to the provider, causing panic when the normalized `"k"` is passed instead.

**Fix required in cache_service_test.go:559-570:**

1. Change mock expectation: `provider.On("Get", mock.Anything, "k")` instead of `provider.On("Get", mock.Anything, "xingran:k")`
2. Change TTL mock: `provider.On("TTL", mock.Anything, "k")` instead of `provider.On("TTL", mock.Anything, "xingran:k")`
3. Update assertion: `assert.Equal(t, "k", info.Key, ...)` instead of `assert.Equal(t, "xingran:k", info.Key, ...)`
4. Rename test or update comment: it no longer documents a "quirk" — it now documents correct behavior

---

_Verified: 2026-09-06T17:40:00Z_
_Verifier: Claude (gsd-verifier)_
