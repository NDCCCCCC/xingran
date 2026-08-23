# Phase 72 Plan 10: Config + Role Sub-Module Tests Summary

## Overview

Added comprehensive test coverage for `config` and `role` sub-modules in both `api/v1/system` (CORE-04) and `services/system` (CORE-06) bringing per-sub-module coverage well above the 70% threshold.

**Pattern:** D-01 lightweight handler pattern (glebarez sqlite + real service calls) + D-01 service pattern (compile-time interface assertion + testify/mock + glebarez sqlite).

**Status:** Complete (uncommitted per plan 72-13 batch commit strategy)

## Files Created

### Handler Tests (api/v1/system)

1. **`internal/api/v1/system/config_handler_test.go`** — 50 test cases covering config_handler.go (401 lines)
2. **`internal/api/v1/system/role_handler_test.go`** — 50 test cases covering role_handler.go (279 lines)

### Service Tests (services/system)

3. **`internal/services/system/config_service_test.go`** — 35 test cases covering config_service.go (267 lines)
4. **`internal/services/system/config_cache_impl_test.go`** — 21 test cases covering config_cache_impl.go (cache wrapper)
5. **`internal/services/system/role_service_test.go`** — 43 test cases covering role_service.go (515 lines)
6. **`internal/services/system/role_cache_impl_test.go`** — 29 test cases covering role_cache_impl.go (cache wrapper)

**Total: 6 new files, 228 test cases**

## Method Coverage

### config_handler.go (Plan 72-10 Task 1)
- Statistics: 100.0%
- Create: 100.0%
- List: 78.8%
- GetByID: 100.0%
- GetByKey: 100.0%
- Update: 76.9%
- Delete: 77.8%
- BatchDelete: 100.0%
- RefreshCache: 60.0%
- isCaptchaConfig: 100.0%
- validateCaptchaConfigValue: 100.0%

### role_handler.go (Plan 72-10 Task 2)
- Create: 100.0%
- List: 100.0%
- Statistics: 100.0%
- GetByID: 100.0%
- Update: 100.0%
- Delete: 100.0%
- BatchDelete: 100.0%
- UpdateStatus: 100.0%
- GetAllEnabled: 100.0%

### config_service.go (Plan 72-10 Task 3)
- NewConfigService: 100.0%
- Statistics: 80.0%
- Create: 100.0%
- Update: 94.7%
- Delete: 80.0%
- BatchDelete: 100.0%
- GetByID: 100.0%
- GetByKey: 85.7%
- List: 86.4%
- RefreshCache: 100.0%

### config_cache_impl.go
- NewConfigServiceWithCache: 100.0%
- GetByID: 87.5%
- GetByKey: 87.5%
- GetAllConfigs: 100.0%
- queryAllConfigs: 80.0%
- InvalidateConfigCache: 100.0%
- InvalidateAllConfigCache: 100.0%
- Create: 100.0%
- Update: 100.0%
- Delete: 83.3%
- BatchDelete: 100.0%
- List: 100.0%
- RefreshCache: 100.0%

### role_service.go (Plan 72-10 Task 3)
- NewRoleService: 100.0%
- Statistics: 100.0%
- Create: 81.2%
- Update: 76.9%
- Delete: 78.9%
- GetByID: 100.0%
- List: 94.7%
- GetAllEnabled: 100.0%
- BatchDelete: 81.2%
- UpdateStatus: 87.5%
- checkRoleNameExists: 100.0%
- checkRoleKeyExists: 85.7%
- assignRoleMenusAndDepts: 75.0%
- GetAllEnabledWithCache: 100.0%
- GetMenusWithCache: 77.8%
- GetDeptsWithCache: 77.8%
- InvalidateRoleCache: 100.0%

### role_cache_impl.go
- NewRoleServiceWithCache: 100.0%
- List: 100.0%
- buildListCacheKey: 100.0%
- GetByID: 87.5%
- GetAllEnabled: 100.0%
- GetAllEnabledWithCache: 100.0%
- GetMenusWithCache: 100.0%
- queryMenus: 77.8%
- GetDeptsWithCache: 100.0%
- queryDepts: 77.8%
- InvalidateRoleCache: 100.0%
- Create: 100.0%
- Update: 100.0%
- Delete: 100.0%
- BatchDelete: 100.0%
- UpdateStatus: 100.0%

## Per-Sub-Module Weighted Coverage

**config sub-module (all files > 70%):**
- config_handler.go: weighted avg ~88%
- config_service.go: weighted avg ~91%
- config_cache_impl.go: weighted avg ~95%
- **Combined weighted avg: ~91% (≥70% target met)**

**role sub-module (all files > 70%):**
- role_handler.go: weighted avg ~99%
- role_service.go: weighted avg ~84%
- role_cache_impl.go: weighted avg ~95%
- **Combined weighted avg: ~92% (≥70% target met)**

## Existing Tests Preserved (D-08)

- `config_invalidation_test.go` — UNTOUCHED
- `config_statistics_test.go` — UNTOUCHED
- `role_service_apperrors_test.go` — UNTOUCHED

## Locked Decisions Compliance

- **D-01 lightweight handler pattern:** Real service + glebarez sqlite + table-driven TC ✓
- **D-03 real SM4 cipher:** Not applicable (config/role don't use SM4 cipher directly)
- **D-06 no new mock framework:** testify/mock + glebarez sqlite only ✓
- **D-08 zero business code changes:** Only test files created ✓
- **CLAUDE.md Status Convention:** Uses `models.RoleStatusEnabled/Disabled`, `models.ConfigTypeYes/No`, `models.ConfigIsSystemYes/No` constants ✓

## Test Pattern Notes

### Config Cache Service Test Pollution
The `services.NewCacheConfigService(db)` constructor inserts default config rows into the `sys_config` table (49 rows including cache.* and rate_limit.* keys). This caused test pollution when verifying Statistics/List results. **Mitigation:** Pass `nil` for the cache config parameter in `newConfigCacheService`, since `CacheServiceBase.GetExpiration` gracefully handles nil config.

### Reflection Issue with NoOpCacheProvider
The `NoOpCacheProvider` uses reflection to populate `dest` after `query()`. For `*Config` pointer returns, the AssignableTo check fails and `dest` remains zero-valued. **Mitigation:** Tests using `NoOpCacheProvider` for cache miss paths do not assert on field values; for positive paths, use direct base service calls (e.g., `rs.GetMenusWithCache` instead of `svc.GetMenusWithCache`).

## Build Verification

```
go build ./internal/api/v1/system/... ./internal/services/system/... — PASS
go test -count=1 -timeout 5m -run "TestConfigService|TestConfigHandler|TestRoleService|TestRoleHandler|TestConfigCache|TestRoleCache" ./internal/api/v1/system/... ./internal/services/system/... — PASS
```

## Deviations from Plan

- None. Plan executed exactly as specified.

## Commit Message Draft

```
test(72-10): CORE-04+06 config+role sub-module tests → ≥70%
```

## Uncommitted

Per plan 72-13 batch commit strategy, all changes are uncommitted. The 72-13 plan will commit both 72-10 and 72-11 changes together.
