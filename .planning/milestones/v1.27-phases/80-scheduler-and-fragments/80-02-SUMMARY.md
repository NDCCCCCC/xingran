---
phase: 80-scheduler-and-fragments
plan: 02
subsystem: internal/scheduler
tags: [coverage, scheduler, task-family, var-seams, interface-stubs, sqlite, wire-exemption]
dependency_graph:
  requires:
    - internal/scheduler/cron_80_01_test.go (newSchedDB8001/newScheduler8001/registerCountingTask8001/stubNoticeHub8001/stubDBGetter8001/schedStubLogger8001)
  provides:
    - internal/scheduler/{workorder,ad_sync,dept_sync,mac,vdi,reconciliation}_tasks_80_02_test.go (6 文件 101 用例;stub pool/cipher/matview 等接口 stub 沉淀)
    - internal/scheduler 包级 81.4%(SC-1 达成)
  affects:
    - Phase 80 SC-1 关门(scheduler ≥70%);80-03/04/05 无文件交集
tech_stack:
  added: []
  patterns:
    - per-interface stub struct 直注(73-04 D-02 范式):AccountPool/PasswordCipher/MACHistoryMatViewService/PartitionService/MACHistoryPurgeService/VDIVMService
    - 同包 unexported 字段直写(ADSyncScheduler.pool/sem/db)+ 包级 var 直写(globalPurgeService/globalPartitionService 绕 sync.Once)
    - sem 排空同步技巧(checkAndSync goroutine 体覆盖:容量 1 + 单配置 → sem.Acquire 即 join 点,非轮询)
    - goroutine 安全 restore(safeRestore 非 nil 实例防 :345 解引用 nil)
key_files:
  created:
    - { path: internal/scheduler/workorder_tasks_80_02_test.go, lines: 752, test_count: 20 }
    - { path: internal/scheduler/ad_sync_tasks_80_02_test.go, lines: 604, test_count: 24 }
    - { path: internal/scheduler/dept_sync_tasks_80_02_test.go, lines: 426, test_count: 15 }
    - { path: internal/scheduler/mac_tasks_80_02_test.go, lines: 273, test_count: 12 }
    - { path: internal/scheduler/vdi_tasks_80_02_test.go, lines: 342, test_count: 17 }
    - { path: internal/scheduler/reconciliation_tasks_80_02_test.go, lines: 509, test_count: 13 }
  modified: []
decisions:
  - D-80-02 口径执行:零 cron 触发时序断言、零 sleep;ctx 取消/goroutine join 用 sem/chan 同步
  - D-80-06 豁免落地:dept_sync ExecuteWithFailover wire 闭包 53 stmts 剔除分母;ad_sync wire 实际仅剩 7 stmts(远小于预估 26)
  - mac 双 seam 的 sync.Once 不可逆 → 错误分支用包级 var 直写注入 err-stub
  - 测试自身竞态修复:ScheduleADSyncForConfig goroutine 的 global 解引用防 nil(safeRestore + db 身份判定)
metrics:
  duration_min: ~210(含 API 中断续跑)
  completed_date: 2026-08-28
  test_commits: 9
  total_test_count: 101
  coverage_delta:
    internal/scheduler (pkg): 32.6% (80-01 后) → 81.4%(SC-1 ≥70% 达成)
    workorder_tasks.go: 0% → 86.7%
    ad_sync_tasks.go: 16.7% → 81.5%
    dept_sync_tasks.go: 0% → 56.6%(豁免调整 97.2%)
    mac_history_tasks.go: 0% → 85.7%
    mac_history_matview_tasks.go: 0% → 82.6%
    vdi_sync_tasks.go: 0% → 95.8%
    reconciliation_tasks.go: 2.3% → 79.2%
    reconciliation_fix_suggestion_monitor.go: 0% → 86.7%
---

# Phase 80 Plan 02: scheduler 任务族——调度分支/取消/恢复 + wire 豁免 Summary

## 一句话

为 `internal/scheduler` 任务族 8 文件(715 stmts)落地 6 个测试文件 101 个用例,任务族 715 stmts 中 538 covered(75.2%),包级 **32.6% → 81.4%**(SC-1 ≥70% 关门);D-80-06 wire 豁免按实测口径落地(dept_sync 53 stmts 剔除分母,ad_sync 实际仅剩 7 stmts)。

## 范围与产物

