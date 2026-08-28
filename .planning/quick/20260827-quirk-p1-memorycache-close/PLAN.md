---
plan: quick-quirk-p1
status: pending
---

# QUIRK-P1: MemoryCache.Close() 二次调用 panic

## 问题

`pkg/cache/memory.go:312` 的 `MemoryCache.Close()` 直接 `close(stopChan)` 无 `sync.Once` 守卫，导致二次 Close panic（`close of closed channel`）。78-02 探针发现，Phase 79/80 记录。

## 修复

`pkg/cache/memory.go` 的 `MemoryCache` 结构体添加 `stopOnce sync.Once` 字段，`Close()` 方法改为:

```go
func (c *MemoryCache) Close() error {
    var err error
    c.stopOnce.Do(func() {
        close(c.stopChan)
    })
    return err
}
```

## 验证

1. `go test -run TestMemoryCache_Close_Idempotent ./pkg/cache/` — 二次 Close 不 panic
2. `go test -run TestMx78_Close_Idempotent ./internal/core/` — core 层回归
3. `go build ./...` exit 0
4. `go test ./...` exit 0
