---
phase: 73-p1-pending
plan: 04
subsystem: services (monitor)
tags: [IMP-06, service-tests, coverage, portwrite-pure-mock, ad_account-sqlite, testify, glebarez]
dependency_graph:
  requires:
    - phase: 73-p1-pending
      provides: "73-03 portwrite pure-mock 范本落地经验(mockCacheProvider + sqlite 最小依赖)"
    - phase: 72-p0-core-supplement
      provides: "api/v1/monitor handler 测试中的 sys_oper_log / sys_logininfor DDL(本计划复用)"
  provides:
    - "internal/services/monitor test suite (95.3% coverage, 143 test funcs)"
    - "IMP-06 monitor 半区交付(monitor 0%→95.3%),IMP-06 整体完成"
  affects:
    - "internal/services/monitor (no business code changes per D-12)"
tech-stack:
  added: []
  patterns:
    - "mockMonitorCacheProvider/mockMonitorFullCacheProvider (testify/mock) — 单个 mock 通过嵌入实现 CacheProvider + MultiLevel/DirectRedis/Stats 可选接口,验证 NewCacheService 类型断言装配"
    - "mockMetricsProvider (testify/mock) 覆盖 MetricsProvider 缓存命中/未命中降级;降级路径走 internal/pkg/system 真实系统指标"
    - "glebarez sqlite :memory: + 手写 DDL(ad_account 范本);login_log Clean 的 quirk 用 stub 表覆盖 happy 分支、无 stub 表锁定 error 分支"
    - "testify Return 的 time.Duration 参数必须显式类型化(Return(-1, nil) 存成 int → 断言 panic)"
key-files:
  created:
    - internal/services/monitor/cache_service_test.go
    - internal/services/monitor/login_log_service_test.go
    - internal/services/monitor/oper_log_service_test.go
    - internal/services/monitor/server_service_test.go
  modified:
    - .planning/REQUIREMENTS.md (IMP-06 marked complete)
key-decisions:
  - "D-02 混合策略落地: cache_service 纯 mock;login/oper/server 真实 DB —— 但 cache_service 的 DB 依赖路径(getCacheListFromDB/历史统计/配置持久化)仍需最小 sqlite,portwrite 范本本身同样如此"
  - "DDL 以 models TableName() 为准: LoginLog 表名是 sys_logininfor(计划写的 sys_login_log 是错的);按计划'Reuse Phase 72 DDL if available'条款复用 72 的正确 DDL,sys_login_log 仅作为 Clean quirk 的 stub 表出现"
  - "实时降级路径(getCurrentServerInfo/getCurrentMetricsRealtime)在测试机上真实执行并断言真实值(runtime.GOOS/GOARCH/HostName),~7s/次的 system.GetSystemMetrics 成本可接受"
requirements-completed: [IMP-06]
metrics:
  duration: "~18 min"
  completed: 2026-08-21
---

# Phase 73 Plan 04: services 中等 monitor (485 stmts) 0% -> 95.3%

## One-liner

Added 143 test functions across 4 files using the D-02 mixed strategy (cache_service via portwrite pure-mock with CacheProvider/CacheConfigProvider mocks; login_log/oper_log/server_service via glebarez sqlite + real services), taking internal/services/monitor from 0.0% to **95.3%** and completing IMP-06 (with oper_log_service covered per D-03 — not exempted).

## Coverage

| Metric | Plan baseline | Pre | Achieved | Target | Delta |
|--------|--------------:|----:|---------:|-------:|------:|
| `internal/services/monitor` (485 stmts) | 485 | 0.0% | **95.3%** | ≥70% | +95.3pp |

Per-file function coverage (all functions ≥66.7%, only sub-70 function is `normalizeCacheKeyForService` whose uncovered line is dead code per quirk Q1):

| File | Functions | Range | Notable |
|------|-----------|-------|---------|
| cache_service.go | 21 | 66.7%–100% | OperateCache/BatchOperateCache/ClearCache/GetCacheStats/GetCacheMonitor/ExportCache/configs 全 100% |
| login_log_service.go | 7 | 75.0%–100% | List 95.5%, Clean 100%(含 quirk 双分支) |
| oper_log_service.go (D-03) | 7 | 75.0%–100% | List 95.8%(含 Q4 broken-filter error 锁定) |
| server_service.go | 8 | 75.0%–100% | convertToServerInfo 100%, 两个 realtime 降级 75%(剩余为不可注入的 system 错误分支) |
| types.go | 0 stmts | — | 纯 struct 定义,无语句 |

Verify command (exit 0):

```
go test -coverprofile=c.out -count=1 ./internal/services/monitor/
# ok  ...  coverage: 95.3% of statements
go tool cover -func=c.out | grep total   # total: (statements) 95.3%
```

