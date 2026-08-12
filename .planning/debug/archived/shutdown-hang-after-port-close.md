---
gsd_state_version: 1.0
slug: shutdown-hang-after-port-close
status: resolved
created: 2026-07-06
updated: 2026-07-06
trigger: 后端关闭流程卡住,关了好半天还在运行 (端口已停但日志还在输出)
---

# Debug Session: shutdown-hang-after-port-close

## Symptoms (gathered 2026-07-06)

### Expected
发送 SIGINT/SIGTERM 后,后端按 `cmd/main.go waitForShutdown` → `srv.Shutdown(ctx, 10s timeout)` → `core.Close()` → `applogger.Close()` 顺序,在合理时间内(<15s)干净退出。

### Actual
**截图实证(用户提供)**:
```
10:17:17 正在关闭服务器...
10:17:17 子进程 reaper 已停止
10:17:17 通知中心已停止
10:17:17 正在停止设备信息采集服务...       ← core.Close() step 3,BLOCKED HERE
[5 秒后,Step 6 Scheduler 仍在跑]
10:17:22 执行任务: 对账-自动转工单high, 目标: reconciliation:createWorkorderHigh
10:17:22 执行任务: 对账-修复建议生成, 目标: reconciliation:generateFixSuggestions
10:17:22 执行任务: 对账-物化视图刷新, 目标: reconciliation:refreshView
10:17:22 [reconciliation:createWorkorderHigh] 完成 (3ms)
10:17:22 [reconciliation:generateFixSuggestions] 完成 (5ms)
[Step 8 RPAScalingService 居然在扩容!]
10:17:23 开始扩容: 当前 2 → 目标 3 (本次 +1)
10:17:23 扩容完成: 创建了 1 个 Worker 容器 [mock-container-1550]
10:17:23 WARN 获取系统指标失败: 无法获取网络统计数据: 执行 wmic网络命令失败: exit status 0xc000013a
10:17:30 [连接池] 移除失败连接实例: aca124c8-..., error: 等待连接空闲超时
10:17:30 设备信息采集任务失败: 设备ID=e61e0625-..., 错误:采集设备信息失败: 获取设备连接失败: 连接池已满: 当前=24, 最大=20
[之后无更多日志,进程不退出,用户报告"关了好半天还在运行"]
```

**端口状态**:已不监听 → srv.Shutdown(10s)已完成 → core.Close() 起步后卡死

### Error Messages
- `0xc000013a` = Windows STATUS_CONTROL_C_EXIT (wmic 被 Ctrl+C 传递)
- `连接池已满: 当前=24, 最大=20` — DB pool 严重过载

### Timeline
- 2026-07-06 10:17:17 — 触发关闭
- 2026-07-06 10:17:17 — DeviceInfoCollectionService.Stop() 调用,**从未完成**
- 2026-07-06 10:17:30 — 连接池满,设备采集任务失败
- 进程仍在(用户报告"好半天还在运行")

### Reproduction
1. 启动后端(连真实 PG + Redis)
2. 等 cron 触发若干对账/物化视图/采集任务
3. Ctrl+C / 发送 SIGINT
4. 观察:10s 后端口停,但进程不退出,日志继续堆(尤其 DeviceInfoCollection + 连接池错误)

### Root Cause Hypothesis (待验证)
- **H1 (主因)**: `core.Close()` 在 step 3 `DeviceInfoCollectionService.Stop()` 上无超时死等 in-flight job,新 cron 任务又被允许触发抢 DB 连接,池满后死锁
- **H2**: `core.Close()` 整体没设 deadline(只有 `srv.Shutdown(ctx, 10s)` 有),`Close()` 内部任何一步阻塞 = 永远不退
- **H3**: `Scheduler.Stop()` 在 step 6 但 DeviceInfoCollectionService.Stop() 在 step 3,**关闭顺序反了** — 应当先停 cron,再停依赖 cron 触发的服务
- **H4 (DB pool)**: `c.DB.Close()` 后续步骤也会卡,因 pool 24/20 的连接很多持有 in-flight 长 query(如 refreshView)
- **H5 (windows)**: `0xc000013a` 显示 wmic 被 Ctrl+C 强杀,Windows 信号传递有问题;但根本问题在 core.Close,跨平台

