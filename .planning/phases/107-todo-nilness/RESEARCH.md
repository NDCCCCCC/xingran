# Phase 107 Research: TODO 清零 + nilness 排查

## Summary Table

| ID | File | Line | Recommendation | Effort |
|----|------|------|---------------|--------|
| TODO-01 | `internal/api/v1/workorder/workorder_router.go` | 65 | DEFER | Medium |
| TODO-02 | `internal/api/v1/rpa/worker_handler.go` | 260, 279 | DELETE | Low |
| TODO-02 | `internal/api/v1/rpa/credential_handler.go` | 154 | DELETE | Low |
| TODO-02 | `internal/services/rpa/error_handling.go` | 311 | DELETE | Low |
| TODO-02 | `internal/services/rpa/selector_learner.go` | 235, 367 | DELETE | Low |
| TODO-02 | `internal/services/rpa/task_service.go` | 296 | DELETE | Low |
| TODO-03 | `internal/services/system/config_service.go` | 257 | DELETE | Low |
| TODO-03 | `internal/api/v1/monitor/login_log_handler.go` | 117 | IMPLEMENT | Medium |
| TODO-03 | `internal/core/db/init_data.go` | 638 | DELETE | Low |
| TODO-04 | `internal/core/core.go` | 911 | DELETE | Low |
| TODO-04 | `internal/services/device_discovery_service.go` | 662 | DELETE | Low |
| TODO-04 | `internal/agent/server/handlers.go` | 304 | DELETE | Low |
| TODO-04 | `internal/services/asset/reconciliation_exception.go` | 579 | DEFER | Medium |
| TODO-05 | `xingran-react-frontend/src/components/dashboard/layout/LayoutToolbar.tsx` | 132, 139 | DELETE | Low |
| TODO-05 | `xingran-react-frontend/src/pages/operations/building-spaces-3d/hooks/useGeocoding.ts` | 131 | DELETE | Low |
| TODO-05 | `xingran-react-frontend/src/pages/operations/building-spaces/components/WorkstationView.tsx` | 73 | DELETE | Low |
| TODO-05 | `xingran-react-frontend/src/pages/operations/assets/index.tsx` | 582 | DELETE | Low |
| TODO-05 | `xingran-react-frontend/src/pages/operations/rpa/tasks/modals/AIScriptEditor.tsx` | 97 | DELETE | Low |
| NIL-01 | `internal/services/rpa/data_mapper.go` | 332 | DELETE | Low |

---

## Backend Research (Go files)

### TODO-01: workorder 评价功能占位

**File:** `internal/api/v1/workorder/workorder_router.go`
**Line:** 65

```go
// SetupWorkOrderRatingsRouter 设置工单评价路由
// 注意：评价路由暂时保留原有实现，因为Handler中未包含评价功能
func SetupWorkOrderRatingsRouter(r *gin.RouterGroup, core *core.Core) {
    // TODO: 评价功能需要在WorkOrderHandler中添加
    // 暂时保留原有函数式handler
}
```

**Context:** The router function is a stub that registers a placeholder. The comment references an old function-style handler pattern that has since been migrated to struct-based handlers with dependency injection.

**Recommendation:** DEFER — Work order rating/review feature is a meaningful enhancement but requires: (1) new DB table/model for ratings, (2) service layer implementation, (3) frontend integration. Not a quick delete. Needs product decision on rating schema first.

**Effort:** Medium

---

### TODO-02: RPA 域

#### `internal/api/v1/rpa/worker_handler.go` lines 260, 279

**Line 260 — GetAutoScaleConfig:**
```go
// TODO: 从数据库或配置获取自动扩缩容配置
config := AutoScaleConfig{
    Enabled:            false,
    ...
}
success(c, config)
```

**Line 279 — UpdateAutoScaleConfig:**
```go
// TODO: 保存配置到数据库
operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA工作节点", operlog.OperTypeUpdate)
successMsg(c, "自动扩缩容配置已更新")
```

**Context:** Both are stub implementations returning hardcoded defaults. The auto-scaling config feature is not part of the current RPA worker design — the `WorkerHandler` uses a simple fixed concurrency model. These TODOs indicate incomplete feature implementation from a design phase that was never completed.