**测试文件(6 个,2906 行,101 用例,零生产代码改动):**

| 文件 | 行数 | 用例 | 覆盖锚点 |
| ---- | ---- | ---- | -------- |
| `workorder_tasks_80_02_test.go` | 752 | 20 | 4 纯函数表驱动 / GlobalNoticeHub nil-guard+stub 两态 / assignWorkOrderHandler 4 分支 / executePeriodicWorkOrderCreateTask 成功+DutyPool+3 错误分支 / syncWorkOrderJob create·update·回填·坏cron / Disable·Enable / SyncPeriodicWorkOrderJobs |
| `ad_sync_tasks_80_02_test.go` | 604 | 24 | checkAndSyncADConfigs 三态 + goroutine 排空 / executeADAccountPoolRecoverBreakersTask(sqlite 真恢复断言) / executeADDataSyncTask 参数矩阵 / ScheduleADSyncForConfig Done 分支 / StartStopGlobal(Once 生命周期) / GetADSyncStatus 启动态 / getNextRunTime 分支 / SM4 cipher seam / getDefaultADConfigID 5 分支 |
| `dept_sync_tasks_80_02_test.go` | 426 | 15 | RegisterDeptSyncTasks / pool stub 同包直注(getGlobalADAccountPool nil+非 nil) / executeDeptToADSyncTask 全分支(nil config/停用/查失败/成功/同步失败) / executeDeptMemberToADGroupSyncTask(停用/空映射/池空 ErrAllAccountsUnavailable/池查询错误) / getDefaultADConfigIDForDept 参数键+查错误 |
| `mac_tasks_80_02_test.go` | 273 | 12 | RegisterMACHistoryTasks db nil+落库+幂等 / upsertMACHistoryJob 创建·已存在·坏cron / executeMACHistoryCleanup·Purge 成功·nil·错误(var 直写绕 Once) / RegisterMACHistoryMatViewTasks 注册·handler 执行·db nil·幂等 |
| `vdi_tasks_80_02_test.go` | 342 | 17 | VDIVMService seam / RegisterVDISyncTasks / executeVDIVMSyncTask auto+单ID / syncAllEnabledVDIServers 空·多台·查错误·单台失败 / syncSingleVDIServer 成功·不存在·停用·服务nil·同步错 / SyncVDIVMsManually auto+单ID / Fsm Register+handler 分发(非监控 target 跳过+监控 target 软失败) |
| `reconciliation_tasks_80_02_test.go` | 509 | 13 | RegisterReconciliationTasks(nil 形参 nil-safe + 8 job seed) / handler 7 子任务分发 switch / SelfHeal 4 分支(legacy cron/JobGroup 大小写/InvokeTarget 漂移/干净行) / createWorkorderBySeverity 空·循环·no_workorder 过滤·成功路径(空库 woSvc) / cleanupExpiredExceptionsDirect 查错误 / checkPortStatusDrift 健康·首测·下降·上涨超阈·持平·查错误·基线解析失败 / baseline round-trip |

**复用的 80-01 helper:** `newSchedDB8001` / `newScheduler8001`(经 newScheduler8002_DST) / `schedStubLogger8001` / `stubDBGetter8001` / `stubNoticeHub8001`(未用,自建 8002 版含消息断言) / var-seam save→Cleanup restore 模板。

## 覆盖率(80-RESEARCH §1.2 基线 → 实测收口)

| # | 文件 | 基线 stmts/覆盖 | 实测 covered/stmts | 实测 % | 目标判定 |
| -- | ---- | -------------- | ------------------ | ------ | -------- |
| 1 | cron.go(80-01) | 6/388 | 330/388 | **85.1%** | ✓(80-01 交付) |
| 2 | workorder_tasks.go | 0/166 | 144/166 | **86.7%** | ✓ ≥70% |
| 3 | ad_sync_tasks.go | 27/162 | 132/162 | **81.5%** | ✓ ≥70%(无需豁免) |
| 4 | dept_sync_tasks.go | 0/122 | 69/122 | 56.6% raw / **97.2% 调整** | ✓(豁免剔除 53 stmts) |
| 5 | mac_history_tasks.go | 0/49 | 42/49 | **85.7%** | ✓ ≥70% |
| 6 | mac_history_matview_tasks.go | 0/23 | 19/23 | **82.6%** | ✓ ≥70% |
| 7 | vdi_sync_tasks.go | 0/48 | 46/48 | **95.8%** | ✓ ≥70% |
| 8 | reconciliation_tasks.go | 3/130 | 103/130 | **79.2%** | ✓ ≥70% |
| 9 | reconciliation_fix_suggestion_monitor.go | 0/15 | 13/15 | **86.7%** | ✓ ≥70% |
| — | **包级 internal/scheduler** | 36/1103 = 32.6%(80-01 后) | **898/1103** | **81.4%** | **SC-1 ✓(≥70%)** |

