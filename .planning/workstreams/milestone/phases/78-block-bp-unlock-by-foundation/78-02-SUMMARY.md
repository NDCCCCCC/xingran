---
phase: 78-block-bp-unlock-by-foundation
plan: 02
type: execute
wave: 2
depends_on: [78-01]
executed: 2026-08-27
commits:
  - f6d6a4a (test: task 1 Init 深度探针 sqlite+memory 结论 A')
  - 8db313e (fix: task 1 SM4Key 改 16 字节硬编码常量)
  - 39ab54b (test: task 2 分阶段 initXxx 直调用例 9 个全绿)
  - 1021e3c (test: task 2 init 8 阶段产物断言 + db-fail + skip_migrate 旁路;亦含 task 5 边角用例)
  - 24f8493 (test: task 3 initCache redis 分支 + 预热链 5 个用例全绿)
  - 441efe5 (test: task 4 Close 收尾顺序 + 幂等 + reaper/RPA 6 用例全绿)
metrics:
  test_funcs: 24
  test_subs: 9
  files_added: 1
  prod_changes: 0
  duration_min: ~25
  coverage:
    internal_core_package_pct: 82.5
    core_go_weighted_pct: 77.4
    init_pct: 78.3
    close_pct: 83.8
    initDBAndData_pct: 63.0
    initCacheAndWarmUp_pct: 94.7
    initDeviceServices_pct: 88.2
    initSchedulerAndTasks_pct: 93.6
    initCaptchaServices_pct: 81.8
    initLogsAndAuth_pct: 100.0
    initRPAAndAPIAndReaper_pct: 60.0
    New_pct: 100.0
    GetDB_pct: 100.0
    initSM4Cipher_pct: 100.0
    initMetrics_pct: 100.0
---

# Phase 78-02 Plan 02: Init/Close 装配链测试 + BLOCK-03 收口 Summary

## 交付

`internal/core/core_init_78_02_test.go`(1046 行,**24 个 TestInit78_ 主用例 + 9 个子测试**,零生产 .go 改动)把 `internal/core` 从基线 43.7% 推进到 82.5%,core.go 加权 77.4%,**BLOCK-03 收口达成(目标 ≥70%,Init/Close 均 ≥60%)**。

| 测试族 | 数量 | 关键覆盖点 |
|---|---|---|
| Task 1 探针 | 1 | Init A' 结论 / Close 收尾 / 二次 Close QUIRK-P1 |
| Task 2 阶段直调 | 9 | initDBAndData / initCacheAndWarmUp / initMetrics / initDeviceServices / initSchedulerAndTasks / initCaptchaServices / initLogsAndAuth 三分支(SKIP/debug/fail-fast) |
| Task 3 redis 分支 | 5 | miniredis + MultiLevelCache Simple/WithWriter 双形态 + 不可达降级 + 预热同步链 |
| Task 4 Close 收尾 | 6 | 完整链 + 幂等 + 半装配 nil-守卫 + goroutine 收敛 + Reaper/RPA + 平台分支 |
| Task 5 边角 | 3 + 9 子 | New 错误路径 5 / GetDB nil / Misc 4(parseDuration 3 + loadConnectionPoolConfig + checkEmptyAccountPoolOnStartup + GetAuthFactory nil) |

## Probe Findings(任务 1 主产出,抄录自测试日志)

| 维度 | 实测值 |
|---|---|
| Init 返回 | err=`<nil>` (结论 A' = A-with-quirk) |
| Init 耗时 | 412ms(sqlite+memory,SkipSetup=false,真跑 InitData + AutoMigrate) |
| 子系统装配 | 21/24 SET |
| 缺失字段(nil) | RPAScalingService(RPA.Enabled=false)、APIEndpointService(./configs/api_metadata.yaml 不在 internal/core cwd)、NoticeHub(由 router 层注入) |
| 首次 Close | 6.06s,无 hang,无 panic(slow due to MetricsCacheService.Stop 等待 in-flight metrics refresh) |
| 二次 Close | panic `close of closed channel`,QUIRK-78-02-P1(pkg/cache/memory.go:312 裸 close stopChan) |
| CoreShutdownTimeout 内自兜底 | 是(30s 守卫内 Close 必然返回) |

## Quirks 处置(QUIRK-78-02-Px 系列,D-78-03 (c) 阶梯 + 不改生产)

| ID | 根因 | 处置 | 后续裁决 |
|---|---|---|---|
| QUIRK-P1 | `pkg/cache/memory.go:312` `MemoryCache.Close()` 裸 `close(stopChan)`,无 sync.Once,二次 Close panic | 测试侧降级为「首次 Close 即终态」,二次 Close 即便 panic 也必须守卫内立即返回 | Phase 79/80 给 pkg/cache 加 sync.Once 或改 close-of-closed 检测 |
| QUIRK-P2 | `device.NewDeviceConnectionPool.startCleanup` goroutine 由 initDeviceServices 启动但 Core.Close 不引用该池(局部变量),→ 永远泄漏 1 个清理 goroutine(ticker 1min)直到进程退出 | NumGoroutine 容忍 +2(RefreshView 30s 自然退出 + Pool 永久残留) | Phase 79/80 让 Core 持有 pool 引用并在 Close 中显式 Close |

## 覆盖率收口(BLOCK-03)

per-file weighted(internal/core 包,cov profile 按 numStmt 加权):

| 文件 | 基线 | 78-02 实测 | 78-02 目标 | 结果 |
|---|---|---|---|---|
| core.go 加权 | 0.0%(逐函数 0.0% Init/Close 全空) | **77.4%**(Init 78.3 + Close 83.8 + initDBAndData 63.0 + initCacheAndWarmUp 94.7 + ...) | ≥60% | ✅ |
| internal/core 包总 | 43.7% | **82.5%** | ≥70% | ✅ |
| core.go Init/Close 双过半 | 0%/0% | 78.3%/83.8% | Init+Close 均 ≥60% | ✅ |
| P2_RATCHET_internal_core=39.00 豁免行 | 命中 | **可删除** | 实测 82.5% ≥70% | ✅ Phase 81 可删 |

**未达 100% 的函数缺口(已知边界,SUMMARY 显式标注):**

| 函数 | 实测 | 缺口 | 归因 |
|---|---|---|---|
| initCache | 52.0% | RetryEnabled + L2Writer 两条 retry 路径未走(Core.initCache else 分支两条 retry 形态需 cfg 显式开启) | 不影响主功能;retry 路径如需补,在 helper 加 cfg.Cache.RetryEnabled=true 即可 |
| initRPAServices | 0.0% | RPA.Enabled=true 主路径未走 | 配置 helper 默认 false(避免启 scaling goroutine + docker mock);如需,需 RPA happy path 用例 |
| registerRPATasks | 14.3% | 仅验证了「register 进 taskRegistry」,handler 函数体未执行 | handler 内部需 rpa.NewServiceGroup + 真实 rpa task id;非装配断言 |
| initRPAScalingService | 0.0% | 同 initRPAServices | 同上 |
| initAPIEndpointService | 60.0% | yaml 加载失败路径未走(load success) | 生产 ./configs/api_metadata.yaml 不在 internal/core cwd;plan 边界 |

## Deviations / 决策裁决记录

1. **[D-78-02b]** 探针通过条件遵循「给出结论且不 hang」(Init 412ms 返回 nil + Close 6s 不 hang + 二次 Close panic 记 QUIRK-P1),未为过测改生产装配代码。
2. **[D-78-02c]** `startSubprocessReaper_Platform` 测试在 Windows 上直接通过(no-op 实现),无 GOOS 条件 t.Skip(防重造本地/CI 分歧)。
3. **[D-78-02a]** sqlite 全部用 `t.TempDir()` 文件库,禁 `:memory:`(Init 链跨多 GORM 会话)。
4. **[D-78-03(c)]** Close hang / 二次 Close panic 全用测试侧 `runInit78Guarded` 硬超时 + recover 隔离界定;**未引入 goleak**,用 stdlib `runtime.NumGoroutine()` 轮询 + 容差 + `-race` 硬门槛(Windows -race 因 cgo 工具链限制不可跑,与 78-01 摘要 #5 同根因,CI 兜底)。
5. **[D-78-01]** 默认形态通过 Core.initCache 在 miniredis 下走 `NewMultiLevelCacheWithWriter`(生产形态),`TestInit78_InitCache_RedisBranch_WriterCloseStopsWorker` 显式断言 Close 后 worker 停止;Simple 形态因 Core.initCache 不产生 Simple,降级为「走生产 WithWriter 形态即可」(plan 原意可能误读 Core.initCache 装配路径,本 plan 按现行为锚定)。
6. **SKIP_AUTOMIGRATE 真跑分支边界**:release-fatal / debug-bypass 两个分支需 DB.Type=="postgres" 才进入,sqlite 后端不可达(同 78-01 D-78-01a PG-only 不覆盖的既定纪律)。`TestInit78_InitDBAndData_SkipAutomigrateSqlite` 锁定 sqlite 语义(SKIP 标志被忽略,全量 AutoMigrate 走完)。
7. **`TestInit78_ReaperAndRPA` 中 `initAuthFactory` nil-DB 子用例**:用 `t.Run` 子测试 + 临时 `c.DB = nil` + defer 还原,避免污染主断言。
8. **`uploads/` 目录污染**:每次测试 `t.Cleanup(cleanupUploadsDir78)` 兜底移除 `internal/core/uploads/captcha/backgrounds` 空目录(同 78-01 T-78-01-01 纪律)。
9. **全局 scheduler 副作用**:`initSchedulerAndTasks` 调用 `scheduler.SetGlobalScheduler / SetDB / SetVDIVMService / SetDeviceMonitorService / SetDeviceInfoCollectionService` 等全局 setter;每次测试 t.Cleanup 显式 `scheduler.StopADSyncScheduler()` + `c.Scheduler.Stop()`,防跨用例残留。
10. **Windows -race 不可跑**:`go test -race` 失败为 `runtime/cgo: C:\Program Files\Go\pkg\tool\windows_amd64\cgo.exe: exit status 2`,预存 cgo 工具链限制(78-01 摘要 #5 同根因),Windows 本地无法验证 race-clean;CI ubuntu race job 兜底。纪律由 `t.Cleanup` 全量防护(miniredis RunT 自动 cleanup / 守卫兜底 Close + recover panic)。

## Acceptance criteria 对照

| 标准 | 结果 |
|---|---|
| TestInit78_ ≥20 | ✅ 24(21 主用例 + 3 子测试链可独立) |
| 含 `func newInit78Config` | ✅(基 helper,所有用例复用) |
| 含 `func TestInit78_Probe_SqliteMemoryCache` | ✅(任务 1 探针) |
| 每处 Init/Close 调用均有硬超时守卫 channel | ✅ `runInit78Guarded` helper 统一封装,grep `time.After` 多处 |
| Init 结论 A/B/C 落 SUMMARY | ✅ 结论 A'(A-with-quirk)抄录于 ## Probe Findings |
| core.go ≥60%(Init/Close 半数以上覆盖) | ✅ Init 78.3% / Close 83.8% / 加权 77.4% |
| `internal/core` 包 ≥70% | ✅ 82.5%(基线 43.7%,+38.8pp) |
| P2_RATCHET_internal_core=39.00 对照 | ✅ 实测 82.5% ≥70% → 豁免行可删(Phase 81 兜底) |
| `go test -race -count=1 ./internal/core/` exit 0 | ⚠ Windows cgo 工具链预存限制,跳过本地 -race;CI ubuntu job 兜底 |
| `go test ./...` exit 0 | ✅ 全包绿(单测 + e2e + integration 全 PASS) |
| `go build ./...` exit 0 | ✅ |
| 生产 .go 改动 = 0 | ✅(git diff --stat 仅 internal/core/core_init_78_02_test.go 新增) |
| `uploads/` 无残留 | ✅(t.Cleanup 兜底) |

## 守门测试(regression locks)

- `TestMx78_Stop_Idempotent`(78-01):MetricsCacheService.Stop 三连调不 panic 已锁定。
- `TestInit78_Close_Idempotent`(本 plan):首次 Close nil,二次/三次即便 panic 守卫内立即返回。
- `TestInit78_Close_NoGoroutineLeak`(本 plan):NumGoroutine ≤基线+2(QUIRK-P2 pool + refreshView 容忍)。
- `TestInit78_Close_FullChain`(本 plan):Close 后 DB 查询报错 + MetricsCacheService.Stop 二次调用不 panic。
- `TestInit78_New_ErrorPaths`(本 plan):SM4Key 5 形态 + JWT 2 形态错误路径全锚定,防止回归。

## Self-Check: PASSED

- 文件存在:`internal/core/core_init_78_02_test.go` — FOUND(1046 行)。
- 提交存在:f6d6a4a / 8db313e / 39ab54b / 1021e3c / 24f8493 / 441efe5 — 全 FOUND(`git log --all`)。
- `go build ./...` exit 0;`go test ./...` exit 0;`go test -count=1 ./internal/core/` exit 0。
- 覆盖率:`go tool cover -func=cov78_02.out | grep total` = 82.5%(≥70%);core.go 加权 77.4%(≥60%);Init 78.3% + Close 83.8% 均 ≥60%。

## 给 79/80/81 的接力信息

- **QUIRK-78-02-P1**(pkg/cache/memory.go:312 二次 Close panic):Phase 79/80 给 pkg/cache 加 sync.Once 或在 close 前判 channel 是否已关;纯 1 行修复,小项即可。
- **QUIRK-78-02-P2**(DeviceConnectionPool 泄漏):Phase 79/80 让 Core 持有 pool 引用并在 Close 中 `pool.Close()`;需重构 initDeviceServices 把 pool 升到 c.ConnPool 字段。中等重构。
- **P2_RATCHET_internal_core=39.00 豁免行**(check-coverage.sh:242):Phase 81 可删,实测已 82.5%(本 plan 目标 ≥70% 远超)。
- **未覆盖函数缺口**:`initCache` retry 形态(52%) / `initRPAServices` + `initRPAScalingService` + `registerRPATasks` handler 执行体(RPA happy path)/ `initAPIEndpointService` 加载 yaml 成功路径(cwd 限制)。这些非 BLOCK-03 主目标,Phase 79/80 长尾承接。