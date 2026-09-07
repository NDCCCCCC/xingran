---
phase: 59-observability-usage-log-fix
plan: 01
subsystem: api-key-auth
tags: [observability, api-key, gin, detached-context, P1-2, P2-b]
dependency_graph:
  requires: [phase-57-auth-chain-core-fix]
  provides: [OBSERV-01, OBSERV-02, OBSERV-03]
  affects: [internal/middleware/apikey.go, internal/services/usage_logger.go]
tech-stack:
  added: []
  patterns: [gin-c.Next()-after-capture, detached-context-write, applogger-errorf-failure]
key-files:
  created: []
  modified:
    - internal/middleware/apikey.go
    - internal/services/usage_logger.go
decisions:
  - "D-01: Success 字段派生为 `statusCode >= 200 && statusCode < 300`（仅 2xx 视为成功，3xx/4xx/5xx 一律 false）"
  - "D-02: logUsageAsync 内部用 `context.WithTimeout(context.Background(), 10*time.Second)` 做 DB 写入，忽略调用方 ctx 的取消信号（先例 pkg/cache/redis.go:601-605）"
  - "D-02a: middleware 在 c.Next() 之后同步调用 usageLogger.LogUsage，去掉 apikey.go 旧实现的 `go func(){}` 外层包装（LogUsage 内部已 go logUsageAsync）"
  - "D-04: DB 写入失败用 `applogger.Errorf(\"[USAGE_LOG] 写入失败 key=%s path=%s: %v\", req.APIKeyID, req.Path, err)` 替换 `_ = err` 静默吞错（先例 config_backup_service.go:247，severity 升级为 Errorf）"
metrics:
  duration: "~10 minutes (incl. one commit-fixup for unrelated-file contamination)"
  completed_date: 2026-08-13
  tasks_completed: 2
  tasks_total: 2
  files_changed: 2
---

# Phase 59 Plan 01: 可观测性 / 使用日志修复 Summary

**One-liner:** Move API Key 使用日志的记录点到 `c.Next()` 之后并填真实 `StatusCode`/`Duration`/`Success`，同时让 `logUsageAsync` 改用 detached context 写 DB 并用 `applogger.Errorf` 暴露写入失败 — 修复 P1-2（successRate ≈ 0% 永久失真）与 P2-b（请求 ctx 取消导致日志写入丢失）。

## Tasks Executed

### Task 1: `internal/middleware/apikey.go` — 记录点后移到 c.Next() 之后 + 填三字段 + 去冗余 goroutine

**Status:** Completed
**Commit:** `e0f4611`

**Change:** Replaced the pre-`c.Next()` `go func(){ LogUsage }` block (old line 61-76) with the standard gin post-`c.Next()` capture pattern. Specifically:

1. Kept `setUserContextForAPIKey(c, apiKey, apiKey.Scopes)` (Phase 57 fix untouched).
2. Added `start := time.Now()` **before** `c.Next()` for latency measurement.
3. Called `c.Next()` (downstream handlers / RequireScope / RateLimitByScope execute).
4. After `c.Next()` returns: `statusCode := c.Writer.Status()` + `duration := time.Since(start).Milliseconds()` capture the real response result.
5. Constructed `LogUsageRequest` with three newly-filled fields: `StatusCode: statusCode` / `Duration: int(duration)` / `Success: statusCode >= 200 && statusCode < 300` (D-01).
6. Called `usageLogger.LogUsage(c.Request.Context(), req)` **synchronously** (D-02a: removed the outer `go func(){}` wrapper since `LogUsage` internally already does `go logUsageAsync`).

**Pre-auth branches preserved:** `c.Abort(); return` at line 36 (format error) / line 44 (ValidateAPIKey failure) / line 53 (IP whitelist rejection) all intact — they exit before `c.Next()` and correctly skip logging (no resolved apiKey to record).

