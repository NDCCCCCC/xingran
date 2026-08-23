---
phase: 74-p2-finalize-and-diff-coverage
plan: 07
subsystem: system-package-gapfill
status: complete
date: 2026-08-22
---

# 74-07 SUMMARY: system package gap-fill (53.5% → 75.6%)

## Result

- **internal/services/system**: 53.5% → **75.6%** statements (target ≥75%) ✓
- **internal/services/operations**: 22.5% → **61.1%** (already at target, unchanged this plan) ✓
- **internal/services/asset**: 40.5% → **70.9%** (already at target, unchanged this plan) ✓
- **D-12 STRICT**: ZERO business code changes — only `*_test.go` files
- **Stability**: 5/5 consecutive runs green after WidgetConfig.deleted_at fix

## Files added/modified (10 files, +2266 / -619 lines)

| File | Purpose |
|------|---------|
| `internal/services/system/cache_infra_test.go` | NEW — CacheProvider / CacheAdapter / pkg cache shim helpers (440 lines) |
| `internal/services/system/dashboard_service_test.go` | NEW — DashboardServiceImpl List/Get/Default + layouts (426 lines) |
| `internal/services/system/file_service_test.go` | MODIFIED — file_service upload/download helpers (563 lines) |
| `internal/services/system/notice_service_gapfill_test.go` | NEW — notice wrapper + cache_impl delegates (197 lines) |
| `internal/services/system/system_gapfill_test.go` | NEW — profile ChangePassword/Avatar + dept Update matrix (200 lines) |
| `internal/services/system/system_menu_cache_gapfill_test.go` | NEW — menu BatchDelete + cache_adapter stats (123 lines) |
| `internal/services/system/user_cache_impl_gapfill_test.go` | NEW — queryRoles / queryPermissions / getRoleIDs (126 lines) |
| `internal/services/system/user_service_crud_test.go` | NEW — user Create/Update/List + fillUserRoles (247 lines) |
| `internal/services/system/widget_data_fetcher_test.go` | MODIFIED — widget fetcher cache + API + registry stub (374 lines) |
| `internal/services/system/apikey_service_test.go` | MODIFIED — flake fix for Windows clock-granularity collision (11 lines) |

## Key learnings / QUIRKS (D-12 — documented, not fixed)

1. **`models.WidgetConfig.Position`** has NO Valuer AND NO Scanner — GORM cannot read or write `widget_configs` rows via the Position column. Affects `widget_data_fetcher.go:224` happy-path tests. Workaround: cover only widget-not-found branch.
2. **`sys_user` BeginTime/EndTime WHERE + LEFT JOIN sys_dept** → "ambiguous column name: created_at" (production QUIRK). Reproduced in sqlite.
3. **sqlite `gen_random_uuid()`** unsupported → manual CREATE TABLE instead of AutoMigrate.
4. **Windows clock granularity** (time.Now().UnixNano collisions) → atomic counter + lock for unique IDs.
5. **appendAncestorMenuIDs** returns self-first order `[c, b, a]` (leaf up), not top-down.
6. **`getRoleIDs` requires `models.Role{BaseModel: models.BaseModel{ID: "..."}}`** — Role embeds BaseModel, can't use struct literal `{ID}` directly.

## Verification

```
go test -cover -count=1 ./internal/services/system/
ok  coverage: 75.6% of statements
```

Stability: 5/5 green.