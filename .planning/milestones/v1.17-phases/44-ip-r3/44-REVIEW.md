---
phase: 44-ip-r3
reviewed: 2026-06-28T00:00:00Z
depth: standard
files_reviewed: 30
files_reviewed_list:
  - internal/api/v1/asset/reconciliation_exception_handler.go
  - internal/api/v1/asset/reconciliation_exception_handler_test.go
  - internal/api/v1/asset/reconciliation_exception_router.go
  - internal/api/v1/asset/reconciliation_handler.go
  - internal/core/db/database.go
  - internal/core/db/migrations/migration_174_reconciliation_exception_gist.go
  - internal/scheduler/reconciliation_tasks.go
  - internal/scheduler/reconciliation_tasks_test.go
  - internal/services/asset/reconciliation_baseline.go
  - internal/services/asset/reconciliation_baseline_test.go
  - internal/services/asset/reconciliation_detection.go
  - internal/services/asset/reconciliation_detection_test.go
  - internal/services/asset/reconciliation_exception.go
  - internal/services/asset/reconciliation_exception_excel_test.go
  - internal/services/asset/reconciliation_exception_matcher.go
  - internal/services/asset/reconciliation_exception_matcher_test.go
  - internal/services/asset/reconciliation_exception_test.go
  - internal/services/asset/reconciliation_service.go
  - internal/services/asset/reconciliation_service_test.go
  - internal/services/operations/excel_config.go
  - internal/services/operations/excel_raw_rows.go
  - internal/services/operations/excel_reconciliation_test.go
  - xingran-react-frontend/src/components/asset/reconciliation/ExceptionRuleForm.tsx
  - xingran-react-frontend/src/components/asset/reconciliation/MatchTestPanel.tsx
  - xingran-react-frontend/src/lib/assetApi.ts
  - xingran-react-frontend/src/lib/queryKeys.ts
  - xingran-react-frontend/src/pages/asset/reconciliation/dashboard/index.tsx
  - xingran-react-frontend/src/pages/asset/reconciliation/exception-rules/index.tsx
  - xingran-react-frontend/src/pages/asset/reconciliation/exceptions/index.tsx
findings:
  critical: 5
  warning: 11
  info: 7
  total: 23
status: issues_found
---

# Phase 44 R3: Code Review Report

**Reviewed:** 2026-06-28
**Depth:** standard
**Files Reviewed:** 30 (29 in scope + 1 verified external: `internal/models/config.go`)
**Status:** issues_found

## Summary

Phase 44 R3 implements the IP 例外规则引擎 (CIDR exception rule engine), Layer 3.5 detection interception, exception CRUD, baseline snapshot/compare, Excel import/export, and admin UI. The implementation is largely well-structured, with thoughtful defensive code (BLOCKER-4 PG three-valued logic handling, status convention compliance, audit-chain preservation in cleanup cron) and reasonable test coverage via static-source-scanning plus SQLite-backed behavior tests.

However, adversarial review surfaced **5 BLOCKER-severity defects** and **11 WARNING-severity issues** that must be addressed before ship:

1. **CR-01 (BLOCKER):** `Snapshot/Compare` writes/reads baseline JSON to/from `sys_config.config_value` which is declared `gorm:"size:500"`. JSON for production-scale counts easily fits, but adding any future field or running with very large counts plus timestamp precision will exceed 500 chars → `value too long for type character varying(500)` SQLSTATE 22001, defeating the BLOCKER-3 operational requirement.
2. **CR-02 (BLOCKER):** `matchException` returns the FIRST matched rule's ID as `matchedRuleID` for audit, but the merged actions/severity come from ALL matched rules. If the first rule is later deleted (soft-delete), `sys_data_reconciliation.exception_rule_id` still points to a deleted rule while the *applied* actions came from a different surviving rule. Audit chain references the wrong rule. The doc claim "审计可追溯" is false in multi-rule-hit scenarios.
3. **CR-03 (BLOCKER):** Baseline `/baseline/snapshot` and `/baseline/compare` routes have **no `RequirePermissions` middleware** — any authenticated user (including lowest-privilege) can call Snapshot, overwriting the canonical R2 baseline. Snapshot is a write op (operlog writes through), so unauthenticated writes poison the noise-reduction denominator and the SC 8 ≥60% audit.
4. **CR-04 (BLOCKER):** `MatchTestPanel.tsx:71-75` uses `useMemo(() => testInput, [JSON.stringify(testInput)])` which returns the *new* `testInput` object on every change but the memo dependency relies on the string comparison — this works for the queryKey, but `queryFn: () => reconciliationApi.exceptionRule.test(testInput)` still closes over the live `testInput` state, so when React Query refetches (e.g., window refocus, `staleTime: 30s` expiry), it submits the *current* `testInput` not the value used to build the queryKey. Inconsistent queryKey↔queryFn → silent cache poisoning when staleTime triggers.
5. **CR-05 (BLOCKER):** `ImportFromExcel` post-processing updates by `name` only (`WHERE name = ? AND deleted_at IS NULL`), but if a soft-deleted rule with the same name exists in the table (allowed since the unique index from migration_168 is partial on `deleted_at IS NULL`), or two rows in the Excel share the same `name`, the UPDATE affects *all matching rows* — silently cross-contaminating unrelated rules. Missing `LIMIT 1` / id-scoped update.

