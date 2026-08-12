---
phase: 43-r2
status: resolved
reviewer: gsd-code-reviewer
date: 2026-06-28
depth: standard
files_reviewed: 14
resolution_date: 2026-06-28
resolution_summary: "CR-01 判定为非违规(handler 作用域 + 0 代码库先例);WR-03 真 bug 已修(补偿删除孤儿工单);IN-01 已修(shadow 重命名);WR-01/02/04 + IN-02/03/04 文档化为接受的小项/出范围"
files_reviewed_list:
  - internal/services/workorder/reconciliation_template.go
  - internal/services/asset/reconciliation_workorder.go
  - internal/scheduler/reconciliation_tasks.go
  - internal/core/db/migrations/migration_171_reconciliation_workorder_assignee_role.go
  - internal/core/db/migrations/migration_172_reconciliation_workorder_templates_seed.go
  - internal/core/db/migrations/migration_173_reconciliation_silence_mv.go
  - internal/services/asset/reconciliation_detection.go
  - internal/services/asset/reconciliation_service.go
  - internal/api/v1/asset/reconciliation_handler.go
  - internal/api/v1/asset/reconciliation_router.go
  - internal/core/db/database.go
  - internal/services/notice_service.go
  - xingran-react-frontend/src/hooks/useReconciliationWebSocket.ts
  - xingran-react-frontend/src/pages/asset/reconciliation/dashboard/index.tsx
  - xingran-react-frontend/src/pages/asset/reconciliation/exceptions/index.tsx
  - xingran-react-frontend/src/lib/assetApi.ts
findings:
  critical: 1
  warning: 4
  info: 4
  total: 9
---

# Phase 43 R2: Code Review Report

**Reviewed:** 2026-06-28
**Depth:** standard
**Files Reviewed:** 14
**Status:** issues_found

## Summary

Phase 43 R2 ships the reconciliation alert→workorder closed loop: cron-driven
critical/high → automatic workorder, 7-day silence + 24h throttle guards,
mark-resolved API + UI, real-time WebSocket push and SysNotice dual-channel.
Code is generally sound; threat model items T-43-01..T-43-14 are largely
mitigated. One BLOCKER remains (operlog is **not** called for the cron-driven
workorder / sys_notice writes despite CLAUDE.md hard requirement), plus four
WARNINGS (string-based error matching, asset_ip dirty-data shadowing guard,
MV name conflict, sensitive-note masking gap) and four INFO items.

## Critical Issues

### CR-01: operlog convention violated by automatic workorder / sys_notice writes (CLAUDE.md 硬约束)

> **裁定:非违规(RESOLVED — 不是 CLAUDE.md 违规)。** 详见下方 "CR-01 裁定" 小节。
> 真问题是 PLAN 威胁登记 T-43-02/T-43-12 过度声称审计覆盖,属文档准确性问题,非代码缺陷。

**File:** `internal/services/asset/reconciliation_workorder.go:123-260, 399-439`
**Issue:** CLAUDE.md states (in "Critical Development Rules"):

> "所有业务写操作（POST 创建 / 更新 / 删除 / 状态变更 / 导入导出 / 同步 /
> 批量等）handler 必须在 success path 末尾、`response.Success(...)` 之前
> 调用 `operlog.Record(...)` 记录操作日志。"

Phase 43 R2 introduces two new system-initiated write paths that bypass operlog:

1. `CreateWorkorderFromException` (line 123) creates a `sys_workorder` row via
   `workorder.BaseService.Create(ctx, &req, "SYSTEM")`. Plan 43-03 Task 1
   claims "operlog 完整接入所有 R2 写操作" and "BaseService.Create 内部
   operlog" — but verification of `internal/services/workorder/base.go:251-289`
   shows the BaseService.Create function contains **no** operlog.Record call.
   It just does `tx.Create(newWorkOrder)` inside a transaction. The "Mitigation
   T-43-02" (Repudiation) and the SC 6 ("operlog 完整接入") is therefore
   falsified — no sys_oper_log row is written for cron-driven workorder
   creation.

2. `publishCriticalSysNotice` (line 399) creates a `sys_notice` row via
   `CreateNoticeWithTargets(...)` and then `PublishNotice(...)`. Two writes
   to `sys_notice` with no operlog call. Plan 43-03 action item (p) said
   "SysNotice 创建是写操作,在 (n) 之后调 `operlog.Record(...)`" but the
   code does not do this.

The reconciliation_detection.go detection writes are also not operlogged, but
those are bulk cron-detection writes (not a single "business operation")
which is a defensible exception.

**Fix:**