## Performance

- **Duration:** ~18 min (04:07:58Z → 04:25:48Z)
- **Tasks:** 5/5
- **Files created:** 4 test files (+1 doc, +REQUIREMENTS.md row update)
- **Test functions:** 143 total (cache 78, login 23, oper 24, server 18)

## Task Commits

| Commit | Type | Content |
|--------|------|---------|
| `870480f` | test | cache_service_test.go — portwrite 纯 mock,78 funcs |
| `f922505` | test | login_log_service_test.go — ad_account,23 funcs |
| `39d8701` | test | oper_log_service_test.go — D-03 不豁免,24 funcs |
| `b19fe05` | test | server_service_test.go — 混合 DB+mock,18 funcs |
| (this commit) | docs | 73-04-SUMMARY.md + REQUIREMENTS.md IMP-06 |

## Test Coverage Map

### cache_service_test.go (78 funcs — portwrite 纯 mock)
- 构造器: NewCacheService 可选接口装配验证(full provider → 3 可选字段非 nil;基础 provider → 全 nil)+ CompileOnly
- GetCacheList: nil-provider DB 路径(空表/过滤+白名单排序/非白名单回退/DB error);simple cache(Keys 成功/错误吞掉/Get 错误跳过/pattern 透传);multi-level(l1/l2-direct-redis/direct 失败回退/direct 空回退/all-both/all-l1miss/KeysByLevel 错误吞掉/系统键+空值跳过);分页(1/2/越界页)
- GetCacheInfo: 空键/provider 错误/命中 TTL 正负/前缀恒等 quirk/空值回退 DB 命中与未命中/nil-provider not-found
- OperateCache: 6 操作 × (成功+错误+校验) 共 13 用例(get/set 默认与显式 TTL/del/exists/expire TTL 校验/ttl/unsupported)
- BatchOperateCache: nil provider/空 keys/unsupported/get 混合/del 混合(deleted+failed map)
- ClearCache: nil/FlushDB 错误/成功(sys_cache_info 同步清空)
- GetCacheStats: 实时(无 StatsProvider/默认实时判定/GetStats 错误/成功 l1+l2+畸形跳过);历史(cache_type 过滤/时间窗/分页 DESC/DB error)
- GetCacheMonitor: 无 provider/错误/L1-only/L2 版本+uptime/空 stats
- ExportCache: 无过滤/有过滤/DB error
- 配置: GetCacheConfigs(nil/成功)、UpdateCacheConfig(nil/非法键/越界/不存在则建/存在则更/Reload 错误透传)、ReloadCacheConfigs(nil/成功/错误)
- 辅助函数直测: normalizeCacheKeyForService(恒等 quirk)/isSystemKeyForService(16 用例)/formatCacheStatsForService(3 分支)/convertToCacheStats

### login_log_service_test.go (23 funcs — ad_account 真实 DB)
- List: 空/全量默认 DESC/用户名与 IP 模糊/状态/时间窗(含空串不过滤)/分页/loginTime ASC/nickname DESC/非法排序列(无注入,无默认排序 quirk)/Count error
- GetByID/Delete: 命中+未命中(Delete 验证行消失)
- BatchDelete: 成功/空 ids/不存在 no-op/DB error
- Clean: quirk Q3 双分支(stub 表 happy + 真实表数据不动锁定;无 stub 表 error = 生产行为)
- DefaultLoginLogListParams

### oper_log_service_test.go (24 funcs — D-03 不豁免, ad_account 真实 DB)
- 文件头 D-03 LOCK 注释: handler 71.2% 不替代 service 覆盖,禁止以"handler 已测"删除本文件
- List: 空/全量默认 DESC/标题模糊/业务类型/状态/时间窗/分页/costTime DESC/operName ASC/非法排序无注入/Count error
- Q4 quirk lock: OperName 过滤列名 operator_name 不存在(真实列 oper_name)→ List 报"查询操作日志总数失败",测试断言 error(生产同样报错)
- GetByID/Delete/BatchDelete/Clean 全分支(Clean 表名匹配 → 真正清空,与 login_log Q3 对照)

### server_service_test.go (18 funcs — 混合 DB + MetricsProvider mock)
- GetServerInfo: provider 全字段 map(含 uint64 换算)/错误类型跳过/空 map/(nil,nil) 也降级/provider 错误降级 —— 后三者断言真实系统值(os.Hostname/runtime.GOOS/GOARCH/CPUCount≥1)
- GetCurrentServerMetrics: provider 命中(Equal 整结构)/错误降级(真实 TotalMemory>0)
- SaveSystemMetrics: 成功(BaseModel uuid 钩子)/DB error
- GetSystemMetricsHistory: 空/server_id 过滤+全量/时间窗/分页默认 timestamp DESC/cpuUsage ASC/显式 timestamp DESC/非法排序无注入/DB error

