# Phase 80: 长尾清欠·scheduler + 碎包 - Research

**Researched:** 2026-08-28
**Domain:** Go backend test coverage — `internal/scheduler` 引擎 + 6 个碎包 + 8 个小尾巴包
**Confidence:** HIGH on all coverage measurements + test-double classification (every package measured and every load-bearing claim read from source this session); MEDIUM on two execution-time probes (internal/api SetupRouter mini-Core yield; scheduler ADSyncScheduler wire paths)

**Measurement provenance (this session, one command per package):** `go test -count=1 -coverprofile=cov80_*.out <pkg>` → `rsplit` per-file aggregation over the coverprofile. Raw profiles deleted after research (repo-root temp-file discipline, CLAUDE.md).

---

## 0. Phase Goal & Scope (from ROADMAP.md L251-278)

**Goal:** internal/scheduler 引擎与全部碎包(api/v1 / models / internal/api / pkg/errors / pkg/cache / 小尾巴)逐一 ≥70%,TAIL 长尾清零、70% 数学缺口关门。

**Depends on:** Phase 76 INFRA-01 (miniredis + httpmock, both in go.mod) — 供 pkg/cache Redis 路径。与 Phase 77/78/79 无硬依赖,可并行穿插。

**Requirements:** TAIL-02, TAIL-03.

**Success Criteria (ROADMAP L257-262):**
1. `internal/scheduler` ≥70%(3.3% → ≥70%,补 ~736 stmts;注册表/执行器/引擎并发与取消分支)
2. 碎包逐包 ≥70%:api/v1(366)+ models(310)+ internal/api(291,装配层纯函数段)+ pkg/errors(183)+ pkg/cache(161,Redis 路径经 miniredis)
3. <50 stmts 小尾巴(permission/websocket/base/lldp/gormutil/middleware/logger/query)合计 ≥70%,确不可测的按既有豁免规则文档化
4. gate 全程绿

### 0.1 Measurement semantics correction (read first — affects every mass number)

ROADMAP's parenthesized numbers (366/310/291/183/161/**+736**) are **gap-to-70% covered-stmt counts** (0.7×total − covered), not totals. Verification: scheduler 0.7×1103−36 = 736.1 → ROADMAP "+~736" matches exactly; same formula reproduces 366/310/291/183 to ±1. Five of six still accurate. **One is stale:**

- **pkg/cache: ROADMAP says +161, actual gap is now +49** (924 tot / 598 cov / 64.7% this session; Phase 76/78 work covered memory.go 94.3%, l2_writer 80.7%, retry 86.0% since ROADMAP was authored). Planner should re-anchor 80-05's pkg/cache slice to +49 (see DQ5) — the freed ~112 stmts of margin flow to the 小尾巴 sweep.

### 0.2 Gate mechanics (SC-4 risk = LOW)

`.coverage-threshold` = 55.5 (weighted-avg only). **None of Phase 80's packages appear in P1_PACKAGES (check-coverage.sh:164) or P2_PACKAGES (check-coverage.sh:228)** — the P2 entry `internal/services/scheduler` is the *job-CRUD service* package (167 stmts @4.8%), a different path from Phase 80's `internal/scheduler` engine (1103 stmts). No P2_RATCHET rows exist for any Phase 80 package. Contribution is purely weighted-avg ratchet: any progress only raises the average; SC-4 cannot regress from Phase 80 work.

---

## 1. Per-Package Gap Tables (all measured this session)

### 1.1 Package summary

| Package | tot | cov | current | **gap to 70%** | ROADMAP said | Δ vs ROADMAP |
|---|---|---|---|---|---|---|
| internal/scheduler | 1103 | 36 | 3.3% | **+736** | +736 | ✓ |
| internal/api/v1 | 578 | 38 | 6.6% | **+367** | 366 | ✓ |
| internal/models | 445 | 1 | 0.2% | **+311** | 310 | ✓ |
| internal/api | 417 | 0 | 0.0% | **+292** | 291 | ✓ |
| pkg/errors | 326 | 45 | 13.8% | **+183** | 183 | ✓ |
| pkg/cache | 924 | 598 | 64.7% | **+49** | 161 | **-112 (shrunk)** |
| 小尾巴 (8 pkgs) | 1387 | 815 | 58.8% agg | **+156** | (SC-3 aggregate) | — |
| **Total** | | | | **+2094** | | |

### 1.2 internal/scheduler per-file (9 files, 55 funcs in cron.go)

