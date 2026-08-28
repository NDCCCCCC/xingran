---
plan: quick-quirk-p2
status: pending
---

# QUIRK-P2: DeviceConnectionPool.startCleanup goroutine 泄漏

## 问题

`internal/device/connection_pool.go` 的 `DeviceConnectionPool.Close()` 关闭 `p.done` channel 后，内部的 `startCleanup` goroutine 仍在运行（若 cleanup interval 未到），导致 `NumGoroutine` +1 的 goroutine 泄漏。78-02 探针发现（通过 `runtime.NumGoroutine` 差值），Phase 79/80 记录。

## 修复

`internal/device/connection_pool.go` 的 `Close()` 方法，在关闭 `p.done` 后向 cleanup goroutine 发信号，确保其退出：

方案 A（推荐）：在 `p.done` 关闭后，`startCleanup` goroutine 的 `select` 需检测到 `p.done` 并返回。在 `startCleanup` 的 `for` 循环 `select` 中已有 `case <-p.done:` 分支，需确认该分支存在且正确。

方案 B（兜底）：若 Close 关闭 `p.done` 后等待短暂时间（如 100ms）让 cleanup goroutine 自然退出。

先读 `internal/device/connection_pool.go` 确认 `startCleanup` 结构，然后决定方案。

## 验证

1. `go test -count=1 -race ./internal/device/` — 无 goroutine 泄漏告警
2. `go build ./...` exit 0
3. `go test ./...` exit 0