**Verification:** `go build ./...` exit 0. Grep confirms `c.Writer.Status()` (post-c.Next()), `time.Since(start)`, `statusCode >= 200 && statusCode < 300`, `StatusCode:`/`Duration:`/`Success:` fields, and `usageLogger.LogUsage(c.Request.Context(), ...)` signature. No `go func(` matches remain in `MultiAuth`.

### Task 2: `internal/services/usage_logger.go` — detached context + applogger 失败可见

**Status:** Completed
**Commit:** `2dcb041` (after a commit fixup — see Deviations)

**Changes:**

**(a) Import block:** Added `applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"` (same alias + path as `config_backup_service.go:18`).

**(b) `logUsageAsync` function body:**
1. Added `detachedCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)` + `defer cancel()` at function entry (D-02, precedent: `pkg/cache/redis.go:601-605`).
2. Added `_ = ctx` explicit marker (callers' ctx is deliberately NOT used for DB-write cancellation control).
3. Changed DB write from `s.db.WithContext(ctx)` to `s.db.WithContext(detachedCtx).Create(&usageLog).Error`.
4. Replaced silent `_ = err` with `applogger.Errorf("[USAGE_LOG] 写入失败 key=%s path=%s: %v", req.APIKeyID, req.Path, err)` (D-04, precedent: `config_backup_service.go:247`, severity upgraded to Errorf since DB-write failure = error-level).

**Interface contract unchanged:**
- `UsageLogger` interface (line 12-17) ✓
- `LogUsage(ctx context.Context, req *LogUsageRequest) error` signature ✓
- `LogUsageRequest` struct (line 20-31) — already had `StatusCode/Duration/Success` fields ✓
- `NewUsageLogger(db *gorm.DB)` constructor ✓
- `LogUsage` internal `go s.logUsageAsync(ctx, req)` call shape ✓

**Verification:** `go build ./...` exit 0. Grep confirms `applogger` import, `context.WithTimeout(context.Background(), 10*time.Second)`, `s.db.WithContext(detachedCtx).Create(...)` (using detachedCtx, NOT ctx), `applogger.Errorf("[USAGE_LOG] 写入失败 key=%s path=%s: %v", ...)`. No `_ = err` silent-swallow remains. `_ = ctx` is the deliberate detached-ctx contract marker, not an error swallow.

## Decisions Made

All four locked decisions (D-01, D-02, D-02a, D-04) are grounded verbatim in the two source files as required by `<must_haves>.truths`. No new decisions made — this plan was a pure logic fix with pre-locked decisions.

| Decision | Literal grounding |
|----------|-------------------|
| D-01 Success 口径 | `apikey.go` line 84: `Success: statusCode >= 200 && statusCode < 300` |
| D-02 detached context | `usage_logger.go` line 60: `context.WithTimeout(context.Background(), 10*time.Second)` |
| D-02a 去冗余 goroutine | `apikey.go` — no `go func(` in `MultiAuth`; call is synchronous |
| D-04 applogger 替换静默吞错 | `usage_logger.go` line 83: `applogger.Errorf("[USAGE_LOG] 写入失败 key=%s path=%s: %v", req.APIKeyID, req.Path, err)` |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Task 2 commit `2a37830` accidentally included the unrelated frontend file `xingran-react-frontend/src/api/apikey.ts`**

- **Found during:** Immediately after Task 2 commit (`git show --stat HEAD` revealed `2 files changed` instead of 1).
- **Root cause investigation:** No active git hooks (`.git/hooks/` contains only `.` and `..`; no `core.hooksPath` set; no husky). The exact mechanism that staged `apikey.ts` between my `git add internal/services/usage_logger.go` and `git commit` is unknown — likely an agent-harness auto-staging behavior. The mechanism is irrelevant to the fix.
- **Impact:** The orchestrator explicitly prohibited touching `apikey.ts` (and the session-start snapshot already listed it as an unrelated dirty file). Including it in my Task 2 commit violated that constraint.
- **Fix applied:**
  1. `git reset --soft HEAD~1` (returned HEAD to Task 1 commit `e0f4611`, kept index + working tree intact).
  2. `git restore --staged internal/services/usage_logger.go xingran-react-frontend/src/api/apikey.ts` (targeted unstage — index only, working tree untouched).
  3. `git commit internal/services/usage_logger.go -m "..."` (used **explicit pathspec commit** to bypass the index entirely, guaranteeing only that file is committed regardless of staging state).
  4. Verified `git show --stat HEAD` shows `1 file changed` (only `usage_logger.go`).
  5. Verified `git status --short` shows `.planning/STATE.md` + 3 unrelated frontend files (`src/api/apikey.ts` + 2 that appeared mid-session: `src/types/apikey.ts` + `src/pages/system/apikeys/index.tsx`) all back to unstaged-modified state, matching orchestrator-owned expectations.
- **Why pathspec commit was the right defense:** It bypasses the index entirely — even if something re-stages `apikey.ts` between staging and committing, pathspec commit cannot include it.
- **Files affected:** Commit `2a37830` was replaced by commit `2dcb041` (identical message, scoped to a single file via pathspec commit).
- **Commits:** Task 2 final = `2dcb041`.

### Out-of-scope discoveries (logged, not fixed)

- **`xingran-react-frontend/src/types/apikey.ts` and `xingran-react-frontend/src/pages/system/apikeys/index.tsx` appeared as unstaged-modified files during this session.** Neither was in the conversation-start `gitStatus` snapshot (which only listed `apikey.ts` of the API folder). Both are frontend files outside this phase's scope (phase 59 is backend-only: `internal/middleware/` + `internal/services/`). They were left untouched — orchestrator owns frontend tracking. This is consistent with the orchestrator's note about unrelated dirty files in the working tree.

## Self-Check

### Files created/modified exist:
- `internal/middleware/apikey.go` — exists, contains `c.Writer.Status()` (line 67), `statusCode >= 200 && statusCode < 300` (line 84), `time.Since(start)` (line 68). FOUND.
- `internal/services/usage_logger.go` — exists, contains `context.WithTimeout(context.Background(), 10*time.Second)` (line 60), `applogger.Errorf("[USAGE_LOG] ...` (line 83), `s.db.WithContext(detachedCtx)` (line 79). FOUND.
- `.planning/phases/59-observability-usage-log-fix/59-01-SUMMARY.md` — this file. FOUND.

### Commits exist:
- `e0f4611` (Task 1): `git log --oneline --all | grep e0f4611` → FOUND.
- `2dcb041` (Task 2 final): `git log --oneline --all | grep 2dcb041` → FOUND.
- `2a37830` (Task 2 buggy intermediate): replaced by `2dcb041`, exists in reflog only.

## Self-Check: PASSED

## Output Deliverables

- **Source fix #1:** `internal/middleware/apikey.go` — recording point moved to post-`c.Next()`, three fields filled, redundant goroutine removed. P1-2 fixed.
- **Source fix #2:** `internal/services/usage_logger.go` — detached context for DB write, applogger exposes write failure. P2-b + D-04 fixed.
- **Build verification:** `go build ./...` exit 0 after both fixes.
- **Interface contract:** Zero breakage (UsageLogger interface, LogUsage signature, LogUsageRequest struct, NewUsageLogger, internal `go logUsageAsync()` all unchanged).
- **Phase 59 Plan 02 hand-off:** Source contracts now correct for the SC#1/SC#2/SC#4 tests that Plan 02 (Wave 2) will add to `internal/middleware/apikey_integration_test.go` + `internal/services/usage_logger_test.go` per `59-VALIDATION.md` Wave 0 requirements.

## Threat Surface

| Flag | File | Description |
|------|------|-------------|
| None | — | Plan did not introduce new security surface. `applogger.Errorf` log content is limited to `apiKeyID` (UUID, not key plaintext) + Path (gin-normalized URL) + `err` — no sensitive fields. `LogUsageRequest` struct has no `key`/`secret`/`password`/`token` fields. |

The plan's `<threat_model>` covered all applicable STRIDE categories (T-59-01/02/03/04 + T-59-SC); all dispositions (mitigate / accept) are realized by the literal implementation.