| # | File | Stmts | Cov | Current | Class | Notes |
|---|------|-------|-----|---------|-------|-------|
| 1 | cron.go | 388 | 6 | 1.5% | c | Scheduler 引擎:Start/Stop/AddJob/UpdateJob/RemoveJob/StartJob/StopJob/ExecuteJob/GetJobStatus/GetJobCount 全 sqlite 驱动;JobExecutor.Execute/executeTask sqlite+registry;parseInvokeTarget/calculateNextRunTime 纯;defaultLogger 3 法直调;Logger/NoticeHub/DBGetter 接口 + SetNoticeHub/SetDB/SetGlobalScheduler var seams (cron.go:494-568) |
| 2 | workorder_tasks.go | 166 | 0 | 0% | c | 纯:generateWorkOrderNo/replaceVariables/getWeekdayName/calculateNextRunTimeCron (~45);sqlite:getTodayDutyPerson/assignWorkOrderHandler(4 分支)/syncWorkOrderJob/createWorkOrderJob/updateExistingWorkOrderJob/Disable·EnablePeriodicWorkOrderJob;var seam:GlobalNoticeHub nil-guard (workorder_tasks.go:242-247) |
| 3 | ad_sync_tasks.go | 162 | 27 | 16.7% | c+e | 已有 Start/Stop/Parallel 测试;新开:checkAndSyncADConfigs(sqlite ADConfig 行驱动 nil-LastSync/超间隔/未超间隔 3 分支 :195-255)/executeADAccountPoolRecoverBreakersTask(需 globalADSyncScheduler 非零 + sqlite sys_ad_service_accounts)/getDefaultADConfigID(sqlite)/ScheduleADSyncForConfig(ctx cancel)/GetADSyncStatus/getNextRunTime(纯 cron 数学)/Set·getADSM4Cipher(seam);syncADConfig 内层走 addomain wire → **部分 e 类豁免候选** |
| 4 | reconciliation_tasks.go | 130 | 3 | 2.3% | c | 已有 sqlite 内存库 fixture(reconciliation_tasks_test.go:29 `file:recon_task_...?mode=memory&cache=shared`)+ 9 tests;未开:RegisterReconciliationTasks 注册体 + createWorkorderBySeverity + checkPortStatusDrift + read/upsertDriftBaseline(全 sqlite);wsSvc/noticeSvc 形参 nil-safe(注册路径不广播) |
| 5 | dept_sync_tasks.go | 122 | 0 | 0% | c | getGlobalADAccountPool var seam(:22)/RegisterDeptSyncTasks/getDefaultADConfigIDForDept(sqlite)/executeDeptToADSyncTask + executeDeptMemberToADGroupSyncTask(pool interface → 同包测试可直接注入 stub 到 `ADSyncScheduler.pool` **unexported 字段,同包可写**) |
| 6 | mac_history_tasks.go | 49 | 0 | 0% | c | PartitionService/MACHistoryPurgeService 双 interface + Set* seams(:42/:51)→ stub 直注;upsertMACHistoryJob(sqlite);executeMACHistoryCleanup/Purge 分发 |
| 7 | vdi_sync_tasks.go | 48 | 0 | 0% | c | SetVDIVMService seam(cron.go:940);executeVDIVMSyncTask 分发 + syncAllEnabledVDIServers/syncSingleVDIServer(sqlite VDIServer 行)+ SyncVDIVMsManually |
| 8 | mac_history_matview_tasks.go | 23 | 0 | 0% | c | 单函数 RegisterMACHistoryMatViewTasks(s, db, matViewSvc services.MACHistoryMatViewService interface)→ stub 注入 + sqlite |
| 9 | reconciliation_fix_suggestion_monitor.go | 15 | 0 | 0% | c | 单注册函数,同 reconciliation 模式 |

**Key structural insight (R2-killer):** 全部 9 文件同属 `package scheduler` — 同包测试可直写 unexported 字段(`ADSyncScheduler.pool`/`sem`/`ctx`/`cancel`)与包级 var(GlobalNoticeHub/globalADSyncScheduler/GlobalDeviceMonitorService)。**Phase 79 的跨包 seeding 死角(device family R2)在此包不存在。** 已有 var-seam 测试惯例:cron_test.go:11-47 (save/restore + 并发)。

**Goroutine lifecycle audit (ROADMAP Notes 落实):**
- `Scheduler.Start` 起 robfig/cron goroutine(:246);`Stop` 先拿 `s.mu.Lock()` 再等 `cron.Stop()` ctx,上限 defaultShutdownTimeout=5s(:254-277)。测试纪律:**不测 Start 的用例一律不起 cron**(AddJob 在 `!running` 分支只落库+cron.go:294-297);凡 Start 必须 `t.Cleanup(s.Stop)`。
- `Stop()` 持锁等待期间,运行中任务若调用 `GetTaskHandler`(RLock)会阻塞到超时 → **测试不要在任务体内调 Scheduler 方法**,handler 用纯函数即可。
- `ADSyncScheduler.Start/Stop` 已有测试覆盖且无泄漏(ad_sync_tasks_test.go:25-75);其 `cron.Stop()` 不等待运行任务(:180-192),风险低。
- `JobExecutor.calculateNextRunTime` 每次起临时 cron 并 Start/Stop(:124-140)——测试调用频繁时注意它不泄漏(有 Stop),但别在循环里调上千次。

### 1.3 internal/api/v1 per-file (现有 5 个测试文件几乎全是"形状验证",未进 handler 体)

| # | File | Stmts | Cov | Current | Class | Reachable plan |
|---|------|-------|-----|---------|-------|----------------|
| 1 | auth.go | 241 | 14 | 5.8% | d | 唯一覆盖函数 parseUserAgent 82.4%;login/logout/refreshToken/loginLocalDirect/getAuthConfig/getPublicKey/testSM2/recordLoginLog/syncErrorReasonMessage/getNicknameOrUsername 全 0%。需 mini-Core fixture(§3.1);AuthFactory nil → fallback loginLocalDirect 分支(auth.go:157-162);loginLocalDirect 走 core.DB sqlite + CaptchaService 具体类型(:272-331) |
| 2 | captcha_background_handler.go | 149 | 0 | 0% | d | 9 个 handler 全走 core.CaptchaBackgroundService.GetDB() sqlite 查询(:33-88 起);fixture 用 78-01 的 sysCaptchaBackgroundDDL + db.Database{DB:sqlite} |
| 3 | job_cron_util.go | 67 | 0 | 0% | c | **全纯**:validate/parse/describe/generate + CronExpressionBuilder 链式 + EveryXxx 常量 → table-driven 一次收满 |
| 4 | ws_notice_handler.go | 53 | 24 | 45.3% | d | containsOrigin/newWebSocketUpgrader 已半覆盖;补 CheckOrigin 分支表(*、同源、localhost、白名单、拒绝)+ 真 WS 握手 httptest(现成 harness 模式,§4) |
| 5 | captcha_handler.go | 35 | 0 | 0% | d | getCaptcha/verifySlider/getConfig/reload → mini-Core + CaptchaService(78-01 pattern,§4) |
| 6 | job_utils.go | 30 | 0 | 0% | c | FormatDuration 纯 + GetJobStatistics sqlite(sys_job/sys_job_log 两表) |
| 7 | monitor_router.go + router.go | 3 | 0 | 0% | c | 装配,trivial |

