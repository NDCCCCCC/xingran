# Phase 79: 长尾清欠·internal/services root - Research

**Researched:** 2026-08-27
**Domain:** Go backend test coverage — `internal/services` root package (45 non-test files, 5202 stmts @ 11.3%)
**Confidence:** HIGH on coverage measurements + test-double classification (all measured/read this session), MEDIUM on executor-path reachability, LOW on SMTP/LDAP wire-fake effort estimates

**Measurement provenance:** `go test -count=1 -coverprofile ./internal/services/` run this session (377s wall) → 5202 stmts / 589 covered / 11.3%. `go tool cover -func` aggregated per file. Gap to SC-1: need 3642 covered (70%), i.e. **+3053 covered stmts** — matches ROADMAP's "~3052" exactly.

---

## 0. Phase Goal & Scope (from ROADMAP.md L218-247)

**Goal:** Push `internal/services` root (5202 stmts @ 11.3%) to ≥70%. Largest single-package work surface in the milestone.

**Depends on:** Phase 76 INFRA-01 (miniredis + httpmock) — **both already in go.mod** [VERIFIED: go.mod:8 `alicedb/miniredis/v2 v2.38.0`, go.mod:17 `jarcoal/httpmock v1.4.2`, both tagged `// test-only (v1.27 D-02)`]. No hard dependency on Phase 77/78.

**Requirement:** TAIL-01.

**Success Criteria (ROADMAP L228-231):**
1. root 包 ≥70%, legacy cache services 群(dept/role/dict/menu/user/post)逐文件 ≥70% ⚠️ **see DQ1 — this clause is factually stale**
2. token blacklist 及其余 root 文件按 profile 倒序补齐, 无单文件留在 <50%
3. 收尾以 `go tool cover -func` 实测回填 `.planning/coverage-baseline.md`
4. gate 全程绿 (不动 `.coverage-threshold`, Phase 81 收口)

**Gate mechanics verified:** `.coverage-threshold` = 55.5 (weighted-avg gate only). `internal/services` root is in NEITHER `P1_PACKAGES` (check-coverage.sh:164) NOR `P2_PACKAGES` (check-coverage.sh:228) — so Phase 79 has **no per-package floor line to retire**; its contribution is purely weighted-avg ratchet. Any partial progress still raises the average; SC-4 is low-risk.

---

## 1. SCOPE CORRECTION — "legacy cache services 群" is no longer in root (DQ1, read first)

ROADMAP SC-1's clause "legacy cache services 群(dept/role/dict/menu/user/post)逐文件 ≥70%" references files that **do not exist in `internal/services` root**:

```
ls internal/services/{dept_service,role_cache_service,dict_cache_service,menu_cache_service,
                     user_cache_service,post_cache_service}.go
→ No such file or directory (all 6)                      [VERIFIED this session]
```

All 6 were migrated to `internal/services/system/*_cache_impl.go` (`department_cache_impl.go` 150L, `role_cache_impl.go` 276L, `dict_cache_impl.go` 272L, `menu_cache_impl.go` 237L, `user_cache_impl.go` 337L, `post_cache_impl.go` 177L) — a **different package** whose 3483 stmts are NOT counted in the 5202 root figure. CLAUDE.md's "Dual Cache Architecture → Legacy root files" section is stale on this point.

**State of those 6 impl files:** they already have dedicated tests (`*_cache_impl_test.go` × 5 + `user_cache_impl_gapfill_test.go` + `system_menu_cache_gapfill_test.go`; 40 test files in system/). coverage-baseline.md's last recorded system figure is 53.5% (Phase 73 后 row).

**Root package's actual cache-named files** (the plausible SC-1 re-interpretation targets):
| File | Stmts | Covered | Current |
|---|---|---|---|
| cache_config_service.go | 124 | 84 | 67.7% |
| data_cache_service.go | 77 | 15 | 19.5% |
| template_cache.go | 18 | 0 | 0% |
| mac_history_cache_decorator.go | 12 | 0 | 0% |

**Planner options (see DQ1):** (a) re-anchor SC-1 to these 4 root cache files 逐文件 ≥70%; (b) pull the 6 system impl files into Phase 79 scope (+~700 stmts of work, cross-package); (c) declare SC-1 clause already satisfied by Phase 73 system work. This research proceeds on interpretation (a) + (c) — root-only scope, all 4 root cache files driven past 70%.