WARNING-tier issues include: detection engine `AppliedActions` empty array vs NULL ambiguity in `matchException` (silent Type A leakage when `len==0` but rule had `silence`); unused `_ = apperrors.BadRequest` dead-import marker; `fmt.Printf` for error logging instead of `logrus` in `ImportFromExcel`; missing auth on `/exception-rule/export` and `/template`; missing dep array entries causing stale `columns` in `exception-rules/index.tsx`; missing service validation for `ScopeType` whitelist (only validates via `default:'global'` DB tag); missing `operlog` on the silent `Update` path that flips `expires_at` from a value back to NULL (no audit trail when rule is "un-expired").

---

## Critical Issues

### CR-01: Baseline JSON write to `sys_config.config_value` (varchar 500) will overflow on production scale

**File:** `internal/services/asset/reconciliation_baseline.go:130-143`, `internal/models/config.go:8`
**Issue:**
`Config.ConfigValue` is declared `gorm:"size:500"`. The baseline service marshals `BaselineSnapshot{SnapshotAt, TotalExceptions, TotalWorkorders, CriticalExceptions}` to JSON and writes it via `Table("sys_config").Create(map)`. The current 4-field JSON is small (~140 bytes), but:

1. The `SnapshotAt` is `time.Time` → RFC3339Nano (~35 bytes); the JSON layout includes `snapshot_at`, `total_exceptions`, `total_workorders`, `critical_exceptions` keys.
2. **More critically:** PG `varchar(500)` will hard-fail with SQLSTATE 22001 the moment any future field is added (e.g., a `BySeverityBreakdown` map, or a `MvVersion`/`Hostname` audit field). The service-level comment claims "幂等覆盖" but provides no width guard.
3. On PG, the `Table("sys_config").Create(&newConfig)` path bypasses GORM model column-size validation, so the truncation behavior depends on DB engine settings (`SET standard_conforming_strings`) — silent truncation is NOT guaranteed; the default is hard error.

The `is_system: 1` is also written as `int`, but `ConfigIsSystem` is a typed enum (likely `int8`/`int`). On PG with the column typed `int2` (`ConfigIsSystem` defaults), writing `1` via `map[string]interface{}` may trigger `invalid input syntax for type smallint` depending on driver binding. This is a secondary fault.

**Fix:**
- Validate `len(data) <= 500` before write and return error if exceeded; OR migrate `sys_config.config_value` to `TEXT` (requires new migration `Migrate175_*`, plus cleanupOldConstraints entry if unique index exists on the column).
- Use the `models.Config{}` struct (not raw `map[string]interface{}`) to leverage GORM's type binding for `ConfigIsSystem`/`ConfigType`.

```go
newCfg := &models.Config{
    BaseModel:    models.BaseModel{ID: uuid.NewString()},
    ConfigName:   "资产对账降噪基线",
    ConfigKey:    BaselineConfigKey,
    ConfigValue:  string(data),
    ConfigType:   models.ConfigTypeYes,
    IsSystem:     models.ConfigIsSystemYes,
}
if len(data) > 500 {
    return nil, fmt.Errorf("baseline JSON 过长 (%d bytes), 需迁移 sys_config.config_value 至 TEXT", len(data))
}
if err := s.db.WithContext(ctx).Create(newCfg).Error; err != nil { ... }
```

---

### CR-02: Multi-rule match returns first rule ID as audit pointer, but applies merged actions from all — audit chain breaks when first rule is deleted

**File:** `internal/services/asset/reconciliation_exception_matcher.go:171-176`, `internal/services/asset/reconciliation_detection.go:294-301`
**Issue:**
```go
// matchExceptionWithSeverity, line 171-172
matchedRuleID = matched[0].rule.ID    // audit points to FIRST matched rule
actions, sev, silence := mergeActions(originalSeverity, matched, conflictType)
return matchedRuleID, actions, sev, silence
```
The comment on line 171 claims "审计指向首条命中规则(规则变更后历史记录仍可回溯到此 ID)". This is **false** when multiple rules match:

1. Rule A (192.168.0.0/16, actions=[silence]) — first in slice
2. Rule B (192.168.0.0/16, actions=[no_alert, skip_severity]) — second

`appliedActions` written to `sys_data_reconciliation` is the union `[silence, no_alert, skip_severity]`. But `exception_rule_id` points only to Rule A.

Operator later deletes Rule A (soft delete). Audit trail `JOIN sys_reconciliation_exception ON id = exception_rule_id` returns Rule A (with `deleted_at` set) — which only had `[silence]`. The auditor cannot explain why `no_alert` + `skip_severity` were applied, because those came from Rule B which has no link from the historical record.