**api/v1 现有测试为何无效(教训):** auth_integration_test.go:458 `TestIntegration_SetupAuthRouter_RouteRegistration` 注册的是**占位 handler**,从未调真 `SetupAuthRouter` → SetupAuthRouter 0.0%;auth_test.go 的 5 个 Login* 加密测试只验证请求结构体形状,不进 `login()` 体。新测试必须真正挂 handler 并发请求。

### 1.4 internal/models per-file — 不是"纯 struct 无逻辑可测"

445 stmts 里 **85 个非 TableName 函数,其中 59 个是 Scan/Value/BeforeCreate/BeforeUpdate 钩子**,其余是纯状态机/DTO 方法。ROADMAP 的"models 纯 struct 无逻辑可测"担忧(DQ 预设)不成立:

| 类别 | 代表 (stmt 数) | 测法 |
|------|----------------|------|
| driver Valuer/Scanner | StringArray(captcha_background:113-155)/DeviceIDList(config_execution:41-58)/TemplateVariable(s)(config_template:29-60)/DataSourceConfig+DisplayConfig+LayoutConfig(dashboard:85-270) | 纯 table-driven(round-trip + nil/坏输入分支) |
| GORM 钩子 | BaseModel/BaseTimeLine BeforeCreate(base.go:22/:37,**不触碰 tx,可 nil 直调**)/WorkOrder 族 6 个 BeforeCreate(workorder.go)/ADSyncLog/ConfigBackup BeforeCreate+BeforeUpdate/APIKeyUsageLog/Dashboard | 钩子体不引用 tx 的 nil 直调;引用的走 sqlite AutoMigrate 插入触发 |
| 状态机/断言 | ADServiceAccount IsAvailable/IsCircuitBroken/IsDisabled/StatusText(ad_service_account.go:54-82)/ADUser Is*ByUAC 三连(ad_domain.go:250-264)/ConfigBackup IsStoredIn*/CaptchaBackground IsEnabled/GetFileURL/GetAllowedShapesList/ToDTO/ParseStringArray | 纯表驱动 |
| 加密对 | vdi.go encryptVDIPassword/decryptVDIPassword(:106/:130,**AES-128-GCM 硬编码 key,纯 round-trip**) | 纯 |
| 树/聚合 | ADOU.Children/ADUser.GetGroupDNs(ad_domain.go:68/:144) | 纯 |
| 最大单文件 | captcha_background.go 46 / workorder.go 45 / vdi.go 36 / dashboard.go 29 / ad_domain.go 25 | — |

models 是同包测试(`package models`),TableName 存根(1 stmt × 51 个)本身就是免费 stmt——AutoMigrate 或显式调用即可命中。**+311 的缺口在 70% 目标下 ≈ 全包 444 unc 中的 311;上表类别基本全覆盖即超 80%。**

### 1.5 internal/api per-file

| File | Stmts | Cov | Current | Reachable plan |
|------|-------|-----|---------|----------------|
| router.go | 417 | 0 | 0% | 单文件包。`SetupRouter(r *gin.RouterGroup, core *core.Core, allowedOrigins []string)`(:98)是一次性装配:mini-Core + sqlite + gin.New() → 静态探查未发现 setup 期副作用(见下) → 预期一次覆盖 85-95% |

**SetupRouter setup-期依赖审计(逐行 grep 过):**
- 必须非 nil:`core.Config`(:53 解引用,零值 Config{} 走 encryption-disabled 分支)、`core.DB`(:619/:659 `core.DB.GetDB()` **直接解引用** → 需 `&db.Database{DB: sqliteDB, Type: "sqlite"}`,db.Database 的 `DB`/`Type` 字段导出,database.go:29-31)。
- 仅作构造参数(nil-safe):core.JWTManager(:116/:335/:359… 传入 middleware 构造器)、core.Cache、core.CacheConfigService(:265/:582/:659)、core.TokenBlacklistService、core.OperLogService(:118)。
- 副作用:setupNoticeHub(:86-95)`go noticeHub.Run()` 起 goroutine + `scheduler.SetNoticeHub(...)` 改全局 → 测试后 `hub.Stop()` + var 恢复(t.Cleanup;77-05 var-seam 纪律)。
- **无** setup 期 DB 查询、无 scheduler 任务注册、无 agent goroutine(grep `go func|.Start(|Register*Tasks|InitDefaultRoles` 在 router.go 零命中)。
- 模块 router 全部遵循 CLAUDE.md Router Pattern(构造器赋值,不执行查询)。

**残余不确定:** 某个模块 Setup 内部若对 core 服务字段做非常规调用会 panic → **80-04 开头 15 分钟 spike 验证**(R1);失败 fallback = 各模块 router 单独挂 + 只测 v1 层(仍是 router.go 的大多数 stmt)。

