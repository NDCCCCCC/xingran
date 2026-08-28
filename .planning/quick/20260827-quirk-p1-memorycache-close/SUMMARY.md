---
plan: quick-quirk-p1
status: complete
completed: 2026-08-27
commit: 4282983
---

# QUIRK-P1: MemoryCache.Close() 幂等化

## 修复

`pkg/cache/memory.go`: struct 添加 `stopOnce sync.Once`; `Close()` 改为:

```go
func (m *MemoryCache) Close() error {
    var err error
    m.stopOnce.Do(func() { close(m.stopChan) })
    return err
}
```

## 回归测试

`pkg/cache/cache_quirk_p1_test.go` — `TestMemoryCache_Close_Idempotent`: 调用 Close() 两次，第二次不 panic。

## 验证

| 验证项 | 结果 |
|--------|------|
| `go build ./pkg/cache/...` | PASS |
| `go test -run TestMemoryCache_Close_Idempotent ./pkg/cache/` | PASS |
| `go build ./...` | PASS |