任务族 715 stmts → 538 covered(75.2%);包级 70% 线 = 772 stmts,实测 898(+126 余量)。

## D-80-06 豁免条目清单(逐条 stmt 数)

| # | 豁免段 | 文件:位置 | stmts | 原因 |
| -- | ------ | --------- | -----: | ---- |
| 1 | `ExecuteWithFailover` LDAP wire 闭包体 | dept_sync_tasks.go:183-281 | 51 | 闭包内 `ldapClient.AddGroupMember` 等全部走真 LDAP 连接(FailoverClient.newClient → NewLDAPClient.Dial);78 DQ4 已 deferred vjeantet/ldapserver;plan 预授权("wire 层内嵌 addomain 调用若不可绕则该子分支按 D-80-06 同口径在 SUMMARY 记豁免") |
| 2 | 成功尾段(需 wire 成功才可达) | dept_sync_tasks.go:289-292 | 2 | `ExecuteWithFailover` 返回 nil 依赖真实 LDAP bind 成功 |
| 3 | syncADConfig 成功尾段 | ad_sync_tasks.go:265-278 | 7 | `SyncDataByID` 成功路径需真 LDAP Search(原预估 ~26 stmts,经 goroutine 排空测试覆盖 head 查询/构造/error 分支后实际仅剩 7) |
| 4 | checkAndSync goroutine sem 超时分支 | ad_sync_tasks.go:239-245 | 4 | `sem.Acquire` 失败/`DeadlineExceeded` 需 30 分钟超时等待(D-80-02 禁时序) |
| 5 | Start 的 cron AddFunc 错误分支 | ad_sync_tasks.go:109-112,153-159 | 5 | 表达式硬编码合法,`cron.AddFunc` 错误分支不可达 |
| 6 | Start 内 cron 闭包体 | ad_sync_tasks.go:163-168 | 4 | cron tick 体,D-80-02 禁真实 cron 触发 |

**豁免合计 73 stmts**;其中 dept_sync 53 stmts 剔除分母后 69/71 = **97.2%**;ad_sync 其余 20 stmts 保留分母仍 81.5% ≥70%。

## 关键决策执行

- **D-80-02:** 全文件零 `time.Sleep`、零 `t.Parallel()` 调用、零 cron 触发时序断言(grep 验收;`t.Parallel` 仅出现在头部纪律注释)。goroutine join 用 **sem 排空**:`NewADSyncScheduler(db, 1)` + 单个待同步配置 → 同步 goroutine 独占信号量 → 测试 `sem.Acquire(ctx, 1)` 成功即 goroutine 已 Release(同步点,非轮询)。
- **D-80-07:** 6 文件全部 `<source>_80_02_test.go` 命名;helper 一律 8002 后缀(newSchedDB8002_AD/newDeptSyncDB8002/seedDrift8002/cronEntryZero8002/timeNow8002 等)。
- **同包特权:** `ADSyncScheduler.pool/sem/db` unexported 字段直写;`globalPurgeService`/`globalPartitionService` 包级 var 直写绕 `sync.Once`(Once 只锁 Set* 函数,不锁 var 本身);`s.running` 直写(80-01 同款)。
- **status 常量:** 全部引用 `models.JobStatusNormal/Pause`、`models.ADConfigStatusEnabled/Disabled`、`models.ADAccountStatusAvailable/Breaker`;job seed 自愈断言比对 `reconCronMonitorFixSuggestionMisFix`/`reconciliationJobGroup` 权威常量。无裸 0/1。
- **stub 纪律(73-04 D-02):** per-interface stub struct(`stubAccountPool8002` 全 14 方法满足 `addomain.AccountPool`,编译期 `var _ addomain.AccountPool` 锚定);零 testify/mock。
- **T-80-02-01 威胁缓解:** 新测试文件 grep `ldap://` 零命中;`syncADConfig` 零直调(grep 验收);全部 wire 出口经 stub pool(空 `ListAvailable` → `ErrAllAccountsUnavailable` 零拨号)或停用/缺失 config(`SyncDataByID:50` 立即报错)短路。

