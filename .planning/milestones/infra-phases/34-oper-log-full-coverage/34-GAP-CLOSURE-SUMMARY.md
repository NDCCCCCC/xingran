---
phase: 34-oper-log-full-coverage
plan: 34-gap
subsystem: operlog (audit logging)
tags: [operlog, audit, compliance, gap-closure, verification]
requires:
  - 34-01 (operlog shared package)
  - 34-02..34-08 (Wave 1-7 instrumentation)
  - 34-09 (e2e harness)
  - 34-10 (convention docs)
provides:
  - "100% write-endpoint operlog coverage (gap-closure of 25 missed endpoints)"
  - "Handler-file-vs-operlog differential e2e check (prevents future regressions)"
affects:
  - internal/api/v1/monitor/cache_enhanced_handler.go
  - internal/api/v1/network/network_export_handler.go
  - internal/api/v1/network/batch_export_helper.go
  - internal/api/v1/system/default_theme_handler.go
  - internal/api/v1/system/settings_router.go
  - internal/api/v1/operations/room_photo_handler.go
  - internal/api/v1/captcha_background_handler.go
  - internal/api/v1/system/user_unlock_handler.go
  - scripts/operlog_e2e_verify.sh
tech-stack:
  added: []
  patterns:
    - "WithCore() chainable core injection (default_theme_handler follows user_handler pattern)"
    - "closure-based handler operlog (room_photo/captcha_background/user_unlock pass core directly)"
key-files:
  created:
    - .planning/phases/34-oper-log-full-coverage/34-GAP-CLOSURE-SUMMARY.md
  modified:
    - internal/api/v1/monitor/cache_enhanced_handler.go
    - internal/api/v1/network/network_export_handler.go
    - internal/api/v1/network/batch_export_helper.go
    - internal/api/v1/system/default_theme_handler.go
    - internal/api/v1/system/settings_router.go
    - internal/api/v1/operations/room_photo_handler.go
    - internal/api/v1/captcha_background_handler.go
    - internal/api/v1/system/user_unlock_handler.go
    - scripts/operlog_e2e_verify.sh
    - .planning/notes/260615-oper-log-coverage-audit.md
decisions:
  - "WarmUpCache (NotImplemented stub) audits the attempt with WithStatus(1)+WithErrorMsg — compliance-sensitive cache op, even failures should be auditable"
  - "user_unlock uses OperTypeOther(0) not OperTypeStatus(10) — unlock clears login-lock cache, does not modify sys_user.status column"
  - "mac_history_handler / mac_history_heatmap_handler added to READONLY_ALLOWLIST — all methods are POST /history/* queries (query-with-body pattern) + 1 GET export, no state mutation"
  - "Differential check counts BOTH operlog.Record AND recordOperLog shim — AD-domain handlers (Wave 1 backward-compat) use the shim which delegates to operlog.Record"
metrics:
  duration: ~25min
  completed: 2026-06-16
  tasks: 8 (6 instrumentation + 1 harness + 1 docs)
  calls-before: 282
  calls-after: 308 (internal/api/v1) / 313 (internal/ 全量)
---

# Phase 34 Plan 34-gap: Operlog Coverage Gap-Closure Summary

Instrumented 25 previously-missed routed write endpoints across 6 handler files and hardened the e2e harness with a handler-file-vs-operlog differential check, closing the verification gaps flagged in 34-VERIFICATION.md.

## What Was Built

### Gap 1 — 6 handler files instrumented (25 endpoints)

| # | Handler | Endpoints | OperType(s) | Commit |
|---|---------|-----------|-------------|--------|
| 1 | `monitor/cache_enhanced_handler.go` | 3 (InvalidateByModule / InvalidateByPattern / WarmUpCache) | Clean×2 / Clean+fail | 7338441 |
| 2 | `network/network_export_handler.go` + `batch_export_helper.go` | 9 (devices/credentials/templates/commands/executions/backups/discoveries/mac/ports exports + BatchExport) | Export | 977b129 |
| 3 | `system/default_theme_handler.go` (+ `settings_router.go` WithCore threading) | 2 (Set / Sync) | Update / Sync | f47eda4 |
| 4 | `operations/room_photo_handler.go` | 6 (upload / setPrimary / updateDescription / updateSort / delete / batchDelete) | Upload / Update×3 / Delete×2 | 1d8c288 |
| 5 | `captcha_background_handler.go` | 4 (upload / update / delete / toggle) | Upload / Update / Delete / Status | 2472a2e |
| 6 | `system/user_unlock_handler.go` | 1 (unlock — **compliance-sensitive**: who-unlocked-whom) | Other + username audit | f750569 |

