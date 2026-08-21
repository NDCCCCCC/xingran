---
phase: 74
plan: 04
subsystem: system-agent-handler-coverage
tags: [coverage, handler-tests, mock-services, router-smoke, p2-finalize]
dependency_graph:
  requires: [phase-72-p0-core-supplement, phase-73-handler-patterns]
  provides: [system-handler-tests-suite, agent-handler-tests, system-router-smoke]
  affects: [internal/api/v1/system, internal/api/v1/agent]
tech_stack:
  added: []
  patterns:
    - interface mock with per-method *Func fields (NoticeCacheService, AccountPool, channel/scheduler)
    - fake PasswordCipher for SM4-dependent Create/Update paths
    - runtime router smoke via sqlite-backed core (Setup*Router invocation, recover-guarded)
    - raw-DLL sqlite tables to avoid PG-only default tags (gen_random_uuid)
key-files:
  created:
    - internal/api/v1/agent/agent_handler_test.go
    - internal/api/v1/system/ad_account_pool_handler_test.go
    - internal/api/v1/system/cache_adapter_test.go
    - internal/api/v1/system/column_config_handler_test.go
    - internal/api/v1/system/dashboard_handler_test.go
    - internal/api/v1/system/helpers_and_fixmenu_test.go
    - internal/api/v1/system/notice_handler_full_test.go
    - internal/api/v1/system/notice_user_handler_test.go
    - internal/api/v1/system/setup_routers_full_test.go
  modified: []
decisions:
  - id: D-15-P2-FLOOR
    summary: system 35.4%→70.4%, agent 0%→78.9% (both ≥70%)
  - id: D-12-STRICT
    summary: zero business code changes; 9 *_test.go files only
  - id: ROUTER-RUNTIME-SMOKE
    summary: Setup*Router invoked at runtime with sqlite core instead of source-grep
    rationale: runtime invocation gives real statement coverage for all 19 *_router.go files at 0% cost beyond one sqlite core; recover-guard skips any router with an un-mockable core dep
  - id: SHARED-MOCK-FUNC-HONORING
    summary: mockNoticeCacheService admin methods (Update/Delete/BatchDelete/List/Publish/Withdraw/GetStatistics/GetStatusStatistics) fixed to honor *Func fields
    rationale: the prior-interrupt draft returned unconditional defaults, silently ignoring injected error paths
metrics:
  completed_date: 2026-08-21
  baseline_coverage: 35.4
  final_coverage: 70.4
  coverage_delta: 35.0
  agent_baseline: 0.0
  agent_final: 78.9
  test_files_added: 9
---

# Phase 74 Plan 04: System + Agent Handler Tests Summary

**One-liner:** Pushed `internal/api/v1/system` (3039 stmts, largest api/v1 package) from 35.4% to 70.4% and `internal/api/v1/agent` (38 stmts) from 0% to 78.9% — both clear the D-15 P2 ≥70% floor.

## Coverage Progression

| Milestone | system pkg | Trigger |
|-----------|-----------|---------|
| Baseline | 35.4% | Phase 73 handoff |
| After interrupt-agent draft (5 files, 1 compile error) | 47.4% | dashboard/column_config/notice_user/cache_adapter/agent files repaired |
| + notice admin handler tests (16 funcs, 3 interface mocks) | 54.3% | notice_handler_full_test.go |
| + ad_account pool handler tests (10 endpoints, AccountPool mock) | 58.3% | ad_account_pool_handler_test.go |
| + runtime router smoke (19 Setup*Router) | 69.7% | setup_routers_full_test.go |
| + helpers/fix_menu tests | **70.4%** | helpers_and_fixmenu_test.go |

**agent package: 0% → 78.9%** (agent_handler_test.go, 454 lines).

## Files Created (commit 1f30b6b, 3662 insertions)

| File | Lines | Covers |
|------|-------|--------|
| `agent/agent_handler_test.go` | 454 | all agent endpoints (38 stmts pkg) |
| `system/dashboard_handler_test.go` | 748 | DashboardHandler 27 funcs |
| `system/notice_user_handler_test.go` | 636 | NoticeUserHandler 12 funcs + shared mockNoticeCacheService |
| `system/notice_handler_full_test.go` | 625 | NoticeHandler admin CRUD 16 funcs (incl. recurring/scheduled cron paths) |
| `system/ad_account_pool_handler_test.go` | 578 | ADAccountHandler 8 endpoints + computeStats |
| `system/cache_adapter_test.go` | 219 | cache adapter paths |
| `system/column_config_handler_test.go` | 214 | ColumnConfigHandler 3 handlers + WithCore |
| `system/setup_routers_full_test.go` | 104 | 19 Setup*Router runtime smoke (sqlite core, recover-guarded) |
| `system/helpers_and_fixmenu_test.go` | 84 | parseInt + FixMenuPathsHandler (DB row assertions) |

## Execution Notes

- This plan was executed across an interrupted background executor (5 draft files, one compile error `UserColumnConfig{ID: ...}` promoted-field literal) and inline completion (4 more files + repairs). Subagent dispatch was unavailable (provider modelCode routing errors) — workflow fallback to sequential inline execution applied.
- Commit subject shortened to pass commitlint subject-max-length (first two attempts rejected).

## Documented Quirks (D-12 — no business code changed)

1. **response.Error int-first-arg → HTTP 400 always** (re-confirmed; also affects `apperrors.NotFound` in fix_menu_handler → 400, not 404).
2. **ADDomainHandler.service is a concrete `*addomainServices.ADDomainService`** — not interface-mockable; ad_domain_handler.go stays at 3.6% this plan. Needs a Phase-75-style service-interface refactor or sqlite+LDAP-stub harness; out of D-12 scope.
3. **sqlite cannot AutoMigrate models with `default:gen_random_uuid()` PG tags** — raw DDL used instead (matches memory `xingran-db-migration-and-pk-gotcha`).
4. **NoticeCreateRequest requires 3 fields** (noticeTitle/noticeType/noticeContent) — bind fails silently return 400 without them.
5. **ou_group/ou_mapping/ad_domain_user_sync handlers** (concrete-service deps) remain ≤33% — same constraint as quirk 2.

## Constraints Honored

- D-12 STRICT: only `*_test.go` in commit (git diff 04855c4..1f30b6b = 9 test files, 0 business)
- D-03: operlog stubs via nil-safe `operlog.Record` (CoreServices{OperLogService: nil} pattern)
- D-15: both packages ≥70%
- No STATE.md/ROADMAP.md updates (orchestrator-owned); no push