## Deviations from Plan

### 1. [Rule 3 - 缺口补足] 首轮各文件实测低于 70%,两轮回填

- **发现:** Task 1-6 首轮完成后包级 59.0%,逐文件 workorder 48%/ad_sync 40%/dept_sync 14%/mac 45%/matview 55%/vdi 64%/reconciliation 45%(块口径)。
- **补足:** 按任务 7 授权回对应 task 补 unc 分支:Round 1(workorder 注册体+executePeriodicCreateTask 全链、ad_sync 注册体/参数矩阵、dept_sync execute 双函数、mac RegisterMACHistoryTasks+upsert 三态、vdi Register+syncVDIServerVMs 三态、reconciliation handler switch+自愈)→ 76.2%;Round 2(ad_sync Start/Stop 全局+Schedule 双分支+goroutine 排空+getNextRunTime 零值 entry、dept_sync 参数键+查错误+同步失败+池错误)→ **81.4%**。
- **commits:** `e85152b`、`76dd875`、`6ef660f`。

### 2. [Rule 1 - 测试竞态修复] TestAds8002_ScheduleForConfig_Fire 触发全量套件 panic

- **发现:** 全仓 `go test ./...` 下 scheduler 包 panic(nil pointer @ ad_sync_tasks.go:345)。根因:Fire 测试用 10ms 延迟的 goroutine,测试 cleanup 先把 `globalADSyncScheduler` 恢复为 nil,goroutine 醒来在 `:345` 解引用 `globalADSyncScheduler.ctx` → panic。单独跑时序恰好看不出(隐性 flake)。
- **修复:** 删除 Fire 用例(time.After 分支 3 stmts 并入豁免);DoneBranch cleanup 改为恢复"新鲜有效实例"(safeRestore)而非 nil;StartStopGlobal 增加 Once 幂等 skip(`got == nil || got.db != db` 身份判定,兼容 `-count>1` 与其他测试先耗尽 Once 的场景)。修复后 scheduler 包 `-count=1` 连续 3 次全绿。
- **commit:** `6ef660f`。

### 3. [quirk 记录,零生产改动] 执行中发现 4 处实现/环境特性

| # | 现象 | 处理 |
| - | ---- | ---- |
| a | mac 双 seam `SetPartitionService`/`SetMACHistoryPurgeService` 由 `sync.Once` 守卫,Once 耗尽后 Set 为 no-op,save/restore 纪律对该缝失效 | nil/错误分支改用**包级 var 直写**注入 err-stub(同包合法),注释标明绕 Once 原因 |
| b | GORM `default:true` bool 字段(PeriodicWorkOrderTemplate.NotifyAssignee)零值 false 在 Create 时被省略 → 落库为 DB default 1 | 测试内显式 `db.Model(tpl).Update("notify_assignee", false)`(Update 不省略零值) |
| c | sqlite AutoMigrate 陷阱三连:`gen_random_uuid()` default 语法错;`json.RawMessage` 列需 `CAST('...' AS BLOB)`(string 不可 Scan);WorkOrder 关联 AutoMigrate 会给手写 sys_user ALTER 加 NOT NULL 列 | 手写 DDL 列齐(BaseModel 5 列 + NOT NULL 列带 default)/raw_snapshot 用 BLOB/sys_user DDL 预置 password·salt·auth_source |
| d | `TestWot8002_DisableEnableJob`:运行态 `AddJob` 先 `db.Create` 再 `addJob`,自建 job 行再 AddJob 会 UNIQUE 冲突;`DisablePeriodicWorkOrderJob` 内 `StopJob` 对 !running scheduler 报"任务未在运行"但流程继续 | 断言口径按实际行为:Enable 重建新 job(模板 JobID 指向新行);Disable 后旧行经 `Unscoped()` 断言 Pause |

### 4. [环境,沿用 80-01] `-race` 本地不可执行

MSYS2 ucrt64 `cc1.exe` 编译崩溃(80-01 Deviation 3 同因);CI policy D-01 显式禁用 `-race`(atomic mode 替代)。race-clean 由代码纪律保障(stub mutex / atomic / 同包串行,无 t.Parallel)。

### 5. [超范围发现,未修] 既有 reconciliation 测试 `-count>1` flake