All follow the established Phase 34 pattern: `operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "模块名", operlog.OperTypeXxx)` placed at the end of each write method's success path, before `response.Success(...)` (or before binary `c.Data(...)` for exports).

### Gap 2 — e2e harness tightened (commit a6aa193)

- Static call-count threshold raised `>=250` → `>=290` (the loose 250 threshold could not distinguish 267 vs 292, hiding the 25-endpoint gap)
- New handler-file-vs-operlog differential check: enumerates every `*_handler.go` file under `internal/api/v1/` with a `*Handler` receiver and FAILs if it contains zero operlog calls (direct `operlog.Record`/`RecordWithBody` OR legacy `recordOperLog` shim). Includes a `READONLY_ALLOWLIST` for genuine query-only handlers (mac_history).
- Added 3 sampled `assert_logged` checks for gap-closure compliance endpoints: user-unlock, cache/invalidate, network/devices/export.

## Compliance-Sensitive Instrumentation

`user_unlock_handler.go` was the highest-priority gap: account unlock is a compliance-sensitive operation that previously had **zero audit trail**. Now records `who-unlocked-whom`:
- operator: extracted automatically from JWT via `utils.GetUsernamePtr(c)`
- target: captured as `oper_param="username=<unlocked-user>"`

Decision: uses `OperTypeOther(0)` rather than `OperTypeStatus(10)` because unlock clears the `login:lock:{username}` cache key and does NOT modify the `sys_user.status` column (which is what OperTypeStatus semantics imply).

## Verification

| Check | Command | Result |
|-------|---------|--------|
| Build | `go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| operlog call count (internal/api/v1) | grep | 282 → **308** (+26 new) |
| operlog call count (internal/ 全量) | grep | **313** |
| operlog regression tests | `go test ./internal/utils/operlog/` | PASS |
| Touched package tests | `go test ./internal/api/v1/{monitor,network,operations,system}/` | PASS |
| e2e static portion | `SKIP_LIVE=1 DEV_MODE=1 bash scripts/operlog_e2e_verify.sh` | exit 0 ("static checks PASSED" + "all *Handler-receiver files contain >=1 operlog call") |

## Deviations from Plan

**1. [Rule 1 - Bug] `grep -c` exit-code handling in bash differential check**
- Found during: Gap 2 harness work
- Issue: `grep -c` exits 1 when match count is 0, so `$(grep -c ... || echo 0)` concatenated `0\n0` into the count variable under `set -o pipefail`, causing `[[: syntax error`
- Fix: capture stdout with `|| true`, then `tr -d '[:space:]'` and default empty to 0
- Commit: a6aa193

**2. [Rule 2 - Missing critical functionality] Differential check must count `recordOperLog` shim**
- Found during: Gap 2 differential check first run flagged AD-domain handlers (ad_domain_handler, dashboard_handler, etc.) as false positives
- Issue: these handlers use the Phase 34 Wave 1 backward-compat `recordOperLog` shim (in `internal/api/v1/system/helper.go`) which delegates to `operlog.Record` — they ARE covered but the direct-grep missed them
- Fix: differential check sums both `operlog.Record|RecordWithBody` AND `recordOperLog(` counts
- Commit: a6aa193

**3. Pre-existing test failure (out of scope, logged not fixed)**
- `internal/api/v1/auth_test.go::TestLoginWithInvalidEncryptedRequest` fails with 404 (expects 400) — pre-existing, unrelated to operlog changes (last modified in Phase 18 commit 44bb78a). Per SCOPE BOUNDARY rule, NOT fixed; documented here.

## Files Modified

- 6 handler files (+1 router) — operlog instrumentation (Gap 1)
- `scripts/operlog_e2e_verify.sh` — threshold + differential check + 3 new samples (Gap 2)
- `.planning/notes/260615-oper-log-coverage-audit.md` — appended §10.7 correcting the 100% claim

## Known Stubs

None. All instrumented methods wire to real services; no placeholder data introduced by this plan. (Note: `WarmUpCache` is a pre-existing NotImplemented stub in `cache_enhanced_handler.go` — left as-is, but now audits the attempt with failure status since cache warmup is compliance-relevant.)

## Threat Flags

None. No new network endpoints, auth paths, or trust-boundary schema changes introduced. The 25 instrumented endpoints are all pre-existing routes; this plan only adds audit logging to them.

## Self-Check: PASSED

All 6 handler files and the e2e script exist on disk and are committed. All 7 gap-closure commits (7338441, 977b129, f47eda4, 1d8c288, 2472a2e, f750569, a6aa193) verified present in `git log`.
