# Phase 102 Completion: 机械常量化（缓存键 / 状态 / 分页）

**Phase**: 102-mechanical-constants
**Status**: COMPLETED
**Completed**: 2026-09-07
**Plans**: 5 plans (102-01 through 102-05)
**Requirements**: CACHE-01, CACHE-02, STATUS-01, PAGI-01

## Summary

Phase 102 delivered mechanical constantization across three orthogonal axes: cache key literals, status numeric literals, and pagination normalization. All changes are behavior-equivalent refactors with zero business-semantic impact.

## Requirements Delivered

| Requirement | Description | Status |
|-------------|-------------|--------|
| CACHE-01 | captcha 域 12 处内联缓存键收敛具名常量 | DONE |
| CACHE-02 | 10 模块 ~47 处缓存键调用点收敛具名常量 + invariants 扫描守护 | DONE |
| STATUS-01 | 12 处 status 字面量收敛 models 具名常量 + AST 扫描守护 | DONE |
| PAGI-01 | 分页双口径收敛 NormalizePaginationWithMax，pagination.go 整文件删除 | DONE |

## Plans Executed

### Wave 1 (parallel, zero file overlap)

**102-01 (CACHE-01 — captcha 键族常量化)**
- 3 commits: d28cac7 (RED test + 6 constants), 2f520a0 (captcha.go 13 sites), 7dcf828 (captcha_background.go 3 sites)
- New constants: `pkg/constants/cache.go` — CaptchaRateLimitKeyFormat, CaptchaDataKeyFormat, CaptchaAttemptsKeyFormat, LoginFailKeyFormat, CaptchaBgListKeyFormat, CaptchaCachePoolPrefixFormat
- Snapshot test: `TestCaptchaCacheKeyEquivalence` — 6 cases

**102-04 (STATUS-01 — status 位点替换)**
- 3 commits: 464bf99 (scheduler 6 files, 14 sites), 9e40c2b (services layer + InfoPointStatus raw SQL), 4b58cdf (TestNoStatusLiteralUsage AST guard + WorkOrderStatus value lock)
- Replaced in: cron.go, vdi_sync_tasks.go, workorder_tasks.go, reconciliation_tasks.go, mac_history_tasks.go, mac_history_matview_tasks.go, job_service.go, workorder/base.go
- New AST scan: `TestNoStatusLiteralUsage` (7 patterns, whitelist with 11 documented entries)

**102-05 (PAGI-01 — 分页口径归一)**
- 2 commits: 046feff (file_handler.go migration), 418110e (pagination.go deletion)
- `file_handler.go:160` migrated to `query.NormalizePaginationWithMax(req.Page, req.PageSize, constants.MaxListPageSize)`
- `internal/utils/pagination.go` entire file deleted (Phase 99 D-03-3 inheritance lock)
- `TestParsePaginationAndOffset` function removed from utils_74_12_test.go

### Wave 2 (blocked on Wave 1, pkg/constants dependency)

**102-02 (CACHE-02 注册面 — 8 模块 + 根包 2 格式)**
- 2 commits: 4433499 (cache_keys.go: 20 constants + 27 helpers + snapshot test), f3648c6 (pkg/constants: UserEndpointsKeyFormat + MacVendorKeyFormat)
- Modules: notice, settings, duty, workorder, knowledge, network, widget, rpa
- Snapshot test: `TestCacheKeyEquivalence` (20 constants + 27 helper outputs)

### Wave 3 (blocked on Wave 2, registration dependency)

**102-03 (CACHE-02 替换面 — 47 调用点替换 + 扫描守护)**
- 3 commits: 6ef2e26 (system package 3 files), 8a03ef5 (5 sub-packages), 74d7eeb (root package 2 files + scan guard)
- 47 call sites replaced across 10 modules
- New helper: `GetNoticeMyNoticesPattern` (discovered required by scan guard, Rule 2 auto-add)
- New scan guard: `TestCacheKeyInlineResidue` (12-file narrow scan, 0 violations)
- D-102-1 revision: root package (api_endpoint_service.go, mac_history_query_service.go) use `pkg/constants` instead of cache_keys.go due to import cycle constraint

## Key Accomplishments

1. **Cache key constantization**: 68 prefix constants + 38 helper functions registered; 47 call sites replaced; 12-file inline residue scan active with 0 violations
2. **Status constantization**: 21+ literals replaced with models constants; AST guard `TestNoStatusLiteralUsage` with 7 patterns and documented whitelist; WorkOrderStatus value lock added
3. **Pagination normalization**: `utils.ParsePagination` eliminated; `pkg/query.NormalizePaginationWithMax` single authority; `internal/utils/pagination.go` file deleted
4. **Invariant compliance**: TTL unchanged (diff gate + existing regression tests); cache key values byte-equivalent (snapshot tests); behavior equivalence verified by all tests passing

## Verification

```
go build ./...                                    ✓ BUILD OK
go test ./pkg/constants/                           ok  0.595s
go test ./internal/services/system/                ok  4.120s
go test ./internal/models/                         ok  1.827s
go test ./internal/core/                           ok  149.843s
go test ./internal/scheduler/                      ok  8.045s
go test ./internal/api/v1/system/                  ok  0.199s
```

## Phase Dependency

- Phase 102 is v1.31 first phase (no dependencies)
- Phase 103 (CONV) depends on Phase 102 (CACHE-02 registers mac vendor / rpa selector cache keys before Phase 103 migrates the closures)

---

_Completed: 2026-09-07_
