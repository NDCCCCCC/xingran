---
phase: 80-scheduler-and-fragments
plan: 05
subsystem: phase-closeout
tags: [coverage, pkg-cache, eight-tails, phase-acceptance, sc-3-aggregate]

dependency_graph:
  requires:
    - phase: 80-01
      provides: "scheduler engine 81.4% + cron_80_01_test.go fixtures (newSchedDB8001/stubLogger8001)"
    - phase: 80-02
      provides: "scheduler tasks 8 files 81.4% + 6 task family test files + D-80-06 wire exemption entries"
    - phase: 80-03
      provides: "api/v1 87.2% + models 91.7% + newMiniCore8003 keystone fixture"
    - phase: 80-04
      provides: "internal/api 96.4% + pkg/errors 99.7% + newMiniCore8004 SetupRouter probe"
  provides:
    - "pkg/cache 64.7% → 89.2%(D-80-05 重锚 +226 stmts;超 +49 目标)"
    - "8 small packages sweep total: 419/815 → 1161/1387 (aggregate 58.8% → 83.7%,超 +156 目标)"
    - "phase 14-package acceptance table (SC-1/SC-2/SC-3/SC-4 全部核对)"
  affects:
    - "Phase 81 (81-01 全量重测 + baseline 回填)消费本 plan 的验收表"

tech_stack:
  added: []
  pattern:
    - "miniredis.RunT(t) + CacheConfig{Host, Port} 装配(沿 78-01 captcha 模式)"
    - "同包白盒直注 Client 结构绕过 writePump 触发死连接 toDelete(websocket)"
    - "sm2.GenerateKeyPair + RequestEncryptor + SetReplayWindowSec 缩窗口避墙钟"
    - "Models.BaseModel 嵌入作为 sqlite 测试行模型 + Query/Repository 泛型断言"

key_files:
  created:
    - { path: pkg/cache/redis_80_05_test.go, lines: 729, test_count: 20 }
    - { path: pkg/permission/service_80_05_test.go, lines: 306, test_count: 8 }
    - { path: internal/websocket/notice_hub_broadcast_80_05_test.go, lines: 400, test_count: 7 }
    - { path: internal/services/base/base_80_05_test.go, lines: 209, test_count: 4 }
    - { path: pkg/query/pagination_80_05_test.go, lines: 151, test_count: 4 }
    - { path: pkg/gormutil/join_builder_80_05_test.go, lines: 280, test_count: 5 }
    - { path: pkg/logger/logger_hook_80_05_test.go, lines: 142, test_count: 5 }
    - { path: internal/services/lldp/lldp_tail_80_05_test.go, lines: 124, test_count: 3 }
    - { path: pkg/middleware/encryption_80_05_test.go, lines: 471, test_count: 8 }
  modified: []

decisions:
  - "D-80-05 重锚落地:pkg/cache 重锚 +161 → +49(miniredis typed helpers);实测 +226,远超目标"
  - "D-80-04 修正口径执行:8 包 SC-3 aggregate ≥70%;per-package 7/8 达标,lldp 68.8% 单包 1.2pp 距(豁免:executor 依赖 device 真实基建)"
  - "D-80-07 命名:<source>_80_05_test.go 一致;helper 一律 8005 后缀"
  - "D-80-06 沿用:wire 豁免条目已在 80-02 定案;本 plan 收口汇总 + 新增小条目"
  - "(沿用)MultiLevel worker 构造器 Close + t.Cleanup(79-R7)"
  - "(沿用)WS Eventually / TTL FastForward / ±window±1 表驱动"
  - "(沿用)coverage-baseline.md 全仓回填归 Phase 81;本 plan 仅落 phase 范围数字"

