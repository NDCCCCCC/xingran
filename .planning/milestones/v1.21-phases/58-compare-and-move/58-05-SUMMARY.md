---
phase: 58-compare-and-move
plan: 05
subsystem: addomain / ad_push_queue / scheduler
tags: [go, cron, sys-job-ban, sys-config, outbox-lite, drainer, phase-58, sc-1]

# Dependency graph
requires:
  - phase: 58-compare-and-move
    plan: 04
    provides: "compareAndMoveOU 终态分流 + F-A 三处载体回写 ad_dn（与 cron 接线解耦）"
  - phase: 58-compare-and-move
    plan: 03
    provides: "addomain.NewADPushDrainer(db, pool) *ADPushDrainer + DrainOnce(ctx) error（接线的唯一构造形态）"
  - phase: 58-compare-and-move
    plan: 01
    provides: "排水测试基座 + addomain 包测试约定"
  - phase: 57-ad-outbox-lite
    provides: "ad_push_queue 模型 + ADPushQueueActivePredicate + EnsureADPushQueue（database.go 启动挂载点）"
  - phase: 36-ad-account-pool
    provides: "ADSyncScheduler.Start() + getPool()（pitfall 4 防线） + cron 6 段 spec 实例"
provides:
  - "addomain.LoadDrainIntervalSeconds(db) int —— 启动读 sys_config.ad_push_queue.drain_interval_seconds（垃圾值兜底 30s）"
  - "addomain.SecondsToCronSpec(seconds int) (string, bool) —— 6 段 spec 映射 + 非法值回落"
  - "migrations.Migrate216ADPushDrainSysConfig(db) error —— 幂等 seed（不覆盖管理员改动）"
  - "internal/scheduler.(*ADSyncScheduler).Start() 第三个 AddJob 驱动（AddJob + NewChain + FuncJob，**禁 sys_job**）"
  - "internal/scheduler.cronLogger() cron.Logger —— applogger 适配器"
affects: [58-06, 58-07]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "AddJob + NewChain(cron.Recover, cron.SkipIfStillRunning).Then(cron.FuncJob)：AddFunc 不暴露 chain wrapper 入口，故 30s 短间隔必须走 AddJob"
    - "SkipIfStillRunning 的互斥是闭包私有 channel，仅防 cron 路径；与手动触发路径共用互斥的事实源仍是 ADPushDrainer.mu（TryLock）"
    - "sys_config 单旋钮 + 启动读取 + 改后重启生效（network.connection_pool.* 先例）"
    - "seed 迁移幂等 + 非阻断：Count > 0 提前 return；缺旋钮只回落默认 30s，不阻断业务"
    - "SecondsToCronSpec 行为断言：合法 spec 必须额外被 cron parser 解析成功（不只字面比对）"

key-files:
  created:
    - internal/core/db/migrations/migration_216_adpush_drain_sysconfig.go
    - internal/core/db/migrations/migration_216_adpush_drain_sysconfig_test.go
  modified:
    - internal/services/addomain/ad_push_queue_drain_service.go
    - internal/services/addomain/ad_push_queue_drain_test.go
    - internal/core/db/database.go
    - internal/scheduler/ad_sync_tasks.go

key-decisions:
  - "D-05 单旋钮仅进 sys_config 一项（ad_push_queue.drain_interval_seconds，默认 30s），与 connection_pool 两项旋钮同档（web 可改、改后重启）"
  - "迁移 216 在 PG / sqlite / SKIP_AUTOMIGRATE 旁路三路径均注册一次（Phase 57 F-03 同款盲区：旁路曾漏建表致全线 500）"
  - "种子失败为非阻断档（与 ensureADPushQueue 的阻断档区分：缺队列表会让入队 ON CONFLICT 42P10 致业务事务回滚，缺旋钮只让频率回落默认 30s）"
  - "Cron 接线只许调 s.getPool()，**禁**在接线处自建账号池（pitfall 4：独立缓存 → 熔断后仍选到坏账号）"
  - "Pool 计数基线 3 不动（:106 熔断恢复任务局部池 + :125 全局单例 + getPool() 惰性兜底）—— 删任一处会破坏既有路径，把断言写成 ==1/==2 同样属于破坏"
  - "本 plan 标记 QUEUE-03 为 complete（cron 接线落地、禁 sys_job 静态证伪、spec 校验闭环）；QUEUE-04/QUEUE-05 沿用分档口径（端到端证据归 58-07，最终回填归 58-07）"

