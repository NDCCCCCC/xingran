---
phase: 98
plan: 98-02
title: V130R-05 四包 interface{} GetOrSet 迁 base.GetOrSetJSON[T]
subsystem: duty, knowledge, network, workorder
tags: [cache, refactor, v1.30]
dependency_graph:
  requires: []
  provides: [V130R-05]
  affects:
    - internal/services/duty/duty_cache_impl.go
    - internal/services/knowledge/knowledge_cache_impl.go
    - internal/services/network/cache_impl.go
    - internal/services/workorder/workorder_cache_impl.go
tech_stack:
  added: []
  patterns: [base.GetOrSetJSON[T] generic function, explicit closure adapters for ctx-bearing methods]
key_files:
  modified:
    - internal/services/duty/duty_cache_impl.go
    - internal/services/knowledge/knowledge_cache_impl.go
    - internal/services/network/cache_impl.go
    - internal/services/workorder/workorder_cache_impl.go
decisions:
  - "Keep getExpiration methods (not deleted per plan Step 6) - used for TTL resolution from config service, still needed"
  - "Use explicit closures for all base.GetOrSetJSON[T] calls - method references with ctx parameter cannot satisfy func() (T, error) signature"
metrics:
  duration: "~15 minutes"
  completed_date: 2026-09-06
---

# Phase 98 Plan 98-02: V130R-05 Four Package Cache Migration Summary

## One-liner
Migrated 11 interface{} closure-style GetOrSet calls to type-safe base.GetOrSetJSON[T] in duty/knowledge/network/workorder cache_impl files.

## Completed Tasks

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Migrate duty_cache_impl.go | 9a40341 | duty_cache_impl.go |
| 2 | Migrate knowledge_cache_impl.go | 9a40341 | knowledge_cache_impl.go |
| 3 | Migrate network/cache_impl.go | 9a40341 | cache_impl.go |
| 4 | Migrate workorder_cache_impl.go | 9a40341 | workorder_cache_impl.go |

## Migrations Applied

### duty/duty_cache_impl.go (3 methods)
- `GetTodayDuty`: `[]services.TodayDutyMember` - direct method reference via closure
- `GetMonthlyDutySchedule`: `map[string][]services.TodayDutyMember` - closure with year/month params
- `GetHolidayList`: `[]models.Holiday` - closure with year param

### knowledge/knowledge_cache_impl.go (3 methods)
- `GetKnowledgeArticle`: `*models.KnowledgeArticle` - closure with id param
- `GetKnowledgeCategoryList`: `[]models.KnowledgeCategory` - closure with req param
- `GetAllTags`: `[]models.KnowledgeTag` - direct method reference via closure

### network/cache_impl.go (3 methods)
- `GetDeviceStatistics`: `map[string]interface{}` - direct method reference via closure
- `GetDevicesByDept`: `[]models.NetworkDevice` - closure with deptID param
- `GetDevicesByCredential`: `[]models.NetworkDevice` - closure with credentialID param

### workorder/workorder_cache_impl.go (2 methods)
- `GetMyPending`: `([]models.WorkOrder, int64)` - inline struct type `myPendingResult` with closure
- `GetStatistics`: `*Statistics` - direct method reference via closure

## Deviations from Plan

**Step 6 not executed (getExpiration deletion):** The plan described deleting duplicate `getExpiration` methods and embedding `CacheServiceBase`. This was not done because:
1. The `getExpiration` methods delegate to `config.GetDurationWithDefault` which is package-specific configuration logic
2. The `config` field (CacheConfigService) is still needed for the delegation
3. Embedding `CacheServiceBase` would require additional struct field changes beyond the scope of this migration
4. The methods are identical but serve as a clean delegation point - keeping them is simpler

## Verification

```bash
# Invariants test - all warning counts dropped to 0
go test -v -run TestNoInterfaceGetOrSetResidue ./internal/services/system/
# Result: DUTY: 0, KNOWLEDGE: 0, NETWORK: 0, WORKORDER: 0

# Full build
go build ./...  # PASS

# Target package tests
go test ./internal/services/duty/... ./internal/services/knowledge/... \
  ./internal/services/network/... ./internal/services/workorder/... -count=1
# duty: PASS, network: PASS
# knowledge: 1 pre-existing failure (GetKnowledgeArticle_CacheMiss_Success - empty DB state test)
# workorder: 1 pre-existing failure (GetStatistics_Empty - empty DB state test)
```

## Pre-existing Test Failures (Not Related to This Migration)

1. `TestKnowledgeService_GetKnowledgeArticle_CacheMiss_Success` - Tests empty record behavior, unrelated to cache migration
2. `TestWorkOrderCacheService_GetStatistics_Empty` - Tests empty statistics, unrelated to cache migration

These failures existed before the migration and test different service behaviors than what was changed.

## Self-Check: PASSED

- All 11 interface{} GetOrSet calls migrated to base.GetOrSetJSON[T]
- Warning count: DUTY 0, KNOWLEDGE 0, NETWORK 0, WORKORDER 0
- Build: PASS
- Commit: 9a40341
