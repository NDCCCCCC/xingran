# internal/core Review Fixes — Batch 6 (P3 Refactors) Summary

> **Scope:** Batches 1-5 already shipped as `1451fc3` on `refactor/core-review-fixes`.
> This summary covers **batch 6 only**: Q3 (database.go split), Q4 (Init() god-function
> split), C1 (cancellable graceful shutdown). Three atomic commits on the same branch.

## Scope Confinement

- All three refactors are confined to `internal/core/core.go` and the
  `internal/core/db/` package. No other packages touched.
- Unrelated untracked files (`internal/middleware/apikey_integration_test.go`,
  the PLAN.md itself) were left untracked — explicit staging used for every commit.
- `internal/core/db/migrations/archive/` not touched (per plan constraint).
- Batches 1-5 work (`1451fc3`) not modified.

## Pre-existing Test Failures (NOT introduced by this batch)

Documented in `CORE-REVIEW-FIXES-PLAN.md` appendix; reproduced identically on the
baseline before any batch-6 commit:

1. `TestCoreSplit_NewConstructorPopulatesInfraAndServices` (internal/core) —
   `minimalTestConfig()` doesn't set `Security.SM4Key`; `New()`→`initSM4Cipher()`
   now requires SM4_KEY non-empty (batch-1 hardening). Out of scope.
2. Four `internal/core/security` integration tests —
   `table sys_user has no column named ad_dn`. Integration test SQLite schema
   missing `ad_dn`/`ad_ou_dn`/`ad_synced_at` columns. Out of scope (schema drift).

The Q4 regression tests that MUST PASS — `TestCoreSplit_BackwardCompat` and
`TestCoreSplit_FieldPromotionMatchesCoreInfra` — are green after every commit.

---

## Q3 — Extract `init_data` from `database.go`

**Plan reference:** batch 6 §Q3 (move seed functions to new file).

### Files changed

| File | Change |
|------|--------|
| `internal/core/db/database.go` | -934 lines (60% rewrite, pure removal) |
| `internal/core/db/init_data.go` | **new file**, +943 lines |

### Commit

`7a113cf` — `refactor(core): extract init_data from database.go`

### What moved to `init_data.go`

Block A (byte-identical to original lines 485-766):
- `initData`
- `createDefaultDept`
- `createDefaultUser`
- `createDefaultRole`
- `createUserRoleRelations`

Block B (byte-identical to original lines 807-1455 / EOF):
- `createNetworkDeviceSystemParams`
- `createNetworkDeviceScheduledJobs`
- `createCaptchaBackgroundSystemParams`
- `createCaptchaBackgroundMenus` (the `/* ... */` commented-out dead code — moved verbatim, not deleted)
- `NULL_STRING_PTR` (only used by moved blocks)
- `createOperationsManagementMenus`
- `createRequestEncryptionToggleConfig`
- `createADAuthConfig`

### What stayed in `database.go`

- Package/imports, `Database` struct, `NewDatabase`, `createSQLiteConnection`,
  `createFilteredLogger`, `createPostgresConnection`, `configureConnectionPool`,
  `Close`, `GetDB`
- `cleanupOldConstraints`, `dropDependentMaterializedViews`, `AutoMigrate`,
  `auditConstraintNaming`, `InitData` (caller of moved `initData(d.DB)`)
- `dbIdentRe` var + `createDatabaseIfNotExists`

### Import changes

- `database.go` dropped `internal/core/security` (no longer referenced after move).
- `init_data.go` declares `package db` + imports
  `fmt`, `internal/core/security`, `internal/models`,
  `applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"`, `gorm.io/gorm`.
- All other imports stay in `database.go` (still used by connection/migration code).

### Logic-equivalence verification (line-by-line)

- `git diff` confirms **only deletions** in `database.go`; the rewrite preserved the
  kept sections byte-for-byte (AutoMigrate model list, all explanatory comments
  including the long `dropDependentMaterializedViews` historical note).
- Per-block byte comparison (extracted via awk):
  - `diff old_blockA.txt new_blockA.txt` → **IDENTICAL**
  - `diff old_blockB.txt new_blockB.txt` → **IDENTICAL**
- `Database.InitData()` still calls `initData(d.DB)`; the cross-file call resolves
  because both files share `package db`.
- Git itself classified the operation as
  `copy internal/core/db/{database.go => init_data.go} (60%)`.

### Gate results

- `go build ./...` — pass (zero errors, zero warnings)
- `go vet ./internal/core/db/...` — pass
- `go test ./internal/core/` — `TestCoreSplit_BackwardCompat` PASS,
  `TestCoreSplit_FieldPromotionMatchesCoreInfra` PASS,
  `TestCoreSplit_NewConstructorPopulatesInfraAndServices` FAIL (pre-existing SM4_KEY)
- No `db` package tests exist (`[no test files]`).

### Deviations from plan

None. Boundary was exactly as the appendix prescribed.

---