requirements-completed: ["QUEUE-03"]

# Metrics
duration: 40min
completed: 2026-10-19
---

# Phase 58 Plan 05: cron 接线 + sys_config 单旋钮 + 禁 sys_job 静态证伪 Summary

**D-05 单旋钮三层齐备（migration_216 幂等 seed + 启动读取 + SecondsToCronSpec 合法值映射 / 非法值回落 + 行为断言可被 cron parser 解析），cron 接线第三个 AddJob 落生产（AddJob + NewChain + FuncJob，cron 日志接入 applogger，账号池经 s.getPool() 字面取单例，**禁注册 sys_job** 静态证伪 0 行）**

## Performance

- **Duration:** ~40 min
- **Started:** 2026-10-19T11:00Z（近似）
- **Completed:** 2026-10-19T11:35Z（近似）
- **Tasks:** 2/2
- **Files modified:** 4（2 新建 + 2 增量）+ ad_push_queue_drain_test.go 增量用例 7 个

## 交付契约（下游 58-06/58-07 直接消费）

```go
// internal/services/addomain/ad_push_queue_drain_service.go
func LoadDrainIntervalSeconds(db *gorm.DB) int
func SecondsToCronSpec(seconds int) (string, bool)

// internal/core/db/migrations/migration_216_adpush_drain_sysconfig.go
func Migrate216ADPushDrainSysConfig(db *gorm.DB) error

// internal/scheduler/ad_sync_tasks.go
type apploggerCronLogger struct{}
func cronLogger() cron.Logger
```

### `SecondsToCronSpec` 映射表（逐行核对 robfig/cron/v3@v3.0.1 源码）

| 输入 | 语义 | 输出 spec | ok | 行为断言（cron parser 可解析） |
|------|------|----------|----|------------------------------|
| 1 | 秒级每 1 秒 | `*/1 * * * * *` | true | ✅ |
| 30 | 秒级每 30 秒（D-05 默认） | `*/30 * * * * *` | true | ✅ |
| 60 | 分钟级每 1 分钟 | `0 */1 * * * *` | true | ✅ |
| 120 | 分钟级每 2 分钟 | `0 */2 * * * *` | true | ✅ |
| 300 | 分钟级每 5 分钟（域控压力大时的逃生门） | `0 */5 * * * *` | true | ✅ |
| 45 | **非 60 因子**（F-E 陷阱：硬拼 `*/45` 会退化成 `{0,45}`） | `""` | **false** | —— |
| 90 | 非 60 因子 | `""` | **false** | —— |
| 100 | 非 60 因子 | `""` | **false** | —— |
| 0 | 非正数 | `""` | **false** | —— |
| -1 | 负数 | `""` | **false** | —— |

**关键陷阱**：秒位 `0/N`（如 `0/30`）的 start=0,end=0 → 每分钟**只触发一次**，不是"每 30 秒"。**必须**用 `*/N` 而不是 `0/N`。本实现的 `60 % s == 0` 守卫把 1-59 之间的所有合法因子值（1/2/3/4/5/6/10/12/15/20/30）放行，其余 45/50 等非因子值与 0/负数一律返回 ok=false。

### `LoadDrainIntervalSeconds` 全文（`ad_push_queue_drain_service.go:533-552`）

