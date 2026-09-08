# Phase 107 Decisions — DEFER Items

## DEFER-01: Work Order Rating Feature (TODO-01)

**File:** `internal/api/v1/workorder/workorder_router.go:65`

**Current state:** Router stub that registers `SetupWorkOrderRatingsRouter` with a TODO comment noting that rating functionality needs to be added to WorkOrderHandler.

**Why DEFER:**
- Requires new database table and model for ratings
- Requires new service layer implementation
- Requires frontend integration (rating UI, display)
- Needs product decision on rating schema (stars? thumbs? detailed survey?)
- Router stub itself is harmless — removing it would break routing if/when the feature is implemented

**Decision:** DEFER to a future product planning phase. Not a bug, not a regression — a future enhancement. When prioritization is needed, create a tracked issue for the work order rating feature.

---

## DEFER-02: Reconciliation Exception Cache Invalidation (TODO-04)

**File:** `internal/services/asset/reconciliation_exception.go:579`

**Current state:** `invalidateCache()` method is a no-op with a `TODO(R3+):` comment noting it should call `core.Cache.Delete(ctx, CacheKeyReconciliationExceptionRuleList)`.

**Why DEFER:**
- Marked as R3+ (future milestone) by original author
- Staleness is architecturally acceptable: reconciliation exception data is read from DB directly in the hot path, not from cache
- The cron cycle window for stale data is acceptable given the alerting architecture
- Implementing cache invalidation would require refactoring the service to accept a CacheProvider injection, which is a non-trivial architectural change for a non-critical path

**Decision:** DEFER. The R3+ tag documents this as a known future optimization. No impact on current correctness.

---

## Confirmed: No TODO in selector_learner.go

After verification, `internal/services/rpa/selector_learner.go` has no placeholder TODOs:
- Line 235: `rec := &SelectorRecommendation{...}` — normal struct initialization
- Line 367: `// 计算最近使用得分（30天内）` — normal algorithm comment

No action needed for this file.