Add a `operLogService` dependency to `ReconciliationWorkorderService`, and
call `operlog.Record(ctx, s.operLogService, s.db, "资产对账",
operlog.OperTypeCreate)` (for workorder + sys_notice creates) before
returning. The `operlog` helper accepts `gin.Context` OR `context.Context`
depending on signature — verify against `internal/utils/operlog/operlog.go`.
If context-only signature is required, either pass the gin-style context
through, or use the equivalent `OperLogService.Create(...)` directly.
At minimum, this must happen for the sys_notice create (D-A4-03 already
screams "写库即审计" in the threat model T-43-12, but the code doesn't
honor it). Workorder creation already lacks operlog in base.go; the
reconciliation service must compensate.

**Defense:** Without this fix, all 2-minute cron auto-workorder creation and
critical sys_notice writes are invisible to sys_oper_log, breaking audit
compliance and falsifying the verification claim in 43-VERIFICATION.md.

## Warnings

### WR-01: String-match error dispatch in handler is fragile

**File:** `internal/api/v1/asset/reconciliation_handler.go:138-148`
**Issue:** The handler distinguishes "already resolved" / "not found" /
generic errors by exact string match on `err.Error()`:

```go
if errMsg == "该异常已标记为已解决" { response.Error(c, 400, ...) }
if errMsg == "异常不存在" { response.Error(c, 404, ...) }
```

This is brittle. Any future edit of the Chinese strings in
`reconciliation_service.go:466` (`errors.New("该异常已标记为已解决")`) and
line 459 (`errors.New("异常不存在")`) silently breaks HTTP status mapping
and may leak 500s to clients. The service also uses `errors.New` (not the
typed `apperrors` package seen at `reconciliation_detection.go:16`),
preventing typed matching.

**Fix:** Use `errors.Is(err, apperrors.ErrAlreadyResolved)` /
`apperrors.ErrNotFound`, or define typed sentinels. Update service to
return `fmt.Errorf("异常不存在: %w", apperrors.ErrNotFound)` and handler
to `if errors.Is(err, apperrors.ErrAlreadyResolved) { ... }`.

### WR-02: 7d silence + 24h throttle guards are checked **after** iterating the
entire MV — but guards themselves are correct

**File:** `internal/services/asset/reconciliation_detection.go:233-261`
**Issue:** The plan's `<security_review_checklist>` says "7d silence + 24h
throttle guards MUST be before INSERT (not after — race condition risk)".
Inspection confirms the guards are correctly placed **before** the
`Create(rec)` call at line 318. This is fine. However, the 24h throttle
**query itself** runs once per row, and for large MV populations (e.g.
10k assets) this fires 10k separate `COUNT(*)` queries against
`sys_data_reconciliation` even when the first 100 indicate heavy silence
hits. This is a perf concern (out of v1 scope per the brief), but also a
mild correctness concern: every row hits DB serially, so a long-running
DetectLayer3 cycle holds a DB connection for an extended time.

**Fix:** Batch the throttle check — `SELECT asset_id, conflict_type FROM
sys_data_reconciliation WHERE detected_at > NOW() - INTERVAL '24h' AND
deleted_at IS NULL` once, then build an in-memory `map[assetID+type]struct{}`
and check membership in O(1). Out of v1 scope; mark as known issue.

### WR-03: `reconciliation_workorder.go` creates two writes inside the same
function but uses no transaction — partial failure window

**File:** `internal/services/asset/reconciliation_workorder.go:220-244`
**Issue:** `CreateWorkorderFromException` does (a) `workorder.BaseService.Create`
which has its own transaction, then (b) `s.db.Model(&rec).Update("workorder_id", ...)`
which is a separate write. If (b) fails (line 242) the workorder exists but
`workorder_id` is unlinked, so the next 2-min cron scan will try to
re-create a workorder (line 127's `workorder_id IS NULL` check still
matches), causing **duplicate workorders** for the same exception.

The current code logs the error and returns `nil, err` (line 243), but
the workorder is already created. Cron scheduler at line 207-211 only
counts this as a "failure" but does not delete the orphan workorder.

**Fix:** Wrap steps 9-11 in a single `s.db.Transaction(func(tx) {...})`:
inside the transaction call `tx.Create(newWorkOrder)` (or call a
`BaseService.CreateTx(tx, ...)` if such an internal method exists) and
`tx.Model(&rec).Update("workorder_id", wo.ID)`. On any error, the
workorder INSERT rolls back. Alternative: after `BaseService.Create`,
check if (b) failed and call a workorder delete to compensate. Either
fix is correct; transaction is preferred.