```go
const drainIntervalSysConfigKey = "ad_push_queue.drain_interval_seconds"
const defaultDrainIntervalSeconds = 30

// LoadDrainIntervalSeconds 启动时读取排水间隔（秒）。读不到/解析失败/非正数 → 回落 30。
// 形状照抄 core.go:44-79 loadConnectionPoolConfig 的 readInt 闭包。
func LoadDrainIntervalSeconds(db *gorm.DB) int {
    const fallback = defaultDrainIntervalSeconds
    var v string
    if err := db.Raw(
        "SELECT config_value FROM sys_config WHERE config_key = ? AND deleted_at IS NULL",
        drainIntervalSysConfigKey,
    ).Scan(&v).Error; err != nil || v == "" {
        applogger.Warnf("[ad_push_queue] 读取 %s 失败(%v)，回落默认 %ds", drainIntervalSysConfigKey, err, fallback)
        return fallback
    }
    n, err := strconv.Atoi(v)
    if err != nil || n <= 0 {
        applogger.Warnf("[ad_push_queue] %s 值非法(%q)，回落默认 %ds", drainIntervalSysConfigKey, v, fallback)
        return fallback
    }
    applogger.Infof("[ad_push_queue] 排水间隔配置: %s = %ds（改后需重启后端生效）", drainIntervalSysConfigKey, n)
    return n
}
```

### Migration 216 全文

```go
func Migrate216ADPushDrainSysConfig(db *gorm.DB) error {
    log.Println("Running migration 216: ad_push_queue drain interval sys_config seed")
    const (
        seedKey    = "ad_push_queue.drain_interval_seconds"
        seedValue  = "30"
        seedName   = "AD推送队列-排水间隔(秒)"
        seedRemark = "ad_push_queue 排水器轮询间隔(秒);建议 30,域控压力大时可放宽到 300;修改后需重启后端生效"
    )
    var existingCount int64
    if err := db.Model(&models.Config{}).Where("config_key = ?", seedKey).Count(&existingCount).Error; err != nil {
        return err
    }
    if existingCount > 0 { // 幂等：已存在则跳过（管理员可能已改过值，绝不覆盖）
        return nil
    }
    cfg := &models.Config{
        ConfigName:  seedName,
        ConfigKey:   seedKey,
        ConfigValue: seedValue,
        ConfigType:  models.ConfigTypeNo,
        IsSystem:    models.ConfigIsSystemNo,
        Remark:      seedRemark,
    }
    if err := db.Create(cfg).Error; err != nil {
        log.Printf("Migration 216: create config %s failed: %v", seedKey, err)
        return nil // 非阻断：回落默认 30s 即可
    }
    applogger.Infof("[迁移] sys_config seed: %s = %s", seedKey, seedValue)
    return nil
}
```

### Database.go 三路径注册（行号与上下文）

| 路径 | 行号 | 上下文 |
|------|------|--------|
| PG 分支 | 1058-1066 | `d.ensureADPushQueue()` 之后（advisory-lock 块内，与 215 同步执行） |
| sqlite 分支 | 1102-1106 | `d.ensureADPushQueue()` 之后 |
| BootstrapMissingTables() SKIP_AUTOMIGRATE 旁路 | 1280-1284 | `d.ensureADPushQueue()` 之后（Phase 57 F-03 同款盲区） |

三处错误档位统一为 `applogger.Errorf` 非阻断；缺旋钮只让 `LoadDrainIntervalSeconds` 回落默认 30s。

### `Start()` 接线全文（`ad_sync_tasks.go:173-213`）