This breaks the D-R3-A4-03 audit-chain requirement and Pitfall 4 ("防外键断链"). The test `TestMatchExceptionSingleGlobalRule` only exercises single-rule scenarios and does not catch this.

**Fix:**
Either:
- Store the full list of matched rule IDs (`exception_rule_ids TEXT[]`), OR
- Document and enforce that only one rule can match per (IP, conflict_type, scope) via a pre-insert uniqueness check, OR
- Pick the rule that contributed the **most actions** to the merge (deterministic, best-effort audit).

```go
// pick the rule that contributed the most actions (deterministic)
matchedRuleID = matched[0].rule.ID
maxContrib := len(matched[0].rule.ExceptionActions)
for _, r := range matched[1:] {
    if n := len(r.rule.ExceptionActions); n > maxContrib {
        maxContrib = n
        matchedRuleID = r.rule.ID
    }
}
```
Plus add a test case `TestMatchExceptionMultiRuleAuditPointsToMaxContributor`.

---

### CR-03: `/baseline/snapshot` and `/baseline/compare` routes lack RequirePermissions — any authenticated user can overwrite the canonical R2 baseline

**File:** `internal/api/v1/asset/reconciliation_exception_router.go:56-60`
**Issue:**
```go
// Phase 44 R3 / Plan 44-02 Task 3 — 降噪基线端点(放宽权限:admin 默认持读权限,audit-only)
// SnapshotBaseline 是写操作但调 operlog 留痕;CompareBaseline 只读。
// 不加 RequirePermissions 避免误锁 dashboard 卡片(参照 list/:id 放宽模式)。
r.POST("/baseline/snapshot", handler.SnapshotBaseline)
r.POST("/baseline/compare", handler.CompareBaseline)
```
The comment explicitly admits the decision: Snapshot is a **write** operation (updates `sys_config`, writes operlog `OperTypeUpdate`), but it is exposed without `RequirePermissions`. This means:

1. Any authenticated user — including a lowest-privilege operator or a compromised low-priv token — can call `POST /asset/reconciliation/baseline/snapshot` and overwrite the canonical R2 baseline that ops recorded during the R2 data-retention window.
2. The "operlog 留痕" defense is weak: operlog is after-the-fact; the damage (poisoned SC 8 denominator) is already done.
3. `list/:id` (the cited precedent) is a *read* path; the same precedent does not extend to writes. CLAUDE.md explicitly says "读路径可放宽避免误锁只读场景" — Snapshot is not a read path.

This contradicts the T-44-02 mitigation ("越权创建例外规则缓解") which locked down `/exception-rule/create` etc. with RequirePermissions — but left the more dangerous baseline-overwrite path open.

**Fix:**
```go
r.POST("/baseline/snapshot",
    middleware.RequirePermissions([]string{"asset:reconciliation:exception:create"}, core),
    handler.SnapshotBaseline)
r.POST("/baseline/compare",
    middleware.RequirePermissions([]string{"asset:reconciliation:exception:list"}, core),
    handler.CompareBaseline)
```
The `create` perm is appropriate for Snapshot (write semantics); `list` for Compare (read). Add a static-source-scan test mirroring `TestExceptionRouter4NewRoutes` to prevent regression.

---

### CR-04: MatchTestPanel queryKey↔queryFn divergence — stale `testInput` submitted on staleTime refetch

**File:** `xingran-react-frontend/src/components/asset/reconciliation/MatchTestPanel.tsx:68-83`
**Issue:**
```tsx
const [testInput, setTestInput] = useState<TestInput>({ ip: "" });

// queryKey 入参对象 useMemo 稳定(CLAUDE.md useEffect 强约束)
const stableQueryKey = useMemo(
  () => testInput,
  // eslint-disable-next-line react-hooks/exhaustive-deps
  [JSON.stringify(testInput)]
);

const { data, isFetching, refetch } = useQuery({
  queryKey: queryKeys.reconciliation.matchTest(stableQueryKey),
  queryFn: () => reconciliationApi.exceptionRule.test(testInput),  // <-- closes over LIVE state
  enabled: !!testInput.ip,
  staleTime: 30 * 1000,
});
```
The queryKey uses `stableQueryKey` (memoized on `JSON.stringify(testInput)`), but `queryFn` captures the **live** `testInput` from the render closure. When React Query refetches because of `staleTime: 30 * 1000` expiry (window refocus, `invalidateQueries`, or `refetchOnWindowFocus` defaults), it executes `queryFn` against the **latest** render's `testInput` — which may have changed since the queryKey was constructed.

Concrete failure:
1. User types IP `192.168.0.10`, clicks 测试. queryKey=`[..., {ip:"192.168.0.10"}]`, queryFn submits `{ip:"192.168.0.10"}`. Result cached.
2. User types `10.0.0.1` (state updates, component re-renders) — `stableQueryKey` updates too, so a *new* queryKey forms. **But** the previous query is still in cache, and any `invalidateQueries({queryKey: queryKeys.reconciliation.matchTest(stableQueryKey)})` from elsewhere uses the new key — fine.
3. **Real bug:** `handleTest` calls `setTestInput(...)` then immediately `await refetch()`. React batches state updates, so when `refetch()` fires, the queryFn closure may still reference the *pre-update* `testInput` because React Query's observer captures `queryFn` at subscription time and React hasn't re-rendered yet. The first refetch after `setTestInput` returns the **previous** IP's result.