### 1.6 pkg/errors per-file — 全部纯函数,+183 低垂

| File | Stmts | Cov | Current | Notes |
|------|-------|-----|---------|-------|
| errors.go | 190 | 32 | 16.8% | ~25 个构造器(BadRequest/ParamMissing/RecordNotFound/Unauthorized/TokenExpired/PermissionDenied/…,errors.go:149-265+)全 0%,纯;Wrap/WrapWithHTTPStatus/WithContexts/GetCode/GetContext 纯;一条 table-driven 测试文件基本收满 |
| codes.go | 136 | 13 | 9.6% | DefaultHTTPStatus 60%/DefaultMessage 5.6%(codes.go:195/:219)→ 全 ErrorCode 枚举表驱动一遍即收 |

### 1.7 pkg/cache per-file — ROADMAP 数字已过时,+49 即达标

| File | Stmts | Cov | Current | 余量 |
|------|-------|-----|---------|------|
| redis.go | 394 | 123 | 31.2% | 271 unc 全部 miniredis 可达:GetJSON/SetJSON/GetInt/SetInt/GetBool/SetBool/MGetJSON/MSetJSON/MDelete/Increment·By/Decrement·By/Exists/FlushDB/getClient/getPrefix(redis.go:151-361)+ MultiLevelCache Get/Set/Delete/MGet/MSet/MDelete/Exists/Increment(:595-732)+ 4 个构造器(:519-578) |
| memory.go | 296 | 279 | 94.3% | 完成(78/79 已覆盖) |
| l2_writer.go | 114 | 92 | 80.7% | 余 drainQueue 0%(异步 goroutine,drain 触发难,豁免候选) |
| retry.go | 114 | 98 | 86.0% | 完成 |
| cache.go / errors.go | 6 | 6 | 100% | 完成 |

**+49 只需 miniredis 驱动 redis.go 的 JSON/typed helper(~6-8 个函数)。** 78-01 已建立 miniredis + 该包 client 的装配先例(§4)。MultiLevel 构造器注意 L2Worker goroutine(79-R7 教训):优先 `NewMultiLevelCacheSimple`(无后台 writer),起 worker 的构造器用后显式 Close + t.Cleanup。

### 1.8 小尾巴 8 包(SC-3,合计口径 58.8% → +156 达标)

| pkg | tot | cov | current | gap | 主要未开 |
|-----|-----|-----|---------|-----|----------|
| pkg/middleware | 609 | 419 | 68.8% | +8 | RequestDecryption 主函数 0%(request_decryption.go 116 tot/47 cov)、ResponseEncryption 12.5%(69 tot/12 cov)、OperLogMiddleware 0%(55 tot/29 cov);httptest + 真 RequestEncryptor(SM2 keypair 经 security.NewJWTManager,78-01 先例)+ sqlite(db 配置分支) |
| pkg/gormutil | 194 | 123 | 63.4% | +13 | join_builder.go 链式方法 Select/Where/Or/Order/Limit/Offset/Count/Scan/Find/First/Pluck/GetDB/Reset/BuildOnClause/BuildJoinClause/ParseSelectFields(:89-209)→ sqlite 链式 + 断言 SQL |
| pkg/permission | 114 | 30 | 26.3% | +50 | service.go **81 stmts 全 0%**:InitDefaultRolesAndMenus/createDefaultAdminRole/assignAllMenusToAdmin/GetRoleMenus/UpdateRoleMenus(sqlite sys_role/sys_menu/sys_role_menu) |
| internal/websocket | 129 | 46 | 35.7% | +45 | notice_hub.go BroadcastToUsers/BroadcastToAll/BroadcastRPAProgress×3/encodeMessage(:190-308)→ 真 WS 对(httptest+Upgrader+DefaultDialer,现成 harness §4)+ Stop() 收尾 |
| pkg/query | 105 | 71 | 67.6% | +3 | pagination.go 28 stmts 0%:NewPaginatedResult/Normalize/GetOffset/ListParams.ApplyOrder·ApplyPagination/DefaultQueryExecutor.Execute(纯+sqlite) |
| internal/services/lldp | 96 | 57 | 59.4% | +11 | ClassifyPort 纯(port_classifier.go:35,0%)+ lldp_service.go cache-hit 分支(:31 起,NewLLDPCache 种子)+ executor 错误分支 |
| pkg/logger | 79 | 55 | 69.1% | +1 | Fire 钩子(:112)/fallbackToStdLogger(:213)/Fatal(:269)→ logrus 直调 |
| internal/services/base | 61 | 14 | 23.0% | +29 | service.go NewGORMRepository/Create/Update/Delete/GetByID/List/BatchDelete/WrapError/IsNotFound/IsDuplicate(:51-159)→ sqlite + 一个内嵌测试 model;list_request.go ApplySort(:75) |

SC-3 的"<50 stmts"措辞与实际不符(pkg/middleware 609 stmts)——原意应为"<50% 覆盖的包"(v1.26 扫描时这 8 包全部 <50%)。语义按"ROADMAP 点名的 8 包合计 ≥70%"执行(见 DQ4)。

---

## 2. Test Double Strategy Table

Class legend: (c) sqlite/纯 · (b) `cache.Cache` → MemoryCache/miniredis · (d) HTTP/WS outbound via httptest · (e) wire-blocked(豁免候选)。