```go
// Phase 58 D-05: ad_push_queue 排水器（SC1：in-process cron）。
// ⚠ 里程碑锁定项：**禁注册 sys_job** —— 不写 seed、不加 RegisterTask、不建独立调度器。
//
// ⚠ 必须用 AddJob 而非 AddFunc：AddFunc 不暴露 chain wrapper 入口，
// 而 30s 短间隔下单轮可能超 30s，需要 SkipIfStillRunning 防轮次重叠。
// ⚠ SkipIfStillRunning 的互斥 channel 是**闭包私有**的，无法被手动触发路径复用 ——
//   互斥的事实源是 ADPushDrainer.mu（TryLock），chain wrapper 只是第二道保险。
// ⚠ 账号池必须经 s.getPool() 取全局单例，**禁止**在此处自建账号池
//   （Pitfall 4：独立缓存 → 熔断后仍选到坏账号；同 dept_sync_tasks.go:16-21 的约束）。
// ⚠ sys_config 旋钮在**启动时读取一次**（D-05：改后需重启后端生效）。
//
// ⚠ s.db 为 nil（仅测试构造）时跳过配置读取，保持安全默认 spec ——
//   LoadDrainIntervalSeconds 假定 db 非 nil（同 core.go 的 loadConnectionPoolConfig 先例）。
drainSpec := "*/30 * * * * *"
if s.db != nil {
    drainInterval := addomain.LoadDrainIntervalSeconds(s.db)
    spec, ok := addomain.SecondsToCronSpec(drainInterval)
    if !ok {
        applogger.Warnf("[ad_push_queue] drain_interval_seconds=%d 无法映射为 cron spec（非 60 因子值），回落默认 30s",
            drainInterval)
    } else {
        drainSpec = spec
    }
}
lg := cronLogger()
_, err = s.cron.AddJob(drainSpec,
    cron.NewChain(cron.Recover(lg), cron.SkipIfStillRunning(lg)).Then(cron.FuncJob(func() {
        // 互斥与超时由 Drainer 内部统一处理（TryLock + 每轮一次 now）
        drainer := addomain.NewADPushDrainer(s.db, s.getPool())
        if err := drainer.DrainOnce(context.Background()); err != nil {
            applogger.Errorf("[ad_push_queue] 排水轮次失败: %v", err)
        }
    })))
if err != nil {
    // 非阻断：排水器启动失败不应阻断已有的全量同步与熔断恢复
    applogger.Errorf("[ad_push_queue] 启动排水定时任务失败（不阻断其它 AD 定时任务）: %v", err)
}
```

启动日志扩展为 `applogger.Infof("AD同步定时任务已启动（全量同步: 每5分钟 + 账号池恢复: 每5分钟 + ad_push_queue 排水: %s）", drainSpec)`。

### 启动日志样例（手动触发用 `godotenv + config.Load` 接 DB 模拟）

```
INFO [ad_push_queue] 排水间隔配置: ad_push_queue.drain_interval_seconds = 30s（改后需重启后端生效）
INFO AD同步定时任务已启动（全量同步: 每5分钟 + 账号池恢复: 每5分钟 + ad_push_queue 排水: */30 * * * * *）
```

非法值（如 `45`）启动日志：

```
WARN [ad_push_queue] drain_interval_seconds=45 无法映射为 cron spec（非 60 因子值），回落默认 30s
INFO AD同步定时任务已启动（全量同步: 每5分钟 + 账号池恢复: 每5分钟 + ad_push_queue 排水: */30 * * * * *）
```

## SC1 静态证伪（**grep 输出 0 行**）

```bash
$ grep -rn "ad_push_queue" internal/scheduler/*.go internal/core/db/migrations/*.go | grep -i "sys_job" | wc -l
0
```

—— 排水器**未**注册 sys_job，**未**写 seed、**未**加 RegisterTask、**未**暴露 HTTP 端点。

## Task Commits

Each task was committed atomically:

1. **Task 1: D-05 单旋钮 + migration_216 三路径注册** — `721d2890` (feat)
2. **Task 2: cron 接线（第三个 AddJob）+ 禁 sys_job 静态证伪** — `1bcd3106` (feat)

## 验收断言实测（全部通过）

### Task 1

| 断言 | 结果 |
|------|------|
| `go build ./...` | exit 0 |
| `go test ./internal/core/db/migrations/ -run "TestPhase58_Migration216" -v -count=1` | 1/1 PASS（含管理员改值后迁移不覆盖断言） |
| `go test ./internal/services/addomain/ -run "TestPhase58_SecondsToCronSpec\|TestPhase58_LoadDrainIntervalSeconds" -v -count=1` | 3/3 PASS（LoadDrainIntervalSeconds 5 子用例全过） |
| `grep -c "Migrate216ADPushDrainSysConfig(d.DB)" internal/core/db/database.go` | **3**（PG + sqlite + BootstrapMissingTables 三路径） |
| `grep -rn "ad_push_queue" internal/core/db/migrations/*.go internal/scheduler/*.go \| grep -i "sys_job"` | **0 行**（P14 SC1 证伪） |
| `grep -c "func SecondsToCronSpec(seconds int) (string, bool)" ...` | 1 |
| `grep -c "func LoadDrainIntervalSeconds(db \*gorm.DB) int" ...` | 1 |
| `grep -c "func Migrate216ADPushDrainSysConfig(db \*gorm.DB) error" ...` | 1 |
| `go test ./internal/core/db/migrations/ ./internal/services/addomain/ -count=1` | exit 0 |

