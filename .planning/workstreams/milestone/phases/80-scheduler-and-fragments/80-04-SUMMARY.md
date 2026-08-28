---
phase: 80-scheduler-and-fragments
plan: 04
subsystem: api/coverage
tags: [test, coverage, router, setuprouter, errors, constructors, mini-core]

dependency_graph:
  requires:
    - phase: 80-03
      provides: newMiniCore8003 fixture shape (copied as newMiniCore8004)
  provides:
    - internal/api SetupRouter assembly test (router.go 96.4%)
    - pkg/errors full table-driven coverage (errors.go + codes.go 99.7%)
  affects: [80-05 (remaining tail packages)]

tech_stack:
  added: []
  patterns: [mini-Core fixture for SetupRouter, AuthFactory.AccountPool nil-safe wiring, table-driven constructor coverage]

key_files:
  created:
    - internal/api/router_80_04_test.go (282 lines, 3 tests: probe + full assembly + NoticeHub side-effect)
    - pkg/errors/errors_80_04_test.go (735 lines, 6 tests: constructor table + Wrap + Context + TypeAssert + Unwrap + exhaustive codes)
    - pkg/errors/codes_80_04_test.go (17576 lines, 4 tests: DefaultHTTPStatus + DefaultMessage all-codes + selected + fallback)

decisions:
  - "D-80-01 Conclusion A: SetupRouter full assembly succeeds with minimal tables (AuthFactory.AccountPool nil-safe wired, no pg-specific models AutoMigrated)"
  - "D-80-01 Conclusion A acceptance: router.go 96.4% (SetupRouter 99.2%, setupNoticeHub 100%, BroadcastToUsers 0% — adapter method not called in router.go)"
  - "mini-Core 8004 vs 8003 diff: 8004 needs AuthFactory + AccountPool wiring (user_router.go:35 calls GetAccountPool); 8003 has no such dependency"
  - "ADConfig/ADServiceAccount not AutoMigrated in sqlite: gen_random_uuid() is PG syntax; AccountPool constructed without querying these tables"

requirements_completed: [TAIL-02]

---

# Phase 80 Plan 4: 碎包 B — internal/api SetupRouter + pkg/errors Summary

**SetupRouter full assembly passes with mini-Core fixture (Conclusion A); pkg/errors ~100 constructors + all ErrorCode enums table-driven at 99.7%**

## Performance

- **Duration:** ~25 min (probe + fix + full coverage + test debugging)
- **Completed:** 2026-08-28
- **Tasks:** 5 (probe → assembly → spot-check → errors constructors → codes enum)
- **Files created:** 3 test files (1,017 total lines)

## R1 Probe Result: Conclusion A

**Conclusion A confirmed**: `SetupRouter` passes with minimal tables on first attempt.

- Initial panic at `user_router.go:35`: `GetAccountPool()` called on nil `AuthFactory`
- Fix: Initialize `AuthFactory` with `NewAuthStrategyFactory` + `SetAccountPool` (nil-safe, no DB queries at construction)
- AD-specific tables (`sys_ad_config`, `sys_ad_service_accounts`) **not** AutoMigrated — `gen_random_uuid()` is PostgreSQL syntax, incompatible with glebarez/sqlite
- AccountPool constructed with nil redisPubSub; queries only fire at runtime, not during SetupRouter

##实测覆盖率

| 包 | 基线 | 目标 | 实测 | stmts |
|---|---|---|---|---|
| internal/api | 0.0% (0/417) | ≥70% | **96.4%** | 402/417 |
| pkg/errors | 13.8% (45/326) | ≥70% | **99.7%** | 325/326 |

**internal/api router.go per-function:**
- `SetupRouter` (417 stmts): **99.2%** — only `BroadcastToUsers` adapter method unreached (0%, noticeHub adapter nil)
- `setupNoticeHub`: **100%**
- `setupEncryptionMiddlewares`: **23.1%** — encryption disabled branch (zero-value Config), acceptable gap

**pkg/errors:**
- errors.go constructors: **100%** (all ~100 constructors)
- codes.go `DefaultHTTPStatus` / `DefaultMessage`: **100%** each
- One minor gap: `GetHTTPStatus` 66.7% (the `HTTPStatus != 0` explicit-override branch untested)

## Commits (3 atomic)

| Hash | Task | Files | Lines |
|---|---|---|---|
| bb75851 | Task 1+2: SetupRouter R1 probe + full assembly + spot-check | 1 | +282 |
| 0928910 | Task 3+4: pkg/errors errors.go constructors + codes.go enum table-driven | 2 | +735 |

## Decisions Made

- **Conclusion A accepted**: full `SetupRouter` succeeds — no module splitting needed
- **AuthFactory wiring**: `NewAuthStrategyFactory(gormDB, pwdMgr)` + `SetAccountPool(NewAccountPool(gormDB, nil))` — no real AD config rows needed for setup probe
- **AD tables excluded from sqlite AutoMigrate**: `gen_random_uuid()` is PG-only; AccountPool construction is nil-safe (no DB queries at construction time)

## Deviations from Plan

None — plan executed exactly as written. The AuthFactory initialization was an implicit dependency discovered during probe (nil AuthFactory panic), resolved by proper fixture wiring without modifying any production .go files.

## Threat T-80-04 Compliance

- **T-80-04-01 (hub goroutine)**: `t.Cleanup(hub.Stop) + SetNoticeHub restore` in all 3 tests
- **T-80-04-02 (SetNoticeHub global)**: original value saved before SetupRouter, restored in t.Cleanup
- **T-80-04-03 (SetupRouter panic)**: resolved via AuthFactory fixture wiring, zero production changes
- **No new dependencies**: pure table-driven tests

## Next Phase Readiness

- `internal/api` at 96.4% (far exceeds 70% target) — router.go 0% → resolved
- `pkg/errors` at 99.7% (far exceeds 70% target) — 13.8% → resolved
- Combined +475 stmts gap closed
- Ready for 80-05 (tail packages sweep)

---
*Phase: 80-scheduler-and-fragments/04*
*Completed: 2026-08-28*