| Plan | Targets | Class | Technique | Prerequisite | Confidence |
|------|---------|-------|-----------|--------------|------------|
| 80-01 | cron.go 引擎 + JobExecutor | c | sqlite(t.TempDir 文件库或 `file::memory:?cache=shared`)+ 手工建 sys_job/sys_job_log DDL;注册 handler 用闭包计数器验证 executeTask 分发;Start/Stop 生命周期用短 cron("*/1 * * * * *")+ t.Cleanup(s.Stop);parseInvokeTarget/calculateNextRunTime 纯表驱动;defaultLogger 三方法直调 | 无(全部现成) | HIGH |
| 80-02 | 6 个 *_tasks.go | c | var seams 直写(同包):SetPartitionService/SetMACHistoryPurgeService/SetVDIVMService/GlobalNoticeHub/globalADSyncScheduler;interface stub(per-interface struct,73-04 D-02 范式);sqlite 行驱动 checkAndSyncADConfigs 3 分支/assignWorkOrderHandler 4 分支/VDI sync;纯函数表驱动(workorder 4 个 + getNextRunTime) | 无 | HIGH(engine 分发)/MEDIUM(ADSyncScheduler wire 内层,§3.2) |
| 80-03 | api/v1 + models | d+c | **mini-Core fixture**(§3.1,CaptchaService 具体类型无法 stub → 必须真装配)+ httptest/gin;models 纯表驱动 + sqlite AutoMigrate 触发钩子 | mini-Core fixture(~150 行) | HIGH |
| 80-04 | internal/api + pkg/errors | d | SetupRouter 单测:mini-Core + sqlite + gin.New() + t.Cleanup(hub.Stop + SetNoticeHub 恢复);pkg/errors 纯表驱动 | 15-min spike(R1) | MEDIUM(spike 前)/HIGH(后) |
| 80-05 | pkg/cache + 小尾巴 | b+d+c | miniredis 驱动 redis.go typed helpers;httptest WS 对(websocket Broadcast)+ 真 RequestEncryptor(middleware 加解密)+ sqlite(permission service/gormutil/base/query/lldp) | 无 | HIGH |

---

## 3. Fixtures To Build (new, this phase)

### 3.1 mini-Core fixture (80-03/80-04 共用,keystone)

```go
// 来源依据: core.Core{CoreInfra, CoreServices} (core.go:104-107);
// db.Database{DB,Type} 导出字段 (database.go:29-31);
// NewCaptchaService(db *db.Database, cache cache.Cache) (core/captcha.go:51);
// security.NewJWTManager(&cfg.JWT) (core.go:127 先例); 78-01 captcha_78_01_test.go 装配范式
func newMiniCore80(t *testing.T) (*core.Core, *gorm.DB) {
    t.Helper()
    gormDB, _ := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "p80.db")), &gorm.Config{})
    t.Cleanup(func() { if db, err := gormDB.DB(); err == nil { db.Close() } })
    coreDB := &coredb.Database{DB: gormDB, Type: "sqlite"}   // 内部包, api/v1 与 internal/api 均可 import
    mem := cache.NewMemoryCache()
    cfg := &config.Config{}                                   // 零值 → RequestEncryption disabled 分支
    jwtMgr, _ := security.NewJWTManager(&cfg.JWT)             // SM2 keypair 由 manager 内生成
    infra := &core.CoreInfra{Config: cfg, DB: coreDB, JWTManager: jwtMgr,
        PwdManager: security.NewPasswordManager(nil),
        CaptchaService: core.NewCaptchaService(coreDB, mem)}
    c := &core.Core{CoreInfra: infra, CoreServices: &core.CoreServices{}}
    // 按用例补建表: sys_user/sys_user_role/sys_role/sys_oper_log/sys_config/sys_captcha_background/...
    return c, gormDB
}
```

注意:该 fixture 在 80-03 与 80-04 两个**不同包**中各放一份(命名 `newMiniCore80NN`),不能跨包共享 _test.go。CoreInfra/CoreServices 字段名以 core_infra.go/core_services.go 实际定义为准(plan 期照抄)。

### 3.2 scheduler 引擎 fixture(80-01/80-02)

- 引擎测试 DB:同包内 `newSchedDB80(t)`(sqlite + `sys_job`/`sys_job_log` DDL;参考 reconciliation_tasks_test.go:29 的内存库 + models.TableName())。
- handler 注册计数器:`registerCountingTask80(s *Scheduler) *int32` — RegisterTask 闭包 atomic.AddInt32,ExecuteJob 后断言。
- var-seam 恢复范式照抄 cron_test.go:11-47(save → defer restore)。

### 3.3 WS 对 harness(80-03/80-05,已有范本)

`internal/websocket/notice_hub_readpump_test.go:26-127` 已有三处完整范式:httptest.NewServer + gorilla Upgrader + `websocket.DefaultDialer.Dial(wsURL, nil)` + defer 关闭。直接复制该模式,不新建基建。

---

## 4. Reusable Helpers Inventory (all verified this session)