### Investigation Plan
1. 读 `internal/services/component_collector/device_info_collection_service.go` (实际路径需 grep 确认) 找 Stop() 实现 — 是 done channel? WaitGroup? context?
2. 读 `internal/scheduler/cron.go` 找 Scheduler.Stop() 实现,确认它是否会等 in-flight job
3. 读 `internal/services/asset/reconciliation_*.go` 找 refreshView 是否有 context
4. 读 `internal/core/db/database.go` 找 pool size 配置 (默认 20,被证实)
5. 验证假设 H3 — 关闭顺序: 是否应当 reaper → NoticeHub → Scheduler → ADSync → DeviceInfoCollection → DeviceMonitor → RPAScaling → MetricsCache → Cache → DB
6. 决定 fix 方向:
   - A. 重排 core.Close() 顺序: Scheduler/ADSync 提到 Step 3 之前
   - B. 给整个 core.Close() 加 deadline (e.g. 30s)
   - C. DeviceInfoCollectionService.Stop() 加 timeout
   - D. DB pool 容量治理 — 关闭期间禁止新 query
   - 推荐 A+B+C 组合

### Current Focus
- **status**: root_cause_found + fix_applied + self_verified
- **completed**: 2026-07-06
- **fixes_applied**:
  1. `internal/core/core.go` — 重排 Close 顺序(Scheduler+ADSync 提前)+ 30s 总 deadline 兜底
  2. `internal/services/device_info_collection_service.go` — Stop 加 8s 内部 timeout(wg.Wait 改 select+time.After)
  3. `internal/services/device_info_collection_service_test.go` — 新增 2 个 Stop 回归测试
- **verification**:
  - `go build ./...` — 编译通过
  - `go vet ./internal/core/... ./internal/services/...` — 无警告
  - `TestStop_TimesOutWhenWorkerHangs` — PASS(8.02s,符合 8s 兜底预期)
  - `TestStop_NoopWhenNotRunning` — PASS(立即返回)
  - `TestCollectDeviceInfo_*` (5 个) — 全部 PASS,无回归
- **awaiting**: 用户本地真实环境验证(启动 + cron 触发 + Ctrl+C 观察 < 15s 干净退出)
  - ✅ **RESOLVED 2026-07-06 + 池满 follow-up**: 扩展为 24/20 池满调查,定位到 3 处 refCount 泄漏反模式(用户正确指出是 release bug 非容量),合并 fix @ 0b5bfd81 落地
- **tdd_checkpoint**:
  - test_file: internal/services/device_info_collection_service_test.go
  - test_names: TestStop_TimesOutWhenWorkerHangs, TestStop_NoopWhenNotRunning
  - status: green
  - failure_output_before_fix: would have hung forever (unbounded wg.Wait)

## Evidence

- timestamp: 2026-07-06T10:30
  checked: cmd/main.go:251-270 (waitForShutdown)
  found: coreModule.Close() called with NO timeout; only srv.Shutdown(ctx) has 10s bound
  implication: 关闭流程无总 deadline,任何 Close 子步骤卡住=整个进程不退
- timestamp: 2026-07-06T10:32
  checked: internal/core/core.go:481-534 (Core.Close)
  found: 11 步串行同步调用,无 ctx 包裹,无超时
  implication: 顺序为 reaper → NoticeHub → DeviceInfoCollection.Stop() → ADSync → DeviceMonitor → Scheduler.Stop() → MetricsCache → RPAScaling → Cache → sleep(100ms) → DB.Close()
- timestamp: 2026-07-06T10:34
  checked: internal/services/device_info_collection_service.go:89-102 (Stop)
  found: 关闭 stopChan + s.wg.Wait() 无超时;worker.processTask 跑 SSH/DB 长 query 无 per-task timeout
  implication: 5 worker + 1 recoverPendingTasks 都可能挂在 ssh 或 DB query 上,wg.Wait 永远不返回
- timestamp: 2026-07-06T10:36
  checked: internal/scheduler/cron.go:254-277 (Scheduler.Stop)
  found: s.cron.Stop() 返回 ctx(robfig cron 内部),5s timeout 等 in-flight job,超时后强制退出
  implication: Scheduler 自带 5s 兜底,但因在 step 6 排 DeviceInfoCollection.Stop 之后,前 5s 仍在 spawn 任务
