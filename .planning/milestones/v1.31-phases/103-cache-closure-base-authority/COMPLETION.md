# Phase 103 Completion: 缓存闭包收敛 base 单一权威

**Phase**: 103-cache-closure-base-authority
**Status**: COMPLETED
**Completed**: 2026-09-08
**Plans**: 4 plans (103-01 through 103-04)
**Requirements**: CONV-01, CONV-02, CONV-03, CONV-04

## Summary

Phase 103 completed the Phase 92 cache unification by migrating the three remaining legacy cache domains (mac_history/heatmap, asset reconciliation, rpa selector) from hand-written interface{} closure GetOrSet and cache-aside patterns to `base.GetOrSetJSON[T]` + `base.CacheProvider` single authority. Extended the invariants AST scan to cover services root + asset + rpa packages.

## Requirements Delivered

| Requirement | Description | Status |
|-------------|-------------|--------|
| CONV-01 | mac_history 4 处 legacy GetOrSet + 手写 cache-aside + heatmap 迁 base.GetOrSetJSON | DONE |
| CONV-02 | reconciliation GetByWorkstation 手写读穿透迁 base.GetOrSetJSON | DONE |
| CONV-03 | rpa selector GetBestSelector/SaveSelector 手写 JSON cache-aside 迁 base.GetOrSetJSON | DONE |
| CONV-04 | invariants 扫描扩至 services 根 / asset / rpa（硬失败档） | DONE |

## Plans Executed

### Plan 103-01 (CONV-01 — mac_history 域迁移)

**Commits**: 44301ef, 4b6f3d6, 5565fc7

- `mac_history_query_service.go`: struct field `cache base.CacheProvider` (dataCache removed); vendor cache-aside (:259-281) → `base.GetOrSetJSON[string]` with 24h TTL; 3 query sites (:309/:434/:835) → `base.GetOrSetJSON[*MACHistoryQueryResult]` strict semantics
- `mac_history_heatmap_service.go`: struct + constructor migrated; `getHeatmapWithCache` wrapper (err → warn + DB fallback)
- `mac_history_router.go`: `system.NewCacheProvider(core.DataCacheService)` wired
- New fixture: `fakeMACHistoryCacheProvider` (base.CacheProvider implementation, import-cycle-free)

**Key deviations fixed**:
- GetVendor nil panic guard: `cache == nil` bare assembly retains `lookupVendorFromDB` direct path (equivalent to original nil-handling)
- Cache value assertion: JSON quoted form due to DataCacheService JSON serialization

### Plan 103-02 (CONV-02 — asset reconciliation 迁移)

**Commits**: 8d1901c, 957d5ae

- `reconciliation_service.go`: struct field `cache base.CacheProvider`; `getByWorkstationWithCache` wrapper with warn-on-set non-blocking semantics; dirty-cache self-heal (cached Workstation.ID == "" → recompute)
- `reconciliation_router.go` + `router.go:619`: `system.NewCacheProvider(core.DataCacheService)` wired
- `asset_gapfill_test.go:346`: test adapter updated

### Plan 103-03 (CONV-03 — rpa selector 迁移)

**Commits**: d5637a1

- `selector_learner.go`: struct + constructor migrated; `getBestSelectorCached` wrapper (best-effort silent: cache err → warn + DB recompute); `computeBestSelector` extracted as primary query function; `RecordSuccess` invalidation → `base.Invalidate`
- `ai_service.go`: third param `cache cache.Cache` → `cacheProvider base.CacheProvider`
- `service.go`: `NewServiceGroup` new 6th param `cacheProvider base.CacheProvider`; `cacheInstance` retained for CredentialService/TaskService
- `rpa_router.go` (2 sites) + `core.go:1056`: `system.NewCacheProvider(core.DataCacheService)` wired
- `ai_selector_excel_test.go`: `fakeSelectorCache` retrofitted to full `base.CacheProvider` with real read-through semantics

### Plan 103-04 (CONV-04 — invariants 扩口)

**Commit**: 37dca76

- New file: `cache_invariants_103_test.go`
- `TestNoInterfaceGetOrSetResidue103`: scanDirs103 (services root + asset + rpa), interface{} closure GetOrSet hard-fail, 0 allowed residues
- `cacheAsideResidue103` AST detector: function-scope position-strictly-increasing triple `cache.Get < json.Unmarshal < cache.Set` (replaces flawed line-window approach; validated with positive control using pre-migration selector_learner)
- `TestNoHandwrittenCacheAside`: 4 migrated files → 0 residue (hard-fail tier)

**Key detector calibration**: Initial line-window approach (Get within 5-10 lines of Unmarshal/Set) structurally missed pre-migration selector_learner where Get@170 and Set@227 are separated by 57 lines (DB query body in between). Corrected to position-ordered triple (g < u < s within same function scope), validated by positive control.

## Key Accomplishments

1. **mac_history domain convergence**: 4 legacy GetOrSet + vendor cache-aside + heatmap all on `base.GetOrSetJSON[T]`; services root interface{} closure GetOrSet zeroed
2. **asset reconciliation convergence**: GetByWorkstation on `base.GetOrSetJSON[T]` with dirty-cache self-heal; warn-on-set non-blocking semantics preserved
3. **rpa selector convergence**: GetBestSelector on `base.GetOrSetJSON[T]`; best-effort silent semantics preserved; `base.Invalidate` for record invalidation
4. **invariants extended**: Phase 92 (system/operations cache_impl) + Phase 103 (root/asset/rpa) two-generation dual file scan active; new handwritten cache-aside AST detector calibrated with positive control

## Verification

```
go build ./...                                                       ✓ BUILD OK
go test ./internal/services/system/ -run "GetOrSetResidue103|HandwrittenCacheAside"  ok
go test ./internal/services/ -run "MACHistory|Heatmap|Vendor|Mhq"   ok  2.581s
go test ./internal/services/asset/ -run "Reconciliation"             ok  0.217s
go test ./internal/services/rpa/ -run "Selector|ServiceGroup"        ok  0.167s
go test ./internal/api/v1/network/ ./internal/api/v1/asset/          ok
go test ./internal/services/... (full suite, 20 packages)            0 FAIL
```

## Phase Dependency

- Phase 103 depends on Phase 102 (CACHE-02 registers mac vendor / rpa selector cache keys; Phase 103 migrates the closures in the same files — order cannot be reversed)
- Phase 103 → Phase 107 (selector_learner file structure stabilized before TODO-02 decision)

---

_Completed: 2026-09-08_