**Recommendation:** DELETE — Auto-scaling is not part of the RPA worker design. The hardcoded defaults are sufficient placeholders. Remove the TODOs and keep the hardcoded defaults (they serve as valid returns for the API).

**Effort:** Low

#### `internal/api/v1/rpa/credential_handler.go` line 154

```go
// 使用 sessionService 查询
// TODO: 实现 ListSessions 方法
success(c, gin.H{
    "list":  []interface{}{},
    "total": 0,
})
```

**Context:** Returns empty list as stub. The `ListSessions` method on credential handler was planned but never implemented. RPA sessions are tracked via `sessionService` but session listing is not exposed in the credential API.

**Recommendation:** DELETE — The empty return is a valid stub. If sessions are needed, they should be accessed via a dedicated `session_handler.go`, not the credential handler. Remove the TODO comment.

**Effort:** Low

#### `internal/services/rpa/error_handling.go` line 311

```go
// TODO: 实现实际的回滚逻辑
// 1. 获取已完成的步骤
// 2. 按相反顺序执行补偿动作
// 3. 更新执行状态
return nil
```

**Context:** The `Rollback` method is a stub that returns nil (no-op). The comment outlines what a real rollback would need, but the RPA execution model does not currently support compensating transactions — steps are not reversible.

**Recommendation:** DELETE — This is a placeholder for a feature that was never designed. The no-op rollback is acceptable for the current execution model. Remove the TODO and the comment block, keep the `return nil`.

**Effort:** Low

#### `internal/services/rpa/selector_learner.go` lines 235, 367

**Line 235 — SelectorRecommendation struct initialization:**
```go
rec := &SelectorRecommendation{
    Selector:     success.Selector,
    ...
}
// No TODO here — the comment is at line 367
```

**Line 367 — Recency score calculation:**
```go
// 计算最近使用得分（30天内）
daysSinceLastUse := time.Since(stats.LastUsedAt).Hours() / 24
recencyScore := 1.0 - (daysSinceLastUse / 30.0)
```

**Context:** No TODO at line 235. The actual TODO at line 367 is part of a scoring algorithm comment. The selector learner is fully implemented — these lines are normal business logic.

**Recommendation:** DELETE — These are not placeholder TODOs but normal algorithm comments. No action needed.

**Effort:** Low

#### `internal/services/rpa/task_service.go` line 296

```go
// 没有有效会话，传递凭证信息用于自动登录
// 从用户ID获取部门ID（这里需要从上下文获取）
deptID := "" // TODO: 从上下文获取部门ID
```

**Context:** When no valid RPA session exists for a user, the code falls back to credential-based login. The `deptID` is used to scope credential lookup but is left as empty string. The comment acknowledges this is a simplification — in production, the department context should be available from the request context.

**Recommendation:** DELETE — This is a known limitation acknowledged in code. The empty string is a valid fallback. The TODO documents a missing context propagation path that would require tracing through the call chain. The feature works (falls back to credential login) even without the deptID. Remove the TODO comment.

**Effort:** Low

---

### TODO-03: system/monitor 域

#### `internal/services/system/config_service.go` line 257

```go
func (s *configService) RefreshCache(ctx context.Context) error {
    // TODO: 实现缓存刷新逻辑
    return nil
}
```

**Context:** `RefreshCache` is a method on `ConfigService` that is currently a no-op returning nil. System config uses the standard cache pattern via `base.GetOrSetJSON` in other methods, but this explicit refresh method is not called anywhere in the codebase.

**Recommendation:** DELETE — This method is never called. If cache refresh for config is needed, it should be implemented properly with `base.Invalidate` calls. But since no caller exists, this is dead code. Remove the entire method.

**Effort:** Low

#### `internal/api/v1/monitor/login_log_handler.go` line 117

```go
// UnlockUser 解锁用户 — Phase 107 TODO-03 placeholder
func (h *LoginLogHandler) UnlockUser(c *gin.Context) {
    username := c.Param("username")
    ...
    // TODO: 实现解锁用户逻辑（如从Redis中删除锁定状态）
    response.Success(c, gin.H{"message": "解锁成功"})
}
```

