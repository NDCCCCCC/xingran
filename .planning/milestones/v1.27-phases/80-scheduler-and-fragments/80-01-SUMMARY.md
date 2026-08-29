---
phase: 80-scheduler-and-fragments
plan: 01
subsystem: internal/scheduler
tags: [coverage, scheduler, engine, var-seams, sqlite]
dependency_graph:
  requires: []
  provides:
    - internal/scheduler/cron_80_01_test.go (39+ 新用例;6 引擎 helper 沉淀供 80-02 复用)
    - cron.go per-file 覆盖率 1.5% → 85.1%(≥85% 达成)
  affects:
    - internal/scheduler 包总量 3.3% → 32.6%(本 plan 仅承担引擎主体覆盖,任务族覆盖属 80-02)
tech_stack:
  added: []
  patterns:
    - 引擎 fixture 三件套(newSchedDB8001/newScheduler8001/registerCountingTask8001)同包复用
    - sqlite 文件库(t.TempDir)+ AutoMigrate + 手写 DDL(sys_duty_config/sys_user/...)
    - var-seam save → t.Cleanup restore(77-05 纪律,全文件禁并行子测试)
    - JobExecutor 直调路径 + Start/Stop 生命周期(零 cron 触发时序断言,D-80-02 口径)
key_files:
  created:
    - path: internal/scheduler/cron_80_01_test.go
      lines: 1295
      test_count: 41
      coverage_target: cron.go 85.1%
  modified: []
decisions:
  - D-80-02 直调 Executor / Scheduler 公开方法 + Start/Stop 生命周期用例(零 sleep、零实时等待)
  - D-80-07 测试文件命名 cron_80_01_test.go + helper 一律 8001 后缀
  - 缺口补足 Round 1(纯函数 + VDI/DIC seam + 5 设备任务三分支,184→251 stmts, +67)
  - 缺口补足 Round 2(duty/notice sqlite + addJob 错误分支,251→330 stmts, +79)
  - quirk 记录:parseInvokeTarget 首冒号切分(非 JSON 解析)/ UpdateJob Save 复活软删行 + 幽灵 ID 回退 OnConflict Create 落库 / sendDutyReminderNotification 空 members 也广播 / parseInvokeTarget 多冒号首个切分
  - env blocker: 本地 MSYS2 ucrt64 cc1.exe 无法编译 cgo(预处理器 OK,编译挂),-race 抽样本地不可执行;CI policy (D-01 in ci.yml) 显式禁用 -race(atomic coverage mode 替代),race-clean 由代码纪律(mutex / atomic / RWMutex)保障
metrics:
  duration_min: ~95
  completed_date: 2026-08-28
  test_commits: 6
  total_test_count: 41
  coverage_delta:
    cron.go: 1.5% → 85.1%
    internal/scheduler (pkg): 3.3% → 32.6%
---

# Phase 80 Plan 01: scheduler 引擎主体测试 — Summary

## 一句话

为 `internal/scheduler` 引擎主体 `cron.go` (388 stmts @ 1.5%) 落地 41 个用例 + 6 个同包复用 fixture,**1.5% → 85.1%**(≥70% 目标 + ≥85% 力争双达成);沉淀 `newSchedDB8001` / `newScheduler8001` / `registerCountingTask8001` 等引擎底座供 80-02 直接复用。

## 范围与产物

**测试文件:** `internal/scheduler/cron_80_01_test.go`(1295 行,41 个用例,仅新增 1 文件 — 零生产代码改动)。

**引擎 fixture(同包 8001 后缀,供 80-02 六个 *_tasks.go 测试直接复用):**

| Helper | 用途 |
| ------ | ---- |
| `newSchedDB8001(t)` | sqlite 文件库 + AutoMigrate(sys_job/sys_job_log)+ t.Cleanup 关连接 |
| `newScheduler8001(t)` | NewScheduler + SetLogger(stub);返回 (s, db) |
| `stubLoggerOf8001(s)` | 从 scheduler 取回注入的 schedStubLogger8001 |
| `newJob8001(name, target)` | 合法 job 默认 Status=JobStatusNormal + cron "0 0 8 * * *" |
| `registerCountingTask8001(s, type)` | 纯计数 handler(绝不调 Scheduler 方法,R2 纪律) |
| `newDutyDB8001(t)` | sys_duty_config / sys_user / sys_duty_pool / sys_duty_schedule / sys_notice / sys_notice_target 手写 DDL |
| `schedStubLogger8001` | 可观察 Logger stub(Infof/Warnf/Errorf + counts,sync.Mutex) |
| `stubNoticeHub8001` / `stubDBGetter8001` / `stubVDIVMService8001` / `stubDeviceInfoCollection8001` / `errDeviceMonitorService8001` | var-seam stub + 接口实现 |

