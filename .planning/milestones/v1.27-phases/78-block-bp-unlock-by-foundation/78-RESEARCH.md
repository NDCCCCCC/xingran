# Phase 78: 阻塞包攻破·基建解锁 (core + device + addomain) - Research

**Researched:** 2026-08-27
**Domain:** Go backend test coverage for 3 structural BLOCK packages (core / device / addomain) using INFRA-01~03 + QUIRK-01 from Phase 75/76
**Confidence:** HIGH on local seams, MEDIUM on go-ldap wire-level testing, LOW on miniredis + scheduler goroutine interaction

---

## 0. Phase Goal & Scope (from ROADMAP.md L183-216)

**Goal:** Push `internal/core` (754 stmts @ 40.2%), `internal/device` (1249 stmts @ 39.1%), `internal/services/addomain` (2415 stmts @ 21.8%) all to ≥70% using INFRA-01 (miniredis) / INFRA-02 (ScrapliWrapper Driver factory) / INFRA-03 (LDAPClientIface) from Phase 76 and QUIRK-01 from Phase 75. Retires 3 P2_RATCHET exemptions in `check-coverage.sh` (core 38.33 / device 39.07 / agent-server 22.08 — agent-server already retired in Phase 77, only core + device remain after Phase 78).

**Depends on:** Phase 75 (Q-7 Stop 幂等 is core Init Close 收尾前置; Q-3/Q-8 修正好让 device 提取器测试断言新行为) + Phase 76 (INFRA-01 miniredis / INFRA-02 Driver 工厂 / INFRA-03 LDAPClientIface).

**Requirements (from REQUIREMENTS.md):** BLOCK-03, BLOCK-04, BLOCK-05.

**Success Criteria:**
1. `internal/core` 根包 ≥70% (40.2% → ≥70%; Init 链 302 stmts 经 miniredis+sqlite config 分支跑通并以 Close 收尾; captcha 98 stmts 纯补 + QUIRK-01 解锁的真实链路直测).
2. `internal/device` ≥70% (39.1% → ≥70%; FileTransport 照搬 portwrite 先例解锁 scrapli_wrapper/connection_pool/executor 346 stmts, x/crypto/ssh fake 补 Open/transport 路径, snmp UDP 对端 + task_scheduler 并发/取消分支).
3. `internal/services/addomain` ≥70% (21.8% → ≥70%, 补 ~1165 stmts; 两段式: sqlite+`[]*ldap.Entry` 段 ~535 先行, Iface stub 段 ldap_client 159 + failover 入口收尾).
4. check-coverage.sh 两条 P2_RATCHET 豁免行 (core 38.33 / device 39.07) 对应包全部实测 ≥70% (豁免行的删除动作统一留 Phase 81 收口).
5. 每个 plan 完成点 `go test ./...` 全绿, gate 不倒退.

**Proposed Plans (ROADMAP L201-208, TBD):**
- 78-01 core captcha 真实链路 (QUIRK-01 解锁) + captcha_background (文件+ DB) + metrics_cache 边缘
- 78-02 core Init 链 (miniredis+sqlite+reaper re-exec+Close 收尾; plan 首任务为 Init 可跑深度探针实验)
- 78-03 device scrapli_wrapper + connection_pool + executor (FileTransport + fake SSH, fake server 输出 prompt)
- 78-04 device snmp_client UDP 对端 + task_scheduler 剩余分支
- 78-05 addomain sqlite 段 A: sync.go 全链 (`[]*ldap.Entry` 驱动, syncOUs/syncGroups/syncUsers/syncGroupMembers)
- 78-06 addomain sqlite 段 B: computer.go + ou_group_mapping/group_config/config 纯 CRUD + account_pool 剩余 (MarkSuccess/RecoverExpiredBreakers)
- 78-07 addomain Iface stub 段: ldap_client 参数/错误分支 + FailoverClient 遍历 + user.go/group.go failover 入口

---

## 1. Pre-existing Research That Applies

### 1.1 v1.27-features.md (signature-by-signature unc analysis, dependency graph)

Already-shipped findings (verified during Phase 75-77 execution, 100% applicable to Phase 78):

| Section | Finding | Phase 78 Impact |
|---------|---------|------------------|
| §1.1 addomain TOP file table | sync.go 316/316 unc, computer.go 219/254, ldap_client.go 159/163 unc | Drives plan 78-05/06/07 statements targets |
| §1.2 seam 1-5 | `LDAPClientIface` 19 methods + mockLDAPClient, `[]*ldap.Entry` data-driven boundary, `AccountPool` interface, `updateUserAttributeFn` hook, `clientFactory` seam in FailoverClient | All seams ready; planner just uses them |
| §2.2 operations | (c) class dominated, no new infra | Out of Phase 78 (already 70%+) |
| §3.2 device seam 1-4 | `e2e_helpers.go` ForTesting factory, FileTransport 先例, connection_pool_test.go, gosnmp无接口抽象 | Confirms Block-04 strategy |
| §4.2 core seam 1-4 | CoreInfra/CoreServices 拆分 + core_split_compat_test.go, MemoryCache, Config 全驱动, go.mod 无 miniredis | Phase 76-01 INFRA-01 already added miniredis |
| §6 input/output ranking | addomain rank 5 (largest) — needs Iface + (a) class | Confirms 2-phase structure |