- timestamp: 2026-07-06T10:38
  checked: internal/scheduler/ad_sync_tasks.go:180-191 (ADSyncScheduler.Stop)
  found: s.cron.Stop() + s.cancel() — robfig cron.Stop() 立即返回,不等待 in-flight
  implication: ADSync 不会被卡;但 step 4 排在 step 3 之后,前 5s 仍可触发新 AD 任务
- timestamp: 2026-07-06T10:40
  checked: internal/services/asset/reconciliation_snapshot.go:64-98 (RefreshView)
  found: 自带 reconciliationRefreshTimeout = 90s(防御网),所以 reconciliation:refreshView 不会无限挂
  implication: 对账任务不是无限挂的真凶;但仍占用 DB 连接 + 在 close 期间新增 DB 压力
- timestamp: 2026-07-06T10:42
  checked: internal/services/device_info_collection_service.go:215-262 (processTask)
  found: CollectDeviceInfo → wrapper.SendCommand (SSH) 阻塞,无 ctx 取消
  implication: worker 在 SSH 命令中卡住时,wg.Wait 永远不返回 — Stop 死锁

## Eliminated

- hypothesis: Windows 0xC000013a 引发 Go runtime 关闭
  evidence: 这是 wmic 子进程被 SIGINT 传递杀掉,Go 进程本身信号处理正常,关闭卡在 core.Close 内部
  timestamp: 2026-07-06T10:45
- hypothesis: DB pool 死锁是单一根因
  evidence: pool 24/20 是结果(close 期间 cron 继续 spawn 任务 + worker 仍 hold 连接),不是根因;治理 pool 需先停 cron
  timestamp: 2026-07-06T10:46

## Resolution

### root_cause

`core.Close()` 关闭顺序错误 + 无总 deadline,导致关闭流程死锁。`DeviceInfoCollectionService.Stop()`(`core.go:496`)在 step 3 持有 `sync.WaitGroup.Wait()`(`device_info_collection_service.go:99`)等待 5 个 worker + 1 recover 协程退出,但 worker 内的 `wrapper.SendCommand`(`device_info_collection_service.go:292`)和 DB 查询无 per-task timeout,任意 SSH 卡住就永远不返回;与此同时 `Scheduler.Stop()` 在 step 6 才调用(`core.go:508`),前 5-15 秒内 cron 引擎继续触发 `reconciliation:refreshView` / `device_info_update` / `createWorkorderHigh` 等任务,新任务持续抢 DB 连接,DB pool 被打满(24>20),in-flight 的 `REFRESH CONCURRENTLY` 因"连接池已满"反复失败,日志刷屏但进程不退。再叠加 `cmd/main.go:266` `coreModule.Close()` 调用无 ctx/timeout,整个关闭流程没有兜底,任何子步骤阻塞=整个进程永远不退。

### fix

1. **重排 `Core.Close()` 关闭顺序**(core.go):Scheduler 与 ADSyncScheduler 提前到 step 3,先切断新任务源;DeviceInfoCollectionService / DeviceMonitor / RPAScaling 等依赖 cron 的服务后停。
2. **为 `Core.Close()` 加 30s 总 deadline**(core.go):`closeDone` channel + goroutine + `time.AfterFunc(30s)`,到点强制 log "强制关闭"并返回。
3. **为 `DeviceInfoCollectionService.Stop()` 加 8s 内部 timeout**(device_info_collection_service.go):把 wg.Wait 改为 select + time.After,超时后 log "Worker 未在超时内退出"继续返回,不阻塞 Close。

### verification

- `go build ./...` — 编译通过
- `go vet ./internal/core/... ./internal/services/...` — 无警告
- 用户本地验证:启动后端 → 等 cron 触发若干对账/物化视图/采集任务 → Ctrl+C → 观察在 15s 内完成关闭日志并退出

### files_changed

- D:\code\ClaudeCode\xingran-go-backend\internal\core\core.go
- D:\code\ClaudeCode\xingran-go-backend\internal\services\device_info_collection_service.go