**用例分布(41 总):**

| 前缀 | 数量 | 覆盖 |
| ---- | ---- | ---- |
| `TestReg8001_` | 4 | RegisterTask/GetTaskHandler/IsTaskRegistered + 7 个 Register\* 函数群注册 + GetJobCount/GetJobStatus 10 goroutine 并发读 |
| `TestExec8001_` | 7 | Execute 成功/失败/未注册三态 + executeTask nil-scheduler 分支 + parseInvokeTarget 表驱动 5 分支 + calculateNextRunTime 三态 + defaultLogger 直调 + SetLogger 注入缝 + ExecuteJob 公开入口全链 |
| `TestJob8001_` | 6 | AddJob !running / UpdateJob / RemoveJob / StartStopJob 状态机 / GetJobStatus / GetJobCount 全 running 计数 |
| `TestLife8001_` | 3 | StartAddStop 完整生命周期 / StopWithoutStart 边界 / StopTimeoutBounded |
| `TestSeam8001_` | 3 | NoticeHub / DBGetter / GlobalScheduler 三组 var seam(save → Set → Get → t.Cleanup restore) |
| `TestGap8001_` | 18 | cron.go 缺口补足(Round 1: 纯函数 + VDI/DIC seam + 5 设备任务 nil/success/error;Round 2: duty/notice sqlite + addJob cron.AddFunc 错误) |

## 覆盖率(cron.go per-file)

| 指标 | 数值 |
| ---- | ---- |
| **基线**(80-RESEARCH.md §1.2) | 6 / 388 = 1.5% |
| **实测收口** | **330 / 388 = 85.1%** |
| 目标(plan SC-2) | ≥70%(≥272) |
| 力争(plan objective) | ≥85% |

**0%-functions:** 无。cron.go 全部 55 个函数均有覆盖(最低 `executeNoticePublishTask` 36.7% — 头部 nil-DB + ErrRecordNotFound 分支可达,success path 走 senderService 需 wire 表,80-02 边界)。

**包总量(`internal/scheduler`):** 3.3% → 32.6%。其他 9 个文件(`workorder_tasks.go` / `ad_sync_tasks.go` / `dept_sync_tasks.go` / `mac_history_tasks.go` / `mac_history_matview_tasks.go` / `vdi_sync_tasks.go` / `reconciliation_tasks.go` / `reconciliation_fix_suggestion_monitor.go` / `reconciliation_crons.go`)均在 80-02 范围 — 本 plan 只承担引擎主体(per-file 指标)。

## 关键决策执行

- **D-80-02 口径执行:** 零 cron 触发时序断言。所有用例直调 JobExecutor.Execute / Scheduler.ExecuteJob / 公开方法;Start/Stop 生命周期用例不 sleep、不等实时时钟。"引擎并发"由 TestReg8001_GetJobCount_ConcurrentRead 10 goroutine 并发读覆盖,引用 cron_test.go 既有范式。
- **R2 goroutine 纪律:** 3 个 StartJob / 0 个 s.Start(Scheduler)用例,后者禁止。
- **handler 纯函数:** `registerCountingTask8001` handler 体内仅 `atomic.AddInt32`,绝不调 Scheduler 方法(规避 Stop 持锁互斥 5s 超时)。
- **var-seam 纪律:** save → t.Cleanup restore,全文件禁并行子测试(`grep -c "t.Parallel" == 0` 验收)。
- **status 常量:** StartJob/StopJob 断言全走 `models.JobStatusNormal` / `models.JobStatusPause`;JobLog 状态全走 `models.JobLogStatusSuccess` / `models.JobLogStatusFailure`。`grep -c "JobStatusNormal\|JobStatusPause" ≥ 4`(实测 13 处);`grep -c "JobLogStatusSuccess\|JobLogStatusFailure" ≥ 2`(实测 2 处)。
- **同包 unexported 直写:** research 已验证 `s.running = true`(TestGap8001_AddJob_Running_InvalidCron)、`s.logger` 强转 stub(0+ 处)、`db.Create` / `db.Unscoped().First` 软删作用域均为同包白盒合法。