**CRITICAL CORRECTION discovered during this research:** v1.27-features.md §1.2 claim #2 stated:

> **FailoverClient 断层**：`ExecuteWithFailover(ctx, func(client *LDAPClient) error)` 收**具体类型** `*LDAPClient` ... mock 无法进入

**THIS IS OUTDATED.** Reading `internal/services/addomain/failover_client.go:44-47` directly:

```go
func (f *FailoverClient) ExecuteWithFailover(
    ctx context.Context,
    operation func(client LDAPClientIface) error,
) error {
```

Phase 76-03 INFRA-03 already converted the signature to `LDAPClientIface` (interface) and added a `clientFactory` injection seam at failover_client.go:25:

```go
clientFactory func(*models.ADConfig, *models.ADServiceAccount) LDAPClientIface
```

Existing test `failover_client_76_03_test.go` proves this works (uses `fc.clientFactory = func(...)` injection for mockLDAPClient). **The "concrete type 断层" no longer exists** — Phase 78-07 (FailoverClient 遍历) does NOT need interface introduction work; it's pure test coverage expansion using existing factory seam.

### 1.2 v1.27-pitfalls.md (R-1..R-8 miniredis, S-1..S-4 SSH fake, M-1..M-2 metrics, Q-1..Q-12 quirks)

All already shipped in Phase 75/76; only the test-time protective measures matter:

| Trap | Mitigation Status | Phase 78 Use |
|------|-------------------|--------------|
| **R-1** miniredis TTL 不自动流逝 | `mr.FastTimeout(d)` 推进 TTL | core 78-02 Init chain + Close → Cache metrics test |
| **R-2** INFO only connected_clients | Don't assert specific Redis stats, only err==nil + key exists | core 78-02 warm-up path |
| **R-3** go-redis v9.5+ CLIENT SETINFO | miniredis v2.38+ 已实现 no-op (Phase 76 added) | Already shipped, no new concern |
| **R-7** MultiLevelCache 构造即启 L2WriteWorker | `NewMultiLevelCacheSimple` (no worker) or t.Cleanup Close | 78-02 Init test — must use Simple variant OR t.Cleanup c.Cache.Close() before goroutine leak in -race |
| **R-8** miniredis ≠ 真件 | 单测过 ≠ 生产过; 保留真实 Redis 冒烟 | (Already shipped) |
| **S-2** NetworkDriver 期望 prompt 形态 | fake SSH server 必须在 pty-req+shell 后输出 `<host>#` 提示符 | 78-03 device SSH fake for Open path |
| **S-3** PasswordCallback 写法 | 拒绝时返回 (nil, false) 而非 error | 78-03 device SSH fake for auth reject scenarios |
| **S-4** ScrapliWrapper 持具体 `*network.Driver` | Phase 76 INFRA-02 已加 `newNetworkDriver` 包级 var | **DONE** (scrapli_wrapper.go:115); 78-03 directly uses var injection per `driver_factory_76_02_test.go` 先例 |
| **S-5** fixture 路径 Windows vs `/` | Use `filepath.Join` (portwrite e2e precedent) | 78-03 FileTransport fixture paths |
| **M-1** 差值采样零增量 | burnCPU 200ms×2; assert [0,100] + err==nil only | core 78-01 metrics_cache_test only covers bool paths; CPU sampling stays low risk |
| **M-2** 包级全局状态无锁 | test 禁 t.Parallel(); QUIRK-02 已加 mutex | (Already shipped) |
| **Q-1** MemoryCache.IncrementBy nil-deref | QUIRK-01 fix shipped | 78-01 captcha GenerateCaptcha 真实链路可直测 |
| **Q-7** MetricsCacheService.Stop 双调 panic | QUIRK-02 fix shipped (sync.Once idempotent) | 78-02 Core.Close 测试可重复调 Stop |

### 1.3 v1.27-architecture.md (test double strategy, build-tag decision, init probe question)

All applicable directly:

| Architecture Decision | Phase 78 Use |
|----------------------|--------------|
| 同包 `<topic>_78_xx_test.go` 共置 (Q1) | Plan filenames follow convention |
| Per-test fixture + t.Cleanup (Q2) | miniredis RunT, sqlite :memory:, LDAP server per-test |
| 无 build tag — A1 命名后缀隔离 (Q3) | ForTesting suffix + AST guard test (for_testing_guard_test.go exists) |
| QUIRK 修复五步法 (Q4) | Not applicable (Phase 75 did all quirks) |
| 单一主 module (Q5) | miniredis already in go.mod from Phase 76 |
| miniredis + iface-stub + FileTransport + re-exec covers all 5 BLOCK packages (Q6) | Phase 78 closes core + device + addomain |

**Open Question #2 (Q4) RESOLVED:** "core.Core 完整 Init 图能否用 miniredis + sqlite 拼出" — Phase 78-02 first task IS the init probe experiment per ROADMAP note "78-02 首任务为 Init 可跑深度探针实验". The probe results will populate this gap.

---

## 2. Phase 78 Specific Knowledge Gaps (NEW research beyond v1.27 files)

### Gap G1: core Init 链探针 (core.go 302 stmts)