## D-02 Verification (mixed strategy)

- [x] cache_service 走 portwrite 纯 mock: 顶部 compile-time assertion(`var _ CacheProvider = (*mockMonitorCacheProvider)(nil)` 等 5 个)+ testify/mock + 真实 cacheServiceImpl
- [x] login_log/oper_log/server_service 走 ad_account: glebarez sqlite :memory: + 手写 DDL + 真实 New*Service + 真实方法调用,DB 状态变迁直接断言
- [x] server_service 混合: DB 方法真实 SQL + MetricsProvider testify mock(降级路径锁定)
- [x] 无新 mock 框架(仅 testify)
- [x] sqlite 仅用于不可避免的 gorm 路径(与 73-03 同款豁免,plan verification 允许)

## D-03 Verification (oper_log_service NOT exempted)

- [x] 文件头注释明确 D-03 LOCK + 禁删说明
- [x] 直接测 operLogService.List/GetByID/Delete/BatchDelete/Clean(24 funcs),非 handler mock-through
- [x] List 95.8% / GetByID 85.7% / Delete 75.0% / BatchDelete 100% / Clean 100% —— service 层独立达标,handler 71.2% 未计入

## Nyquist 8-Dimension Self-Audit (per VALIDATION.md template)

| Dim | 73-04 | Evidence |
|-----|-------|----------|
| D1 Functional Correctness | **PASS** | 143 funcs;DB 状态变迁断言(delete 后 count=0、config create/update 后回读、metrics uuid 钩子) |
| D2 API Contract | **SKIP** | service-layer plan(VALIDATION.md plan→dimension mapping: D2 = —) |
| D3 Error Handling | **PASS** | 每方法 ≥1 error 分支:mock 错误透传(Get/Set/Delete/Exists/Expire/FlushDB/GetStats/Reload)、DB error(DROP TABLE 触发 Count/Find/Create/Delete/Exec error) |
| D4 Boundary Cases | **PASS** | 空表/空 keys/空 key/越界分页/非白名单排序(注入串)/TTL 边界(默认 1h vs 显式)/值域校验(Min-Max)/畸形类型跳过/(nil,nil) 降级/不存在 id no-op/系统键与数字开头键过滤 |
| D5 Security | **PASS** | 排序白名单注入守护 ×3(login/oper/metrics: `evil; DROP TABLE` 不破坏表,表行数复验) |
| D6 Performance | **N/A** | Phase 73 不强制 |
| D7 Observability | **SKIP** | operlog.Record 是 handler 层约定(CLAUDE.md);本包 service 不调用 |
| D8 Validation Strategy | **PASS** | 本表 + coverage map 源自 73-VALIDATION.md 模板 |

## D-01..D-13 Lock Verification

| Lock | Status | Evidence |
|------|--------|----------|
| D-01 plan split | honored | 本计划仅 "services 中等 monitor" 切片 |
| D-02 双范本 | honored | 见上节;MetricsProvider 双方法签名以源码为准(计划猜的单方法) |
| D-03 oper_log 不豁免 | honored | 见上节,24 funcs service 直测 |
| D-10 per-sub-package ≥70% | honored | monitor 95.3%(唯一子包) |
| D-11 ratchet | deferred | `.coverage-threshold` 原子更新属 Plan 73-05(全 4 计划闸门) |
| D-12 零业务代码改动 | honored | 4 个 commit 仅 `*_test.go`(git diff --stat 验证) |
| D-13 baseline append | deferred | Phase 73 后由 73-05 写入 |

## SC Mapping

- [x] **SC#6 monitor 半区 (IMP-06)**: services/monitor ≥70%(485 stmts,含 oper_log_service per D-03)—— 95.3%
- [x] **IMP-06 整体完成**: network 92.1%(73-03)+ monitor 95.3%(本计划)→ REQUIREMENTS.md 已勾选 + traceability 表更新为 Done
- [x] SC#8 contribution: 零业务代码改动

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] testify Return 的 time.Duration 需显式类型化**
- **Found during:** Task 1
- **Issue:** `.Return(-1, nil)` 把无类型常量存为 int,mock TTL 方法的 `args.Get(0).(time.Duration)` 断言 panic。
- **Fix:** 改为 `time.Duration(-1)`;其余 Duration 值均显式类型化。
- **Files:** cache_service_test.go
- **Commit:** `870480f`