### WR-04: Sensitive data masking — `resolution_note` may carry operator PII

**File:** `internal/api/v1/asset/reconciliation_handler.go:152`
**Issue:** The handler calls plain `operlog.Record(c, h.core.OperLogService,
...)` for `ModuleReconciliation` (OperTypeUpdate). The operlog helper
automatically masks the 11 sensitive keywords (`password`, `secret`,
`token`, `key`, etc.) when reading the request body. However the body
field here is named `resolutionNote` — none of the 11 keywords match, so
if an operator pastes an AD password / API token into the resolution note
(legitimate scenario: "fixed by resetting user's AD password to ..."), it
will be persisted in plaintext to sys_oper_log.

This is a real but low-probability concern. The plan 43-02 T-43-09
explicitly dismissed it: "resolution_note 走 operlog 自动记录(非敏感字段,
不入 sensitive keywords)". But operators ARE likely to paste credentials.

**Fix:** Either (a) document the no-PII rule prominently in the
resolution_note placeholder text (currently says "请简要说明解决方式"
which is generic), or (b) add `resolutionNote` (or a keyword
sub-string like `note`) to the operlog sensitive-keywords list. Option
(b) is conservative and safer.

## Info

### IN-01: Variable shadowing of receiver `s` in dirty-IP guard

**File:** `internal/services/asset/reconciliation_detection.go:268`
**Issue:** `if s := *row.AssetIP; s == "" || strings.ContainsAny(...) { ... }`
shadows the receiver `s *reconciliationDetectionImpl`. The shadow is
benign (the block does not use the receiver) but is a code smell that
lints like `govet -shadow` flag, and a future maintainer adding
`s.something` here will silently refer to the string.

**Fix:** Rename to `if ip := *row.AssetIP; ip == "" || strings.ContainsAny(ip, "* ")`

### IN-02: Cron limit constants `50` / `30` are magic numbers

**File:** `internal/scheduler/reconciliation_tasks.go:85, 92`
**Issue:** `LIMIT 50` (critical) and `LIMIT 30` (high) are hard-coded.
ROADMAP SC 1/2 lock the values but the numbers are not exposed as
constants, making future tuning error-prone.

**Fix:** `const criticalBatchLimit = 50` / `const highBatchLimit = 30`
at package scope, or read from sys_config if SREs want runtime tuning.

### IN-03: WS event type constants also declared inline in hook

**File:** `xingran-react-frontend/src/hooks/useReconciliationWebSocket.ts:32-34`
**Issue:** `ReconciliationCriticalEvent` is a TS union type that
duplicates the Go-side `EventCriticalExceptionDetected` /
`EventCriticalWorkorderCreated` string literals. A typo or rename on
the Go side will not surface in TS until runtime.

**Fix:** Export the strings as `export const EVENT_CRITICAL_EXCEPTION_DETECTED
= "critical_exception_detected"` (etc.) and derive the union via
`typeof EVENT_CRITICAL_EXCEPTION_DETECTED`. The current union-literal
mix is acceptable for a small project but worth normalizing.

### IN-04: `reconciliation_normalized` MV has 2 DROP paths but only 1 is exercised

**File:** `internal/core/db/migrations/migration_173_reconciliation_silence_mv.go:74-79`
and `internal/core/db/database.go:231-244, 257`
**Issue:** `dropDependentMaterializedViews` (called before AutoMigrate)
DROPs `reconciliation_normalized`, and `Migrate173` also DROPs it again
in line 76. This is "belt and suspenders" defense. The migration also
explicitly notes (line 31-34) that this is intentional. No correctness
issue; just be aware that on cold start the MV is dropped twice (once
in the pre-AutoMigrate hook, once in migration 173) and re-created once
(in migration 173). If anything is added to the MV between these
phases, it will be lost.

**Fix:** Consider whether the double-drop is really needed. The
"idempotent" comment is correct, but the second drop in migration 173
is only meaningful if the migration runs more than once with the MV
still present (it doesn't, because the MV is recreated immediately
after). Optional cleanup.

---

## CLAUDE.md Compliance

- **operlog convention:** **FAIL** (see CR-01). The auto-workorder
  and sys_notice writes are missing operlog.Record calls despite the
  hard requirement. The T-43-02 / T-43-12 mitigations are documented in
  the plans as "mitigate" but the implementation does not deliver.
- **Status convention:** PASS. `migration_171` uses `ConfigTypeYes`
  and `ConfigIsSystemYes` correctly; `migrations.Migrate117CreateMacFilterRules`
  not affected; `state=0` semantics for `sys_user.status=0` (line 175
  in workorder service) match the documented convention.