**Verified by reading core.go L196-249 (Init) + L692-778 (Close) + L784-862 (initCache):**

**Init 编排链** (L196-249):
1. SM4Cipher setup (no side effects if nil)
2. `initDBAndData()` — glebarez sqlite config + AutoMigrate + InitData seed + permission init
4. `initCacheAndWarmUp()` — miniredis → RedisCache → MultiLevelCache OR pure MemoryCache
5. `initMetrics()` — MetricsCacheService (no goroutines)
6. `initDeviceServices()` — DeviceConnectionPool (启动 startCleanup goroutine) + DeviceExecutor + DeviceDiscoveryService + DeviceInfoCollectionService
7. `initSchedulerAndTasks()` — internal/scheduler cron 引擎 (cron 引擎启动带 goroutines)
8. `initCaptchaServices()` — CaptchaService + CaptchaBackgroundService
9. `initLogsAndAuth()` — OperLogService + TokenBlacklistService + AuthFactory
10. `initRPAAndAPIAndReaper()` — RPA service + APIEndpointService + 子进程 reaper (reaperCancel 注入)
11. goroutine: refreshView 30s 超时 (Init return 后启动)

**Close 编排链** (L692-778) 收尾顺序（critical: 必须与 Init 顺序对应）:
1. reaperCancel (RPA reaper 停)
2. NoticeHub.Stop()
3. Scheduler.Stop() + scheduler.StopADSyncScheduler()
4. DeviceInfoCollectionService.Stop()
5. DeviceMonitorService.Close()
6. MetricsCacheService.Stop() (QUIRK-02 幂等修复)
7. RPAScalingService.Stop()
8. Cache.Close() (MultiLevelCache 走 L2Writer Stop 兜底)
9. operlog 100ms heuristic sleep (无 WaitGroup flush 暴露)
10. DB.Close()

**initCache 分支** (L784-862):
- `c.Config.Cache.Type == "redis"` → cache.NewRedisCache → MultiLevelCache (R-7 worker goroutine 启动)
- 否则 → `cache.NewMemoryCache(MaxSize, CleanupTime)` → **纯内存缓存，无 L2Worker**

**Probe Task 78-02 首任务**:
- 写一个 `core_init_probe_test.go`: 用 `glebarez/sqlite` + miniredis config, 调 `core.Init()`, 验证每步产物非 nil, 记录启动耗时; 然后 `core.Close()`, 验证退出无 panic. **预期 HIGH 风险**: scheduler cron 引擎 + device pool cleanup goroutine + RPA reaper + 缓存预热 goroutine + refreshView goroutine 一起跑时, Close 路径可能 hang (历史教训: 见 core.go:725-732 注释 "原顺序在 DeviceInfoCollectionService.Stop() 之后,前 5-15s 仍在 spawn ... WaitGroup 死等").

### Gap G2: device scrapli_wrapper ForTesting 工厂设计

**Verified by reading scrapli_wrapper.go L54-131 + e2e_helpers.go L32-60 + driver_factory_76_02_test.go L60-115:**

**INFRA-02 var-seam 已落地** (scrapli_wrapper.go:115):
```go
var newNetworkDriver = func(platformName interface{}, host string, opts ...util.Option) (*network.Driver, error) {
    p, err := platform.NewPlatform(...)
    d, err := p.GetNetworkDriver()
    return d, nil
}
```

**e2e_helpers.go 既有工厂** (L32-60):
- `NewPooledConnectionForTesting(d *network.Driver) *PooledConnection` — 跨包注入
- `newScrapliWrapperForTesting(d *network.Driver) *ScrapliWrapper` — 同包私有工厂, state=StateReady 让 acquireOp 通过

**Phase 76-02 已实证**: driver_factory_76_02_test.go 展示了 `newNetworkDriver = func(... FileTransport fixture ...)` 完整驱动 `NewScrapliWrapper → Open → SendCommand` 全链 fixture 回放. **这一先例是 78-03 整个 device ≥70% 的钥匙.**

**Phase 78-03 额外表面**:
- `executor.go:124 stmts` — ExecuteMultipleOnDevice (L101), ExecuteCustom (L161), executeWithRetry (L201), GetConfig (L267). 所有路径需要 `*DeviceTaskScheduler` (task_scheduler.go L33-43) + `*PooledConnection`. e2e_helpers 的 NewPooledConnectionForTesting 可直接驱动 task_scheduler 的 Submit + executeTask; **关键点**: task_scheduler L205 `s.connectionPool.GetConnection(ctx, task.DeviceID)` — GetConnection 内部 `createConnection(ctx, deviceID)` 走真实 SSH 链路. 需要绕开: **解决方案** 是直接用 task.Execute 的 conn 参数（task_scheduler.go:238 `execErr = task.Execute(ctx, conn)`），不通过 Submit; task.Execute 接受 `*PooledConnection`, 我们用 `NewPooledConnectionForTesting(FileTransport driver)` 直接喂入.
- `connection_pool.go:144 stmts` — Release(L82,0%)/Execute(L111,18.2%)/GetConnection(L236)/createConnection(L393)/removeConnection. **关键**: Release 不需 SSH, 可直测; Execute 走 wrapper 操作, FileTransport 驱动 wrapper 即可; createConnection 走 DB + SSH, 需要 e2e_helpers 拼装 + 跳过或者注 DB row.