### Task 2

| 断言 | 结果 |
|------|------|
| `go build ./...` | exit 0 |
| `go vet ./internal/scheduler/` | 无输出 |
| `go test ./internal/scheduler/ -count=1` | exit 0（含 `ad_sync_tasks_test.go` / `dept_sync_tasks_80_02_test.go`） |
| `grep -rn "ad_push_queue" internal/scheduler/*.go internal/core/db/migrations/*.go \| grep -i "sys_job"` | **0 行**（SC1 核心） |
| `grep -c "s.cron.AddJob(drainSpec" internal/scheduler/ad_sync_tasks.go` | 1（用 AddJob 而非 AddFunc） |
| `grep -c "cron.NewChain(cron.Recover(lg), cron.SkipIfStillRunning(lg))" ...` | 1 |
| `grep -c "addomain.NewAccountPool" internal/scheduler/ad_sync_tasks.go` | **3**（与改动前基线一致：`:106` 熔断恢复任务局部池 + `:125` 全局单例 + `getPool()` 惰性兜底） |
| `grep -c "addomain.NewAccountPool(s.db, nil)" ...` | 1（把 `getPool()` 逐字锁死） |
| `grep -c "addomain.NewADPushDrainer(s.db, s.getPool())" ...` | 1（第三个实参字面取全局池） |
| `grep -c "addomain.NewAccountPool(globalADSyncScheduler.db, nil)" ...` | 1（熔断恢复任务局部池仍在） |
| `grep -c "globalADSyncScheduler.pool = addomain.NewAccountPool(db, nil)" ...` | 1（全局单例仍在） |
| `grep -c "addomain.SecondsToCronSpec(drainInterval)" ...` | 1（含 `!ok` 的 WARN 回落分支） |
| `grep -c "func cronLogger() cron.Logger" ...` | 1 |
| `grep -c "RegisterTask(" internal/scheduler/ad_sync_tasks.go` | 2（与改动前一致，未新增 sys_job 风格注册） |
| `go test ./internal/services/system/ -run TestPhase57 -count=1`（sentinel） | ok |

### 采样命令（phase 约束）

```
go build ./...                                                                          → exit 0
go test ./internal/scheduler/ -count=1                                                  → ok (10.0s)
go test ./internal/core/db/migrations/ -run "TestPhase58_Migration216" -count=1         → ok
go test ./internal/services/addomain/ -run "TestPhase58_SecondsToCronSpec|TestPhase58_LoadDrainIntervalSeconds" -count=1
                                                                                        → ok
go test ./internal/services/system/ -run TestPhase57 -count=1（sentinel）                → ok
```

## Test Results

```
go build ./...                                                                → exit 0
go test ./internal/core/db/migrations/ -count=1                               → ok (1.7s)
go test ./internal/services/addomain/ -count=1                               → ok (7.8s)
go test ./internal/scheduler/ -count=1                                        → ok (10.0s)
go test ./internal/services/system/ -run TestPhase57 -count=1（sentinel）     → ok
go test ./...                                                                 → 全仓绿（`TestAuth8003_LoginLocalDirect_WrongPassword` 异步日志在满载下偶发超时，单独跑 exit 0；非本 plan 回归）
go vet ./internal/scheduler/ ./internal/services/addomain/ ./internal/core/db/migrations/
                                                                              → 无输出
```

## Decisions Made

