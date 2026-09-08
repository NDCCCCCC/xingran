---
phase: 107
plan: "04"
type: execute
wave: 2
depends_on: []
files_modified:
  - .planning/phases/107-todo-nilness/DECISIONS.md
autonomous: true
requirements_addressed: [TODO-01, TODO-04]
---

<objective>
Create DECISIONS.md documenting DEFER items and verify grep-based regression guard. No code changes in this plan — documentation only.
</objective>

<tasks>

## TASK-1: Create DECISIONS.md

**File:** `.planning/phases/107-todo-nilness/DECISIONS.md`

Document the 2 DEFER items from the research with rationale for why no action is taken.

### DEFER-01: Work Order Rating Feature (TODO-01)

**File:** `internal/api/v1/workorder/workorder_router.go:65`

```go
// SetupWorkOrderRatingsRouter 设置工单评价路由
// 注意：评价路由暂时保留原有实现，因为Handler中未包含评价功能
func SetupWorkOrderRatingsRouter(r *gin.RouterGroup, core *core.Core) {
    // TODO: 评价功能需要在WorkOrderHandler中添加
    // 暂时保留原有函数式handler
}
```

**Why DEFER:**
- Requires new database table and model for ratings
- Requires new service layer implementation
- Requires frontend integration (rating UI, display)
- Needs product decision on rating schema (stars? thumbs? detailed survey?)
- Not a quick delete — scope is medium-to-large enhancement
- Router function itself is a stub; removing it would break routing for a real feature

**Decision:** DEFER to a future product planning phase. Create a tracked issue for the work order rating feature when prioritization is needed.

### DEFER-02: Reconciliation Exception Cache Invalidation (TODO-04)

**File:** `internal/services/asset/reconciliation_exception.go:579`

```go
func (s *reconciliationExceptionServiceImpl) invalidateCache() {
    // TODO(R3+): core.Cache.Delete(ctx, CacheKeyReconciliationExceptionRuleList)
    // 当前 service 层 ctx-agnostic,CacheProvider 注入由 handler 完成。
    // 数据写入 DB 后,下次 List 查询会从 DB 读取,缓存陈旧窗口在 cron 周期内
    // 可接受(告警通路由 DetectLayer3 内存匹配驱动,不依赖此缓存)。
}
```

**Why DEFER:**
- Marked as R3+ (future milestone) by original author
- Staleness is architecturally acceptable: reconciliation exception data is read from DB directly, not cached in the hot path
- The cron cycle window for stale data is acceptable given the alerting architecture
- Implementing cache invalidation would require refactoring the service to accept a CacheProvider, which is a non-trivial change for a non-critical path

**Decision:** DEFER. The R3+ tag documents this as a known future optimization. No impact on current correctness.

## TASK-2: Verify selector_learner.go TODO status

**Research verification:**

Read `internal/services/rpa/selector_learner.go` at lines 235 and 367:

- **Line 235**: `rec := &SelectorRecommendation{...}` — no TODO comment present
- **Line 367**: `// 计算最近使用得分（30天内）` — this is a normal algorithm description comment, NOT a placeholder TODO indicating incomplete code

**Conclusion:** No actual TODO placeholder exists in selector_learner.go. The Research summary incorrectly listed it. No changes needed to this file.

</tasks>

<success_criteria>
- DECISIONS.md created with both DEFER items documented
- selector_learner.go confirmed to have no placeholder TODOs
- No code changes in this plan
</success_criteria>

<output>
Part of combined SUMMARY.md after Wave 1 + Wave 2 complete
</output>