**FileTransport 100% works for non-prompt commands** (verified by driver_factory_76_02_test.go:99-113 SendCommand("display version") returns Huawei fixture result). **WaitForReady prompt-style commands** (scrapli_wrapper.go L356-397 GetPrompt 在 ticker 中轮询) 需要 fixture 包含 pty-req+shell 后的 `<host>#` prompt — portwrite 已用 `huawei_vrp_open.fixture` 验证此先例.

**fake SSH (x/crypto/ssh) 需求边界**: 仅在 Open 路径需要 wire-level 模拟 (即网络交互 + pty-req+shell 协议) 时才用. FileTransport 跨过这些协议层, 所以 `Open` 也可由 FileTransport fixture 驱动 (portwrite 先例证实). **建议**: 78-03 优先 FileTransport 全链, fake SSH 仅在 FileTransport 不能覆盖的少数分支 (如 privilege level 升降提示符) 按需扩展.

### Gap G3: device snmp_client UDP peer

**Verified by reading snmp_client.go L18-287 + snmp_client_test.go (full file) + Grep `PacketConn|fakeConn|udpPeer` across `internal/`:**

**Current state**: `snmp_client.go` 包裹具体 `*gosnmp.GoSNMP` (snmp_client.go:20-21), 所有方法 (Connect/Get/GetNext/Walk/GetBulk) 都通过 `c.client.Connect()` + `c.client.Get()` 等具体 API 触发真实 UDP 发送. 

**No UDP peer precedent in codebase** (grep returned "No matches found" for `PacketConn|fakeConn|udpPeer` across `internal/`). This is GREENFIELD territory.

**Two implementation paths:**

