---
plan: quick-quirk-p2
status: complete
completed: 2026-08-27
commit: 05afbc8
---

# QUIRK-P2: DeviceConnectionPool.Close() goroutine 泄漏修复

## 修复

`internal/device/connection_pool.go` `Close()` 方法:将 `cleanupTicker.Stop()` 移到 `close(p.done)` **之前**，确保 `startCleanup` goroutine 通过 `case <-p.done:` 退出，而非在 ticker channel 有残留值时跳过退出分支。

## 回归测试

`internal/device/connection_pool_quirk_p2_test.go` — `TestDeviceConnectionPool_Close_GoroutineLeak`: 用 `runtime.NumGoroutine` 差值检测泄漏。

## 验证

| 验证项 | 结果 |
|--------|------|
| `go build ./internal/device/...` | PASS |
| `go test -run TestDeviceConnectionPool_Close_GoroutineLeak ./internal/device/` | PASS |
| `go test ./...` | PASS (全包) |