1. **D-05 单旋钮仅进 sys_config 一项**（`ad_push_queue.drain_interval_seconds`）—— 见 Deviations #1。
2. **迁移 216 三路径注册** —— PG / sqlite / SKIP_AUTOMIGRATE 旁路（Phase 57 F-03 同款盲区）。
3. **Pool 计数基线 3 不动**（`:106` 熔断恢复任务局部池 + `:125` 全局单例 + `getPool()` 惰性兜底）—— 见 Deviations #2。
4. **`s.db == nil` 守卫**（仅测试构造）—— 见 Deviations #3。
5. **本 plan 标记 QUEUE-03 为 complete**（cron 接线 + 禁 sys_job 静态证伪 + spec 校验闭环），QUEUE-04/QUEUE-05 沿用分档口径由 58-07 回填。
6. **规整 SC1 注释措辞** —— 注释本身不能与 `ad_push_queue` 同行出现 `sys_job` 字面（否则 plan 自己的 grep 证伪会因注释自指假阳），见 Deviations #4。

## Deviations from Plan

**1. [Rule 2 - Auto-add critical] 注释不能让 plan 自己的 SC1 grep 证伪失败**

- **Found during:** Task 2（写完接线后第一次跑 `grep -rn "ad_push_queue" ... | grep -i "sys_job"`）
- **Issue:** plan 给出的注释模板把 `ad_push_queue` 与 `sys_job` 写在同一行（如「ad_push_queue 排水器（SC1：in-process cron，**禁注册 sys_job**」），会让 SC1 的证伪 grep（输出应 == 0）被注释自指命中。**这是 plan 内部的 grep 自指矛盾**，与 58-02/58-03/58-04 同样性质的"plan 自己写的可证伪断言"处置。
- **Fix:** 把 `禁注册 sys_job` 单独换行，保留语义（**禁注册 sys_job** —— 不写 seed、不加 RegisterTask、不建独立调度器）。其它威胁登记项、决策注释同理，**禁**与 `ad_push_queue` 同行的 `sys_job` 字面。复跑 `grep ... | grep -i "sys_job" | wc -l` == 0。
- **Files modified:** `internal/scheduler/ad_sync_tasks.go`
- **Committed in:** `1bcd3106`

**2. [Rule 2 - Auto-add critical] 注释里不能出现 `addomain.NewAccountPool` 字面（保持 grep 计数 == 3）**

- **Found during:** Task 2（写完接线后第一次跑 `grep -c "addomain.NewAccountPool" ad_sync_tasks.go`）
- **Issue:** 注释模板「**禁止** addomain.NewAccountPool（...）」把字面量写进注释，命中数从 3 变成 4 → 让 plan 自己的 pitfall 4 防线断言失败。**plan 内部矛盾**：plan 要求「**禁**在接线处 `addomain.NewAccountPool`」作为行为约束，但同时要求 grep 计数 == 3 —— 这两条只有"注释里也不能出现该字面"才能共存。
- **Fix:** 改写为「**禁止**在此处自建账号池（必须复用全局单例）」，行为约束不变（写代码层面），注释字面不再含 `addomain.NewAccountPool` → 命中数 3。复跑 `grep -c "addomain.NewAccountPool" ad_sync_tasks.go` == 3，行号 `106 / 125 / 455`（`getPool()` 惰性兜底的位置，基线 `:412` 实际位移）。
- **Files modified:** `internal/scheduler/ad_sync_tasks.go`
- **Committed in:** `1bcd3106`

**3. [Rule 3 - Blocking] `s.db == nil` 守卫（仅测试构造）**

- **Found during:** Task 2（`go test ./internal/scheduler/ -count=1` panic at `ad_sync_tasks.go:183`）
- **Issue:** 既有的 `TestADSyncScheduler_StartStop` / `TestADSyncScheduler_ParallelStartStop` 调 `NewADSyncScheduler(nil, 5)` 然后 `Start()`，`LoadDrainIntervalSeconds(nil)` 在 `db.Raw(...)` 处 nil 解引用 panic。**Phase 58 之前这些测试不依赖 DB**，因为 `Start()` 只调 `AddFunc`（无 DB 依赖）。`AddJob` 前的 `LoadDrainIntervalSeconds` 把 DB 依赖前置到了 Start 时点。
- **Fix:** 在 `Start()` 内 `LoadDrainIntervalSeconds` 之前加 `if s.db != nil { ... }` 守卫（默认 `drainSpec = "*/30 * * * * *"`）。生产路径（`StartADSyncScheduler(db)` 永远非 nil）语义不变；nil-db 测试构造保持原样。**没有弱化任何验证断言**（spec 计算、AddJob 注册、cron 启动日志全保留）。
- **Files modified:** `internal/scheduler/ad_sync_tasks.go`
- **Committed in:** `1bcd3106`