| Path | Pros | Cons | Recommendation |
|------|------|------|----------------|
| **A. fakePacketConn 30-line impl** implementing `net.PacketConn` interface, listening on `127.0.0.1:0` UDP, replying canned BER-encoded SNMP responses for requested OIDs | Zero new deps; targeted; tests specific OIDs (sysDescr etc.) | Must handle BER encode/decode for v2c GET/GETNEXT/GETBULK responses (depends on `asn1-ber` package which IS in the dep tree via go-ldap transitive — VERIFY via go.mod) | **RECOMMENDED** for primary 78-04 plan |
| **B. gosnmp 自身 testserver** (gosnmp repo's `helpers.go` test server) | Battle-tested by gosnmp maintainers | Would require copying 200+ lines from gosnmp repo into test/ folder; no precedent in our codebase | AVOID unless Path A infeasible |

**Required UDP peer surface** (which GoSNMP calls into PacketConn):
- `WriteTo(p []byte, addr net.Addr) (n int, err error)` — gosnmp sends BER request
- `ReadFrom(p []byte) (n int, addr net.Addr, err error)` — fake server replies
- `Close() error`
- `LocalAddr() net.Addr`
- `SetDeadline(t time.Time) error / SetReadDeadline(t time.Time) error / SetWriteDeadline(t time.Time) error`

**BER encode for canned SNMPv2c GET response** is ~50 lines minimum:
```
SEQUENCE {
  INTEGER 0x30 (request-id echo)
  INTEGER 0 (error)
  INTEGER 0 (error index)
  SEQUENCE OF VarBind {
    SEQUENCE {
      OID <requested-oid>
      OCTET STRING "test value"
    }
  }
}
```

`encoding-asn1` from stdlib can encode this; alternatively the `github.com/go-asn1-ber/asn1-ber` (go-ldap transitive dep) has `Marshal` helpers — VERIFY in go.mod.

**Test scope**: 78-04 covers Get(oidSysDescr)/GetNext/Walk(partial tree)/GetBulk + Connect (to localhost UDP) + Close. About 237 unc stmts in snmp_client.go, achievable with ~6-8 tests.

### Gap G4: addomain FailoverClient concrete-type dispatch

**ALREADY RESOLVED** by Gap 1.1 correction. Reading failover_client.go:44 + failover_client_76_03_test.go (full file) confirms:
- `ExecuteWithFailover(ctx, operation func(client LDAPClientIface) error)` — **interface**, not concrete
- `clientFactory func(*models.ADConfig, *models.ADServiceAccount) LDAPClientIface` — injection seam exists, used by tests
- mockLDAPClient (ldap_client_mock_test.go) implements LDAPClientIface fully (per the `var _ LDAPClientIface = (*LDAPClient)(nil)` compile-time assertion at ldap_iface.go:56)

**Phase 78-07 surface** that needs new test coverage:
- `user.go:108 stmts` — GetList/GetByID/Create/Update/Delete + 4 failover-wrapped ops (UpdateUserAttribute L154, EnableUser L199, DisableUser L220, MoveUser L241)
- `group.go:78 stmts` — GetList/GetMembers + 3 failover-wrapped ops (AddGroupMember L129, RemoveGroupMember L159, UpdateGroupAttribute L200)
- `failover_client.go` already has 76-03 test; phase 78 may add edge cases (empty pool, single account, mid-loop recovery)

**No new interface introduction required.** Phase 78-07 is pure coverage expansion using existing seams.

### Gap G5: MultiLevelCache race-safety in test

**Verified by reading pkg/cache/redis.go L547-555:**

```go
// NewMultiLevelCacheSimple 创建多级缓存（不使用L2Writer，保持原有行为）
func NewMultiLevelCacheSimple(l1, l2 Cache) *MultiLevelCache {
    return &MultiLevelCache{
        l1Cache:         l1,
        l2Cache:         l2,
        l2WriterEnabled: false,
        retryEnabled:    false,
    }
}
```

**Confirmed**:
- `NewMultiLevelCacheSimple` exists in current pinned version (no v0.38+ dependency; it's in our codebase)
- Import path: `github.com/xingran-next/xingran-go-backend/pkg/cache`
- Signature: `func(l1, l2 Cache) *MultiLevelCache` — no L2Worker started (R-7 mitigation built-in)
- Safe for -race tests with `t.Cleanup(c.Cache.Close())` still recommended (defensive)

**78-02 use case**: core Init probe test must either (a) `NewMultiLevelCacheSimple(mem, redis)` then `t.Cleanup(c.Cache.Close())`, OR (b) `NewMultiLevelCacheWithWriter` + assert L2Writer Stop in Close path. **Recommend (a) for probe simplicity**, then write separate test that goes through `NewRedisCache + NewMultiLevelCacheWithWriter` to validate the production path Close also works.

---

## 3. Test Double Strategy Per Package

| Package | Target File | Stmts Gap | Technique | Prerequisite | Risk |
|---------|------------|-----------|-----------|--------------|------|
| **core** | core.go | 302 | miniredis RunT + glebarez sqlite + Config.Cache.Type="memory" skip | INFRA-01 done; Q-7 fix done | **HIGH**: scheduler cron 引擎 + device pool cleanup goroutine + reaper goroutine 同时跑时 Close 可能 hang (核心 Init probe 任务验证) |
| **core** | core.go initCache redis 分支 | ~50 | miniredis + NewRedisCache + NewMultiLevelCacheSimple (or WithWriter) + t.Cleanup Close | INFRA-01 done | MEDIUM: R-7 L2Worker goroutine 泄漏 if not Closed |
| **core** | captcha.go | 98 | sqlite + MemoryCache + QUIRK-01 fix allows 直测 GenerateCaptcha 真实链路 | Q-1 fix done | LOW: existing captcha_test.go 在 core_74_08_test.go 已覆盖主体 |
| **core** | captcha_background.go | 48 | 文件 fixture + sqlite | none | LOW: pure I/O |
| **core** | metrics_cache.go | 2 | (a) class 纯补 | none | LOW |
| **device** | scrapli_wrapper.go | 222 | INFRA-02 var-seam `newNetworkDriver` + FileTransport fixture (portwrite 先例 + driver_factory_76_02_test.go 先例) | INFRA-02 done | LOW: 先例已实证 |
| **device** | executor.go | 124 | e2e_helpers.NewPooledConnectionForTesting(FileTransport driver) + 跳过 task_scheduler.Submit, 直接调 task.Execute(conn) | INFRA-02 done | MEDIUM: task.Execute 私有调用需 internal/device 包内 test (driver_factory_76_02_test.go 同包先例已证) |
| **device** | connection_pool.go | 144 | Release/Execute 直测 (FileTransport wrapper); createConnection 走 DB row seed + 跳过 SSH (或 stub createConnection) | INFRA-02 done | MEDIUM: createConnection 强耦合 SSH; 可能需小重构暴露 helper 或在测试中用 e2e_helpers 替代 NewScrapliWrapper 调用点 |
| **device** | snmp_client.go | 237 | **fakePacketConn** (净 30 行) + BER encode canned responses (50 行) for GET/GETNEXT/GETBULK; 127.0.0.1 UDP | NONE (greenfield; verify asn1-ber available via go-ldap transitive) | **HIGH**: BER encoding + 多 method 覆盖需要 8-10 个测试, ~200 行 test fixture; net 80 行 test helper + 150 行 test |
| **device** | task_scheduler.go | 25 | 用 NewPooledConnectionForTesting 喂 task.Execute; t.Cleanup s.Stop() 防 -race 泄漏 | INFRA-02 done | LOW |
| **addomain** | sync.go | 316 | sqlite + 模拟 `[]*ldap.Entry` 数据驱动 + FailoverClient.clientFactory 注入 mockLDAPClient | INFRA-03 done + 76-03 test 先例 | LOW: `[]*ldap.Entry` 数据驱动边界是 v1.27-features §1.2 seam #5 |
| **addomain** | computer.go | 219 | sqlite + ldap.NewEntry 字面量 + ComputerService 直构造 | INFRA-03 done | LOW |
| **addomain** | ldap_client.go | 159 | 参数组装 / 错误分支 (Pure Go, no wire) + 拨号错误路径 | INFRA-03 done | MEDIUM: Connect/dialConnection/tryBindAttempts 走真实 dialURL; mock LDAP server 仅在 70% 缺口时启用 |
| **addomain** | group_sync_service.go | 132 | mockLDAPClient + iface stub 驱动 | INFRA-03 done | LOW |
| **addomain** | user.go | 108 | failover 包裹的方法 mockLDAPClient 驱动 + 非包裹方法 (GetList 走 cache) | INFRA-03 done | LOW |
| **addomain** | group.go | 78 | 同 user.go | INFRA-03 done | LOW |
| **addomain** | account_pool.go | 72 | sqlite + MarkSuccess/RecoverExpiredBreakers 直测 | none | LOW: account_pool_test.go 已覆盖主体 |
| **addomain** | user_ou_service.go | 69 | sqlite + 部分 iface stub | INFRA-03 done | LOW |
| **addomain** | dept_sync_service.go | 61 | mockLDAPClient + CreateOU 走 iface | INFRA-03 done | LOW |
| **addomain** | group_config_service.go | 60 | 纯 sqlite CRUD | none | LOW |
| **addomain** | config.go | 53 | sqlite CRUD + TestConnection 走 mock LDAP | INFRA-03 done | LOW |

**Total covered**: ~2400 stmts across 3 packages. Combined 75 / 76 / 77 / 78 will retire 3 P2_RATCHET exemptions.

---

## 4. Risk Register

### HIGH Risks

| ID | Risk | Mitigation |
|----|------|------------|
| R1 | **core Init + Close 路径可能 hang** —— scheduler cron 引擎 + device pool cleanup goroutine + RPA reaper + 缓存预热 + refreshView goroutine 一起跑时 Close 顺序错位可能死锁 (历史 core.go:725-732 注释明确记录) | 78-02 first task 是 **init probe 探针实验** (ROADMAP note 明示). 先写 probe test, 发现 hang 即修复 Close 顺序 + 加 t.Cleanup 防御. 不许绕过探针直接写覆盖测试 |
| R3 | **snmp UDP fake 必须实现 BER 编码** —— SNMP 协议响应是 ASN.1 BER 编码, 30 行 PacketConn + 50 行 BER helper 可能低估. 若 fake 不能覆盖 GetBulk 多响应路径, ~80 stmts 失守 | 优先覆盖 Get/GetNext 单响应 + Connect/Close. 若 GetBulk/Walk 测试受阻, fallback: 仅覆盖 ~120 stmts (≥40% 提升), 留作 Phase 79 长尾. 切勿硬上 multi-PDU BER 编码 |
| R7 | **device createConnection 强耦合 SSH** —— connection_pool.go L393 createConnection 直接调 NewScrapliWrapper 真实 SSH, 无法用 FileTransport 替代 | 78-03 plan 拆分: connection_pool.Release/Execute 走 FileTransport (covered); createConnection 接受 SSH 真连的"已知不可测"小重构 - 不强求 100%. 或拆 `p.passwordCipher.Decrypt()` 路径部分覆盖 |

### MEDIUM Risks

| ID | Risk | Mitigation |
|----|------|------------|
| R2 | **scrapli_wrapper OpenContext 30s 超时 + goroutine 关闭竞争** —— e2e_helpers 的 newScrapliWrapperForTesting 强制 state=StateReady, 但生产代码走 OpenContext 内部 ticker 循环 GetPrompt 验证 (scrapli_wrapper.go:356-397). 测 FileTransport 时此 ticker 会被 fixture 喂到. 但若 fixture 不含 prompt, OpenContext 会 hang 30s | portwrite 先例 `huawei_vrp_open.fixture` 含 `<dummy-host>` prompt, 验证过. 78-03 必须照搬同一 fixture pattern |
| R4 | **device executor.go 测试需绕过 task_scheduler.Submit** —— 正常路径 Submit → executeTask → task.Execute(conn); 但 Submit 内部 connectionPool.GetConnection 走 createConnection. 解决: 直接构造 task.Execute 闭包喂入 NewPooledConnectionForTesting 绕开 Submit. task.Execute 是 Execute 字段 (task_scheduler.go:25), 但 executor.go:61 喂的是构造好的 *PooledConnection | 同包测试可直构造. driver_factory_76_02_test.go 同包先例已证. 风险: task.Execute 是私有字段, 需 internal/device 包内 test |
| R5 | **addomain ldap_client.go Connect 真拨号** —— Connect() (ldap_client.go:75) 调 dialConnection 真连 AD server. 测试需 Connect error 分支 (refuse / DNS / timeout) + 参数组装分支. wire-level 需 mock LDAP server, INFRA-03 stub 走的是 mockLDAPClient 接口注入. 真实 *LDAPClient 实例无法 mock | 78-07 plan: Connect 失败路径 (dialConnection 三分支 UseSSL/UseTLS/default) 测参数 + 错误 wrapper; 不测真 bind 成功路径. 真 bind 留作 optional Phase 79 长尾 |
| R6 | **R-7 L2Writer goroutine 泄漏** —— core Init probe 用 NewMultiLevelCacheWithWriter 时, Close 路径需正确停止 worker. 若 t.Cleanup 顺序错 (先 c.DB.Close() 再 c.Cache.Close()), worker 持有死 DB conn 句柄 | 78-02 Init probe test 必须显式 assert "Close 后无 goroutine 泄漏" (用 goleak 或 -race 静态验证) |
| R8 | **FailoverClient production 路径仍走 NewLDAPClient** —— failover_client.go:38 `NewLDAPClient(f.config, acct)` 是生产默认, clientFactory nil 时走此分支. 测生产路径需真连 LDAP. Phase 78-07 plan 限制: 仅测 factory != nil 路径 (mock 注入) | INFRA-03 边界已在 76-03 测试覆盖. Phase 78-07 不强求生产路径测试 |

### LOW Risks

| ID | Risk | Mitigation |
|----|------|------------|
| R9 | Windows fixture 换行符差异 | filepath.Join + TrimSpace 比对 (先例) |
| R10 | QUIRK-02 翻转断言遗漏 | 已 Phase 75 处理, 与本 phase 无交叉 |
| R11 | Windows dev 无 Docker | 本 phase 不引入 Docker 依赖 |
| R12 | -race 仅 CI 跑 | 本地至少跑一次 `go test -race ./internal/core/... ./internal/device/... ./internal/services/addomain/...` |

---

## 5. Decision Queue for Discuss-Phase

The planner should raise these with the user before /gsd:plan-phase creates PLAN.md files:

### DQ1. MultiLevelCache import path confirmation
- **Context**: R-7 防护 + core Init probe 路径
- **Known**: `NewMultiLevelCacheSimple(l1, l2 Cache) *MultiLevelCache` exists at pkg/cache/redis.go:547-555. No external dependency required.
- **Question**: Should 78-02 plan (a) always use NewMultiLevelCacheSimple in probe tests (defensive), or (b) use the production NewMultiLevelCacheWithWriter path with explicit goroutine leak assertion?
- **Recommendation**: (a) for the probe (minimize surface), (b) as a separate validation test. Both included.

### DQ2. snmp fake scope decision
- **Context**: Gap G3, R3
- **Known**: fakePacketConn ~30 行 + BER encode ~50 行 = 80 行 helper. Covers ~120-160 stmts in snmp_client.go (Get/GetNext/GetBulk + Connect/Close). Walk 全树响应序列编码更复杂, 估需 +50 行.
- **Question**: Aim for snmp_client.go ≥70% (需 ~170 stmts of 237 = ~72%), or settle for current ~40% + Get/GetNext/GetBulk/Connect/Close + leave Walk + GetBulk P2 stub to Phase 79?
- **Recommendation**: Aim for ≥70% with full coverage. Walk fn 在 cmd 设备发现路径, 不是 P0 高频路径; 但 GetBulk 在 v2c 信息采集中用得多. Planner 决定 priority.

### DQ3. core Init Close ordering vs goroutine leak
- **Context**: R1, R6, G1
- **Known**: Close 编排顺序在 core.go:692-778 已经按 "scheduler → 设备采集 → 设备监控 → 缓存 → DB" 排序, 历史修过 hang bug. 但 init probe 测试可能暴露 scheduler cron 引擎启动带 goroutine 在 Close 后仍持引用.
- **Question**: 若 probe test 发现 hang, 是 (a) 修 Close 顺序 (小重构, 风险扩散), 还是 (b) 给 scheduler 加 ctx-aware Stop 方法 (中等重构), 还是 (c) 在 test 里用 goleak.ExpectedGoroutines 容忍已知 worker goroutine (最廉价)?
- **Recommendation**: 先 (c) 验证问题范围; 若 (c) 失败, 上 (a); (b) 留 Phase 79 长尾.

### DQ4. addomain Iface stub 段是否需要 mock LDAP server
- **Context**: R5, G3 of v1.27-stack
- **Known**: 主推线 = mockLDAPClient iface stub (零新依赖). 备用线 = vjeantet/ldapserver (进程内 wire, MEDIUM 维护风险). v1.27-stack §2 + v1.27-architecture 开放问题 1.
- **Question**: 是否引入 vjeantet/ldapserver 测真实 *LDAPClient Connect + dialConnection + tryBindAttempts 三分支? 或仅测 iface mock 层 (~70% 缺口是否够)?
- **Recommendation**: 暂不引入. iface mock + 参数组装测试可到 ~70%. 若 78-07 跑完仍 <70%, 在 78-07 plan 内追加 mini-ldapserver helper (~150 行).

### DQ5. device createConnection 测试边界
- **Context**: R7, Gap G2 §executor.go
- **Known**: createConnection 强耦合 NewScrapliWrapper 真实 SSH. e2e_helpers.NewPooledConnectionForTesting 提供 PooledConnection 但不替代 createConnection 函数体.
- **Question**: 78-03 是 (a) 仅测 Release/Execute/removeConnection (避开 createConnection), 还是 (b) 给 createConnection 加 transport option seam (类似 INFRA-02)?
- **Recommendation**: (a) for 78-03 (避免扩大 INFRA scope); (b) 留 Phase 79 长尾. connection_pool.go 144 unc 中约 80 stmts 可通过 (a) 覆盖 (Release + Execute + removeConnection + 部分 GetConnection 复用路径).

---

## 6. Source Documents

### Primary (HIGH confidence — verified by reading files in this session)

- `internal/core/core.go` L196-249 (Init), L338-396 (initCacheAndWarmUp), L692-778 (Close), L784-862 (initCache), L460-580 (initSchedulerAndTasks — Init probe must verify this)
- `internal/core/captcha.go` (98 stmts gap, mostly c-class)
- `internal/core/captcha_background.go` (48 stmts, c-class)
- `internal/device/scrapli_wrapper.go` L115-131 (newNetworkDriver var-seam), L297-301 (GetPrompt), L303-320 (Open), L323-403 (OpenContext)
- `internal/device/e2e_helpers.go` L32-60 (ForTesting factory)
- `internal/device/driver_factory_76_02_test.go` L60-115 (FileTransport var injection 先例)
- `internal/device/executor.go` (124 stmts unc: ExecuteMultipleOnDevice/ExecuteCustom/executeWithRetry/GetConfig)
- `internal/device/task_scheduler.go` (Submit + executeTask pattern)
- `internal/device/connection_pool.go` L82-156 (Release/Execute), L393+ (createConnection)
- `internal/device/snmp_client.go` L18-287 (Connect/Get/GetNext/Walk/GetBulk — all use *gosnmp.GoSNMP concrete)
- `internal/device/snmp_client_test.go` (full — only WaitForReady covered, 135 lines)
- `internal/services/addomain/failover_client.go` L1-131 (ExecuteWithFailover already takes LDAPClientIface — Phase 76-03 fix)
- `internal/services/addomain/ldap_iface.go` L18-56 (LDAPClientIface 19 methods + compile-time assert)
- `internal/services/addomain/failover_client_76_03_test.go` L29-156 (sequential traversal + maxHops + searchUsersFn multi-batch)
- `internal/services/addomain/account_pool.go` L46-106 (AccountPool interface)
- `internal/services/addomain/ldap_client.go` L45-125 (LDAPClient struct + Connect + dialConnection 三分支)
- `pkg/cache/redis.go` L547-555 (NewMultiLevelCacheSimple — no L2Worker, race-safe)

### Secondary (MEDIUM confidence — derived from prior research files)

- `.planning/research/v1.27-features.md` §1.1 (addomain unc table), §1.2 (seam catalog), §3.1 (device unc table), §3.2 (seam catalog), §4.1 (core unc table), §4.2 (seam catalog), §6 (priority ranking)
- `.planning/research/v1.27-pitfalls.md` R-1..R-7 (miniredis), S-1..S-5 (SSH fake), M-1..M-2 (metrics), Q-1, Q-7 (QUIRK fix context)
- `.planning/research/v1.27-architecture.md` Q1-Q6 (test placement, fixture lifecycle, ForTesting vs build-tag, go.mod strategy, CI strategy)
- `.planning/research/v1.27-stack.md` §1 (Redis), §2 (LDAP), §3 (SSH/transport)
- `.planning/workstreams/milestone/ROADMAP.md` L183-216 (Phase 78 goal + plan list)
- `.planning/workstreams/milestone/REQUIREMENTS.md` L28-34 (BLOCK-03/04/05 requirements)
- `.planning/workstreams/milestone/STATE.md` (Phase 75/76/77 progress + completed context)

### Tertiary (LOW confidence — gaps requiring Phase 78 execution)

- snmp_client.go Walk / GetBulk wire-level fake testability (need probe to validate)
- core.Init Close 完整 goroutine 拓扑 (需要 78-02 探针实验)
- device executor.go 绕开 Submit 模式的可行性 (待 78-03 实证)

---

## 7. Confidence Breakdown

| Area | Level | Reason |
|------|-------|--------|
| Standard Stack (miniredis/FileTransport/mockLDAPClient) | **HIGH** | All INFRA-01/02/03 already shipped in Phase 76; verified by reading pkg/cache/redis.go, e2e_helpers.go, ldap_iface.go, failover_client.go |
| Architecture (test double strategy, fixture lifecycle) | HIGH | All patterns in portwrite e2e + driver_factory_76_02_test.go + failover_client_76_03_test.go 已实证 |
| Pitfalls (R-1..R-8, S-1..S-5) | HIGH | Phase 75/76 QUIRK 修复 + INFRA 已落地, test-time 防护仍生效 |
| FailoverClient seam status | **HIGH (corrected)** | Previous research §1.2 stale; verified direct read of failover_client.go:44 confirms interface signature |
| MultiLevelCache race safety | HIGH | NewMultiLevelCacheSimple 存在并已读源码确认 |
| snmp fakePacketConn feasibility | **MEDIUM** | 路径明确但 BER encoding 复杂度未实测; 80 行 helper 估算是 lower bound |
| core Init probe outcome | **LOW** | 这是探针实验的原因, 不是研究能回答的; 必须在 78-02 first task 验证 |

**Research date:** 2026-08-27
**Valid until:** 7 days (Phase 78 立即执行, 资料时效要求高)

---

## Metadata

**Phase:** 78 — 阻塞包攻破·基建解锁 (core + device + addomain)
**Milestone:** v1.27 后端测试覆盖率优秀 II
**Authored by:** gsd-phase-researcher agent (spawned by `/gsd:plan-phase 78`)
**Out of scope (deferred to Phase 79/80):**
- 4 个 P2_RATCHET 豁免行的实际删除 (Phase 81 收口)
- TAIL-01 (Phase 79) / TAIL-02 (Phase 80) / TAIL-03 (Phase 80) 长尾包
- scheduler 引擎深度并发分支 (Phase 80)
- GATE-01/02/03 milestone audit (Phase 81)