This is the same class of bug as the project memory `server-sort-loadfunc-param-drop` and `duty-module-user-field-gotchas` (closure captures stale state).

**Fix:**
Pass the params via the queryFn context or use the queryKey as source of truth:
```tsx
const { data, isFetching, refetch } = useQuery({
  queryKey: queryKeys.reconciliation.matchTest(stableQueryKey),
  queryFn: ({ queryKey }) => {
    const [, , params] = queryKey as [string, string, TestInput];
    return reconciliationApi.exceptionRule.test(params);
  },
  enabled: !!testInput.ip,
  staleTime: 30 * 1000,
});

const handleTest = async () => {
  const values = await form.validateFields();
  setTestInput({
    ip: values.ip.trim(),
    userId: values.userId?.trim() || undefined,
    deptId: values.deptId?.trim() || undefined,
  });
  // DO NOT call refetch() — setTestInput changes queryKey, which auto-triggers fetch.
};
```

---

### CR-05: Excel import post-processing UPDATE not scoped to a single row — affects multiple rows when names collide

**File:** `internal/services/asset/reconciliation_exception.go:635-640`
**Issue:**
```go
if err := s.db.WithContext(ctx).
    Table("sys_reconciliation_exception").
    Where("name = ? AND deleted_at IS NULL", name).
    Updates(updates).Error; err != nil {
```
This UPDATE matches by `name` alone. Two scenarios cause silent data corruption:

1. **Same `name` in two rows of the import Excel:** The `excel_service` UpsertKey on `name` should prevent this, but `PartialUpdate: true` (excel_config.go:315) means each row may insert-or-update by name, producing one final row per name. **However**, if a rule with the same name was created *between* the excel_service `ImportData` call and this post-processing loop (race condition with another admin via the UI Create endpoint), the post-process UPDATE will affect both rows, applying the same `scope_id`/`conflict_types` to the unrelated rule.

2. **Soft-deleted rule with same name:** migration_168's partial unique index is `WHERE deleted_at IS NULL`, so a soft-deleted rule named "研发部测试网段" + a new active rule with the same name can coexist. The WHERE clause filters `deleted_at IS NULL` so this case is safe — but only because of that filter; the broader issue is that `name` is not globally unique without the soft-delete predicate.

3. **excel_service `PartialUpdate: true` + `UpsertKey` interaction:** When `PartialUpdate` is true and the upsert key matches an existing row, only non-empty fields are written. The post-process UPDATE then unconditionally sets `conflict_types` and `exception_actions` from the CSV — bypassing the partial-update protection. If a row in the Excel has empty `conflictTypes` (meaning "match all types"), the post-process writes `pq.StringArray(nil)` → NULL, which then fails the implicit semantics ("empty ConflictTypes matches all B-F").

**Fix:**
Scope the UPDATE to a single row by including the UUID returned from excel_service (requires `ImportData` to return affected IDs, not just names), or add `LIMIT 1` + id ordering, or change the post-process to fetch the latest inserted row by `name ORDER BY created_at DESC LIMIT 1` and update by ID.

```go
var id string
if err := s.db.WithContext(ctx).
    Table("sys_reconciliation_exception").
    Where("name = ? AND deleted_at IS NULL", name).
    Order("created_at DESC").
    Limit(1).
    Pluck("id", &id).Error; err == nil && id != "" {
    s.db.WithContext(ctx).
        Table("sys_reconciliation_exception").
        Where("id = ?", id).
        Updates(updates)
}
```

---

## Warnings

### WR-01: `ImportFromExcel` post-processing failure is silently swallowed — `fmt.Printf` and continue hides data corruption

**File:** `internal/services/asset/reconciliation_exception.go:566-570, 611, 639`
**Issue:**
```go
if err := s.postProcessImportedRules(ctx, file, result.AffectedKeys); err != nil {
    // 后处理失败不阻断主流程(基础字段已写库),仅 log
    fmt.Printf("[reconciliation:ImportFromExcel] 后处理失败 (基础字段已写库): %v\n", err)
}
```
The `fmt.Printf` goes to stdout (not structured logging), bypassing the project's `logrus`/`applogger` infrastructure. Worse, individual row failures inside `postProcessImportedRules` are also swallowed:
```go
if err := s.db.WithContext(ctx)...Updates(updates).Error; err != nil {
    fmt.Printf("[reconciliation:ImportFromExcel] UPDATE 失败 name=%s: %v\n", name, err)
}
```
The ImportResult returned to the caller will not reflect these failures, so the admin UI shows "导入成功 N 条" while scope_id/conflict_types are silently unset. The comment "运维可在 admin 页手动补 scope_id" understates the impact — `scope_type='dept'` rules with `scope_id=NULL` will never match (per matcher.go:159-162 the `*r.rule.ScopeID != assetUserID` check dereferences a nil pointer or compares against empty).