**4. [Rule 1 - 计划内部矛盾] plan 模板的"ad_push_queue + sys_job 同行"自指（见 #1）**

见 Deviations #1 的同类处置口径。

---

**Total deviations:** 4 auto-fixed（2 处 Rule 2 注释自指 + 1 处 Rule 3 blocking + 1 处 Rule 1 与 #1 同一矛盾的不同侧面）
**Impact on plan:** 均未降低任何验收断言的强度；两处注释字面矛盾以"plan 自己写的可证伪断言"为准裁决（与 58-02/58-03/58-04 同款处置口径）。

## Issues Encountered

- **`gsd-sdk` CLI 在本执行环境不可用**（沿用 58-01~58-04 结论，`NO_GSD_SDK`）。STATE.md / ROADMAP.md / VALIDATION.md / REQUIREMENTS.md 的进度回填按现有结构手工完成，未调用 `state.advance-plan` 等 SDK 动词。
- 工作区存在大量**与本 plan 无关**的未暂存 `.planning/phases/*` 删除（执行前既有状态）。全程只用 `git add <具体文件>` 逐文件暂存。
- 全量 `go test ./...` 偶发 `TestAuth8003_LoginLocalDirect_WrongPassword` 超时（CPU 满载下异步写日志的 poll 超 7.99s）；单独跑 `go test ./internal/api/v1/ -run "TestAuth8003_LoginLocalDirect_WrongPassword" -count=1` exit 0.32s 通过。**非本 plan 回归**（不依赖 scheduler / drainer / 任何 addomain 代码）。

## Known Stubs

无。本 plan 交付的全部符号均被生产代码或测试真实引用：

| 符号 | 引用点 |
|------|--------|
| `LoadDrainIntervalSeconds` | `internal/scheduler/ad_sync_tasks.go:188`（生产）+ 5 个测试子用例 |
| `SecondsToCronSpec` | `internal/scheduler/ad_sync_tasks.go:189`（生产）+ 5 个合法值 + 5 个非法值用例 |
| `Migrate216ADPushDrainSysConfig` | `database.go:1063 / 1105 / 1283`（三路径生产）+ `TestPhase58_Migration216_SeedIdempotent` |
| `AddJob` 接线 | `Start()` 第三行（生产路径，每 30s 触发 DrainOnce）|
| `cronLogger` | `Start()` 内 `lg := cronLogger()` + `cron.Recover(lg)` / `cron.SkipIfStillRunning(lg)` 两处消费 |

⚠ **预期内但须下游知悉**：`s.db == nil` 守卫下生产的 `drainSpec = "*/30 * * * * *"`（deviations #3）仅在测试构造生效；生产 `StartADSyncScheduler(db)` 路径走 sys_config 真值，行为完全一致。

## Threat Flags

无新增安全相关表面。本 plan 的威胁登记项（T-58-05-01 ~ T-58-05-08）全部以**代码 + 可证伪断言**落实：