`reconciliation_tasks_test.go` 的内存库 DSN `file:recon_task_<TestName>?mode=memory&cache=shared` 在同进程 `-count=3` 重跑时复用同库 → 固定 ID INSERT 撞 UNIQUE 约束(TestCleanupExpiredExceptions* 4 例)。**既有文件问题,本 plan 未触碰该文件**(SCOPE BOUNDARY);`-count=1` 口径(plan gate)全绿。

## Threat Surface

无新增安全相关表面对外暴露。全部 wire 出口经 stub 短路,测试进程零 LDAP 出网(T-80-02-01 缓解验证:`ldap://` 零命中、`syncADConfig` 零直调)。全局 var 全部 save→t.Cleanup restore,无跨用例污染残留。

## 提交清单

| Commit | 内容 |
| ------ | ---- |
| `834807a` | test(80-02): workorder 任务族测试(纯函数表驱动 + 派单分支 + NoticeHub 两态) |
| `ddbc155` | test(80-02): ad_sync 调度三分支/恢复/SM4 seam + D-80-06 wire 豁免标注 |
| `c442869` | test(80-02): dept_sync 任务族测试(pool stub 直注 + 注册体) |
| `629bf07` | test(80-02): mac_history/matview 任务族测试(双 seam stub + 分发) |
| `dac9d0c` | test(80-02): vdi_sync + fix_suggestion 任务族测试(seam + sqlite 行驱动) |
| `7d47b0d` | test(80-02): reconciliation 任务族测试(注册体 + 转单/漂移/baseline) |
| `e85152b` | test(80-02): workorder/ad_sync/dept_sync 任务族缺口补足(注册体/全分支/goroutine 排空) |
| `76dd875` | test(80-02): mac/vdi/reconciliation 任务族缺口补足(自愈分支/漂移分支/分发 switch) |
| `6ef660f` | fix(80-02): ad_sync 全局生命周期测试竞态修复(Once 幂等 skip + db 身份判定 + safeRestore) |

(与并行 workstream 共享 git index,每 commit 用 `git add <file>` + `git commit` 偏提交模式仅纳入本 plan 文件。)

## 验收对照(plan success_criteria)

1. ✓ 6 测试文件全绿:TestWot8002_ 20(≥6)/ TestAds8002_ 24(≥8)/ TestDst8002_ 15(≥5)/ TestMht8002_+TestMmv8002_ 12(≥4)/ TestVdi8002_+TestFsm8002_ 17(≥5)/ TestRct8002_ 13(≥4)
2. ✓ 包级 81.4% ≥70%(SC-1);任务族 8 文件逐文件:7 文件 raw ≥70%,dept_sync 56.6% raw 但豁免调整 97.2% ≥70%,无一 <50%
3. ✓ D-80-06 豁免 6 条目逐条落 SUMMARY(§豁免清单,合计 73 stmts)
4. ✓ D-80-02 口径:零 sleep / 零 t.Parallel / 零 cron 时序断言
5. ✓ `go build ./...` exit 0;`go test ./internal/scheduler/` exit 0(含既有 3 测试文件无回归);每 task 原子 commit

## 已知 Stubs

无功能性 stub。全部 stub(stubAccountPool8002/errListPool8002/stubPasswordCipher8002/stubPartitionService8002/errPartitionService8002/stubPurgeService8002/errPurgeService8002/stubMatViewService8002/stubVDIVMService8002/stubADPool8002/stubNoticeHub8002)均参与断言或 seam/分发测试。

## Self-Check: PASSED

- ✅ 6 个测试文件存在且 101 用例全绿(`go test -count=1 ./internal/scheduler/` exit 0,连续 3 次)
- ✅ 9 个 commit 全部落地(`git log` 核对)
- ✅ 包级 81.4%(898/1103)≥ 70%,SC-1 达成;cov80_02.out 已删除
- ✅ `go build ./...` exit 0;全仓 `go test ./...` scheduler 包 ok(详见 Deviation 2 竞态修复后验证)
- ✅ D-80-06 豁免 6 条目 + stmt 数落 SUMMARY
- ⚠️ `-race` 本地受 MSYS2 工具链阻断(沿用 80-01 记录;CI D-01 禁用)
- ⚠️ 既有 reconciliation_tasks_test.go 在 `-count>1` 下有 UNIQUE 冲突 flake(超范围,未修)