**2. [Rule 1 - Bug] TTL 在空值跳过判定之前被调用**
- **Found during:** Task 1
- **Issue:** multi-level 列表路径对 value=="" 的键先取 TTL 再跳过,测试缺该期望 → unexpected call panic(测试侧对调用顺序的假设错误,非业务 bug)。
- **Fix:** 补注册 empty:val 的 TTL 期望并注释调用顺序。
- **Files:** cache_service_test.go
- **Commit:** `870480f`

**3. [Rule 3 - Blocking] 局部变量 `base` 遮蔽 `services/base` 包名**
- **Found during:** Tasks 2-3
- **Issue:** 测试内 `base := time.Date(...)` 遮蔽导入的 base 包 → `base.BaseListRequest is not a type` 编译失败。
- **Fix:** 相关用例局部变量改名 loginBase/opBase。
- **Files:** login_log_service_test.go, oper_log_service_test.go
- **Commits:** `f922505`, `39d8701`

**4. [Rule 1 - Bug] 两处测试自身断言错误**
- **Found during:** Task 4
- **Issue:** (a) `len(OS)==8` 的笔误断言(应为 runtime.GOOS);(b) cpuUsage 排序用例忽略 IsAsc=false 默认即 DESC。
- **Fix:** 改为 runtime.GOOS/GOARCH 直比;排序用例显式 IsAsc:true。
- **Files:** server_service_test.go
- **Commit:** `b19fe05`

### Business-code quirks discovered — NOT fixed (D-12), for follow-up

1. **Q1 — normalizeCacheKeyForService 恒等**: `key[:6] == "xingran:"` 用 6 字节切片比 8 字节字面量,恒 false → 前缀永不被剥离(cache_service.go:764)。该函数覆盖 66.7% 因 `return key[6:]` 是死代码。与 CLAUDE.md "cache prefix strip" gotcha 直接相关 —— GetCacheInfo/OperateCache(dal/exists/expire)对带 `xingran:` 前缀的键全部原样透传。
2. **Q3 — loginLogService.Clean() 删不存在的表**: 原生 SQL `DELETE FROM sys_login_log`,但模型表名是 `sys_logininfor`,且全库无 sys_login_log 迁移 → 生产 Postgres 上"清空登录日志"必然报"清空登录日志失败"。测试锁定双分支(stub 表 happy / 无表 error)。
3. **Q4 — operLogService.List 的 OperName 过滤列名错误**: `operator_name LIKE` 但真实列是 `oper_name` → 生产查询该过滤器必报"查询操作日志总数失败"。测试锁定 error 行为(TestOperLogService_List_FilterOperName_BrokenColumn_Error)。
4. **Q5 — 非法排序列时默认排序丢失**(login/oper/metrics 三处同款): OrderByColumn 非空但不在白名单时,ApplySort 忽略后 service 不再追加默认 time DESC(默认排序仅在 OrderByColumn=="" 时加)→ 结果顺序不确定。测试锁定为"无错 + 全量返回 + 无注入"。

### Plan-note corrections

- 计划的 key_links 要求 `CREATE TABLE sys_login_log`,但 models.LoginLog.TableName() 是 **sys_logininfor**(Phase 72 handler 测试同款)。按计划自身 "Reuse DDL from Phase 72 if available" 条款采用正确表名;`db.Exec(\`CREATE TABLE sys_login_log\` 模式以 Clean quirk stub 表形式满足。
- 计划把 MetricsProvider 描述为单方法接口;实际是 `GetServerInfo + GetCurrentMetrics` 双方法(server_service.go:56),mock 按实际签名实现。
- 计划 server DDL 猜测的列(last_heartbeat 等)以 models/monitor.go 为准重建(BaseModel 软删列 + last_active_at)。
- commit 粒度: 计划 Task 5 写"4 test files + SUMMARY 单 commit",按 orchestrator 指令与 73-03 先例采用 per-task commit(4 test commits + 1 docs commit)。

## Auth Gates

None.

## Known Stubs

None — cache/config/metrics providers are deliberate testify mocks (test doubles, not stubs); all DB paths exercise real SQL against in-memory sqlite; realtime fallback paths execute real system metrics.

## Threat Flags

None — test-only changes; no new network endpoints, auth paths, or schema changes.

## Self-Check: PASSED

- Files: cache_service_test.go / login_log_service_test.go / oper_log_service_test.go / server_service_test.go / 73-04-SUMMARY.md — all FOUND
- Commits: 870480f / f922505 / 39d8701 / b19fe05 — all FOUND in git log
- Coverage gate re-verified at write time: `go test -cover -count=1 ./internal/services/monitor/` → ok, 95.3% (≥70%)
- D-12: 4 个 test commit 的 git diff --stat 仅触及 `internal/services/monitor/*_test.go`

---
*Phase: 73-p1-pending*
*Completed: 2026-08-21*