| Helper | Location | Status | Phase 80 use |
|--------|----------|--------|--------------|
| miniredis/v2 v2.38.0 | go.mod:8;装配先例 internal/core/captcha_78_01_test.go(:37-140) | shipped 78-01 | pkg/cache redis.go typed helpers;CaptchaService fixture |
| `cache.NewMemoryCache` | pkg/cache/memory.go | shipped | mini-Core cache 依赖;(b) 类缺省 |
| `core.NewCaptchaService(db,cache)` | internal/core/captcha.go:51(具体类型,非接口) | shipped | mini-Core 必备(login captcha 分支/captcha_handler) |
| `security.NewJWTManager(&cfg.JWT)` | internal/core/security/jwt.go | shipped | login 令牌对/logout 黑名单/refresh/getPublicKey/testSM2 |
| 78-01 sysCaptchaBackgroundDDL | internal/core/captcha_78_01_test.go:41-60 | shipped | captcha_background_handler 表结构(直接复制到 80-03 fixture) |
| WS httptest harness | internal/websocket/notice_hub_readpump_test.go:26/:127/:205 | shipped | websocket Broadcast 测试 + api/v1 WS 握手 |
| sqlite 内存库 + DDL 范式 | internal/scheduler/reconciliation_tasks_test.go:29 | shipped | 80-01/80-02 引擎 fixture |
| var-seam 纪律(save/Cleanup/restore,禁 t.Parallel) | cron_test.go:11-47 现例 + 77-05 BLOCK-02 惯例(STATE.md) | established | 所有全局 var(GlobalNoticeHub/SetNoticeHub/globalADSyncScheduler/GlobalDeviceMonitorService) |
| per-interface stub struct(无 testify/mock) | 73-04 D-02 范本(STATE.md Decisions) | established | PartitionService/MACHistoryPurgeService/VDIVMService/MACHistoryMatViewService/NoticeHub |
| 命名 D-78-08 | `<source>_80_NN_test.go`,NN=plan 号 | established | 全部新测试文件 |
| glebarez/sqlite v1.11.0 / testify v1.11.1 / httpmock v1.4.2 | go.mod:12/:27/:17 | shipped | 全 (c)/(d) 类 |

**不可复用项:** 79 的跨包 seeding 教训(device unexported 字段)在此不适用——scheduler 全同包;internal/core 的 captcha 测试是 in-package _test.go,fixture 代码需复制而非 import。

---

## 5. Risk Register

| ID | Risk | Severity | Mitigation |
|----|------|----------|------------|
| R1 | **SetupRouter mini-Core 未知 panic**——417 stmts 一次性装配,某模块 Setup 对 nil 服务做非常规调用即中断 | HIGH | 80-04 首任务 15-min spike:先只挂 system 组,逐组扩展;fallback = 按模块组拆测试,只测可达组 + 差额由 pkg/errors(纯,余量大)吸收。**勿为此改生产代码**(零业务变更纪律) |
| R2 | **Stop() 持锁 5s 等待 × 多用例 → 套件变慢 + -race 下 goroutine 泄漏告警**;GetTaskHandler(RLock) 与 Stop(Lock) 互斥 | HIGH(质量)/MEDIUM(时间) | 凡 Start 必 t.Cleanup(Stop);handler 内不调 Scheduler 方法;不 Start 即可测的路径(AddJob !running 分支/StartJob/StopJob/ExecuteJob 直调 Executor)优先;每 plan 一次 -race |
| R3 | **api/v1 login fixture 复杂度**——需真 CaptchaService+JWTManager+5 张表,表缺列 → 级联失败(78-01 DDL 精简先例) | MEDIUM | fixture 只建 handler 实际引用的列(照 78-01 精简哲学);先落 loginLocalDirect 成功/锁定/禁用/密码错 4 分支再扩 |
| R4 | **models Scan/Value 的 pq.StringArray(lib/pq)驱动值形差异**——sqlite 下 driver.Value 形状与 PG 不同 | MEDIUM | Scan/Value 纯直调表驱动([]byte/string/nil 三态),不走真实 PG 数组路径;AutoMigrate 触发仅验 BeforeCreate 链 |
| R5 | **websocket Broadcast 测试 -race 抖动**(goroutine 写 conn 时序) | MEDIUM | 照抄 readpump 测试的关闭顺序(server shutdown → conn close → hub Stop);断言用 Eventually 而非 sleep(记忆:local-vs-ci 教训) |
| R6 | **pkg/middleware 加解密中间件**——response 缓冲语义(writeBuffer 于 flush)与 replay 窗口分支边界 | MEDIUM | 参照 pkg/crypto 自身测试的 keypair 生成;timestamp 边界用 ±window±1 表驱动,不依赖墙时钟等待 |
| R7 | **internal/api setupNoticeHub 全局副作用**(SetNoticeHub + go Run) | LOW | t.Cleanup:hub.Stop() + SetNoticeHub(原值);禁 t.Parallel |
| R8 | **SC-1/SC-3 措辞过时引发验收争议**(pkg/cache +161→+49;"<50 stmts"→"<50% 包") | LOW | DQ4/DQ5 在 plan 写明修正口径,VERIFICATION 按"8 包点名 + 修正后 gap"验收 |

---

## 6. Decision Queue

### DQ1. internal/api SetupRouter 测试策略(BLOCKING 80-04 plan 写作)
(a) **单一大装配测试**(一个 mini-Core + gin.New() 挂全 SetupRouter,一次覆盖 ~85-95%,失败时二分定位)——推荐,spike 先行;(b) 按模块组拆 10+ 个小测试(定位快但 fixture 重复 10 倍)。**推荐 (a),另加 2-3 个 spot-check 请求(如 GET /api/v1/system/auth/config 走通 404/405 分支)。**

### DQ2. scheduler 引擎并发/时序断言口径
(a) **直调 Executor.Execute + Start/Stop 生命周期用例,不做 cron 触发时序断言**(无 sleep、无实时等待)——推荐,符合"测试禁时序 flake"纪律;(b) 附加 1-2 个真实 cron 触发用例(短表达式 + Eventually 轮询)。**推荐 (a) 为主;若 80-01 收口后缺口 >50 stmts 再加 (b) 的单用例。** 引擎"并发"SC 由既有 cron_test.go 并发范式 + 新增 GetJobCount/GetJobStatus 并发读覆盖。