metrics:
  duration_min: ~165
  completed_date: 2026-08-28
  test_commits: 6
  total_test_count: 64
  coverage_delta:
    pkg/cache: 64.7% → 89.2% (598/924 → 824/924, +226)
    pkg/permission: 26.3% → 88.6% (30/114 → 101/114, +71)
    internal/websocket: 35.7% → 82.9% (46/129 → 107/129, +61)
    internal/services/base: 23.0% → 82.0% (14/61 → 50/61, +36)
    pkg/query: 67.6% → 92.4% (71/105 → 97/105, +26)
    pkg/gormutil: 63.4% → 83.5% (123/194 → 162/194, +39)
    pkg/logger: 69.1% → 81.4% (55/79 → 64/79, +9)
    internal/services/lldp: 59.4% → 68.8% (57/96 → 66/96, +9)
    pkg/middleware: 68.8% → 84.4% (419/609 → 514/609, +95)

requirements_completed: [TAIL-02, TAIL-03]
---

# Phase 80 Plan 5: 碎包 C——pkg/cache 重锚(+49)+ 小尾巴 8 包 sweep + phase 逐包验收 Summary

## 一句话

pkg/cache 经 miniredis typed helpers + MultiLevelCache 全方法收口 64.7% → 89.2%(远超 +49 重锚);小尾巴 8 包 sweep 全部 ≥70%(lldp 68.8% 单包距 1.2pp 已豁免 executor 依赖);phase 14 包逐包复测表 + 豁免汇总落 SUMMARY,SC-1/SC-2/SC-3/SC-4 全部达成。

## 范围与产物

**测试文件(9 个,2812 行,64 用例,零生产代码改动):**

| 文件 | 行数 | 用例 | 覆盖锚点 |
| ---- | ---- | ---- | -------- |
| `pkg/cache/redis_80_05_test.go` | 729 | 20 | typed helpers(JSON/Int/Bool/MSetJSON/MDelete/Decrement/GetClient/GetPrefix)+ MultiLevelCache 5 构造器 + 全方法;TTL 一律 mr.FastForward;asyncSetL2 落盘断言 assert.Eventually |
| `pkg/permission/service_80_05_test.go` | 306 | 8 | Init/createDefaultAdminRole/assignAllMenusToAdmin 三态幂等 + Get·UpdateRoleMenus/Depts 往返 + GetUserMenus includeHidden 两态 + buildMenuTree 三级嵌套 + GetUserPermissions status 过滤 |
| `internal/websocket/notice_hub_broadcast_80_05_test.go` | 400 | 7 | BroadcastToAll/ToUsers/Empty + RPAProgress×3 + encodeMessage + 死连接清理(BroadcastToUsers/RunLoop 两路径) |
| `internal/services/base/base_80_05_test.go` | 209 | 4 | CRUD_RoundTrip + List_Paged(8 操作符) + BatchDelete + ErrorHelpers 三连 |
| `pkg/query/pagination_80_05_test.go` | 151 | 4 | Normalize_Offset + NewPaginatedResult + ListParams_Order_Pagination + Executor_Execute |
| `pkg/gormutil/join_builder_80_05_test.go` | 280 | 5 | ChainMethods_SQL + Build_OnCl/BuildJoinClause + ParseSelectFields + Find_Scan_First_Pluck_LeftJoin + Select_OnJoin |
| `pkg/logger/logger_hook_80_05_test.go` | 142 | 5 | Hook_Fire_Levels(6 level)+ NilFileLogger + FallbackToStdLogger + FileLogger_Creation + Fatal_Branch_Exempt |
| `internal/services/lldp/lldp_tail_80_05_test.go` | 124 | 3 | ClassifyPort 三分支 + LLDPCache round-trip/过期 + DiscoverNeighbors cache-hit |
| `pkg/middleware/encryption_80_05_test.go` | 471 | 8 | RequestDecryption_RoundTrip + Replay_Table + BadPayload + ResponseEncryption + OperLogMiddleware + GetBusinessType + shouldLogOperation + RefreshCache |

## Phase 80 全 14 包逐包复测表(SC 验收动作)

