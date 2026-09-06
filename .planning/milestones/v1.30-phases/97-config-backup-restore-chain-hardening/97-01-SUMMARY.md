# Plan 97-01 Summary: V130R-01 超时互斥原子化

**Status:** ✅ COMPLETED
**Date:** 2026-09-06
**Plans:** 97-01

## Changes Made

### 1. pkg/constants/timeouts.go — 常量拆分
- 删除 `RestoreConfigTimeout = 10 * time.Minute`
- 新增 `RestoreBackupTimeout = 30 * time.Second`（阶段①：恢复前备份）
- 新增 `RestoreConfigExecTimeout = 5 * time.Minute`（阶段②：RestoreConfig 下发）

### 2. internal/services/config_restore_task_service.go — 两段 context 重构
- `StartRestore`：不再传递 `runCtx` 到 goroutine，goroutine 内部自己管理两段 context
- `runRestore(baseCtx, taskID)`：
  - 阶段①：独立 `backupCtx`（30s budget），CreateBackup 超时则 failTask 并退出
  - 阶段②：独立 `restoreCtx`（5min budget），RestoreConfig 超时则 failTask 并退出
  - 任一段 ctx cancel 后，goroutine 检测 `ctx.Done()` 立即退出，不依赖 ExecuteCustom 内部超时

### 3. internal/device/restore_config.go — 超时常量引用
- `ExecuteCustom` 超时参数从 `RestoreConfigTimeout` 更新为 `RestoreConfigExecTimeout`

### 4. pkg/constants/timeouts_test.go — 测试期望更新
- `TestTimeoutsConstantCount`：`want` 从 9 改为 10（RestoreConfigTimeout → 2 个常量）

### 5. internal/services/config_restore_task_service_97_01_test.go — 回归测试（新增）
- `TestV130R01_RestoreConfigTimeoutSplit`：验证两段常量值和关系
- `TestV130R01_StartRestoreGoroutineLaunch`：验证 goroutine 异步执行
- `TestV130R01_RunRestorePhaseOrdering`：验证两阶段按序执行

## Verification

| Check | Result |
|-------|--------|
| `go build ./...` | ✅ PASS |
| `go test -run "TestV130R01|TestTimeouts" ./internal/services/ ./pkg/constants/` | ✅ PASS (5 tests) |
| 常量计数稳定性测试 | ✅ PASS |

## Truths Verified

1. ✅ 两段 timeout 互相独立（30s backup + 5min restore）
2. ✅ goroutine 不再依赖传入的 runCtx，自己管理 ctx 生命周期
3. ✅ `RestoreConfigExecTimeout` 替代旧 `RestoreConfigTimeout`（10min → 5min for 下发阶段）

## Notes

- V130R-01 D-01 + D-02 实现完成
- 回归测试覆盖超时路径和正常路径
- Phase 93 的 `ExecuteCustom` 内部超时引用同步更新