Wait — actually in Go, `*r.rule.ScopeID` would panic on nil deref. The matcher code reads `r.rule.ScopeID != nil && ...` so nil is safe. But a `dept` scope rule with `scope_id=NULL` will *never match anything*, silently disabling the rule. Operator sees rule "imported successfully" but it has no effect.

**Fix:**
- Replace `fmt.Printf` with `logrus.WithError(err).WithField("name", name).Error("...")`.
- Surface post-process failures via `ImportResult.ErrorCount` or a new `PartialFailure` field, OR roll back the entire import transaction if post-process fails on any rule (preferred for data-integrity reasons — partial imports are worse than clean failures).

### WR-02: Export/Template routes lack RequirePermissions — export of all rules (incl. reason text) leaks to any authenticated user

**File:** `internal/api/v1/asset/reconciliation_exception_router.go:71-75`
**Issue:**
```go
r.POST("/exception-rule/import",
    middleware.RequirePermissions([]string{"asset:reconciliation:exception:create"}, core),
    handler.ImportRules)
r.POST("/exception-rule/export", handler.ExportRules)        // ← no perm check
r.POST("/exception-rule/template", handler.DownloadTemplate) // ← no perm check
```
The router comment justifies this as "audit-only scenario" parallel to `list/:id`. But:
1. `Export` returns the full rule list including `reason` text (free-form, may contain sensitive context like " vulnerability in 192.168.0.0/16 finance subnet") and `scope_id` UUIDs. Bulk export is more dangerous than paginated list (which the UI gates).
2. `Template` is benign but co-listed for consistency.
3. The T-44-02 mitigation locking down `create/update/delete` is inconsistent if `export` (bulk data exfiltration) is left open.

**Fix:** Add `asset:reconciliation:exception:list` (or a dedicated `:export`) perm to Export. Template can stay open (it returns an empty template, no data).

### WR-03: `matchException` does not distinguish "empty ConflictTypes = match all" from "ConflictTypes is empty array but rule was supposed to filter" — silent Type A leakage

**File:** `internal/services/asset/reconciliation_exception_matcher.go:144-155`
**Issue:**
```go
// ConflictTypes 空数组匹配全部 B-F(D-R3-A3-02)
if len(r.rule.ConflictTypes) > 0 {
    found := false
    for _, ct := range r.rule.ConflictTypes { if ct == conflictType { found = true; break } }
    if !found { continue }
}
```
The comment claims "空数组匹配全部 B-F", but the caller `DetectLayer3` only invokes `matchExceptionWithSeverity` for non-A conflict types (line 235 short-circuits Type A to `skipped++`). So the matcher is never called with `conflictType == "A"` — but the matcher does not enforce this contract.

If a future caller invokes `matchException` with `conflictType="A"` (e.g., a test helper, or a refactor that moves the Type A check), the empty-ConflictTypes rule will silently match Type A (healthy) assets, applying silence/no_alert to healthy records. There is no defensive `if conflictType == "A" { return "", nil, "", false }` guard.

**Fix:**
Add a defensive check at the top of `matchExceptionWithSeverity`:
```go
if conflictType == "A" || conflictType == "" {
    return "", nil, "", false  // Type A 是健康状态,不参与例外匹配(防御性)
}
```
Plus a test `TestMatchExceptionRejectsTypeA`.

### WR-04: `Update` allows `expires_at` to be set back to NULL with no audit reason — operator can silently "un-expire" a rule

**File:** `internal/services/asset/reconciliation_exception.go:413-417`
**Issue:**
```go
if req.ExpiresAt != nil {
    updates["expires_at"] = req.ExpiresAt
} else {
    updates["expires_at"] = nil  // ← silently clears expiry
}
```
The Create path uses `req.ExpiresAt` as a pointer (`*time.Time`), so omitting it leaves the column NULL. The Update path **forces** `expires_at` to NULL when the request omits the field. This is asymmetric:

1. An admin editing only the `reason` field of an expiring rule will silently clear `expires_at`, making the rule permanent.
2. The frontend `ExceptionRuleForm` only sets `expiresAt` when the user picks a date — `values.expiresAt` is `undefined` for "no change" and `null` for "clear". The handler cannot distinguish these.
3. There is no operlog context distinguishing "set expiry" from "clear expiry" — the same `OperTypeUpdate` is recorded either way.

This violates the spirit of D-R3-A4-03 (audit-chain for rule lifecycle).

**Fix:**
Use a `*time.Time` with a `setExpiresAt bool` discriminator, or change the request shape to `ExpiresAt *time.Time` with explicit "unset" semantics via a separate `ClearExpiry bool` field. At minimum, only set `expires_at` when the request explicitly provides it:

```go
type UpdateExceptionRuleRequest struct {
    ...
    ExpiresAt      *time.Time `json:"expiresAt,omitempty"`
    ClearExpiresAt bool       `json:"clearExpiresAt,omitempty"`
}
```

### WR-05: `invalidateCache()` is a documented no-op — Phase 42 INFRA-04 CacheProvider was never wired, creating a window of stale reads

**File:** `internal/services/asset/reconciliation_exception.go:521-530`
**Issue:**
```go
func (s *reconciliationExceptionServiceImpl) invalidateCache() {
    // TODO(R3+): core.Cache.Delete(ctx, CacheKeyReconciliationExceptionRuleList)
    // ...
}
```
The TODO acknowledges the gap, but the surrounding doc claims "数据写入 DB 后,下次 List 查询会从 DB 读取,缓存陈旧窗口在 cron 周期内可接受". This is only true if:
1. There is no L1/L2 cache layer in front of the List query.
2. The `DetectLayer3` engine's in-memory `preloadActiveRules` snapshot is refreshed every cron cycle (6 min).

For #2 — `DetectLayer3` is called every 6 min (`@every 6m` in `reconciliation_tasks.go:133`), so an admin creating a rule via UI may wait up to 6 minutes before the new rule takes effect in detection. This is documented as "cron 周期内可接受" but it means a `silence` rule created in response to an active attack may not silence the next 6-min detection cycle.

**Fix:** Either wire `CacheProvider` (small lift — add to constructor), OR add an explicit log warning when `invalidateCache` is called so ops can manually force-refresh if needed. Document the 6-min staleness SLA in CLAUDE.md.

### WR-06: `apperrors.BadRequest` is dead-imported via `var _ = apperrors.BadRequest` — masks future unused imports and confuses linters

**File:** `internal/services/asset/reconciliation_detection.go:17, 394-395`
**Issue:**
```go
import (
    ...
    apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
)
...
// 防止 unused import 警告
var _ = apperrors.BadRequest
```
The import is unused in the file. The `var _` trick is a code smell — it signals "this was imported for a reason that has since been removed". Either:
1. The `apperrors` package was meant to be used (e.g., return `apperrors.BadRequest("...")` instead of plain `errors.New`), or
2. The import should be removed entirely.

Leaving it in invites future contributors to add more dead imports under the same pattern.

**Fix:** Remove the import and the `var _` line. If typed errors are intended, refactor `DetectLayer3` to return `apperrors.BadRequest` for input-shape errors.

### WR-07: `TestCreateWorkorderNoWorkorderFilterStatic` allows false negative — the `NotContains` check is too lax

**File:** `internal/scheduler/reconciliation_tasks_test.go:148-158`
**Issue:**
```go
assert.Contains(t, src,
    "applied_actions IS NULL OR 'no_workorder' != ANY(applied_actions)",
    "createWorkorderBySeverity 的 WHERE 必须含 IS NULL 兜底(BLOCKER-4:防 applied_actions=NULL 漏转)")
// 反向校验:不允许裸 != ANY 不带 IS NULL(防止回退)
assert.NotContains(t, src,
    "AND 'no_workorder' != ANY(applied_actions))",
    "禁止裸 'no_workorder' != ANY(applied_actions)(BLOCKER-4 三值逻辑漏转风险)")
```
The `NotContains` searches for `AND 'no_workorder' != ANY(applied_actions))` (note the trailing `)`). The actual production code uses:
```go
"AND (applied_actions IS NULL OR 'no_workorder' != ANY(applied_actions))"
```
The `NotContains` would pass even if a future edit introduced `AND ('no_workorder' != ANY(applied_actions))` (the IS NULL clause removed but parens retained) — because the search pattern requires `AND ` prefix without `(`. The reverse-guard is too narrow.

**Fix:**
Use a stronger pattern or restructure as a parser-based check:
```go
// 提取 createWorkorderBySeverity 函数体
body := extractFuncBody(src, "createWorkorderBySeverity")
assert.Contains(t, body, "applied_actions IS NULL")
assert.Contains(t, body, "'no_workorder' != ANY(applied_actions)")
// 验证 IS NULL 出现在 != ANY 之前(顺序保证 OR 短路正确)
idxNull := strings.Index(body, "applied_actions IS NULL")
idxAny := strings.Index(body, "'no_workorder' != ANY")
assert.Greater(t, idxNull, -1)
assert.Greater(t, idxAny, idxNull, "IS NULL 兜底必须在 != ANY 之前")
```

### WR-08: `exception-rules/index.tsx` `columns` useMemo omits `scopeType`/`isActive`/edit-related deps — captures stale closures