### DQ3. api/v1 mini-Core 保真度
(a) **真 CaptchaService + 真 JWTManager + sqlite 表**(唯一可行——CaptchaService 是具体类型 core/captcha.go:22,无法接口 stub)——推荐;(b) 为可测性把 CoreServices.CaptchaService 改接口 = **生产结构变更,违反零业务变更纪律,否决**。fixture 按 §3.1,80-03 与 80-04 各持一份同形副本。

### DQ4. SC-3 口径修正
"<50 stmts 小尾巴"实际为 8 个点名包(middleware 609 stmts 并非 <50 stmts)。**建议按"ROADMAP 点名的 8 包合计 ≥70%"执行并允许逐包豁免**(如 logger Fire 钩子、l2_writer drainQueue、middleware 加解密的极端分支),豁免条目按既有豁免规则在 plan/VERIFICATION 中文档化。计划阶段确认即可,无需用户裁决(语义唯一合理解读)。

### DQ5. pkg/cache 缺口修正(+161 → +49)
Phase 76/78 已把包覆盖推到 64.7%。80-05 的 pkg/cache 切片重锚为 **+49(miniredis typed helpers)**,省出的 plan 容量拨给小尾巴 sweep。计划阶段确认即可。

### DQ6. ADSyncScheduler wire 内层豁免口径
syncADConfig 内层走 addomain LDAP wire(78 DQ4 已 deferred vjeantet/ldapserver)。**建议:checkAndSyncADConfigs 的 3 个调度分支 + executeADAccountPoolRecoverBreakersTask(sqlite)+ 纯函数收满,wire 段(syncADConfig 体 ~26 stmts)按 e 类豁免文档化**,与 79-DQ4 口径一致。

---

## 7. Plan-Split Recommendation (维持 ROADMAP 5 plans,边界微调)

ROADMAP L264-269 的 5-plan 切分与实测质量分布吻合,仅按 §1 实测数字校准范围与预期:

| Plan | Scope | gap | 关键交付 |
|------|-------|-----|----------|
| **80-01** scheduler 引擎 + 执行器 | cron.go 全部:JobExecutor.Execute/executeTask/parseInvokeTarget/calculateNextRunTime + Scheduler 10 个公开方法 + defaultLogger + var seams(GetNoticeHub/GetDB/GetGlobalScheduler) | ~382 | sys_job/sys_job_log fixture;Start/Stop 生命周期 + t.Cleanup 纪律;为 80-02 提供 handler 注册底座 |
| **80-02** scheduler 任务族 | workorder/ad_sync/dept_sync/mac_history/mac_matview/vdi_sync/reconciliation(+fix_suggestion) 7 任务文件 | ~354 | var-seam + interface stub + sqlite 分支表;DQ6 wire 豁免文档化 |
| **80-03** 碎包 A:api/v1 + models | api/v1 六文件(mini-Core handler 测试 + job_cron_util 纯)+ models 全类别 | 367+311 = **678(最大)** | mini-Core fixture 建立并沉淀给 80-04;models 纯表驱动批量收口 |
| **80-04** 碎包 B:internal/api + pkg/errors | SetupRouter 装配(spike 先行,DQ1-a)+ pkg/errors 纯表驱动 | 292+183 = 475(边际工作量小) | R1 spike 结论回填;pkg/errors 一条 table-driven 大文件 |
| **80-05** 碎包 C:pkg/cache + 小尾巴清尾 + 回填 | pkg/cache +49(DQ5)→ 8 小包 sweep(+156)→ 全包复测 + coverage-baseline.md 回填 | 49+156 = 205 | miniredis typed helpers;WS Broadcast;permission service sqlite;**SC 逐包验收表** |

**顺序依据:** 80-01 先行(引擎是 80-02 的底座 + fixture 最独立);80-02/80-03 无相互依赖可穿插;80-04 依赖 80-03 的 mini-Core 形状(复制);80-05 收尾(聚合 8 包口径 + 复测回填是 SC 验收动作)。总量 +2094,5-plan 均摊 ~420/plan,与 79 的 6-plan×~510 节奏相当。

**Per-plan discipline(全 plan 通用):** 每任务 `go build ./...` + 定向 `go test -run`;每 plan 收口跑全包 `-count=1`;每 plan 至少一次 `-race`(goroutine 包:scheduler/websocket 必跑);命名 `<source>_80_NN_test.go`;quirk 锁定不扩修;临时 cov*.out 即用即删。

---

## 8. Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | 全部 | ✓ | go.mod 1.24 | — |
| miniredis/v2 | pkg/cache typed helpers、CaptchaService | ✓ go.mod:8 | v2.38.0 | MemoryCache(仅 (b) 类) |
| glebarez/sqlite | 全部 sqlite fixture | ✓ go.mod:12 | v1.11.0 | — |
| testify | 断言 | ✓ go.mod:27 | v1.11.1 | — |
| gorilla/websocket | WS harness | ✓ (websocket 包已在用) | — | — |
| httpmock | (本阶段无 HTTP outbound 需求) | ✓ go.mod:17 | v1.4.2 | — |
| Redis/Docker/PG | 无需(miniredis 顶替) | n/a | — | — |

Missing-with-no-fallback: none。Windows 注意事项沿袭:t.TempDir() 文件库、无 ICMP、fixture 路径 filepath.Join、禁本地绝对路径断言。

---