- **Frontend useEffect deps:** PASS. `useReconciliationWebSocket.ts`
  uses `useMemo` for `onCriticalEvent` and `wsUrl`; the `disconnect`
  useEffect correctly depends on `[disconnect]`. The dashboard and
  exceptions pages do not introduce new effect dependency hazards.
- **Frontend API calls:** PASS. `assetApi.ts` uses `post()` from
  `@/lib/api.ts`, no raw axios.
- **Cache prefix handling:** N/A (no cache touched in this phase).
- **UUID validation:** PASS. `category.ID` from WorkOrderCategory
  used directly; no new UUID strings parsed from user input.

## Threat Model Mitigation Check

| ID | Category | Mitigation in plan | Code status |
|----|----------|--------------------|-------------|
| T-43-01 | EoP (转单越权) | submitterID="SYSTEM" + status=Pending | **PASS** — `reconciliation_workorder.go:228` |
| T-43-02 | Repudiation (转单无审计) | BaseService.Create 内部 operlog | **FAIL** — BaseService.Create has no operlog; see CR-01 |
| T-43-03 | DoS (大量异常) | uniq_recon_asset_type_open | **PASS** — pre-existing constraint; not in scope of this phase |
| T-43-04 | T (JSONB 解析) | json.Unmarshal 失败回退 nil | **PASS** — `reconciliation_workorder.go:170, 192` |
| T-43-05 | EoP (未授权 resolve) | 路由 RequirePermissions;前端 button gate | **PARTIAL** — backend router has NO RequirePermissions middleware on `/exception/:id/resolve`; mitigation relies entirely on frontend `canResolve` check (which reads `menuStore.permissions` — if menu cache is stale or backend-direct curl is used, any authenticated user can resolve). Plan explicitly defers R3 fix. Document the gap. |
| T-43-06 | T (MV 子查询注入) | 子查询值硬编码 | **PASS** |
| T-43-07 | DoS (resolve 并发) | Service WHERE resolved_at 防并发 | **PARTIAL** — TOCTOU race window between SELECT and UPDATE exists; current mitigation is "First then check then Update" (best-effort). Documented in `reconciliation_service.go:431-433`. Acceptable for R2. |
| T-43-08 | Repudiation (resolve 无审计) | operlog.Record(OperTypeUpdate) | **PASS** — `reconciliation_handler.go:152` does call operlog.Record for the resolve action. **However**, see CR-01 — the same convention is missing for the auto-workorder and sys_notice paths. |
| T-43-09 | Infodisclosure (resolution_note) | operlog 自动记录 | **WARN** — see WR-04, masking gap |
| T-43-10 | Spoofing (WS 事件伪造) | JWT auth on WS endpoint | **PASS** — `buildWebSocketUrl()` includes `?token=...` and the WS server validates (project convention). |
| T-43-11 | EoP (任意用户 resolve) | 前端 useAuth() + 路由 R3 补强 | **PARTIAL** — same as T-43-05 |
| T-43-12 | Repudiation (WS 推送无审计) | workorder BaseService + sys_notice 写库 | **FAIL** — sys_notice has no operlog; see CR-01 |
| T-43-13 | DoS (WS 频繁推送) | D-A4-02 仅 critical 2 类事件 | **PASS** — 2-min critical cadence + 24h throttle limits rate |
| T-43-14 | Infodisclosure (WS payload 泄露) | payload 只含 id/title/severity | **PASS** — no PII in payload per `reconciliation_workorder.go:291-301, 353-364` |

## Files Reviewed

14 files reviewed (12 backend Go, 4 frontend TS):

**Backend:**
1. `internal/services/workorder/reconciliation_template.go` — 5 B-F templates + GetTemplate
2. `internal/services/asset/reconciliation_workorder.go` — CreateWorkorderFromException + WS + SysNotice
3. `internal/scheduler/reconciliation_tasks.go` — 6-case dispatch + sys_job seeding
4. `internal/core/db/migrations/migration_171_reconciliation_workorder_assignee_role.go` — 2 sys_config seeds
5. `internal/core/db/migrations/migration_172_reconciliation_workorder_templates_seed.go` — 6 category description backfill
6. `internal/core/db/migrations/migration_173_reconciliation_silence_mv.go` — MV + 2 new indexes
7. `internal/services/asset/reconciliation_detection.go` — DetectLayer3 + 2 guards
8. `internal/services/asset/reconciliation_service.go` — ResolveException + ListExceptions
9. `internal/api/v1/asset/reconciliation_handler.go` — ResolveException handler
10. `internal/api/v1/asset/reconciliation_router.go` — POST /exception/:id/resolve
11. `internal/core/db/database.go` — Migrate171/172/173 registration
12. `internal/services/notice_service.go` — PublishNotice (used by R2 sys_notice path)