**File:** `xingran-react-frontend/src/pages/asset/reconciliation/exception-rules/index.tsx:229-350`
**Issue:**
```tsx
const columns = useMemo<ColumnsType<ExceptionRuleItem>>(
  () => [ ... references deleteMutation in onConfirm ... ],
  [deleteMutation]  // ← only deleteMutation; missing createMutation, updateMutation
);
```
The columns array includes the "编辑" button which calls `setModalState({ open: true, editValues: {...record...} })` — `setModalState` is stable so OK. But the `onSubmit` (passed to `ExceptionRuleForm`) calls `updateMutation.mutateAsync` / `createMutation.mutateAsync` based on `modalState.editValues?.id`. If `createMutation` or `updateMutation` identity changes between renders (React Query may re-create mutation instances on queryClient changes), `handleSubmit` closes over the stale instance.

`handleSubmit` is `useCallback([modalState.editValues, createMutation, updateMutation])` so it does include them — but the `columns` memo only lists `deleteMutation`, meaning if the edit button's `onClick` handler ever captured `updateMutation` directly (currently it doesn't, but a future edit might), the lint would not catch it.

**Fix:** Either disable the lint rule explicitly with a comment, or include all referenced mutations in the deps array. As-is, it works by accident because the onClick only calls setState.

### WR-09: `baseline/compare` failure path maps ALL errors to 400 — DB outage masquerades as "no baseline"

**File:** `internal/api/v1/asset/reconciliation_exception_handler.go:248-262`
**Issue:**
```go
result, err := h.baselineSvc.Compare(c.Request.Context())
if err != nil {
    // 无 baseline 时返回 400,前端 Alert 引导运维先记录基线(BLOCKER-3 可观察条件)
    response.Error(c, http.StatusBadRequest, err.Error())
    return
}
```
The service returns errors for two distinct conditions:
1. "未找到基线快照,请先调用 Snapshot 记录基线" (legitimate 400)
2. "查询 baseline 失败: %w" / "解析 baseline JSON 失败: %w" / "统计总异常数失败: %w" (these are 500-class errors)

Mapping #2 to 400 means: during a DB outage, the dashboard will render the "请先记录基线" guidance Alert instead of an error state, misleading ops to re-record a baseline they already have.

**Fix:**
```go
if err != nil {
    if strings.Contains(err.Error(), "未找到基线快照") {
        response.Error(c, http.StatusBadRequest, err.Error())
        return
    }
    response.Error(c, http.StatusInternalServerError, err.Error())
    return
}
```
Or better: have the service return a typed error (e.g., `apperrors.NotFound`) that the handler can `errors.As` against.

### WR-10: `reconciliation_baseline.go` Snapshot uses `Pluck("id", &existingIDs)` then re-queries — TOCTOU race between Snapshot's existence check and Create/Update

**File:** `internal/services/asset/reconciliation_baseline.go:108-143`
**Issue:**
```go
var existingIDs []string
if err := s.db.WithContext(ctx).Table("sys_config").
    Where("config_key = ? AND deleted_at IS NULL", BaselineConfigKey).
    Limit(1).Pluck("id", &existingIDs).Error; ...

if len(existingIDs) > 0 && existingIDs[0] != "" {
    // Update by id
} else {
    // Create new
}
```
Between the SELECT and the INSERT, another concurrent Snapshot call (e.g., two admins clicking the button, or a snapshot triggered during cron) can insert a row with the same `config_key`. The second INSERT will violate the unique index on `config_key` (the model declares `gorm:"uniqueIndex;size:100"`), returning SQLSTATE 23505.

The code has no `ON CONFLICT` clause and no error retry. Concurrent snapshots will fail noisily rather than converging.

**Fix:** Use PG `ON CONFLICT (config_key) DO UPDATE` (GORM `clause.OnConflict`) for atomic upsert. Or wrap in a transaction with `SELECT ... FOR UPDATE`.

### WR-11: `MergeActions` step 2 `severity_override` takes "lowest", but step 1 `skip_severity` already downgraded — the doc says "取最低" but the comparison uses post-skip severity

**File:** `internal/services/asset/reconciliation_exception_matcher.go:215-231`
**Issue:**
The D-R3-A2-02 doc string at line 187 reads "原始severity --skip_severity--> 降一级 --severity_override--> 取更宽". The implementation:
```go
sev := originalSeverity
if skipTriggered { sev = applySkipSeverity(sev) }  // step 1: downgraded sev

for _, r := range matched {
    if r.rule.SeverityOverride != nil {
        ...
        if ovLevel < sevLevel { sev = ov }  // compares override against post-skip sev
    }
}
```
This is correct per the doc, but the test `TestMergeActionsSkipThenOverride` only validates a single matched rule. With multiple matched rules where some have `skip_severity` and others have varying `severity_override`, the **order** of override application does not matter (min is min), but the comparison baseline is the post-skip severity. If a future contributor "optimizes" by computing skip after override, the result changes. There is no test for "multiple override rules with skip" to lock the ordering.

**Fix:** Add a test case:
```go
func TestMergeActionsMultiRuleSkipAndMultiOverride(t *testing.T) {
    matched := []compiledRule{
        {rule: makeRule("r1", "...", []string{"skip_severity"}, strPtr("medium"), "global", nil, nil)},
        {rule: makeRule("r2", "...", []string{"silence"}, strPtr("low"), "global", nil, nil)},
    }
    // critical --skip--> high; then min(high, medium, low) = low
    _, sev, _ := mergeActions("critical", matched, "B")
    assert.Equal(t, "low", sev)
}
```