## 9. Project Constraints (from CLAUDE.md)

- **GSD workflow enforcement:** 文件改动须经 GSD 命令入口;本 phase 由 `/gsd:plan-phase 80` 派生,合规。
- **零业务变更纪律**(milestone 惯例):不为可测性改生产结构(DQ3-b 否决即此);唯一可能的例外形态是 78 先例的 ForTesting helper——**本阶段未识别出此需求**(同包测试覆盖了 scheduler,无跨包 seeding 死角)。
- **测试纪律(用户记忆 local-vs-ci):** 禁本地绝对路径/目录名断言;map 序/异步用 ElementsMatch + Eventually;`-count=10` 筛 flake;push 前三件套 build+test+lint。
- **临时文件清理:** 根目录 temp_*.go/test_*.go 会 main redeclared;cov*.out 用后即删。
- **status 0/1 惯例:** sys_job 的 status(0=正常/1=暂停)在 StartJob/StopJob 断言中按 `models` 常量引用,不硬编码字面量。
- **Commit 需用户确认**(CLAUDE.md Git Workflow);本 RESEARCH.md 的提交走 gsd-sdk commit 通道。

---

## 10. Phase Requirements

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TAIL-02 | scheduler 引擎与碎包逐包 ≥70%(SC-1/SC-2) | §1.1-1.7 逐文件 gap 表 + §2/§3 test-double 策略 + §4 复用清单 |
| TAIL-03 | 小尾巴 8 包合计 ≥70% + 豁免文档化(SC-3) | §1.8 逐包余量表(+156)+ §6 DQ4 口径 + §5 R8 |
| (gate) | SC-4 gate 全程绿 | §0.2:80 的包均无 P1/P2 floor,只推加权均值,单向上升 |
</phase_requirements>

---

## 11. Open Questions

1. **SetupRouter 实际可达率**(R1/DQ1)——静态审计强烈支持,但 417 stmts 单测只有跑了才知道。Recommendation: 80-04 首任务 spike,plan 里写成 checkpoint-able 验证步骤。
2. **models BeforeCreate 钩子里引用 tx 的具体个数**——base.go 两个已确认 nil-safe,workorder/config_backup/dashboard 等未逐一读体。Recommendation: 80-03 统一用 sqlite AutoMigrate 插入触发(不依赖 nil-tx 假设),纯 nil 直调仅限已确认的。
3. **captcha_background_handler 9 handler 的 core.CaptchaBackgroundService 字段装配位置**(CoreServices 内具体类型与构造器签名)——80-03 落 fixture 时照 core_services.go 现场确认,风险低。

## 12. Sources

### Primary (HIGH — measured or read this session)
- `go test -count=1 -coverprofile` × 8 组(scheduler / api/v1 / models / internal/api / pkg/errors / pkg/cache / 小尾巴 8 包)+ per-file rsplit 聚合 — §1 全部数字
- internal/scheduler/cron.go:1-160/:160-499(引擎/Executor/Stop 5s 超时/全局 seams)、workorder_tasks.go:139-262、ad_sync_tasks.go:20-260、dept/mac/vdi/recon 各 outline
- internal/api/v1/auth.go:61-331(login 分支/loginLocalDirect 依赖)、auth_test.go:342-410、auth_integration_test.go:458-505(占位 handler 教训)、ws_notice_handler.go:1-80、captcha_background_handler.go:33-60
- internal/api/router.go:38-120/:258-265/:619/:659(setup 期依赖审计)
- internal/core/core.go:104-151(Core/Infra/Services)、core_infra.go:33、core/captcha.go:21-52、core/db/database.go:29-40
- internal/models/base.go:22-40、vdi.go:106-140、workorder.go:110-365、captcha_background.go:113-188、ad_service_account.go:54-82、ad_domain.go:68-264
- pkg/errors codes.go:195/:219、errors.go:149-265;pkg/cache redis.go:151-361/:519-732
- 小尾巴:websocket/notice_hub.go:74-349 + notice_hub_readpump_test.go(WS harness)、permission/service.go、gormutil/join_builder.go、base/service.go、query/pagination.go、lldp lldp_service.go:31/port_classifier.go:35、pkg/middleware response/request_encryption + oper_log
- .github/scripts/check-coverage.sh:164/:228/:242-244、.coverage-threshold(55.5)
- .planning/workstreams/milestone/ROADMAP.md L251-298(Phase 80/81 定义)、coverage-baseline.md(起点口径)

### Secondary (MEDIUM)
- 78-RESEARCH.md §1.3/§4(fixture 生命周期/D-78-08 命名)、79-RESEARCH.md §3-§7(范式、命名、缺口语义)— 本文件结构与其对齐
- STATE.md(77-05 var-seam 纪律、73-04 D-02 mock 范本)

### Tertiary (LOW — execution-time probe)
- SetupRouter 实际 stmt 产量(spike 后回填)
- models 部分钩子 tx 依赖细节
- ADSyncScheduler wire 段豁免后的实际包内百分比落点

---

## Metadata

**Phase:** 80 — 长尾清欠·scheduler + 碎包
**Milestone:** v1.27 后端测试覆盖率优秀 II
**Authored by:** gsd-phase-researcher agent (spawned by `/gsd:plan-phase 80`)
**Out of scope:** `internal/services/scheduler`(job CRUD 服务包,P2 floor 已达标轨道,勿与引擎混淆)、`.coverage-threshold` 上调与 P2_RATCHET 行删除(Phase 81)、生产代码变更(DQ3-b 否决)、system 子包测试补齐