---

## 2. Per-File Coverage Gap Table (ranked by uncovered stmts, descending)

Measured this session via `go test -count=1 -coverprofile=cov79.out ./internal/services/` → `go tool cover -func` per-file aggregation:

| # | File | Stmts | Unc | Cov | Class | Primary blockers |
|---|------|-------|-----|-----|-------|------------------|
| 1 | mac_history_query_service.go | 389 | 358 | 8.0% | c+b | sqlite query paths (5×queryFromDB), ExportHistory (excelize→io.Writer), ImportOUIData (file I/O); existing `setupTestService` uses `cache:nil` so cache decorator paths also open |
| 2 | device_discovery_service.go | 293 | 289 | 1.4% | c+e | pure IP math (calculateIPCount/generateIPList/ipLessEqual/incrementIP ~80 stmts) + CRUD sqlite; `snmpProbe`/`discoverBySNMP` need UDP fake (78-04's fake is in-package test-only, NOT importable); `isAlive` is TCP-connect (net.Listen fake = trivial) |
| 3 | knowledge_service.go | 289 | 285 | 1.4% | c | pure CRUD sqlite (List/Get/Create/Update/Delete/tag links) |
| 4 | device_info_collection_service.go | 390 | 275 | 29.5% | c+a | parse fns already covered by 17 existing tests; open: Start/Enqueue/EnqueueAllOnlineDevices/worker/recoverPendingTasks (goroutines, has `Stop()` at :104), processTask/CollectDeviceInfo (holds `*device.DeviceExecutor` :38), loadConfigFromDB/getCommandsByVendor (sqlite) |
| 5 | config_backup_service.go | 244 | 244 | 0.0% | c+a | holds `*device.DeviceExecutor` :25; pure calculateHash/generateDiff/getDefaultThreshold/backup-dir fns; GetBackupList/GetBackupByID/DiffBackups/RestoreBackup/GetBackupStatistics are sqlite+file paths |
| 6 | mac_collection_service.go | 291 | 208 | 28.5% | c+a | parseMACAddressTable/parseMACLine/parseRuijiePortSecurityLine/mergeMACEntries/cleanTimestampFromInterface = pure (~100 stmts); GetMACAddressList/Stats/CleanOldRecords/BatchDelete sqlite; collectDeviceMAC needs executor |
| 7 | network_device_service.go | 202 | 202 | 0.0% | c | pure CRUD sqlite + association preload |
| 8 | email_sender_service.go | 189 | 189 | 0.0% | d | pure builders (buildEmailContent/buildNoticeHTMLBody/getNoticeTypeLabel/getPriorityClass/getPriorityLabel/plainAuth.Start/Next ~90 stmts); `sendPlainSMTP` testable with plain fake SMTP server; **`sendWithTLS`/`sendWithSTARTTLS` hardcode `InsecureSkipVerify: false` (email_sender_service.go:203-204, :271-272) → self-signed fake is rejected, only dial-error branch reachable without production change** |
| 9 | device_monitor_service.go | 189 | 189 | 0.0% | c+e | SetExecutor/SetPortCollectionService/SetMACCollectionService/SetConfigBackupService setters + nil-guards (:74-99), loadConfigFromDB/ReloadConfig/Close, convertSNMPVersion pure; CheckDeviceStatus/pingDeviceViaSNMP need UDP fake; Collect* delegate to injected services (nil-branch testable) |
| 10 | ad_ldap_client.go | 179 | 179 | 0.0% | e | pure: loadADTLSSkipVerify/formatUsername/extractRDN/parseIntOrDefault/encodePassword + NewLDAPClient assembly + Connect dial-error (~70 stmts); Search*/Update*/Move*/Enable/Disable/Unlock/ResetPassword need live LDAP wire — root `LDAPClient` is concrete (:41), addomain's `LDAPClientIface` does NOT apply here |
| 11 | duty_schedule_service.go | 174 | 174 | 0.0% | c | GenerateSchedule algorithm + getDutyType/isWeekend pure + 6 CRUD fns — all sqlite |
| 12 | template_service.go | 166 | 166 | 0.0% | c | CRUD + Preview/Render/ValidateVariables/Clone/Export/Import (no text/template import — string substitution, pure) |
| 13 | config_execution_service.go | 152 | 152 | 0.0% | a | executor-centric (errgroup fan-out to `executor` :19); validate/param-assembly portions partial |
| 14 | auth_credential_service.go | 142 | 142 | 0.0% | c | CRUD + GetStatistics + GetByIDWithPassword (decrypt via `addomain.PasswordCipher` interface — stub in test) |
| 15 | notification_config_service.go | 127 | 127 | 0.0% | c | CRUD + exported `EncryptPassword`/`DecryptPassword` AES-GCM pair (:246/:277) — pure round-trip |
| 16 | notification_sender_service.go | 120 | 120 | 0.0% | c | dispatch to EmailSender/APISender (nil-injected services → error branches) + sqlite config load |
| 17 | api_sender_service.go | 119 | 119 | 0.0% | d | **httpmock already in go.mod** — sendRequest round-trip; buildRequestBody/buildFromTemplate/buildDefaultBody/setRequestHeaders/setAuthentication pure |
| 18 | notice_service.go | 120 | 116 | 3.3% | c | CRUD + Publish/Withdraw + CreateNoticeWithTargets |
| 19 | command_dispatch_service.go | 116 | 112 | 3.4% | a | executor dispatch + parse/validate portions |
| 20 | duty_pool_service.go | 102 | 97 | 4.9% | c | CRUD + stats |
| 21 | mac_history_partition.go | 95 | 95 | 0.0% | c | PG-only `PARTITION OF` DDL generation (:91-94) — string builders pure-testable; sqlite branch "跳过分区管理" (:110) directly testable; file already has explicit dialect guard comments (:49) |
| 22 | mac_history_service.go | 209 | 78 | 62.7% | c | remaining branches sqlite |
| 23 | notice_target_service.go | 71 | 71 | 0.0% | c | sqlite |
| 24 | notice_read_service.go | 70 | 70 | 0.0% | c | sqlite read-model |
| 25 | data_cache_service.go | 77 | 62 | 19.5% | b | takes `cache.Cache` interface (:39-45) — MemoryCache suffices; GetOrSet/Set/Delete/DeleteByPattern (`Keys`)/stats; existing test only covers CacheKeyBuilder |
| 26 | oper_log_service.go | 56 | 56 | 0.0% | c | RecordOperLog/RecordAsync sqlite |
| 27 | mac_history_heatmap_service.go | 52 | 52 | 0.0% | c | sqlite aggregation |
| 28 | device_credential_helper.go | 47 | 47 | 0.0% | c | helper fns, sqlite/pure |
| 29 | api_endpoint_service.go | 46 | 46 | 0.0% | c | sqlite CRUD |
| 30 | cache_config_service.go | 124 | 40 | 67.7% | c | already near target; 5 existing tests cover rate-limit; remaining = cache-duration config paths (sqlite sys_config-driven) |
| 31 | duty_stats_service.go | 32 | 32 | 0.0% | c | sqlite |
| 32 | mac_history_matview_service.go | 30 | 30 | 0.0% | c | PG matview DDL strings pure; sqlite skip branch |
| 33 | duty_holiday_service.go | 29 | 29 | 0.0% | c | sqlite |
| 34 | duty_config_service.go | 24 | 24 | 0.0% | c | sqlite |
| 35 | duty_service.go | 24 | 24 | 0.0% | c | sqlite |
| 36 | notice_cron_util.go | 22 | 22 | 0.0% | c | pure cron-expression builders (CalculateCronExpression/GenerateCronExpression/GetNoticeJobName/GetCommonCronExpressions) |
| 37 | template_cache.go | 18 | 18 | 0.0% | c | sync.Map cache — Get/Clear pure |
| 38 | swagger_extractor.go | 18 | 18 | 0.0% | c | pure swagger-JSON parsing (needs embedded swagger doc fixture) |
| 39 | token_blacklist_service.go | 45 | 12 | 73.3% | b | **already ≥70%** — SC-2 named it first but it needs only RemoveFromBlacklist (0%, :129) + rememberNegative branches (:115, 57.1%) + AddToBlacklist last branch (:61, 80%) to clear <50% floor trivially (already clear) |
| 40 | mac_history_cache_decorator.go | 12 | 12 | 0.0% | b | decorator over DataCacheService — MemoryCache |
| 41 | mac_perf_config_seed.go | 11 | 11 | 0.0% | c | sqlite seed |
| 42 | rate_limiter.go | 63 | 10 | 84.1% | c | already healthy |
| 43 | notice_query_service.go | 10 | 10 | 0.0% | c | buildUserVisibleQuery pure |
| 44 | mac_normalize.go | 15 | 1 | 93.3% | c | done |
| 45 | usage_logger.go | 9 | 0 | 100.0% | c | done |

**Cluster sums (unc):** cache-infra+small-pure ≈216 · duty family 380 · notice/template/operlog/api-endpoint 525 · knowledge+network+notification+auth 876 · mac family 763 · 外呼(email/api/ldap) 487 · device family 1261. Total 4613 unc.

**Reachability math for 70%:** executor/wire-blocked masses ≈ device family partial-block (~600-900 of 1261) + ad_ldap wire (~90) + email TLS happy (~50). Conservative "no-new-infra" reachable ≈ 3800 unc stmts; need 3053 → requires ~80% capture of reachable mass. **Tight but feasible; comfortable if DQ2 (device ForTesting export) is approved** — that alone unlocks ~500-700 additional executor-path stmts.

---

## 3. Test Double Strategy Table

Class legend: (c) sqlite/pure · (b) `cache.Cache` interface → MemoryCache (miniredis optional) · (a) `*device.DeviceExecutor` concrete · (d) HTTP/SMTP outbound · (e) LDAP/SNMP wire.

| Cluster | Files | Class | Technique | Prerequisite | Confidence |
|---------|-------|-------|-----------|--------------|------------|
| Cache infra | data_cache / cache_config / token_blacklist / template_cache / mac_history_cache_decorator / rate_limiter | b+c | `cache.NewMemoryCache()` implements `cache.Cache`; GetOrSet happy/miss/query-error/triple; DeleteByPattern via MemoryCache Keys; token RemoveFromBlacklist + negative-cache branches; template_cache sync.Map Get/Clear | none — all existing | HIGH |
| HTTP outbound | api_sender_service | d | **httpmock v1.4.2 (go.mod:17, INFRA-01 shipped)** — `httpmock.ActivateNonDefault` or transport-level; build* fns pure-table | none | HIGH |
| SMTP outbound | email_sender_service | d | plain fake SMTP server: `net.Listen("127.0.0.1:0")` + bufio converse `220→EHLO→250→AUTH 235→MAIL/RCPT/DATA 250→QUIT 221` (~80 lines) drives `sendPlainSMTP` + `Send` happy path; TLS paths: **dial-error branch only** (`InsecureSkipVerify:false` blocks self-signed); pure builders table-driven | none | HIGH for plan, MEDIUM for stmt yield |
| LDAP | ad_ldap_client | e | param-assembly + dial-error branches (`ldap.DialURL` against closed port) + pure helpers (formatUsername/extractRDN/encodePassword utf-16le + quote bytes/parseIntOrDefault/loadADTLSSkipVerify env matrix); wire ops deferred (78 DQ4 deferred vjeantet/ldapserver — do not reopen) | none | HIGH for partial yield |
| SNMP/discovery | device_discovery / device_monitor pingDeviceViaSNMP | e | pure IP math + CRUD first; `isAlive` via `net.Listen` real TCP peer (6-port loop hits listener on first port); snmpProbe: 78-04's `snmp_fake_server_78_04_test.go` is in-package **test-only — cannot import**; either copy ~150-200 lines into a root `_test.go` (DQ5) or skip | net.Listen trivial; copy decision = DQ5 | MEDIUM |
| Device executor family | device_info_collection / config_backup / mac_collection / device_monitor / config_execution / command_dispatch | a | Tier 1 (no infra): zero-value struct + direct parse-fn calls (existing precedent: `device_info_collection_service_test.go` `&DeviceInfoCollectionService{}` + `enrichChassisSerial`, 17 tests); sqlite CRUD; nil-executor guard branches. Tier 2 (needs DQ2): exported ForTesting helper in `internal/device` to seed `pool.connections` (all of `DeviceConnectionPool`/`DeviceTaskScheduler`/`DeviceExecutor` fields are **unexported** — cross-package seeding impossible today) | Tier 2 = DQ2 | HIGH Tier 1 / LOW Tier 2 until probed |
| sqlite CRUD masses | knowledge / network_device / duty×5 / notice×4 / template_service / notification_config / notification_sender / auth_credential / oper_log / api_endpoint / mac_history×4 / device_credential_helper / swagger_extractor / notice_cron_util / mac_perf_config_seed / mac_normalize | c | glebarez sqlite in `t.TempDir()` + `AutoMigrate(&models.X{})` (root precedent: `mac_history_query_service_test.go:66-73` AutoMigrate of MACOUIVendor) or raw DDL keyed to `models.TableName()` (Phase 73 P04 precedent); PG-only DDL string builders tested as pure string assertions | none | HIGH |

**Same-package collision note:** root is ONE package (45 files share `package services`); existing helper names already taken: `setupTestService` (mac_history_query_service_test.go:69), `loadSampleFixture` (device_info_collection_service_test.go:21). New helpers must use plan-suffixed names (`newSvc79xx`) per the 78 convention.

---

## 4. Reusable Helpers Inventory (all verified this session)

| Helper | Location | Status | Phase 79 use |
|--------|----------|--------|--------------|
| miniredis v2.38.0 | go.mod:8; usage precedent `internal/core/captcha_78_01_test.go:37-140` (`newCap78Mem`/`newCap78Redis`, `t.Cleanup` everything, NO t.Parallel) | shipped 78-01 | optional Redis-parity for DataCacheService DeleteByPattern/Keys; MemoryCache usually suffices |
| `cache.NewMemoryCache` | pkg/cache (used by captcha_78_01_test.go) | shipped | primary (b)-class double for `cache.Cache` params |
| `NewMultiLevelCacheSimple(l1,l2)` | pkg/cache/redis.go:547-555 (78-RESEARCH Gap G5) | shipped | only if a test needs MultiLevel semantics; R-7 safe (no L2Worker) |
| httpmock v1.4.2 | go.mod:17 | shipped 76-01 | api_sender_service sendRequest |
| `device.NewPooledConnectionForTesting(*network.Driver)` | internal/device/e2e_helpers.go:32 (non-test file, ForTesting contract + AST guard `for_testing_guard_test.go`) | shipped 76 | **only gives a PooledConnection — cannot seed pool.connections from another package** (fields unexported) |
| scrapligo public FileTransport API | `platform.NewPlatform(..., options.WithTransportType(transport.FileTransport), options.WithFileTransportFile(path))` — public repo APIs, used by `scrapli_wrapper_78_03_test.go:46-58` | shipped | lets an internal/services test build a `*network.Driver` WITHOUT the in-package var seam |
| 78-03 executor assembly helpers (`newFTWrapper78`/`newPool78`/`seedPool78`) | internal/device/*_78_03_test.go — **_test.go files, in-package only** | NOT reusable cross-package | explains the Tier-1/Tier-2 split above |
| root sqlite pattern | mac_history_query_service_test.go:69 `setupTestService` (note: uses `file::memory:?cache=shared` — prefer `t.TempDir()` file DB for new fixtures to avoid cross-test bleed) | existing | fixture template |
| mock-interface pattern (per-interface *Func fields, no testify/mock) | Phase 73 D-02 "portwrite 纯 mock 范本" (STATE.md Decisions) | established | `addomain.PasswordCipher` stub for auth_credential; notification sender deps |
| naming convention D-78-08 | `<source>_79_NN_test.go` same-package co-located | established | all new test files |
| var-seam discipline | "var 初值即原直调, 改前先 Cleanup 后覆盖, 禁 t.Parallel" (77-05 BLOCK-02 precedent, STATE.md) | established | only if any root var seam gets introduced (none identified — root files hold deps as struct fields, not package vars) |

---

## 5. Risk Register

| ID | Risk | Severity | Mitigation |
|----|------|----------|------------|
| R1 | **SC-1 "legacy cache services 群" clause references files not in root** (§1). Planner bakes a task list around nonexistent files → plan-checker/verifier fails SC-1 | HIGH | DQ1 before plan write. Re-anchor to the 4 root cache files or get user ruling |
| R2 | **Executor-path stmts unreachable cross-package** — DeviceConnectionPool.connections / TaskScheduler fields / Executor.scheduler all unexported (connection_pool.go, task_scheduler.go, executor.go struct defs verified); 78-03 seeding helpers are _test.go in-package. 1261-unc device family could stall at ~40-50% capture → package math tight | HIGH | DQ2. Tier-1 capture (parse/CRUD/nil-guards ≈ 600) + DQ2 approval unlocks remainder. Do NOT modify production structs to export fields |
| R3 | **email TLS happy paths unreachable** — `InsecureSkipVerify: false` hardcoded (email_sender_service.go:204,272); self-signed fake SMTP rejected by tls.Dial cert verify; no injection point | MEDIUM | Cover sendPlainSMTP (plain listener) + TLS dial-error branches + all pure builders ≈ 75% of file. Do not change production TLS config (D-12 zero-business-change discipline) |
| R4 | **Package test runtime already 377s** at 11.3%; adding ~30+ test files with sqlite fixtures could push `go test ./internal/services/` toward CI timeouts and slow per-task verification loops | MEDIUM | Use `t.TempDir()` file sqlite (fast open), no sleeps, table-driven, `go test -run TestXxx ./internal/services/` for per-task sampling; avoid `-race` full-package in every task (once per plan) |
| R5 | **Helper-name collisions in single package** — `setupTestService`, `loadSampleFixture` taken; 6 plans adding helpers in parallel risk redeclaration errors at build | MEDIUM | Plan-suffix every helper (`newXxx79NN`); planner enforces in plan have-slices |
| R6 | **PG-only paths (partition/matview)** — sqlite lacks `PARTITION OF`/matviews; over-ambitious DB-path tests would fail on dialect | MEDIUM | Files already carry explicit dialect guards (mac_history_partition.go:49,:110); test the DDL **string builders** as pure functions + sqlite skip branch only |
| R7 | **Quirk-lock discipline** — root files contain v1.26-era quirks; over-eager fixing violates the milestone's "锁定+注释, 不擅自扩修" rule (ROADMAP Phase 79 Notes L245-246) | MEDIUM | Follow Phase 73-04 precedent: 4 quirk locks documented NOT fixed; any newly-found quirk → comment + lock, zero behavior change |
| R8 | **token_blacklist already ≥70% but SC-2 names it first** — planner may over-invest | LOW | 12 unc stmts (RemoveFromBlacklist + 2 branch tails); fold into cache-infra plan, ~15 min work |

---

## 6. Decision Queue

### DQ1. SC-1 "legacy cache services 群" re-anchoring (BLOCKING — resolve before plan write)
The 6 named files live in `internal/services/system` (§1). Options: (a) re-anchor to root's 4 cache files (data_cache/cache_config/template_cache/mac_history_cache_decorator) 逐文件 ≥70% — recommended, zero scope change; (b) additionally pull system's 6 impl files into 79 (cross-package scope expansion, +~700 stmts); (c) declare clause satisfied by Phase 73 system coverage (53.5% package-level, per-file unverified). **Recommendation: (a)+(c) combined — state in plans that per-file ≥70% applies to the 4 root cache files, and note system impls already carry dedicated test files.** Requires user confirmation because it rewrites a ROADMAP success criterion.

### DQ2. Device executor path: export a ForTesting pool-seed helper, or cap Tier-1?
Adding e.g. `func SeedConnectionForTesting(pool *DeviceConnectionPool, deviceID string, conn *PooledConnection)` (+ maybe `NewDeviceConnectionPoolForTesting`) to `internal/device/e2e_helpers.go` follows the shipped ForTesting contract (non-test file + AST guard, precedent e2e_helpers.go:22-34) and unlocks executor fan-out paths in 6 root services (~500-700 stmts). Cost: one production-tree file touched (test-infra only, zero behavior change — same class as INFRA-02). Without it, device family tops out ~45-55% and the 70% package math leans on ~80% capture everywhere else. **Recommendation: approve the helper (mirrors INFRA-02 precedent); if user declines, plan 79-05 scope drops to Tier-1 + package target still reachable but with thin margin.**

### DQ3. Email fake SMTP scope
(a) plain fake SMTP (~80 lines) + dial-error branches + pure builders ≈ 70-80% of file — recommended; (b) additionally cover sendWithSTARTTLS plaintext-negotiation prefix (server replies 220 then STARTTLS 250 then test aborts at TLS handshake error) — adds ~20 stmts for ~30 lines. **Recommendation (a); (b) optional garnish inside the same plan.**

### DQ4. ad_ldap_client wire segment
Phase 78 DQ4 deferred vjeantet/ldapserver; addomain satisfied 70% via iface stubs. Root `LDAPClient` has no iface. Options: (a) param-assembly + dial-error + pure helpers only (~55-65% file, acceptable since SC-2 floor is <50% and package 70% is aggregate) — recommended; (b) copy 78-04-style minimal LDAP wire fake (high effort, low yield). **Recommendation (a).**

### DQ5. device_discovery snmpProbe / device_monitor pingDeviceViaSNMP UDP fake
78-04's fake server is in-package test-only. (a) copy the ~150-200-line fake into a root `_79_05_test.go` (self-contained, no production change); (b) skip SNMP probe paths, cover IP math + CRUD + isAlive-TCP only. **Recommendation: decide during 79-05 planning based on remaining gap math; (a) preferred if package <69% at that point, else (b).**

### DQ6. Test-file naming NN index
D-78-08 pattern = `<source>_78_NN_test.go` where NN = plan number. For Phase 79: `<source>_79_NN_test.go`, NN ∈ 01..06 matching the plan that creates it. Confirm so all 6 plans use one convention (recommended: yes, NN = plan number).

---

## 7. Plan-Split Recommendation (6 plans)

Cut by tech-domain cohesion (fixtures/helpers shared within a plan) + descending unc mass, honoring ROADMAP's "profile 倒序" intent at cluster level:

| Plan | Scope (files) | Unc mass | Tech | Key deliverable |
|------|--------------|----------|------|-----------------|
| **79-01** root cache 基建 + SC-2 named files | data_cache_service(62) + cache_config_service(40) + token_blacklist(12) + template_cache(18) + mac_history_cache_decorator(12) + rate_limiter(10) + mac_normalize(1) + notice_query(10) | ~165 unc → need all 4 cache files ≥70% (DQ1-a anchor) | MemoryCache/miniredis; sqlite for cache_config config paths | SC-1(cache-files) + SC-2 named-file discharge, tiny plan = fast first win + establishes fixture conventions |
| **79-02** duty family | duty_schedule(174) + duty_pool(97) + duty_stats(32) + duty_holiday(29) + duty_config(24) + duty_service(24) | 380 | sqlite + pure schedule algorithm | 6 files 0→70%+, incl. GenerateSchedule branch table |
| **79-03** notice/template/operlog/api-endpoint | notice_service(116) + notice_read(70) + notice_target(71) + notice_cron_util(22) + template_service(166) + oper_log_service(56) + api_endpoint(46) + swagger_extractor(18) | 565 | sqlite + pure render/cron/swagger-parse | 8 files 0→70%+ |
| **79-04** knowledge + network + notification + auth | knowledge(285) + network_device(202) + notification_config(127) + notification_sender(120) + auth_credential(142) | 876 | sqlite CRUD + AES round-trip + PasswordCipher stub + nil-sender dispatch | biggest sqlite CRUD plan |
| **79-05** 外呼三件套 + mac family | email_sender(189) + api_sender(119, httpmock) + ad_ldap_client(179 partial) + mac_history_query(358) + mac_collection(208 partial) + mac_history_service(78) + mac_history_partition(95) + mac_history_heatmap(52) + mac_history_matview(30) | ~1300 | httpmock + plain SMTP fake + dial-error LDAP + sqlite query/export + pure parse/DDL-string | largest plan; mac query alone is #1 gap file |
| **79-06** device family + phase 复测回填 | device_info_collection(275) + device_discovery(289) + config_backup(244) + device_monitor(189) + config_execution(152) + command_dispatch(112) + mac_perf_config_seed(11); then full-package re-measure + coverage-baseline.md 回填 + DQ2 helper (if approved) | 1272 (Tier-1 realistic 700-900) | zero-value parse tests + nil-guards + sqlite; optional ForTesting seed helper + SNMP fake copy (DQ5) | closes package ≥70%; SC-3 baseline row with `go tool cover -func` numbers |

**Ordering rationale:** 79-01 first (conventions + SC-2 named files + DQ1 anchor validated), 79-02/03/04 are independent sqlite clusters (any order; 79-04 largest → schedule while energy high), 79-05 second-largest + needs the most novel fixtures, 79-06 last (depends on DQ2 ruling + needs the gap math from prior plans to decide SNMP-fake depth). If planner prefers ROADMAP's literal grouping (cache A/B split), split 79-01 into dept-role-dict-equivalent → **not applicable** (see DQ1) — the literal ROADMAP plan titles are stale for the same reason as SC-1.

**Per-plan discipline (all plans):** `go build ./...` + targeted `go test ./internal/services/ -run ...` per task; full `go test ./internal/services/ -count=1` per plan close; one `-race` run per plan (no t.Parallel in var-seam/goroutine tests); quirk locks documented-not-fixed; file naming DQ6.

---

## 8. Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | all | ✓ | go.mod-driven (1.24) | — |
| miniredis/v2 | (b)-class parity tests | ✓ go.mod:8 | v2.38.0 | MemoryCache |
| httpmock | api_sender | ✓ go.mod:17 | v1.4.2 | httptest.Server |
| glebarez/sqlite | all (c)-class | ✓ go.mod:12 | v1.11.0 | — |
| testify | assertions | ✓ go.mod:27 | v1.11.1 | — |
| scrapligo public API | (optional) cross-package driver build | ✓ vendored via internal/device | — | skip executor paths |
| Redis server / Docker | none required | n/a | — | miniredis per INFRA-01 |

Missing-with-no-fallback: none. Windows dev caveats from prior phases still apply (fixture paths via filepath.Join; no ICMP in isAlive tests — TCP listener only).

---

## 9. Sources

### Primary (HIGH — measured or read this session)
- `go test -count=1 -coverprofile ./internal/services/` → 5202/589/11.3% + `go tool cover -func` per-file table (§2) — this session, 377s
- `ls internal/services/*.go` root inventory (45 non-test files) — this session
- `ls internal/services/{dept,role,dict,menu,user,post}*cache*.go` → all absent; system/ impl files present — this session (§1, DQ1)
- go.mod:8/:17/:12/:27 (miniredis/httpmock/sqlite/testify) — this session
- internal/device struct fields (connection_pool.go / task_scheduler.go / executor.go — all-unexported fields) — this session (R2)
- internal/device/e2e_helpers.go:22-34, scrapli_wrapper_78_03_test.go:40-75, executor_78_03_test.go:1-70, snmp_fake_server_78_04_test.go (in-package test-only) — this session
- email_sender_service.go:203-204/:271-272 (InsecureSkipVerify:false) — this session (R3)
- device_discovery_service.go isAlive (TCP-connect, 6 ports × 1s) — this session
- data_cache_service.go:39-120 (cache.Cache interface, GetOrSet sync-write comment P0 #9) — this session
- token_blacklist_service.go:32-53 + cov data (73.3%) — this session
- mac_history_query_service_test.go:69-75 (existing root sqlite fixture), device_info_collection_service_test.go:21-51 (loadSampleFixture + zero-value parse precedent)
- mac_history_partition.go:49/:91-94/:110 (PG-only guards), notification_config_service.go:246/:277 (AES pair), auth_credential_service.go:20 (PasswordCipher interface)
- .github/scripts/check-coverage.sh:164/:228 (root not in P1/P2 lists), :242-244 (P2_RATCHET rows — none for services root), .coverage-threshold (55.5)
- .planning/workstreams/milestone/ROADMAP.md L218-247 (phase def), STATE.md (73-04 quirk-lock + D-02 mock precedent, 77-05 var-seam discipline)

### Secondary (MEDIUM)
- .planning/research/v1.27-stack.md §1 (miniredis selection rationale), .planning/research/v1.27-features.md §6 (ranking)
- .planning/coverage-baseline.md (root 5202/589/11.3% row + system 53.5% Phase-73 row)
- 78-RESEARCH.md §1.3/§3/Gap G5 (naming, fixture lifecycle, MultiLevelCacheSimple)

### Tertiary (LOW — needs execution-time probe)
- Exact executor-path stmt yield with DQ2 helper (Tier-2 estimate 500-700 unprobed)
- SMTP fake line-count vs actual net/smtp conversation quirks (AUTH mech negotiation)
- Root-package `go test` wall-time after +30 test files

---

## Metadata

**Phase:** 79 — 长尾清欠·internal/services root
**Milestone:** v1.27 后端测试覆盖率优秀 II
**Authored by:** gsd-phase-researcher agent (spawned by `/gsd:plan-phase 79`)
**Out of scope:** internal/services/system package (DQ1-b would change this), .coverage-threshold bump (Phase 81), P2_RATCHET row deletions (Phase 81), production code changes (zero-business-change discipline; DQ2 helper would be the single sanctioned test-infra exception)