| Threat | 落实位置 |
|--------|----------|
| T-58-05-01 DoS（非 60 因子值） | `SecondsToCronSpec` 的 `60%seconds == 0` + `seconds%60 == 0` 守卫 + `TestPhase58_SecondsToCronSpec_InvalidFallsBack`（5 非法值）+ `TestPhase58_SecondsToCronSpec_Valid`（5 合法值含 cron parser 行为断言）|
| T-58-05-02 DoS（轮次重叠） | 三重：①`cron.SkipIfStillRunning`（闭包私有 channel）②`ADPushDrainer.mu.TryLock`（事实源）③单轮 `LIMIT 100`。Plan 03/04 的 `TestPhase58_Drain_MutualExclusionSkipsSecondRound` 锁 TryLock 语义 |
| T-58-05-03 DoS（极小值） | 接受 1 秒配置；启动日志 `applogger.Infof` 印出实际生效值；remark 文本明示「建议 30，可放宽到 300」 |
| T-58-05-04 Elevation（外部触发面） | SC1 + P14 静态证伪（grep 0 行） + `RegisterTask` 计数不变（2） |
| T-58-05-05 Repudiation（cron 日志打 stdout） | `apploggerCronLogger` 把 `cron.Logger.Info/Error` 接到 `applogger.Debugf/Errorf`；前缀 `[cron]` 落进项目日志 |
| T-58-05-06 Tampering（覆盖管理员改动） | 幂等分支 `Count > 0 → return nil` 提前退出，绝不 Update；`TestPhase58_Migration216_SeedIdempotent` 第 3 步断言手工改成 300 后重跑仍为 300 |
| T-58-05-07 DoS（seed 失败阻断启动） | 非阻断档：`Create` 失败仅 `log.Printf` + `return nil`；三路径调用点错误仅 `applogger.Errorf` |
| T-58-05-08 Tampering（独立账号池） | Pitfall 4：`grep -c "addomain.NewAccountPool" == 3`（与改动前基线一致）+ `grep -c "addomain.NewADPushDrainer(s.db, s.getPool())" == 1`（字面取全局池） |

## 状态更新

- **STATE.md**：Current Position 更新为 Phase 58 / Plan 05 完成（Wave 4/6）；Decisions 追加本 plan 的 6 条执行期决策；progress `completed_plans 7→8`、`percent 70→80`；last_activity 改写。
- **ROADMAP.md**：Phase 58 Plans 列表 58-05 标记 complete；Progress Table 行更新为 `5/7 | In progress (wave 4 done 2026-10-19)`。
- **VALIDATION.md**：58-05/T1、T2 两行 `⬜ pending` → `✅ green <commit>`。
- **REQUIREMENTS.md**：**QUEUE-03 → Complete（58-05）**（cron 接线 + 禁 sys_job 静态证伪 + spec 校验闭环三件齐备）；QUEUE-04/QUEUE-05 沿用分档口径由 58-07 回填。

## Next Phase Readiness

- **Wave 5（58-06）可立即开** —— 排水器已经过 in-process cron 真正接入生产面（不再是 Phase 57 既有 `RegisterTask` 风格）；`DrainOnce` 行为面已完整（含 OU 移动、OU 缺失转 dead、互斥事实源），无未了契约。
- **58-06 必须遵守**：
  1. 删 `EnqueueUpdateBatch` 时**必须**保留 F-07 行为（输入 null/全空数组 → 全空入队，不报错），且 `Pluck` 的查询列与 `LastError` / `LastErrorAt` 落库；
  2. 删两条直推路径时**一并清理** 58-04 SUMMARY「Plan 06 调用点清单」的 5 项；
  3. 排水器**不依赖** `moveUserToNewOU` / `moveSingleUserToOU`（`compareAndMoveOU` 是独立实现），删除它们是 06 的安全项。
- **58-07 必须遵守**：
  1. 端到端用例覆盖"入队 → 排水 → compare-and-move → 终态"全链；
  2. `unrecoverable:no-ou-mapping` 的**批量 dead 可见后果**须进人工 UAT 清单（部门同步未跑过的环境）；
  3. 五条 SC 的 grep 断言沿用本 SUMMARY 的表格口径（特别是 `grep -rn "ad_push_queue" ... | grep -i "sys_job" | wc -l` == 0 + `grep -c "addomain.NewAccountPool" == 3`）。
- **无阻塞**：全仓 `go test ./...` exit 0（除已知 CPU 满载下的异步日志 poll flake，单跑 exit 0）；Phase 57 sentinel（`TestPhase57`）绿；`go vet ./internal/scheduler/ ./internal/services/addomain/ ./internal/core/db/migrations/` 无输出。

---

*Phase: 58-compare-and-move*
*Completed: 2026-10-19*

## Self-Check: PASSED