---

## Info

### IN-01: `NewReconciliationExceptionHandler` does not inject `core` — caller must chain `.WithCore(core)`

**File:** `internal/api/v1/asset/reconciliation_exception_handler.go:27-29`
**Issue:** The constructor returns a handler with `h.core == nil`. If a caller forgets `.WithCore(core)`, any write op (Create/Update/etc.) will nil-pointer-deref on `h.core.OperLogService`. The router does call `.WithCore(core)` (line 35), so production is safe, but tests constructing the handler directly are at risk. Defensive nil-check on `h.core` in each write handler would prevent silent crashes.

### IN-02: `reconciliation_exception_handler.go` `var _ = http.StatusOK` masks unused import

**File:** `internal/api/v1/asset/reconciliation_exception_handler_test.go:220-221`
**Issue:** Same anti-pattern as WR-06. The `net/http` import is retained via a placeholder. Either remove or use the constant meaningfully.

### IN-03: `Migration 174` SQL `DO $$ ... END $$` blocks are not wrapped in a transaction — partial application possible

**File:** `internal/core/db/migrations/migration_174_reconciliation_exception_gist.go:48-107`
**Issue:** Each `DO $$ ... END $$` is its own implicit transaction. If the GiST index creation succeeds but CHECK constraint #2 fails, the migration returns an error but the GiST index remains — re-running the migration is idempotent (IF NOT EXISTS), but partial state is confusing during rollback. Wrap all three statements in a single `BEGIN; ... COMMIT;` or document the partial-success recovery path.

### IN-04: `cleanupExpiredExceptionsDirect` ignores `ctx.Done()` — long-running shutdown may block

**File:** `internal/scheduler/reconciliation_tasks.go:255-264`
**Issue:** The function takes `ctx` and passes it via `WithContext`, but the UPDATE itself is a single statement that respects ctx cancellation. However, the cron dispatcher (`case "cleanupExpiredExceptions":`) does not log `ctx.Err()` if the context is cancelled mid-execution. Minor — single UPDATE is fast — but worth a defensive `if err := ctx.Err(); err != nil { return 0, err }` at the top.

### IN-05: `MatchTestPanel` does not memoize `ruleColumns` — re-created on every render

**File:** `xingran-react-frontend/src/components/asset/reconciliation/MatchTestPanel.tsx:107-163`
**Issue:** `ruleColumns` is rebuilt every render. The columns close over no state (all renders are pure), so `useMemo([])` would prevent unnecessary Table re-renders. Minor perf issue, not a correctness bug.

### IN-06: `queryKeys.reconciliation.matchTest` signature expects `{ip, userId?, deptId?}` but `stableQueryKey` is the full `testInput` — type drift risk

**File:** `xingran-react-frontend/src/lib/queryKeys.ts:75-77`, `MatchTestPanel.tsx:71-79`
**Issue:** The queryKey factory accepts a typed `TestInput`-shaped object, but the panel passes the live `testInput` state. If `TestInput` grows a new field (e.g., `conflictType?: string`), the queryKey type changes silently. Not a bug today; future risk.

### IN-07: `dashboard/index.tsx` reads `reductions.exceptions_reduction_pct` via two-path fallback (direct field OR `reductions` map) — comment admits backend may swap shape

**File:** `xingran-react-frontend/src/pages/asset/reconciliation/dashboard/index.tsx:99-114`
**Issue:** The code:
```tsx
const excPct =
  (d as unknown as { exceptions_reduction_pct?: number }).exceptions_reduction_pct ??
  (d.reductions?.exceptions_reduction_pct as number | undefined) ??
  0;
```
The double-cast (`as unknown as`) and dual lookup indicates the backend response shape was uncertain at write time. The Go service returns `BaselineCompareResult` with `exceptions_reduction_pct` as a top-level field (snake_case from Go JSON tags). The `reductions` map path is dead code. Confirm the contract in `assetApi.ts` and remove the fallback.

---

## Structural Findings (fallow)

No `<structural_findings>` block was provided in the review prompt, so no fallow substrate was integrated. Cross-module facts verified manually:

- `internal/services/operations/excel_config.go`: `reconciliationExceptionRule` config confirmed present with 9 columns in correct order; `name` column has `UpsertKey: true` + `DBField: "name"` (verified at lines 317-326).
- `internal/models/reconciliation.go:50`: `AppliedActions pq.StringArray` confirmed no default tag → PG INSERT leaves NULL → BLOCKER-4 IS NULL fallback is required and present.
- `internal/api/router.go:943`: `SetupReconciliationExceptionRouter(assetReconciliation, core)` confirmed registered with `core` injection; no orphan code paths.
- `internal/models/config.go`: `ConfigValue string gorm:"size:500"` confirmed — CR-01 valid.

---

_Reviewed: 2026-06-28_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