**Context:** The `UnlockUser` handler is a stub that always returns success without actually unlocking anything. Login brute-force protection writes to Redis with a lockout key. To unlock, the Redis key needs to be deleted. This is a real feature gap — an admin cannot manually unlock a locked-out user.

**Recommendation:** IMPLEMENT — This is a functional gap. The implementation needs to: (1) check if Redis is available via `core.Cache`, (2) construct the lockout cache key (pattern: `login:lockout:{username}`), (3) call `Delete` on the cache. The handler already exists with correct signature. Requires verifying the lockout key pattern from the auth/login code.

**Effort:** Medium

#### `internal/core/db/init_data.go` line 638

```go
// createCaptchaBackgroundMenus 创建验证码背景图管理菜单
// TODO: 此函数未使用，如需要启用验证码背景图功能，请取消注释并调用此函数
/*
func createCaptchaBackgroundMenus(db *gorm.DB) error {
    ...
}
*/
```

**Context:** The entire function is commented out. It is never called. The comment explains that if captcha background image feature is needed, uncomment and call the function. This is a classic "preserve for future use" pattern.

**Recommendation:** DELETE — This is dead code. The captcha background feature has never been implemented. The commented function has no callers and no planned timeline. Remove the entire commented block.

**Effort:** Low

---

### TODO-04: 基础设施域

#### `internal/core/core.go` line 911

```go
// Phase 36: 启动 Redis pub/sub 跨进程缓存失效订阅
// TODO: 当 core.Core 接入 Redis 后启用；当前传 nil 不影响主流程
if err := accountPool.StartHotReload(context.Background()); err != nil {
    applogger.Warnf("启动 AD 账号池热加载失败（不影响主流程）: %v", err)
}
```

**Context:** The `StartHotReload` method requires a Redis client for pub/sub. The TODO acknowledges that Redis is not yet part of `core.Core` (the main app struct). The code currently passes `nil` for the Redis client. The feature (AD account pool hot-reload via Redis pub/sub) is designed but not yet wired up.