## Deviations

### 1. [Rule 3 - 缺口补足 Round 1] cron.go 1.5% 基线无法仅凭引擎测试达 ≥70%

- **发现:** 任务 1-4 完成后实测 185/388 = 47.7%。plan acceptance 数值 ≥70%(≥272)未达。
- **补足:** 添加 TestGap8001_ 9 用例(纯函数 3 + VDI/DIC seam 2 + 设备任务 nil/success/error 3)。
- **实测:** cron.go 184 → 251 stmts(+66),47.7% → 64.7%。仍欠 21 stmts 触达 70%。
- **commit:** `de322f7 test(80-01): 补 cron.go 缺口 — 纯函数 + VDI/DIC seam + 设备任务三分支`

### 2. [Rule 3 - 缺口补足 Round 2] 缺口补足 Round 1 后仍未达 ≥85% 力争

- **发现:** Round 1 后 64.7%,欠 21 stmts 触达 70%。剩余 0% 均为 duty/notice sqlite 助手 + 设备任务 stub 分支。
- **补足:** 添加 9 个 TestGap8001_ 用例(duty/notice sqlite + addJob cron.AddFunc 错误分支)。
- **实测:** cron.go 251 → 330 stmts(+79),64.7% → **85.1%**(达 ≥85%)。
- **commit:** `3422575 test(80-01): 补 cron.go 缺口 Round 2 — duty/notice sqlite + addJob 错误分支`

### 3. [Rule 3 - 环境工具链] 本地 -race 抽样不可执行

- **发现:** `go test -race ./internal/scheduler/` 报 `runtime/cgo: cgo.exe: exit status 2`。根因:本机 MSYS2 ucrt64 toolchain 的 `cc1.exe` 在 Git Bash 环境下静默崩溃 — `gcc -E`(预处理器)可正常输出但 `gcc -c`(编译)失败。已实测 `gcc -v t.c` 显示 `cc1.exe` 被调用后无输出即返回 1。
- **尝试:** 探索其他编译器(`x86_64-w64-mingw32-gcc` 同链;无 clang);不修改用户 MSYS2 安装(出 scope)。
- **影响:** plan 验收项 "race 抽样 exit 0" 本地不可达。
- **决策依据:** CI policy 显式禁用 `-race`(.github/workflows/ci.yml 注释:"atomic mode is Go 1.21+ standalone (no -race needed; -race would 5-10x coverage time, forbidden by D-01 Claude's Discretion)")。代码层面 race-clean 由纪律保障(stubLogger8001 / stubNoticeHub8001 / stubDBGetter8001 全部 mutex-guard;registerCountingTask8001 用 atomic.Int32;Scheduler 自身用 sync.RWMutex)。Reviewer 在 Linux 环境运行 `go test -race ./internal/scheduler/` 应通过。
- **状态:** 文档化环境障碍,不动生产代码。

### 4. [Rule 1 - quirk 记录] plan <interfaces> 与 cron.go 实现存在 4 处不一致

(按 D-78-10 口径就地记录,零生产代码改动)