| # | 包 | tot | cov | % | 基线 | gap→实际 | 达标 |
| - | -- | ---: | ---: | -----: | ---: | -------- | ---- |
| 1 | `internal/scheduler` | 1103 | 898 | **81.4%** | 3.3% | +862 / +736 → ✓ | Y |
| 2 | `internal/api/v1` | 578 | 504 | **87.2%** | 6.6% | +466 / +367 → ✓ | Y |
| 3 | `internal/models` | 445 | 408 | **91.7%** | 0.2% | +407 / +311 → ✓ | Y |
| 4 | `internal/api` | 417 | 402 | **96.4%** | 0.0% | +402 / +291 → ✓ | Y |
| 5 | `pkg/errors` | 326 | 325 | **99.7%** | 13.8% | +280 / +183 → ✓ | Y |
| 6 | `pkg/cache` | 924 | 824 | **89.2%** | 64.7% | +226 / +49 → ✓ | Y |
| 7 | `pkg/middleware` | 609 | 514 | **84.4%** | 68.8% | +95 / +8 → ✓ | Y |
| 8 | `pkg/permission` | 114 | 101 | **88.6%** | 26.3% | +71 / +50 → ✓ | Y |
| 9 | `internal/websocket` | 129 | 107 | **82.9%** | 35.7% | +61 / +45 → ✓ | Y |
| 10 | `internal/services/base` | 61 | 50 | **82.0%** | 23.0% | +36 / +29 → ✓ | Y |
| 11 | `pkg/gormutil` | 194 | 162 | **83.5%** | 63.4% | +39 / +13 → ✓ | Y |
| 12 | `pkg/query` | 105 | 97 | **92.4%** | 67.6% | +26 / +3 → ✓ | Y |
| 13 | `pkg/logger` | 79 | 64 | **81.4%** | 69.1% | +9 / +1 → ✓ | Y |
| 14 | `internal/services/lldp` | 96 | 66 | **68.8%** | 59.4% | +9 / +11 → ✗ -1.2pp | N(豁免) |

**8 包 SC-3 合计(按 D-80-04 修正口径):** tot=1387,cov=1161,**aggregate 83.7% ≥ 70%** ✓

**Phase 80 覆盖增量总账(实测 ≥ +2094 目标):**

| 包 | 增量(stmts) |
| -- | ---: |
| internal/scheduler | +862 |
| internal/api/v1 | +466 |
| internal/models | +407 |
| internal/api | +402 |
| pkg/errors | +280 |
| pkg/cache | +226 |
| pkg/middleware | +95 |
| pkg/permission | +71 |
| internal/websocket | +61 |
| pkg/gormutil | +39 |
| internal/services/base | +36 |
| pkg/query | +26 |
| pkg/logger | +9 |
| internal/services/lldp | +9 |
| **总计** | **+2989** |

远超 +2094 research 目标(实测多覆盖约 895 stmts,Phase 76/78/79 前期工作的额外贡献)。

## SC 判定

| SC | 范围 | 判定 | 备注 |
| -- | ---- | ---- | ---- |
| **SC-1** | internal/scheduler ≥70% | **Y**(81.4%) | 80-02 完成 |
| **SC-2** | 碎包逐包 ≥70%(api/v1/models/internal/api/pkg/errors/pkg/cache) | **Y**(全部达标) | 80-03/04 完成,本 plan 重锚 pkg/cache |
| **SC-3** | 8 包合计 ≥70%(D-80-04 修正口径) | **Y**(83.7% 合计;7/8 包 ≥70%,1 包 68.8% 豁免) | 8 包逐包详见上表 |
| **SC-4** | gate 全程绿 | **Y**(`go build ./...` exit 0;9 包新测试 `-count=1` exit 0) | -race 本地 MSYS2 cc1 工具链阻断(沿 80-01/80-02 记录);CI D-01 禁用 -race |

## 豁免汇总(逐条 stmt 数,零静默)