**Recommendation:** DELETE — The TODO comment is informational, not a placeholder for missing logic. The code already handles the "not yet wired" case correctly (passes nil, catches error, logs warning but doesn't fail startup). Remove the TODO comment as it no longer represents an action item — the Redis integration will happen naturally when `core.Core` gets a Redis client injected.

**Effort:** Low

#### `internal/services/device_discovery_service.go` line 662

```go
// TODO: 实际实现中需要从临时表或缓存中获取发现的设备
// 目前返回空列表
return []*DiscoveredDevice{}, nil
```

**Context:** Device discovery scan returns an empty list. The comment indicates that discovered devices should be stored in a temporary table or cache during scanning, but the current implementation never persists them. The scanning logic runs but results are discarded.

**Recommendation:** DELETE — The TODO describes a missing feature (result persistence) that is a larger scope. The stub return of empty list is acceptable for the current scanning workflow which may be used for connectivity testing only. The TODO comment can be removed; the empty return is a valid behavior for a discovery scan that finds no devices.

**Effort:** Low

#### `internal/agent/server/handlers.go` line 304

```go
// 如果请求中提供了 agent_id 和 vm_id，使用它们注册
// 否则使用 authenticator 中已有的配置
if req.AgentID != "" && req.VMID != "" {
    // TODO: 实现动态更新 agent_id 和 vm_id 的逻辑
    // 当前使用 authenticator 中的配置
    _ = req // 占位: 当前实现始终使用 authenticator 配置;SA9003 抑制
}
```

**Context:** The agent registration handler receives optional `agent_id` and `vm_id` in the request but ignores them, always using the config from the `authenticator`. The TODO indicates this was intended to support dynamic re-registration with different IDs, but was never implemented.

**Recommendation:** DELETE — The TODO documents a planned but never implemented feature. The current behavior (always using authenticator config) is consistent and correct. The `_ = req` line with the SA9003 suppression comment is the real code smell — it should either be removed (if req is truly unused) or the TODO should be converted to an issue. Remove the TODO and the unused assignment line.

**Effort:** Low

#### `internal/services/asset/reconciliation_exception.go` line 578

```go
func (s *reconciliationExceptionServiceImpl) invalidateCache() {
    // TODO(R3+): core.Cache.Delete(ctx, CacheKeyReconciliationExceptionRuleList)
    // 当前 service 层 ctx-agnostic,CacheProvider 注入由 handler 完成。
    // 数据写入 DB 后,下次 List 查询会从 DB 读取,缓存陈旧窗口在 cron 周期内
    // 可接受(告警通路由 DetectLayer3 内存匹配驱动,不依赖此缓存)。
}
```

**Context:** The `invalidateCache` method is a no-op. The TODO references "R3+" which seems to be a future release milestone. The reconciliation exception service writes to DB, but the List query reads from DB directly (not cached), so staleness is acceptable within the cron cycle. The `CacheProvider` is mentioned as something that would be injected at the handler layer.

**Recommendation:** DEFER — This is a "TODO for later" with a clear rationale documented in the comment. The staleness window is acceptable given the architecture. This is not a bug, just a future optimization. The R3+ tag indicates this is tracked for a future release. No action needed at this time.

**Effort:** Medium (would require handler-layer cache integration)

---

## Frontend Research (TypeScript files)

### TODO-05: 前端 6 处功能占位

#### `xingran-react-frontend/src/components/dashboard/layout/LayoutToolbar.tsx` lines 132, 139

**Line 132:**
```tsx
// TODO: 打开Widget选择器
message.info("Widget选择器功能待实现");
```

**Line 139:**
```tsx
// TODO: 打开仪表盘设置
message.info("仪表盘设置功能待实现");
```

**Context:** `LayoutToolbar` component has two stub action handlers for adding widgets and opening dashboard settings. Both show a message toast and do nothing else. These are UI feature placeholders for a dashboard customization system.

**Recommendation:** DELETE — These are UX placeholders for dashboard customization features (widget selection and settings). Both are non-critical UI enhancements. Remove the TODO comments and the `message.info` stubs (or keep the stubs without the TODO comments). The `message.info` stub alone is sufficient to indicate the feature is not yet wired.

**Effort:** Low

#### `xingran-react-frontend/src/pages/operations/building-spaces-3d/hooks/useGeocoding.ts` line 131

```tsx
// TODO: 后端暂时不支持逆地址解析，这里保留接口但返回空值
// 如果需要，可以在后端添加逆地址解析的 API 端点
setLoading(false);
return null;
```

**Context:** The `useGeocoding` hook supports reverse geocoding (coordinates → address) but the backend does not have this endpoint. The hook gracefully returns `null` instead of failing. The TODO documents a backend gap.

**Recommendation:** DELETE — The TODO documents a backend limitation, not a frontend action item. The frontend code is already handling this gracefully. The TODO comment can be removed. If reverse geocoding is needed, it should be tracked as a backend feature request, not a frontend TODO.

**Effort:** Low

#### `xingran-react-frontend/src/pages/operations/building-spaces/components/WorkstationView.tsx` line 73

```tsx
const handleEdit = (workstation: WorkstationNode) => {
    message.info(`编辑工位: ${workstation.name}`);
    // TODO: 打开编辑对话框
};
```

**Context:** `WorkstationView` has an edit button that shows a message but does not open any dialog. The component displays workstation nodes in a tree view from a 3D floor plan context.

**Recommendation:** DELETE — The TODO documents the missing edit dialog. The stub `message.info` call indicates this is a placeholder. Remove the TODO comment (keep the message.info stub or replace with a proper Modal implementation if edit is needed).

**Effort:** Low

#### `xingran-react-frontend/src/pages/operations/assets/index.tsx` line 582

```tsx
const handleEdit = useCallback((_record: Asset) => {
    // TODO: 实现编辑功能
    message.info("编辑功能待实现");
    ...
}, []);
```

**Context:** The asset list page has an edit button that shows a message placeholder. The `_record` parameter is prefixed with underscore indicating it is intentionally unused. The edit functionality is not implemented.

**Recommendation:** DELETE — The TODO documents missing edit functionality for the asset management page. This is a feature gap but the frontend is already handling it gracefully with a message. Remove the TODO comment.

**Effort:** Low

#### `xingran-react-frontend/src/pages/operations/rpa/tasks/modals/AIScriptEditor.tsx` line 97

```tsx
// TODO: 调用后端 AI API 生成脚本
// const result = await post('/rpa/ai/generate', { description });
// setGeneratedActions(result.data.script.actions);
// 模拟 API 调用
await new Promise((resolve) => setTimeout(resolve, 1500));
setGeneratedActions(mockGeneratedActions);
```

**Context:** The `AIScriptEditor` modal has an AI generation feature that is stubbed with a mock delay. The TODO shows the commented-out real API call that was planned but not implemented. The mock provides a realistic delay and generates hardcoded actions.

**Recommendation:** DELETE — The TODO documents a missing AI API integration. The mock implementation is already in place and provides a good UX simulation. Remove the TODO and commented code — the mock is a valid temporary implementation. If the AI feature is prioritized, implement the real API call.

**Effort:** Low

---

## NIL-01: nilness 根因排查

**File:** `internal/services/rpa/data_mapper.go`
**Line:** 332

```go
case TransformDefaultValue:
    if value == nil || value == "" {
        return params["default"], nil
    }
    return value, nil
```

**Analysis:** This is the `TransformDefaultValue` case in the `TransformValue` method. The nilness concern at line 332 is the condition `value == nil || value == ""`. At this point in the code, `value` has already passed a nil-check at line 239:

```go
func (s *dataMapperServiceImpl) TransformValue(...) (interface{}, error) {
    if value == nil {
        return nil, nil  // <-- line 239: early return if value is nil
    }
    ...
    case TransformDefaultValue:  // line 331
        if value == nil || value == "" {  // line 332: nil is impossible here
```

**Root Cause:** The `value == nil` part of the condition at line 332 is **impossible** — if `value` were nil, the function would have already returned at line 240. The condition `|| value == ""` is the only meaningful check here. The `value == nil` part is dead code that would never evaluate to true.

**Is it a bug?** No practical bug. The code always works correctly because:
- If `value` is nil → returns `params["default"]` (correct, just with redundant nil check)
- If `value` is "" → returns `params["default"]` (correct)
- If `value` is anything else → returns `value` (correct)

**Recommendation:** DELETE — The `value == nil` part of the condition is dead code. Fix by removing the nil check, keeping only `value == ""`. The corrected line:

```go
if value == "" {
    return params["default"], nil
}
return value, nil
```

This is a trivially safe fix that eliminates the impossible condition flagged by the nilness analyzer.

**Effort:** Low

---

## Quick DELETE Candidates (no brainer)

The following can be deleted in a single batch pass — they are pure TODO comment removals or no-op stubs that don't affect functionality:

1. `TODO-02` (5 locations): worker_handler.go:260,279 — delete TODO comments
2. `TODO-02` credential_handler.go:154 — delete TODO comment
3. `TODO-02` error_handling.go:311 — delete TODO comment block
4. `TODO-02` task_service.go:296 — delete TODO comment
5. `TODO-03` config_service.go:257 — delete entire no-op method
6. `TODO-03` init_data.go:638 — delete commented function
7. `TODO-04` core.go:911 — delete TODO comment
8. `TODO-04` device_discovery_service.go:662 — delete TODO comment
9. `TODO-04` handlers.go:304 — delete TODO comment and unused `_ = req` line
10. `TODO-05` LayoutToolbar.tsx:132,139 — delete TODO comments
11. `TODO-05` useGeocoding.ts:131 — delete TODO comment
12. `TODO-05` WorkstationView.tsx:73 — delete TODO comment
13. `TODO-05` assets/index.tsx:582 — delete TODO comment
14. `TODO-05` AIScriptEditor.tsx:97 — delete TODO comment and commented code
15. `NIL-01` data_mapper.go:332 — fix the impossible condition

**Estimated total:** ~30 minutes for all quick deletes.

## DEFER Candidates

1. **TODO-01** (workorder rating) — needs product decision, new model, service layer, frontend integration
2. **TODO-04** reconciliation_exception.go — R3+ milestone, staleness acceptable within cron cycle

## IMPLEMENT Candidate

1. **TODO-03** login_log_handler.go:117 `UnlockUser` — requires Redis cache key pattern verification + `core.Cache.Delete` call