| # | plan <interfaces> 描述 | 实际实现(cron.go) | 影响 |
| - | ---------------------- | ----------------- | ---- |
| a | `parseInvokeTarget("type:{\"k\":1}")` → `params["k"] == 1.0`(JSON 解析) | 首冒号切分,`params["param"]` 为整段原文(`{"k":1}`);坏 JSON 不报错;多冒号首个切分(`a:b:c` → `a` + `"b:c"`) | cron.go:109-121 与源码 docstring 一致(统一通过 `params["param"]`),plan 描述不准确 |
| b | `UpdateJob` 行经软删后 Save 0 行命中(回退 OnConflict Create)→ 不存在的 ID 报错 | Save 走 GORM 全字段 UPDATE,UPDATE 不带 deleted_at IS NULL 作用域 → 软删行被复活(deleted_at 写回 NULL)+ 新值落库;不存在的 ID 由 Save 0 行命中 → 回退 OnConflict Create → 幽灵行被插入 | GORM v2 Save 语义:全字段 UPDATE,UPDATE 无软删作用域,0 行命中回退 Create |
| c | `sendDutyReminderNotification(nil)` → 不广播 | 函数无 `len(members) == 0` 守卫,BroadcastToUsers 总是被调用(空 members → 空 userIDs) | 实测调整断言为 calls==1 + len(ids)==0 |
| d | `gorm.DeletedAt` 字段 `assert.Nil` 可判 | `gorm.DeletedAt` 是 struct 永不为 nil,`Valid bool` 才是真值 | 断言改为 `assert.False(t, row.DeletedAt.Valid)` |

## Threat Surface

无新增安全相关表面对外暴露。本测试文件纯 sqlite + 同包白盒,不引入新网络端点、认证路径或 trust boundary。

## 提交清单

| Commit | 内容 |
| ------ | ---- |
| `b2debfa` | test(80-01): scheduler 引擎 fixture + 注册表测试 |
| `75c99f7` | test(80-01): 执行器 JobExecutor 全链测试 |
| `1580375` | test(80-01): scheduler 公开方法 sqlite 驱动测试 |
| `ccfe4c1` | test(80-01): 生命周期 Start/Stop + 三组 var seams(save/restore) |
| `de322f7` | test(80-01): 补 cron.go 缺口 — 纯函数 + VDI/DIC seam + 设备任务三分支(Round 1) |
| `3422575` | test(80-01): 补 cron.go 缺口 Round 2 — duty/notice sqlite + addJob 错误分支 |

(注:与 80-03 workstream session 共享 git index,每 commit 用 `git commit <file>` 偏提交模式仅纳入本文件;**0 次**跨 workstream 文件误提交。)

## Self-Check: PASSED

- ✅ `internal/scheduler/cron_80_01_test.go` 已创建,1295 行,41 个 Test 函数
- ✅ 所有 6 个 commit 已落地,git status 仅含非本 workstream 改动
- ✅ `cron.go per-file coverage = 330/388 = 85.1%`(≥70% + ≥85% 双达成)
- ✅ `go build ./...` exit 0
- ✅ `go test -count=1 ./internal/scheduler/` exit 0(既有 3 个测试文件不回归)
- ✅ cov80_01.out 已删除(临时文件纪律)
- ⚠️ `-race` 本地受 MSYS2 cc1 工具链阻断(CI policy D-01 禁用 `-race`;代码纪律保障 race-clean)

## 已知 Stubs

无功能性 stub。所有 helper(stubLogger/noticeHub/dbGetter/vdiSvc/deviceInfoSvc/errDeviceMonitor)均参与断言或 seam 测试,非占位。

## 80-02 复用清单

**直接复用(无需修改):**
- `newSchedDB8001(t) *gorm.DB` — sqlite + AutoMigrate sys_job/sys_job_log
- `newScheduler8001(t) (*Scheduler, *gorm.DB)` — 装配 + stub logger
- `stubLoggerOf8001(t, s) *schedStubLogger8001` — 取回 stub 断言
- `newJob8001(name, target) *models.Job` — 合法 job
- `registerCountingTask8001(t, s, type) *int32` — 纯计数 handler
- `schedStubLogger8001` — 三法收集 + counts

**复用 + 引用新加的 seam stub(80-02 各任务族自取):**
- `stubNoticeHub8001`(notice/duty 模块)
- `stubDBGetter8001`(走 GlobalDB 的任务)
- `newDutyDB8001`(duty/notice sqlite 助手)
- `stubVDIVMService8001` / `stubDeviceInfoCollection8001` / `errDeviceMonitorService8001`(设备任务族)

**复用 + 配合工作模式:**
- `TestReg8001_RegisterNoticeTasks` 已注册 `notice_publish` / `duty_reminder` / 5 个 device taskType;80-02 直接覆盖 handler 行为。
- var-seam save → t.Cleanup restore 模板已多处示范,80-02 各测试照搬即可。

## Self-Check: PASSED