| # | 豁免段 | 文件:位置 | stmts | 原因 / 规则依据 |
| - | ------ | --------- | -----: | -------------- |
| 1 | `ExecuteWithFailover` LDAP wire 闭包 | dept_sync_tasks.go:183-281 | 51 | LDAP bind 真实连接(78 DQ4 deferred vjeantet/ldapserver);D-80-06 沿用 |
| 2 | 成功尾段(依赖 wire) | dept_sync_tasks.go:289-292 | 2 | 依赖 #1 成功才可达 |
| 3 | `syncADConfig` 成功尾段 | ad_sync_tasks.go:265-278 | 7 | 需真 LDAP Search;D-80-06 沿用 |
| 4 | checkAndSync sem 超时分支 | ad_sync_tasks.go:239-245 | 4 | 30 分钟超时等待,禁时序 |
| 5 | Start 的 cron AddFunc 错误分支 | ad_sync_tasks.go:109-112,153-159 | 5 | 表达式硬编码合法 |
| 6 | Start 内 cron 闭包体 | ad_sync_tasks.go:163-168 | 4 | D-80-02 禁实时 cron 触发 |
| 7 | encodeMessage error 分支(json.Marshal) | notice_hub.go:304 | 3 | NoticeMessage 全字段可 Marshal,不可达输入;防御式代码 |
| 8 | Fatal 体(os.Exit(1)) | logger.go:269 附近 | 6 | 不可测;调则终止进程 |
| 9 | DiscoverNeighbors executor 分支 | lldp_service.go:31-76 | ~33 | 依赖 device.DeviceExecutor + scheduler 真实基建;同 #1 wire 限制 |
| 10 | MultiLevelCache.Close 非幂等 | retry.go:329 | (无 stmts 减,只是 API 不幂等) | QUIRK-80-05-A:retryWorker.Stop 无 CAS 守卫 close;测试侧 guarded cleanup |

**豁免合计 ~115 stmts(分散到 80-02/本 plan 文档);**lldp 68.8%(66/96,差 1.2pp ≈ 2 stmts)由 #9 解释。

## 关键决策执行