## Q4 — Split `Init()` god function

**Plan reference:** batch 6 §Q4 (extract ~20 steps into private methods).

### Files changed

| File | Change |
|------|--------|
| `internal/core/core.go` | +116 / -15 (Init shrinks to orchestration; 8 methods added) |

### Commit

`56ab9ca` — `refactor(core): split Init() god function`

### Extracted methods

| Method | Steps | Returns error | Fail-fast inside |
|--------|-------|---------------|------------------|
| `initDBAndData()` | 1-4 (DB / AutoMigrate / InitData / permissions) | yes | step 1 (F-15), step 2 (P0 #16) |
| `initCacheAndWarmUp()` | 5-6 (cache system + cache services + async warmup) | yes | step 5 (P0 #16 cache) |
| `initMetrics()` | 7 | no | — |
| `initDeviceServices()` | 8-9.5 (pool / executor / discovery / collection / partition) | no | — (all warn-only) |
| `initSchedulerAndTasks()` | 10-12 (scheduler + Register*Tasks + device monitor) | yes | step 10 (F-16) |
| `initCaptchaServices()` | 13-14.1 | no | — |
| `initLogsAndAuth()` | 15-16.5 | no | — |
| `initRPAAndAPIAndReaper()` | 17-19 | no | — (warn-only) |

### What stayed inline in `Init()`

- Step 0 (AD SM4 cipher one-liner setup) — kept inline as setup before any sub-call.
- Step 20 (Phase 42 R1 RefreshView goroutine) — kept inline as final post-init goroutine.
- `Init()` body is now pure orchestration: 8 method calls + step 0 + step 20 + `return nil`.

### Duplicate "9." step numbering check

Verified via grep `^\s*// \d+(\.\d*)?\.?\s` — batch 4 already renumbered the duplicate
"9." to `9`, `9.1`, `9.5`. No duplicate step numbering remains; nothing to fix in Q4.

### Logic-equivalence verification (line-by-line)

Pure extraction. Execution order preserved exactly:
0 → initDBAndData → initCacheAndWarmUp → initMetrics → initDeviceServices →
initSchedulerAndTasks → initCaptchaServices → initLogsAndAuth →
initRPAAndAPIAndReaper → step 20 goroutine → return nil.

Every fail-fast / warn-continue decision preserved verbatim, including the
comments explaining the policy (F-15, P0 #16, F-16, SkipSetup semantics, etc.).
Specifically verified from the diff:
- Step 1: `applogger.Errorf("数据库初始化失败: %v", err)` then
  `return fmt.Errorf("数据库初始化失败: %w", err)` — preserved.
- Step 2: `return fmt.Errorf("数据库迁移失败: %w", err)` — preserved.
- Step 5: `return fmt.Errorf("初始化缓存系统失败: %w", err)` — preserved.
- Step 10: `return fmt.Errorf("启动调度器失败: %w", err)` — preserved.
- Step 3, 4, 9.1, 9.5, 13, 17, 18 — all warn-only paths preserved.

Only structural additions (no behavior change):
- Method declarations (`func (c *Core) initXxx() ...`).
- Doc comments per method describing which steps it owns and the fail-fast policy.
- `return nil` at end of each error-returning method.
- One `var err error` declaration added inside `initCacheAndWarmUp` (step 5) —
  needed because the original reused the outer-scope `err` declared at step 1,
  which is now in a separate method.

### Gate results

- `go build ./...` — pass
- `go vet ./internal/core/...` — pass
- `go test ./internal/core/ -run 'TestCoreSplit_BackwardCompat|TestCoreSplit_FieldPromotionMatchesCoreInfra'` — **both PASS**

### Deviations from plan

Minor signature deviation (mandated by the plan's own "PURE EXTRACTION" constraint):
the plan's sketch listed `initDeviceServices() error` and `initRPAAndAPIAndReaper() error`,
but the plan's preceding sentence also says "if the original step did `return err`, the
extracted method returns err" — the contrapositive being that warn-only steps must not
return err. Steps 8-9.5 and 17-19 are all warn-only or have no error path, so these two
methods correctly have no `error` return. Documented in the methods' doc comments.

---

## C1 — Cancellable graceful shutdown

**Plan reference:** batch 6 §C1 (cautious two-step approach).

### Files changed

| File | Change |
|------|--------|
| `internal/core/core.go` | +46 / -16 (Close() body) |

### Commit

`f2364a0` — `refactor(core): cancellable graceful shutdown`

### What changed in `Close()`

1. **Added** `shutdownCtx, cancel := context.WithTimeout(context.Background(), coreShutdownTimeout)` + `defer cancel()` at the top. This is the foundation for ctx-aware sub-services.
2. **Rewired** the deadline-watcher goroutine to `select` on `shutdownCtx.Done()` instead of a separate `time.NewTimer`. Added `if shutdownCtx.Err() == context.DeadlineExceeded` guard so that `defer cancel()` on normal return does not produce a spurious force-exit warning (the original prevented this via `closeDone` winning the select; the new code preserves that semantic and additionally distinguishes real deadline from manual cancel).
3. **Documented per sub-service** that none of the following currently accept a `context.Context` parameter: `reaperCancel` (CancelFunc), `NoticeHub.Stop`, `Scheduler.Stop`, `scheduler.StopADSyncScheduler`, `DeviceInfoCollectionService.Stop`, `DeviceMonitorService.Close`, `MetricsCacheService.Stop`, `RPAScalingService.Stop`, `Cache.Close`, `Database.Close`. Per the plan's explicit instruction ("Do NOT change signatures that don't currently take a context"), none of these signatures were modified.
4. **`time.Sleep(100ms)` kept** with a specific comment: `operlog.RecordAsync` (`internal/services/oper_log_service.go:67`) uses fire-and-forget goroutines with no exposed WaitGroup/Flush primitive, so a deterministic wait is not viable. The plan explicitly allows "keep current behavior and add a brief comment noting why." `MultiLevelCache.Close()` already deterministically flushes its own L2 async writes via `L2WriteWriter.Stop()` → `wg.Wait()` (`pkg/cache/l2_writer.go:181`), so the sleep is no longer about cache writes — comment now reflects that.

### Logic-equivalence verification (line-by-line)

Behavior is identical to the original `Close()`:
- **Sub-service call order:** unchanged (reaper → NoticeHub → Scheduler → ADSync → DeviceInfoCollection → DeviceMonitor → MetricsCache → RPAScaling → Cache → sleep → DB).
- **Sub-service invocations:** unchanged — same `if c.X != nil { c.X.Stop()/Close() }` pattern, same error-ignoring behavior (e.g., `DeviceMonitorService.Close() error` return value still discarded, as in the original).
- **Warning text on timeout:** identical — `[Core.Close] 已超过 %v 强制结束(子步骤阻塞,资源将由 OS 回收)`.
- **Sleep duration:** identical — `100 * time.Millisecond`.
- **Shutdown ordering rationale comment (the 2026-07-06 shutdown-hang-after-port-close fix):** preserved verbatim.

The only runtime nuance: the new `Err() == DeadlineExceeded` check makes the warning fire on true timeouts only. On normal `Close()` return, `defer close(closeDone)` (registered after `defer cancel()`) runs first via LIFO, so the watcher goroutine exits via `<-closeDone` before `cancel()` even runs — identical to the original behavior.

### Gate results

- `go build ./...` — pass
- `go vet ./internal/core/...` — pass
- `go test ./internal/core/ -run 'TestCoreSplit_BackwardCompat|TestCoreSplit_FieldPromotionMatchesCoreInfra'` — **both PASS**

### Deviations from plan

None. The plan's §C1 explicitly recommends this two-step approach: "(1) 先把 Close() 顶层建一个 shutdownCtx, cancel := context.WithTimeout(...) ... (2) time.Sleep 改 WaitGroup. 无法传 ctx 的子服务保留现状并在注释标明." This commit lands step 1 and the documentation half of step 2; the deterministic-WaitGroup half is correctly deferred because no exposed WaitGroup target exists for operlog.

---

## Overall Batch-6 Gate (after all 3 commits)

| Gate | Result |
|------|--------|
| `go build ./...` | pass (zero errors / zero warnings) |
| `go vet ./internal/core/...` | pass |
| `TestCoreSplit_BackwardCompat` | **PASS** |
| `TestCoreSplit_FieldPromotionMatchesCoreInfra` | **PASS** |
| `TestCoreSplit_NewConstructorPopulatesInfraAndServices` | FAIL (pre-existing, SM4_KEY missing in `minimalTestConfig`) |
| `TestIntegration_LocalAuthenticator_*` (3) + `TestIntegration_HybridAuthenticator_LocalSuccess` | FAIL (pre-existing, `sys_user.ad_dn` column missing in test schema) |

## File footprint (batch 6 only)

```
internal/core/core.go         | +162 / -31  (Q4 + C1 combined)
internal/core/db/database.go  | -934 / +0   (Q3 removals)
internal/core/db/init_data.go | +943 / -0   (Q3 new file)
```

3 files, 3 commits, no other modules touched.

## Commits (batch 6)

| Hash | Subject |
|------|---------|
| `7a113cf` | `refactor(core): extract init_data from database.go` |
| `56ab9ca` | `refactor(core): split Init() god function` |
| `f2364a0` | `refactor(core): cancellable graceful shutdown` |

## Self-Check: PASSED

- `[ -f internal/core/db/init_data.go ]` → FOUND
- `[ -f internal/core/db/database.go ]` → FOUND (521 lines, down from 1456)
- `[ -f internal/core/core.go ]` → FOUND
- `git log --oneline | grep 7a113cf` → FOUND
- `git log --oneline | grep 56ab9ca` → FOUND
- `git log --oneline | grep f2364a0` → FOUND