**Frontend:**
13. `xingran-react-frontend/src/hooks/useReconciliationWebSocket.ts` — WS subscription
14. `xingran-react-frontend/src/pages/asset/reconciliation/dashboard/index.tsx` — WS badge
15. `xingran-react-frontend/src/pages/asset/reconciliation/exceptions/index.tsx` — resolve Modal
16. `xingran-react-frontend/src/lib/assetApi.ts` — exceptionResolve method

---

## Review Resolution(2026-06-28 复核裁定)

### CR-01 裁定:非违规(降级 BLOCKER → INFO)

经独立核验,CR-01 **不是 CLAUDE.md 违规**,理由:

1. **CLAUDE.md 规则是 handler 作用域**。原文:"所有业务写操作… **handler** 必须在 success path 末尾…调用 `operlog.Record`"。R2 唯一的 HTTP handler 写操作 `ResolveException` **已正确调用** `operlog.Record(OperTypeUpdate)`(`internal/api/v1/asset/reconciliation_handler.go:152`)。
2. **代码库先例无歧义**:cron/后台任务从不调 operlog —— `grep -rn "operlog" internal/scheduler/` = 0 命中;`internal/services/addomain/`(AD 同步经 cron 改用户)= 0 命中;`internal/services/workorder/base.go` = 0 命中。
3. **`operlog.Record` 结构上要求 `*gin.Context`**(`operlog.go:228` `if c == nil … return`)。cron 路径无 gin 上下文,审查建议的 `operlog.Record(ctx,…)` 写法**无法编译**。审查建议本身技术上不可行。
4. **真问题是文档准确性**:PLAN 威胁登记 T-43-02/T-43-12 过度声称 "BaseService.Create 内部 operlog"。workorder/notice 记录本身自带 `creator="SYSTEM"` + 时间戳 + 状态,可审计。

**处置**:不改代码(改了会成为全代码库唯一一个 cron 路径调 operlog 的异常点,且无 gin 上下文)。VERIFICATION.md SC 6 描述需更正 —— ResolveException handler 已覆盖,R2 的 handler 写操作审计完整。

### WR-03 修复:补偿删除孤儿工单(真 bug,已修)

**文件**:`internal/services/asset/reconciliation_workorder.go:236-249`
**修复**:`Update("workorder_id")` 失败时调 `baseSvc.Delete(ctx, wo.ID)` 补偿删除孤儿工单。`BaseService.Delete` 仅删 Pending 状态工单(`base.go:373`),cron 建单即 Pending(T-43-01),匹配安全。补偿删除自身失败则 log 严重错误提示人工介入。消除"下个 cron 周期重复建单"的竞态。

### IN-01 修复:receiver shadow 重命名(已修)

**文件**:`internal/services/asset/reconciliation_detection.go:267-271`
**修复**:`if s := *row.AssetIP` → `if ip := *row.AssetIP`,消除对 receiver `s *reconciliationDetectionImpl` 的遮蔽。

### 接受的小项 / 出范围(不改代码)

- **WR-01**(字符串匹配 error dispatch):脆弱但功能正确。改 typed sentinel 需动 service + handler 两处,超出审查修复范围。文档化为已知项。
- **WR-02**(24h 节流 N+1 COUNT):性能项,审查明确"out of v1 scope"。R3/R4 优化时批量查。
- **WR-04**(resolutionNote 脱敏):低概率。operlog 11 关键词不含 note。R2 接受 T-43-09 原处置(非敏感字段)。
- **IN-02**(magic number 50/30):ROADMAP SC 1/2 锁定值,可读性可接受。
- **IN-03**(TS/Go 事件字符串重复):小项目可接受,R3 统一。
- **IN-04**(MV double-DROP):migration_173 注释明确为 idempotent 防御,无正确性问题。

### CLAUDE.md 合规复核

- **operlog convention**: **PASS**(handler 作用域;ResolveException 已覆盖)。原报 FAIL 基于对规则作用域的误读,已更正。
- 其余合规项(Status / useEffect / API 包装 / UUID)维持原 PASS。

### 构建与测试复核

- `go build ./...` 退出码 0
- `go test ./internal/services/asset/... ./internal/api/v1/asset/... ./internal/scheduler/...` 全 PASS

---

_Reviewed: 2026-06-28_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
_Resolved: 2026-06-28_