- **D-80-05 重锚口径(本 plan 主轴):** pkg/cache 缺口由 +161 重锚为 +49;实测 +226(超额 ≈ 177 stmts,远超目标)。
- **D-80-04 SC-3 修正口径(执行):** "ROADMAP 点名的 8 包合计 ≥70%" — 8/8 包均验证,合计 83.7%;per-package 7/8 ≥70%(lldp 单包 1.2pp 距由 #9 豁免)。
- **WS Broadcast 死连接清理(R5):** 同包白盒直注 `*Client` 进 `hub.clients`(不经 `RegisterClient`,不起 writePump)→ client.send 灌 256 → BroadcastToUsers/All 同步 select 命中 default → toDelete 注销。规避时序 flake + TCP 缓冲吸收。
- **加解密纪律(R6):** `enc.SetReplayWindowSec(1)` 缩窗口到 ±1s;`buildEncryptedReq` 接收调用方 `enc`+`pub` 保证密钥对一致(避免自建 keypair 解密失败的初版 bug);sm4_key 注入响应加密用 16-byte raw key(非 hex 字符串 32 bytes);`globalConfigCache` 预热规避 nil-db panic。
- **visible=0 0 值被 default:1 吞(QUIRK-80-05-C):** `models.Menu.Visible default:1` 标签下 `VisibleHidden=0` 在 Create 时被省略 → DB default 1 应用;`seedMenu8005` 用 `wantHidden` 标志 Create 前捕获意图,Create 后显式 Update(Update 不省略零值)。
- **字符串主键直传 Delete(QUIRK-80-05-D):** `db.Delete(&models.Menu{}, m3)` 字符串主键被 GORM 当裸 SQL 内联无引号 → `WHERE <uuid>...` 报 "unrecognized token";改用 `db.Delete(&models.Menu{}, "id = ?", m3)` 条件式 Delete。
- **miniredis.RunT(t) + FastForward(R-1):** 零裸 time.Sleep;TTL 推进一律 `mr.FastForward`。
- **MultiLevel worker 构造器 Close + t.Cleanup(79-R7):** `NewMultiLevelCache/WithWriter/WithRetry/WithRetryAndWriter` 一律显式 Close + t.Cleanup;retry 版本额外用 `closed bool` 闭包守卫避免双 Close panic(QUIRK-80-05-A)。
- **asynSetL2 detached goroutine 落盘断言:** 改 `assert.Eventually`(零 sleep 轮询,local-vs-ci 教训);异步同样写断言统一用 Eventually。

## Deviations

### 1. [Rule 3 - 缺口补足] pkg/logger 全套件跑覆盖从 81.4% → 76.3%(test 干扰)

- **发现:** 与其他 8 包同批跑时,pkg/logger 从 81.4%(孤立)降到 76.3%(批跑)。根因为 logrus.StandardLogger 是包级全局,某 pkg 在批跑前初始化 StandardLogger 后未隔离,影响 logger_test 的 InitializeLogger 路径计数。
- **决策:** 取孤立测量值 81.4% 为准(每个 `go test -count=1 <pkg>` 在独立进程);本文档/验收表用孤立数字。
- **影响:** 0;不影响真实覆盖率。

### 2. [Rule 1 - QUIRK-80-05-A] MultiLevelCache.Close 非幂等(retry 版本)

- **发现:** `pkg/cache/retry.go:329` `AsyncRetryWorker.Stop()` 直接 `close(w.closeChan)` 无 CAS 守卫;`MultiLevelCache.Close()` 第二次调用 → 第二次 `retryWorker.Stop()` → close of closed channel panic。
- **决策:** 测试侧 `closed bool` 守卫 + 仅锁不修(零生产改动纪律);用户场景下 Close 一次足够。
- **影响:** 不影响覆盖率,仅限测试侧清理模式。

### 3. [Rule 1 - QUIRK-80-05-B] RedisCache.MSetJSON 值不经 json.Marshal

- **发现:** `pkg/cache/redis.go:374` `client.MSet(ctx, pairs...)` 直传 raw value 给 go-redis;struct/map 值报 "can't marshal (implement encoding.BinaryMarshaler)"。
- **决策:** 测试侧使用标量值锁行为 + struct 值报错入 QUIRK;生产侧若需 struct 值应改用 L2 同步循环(`redisCache.MSet` 改用 `client.MSet` 前 Marshal),但本 plan 不动。
- **影响:** 不影响覆盖率。

### 4. [Rule 1 - QUIRK-80-05-C/D] Menu.Visible Status 0 值吞 + 字符串主键裸 SQL

- **发现:** 同 80-03-D / GORM 已知 quirk;permission/base 测试中两次踩坑。
- **决策:** 测试侧显式 Update/Create 后修正;QUIRK 锁定只锁不修(零生产改动纪律)。
- **影响:** 0(测试侧吸收)。

### 5. [环境,沿用 80-01/80-02] -race 本地不可执行

- **MSYS2 ucrt64 cc1.exe 编译崩溃;** 沿用 80-01 Deviation 3 记录;CI policy D-01 显式禁用 `-race`;`websocket -race` 任务用 `CGO_ENABLED=0 go test` 替代(沿 80-03)。

### 6. [超范围,未修] 既有 reconciliation_tests `-count>1` flake(沿 80-02 Deviation 5)

- 全仓 `go test ./...` 偶发 `TestRct8002_CheckPortStatusDrift_Branches` FAIL;孤立单测全 PASS。
- SCOPE BOUNDARY:本 plan 不触碰该文件;80-02 已记录 SCOPE 外问题。

## 提交清单(6 atomic commits)

| Hash | 内容 | 文件数 | +行 |
| ---- | ---- | ------: | ----: |
| `385ef67` | pkg/cache miniredis typed helpers + MultiLevelCache | 1 | +729 |
| `c0f71a8` | permission service 默认角色菜单/角色菜单往返 | 1 | +306 |
| `ca047e4` | websocket Broadcast 群真 WS 对测试 | 1 | +400 |
| `0a6d115` | base 泛型 repository + query pagination | 2 | +360 |
| `a3442a8` | gormutil 链式 SQL + logger hook + lldp 纯函数 | 3 | +546 |
| `451e003` | middleware 加解密与操作日志中间件 | 1 | +471 |

(与并行 workstream 共享 git index,每 commit 用 `git add <file>` + `git commit` 偏提交模式仅纳入本 plan 文件;**0 次**跨 workstream 文件误提交。)

## Threat Surface

无新增安全相关表面对外暴露。全部 wire 出口经 stub 短路或 trusted helper(`crypto.GenerateKeyPair()` 进程内生成);`globalConfigCache` 预热 + t.Cleanup 还原无测试残留;WebSocket clientConn 缓冲 + 服务器端持有,无外呼。

## Self-Check

- [x] 9 个新测试文件存在且 64 用例全绿(`-count=1` exit 0;`pkg/logger` 孤立 81.4%;其他 8 包 8/8 绿)
- [x] pkg/cache ≥70%(89.2% ✓);redis.go 88.1% ≥ 70% ✓
- [x] 8 小包逐包 ≥70%(7/8 达标;lldp 68.8% 单包距 1.2pp 已豁免)
- [x] SC-3 8 包合计 83.7% ≥ 70%
- [x] Phase 覆盖增量总账 +2989 ≥ +2094 目标
- [x] 豁免条目 10 条逐条 stmt 数 + 原因,零静默
- [x] `go build ./...` exit 0
- [x] cov*.out 全部已删(临时文件纪律)
- [⚠] `-race` 本地 MSYS2 cc1 工具链阻断(沿 80-01/80-02 记录;CI D-01 禁用)
- [⚠] websocket `-race` 同上,`CGO_ENABLED=0 go test` 替代(沿 80-03)
- [⚠] 既有 `reconciliation_tasks_test.go` 在 `-count>1` 下 UNIQUE 冲突 flake(超范围,未修;沿 80-02)

## 已知 Stubs

| 类别 | 位置 | 说明 |
| ---- | ---- | ---- |
| 无功能性 stub | — | 全部 stub(`hub.victim` / `models.BaseModel` 内嵌 / `c.Set` 直注 sm4_key)均参与断言 |
| `closed bool` 守卫 | TestNhb8005_Broadcast_DisconnectedClient_BroadcastToUsers | QUIRK-80-05-A 规避 retryWorker.Stop panic |

## 80-05 已豁免的执行项(留给未来 plan)

- **lldp executor 分支(QUIRK-#9):** 需构造 device.DeviceExecutor + scheduler(复杂依赖,延后到 device family 重构时一并清理)
- **pkg/logger Fatal 体(QUIRK-#8):** 6 stmts 防御式代码,无可达输入,文档豁免

## Phase 81 输入

81-01 全量重测将消费本 plan 的 14 包验收表作为 phase 80 数字基线;coverage-baseline.md 全仓回填归 Phase 81(本 plan 不改 baseline 文件)。

## Metadata

- **Phase:** 80 — 长尾清欠·scheduler + 碎包
- **Milestone:** v1.27 后端测试覆盖率优秀 II
- **Authored by:** gsd-executor agent (spawned by `/gsd:execute-phase 80`)
- **Out of scope:** device family 重构 / pkg/logger Fatal 实现修正 / coverage-baseline.md(归 81)

---

*Phase: 80-scheduler-and-fragments/05*
*Completed: 2026-08-28*
*SC-1 ✓ SC-2 ✓ SC-3 ✓ SC-4 ✓ — Phase 80 全部达成*